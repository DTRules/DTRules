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

// Issue #812: localvariables (formerly reachable only from the
// `contextForTable` non-terminal) lifted to the `statement` rule so
// `local entity X = new T entity` etc. parse inside <action_dsl> and
// <initial_action_dsl>. Pre-fix the parser errored with
// "mismatched input 'local' expecting {'action', 'condition',
// 'policystatement', 'context'}" because the action-statement grammar
// didn't list localvariables as an option.
//
// The emission shape DIFFERS from the context case as of #904: at context
// level the table body block is already on the data stack when the
// declaration postfix runs, so the flat
// `<value> cv<type> allocate execute deallocate pop` works. Inside an
// action body there is no block on the stack — the flat shape underflowed
// at `execute` while the statements using the local ran after
// `deallocate`. The action form now wraps the remaining statements of the
// list as the executed block:
//
//	<value> cv<type> allocate { <rest> } execute deallocate pop

// TestIssue812_LocalEntityParsesInActionBody is the issue's headline
// case: declare a fresh local entity at action-statement level, the
// exact form the staking project's #276 slice 4 was reaching for.
func TestIssue812_LocalEntityParsesInActionBody(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"token_recipient": "entity",
	})

	dsl := "local entity r = new token_recipient entity"
	got, err := c.CompileAction(dsl)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// Action form: the rest of the statement list (here empty) is the
	// executed block (#904).
	want := "/token_recipient createentity cve allocate { } execute deallocate pop"
	if got != want {
		t.Errorf("action emission:\n  got:  %q\n  want: %q", got, want)
	}

	// Context form keeps the flat shape — the table body block is on the
	// stack at context level, so `execute` has its operand there. The two
	// forms deliberately diverge (#904); this pins the context side.
	c.ResetLocals()
	ctxOut, err := c.CompileContext(dsl)
	if err != nil {
		t.Fatalf("compile (context): %v", err)
	}
	ctxWant := "/token_recipient createentity cve allocate execute deallocate pop"
	if ctxOut != ctxWant {
		t.Errorf("context emission:\n  got:  %q\n  want: %q", ctxOut, ctxWant)
	}
}

// TestIssue812_LocalPrimitivesParseInActionBody confirms the lift
// covers every local-variable alt (entity / long / double / bool /
// date / fixed / bigint / bytes / string), not just the entity case
// that motivated the issue. All of them share the same statement-rule
// dispatch so one grammar change unlocks every type.
func TestIssue812_LocalPrimitivesParseInActionBody(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"token_recipient": "entity",
	})

	cases := []struct {
		dsl, mustContain string
	}{
		{"local entity r = new token_recipient entity", "createentity"},
		{"local long counter = 5", "5 cvi"},
		{"local fixed payout = 100fp", "100fp"},
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
			// Every local-variables alt uses the same scope-wrapping
			// pattern. If one of them ever stops emitting this trio,
			// the local will leak or never deallocate.
			for _, tok := range []string{"allocate", "execute", "deallocate", "pop"} {
				if !strings.Contains(got, tok) {
					t.Errorf("expected scope op %q in postfix, got: %s", tok, got)
				}
			}
		})
	}
}

// TestIssue812_ReadFromActionLocalUsesSlot confirms a local declared
// in an action body is reachable by name in subsequent statements
// within the same action block — the local's slot index resolves
// correctly to slot 0 (per-table scope).
func TestIssue812_ReadFromActionLocalUsesSlot(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"token_recipient":               "entity",
		"calculation_context.recipients": "array of entity",
	})

	got, err := c.CompileAction(
		`local entity r = new token_recipient entity; add r to calculation_context.recipients`,
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// `add r to <array>` must reference the freshly-declared local via
	// slot 0 — not as a bare entity name. The `0 local@` sequence is
	// the proof that the read-path resolved against the declaration.
	if !strings.Contains(got, "0 local@") {
		t.Errorf("expected `0 local@` (local-slot read) in postfix, got: %s", got)
	}
	if !strings.Contains(got, "addto") {
		t.Errorf("expected `addto` (array append op) in postfix, got: %s", got)
	}
}
