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

// Issue #693: the field-mutation helpers in postfix_emitter.go emitted
// raw `<name>` / `/<name> xdef` sequences, which work for entity fields
// but fail for local variables declared with `local bigint x` etc. Local
// variables must use frame-stack accessors: `<index> local@` to read and
// `<index> local!` to write. This file covers the local-var dispatch for
// increment / decrement / addto / subfrom, and cross-checks that entity
// fields still use the bare-name form.

// compileWithLocalCtx declares a local via CompileContext then compiles
// the action against the same compiler instance so the local survives.
func compileWithLocalCtx(t *testing.T, context, action string) string {
	t.Helper()
	c := NewCompiler()
	if _, err := c.CompileContext(context); err != nil {
		t.Fatalf("CompileContext(%q): %v", context, err)
	}
	postfix, err := c.CompileAction(action)
	if err != nil {
		t.Fatalf("CompileAction(%q): %v", action, err)
	}
	return postfix
}

// TestLocalVar_IncrementLong — `local long i; increment i` must use
// `local@` / `local!`, not the bare identifier or `/i xdef`.
func TestLocalVar_IncrementLong(t *testing.T) {
	got := compileWithLocalCtx(t, "local long i = 0;", "increment i;")
	// Expect: <value> <push> + <store>. No /i xdef, no bare `i` token.
	want := []string{"0 local@", "0 local!", " + "}
	notWant := []string{"/i xdef", " i ", "i + /i"}
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

// TestLocalVar_IncrementBigInt — bigint local must keep the b+ op
// while routing the read/store through local@/local!.
func TestLocalVar_IncrementBigInt(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local bigint x = (bigint) 100;", "increment x;")
	want := []string{"cvbi", "0 local@", "b+", "0 local!"}
	notWant := []string{"/x xdef", " x ", "fp+", "f+"}
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

// TestLocalVar_IncrementDouble — double local routes the 1.0 push and
// the f+ op through local@/local!.
func TestLocalVar_IncrementDouble(t *testing.T) {
	got := compileWithLocalCtx(t, "local double d = 0.0;", "increment d;")
	want := []string{"1.0", "cvd", "0 local@", "f+", "0 local!"}
	notWant := []string{"/d xdef", "fp+", "b+"}
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

// TestLocalVar_DecrementLong — `decrement` on a long local must still
// produce `field - value` semantics (via swap) and write back with local!.
func TestLocalVar_DecrementLong(t *testing.T) {
	got := compileWithLocalCtx(t, "local long i = 5;", "decrement i;")
	// `1 <push> swap - <store>` — swap ensures we compute field-value.
	want := []string{"0 local@", "swap", " - ", "0 local!"}
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

// TestLocalVar_DecrementBigInt — bigint local decrement.
func TestLocalVar_DecrementBigInt(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local bigint x = (bigint) 100;", "decrement x;")
	want := []string{"cvbi", "0 local@", "swap", "b-", "0 local!"}
	notWant := []string{"/x xdef", "fp-"}
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

// TestLocalVar_AddToLong — `add <value> to <local>`. The bare-IDENT
// numeric destination path in VisitAddArrayToArray routes through
// emitTypeAwareAddSub, which now stores into the local-frame slot.
func TestLocalVar_AddToLong(t *testing.T) {
	got := compileWithLocalCtx(t, "local long i = 0;", "add 5 to i;")
	want := []string{"5", "0 local@", " + ", "0 local!"}
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

// TestLocalVar_AddToBigInt — `add 5 to x` where x is a local bigint.
func TestLocalVar_AddToBigInt(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local bigint x = (bigint) 100;", "add 5 to x;")
	want := []string{"cvbi", "0 local@", "b+", "0 local!"}
	notWant := []string{"/x xdef"}
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

// TestLocalVar_SubtractFromLong — `subtract <value> from <local>`.
func TestLocalVar_SubtractFromLong(t *testing.T) {
	got := compileWithLocalCtx(t,
		"local long i = 10;", "subtract 3 from i;")
	want := []string{"3", "0 local@", "swap", " - ", "0 local!"}
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

// TestLocalVar_MultipleLocals — confirms that the second local gets
// index 1 (not 0) and that its mutation uses that index correctly.
func TestLocalVar_MultipleLocals(t *testing.T) {
	c := NewCompiler()
	if _, err := c.CompileContext("local long i = 0;"); err != nil {
		t.Fatalf("CompileContext i: %v", err)
	}
	if _, err := c.CompileContext("local long j = 5;"); err != nil {
		t.Fatalf("CompileContext j: %v", err)
	}
	got, err := c.CompileAction("increment j;")
	if err != nil {
		t.Fatalf("CompileAction: %v", err)
	}
	// j should be index 1.
	want := []string{"1 local@", "1 local!"}
	notWant := []string{"/j xdef", "0 local!"}
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

// TestEntityField_StillWorks — cross-check that entity-field mutations
// still use the bare-name / xdef form. An EDD-declared field without a
// matching local must not emit local@/local!.
func TestEntityField_StillWorks(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.count": TypeInteger,
	})
	got, err := c.CompileAction("increment taxpayer.count;")
	if err != nil {
		t.Fatalf("CompileAction: %v", err)
	}
	want := []string{"taxpayer.count", "/taxpayer.count xdef"}
	notWant := []string{"local@", "local!"}
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

// TestEntityField_BigIntUnchanged — bigint entity field keeps the
// cvbi / b+ / /field xdef form (not the local-frame accessors).
func TestEntityField_BigIntUnchanged(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"total": TypeBigInt})
	got, err := c.CompileAction("add 1 to total;")
	if err != nil {
		t.Fatalf("CompileAction: %v", err)
	}
	want := []string{"cvbi", "b+", "/total xdef"}
	notWant := []string{"local@", "local!"}
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
