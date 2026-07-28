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

package operators

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// fakePolicyTable stands in for RDecisionTable. The operator reads the table
// structurally, so the decisiontable package (which reaches operators through
// the interpreter) stays out of this test's imports.
type fakePolicyTable struct {
	text     []string
	compiled []dtrules.Object
}

func (f *fakePolicyTable) GetPolicyStatements() []string          { return f.text }
func (f *fakePolicyTable) GetRPolicyStatements() []dtrules.Object { return f.compiled }

// fakeANode stands in for the action node currently executing.
type fakeANode struct{ cols []int }

func (f *fakeANode) Columns() []int { return f.cols }

// pushString is a compiled policy statement that leaves one string behind,
// the way the postfix `"text"` does.
type pushString struct {
	dtrules.Object
	value string
}

func (p *pushString) Execute(state dtrules.State) error {
	return state.DataPush(dtrules.NewRString(p.value))
}

func policyResult(t *testing.T, state dtrules.State) []string {
	t.Helper()
	obj, err := state.DataPop()
	if err != nil {
		t.Fatalf("policystatements pushed nothing: %v", err)
	}
	arr, err := obj.RArrayValue()
	if err != nil {
		t.Fatalf("policystatements pushed a %T, want an array: %v", obj, err)
	}
	var out []string
	for i := 0; i < arr.Size(); i++ {
		e, err := arr.Get(i)
		if err != nil {
			t.Fatalf("array element %d: %v", i, err)
		}
		out = append(out, e.StringValue())
	}
	return out
}

func TestPolicyStatementsForFiredColumn(t *testing.T) {
	state := newTestState()
	state.SetCurrentTable(&fakePolicyTable{
		text: []string{"", "value is 1", "value is 2"},
		compiled: []dtrules.Object{
			nil,
			&pushString{value: "value is 1"},
			&pushString{value: "value is 2"},
		},
	})
	state.SetANode(&fakeANode{cols: []int{2}})

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	got := policyResult(t, state)
	if len(got) != 1 || got[0] != "value is 2" {
		t.Errorf("got %q, want [\"value is 2\"]", got)
	}
	if depth := state.DataStackDepth(); depth != 0 {
		t.Errorf("operator left %d extra values on the stack", depth)
	}
}

// A node reached by several columns — an ALL table, or columns the optimizer
// merged — reports each of their statements, in column order.
func TestPolicyStatementsForMergedColumns(t *testing.T) {
	state := newTestState()
	state.SetCurrentTable(&fakePolicyTable{
		text: []string{"", "first", "", "third"},
		compiled: []dtrules.Object{
			nil,
			&pushString{value: "first"},
			nil,
			&pushString{value: "third"},
		},
	})
	state.SetANode(&fakeANode{cols: []int{1, 2, 3}})

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	got := policyResult(t, state)
	if len(got) != 2 || got[0] != "first" || got[1] != "third" {
		t.Errorf("got %q, want [\"first\" \"third\"] — column 2 has no statement", got)
	}
}

// Without a compiled form (older XML that carries only the description) the
// authored text is used as-is rather than dropped.
func TestPolicyStatementsFallsBackToText(t *testing.T) {
	state := newTestState()
	state.SetCurrentTable(&fakePolicyTable{text: []string{"", "uncompiled"}})
	state.SetANode(&fakeANode{cols: []int{1}})

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	got := policyResult(t, state)
	if len(got) != 1 || got[0] != "uncompiled" {
		t.Errorf("got %q, want [\"uncompiled\"]", got)
	}
}

// Outside a decision-table action there is no column, so the operator yields
// an empty array instead of failing the rule.
func TestPolicyStatementsOutsideATable(t *testing.T) {
	state := newTestState()

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	if got := policyResult(t, state); len(got) != 0 {
		t.Errorf("got %q, want an empty array", got)
	}
}
