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

// Issue #803 batch 5: date constructor / cast family. Pre-fix, each
// form produced empty RHS for a date-target SET — only the assignment
// trailer `cvdate /<field> xdef` emitted, the inner expression
// dropped.

func issue803Batch5Symbols() map[string]string {
	return map[string]string{
		"a.d":     "date",
		"a.dlist": "array of date",
		"Client":  "entity",
	}
}

// TestIssue803_DateCastFormsEmitInnerExpr exercises the four cast/
// constructor forms that produce a date from another value: explicit
// (date) on a string, the date() function-call form, and the two
// indexed forms.
func TestIssue803_DateCastFormsEmitInnerExpr(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch5Symbols())

	cases := []struct {
		dsl, mustContain string
	}{
		// dateFromStrCast: `(date) "literal"`
		{`set a.d = (date) "2026-01-15"`, `"2026-01-15"`},
		// dateFromStrFunc: `date("literal")` — same emission as the cast.
		{`set a.d = date("2026-01-15")`, `"2026-01-15"`},
		// dateFromIndex: `(date) <indxExpr>` — relies on the
		// VisitIndxExpr fix from batch 1 for the inner array+index.
		{`set a.d = (date) a.dlist[0]`, "a.dlist 0 bytesidx"},
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
			// The cvdate op must always appear — the cast op itself.
			if !strings.Contains(got, "cvdate") {
				t.Errorf("expected cvdate in postfix, got: %s", got)
			}
		})
	}
}

// TestIssue803_DateTableLookupErrorsLoudly: hash-table operators were
// removed; the grammar still parses `(date) <table>(<args>)` so the
// alt must emit an elstmterror with a clear runtime message. Mirrors
// the StrTableLookup / TableNew placeholders.
func TestIssue803_DateTableLookupErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch5Symbols())

	got, err := c.CompileAction(`set a.d = (date) ClientTable("dob", "Alice")`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{
		`"hash tables removed`,
		"elstmterror",
	} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix (loud-runtime-error pattern), got: %s", tok, got)
		}
	}
}
