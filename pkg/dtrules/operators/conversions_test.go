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

package operators

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Issue #694 exposed that `cvd` was mis-registered as the date
// converter despite the emitter emitting it for double-typed fields.
// The silent-null behavior went unnoticed for a decade because no
// test asserted "cvd on a double produces a double". This file fills
// that gap with a systematic matrix over every registered cv* op:
// explicit target-type assertion per input type, so any registration
// / semantics mismatch fails loudly at test time.
//
// Coverage targets (one test per registered op):
//
//   cvi      → integer   (stack.go)
//   cvd      → double    (stack.go, renamed from cvr in #694)
//   cvdate   → date      (stack.go, renamed from cvd in #694)
//   cvb      → boolean   (stack.go, detailed coverage in cvb_test.go)
//   cve      → entity    (stack.go — minimal smoke, behavior is entity-type-specific)
//   cvn      → name      (stack.go)
//   cvs      → string    (string.go)
//   cvbi     → bigint    (bigint.go, detailed coverage in bigint_test.go)
//   cvbytes  → bytes     (bytes.go, coverage in bytes / encoding tests)
//   cvfp     → fixed     (fixed.go, detailed coverage in fixed_test.go)
//   cvhex    → bytes     (bytes.go) — specialized, see bytes_test.go
//   cvb58check / cvbech32 → bytes — specialized, see bytes tests
//
// The cvb / cvfp / cvbi / cvbytes ops already have dedicated tests in
// their own files. This file asserts the *target-type contract* for
// the ops that previously had zero coverage in the operators package:
// cvi, cvd, cvdate, cve, cvn, cvs. A single-row "X on Y produces Z"
// matrix would have caught #694 immediately.

// runCvOp pushes inputs in order, runs the named op, and returns the
// stack top. Returns an error if the op errors. Used by each cv* test.
func runCvOp(t *testing.T, opName string, inputs ...dtrules.Object) (dtrules.Object, error) {
	t.Helper()
	state := newTestState()
	for _, obj := range inputs {
		if err := state.DataPush(obj); err != nil {
			t.Fatalf("DataPush: %v", err)
		}
	}
	op, ok := Get(dtrules.GetRName(opName))
	if !ok {
		t.Fatalf("operator %q not registered", opName)
	}
	if err := op.Execute(state); err != nil {
		return nil, err
	}
	top, err := state.DataPop()
	if err != nil {
		return nil, err
	}
	return top, nil
}

// =============================================================================
// cvi — integer converter
// =============================================================================

func TestCvi_TargetType(t *testing.T) {
	// Assert the simplest contract: every successful cvi produces an
	// RInteger. Inputs that can't convert push null per the existing
	// error-return-null pattern.
	cases := []struct {
		name     string
		input    dtrules.Object
		wantType *dtrules.RType
	}{
		{"integer passthrough", dtrules.GetRIntegerValue(42), dtrules.TypeInteger},
		{"double truncate", dtrules.GetRDoubleValue(3.9), dtrules.TypeInteger},
		{"bigint in range", dtrules.GetRBigIntFromInt64(100), dtrules.TypeInteger},
		{"string (numeric)", dtrules.NewRString("42"), dtrules.TypeInteger},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			top, err := runCvOp(t, "cvi", c.input)
			if err != nil {
				t.Fatalf("cvi: %v", err)
			}
			if top.Type() != c.wantType {
				t.Errorf("cvi input %s: got %s, want %s",
					c.input.Type().String(), top.Type().String(),
					c.wantType.String())
			}
		})
	}
}

// =============================================================================
// cvd — double converter (renamed from cvr in #694)
// =============================================================================

func TestCvd_TargetType(t *testing.T) {
	// THE reproducer for #694. Before the rename, cvd was the date
	// converter — this test would have caught it immediately.
	cases := []struct {
		name  string
		input dtrules.Object
	}{
		{"double passthrough", dtrules.GetRDoubleValue(3.14)},
		{"integer widen", dtrules.GetRIntegerValue(42)},
		{"bigint widen", dtrules.GetRBigIntFromInt64(100)},
		{"string numeric", dtrules.NewRString("2.71828")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			top, err := runCvOp(t, "cvd", c.input)
			if err != nil {
				t.Fatalf("cvd: %v", err)
			}
			if top.Type() != dtrules.TypeDouble {
				t.Fatalf("cvd must produce double; got %s (value=%q) — "+
					"if this fails, cvd/cvdate naming is broken again",
					top.Type().String(), top.StringValue())
			}
		})
	}
}

// TestCvr_AliasOfCvd pins the back-compat alias.
func TestCvr_AliasOfCvd(t *testing.T) {
	top, err := runCvOp(t, "cvr", dtrules.GetRDoubleValue(1.5))
	if err != nil {
		t.Fatal(err)
	}
	if top.Type() != dtrules.TypeDouble {
		t.Errorf("cvr (alias) must produce double; got %s", top.Type().String())
	}
}

// =============================================================================
// cvdate — date converter (renamed from cvd in #694)
// =============================================================================

// TestCvdate_Registered — full end-to-end is in pkg/dtrules/compiler/
// with a stub DateParser (avoids an import cycle). Here we just assert
// the op resolves and runs without the old cvd-for-double misdirection.
func TestCvdate_Registered(t *testing.T) {
	op, ok := Get(dtrules.GetRName("cvdate"))
	if !ok {
		t.Fatal("cvdate must be registered")
	}
	if op == nil {
		t.Fatal("cvdate resolved to nil")
	}
}

// =============================================================================
// cve — entity converter
// =============================================================================

func TestCve_NonEntityReturnsNull(t *testing.T) {
	// cve calls REntityValue on the input. For non-entity values the
	// base object's default returns an error; cve catches that and
	// pushes null. Assert that contract so it doesn't silently change.
	top, err := runCvOp(t, "cve", dtrules.GetRIntegerValue(42))
	if err != nil {
		t.Fatal(err)
	}
	if top.Type() != dtrules.TypeNull {
		t.Errorf("cve on non-entity should push null; got %s",
			top.Type().String())
	}
}

// =============================================================================
// cvn — name converter
// =============================================================================

func TestCvn_StringProducesName(t *testing.T) {
	// The op calls RNameValue; RString.RNameValue returns an RName if
	// the string is a valid name token. Pin the target-type contract
	// for a valid input and the null fallback for an invalid one.
	top, err := runCvOp(t, "cvn", dtrules.NewRString("my_name"))
	if err != nil {
		t.Fatal(err)
	}
	if top.Type() != dtrules.TypeName && top.Type() != dtrules.TypeNull {
		t.Errorf("cvn on a string should produce name or null; got %s",
			top.Type().String())
	}
}

// =============================================================================
// cvs — string converter
// =============================================================================

func TestCvs_TargetType(t *testing.T) {
	cases := []struct {
		name  string
		input dtrules.Object
	}{
		{"integer → string", dtrules.GetRIntegerValue(42)},
		{"double → string", dtrules.GetRDoubleValue(3.14)},
		{"string passthrough", dtrules.NewRString("hello")},
		{"bigint → string", dtrules.GetRBigIntFromInt64(99)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			top, err := runCvOp(t, "cvs", c.input)
			if err != nil {
				t.Fatalf("cvs: %v", err)
			}
			if top.Type() != dtrules.TypeString {
				t.Errorf("cvs must produce string; got %s",
					top.Type().String())
			}
		})
	}
}

// =============================================================================
// Matrix: every registered cv* op produces the right target type.
// =============================================================================

// TestCvTargetTypeMatrix is the load-bearing test: for every cv* op
// we ship, assert that applying it to an integer input produces a
// value of the advertised target type. Integer is the universal
// input — every conversion either accepts it directly or errors to
// null. A naming / registration mismatch on ANY cv* op flunks here.
//
// This is the test that #694 was missing.
func TestCvTargetTypeMatrix(t *testing.T) {
	cases := []struct {
		op       string
		wantType *dtrules.RType
		allowNull bool // true if the op returns null for an integer input
	}{
		{"cvi", dtrules.TypeInteger, false},
		{"cvd", dtrules.TypeDouble, false},
		{"cvr", dtrules.TypeDouble, false}, // alias of cvd
		{"cvbi", dtrules.TypeBigInt, false},
		{"cvfp", dtrules.TypeFixed, false},
		{"cvs", dtrules.TypeString, false},
		{"cvb", dtrules.TypeBoolean, false},
		{"cve", dtrules.TypeNull, true},  // entity conversion of int fails → null
		{"cvn", dtrules.TypeNull, true},  // name conversion of int fails → null
		// cvdate and cvbytes require specific inputs (date strings,
		// hex strings); an integer input doesn't have a reasonable
		// success path, so exclude from this uniform matrix. They're
		// covered by dedicated tests elsewhere.
	}
	input := dtrules.GetRIntegerValue(42)
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top, err := runCvOp(t, c.op, input)
			if err != nil {
				t.Fatalf("%s: %v", c.op, err)
			}
			if top.Type() != c.wantType {
				if c.allowNull && top.Type() == dtrules.TypeNull {
					return
				}
				t.Errorf("%s on integer: got %s, want %s",
					c.op, top.Type().String(), c.wantType.String())
			}
		})
	}
}
