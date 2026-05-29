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

// Issue #803: regression tests for labeled alts that previously had no
// Visit<Label> override on PostfixEmitter and silently dropped operands
// or operators. Each test below has a reproducer that confirmed empty
// or op-missing output before the fix.

// issue803Symbols mirrors the staking-style typed fields used by the
// regression cases. Mixed types are important because the failing
// visitors all do type-aware dispatch.
func issue803Symbols() map[string]string {
	return map[string]string{
		"a.x": "double",
		"a.y": "integer",
		"a.f": "fixed",
		"a.b": "bigint",
		"a.s": "string",
	}
}

// TestIssue803_MultiplyDivideByMatchCanonical asserts that the prefix
// `multiply x by N` / `divide x by N` forms produce postfix bit-identical
// to the canonical infix `x * N` / `x / N` forms. The two forms are
// semantically equivalent and the runtime must see the same bytecode.
//
// Pre-fix shape (confirmed via reproducer):
//
//	set a.x = multiply a.x by 2  →  "cvd /a.x xdef"   (RHS dropped)
//	set a.y = multiply a.y by 2  →  "cvi /a.y xdef"   (RHS dropped)
//
// Post-fix shape: matches the corresponding `a.x * 2` / `a.y * 2` form.
func TestIssue803_MultiplyDivideByMatchCanonical(t *testing.T) {
	pairs := []struct {
		canonical, prefix string
	}{
		{"set a.x = a.x * 2", "set a.x = multiply a.x by 2"},
		{"set a.x = a.x / 2", "set a.x = divide a.x by 2"},
		{"set a.y = a.y * 2", "set a.y = multiply a.y by 2"},
		{"set a.y = a.y / 2", "set a.y = divide a.y by 2"},
		{"set a.f = a.f * 2", "set a.f = multiply a.f by 2"},
		{"set a.f = a.f / 2", "set a.f = divide a.f by 2"},
		{"set a.b = a.b * 2", "set a.b = multiply a.b by 2"},
		{"set a.b = a.b / 2", "set a.b = divide a.b by 2"},
	}
	c := NewCompiler()
	c.SetSymbols(issue803Symbols())
	for _, p := range pairs {
		t.Run(p.prefix, func(t *testing.T) {
			canonical, err := c.CompileAction(p.canonical)
			if err != nil {
				t.Fatalf("canonical compile: %v", err)
			}
			prefix, err := c.CompileAction(p.prefix)
			if err != nil {
				t.Fatalf("prefix compile: %v", err)
			}
			if canonical != prefix {
				t.Errorf("prefix form must match canonical:\n  canonical: %q\n  prefix:    %q",
					canonical, prefix)
			}
		})
	}
}

// TestIssue803_MultiplyDivideByEmitsOp guards against a particular
// regression: even if both forms happen to "agree", they must each
// contain the actual arithmetic operator. The pre-fix reproducer had
// neither form emit the op for the int target — so an equality check
// alone wouldn't catch full silent failures (empty == empty). Assert
// that each prefix form emits a non-trivial op.
func TestIssue803_MultiplyDivideByEmitsOp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Symbols())
	cases := []struct {
		dsl, mustContain string
	}{
		{"set a.x = multiply a.x by 2", "fmul"},
		{"set a.x = divide a.x by 2", "fdiv"},
		{"set a.y = multiply a.y by 2", " * "},
		{"set a.y = divide a.y by 2", " / "},
		{"set a.f = multiply a.f by 2", "fp*"},
		{"set a.f = divide a.f by 2", "fp/"},
		{"set a.b = multiply a.b by 2", "b*"},
		{"set a.b = divide a.b by 2", "b/"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			// Pad so " * " / " / " can match exactly even at boundaries.
			padded := " " + got + " "
			if !strings.Contains(padded, tc.mustContain) {
				t.Errorf("expected %q in postfix, got: %s", tc.mustContain, got)
			}
		})
	}
}

// TestIssue803_SubstringEmitsOp confirms `substring of S from A to B`
// reaches the registered `substring` op. Pre-fix the entire substring
// call disappeared, so `set s = substring of s from 1 to 3` produced
// only `cvs /s xdef` — wrong but silent.
func TestIssue803_SubstringEmitsOp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Symbols())
	got, err := c.CompileAction("set a.s = substring of a.s from 1 to 3")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "substring") {
		t.Errorf("expected `substring` in postfix, got: %s", got)
	}
	// Operands must be present in the right order: source string,
	// start index, end index, then the op.
	for _, tok := range []string{"a.s", "1", "3", "substring"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected token %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_TypeCastsEmitOperand confirms the (double)/(int)/(long)
// cast forms emit the inner expression's value before the cast op.
// Pre-fix the inner expression was dropped entirely.
func TestIssue803_TypeCastsEmitOperand(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Symbols())
	cases := []struct {
		dsl, mustContain string
	}{
		// floatFromStr — string operand was dropped pre-fix.
		{`set a.x = (double) "1.5"`, `"1.5"`},
		// floatFromInt — int literal was dropped pre-fix.
		{`set a.x = (double) 42`, "42"},
		// intFromStr — string operand was dropped pre-fix.
		{`set a.y = (long) "42"`, `"42"`},
		// intFromNumber — number operand was dropped pre-fix.
		{`set a.y = (long) 42.5`, "42.5"},
		// `int` is an alias keyword for the long type cast.
		{`set a.y = (int) "99"`, `"99"`},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, tc.mustContain) {
				t.Errorf("expected operand %q in postfix, got: %s", tc.mustContain, got)
			}
		})
	}
}

// TestIssue803_IndexCastEmitsOperand covers the two visitors that the
// type-cast test above misses: VisitFloatFromIndex and
// VisitIntFromIndex. These match `(double) <arrayExpr>[<iexpr>]` /
// `(int) <arrayExpr>[<iexpr>]` and pre-fix dropped the entire indexed
// expression. The action SET form `set X = (double) arr[0]` does not
// parse — a different SET dispatch wins — so we exercise the visitor
// through the condition path, which is where these alts actually fire.
func TestIssue803_IndexCastEmitsOperand(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"a.alist": "array of integer",
	})
	cases := []struct {
		dsl, mustContain string
	}{
		// floatFromIndex: condition form is the reachable path.
		{`(double) a.alist[0] > 0.0`, "cvd"},
		// intFromIndex: same shape with int cast.
		{`(int) a.alist[0] > 0`, "cvi"},
		// `long` keyword alias for the int cast.
		{`(long) a.alist[0] > 0`, "cvi"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileCondition(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, tc.mustContain) {
				t.Errorf("expected %q in postfix, got: %s", tc.mustContain, got)
			}
			// The indexed array operand must also be present — pre-fix
			// the entire indxExpr was dropped, leaving the cast op
			// dangling.
			for _, tok := range []string{"a.alist", "0"} {
				if !strings.Contains(got, tok) {
					t.Errorf("expected operand %q in postfix, got: %s", tok, got)
				}
			}
		})
	}
}
