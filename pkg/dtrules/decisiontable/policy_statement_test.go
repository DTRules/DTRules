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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// policyTable builds a table whose policy statements are indexed by 1-based
// column number, matching what the loader hands the builder.
func policyTable(statements ...string) *RDecisionTable {
	dt := &RDecisionTable{}
	dt.policyStatements = append([]string{""}, statements...)
	return dt
}

// TestEqualsNodeSeparatesDistinctPolicyStatements is the guard for #949: the
// optimizer collapses branches that run the same actions, which erases which
// column got there. When each column carries its own policy statement, that
// erasure is a behavior change — the fired column's statement is part of what
// the column does — so such nodes must not compare equal.
func TestEqualsNodeSeparatesDistinctPolicyStatements(t *testing.T) {
	dt := policyTable("value is 1", "value is 2")
	col1 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{1}}
	col2 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{2}}

	if col1.EqualsNode(nil, col2) {
		t.Error("columns with different policy statements must not be interchangeable")
	}
}

// TestEqualsNodeMergesSharedPolicyStatements keeps the optimization working
// where it is safe: same actions and the same statement (typically none at
// all) means the columns really are interchangeable.
func TestEqualsNodeMergesSharedPolicyStatements(t *testing.T) {
	dt := policyTable("", "")
	col1 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{1}}
	col2 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{2}}

	if !col1.EqualsNode(nil, col2) {
		t.Error("columns with no policy statements and the same actions must still merge")
	}

	same := policyTable("same text", "same text")
	a := &ANode{decisionTable: same, actionNumbers: []int{0}, columns: []int{1}}
	b := &ANode{decisionTable: same, actionNumbers: []int{0}, columns: []int{2}}
	if !a.EqualsNode(nil, b) {
		t.Error("columns whose policy statements are identical must still merge")
	}
}

// TestEqualsNodeStillComparesActions confirms the policy check is additional,
// not a replacement: differing action sets remain unequal.
func TestEqualsNodeStillComparesActions(t *testing.T) {
	dt := policyTable("", "")
	col1 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{1}}
	col2 := &ANode{decisionTable: dt, actionNumbers: []int{1}, columns: []int{2}}

	if col1.EqualsNode(nil, col2) {
		t.Error("nodes running different actions must not be interchangeable")
	}
}

// TestPolicyStatementOfMergedNode covers a node that already answers for
// several columns: it is only mergeable further if those columns agree.
func TestPolicyStatementOfMergedNode(t *testing.T) {
	dt := policyTable("a", "a", "b")

	agreeing := &ANode{decisionTable: dt, columns: []int{1, 2}}
	if got, ok := agreeing.policyStatement(); !ok || got != "a" {
		t.Errorf("policyStatement() = %q, %v; want \"a\", true", got, ok)
	}

	disagreeing := &ANode{decisionTable: dt, columns: []int{1, 3}}
	if _, ok := disagreeing.policyStatement(); ok {
		t.Error("a node spanning columns with different statements must report disagreement")
	}
}

// TestColumnsAccessor is what the policystatements operator reads through.
func TestColumnsAccessor(t *testing.T) {
	a := &ANode{columns: []int{2, 5}}
	got := a.Columns()
	if len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Errorf("Columns() = %v, want [2 5]", got)
	}
}

// stringStatement is a compiled policy statement that renders to a fixed
// value, standing in for the postfix the build compiles from the template.
type stringStatement struct {
	dtrules.BaseObject
	value string
}

func (s *stringStatement) Type() *dtrules.RType             { return dtrules.TypeString }
func (s *stringStatement) IsExecutable() bool               { return true }
func (s *stringStatement) GetExecutable() dtrules.Object    { return s }
func (s *stringStatement) GetNonExecutable() dtrules.Object { return s }
func (s *stringStatement) RClone() dtrules.Object           { return s }
func (s *stringStatement) StringValue() string              { return s.value }
func (s *stringStatement) PostFix() string                  { return s.value }
func (s *stringStatement) RStringValue() *dtrules.RString   { return dtrules.NewRString(s.value) }

func (s *stringStatement) Clone(_ dtrules.Session) (dtrules.Object, error) { return s, nil }

func (s *stringStatement) Equals(o dtrules.Object) (bool, error) {
	other, ok := o.(*stringStatement)
	return ok && s == other, nil
}

func (s *stringStatement) Execute(state dtrules.State) error {
	return state.DataPush(dtrules.NewRString(s.value))
}

func (s *stringStatement) ArrayExecute(state dtrules.State) error {
	return s.Execute(state)
}

func collected(t *testing.T, state dtrules.State) []string {
	t.Helper()
	arr := state.PolicyStatements()
	var out []string
	for i := 0; i < arr.Size(); i++ {
		e, err := arr.Get(i)
		if err != nil {
			t.Fatalf("accumulator element %d: %v", i, err)
		}
		out = append(out, e.StringValue())
	}
	return out
}

// TestCollectPolicyStatementsRendersFiredColumns is the automatic collection
// (#956): a column that fires records what it concluded without any rule
// asking for it.
func TestCollectPolicyStatementsRendersFiredColumns(t *testing.T) {
	dt := policyTable("value is 1", "value is 2")
	dt.rpolicyStatements = []dtrules.Object{
		nil,
		&stringStatement{value: "value is 1"},
		&stringStatement{value: "value is 2"},
	}
	node := &ANode{decisionTable: dt, columns: []int{2}}

	state := newTestState()
	if err := node.collectPolicyStatements(state); err != nil {
		t.Fatalf("collectPolicyStatements: %v", err)
	}
	got := collected(t, state)
	if len(got) != 1 || got[0] != "value is 2" {
		t.Errorf("got %q, want [\"value is 2\"]", got)
	}
	if depth := state.DataStackDepth(); depth != 0 {
		t.Errorf("collection left %d values on the data stack", depth)
	}
}

// Statements accumulate in fire order across every column that fires, which
// is what lets a driver table document what the tables it performed decided.
func TestCollectPolicyStatementsAccumulate(t *testing.T) {
	dt := policyTable("first", "second")
	dt.rpolicyStatements = []dtrules.Object{
		nil,
		&stringStatement{value: "first"},
		&stringStatement{value: "second"},
	}
	state := newTestState()

	for _, col := range []int{1, 2, 1} {
		node := &ANode{decisionTable: dt, columns: []int{col}}
		if err := node.collectPolicyStatements(state); err != nil {
			t.Fatalf("collectPolicyStatements: %v", err)
		}
	}

	got := collected(t, state)
	want := []string{"first", "second", "first"}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("statement %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// A node reached by several columns — an ALL table, or columns the optimizer
// merged — records each of their statements, and columns with no statement
// contribute nothing.
func TestCollectPolicyStatementsForMergedColumns(t *testing.T) {
	dt := policyTable("first", "", "third")
	dt.rpolicyStatements = []dtrules.Object{
		nil,
		&stringStatement{value: "first"},
		nil,
		&stringStatement{value: "third"},
	}
	node := &ANode{decisionTable: dt, columns: []int{1, 2, 3}}

	state := newTestState()
	if err := node.collectPolicyStatements(state); err != nil {
		t.Fatalf("collectPolicyStatements: %v", err)
	}
	got := collected(t, state)
	if len(got) != 2 || got[0] != "first" || got[1] != "third" {
		t.Errorf("got %q, want [\"first\" \"third\"] — column 2 has no statement", got)
	}
}

// Without a compiled form (older XML carrying only the description) the
// authored text is recorded rather than dropped.
func TestCollectPolicyStatementsFallsBackToText(t *testing.T) {
	dt := policyTable("uncompiled")
	node := &ANode{decisionTable: dt, columns: []int{1}}

	state := newTestState()
	if err := node.collectPolicyStatements(state); err != nil {
		t.Fatalf("collectPolicyStatements: %v", err)
	}
	if got := collected(t, state); len(got) != 1 || got[0] != "uncompiled" {
		t.Errorf("got %q, want [\"uncompiled\"]", got)
	}
}

// A table with no policy statements does no work and records nothing.
func TestCollectPolicyStatementsNoStatements(t *testing.T) {
	node := &ANode{decisionTable: &RDecisionTable{}, columns: []int{1}}

	state := newTestState()
	if err := node.collectPolicyStatements(state); err != nil {
		t.Fatalf("collectPolicyStatements: %v", err)
	}
	if got := collected(t, state); len(got) != 0 {
		t.Errorf("got %q, want nothing", got)
	}
}
