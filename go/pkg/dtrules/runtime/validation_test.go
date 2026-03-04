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

// Package runtime provides cross-runtime validation testing for DTRules bytecode execution.
package runtime

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/interpreter"
)

// TestVectorFile represents the structure of a test vector JSON file
type TestVectorFile struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Version     string       `json:"version"`
	Vectors     []TestVector `json:"vectors"`
}

// TestVector represents a single test case
type TestVector struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Bytecode    BytecodeSpec    `json:"bytecode"`
	Expected    ExpectedResult  `json:"expected"`
	ShouldError bool            `json:"should_error,omitempty"`
}

// BytecodeSpec defines the bytecode operations to execute
type BytecodeSpec struct {
	Operations []Operation `json:"operations"`
}

// Operation represents a single bytecode operation
type Operation struct {
	Op    string      `json:"op"`
	Value interface{} `json:"value,omitempty"`
}

// ExpectedResult defines the expected outcome of a test
type ExpectedResult struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

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

// findTestVectorsDir attempts to locate the test/vectors directory
func findTestVectorsDir() string {
	// Try multiple possible locations
	candidates := []string{
		// From go/pkg/dtrules/runtime/ to test/vectors/
		filepath.Join("..", "..", "..", "..", "test", "vectors"),
		// Absolute path fallback (for CI environments)
		"test/vectors",
		// From project root
		filepath.Join("..", "..", "..", "..", "..", "test", "vectors"),
	}

	// Also try to find based on go.mod location
	dir, err := os.Getwd()
	if err == nil {
		// Walk up to find go.mod
		for d := dir; d != "/" && d != "."; d = filepath.Dir(d) {
			goMod := filepath.Join(d, "go.mod")
			if _, err := os.Stat(goMod); err == nil {
				// Found go.mod, look for test/vectors relative to parent
				projectRoot := filepath.Dir(d)
				vectorsDir := filepath.Join(projectRoot, "test", "vectors")
				candidates = append([]string{vectorsDir}, candidates...)
				break
			}
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// TestCrossRuntimeValidation runs all test vectors found in test/vectors/*.json
func TestCrossRuntimeValidation(t *testing.T) {
	// Find test vector files
	vectorsDir := findTestVectorsDir()

	// Check if directory exists
	if vectorsDir == "" {
		t.Log("Test vectors directory not found, running embedded tests only")
		runEmbeddedTests(t)
		return
	}

	files, err := filepath.Glob(filepath.Join(vectorsDir, "*.json"))
	if err != nil {
		t.Fatalf("Failed to find test vector files: %v", err)
	}

	if len(files) == 0 {
		t.Log("No test vector files found, running embedded tests only")
		runEmbeddedTests(t)
		return
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			runTestVectorFile(t, file)
		})
	}
}

// runTestVectorFile executes all test vectors in a single file
func runTestVectorFile(t *testing.T, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read test vector file %s: %v", filename, err)
	}

	var vectorFile TestVectorFile
	if err := json.Unmarshal(data, &vectorFile); err != nil {
		t.Fatalf("Failed to parse test vector file %s: %v", filename, err)
	}

	t.Logf("Running test vectors from: %s (%s)", vectorFile.Name, vectorFile.Description)

	for _, vector := range vectorFile.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			runSingleVector(t, vector)
		})
	}
}

// runSingleVector executes a single test vector
func runSingleVector(t *testing.T, vector TestVector) {
	session := &mockSession{}
	state := interpreter.NewDTState(session)

	// Build bytecode from the vector specification
	bc := dtrules.NewBytecodeChunk()
	for _, op := range vector.Bytecode.Operations {
		if err := emitOperation(bc, op); err != nil {
			if vector.ShouldError {
				return // Expected error during bytecode construction
			}
			t.Fatalf("Failed to emit operation %s: %v", op.Op, err)
		}
	}

	// Clear the value stack
	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	// Execute the bytecode
	err := state.ExecuteBytecode(bc)
	if vector.ShouldError {
		if err == nil {
			t.Error("Expected error but execution succeeded")
		}
		return
	}
	if err != nil {
		t.Fatalf("Bytecode execution failed: %v", err)
	}

	// Check the result
	if state.ValueStackDepth() == 0 {
		t.Fatal("No result on stack after execution")
	}

	result, err := state.ValuePop()
	if err != nil {
		t.Fatalf("Failed to pop result: %v", err)
	}

	// Verify the result matches expected
	if !verifyResult(t, result, vector.Expected) {
		t.Errorf("Result mismatch for %s: got %v, expected %v",
			vector.Name, result, vector.Expected.Value)
	}
}

// emitOperation converts a JSON operation to bytecode
func emitOperation(bc *dtrules.BytecodeChunk, op Operation) error {
	switch op.Op {
	// Push constants
	case "push_true":
		bc.Emit(dtrules.OpPushTrue)
	case "push_false":
		bc.Emit(dtrules.OpPushFalse)
	case "push_null":
		bc.Emit(dtrules.OpPushNull)
	case "push_zero":
		bc.Emit(dtrules.OpPushZero)
	case "push_one":
		bc.Emit(dtrules.OpPushOne)
	case "push_int":
		val := toInt64(op.Value)
		bc.EmitPushConstant(dtrules.NewValueInteger(val))
	case "push_double":
		val := toFloat64(op.Value)
		bc.EmitPushConstant(dtrules.NewValueDouble(val))
	case "push_string":
		val := toString(op.Value)
		bc.EmitPushConstant(dtrules.NewValueString(val))

	// Arithmetic operations
	case "add":
		bc.Emit(dtrules.OpAdd)
	case "sub":
		bc.Emit(dtrules.OpSub)
	case "mul":
		bc.Emit(dtrules.OpMul)
	case "div":
		bc.Emit(dtrules.OpDiv)
	case "neg":
		bc.Emit(dtrules.OpNeg)
	case "inc":
		bc.Emit(dtrules.OpInc)
	case "dec":
		bc.Emit(dtrules.OpDec)

	// Comparison operations
	case "eq":
		bc.Emit(dtrules.OpEq)
	case "ne":
		bc.Emit(dtrules.OpNe)
	case "lt":
		bc.Emit(dtrules.OpLt)
	case "le":
		bc.Emit(dtrules.OpLe)
	case "gt":
		bc.Emit(dtrules.OpGt)
	case "ge":
		bc.Emit(dtrules.OpGe)

	// Boolean operations
	case "and":
		bc.Emit(dtrules.OpAnd)
	case "or":
		bc.Emit(dtrules.OpOr)
	case "not":
		bc.Emit(dtrules.OpNot)
	case "xor":
		bc.Emit(dtrules.OpXor)

	// Stack operations
	case "pop":
		bc.Emit(dtrules.OpPop)
	case "dup":
		bc.Emit(dtrules.OpDup)
	case "swap":
		bc.Emit(dtrules.OpSwap)
	case "rot":
		bc.Emit(dtrules.OpRot)
	case "nop":
		bc.Emit(dtrules.OpNop)

	default:
		return dtrules.NewRulesError("Invalid Operation", "emitOperation",
			"unknown operation: "+op.Op)
	}
	return nil
}

// verifyResult checks if the actual result matches the expected result
func verifyResult(t *testing.T, actual dtrules.Value, expected ExpectedResult) bool {
	switch expected.Type {
	case "integer":
		if !actual.IsInteger() {
			t.Logf("Expected integer, got tag %d", actual.Tag())
			return false
		}
		expectedVal := toInt64(expected.Value)
		actualVal := actual.AsInteger()
		if actualVal != expectedVal {
			t.Logf("Integer mismatch: got %d, expected %d", actualVal, expectedVal)
			return false
		}
		return true

	case "double":
		if !actual.IsDouble() && !actual.IsInteger() {
			t.Logf("Expected numeric, got tag %d", actual.Tag())
			return false
		}
		expectedVal := toFloat64(expected.Value)
		actualVal := actual.AsDouble()
		// Use epsilon comparison for floats
		if math.Abs(actualVal-expectedVal) > 1e-9 {
			t.Logf("Double mismatch: got %f, expected %f", actualVal, expectedVal)
			return false
		}
		return true

	case "boolean":
		if !actual.IsBoolean() {
			t.Logf("Expected boolean, got tag %d", actual.Tag())
			return false
		}
		expectedVal := toBool(expected.Value)
		actualVal := actual.AsBoolean()
		if actualVal != expectedVal {
			t.Logf("Boolean mismatch: got %t, expected %t", actualVal, expectedVal)
			return false
		}
		return true

	case "string":
		if !actual.IsString() {
			t.Logf("Expected string, got tag %d", actual.Tag())
			return false
		}
		expectedVal := toString(expected.Value)
		actualVal := actual.AsString()
		if actualVal != expectedVal {
			t.Logf("String mismatch: got %s, expected %s", actualVal, expectedVal)
			return false
		}
		return true

	case "null":
		if !actual.IsNull() {
			t.Logf("Expected null, got tag %d", actual.Tag())
			return false
		}
		return true

	default:
		t.Logf("Unknown expected type: %s", expected.Type)
		return false
	}
}

// Helper functions for type conversion from JSON values
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// runEmbeddedTests runs a set of hardcoded test vectors when no external files are available
func runEmbeddedTests(t *testing.T) {
	embeddedVectors := []TestVector{
		{
			Name:        "embedded_add",
			Description: "Basic addition test",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 10},
					{Op: "push_int", Value: 20},
					{Op: "add"},
				},
			},
			Expected: ExpectedResult{Type: "integer", Value: int64(30)},
		},
		{
			Name:        "embedded_sub",
			Description: "Basic subtraction test",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 50},
					{Op: "push_int", Value: 20},
					{Op: "sub"},
				},
			},
			Expected: ExpectedResult{Type: "integer", Value: int64(30)},
		},
		{
			Name:        "embedded_mul",
			Description: "Basic multiplication test",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 6},
					{Op: "push_int", Value: 7},
					{Op: "mul"},
				},
			},
			Expected: ExpectedResult{Type: "integer", Value: int64(42)},
		},
		{
			Name:        "embedded_div",
			Description: "Basic division test",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 100},
					{Op: "push_int", Value: 5},
					{Op: "div"},
				},
			},
			Expected: ExpectedResult{Type: "double", Value: 20.0},
		},
		{
			Name:        "embedded_comparison_eq",
			Description: "Equality comparison",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 5},
					{Op: "push_int", Value: 5},
					{Op: "eq"},
				},
			},
			Expected: ExpectedResult{Type: "boolean", Value: true},
		},
		{
			Name:        "embedded_comparison_lt",
			Description: "Less than comparison",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 5},
					{Op: "push_int", Value: 10},
					{Op: "lt"},
				},
			},
			Expected: ExpectedResult{Type: "boolean", Value: true},
		},
		{
			Name:        "embedded_boolean_and",
			Description: "Boolean AND operation",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_true"},
					{Op: "push_true"},
					{Op: "and"},
				},
			},
			Expected: ExpectedResult{Type: "boolean", Value: true},
		},
		{
			Name:        "embedded_boolean_not",
			Description: "Boolean NOT operation",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_true"},
					{Op: "not"},
				},
			},
			Expected: ExpectedResult{Type: "boolean", Value: false},
		},
		{
			Name:        "embedded_stack_dup",
			Description: "Stack duplicate operation",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 21},
					{Op: "dup"},
					{Op: "add"},
				},
			},
			Expected: ExpectedResult{Type: "integer", Value: int64(42)},
		},
		{
			Name:        "embedded_complex",
			Description: "Complex arithmetic: (10 + 20) * 3",
			Bytecode: BytecodeSpec{
				Operations: []Operation{
					{Op: "push_int", Value: 10},
					{Op: "push_int", Value: 20},
					{Op: "add"},
					{Op: "push_int", Value: 3},
					{Op: "mul"},
				},
			},
			Expected: ExpectedResult{Type: "integer", Value: int64(90)},
		},
	}

	for _, vector := range embeddedVectors {
		t.Run(vector.Name, func(t *testing.T) {
			runSingleVector(t, vector)
		})
	}
}

// TestValidationFramework verifies the validation framework itself works correctly
func TestValidationFramework(t *testing.T) {
	// Test type conversion helpers
	t.Run("toInt64", func(t *testing.T) {
		if toInt64(float64(42)) != 42 {
			t.Error("Failed to convert float64 to int64")
		}
		if toInt64(int(42)) != 42 {
			t.Error("Failed to convert int to int64")
		}
		if toInt64(int64(42)) != 42 {
			t.Error("Failed to convert int64 to int64")
		}
	})

	t.Run("toFloat64", func(t *testing.T) {
		if toFloat64(float64(3.14)) != 3.14 {
			t.Error("Failed to convert float64")
		}
		if toFloat64(int(42)) != 42.0 {
			t.Error("Failed to convert int to float64")
		}
	})

	t.Run("toBool", func(t *testing.T) {
		if !toBool(true) {
			t.Error("Failed to convert true")
		}
		if toBool(false) {
			t.Error("Failed to convert false")
		}
	})

	t.Run("toString", func(t *testing.T) {
		if toString("hello") != "hello" {
			t.Error("Failed to convert string")
		}
	})
}

// TestBytecodeEmission verifies that all supported operations can be emitted
func TestBytecodeEmission(t *testing.T) {
	operations := []string{
		"push_true", "push_false", "push_null", "push_zero", "push_one",
		"add", "sub", "mul", "div", "neg", "inc", "dec",
		"eq", "ne", "lt", "le", "gt", "ge",
		"and", "or", "not", "xor",
		"pop", "dup", "swap", "rot", "nop",
	}

	for _, opName := range operations {
		t.Run(opName, func(t *testing.T) {
			bc := dtrules.NewBytecodeChunk()
			err := emitOperation(bc, Operation{Op: opName})
			if err != nil {
				t.Errorf("Failed to emit %s: %v", opName, err)
			}
			if len(bc.Code()) == 0 {
				t.Errorf("No bytecode emitted for %s", opName)
			}
		})
	}

	// Test operations with values
	t.Run("push_int", func(t *testing.T) {
		bc := dtrules.NewBytecodeChunk()
		err := emitOperation(bc, Operation{Op: "push_int", Value: 42})
		if err != nil {
			t.Errorf("Failed to emit push_int: %v", err)
		}
	})

	t.Run("push_double", func(t *testing.T) {
		bc := dtrules.NewBytecodeChunk()
		err := emitOperation(bc, Operation{Op: "push_double", Value: 3.14})
		if err != nil {
			t.Errorf("Failed to emit push_double: %v", err)
		}
	})

	t.Run("push_string", func(t *testing.T) {
		bc := dtrules.NewBytecodeChunk()
		err := emitOperation(bc, Operation{Op: "push_string", Value: "hello"})
		if err != nil {
			t.Errorf("Failed to emit push_string: %v", err)
		}
	})

	// Test unknown operation
	t.Run("unknown_op", func(t *testing.T) {
		bc := dtrules.NewBytecodeChunk()
		err := emitOperation(bc, Operation{Op: "unknown_operation"})
		if err == nil {
			t.Error("Expected error for unknown operation")
		}
	})
}
