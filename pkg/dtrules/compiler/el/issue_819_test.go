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

// Issue #819: writing to a local-entity field (`set r.field = X` where
// r was declared as `local entity r = new T entity`) used to emit two
// wrong pieces of postfix:
//
//  1. The store path: `/r.field xdef` — treating `r.field` as a global
//     field path instead of slot-indexed access on the local entity.
//  2. The type cast: `cvi` — defaulting to integer because the
//     symbol-table lookup for `r.field` didn't resolve to anything
//     (the local's specific entity type wasn't tracked).
//
// Both are fixed in the same PR:
//
//   - emitFieldStore now tries emitAliasFieldStore, which emits the
//     entity-stack-mediated assignment sequence
//     `<slot> local@ entitypush /<field> xdef entitypop pop`.
//   - LocalVar tracks the entity type captured from
//     `new T entity` declarations; mutationType uses it to look up
//     `<T>.<field>` in the EDD for type-aware cv* dispatch.

// issue819Symbols mirrors the staking-style fixture: a fresh
// token_recipient entity gets stored in a local, with field types
// covering string / fixed / integer / boolean (the four cv*
// dispatches we need to verify).
func issue819Symbols() map[string]string {
	return map[string]string{
		"token_recipient":         "entity",
		"token_recipient.url":     "string",
		"token_recipient.amount":  "fixed",
		"token_recipient.count":   "integer",
		"token_recipient.active":  "boolean",
		"payout_url":              "string",
		"net_reward":              "fixed",
	}
}

// TestIssue819_SetLocalEntityFieldEmitsSlotStore confirms the
// entity-stack-mediated store sequence replaces the old `/r.field xdef`
// global-write that was the original bug.
func TestIssue819_SetLocalEntityFieldEmitsSlotStore(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue819Symbols())
	if _, err := c.CompileContext("local entity r = new token_recipient entity"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cases := []struct {
		dsl, mustContain, mustNotContain string
	}{
		{
			dsl:            `set r.url = "literal"`,
			mustContain:    "0 local@ entitypush /url xdef entitypop pop",
			mustNotContain: "/r.url xdef",
		},
		{
			dsl:            `set r.amount = 100fp`,
			mustContain:    "0 local@ entitypush /amount xdef entitypop pop",
			mustNotContain: "/r.amount xdef",
		},
		{
			dsl:            `set r.count = 5`,
			mustContain:    "0 local@ entitypush /count xdef entitypop pop",
			mustNotContain: "/r.count xdef",
		},
		{
			dsl:            `set r.active = true`,
			mustContain:    "0 local@ entitypush /active xdef entitypop pop",
			mustNotContain: "/r.active xdef",
		},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, tc.mustContain) {
				t.Errorf("missing slot-based store sequence %q in postfix:\n  %s", tc.mustContain, got)
			}
			if strings.Contains(got, tc.mustNotContain) {
				t.Errorf("postfix still contains pre-fix global-write %q:\n  %s", tc.mustNotContain, got)
			}
		})
	}
}

// TestIssue819_LocalEntityFieldCvDispatch confirms the cv* cast on
// the value matches the FIELD's declared type (looked up via the
// local's tracked entity type), not the parser's default integer
// fallback that fired pre-fix.
func TestIssue819_LocalEntityFieldCvDispatch(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue819Symbols())
	if _, err := c.CompileContext("local entity r = new token_recipient entity"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cases := []struct {
		dsl, wantCv string
	}{
		// String field — value comes from another string field; cv must
		// be cvs (the type of the destination field), not cvi.
		{`set r.url = payout_url`, "cvs"},
		// Fixed field — pre-fix this got cvi because mutationType
		// returned "" and the SET defaulted to integer.
		{`set r.amount = net_reward`, "cvfp"},
		// Integer field — happens to match the default but assert
		// anyway so future changes can't drift it.
		{`set r.count = 5`, "cvi"},
		// Boolean field — distinct cv to catch any "default to int"
		// regression.
		{`set r.active = true`, "cvb"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			// The cv* must precede the slot-store sequence. Pad with
			// spaces so " cv* " matches at boundaries.
			padded := " " + got + " "
			want := " " + tc.wantCv + " "
			if !strings.Contains(padded, want) {
				t.Errorf("expected cast op %q in postfix, got: %s", tc.wantCv, got)
			}
		})
	}
}

// TestIssue819_ReadPathStillWorks is a regression guard: the existing
// emitAliasFieldAccess (read-side) must continue to emit the slot-based
// `<slot> local@ /<field> get` sequence and nothing about the
// emitAliasFieldStore (write-side) addition should regress it.
func TestIssue819_ReadPathStillWorks(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue819Symbols())
	if _, err := c.CompileContext("local entity r = new token_recipient entity"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := c.CompileCondition(`r.url == "x"`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "0 local@ /url get") {
		t.Errorf("expected slot-based read sequence in postfix, got: %s", got)
	}
}
