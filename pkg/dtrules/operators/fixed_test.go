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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

func newFixedTestState() *interpreter.DTState {
	return interpreter.NewDTState(&mockSession{})
}

func mustOp(t *testing.T, name string) dtrules.Object {
	t.Helper()
	op, ok := Get(dtrules.GetRName(name))
	if !ok {
		t.Fatalf("operator %q not registered", name)
	}
	return op
}

func pushFp(t *testing.T, state dtrules.State, s string) {
	t.Helper()
	fp, err := dtrules.GetRFixedFromString(s)
	if err != nil {
		t.Fatalf("GetRFixedFromString(%q): %v", s, err)
	}
	if err := state.DataPush(fp); err != nil {
		t.Fatalf("DataPush: %v", err)
	}
}

func popFpString(t *testing.T, state dtrules.State) string {
	t.Helper()
	top, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	fp, ok := top.(*dtrules.RFixed)
	if !ok {
		t.Fatalf("expected RFixed on stack, got %T", top)
	}
	return fp.StringValue()
}

// =============================================================================
// fp arithmetic operators
// =============================================================================

func TestFpAddOperator(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	pushFp(t, state, "2.25")
	if err := mustOp(t, "fp+").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "3.75000000" {
		t.Errorf("1.5 + 2.25 = %q, want 3.75000000", got)
	}
}

func TestFpMulTruncatesTowardZero(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.00000001")
	pushFp(t, state, "1.00000001")
	if err := mustOp(t, "fp*").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "1.00000002" {
		t.Errorf("1.00000001^2 = %q, want 1.00000002", got)
	}
}

func TestFpDivByZeroErrors(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1")
	pushFp(t, state, "0")
	if err := mustOp(t, "fp/").Execute(state); err == nil {
		t.Error("expected divide-by-zero error")
	}
}

func TestFpAddPromotesInteger(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	if err := state.DataPush(dtrules.GetRIntegerValue(2)); err != nil {
		t.Fatal(err)
	}
	if err := mustOp(t, "fp+").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "3.50000000" {
		t.Errorf("1.5 + int(2) = %q, want 3.50000000", got)
	}
}

func TestFpAddPromotesBigInt(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	if err := state.DataPush(dtrules.GetRBigIntFromInt64(10)); err != nil {
		t.Fatal(err)
	}
	if err := mustOp(t, "fp+").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "11.50000000" {
		t.Errorf("1.5 + bigint(10) = %q, want 11.50000000", got)
	}
}

func TestFpAddRejectsDouble(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	if err := state.DataPush(dtrules.GetRDoubleValue(2.5)); err != nil {
		t.Fatal(err)
	}
	err := mustOp(t, "fp+").Execute(state)
	if err == nil {
		t.Fatal("expected error mixing fp with double")
	}
	if !strings.Contains(err.Error(), "cvfp") {
		t.Errorf("error should mention cvfp: %v", err)
	}
}

// =============================================================================
// fp comparison operators
// =============================================================================

func TestFpEqualAndLess(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	pushFp(t, state, "1.5")
	if err := mustOp(t, "fp==").Execute(state); err != nil {
		t.Fatal(err)
	}
	top, _ := state.DataPop()
	if b, _ := top.BooleanValue(); !b {
		t.Error("fp==: 1.5 == 1.5 should be true")
	}

	pushFp(t, state, "1.5")
	pushFp(t, state, "2.5")
	if err := mustOp(t, "fp<").Execute(state); err != nil {
		t.Fatal(err)
	}
	top, _ = state.DataPop()
	if b, _ := top.BooleanValue(); !b {
		t.Error("fp<: 1.5 < 2.5 should be true")
	}
}

// =============================================================================
// cvfp cast operator
// =============================================================================

func TestCvfpFromInteger(t *testing.T) {
	state := newFixedTestState()
	state.DataPush(dtrules.GetRIntegerValue(42))
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "42.00000000" {
		t.Errorf("cvfp(int 42) = %q", got)
	}
}

func TestCvfpFromBigInt(t *testing.T) {
	state := newFixedTestState()
	state.DataPush(dtrules.GetRBigIntFromInt64(1_000_000))
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "1000000.00000000" {
		t.Errorf("cvfp(bigint 1e6) = %q", got)
	}
}

func TestCvfpFromDoubleSnapsToGrid(t *testing.T) {
	// 1.5 is exactly representable in float64; cvfp must give exactly 1.50000000.
	state := newFixedTestState()
	state.DataPush(dtrules.GetRDoubleValue(1.5))
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "1.50000000" {
		t.Errorf("cvfp(1.5) = %q", got)
	}

	// 0.1 is NOT exactly representable — stored as 0.10000000000000000555...
	// Multiplied by 10^8 that's 10000000.00000000555..., which truncates to
	// 10000000, giving StringValue "0.10000000". The author opted in to this.
	state.DataPush(dtrules.GetRDoubleValue(0.1))
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "0.10000000" {
		t.Errorf("cvfp(0.1) = %q, want 0.10000000", got)
	}
}

func TestCvfpFromDoubleTruncatesTowardZero(t *testing.T) {
	// A negative double slightly short of an integer satoshi should
	// truncate toward zero, not away.
	state := newFixedTestState()
	state.DataPush(dtrules.GetRDoubleValue(-0.099999999))
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	// -0.099999999 * 1e8 = -9999999.9... truncate-toward-zero → -9999999 → -0.09999999
	if got := popFpString(t, state); got != "-0.09999999" {
		t.Errorf("cvfp(-0.099999999) = %q, want -0.09999999", got)
	}
}

func TestCvfpFromString(t *testing.T) {
	state := newFixedTestState()
	state.DataPush(dtrules.NewRString("1680748.45091643"))
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "1680748.45091643" {
		t.Errorf("cvfp(\"1680748.45091643\") = %q", got)
	}
}

func TestCvfpIdempotent(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	if err := mustOp(t, "cvfp").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "1.50000000" {
		t.Errorf("cvfp(fp 1.5) = %q", got)
	}
}
