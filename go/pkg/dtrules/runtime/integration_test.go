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

//go:build integration
// +build integration

package runtime_test

import (
	"fmt"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/asmruntime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/goruntime"
)

// TestCase represents a bytecode test case
type TestCase struct {
	name     string
	setup    func(bc *dtrules.BytecodeChunk)
	expected dtrules.Value
}

var testCases = []TestCase{
	{
		name: "simple addition",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.Emit(dtrules.OpAdd)
		},
		expected: dtrules.NewValueInteger(30),
	},
	{
		name: "subtraction",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.EmitPushConstant(dtrules.NewValueInteger(30))
			bc.Emit(dtrules.OpSub)
		},
		expected: dtrules.NewValueInteger(70),
	},
	{
		name: "multiplication",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(7))
			bc.EmitPushConstant(dtrules.NewValueInteger(6))
			bc.Emit(dtrules.OpMul)
		},
		expected: dtrules.NewValueInteger(42),
	},
	{
		name: "boolean and",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpAnd)
		},
		expected: dtrules.NewValueBoolean(true),
	},
	{
		name: "boolean or",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpOr)
		},
		expected: dtrules.NewValueBoolean(true),
	},
	{
		name: "boolean not",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpNot)
		},
		expected: dtrules.NewValueBoolean(true),
	},
	{
		name: "comparison greater than",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(50))
			bc.EmitPushConstant(dtrules.NewValueInteger(25))
			bc.Emit(dtrules.OpGt)
		},
		expected: dtrules.NewValueBoolean(true),
	},
	{
		name: "comparison less than",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.Emit(dtrules.OpLt)
		},
		expected: dtrules.NewValueBoolean(true),
	},
	{
		name: "comparison equal",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpEq)
		},
		expected: dtrules.NewValueBoolean(true),
	},
	{
		name: "negation",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpNeg)
		},
		expected: dtrules.NewValueInteger(-42),
	},
	{
		name: "increment",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(99))
			bc.Emit(dtrules.OpInc)
		},
		expected: dtrules.NewValueInteger(100),
	},
	{
		name: "decrement",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.Emit(dtrules.OpDec)
		},
		expected: dtrules.NewValueInteger(99),
	},
	{
		name: "dup",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(5))
			bc.Emit(dtrules.OpDup)
			bc.Emit(dtrules.OpAdd)
		},
		expected: dtrules.NewValueInteger(10),
	},
	{
		name: "swap",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(3))
			bc.Emit(dtrules.OpSwap)
			bc.Emit(dtrules.OpSub) // 3 - 10 = -7
		},
		expected: dtrules.NewValueInteger(-7),
	},
	{
		name: "complex expression",
		setup: func(bc *dtrules.BytecodeChunk) {
			// (10 + 20) * 2 = 60
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.Emit(dtrules.OpAdd)
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.Emit(dtrules.OpMul)
		},
		expected: dtrules.NewValueInteger(60),
	},
	{
		name: "push constants",
		setup: func(bc *dtrules.BytecodeChunk) {
			bc.Emit(dtrules.OpPushZero)
			bc.Emit(dtrules.OpPushOne)
			bc.Emit(dtrules.OpAdd)
		},
		expected: dtrules.NewValueInteger(1),
	},
}

func runTestOnRuntime(t *testing.T, rt runtime.Runtime, tc TestCase) {
	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("[%s] CreateContext failed: %v", rt.Name(), err)
	}
	defer ctx.Close()

	bc := dtrules.NewBytecodeChunk()
	tc.setup(bc)

	if err := ctx.ExecuteBytecode(bc); err != nil {
		t.Fatalf("[%s] %s: ExecuteBytecode failed: %v", rt.Name(), tc.name, err)
	}

	result, err := ctx.Pop()
	if err != nil {
		t.Fatalf("[%s] %s: Pop failed: %v", rt.Name(), tc.name, err)
	}

	// Compare results
	if !valuesEqual(result, tc.expected) {
		t.Errorf("[%s] %s: expected %v, got %v", rt.Name(), tc.name, tc.expected, result)
	}
}

func valuesEqual(a, b dtrules.Value) bool {
	if a.Tag() != b.Tag() {
		// Allow integer/double comparison for numeric results
		if a.IsNumeric() && b.IsNumeric() {
			return a.AsDouble() == b.AsDouble()
		}
		return false
	}

	switch a.Tag() {
	case dtrules.VTagNull:
		return true
	case dtrules.VTagInteger:
		return a.AsInteger() == b.AsInteger()
	case dtrules.VTagDouble:
		return a.AsDouble() == b.AsDouble()
	case dtrules.VTagBoolean:
		return a.AsBoolean() == b.AsBoolean()
	case dtrules.VTagString:
		return a.AsString() == b.AsString()
	default:
		return false
	}
}

func TestGoRuntime(t *testing.T) {
	rt := goruntime.New()
	defer rt.Close()

	t.Logf("Testing Go Runtime: %s v%s", rt.Name(), rt.Version())
	caps := rt.Capabilities()
	t.Logf("  ConcurrentContexts: %v", caps.ConcurrentContexts)
	t.Logf("  MaxStackDepth: %d", caps.MaxStackDepth)
	t.Logf("  SupportsAllOperators: %v", caps.SupportsAllOperators)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestOnRuntime(t, rt, tc)
		})
	}
}

func TestASMRuntime(t *testing.T) {
	rt, err := asmruntime.New()
	if err != nil {
		t.Skipf("ASM runtime not available: %v", err)
	}
	defer rt.Close()

	t.Logf("Testing ASM Runtime: %s v%s", rt.Name(), rt.Version())
	caps := rt.Capabilities()
	t.Logf("  ConcurrentContexts: %v", caps.ConcurrentContexts)
	t.Logf("  MaxStackDepth: %d", caps.MaxStackDepth)
	t.Logf("  SupportsAllOperators: %v", caps.SupportsAllOperators)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestOnRuntime(t, rt, tc)
		})
	}
}

func TestBothRuntimesProduceSameResults(t *testing.T) {
	goRT := goruntime.New()
	defer goRT.Close()

	asmRT, err := asmruntime.New()
	if err != nil {
		t.Skipf("ASM runtime not available: %v", err)
	}
	defer asmRT.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run on Go runtime
			goCtx, _ := goRT.CreateContext()
			defer goCtx.Close()

			bc := dtrules.NewBytecodeChunk()
			tc.setup(bc)

			if err := goCtx.ExecuteBytecode(bc); err != nil {
				t.Fatalf("Go runtime failed: %v", err)
			}
			goResult, _ := goCtx.Pop()

			// Run on ASM runtime
			asmCtx, _ := asmRT.CreateContext()
			defer asmCtx.Close()

			bc2 := dtrules.NewBytecodeChunk()
			tc.setup(bc2)

			if err := asmCtx.ExecuteBytecode(bc2); err != nil {
				t.Fatalf("ASM runtime failed: %v", err)
			}
			asmResult, _ := asmCtx.Pop()

			// Compare results
			if !valuesEqual(goResult, asmResult) {
				t.Errorf("Results differ: Go=%v, ASM=%v", goResult, asmResult)
			}
		})
	}
}

func TestRuntimeInterface(t *testing.T) {
	// Test that both runtimes implement the Runtime interface
	var runtimes []runtime.Runtime

	goRT := goruntime.New()
	runtimes = append(runtimes, goRT)
	defer goRT.Close()

	asmRT, err := asmruntime.New()
	if err == nil {
		runtimes = append(runtimes, asmRT)
		defer asmRT.Close()
	}

	for _, rt := range runtimes {
		t.Run(rt.Name(), func(t *testing.T) {
			// Test Name()
			if rt.Name() == "" {
				t.Error("Name() returned empty string")
			}

			// Test Version()
			if rt.Version() == "" {
				t.Error("Version() returned empty string")
			}

			// Test Capabilities()
			caps := rt.Capabilities()
			if caps.MaxStackDepth <= 0 {
				t.Error("MaxStackDepth should be positive")
			}

			// Test CreateContext()
			ctx, err := rt.CreateContext()
			if err != nil {
				t.Fatalf("CreateContext failed: %v", err)
			}

			// Test context operations
			if ctx.StackDepth() != 0 {
				t.Error("New context should have empty stack")
			}

			if err := ctx.Push(dtrules.NewValueInteger(42)); err != nil {
				t.Errorf("Push failed: %v", err)
			}

			if ctx.StackDepth() != 1 {
				t.Error("Stack depth should be 1 after push")
			}

			v, err := ctx.Pop()
			if err != nil {
				t.Errorf("Pop failed: %v", err)
			}
			if v.AsInteger() != 42 {
				t.Errorf("Expected 42, got %d", v.AsInteger())
			}

			ctx.Close()
		})
	}
}

func BenchmarkGoRuntime(b *testing.B) {
	rt := goruntime.New()
	defer rt.Close()

	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(20))
	bc.Emit(dtrules.OpAdd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExecuteBytecode(bc)
		ctx.Pop()
	}
}

func BenchmarkASMRuntime(b *testing.B) {
	rt, err := asmruntime.New()
	if err != nil {
		b.Skipf("ASM runtime not available: %v", err)
	}
	defer rt.Close()

	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(20))
	bc.Emit(dtrules.OpAdd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExecuteBytecode(bc)
		ctx.Pop()
	}
}

func ExampleGoRuntime() {
	// Create a Go runtime
	rt := goruntime.New()
	defer rt.Close()

	// Create an execution context
	ctx, err := rt.CreateContext()
	if err != nil {
		panic(err)
	}
	defer ctx.Close()

	// Create bytecode: 10 + 20
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(20))
	bc.Emit(dtrules.OpAdd)

	// Execute
	if err := ctx.ExecuteBytecode(bc); err != nil {
		panic(err)
	}

	// Get result
	result, _ := ctx.Pop()
	fmt.Printf("10 + 20 = %d\n", result.AsInteger())
	// Output: 10 + 20 = 30
}
