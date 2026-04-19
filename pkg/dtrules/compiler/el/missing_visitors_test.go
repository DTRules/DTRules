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

package el

import (
	"strings"
	"testing"
)

// Issue #687: missing fexpr visitors. Three absolute-value grammar
// alternatives (floatAbs / intAbs / bigAbs) but only bigAbs had a
// visitor — `absolute value of <field>` produced empty postfix for fp,
// int, and double fields. The rounded family emitted `round`, an op
// that was never registered, so every `X rounded` rule failed at
// runtime.

// TestAbsoluteValue_AllTypes guards that `absolute value of <field>`
// emits the right abs op for each declared field type.
func TestAbsoluteValue_AllTypes(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"ap.count":  TypeInteger,
		"bp.total":  TypeBigInt,
		"dp.rate":   TypeDouble,
		"wp.amount": TypeFixed,
	})
	cases := []struct {
		dsl, wantOp string
	}{
		{"set ap.count = absolute value of ap.count", "abs"},
		{"set bp.total = absolute value of bp.total", "babs"},
		{"set dp.rate = absolute value of dp.rate", "fabs"},
		{"set wp.amount = absolute value of wp.amount", "fpabs"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			postfix, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			if !strings.Contains(postfix, tc.wantOp) {
				t.Errorf("expected %q, got: %s", tc.wantOp, postfix)
			}
			// Regression pin: the old broken emission produced just the
			// assignment cast (e.g. "cvfp /wp.amount xdef") with no field
			// read or abs op.
			fieldName := strings.TrimPrefix(strings.Split(tc.dsl, " = ")[0], "set ")
			if !strings.Contains(postfix, fieldName+" ") && !strings.HasPrefix(postfix, fieldName) {
				t.Errorf("expected field read for %q, got: %s", fieldName, postfix)
			}
		})
	}
}

// TestRounded_UsesRegisteredOp guards that all three rounded-family
// visitors emit the registered `roundto` op rather than the previously-
// emitted (unregistered) `round`. `roundto` signature is
// `(number places boundary -- result)`; defaults are places=0 and
// boundary=0.5 (half-up) when the DSL omits them.
func TestRounded_UsesRegisteredOp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"dp.rate": TypeDouble})
	cases := []struct {
		dsl      string
		mustHit  []string
		mustNot  []string
	}{
		{
			// Unqualified `rounded` — fills in defaults.
			dsl:     "set dp.rate = dp.rate rounded",
			mustHit: []string{"dp.rate", "0", "0.5", "roundto"},
			mustNot: []string{" round ", " round\n", " round\""},
		},
		{
			// Explicit places, default boundary.
			dsl:     "set dp.rate = dp.rate rounded to 2 decimal places",
			mustHit: []string{"dp.rate", "2", "0.5", "roundto"},
			mustNot: []string{" round "},
		},
		{
			// Explicit places and boundary. Note the keyword is spelled
			// `boundry` in the EL grammar (not `boundary`).
			dsl:     "set dp.rate = dp.rate rounded to 4 decimal places with boundry 0.25",
			mustHit: []string{"dp.rate", "4", "0.25", "roundto"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			postfix, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			for _, want := range tc.mustHit {
				if !strings.Contains(postfix, want) {
					t.Errorf("expected %q, got: %s", want, postfix)
				}
			}
			for _, bad := range tc.mustNot {
				if strings.Contains(postfix, bad) {
					t.Errorf("unexpected %q in: %s", bad, postfix)
				}
			}
		})
	}
}
