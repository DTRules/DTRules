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

import "testing"

// Issue #1251: a string joined to a literal number or a date parsed as
// strConcatInt / strConcatFloat / strConcatDate, which had no emitter, so
// the concatenation compiled to nothing and only the right-hand side of the
// comparison was left.
func TestStrConcatNonString_Runs(t *testing.T) {
	for _, tc := range []struct {
		el   string
		want string
	}{
		{`set calc.text = "x" + 1`, "x1"},
		{`set calc.text = "x" + 1.5`, "x1.5"},
		{`set calc.text = "n=" + (2 + 3)`, "n=5"},
	} {
		t.Run(tc.el, func(t *testing.T) {
			calc, _, err := elrtRun(t, nil, []string{tc.el}, nil)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got := elrtGet(t, calc, "text").StringValue(); got != tc.want {
				t.Errorf("text = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStrConcatNonString_Condition(t *testing.T) {
	for _, cond := range []string{
		`"x" + 1 == "x1"`,
		`"x" + current date == "x" + string value of current date`,
	} {
		t.Run(cond, func(t *testing.T) {
			calc, _, err := elrtRun(t, []string{cond}, []string{`set calc.hit = true`}, nil)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got, _ := elrtGet(t, calc, "hit").BooleanValue(); !got {
				t.Errorf("condition %s was false", cond)
			}
		})
	}
}
