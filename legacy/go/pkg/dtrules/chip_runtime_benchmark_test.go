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

package dtrules_test

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/asmruntime"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/runtime/nativeasm"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestCHIPRuntimeBenchmark runs all 13 CHIP test cases across all 4 runtimes,
// reporting memory footprint and ops/sec per test per runtime.
func TestCHIPRuntimeBenchmark(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	// Collect all test case files
	var testFiles []string
	scenarioFiles, _ := filepath.Glob(filepath.Join(chipDir, "testfiles/TestScenarios/*.xml"))
	testFiles = append(testFiles, scenarioFiles...)
	perfFiles, _ := filepath.Glob(filepath.Join(chipDir, "testfiles/PerformanceTests/*.xml"))
	testFiles = append(testFiles, perfFiles...)
	sort.Strings(testFiles)

	if len(testFiles) == 0 {
		t.Skip("No test case files found")
	}

	// Load rule set once
	rs := session.NewRuleSet("CHIP")

	eddPath := filepath.Join(chipDir, "repository/xml/CHIP_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	dtPath := filepath.Join(chipDir, "repository/xml/CHIP_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	err = rs.LoadDecisionTables(dtFile)
	dtFile.Close()
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	// Define Go-based runtimes
	type runtimeDef struct {
		name     string
		executor interpreter.BytecodeExecutor
		skip     bool
		skipMsg  string
	}

	// Try to create x86-64-asm executor
	var x86Executor *asmruntime.Executor
	var x86Skip bool
	var x86SkipMsg string
	x86Executor, err = asmruntime.NewExecutor()
	if err != nil {
		x86Skip = true
		x86SkipMsg = fmt.Sprintf("x86-64-asm init failed: %v", err)
	} else {
		// Known to segfault -- skip gracefully
		x86Skip = true
		x86SkipMsg = "x86-64-asm skipped: CGO bridge not yet refactored for DTState"
	}
	_ = x86Executor // may be nil

	goRuntimes := []runtimeDef{
		{"go", nil, false, ""},
		{"nativeasm", nativeasm.NewExecutor(), false, ""},
		{"x86-64-asm", x86Executor, x86Skip, x86SkipMsg},
	}

	const iterations = 100

	// Results storage: runtime -> testcase -> ops/sec
	opsPerSec := make(map[string]map[string]float64)
	// Memory storage: runtime -> {totalAlloc, heapInuse}
	type memInfo struct {
		totalAlloc uint64
		heapInuse  uint64
	}
	memResults := make(map[string]memInfo)

	mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")

	// Run Go-based runtimes
	for _, rt := range goRuntimes {
		opsPerSec[rt.name] = make(map[string]float64)

		if rt.skip {
			t.Logf("Skipping runtime: %s (%s)", rt.name, rt.skipMsg)
			continue
		}

		t.Logf("Benchmarking runtime: %s (%d iterations per test case)", rt.name, iterations)

		// Memory measurement: GC and snapshot before
		goruntime.GC()
		var memBefore goruntime.MemStats
		goruntime.ReadMemStats(&memBefore)

		for _, testFile := range testFiles {
			testName := filepath.Base(testFile)
			var totalExec time.Duration

			for i := 0; i < iterations; i++ {
				iterStart := time.Now()

				// Create session
				sess, err := rs.NewSession()
				if err != nil {
					t.Fatalf("[%s] Failed to create session: %v", rt.name, err)
				}

				// Set the bytecode executor
				state := sess.GetState().(*interpreter.DTState)
				state.SetBytecodeExecutor(rt.executor)

				// Load mapping
				mapFile, err := os.Open(mapPath)
				if err != nil {
					t.Fatalf("[%s] Failed to open map file: %v", rt.name, err)
				}
				m := mapping.NewMapping(sess)
				m.LoadMapping(mapFile)
				mapFile.Close()
				m.Initialize()

				// Load test data
				testDataFile, err := os.Open(testFile)
				if err != nil {
					t.Fatalf("[%s] Failed to open test file %s: %v", rt.name, testName, err)
				}
				m.LoadData(testDataFile)
				testDataFile.Close()

				// Execute
				dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
				if err != nil {
					t.Fatalf("[%s] Failed to get decision table: %v", rt.name, err)
				}
				err = dtObj.Execute(sess.GetState())
				if err != nil && i == 0 {
					t.Logf("  [%s] %s execution error: %v", rt.name, testName, err)
				}

				totalExec += time.Since(iterStart)
			}

			avgSec := totalExec.Seconds() / float64(iterations)
			if avgSec > 0 {
				opsPerSec[rt.name][testName] = 1.0 / avgSec
			}
		}

		// Memory measurement: snapshot after
		var memAfter goruntime.MemStats
		goruntime.ReadMemStats(&memAfter)

		memResults[rt.name] = memInfo{
			totalAlloc: memAfter.TotalAlloc - memBefore.TotalAlloc,
			heapInuse:  memAfter.HeapInuse,
		}

		t.Logf("  %s complete", rt.name)
	}

	// Run Java benchmark
	javaOps := runJavaBenchmarkOps(t, chipDir, testFiles)
	opsPerSec["java"] = javaOps

	// Collect ordered runtime names
	allRuntimes := []string{"go", "nativeasm", "x86-64-asm", "java"}

	// Print header
	t.Log("")
	t.Log("=== CHIP Runtime Benchmark ===")
	t.Logf("Runtimes: %d | Test cases: %d | Iterations: %d", len(allRuntimes), len(testFiles), iterations)
	t.Log("")

	// Print memory table
	t.Log("Memory Footprint:")
	t.Logf("  %-16s %12s %12s", "Runtime", "TotalAlloc", "HeapInuse")
	t.Log("  " + strings.Repeat("\u2500", 42))
	for _, rtName := range allRuntimes {
		if rtName == "java" {
			t.Logf("  %-16s %12s %12s", rtName, "N/A", "(external)")
			continue
		}
		if mi, ok := memResults[rtName]; ok {
			t.Logf("  %-16s %12s %12s", rtName, formatBytes(mi.totalAlloc), formatBytes(mi.heapInuse))
		} else {
			t.Logf("  %-16s %12s %12s", rtName, "SKIPPED", "SKIPPED")
		}
	}
	t.Log("")

	// Print ops/sec table
	t.Log("Per-Test Results (ops/sec):")
	header := fmt.Sprintf("  %-45s", "Test Case")
	for _, rtName := range allRuntimes {
		header += fmt.Sprintf(" %12s", rtName)
	}
	t.Log(header)
	t.Log("  " + strings.Repeat("\u2500", 45+13*len(allRuntimes)))

	for _, testFile := range testFiles {
		testName := filepath.Base(testFile)
		row := fmt.Sprintf("  %-45s", testName)
		for _, rtName := range allRuntimes {
			if ops, ok := opsPerSec[rtName][testName]; ok {
				row += fmt.Sprintf(" %12.0f", ops)
			} else {
				// Check if this runtime was skipped
				skipped := false
				for _, rt := range goRuntimes {
					if rt.name == rtName && rt.skip {
						skipped = true
						break
					}
				}
				if skipped {
					row += fmt.Sprintf(" %12s", "SKIPPED")
				} else if rtName == "java" && len(javaOps) == 0 {
					row += fmt.Sprintf(" %12s", "N/A")
				} else {
					row += fmt.Sprintf(" %12s", "N/A")
				}
			}
		}
		t.Log(row)
	}

	// Averages
	t.Log("  " + strings.Repeat("\u2500", 45+13*len(allRuntimes)))
	avgRow := fmt.Sprintf("  %-45s", "AVERAGE")
	for _, rtName := range allRuntimes {
		rtOps := opsPerSec[rtName]
		if len(rtOps) > 0 {
			var total float64
			for _, ops := range rtOps {
				total += ops
			}
			avg := total / float64(len(rtOps))
			avgRow += fmt.Sprintf(" %12.0f", avg)
		} else {
			skipped := false
			for _, rt := range goRuntimes {
				if rt.name == rtName && rt.skip {
					skipped = true
					break
				}
			}
			if skipped {
				avgRow += fmt.Sprintf(" %12s", "SKIPPED")
			} else {
				avgRow += fmt.Sprintf(" %12s", "N/A")
			}
		}
	}
	t.Log(avgRow)
	t.Log("")
}

// runJavaBenchmarkOps runs the Java CHIP benchmark and returns ops/sec per test case.
func runJavaBenchmarkOps(t *testing.T, chipDir string, testFiles []string) map[string]float64 {
	results := make(map[string]float64)

	javaPath, err := exec.LookPath("java")
	if err != nil {
		t.Log("Note: Java benchmark skipped - java not in PATH")
		return results
	}

	targetClasses := filepath.Join(chipDir, "target/classes")
	if _, err := os.Stat(targetClasses); os.IsNotExist(err) {
		t.Log("Note: Java benchmark skipped - target/classes not found (run mvn compile)")
		return results
	}

	// Find DTRules engine and compiler classes
	projectRoot := filepath.Dir(filepath.Dir(chipDir))
	engineClasses := filepath.Join(projectRoot, "dtrules-engine/target/classes")
	compilerClasses := filepath.Join(projectRoot, "compilerutil/target/classes")

	// Build classpath
	libDir := filepath.Join(chipDir, "lib")
	libJars, _ := filepath.Glob(filepath.Join(libDir, "*.jar"))

	classpath := targetClasses + ":" + engineClasses + ":" + compilerClasses
	for _, jar := range libJars {
		classpath += ":" + jar
	}

	t.Log("Running Java benchmark...")

	cmd := exec.Command(javaPath, "-cp", classpath,
		"com.dtrules.samples.chipeligibility.TestChip")
	cmd.Dir = chipDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Note: Java benchmark had error (may still have results): %v", err)
	}

	// Parse output for timing information
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Avg execution time:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				timeStr := strings.TrimSpace(parts[len(parts)-1])
				if ms, err := strconv.ParseFloat(timeStr, 64); err == nil {
					// Convert ms to ops/sec
					if ms > 0 {
						ops := 1000.0 / ms // ms -> ops/sec
						for _, tf := range testFiles {
							results[filepath.Base(tf)] = ops
						}
					}
				}
			}
		}
	}

	if len(results) > 0 {
		t.Logf("Java benchmark completed: %d test cases", len(results))
	}

	return results
}

// formatBytes formats a byte count into a human-readable string.
func formatBytes(b uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
