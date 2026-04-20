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

// Issue #709: the set-statement visitors emitted `cvi /<name> xdef` even
// when the target was a declared local of a different type. That silently
// truncated fp/bigint values via cvi and errored at runtime because xdef
// can't resolve a name that lives in a local-frame slot. These tests pin
// the fix: dispatch the type converter against mutationType (local first,
// then EDD) and route the store through emitFieldStore so locals land in
// the frame via `<idx> local!`.

// TestSetLocalFixed — the exact bug from #709: `set amount = 1.5fp` on a
// `local fixed amount` must preserve the fp value and write to the local
// frame. Before the fix this emitted `1.5fp cvi /amount xdef`.
func TestSetLocalFixed(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local fixed amount = 0.0fp;", "set amount = 1.5fp;")
	want := []string{"1.5fp", "cvfp", "0 local!"}
	notWant := []string{"cvi", "/amount xdef", " amount "}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalBigInt — bigint local assignment must stay on the bigint
// grid (cvbi) rather than truncating via cvi.
func TestSetLocalBigInt(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local bigint n = (bigint) 0;", "set n = (bigint) 100;")
	want := []string{"cvbi", "0 local!"}
	notWant := []string{"/n xdef", "cvi"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalDouble — double local assignment keeps the double coercion
// and writes to the local-frame slot.
func TestSetLocalDouble(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local double d = 0.0;", "set d = 3.14;")
	want := []string{"3.14", "cvd", "0 local!"}
	notWant := []string{"/d xdef", "cvi", "cvfp"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalLong — plain long local still gets cvi and local!, which
// matches the int-default path but through the local-aware store.
func TestSetLocalLong(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local long i = 0;", "set i = 42;")
	want := []string{"42", "cvi", "0 local!"}
	notWant := []string{"/i xdef"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalBool — boolean local assignment must use cvb and local!.
func TestSetLocalBool(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local boolean b = true;", "set b = false;")
	want := []string{"0 local!"}
	notWant := []string{"/b xdef"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalString — string local assignment routes through local!.
func TestSetLocalString(t *testing.T) {
	got := compileWithLocalCtx(t,
		`local string s = "";`, `set s = "hi";`)
	want := []string{`"hi"`, "0 local!"}
	notWant := []string{"/s xdef"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalFixed_FromBigInt — cross-type promotion: a bigint RHS into
// a fixed local must land as fixed. The converter emitted from the set
// visitor is cvfp (the target's type), regardless of what the RHS was.
func TestSetLocalFixed_FromBigInt(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local fixed bal = 0.0fp;", "set bal = (bigint) 100;")
	want := []string{"cvfp", "0 local!"}
	notWant := []string{"/bal xdef"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetLocalMultiple — second local gets index 1 and uses that index
// when set is invoked. Cross-checks the frame-slot dispatch.
func TestSetLocalMultiple(t *testing.T) {
	c := NewCompiler()
	if _, err := c.CompileContext("local fixed amt = 0.0fp;"); err != nil {
		t.Fatalf("CompileContext amt: %v", err)
	}
	if _, err := c.CompileContext("local fixed bal = 0.0fp;"); err != nil {
		t.Fatalf("CompileContext bal: %v", err)
	}
	got, err := c.CompileAction("set bal = 2.5fp;")
	if err != nil {
		t.Fatalf("CompileAction: %v", err)
	}
	want := []string{"2.5fp", "cvfp", "1 local!"}
	notWant := []string{"/bal xdef", "0 local!"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetEntityField_Regression — entity fields (no local shadow) must
// still use the `/<name> xdef` form. Guards against accidentally breaking
// the non-local path while wiring up local dispatch.
func TestSetEntityField_Regression(t *testing.T) {
	c := NewCompiler()
	got, err := c.CompileAction("set some_field = 5;")
	if err != nil {
		t.Fatalf("CompileAction: %v", err)
	}
	want := []string{"5", "cvi", "/some_field", "xdef"}
	notWant := []string{"local!", "local@"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}

// TestSetEntityField_FixedType — when the EDD symbol table marks a
// field as fixed, the set must dispatch as fixed even with no local.
// This exercises the same mutationType fallback path used by the
// increment/addto helpers.
func TestSetEntityField_FixedType(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"balance": TypeFixed})
	got, err := c.CompileAction("set balance = 1.5fp;")
	if err != nil {
		t.Fatalf("CompileAction: %v", err)
	}
	want := []string{"1.5fp", "cvfp", "/balance", "xdef"}
	notWant := []string{"cvi", "local!"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in postfix, got: %s", w, got)
		}
	}
	for _, n := range notWant {
		if strings.Contains(got, n) {
			t.Errorf("unexpected %q in postfix: %s", n, got)
		}
	}
}
