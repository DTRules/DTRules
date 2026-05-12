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

package decisiontable

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func TestCheckHandCodedPostfix_FlagsPostfixOnly(t *testing.T) {
	entries := []PostfixEntry{
		{Kind: "action", Number: 1, DSL: "", Postfix: "0 /result.foo xdef"},
		{Kind: "action", Number: 2, DSL: "set result.foo = 0", Postfix: "0 /result.foo xdef"},
	}
	ws := CheckHandCodedPostfix("T", entries)
	if len(ws) != 1 {
		t.Fatalf("got %d warnings, want 1: %v", len(ws), ws)
	}
	if ws[0].Kind != "hand-coded postfix" {
		t.Errorf("warning kind = %q", ws[0].Kind)
	}
	if !strings.Contains(ws[0].Reason, "action 1") {
		t.Errorf("warning reason should call out action 1: %q", ws[0].Reason)
	}
}

func TestCheckHandCodedPostfix_IgnoresCommentOnlyPostfix(t *testing.T) {
	entries := []PostfixEntry{
		{Kind: "action", Number: 1, Postfix: "// nothing to do here"},
		{Kind: "action", Number: 2, Postfix: "/* still nothing */"},
		{Kind: "action", Number: 3, Postfix: "  \n\n  "},
	}
	if ws := CheckHandCodedPostfix("T", entries); len(ws) != 0 {
		t.Errorf("comment/empty postfix should be ignored, got: %v", ws)
	}
}

func TestHasAnyHandCodedPostfix_TrueWhenAllPostfix(t *testing.T) {
	entries := []PostfixEntry{
		{Kind: "action", Number: 1, Postfix: "0 /result.x xdef"},
		{Kind: "action", Number: 2, Postfix: "1 /result.y xdef"},
	}
	if !HasAnyHandCodedPostfix(entries) {
		t.Error("expected true: all entries are postfix-only")
	}
}

// Strict rule (post-2026-05-12): a partially-authored table — some
// entries in EL, others still in hand-coded postfix — is rejected. Any
// single offending row fails the whole table.
func TestHasAnyHandCodedPostfix_TrueWhenMixed(t *testing.T) {
	entries := []PostfixEntry{
		{Kind: "action", Number: 1, Postfix: "0 /result.x xdef"}, // hand-coded
		{Kind: "action", Number: 2, DSL: "set result.y = 1", Postfix: "1 /result.y xdef"},
	}
	if !HasAnyHandCodedPostfix(entries) {
		t.Error("expected true: action 1 has hand-coded postfix even though action 2 has DSL")
	}
}

// All entries authored in EL — nothing hand-coded, nothing to block.
func TestHasAnyHandCodedPostfix_FalseWhenAllAuthored(t *testing.T) {
	entries := []PostfixEntry{
		{Kind: "action", Number: 1, DSL: "set result.x = 0", Postfix: "0 /result.x xdef"},
		{Kind: "action", Number: 2, DSL: "set result.y = 1", Postfix: "1 /result.y xdef"},
	}
	if HasAnyHandCodedPostfix(entries) {
		t.Error("expected false: every entry has DSL")
	}
}

func TestHasAnyHandCodedPostfix_FalseWhenNoPostfix(t *testing.T) {
	entries := []PostfixEntry{
		{Kind: "action", Number: 1, DSL: "set result.x = 0"},
	}
	if HasAnyHandCodedPostfix(entries) {
		t.Error("expected false: no postfix anywhere")
	}
}

// Smoke test for the runtime gate: a table flagged with SetHandCodedPostfix
// must refuse to execute. We exercise this without a full Session by
// asserting the refusal path on the RDecisionTable directly. Note: full
// integration testing happens through the loader-level test
// (TestLegacyPostfixTable_RefusesExecute) which runs the full loader →
// execute pipeline.
func TestRDecisionTable_RefuseHandCodedPostfix(t *testing.T) {
	dt := &RDecisionTable{compiled: true, name: dtrules.GetRName("LegacyTable")}
	dt.SetHandCodedPostfix(true, "loaded from Test.xls")
	if !dt.HasHandCodedPostfix() {
		t.Fatal("HasHandCodedPostfix should be true after SetHandCodedPostfix")
	}
	err := dt.refuseIfHandCodedPostfix("ExecuteTable")
	if err == nil {
		t.Fatal("expected error from refuseIfHandCodedPostfix when flag is set")
	}
	if !strings.Contains(err.Error(), "hand-coded postfix") {
		t.Errorf("error should mention hand-coded postfix: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "loaded from Test.xls") {
		t.Errorf("error should include reason: %q", err.Error())
	}

	// Clearing the flag must un-block execution.
	dt.SetHandCodedPostfix(false, "")
	if err := dt.refuseIfHandCodedPostfix("ExecuteTable"); err != nil {
		t.Errorf("after clearing flag, refuseIfHandCodedPostfix returned: %v", err)
	}
}
