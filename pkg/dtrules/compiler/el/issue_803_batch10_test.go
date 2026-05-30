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

// Issue #803 batch 10: string-membership tests against an inline
// list. Both rules previously emitted empty postfix.

func issue803Batch10Symbols() map[string]string {
	return map[string]string{
		"client.state": "string",
		"client.code":  "string",
	}
}

// TestIssue803_BoolStrEqList: `<str> = "a", "b" or "c"` is a
// membership check. Verify each literal appears in the postfix, the
// equality op is `s==`, and the chain ends with `or`s to combine the
// results.
func TestIssue803_BoolStrEqList(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch10Symbols())
	got, err := c.CompileCondition(`client.state == "CA", "NY", or "TX"`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{`"CA"`, `"NY"`, `"TX"`, "s==", "or"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
	// Three values → two `or` operators total. Cheap arity check.
	if strings.Count(got, "or") < 2 {
		t.Errorf("expected at least two `or` operators for a 3-value list, got: %s", got)
	}
}

// TestIssue803_BoolStrEqIcList: case-insensitive variant. Same shape
// but `s==i` for each comparison.
func TestIssue803_BoolStrEqIcList(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch10Symbols())
	got, err := c.CompileCondition(`client.state is equal to ignore case "ca", "ny", or "tx"`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, tok := range []string{`"ca"`, `"ny"`, `"tx"`, "s==i", "or"} {
		if !strings.Contains(got, tok) {
			t.Errorf("expected %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_BoolStrEqList_TwoValues: minimum non-trivial list is
// two values. Confirms the visitor doesn't emit a leading `or` and
// produces exactly one combining op.
func TestIssue803_BoolStrEqList_TwoValues(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch10Symbols())
	got, err := c.CompileCondition(`client.state == "CA", or "NY"`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Count(got, "or") != 1 {
		t.Errorf("expected exactly one `or` for a 2-value list, got %d occurrences: %s",
			strings.Count(got, "or"), got)
	}
	if strings.Count(got, "s==") != 2 {
		t.Errorf("expected exactly two `s==` ops for a 2-value list, got %d: %s",
			strings.Count(got, "s=="), got)
	}
}
