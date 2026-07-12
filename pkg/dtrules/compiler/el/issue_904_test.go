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

package el

import (
	"strings"
	"testing"
)

// Issue #904 compile-shape pins. Execution coverage lives in
// pkg/dtrules/issue_904_exec_test.go; these pin the emitted postfix for the
// shapes whose token order or count was the bug.

func issue904Symbols() map[string]string {
	return map[string]string{
		"coll": "array", "coll2": "array", "s": "string", "url": "string",
		"recip_url":              "string",
		"token_recipient.url":    "string",
		"token_recipient.amount": "fixed",
	}
}

// TestIssue904_AddNewEntitySingleAddto: `add new T entity to coll` used to
// double-emit `swap addto` (the visitor appended a trailer the destination
// visitor already owns) — stack corruption at runtime.
func TestIssue904_AddNewEntitySingleAddto(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue904Symbols())

	got, err := c.CompileAction("add new token_recipient entity to coll")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "/token_recipient createentity coll swap addto"
	if normalizeSpaces(got) != want {
		t.Errorf("add-new shape\n got: %q\nwant: %q", got, want)
	}
}

// TestIssue904_DupDestsRelyOnDestStore: the two-destination family shares
// the same dest-owns-the-store contract; the explicit trailers doubled every
// array dest's addto.
func TestIssue904_DupDestsRelyOnDestStore(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue904Symbols())

	for _, tc := range []struct{ src, want string }{
		{
			"add new token_recipient entity to coll and to coll2",
			"/token_recipient createentity dup coll swap addto coll2 swap addto",
		},
		{
			"add s to coll and to coll2",
			"s dup coll swap addto coll2 swap addto",
		},
	} {
		got, err := c.CompileAction(tc.src)
		if err != nil {
			t.Fatalf("%q compile: %v", tc.src, err)
		}
		if normalizeSpaces(got) != tc.want {
			t.Errorf("%q\n got: %q\nwant: %q", tc.src, got, tc.want)
		}
	}
}

// TestIssue904_LowercaseOfSurface: `lowercase of <s>` / `uppercase of <s>`
// emit the registered string ops. Before the dedicated tokens the form
// parsed as relationship traversal (`url lowercase getrelationship`) and
// errored at runtime on strings.
func TestIssue904_LowercaseOfSurface(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue904Symbols())

	got, err := c.CompileAction("set s = lowercase of url")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "url lowercase") || strings.Contains(got, "getrelationship") {
		t.Errorf("lowercase-of shape: %q", got)
	}

	got, err = c.CompileAction("set s = uppercase of url")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "url uppercase") || strings.Contains(got, "getrelationship") {
		t.Errorf("uppercase-of shape: %q", got)
	}

	// The staking dedup shape: string comparison, not entity req.
	got, err = c.CompileCondition("lowercase of url == lowercase of recip_url")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "url lowercase recip_url lowercase streq"
	if normalizeSpaces(got) != want {
		t.Errorf("dedup condition\n got: %q\nwant: %q", got, want)
	}
}
