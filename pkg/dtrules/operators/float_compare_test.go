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

package operators

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Float comparison ops (f>, f>=, f<, f<=, f==, f!=). The EL compiler emits
// these for double-typed operands; before they were registered the runtime
// treated the token as an undefined name lookup and execution failed.

// runCmpFloat pushes two doubles, runs op, and returns the boolean result.
func runCmpFloat(t *testing.T, op string, a, b float64) bool {
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
	v, err := top.BooleanValue()
	if err != nil {
		t.Fatalf("BooleanValue: %v", err)
	}
	return v
}

func TestFloatComparisons(t *testing.T) {
	cases := []struct {
		op   string
		a, b float64
		want bool
	}{
		{"f>", 1.85, 1.4, true},
		{"f>", 1.4, 1.85, false},
		{"f>", 1.4, 1.4, false},
		{"f>=", 1.4, 1.4, true},
		{"f>=", 1.39, 1.4, false},
		{"f<", 48.02, 50.0, true},
		{"f<", 50.0, 48.02, false},
		{"f<", 50.0, 50.0, false},
		{"f<=", 50.0, 50.0, true},
		{"f<=", 50.01, 50.0, false},
		{"f==", 1.5, 1.5, true},
		{"f==", 1.5, 1.6, false},
		{"f!=", 1.5, 1.6, true},
		{"f!=", 1.5, 1.5, false},
	}
	for _, c := range cases {
		got := runCmpFloat(t, c.op, c.a, c.b)
		if got != c.want {
			t.Errorf("%g %s %g = %v, want %v", c.a, c.op, c.b, got, c.want)
		}
	}
}

// TestFloatComparisonCoercesInteger confirms a double field compared against
// an integer literal (mixed operand types on the stack) coerces cleanly.
func TestFloatComparisonCoercesInteger(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRDoubleValue(2.5))
	state.DataPush(dtrules.GetRIntegerValue(2))
	o, ok := Get(dtrules.GetRName("f>"))
	if !ok {
		t.Fatal("f> not registered")
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("f>: %v", err)
	}
	top, _ := state.DataPop()
	v, err := top.BooleanValue()
	if err != nil {
		t.Fatalf("BooleanValue: %v", err)
	}
	if !v {
		t.Errorf("2.5 f> 2 = false, want true")
	}
}
