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
	"math"
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

func TestFpSubOperator(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "3.75")
	pushFp(t, state, "1.50")
	if err := mustOp(t, "fp-").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "2.25000000" {
		t.Errorf("3.75 - 1.5 = %q, want 2.25000000", got)
	}
}

func TestFpAbsOperator(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "-1.23456789")
	if err := mustOp(t, "fpabs").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "1.23456789" {
		t.Errorf("fpabs(-1.23456789) = %q, want 1.23456789", got)
	}
}

func TestFpNegateOperator(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "1.5")
	if err := mustOp(t, "fpnegate").Execute(state); err != nil {
		t.Fatal(err)
	}
	if got := popFpString(t, state); got != "-1.50000000" {
		t.Errorf("fpnegate(1.5) = %q, want -1.50000000", got)
	}
}

func TestFpTruncOperator(t *testing.T) {
	state := newFixedTestState()
	pushFp(t, state, "-1.99999999")
	if err := mustOp(t, "fptrunc").Execute(state); err != nil {
		t.Fatal(err)
	}
	// Truncate toward zero: -1.99999999 → -1.00000000 (not -2)
	if got := popFpString(t, state); got != "-1.00000000" {
		t.Errorf("fptrunc(-1.99999999) = %q, want -1.00000000 (truncate toward zero)", got)
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

// runFpCmp pushes a and b onto the stack, runs op, and returns the boolean on top.
func runFpCmp(t *testing.T, op, a, b string) bool {
	t.Helper()
	state := newFixedTestState()
	pushFp(t, state, a)
	pushFp(t, state, b)
	if err := mustOp(t, op).Execute(state); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	top, err := state.DataPop()
	if err != nil {
		t.Fatal(err)
	}
	got, err := top.BooleanValue()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// TestFpComparisonOperators exercises every registered fp comparison operator
// on a matrix of positive/negative/equal/zero-crossing pairs so a dispatch-
// table typo on any of fp==, fp!=, fp<, fp<=, fp>, fp>= would fail.
func TestFpComparisonOperators(t *testing.T) {
	type tri struct {
		lt, eq, gt bool // expected for left vs right
	}
	cases := []struct {
		a, b string
		want tri
	}{
		{"1.5", "1.5", tri{eq: true}},
		{"1.5", "2.5", tri{lt: true}},
		{"2.5", "1.5", tri{gt: true}},
		{"-1.5", "1.5", tri{lt: true}},
		{"0.00000001", "0", tri{gt: true}},
		{"-0.00000001", "0", tri{lt: true}},
	}
	for _, c := range cases {
		name := c.a + "_vs_" + c.b
		t.Run(name, func(t *testing.T) {
			if got, want := runFpCmp(t, "fp==", c.a, c.b), c.want.eq; got != want {
				t.Errorf("fp== %s %s: got %v, want %v", c.a, c.b, got, want)
			}
			if got, want := runFpCmp(t, "fp!=", c.a, c.b), !c.want.eq; got != want {
				t.Errorf("fp!= %s %s: got %v, want %v", c.a, c.b, got, want)
			}
			if got, want := runFpCmp(t, "fp<", c.a, c.b), c.want.lt; got != want {
				t.Errorf("fp< %s %s: got %v, want %v", c.a, c.b, got, want)
			}
			if got, want := runFpCmp(t, "fp<=", c.a, c.b), c.want.lt || c.want.eq; got != want {
				t.Errorf("fp<= %s %s: got %v, want %v", c.a, c.b, got, want)
			}
			if got, want := runFpCmp(t, "fp>", c.a, c.b), c.want.gt; got != want {
				t.Errorf("fp> %s %s: got %v, want %v", c.a, c.b, got, want)
			}
			if got, want := runFpCmp(t, "fp>=", c.a, c.b), c.want.gt || c.want.eq; got != want {
				t.Errorf("fp>= %s %s: got %v, want %v", c.a, c.b, got, want)
			}
		})
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

func TestCvfpFromNaNErrors(t *testing.T) {
	state := newFixedTestState()
	state.DataPush(dtrules.GetRDoubleValue(math.NaN()))
	if err := mustOp(t, "cvfp").Execute(state); err == nil {
		t.Error("expected cvfp to reject NaN")
	}
}

func TestCvfpFromInfErrors(t *testing.T) {
	for _, f := range []float64{math.Inf(1), math.Inf(-1)} {
		state := newFixedTestState()
		state.DataPush(dtrules.GetRDoubleValue(f))
		if err := mustOp(t, "cvfp").Execute(state); err == nil {
			t.Errorf("expected cvfp to reject %v", f)
		}
	}
}

// TestCvfpFromUnsupportedTypeErrors guards the `default:` arm of opCvFixed —
// boolean, entity, array, null, etc. must all error rather than silently
// snap onto the grid.
func TestCvfpFromUnsupportedTypeErrors(t *testing.T) {
	unsupported := []dtrules.Object{
		dtrules.GetRBoolean(true),
		dtrules.GetRNull(),
	}
	for _, obj := range unsupported {
		state := newFixedTestState()
		state.DataPush(obj)
		err := mustOp(t, "cvfp").Execute(state)
		if err == nil {
			t.Errorf("expected cvfp to reject %s", obj.Type().String())
			continue
		}
		if !strings.Contains(err.Error(), "cannot convert") {
			t.Errorf("cvfp %s error should mention 'cannot convert': %v",
				obj.Type().String(), err)
		}
	}
}

// TestCvbOnFixed guards cvb → boolean coercion for RFixed, matching the
// "0 → false, non-zero → true" rule the op applies to integer and bigint.
func TestCvbOnFixed(t *testing.T) {
	cases := []struct {
		lit  string
		want bool
	}{
		{"0", false},
		{"0.00000000", false},
		{"0.00000001", true},
		{"-0.00000001", true},
		{"1.5", true},
		{"-1.5", true},
	}
	for _, c := range cases {
		t.Run(c.lit, func(t *testing.T) {
			state := newFixedTestState()
			pushFp(t, state, c.lit)
			if err := mustOp(t, "cvb").Execute(state); err != nil {
				t.Fatalf("cvb: %v", err)
			}
			top, err := state.DataPop()
			if err != nil {
				t.Fatal(err)
			}
			got, err := top.BooleanValue()
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("cvb(%sfp) = %v, want %v", c.lit, got, c.want)
			}
		})
	}
}
