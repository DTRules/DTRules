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
	"math/big"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

// newBigIntTestState creates a fresh state for testing
func newBigIntTestState() *interpreter.DTState {
	return interpreter.NewDTState(&mockSession{})
}

// =============================================================================
// BigInt Arithmetic Operators
// =============================================================================

func TestBigIntAddOperator(t *testing.T) {
	state := newBigIntTestState()

	// Push two BigInt values
	state.DataPush(dtrules.GetRBigIntFromInt64(100))
	state.DataPush(dtrules.GetRBigIntFromInt64(200))

	op, ok := Get(dtrules.GetRName("b+"))
	if !ok {
		t.Fatal("b+ operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b+ operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	bi, err := result.RBigIntValue()
	if err != nil {
		t.Fatalf("RBigIntValue failed: %v", err)
	}

	if bi.StringValue() != "300" {
		t.Errorf("Expected 300, got %s", bi.StringValue())
	}

	// Stack should be empty
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got size %d", state.DataStackDepth())
	}
}

func TestBigIntAddLargeValues(t *testing.T) {
	state := newBigIntTestState()

	// Create values larger than int64
	a, _ := dtrules.GetRBigIntFromString("99999999999999999999999999999999")
	b, _ := dtrules.GetRBigIntFromString("1")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("b+"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b+ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	expected := "100000000000000000000000000000000"
	if bi.StringValue() != expected {
		t.Errorf("Expected %s, got %s", expected, bi.StringValue())
	}
}

func TestBigIntAddAlias(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(10))
	state.DataPush(dtrules.GetRBigIntFromInt64(20))

	// Test alias "bigadd"
	op, ok := Get(dtrules.GetRName("bigadd"))
	if !ok {
		t.Fatal("bigadd alias not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("bigadd operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "30" {
		t.Errorf("Expected 30, got %s", bi.StringValue())
	}
}

func TestBigIntSubOperator(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(500))
	state.DataPush(dtrules.GetRBigIntFromInt64(200))

	op, _ := Get(dtrules.GetRName("b-"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "300" {
		t.Errorf("Expected 300, got %s", bi.StringValue())
	}
}

func TestBigIntSubNegativeResult(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(100))
	state.DataPush(dtrules.GetRBigIntFromInt64(300))

	op, _ := Get(dtrules.GetRName("b-"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "-200" {
		t.Errorf("Expected -200, got %s", bi.StringValue())
	}
}

func TestBigIntSubLargeValues(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("100000000000000000000000000000000")
	b, _ := dtrules.GetRBigIntFromString("1")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("b-"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	expected := "99999999999999999999999999999999"
	if bi.StringValue() != expected {
		t.Errorf("Expected %s, got %s", expected, bi.StringValue())
	}
}

func TestBigIntMulOperator(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(6))
	state.DataPush(dtrules.GetRBigIntFromInt64(7))

	op, _ := Get(dtrules.GetRName("b*"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b* operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "42" {
		t.Errorf("Expected 42, got %s", bi.StringValue())
	}
}

func TestBigIntMulLargeValues(t *testing.T) {
	state := newBigIntTestState()

	// Multiply two 50-digit numbers
	a, _ := dtrules.GetRBigIntFromString("99999999999999999999999999999999999999999999999999")
	b, _ := dtrules.GetRBigIntFromString("2")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("b*"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b* operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	expected := "199999999999999999999999999999999999999999999999998"
	if bi.StringValue() != expected {
		t.Errorf("Expected %s, got %s", expected, bi.StringValue())
	}
}

func TestBigIntMulByZero(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("123456789012345678901234567890")
	state.DataPush(a)
	state.DataPush(dtrules.GetRBigIntFromInt64(0))

	op, _ := Get(dtrules.GetRName("b*"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b* operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "0" {
		t.Errorf("Expected 0, got %s", bi.StringValue())
	}
}

func TestBigIntDivOperator(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(100))
	state.DataPush(dtrules.GetRBigIntFromInt64(5))

	op, _ := Get(dtrules.GetRName("b/"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b/ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "20" {
		t.Errorf("Expected 20, got %s", bi.StringValue())
	}
}

func TestBigIntDivTruncates(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(10))
	state.DataPush(dtrules.GetRBigIntFromInt64(3))

	op, _ := Get(dtrules.GetRName("b/"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b/ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	// Integer division: 10 / 3 = 3 (truncated)
	if bi.StringValue() != "3" {
		t.Errorf("Expected 3 (truncated), got %s", bi.StringValue())
	}
}

func TestBigIntDivByZero(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(100))
	state.DataPush(dtrules.GetRBigIntFromInt64(0))

	op, _ := Get(dtrules.GetRName("b/"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected divide by zero error")
	}
}

func TestBigIntDivLargeValues(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("1000000000000000000000000000000000000000000000000000")
	b, _ := dtrules.GetRBigIntFromString("1000000000000000000000000")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("b/"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b/ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	// 10^51 / 10^24 = 10^27
	expected := "1000000000000000000000000000"
	if bi.StringValue() != expected {
		t.Errorf("Expected %s, got %s", expected, bi.StringValue())
	}
}

func TestBigIntModOperator(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(17))
	state.DataPush(dtrules.GetRBigIntFromInt64(5))

	op, _ := Get(dtrules.GetRName("bmod"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("bmod operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.StringValue() != "2" {
		t.Errorf("Expected 2, got %s", bi.StringValue())
	}
}

func TestBigIntModByZero(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(17))
	state.DataPush(dtrules.GetRBigIntFromInt64(0))

	op, _ := Get(dtrules.GetRName("bmod"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected mod by zero error")
	}
}

func TestBigIntModLargeValues(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("123456789012345678901234567890123")
	b, _ := dtrules.GetRBigIntFromString("1000000000000000000")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("bmod"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("bmod operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	// Calculate expected: 123456789012345678901234567890123 mod 1000000000000000000
	expected := new(big.Int)
	a_big, _ := new(big.Int).SetString("123456789012345678901234567890123", 10)
	b_big, _ := new(big.Int).SetString("1000000000000000000", 10)
	expected.Mod(a_big, b_big)

	if bi.StringValue() != expected.String() {
		t.Errorf("Expected %s, got %s", expected.String(), bi.StringValue())
	}
}

func TestBigIntAbsOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"positive stays positive", 42, "42"},
		{"negative becomes positive", -42, "42"},
		{"zero stays zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.input))

			op, _ := Get(dtrules.GetRName("babs"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("babs operator failed: %v", err)
			}

			result, _ := state.DataPop()
			bi, _ := result.RBigIntValue()
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
		})
	}
}

func TestBigIntAbsLargeNegative(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("-99999999999999999999999999999999")
	state.DataPush(a)

	op, _ := Get(dtrules.GetRName("babs"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("babs operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	expected := "99999999999999999999999999999999"
	if bi.StringValue() != expected {
		t.Errorf("Expected %s, got %s", expected, bi.StringValue())
	}
}

func TestBigIntNegateOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"positive becomes negative", 42, "-42"},
		{"negative becomes positive", -42, "42"},
		{"zero stays zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.input))

			op, _ := Get(dtrules.GetRName("bnegate"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("bnegate operator failed: %v", err)
			}

			result, _ := state.DataPop()
			bi, _ := result.RBigIntValue()
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
		})
	}
}

func TestBigIntNegateLargeValue(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("123456789012345678901234567890")
	state.DataPush(a)

	op, _ := Get(dtrules.GetRName("bnegate"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("bnegate operator failed: %v", err)
	}

	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()

	expected := "-123456789012345678901234567890"
	if bi.StringValue() != expected {
		t.Errorf("Expected %s, got %s", expected, bi.StringValue())
	}
}

// =============================================================================
// BigInt Comparison Operators
// =============================================================================

func TestBigIntEqualOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected bool
	}{
		{"equal values", 42, 42, true},
		{"unequal values", 42, 43, false},
		{"both zero", 0, 0, true},
		{"positive vs negative", 5, -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.a))
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.b))

			op, _ := Get(dtrules.GetRName("b=="))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("b== operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

func TestBigIntEqualLargeValues(t *testing.T) {
	state := newBigIntTestState()

	a, _ := dtrules.GetRBigIntFromString("123456789012345678901234567890")
	b, _ := dtrules.GetRBigIntFromString("123456789012345678901234567890")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("b=="))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b== operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("Expected large equal values to be equal")
	}
}

func TestBigIntNotEqualOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected bool
	}{
		{"equal values", 42, 42, false},
		{"unequal values", 42, 43, true},
		{"both zero", 0, 0, false},
		{"positive vs negative", 5, -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.a))
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.b))

			op, _ := Get(dtrules.GetRName("b!="))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("b!= operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

func TestBigIntNotEqualAlias(t *testing.T) {
	state := newBigIntTestState()

	state.DataPush(dtrules.GetRBigIntFromInt64(42))
	state.DataPush(dtrules.GetRBigIntFromInt64(43))

	// Test alias "b<>"
	op, ok := Get(dtrules.GetRName("b<>"))
	if !ok {
		t.Fatal("b<> alias not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b<> operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("Expected 42 != 43 to be true")
	}
}

func TestBigIntGreaterOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected bool
	}{
		{"a > b", 10, 5, true},
		{"a < b", 5, 10, false},
		{"a == b", 5, 5, false},
		{"negative comparison", -5, -10, true},
		{"mixed signs", 5, -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.a))
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.b))

			op, _ := Get(dtrules.GetRName("b>"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("b> operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

func TestBigIntGreaterEqualOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected bool
	}{
		{"a > b", 10, 5, true},
		{"a < b", 5, 10, false},
		{"a == b", 5, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.a))
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.b))

			op, _ := Get(dtrules.GetRName("b>="))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("b>= operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

func TestBigIntLessOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected bool
	}{
		{"a < b", 5, 10, true},
		{"a > b", 10, 5, false},
		{"a == b", 5, 5, false},
		{"negative comparison", -10, -5, true},
		{"mixed signs", -5, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.a))
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.b))

			op, _ := Get(dtrules.GetRName("b<"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("b< operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

func TestBigIntLessEqualOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected bool
	}{
		{"a < b", 5, 10, true},
		{"a > b", 10, 5, false},
		{"a == b", 5, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newBigIntTestState()
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.a))
			state.DataPush(dtrules.GetRBigIntFromInt64(tt.b))

			op, _ := Get(dtrules.GetRName("b<="))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("b<= operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

func TestBigIntComparisonLargeValues(t *testing.T) {
	state := newBigIntTestState()

	// Compare two large values that differ by 1
	a, _ := dtrules.GetRBigIntFromString("123456789012345678901234567891")
	b, _ := dtrules.GetRBigIntFromString("123456789012345678901234567890")

	state.DataPush(a)
	state.DataPush(b)

	op, _ := Get(dtrules.GetRName("b>"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("b> operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("Expected larger value to be greater")
	}
}

// =============================================================================
// BigInt Type Conversion Operator (cvbi)
// =============================================================================

func TestBigIntCvbiOperator(t *testing.T) {
	t.Run("convert RInteger to BigInt", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.GetRIntegerValue(12345))

		op, ok := Get(dtrules.GetRName("cvbi"))
		if !ok {
			t.Fatal("cvbi operator not found")
		}

		err := op.Execute(state)
		if err != nil {
			t.Fatalf("cvbi operator failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.Type() != dtrules.TypeBigInt {
			t.Errorf("Expected TypeBigInt, got %v", result.Type())
		}

		bi, _ := result.RBigIntValue()
		if bi.StringValue() != "12345" {
			t.Errorf("Expected 12345, got %s", bi.StringValue())
		}
	})

	t.Run("convert RString to BigInt", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.NewRString("9876543210"))

		op, _ := Get(dtrules.GetRName("cvbi"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("cvbi operator failed: %v", err)
		}

		result, _ := state.DataPop()
		bi, _ := result.RBigIntValue()
		if bi.StringValue() != "9876543210" {
			t.Errorf("Expected 9876543210, got %s", bi.StringValue())
		}
	})

	t.Run("convert large string to BigInt", func(t *testing.T) {
		state := newBigIntTestState()
		largeNum := "123456789012345678901234567890123456789012345678901234567890"
		state.DataPush(dtrules.NewRString(largeNum))

		op, _ := Get(dtrules.GetRName("cvbi"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("cvbi operator failed: %v", err)
		}

		result, _ := state.DataPop()
		bi, _ := result.RBigIntValue()
		if bi.StringValue() != largeNum {
			t.Errorf("Expected %s, got %s", largeNum, bi.StringValue())
		}
	})

	t.Run("convert RBoolean true to BigInt", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.True)

		op, _ := Get(dtrules.GetRName("cvbi"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("cvbi operator failed: %v", err)
		}

		result, _ := state.DataPop()
		bi, _ := result.RBigIntValue()
		if bi.StringValue() != "1" {
			t.Errorf("Expected 1, got %s", bi.StringValue())
		}
	})

	t.Run("convert RNull to BigInt", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.GetRNull())

		op, _ := Get(dtrules.GetRName("cvbi"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("cvbi operator failed: %v", err)
		}

		result, _ := state.DataPop()
		bi, _ := result.RBigIntValue()
		if bi.StringValue() != "0" {
			t.Errorf("Expected 0, got %s", bi.StringValue())
		}
	})
}

// =============================================================================
// Mixed Operations with RInteger (type coercion)
// =============================================================================

func TestBigIntOperatorsWithRInteger(t *testing.T) {
	t.Run("b+ with RInteger", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.GetRBigIntFromInt64(100))
		state.DataPush(dtrules.GetRIntegerValue(50))

		op, _ := Get(dtrules.GetRName("b+"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("b+ operator failed: %v", err)
		}

		result, _ := state.DataPop()
		bi, _ := result.RBigIntValue()
		if bi.StringValue() != "150" {
			t.Errorf("Expected 150, got %s", bi.StringValue())
		}
	})

	t.Run("b* with RInteger", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.GetRIntegerValue(100))
		state.DataPush(dtrules.GetRBigIntFromInt64(50))

		op, _ := Get(dtrules.GetRName("b*"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("b* operator failed: %v", err)
		}

		result, _ := state.DataPop()
		bi, _ := result.RBigIntValue()
		if bi.StringValue() != "5000" {
			t.Errorf("Expected 5000, got %s", bi.StringValue())
		}
	})

	t.Run("b== with RInteger", func(t *testing.T) {
		state := newBigIntTestState()
		state.DataPush(dtrules.GetRBigIntFromInt64(42))
		state.DataPush(dtrules.GetRIntegerValue(42))

		op, _ := Get(dtrules.GetRName("b=="))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("b== operator failed: %v", err)
		}

		result, _ := state.DataPop()
		val, _ := result.BooleanValue()
		if !val {
			t.Error("Expected BigInt(42) == RInteger(42)")
		}
	})
}

// =============================================================================
// Edge cases and error handling
// =============================================================================

func TestBigIntOperatorsEmptyStack(t *testing.T) {
	state := newBigIntTestState()

	// Test binary operator with empty stack
	op, _ := Get(dtrules.GetRName("b+"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected error for empty stack")
	}
}

func TestBigIntOperatorsOneElement(t *testing.T) {
	state := newBigIntTestState()
	state.DataPush(dtrules.GetRBigIntFromInt64(42))

	// Test binary operator with only one element
	op, _ := Get(dtrules.GetRName("b+"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected error for stack with only one element")
	}
}

func TestBigIntUnaryOperatorEmptyStack(t *testing.T) {
	state := newBigIntTestState()

	// Test unary operator with empty stack
	op, _ := Get(dtrules.GetRName("babs"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected error for empty stack")
	}
}
