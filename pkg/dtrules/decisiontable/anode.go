// Copyright 2004-2011 DTRules.com, Inc.
// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package decisiontable

import (
	"fmt"
	"sort"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// ANode executes a list of actions.
// It represents a leaf node in the decision tree.
type ANode struct {
	decisionTable   *RDecisionTable          // The Decision Table to which this ANode belongs
	actions         []dtrules.Object         // The compiled action code to execute (Go interpreter)
	actionBytecodes []*dtrules.BytecodeChunk // Compiled bytecode for actions (for ASM execution)
	actionNumbers   []int                    // The action numbers (0-based, for tracing)
	columns         []int                    // Column numbers that lead to this node (1-based)
	star            bool                     // Whether this column has a star
}

// SetActionBytecodes sets the bytecode for all actions.
func (a *ANode) SetActionBytecodes(bcs []*dtrules.BytecodeChunk) {
	a.actionBytecodes = bcs
}

// GetActionBytecodes returns the bytecode for all actions.
func (a *ANode) GetActionBytecodes() []*dtrules.BytecodeChunk {
	return a.actionBytecodes
}

// NewANode creates a new empty ANode for the given decision table
func NewANode(dt *RDecisionTable) *ANode {
	return &ANode{
		decisionTable: dt,
		actions:       make([]dtrules.Object, 0),
		actionNumbers: make([]int, 0),
		columns:       make([]int, 0),
	}
}

// NewANodeForColumn creates an ANode containing all actions marked with "x"
// for the given column in the decision table.
func NewANodeForColumn(dt *RDecisionTable, col int) *ANode {
	actions := make([]dtrules.Object, 0)
	numbers := make([]int, 0)

	for i := 0; i < len(dt.actionTable); i++ {
		if col < len(dt.actionTable[i]) && equalsIgnoreCase(dt.actionTable[i][col], "x") {
			if dt.ractions != nil && i < len(dt.ractions) {
				actions = append(actions, dt.ractions[i])
				numbers = append(numbers, i)
			}
		}
	}

	anode := &ANode{
		decisionTable: dt,
		actions:       actions,
		actionNumbers: numbers,
		columns:       []int{col + 1}, // 1-based column number
	}
	return anode
}

// GetRow returns -1 for action nodes (they don't correspond to condition rows)
func (a *ANode) GetRow() int {
	return -1
}

// CloneDTNode creates a deep copy of this ANode
func (a *ANode) CloneDTNode() DTNode {
	newNode := &ANode{
		decisionTable: a.decisionTable,
		actions:       make([]dtrules.Object, len(a.actions)),
		actionNumbers: make([]int, len(a.actionNumbers)),
		columns:       make([]int, len(a.columns)),
		star:          a.star,
	}
	copy(newNode.actions, a.actions)
	copy(newNode.actionNumbers, a.actionNumbers)
	copy(newNode.columns, a.columns)
	return newNode
}

// CountColumns returns 1 since an ANode represents a single endpoint
func (a *ANode) CountColumns() int {
	return 1
}

// Columns returns the 1-based rule-column numbers that lead to this node.
// Identical nodes are merged during tree building, so a node can answer for
// more than one column. The `policystatements` operator reads this to know
// which columns' policy statements fired (#949).
func (a *ANode) Columns() []int {
	return a.columns
}

// Execute runs all actions in this node
func (a *ANode) Execute(state dtrules.State) error {
	if cb := a.decisionTable.ColumnSelectedCallback; cb != nil && len(a.columns) > 0 {
		cb(a.columns[0])
	}

	// Trace: the fired column wraps its actions; each action is an open
	// element so a performed table's trace nests inside it.
	col := ""
	if len(a.columns) > 0 {
		col = fmt.Sprintf("%d", a.columns[0])
	}
	state.TraceOpen("column", "n", col)
	defer state.TraceClose("column")

	// Publish the table and node so operators that need to know which column
	// fired can ask (`policystatements`, #949). Saved and restored rather than
	// cleared: an action may perform another table, and that table's actions
	// must not leave this one's node behind when they finish.
	prevTable := state.GetCurrentTable()
	prevNode := state.GetANode()
	state.SetCurrentTable(a.decisionTable)
	state.SetANode(a)
	defer func() {
		state.SetCurrentTable(prevTable)
		state.SetANode(prevNode)
	}()

	// Record what this column concluded before its actions run, so an action
	// in the same column can read its own statement and so the statement
	// renders against the data as of the decision (#956).
	if err := a.collectPolicyStatements(state); err != nil {
		return fmt.Errorf("policy statement in table %s: %w", a.decisionTable.GetName(), err)
	}

	for i, action := range a.actions {
		num := a.actionNumbers[i]

		// Save current section info
		section := state.GetCurrentTableSection()
		numHld := state.GetNumberInSection()

		// Set section to this action
		state.SetCurrentTableSection("Action", num)

		// Execute the action
		state.TraceOpen("action", "n", fmt.Sprintf("%d", num+1))
		if err := state.Evaluate(action); err != nil {
			// Restore section and return error with context
			state.TraceClose("action")
			state.SetCurrentTableSection(section, numHld)
			return fmt.Errorf("action %d in table %s: %w", num+1, a.decisionTable.GetName(), err)
		}
		state.TraceClose("action")

		// Restore section
		state.SetCurrentTableSection(section, numHld)
	}
	return nil
}

// collectPolicyStatements appends this node's columns' statements to the
// run's accumulator (#956). Statements collect on their own — no rule has to
// ask — which is what lets a driver table document conclusions the tables it
// performed reached.
//
// Each statement is a template compiled to postfix at build time (see
// excel.CompilePolicyStatement), so `{expr}` substitutions are evaluated here
// against live data rather than reported as literal braces. Columns with no
// statement contribute nothing, and a table with no statements at all does no
// work here.
func (a *ANode) collectPolicyStatements(state dtrules.State) error {
	dt := a.decisionTable
	if dt == nil || len(dt.policyStatements) == 0 {
		return nil
	}

	for _, col := range a.columns {
		if col < 0 || col >= len(dt.policyStatements) || dt.policyStatements[col] == "" {
			continue
		}

		if col < len(dt.rpolicyStatements) && dt.rpolicyStatements[col] != nil {
			// Execute directly rather than through State.Evaluate: Evaluate
			// discards whatever the code leaves on the stack, and the value
			// it leaves is the statement.
			depth := state.DataStackDepth()
			if err := dt.rpolicyStatements[col].Execute(state); err != nil {
				return err
			}
			if state.DataStackDepth() > depth {
				value, err := state.DataPop()
				if err != nil {
					return err
				}
				state.AppendPolicyStatement(value)
				// Drop anything else the statement left behind so
				// collection has no net effect on the data stack.
				for state.DataStackDepth() > depth {
					if _, err := state.DataPop(); err != nil {
						return err
					}
				}
				continue
			}
		}
		// No compiled form (older XML): the authored text is the statement.
		state.AppendPolicyStatement(dtrules.NewRString(dt.policyStatements[col]))
	}
	return nil
}

// Validate always returns nil for ANodes (they are always valid leaf nodes)
func (a *ANode) Validate() *Coordinate {
	return nil
}

// EqualsNode returns true if the other node executes exactly the same actions
func (a *ANode) EqualsNode(state dtrules.State, node DTNode) bool {
	other := node.GetCommonANode(state)
	if other == nil {
		return false
	}
	if len(other.actionNumbers) != len(a.actionNumbers) {
		return false
	}
	for i := 0; i < len(a.actionNumbers); i++ {
		if other.actionNumbers[i] != a.actionNumbers[i] {
			return false
		}
	}
	return a.samePolicyStatement(other)
}

// samePolicyStatement reports whether two action nodes are interchangeable as
// far as policy statements go. The optimizer collapses branches that run the
// same actions, which throws away which column got there — fine until the
// column's policy statement is part of what it does (#949). Nodes whose
// columns carry different statements have different effects and must stay
// apart.
func (a *ANode) samePolicyStatement(other *ANode) bool {
	mine, ok := a.policyStatement()
	if !ok {
		return false
	}
	theirs, ok := other.policyStatement()
	if !ok {
		return false
	}
	return mine == theirs
}

// policyStatement returns the policy statement shared by every column of this
// node. ok is false when the columns disagree, which makes the node itself
// unmergeable.
func (a *ANode) policyStatement() (statement string, ok bool) {
	if a.decisionTable == nil {
		return "", true
	}
	statements := a.decisionTable.policyStatements
	first := true
	for _, col := range a.columns {
		var s string
		if col >= 0 && col < len(statements) {
			s = statements[col]
		}
		if first {
			statement, first = s, false
			continue
		}
		if s != statement {
			return "", false
		}
	}
	return statement, true
}

// GetCommonANode returns this node (an ANode has only one path)
func (a *ANode) GetCommonANode(state dtrules.State) *ANode {
	return a
}

// AddNode combines another node's actions into this one
// Returns error if the nodes are incompatible types
func (a *ANode) AddNode(node DTNode) error {
	other, ok := node.(*ANode)
	if !ok {
		return dtrules.TypeCheckError("ANode.AddNode", "cannot combine ANode with different node type")
	}

	// Add columns (sorted, no duplicates)
	for _, col := range other.columns {
		found := false
		for _, c := range a.columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			a.columns = append(a.columns, col)
		}
	}
	sort.Ints(a.columns)

	// Add actions (sorted by action number, no duplicates)
	for i, idx := range other.actionNumbers {
		found := false
		for _, num := range a.actionNumbers {
			if num == idx {
				found = true
				break
			}
		}
		if !found {
			// Insert in sorted order
			pos := 0
			for pos < len(a.actionNumbers) && a.actionNumbers[pos] < idx {
				pos++
			}
			// Insert at position
			a.actionNumbers = append(a.actionNumbers, 0)
			copy(a.actionNumbers[pos+1:], a.actionNumbers[pos:])
			a.actionNumbers[pos] = idx

			a.actions = append(a.actions, nil)
			copy(a.actions[pos+1:], a.actions[pos:])
			a.actions[pos] = other.actions[i]
		}
	}
	return nil
}

// GetStar returns whether this column has a star
func (a *ANode) GetStar() bool {
	return a.star
}

// SetStar sets the star flag
func (a *ANode) SetStar(star bool) {
	a.star = star
}

// GetColumns returns the list of column numbers that lead to this node
func (a *ANode) GetColumns() []int {
	return a.columns
}

// GetDecisionTable returns the decision table this node belongs to
func (a *ANode) GetDecisionTable() *RDecisionTable {
	return a.decisionTable
}

// String returns a string representation of this ANode
func (a *ANode) String() string {
	return fmt.Sprintf("Action Node for columns %v", a.columns)
}

// Helper function for case-insensitive comparison
func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
