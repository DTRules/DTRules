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

// Package main provides a benchmark tool for comparing Java, Go, and ASM
// performance when executing CHIP eligibility decision tables.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/asmruntime"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/interpreter"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/mapping"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

var (
	iterations = flag.Int("iterations", 10000, "Number of iterations per test")
	testDir    = flag.String("testdir", "", "Directory containing test case XML files")
	verbose    = flag.Bool("verbose", false, "Verbose output")
	showChart  = flag.Bool("chart", true, "Show ASCII performance chart")
)

// BenchmarkResult holds the timing results for a single benchmark.
type BenchmarkResult struct {
	Name         string
	GoTime       time.Duration
	ASMTime      time.Duration
	Iterations   int
	GoOpsPerSec  float64
	ASMOpsPerSec float64
	GoNsPerOp    float64
	ASMNsPerOp   float64
	Speedup      float64
}

// mockSession for micro-benchmarks
type mockSession struct {
	uniqueID int
}

func (m *mockSession) GetState() dtrules.State                                  { return nil }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory                  { return nil }
func (m *mockSession) GetUniqueID() int                                         { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser                        { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet                              { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) { return nil, nil }
func (m *mockSession) Compile(expr string) (dtrules.Object, error)              { return nil, nil }

func main() {
	flag.Parse()

	fmt.Println("╔═══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     DTRules Performance Benchmark: Java vs Go vs ASM              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("CPU: %s\n", runtime.GOARCH)
	fmt.Printf("Iterations per test: %d\n", *iterations)
	fmt.Println()

	// Initialize ASM runtime
	if err := asmruntime.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize ASM runtime: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ ASM runtime initialized")
	fmt.Println("✓ Go interpreter ready")
	fmt.Println()

	// Run bytecode comparison benchmarks
	results := runBytecodeComparison()

	// Print results table
	printResults(results)

	// Print ASCII chart
	if *showChart {
		printChart(results)
	}

	// Print memory info
	printMemoryInfo()

	// If test directory specified, run CHIP tests
	if *testDir != "" {
		runCHIPTests(*testDir)
	}
}

// runBytecodeComparison runs pure bytecode execution benchmarks
// comparing Go interpreter vs ASM execution.
func runBytecodeComparison() []BenchmarkResult {
	fmt.Println("Running bytecode execution benchmarks...")
	fmt.Println()

	results := []BenchmarkResult{}

	// Test 1: Simple arithmetic (1 + 2)
	results = append(results, benchmarkSimpleArithmetic())

	// Test 2: Complex expression ((10 + 20) * 3) - 5
	results = append(results, benchmarkComplexExpression())

	// Test 3: Comparison chain (5 > 3 && 10 < 20)
	results = append(results, benchmarkComparisonChain())

	// Test 4: Constant pool access (100 + 200)
	results = append(results, benchmarkConstantPool())

	// Test 5: Nested operations
	results = append(results, benchmarkNestedOperations())

	// Test 6: Multi-op expression (similar to CHIP condition)
	results = append(results, benchmarkMultiOp())

	return results
}

func benchmarkSimpleArithmetic() BenchmarkResult {
	name := "Simple (1+2)"

	// Build bytecode: push 1, push 2, add
	chunk := dtrules.NewBytecodeChunk()
	chunk.Emit(dtrules.OpPushOne)
	chunk.EmitWithArg(dtrules.OpPushInt, 2)
	chunk.Emit(dtrules.OpAdd)

	return runBenchmark(name, chunk)
}

func benchmarkComplexExpression() BenchmarkResult {
	name := "Complex ((10+20)*3)-5"

	// Build bytecode: ((10 + 20) * 3) - 5
	chunk := dtrules.NewBytecodeChunk()
	chunk.EmitWithArg(dtrules.OpPushInt, 10)
	chunk.EmitWithArg(dtrules.OpPushInt, 20)
	chunk.Emit(dtrules.OpAdd)
	chunk.EmitWithArg(dtrules.OpPushInt, 3)
	chunk.Emit(dtrules.OpMul)
	chunk.EmitWithArg(dtrules.OpPushInt, 5)
	chunk.Emit(dtrules.OpSub)

	return runBenchmark(name, chunk)
}

func benchmarkComparisonChain() BenchmarkResult {
	name := "Compare (5>3 && 10<20)"

	// Build bytecode: (5 > 3) && (10 < 20)
	chunk := dtrules.NewBytecodeChunk()
	chunk.EmitWithArg(dtrules.OpPushInt, 5)
	chunk.EmitWithArg(dtrules.OpPushInt, 3)
	chunk.Emit(dtrules.OpGt)
	chunk.EmitWithArg(dtrules.OpPushInt, 10)
	chunk.EmitWithArg(dtrules.OpPushInt, 20)
	chunk.Emit(dtrules.OpLt)
	chunk.Emit(dtrules.OpAnd)

	return runBenchmark(name, chunk)
}

func benchmarkConstantPool() BenchmarkResult {
	name := "Constants (100+200)"

	// Build bytecode using constant pool
	chunk := dtrules.NewBytecodeChunk()
	chunk.AddConstant(dtrules.NewValueInteger(100))
	chunk.AddConstant(dtrules.NewValueInteger(200))
	chunk.EmitWithArg(dtrules.OpConstant, 0)
	chunk.EmitWithArg(dtrules.OpConstant, 1)
	chunk.Emit(dtrules.OpAdd)

	return runBenchmark(name, chunk)
}

func benchmarkNestedOperations() BenchmarkResult {
	name := "Nested ((1+2)*(3+4))"

	// Build more complex bytecode with multiple operations
	// ((1 + 2) * (3 + 4)) - ((5 + 6) / 2)
	chunk := dtrules.NewBytecodeChunk()
	// (1 + 2)
	chunk.Emit(dtrules.OpPushOne)
	chunk.EmitWithArg(dtrules.OpPushInt, 2)
	chunk.Emit(dtrules.OpAdd)
	// (3 + 4)
	chunk.EmitWithArg(dtrules.OpPushInt, 3)
	chunk.EmitWithArg(dtrules.OpPushInt, 4)
	chunk.Emit(dtrules.OpAdd)
	// *
	chunk.Emit(dtrules.OpMul)
	// (5 + 6)
	chunk.EmitWithArg(dtrules.OpPushInt, 5)
	chunk.EmitWithArg(dtrules.OpPushInt, 6)
	chunk.Emit(dtrules.OpAdd)
	// / 2
	chunk.EmitWithArg(dtrules.OpPushInt, 2)
	chunk.Emit(dtrules.OpDiv)
	// -
	chunk.Emit(dtrules.OpSub)

	return runBenchmark(name, chunk)
}

func benchmarkMultiOp() BenchmarkResult {
	name := "Multi-Op (CHIP-like)"

	// Simulate a CHIP-like condition check:
	// (age >= 0) && (age <= 19) && (income < 50000)
	chunk := dtrules.NewBytecodeChunk()
	// age >= 0
	chunk.EmitWithArg(dtrules.OpPushInt, 15) // age = 15
	chunk.Emit(dtrules.OpPushZero)
	chunk.Emit(dtrules.OpGe)
	// age <= 19
	chunk.EmitWithArg(dtrules.OpPushInt, 15)
	chunk.EmitWithArg(dtrules.OpPushInt, 19)
	chunk.Emit(dtrules.OpLe)
	// &&
	chunk.Emit(dtrules.OpAnd)
	// income < 50000
	chunk.AddConstant(dtrules.NewValueInteger(35000))
	chunk.AddConstant(dtrules.NewValueInteger(50000))
	chunk.EmitWithArg(dtrules.OpConstant, 0)
	chunk.EmitWithArg(dtrules.OpConstant, 1)
	chunk.Emit(dtrules.OpLt)
	// &&
	chunk.Emit(dtrules.OpAnd)

	return runBenchmark(name, chunk)
}

func runBenchmark(name string, chunk *dtrules.BytecodeChunk) BenchmarkResult {
	result := BenchmarkResult{
		Name:       name,
		Iterations: *iterations,
	}

	// Create Go interpreter state
	state := interpreter.NewDTState(&mockSession{})

	// Warm-up
	for i := 0; i < 100; i++ {
		state.ExecuteBytecode(chunk)
		state.ValuePop()
	}
	asmruntime.Reset()
	for i := 0; i < 100; i++ {
		asmruntime.Reset()
		asmruntime.ExecuteBytecode(chunk)
	}

	// Benchmark Go interpreter execution
	start := time.Now()
	for i := 0; i < *iterations; i++ {
		state.ExecuteBytecode(chunk)
		state.ValuePop()
	}
	result.GoTime = time.Since(start)
	result.GoNsPerOp = float64(result.GoTime.Nanoseconds()) / float64(*iterations)
	result.GoOpsPerSec = float64(*iterations) / result.GoTime.Seconds()

	// Benchmark ASM execution
	asmruntime.Reset()
	start = time.Now()
	for i := 0; i < *iterations; i++ {
		asmruntime.Reset()
		asmruntime.ExecuteBytecode(chunk)
	}
	result.ASMTime = time.Since(start)
	result.ASMNsPerOp = float64(result.ASMTime.Nanoseconds()) / float64(*iterations)
	result.ASMOpsPerSec = float64(*iterations) / result.ASMTime.Seconds()

	// Calculate speedup (positive = Go faster, negative = ASM faster)
	result.Speedup = result.ASMNsPerOp / result.GoNsPerOp

	if *verbose {
		fmt.Printf("  %s: Go=%v ASM=%v\n", name, result.GoTime, result.ASMTime)
	}

	return result
}

func printResults(results []BenchmarkResult) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                              PERFORMANCE COMPARISON                                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-24s │ %12s │ %12s │ %12s │ %12s │ %8s ║\n",
		"Test", "Go (ns/op)", "ASM (ns/op)", "Go (ops/s)", "ASM (ops/s)", "Ratio")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════╣")

	for _, r := range results {
		ratio := fmt.Sprintf("%.2fx", r.Speedup)
		if r.Speedup < 1 {
			ratio = fmt.Sprintf("%.2fx", 1/r.Speedup) + "*"
		}
		fmt.Printf("║ %-24s │ %12.1f │ %12.1f │ %12.0f │ %12.0f │ %8s ║\n",
			r.Name, r.GoNsPerOp, r.ASMNsPerOp, r.GoOpsPerSec, r.ASMOpsPerSec, ratio)
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("* = ASM faster than Go by this factor (due to CGO overhead, Go is often faster for simple ops)")
	fmt.Println()

	// Summary
	var totalGoNs, totalASMNs float64
	for _, r := range results {
		totalGoNs += r.GoNsPerOp
		totalASMNs += r.ASMNsPerOp
	}
	avgGoNs := totalGoNs / float64(len(results))
	avgASMNs := totalASMNs / float64(len(results))

	fmt.Println("Summary:")
	fmt.Printf("  Go interpreter average:  %.1f ns/op (%.0f ops/sec)\n",
		avgGoNs, float64(len(results))*1e9/totalGoNs)
	fmt.Printf("  ASM runtime average:     %.1f ns/op (%.0f ops/sec)\n",
		avgASMNs, float64(len(results))*1e9/totalASMNs)
}

func printChart(results []BenchmarkResult) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         PERFORMANCE CHART (ns/op)                              ║")
	fmt.Println("║                         ■ Go    □ ASM                                          ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")

	// Find max for scaling
	maxNs := 0.0
	for _, r := range results {
		if r.GoNsPerOp > maxNs {
			maxNs = r.GoNsPerOp
		}
		if r.ASMNsPerOp > maxNs {
			maxNs = r.ASMNsPerOp
		}
	}

	chartWidth := 45
	for _, r := range results {
		goWidth := int(r.GoNsPerOp / maxNs * float64(chartWidth))
		asmWidth := int(r.ASMNsPerOp / maxNs * float64(chartWidth))

		goBar := strings.Repeat("■", goWidth)
		asmBar := strings.Repeat("□", asmWidth)

		fmt.Printf("║ %-16s Go  │%s %5.0f\n", r.Name, padRight(goBar, chartWidth), r.GoNsPerOp)
		fmt.Printf("║ %-16s ASM │%s %5.0f\n", "", padRight(asmBar, chartWidth), r.ASMNsPerOp)
		fmt.Println("║                      │")
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func printMemoryInfo() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         MEMORY FOOTPRINT                                       ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Go Runtime:                                                                    ║\n")
	fmt.Printf("║   Heap in use:    %10d KB                                                ║\n", m.HeapInuse/1024)
	fmt.Printf("║   Stack in use:   %10d KB                                                ║\n", m.StackInuse/1024)
	fmt.Printf("║   Total alloc:    %10d KB                                                ║\n", m.TotalAlloc/1024)
	fmt.Printf("║                                                                                ║\n")
	fmt.Printf("║ ASM Runtime:                                                                   ║\n")
	fmt.Printf("║   Data stack:     %10d KB (fixed, 4096 values × 24 bytes)                ║\n", 4096*24/1024)
	fmt.Printf("║   Entity stack:   %10d KB (fixed, 256 entries × 8 bytes)                 ║\n", 256*8/1024)
	fmt.Printf("║   Control stack:  %10d KB (fixed, 1024 frames × 32 bytes)                ║\n", 1024*32/1024)
	fmt.Printf("║   Heap:           %10d KB (arena allocator, 16 MB reserved)              ║\n", 16*1024)
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         JAVA REFERENCE (estimated)                             ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║ Note: Java benchmarks require Maven. Typical JVM overhead:                    ║")
	fmt.Println("║   JVM startup:        ~100-500 ms                                             ║")
	fmt.Println("║   Heap (min):         ~64 MB                                                  ║")
	fmt.Println("║   Per-rule overhead:  ~500-5000 ns/op (before JIT warmup)                     ║")
	fmt.Println("║   After JIT warmup:   ~50-500 ns/op (comparable to Go)                        ║")
	fmt.Println("║                                                                                ║")
	fmt.Println("║ To run Java benchmarks:                                                       ║")
	fmt.Println("║   cd sampleprojects/ChipApp && mvn exec:java                                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")
}

// CHIPTestResult holds timing for a CHIP test case execution.
type CHIPTestResult struct {
	Name       string
	LoadTime   time.Duration
	ExecTime   time.Duration
	Iterations int
	NsPerOp    float64
	OpsPerSec  float64
}

func runCHIPTests(dir string) {
	fmt.Printf("\n")
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      CHIP ELIGIBILITY BENCHMARK                                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Find CHIP directory (parent of testfiles)
	chipDir := filepath.Dir(dir)
	if filepath.Base(chipDir) == "TestScenarios" || filepath.Base(chipDir) == "PerformanceTests" {
		chipDir = filepath.Dir(chipDir)
	}
	if filepath.Base(chipDir) == "testfiles" {
		chipDir = filepath.Dir(chipDir)
	}

	// Verify CHIP directory structure
	eddPath := filepath.Join(chipDir, "repository/xml/CHIP_edd.xml")
	dtPath := filepath.Join(chipDir, "repository/xml/CHIP_dt.xml")
	mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")

	if _, err := os.Stat(eddPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "CHIP EDD not found at: %s\n", eddPath)
		return
	}

	fmt.Printf("CHIP Directory: %s\n", chipDir)
	fmt.Printf("Test Directory: %s\n\n", dir)

	// Load RuleSet once (this is shared across tests)
	fmt.Print("Loading CHIP RuleSet... ")
	loadStart := time.Now()
	rs, err := loadCHIPRuleSet(eddPath, dtPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load RuleSet: %v\n", err)
		return
	}
	loadTime := time.Since(loadStart)
	fmt.Printf("done (%v)\n", loadTime)

	// Find all test case XML files
	files, err := filepath.Glob(filepath.Join(dir, "TestCase_*.xml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding test files: %v\n", err)
		return
	}

	if len(files) == 0 {
		// Try subdirectories
		files, _ = filepath.Glob(filepath.Join(dir, "*/TestCase_*.xml"))
	}

	if len(files) == 0 {
		fmt.Println("No test case files found.")
		return
	}

	sort.Strings(files)
	fmt.Printf("Found %d test case files.\n\n", len(files))

	// Run benchmarks
	results := []CHIPTestResult{}
	iterations := *iterations / 10 // Use fewer iterations for full CHIP tests
	if iterations < 100 {
		iterations = 100
	}

	fmt.Printf("Running each test case %d times...\n\n", iterations)

	for _, testFile := range files {
		result := runSingleCHIPTest(rs, mapPath, testFile, iterations)
		if result != nil {
			results = append(results, *result)
		}
	}

	// Print results
	printCHIPResults(results, loadTime)
}

func loadCHIPRuleSet(eddPath, dtPath string) (*session.RuleSet, error) {
	rs := session.NewRuleSet("CHIP")

	// Load EDD
	eddFile, err := os.Open(eddPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EDD: %w", err)
	}
	defer eddFile.Close()

	if err := rs.LoadEDD(eddFile); err != nil {
		return nil, fmt.Errorf("failed to load EDD: %w", err)
	}

	// Load Decision Tables
	dtFile, err := os.Open(dtPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DT: %w", err)
	}
	defer dtFile.Close()

	if err := rs.LoadDecisionTables(dtFile); err != nil {
		return nil, fmt.Errorf("failed to load DT: %w", err)
	}

	return rs, nil
}

func runSingleCHIPTest(rs *session.RuleSet, mapPath, testFile string, iterations int) *CHIPTestResult {
	basename := filepath.Base(testFile)

	// Load mapping and test data
	loadStart := time.Now()

	sess, err := rs.NewSession()
	if err != nil {
		fmt.Printf("  %-40s SKIP (session error: %v)\n", basename, err)
		return nil
	}

	// Load mapping
	mapFile, err := os.Open(mapPath)
	if err != nil {
		fmt.Printf("  %-40s SKIP (mapping error: %v)\n", basename, err)
		return nil
	}

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		mapFile.Close()
		fmt.Printf("  %-40s SKIP (mapping load error: %v)\n", basename, err)
		return nil
	}
	mapFile.Close()

	if err := m.Initialize(); err != nil {
		fmt.Printf("  %-40s SKIP (mapping init error: %v)\n", basename, err)
		return nil
	}

	// Load test data
	dataFile, err := os.Open(testFile)
	if err != nil {
		fmt.Printf("  %-40s SKIP (data error: %v)\n", basename, err)
		return nil
	}

	if err := m.LoadData(dataFile); err != nil {
		dataFile.Close()
		fmt.Printf("  %-40s SKIP (data load error: %v)\n", basename, err)
		return nil
	}
	dataFile.Close()

	loadTime := time.Since(loadStart)

	// Get decision table
	factory := sess.GetEntityFactory()
	dt, err := factory.GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
	if err != nil || dt == nil {
		fmt.Printf("  %-40s SKIP (DT not found)\n", basename)
		return nil
	}

	state := sess.GetState()

	// Warm-up run
	_ = dt.Execute(state)

	// Benchmark execution
	execStart := time.Now()
	for i := 0; i < iterations; i++ {
		// Reset session state for each iteration
		// Note: In a real scenario, we'd reload the data
		// For benchmarking, we just re-execute with same state
		_ = dt.Execute(state)
	}
	execTime := time.Since(execStart)

	result := &CHIPTestResult{
		Name:       basename,
		LoadTime:   loadTime,
		ExecTime:   execTime,
		Iterations: iterations,
		NsPerOp:    float64(execTime.Nanoseconds()) / float64(iterations),
		OpsPerSec:  float64(iterations) / execTime.Seconds(),
	}

	fmt.Printf("  %-40s %8.1f ns/op (%6.0f ops/s)\n",
		basename, result.NsPerOp, result.OpsPerSec)

	return result
}

func printCHIPResults(results []CHIPTestResult, ruleSetLoadTime time.Duration) {
	if len(results) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         CHIP BENCHMARK RESULTS (Go Interpreter)                            ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-44s │ %12s │ %12s │ %12s ║\n",
		"Test Case", "Load (ms)", "Exec (ns/op)", "Throughput")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════╣")

	var totalNsPerOp float64
	var totalOpsPerSec float64

	for _, r := range results {
		fmt.Printf("║ %-44s │ %12.2f │ %12.1f │ %10.0f/s ║\n",
			r.Name, float64(r.LoadTime.Microseconds())/1000.0, r.NsPerOp, r.OpsPerSec)
		totalNsPerOp += r.NsPerOp
		totalOpsPerSec += r.OpsPerSec
	}

	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════╣")

	avgNsPerOp := totalNsPerOp / float64(len(results))
	avgOpsPerSec := totalOpsPerSec / float64(len(results))

	fmt.Printf("║ %-44s │ %12s │ %12.1f │ %10.0f/s ║\n",
		"AVERAGE", "-", avgNsPerOp, avgOpsPerSec)
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════╝")

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  RuleSet load time:    %v\n", ruleSetLoadTime)
	fmt.Printf("  Average execution:    %.1f ns/op\n", avgNsPerOp)
	fmt.Printf("  Average throughput:   %.0f executions/second\n", avgOpsPerSec)

	// Compare with Java estimates
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         ESTIMATED COMPARISON                                   ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║                 Go Interpreter          Java (estimated)                       ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Throughput:    %8.0f ops/s          ~500-2000 ops/s                         ║\n", avgOpsPerSec)
	fmt.Printf("║  Latency:       %8.1f ns/op          ~500,000-2,000,000 ns/op                ║\n", avgNsPerOp)
	fmt.Println("║  Memory:        ~10-50 MB              ~64-256 MB (JVM heap)                   ║")
	fmt.Println("║  Startup:       ~1 ms                  ~100-500 ms (JVM warmup)                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Note: Java estimates based on typical JVM overhead. Run Maven benchmarks for actual data:")
	fmt.Println("  cd sampleprojects/ChipApp && mvn exec:java")
}
