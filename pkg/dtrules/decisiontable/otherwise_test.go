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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// The otherwise column (#1215). '*' does not mean "don't care" -- that is
// '-'. It marks the otherwise column: the last column, holding no Y or N,
// which executes iff no other column executed, in every table type. There is
// no "always" column; an action that must always run takes an X in every
// column.

// otherwiseFixture builds the three-column table the vectors use: column 1
// wants condition 1, column 2 wants condition 2 false, column 3 is the
// otherwise column. Each column has its own action.
func otherwiseFixture(t *testing.T, tt TableType, cond1, cond2 bool, executed *[]int) *RDecisionTable {
	t.Helper()
	conditions := []dtrules.Object{
		&boolCondition{value: cond1},
		&boolCondition{value: cond2},
	}
	actions := []dtrules.Object{
		&testAction{num: 1, executed: executed},
		&testAction{num: 2, executed: executed},
		&testAction{num: 3, executed: executed},
	}
	condTable := [][]string{
		{"Y", "-", "*"},
		{"-", "N", "*"},
	}
	actTable := [][]string{
		{"x", "", ""},
		{"", "x", ""},
		{"", "", "x"},
	}
	dt, err := buildTestTable("Fixture", tt, conditions, actions, condTable, actTable, 3)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return dt
}

// TestOtherwiseFiresWhenNoColumnFires covers FIRST, ALL and BALANCED (the
// type a NONE table loads as): the otherwise column executes exactly when no
// other column did.
func TestOtherwiseFiresWhenNoColumnFires(t *testing.T) {
	cases := []struct {
		name         string
		tableType    TableType
		cond1, cond2 bool
		want         []int
	}{
		{"first/no column matches", FIRST, false, true, []int{3}},
		{"first/column 1 matches", FIRST, true, true, []int{1}},
		{"first/column 2 matches", FIRST, false, false, []int{2}},
		{"all/no column matches", ALL, false, true, []int{3}},
		{"all/both columns match", ALL, true, false, []int{1, 2}},
		{"balanced/no column matches", BALANCED, false, true, []int{3}},
		{"balanced/column 1 matches", BALANCED, true, true, []int{1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			executed := make([]int, 0)
			dt := otherwiseFixture(t, c.tableType, c.cond1, c.cond2, &executed)
			if err := dt.ExecuteTable(newTestState()); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if !sameInts(executed, c.want) {
				t.Errorf("executed %v, want %v", executed, c.want)
			}
		})
	}
}

// TestOtherwiseWithActionInEveryColumn: an action marked X in every column --
// the way to make an action always run -- runs alongside whichever column
// fired, including the otherwise column.
func TestOtherwiseWithActionInEveryColumn(t *testing.T) {
	build := func(cond1 bool, executed *[]int) *RDecisionTable {
		conditions := []dtrules.Object{&boolCondition{value: cond1}}
		actions := []dtrules.Object{
			&testAction{num: 1, executed: executed},
			&testAction{num: 9, executed: executed},
		}
		dt, err := buildTestTable("Always", FIRST,
			conditions, actions,
			[][]string{{"Y", "*"}},
			[][]string{{"x", ""}, {"x", "x"}},
			2)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		return dt
	}

	executed := make([]int, 0)
	dt := build(false, &executed)
	if err := dt.ExecuteTable(newTestState()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !sameInts(executed, []int{9}) {
		t.Errorf("no column matched: executed %v, want [9] (otherwise plus the every-column action)", executed)
	}

	executed = make([]int, 0)
	dt = build(true, &executed)
	if err := dt.ExecuteTable(newTestState()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !sameInts(executed, []int{1, 9}) {
		t.Errorf("column 1 matched: executed %v, want [1 9]", executed)
	}
}

// TestOtherwiseIsNotAlways: there is no "always" column. A condition whose
// text is "always" gets no special treatment -- the column is an ordinary
// otherwise column, and it does not fire when another column does.
func TestOtherwiseIsNotAlways(t *testing.T) {
	executed := make([]int, 0)
	conditions := []dtrules.Object{&boolCondition{value: true}}
	actions := []dtrules.Object{
		&testAction{num: 1, executed: &executed},
		&testAction{num: 2, executed: &executed},
	}
	dt, err := buildTestTable("NoAlways", FIRST,
		conditions, actions,
		[][]string{{"Y", "*"}},
		[][]string{{"x", ""}, {"", "x"}},
		2)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	dt.conditions[0] = "always"
	if err := dt.ExecuteTable(newTestState()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !sameInts(executed, []int{1}) {
		t.Errorf("executed %v, want [1]: an 'always' condition names nothing special", executed)
	}
}

// TestOtherwiseSoleColumn is the Poker / KidAid / TestProject shape: the
// table's only column is the otherwise column, so its actions always run.
func TestOtherwiseSoleColumn(t *testing.T) {
	executed := make([]int, 0)
	dt, err := buildTestTable("Sole", FIRST,
		[]dtrules.Object{&boolCondition{value: false}},
		[]dtrules.Object{&testAction{num: 1, executed: &executed}},
		[][]string{{"*"}},
		[][]string{{"x"}},
		1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if dt.otherwiseColumn != 0 {
		t.Fatalf("otherwiseColumn = %d, want 0", dt.otherwiseColumn)
	}
	if err := dt.ExecuteTable(newTestState()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !sameInts(executed, []int{1}) {
		t.Errorf("executed %v, want [1]", executed)
	}
}

// TestOtherwiseInLazyTable: wide tables execute as a LazyTable, which must
// obey the same rule -- the otherwise column never survives elimination on
// its own, it runs only when nothing else did.
func TestOtherwiseInLazyTable(t *testing.T) {
	run := func(cond1 bool) []int {
		executed := make([]int, 0)
		conditions := []dtrules.Object{&boolCondition{value: cond1}}
		actions := []dtrules.Object{
			&testAction{num: 1, executed: &executed},
			&testAction{num: 2, executed: &executed},
		}
		dt, err := buildTestTable("Lazy", ALL,
			conditions, actions,
			[][]string{{"Y", "*"}},
			[][]string{{"x", ""}, {"", "x"}},
			2)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		// Force the lazy representation the wide-table path uses.
		dt.decisionTree = buildLazyTable(dt)
		if err := dt.ExecuteTable(newTestState()); err != nil {
			t.Fatalf("execute: %v", err)
		}
		return executed
	}

	if got := run(false); !sameInts(got, []int{2}) {
		t.Errorf("no column matched: executed %v, want [2]", got)
	}
	if got := run(true); !sameInts(got, []int{1}) {
		t.Errorf("column 1 matched: executed %v, want [1]", got)
	}
}

// TestOtherwiseColumnLoadErrors: ill-formed '*' is refused when the table is
// built -- at load, not at execute time -- and the message names the table,
// the column and the rule.
func TestOtherwiseColumnLoadErrors(t *testing.T) {
	cases := []struct {
		name      string
		condTable [][]string
		actTable  [][]string
		numCols   int
		wants     []string
	}{
		{
			name:      "not the last column",
			condTable: [][]string{{"Y", "*", "-"}, {"-", "*", "N"}},
			actTable:  [][]string{{"x", "", ""}, {"", "x", ""}, {"", "", "x"}},
			numCols:   3,
			wants:     []string{"Ill", "condition 1", "column 2", "last column (column 3)"},
		},
		{
			name:      "two otherwise columns",
			condTable: [][]string{{"Y", "*", "*"}, {"-", "*", "*"}},
			actTable:  [][]string{{"x", "", ""}, {"", "x", ""}, {"", "", "x"}},
			numCols:   3,
			wants:     []string{"Ill", "column 2", "last column (column 3)"},
		},
		{
			name:      "last column also has a Y",
			condTable: [][]string{{"Y", "-", "*"}, {"-", "N", "Y"}},
			actTable:  [][]string{{"x", "", ""}, {"", "x", ""}, {"", "", "x"}},
			numCols:   3,
			wants:     []string{"Ill", "condition 2", "column 3", "no Y or N"},
		},
		{
			name:      "star cell in an ordinary column",
			condTable: [][]string{{"Y", "*", "-"}, {"-", "N", "-"}},
			actTable:  [][]string{{"x", "", ""}, {"", "x", ""}, {"", "", "x"}},
			numCols:   3,
			wants:     []string{"Ill", "column 2", "last column (column 3)"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conditions := []dtrules.Object{&boolCondition{}, &boolCondition{}}
			executed := make([]int, 0)
			actions := []dtrules.Object{
				&testAction{num: 1, executed: &executed},
				&testAction{num: 2, executed: &executed},
				&testAction{num: 3, executed: &executed},
			}
			_, err := buildTestTable("Ill", FIRST, conditions, actions, c.condTable, c.actTable, c.numCols)
			if err == nil {
				t.Fatal("built an ill-formed '*' table; want a load error")
			}
			for _, want := range c.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

// TestOtherwiseColumnIgnoresPadding: the sparse XML form pads a one-column
// table out to the table's declared width with '-' cells. Those columns hold
// nothing, so the '*' is still in the last column the author wrote.
func TestOtherwiseColumnIgnoresPadding(t *testing.T) {
	executed := make([]int, 0)
	condRow := []string{"*", "-", "-", "-", "-"}
	actRow := []string{"x", "", "", "", ""}
	dt, err := buildTestTable("Padded", FIRST,
		[]dtrules.Object{&boolCondition{}},
		[]dtrules.Object{&testAction{num: 1, executed: &executed}},
		[][]string{condRow}, [][]string{actRow}, 5)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if dt.otherwiseColumn != 0 {
		t.Errorf("otherwiseColumn = %d, want 0 (columns 2-5 are padding)", dt.otherwiseColumn)
	}
	if err := dt.ExecuteTable(newTestState()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !sameInts(executed, []int{1}) {
		t.Errorf("executed %v, want [1]", executed)
	}
}

// TestOtherwiseColumnDetection records which column is the otherwise column
// for the shapes the loader sees: one '*' marks it, the other cells being
// empty is fine, and a table without '*' has none.
func TestOtherwiseColumnDetection(t *testing.T) {
	cases := []struct {
		name      string
		condTable [][]string
		want      int
	}{
		{"star in every cell", [][]string{{"Y", "*"}, {"-", "*"}}, 1},
		{"star in the first row only", [][]string{{"Y", "*"}, {"-", "-"}}, 1},
		{"star in the second row only", [][]string{{"Y", "-"}, {"-", "*"}}, 1},
		{"no star at all", [][]string{{"Y", "N"}, {"-", "N"}}, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			executed := make([]int, 0)
			dt, err := buildTestTable("Detect", FIRST,
				[]dtrules.Object{&boolCondition{}, &boolCondition{}},
				[]dtrules.Object{
					&testAction{num: 1, executed: &executed},
					&testAction{num: 2, executed: &executed},
				},
				c.condTable,
				[][]string{{"x", ""}, {"", "x"}},
				2)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if dt.otherwiseColumn != c.want {
				t.Errorf("otherwiseColumn = %d, want %d", dt.otherwiseColumn, c.want)
			}
		})
	}
}

func sameInts(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
