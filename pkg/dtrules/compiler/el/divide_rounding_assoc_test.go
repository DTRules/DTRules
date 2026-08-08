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

import (
	"strings"
	"testing"
)

func compileFixed(t *testing.T, dsl string) string {
	t.Helper()
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"tot": "fixed", "n": "fixed", "x": "fixed", "y": "fixed", "z": "fixed",
		"weekly_budget": "fixed", "unissued_supply": "fixed", "days_per_year": "fixed",
		"days_per_period_num": "long", "days_per_period_den": "long",
		"token_issuance_rate_num": "long", "token_issuance_rate_den": "long",
	})
	pf, err := c.CompileAction(dsl)
	if err != nil {
		t.Fatalf("compile %q: %v", dsl, err)
	}
	return strings.Join(strings.Fields(pf), " ")
}

// TestDivideRoundingByGroupsLeft pins #1015.
//
// A bare `x * y * z` groups left everywhere in the language except the two
// operands of `divide … by … rounding by`, where it grouped right: `iexpr
// TIMES fexpr` is not a left-recursive alternative of fexpr, so ANTLR tries it
// as a primary here and its right operand swallows the rest of the chain.
//
// This is not a cosmetic difference. fp* rounds, so moving the grouping moves
// the rounding point. On the Accumulate staking rules the right-nested form
// shifted a payout by 1 nanoACME (3410864457 vs 3410864458) and broke their
// on-chain period reproduction — from a recompile of unchanged DSL, with
// nothing in the authored source to show why.
func TestDivideRoundingByGroupsLeft(t *testing.T) {
	tests := []struct {
		name string
		dsl  string
		want string
	}{
		{
			"divisor chain groups left",
			"set tot = divide n by x * y * z rounding by 0.5fp",
			"n x y fp* z fp* fphalfup/ cvfp /tot xdef",
		},
		{
			"numerator chain groups left",
			"set tot = divide n * x * y by z rounding by 0.5fp",
			"n x fp* y fp* z fphalfup/ cvfp /tot xdef",
		},
		{
			"explicit left grouping is unchanged",
			"set tot = divide n by (x * y) * z rounding by 0.5fp",
			"n x y fp* z fp* fphalfup/ cvfp /tot xdef",
		},
		{
			// The author asked for right grouping, so they get it. This is the
			// case that makes the fix a re-association of bare chains rather
			// than a blanket rewrite.
			"explicit right grouping is honoured",
			"set tot = divide n by x * (y * z) rounding by 0.5fp",
			"n x y z fp* fp* fphalfup/ cvfp /tot xdef",
		},
		{
			// Two operands have nothing to re-associate; guards the len<3 path.
			"two-operand divisor is unchanged",
			"set tot = divide n by x * y rounding by 0.5fp",
			"n x y fp* fphalfup/ cvfp /tot xdef",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := compileFixed(t, tc.dsl); got != tc.want {
				t.Errorf("\n dsl:  %s\n got:  %s\n want: %s", tc.dsl, got, tc.want)
			}
		})
	}
}

// TestOtherContextsStillGroupLeft is the control: every other construct was
// already correct, and the fix must not have touched them.
func TestOtherContextsStillGroupLeft(t *testing.T) {
	for dsl, want := range map[string]string{
		"set tot = x * y * z":       "x y fp* z fp* cvfp /tot xdef",
		"set tot = x - y - z":       "x y fp- z fp- cvfp /tot xdef",
		"set tot = x / y / z":       "x y fp/ z fp/ cvfp /tot xdef",
		"set tot = n + x * y * z":   "n x y fp* z fp* fp+ cvfp /tot xdef",
		"set tot = (x * y * z) + n": "x y fp* z fp* n fp+ cvfp /tot xdef",
	} {
		if got := compileFixed(t, dsl); got != want {
			t.Errorf("\n dsl:  %s\n got:  %s\n want: %s", dsl, got, want)
		}
	}
}

// TestStakingWeeklyBudgetPostfix pins the exact production expression that
// exposed this, against the postfix its repository has committed — which was
// compiled correctly before the regression. Reproducing it byte for byte is
// what makes an Excel-authored rebuild of that ruleset behaviour-neutral.
func TestStakingWeeklyBudgetPostfix(t *testing.T) {
	const dsl = "set weekly_budget = divide unissued_supply * (fixed) days_per_period_num * " +
		"(fixed) token_issuance_rate_num by (fixed) token_issuance_rate_den * days_per_year * " +
		"(fixed) days_per_period_den rounding by 0.5fp"
	const want = "unissued_supply days_per_period_num cvfp token_issuance_rate_num cvfp fp* fp* " +
		"token_issuance_rate_den cvfp days_per_year fp* days_per_period_den cvfp fp* " +
		"fphalfup/ cvfp /weekly_budget xdef"

	if got := compileFixed(t, dsl); got != want {
		t.Errorf("\n got:  %s\n want: %s", got, want)
	}
}
