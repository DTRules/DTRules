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

package compiler

import (
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
)

// mockSession implements minimal Session for testing
type mockSession struct {
	uniqueID int
	factory  *entity.Factory
}

func (m *mockSession) GetState() dtrules.State                 { return nil }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory { return m.factory }
func (m *mockSession) GetUniqueID() int                        { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser       { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet             { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return m.factory.CreateEntity(m, name)
}
func (m *mockSession) Compile(expr string) (dtrules.Object, error) {
	return nil, nil
}

func newTestCompiler() *Compiler {
	factory := entity.NewFactory(nil)
	session := &mockSession{factory: factory}
	return NewCompiler(session, factory)
}

func TestCompileEmpty(t *testing.T) {
	c := newTestCompiler()

	// Empty string
	result, err := c.Compile("")
	if err != nil {
		t.Fatalf("Compile empty failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}
	if result.Type() != dtrules.TypeArray {
		t.Errorf("Expected array type, got %v", result.Type())
	}
}

func TestCompileInteger(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("42")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, err := result.ArrayValue()
	if err != nil {
		t.Fatalf("ArrayValue failed: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(arr))
	}

	val, err := arr[0].IntValue()
	if err != nil {
		t.Fatalf("IntValue failed: %v", err)
	}
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestCompileNegativeInteger(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("-123")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	val, _ := arr[0].IntValue()
	if val != -123 {
		t.Errorf("Expected -123, got %d", val)
	}
}

func TestCompileDouble(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("3.14")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	val, _ := arr[0].DoubleValue()
	if val != 3.14 {
		t.Errorf("Expected 3.14, got %f", val)
	}
}

func TestCompileString(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("\"hello world\"")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if arr[0].StringValue() != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", arr[0].StringValue())
	}
}

func TestCompileSingleQuoteString(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("'hello'")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if arr[0].StringValue() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", arr[0].StringValue())
	}
}

func TestCompileBoolean(t *testing.T) {
	c := newTestCompiler()

	// True
	result, err := c.Compile("true")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	arr, _ := result.ArrayValue()
	val, _ := arr[0].BooleanValue()
	if !val {
		t.Error("Expected true")
	}

	// False
	result, err = c.Compile("false")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	arr, _ = result.ArrayValue()
	val, _ = arr[0].BooleanValue()
	if val {
		t.Error("Expected false")
	}
}

func TestCompileNull(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("null")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if arr[0].Type() != dtrules.TypeNull {
		t.Errorf("Expected null type, got %v", arr[0].Type())
	}
}

func TestCompileOperator(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("+")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if arr[0].Type() != dtrules.TypeOperator {
		t.Errorf("Expected operator type, got %v", arr[0].Type())
	}
}

func TestCompileExpression(t *testing.T) {
	c := newTestCompiler()

	// Simple addition: 10 20 +
	result, err := c.Compile("10 20 +")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}

	// Check first element (10)
	v1, _ := arr[0].IntValue()
	if v1 != 10 {
		t.Errorf("Expected 10, got %d", v1)
	}

	// Check second element (20)
	v2, _ := arr[1].IntValue()
	if v2 != 20 {
		t.Errorf("Expected 20, got %d", v2)
	}

	// Check third element (+ operator)
	if arr[2].Type() != dtrules.TypeOperator {
		t.Errorf("Expected operator, got %v", arr[2].Type())
	}
}

func TestCompileCodeBlock(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("{ 1 2 + }")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 1 {
		t.Fatalf("Expected 1 element (the code block), got %d", len(arr))
	}

	// The code block should be an array
	if arr[0].Type() != dtrules.TypeArray {
		t.Errorf("Expected array type for code block, got %v", arr[0].Type())
	}

	// Check contents of code block
	innerArr, _ := arr[0].ArrayValue()
	if len(innerArr) != 3 {
		t.Errorf("Expected 3 elements in code block, got %d", len(innerArr))
	}
}

func TestCompileNestedCodeBlock(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("{ { 1 } { 2 } }")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 1 {
		t.Fatalf("Expected 1 outer element, got %d", len(arr))
	}

	innerArr, _ := arr[0].ArrayValue()
	if len(innerArr) != 2 {
		t.Errorf("Expected 2 nested code blocks, got %d", len(innerArr))
	}
}

func TestCompileMultipleSpaces(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("1   2    3")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}
}

func TestCompileWithNewlines(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("1\n2\n3")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}
}

func TestCompileName(t *testing.T) {
	c := newTestCompiler()

	result, err := c.Compile("myVariable")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	// Names that aren't operators should be compiled as executable names
	if arr[0].Type() != dtrules.TypeName {
		t.Errorf("Expected name type, got %v", arr[0].Type())
	}
}

func TestCompileComplexExpression(t *testing.T) {
	c := newTestCompiler()

	// A more complex expression
	result, err := c.Compile("10 20 + 5 * 2 /")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 7 {
		t.Fatalf("Expected 7 elements, got %d", len(arr))
	}
}

func TestCompileConditional(t *testing.T) {
	c := newTestCompiler()

	// true { 1 } { 2 } ifelse
	result, err := c.Compile("true { 1 } { 2 } ifelse")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	arr, _ := result.ArrayValue()
	if len(arr) != 4 {
		t.Fatalf("Expected 4 elements, got %d", len(arr))
	}

	// First should be boolean
	b, _ := arr[0].BooleanValue()
	if !b {
		t.Error("Expected true")
	}

	// Second and third should be arrays (code blocks)
	if arr[1].Type() != dtrules.TypeArray {
		t.Errorf("Expected array for then-block, got %v", arr[1].Type())
	}
	if arr[2].Type() != dtrules.TypeArray {
		t.Errorf("Expected array for else-block, got %v", arr[2].Type())
	}

	// Fourth should be ifelse operator
	if arr[3].Type() != dtrules.TypeOperator {
		t.Errorf("Expected operator for ifelse, got %v", arr[3].Type())
	}
}
