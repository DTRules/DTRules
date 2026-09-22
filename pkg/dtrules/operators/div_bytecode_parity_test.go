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
	"math"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

// Issue #1261: the bytecode VM's OpDiv must agree with the "/" operator the
// session path runs (opDiv): both operands truncated to integers, integer
// division truncating toward zero, and "Math Exception: Division by zero"
// when the (truncated) divisor is zero.

type divOperand struct {
	obj dtrules.Object
	val dtrules.Value
}

func divInt(v int64) divOperand {
	return divOperand{dtrules.GetRIntegerValue(v), dtrules.NewValueInteger(v)}
}

func divDbl(v float64) divOperand {
	return divOperand{dtrules.GetRDoubleValue(v), dtrules.NewValueDouble(v)}
}

// runOpDiv runs the registered "/" operator on the object stack.
func runOpDiv(t *testing.T, a, b divOperand) (int64, error) {
	t.Helper()
	state := interpreter.NewDTState(&mockSession{})
	op, ok := Get(dtrules.GetRName("/"))
	if !ok {
		t.Fatal(`"/" operator not registered`)
	}
	state.DataPush(a.obj)
	state.DataPush(b.obj)
	if err := op.Execute(state); err != nil {
		return 0, err
	}
	r, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	if r.Type() != dtrules.TypeInteger {
		t.Fatalf("opDiv returned %v, want an integer", r.Type())
	}
	v, _ := r.LongValue()
	return v, nil
}

// runVMDiv runs `a b OpDiv` through the Go bytecode VM. The assembly
// dispatch loop (ExecuteBytecodeASM) is deliberately not exercised: it
// addresses a DTState layout that no longer exists, so calling it reads and
// writes outside the struct.
func runVMDiv(t *testing.T, a, b divOperand) (dtrules.Value, error) {
	t.Helper()
	state := interpreter.NewDTState(&mockSession{})
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(a.val)
	bc.EmitPushConstant(b.val)
	bc.Emit(dtrules.OpDiv)
	if err := state.ExecuteBytecode(bc); err != nil {
		return dtrules.ValueNull, err
	}
	r, perr := state.ValuePop()
	if perr != nil {
		t.Fatalf("ValuePop: %v", perr)
	}
	return r, nil
}

func TestBytecodeOpDivMatchesOperator(t *testing.T) {
	cases := []struct {
		name string
		a, b divOperand
	}{
		{"7 / 2", divInt(7), divInt(2)},
		{"-7 / 2", divInt(-7), divInt(2)},
		{"7.0 / 2", divDbl(7), divInt(2)},
		{"7 / 2.0", divInt(7), divDbl(2)},
		{"7.9 / 2.0", divDbl(7.9), divDbl(2)},
		{"-7.9 / 2", divDbl(-7.9), divInt(2)},
		{"MinInt64 / -1", divInt(math.MinInt64), divInt(-1)},
		{"MinInt64 / -1.0", divInt(math.MinInt64), divDbl(-1)},
		{"1 / 0", divInt(1), divInt(0)},
		{"1.0 / 0.0", divDbl(1), divDbl(0)},
		{"1 / 0.5", divInt(1), divDbl(0.5)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want, wantErr := runOpDiv(t, c.a, c.b)
			got, err := runVMDiv(t, c.a, c.b)
			if wantErr != nil {
				if err == nil || err.Error() != wantErr.Error() {
					t.Fatalf("error = %v, want %v (opDiv); result %v", err, wantErr, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v, want %d (opDiv)", err, want)
			}
			if !got.IsInteger() || got.AsInteger() != want {
				t.Errorf("result = %v (integer %v), want integer %d (opDiv)", got, got.IsInteger(), want)
			}
		})
	}
}

// A non-numeric operand is a conversion error, not a reinterpretation of
// the value's bits.
func TestBytecodeOpDivRejectsNonNumeric(t *testing.T) {
	s := divOperand{dtrules.NewRString("x"), dtrules.NewValueString("x")}
	for _, pair := range [][2]divOperand{{s, divInt(2)}, {divInt(2), s}} {
		got, err := runVMDiv(t, pair[0], pair[1])
		if err == nil {
			t.Errorf("dividing with a string gave %v, want an error", got)
		} else if !strings.Contains(err.Error(), "Conversion") {
			t.Errorf("error = %v, want a conversion error", err)
		}
	}
}
