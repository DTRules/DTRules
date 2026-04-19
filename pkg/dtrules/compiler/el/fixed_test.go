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
