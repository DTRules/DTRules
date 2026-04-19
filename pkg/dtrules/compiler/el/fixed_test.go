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

func stakingFixedSymbols() map[string]string {
	return map[string]string{
		"wp.reward_per_block": "fixed",
		"wp.staker_balance":   "fixed",
		"wp.weekly_reward":    "fixed",
		"wp.periods":          "integer",
		"bp.treasury_fee":     "bigint",
	}
}

// TestFixedArithmetic_EmitsFpOps verifies that when both operands are fixed,
// the emitter picks fp-family operators.
func TestFixedArithmetic_EmitsFpOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())

	postfix, err := c.CompileAction("set wp.weekly_reward = wp.reward_per_block * wp.staker_balance")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "fp*") {
		t.Errorf("expected fp* in postfix, got: %s", postfix)
	}
	if strings.Contains(postfix, " * ") || strings.HasSuffix(postfix, " *") {
		t.Errorf("unexpected plain '*' in fixed postfix: %s", postfix)
	}
}

// TestFixedMixedWithInteger_PromotesViaCvfp verifies that an integer operand
// mixed with a fixed operand is promoted via cvfp before the fp op.
func TestFixedMixedWithInteger_PromotesViaCvfp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())

	postfix, err := c.CompileAction("set wp.weekly_reward = wp.reward_per_block * wp.periods")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected cvfp promotion in postfix, got: %s", postfix)
	}
	if !strings.Contains(postfix, "fp*") {
		t.Errorf("expected fp* operator, got: %s", postfix)
	}
}

// TestFixedMixedWithBigInt_PromotesViaCvfp verifies that a bigint operand
// mixed with a fixed operand is promoted via cvfp before the fp op.
func TestFixedMixedWithBigInt_PromotesViaCvfp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())

	postfix, err := c.CompileAction("set wp.weekly_reward = wp.reward_per_block + bp.treasury_fee")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected cvfp promotion in postfix, got: %s", postfix)
	}
	if !strings.Contains(postfix, "fp+") {
		t.Errorf("expected fp+ operator, got: %s", postfix)
	}
	// The plain 'b+' must NOT appear — fixed wins over bigint.
	if strings.Contains(postfix, " b+") {
		t.Errorf("unexpected b+ when fixed operand is present: %s", postfix)
	}
}

// TestFixedComparison_EmitsFpCompare verifies that a comparison between fixed
// operands emits fp-family comparison ops.
func TestFixedComparison_EmitsFpCompare(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())

	// Conditions are compiled via CompileCondition.
	postfix, err := c.CompileCondition("wp.reward_per_block > wp.staker_balance")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "fp>") {
		t.Errorf("expected fp> in postfix, got: %s", postfix)
	}
}

// TestFixedNegation_EmitsFpNegate verifies that negating a fixed-typed
// expression emits fpnegate, not the integer `neg` or bigint `bnegate`.
func TestFixedNegation_EmitsFpNegate(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())
	postfix, err := c.CompileAction("set wp.weekly_reward = - wp.reward_per_block")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "fpnegate") {
		t.Errorf("expected fpnegate in postfix, got: %s", postfix)
	}
	if strings.Contains(postfix, " neg ") || strings.HasSuffix(postfix, " neg") {
		t.Errorf("unexpected integer 'neg' in fp negation: %s", postfix)
	}
	if strings.Contains(postfix, "bnegate") {
		t.Errorf("unexpected 'bnegate' in pure-fixed negation: %s", postfix)
	}
}

// TestFixedComparisonOperators exercises the full comparison family on the
// VisitBoolInt* visitors that my refactor touched. `>=` and `<=` are the
// paths this test is specifically guarding — they share the same dispatch
// code as `>` and `<` (already covered) but route to distinct operators.
//
// Note: the grammar routes `!=` between two plain field refs to a string-
// compare path (`streq not`), not BoolIntNeq, so that specific combination
// is a separate concern tracked as a follow-up.
func TestFixedComparisonOperators_AllEmitFpOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())

	cases := []struct {
		dsl  string
		want string
	}{
		{"wp.reward_per_block > wp.staker_balance", "fp>"},
		{"wp.reward_per_block >= wp.staker_balance", "fp>="},
		{"wp.reward_per_block < wp.staker_balance", "fp<"},
		{"wp.reward_per_block <= wp.staker_balance", "fp<="},
	}
	for _, c2 := range cases {
		t.Run(c2.want, func(t *testing.T) {
			postfix, err := c.CompileCondition(c2.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", c2.dsl, err)
			}
			if !strings.Contains(postfix, c2.want) {
				t.Errorf("expected %s in postfix, got: %s", c2.want, postfix)
			}
		})
	}
}

// TestIntegerNotEqual_UnchangedByFixedRefactor is a regression guard on my
// VisitBoolIntNeq refactor: when both operands are plain integers, the
// emitter must still fall back to the historic `== not` pattern, not
// accidentally emit `fp!=` or `b!=`.
func TestIntegerNotEqual_UnchangedByFixedRefactor(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"ap.count":  "integer",
		"ap.amount": "integer",
	})
	postfix, err := c.CompileCondition("ap.count != ap.amount")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if strings.Contains(postfix, "fp!=") || strings.Contains(postfix, "b!=") {
		t.Errorf("pure-int != must not emit fp!=/b!=: %s", postfix)
	}
}

// TestIntegerArithmetic_UnchangedByFixedRefactor is the symmetric regression
// guard to TestBigIntArithmetic_UnchangedByFixedRefactor — pure int+int must
// still emit plain `+`, not accidentally promote to fp+ or b+.
func TestIntegerArithmetic_UnchangedByFixedRefactor(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"ap.count":  "integer",
		"ap.amount": "integer",
		"ap.total":  "integer",
	})
	postfix, err := c.CompileAction("set ap.total = ap.count + ap.amount")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, " +") {
		t.Errorf("expected plain `+` for int+int, got: %s", postfix)
	}
	if strings.Contains(postfix, "fp+") || strings.Contains(postfix, "b+") {
		t.Errorf("pure-int expression must not emit fp+/b+: %s", postfix)
	}
	if strings.Contains(postfix, "cvfp") || strings.Contains(postfix, "cvbi") {
		t.Errorf("pure-int expression must not emit promotion casts: %s", postfix)
	}
}

// TestBigIntArithmetic_UnchangedByFixedRefactor is a regression guard: when
// no operand is fixed, bigint arithmetic must still emit the b-family ops.
func TestBigIntArithmetic_UnchangedByFixedRefactor(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"bp.a": "bigint",
		"bp.b": "bigint",
		"bp.c": "bigint",
	})
	postfix, err := c.CompileAction("set bp.c = bp.a * bp.b")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "b*") {
		t.Errorf("expected b*, got: %s", postfix)
	}
	if strings.Contains(postfix, "fp") {
		t.Errorf("unexpected fp in pure-bigint postfix: %s", postfix)
	}
}
