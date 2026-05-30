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

// Issue #803 batch 8: aggregation + name-coercion family.
// All five rules below previously emitted empty postfix (no override;
// antlr's VisitChildren is a no-op).

func issue803Batch8Symbols() map[string]string {
	return map[string]string{
		"client.income":  "double",
		"client.tax":     "integer",
		"client.code":    "string",
		"client.taxname": "name",
		"client.kids":    "array",
		"client.names":   "array",
		"family.kids":    "array",
		// Bare IDENTs for typedDouble/typedLong resolution inside fold
		// bodies — the floatSumOf alt requires a typedDouble (IDENT),
		// not a possessive/colon attr ref.
		"income":         "double",
		"tax":            "integer",
	}
}

// TestIssue803_IntSumOf: integer fold across an array. Element
// entity is auto-pushed by forall, so `client.tax` resolves against
// each element.
func TestIssue803_IntSumOf(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())
	got, err := c.CompileAction(`set client.tax = sum of client.tax in family.kids`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{"0", "forall", "+"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_FloatSumOf_Unreachable documents that floatSumOf is
// currently a grammar zombie: ANTLR's LL(*) prediction picks
// intSumOf at every fexpr-allowing context because `typedLong` and
// `typedDouble` both lex as bare IDENT, and `number : iexpr | fexpr`
// lists iexpr first. The VisitFloatSumOf override is still defined
// (right shape for when the grammar is disambiguated by a future
// token-classification pass), but the parser never reaches it. This
// test pins that finding so a future grammar fix breaks here loudly
// instead of silently — at which point the body should be replaced
// with the positive assertion (`f+`, `0.0` present).
func TestIssue803_FloatSumOf_Unreachable(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())
	got, err := c.CompileAction(`print sum of income in family.kids`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// Today the parse goes through intSumOf — `+` and a bare `0` seed.
	// If this ever flips to `f+`/`0.0`, replace this test body.
	if strings.Contains(got, "f+") {
		t.Errorf("floatSumOf is now reachable; replace this test with positive assertions. got: %s", got)
	}
	if !strings.Contains(got, "+") || !strings.Contains(got, "forall") {
		t.Errorf("expected intSumOf fallback postfix (+, forall), got: %s", got)
	}
}

// TestIssue803_IntIndexOf: emits the haystack and needle, then
// `indexof`. opIndexOf signature is ( string substring -- index ),
// so the haystack must be pushed first.
func TestIssue803_IntIndexOf(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())
	got, err := c.CompileAction(`set client.tax = index of "AB" in client.code`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "indexof") {
		t.Errorf("expected indexof in postfix, got: %s", got)
	}
	// Both operands must appear; the haystack must precede the needle
	// in the emitted token stream so the op pops needle first.
	hayIdx := strings.Index(got, "client.code")
	needleIdx := strings.Index(got, `"AB"`)
	if hayIdx < 0 || needleIdx < 0 {
		t.Fatalf("expected both operands in postfix, got: %s", got)
	}
	if !(hayIdx < needleIdx) {
		t.Errorf("expected haystack to precede needle, got: %s", got)
	}
}

// TestIssue803_NameFromStr: `(name) <strexpr>` must coerce via
// `cvn` — same as `the name <strexpr>`.
func TestIssue803_NameFromStr(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())
	got, err := c.CompileAction(`set client.taxname = (name) client.code`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "cvn") {
		t.Errorf("expected cvn in postfix, got: %s", got)
	}
}

// TestIssue803_NameArrayAt: `name <typedArray>[<iexpr>]` must emit
// the array name, the index, and a `getat` dispatch.
func TestIssue803_NameArrayAt(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch8Symbols())
	got, err := c.CompileAction(`set client.taxname = name client.names[2]`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{"client.names", "2", "getat"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}
