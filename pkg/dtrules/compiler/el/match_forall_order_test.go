// Copyright 2026 Paul Snow
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

package el

import "testing"

// TestMatchForallOperandOrder pins the corrected forall operand order for the
// nested `there is match for all … to … in …` construct (#877). Both folds
// must emit the block BEFORE the array (opForall is ( body array -- )); the
// reorder changes the token SEQUENCE, so an exact-postfix match catches a
// regression even though the token multiset is unchanged.
//
// This construct is exercised by exact-postfix rather than execution because
// its runtime semantics are an approximation (entity-attribute matching across
// two arrays); the operand-order bug manifested as a compile-shape error that
// this pins precisely. The other reachable forall aggregations (`all … have`,
// `one of … has a`) are covered by execution in
// pkg/dtrules/boolean_aggregation_exec_test.go. The `there is … in … where`
// array variants (boolThereIsInArrayWhere/NoInArrayWhere) are unreachable —
// the entity alternatives shadow them in the grammar — so the fix to those is
// defensive only.
func TestMatchForallOperandOrder(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"kids": "array", "cases": "array", "tax": "integer"})

	got, err := c.CompileCondition("there is match for all kids to tax in cases")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// Inner fold: `false { … or } cases forall`; outer: `true { … and } kids forall`.
	// Both arrays (cases, kids) appear AFTER their blocks.
	want := "true { false { tax 1 entityfetch == or } cases forall and } kids forall"
	if got != want {
		t.Errorf("operand order wrong\n got: %q\nwant: %q", got, want)
	}
}
