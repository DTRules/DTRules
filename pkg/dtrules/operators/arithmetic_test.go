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

// Integer and float arithmetic / math op coverage. Bigint and fixed
// already have their own suites (bigint_test.go, fixed_test.go).
// These tests pin the VALUE produced by each op — a dispatch typo
// that swapped + with - would pass the registration test but fail here.
//
// Covered ops:
//   Integer:  +, -, *, /, mod, abs, negate, max, min
//   Float:    f+, f-, f*, fdiv, fabs, fnegate, fmax, fmin,
//             ceiling, floor, roundto (roundto has its own file)

// runBinInt pushes two integers, runs op, returns the integer result.
func runBinInt(t *testing.T, op string, a, b int64) int64 {
	t.Helper()
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(a))
	state.DataPush(dtrules.GetRIntegerValue(b))
	o, ok := Get(dtrules.GetRName(op))
	if !ok {
		t.Fatalf("op %q not registered", op)
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	top, _ := state.DataPop()
	v, err := top.LongValue()
	if err != nil {
		t.Fatalf("LongValue: %v", err)
	}
	return v
}

func runBinFloat(t *testing.T, op string, a, b float64) float64 {
	t.Helper()
	state := newTestState()
	state.DataPush(dtrules.GetRDoubleValue(a))
	state.DataPush(dtrules.GetRDoubleValue(b))
	o, ok := Get(dtrules.GetRName(op))
	if !ok {
		t.Fatalf("op %q not registered", op)
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	top, _ := state.DataPop()
	v, err := top.DoubleValue()
	if err != nil {
		t.Fatalf("DoubleValue: %v", err)
	}
	return v
}

// TestIntegerArithmetic pins the behavior of every integer arithmetic
// op. Includes a "swap detection" case — if + and - got swapped in a
// refactor, different cases would flip sign and we'd notice.
func TestIntegerArithmetic(t *testing.T) {
	cases := []struct {
		op   string
		a, b int64
		want int64
	}{
		{"+", 10, 3, 13},
		{"+", -5, 5, 0},
		{"-", 10, 3, 7},
		{"-", 3, 10, -7},
		{"*", 4, 5, 20},
		{"*", -4, 5, -20},
		{"/", 20, 4, 5},
		{"/", -20, 4, -5},
		{"mod", 10, 3, 1},
		{"mod", 10, -3, 1}, // Go semantics: sign of dividend
		{"max", 5, 10, 10},
		{"max", 10, 5, 10},
		{"min", 5, 10, 5},
		{"min", 10, 5, 5},
	}
	for _, c := range cases {
		t.Run(c.op+"_"+itoa(c.a)+"_"+itoa(c.b), func(t *testing.T) {
			got := runBinInt(t, c.op, c.a, c.b)
			if got != c.want {
				t.Errorf("%s %d %d: got %d, want %d", c.op, c.a, c.b, got, c.want)
			}
		})
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// TestIntegerUnaryArith covers abs and negate for integers.
func TestIntegerUnaryArith(t *testing.T) {
	cases := []struct {
		op   string
		in   int64
		want int64
	}{
		{"abs", 5, 5},
		{"abs", -5, 5},
		{"abs", 0, 0},
		{"negate", 5, -5},
		{"negate", -5, 5},
		{"negate", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.op+"_"+itoa(c.in), func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRIntegerValue(c.in))
			o, _ := Get(dtrules.GetRName(c.op))
			if err := o.Execute(state); err != nil {
				t.Fatalf("%s: %v", c.op, err)
			}
			top, _ := state.DataPop()
			v, _ := top.LongValue()
			if v != c.want {
				t.Errorf("%s %d: got %d, want %d", c.op, c.in, v, c.want)
			}
		})
	}
}

// TestIntegerDivideByZero guards the documented error path.
func TestIntegerDivideByZero(t *testing.T) {
	for _, op := range []string{"/", "mod"} {
		t.Run(op, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRIntegerValue(10))
			state.DataPush(dtrules.GetRIntegerValue(0))
			o, _ := Get(dtrules.GetRName(op))
			if err := o.Execute(state); err == nil {
				t.Errorf("%s by zero should error", op)
			}
		})
	}
}

// TestIntegerOverflow guards the overflow detection added in opAdd /
// opSub / opMul.
func TestIntegerOverflow(t *testing.T) {
	cases := []struct {
		op   string
		a, b int64
	}{
		{"+", 1<<62 + 1, 1 << 62},    // positive overflow
		{"-", -(1 << 62), 1<<62 + 1}, // negative overflow
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRIntegerValue(c.a))
			state.DataPush(dtrules.GetRIntegerValue(c.b))
			o, _ := Get(dtrules.GetRName(c.op))
			if err := o.Execute(state); err == nil {
				t.Errorf("%s %d %d: expected overflow error", c.op, c.a, c.b)
			}
		})
	}
}

// TestFloatArithmetic pins the behavior of every float arithmetic op.
func TestFloatArithmetic(t *testing.T) {
	eps := 1e-9
	cases := []struct {
		op   string
		a, b float64
		want float64
	}{
		{"f+", 1.5, 2.5, 4.0},
		{"f-", 3.0, 1.5, 1.5},
		{"f*", 1.5, 2.0, 3.0},
		{"fdiv", 10.0, 4.0, 2.5},
		{"fmax", 1.5, 2.5, 2.5},
		{"fmin", 1.5, 2.5, 1.5},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			got := runBinFloat(t, c.op, c.a, c.b)
			diff := got - c.want
			if diff < -eps || diff > eps {
				t.Errorf("%s %v %v: got %v, want %v", c.op, c.a, c.b, got, c.want)
			}
		})
	}
}

// TestFloatUnaryArith covers fabs / fnegate / ceiling / floor.
func TestFloatUnaryArith(t *testing.T) {
	cases := []struct {
		op   string
		in   float64
		want float64
	}{
		{"fabs", 1.5, 1.5},
		{"fabs", -1.5, 1.5},
		{"fnegate", 1.5, -1.5},
		{"fnegate", -1.5, 1.5},
		{"ceiling", 1.1, 2.0},
		{"ceiling", -1.1, -1.0},
		{"floor", 1.9, 1.0},
		{"floor", -1.1, -2.0},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRDoubleValue(c.in))
			o, _ := Get(dtrules.GetRName(c.op))
			if err := o.Execute(state); err != nil {
				t.Fatalf("%s: %v", c.op, err)
			}
			top, _ := state.DataPop()
			v, _ := top.DoubleValue()
			if v != c.want {
				t.Errorf("%s %v: got %v, want %v", c.op, c.in, v, c.want)
			}
		})
	}
}

// TestFloatDivideByZeroMatrix guards fdiv's error path.
// (Named uniquely to avoid collision with operators_test.go's
// pre-existing TestFloatDivideByZero.)
func TestFloatDivideByZeroMatrix(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRDoubleValue(1.0))
	state.DataPush(dtrules.GetRDoubleValue(0.0))
	o, _ := Get(dtrules.GetRName("fdiv"))
	if err := o.Execute(state); err == nil {
		t.Error("fdiv by zero should error")
	}
}
