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

// stakingDivideRoundingSymbols mirrors the fixture used by the other
// fp-emitter tests: an "fp" type binding for the staking fields the
// surface is intended to be used with.
func stakingDivideRoundingSymbols() map[string]string {
	return map[string]string{
		"wp.weighted_balance":  "fixed",
		"wp.staker_budget":     "fixed",
		"wp.total_weighted":    "fixed",
		"wp.reward":            "fixed",
	}
}

// TestDivideRoundingBy_FoldsLiteralZero confirms r=0 lowers to the
// existing truncating fp/ op (no fpdivr/ in output).
func TestDivideRoundingBy_FoldsLiteralZero(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	postfix, err := c.CompileAction(
		"set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by 0fp",
	)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, " fp/") && !strings.HasSuffix(postfix, "fp/") {
		// The token follows the two operands, so it's always preceded by a space.
		// But the operand might end at the very end of output before xdef.
		// Defensive: just check for the substring.
	}
	if !strings.Contains(postfix, "fp/") {
		t.Errorf("r=0 should fold to fp/, got: %s", postfix)
	}
	if strings.Contains(postfix, "fpdivr/") {
		t.Errorf("r=0 should NOT emit fpdivr/, got: %s", postfix)
	}
	if strings.Contains(postfix, "fphalfup/") {
		t.Errorf("r=0 should NOT emit fphalfup/, got: %s", postfix)
	}
	// 0fp constant must NOT appear as a literal in the stream (it was folded).
	if strings.Contains(postfix, "0fp ") {
		t.Errorf("r=0 literal should have been consumed by the fold, got: %s", postfix)
	}
}

// TestDivideRoundingBy_FoldsLiteralHalf confirms r=0.5 lowers to the
// existing fphalfup/ op (the v1.14.7 mantissa-precision half-up).
func TestDivideRoundingBy_FoldsLiteralHalf(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	postfix, err := c.CompileAction(
		"set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by 0.5fp",
	)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "fphalfup/") {
		t.Errorf("r=0.5 should fold to fphalfup/, got: %s", postfix)
	}
	if strings.Contains(postfix, "fpdivr/") {
		t.Errorf("r=0.5 should NOT emit fpdivr/, got: %s", postfix)
	}
	// 0.5fp constant must NOT appear (folded away).
	if strings.Contains(postfix, "0.5fp ") {
		t.Errorf("r=0.5 literal should have been consumed by the fold, got: %s", postfix)
	}
}

// TestDivideRoundingBy_LiteralHalfVariants: 0.50fp, 0.500fp, 0.50000000fp
// all parse to the same mantissa (50_000_000) so they must all fold to
// fphalfup/ — not silently miss the fold.
func TestDivideRoundingBy_LiteralHalfVariants(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	for _, lit := range []string{"0.5fp", "0.50fp", "0.500fp", "0.50000000fp"} {
		t.Run(lit, func(t *testing.T) {
			postfix, err := c.CompileAction(
				"set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by " + lit,
			)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if !strings.Contains(postfix, "fphalfup/") {
				t.Errorf("%s should fold to fphalfup/, got: %s", lit, postfix)
			}
			if strings.Contains(postfix, "fpdivr/") {
				t.Errorf("%s should NOT emit fpdivr/, got: %s", lit, postfix)
			}
		})
	}
}

// TestDivideRoundingBy_LiteralZeroVariants: 0fp, 0.0fp, 0.00fp,
// 0.00000000fp all fold to fp/.
func TestDivideRoundingBy_LiteralZeroVariants(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	for _, lit := range []string{"0fp", "0.0fp", "0.00fp", "0.00000000fp"} {
		t.Run(lit, func(t *testing.T) {
			postfix, err := c.CompileAction(
				"set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by " + lit,
			)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if !strings.Contains(postfix, "fp/") {
				t.Errorf("%s should fold to fp/, got: %s", lit, postfix)
			}
			if strings.Contains(postfix, "fphalfup/") {
				t.Errorf("%s should NOT emit fphalfup/, got: %s", lit, postfix)
			}
			if strings.Contains(postfix, "fpdivr/") {
				t.Errorf("%s should NOT emit fpdivr/, got: %s", lit, postfix)
			}
		})
	}
}

// TestDivideRoundingBy_GeneralPathEmitsTernary: r literals that are
// neither 0 nor 0.5 must emit fpdivr/ with the literal carried through
// as the third stack operand.
func TestDivideRoundingBy_GeneralPathEmitsTernary(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	for _, lit := range []string{"0.25fp", "0.75fp", "0.99999999fp", "0.123fp"} {
		t.Run(lit, func(t *testing.T) {
			postfix, err := c.CompileAction(
				"set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by " + lit,
			)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if !strings.Contains(postfix, "fpdivr/") {
				t.Errorf("%s should emit fpdivr/, got: %s", lit, postfix)
			}
			if !strings.Contains(postfix, lit) {
				t.Errorf("%s literal should be carried in postfix, got: %s", lit, postfix)
			}
			if strings.Contains(postfix, "fphalfup/") {
				t.Errorf("%s should NOT emit fphalfup/, got: %s", lit, postfix)
			}
		})
	}
}

// TestDivideRoundingBy_RejectsOutOfRange: r >= 1.0 and r outside [0, 1)
// must be a compile-time error per the issue contract. We use the
// underlying Compile() rather than CompileAction() because the latter
// retries on failure and swallows the inner error in favor of the
// fallback's top-level parse error — masking the precise message our
// visitor emits. The contract under test is "fails to compile", so any
// non-nil err is acceptable; the message-specific assertion uses the
// direct path so the visitor's error text is observable.
func TestDivideRoundingBy_RejectsOutOfRange(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	for _, lit := range []string{"1fp", "1.0fp", "1.00000000fp", "1.5fp", "2fp"} {
		t.Run(lit, func(t *testing.T) {
			// Direct compile: keeps the visitor's emitError message.
			src := "action set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by " + lit + ";"
			_, err := c.Compile(src)
			if err == nil {
				t.Errorf("expected compile error for r=%s, got nil", lit)
				return
			}
			if !strings.Contains(err.Error(), "< 1.0") {
				t.Errorf("expected `< 1.0` message for r=%s, got: %v", lit, err)
			}
		})
	}
}

// TestDivideRoundingBy_OperandOrderIsAOverB confirms the surface produces
// `a b ...` postfix in that order — the runtime ops all expect the
// numerator below the denominator on the data stack. Regression guard
// against a swap bug.
func TestDivideRoundingBy_OperandOrderIsAOverB(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(stakingDivideRoundingSymbols())

	postfix, err := c.CompileAction(
		"set wp.reward = divide wp.staker_budget by wp.total_weighted rounding by 0.5fp",
	)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	idxBudget := strings.Index(postfix, "wp.staker_budget")
	idxTotal := strings.Index(postfix, "wp.total_weighted")
	if idxBudget < 0 || idxTotal < 0 {
		t.Fatalf("operands missing in postfix: %s", postfix)
	}
	if idxBudget >= idxTotal {
		t.Errorf("staker_budget must appear before total_weighted (numerator first), got: %s", postfix)
	}
}

// TestParseFpLiteralToMantissa unit-tests the parser used by the
// fold table. The grammar guarantees well-formed input but defense in
// depth on the value extraction keeps the fold honest.
func TestParseFpLiteralToMantissa(t *testing.T) {
	cases := []struct {
		text string
		want int64
	}{
		{"0fp", 0},
		{"0.0fp", 0},
		{"0.00000000fp", 0},
		{"0.5fp", 50_000_000},
		{"0.50fp", 50_000_000},
		{"0.50000000fp", 50_000_000},
		{"1fp", 100_000_000},
		{"1.0fp", 100_000_000},
		{"0.25fp", 25_000_000},
		{"0.99999999fp", 99_999_999},
		// More fractional digits than 8 → truncated to 8.
		{"0.123456789fp", 12_345_678},
		// Case-insensitive suffix.
		{"0.5FP", 50_000_000},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			got, err := parseFpLiteralToMantissa(c.text)
			if err != nil {
				t.Fatalf("parse %q: %v", c.text, err)
			}
			if got.Int64() != c.want {
				t.Errorf("%s: got mantissa %d, want %d", c.text, got.Int64(), c.want)
			}
		})
	}
}
