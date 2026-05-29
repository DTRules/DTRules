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

// Issue #803 batch 4: colon-ref field access + AddTo/SubFrom dead-grammar
// verification.

func issue803Batch4Symbols() map[string]string {
	return map[string]string{
		"Client":         "entity",
		"client.fee":     "bigint",
		"client.count":   "integer",
		"client.balance": "double",
		"client.active":  "boolean",
		"client.dob":     "date",
	}
}

// TestIssue803_ColonRefEmitsField confirms `:Client:<field>` produces
// non-empty postfix. Pre-fix the LHS was dropped entirely — for
// instance `:Client:fee > 0` compiled to "0 >" with the field gone.
//
// Reachable alts (per parse-tree inspection): intColonRef (numeric
// IDENT) and boolColonRef (after `is true/false`). The other 5 alts
// are defensive overrides; the parser doesn't reach them for normal
// IDENT-prefixed forms but the visitors handle the case symmetrically
// if the grammar ever shifts.
func TestIssue803_ColonRefEmitsField(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch4Symbols())

	cases := []struct {
		dsl, mustContain string
	}{
		// intColonRef: typedLong wins parser-side for any IDENT
		// matching a numeric type.
		{`:Client:fee > 0`, "fee"},
		{`:Client:count > 0`, "count"},
		{`:Client:balance > 0.0`, "balance"},
		{`:Client:dob > today`, "dob"},
		// boolColonRef: `is true/false` forces the boolean path.
		{`:Client:active is true`, "active"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileCondition(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, tc.mustContain) {
				t.Errorf("expected field %q in postfix, got: %s", tc.mustContain, got)
			}
			// Client (the colon-ref entity name) must also appear —
			// pre-fix the entire colonRef walk was a no-op too.
			if !strings.Contains(got, "Client") {
				t.Errorf("expected entity `Client` in postfix, got: %s", got)
			}
		})
	}
}

// TestIssue803_AddToSubFromUseAddtostatement confirms the action forms
// `add X to Y` and `subtract X from Y` route through the
// addtostatement rule, not the fexpr/iexpr AddTo/SubFrom alts. The
// fexpr alts are dead grammar; this test pins the actually-firing
// path so a future grammar reorder that flips dispatch lands here.
func TestIssue803_AddToSubFromUseAddtostatement(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"a.x": "double", "a.y": "integer", "a.f": "fixed",
	})
	cases := []struct {
		dsl, mustContain string
	}{
		// double field: cvd promotes int operand; f+ does the work.
		{"add 2 to a.x", "f+"},
		{"subtract 2 from a.x", "f-"},
		// integer field: plain + / -.
		{"add 2 to a.y", " + "},
		{"subtract 2 from a.y", " - "},
		// fixed field: cvfp promotes int; fp+ does the work.
		{"add 2 to a.f", "fp+"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			padded := " " + got + " "
			if !strings.Contains(padded, tc.mustContain) {
				t.Errorf("expected %q in postfix, got: %s", tc.mustContain, got)
			}
		})
	}
}
