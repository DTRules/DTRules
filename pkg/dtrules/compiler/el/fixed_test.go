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

// TestFixedComparisonOperators_AllEmitFpOps covers the full comparison
// family. The `==` / `!=` cases specifically exercise the BoolName*
// numeric-dispatch path (the grammar routes `field CMP field` through the
// nexpr visitors regardless of type); `<`, `<=`, `>`, `>=` exercise the
// iexpr-routed BoolInt* visitors. A dispatch typo on any of them would
// fail.
func TestFixedComparisonOperators_AllEmitFpOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingFixedSymbols())

	cases := []struct {
		dsl  string
		want string
	}{
		{"wp.reward_per_block == wp.staker_balance", "fp=="},
		{"wp.reward_per_block != wp.staker_balance", "fp!="},
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
			// The string-compare fallback must not leak in — that was the
			// pre-fix behavior BoolName{Eq,Neq} had for numeric operands.
			if strings.Contains(postfix, "streq") {
				t.Errorf("unexpected streq for numeric compare: %s", postfix)
			}
		})
	}
}

// TestBigIntComparison_NameDispatch guards that the numeric-name dispatch
// also lights up for pure bigint comparisons — previously bigint == bigint
// via two field refs fell through to streq (documented in the pre-fix
// TestBigIntComparisonWithLocalVariables).
func TestBigIntComparison_NameDispatch(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"bp.a": "bigint",
		"bp.b": "bigint",
	})
	cases := []struct {
		dsl, want string
	}{
		{"bp.a == bp.b", "b=="},
		{"bp.a != bp.b", "b!="},
	}
	for _, c2 := range cases {
		t.Run(c2.want, func(t *testing.T) {
			postfix, err := c.CompileCondition(c2.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", c2.dsl, err)
			}
			if !strings.Contains(postfix, c2.want) {
				t.Errorf("expected %s, got: %s", c2.want, postfix)
			}
			if strings.Contains(postfix, "streq") {
				t.Errorf("bigint compare must not emit streq: %s", postfix)
			}
		})
	}
}

// TestMixedTypeComparison_PromotesToWidest verifies that a mixed numeric
// comparison on the BoolName path picks the widest type and emits the
// right cast on the narrower side.
func TestMixedTypeComparison_PromotesToWidest(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"wp.amount": "fixed",
		"ap.count":  "integer",
	})
	postfix, err := c.CompileCondition("wp.amount == ap.count")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected cvfp promotion in postfix, got: %s", postfix)
	}
	if !strings.Contains(postfix, "fp==") {
		t.Errorf("expected fp== after promotion, got: %s", postfix)
	}
}

// TestFixedLiteral_InIntContext_PreservesPrecision is the reproducer for the
// most severe precision bug found in the deep review: when an fp-suffixed
// literal appears alongside an integer operand, the grammar routes the
// expression through iexpr (not fexpr), and the emitter's type inference
// previously read the literal as an integer. The compiled postfix then used
// plain `+` / `*`, and at runtime the RFixed's LongValue truncated the
// fractional part silently. Now the emitter sees the `fp` suffix on the
// literal's raw text and promotes the integer side via cvfp.
func TestFixedLiteral_InIntContext_PreservesPrecision(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"ap.count": "integer",
		"wp.total": "fixed",
	})
	postfix, err := c.CompileAction("set wp.total = ap.count + 1.5fp")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected integer side promoted via cvfp, got: %s", postfix)
	}
	if !strings.Contains(postfix, "fp+") {
		t.Errorf("expected fp+ (not plain +), got: %s", postfix)
	}
	// The old buggy emission would have a lone ` + ` between ap.count and
	// 1.5fp — guard against regression.
	if strings.Contains(postfix, "count 1.5fp +") {
		t.Errorf("plain `+` between int and fp literal would silently truncate: %s", postfix)
	}
}

// TestFixedLiteral_PureFpArithmetic_UsesFpOps covers the case where both
// operands are fp literals — same root cause as the mixed case, different
// trigger. Without the fix, `1.5fp + 2.5fp` compiled to plain `+` and lost
// the fractional parts of both operands.
func TestFixedLiteral_PureFpArithmetic_UsesFpOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"wp.total": "fixed"})
	cases := []struct {
		dsl, want string
	}{
		{"set wp.total = 1.5fp + 2.5fp", "fp+"},
		{"set wp.total = 1.5fp * 2.0fp", "fp*"},
		{"set wp.total = 3.75fp - 1.5fp", "fp-"},
		{"set wp.total = 1.5fp / 3.0fp", "fp/"},
	}
	for _, c2 := range cases {
		t.Run(c2.want, func(t *testing.T) {
			postfix, err := c.CompileAction(c2.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", c2.dsl, err)
			}
			if !strings.Contains(postfix, c2.want) {
				t.Errorf("expected %s, got: %s", c2.want, postfix)
			}
		})
	}
}

// TestIntegerNegation_EmitsNegate guards the pre-existing bug where
// VisitIntNegate emitted `neg` — an operator name that was never
// registered — so every rule doing `-<int_expr>` failed at runtime.
// The fix was at the root: the emitter now emits `negate`, the operator
// that actually exists. This test asserts the emission, not a runtime
// alias, so any regression back to `neg` is caught at compile time.
func TestIntegerNegation_EmitsNegate(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"ap.count": "integer"})
	postfix, err := c.CompileAction("set ap.count = - ap.count")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "negate") {
		t.Errorf("expected `negate` for integer negation, got: %s", postfix)
	}
	// Pin the regression: the old buggy emission would have a lone ` neg`.
	if strings.Contains(postfix, " neg ") || strings.HasSuffix(postfix, " neg") {
		t.Errorf("emitter regressed to unregistered `neg`: %s", postfix)
	}
}

// TestFixedIncrement_PreservesPrecision guards the silent-truncation bug in
// VisitIncrementLong: `increment <field>` used to emit `field 1 + /field
// xdef` regardless of the field's declared type. For fp fields, the plain
// `+` truncates via LongValue(). Now the visitor dispatches on the field's
// EDD type and emits fp+ / b+ with the appropriate cast on the `1`.
func TestFixedIncrement_PreservesPrecision(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"wp.amount": "fixed"})
	postfix, err := c.CompileAction("increment wp.amount")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "fp+") {
		t.Errorf("expected fp+ for fp-field increment, got: %s", postfix)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected cvfp on the `1` literal, got: %s", postfix)
	}
	// Regression pin: old buggy emission ended with `1 +` before xdef.
	if strings.Contains(postfix, "1 + /") {
		t.Errorf("plain + on fp increment would silently truncate: %s", postfix)
	}
}

// TestFixedDecrement_PreservesPrecision — parallel to increment.
func TestFixedDecrement_PreservesPrecision(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"wp.amount": "fixed"})
	postfix, err := c.CompileAction("decrement wp.amount")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "fp-") {
		t.Errorf("expected fp- for fp-field decrement, got: %s", postfix)
	}
	if strings.Contains(postfix, "1 - /") {
		t.Errorf("plain - on fp decrement would silently truncate: %s", postfix)
	}
}

// TestFixedSubtractFrom_PreservesPrecision guards VisitSubDestLong: the
// `subtract <value> from <field>` pattern compiled to `field swap -`
// regardless of target type. For fp fields, `-` truncates. Now the visitor
// emits `field swap cvfp fp-` so integer RHS values are promoted and the
// subtract happens at fp precision.
func TestFixedSubtractFrom_PreservesPrecision(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"wp.amount": "fixed"})
	cases := []struct {
		dsl, wantOp string
	}{
		{"subtract 1.5fp from wp.amount", "fp-"}, // fp literal RHS
		{"subtract 1 from wp.amount", "fp-"},     // integer RHS — must promote
	}
	for _, c2 := range cases {
		t.Run(c2.dsl, func(t *testing.T) {
			postfix, err := c.CompileAction(c2.dsl)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if !strings.Contains(postfix, c2.wantOp) {
				t.Errorf("expected %s in postfix, got: %s", c2.wantOp, postfix)
			}
			if !strings.Contains(postfix, "cvfp") {
				t.Errorf("expected cvfp cast in postfix, got: %s", postfix)
			}
			// Plain `-` on the fp field is the bug we're guarding.
			if strings.Contains(postfix, "swap - /") {
				t.Errorf("plain swap - on fp field would silently truncate: %s", postfix)
			}
		})
	}
}

// TestIntegerIncrement_UnchangedByFixedDispatch guards that plain integer
// fields still emit the historic `field 1 + /field xdef` pattern after the
// fp / bigint dispatch was added to VisitIncrementLong.
func TestIntegerIncrement_UnchangedByFixedDispatch(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"ap.count": "integer"})
	postfix, err := c.CompileAction("increment ap.count")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if strings.Contains(postfix, "fp+") || strings.Contains(postfix, "b+") ||
		strings.Contains(postfix, "cvfp") || strings.Contains(postfix, "cvbi") {
		t.Errorf("pure-int increment must not emit fp/b ops: %s", postfix)
	}
}

// TestStringComparison_UnchangedByNumericDispatch is a regression guard on
// BoolName{Eq,Neq}: string-vs-string name compares must still land on the
// existing streq/`streq not` fallback. The numeric-dispatch branch is only
// meant to intercept when BOTH operands are declared numeric.
func TestStringComparison_UnchangedByNumericDispatch(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"p.first_name": "string",
		"p.last_name":  "string",
	})
	postfix, err := c.CompileCondition("p.first_name != p.last_name")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "streq") {
		t.Errorf("string != must keep emitting streq: %s", postfix)
	}
	if strings.Contains(postfix, "fp!=") || strings.Contains(postfix, "b!=") {
		t.Errorf("string compare must not emit numeric ops: %s", postfix)
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
