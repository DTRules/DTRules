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

// Issue #803 batch 2: date arithmetic expressions in `set` actions.
// Pre-fix, every `set <date-field> = <complex-dexpr>` produced empty
// postfix because:
//
//  1. ANTLR matched the SET alt as `setStringFromDate` (which had no
//     override) instead of `setDate` (which did) — leftStrexpr accepts
//     IDENT before leftDexpr gets a chance, and adaptive prediction
//     picks the earlier alt. This made `setDate` dead grammar.
//  2. Even when the dispatch reached a dexpr alt like `dateExprAddDays`,
//     that alt had no override either — antlr's BaseParseTreeVisitor.
//     VisitChildren is a no-op so nothing got emitted.
//
// Fix: VisitSetStringFromDate (the actual entry point) plus the 9 dexpr
// alts: dateExprAddDays/Months/Years, dateExprSubDays/Months/Years,
// dateFirstOfMonth, dateFirstOfYear, dateEndOfMonth.

func issue803Batch2Symbols() map[string]string {
	return map[string]string{
		"a.d": "date",
		"a.s": "string",
	}
}

// TestIssue803_DateExprArithmeticEmitsOp confirms each of the 9 dexpr
// alternatives produces postfix with the expected op.
func TestIssue803_DateExprArithmeticEmitsOp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch2Symbols())

	cases := []struct {
		dsl, mustContain string
	}{
		// dateExprAdd* family
		{"set a.d = add 3 days to a.d", "adddays"},
		{"set a.d = add 3 months to a.d", "addmonths"},
		{"set a.d = add 3 years to a.d", "addyears"},
		// dateExprSub* family — emits `negate <addop>` because no
		// subdays/submonths/subyears ops are registered.
		{"set a.d = subtract 3 days from a.d", "negate adddays"},
		{"set a.d = subtract 3 months from a.d", "negate addmonths"},
		{"set a.d = subtract 3 years from a.d", "negate addyears"},
		// dateFirst*/dateEndOf*
		{"set a.d = first of months of a.d", "firstofmonth"},
		{"set a.d = first of years of a.d", "firstofyear"},
		{"set a.d = end of months of a.d", "endofmonth"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, tc.mustContain) {
				t.Errorf("expected %q in postfix, got: %s", tc.mustContain, got)
			}
			// The assignment trailer must also be present — empty RHS
			// before the fix meant the entire emission was suppressed.
			for _, tok := range []string{"cvdate", "/a.d", "xdef"} {
				if !strings.Contains(got, tok) {
					t.Errorf("expected trailer token %q in postfix, got: %s", tok, got)
				}
			}
		})
	}
}

// TestIssue803_SetStringFromDateHonorsTrueLHSType confirms the
// SetStringFromDate alt — which is the actual ANTLR pick for both
// `set <date-field> = <dexpr>` and `set <string-field> = <dexpr>` — uses
// resolveSetTarget so the trailer matches the LHS's *declared* type:
//
//	date target   → cvdate trailer (date assignment)
//	string target → cvs trailer    (stringified date)
func TestIssue803_SetStringFromDateHonorsTrueLHSType(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch2Symbols())

	// Date target: must use cvdate trailer.
	got, err := c.CompileAction("set a.d = add 1 days to a.d")
	if err != nil {
		t.Fatalf("compile date target: %v", err)
	}
	if !strings.Contains(got, "cvdate") {
		t.Errorf("date target should emit cvdate, got: %s", got)
	}

	// String target: must use cvs trailer.
	got, err = c.CompileAction("set a.s = a.d")
	if err != nil {
		t.Fatalf("compile string target: %v", err)
	}
	if !strings.Contains(got, "cvs") {
		t.Errorf("string target should emit cvs trailer, got: %s", got)
	}
	if strings.Contains(got, "cvdate") {
		t.Errorf("string target must not emit cvdate, got: %s", got)
	}
}

// TestIssue803_DeadGrammarStillProducesPostfix is a sanity check for the
// allowlist's "dead grammar" entries: the SetStringFromNumber and the
// StrConcat<Type> family are documented as unreachable, but we want a
// guard that confirms the actually-reached sibling produces sane
// postfix for the same input. Future drift in the grammar could make
// these alts reachable, and we'd want to know.
func TestIssue803_DeadGrammarStillProducesPostfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"a.s": "string", "a.y": "integer", "a.x": "double", "a.d": "date",
	})
	// `"x=" + a.y` is documented to route through the base strConcat
	// (NOT through strConcatInt); the output must include `strconcat`.
	got, err := c.CompileAction(`set a.s = "x=" + a.y`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "strconcat") {
		t.Errorf("expected base strConcat to emit `strconcat`, got: %s", got)
	}
}
