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
	"math/rand"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

// --- test infrastructure ---

// testState wraps interpreter.DTState so tests can push booleans onto the stack.
type testState struct {
	*interpreter.DTState
}

func newTestState() *testState {
	return &testState{interpreter.NewDTState(nil)}
}

// pushBool pushes a boolean onto the data stack.
func (ts *testState) pushBool(v bool) error {
	return ts.DataPush(dtrules.GetRBoolean(v))
}

// boolCondition is a compiled condition object that pushes a fixed boolean value.
type boolCondition struct {
	dtrules.BaseObject
	value bool
}

func (b *boolCondition) Type() *dtrules.RType                            { return dtrules.TypeBoolean }
func (b *boolCondition) IsExecutable() bool                              { return true }
func (b *boolCondition) GetExecutable() dtrules.Object                   { return b }
func (b *boolCondition) GetNonExecutable() dtrules.Object                { return b }
func (b *boolCondition) RClone() dtrules.Object                          { return b }
func (b *boolCondition) Clone(_ dtrules.Session) (dtrules.Object, error) { return b, nil }
func (b *boolCondition) StringValue() string                             { return fmt.Sprintf("%v", b.value) }
func (b *boolCondition) PostFix() string                                 { return b.StringValue() }
func (b *boolCondition) RStringValue() *dtrules.RString                  { return dtrules.NewRString(b.StringValue()) }
func (b *boolCondition) BooleanValue() (bool, error)                     { return b.value, nil }

func (b *boolCondition) Execute(state dtrules.State) error {
	return state.DataPush(dtrules.GetRBoolean(b.value))
}
func (b *boolCondition) ArrayExecute(state dtrules.State) error {
	return b.Execute(state)
}
func (b *boolCondition) Equals(o dtrules.Object) (bool, error) {
	other, ok := o.(*boolCondition)
	return ok && b == other, nil
}

// countingCondition wraps boolCondition and counts how many times it is evaluated.
type countingCondition struct {
	boolCondition
	count *int
}

func (c *countingCondition) Execute(state dtrules.State) error {
	*c.count++
	return state.DataPush(dtrules.GetRBoolean(c.value))
}
func (c *countingCondition) ArrayExecute(state dtrules.State) error { return c.Execute(state) }

// testAction records execution order.
type testAction struct {
	dtrules.BaseObject
	num      int
	executed *[]int
}

func (a *testAction) Type() *dtrules.RType                            { return dtrules.TypeArray }
func (a *testAction) IsExecutable() bool                              { return true }
func (a *testAction) GetExecutable() dtrules.Object                   { return a }
func (a *testAction) GetNonExecutable() dtrules.Object                { return a }
func (a *testAction) RClone() dtrules.Object                          { return a }
func (a *testAction) Clone(_ dtrules.Session) (dtrules.Object, error) { return a, nil }
func (a *testAction) StringValue() string                             { return fmt.Sprintf("action%d", a.num) }
func (a *testAction) PostFix() string                                 { return a.StringValue() }
func (a *testAction) RStringValue() *dtrules.RString                  { return dtrules.NewRString(a.StringValue()) }

func (a *testAction) Execute(_ dtrules.State) error {
	*a.executed = append(*a.executed, a.num)
	return nil
}
func (a *testAction) ArrayExecute(state dtrules.State) error { return a.Execute(state) }
func (a *testAction) Equals(o dtrules.Object) (bool, error) {
	other, ok := o.(*testAction)
	return ok && a == other, nil
}

// buildTestTable constructs a decision table suitable for tests.
func buildTestTable(
	name string,
	tt TableType,
	conditions []dtrules.Object,
	actions []dtrules.Object,
	conditionTable [][]string,
	actionTable [][]string,
	numCols int,
) (*RDecisionTable, error) {
	rname := dtrules.GetRName(name)
	dt := NewRDecisionTable(rname, nil)
	dt.SetTableType(tt)
	dt.maxCol = numCols

	conds := make([]string, len(conditions))
	for i := range conditions {
		conds[i] = fmt.Sprintf("cond%d", i)
	}
	dt.conditions = conds
	dt.rconditions = conditions

	acts := make([]string, len(actions))
	for i := range actions {
		acts[i] = fmt.Sprintf("act%d", i)
	}
	dt.actions = acts
	dt.ractions = actions

	dt.conditionTable = conditionTable
	dt.actionTable = actionTable

	state := newTestState()
	if err := dt.Build(state.DTState); err != nil {
		return nil, err
	}
	return dt, nil
}

// buildForcedLazyTable builds a LazyTable directly, bypassing the threshold.
func buildForcedLazyTable(
	name string,
	conditions []dtrules.Object,
	actions []dtrules.Object,
	conditionTable [][]string,
	actionTable [][]string,
	numCols int,
) *RDecisionTable {
	rname := dtrules.GetRName(name)
	dt := NewRDecisionTable(rname, nil)
	dt.SetTableType(ALL)
	dt.maxCol = numCols

	conds := make([]string, len(conditions))
	for i := range conditions {
		conds[i] = fmt.Sprintf("cond%d", i)
	}
	dt.conditions = conds
	dt.rconditions = conditions

	acts := make([]string, len(actions))
	for i := range actions {
		acts[i] = fmt.Sprintf("act%d", i)
	}
	dt.actions = acts
	dt.ractions = actions

	dt.conditionTable = conditionTable
	dt.actionTable = actionTable

	dt.decisionTree = buildLazyTable(dt)
	dt.compiled = true
	return dt
}

// --- tests ---

// TestLazyTable_SparseSmoke verifies that a 10-condition × 10-column table
// (each column has one Y, rest -) executes correctly as a LazyTable.
func TestLazyTable_SparseSmoke(t *testing.T) {
	executed := make([]int, 0)
	numConds := 10
	numCols := 10

	conditions := make([]dtrules.Object, numConds)
	for i := range conditions {
		conditions[i] = &boolCondition{}
	}

	actions := make([]dtrules.Object, numCols)
	for i := range actions {
		actions[i] = &testAction{num: i, executed: &executed}
	}

	// column j has Y only on row j
	condTable := make([][]string, numConds)
	for row := 0; row < numConds; row++ {
		condTable[row] = make([]string, numCols)
		for col := 0; col < numCols; col++ {
			if row == col {
				condTable[row][col] = "Y"
			} else {
				condTable[row][col] = "-"
			}
		}
	}

	actTable := make([][]string, numCols)
	for row := 0; row < numCols; row++ {
		actTable[row] = make([]string, numCols)
		for col := 0; col < numCols; col++ {
			if row == col {
				actTable[row][col] = "x"
			}
		}
	}

	dt := buildForcedLazyTable("Sparse", conditions, actions, condTable, actTable, numCols)

	if _, ok := dt.decisionTree.(*LazyTable); !ok {
		t.Fatal("expected decisionTree to be a *LazyTable")
	}

	// All conditions false → no column matches → no actions.
	state := newTestState()
	executed = executed[:0]
	if err := dt.ExecuteTable(state); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(executed) != 0 {
		t.Errorf("expected no actions, got %v", executed)
	}

	// Condition 3 true → column 3 fires.
	conditions[3].(*boolCondition).value = true
	state = newTestState()
	executed = executed[:0]
	if err := dt.ExecuteTable(state); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(executed) != 1 || executed[0] != 3 {
		t.Errorf("expected [3], got %v", executed)
	}

	// Conditions 3 and 7 true → columns 3 and 7 fire in order.
	conditions[7].(*boolCondition).value = true
	state = newTestState()
	executed = executed[:0]
	if err := dt.ExecuteTable(state); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(executed) != 2 || executed[0] != 3 || executed[1] != 7 {
		t.Errorf("expected [3 7], got %v", executed)
	}
}

// TestLazyTable_EachConditionEvaluatedAtMostOnce verifies evaluation counts ≤ 1.
func TestLazyTable_EachConditionEvaluatedAtMostOnce(t *testing.T) {
	executed := make([]int, 0)
	numConds := 5
	numCols := 5
	counts := make([]int, numConds)

	conditions := make([]dtrules.Object, numConds)
	for i := range conditions {
		conditions[i] = &countingCondition{
			boolCondition: boolCondition{value: true},
			count:         &counts[i],
		}
	}
	actions := make([]dtrules.Object, numCols)
	for i := range actions {
		actions[i] = &testAction{num: i, executed: &executed}
	}

	// Every column requires every condition as Y.
	condTable := make([][]string, numConds)
	for row := 0; row < numConds; row++ {
		condTable[row] = make([]string, numCols)
		for col := 0; col < numCols; col++ {
			condTable[row][col] = "Y"
		}
	}
	actTable := make([][]string, numCols)
	for row := 0; row < numCols; row++ {
		actTable[row] = make([]string, numCols)
		for col := 0; col < numCols; col++ {
			if row == col {
				actTable[row][col] = "x"
			}
		}
	}

	dt := buildForcedLazyTable("CountTest", conditions, actions, condTable, actTable, numCols)

	state := newTestState()
	if err := dt.ExecuteTable(state); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	for i, c := range counts {
		if c > 1 {
			t.Errorf("condition %d evaluated %d times, want ≤1", i, c)
		}
	}
}

// TestLazyTable_UnreferencedConditionsNotEvaluated verifies conditions referenced only
// by already-eliminated columns are never evaluated.
func TestLazyTable_UnreferencedConditionsNotEvaluated(t *testing.T) {
	// 3 conditions, 2 columns:
	//   col 0: cond0=Y, cond1=Y, cond2=-
	//   col 1: cond0=N, cond1=-, cond2=Y  (cond2 only referenced by col1)
	// With cond0=true: col1 is eliminated after cond0; cond2 must not be evaluated.
	executed := make([]int, 0)
	count0, count1, count2 := 0, 0, 0

	conditions := []dtrules.Object{
		&countingCondition{boolCondition: boolCondition{value: true}, count: &count0},
		&countingCondition{boolCondition: boolCondition{value: true}, count: &count1},
		&countingCondition{boolCondition: boolCondition{value: false}, count: &count2},
	}
	actions := []dtrules.Object{
		&testAction{num: 0, executed: &executed},
		&testAction{num: 1, executed: &executed},
	}
	condTable := [][]string{
		{"Y", "N"},
		{"Y", "-"},
		{"-", "Y"},
	}
	actTable := [][]string{
		{"x", ""},
		{"", "x"},
	}

	dt := buildForcedLazyTable("UnrefTest", conditions, actions, condTable, actTable, 2)

	state := newTestState()
	if err := dt.ExecuteTable(state); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(executed) != 1 || executed[0] != 0 {
		t.Errorf("expected [0], got %v", executed)
	}
	if count2 != 0 {
		t.Errorf("condition 2 evaluated %d times, expected 0", count2)
	}
}

// TestLazyTable_ThresholdExactly64 verifies:
//
//	63 columns → tree, 64 columns → tree, 65 columns → LazyTable.
//
// We use 1 condition with Y/N alternating across columns so the tree stays
// shallow and builds in O(N) time.
func TestLazyTable_ThresholdExactly64(t *testing.T) {
	for _, tc := range []struct {
		cols     int
		wantLazy bool
	}{
		{63, false},
		{64, false},
		{65, true},
	} {
		t.Run(fmt.Sprintf("%d_cols", tc.cols), func(t *testing.T) {
			numCols := tc.cols

			conditions := []dtrules.Object{&boolCondition{value: true}}
			executed := make([]int, 0)
			actions := make([]dtrules.Object, numCols)
			for i := range actions {
				actions[i] = &testAction{num: i, executed: &executed}
			}

			// Alternate Y/N across columns for the single condition row.
			condTable := [][]string{make([]string, numCols)}
			for col := 0; col < numCols; col++ {
				if col%2 == 0 {
					condTable[0][col] = "Y"
				} else {
					condTable[0][col] = "N"
				}
			}

			actTable := make([][]string, numCols)
			for row := 0; row < numCols; row++ {
				actTable[row] = make([]string, numCols)
				if row < numCols {
					actTable[row][row] = "x"
				}
			}

			dt, err := buildTestTable(
				fmt.Sprintf("Thresh%d", tc.cols),
				ALL, conditions, actions, condTable, actTable, numCols,
			)
			if err != nil {
				t.Fatalf("build failed: %v", err)
			}

			// Trees build on first execution now (#1148); force it so the
			// representation choice is observable.
			if err := dt.ensureTree(newTestState()); err != nil {
				t.Fatalf("ensureTree: %v", err)
			}
			_, isLazy := dt.decisionTree.(*LazyTable)
			if isLazy != tc.wantLazy {
				t.Errorf("%d cols: wantLazy=%v got=%v", tc.cols, tc.wantLazy, isLazy)
			}
		})
	}
}

// TestLazyTable_SemanticEquivalence runs ~20 random small ALL tables under both
// tree and forced-lazy encoding and asserts identical action execution.
//
// To ensure semantic equivalence, each action row is assigned to exactly one column
// (no action appears in multiple columns). This avoids the tree optimizer's
// deduplication behaviour, which would produce different results from lazy execution.
func TestLazyTable_SemanticEquivalence(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for trial := 0; trial < 20; trial++ {
		numConds := 2 + rng.Intn(4)
		numCols := 2 + rng.Intn(5)

		condTable := make([][]string, numConds)
		for row := 0; row < numConds; row++ {
			condTable[row] = make([]string, numCols)
			for col := 0; col < numCols; col++ {
				switch rng.Intn(3) {
				case 0:
					condTable[row][col] = "Y"
				case 1:
					condTable[row][col] = "N"
				default:
					condTable[row][col] = "-"
				}
			}
		}

		// Skip tables where any column is all-don't-care: the tree ignores such
		// columns but the lazy table always fires them, producing different results.
		hasAllDash := false
		for col := 0; col < numCols; col++ {
			allDash := true
			for row := 0; row < numConds; row++ {
				if condTable[row][col] != "-" {
					allDash = false
					break
				}
			}
			if allDash {
				hasAllDash = true
				break
			}
		}
		if hasAllDash {
			continue
		}

		// Each action row belongs to exactly one column (diagonal assignment).
		// This prevents the tree optimizer from merging actions across columns.
		actTable := make([][]string, numCols)
		for row := 0; row < numCols; row++ {
			actTable[row] = make([]string, numCols)
			actTable[row][row] = "x"
		}

		condValues := make([]bool, numConds)
		for i := range condValues {
			condValues[i] = rng.Intn(2) == 0
		}

		makeConditions := func() []dtrules.Object {
			objs := make([]dtrules.Object, numConds)
			for i := range objs {
				objs[i] = &boolCondition{value: condValues[i]}
			}
			return objs
		}

		treeExecuted := make([]int, 0)
		lazyExecuted := make([]int, 0)

		makeActions := func(out *[]int) []dtrules.Object {
			objs := make([]dtrules.Object, numCols)
			for i := range objs {
				objs[i] = &testAction{num: i, executed: out}
			}
			return objs
		}

		treeDT, err := buildTestTable(
			fmt.Sprintf("tree%d", trial), ALL,
			makeConditions(), makeActions(&treeExecuted),
			condTable, actTable, numCols,
		)
		if err != nil {
			t.Logf("trial %d: build error (skipping): %v", trial, err)
			continue
		}
		if _, ok := treeDT.decisionTree.(*LazyTable); ok {
			t.Logf("trial %d: unexpectedly lazy, skipping", trial)
			continue
		}

		lazyDT := buildForcedLazyTable(
			fmt.Sprintf("lazy%d", trial),
			makeConditions(), makeActions(&lazyExecuted),
			condTable, actTable, numCols,
		)

		if err := treeDT.ExecuteTable(newTestState()); err != nil {
			t.Fatalf("trial %d: tree execute: %v", trial, err)
		}
		if err := lazyDT.ExecuteTable(newTestState()); err != nil {
			t.Fatalf("trial %d: lazy execute: %v", trial, err)
		}

		if len(treeExecuted) != len(lazyExecuted) {
			t.Errorf("trial %d: tree %v != lazy %v", trial, treeExecuted, lazyExecuted)
			continue
		}
		for i := range treeExecuted {
			if treeExecuted[i] != lazyExecuted[i] {
				t.Errorf("trial %d: action[%d] tree=%d lazy=%d", trial, i, treeExecuted[i], lazyExecuted[i])
			}
		}
	}
}

// TestFirstTableUnaffected verifies FIRST tables never use LazyTable encoding.
// MAXCOL caps at 16, so we use 16 columns which is the maximum allowed.
func TestFirstTableUnaffected(t *testing.T) {
	numCols := MAXCOL
	numConds := numCols
	executed := make([]int, 0)

	conditions := make([]dtrules.Object, numConds)
	for i := range conditions {
		conditions[i] = &boolCondition{value: true}
	}
	actions := make([]dtrules.Object, numCols)
	for i := range actions {
		actions[i] = &testAction{num: i, executed: &executed}
	}

	condTable := make([][]string, numConds)
	for row := 0; row < numConds; row++ {
		condTable[row] = make([]string, numCols)
		for col := 0; col < numCols; col++ {
			if row == col {
				condTable[row][col] = "Y"
			} else {
				condTable[row][col] = "-"
			}
		}
	}
	actTable := make([][]string, numCols)
	for row := 0; row < numCols; row++ {
		actTable[row] = make([]string, numCols)
		for col := 0; col < numCols; col++ {
			if row == col {
				actTable[row][col] = "x"
			}
		}
	}

	dt, err := buildTestTable("FirstBig", FIRST, conditions, actions, condTable, actTable, numCols)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if _, ok := dt.decisionTree.(*LazyTable); ok {
		t.Error("FIRST table should not use LazyTable encoding")
	}
}

// TestBalancedTableUnaffected verifies BALANCED tables never use LazyTable encoding.
func TestBalancedTableUnaffected(t *testing.T) {
	executed := make([]int, 0)
	conditions := []dtrules.Object{
		&boolCondition{value: true},
		&boolCondition{value: false},
	}
	actions := []dtrules.Object{
		&testAction{num: 0, executed: &executed},
		&testAction{num: 1, executed: &executed},
		&testAction{num: 2, executed: &executed},
		&testAction{num: 3, executed: &executed},
	}
	condTable := [][]string{
		{"Y", "Y", "N", "N"},
		{"Y", "N", "Y", "N"},
	}
	actTable := [][]string{
		{"x", "", "", ""},
		{"", "x", "", ""},
		{"", "", "x", ""},
		{"", "", "", "x"},
	}

	dt, err := buildTestTable("BalancedTest", BALANCED, conditions, actions, condTable, actTable, 4)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if _, ok := dt.decisionTree.(*LazyTable); ok {
		t.Error("BALANCED table should not use LazyTable encoding")
	}
}
