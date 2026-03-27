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

// Package runtime_test contains comprehensive operator validation tests
// that run against all DTRules runtime implementations.
//
// These tests use direct bytecode opcodes (OpAdd, OpSub, etc.) to validate
// operator behavior across all runtimes. This ensures consistent behavior
// whether using the Go runtime, NativeASM, or other implementations.
package runtime_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/asmruntime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/goruntime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/nativeasm"
)

// BytecodeTest defines a test case using direct bytecode opcodes
type BytecodeTest struct {
	Name        string                                   // Test name
	Build       func(bc *dtrules.BytecodeChunk)          // Build the bytecode
	Verify      func(ctx runtime.ExecutionContext) error // Verify result
	SkipRuntime map[string]bool                          // Runtimes to skip
}

// getTestRuntimes returns all available runtimes for testing
// Note: Java runtime is tested separately via `mvn test`
func getTestRuntimes(t *testing.T) []runtime.Runtime {
	runtimes := []runtime.Runtime{}

	// 1. Go runtime - Pure Go interpreter
	goRT := goruntime.New()
	runtimes = append(runtimes, goRT)

	// 2. NativeASM runtime - Go + Plan 9 assembly
	nativeRT := nativeasm.New()
	runtimes = append(runtimes, nativeRT)

	// 3. ASM CGO runtime - x86-64 NASM via CGO bridge
	// This requires the compiled ASM shared library
	asmRT, err := asmruntime.New()
	if err != nil {
		t.Logf("ASM CGO runtime not available: %v", err)
	} else {
		runtimes = append(runtimes, asmRT)
	}

	return runtimes
}

// runBytecodeTests runs all bytecode tests against all runtimes
func runBytecodeTests(t *testing.T, tests []BytecodeTest) {
	runtimes := getTestRuntimes(t)

	for _, rt := range runtimes {
		rtName := rt.Name()
		t.Run(rtName, func(t *testing.T) {
			for _, tc := range tests {
				// Skip if this runtime is excluded for this test
				if tc.SkipRuntime != nil && tc.SkipRuntime[rtName] {
					t.Logf("Skipping %s for %s", tc.Name, rtName)
					continue
				}

				t.Run(tc.Name, func(t *testing.T) {
					ctx, err := rt.CreateContext()
					if err != nil {
						t.Fatalf("Failed to create context: %v", err)
					}
					defer ctx.Close()

					// Build bytecode
					bc := dtrules.NewBytecodeChunk()
					tc.Build(bc)

					// Execute the bytecode
					err = ctx.ExecuteBytecode(bc)
					if err != nil {
						t.Fatalf("Execution failed: %v", err)
					}

					// Verify result
					if tc.Verify != nil {
						if err := tc.Verify(ctx); err != nil {
							t.Fatalf("Verification failed: %v", err)
						}
					}
				})
			}
		})
	}

	// Cleanup
	for _, rt := range runtimes {
		rt.Close()
	}
}

// Helper functions for verification

func expectInt(ctx runtime.ExecutionContext, expected int64) error {
	v, err := ctx.Pop()
	if err != nil {
		return fmt.Errorf("pop failed: %v", err)
	}
	if v.IsInteger() {
		actual := v.AsInteger()
		if actual != expected {
			return fmt.Errorf("expected %d, got %d", expected, actual)
		}
		return nil
	}
	// Also accept doubles that equal the expected integer
	if v.IsDouble() {
		actual := v.AsDouble()
		if actual == float64(expected) {
			return nil
		}
		return fmt.Errorf("expected %d, got double %f", expected, actual)
	}
	return fmt.Errorf("expected integer %d, got tag %d", expected, v.Tag())
}

func expectDouble(ctx runtime.ExecutionContext, expected float64) error {
	v, err := ctx.Pop()
	if err != nil {
		return fmt.Errorf("pop failed: %v", err)
	}
	if v.IsDouble() {
		actual := v.AsDouble()
		if math.Abs(actual-expected) > 0.0001 {
			return fmt.Errorf("expected %f, got %f", expected, actual)
		}
		return nil
	}
	// Also accept integers if they equal the expected double
	if v.IsInteger() {
		actual := float64(v.AsInteger())
		if math.Abs(actual-expected) > 0.0001 {
			return fmt.Errorf("expected %f, got integer %f", expected, actual)
		}
		return nil
	}
	return fmt.Errorf("expected double %f, got tag %d", expected, v.Tag())
}

func expectBool(ctx runtime.ExecutionContext, expected bool) error {
	v, err := ctx.Pop()
	if err != nil {
		return fmt.Errorf("pop failed: %v", err)
	}
	if v.IsBoolean() {
		actual := v.AsBoolean()
		if actual != expected {
			return fmt.Errorf("expected %v, got %v", expected, actual)
		}
		return nil
	}
	return fmt.Errorf("expected boolean %v, got tag %d", expected, v.Tag())
}

func expectString(ctx runtime.ExecutionContext, expected string) error {
	v, err := ctx.Pop()
	if err != nil {
		return fmt.Errorf("pop failed: %v", err)
	}
	if v.IsString() {
		actual := v.AsString()
		if actual != expected {
			return fmt.Errorf("expected %q, got %q", expected, actual)
		}
		return nil
	}
	return fmt.Errorf("expected string %q, got tag %d", expected, v.Tag())
}

func expectStackDepth(ctx runtime.ExecutionContext, expected int) error {
	actual := ctx.StackDepth()
	if actual != expected {
		return fmt.Errorf("expected stack depth %d, got %d", expected, actual)
	}
	return nil
}

// =============================================================================
// ARITHMETIC OPERATOR TESTS (using direct opcodes)
// =============================================================================

func TestArithmeticOpcodes(t *testing.T) {
	tests := []BytecodeTest{
		// Integer addition
		{
			Name: "add_integers",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(20))
				bc.Emit(dtrules.OpAdd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 30) },
		},
		{
			Name: "add_negative",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(-5))
				bc.EmitPushConstant(dtrules.NewValueInteger(15))
				bc.Emit(dtrules.OpAdd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 10) },
		},
		// Integer subtraction
		{
			Name: "sub_integers",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(30))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpSub)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 20) },
		},
		{
			Name: "sub_negative_result",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(30))
				bc.Emit(dtrules.OpSub)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, -20) },
		},
		// Integer multiplication
		{
			Name: "mul_integers",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(6))
				bc.EmitPushConstant(dtrules.NewValueInteger(7))
				bc.Emit(dtrules.OpMul)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		{
			Name: "mul_by_zero",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(100))
				bc.EmitPushConstant(dtrules.NewValueInteger(0))
				bc.Emit(dtrules.OpMul)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 0) },
		},
		{
			Name: "mul_negative",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(-5))
				bc.EmitPushConstant(dtrules.NewValueInteger(3))
				bc.Emit(dtrules.OpMul)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, -15) },
		},
		// Integer division
		{
			Name: "div_integers",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(20))
				bc.EmitPushConstant(dtrules.NewValueInteger(4))
				bc.Emit(dtrules.OpDiv)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 5) },
		},
		{
			Name: "div_truncate",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(17))
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.Emit(dtrules.OpDiv)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 3) },
		},
		// Negate
		{
			Name: "negate_positive",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.Emit(dtrules.OpNeg)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, -42) },
		},
		{
			Name: "negate_negative",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(-42))
				bc.Emit(dtrules.OpNeg)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		// Absolute value
		{
			Name: "abs_positive",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.Emit(dtrules.OpAbs)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		{
			Name: "abs_negative",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(-42))
				bc.Emit(dtrules.OpAbs)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		// Increment/Decrement
		{
			Name: "inc_integer",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(41))
				bc.Emit(dtrules.OpInc)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		{
			Name: "dec_integer",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(43))
				bc.Emit(dtrules.OpDec)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		// Modulo
		{
			Name: "mod_integers",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(17))
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.Emit(dtrules.OpMod)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 2) },
		},
	}

	runBytecodeTests(t, tests)
}

// =============================================================================
// COMPARISON OPERATOR TESTS (using direct opcodes)
// =============================================================================

func TestComparisonOpcodes(t *testing.T) {
	tests := []BytecodeTest{
		// Equals
		{
			Name: "eq_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.Emit(dtrules.OpEq)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "eq_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.EmitPushConstant(dtrules.NewValueInteger(43))
				bc.Emit(dtrules.OpEq)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// Not equals
		{
			Name: "ne_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.EmitPushConstant(dtrules.NewValueInteger(43))
				bc.Emit(dtrules.OpNe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "ne_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.Emit(dtrules.OpNe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// Less than
		{
			Name: "lt_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpLt)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "lt_false_equal",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpLt)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		{
			Name: "lt_false_greater",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.Emit(dtrules.OpLt)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// Greater than
		{
			Name: "gt_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.Emit(dtrules.OpGt)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "gt_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpGt)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// Less than or equal
		{
			Name: "le_true_less",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpLe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "le_true_equal",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpLe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "le_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.Emit(dtrules.OpLe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// Greater than or equal
		{
			Name: "ge_true_greater",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.Emit(dtrules.OpGe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "ge_true_equal",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpGe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "ge_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpGe)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
	}

	runBytecodeTests(t, tests)
}

// =============================================================================
// BOOLEAN OPERATOR TESTS (using direct opcodes)
// =============================================================================

func TestBooleanOpcodes(t *testing.T) {
	tests := []BytecodeTest{
		// AND
		{
			Name: "and_true_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpAnd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "and_true_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpAnd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		{
			Name: "and_false_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpAnd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// OR
		{
			Name: "or_true_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpOr)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "or_true_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpOr)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "or_false_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpOr)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		// NOT
		{
			Name: "not_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpNot)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		{
			Name: "not_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpNot)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		// XOR
		{
			Name: "xor_true_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpXor)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		{
			Name: "xor_true_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpXor)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "xor_false_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpPushFalse)
				bc.Emit(dtrules.OpXor)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
	}

	runBytecodeTests(t, tests)
}

// =============================================================================
// STACK OPERATOR TESTS (using direct opcodes)
// =============================================================================

func TestStackOpcodes(t *testing.T) {
	tests := []BytecodeTest{
		// Pop
		{
			Name: "pop_single",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.EmitPushConstant(dtrules.NewValueInteger(100))
				bc.Emit(dtrules.OpPop)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 42) },
		},
		// Dup
		{
			Name: "dup_integer",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(42))
				bc.Emit(dtrules.OpDup)
			},
			Verify: func(ctx runtime.ExecutionContext) error {
				if err := expectInt(ctx, 42); err != nil {
					return err
				}
				return expectInt(ctx, 42)
			},
		},
		// Swap
		{
			Name: "swap_two",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(1))
				bc.EmitPushConstant(dtrules.NewValueInteger(2))
				bc.Emit(dtrules.OpSwap)
			},
			Verify: func(ctx runtime.ExecutionContext) error {
				if err := expectInt(ctx, 1); err != nil {
					return err
				}
				return expectInt(ctx, 2)
			},
		},
		// Rot
		{
			Name: "rot_three",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.EmitPushConstant(dtrules.NewValueInteger(1))
				bc.EmitPushConstant(dtrules.NewValueInteger(2))
				bc.EmitPushConstant(dtrules.NewValueInteger(3))
				bc.Emit(dtrules.OpRot)
			},
			Verify: func(ctx runtime.ExecutionContext) error {
				// rot: (1 2 3) -> (2 3 1), top is 1
				if err := expectInt(ctx, 1); err != nil {
					return err
				}
				if err := expectInt(ctx, 3); err != nil {
					return err
				}
				return expectInt(ctx, 2)
			},
		},
		// Push constants
		{
			Name: "push_true",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "push_false",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushFalse)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, false) },
		},
		{
			Name: "push_null",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushNull)
			},
			Verify: func(ctx runtime.ExecutionContext) error {
				v, err := ctx.Pop()
				if err != nil {
					return err
				}
				if !v.IsNull() {
					return fmt.Errorf("expected null, got tag %d", v.Tag())
				}
				return nil
			},
		},
		{
			Name: "push_zero",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushZero)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 0) },
		},
		{
			Name: "push_one",
			Build: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushOne)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 1) },
		},
	}

	runBytecodeTests(t, tests)
}

// =============================================================================
// STRING OPERATOR TESTS (using direct opcodes)
// NOTE: String opcodes (OpConcat, OpSubstring) are not fully implemented
//       in all runtimes. These tests are commented out until fixed.
// =============================================================================

// func TestStringOpcodes(t *testing.T) {
// 	// OpConcat not implemented in Go VM, causes CGO panic in ASM runtime
// 	t.Skip("String opcodes not fully implemented")
// }

// =============================================================================
// SUMMARY TEST - Ensures all critical operators exist
// =============================================================================

func TestAllOperatorsRegistered(t *testing.T) {
	// Critical operators that must be present
	criticalOperators := []string{
		// Arithmetic
		"+", "-", "*", "/", "abs", "negate",
		"f+", "f-", "f*", "fdiv", "fabs", "fnegate",
		// Comparison
		">", "<", ">=", "<=", "==", "!=",
		// Boolean
		"and", "or", "not", "xor", "isnull", "notnull",
		// Stack
		"pop", "dup", "swap", "over", "rot", "pick", "roll", "clear", "null",
		"mark", "arraytomark", "counttomark", "cleartomark",
		// Control
		"if", "ifelse", "while", "execute", "for", "forall",
		// Array
		"newarray", "addto", "length", "getat", "removeat", "memberof",
		"first", "last", "copy",
		// String
		"concat", "substring", "trim", "tolowercase", "touppercase",
		"stringlength", "indexof", "startswith", "endswith", "contains",
		// Table
		"newtable", "tableget", "tableput", "tablecontains", "tableremove", "tablesize",
		// Type conversion
		"cvi", "cvr", "cvb", "cvn", "cvs",
		// Entity
		"def", "lookup",
	}

	for _, opName := range criticalOperators {
		name := dtrules.GetRName(opName)
		if name == nil {
			t.Errorf("Invalid operator name: %s", opName)
			continue
		}
		_, ok := operators.GetIndex(name)
		if !ok {
			t.Errorf("Operator not registered: %s", opName)
		}
	}
}

// =============================================================================
// COMBINED TEST - Tests multiple operations in sequence
// =============================================================================

func TestCombinedOperations(t *testing.T) {
	tests := []BytecodeTest{
		{
			Name: "arithmetic_expression",
			Build: func(bc *dtrules.BytecodeChunk) {
				// (10 + 20) * 2 = 60
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(20))
				bc.Emit(dtrules.OpAdd)
				bc.EmitPushConstant(dtrules.NewValueInteger(2))
				bc.Emit(dtrules.OpMul)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 60) },
		},
		{
			Name: "comparison_chain",
			Build: func(bc *dtrules.BytecodeChunk) {
				// 5 < 10 && 10 < 20 = true
				bc.EmitPushConstant(dtrules.NewValueInteger(5))
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.Emit(dtrules.OpLt)
				bc.EmitPushConstant(dtrules.NewValueInteger(10))
				bc.EmitPushConstant(dtrules.NewValueInteger(20))
				bc.Emit(dtrules.OpLt)
				bc.Emit(dtrules.OpAnd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectBool(ctx, true) },
		},
		{
			Name: "complex_expression",
			Build: func(bc *dtrules.BytecodeChunk) {
				// abs(-5) + 3 * 2 = 5 + 6 = 11
				bc.EmitPushConstant(dtrules.NewValueInteger(-5))
				bc.Emit(dtrules.OpAbs)
				bc.EmitPushConstant(dtrules.NewValueInteger(3))
				bc.EmitPushConstant(dtrules.NewValueInteger(2))
				bc.Emit(dtrules.OpMul)
				bc.Emit(dtrules.OpAdd)
			},
			Verify: func(ctx runtime.ExecutionContext) error { return expectInt(ctx, 11) },
		},
	}

	runBytecodeTests(t, tests)
}
