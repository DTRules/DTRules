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

package authoring_test

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Issue #1253: the expression forms `add to <x> <number>` and `subtract from
// <x> <number>` (intAddTo, intSubFrom, floatAddTo, floatSubFrom) emitted
// nothing. Like `multiply <x> by <number>`, each is now the value x + number
// or x - number; the field itself is not changed.
func elrt1253Seed(calc, _ dtrules.Entity) {
	calc.Put(dtrules.GetRName("n"), dtrules.GetRIntegerValue(3))
	calc.Put(dtrules.GetRName("d"), dtrules.GetRDoubleValue(-3))
}

func TestAddToExpression_Values(t *testing.T) {
	for _, tc := range []struct {
		el    string
		field string
		want  string
	}{
		{`set calc.r = add to n 5`, "r", "8"},
		{`set calc.r = subtract from n 5`, "r", "-2"},
		{`set calc.rd = add to d 5`, "rd", "2"},
		{`set calc.rd = subtract from d 0.5`, "rd", "-3.5"},
		{`set calc.text = substring of "abcdef" from add to n 1 to 6`, "text", "ef"},
	} {
		t.Run(tc.el, func(t *testing.T) {
			calc, _, err := elrtRun(t, nil, []string{tc.el}, elrt1253Seed)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got := elrtGet(t, calc, tc.field).StringValue(); got != tc.want {
				t.Errorf("%s = %q, want %q", tc.field, got, tc.want)
			}
			// The expression reads the field; it does not store to it.
			if got := elrtGet(t, calc, "n").StringValue(); got != "3" {
				t.Errorf("n = %s, want 3 (unchanged)", got)
			}
		})
	}
}

func TestAddToExpression_Conditions(t *testing.T) {
	for _, cond := range []string{
		`add to d 5 > 0`,
		`subtract from d 5 < -7`,
		`add to n 5 == 8`,
	} {
		t.Run(cond, func(t *testing.T) {
			calc, _, err := elrtRun(t, []string{cond}, []string{`set calc.hit = true`}, elrt1253Seed)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got, _ := elrtGet(t, calc, "hit").BooleanValue(); !got {
				t.Errorf("condition %s was false", cond)
			}
		})
	}
}
