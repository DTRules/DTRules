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

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// ANode executes a list of actions.
// It represents a leaf node in the decision tree.
type ANode struct {
	decisionTable   *RDecisionTable           // The Decision Table to which this ANode belongs
	actions         []dtrules.Object          // The compiled action code to execute (Go interpreter)
	actionBytecodes []*dtrules.BytecodeChunk  // Compiled bytecode for actions (for ASM execution)
	actionNumbers   []int                     // The action numbers (0-based, for tracing)
	columns         []int                     // Column numbers that lead to this node (1-based)
	star            bool                      // Whether this column has a star
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

// Execute runs all actions in this node
func (a *ANode) Execute(state dtrules.State) error {
	for i, action := range a.actions {
		num := a.actionNumbers[i]

		// Save current section info
		section := state.GetCurrentTableSection()
		numHld := state.GetNumberInSection()

		// Set section to this action
		state.SetCurrentTableSection("Action", num)

		// Execute the action
		if err := state.Evaluate(action); err != nil {
			// Restore section and return error with context
			state.SetCurrentTableSection(section, numHld)
			return fmt.Errorf("action %d in table %s: %w", num+1, a.decisionTable.GetName(), err)
		}

		// Restore section
		state.SetCurrentTableSection(section, numHld)
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
	return true
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
