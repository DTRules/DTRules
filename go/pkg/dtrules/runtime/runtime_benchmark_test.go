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

package runtime_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/runtime"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/runtime/asmruntime"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/runtime/goruntime"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/runtime/nativeasm"
)

// BenchmarkResult holds benchmark data for JSON export
type BenchmarkResult struct {
	Runtime    string  `json:"runtime"`
	Category   string  `json:"category"`
	Operation  string  `json:"operation"`
	NsPerOp    float64 `json:"ns_per_op"`
	OpsPerSec  float64 `json:"ops_per_sec"`
	Iterations int     `json:"iterations"`
}

// getBenchmarkRuntimes returns all available runtimes for benchmarking
func getBenchmarkRuntimes(b *testing.B) []runtime.Runtime {
	runtimes := []runtime.Runtime{}

	// Go runtime (always available)
	goRT := goruntime.New()
	runtimes = append(runtimes, goRT)

	// Native ASM (Plan 9 assembly, always available on amd64)
	nativeRT := nativeasm.New()
	runtimes = append(runtimes, nativeRT)

	// ASM CGO (x86-64 NASM via CGO)
	asmRT, err := asmruntime.New()
	if err != nil {
		b.Logf("ASM CGO runtime not available: %v", err)
	} else {
		runtimes = append(runtimes, asmRT)
	}

	return runtimes
}

// =============================================================================
// ARITHMETIC BENCHMARKS
// =============================================================================

func BenchmarkArithmeticAdd(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.EmitPushConstant(dtrules.NewValueInteger(200))
			bc.Emit(dtrules.OpAdd)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkArithmeticSub(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(500))
			bc.EmitPushConstant(dtrules.NewValueInteger(200))
			bc.Emit(dtrules.OpSub)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkArithmeticMul(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(25))
			bc.EmitPushConstant(dtrules.NewValueInteger(40))
			bc.Emit(dtrules.OpMul)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkArithmeticDiv(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1000))
			bc.EmitPushConstant(dtrules.NewValueInteger(25))
			bc.Emit(dtrules.OpDiv)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkArithmeticComplex(b *testing.B) {
	// (100 + 50) * 2 - 10 / 5
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.EmitPushConstant(dtrules.NewValueInteger(50))
			bc.Emit(dtrules.OpAdd)
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.Emit(dtrules.OpMul)
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(5))
			bc.Emit(dtrules.OpDiv)
			bc.Emit(dtrules.OpSub)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

// =============================================================================
// COMPARISON BENCHMARKS
// =============================================================================

func BenchmarkComparisonLt(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(50))
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.Emit(dtrules.OpLt)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkComparisonEq(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpEq)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkComparisonChain(b *testing.B) {
	// 10 < 20 AND 20 < 30 AND 30 < 40
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.Emit(dtrules.OpLt)
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.EmitPushConstant(dtrules.NewValueInteger(30))
			bc.Emit(dtrules.OpLt)
			bc.Emit(dtrules.OpAnd)
			bc.EmitPushConstant(dtrules.NewValueInteger(30))
			bc.EmitPushConstant(dtrules.NewValueInteger(40))
			bc.Emit(dtrules.OpLt)
			bc.Emit(dtrules.OpAnd)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

// =============================================================================
// BOOLEAN BENCHMARKS
// =============================================================================

func BenchmarkBooleanAnd(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpAnd)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkBooleanOr(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpOr)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkBooleanNot(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpNot)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkBooleanComplex(b *testing.B) {
	// (true AND false) OR (NOT false)
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpAnd)
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpNot)
			bc.Emit(dtrules.OpOr)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

// =============================================================================
// STACK BENCHMARKS
// =============================================================================

func BenchmarkStackPushPop(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpPop)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkStackDup(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpDup)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkStackSwap(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1))
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.Emit(dtrules.OpSwap)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkStackRot(b *testing.B) {
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1))
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.EmitPushConstant(dtrules.NewValueInteger(3))
			bc.Emit(dtrules.OpRot)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

// =============================================================================
// COMPREHENSIVE BENCHMARKS
// =============================================================================

func BenchmarkMixedOperations(b *testing.B) {
	// Simulates a real rule condition: (age >= 18 AND income > 30000) OR waiver == true
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			// age >= 18
			bc.EmitPushConstant(dtrules.NewValueInteger(25)) // age
			bc.EmitPushConstant(dtrules.NewValueInteger(18))
			bc.Emit(dtrules.OpGe)
			// income > 30000
			bc.EmitPushConstant(dtrules.NewValueInteger(50000)) // income
			bc.EmitPushConstant(dtrules.NewValueInteger(30000))
			bc.Emit(dtrules.OpGt)
			// AND
			bc.Emit(dtrules.OpAnd)
			// waiver == true
			bc.Emit(dtrules.OpPushFalse) // waiver
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpEq)
			// OR
			bc.Emit(dtrules.OpOr)

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkHeavyArithmetic(b *testing.B) {
	// Many arithmetic operations: sum = ((a+b)*c - d/e + f) * g
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(10))  // a
			bc.EmitPushConstant(dtrules.NewValueInteger(20))  // b
			bc.Emit(dtrules.OpAdd)                            // a+b
			bc.EmitPushConstant(dtrules.NewValueInteger(3))   // c
			bc.Emit(dtrules.OpMul)                            // (a+b)*c
			bc.EmitPushConstant(dtrules.NewValueInteger(100)) // d
			bc.EmitPushConstant(dtrules.NewValueInteger(5))   // e
			bc.Emit(dtrules.OpDiv)                            // d/e
			bc.Emit(dtrules.OpSub)                            // (a+b)*c - d/e
			bc.EmitPushConstant(dtrules.NewValueInteger(7))   // f
			bc.Emit(dtrules.OpAdd)                            // ... + f
			bc.EmitPushConstant(dtrules.NewValueInteger(2))   // g
			bc.Emit(dtrules.OpMul)                            // ... * g

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkManyPushPop(b *testing.B) {
	// Push 10 values, then pop them all
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			for i := 0; i < 10; i++ {
				bc.EmitPushConstant(dtrules.NewValueInteger(int64(i)))
			}
			for i := 0; i < 10; i++ {
				bc.Emit(dtrules.OpPop)
			}

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

func BenchmarkLongExpression(b *testing.B) {
	// Sum of 20 numbers: 1 + 2 + 3 + ... + 20
	runtimes := getBenchmarkRuntimes(b)
	for _, rt := range runtimes {
		b.Run(rt.Name(), func(b *testing.B) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1))
			for i := 2; i <= 20; i++ {
				bc.EmitPushConstant(dtrules.NewValueInteger(int64(i)))
				bc.Emit(dtrules.OpAdd)
			}

			ctx, _ := rt.CreateContext()
			defer ctx.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
		})
	}
}

// =============================================================================
// BENCHMARK RUNNER FOR JSON EXPORT
// =============================================================================

// TestRunBenchmarksForExport runs all benchmarks and exports results to JSON
// Run with: go test -v -run TestRunBenchmarksForExport -count=1
func TestRunBenchmarksForExport(t *testing.T) {
	if os.Getenv("RUN_BENCHMARKS") != "1" {
		t.Skip("Set RUN_BENCHMARKS=1 to run benchmark export")
	}

	results := runAllBenchmarksForExport(t)

	// Export to JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal results: %v", err)
	}

	outputPath := os.Getenv("BENCHMARK_OUTPUT")
	if outputPath == "" {
		outputPath = "/tmp/benchmark_results.json"
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write results: %v", err)
	}

	fmt.Printf("Exported %d benchmark results to %s\n", len(results), outputPath)

	// Also print summary
	printBenchmarkSummary(results)
}

func runAllBenchmarksForExport(t *testing.T) []BenchmarkResult {
	var results []BenchmarkResult

	runtimes := []runtime.Runtime{goruntime.New(), nativeasm.New()}
	if asmRT, err := asmruntime.New(); err == nil {
		runtimes = append(runtimes, asmRT)
	}

	type benchCase struct {
		category  string
		operation string
		setup     func() *dtrules.BytecodeChunk
	}

	benchmarks := []benchCase{
		{"arithmetic", "add", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.EmitPushConstant(dtrules.NewValueInteger(200))
			bc.Emit(dtrules.OpAdd)
			return bc
		}},
		{"arithmetic", "sub", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(500))
			bc.EmitPushConstant(dtrules.NewValueInteger(200))
			bc.Emit(dtrules.OpSub)
			return bc
		}},
		{"arithmetic", "mul", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(25))
			bc.EmitPushConstant(dtrules.NewValueInteger(40))
			bc.Emit(dtrules.OpMul)
			return bc
		}},
		{"arithmetic", "div", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1000))
			bc.EmitPushConstant(dtrules.NewValueInteger(25))
			bc.Emit(dtrules.OpDiv)
			return bc
		}},
		{"arithmetic", "complex", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.EmitPushConstant(dtrules.NewValueInteger(50))
			bc.Emit(dtrules.OpAdd)
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.Emit(dtrules.OpMul)
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(5))
			bc.Emit(dtrules.OpDiv)
			bc.Emit(dtrules.OpSub)
			return bc
		}},
		{"comparison", "lt", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(50))
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.Emit(dtrules.OpLt)
			return bc
		}},
		{"comparison", "eq", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpEq)
			return bc
		}},
		{"comparison", "chain", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.Emit(dtrules.OpLt)
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.EmitPushConstant(dtrules.NewValueInteger(30))
			bc.Emit(dtrules.OpLt)
			bc.Emit(dtrules.OpAnd)
			return bc
		}},
		{"boolean", "and", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpAnd)
			return bc
		}},
		{"boolean", "or", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpOr)
			return bc
		}},
		{"boolean", "not", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpNot)
			return bc
		}},
		{"boolean", "complex", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.Emit(dtrules.OpPushTrue)
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpAnd)
			bc.Emit(dtrules.OpPushFalse)
			bc.Emit(dtrules.OpNot)
			bc.Emit(dtrules.OpOr)
			return bc
		}},
		{"stack", "push_pop", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpPop)
			return bc
		}},
		{"stack", "dup", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(42))
			bc.Emit(dtrules.OpDup)
			return bc
		}},
		{"stack", "swap", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1))
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.Emit(dtrules.OpSwap)
			return bc
		}},
		{"stack", "rot", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1))
			bc.EmitPushConstant(dtrules.NewValueInteger(2))
			bc.EmitPushConstant(dtrules.NewValueInteger(3))
			bc.Emit(dtrules.OpRot)
			return bc
		}},
		{"complex", "mixed_ops", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(25))
			bc.EmitPushConstant(dtrules.NewValueInteger(18))
			bc.Emit(dtrules.OpGe)
			bc.EmitPushConstant(dtrules.NewValueInteger(50000))
			bc.EmitPushConstant(dtrules.NewValueInteger(30000))
			bc.Emit(dtrules.OpGt)
			bc.Emit(dtrules.OpAnd)
			return bc
		}},
		{"complex", "heavy_arithmetic", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(10))
			bc.EmitPushConstant(dtrules.NewValueInteger(20))
			bc.Emit(dtrules.OpAdd)
			bc.EmitPushConstant(dtrules.NewValueInteger(3))
			bc.Emit(dtrules.OpMul)
			bc.EmitPushConstant(dtrules.NewValueInteger(100))
			bc.EmitPushConstant(dtrules.NewValueInteger(5))
			bc.Emit(dtrules.OpDiv)
			bc.Emit(dtrules.OpSub)
			return bc
		}},
		{"complex", "long_expression", func() *dtrules.BytecodeChunk {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(1))
			for i := 2; i <= 20; i++ {
				bc.EmitPushConstant(dtrules.NewValueInteger(int64(i)))
				bc.Emit(dtrules.OpAdd)
			}
			return bc
		}},
	}

	const iterations = 100000

	for _, rt := range runtimes {
		ctx, err := rt.CreateContext()
		if err != nil {
			t.Logf("Failed to create context for %s: %v", rt.Name(), err)
			continue
		}

		for _, bench := range benchmarks {
			bc := bench.setup()

			// Warm up
			for i := 0; i < 1000; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}

			// Measure
			start := time.Now()
			for i := 0; i < iterations; i++ {
				ctx.Reset()
				ctx.ExecuteBytecode(bc)
			}
			elapsed := time.Since(start)

			nsPerOp := float64(elapsed.Nanoseconds()) / float64(iterations)
			opsPerSec := 1e9 / nsPerOp

			results = append(results, BenchmarkResult{
				Runtime:    rt.Name(),
				Category:   bench.category,
				Operation:  bench.operation,
				NsPerOp:    nsPerOp,
				OpsPerSec:  opsPerSec,
				Iterations: iterations,
			})
		}

		ctx.Close()
	}

	return results
}

func printBenchmarkSummary(results []BenchmarkResult) {
	fmt.Println("\n=== Benchmark Summary ===")
	fmt.Println()

	// Group by category
	categories := make(map[string][]BenchmarkResult)
	for _, r := range results {
		categories[r.Category] = append(categories[r.Category], r)
	}

	for cat, benchmarks := range categories {
		fmt.Printf("Category: %s\n", cat)
		fmt.Printf("%-20s", "Operation")

		// Get unique runtimes
		runtimeMap := make(map[string]bool)
		for _, b := range benchmarks {
			runtimeMap[b.Runtime] = true
		}
		runtimes := make([]string, 0, len(runtimeMap))
		for rt := range runtimeMap {
			runtimes = append(runtimes, rt)
		}

		for _, rt := range runtimes {
			fmt.Printf("%15s", rt)
		}
		fmt.Println()
		fmt.Println(string(make([]byte, 20+15*len(runtimes))))

		// Group by operation
		ops := make(map[string]map[string]float64)
		for _, b := range benchmarks {
			if ops[b.Operation] == nil {
				ops[b.Operation] = make(map[string]float64)
			}
			ops[b.Operation][b.Runtime] = b.NsPerOp
		}

		for op, rts := range ops {
			fmt.Printf("%-20s", op)
			for _, rt := range runtimes {
				if ns, ok := rts[rt]; ok {
					fmt.Printf("%12.1f ns", ns)
				} else {
					fmt.Printf("%15s", "-")
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
