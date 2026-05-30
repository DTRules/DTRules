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

// Issue #803 batch 12: leftovers — if/then, operatorlist entity tail,
// left-array-colon LHS, and three forms with no runtime op
// (entityFirst / entityFirstIn / dateEarliestAfter) that emit
// elstmterror placeholders.

func issue803Batch12Symbols() map[string]string {
	return map[string]string{
		"client.flag":   "boolean",
		"client.tax":    "integer",
		"client.code":   "string",
		"client.kids":   "array",
		"client.dates":  "array",
		"client.due":    "date",
		"family":        "entity",
		"family.kids":   "array",
		"family.parent": "entity",
		"Client":        "entity",
	}
}

// TestIssue803_IfThen: `if <b> then <stmts> endif` must emit
// `<b> { <stmts> } if`. Pre-fix the entire conditional was dropped.
func TestIssue803_IfThen(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`if client.flag then set client.tax = 5; endif`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{"client.flag", "{", "}", "if", "5"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_IfThenElse: `if/then/else` must emit `<b> { t } { e }
// ifelse`. Pre-fix both branches were dropped.
func TestIssue803_IfThenElse(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`if client.flag then set client.tax = 5; else set client.tax = 10; endif`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{"client.flag", "5", "10", "ifelse"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_OpListEntity: a non-terminal entity argument in an
// operator call must be emitted. Pre-fix `op(<e1>, <s>, <e2>)` would
// drop e1 silently.
func TestIssue803_OpListEntity(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	// Use a generic operator name; the type-check runs through
	// operatorstatements which dispatches via the operatorlist tree.
	got, err := c.CompileAction(`set client.code = string value of myop(family, "x")`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// The entity arg must appear in the emitted postfix.
	if !strings.Contains(got, "family") {
		t.Errorf("expected entity arg `family` in postfix, got: %s", got)
	}
	if !strings.Contains(got, "myop") {
		t.Errorf("expected op name in postfix, got: %s", got)
	}
}

// TestIssue803_OpListEntitySingle: terminal single entity arg.
func TestIssue803_OpListEntitySingle(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`set client.code = string value of myop(family)`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "family") {
		t.Errorf("expected entity arg `family` in postfix, got: %s", got)
	}
	if !strings.Contains(got, "myop") {
		t.Errorf("expected op name in postfix, got: %s", got)
	}
}

// TestIssue803_LeftArrayColon: `:Entity:<array-field>` on the LHS of
// `set X = arrayExpr`. Mirrors the existing LeftXxxColon family
// (visit colonRef → visit inner left-ref). The runtime xdef handles
// the entity-aware store; no entitypush/pop wrapping is needed at the
// emitter level (matches VisitLeftIexprColon behavior).
//
// Pre-fix the whole assignment was dropped because LeftArrayColon
// collapsed to empty postfix.
func TestIssue803_LeftArrayColon(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`set :Client:kids = family.kids`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{"Client", "/kids", "xdef", "family.kids"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_EntityFirst_ErrorsLoudly: `first <e> where ...` has
// no runtime op. Must emit elstmterror.
func TestIssue803_EntityFirst_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`set family.parent = first family where client.flag`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_EntityFirstIn_ErrorsLoudly: `first <e> in <arr>
// where ...` likewise.
func TestIssue803_EntityFirstIn_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`set family.parent = first family in family.kids where client.flag`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_DateEarliestAfter_ErrorsLoudly: `earliest of <arr>
// after <d>` has no runtime op. Must emit elstmterror.
func TestIssue803_DateEarliestAfter_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch12Symbols())
	got, err := c.CompileAction(`set client.due = earliest of client.dates after client.due`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}
