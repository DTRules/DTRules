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
	"time"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
)

// mockSession implements minimal Session for testing
type mockSession struct {
	uniqueID int
}

func (m *mockSession) GetState() dtrules.State                 { return nil }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory { return nil }
func (m *mockSession) GetUniqueID() int                        { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser       { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet             { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return nil, nil
}
func (m *mockSession) Compile(expr string) (dtrules.Object, error) {
	return nil, nil
}

func newTestState() *interpreter.DTState {
	return interpreter.NewDTState(&mockSession{})
}

func TestOperatorRegistry(t *testing.T) {
	// Test that operators are registered
	addOp, ok := Get(dtrules.GetRName("+"))
	if !ok {
		t.Fatal("+ operator not found")
	}
	if addOp == nil {
		t.Fatal("+ operator is nil")
	}

	// Test alias
	subOp, ok := Get(dtrules.GetRName("-"))
	if !ok {
		t.Fatal("- operator not found")
	}
	if subOp == nil {
		t.Fatal("- operator is nil")
	}

	// Test GetByString
	mulOp, ok := GetByString("*")
	if !ok {
		t.Fatal("* operator not found via GetByString")
	}
	if mulOp == nil {
		t.Fatal("* operator is nil")
	}
}

func TestAddOperator(t *testing.T) {
	state := newTestState()

	// Test integer addition
	state.DataPush(dtrules.GetRIntegerValue(10))
	state.DataPush(dtrules.GetRIntegerValue(20))

	op, _ := Get(dtrules.GetRName("+"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("+ operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	val, err := result.IntValue()
	if err != nil {
		t.Fatalf("IntValue failed: %v", err)
	}
	if val != 30 {
		t.Errorf("Expected 30, got %d", val)
	}

	// Stack should be empty
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got size %d", state.DataStackDepth())
	}
}

func TestSubtractOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(50))
	state.DataPush(dtrules.GetRIntegerValue(20))

	op, _ := Get(dtrules.GetRName("-"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 30 {
		t.Errorf("Expected 30, got %d", val)
	}
}

func TestMultiplyOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(6))
	state.DataPush(dtrules.GetRIntegerValue(7))

	op, _ := Get(dtrules.GetRName("*"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("* operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestDivideOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(100))
	state.DataPush(dtrules.GetRIntegerValue(5))

	op, _ := Get(dtrules.GetRName("/"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("/ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 20 {
		t.Errorf("Expected 20, got %d", val)
	}
}

func TestDivideByZero(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(100))
	state.DataPush(dtrules.GetRIntegerValue(0))

	op, _ := Get(dtrules.GetRName("/"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected divide by zero error")
	}
}

// Issue #132: mod operator is now implemented

func TestNegateOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(42))

	op, _ := Get(dtrules.GetRName("negate"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("negate operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != -42 {
		t.Errorf("Expected -42, got %d", val)
	}
}

func TestAbsOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(-42))

	op, _ := Get(dtrules.GetRName("abs"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("abs operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestFloatAddOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRDoubleValue(1.5))
	state.DataPush(dtrules.GetRDoubleValue(2.5))

	op, _ := Get(dtrules.GetRName("f+"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("f+ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.DoubleValue()
	if val != 4.0 {
		t.Errorf("Expected 4.0, got %f", val)
	}
}

func TestCompareOperators(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		a, b     int
		expected bool
	}{
		{"less than true", "<", 5, 10, true},
		{"less than false", "<", 10, 5, false},
		{"less than equal", "<", 5, 5, false},
		{"greater than true", ">", 10, 5, true},
		{"greater than false", ">", 5, 10, false},
		{"greater than equal", ">", 5, 5, false},
		{"less equal true 1", "<=", 5, 10, true},
		{"less equal true 2", "<=", 5, 5, true},
		{"less equal false", "<=", 10, 5, false},
		{"greater equal true 1", ">=", 10, 5, true},
		{"greater equal true 2", ">=", 5, 5, true},
		{"greater equal false", ">=", 5, 10, false},
		{"equal true", "==", 5, 5, true},
		{"equal false", "==", 5, 10, false},
		{"not equal true", "!=", 5, 10, true},
		{"not equal false", "!=", 5, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRIntegerValue(int64(tt.a)))
			state.DataPush(dtrules.GetRIntegerValue(int64(tt.b)))

			op, ok := Get(dtrules.GetRName(tt.op))
			if !ok {
				t.Fatalf("Operator %s not found", tt.op)
			}

			err := op.Execute(state)
			if err != nil {
				t.Fatalf("%s operator failed: %v", tt.op, err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

func TestBooleanOperators(t *testing.T) {
	// Test AND
	state := newTestState()
	state.DataPush(dtrules.True)
	state.DataPush(dtrules.True)

	op, _ := Get(dtrules.GetRName("and"))
	op.Execute(state)
	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("true AND true should be true")
	}

	// Test AND with false
	state.DataPush(dtrules.True)
	state.DataPush(dtrules.False)
	op.Execute(state)
	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if val {
		t.Error("true AND false should be false")
	}

	// Test OR
	state.DataPush(dtrules.False)
	state.DataPush(dtrules.True)

	op, _ = Get(dtrules.GetRName("or"))
	op.Execute(state)
	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if !val {
		t.Error("false OR true should be true")
	}

	// Test NOT
	state.DataPush(dtrules.True)

	op, _ = Get(dtrules.GetRName("not"))
	op.Execute(state)
	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if val {
		t.Error("NOT true should be false")
	}
}

func TestStackOperators(t *testing.T) {
	// Test dup
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(42))

	op, _ := Get(dtrules.GetRName("dup"))
	op.Execute(state)

	if state.DataStackDepth() != 2 {
		t.Errorf("Expected stack size 2 after dup, got %d", state.DataStackDepth())
	}

	v1, _ := state.DataPop()
	v2, _ := state.DataPop()
	val1, _ := v1.IntValue()
	val2, _ := v2.IntValue()
	if val1 != 42 || val2 != 42 {
		t.Error("dup should duplicate top of stack")
	}

	// Test pop
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))

	op, _ = Get(dtrules.GetRName("pop"))
	op.Execute(state)

	if state.DataStackDepth() != 1 {
		t.Errorf("Expected stack size 1 after pop, got %d", state.DataStackDepth())
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 1 {
		t.Errorf("Expected 1 remaining after pop, got %d", val)
	}

	// Test swap
	state.DataPush(dtrules.GetRIntegerValue(10))
	state.DataPush(dtrules.GetRIntegerValue(20))

	op, _ = Get(dtrules.GetRName("swap"))
	op.Execute(state)

	v1, _ = state.DataPop()
	v2, _ = state.DataPop()
	val1, _ = v1.IntValue()
	val2, _ = v2.IntValue()
	if val1 != 10 || val2 != 20 {
		t.Error("exch should swap top two elements")
	}
}

func TestIsNullOperator(t *testing.T) {
	state := newTestState()

	// Test with null
	state.DataPush(dtrules.GetRNull())
	op, _ := Get(dtrules.GetRName("isnull"))
	op.Execute(state)

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("isnull should return true for null")
	}

	// Test with non-null
	state.DataPush(dtrules.GetRIntegerValue(42))
	op.Execute(state)

	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if val {
		t.Error("isnull should return false for non-null")
	}
}

func TestConcatOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("Hello"))
	state.DataPush(dtrules.NewRString(" World"))

	op, _ := Get(dtrules.GetRName("concat"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("concat operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", result.StringValue())
	}
}

func TestTrimOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("  hello  "))

	op, _ := Get(dtrules.GetRName("trim"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("trim operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result.StringValue())
	}
}

func TestLowercaseOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("HELLO"))

	op, _ := Get(dtrules.GetRName("lowercase"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("lowercase operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result.StringValue())
	}
}

func TestUppercaseOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("hello"))

	op, _ := Get(dtrules.GetRName("uppercase"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("uppercase operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "HELLO" {
		t.Errorf("Expected 'HELLO', got '%s'", result.StringValue())
	}
}

func TestStringLengthOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("hello"))

	op, _ := Get(dtrules.GetRName("stringlength"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("stringlength operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 5 {
		t.Errorf("Expected 5, got %d", val)
	}
}

func TestSubstringOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("hello world"))
	state.DataPush(dtrules.GetRIntegerValue(0))
	state.DataPush(dtrules.GetRIntegerValue(5))

	op, _ := Get(dtrules.GetRName("substring"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("substring operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result.StringValue())
	}
}

func TestPrimitivesEntity(t *testing.T) {
	primitives := GetPrimitives()
	if primitives == nil {
		t.Fatal("GetPrimitives returned nil")
	}

	// Test that we can get operators through the primitives entity
	addOp, err := primitives.Get(dtrules.GetRName("+"))
	if err != nil {
		t.Fatalf("Failed to get + operator: %v", err)
	}
	if addOp == nil {
		t.Error("Expected + operator from primitives entity")
	}

	// Test ContainsAttribute
	if !primitives.ContainsAttribute(dtrules.GetRName("+")) {
		t.Error("Primitives should contain + operator")
	}

	if primitives.ContainsAttribute(dtrules.GetRName("nonexistent_operator")) {
		t.Error("Primitives should not contain nonexistent operator")
	}
}

// =============================================================================
// Array Operator Tests
// =============================================================================

func TestNewArrayOperator(t *testing.T) {
	state := newTestState()

	op, ok := Get(dtrules.GetRName("newarray"))
	if !ok {
		t.Fatal("newarray operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("newarray operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	arr, err := result.RArrayValue()
	if err != nil {
		t.Fatalf("Result is not an array: %v", err)
	}

	if arr.Size() != 0 {
		t.Errorf("Expected empty array, got size %d", arr.Size())
	}
}

func TestAddToOperator(t *testing.T) {
	state := newTestState()

	// Create array
	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	state.DataPush(arr)
	state.DataPush(dtrules.GetRIntegerValue(42))

	op, _ := Get(dtrules.GetRName("addto"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("addto operator failed: %v", err)
	}

	// Stack should be empty (addto consumes both arguments)
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got size %d", state.DataStackDepth())
	}

	// Array should have one element
	if arr.Size() != 1 {
		t.Errorf("Expected array size 1, got %d", arr.Size())
	}

	elem, _ := arr.Get(0)
	val, _ := elem.IntValue()
	if val != 42 {
		t.Errorf("Expected element 42, got %d", val)
	}
}

func TestRemoveOperator(t *testing.T) {
	state := newTestState()

	// Create array with elements
	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(10))
	arr.Add(dtrules.GetRIntegerValue(20))
	arr.Add(dtrules.GetRIntegerValue(30))

	state.DataPush(arr)
	state.DataPush(dtrules.GetRIntegerValue(20))

	op, _ := Get(dtrules.GetRName("remove"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("remove operator failed: %v", err)
	}

	if arr.Size() != 2 {
		t.Errorf("Expected array size 2, got %d", arr.Size())
	}

	// Check remaining elements
	elem0, _ := arr.Get(0)
	elem1, _ := arr.Get(1)
	val0, _ := elem0.IntValue()
	val1, _ := elem1.IntValue()

	if val0 != 10 || val1 != 30 {
		t.Errorf("Expected [10, 30], got [%d, %d]", val0, val1)
	}
}

func TestRemoveAtOperator(t *testing.T) {
	state := newTestState()

	// Create array with elements
	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(10))
	arr.Add(dtrules.GetRIntegerValue(20))
	arr.Add(dtrules.GetRIntegerValue(30))

	state.DataPush(arr)
	state.DataPush(dtrules.GetRIntegerValue(1)) // Remove at index 1

	op, _ := Get(dtrules.GetRName("removeat"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("removeat operator failed: %v", err)
	}

	if arr.Size() != 2 {
		t.Errorf("Expected array size 2, got %d", arr.Size())
	}

	elem0, _ := arr.Get(0)
	elem1, _ := arr.Get(1)
	val0, _ := elem0.IntValue()
	val1, _ := elem1.IntValue()

	if val0 != 10 || val1 != 30 {
		t.Errorf("Expected [10, 30], got [%d, %d]", val0, val1)
	}
}

func TestLengthOperator(t *testing.T) {
	state := newTestState()

	// Create array with 3 elements
	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(1))
	arr.Add(dtrules.GetRIntegerValue(2))
	arr.Add(dtrules.GetRIntegerValue(3))

	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("length"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("length operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 3 {
		t.Errorf("Expected length 3, got %d", val)
	}
}

func TestLengthEmptyArray(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("length"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("length operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 0 {
		t.Errorf("Expected length 0, got %d", val)
	}
}

func TestGetAtOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.NewRString("first"))
	arr.Add(dtrules.NewRString("second"))
	arr.Add(dtrules.NewRString("third"))

	state.DataPush(arr)
	state.DataPush(dtrules.GetRIntegerValue(1))

	op, _ := Get(dtrules.GetRName("getat"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("getat operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "second" {
		t.Errorf("Expected 'second', got '%s'", result.StringValue())
	}
}

func TestAddAtOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.NewRString("first"))
	arr.Add(dtrules.NewRString("third"))

	state.DataPush(arr)
	state.DataPush(dtrules.NewRString("second"))
	state.DataPush(dtrules.GetRIntegerValue(1))

	op, _ := Get(dtrules.GetRName("addat"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("addat operator failed: %v", err)
	}

	if arr.Size() != 3 {
		t.Errorf("Expected size 3, got %d", arr.Size())
	}

	elem, _ := arr.Get(1)
	if elem.StringValue() != "second" {
		t.Errorf("Expected 'second' at index 1, got '%s'", elem.StringValue())
	}
}

func TestFirstOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(100))
	arr.Add(dtrules.GetRIntegerValue(200))

	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("first"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("first operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}
}

func TestFirstEmptyArray(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("first"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("first operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.Type() != dtrules.TypeNull {
		t.Errorf("Expected null for empty array, got %v", result.Type())
	}
}

func TestLastOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(100))
	arr.Add(dtrules.GetRIntegerValue(200))
	arr.Add(dtrules.GetRIntegerValue(300))

	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("last"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("last operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 300 {
		t.Errorf("Expected 300, got %d", val)
	}
}

func TestLastEmptyArray(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("last"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("last operator failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.Type() != dtrules.TypeNull {
		t.Errorf("Expected null for empty array, got %v", result.Type())
	}
}

func TestCopyOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(1))
	arr.Add(dtrules.GetRIntegerValue(2))

	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("copy"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("copy operator failed: %v", err)
	}

	result, _ := state.DataPop()
	copied, _ := result.RArrayValue()

	// Check copy has same elements
	if copied.Size() != 2 {
		t.Errorf("Expected copied size 2, got %d", copied.Size())
	}

	// Modify original, check copy is independent
	arr.Add(dtrules.GetRIntegerValue(3))
	if copied.Size() != 2 {
		t.Error("Copy should be independent of original")
	}
}

func TestMemberOfOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.NewRString("apple"))
	arr.Add(dtrules.NewRString("banana"))
	arr.Add(dtrules.NewRString("cherry"))

	// Test element that exists
	state.DataPush(arr)
	state.DataPush(dtrules.NewRString("banana"))

	op, _ := Get(dtrules.GetRName("memberof"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("memberof operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("Expected true for 'banana' in array")
	}

	// Test element that doesn't exist
	state.DataPush(arr)
	state.DataPush(dtrules.NewRString("grape"))

	err = op.Execute(state)
	if err != nil {
		t.Fatalf("memberof operator failed: %v", err)
	}

	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if val {
		t.Error("Expected false for 'grape' not in array")
	}
}

func TestMergeOperator(t *testing.T) {
	state := newTestState()

	arr1, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr1.Add(dtrules.GetRIntegerValue(1))
	arr1.Add(dtrules.GetRIntegerValue(2))

	arr2, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr2.Add(dtrules.GetRIntegerValue(3))
	arr2.Add(dtrules.GetRIntegerValue(4))

	state.DataPush(arr1)
	state.DataPush(arr2)

	op, _ := Get(dtrules.GetRName("merge"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("merge operator failed: %v", err)
	}

	result, _ := state.DataPop()
	merged, _ := result.RArrayValue()

	if merged.Size() != 4 {
		t.Errorf("Expected merged size 4, got %d", merged.Size())
	}
}

func TestClearArrayOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(1))
	arr.Add(dtrules.GetRIntegerValue(2))
	arr.Add(dtrules.GetRIntegerValue(3))

	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("cleararray"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("cleararray operator failed: %v", err)
	}

	if arr.Size() != 0 {
		t.Errorf("Expected cleared array, got size %d", arr.Size())
	}
}

func TestAddNoDupsOperator(t *testing.T) {
	state := newTestState()

	arr, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr.Add(dtrules.GetRIntegerValue(1))
	arr.Add(dtrules.GetRIntegerValue(2))

	// Add existing element (should not add)
	state.DataPush(arr)
	state.DataPush(dtrules.GetRIntegerValue(1))

	op, _ := Get(dtrules.GetRName("add_no_dups"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("add_no_dups operator failed: %v", err)
	}

	if arr.Size() != 2 {
		t.Errorf("Expected size 2 (no dup added), got %d", arr.Size())
	}

	// Add new element (should add)
	state.DataPush(arr)
	state.DataPush(dtrules.GetRIntegerValue(3))

	err = op.Execute(state)
	if err != nil {
		t.Fatalf("add_no_dups operator failed: %v", err)
	}

	if arr.Size() != 3 {
		t.Errorf("Expected size 3 (new element added), got %d", arr.Size())
	}
}

func TestIntersectionOperator(t *testing.T) {
	state := newTestState()

	arr1, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr1.Add(dtrules.GetRIntegerValue(1))
	arr1.Add(dtrules.GetRIntegerValue(2))
	arr1.Add(dtrules.GetRIntegerValue(3))

	arr2, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr2.Add(dtrules.GetRIntegerValue(2))
	arr2.Add(dtrules.GetRIntegerValue(3))
	arr2.Add(dtrules.GetRIntegerValue(4))

	state.DataPush(arr1)
	state.DataPush(arr2)

	op, _ := Get(dtrules.GetRName("intersection"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("intersection operator failed: %v", err)
	}

	result, _ := state.DataPop()
	intersection, _ := result.RArrayValue()

	if intersection.Size() != 2 {
		t.Errorf("Expected intersection size 2 (2,3), got %d", intersection.Size())
	}
}

func TestIntersectsOperator(t *testing.T) {
	state := newTestState()

	arr1, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr1.Add(dtrules.GetRIntegerValue(1))
	arr1.Add(dtrules.GetRIntegerValue(2))

	arr2, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr2.Add(dtrules.GetRIntegerValue(2))
	arr2.Add(dtrules.GetRIntegerValue(3))

	// Test arrays that intersect
	state.DataPush(arr1)
	state.DataPush(arr2)

	op, _ := Get(dtrules.GetRName("intersects"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("intersects operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if !val {
		t.Error("Expected true for intersecting arrays")
	}

	// Test arrays that don't intersect
	arr3, _ := dtrules.NewArray(state.GetSession(), true, false)
	arr3.Add(dtrules.GetRIntegerValue(10))
	arr3.Add(dtrules.GetRIntegerValue(20))

	state.DataPush(arr1)
	state.DataPush(arr3)

	err = op.Execute(state)
	if err != nil {
		t.Fatalf("intersects operator failed: %v", err)
	}

	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if val {
		t.Error("Expected false for non-intersecting arrays")
	}
}

func TestTokenizeOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("a,b,c"))
	state.DataPush(dtrules.NewRString(","))

	op, _ := Get(dtrules.GetRName("tokenize"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("tokenize operator failed: %v", err)
	}

	result, _ := state.DataPop()
	arr, _ := result.RArrayValue()

	if arr.Size() != 3 {
		t.Errorf("Expected 3 tokens, got %d", arr.Size())
	}

	elem0, _ := arr.Get(0)
	elem1, _ := arr.Get(1)
	elem2, _ := arr.Get(2)

	if elem0.StringValue() != "a" || elem1.StringValue() != "b" || elem2.StringValue() != "c" {
		t.Errorf("Expected [a,b,c], got [%s,%s,%s]",
			elem0.StringValue(), elem1.StringValue(), elem2.StringValue())
	}
}

// =============================================================================
// XOR Operator Tests
// =============================================================================

func TestXorOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     bool
		expected bool
	}{
		{"true xor true", true, true, false},
		{"true xor false", true, false, true},
		{"false xor true", false, true, true},
		{"false xor false", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRBoolean(tt.a))
			state.DataPush(dtrules.GetRBoolean(tt.b))

			op, ok := Get(dtrules.GetRName("xor"))
			if !ok {
				t.Fatal("xor operator not found")
			}

			err := op.Execute(state)
			if err != nil {
				t.Fatalf("xor operator failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

// =============================================================================
// Edge Cases for Existing Operators
// =============================================================================

func TestNegativeNumberOperations(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		a, b     int64
		expected int64
	}{
		{"add negative", "+", -10, 5, -5},
		{"add two negatives", "+", -10, -5, -15},
		{"subtract negative", "-", 10, -5, 15},
		{"multiply negatives", "*", -6, -7, 42},
		{"multiply neg pos", "*", -6, 7, -42},
		{"divide negative", "/", -100, 5, -20},
		{"divide by negative", "/", 100, -5, -20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRIntegerValue(tt.a))
			state.DataPush(dtrules.GetRIntegerValue(tt.b))

			op, _ := Get(dtrules.GetRName(tt.op))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("%s operator failed: %v", tt.op, err)
			}

			result, _ := state.DataPop()
			val, _ := result.IntValue()
			if int64(val) != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, val)
			}
		})
	}
}

func TestNegateNegativeNumber(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(-42))

	op, _ := Get(dtrules.GetRName("negate"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("negate failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestAbsZero(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(0))

	op, _ := Get(dtrules.GetRName("abs"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("abs failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}
}

func TestEmptyStringOperations(t *testing.T) {
	// Concat with empty strings
	t.Run("concat empty strings", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString(""))
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("concat"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("concat failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.StringValue() != "" {
			t.Errorf("Expected '', got '%s'", result.StringValue())
		}
	})

	// Concat empty with non-empty
	t.Run("concat with empty", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString("hello"))
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("concat"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("concat failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.StringValue() != "hello" {
			t.Errorf("Expected 'hello', got '%s'", result.StringValue())
		}
	})

	// String length of empty
	t.Run("stringlength empty", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("stringlength"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("stringlength failed: %v", err)
		}

		result, _ := state.DataPop()
		val, _ := result.IntValue()
		if val != 0 {
			t.Errorf("Expected 0, got %d", val)
		}
	})

	// Trim empty string
	t.Run("trim empty", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("trim"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("trim failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.StringValue() != "" {
			t.Errorf("Expected '', got '%s'", result.StringValue())
		}
	})

	// Trim whitespace-only string
	t.Run("trim whitespace only", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString("   \t\n  "))

		op, _ := Get(dtrules.GetRName("trim"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("trim failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.StringValue() != "" {
			t.Errorf("Expected '', got '%s'", result.StringValue())
		}
	})

	// Lowercase empty
	t.Run("lowercase empty", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("lowercase"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("lowercase failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.StringValue() != "" {
			t.Errorf("Expected '', got '%s'", result.StringValue())
		}
	})

	// Uppercase empty
	t.Run("uppercase empty", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("uppercase"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("uppercase failed: %v", err)
		}

		result, _ := state.DataPop()
		if result.StringValue() != "" {
			t.Errorf("Expected '', got '%s'", result.StringValue())
		}
	})
}

func TestSubstringEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		start    int64
		length   int64
		expected string
	}{
		{"start beyond end", "hello", 10, 5, ""},
		{"negative start", "hello", -1, 3, "hel"},
		{"negative length", "hello", 0, -1, ""},
		{"zero length", "hello", 2, 0, ""},
		{"length beyond end", "hello", 3, 100, "lo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.str))
			state.DataPush(dtrules.GetRIntegerValue(tt.start))
			state.DataPush(dtrules.GetRIntegerValue(tt.length))

			op, _ := Get(dtrules.GetRName("substring"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("substring failed: %v", err)
			}

			result, _ := state.DataPop()
			if result.StringValue() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.StringValue())
			}
		})
	}
}

func TestStringComparisonEdgeCases(t *testing.T) {
	// Test empty string comparisons
	t.Run("empty strings equal", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString(""))
		state.DataPush(dtrules.NewRString(""))

		op, _ := Get(dtrules.GetRName("s=="))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("s== failed: %v", err)
		}

		result, _ := state.DataPop()
		val, _ := result.BooleanValue()
		if !val {
			t.Error("Empty strings should be equal")
		}
	})

	// Test case-insensitive equality
	t.Run("case insensitive equal", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString("HELLO"))
		state.DataPush(dtrules.NewRString("hello"))

		op, _ := Get(dtrules.GetRName("s==i"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("s==i failed: %v", err)
		}

		result, _ := state.DataPop()
		val, _ := result.BooleanValue()
		if !val {
			t.Error("'HELLO' and 'hello' should be equal (case insensitive)")
		}
	})
}

func TestFloatOperations(t *testing.T) {
	// Float divide by zero
	t.Run("float divide by zero", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.GetRDoubleValue(1.5))
		state.DataPush(dtrules.GetRDoubleValue(0.0))

		op, _ := Get(dtrules.GetRName("fdiv"))
		err := op.Execute(state)
		if err == nil {
			t.Error("Expected error for float divide by zero")
		}
	})

	// Float negate
	t.Run("float negate", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.GetRDoubleValue(-3.14))

		op, _ := Get(dtrules.GetRName("fnegate"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("fnegate failed: %v", err)
		}

		result, _ := state.DataPop()
		val, _ := result.DoubleValue()
		if val != 3.14 {
			t.Errorf("Expected 3.14, got %f", val)
		}
	})

	// Float abs
	t.Run("float abs negative", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.GetRDoubleValue(-2.5))

		op, _ := Get(dtrules.GetRName("fabs"))
		err := op.Execute(state)
		if err != nil {
			t.Fatalf("fabs failed: %v", err)
		}

		result, _ := state.DataPop()
		val, _ := result.DoubleValue()
		if val != 2.5 {
			t.Errorf("Expected 2.5, got %f", val)
		}
	})
}

// =============================================================================
// Error Condition Tests
// =============================================================================

func TestStackUnderflow(t *testing.T) {
	tests := []struct {
		name string
		op   string
	}{
		{"add underflow", "+"},
		{"subtract underflow", "-"},
		{"multiply underflow", "*"},
		{"divide underflow", "/"},
		{"and underflow", "and"},
		{"or underflow", "or"},
		{"xor underflow", "xor"},
		{"not underflow", "not"},
		{"dup underflow", "dup"},
		{"swap underflow", "swap"},
		{"pop underflow", "pop"},
		{"concat underflow", "concat"},
		{"trim underflow", "trim"},
		{"negate underflow", "negate"},
		{"abs underflow", "abs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()

			op, ok := Get(dtrules.GetRName(tt.op))
			if !ok {
				t.Fatalf("%s operator not found", tt.op)
			}

			err := op.Execute(state)
			if err == nil {
				t.Errorf("Expected stack underflow error for %s", tt.op)
			}
		})
	}
}

func TestStackUnderflowPartial(t *testing.T) {
	// Test operators that need 2+ arguments when only 1 is provided
	tests := []struct {
		name string
		op   string
	}{
		{"add partial", "+"},
		{"subtract partial", "-"},
		{"and partial", "and"},
		{"or partial", "or"},
		{"swap partial", "swap"},
		{"concat partial", "concat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRIntegerValue(42))

			op, _ := Get(dtrules.GetRName(tt.op))
			err := op.Execute(state)
			if err == nil {
				t.Errorf("Expected stack underflow error for %s with single element", tt.op)
			}
		})
	}
}

func TestTypeMismatch(t *testing.T) {
	// Add with boolean (should fail to convert)
	t.Run("add type mismatch", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.GetRIntegerValue(10))
		state.DataPush(dtrules.True)

		op, _ := Get(dtrules.GetRName("+"))
		err := op.Execute(state)
		if err == nil {
			t.Error("Expected type mismatch error for add with boolean")
		}
	})

	// And with integer (should fail)
	t.Run("and type mismatch", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.True)
		state.DataPush(dtrules.GetRIntegerValue(42))

		op, _ := Get(dtrules.GetRName("and"))
		err := op.Execute(state)
		if err == nil {
			t.Error("Expected type mismatch error for and with integer")
		}
	})

	// Length on non-array
	t.Run("length type mismatch", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.GetRIntegerValue(42))

		op, _ := Get(dtrules.GetRName("length"))
		err := op.Execute(state)
		if err == nil {
			t.Error("Expected type mismatch error for length on integer")
		}
	})

	// getat on non-array
	t.Run("getat type mismatch", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.NewRString("not an array"))
		state.DataPush(dtrules.GetRIntegerValue(0))

		op, _ := Get(dtrules.GetRName("getat"))
		err := op.Execute(state)
		if err == nil {
			t.Error("Expected type mismatch error for getat on string")
		}
	})

	// addto on non-array
	t.Run("addto type mismatch", func(t *testing.T) {
		state := newTestState()
		state.DataPush(dtrules.GetRIntegerValue(42))
		state.DataPush(dtrules.GetRIntegerValue(1))

		op, _ := Get(dtrules.GetRName("addto"))
		err := op.Execute(state)
		if err == nil {
			t.Error("Expected type mismatch error for addto on integer")
		}
	})
}

func TestPickOutOfBounds(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))
	state.DataPush(dtrules.GetRIntegerValue(10)) // pick index 10, but only 2 elements

	op, _ := Get(dtrules.GetRName("pick"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected out of bounds error for pick")
	}
}

func TestRollNotEnoughElements(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(5)) // roll 5 elements, but only 1 available

	op, _ := Get(dtrules.GetRName("roll"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected error for roll with not enough elements")
	}
}

// =============================================================================
// Additional Stack Operator Tests
// =============================================================================

func TestOverOperator(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))

	op, _ := Get(dtrules.GetRName("over"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("over failed: %v", err)
	}

	// Stack should be: 1 2 1
	if state.DataStackDepth() != 3 {
		t.Errorf("Expected depth 3, got %d", state.DataStackDepth())
	}

	top, _ := state.DataPop()
	val, _ := top.IntValue()
	if val != 1 {
		t.Errorf("Expected 1 on top, got %d", val)
	}
}

func TestRotOperator(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))
	state.DataPush(dtrules.GetRIntegerValue(3))

	op, _ := Get(dtrules.GetRName("rot"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("rot failed: %v", err)
	}

	// Stack should be: 2 3 1 (top is 1)
	v1, _ := state.DataPop()
	v2, _ := state.DataPop()
	v3, _ := state.DataPop()

	val1, _ := v1.IntValue()
	val2, _ := v2.IntValue()
	val3, _ := v3.IntValue()

	if val1 != 1 || val2 != 3 || val3 != 2 {
		t.Errorf("Expected [2,3,1], got [%d,%d,%d]", val3, val2, val1)
	}
}

func TestClearOperator(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))
	state.DataPush(dtrules.GetRIntegerValue(3))

	op, _ := Get(dtrules.GetRName("clear"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got depth %d", state.DataStackDepth())
	}
}

func TestNotNullOperator(t *testing.T) {
	state := newTestState()

	// Test with null
	state.DataPush(dtrules.GetRNull())
	op, _ := Get(dtrules.GetRName("notnull"))
	op.Execute(state)

	result, _ := state.DataPop()
	val, _ := result.BooleanValue()
	if val {
		t.Error("notnull should return false for null")
	}

	// Test with non-null
	state.DataPush(dtrules.GetRIntegerValue(42))
	op.Execute(state)

	result, _ = state.DataPop()
	val, _ = result.BooleanValue()
	if !val {
		t.Error("notnull should return true for non-null")
	}
}

// =============================================================================
// Boolean Equality Tests
// =============================================================================

func TestBeqOperator(t *testing.T) {
	tests := []struct {
		name     string
		a, b     bool
		expected bool
	}{
		{"true beq true", true, true, true},
		{"true beq false", true, false, false},
		{"false beq true", false, true, false},
		{"false beq false", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRBoolean(tt.a))
			state.DataPush(dtrules.GetRBoolean(tt.b))

			op, _ := Get(dtrules.GetRName("beq"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("beq failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

// =============================================================================
// String Operator Tests
// =============================================================================

func TestIndexOfOperator(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected int
	}{
		{"found at start", "hello world", "hello", 0},
		{"found in middle", "hello world", "world", 6},
		{"not found", "hello world", "xyz", -1},
		{"empty substring", "hello", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.str))
			state.DataPush(dtrules.NewRString(tt.substr))

			op, _ := Get(dtrules.GetRName("indexof"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("indexof failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.IntValue()
			if val != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, val)
			}
		})
	}
}

func TestStartsWithOperator(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		prefix   string
		expected bool
	}{
		{"starts with", "hello world", "hello", true},
		{"does not start with", "hello world", "world", false},
		{"empty prefix", "hello", "", true},
		{"empty string", "", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.str))
			state.DataPush(dtrules.NewRString(tt.prefix))

			op, _ := Get(dtrules.GetRName("startswith"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("startswith failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

func TestEndsWithOperator(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		suffix   string
		expected bool
	}{
		{"ends with", "hello world", "world", true},
		{"does not end with", "hello world", "hello", false},
		{"empty suffix", "hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.str))
			state.DataPush(dtrules.NewRString(tt.suffix))

			op, _ := Get(dtrules.GetRName("endswith"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("endswith failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

func TestContainsOperator(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"contains", "hello world", "lo wo", true},
		{"does not contain", "hello world", "xyz", false},
		{"empty substring", "hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.str))
			state.DataPush(dtrules.NewRString(tt.substr))

			op, _ := Get(dtrules.GetRName("contains"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("contains failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

func TestReplaceOperator(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		old      string
		new      string
		expected string
	}{
		{"replace single", "hello world", "world", "there", "hello there"},
		{"replace multiple", "aaa", "a", "b", "bbb"},
		{"replace with empty", "hello world", "world", "", "hello "},
		{"no match", "hello world", "xyz", "abc", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.str))
			state.DataPush(dtrules.NewRString(tt.old))
			state.DataPush(dtrules.NewRString(tt.new))

			op, _ := Get(dtrules.GetRName("replace"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("replace failed: %v", err)
			}

			result, _ := state.DataPop()
			if result.StringValue() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.StringValue())
			}
		})
	}
}

func TestSplitOperator(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.NewRString("a:b:c"))
	state.DataPush(dtrules.NewRString(":"))

	op, _ := Get(dtrules.GetRName("split"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	result, _ := state.DataPop()
	arr, _ := result.RArrayValue()

	if arr.Size() != 3 {
		t.Errorf("Expected 3 parts, got %d", arr.Size())
	}
}

func TestToStringOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    dtrules.Object
		expected string
	}{
		{"integer", dtrules.GetRIntegerValue(42), "42"},
		{"boolean true", dtrules.True, "true"},
		{"boolean false", dtrules.False, "false"},
		{"string", dtrules.NewRString("hello"), "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(tt.input)

			op, _ := Get(dtrules.GetRName("tostring"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("tostring failed: %v", err)
			}

			result, _ := state.DataPop()
			if result.StringValue() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.StringValue())
			}
		})
	}
}

func TestRegexMatchOperator(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		str      string
		expected bool
	}{
		{"simple match", "hello", "hello world", true},
		{"no match", "xyz", "hello world", false},
		{"regex pattern", "^hello", "hello world", true},
		{"regex no match", "^world", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(tt.pattern))
			state.DataPush(dtrules.NewRString(tt.str))

			op, _ := Get(dtrules.GetRName("regexmatch"))
			err := op.Execute(state)
			if err != nil {
				t.Fatalf("regexmatch failed: %v", err)
			}

			result, _ := state.DataPop()
			val, _ := result.BooleanValue()
			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}
		})
	}
}

// =============================================================================
// Iteration Limit Tests
// =============================================================================

func TestErrMaxIterationsExceeded(t *testing.T) {
	// Test that ErrMaxIterationsExceeded returns an error with the correct limit
	err := ErrMaxIterationsExceeded(100)
	if err == nil {
		t.Fatal("Expected non-nil error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "100") {
		t.Errorf("Expected error message to contain limit '100', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "maximum iterations") {
		t.Errorf("Expected error message to mention 'maximum iterations', got: %s", errMsg)
	}
}

func TestDefaultMaxIterationsConstant(t *testing.T) {
	// Verify DefaultMaxIterations is set to a reasonable value (1 million)
	if DefaultMaxIterations != 1000000 {
		t.Errorf("Expected DefaultMaxIterations to be 1000000, got: %d", DefaultMaxIterations)
	}
}

func TestDoloopCompletesWithinLimit(t *testing.T) {
	// Test that doloop completes normally when within the iteration limit
	state := newTestState()

	// Create a body that pops the loop counter
	body, _ := dtrules.NewArray(state.GetSession(), true, true)
	popOp, _ := Get(dtrules.GetRName("pop"))
	body.Add(popOp)

	// doloop ( body start increment limit -- )
	// This will iterate 10 times (0 to 9), well within the default limit
	state.DataPush(body)
	state.DataPush(dtrules.GetRIntegerValue(0))  // start
	state.DataPush(dtrules.GetRIntegerValue(1))  // increment
	state.DataPush(dtrules.GetRIntegerValue(10)) // limit

	op, _ := Get(dtrules.GetRName("doloop"))
	err := op.Execute(state)

	if err != nil {
		t.Errorf("Expected doloop to complete successfully, got: %v", err)
	}
}

func TestDoloopZeroIncrement(t *testing.T) {
	state := newTestState()

	body, _ := dtrules.NewArray(state.GetSession(), true, true)
	popOp, _ := Get(dtrules.GetRName("pop"))
	body.Add(popOp)

	// doloop with zero increment should fail immediately
	state.DataPush(body)
	state.DataPush(dtrules.GetRIntegerValue(0))  // start
	state.DataPush(dtrules.GetRIntegerValue(0))  // increment = 0 (invalid!)
	state.DataPush(dtrules.GetRIntegerValue(10)) // limit

	op, _ := Get(dtrules.GetRName("doloop"))
	err := op.Execute(state)

	if err == nil {
		t.Fatal("Expected error for zero increment")
	}
	// Check that it's an undefined error containing the right message
	if !strings.Contains(err.Error(), "increment cannot be zero") {
		t.Errorf("Expected 'increment cannot be zero' error, got: %v", err)
	}
}

func TestDoloopNegativeIncrement(t *testing.T) {
	state := newTestState()

	// Track values pushed
	values := []int{}
	body, _ := dtrules.NewArray(state.GetSession(), true, true)
	// Body just pops the counter value
	popOp, _ := Get(dtrules.GetRName("pop"))
	body.Add(popOp)

	// doloop with negative increment: count down from 10 to 0
	state.DataPush(body)
	state.DataPush(dtrules.GetRIntegerValue(10)) // start
	state.DataPush(dtrules.GetRIntegerValue(-1)) // increment (negative)
	state.DataPush(dtrules.GetRIntegerValue(0))  // limit

	op, _ := Get(dtrules.GetRName("doloop"))
	err := op.Execute(state)

	if err != nil {
		t.Fatalf("doloop with negative increment failed: %v", err)
	}

	// Loop should have run 10 times (10, 9, 8, 7, 6, 5, 4, 3, 2, 1)
	// Stack should be empty since body pops each value
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got depth %d", state.DataStackDepth())
	}

	_ = values // suppress unused warning
}

// =============================================================================
// PerformCatchError Operator Tests
// =============================================================================

func TestPerformCatchErrorOperatorRegistered(t *testing.T) {
	op, ok := Get(dtrules.GetRName("performcatcherror"))
	if !ok {
		t.Fatal("performcatcherror operator not found")
	}
	if op == nil {
		t.Fatal("performcatcherror operator is nil")
	}
}

func TestPerformCatchErrorStackUnderflow(t *testing.T) {
	// Test with empty stack
	state := newTestState()
	op, _ := Get(dtrules.GetRName("performcatcherror"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected stack underflow error with empty stack")
	}

	// Test with only 1 argument
	state = newTestState()
	state.DataPush(dtrules.GetRName("error_entity"))
	err = op.Execute(state)
	if err == nil {
		t.Error("Expected stack underflow error with 1 argument")
	}

	// Test with only 2 arguments
	state = newTestState()
	state.DataPush(dtrules.GetRName("error_table"))
	state.DataPush(dtrules.GetRName("error_entity"))
	err = op.Execute(state)
	if err == nil {
		t.Error("Expected stack underflow error with 2 arguments")
	}
}

// =============================================================================
// Date Interval Operator Tests
// =============================================================================

func TestDaysOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(30))

	op, ok := Get(dtrules.GetRName("days"))
	if !ok {
		t.Fatal("days operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("days operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	interval, ok := dtrules.AsInterval(result)
	if !ok {
		t.Fatal("Result is not an interval")
	}

	if interval.GetAmount() != 30 {
		t.Errorf("Expected amount 30, got %d", interval.GetAmount())
	}
	if interval.GetUnit() != dtrules.IntervalDays {
		t.Errorf("Expected unit IntervalDays, got %v", interval.GetUnit())
	}
}

func TestMonthsOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(6))

	op, ok := Get(dtrules.GetRName("months"))
	if !ok {
		t.Fatal("months operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("months operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	interval, ok := dtrules.AsInterval(result)
	if !ok {
		t.Fatal("Result is not an interval")
	}

	if interval.GetAmount() != 6 {
		t.Errorf("Expected amount 6, got %d", interval.GetAmount())
	}
	if interval.GetUnit() != dtrules.IntervalMonths {
		t.Errorf("Expected unit IntervalMonths, got %v", interval.GetUnit())
	}
}

func TestYearsOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(2))

	op, ok := Get(dtrules.GetRName("years"))
	if !ok {
		t.Fatal("years operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("years operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	interval, ok := dtrules.AsInterval(result)
	if !ok {
		t.Fatal("Result is not an interval")
	}

	if interval.GetAmount() != 2 {
		t.Errorf("Expected amount 2, got %d", interval.GetAmount())
	}
	if interval.GetUnit() != dtrules.IntervalYears {
		t.Errorf("Expected unit IntervalYears, got %v", interval.GetUnit())
	}
}

func TestDatePlusWithDaysInterval(t *testing.T) {
	state := newTestState()

	// Create a date: 2024-01-15
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(30))

	// Create interval
	daysOp, _ := Get(dtrules.GetRName("days"))
	err := daysOp.Execute(state)
	if err != nil {
		t.Fatalf("days operator failed: %v", err)
	}

	// Now we have date and interval on stack
	// Add the interval to the date
	plusOp, _ := Get(dtrules.GetRName("d+"))
	err = plusOp.Execute(state)
	if err != nil {
		t.Fatalf("d+ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(0, 0, 30)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestDatePlusWithMonthsInterval(t *testing.T) {
	state := newTestState()

	// Create a date: 2024-01-15
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(3))

	// Create interval
	monthsOp, _ := Get(dtrules.GetRName("months"))
	err := monthsOp.Execute(state)
	if err != nil {
		t.Fatalf("months operator failed: %v", err)
	}

	// Add the interval to the date
	plusOp, _ := Get(dtrules.GetRName("d+"))
	err = plusOp.Execute(state)
	if err != nil {
		t.Fatalf("d+ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(0, 3, 0)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestDatePlusWithYearsInterval(t *testing.T) {
	state := newTestState()

	// Create a date: 2024-01-15
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(2))

	// Create interval
	yearsOp, _ := Get(dtrules.GetRName("years"))
	err := yearsOp.Execute(state)
	if err != nil {
		t.Fatalf("years operator failed: %v", err)
	}

	// Add the interval to the date
	plusOp, _ := Get(dtrules.GetRName("d+"))
	err = plusOp.Execute(state)
	if err != nil {
		t.Fatalf("d+ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(2, 0, 0)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestDateMinusWithDaysInterval(t *testing.T) {
	state := newTestState()

	// Create a date: 2024-01-15
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(10))

	// Create interval
	daysOp, _ := Get(dtrules.GetRName("days"))
	err := daysOp.Execute(state)
	if err != nil {
		t.Fatalf("days operator failed: %v", err)
	}

	// Subtract the interval from the date
	minusOp, _ := Get(dtrules.GetRName("d-"))
	err = minusOp.Execute(state)
	if err != nil {
		t.Fatalf("d- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(0, 0, -10)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestDateMinusWithMonthsInterval(t *testing.T) {
	state := newTestState()

	// Create a date: 2024-06-15
	baseDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(2))

	// Create interval
	monthsOp, _ := Get(dtrules.GetRName("months"))
	err := monthsOp.Execute(state)
	if err != nil {
		t.Fatalf("months operator failed: %v", err)
	}

	// Subtract the interval from the date
	minusOp, _ := Get(dtrules.GetRName("d-"))
	err = minusOp.Execute(state)
	if err != nil {
		t.Fatalf("d- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(0, -2, 0)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestDateMinusWithYearsInterval(t *testing.T) {
	state := newTestState()

	// Create a date: 2024-01-15
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(5))

	// Create interval
	yearsOp, _ := Get(dtrules.GetRName("years"))
	err := yearsOp.Execute(state)
	if err != nil {
		t.Fatalf("years operator failed: %v", err)
	}

	// Subtract the interval from the date
	minusOp, _ := Get(dtrules.GetRName("d-"))
	err = minusOp.Execute(state)
	if err != nil {
		t.Fatalf("d- operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(-5, 0, 0)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestIntervalWithNegativeAmount(t *testing.T) {
	state := newTestState()

	// Using negative amount: -30 days
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	state.DataPush(dtrules.GetRTime(baseDate))
	state.DataPush(dtrules.GetRIntegerValue(-30))

	// Create interval with negative amount
	daysOp, _ := Get(dtrules.GetRName("days"))
	err := daysOp.Execute(state)
	if err != nil {
		t.Fatalf("days operator failed: %v", err)
	}

	// Add the interval to the date (adding negative is like subtracting)
	plusOp, _ := Get(dtrules.GetRName("d+"))
	err = plusOp.Execute(state)
	if err != nil {
		t.Fatalf("d+ operator failed: %v", err)
	}

	result, _ := state.DataPop()
	resultTime, err := result.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue failed: %v", err)
	}

	expected := baseDate.AddDate(0, 0, -30)
	if !resultTime.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, resultTime)
	}
}

func TestIntervalStringValue(t *testing.T) {
	tests := []struct {
		amount   int
		unit     dtrules.IntervalUnit
		expected string
	}{
		{30, dtrules.IntervalDays, "30 days"},
		{1, dtrules.IntervalDays, "1 days"},
		{6, dtrules.IntervalMonths, "6 months"},
		{2, dtrules.IntervalYears, "2 years"},
		{-5, dtrules.IntervalDays, "-5 days"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			interval := dtrules.NewRInterval(tt.amount, tt.unit)
			if interval.StringValue() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, interval.StringValue())
			}
		})
	}
}

func TestIntervalEquals(t *testing.T) {
	i1 := dtrules.NewRInterval(30, dtrules.IntervalDays)
	i2 := dtrules.NewRInterval(30, dtrules.IntervalDays)
	i3 := dtrules.NewRInterval(31, dtrules.IntervalDays)
	i4 := dtrules.NewRInterval(30, dtrules.IntervalMonths)

	// Same interval
	eq, err := i1.Equals(i2)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if !eq {
		t.Error("Expected i1 == i2")
	}

	// Different amount
	eq, err = i1.Equals(i3)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if eq {
		t.Error("Expected i1 != i3")
	}

	// Different unit
	eq, err = i1.Equals(i4)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if eq {
		t.Error("Expected i1 != i4")
	}
}

func TestIntervalType(t *testing.T) {
	interval := dtrules.NewRInterval(30, dtrules.IntervalDays)
	if interval.Type() != dtrules.TypeInterval {
		t.Errorf("Expected TypeInterval, got %v", interval.Type())
	}
}

// =============================================================================
// XML Operator Tests
// =============================================================================

func TestNewXmlAttributeOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.NewRString("person"))

	op, ok := Get(dtrules.GetRName("newxmlattribute"))
	if !ok {
		t.Fatal("newxmlattribute operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("newxmlattribute operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	xmlValue, ok := result.(*dtrules.RXmlValue)
	if !ok {
		t.Fatalf("Result is not an RXmlValue: got %T", result)
	}

	if xmlValue.GetName() != "person" {
		t.Errorf("Expected element name 'person', got '%s'", xmlValue.GetName())
	}
}

func TestSetXmlAttributeOperator(t *testing.T) {
	state := newTestState()

	// Create XML element
	xmlValue := dtrules.NewRXmlValue("person")
	state.DataPush(xmlValue)
	state.DataPush(dtrules.NewRString("name"))
	state.DataPush(dtrules.NewRString("John"))

	op, ok := Get(dtrules.GetRName("setxmlattribute"))
	if !ok {
		t.Fatal("setxmlattribute operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("setxmlattribute operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	resultXml, ok := result.(*dtrules.RXmlValue)
	if !ok {
		t.Fatalf("Result is not an RXmlValue: got %T", result)
	}

	// Verify the attribute was set
	if resultXml.GetAttribute("name") != "John" {
		t.Errorf("Expected attribute 'name' to be 'John', got '%s'", resultXml.GetAttribute("name"))
	}
}

func TestGetXmlAttributeOperator(t *testing.T) {
	state := newTestState()

	// Create XML element with attribute
	xmlValue := dtrules.NewRXmlValue("person")
	xmlValue.SetAttribute("age", "30")
	state.DataPush(xmlValue)
	state.DataPush(dtrules.NewRString("age"))

	op, ok := Get(dtrules.GetRName("getxmlattribute"))
	if !ok {
		t.Fatal("getxmlattribute operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("getxmlattribute operator failed: %v", err)
	}

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}

	if result.StringValue() != "30" {
		t.Errorf("Expected '30', got '%s'", result.StringValue())
	}
}

func TestGetXmlAttributeNotFound(t *testing.T) {
	state := newTestState()

	// Create XML element without the requested attribute
	xmlValue := dtrules.NewRXmlValue("person")
	state.DataPush(xmlValue)
	state.DataPush(dtrules.NewRString("nonexistent"))

	op, _ := Get(dtrules.GetRName("getxmlattribute"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("getxmlattribute operator failed: %v", err)
	}

	result, _ := state.DataPop()
	// Should return empty string for non-existent attribute
	if result.StringValue() != "" {
		t.Errorf("Expected empty string, got '%s'", result.StringValue())
	}
}

func TestSetXmlAttributeTypeMismatch(t *testing.T) {
	state := newTestState()

	// Push a non-XML value
	state.DataPush(dtrules.GetRIntegerValue(42))
	state.DataPush(dtrules.NewRString("attr"))
	state.DataPush(dtrules.NewRString("value"))

	op, _ := Get(dtrules.GetRName("setxmlattribute"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected type mismatch error for setxmlattribute on integer")
	}
}

func TestGetXmlAttributeTypeMismatch(t *testing.T) {
	state := newTestState()

	// Push a non-XML value
	state.DataPush(dtrules.NewRString("not xml"))
	state.DataPush(dtrules.NewRString("attr"))

	op, _ := Get(dtrules.GetRName("getxmlattribute"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected type mismatch error for getxmlattribute on string")
	}
}

func TestXmlValueStringValue(t *testing.T) {
	xmlValue := dtrules.NewRXmlValue("person")
	xmlValue.SetAttribute("name", "John")
	xmlValue.SetAttribute("age", "30")

	str := xmlValue.StringValue()
	// The string should contain the element name and attributes
	if !strings.Contains(str, "person") {
		t.Errorf("StringValue should contain element name: %s", str)
	}
	if !strings.Contains(str, "name") {
		t.Errorf("StringValue should contain attribute name: %s", str)
	}
}

func TestXmlOperatorChaining(t *testing.T) {
	state := newTestState()

	// Test: newxmlattribute then setxmlattribute then getxmlattribute
	state.DataPush(dtrules.NewRString("element"))

	newOp, _ := Get(dtrules.GetRName("newxmlattribute"))
	err := newOp.Execute(state)
	if err != nil {
		t.Fatalf("newxmlattribute failed: %v", err)
	}

	// Set attribute
	state.DataPush(dtrules.NewRString("key"))
	state.DataPush(dtrules.NewRString("value"))

	setOp, _ := Get(dtrules.GetRName("setxmlattribute"))
	err = setOp.Execute(state)
	if err != nil {
		t.Fatalf("setxmlattribute failed: %v", err)
	}

	// Duplicate the XML element for getting (since get consumes it)
	dupOp, _ := Get(dtrules.GetRName("dup"))
	err = dupOp.Execute(state)
	if err != nil {
		t.Fatalf("dup failed: %v", err)
	}

	// Get attribute
	state.DataPush(dtrules.NewRString("key"))

	getOp, _ := Get(dtrules.GetRName("getxmlattribute"))
	err = getOp.Execute(state)
	if err != nil {
		t.Fatalf("getxmlattribute failed: %v", err)
	}

	result, _ := state.DataPop()
	if result.StringValue() != "value" {
		t.Errorf("Expected 'value', got '%s'", result.StringValue())
	}
}

// =============================================================================
// Tests for issues #126-133 (Go runtime bug fixes)
// =============================================================================

// TestPickOperator tests issue #126: opPick should use direct indexed access
func TestPickOperator(t *testing.T) {
	state := newTestState()

	// Push 5 values: 10, 20, 30, 40, 50 (50 is on top)
	state.DataPush(dtrules.GetRIntegerValue(10))
	state.DataPush(dtrules.GetRIntegerValue(20))
	state.DataPush(dtrules.GetRIntegerValue(30))
	state.DataPush(dtrules.GetRIntegerValue(40))
	state.DataPush(dtrules.GetRIntegerValue(50))

	// Pick element at index 2 (should be 30, counting from top: 50=0, 40=1, 30=2)
	state.DataPush(dtrules.GetRIntegerValue(2))

	op, _ := Get(dtrules.GetRName("pick"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("pick operator failed: %v", err)
	}

	// Should have 30 on top now
	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 30 {
		t.Errorf("Expected 30 from pick 2, got %d", val)
	}

	// Original stack should still have all 5 elements
	if state.DataStackDepth() != 5 {
		t.Errorf("Expected stack depth 5, got %d", state.DataStackDepth())
	}
}

// TestFloatDivideByZero tests issue #127: float division by zero returns error
func TestFloatDivideByZero(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRDoubleValue(100.0))
	state.DataPush(dtrules.GetRDoubleValue(0.0))

	op, _ := Get(dtrules.GetRName("fdiv"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected float divide by zero error, got nil")
	}
}

// TestFloatEqualityEpsilon tests issue #128: float equality uses epsilon comparison
func TestFloatEqualityEpsilon(t *testing.T) {
	// 0.1 + 0.2 is not exactly 0.3 in IEEE 754
	a := dtrules.GetRDoubleValue(0.1 + 0.2)
	b := dtrules.GetRDoubleValue(0.3)

	equal, err := a.Equals(b)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if !equal {
		t.Error("Expected 0.1 + 0.2 to equal 0.3 with epsilon comparison")
	}

	// Test compare returns 0 for nearly equal values
	cmp, err := a.Compare(b)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp != 0 {
		t.Errorf("Expected Compare to return 0, got %d", cmp)
	}
}

// TestIntegerOverflowAdd tests issue #129: integer addition overflow detection
func TestIntegerOverflowAdd(t *testing.T) {
	state := newTestState()

	// MaxInt64 + 1 should overflow
	state.DataPush(dtrules.GetRIntegerValue(9223372036854775807)) // MaxInt64
	state.DataPush(dtrules.GetRIntegerValue(1))

	op, _ := Get(dtrules.GetRName("+"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected integer overflow error for MaxInt64 + 1")
	}
	if err != nil && !strings.Contains(err.Error(), "overflow") {
		t.Errorf("Expected overflow error, got: %v", err)
	}
}

// TestIntegerOverflowSub tests issue #129: integer subtraction overflow detection
func TestIntegerOverflowSub(t *testing.T) {
	state := newTestState()

	// MinInt64 - 1 should overflow
	state.DataPush(dtrules.GetRIntegerValue(-9223372036854775808)) // MinInt64
	state.DataPush(dtrules.GetRIntegerValue(1))

	op, _ := Get(dtrules.GetRName("-"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected integer overflow error for MinInt64 - 1")
	}
	if err != nil && !strings.Contains(err.Error(), "overflow") {
		t.Errorf("Expected overflow error, got: %v", err)
	}
}

// TestIntegerOverflowMul tests issue #129: integer multiplication overflow detection
func TestIntegerOverflowMul(t *testing.T) {
	state := newTestState()

	// MaxInt64 * 2 should overflow
	state.DataPush(dtrules.GetRIntegerValue(9223372036854775807)) // MaxInt64
	state.DataPush(dtrules.GetRIntegerValue(2))

	op, _ := Get(dtrules.GetRName("*"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected integer overflow error for MaxInt64 * 2")
	}
	if err != nil && !strings.Contains(err.Error(), "overflow") {
		t.Errorf("Expected overflow error, got: %v", err)
	}
}

// TestForrOperator tests issue #130: opForr uses RArrayValue consistently
func TestForrOperator(t *testing.T) {
	state := newTestState()

	// Create array [1, 2, 3]
	arr, _ := dtrules.NewArray(&mockSession{}, false, false)
	arr.Add(dtrules.GetRIntegerValue(1))
	arr.Add(dtrules.GetRIntegerValue(2))
	arr.Add(dtrules.GetRIntegerValue(3))

	// Create body that just does 'pop' to consume each element
	// This tests that forr correctly iterates over the array
	body, _ := dtrules.NewArray(&mockSession{}, true, false) // executable
	popOp, _ := Get(dtrules.GetRName("pop"))
	body.Add(popOp)

	// Stack order: ( body array -- ) means push body first, then array
	state.DataPush(body)
	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("forr"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("forr operator failed: %v", err)
	}

	// Data stack should be empty (each element was popped)
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty data stack, got depth %d", state.DataStackDepth())
	}
}

// TestForrReverseOrder tests that forr iterates in reverse order
func TestForrReverseOrder(t *testing.T) {
	state := newTestState()

	// Create array [10, 20, 30]
	arr, _ := dtrules.NewArray(&mockSession{}, false, false)
	arr.Add(dtrules.GetRIntegerValue(10))
	arr.Add(dtrules.GetRIntegerValue(20))
	arr.Add(dtrules.GetRIntegerValue(30))

	// Create empty body (does nothing, leaves element on stack)
	body, _ := dtrules.NewArray(&mockSession{}, true, false) // executable

	// Stack order: ( body array -- ) means push body first, then array
	state.DataPush(body)
	state.DataPush(arr)

	op, _ := Get(dtrules.GetRName("forr"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("forr operator failed: %v", err)
	}

	// Stack should have 3 elements (each iteration pushed one)
	if state.DataStackDepth() != 3 {
		t.Fatalf("Expected stack depth 3, got %d", state.DataStackDepth())
	}

	// Pop in order: forr iterates from end to start, so pushes 30, 20, 10
	// Top of stack is 10 (last pushed)
	v1, _ := state.DataPop()
	v2, _ := state.DataPop()
	v3, _ := state.DataPop()

	val1, _ := v1.IntValue()
	val2, _ := v2.IntValue()
	val3, _ := v3.IntValue()

	// forr iterates in reverse (index 2, 1, 0), pushing 30, 20, 10
	if val1 != 10 || val2 != 20 || val3 != 30 {
		t.Errorf("Expected 10,20,30 (reverse iteration order), got %d,%d,%d", val1, val2, val3)
	}
}

// TestMaxIterationsConstant tests issue #131: MaxIterations is const, not mutable
func TestMaxIterationsConstant(t *testing.T) {
	// Verify DefaultMaxIterations is the expected value
	if DefaultMaxIterations != 1000000 {
		t.Errorf("Expected DefaultMaxIterations to be 1000000, got %d", DefaultMaxIterations)
	}
}

// TestModOperator tests issue #132: mod operator exists and works
func TestModOperator(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(17))
	state.DataPush(dtrules.GetRIntegerValue(5))

	op, ok := Get(dtrules.GetRName("mod"))
	if !ok {
		t.Fatal("mod operator not found")
	}

	err := op.Execute(state)
	if err != nil {
		t.Fatalf("mod operator failed: %v", err)
	}

	result, _ := state.DataPop()
	val, _ := result.IntValue()
	if val != 2 {
		t.Errorf("Expected 17 mod 5 = 2, got %d", val)
	}
}

// TestModByZero tests issue #132: mod by zero returns error
func TestModByZero(t *testing.T) {
	state := newTestState()

	state.DataPush(dtrules.GetRIntegerValue(17))
	state.DataPush(dtrules.GetRIntegerValue(0))

	op, _ := Get(dtrules.GetRName("mod"))
	err := op.Execute(state)
	if err == nil {
		t.Error("Expected mod by zero error")
	}
}

// TestNewDateUsesUTC tests issue #133: newdate uses UTC timezone
func TestNewDateUsesUTC(t *testing.T) {
	state := newTestState()

	// Push year, month, day
	state.DataPush(dtrules.GetRIntegerValue(2024))
	state.DataPush(dtrules.GetRIntegerValue(6))
	state.DataPush(dtrules.GetRIntegerValue(15))

	op, _ := Get(dtrules.GetRName("newdate"))
	err := op.Execute(state)
	if err != nil {
		t.Fatalf("newdate operator failed: %v", err)
	}

	result, _ := state.DataPop()
	timeVal, err := result.RTimeValue()
	if err != nil {
		t.Fatalf("RTimeValue failed: %v", err)
	}

	// Check that the timezone is UTC
	loc := timeVal.Time().Location()
	if loc != time.UTC {
		t.Errorf("Expected UTC timezone, got %v", loc)
	}

	// Verify the date components
	year, month, day := timeVal.Time().Date()
	if year != 2024 || month != time.June || day != 15 {
		t.Errorf("Expected 2024-06-15, got %d-%d-%d", year, month, day)
	}
}
