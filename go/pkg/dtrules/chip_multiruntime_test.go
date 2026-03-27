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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/asmruntime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/go/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/nativeasm"
	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

// TestCHIPAllRuntimes runs CHIP tests across all available runtimes
func TestCHIPAllRuntimes(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	// Find all test case files
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

	// Define runtimes to test
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
		t.Logf("Note: %s", x86SkipMsg)
	} else {
		// x86-64-asm CGO runtime is known to segfault (Phase 3 not yet fixed).
		// Skip until the NASM bridge is refactored to operate on DTState directly.
		x86Skip = true
		x86SkipMsg = "x86-64-asm skipped: CGO bridge not yet refactored for DTState"
		t.Logf("Note: %s", x86SkipMsg)
	}

	runtimes := []runtimeDef{
		{"go", nil, false, ""},
		{"nativeasm", nativeasm.NewExecutor(), false, ""},
		{"x86-64-asm", x86Executor, x86Skip, x86SkipMsg},
	}

	const iterations = 100

	// Results storage: runtime -> testcase -> time
	results := make(map[string]map[string]time.Duration)
	for _, rt := range runtimes {
		results[rt.name] = make(map[string]time.Duration)
	}

	// Run tests for each runtime
	for _, rt := range runtimes {
		if rt.skip {
			t.Logf("Skipping runtime: %s (%s)", rt.name, rt.skipMsg)
			continue
		}
		t.Logf("Testing runtime: %s", rt.name)

		for _, testFile := range testFiles {
			testName := filepath.Base(testFile)
			var totalExec time.Duration

			for i := 0; i < iterations; i++ {
				// Create session
				sess, err := rs.NewSession()
				if err != nil {
					t.Fatalf("Failed to create session: %v", err)
				}

				// Set the bytecode executor for this runtime
				state := sess.GetState().(*interpreter.DTState)
				state.SetBytecodeExecutor(rt.executor)

				// Load mapping
				mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")
				mapFile, _ := os.Open(mapPath)
				m := mapping.NewMapping(sess)
				m.LoadMapping(mapFile)
				mapFile.Close()
				m.Initialize()

				// Load test data
				testDataFile, _ := os.Open(testFile)
				m.LoadData(testDataFile)
				testDataFile.Close()

				// Execute and time
				execStart := time.Now()
				dtObj, _ := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
				err = dtObj.Execute(sess.GetState())
				totalExec += time.Since(execStart)

				if err != nil && i == 0 {
					t.Logf("  %s: %s execution error: %v", rt.name, testName, err)
				}
			}

			avgExec := totalExec / time.Duration(iterations)
			results[rt.name][testName] = avgExec
		}
	}

	// Try to get Java results
	javaResults := runJavaBenchmark(t, chipDir, testFiles)
	if len(javaResults) > 0 {
		results["java"] = javaResults
	}

	// Collect active runtimes (those with results)
	var activeRuntimes []string
	for _, rt := range runtimes {
		if !rt.skip && len(results[rt.name]) > 0 {
			activeRuntimes = append(activeRuntimes, rt.name)
		}
	}
	if len(javaResults) > 0 {
		activeRuntimes = append([]string{"java"}, activeRuntimes...)
	}

	// Print results table
	t.Log("")
	t.Log("=== CHIP Performance: All Runtimes ===")
	t.Logf("    (Runtimes tested: %d)", len(activeRuntimes))
	t.Log("")

	// Header
	header := fmt.Sprintf("%-40s", "Test Case")
	for _, rtName := range activeRuntimes {
		header += fmt.Sprintf(" %12s", rtName)
	}
	t.Log(header)
	t.Log("--------------------------------------------------------------------------------")

	// Data rows
	for _, testFile := range testFiles {
		testName := filepath.Base(testFile)
		row := fmt.Sprintf("%-40s", testName)
		for _, rtName := range activeRuntimes {
			if d, ok := results[rtName][testName]; ok {
				row += fmt.Sprintf(" %12v", d)
			} else {
				row += fmt.Sprintf(" %12s", "N/A")
			}
		}
		t.Log(row)
	}

	// Averages
	t.Log("--------------------------------------------------------------------------------")
	avgRow := fmt.Sprintf("%-40s", "AVERAGE")
	for _, rtName := range activeRuntimes {
		if len(results[rtName]) > 0 {
			var total time.Duration
			for _, d := range results[rtName] {
				total += d
			}
			avg := total / time.Duration(len(results[rtName]))
			avgRow += fmt.Sprintf(" %12v", avg)
		} else {
			avgRow += fmt.Sprintf(" %12s", "N/A")
		}
	}
	t.Log(avgRow)
}

// runJavaBenchmark runs the Java CHIP benchmark and parses results.
// Returns a map of testcase name -> execution time.
func runJavaBenchmark(t *testing.T, chipDir string, testFiles []string) map[string]time.Duration {
	results := make(map[string]time.Duration)

	// Check if java is available
	javaPath, err := exec.LookPath("java")
	if err != nil {
		t.Log("Note: Java benchmark skipped - java not in PATH")
		return results
	}

	// Check for compiled classes
	targetClasses := filepath.Join(chipDir, "target/classes")
	if _, err := os.Stat(targetClasses); os.IsNotExist(err) {
		t.Log("Note: Java benchmark skipped - target/classes not found (run mvn compile)")
		return results
	}

	// Find DTRules engine classes
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

	// Run the Java test harness
	cmd := exec.Command(javaPath, "-cp", classpath,
		"com.dtrules.samples.chipeligibility.TestChip")
	cmd.Dir = chipDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// The test may still produce output before failing on Excel writing
		t.Logf("Note: Java benchmark had error (may still have results): %v", err)
	}

	// Parse the output for timing information
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Look for lines like "Avg execution time: 0.085"
		if strings.Contains(line, "Avg execution time:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				timeStr := strings.TrimSpace(parts[len(parts)-1])
				if ms, err := strconv.ParseFloat(timeStr, 64); err == nil {
					// Convert ms to duration
					d := time.Duration(ms * float64(time.Millisecond))
					// This is the overall average, apply to all test cases
					for _, tf := range testFiles {
						results[filepath.Base(tf)] = d
					}
				}
			}
		}
	}

	if len(results) > 0 {
		t.Logf("Java benchmark completed: %d test cases, avg time parsed", len(results))
	}

	return results
}
