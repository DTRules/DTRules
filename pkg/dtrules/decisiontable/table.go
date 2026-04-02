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
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// DASH represents a "don't care" entry in the condition/action tables
const DASH = "-"

// MAXCOL is the maximum number of columns in a decision table
const MAXCOL = 16

// TableType represents the type of decision table
type TableType int

const (
	// BALANCED tables expect all branches to be defined in the condition table
	BALANCED TableType = iota
	// FIRST executes only the first column whose conditions are met
	FIRST
	// ALL executes all columns whose conditions are met, in order
	ALL
)

// String returns the string representation of the table type
func (t TableType) String() string {
	switch t {
	case BALANCED:
		return "BALANCED"
	case FIRST:
		return "FIRST"
	case ALL:
		return "ALL"
	default:
		return "UNKNOWN"
	}
}

// RDecisionTable holds the rules for a set of policy implemented using DTRules.
type RDecisionTable struct {
	dtrules.BaseObject

	name      *dtrules.RName // The decision table's name
	filename  string         // Filename where the table is defined
	filePath  string         // FILE_PATH: canonical path for Excel generation
	tableType TableType      // Type of decision table

	maxCol int // Number of columns in this decision table

	session dtrules.Session   // Session for compilation context
	fields  map[string]string // Meta information about this decision table

	compiled bool // Whether the table has been compiled

	// Context setup
	contexts        []string       // Context entity names
	contextsPostfix []string       // Compiled postfix for context setup
	contextsComment []string       // Comments on contexts
	rcontext        dtrules.Object // Compiled context setup code

	// Initial actions (executed before conditions)
	initialActions        []string
	initialActionsPostfix []string
	initialActionsComment []string
	rinitialActions       []dtrules.Object         // Go interpreter
	binitialActions       []*dtrules.BytecodeChunk // ASM bytecode

	// Conditions
	conditionTable    [][]string               // conditionTable[row][col] - "y", "n", "-", "*"
	conditions        []string                 // Condition expressions (formal)
	conditionsPostfix []string                 // Compiled postfix
	conditionsComment []string                 // Comments
	rconditions       []dtrules.Object         // Compiled condition code (Go interpreter)
	bconditions       []*dtrules.BytecodeChunk // Compiled bytecode (for ASM execution)

	// Actions
	actionTable    [][]string               // actionTable[row][col] - "x" or ""
	actions        []string                 // Action expressions (formal)
	actionsPostfix []string                 // Compiled postfix
	actionsComment []string                 // Comments
	ractions       []dtrules.Object         // Compiled action code (Go interpreter)
	bactions       []*dtrules.BytecodeChunk // Compiled bytecode (for ASM execution)

	// Policy statements
	policyStatements        []string
	policyStatementsPostfix []string
	rpolicyStatements       []dtrules.Object

	// Tracking which columns/conditions/actions are used
	columnsSpecified  []bool
	columnsUsed       []bool
	conditionsUsed    []bool
	actionsUsed       []bool
	columnUnreachable []bool
	hasNullColumn     bool

	// Star column handling
	starColumn      int // -1 if none
	otherwiseColumn int // -1 if none
	alwaysColumn    int // -1 if none

	optimize bool // Whether to optimize the decision tree

	// The compiled decision tree
	decisionTree DTNode

	// Errors found during compilation
	errors []error
}

// NewRDecisionTable creates a new decision table with the given name
func NewRDecisionTable(name *dtrules.RName, session dtrules.Session) *RDecisionTable {
	return &RDecisionTable{
		name:            name,
		session:         session,
		tableType:       BALANCED,
		maxCol:          1,
		fields:          make(map[string]string),
		starColumn:      -1,
		otherwiseColumn: -1,
		alwaysColumn:    -1,
		optimize:        true,
		errors:          make([]error, 0),
	}
}

// Type returns the type metadata for decision tables
func (dt *RDecisionTable) Type() *dtrules.RType {
	return dtrules.TypeDecisionTable
}

// GetName returns the decision table's name as a string
func (dt *RDecisionTable) GetName() string {
	return dt.name.StringValue()
}

// GetRName returns the decision table's name
func (dt *RDecisionTable) GetRName() *dtrules.RName {
	return dt.name
}

// GetTableType returns the table type
func (dt *RDecisionTable) GetTableType() TableType {
	return dt.tableType
}

// SetTableType sets the table type
func (dt *RDecisionTable) SetTableType(t TableType) {
	dt.tableType = t
}

// IsCompiled returns whether the table has been compiled
func (dt *RDecisionTable) IsCompiled() bool {
	return dt.compiled
}

// GetMaxCol returns the number of columns
func (dt *RDecisionTable) GetMaxCol() int {
	return dt.maxCol
}

// SetMaxCol sets the number of columns.
// Values are clamped to the range [1, MAXCOL].
func (dt *RDecisionTable) SetMaxCol(n int) {
	if n < 1 {
		n = 1
	} else if n > MAXCOL {
		n = MAXCOL
	}
	dt.maxCol = n
}

// GetConditions returns the condition expressions
func (dt *RDecisionTable) GetConditions() []string {
	return dt.conditions
}

// SetConditions sets the condition expressions
func (dt *RDecisionTable) SetConditions(c []string) {
	dt.conditions = c
}

// GetConditionsPostfix returns the compiled condition postfix strings
func (dt *RDecisionTable) GetConditionsPostfix() []string {
	return dt.conditionsPostfix
}

// GetActions returns the action expressions
func (dt *RDecisionTable) GetActions() []string {
	return dt.actions
}

// SetActions sets the action expressions
func (dt *RDecisionTable) SetActions(a []string) {
	dt.actions = a
}

// GetActionsPostfix returns the compiled action postfix strings
func (dt *RDecisionTable) GetActionsPostfix() []string {
	return dt.actionsPostfix
}

// GetConditionTable returns the condition table
func (dt *RDecisionTable) GetConditionTable() [][]string {
	return dt.conditionTable
}

// SetConditionTable sets the condition table
func (dt *RDecisionTable) SetConditionTable(t [][]string) {
	dt.conditionTable = t
}

// GetActionTable returns the action table
func (dt *RDecisionTable) GetActionTable() [][]string {
	return dt.actionTable
}

// SetActionTable sets the action table
func (dt *RDecisionTable) SetActionTable(t [][]string) {
	dt.actionTable = t
}

// GetRConditions returns the compiled conditions
func (dt *RDecisionTable) GetRConditions() []dtrules.Object {
	return dt.rconditions
}

// SetRConditions sets the compiled conditions
func (dt *RDecisionTable) SetRConditions(c []dtrules.Object) {
	dt.rconditions = c
}

// GetRActions returns the compiled actions
func (dt *RDecisionTable) GetRActions() []dtrules.Object {
	return dt.ractions
}

// SetRActions sets the compiled actions
func (dt *RDecisionTable) SetRActions(a []dtrules.Object) {
	dt.ractions = a
}

// GetRPolicyStatements returns the compiled policy statements
func (dt *RDecisionTable) GetRPolicyStatements() []dtrules.Object {
	return dt.rpolicyStatements
}

// GetPolicyStatements returns the policy statement strings
func (dt *RDecisionTable) GetPolicyStatements() []string {
	return dt.policyStatements
}

// GetDecisionTree returns the compiled decision tree
func (dt *RDecisionTable) GetDecisionTree() DTNode {
	return dt.decisionTree
}

// GetErrors returns compilation errors
func (dt *RDecisionTable) GetErrors() []error {
	return dt.errors
}

// IsExecutable returns true (decision tables are always executable)
func (dt *RDecisionTable) IsExecutable() bool {
	return true
}

// isColumnAllStars checks if all conditions in a column have star (*) or dash (-) values.
// Returns true only if the column is a true "otherwise" column where no conditions are tested.
func (dt *RDecisionTable) isColumnAllStars(col int) bool {
	for row := 0; row < len(dt.conditionTable); row++ {
		if col >= len(dt.conditionTable[row]) {
			continue
		}
		v := strings.ToUpper(strings.TrimSpace(dt.conditionTable[row][col]))
		if v != "*" && v != DASH && v != "" {
			return false
		}
	}
	return true
}

// ArrayExecute executes this decision table when it appears in a code block.
func (dt *RDecisionTable) ArrayExecute(state dtrules.State) error {
	return dt.Execute(state)
}

// StringValue returns the table name
func (dt *RDecisionTable) StringValue() string {
	return dt.name.StringValue()
}

// Execute runs the decision table with context handling
func (dt *RDecisionTable) Execute(state dtrules.State) error {
	if !dt.compiled {
		return dtrules.NewRulesError("Execute", "RDecisionTable",
			"Decision table "+dt.name.StringValue()+" has not been compiled")
	}

	// If there's a context, execute it (the context will call ExecuteTable internally)
	// If no context, execute the table directly
	if dt.rcontext != nil {
		return state.Evaluate(dt.rcontext)
	}

	return dt.ExecuteTable(state)
}

// ExecuteTable runs the decision table's initial actions and decision tree directly.
// This is called by the context or by Execute when there's no context.
func (dt *RDecisionTable) ExecuteTable(state dtrules.State) error {
	if !dt.compiled {
		return dtrules.NewRulesError("ExecuteTable", "RDecisionTable",
			"Decision table "+dt.name.StringValue()+" has not been compiled")
	}

	// Execute initial actions
	for i, action := range dt.rinitialActions {
		if action != nil {
			state.SetCurrentTableSection("InitialAction", i)
			if err := state.Evaluate(action); err != nil {
				return err
			}
		}
	}

	// Execute the decision tree
	if dt.decisionTree != nil {
		return dt.decisionTree.Execute(state)
	}

	return nil
}

// Build compiles and builds the decision tree based on table type
func (dt *RDecisionTable) Build(state dtrules.State) error {
	// Validate table configuration before building
	if err := dt.validateConfig(); err != nil {
		return err
	}

	// Compile expressions to executable form
	if err := dt.compile(); err != nil {
		return err
	}

	// Build the decision tree based on type
	switch dt.tableType {
	case BALANCED:
		dt.buildBalanced()
	case FIRST:
		dt.buildUnbalanced(state, false)
	case ALL:
		dt.buildUnbalanced(state, true)
	}

	// Check for errors
	if len(dt.errors) > 0 {
		return dt.errors[0]
	}

	// Mark what's used
	dt.whatsUsed()
	dt.setUnreachable()

	dt.compiled = true
	return nil
}

// validateConfig validates the table configuration before building
func (dt *RDecisionTable) validateConfig() error {
	// Check conditions array matches condition table rows
	if len(dt.conditionTable) > 0 {
		if len(dt.rconditions) < len(dt.conditionTable) {
			return fmt.Errorf("table %s: compiled conditions (%d) fewer than condition table rows (%d)",
				dt.name.StringValue(), len(dt.rconditions), len(dt.conditionTable))
		}
		// Check column count consistency
		for i, row := range dt.conditionTable {
			if len(row) != dt.maxCol {
				return fmt.Errorf("table %s: condition row %d has %d columns, expected %d",
					dt.name.StringValue(), i, len(row), dt.maxCol)
			}
		}
	}

	// Check action table consistency
	if len(dt.actionTable) > 0 {
		if len(dt.ractions) < len(dt.actionTable) {
			return fmt.Errorf("table %s: compiled actions (%d) fewer than action table rows (%d)",
				dt.name.StringValue(), len(dt.ractions), len(dt.actionTable))
		}
	}

	return nil
}

// compile converts postfix strings to executable objects
func (dt *RDecisionTable) compile() error {
	// This is a placeholder - actual compilation would parse postfix
	// and create executable arrays. For now, we assume rconditions and
	// ractions have already been set by the loader.
	return nil
}

// buildBalanced builds a decision tree for a balanced table
func (dt *RDecisionTable) buildBalanced() {
	// Handle empty or star-only tables
	if len(dt.conditionTable) == 0 || len(dt.conditionTable[0]) == 0 {
		dt.decisionTree = NewANodeForColumn(dt, 0)
		return
	}
	if dt.conditionTable[0][0] == "*" {
		dt.decisionTree = NewANodeForColumn(dt, 0)
		return
	}

	// Allocate root node
	dt.decisionTree = NewCNode(dt, 0, 0, dt.rconditions[0])

	// For each column, trace the path through the tree
	for col := 0; col < dt.maxCol; col++ {
		lastStep := equalsIgnoreCase(dt.conditionTable[0][col], "y")
		last := dt.decisionTree.(*CNode)

		star := false
		for i := 1; i < len(dt.conditionTable); i++ {
			t := dt.conditionTable[i][col]

			yes := equalsIgnoreCase(t, "y")
			no := equalsIgnoreCase(t, "n")

			if star {
				dt.errors = append(dt.errors,
					fmt.Errorf("cannot follow '*' with '%s' at row %d, col %d", t, i, col))
			}
			star = equalsIgnoreCase(t, "*")

			if yes || no {
				var here *CNode
				if lastStep {
					if last.IfTrue != nil {
						here, _ = last.IfTrue.(*CNode)
					}
				} else {
					if last.IfFalse != nil {
						here, _ = last.IfFalse.(*CNode)
					}
				}

				if here == nil {
					here = NewCNode(dt, col, i, dt.rconditions[i])
					if lastStep {
						last.IfTrue = here
					} else {
						last.IfFalse = here
					}
				}

				if here.conditionNumber != i {
					dt.errors = append(dt.errors,
						fmt.Errorf("condition table compile error at row %d, col %d", i, col))
					return
				}

				last = here
				lastStep = yes
			}
		}

		// Add actions at the leaf
		if lastStep {
			last.IfTrue = NewANodeForColumn(dt, col)
		} else {
			last.IfFalse = NewANodeForColumn(dt, col)
		}
	}

	// Validate the tree is complete
	coord := dt.decisionTree.Validate()
	if coord != nil {
		dt.errors = append(dt.errors,
			fmt.Errorf("condition table isn't balanced at row %d, col %d", coord.Row, coord.Col))
		dt.compiled = false
	}
}

// buildUnbalanced builds a decision tree for FIRST or ALL type tables
func (dt *RDecisionTable) buildUnbalanced(state dtrules.State, executeAll bool) {
	// Handle empty tables
	if len(dt.conditionTable) == 0 || len(dt.conditionTable[0]) == 0 {
		return
	}
	if dt.conditionTable[0][0] == "*" {
		dt.decisionTree = NewANodeForColumn(dt, 0)
		return
	}

	if len(dt.conditions) < 1 {
		dt.errors = append(dt.errors,
			fmt.Errorf("must have at least one condition in a decision table"))
		return
	}

	// Create the root node
	top := NewCNode(dt, 1, 0, dt.rconditions[0])

	// Process each column
	for col := 0; col < dt.maxCol; col++ {
		nonEmptyColumn := false
		for row := 0; row < len(dt.conditions); row++ {
			v := dt.conditionTable[row][col]
			if v == DASH || v == " " {
				dt.conditionTable[row][col] = DASH
			} else {
				nonEmptyColumn = true
			}
		}
		if nonEmptyColumn {
			dt.processCol(executeAll, top, 0, col, -1)
		}
	}

	// Add defaults to unmapped branches
	defaults := NewANode(dt)
	dt.addDefaults(top, defaults)

	// Optimize the tree
	dt.decisionTree = dt.optimizeTree(state, top)
}

// processCol builds a path through the decision tree for a particular column
func (dt *RDecisionTable) processCol(executeAll bool, here DTNode, row, col, istar int) DTNode {
	// Note: We no longer enforce strict "one star column must be last" constraint.
	// Stars in individual cells are treated as "don't care" for that condition.
	// Only a true "otherwise" column (all conditions are stars) becomes the star column.

	// End of column - create action node
	if row >= len(dt.conditions) {
		thisCol := NewANodeForColumn(dt, col)

		// Check if this is a true "star column" (all conditions evaluated via star)
		// by checking if istar was set AND it's the first row (meaning all rows had stars)
		isFullStarColumn := istar == 0 && dt.isColumnAllStars(col)
		thisCol.SetStar(isFullStarColumn)

		if isFullStarColumn {
			dt.starColumn = col

			condition := strings.TrimSpace(strings.ToLower(dt.conditions[istar]))
			always := condition == "always"
			// Treat any non-"always" star condition as "otherwise" (default behavior)
			otherwise := !always

			if here == nil {
				return thisCol
			}

			if otherwise {
				dt.otherwiseColumn = col
				return here
			}

			if always {
				dt.alwaysColumn = col
				if anode, ok := here.(*ANode); ok {
					if err := thisCol.AddNode(anode); err != nil {
						dt.errors = append(dt.errors, err)
					}
				}
				return thisCol
			}
		}

		if here != nil && !executeAll {
			// FIRST type - keep existing path
			return here
		}

		if here != nil && executeAll {
			// ALL type - combine actions
			if anode, ok := here.(*ANode); ok {
				if err := thisCol.AddNode(anode); err != nil {
					dt.errors = append(dt.errors, err)
				}
			}
			return thisCol
		}

		return thisCol
	}

	v := dt.conditionTable[row][col]
	dcare := v == DASH
	yes := equalsIgnoreCase(v, "y")
	no := equalsIgnoreCase(v, "n")

	if istar >= 0 && (yes || no) {
		dt.errors = append(dt.errors,
			fmt.Errorf("cannot follow '*' with '%s' at row %d, col %d", v, row+1, col+1))
	}

	if equalsIgnoreCase(v, "*") {
		istar = row
	}

	// Skip don't-care on non-matching row
	if (here == nil || here.GetRow() != row) && dcare {
		return dt.processCol(executeAll, here, row+1, col, istar)
	}

	// Handle star
	if istar >= 0 {
		t := dt.processCol(executeAll, here, row+1, col, istar)
		t.SetStar(true)
		return t
	}

	// Create or navigate CNode
	var cnode *CNode
	if here == nil {
		cnode = NewCNode(dt, col, row, dt.rconditions[row])
	} else if here.GetRow() != row {
		cnode = NewCNode(dt, col, row, dt.rconditions[row])
		cnode.IfFalse = here
		cnode.IfTrue = here.CloneDTNode()
	} else {
		cnode, _ = here.(*CNode)
	}

	// Recurse on true/false branches
	if yes || dcare {
		next := cnode.IfTrue
		t := dt.processCol(executeAll, next, row+1, col, -1)
		cnode.IfTrue = t
	}
	if no || dcare {
		next := cnode.IfFalse
		t := dt.processCol(executeAll, next, row+1, col, -1)
		cnode.IfFalse = t
	}

	return cnode
}

// addDefaults fills unmapped branches with the default node
func (dt *RDecisionTable) addDefaults(node DTNode, defaults *ANode) DTNode {
	if node == nil {
		return defaults
	}
	if _, ok := node.(*ANode); ok {
		return node
	}
	cnode := node.(*CNode)
	cnode.IfFalse = dt.addDefaults(cnode.IfFalse, defaults)
	cnode.IfTrue = dt.addDefaults(cnode.IfTrue, defaults)
	return node
}

// optimizeTree collapses branches that execute the same actions
func (dt *RDecisionTable) optimizeTree(state dtrules.State, node DTNode) DTNode {
	if !dt.optimize {
		return node
	}
	return dt.optimizeNode(state, node)
}

func (dt *RDecisionTable) optimizeNode(state dtrules.State, node DTNode) DTNode {
	opt := node.GetCommonANode(state)
	if opt != nil {
		return opt
	}

	cnode := node.(*CNode)
	cnode.IfTrue = dt.optimizeNode(state, cnode.IfTrue)
	cnode.IfFalse = dt.optimizeNode(state, cnode.IfFalse)

	if cnode.IfTrue.EqualsNode(state, cnode.IfFalse) {
		_ = cnode.IfTrue.AddNode(cnode.IfFalse) // error ignored during optimization
		return cnode.IfTrue
	}
	return cnode
}

// whatsUsed marks which columns, conditions, and actions are actually used
func (dt *RDecisionTable) whatsUsed() {
	conditionCnt := 0
	if len(dt.conditionTable) > 0 {
		conditionCnt = len(dt.conditionTable[0])
	}

	dt.columnsSpecified = make([]bool, conditionCnt)
	dt.columnsUsed = make([]bool, conditionCnt)
	dt.columnUnreachable = make([]bool, conditionCnt)
	dt.conditionsUsed = make([]bool, len(dt.conditions))
	dt.actionsUsed = make([]bool, len(dt.actions))

	for col := 0; col < conditionCnt; col++ {
		for row := 0; row < len(dt.conditionTable); row++ {
			v := dt.conditionTable[row][col]
			if equalsIgnoreCase(v, "y") || equalsIgnoreCase(v, "n") || equalsIgnoreCase(v, "*") {
				dt.columnsSpecified[col] = true
			}
		}

		for row := 0; row < len(dt.actions); row++ {
			if col < len(dt.actionTable[row]) &&
				dt.columnsSpecified[col] &&
				equalsIgnoreCase(dt.actionTable[row][col], "x") {
				dt.columnsUsed[col] = true
				dt.actionsUsed[row] = true
			}
		}
	}
}

// setUnreachable marks columns that don't appear in the decision tree
func (dt *RDecisionTable) setUnreachable() {
	for i := 0; i < len(dt.columnsUsed); i++ {
		dt.columnUnreachable[i] = dt.columnsUsed[i]
	}
	dt.setUnreachableNode(dt.decisionTree)
}

func (dt *RDecisionTable) setUnreachableNode(node DTNode) {
	if node == nil {
		return
	}

	if cnode, ok := node.(*CNode); ok {
		dt.setUnreachableNode(cnode.IfTrue)
		dt.setUnreachableNode(cnode.IfFalse)
		dt.conditionsUsed[cnode.conditionNumber] = true
		return
	}

	if anode, ok := node.(*ANode); ok {
		for _, col := range anode.columns {
			if col <= len(dt.columnUnreachable) {
				dt.columnUnreachable[col-1] = false
			}
		}
		if len(anode.columns) == 0 {
			dt.hasNullColumn = true
		}
		for _, action := range anode.actionNumbers {
			dt.actionsUsed[action] = true
		}
	}
}

// SetField sets a metadata field
func (dt *RDecisionTable) SetField(name, value string) {
	dt.fields[name] = value
}

// GetField gets a metadata field
func (dt *RDecisionTable) GetField(name string) string {
	return dt.fields[name]
}

// Clone creates a copy of this decision table (not typically needed)
func (dt *RDecisionTable) Clone(session dtrules.Session) (dtrules.Object, error) {
	// Decision tables are typically not cloned - return self
	return dt, nil
}

// RClone returns this decision table
func (dt *RDecisionTable) RClone() dtrules.Object {
	return dt
}

// GetExecutable returns this decision table (always executable)
func (dt *RDecisionTable) GetExecutable() dtrules.Object {
	return dt
}

// GetNonExecutable returns this decision table
func (dt *RDecisionTable) GetNonExecutable() dtrules.Object {
	return dt
}

// PostFix returns the postfix representation
func (dt *RDecisionTable) PostFix() string {
	return dt.name.StringValue()
}

// Equals compares this decision table with another object
func (dt *RDecisionTable) Equals(o dtrules.Object) (bool, error) {
	if o.Type() != dtrules.TypeDecisionTable {
		return false, nil
	}
	other, ok := o.(*RDecisionTable)
	if !ok {
		return false, nil
	}
	return dt == other, nil
}

// Compare is not supported for decision tables
func (dt *RDecisionTable) Compare(o dtrules.Object) (int, error) {
	return 0, dtrules.UndefinedError("Compare", "Decision tables do not support Compare")
}

// RStringValue returns an RString for this decision table's name
func (dt *RDecisionTable) RStringValue() *dtrules.RString {
	return dtrules.NewRString(dt.name.StringValue())
}

// GetFilename returns the source filename
func (dt *RDecisionTable) GetFilename() string {
	return dt.filename
}

// GetFilePath returns the canonical file path with fallback to filename
func (dt *RDecisionTable) GetFilePath() string {
	// Check if FILE_PATH is in fields (multi-file architecture)
	if filePath, ok := dt.fields["FILE_PATH"]; ok && filePath != "" {
		return filePath
	}
	if dt.filePath != "" {
		return dt.filePath
	}
	return dt.filename // Fallback to legacy xls_file
}

// SetFilePath sets the canonical file path
func (dt *RDecisionTable) SetFilePath(path string) {
	dt.filePath = path
}

// GetContexts returns the context expressions
func (dt *RDecisionTable) GetContexts() []string {
	return dt.contexts
}

// GetContextsComment returns the context comments
func (dt *RDecisionTable) GetContextsComment() []string {
	return dt.contextsComment
}

// GetInitialActions returns the initial action expressions
func (dt *RDecisionTable) GetInitialActions() []string {
	return dt.initialActions
}

// GetInitialActionsComment returns the initial action comments
func (dt *RDecisionTable) GetInitialActionsComment() []string {
	return dt.initialActionsComment
}

// GetConditionsComment returns the condition comments
func (dt *RDecisionTable) GetConditionsComment() []string {
	return dt.conditionsComment
}

// GetActionsComment returns the action comments
func (dt *RDecisionTable) GetActionsComment() []string {
	return dt.actionsComment
}

// GetFields returns all metadata fields
func (dt *RDecisionTable) GetFields() map[string]string {
	return dt.fields
}
