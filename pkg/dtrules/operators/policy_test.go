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

// TestPolicyStatementsReturnsWhatAccumulated is the operator's whole job:
// hand back what the columns that fired have concluded so far. Collection
// happens in the decision table as columns fire (#956), so the operator is a
// read — which is what lets a rule report on tables it merely performed.
func TestPolicyStatementsReturnsWhatAccumulated(t *testing.T) {
	state := newTestState()
	state.AppendPolicyStatement(dtrules.NewRString("first"))
	state.AppendPolicyStatement(dtrules.NewRString("second"))

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	got := policyResult(t, state)
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Errorf("got %q, want [\"first\" \"second\"] in fire order", got)
	}
	if depth := state.DataStackDepth(); depth != 0 {
		t.Errorf("operator left %d extra values on the stack", depth)
	}
}

// TestPolicyStatementsBeforeAnyColumnFires: a run that has concluded nothing
// reports nothing, rather than failing.
func TestPolicyStatementsBeforeAnyColumnFires(t *testing.T) {
	state := newTestState()

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	if got := policyResult(t, state); len(got) != 0 {
		t.Errorf("got %q, want an empty array", got)
	}
}

// TestPolicyStatementsArrayIsLive is what makes `clear the policy statements`
// work: it compiles to `policystatements cleararray`, so the operator has to
// hand back the accumulator itself and not a copy of it.
func TestPolicyStatementsArrayIsLive(t *testing.T) {
	state := newTestState()
	state.AppendPolicyStatement(dtrules.NewRString("before the clear"))

	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	if err := opClearArray(state); err != nil {
		t.Fatalf("cleararray: %v", err)
	}

	if got := state.PolicyStatements().Size(); got != 0 {
		t.Errorf("accumulator still holds %d statements after clear", got)
	}

	state.AppendPolicyStatement(dtrules.NewRString("after the clear"))
	if err := opPolicyStatements(state); err != nil {
		t.Fatalf("opPolicyStatements: %v", err)
	}
	got := policyResult(t, state)
	if len(got) != 1 || got[0] != "after the clear" {
		t.Errorf("got %q, want only the statement recorded after the clear", got)
	}
}
