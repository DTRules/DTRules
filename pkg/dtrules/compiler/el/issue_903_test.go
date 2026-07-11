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

// Issue #903: the fexpr mul/div visitors emitted fmul/fdiv unconditionally,
// so the top-level multiply of a `divide … rounding by` dividend degraded to
// double math even for fixed×fixed operands — while the identical expression
// in a plain `set` dispatched to fp* correctly. Same class as #874/#884:
// fp dispatch missed in one emitter context.

func issue903Symbols() map[string]string {
	return map[string]string{
		"weekly_budget":           "fixed",
		"rate_fixed":              "fixed",
		"withholding_denom":       "fixed",
		"withholding_rate_num":    "integer",
		"staker_budget":           "fixed",
		"unissued_supply":         "fixed",
		"token_issuance_rate_num": "integer",
		"budget_denom":            "fixed",
		"dbl":                     "double",
	}
}

// TestIssue903_DividendMultiplyDispatchesFp covers the four repro shapes from
// the issue: plain product dividend, cast operand, parenthesized dividend, and
// the three-factor dividend whose OUTER multiply used to emit fmul while the
// inner one got fp*.
func TestIssue903_DividendMultiplyDispatchesFp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue903Symbols())

	cases := []struct {
		name string
		src  string
	}{
		{"plain-product", "set staker_budget = divide weekly_budget * rate_fixed by withholding_denom rounding by 0.5fp"},
		{"cast-operand", "set staker_budget = divide weekly_budget * (fixed) withholding_rate_num by withholding_denom rounding by 0.5fp"},
		{"parenthesized", "set staker_budget = divide (weekly_budget * rate_fixed) by withholding_denom rounding by 0.5fp"},
		{"three-factor", "set staker_budget = divide unissued_supply * 7fp * (fixed) token_issuance_rate_num by budget_denom rounding by 0.5fp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pf, err := c.CompileAction(tc.src)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if strings.Contains(pf, "fmul") {
				t.Errorf("dividend multiply degraded to fmul (double): %s", pf)
			}
			if !strings.Contains(pf, "fp*") {
				t.Errorf("expected fp* dispatch for fixed operands, got: %s", pf)
			}
			if !strings.Contains(pf, "fphalfup/") {
				t.Errorf("expected fphalfup/ fold, got: %s", pf)
			}
		})
	}
}

// TestIssue903_ThreeFactorAllMultipliesFp: the three-factor dividend must fp*
// BOTH multiplies (the outer one used to be the stray fmul).
func TestIssue903_ThreeFactorAllMultipliesFp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue903Symbols())

	pf, err := c.CompileAction(
		"set staker_budget = divide unissued_supply * 7fp * (fixed) token_issuance_rate_num by budget_denom rounding by 0.5fp")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if got := strings.Count(pf, "fp*"); got != 2 {
		t.Errorf("expected 2 fp* multiplies, got %d in: %s", got, pf)
	}
}

// TestIssue903_PlainFexprArithAlsoFixed: the fix is in the shared fexpr
// mul/div visitors, so fixed×fixed dispatches to fp-family ops in any fexpr
// context, not just inside divide-rounding-by. A genuinely double operand
// still dispatches to the double ops.
func TestIssue903_PlainFexprArithAlsoFixed(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue903Symbols())

	// fixed * fp-literal parses through an fexpr multiply alternative.
	pf, err := c.CompileAction("set staker_budget = weekly_budget * 7fp")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if strings.Contains(pf, "fmul") || !strings.Contains(pf, "fp*") {
		t.Errorf("fixed * fp-literal should dispatch fp*, got: %s", pf)
	}

	// double * double stays on the double op.
	pf, err = c.CompileAction("set dbl = dbl * 2.0")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(pf, "fmul") {
		t.Errorf("double * double should still dispatch fmul, got: %s", pf)
	}
}

// TestIssue903_IntegerOperandsPromoteToFixed: divide-rounding-by promotes
// integer/bigint operands to fixed via cvfp — the fp-family divide ops
// require fixed operands.
func TestIssue903_IntegerOperandsPromoteToFixed(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue903Symbols())

	pf, err := c.CompileAction(
		"set staker_budget = divide withholding_rate_num by withholding_denom rounding by 0.5fp")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(pf, "cvfp") {
		t.Errorf("integer dividend should promote via cvfp, got: %s", pf)
	}
	if !strings.Contains(pf, "fphalfup/") {
		t.Errorf("expected fphalfup/, got: %s", pf)
	}
}

// TestIssue903_DoubleOperandRejected: a double operand in divide-rounding-by
// is a compile error (per the #876 no-implicit-double-promotion policy) —
// authors must cast explicitly.
func TestIssue903_DoubleOperandRejected(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue903Symbols())

	src := "action set staker_budget = divide dbl by withholding_denom rounding by 0.5fp;"
	_, err := c.Compile(src)
	if err == nil {
		t.Fatal("expected compile error for double operand, got nil")
	}
	if !strings.Contains(err.Error(), "fixed operands") {
		t.Errorf("expected fixed-operands message, got: %v", err)
	}
}
