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

// Issue #803 batch 6: table-lookup family (hash tables were removed)
// + the actually-reached `using <ident>(<expr>)` form.

func issue803Batch6Symbols() map[string]string {
	return map[string]string{
		"a.x":    "double",
		"a.y":    "integer",
		"a.e":    "entity",
		"Client": "entity",
	}
}

// TestIssue803_TableLookupErrorsLoudly: every (entity/double/long)
// table-lookup form, plus `set <table> = <table>` and the
// `tableinformation` keyword, must produce non-empty postfix with
// the elstmterror sentinel so the runtime fails with a clear message
// — mirrors the StrTableLookup / TableNew placeholders.
func TestIssue803_TableLookupErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch6Symbols())
	cases := []struct {
		dsl, kind string
	}{
		{`set a.x = (double) ClientTable("x")`, "double"},
		{`set a.y = (long) ClientTable("x")`, "long"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			for _, tok := range []string{
				`"hash tables removed`,
				"elstmterror",
			} {
				if !strings.Contains(got, tok) {
					t.Errorf("expected %q in postfix, got: %s", tok, got)
				}
			}
		})
	}
}

// TestIssue803_UsingArrayEmitsEntityPush: `using <ident> (<expr>)`
// matches `intUsingArray` parser-side (the intUsing family loses to
// it on IDENT inputs). The semantic intent is entity-stack
// delegation — mirror VisitBigUsing's emission.
func TestIssue803_UsingArrayEmitsEntityPush(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch6Symbols())

	cases := []struct {
		dsl string
	}{
		{`set a.x = using Client (a.x)`},
		{`set a.y = using Client (a.y)`},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			for _, tok := range []string{"entitypush", "entitypop"} {
				if !strings.Contains(got, tok) {
					t.Errorf("expected %q in postfix, got: %s", tok, got)
				}
			}
			// Pre-fix the entire RHS was dropped — only the assignment
			// trailer remained.
			if !strings.Contains(got, "Client") {
				t.Errorf("expected entity name in postfix, got: %s", got)
			}
		})
	}
}
