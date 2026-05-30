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

// Issue #803 batch 9: array-family alts.
// All five rules previously emitted empty postfix.

func issue803Batch9Symbols() map[string]string {
	return map[string]string{
		"client.kids":   "array",
		"client.backup": "array",
		"client.parts":  "array",
		"client.csv":    "string",
		"client.code":   "string",
	}
}

// TestIssue803_ArrayDeepCopy: both `get deepcopy of` and `deepcopy
// of` must emit `<array> deepcopy`. Pre-fix both silently emitted
// nothing — the destination array stayed unbound.
func TestIssue803_ArrayDeepCopy(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch9Symbols())
	cases := []string{
		`set client.backup = get deep copy of client.kids`,
		`set client.backup = deep copy of client.kids`,
	}
	for _, dsl := range cases {
		t.Run(dsl, func(t *testing.T) {
			got, err := c.CompileAction(dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if !strings.Contains(got, "deepcopy") {
				t.Errorf("expected deepcopy in postfix, got: %s", got)
			}
			if !strings.Contains(got, "client.kids") {
				t.Errorf("expected source array in postfix, got: %s", got)
			}
		})
	}
}

// TestIssue803_ArrayTokenize: `tokenize <src> by <delim>` must emit
// the existing `split` op with the right operand order. opSplit pops
// delimiter then string.
func TestIssue803_ArrayTokenize(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch9Symbols())
	got, err := c.CompileAction(`set client.parts = tokenize client.csv by ","`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "split") {
		t.Errorf("expected split in postfix, got: %s", got)
	}
	srcIdx := strings.Index(got, "client.csv")
	delimIdx := strings.Index(got, `","`)
	if srcIdx < 0 || delimIdx < 0 {
		t.Fatalf("expected both operands in postfix, got: %s", got)
	}
	if !(srcIdx < delimIdx) {
		t.Errorf("expected source to precede delimiter, got: %s", got)
	}
}

// TestIssue803_ArrayMap_ErrorsLoudly: `map <array> through <table>`
// has no runtime op yet; the visitor must emit an elstmterror so it
// fails loudly at runtime instead of silently producing wrong
// results.
func TestIssue803_ArrayMap_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch9Symbols())
	got, err := c.CompileAction(`set client.backup = map client.kids through some_table`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_ArrayName_ErrorsLoudly: `(array) NAME` is a
// name-to-value resolution that has no runtime op yet; the visitor
// must emit elstmterror so this fails loudly instead of silently.
func TestIssue803_ArrayName_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch9Symbols())
	got, err := c.CompileAction(`set client.backup = (array) $kids`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}
