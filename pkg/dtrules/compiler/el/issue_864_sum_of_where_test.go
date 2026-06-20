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

// Issue #864 (Accumulate staking): `sum of <expr> in <arr> where <bexpr>`
// is the predicated-sum parity fill for the existing `number of … where`.
//
// It lowers to the same forall-fold as `sum of … in …` but gates each
// element's contribution on the predicate:
//
//	0 <arr> { <bexpr> { <iexpr> + } if } forall
//
// Like plain `sum of`, the int alternative (intSumOfWhere) is the one
// ANTLR reaches — `number : iexpr | fexpr` lists iexpr first and both
// numeric types lex as a bare IDENT, so floatSumOfWhere stays a parallel
// zombie (see TestIssue803_FloatSumOf_Unreachable).
func TestIssue864_IntSumOfWhere(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())

	got, err := c.CompileAction(`set client.tax = sum of tax in family.kids where tax > 0`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// The predicated fold: a seed, a forall, the `+` accumulation, and an
	// `if` gating it on the predicate. The bare `sum of` form has no `if`.
	for _, tok := range []string{"0", "forall", "+", "if"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue864_FloatSumOfWhere_Unreachable pins that the float `where`
// alternative is a grammar zombie, for the same LL(*) reason as floatSumOf
// (number : iexpr | fexpr lists iexpr first; both numeric types lex as a bare
// IDENT). The IDENT form routes through intSumOfWhere (`+`), never the float
// visitor (`f+`). If a future token-classification pass makes the float path
// reachable, this breaks loudly so VisitFloatSumOfWhere gets exercised.
func TestIssue864_FloatSumOfWhere_Unreachable(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())

	got, err := c.CompileAction(`print sum of income in family.kids where income > 0.0`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(got, "f+") {
		t.Errorf("floatSumOfWhere is now reachable; replace this pin with positive float assertions. got: %s", got)
	}
	if !strings.Contains(got, "+") || !strings.Contains(got, "forall") || !strings.Contains(got, "if") {
		t.Errorf("expected intSumOfWhere fallback (+ forall if), got: %s", got)
	}
}

// The plain `sum of … in …` form must keep emitting an unconditional fold
// (no `if`), so the two forms stay distinct.
func TestIssue864_SumOfWithoutWhere_HasNoPredicate(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())

	got, err := c.CompileAction(`set client.tax = sum of tax in family.kids`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(got, "if") {
		t.Errorf("plain `sum of` should not gate on a predicate, got: %s", got)
	}
}
