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

// Issue #803 batch 7: SubDestColon + BoolFunction.
//
// Before this batch:
//   - `subtract <n> from :Client:<field>` silently emitted empty
//     postfix (no VisitSubDestColon; antlr's VisitChildren is a
//     no-op).
//   - `<func>(<args>)` where <func> is a TypedBoolFunction
//     (`isstring`, `isnumber`, `isnull`, `isarray`) silently emitted
//     empty postfix for the same reason.
//
// `VisitAddDestColon` already existed but only handled the
// possessive form (`Client's <field>`) — the colon-chain form
// (`:Client:<field>`) was missing. Both forms now share a
// `colonRefEntityName` helper and produce matching postfix.

func issue803Batch7Symbols() map[string]string {
	return map[string]string{
		"client.my_int":    "integer",
		"client.my_double": "double",
		"Client":           "entity",
	}
}

// TestIssue803_AddSubDestColon: add/subtract destination via the
// `:Entity:field` form must walk the entity stack just like the
// `Entity's field` form. Both forms must emit identical postfix
// shape (entitypush ... entitypop) so runtime semantics are stable
// across spellings.
func TestIssue803_AddSubDestColon(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch7Symbols())

	cases := []struct {
		dsl  string
		want []string
	}{
		{`add 5 to :Client:my_int`, []string{"Client", "entitypush", "my_int", "entitypop"}},
		{`add 5 to Client's my_int`, []string{"Client", "entitypush", "my_int", "entitypop"}},
		{`subtract 5 from :Client:my_int`, []string{"Client", "entitypush", "my_int", "entitypop"}},
		{`subtract 5 from Client's my_int`, []string{"Client", "entitypush", "my_int", "entitypop"}},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			for _, tok := range tc.want {
				if !strings.Contains(got, tok) {
					t.Errorf("expected %q in postfix, got: %s", tok, got)
				}
			}
		})
	}
}

// TestIssue803_BoolFunctionEmitsName: a bare TypedBoolFunction
// reference (the grammar production is `typedBoolFunction : IDENT`,
// no arg list) must emit the IDENT as the dispatch operator. Pre-fix
// this produced empty postfix because there was no override for
// VisitBoolFunction and antlr's VisitChildren is a no-op.
func TestIssue803_BoolFunctionEmitsName(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch7Symbols())

	// Niladic form: the IDENT alone resolves through bexpr's
	// boolFunction alt; the resulting postfix must contain the IDENT
	// so the runtime dispatches it.
	cases := []string{"some_bool_fn", "another_bool_fn"}
	for _, dsl := range cases {
		t.Run(dsl, func(t *testing.T) {
			got, err := c.CompileCondition(dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, dsl) {
				t.Errorf("expected operator %q in postfix, got: %s", dsl, got)
			}
		})
	}
}
