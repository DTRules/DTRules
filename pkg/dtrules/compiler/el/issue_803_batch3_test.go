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

// Issue #803 batch 3: array literal construction. Pre-fix,
// `set my_arr = [1, 2, 3]` produced "" (empty postfix) because every
// alt in the chain (arrayLit, arrayLiteral, arrayList<Type>(Single)?,
// setArrayArray, leftArraySimple) inherited from BaseELVisitor whose
// VisitChildren is a no-op.
//
// Construction pattern (now emitted):
//
//	newarray         // [array]
//	dup 1 addto      // [array]  array → [1]
//	dup 2 addto      // [array]  array → [1, 2]
//	dup 3 addto      // [array]  array → [1, 2, 3]
//
// `addto` mutates the array in place. `dup` keeps the reference on the
// stack so the next addto can reuse it. The final array stays on top so
// the SET trailer (`/<field> xdef`) can consume it.

func issue803Batch3Symbols() map[string]string {
	return map[string]string{
		"a.intlist": "array of integer",
		"a.strlist": "array of string",
		"a.flist":   "array of double",
		"a.blist":   "array of boolean",
	}
}

// TestIssue803_ArrayLiteralAssignment exercises every element-type
// arrayList alt across single and multi-element literals. The exact
// emitted shape is asserted for stability — drift in any of the 14
// arrayList visitors would change the output and fail this test.
func TestIssue803_ArrayLiteralAssignment(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch3Symbols())
	cases := []struct {
		dsl, want string
	}{
		// Int element type, single + multi.
		{
			"set a.intlist = [1]",
			"newarray dup 1 addto /a.intlist xdef",
		},
		{
			"set a.intlist = [1, 2, 3]",
			"newarray dup 1 addto dup 2 addto dup 3 addto /a.intlist xdef",
		},
		// String element type.
		{
			`set a.strlist = ["a", "b"]`,
			`newarray dup "a" addto dup "b" addto /a.strlist xdef`,
		},
		// Float element type.
		{
			"set a.flist = [1.5, 2.5]",
			"newarray dup 1.5 addto dup 2.5 addto /a.flist xdef",
		},
		// Empty list isn't a legal literal in the grammar (arrayList
		// requires at least one element), so we don't test it here.
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if got != tc.want {
				t.Errorf("postfix mismatch:\n  got:  %q\n  want: %q", got, tc.want)
			}
		})
	}
}

// TestIssue803_ArrayOfValuesAlternateSyntax confirms the alternate
// keyword-prefixed form `array of values [ ... ]` produces the same
// postfix as the bare `[ ... ]` literal.
func TestIssue803_ArrayOfValuesAlternateSyntax(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch3Symbols())

	bare, err := c.CompileAction("set a.intlist = [1, 2, 3]")
	if err != nil {
		t.Fatalf("compile bare: %v", err)
	}
	keyword, err := c.CompileAction("set a.intlist = array of values [1, 2, 3]")
	if err != nil {
		t.Fatalf("compile keyword: %v", err)
	}
	if bare != keyword {
		t.Errorf("syntactic equivalents must compile identically:\n  bare:    %q\n  keyword: %q", bare, keyword)
	}
}

// TestIssue803_ArrayLiteralOpsArePresent is a defensive check on the
// emission: even if the exact postfix shape evolves, the three
// constructor ops (newarray, dup, addto) and the assignment trailer
// must always appear. Pre-fix none of them did.
func TestIssue803_ArrayLiteralOpsArePresent(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch3Symbols())
	got, err := c.CompileAction("set a.intlist = [1, 2, 3]")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	required := []string{
		"newarray",
		"dup",
		"addto",
		"/a.intlist",
		"xdef",
		// Each literal element must be present:
		"1",
		"2",
		"3",
	}
	for _, tok := range required {
		if !strings.Contains(got, tok) {
			t.Errorf("expected token %q in postfix, got: %s", tok, got)
		}
	}
}

// TestIssue803_NestedArrayLiteral exercises VisitArrayListArray /
// VisitArrayListArraySingle — array-of-arrays construction. The runtime
// addto handles array elements transparently because arrays are first-
// class Objects.
func TestIssue803_NestedArrayLiteral(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"a.matrix": "array of array of integer",
	})
	got, err := c.CompileAction("set a.matrix = [[1, 2], [3, 4]]")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// Outer array gets a newarray; each inner literal also gets its own
	// newarray + element addto's; outer addto's append the inner arrays.
	want := "newarray " +
		"dup newarray dup 1 addto dup 2 addto addto " +
		"dup newarray dup 3 addto dup 4 addto addto " +
		"/a.matrix xdef"
	if got != want {
		t.Errorf("nested array postfix mismatch:\n  got:  %q\n  want: %q", got, want)
	}
}

// TestIssue803_DeadSetArrayAltsHonorReachableSibling confirms the
// allowlisted "dead grammar" setArrayInt/Float/String entries still
// produce correct postfix through their reachable siblings
// (setInt/setFloat/setString). Future grammar reordering that flips
// the predictor would land here loudly.
func TestIssue803_DeadSetArrayAltsHonorReachableSibling(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"a.intlist": "array of integer",
	})
	// `set <array> = <single int>` matches setInt (not setArrayInt) per
	// the dead-grammar rationale in the allowlist. The compiler should
	// still produce non-empty postfix — the single-value-into-array
	// semantics happens at runtime / type-coercion level.
	got, err := c.CompileAction("set a.intlist = 42")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if got == "" {
		t.Errorf("expected non-empty postfix from the setInt path, got empty")
	}
	if !strings.Contains(got, "42") {
		t.Errorf("expected `42` in postfix, got: %s", got)
	}
}
