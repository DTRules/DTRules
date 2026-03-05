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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

// BenchmarkCHIPDecisionTable benchmarks the CHIP decision table execution
func BenchmarkCHIPDecisionTable(b *testing.B) {
	chipDir := findCHIPDirB(b)
	if chipDir == "" {
		b.Skip("CHIP sample project not found")
	}

	// Load rule set once
	rs := session.NewRuleSet("CHIP")

	eddPath := filepath.Join(chipDir, "repository/xml/CHIP_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		b.Fatalf("Failed to open EDD file: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		b.Fatalf("Failed to load EDD: %v", err)
	}

	dtPath := filepath.Join(chipDir, "repository/xml/CHIP_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		b.Fatalf("Failed to open DT file: %v", err)
	}
	err = rs.LoadDecisionTables(dtFile)
	dtFile.Close()
	if err != nil {
		b.Fatalf("Failed to load decision tables: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create session
		sess, err := rs.NewSession()
		if err != nil {
			b.Fatalf("Failed to create session: %v", err)
		}

		// Load mapping
		mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")
		mapFile, err := os.Open(mapPath)
		if err != nil {
			b.Fatalf("Failed to open mapping file: %v", err)
		}

		m := mapping.NewMapping(sess)
		err = m.LoadMapping(mapFile)
		mapFile.Close()
		if err != nil {
			b.Fatalf("Failed to load mapping: %v", err)
		}

		err = m.Initialize()
		if err != nil {
			b.Fatalf("Failed to initialize mapping: %v", err)
		}

		// Load test data
		testDataPath := filepath.Join(chipDir, "testfiles/TestScenarios/TestCase_001.xml")
		testDataFile, err := os.Open(testDataPath)
		if err != nil {
			b.Fatalf("Failed to open test data file: %v", err)
		}
		err = m.LoadData(testDataFile)
		testDataFile.Close()
		if err != nil {
			b.Fatalf("Failed to load test data: %v", err)
		}

		// Execute decision table
		dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
		if err != nil {
			b.Fatalf("Failed to get decision table: %v", err)
		}

		err = dtObj.Execute(sess.GetState())
		if err != nil {
			b.Fatalf("Decision table execution failed: %v", err)
		}
	}
}

// TestCHIPPerformance runs timing tests on CHIP decision tables
func TestCHIPPerformance(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	// Find all test case files from both directories
	var testFiles []string

	scenarioFiles, _ := filepath.Glob(filepath.Join(chipDir, "testfiles/TestScenarios/*.xml"))
	testFiles = append(testFiles, scenarioFiles...)

	perfFiles, _ := filepath.Glob(filepath.Join(chipDir, "testfiles/PerformanceTests/*.xml"))
	testFiles = append(testFiles, perfFiles...)

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

	t.Logf("Loaded %d decision tables", len(rs.GetDecisionTableNames()))
	t.Logf("Found %d test case files", len(testFiles))

	// Run performance tests
	const iterations = 100
	var totalExecTime time.Duration
	var totalLoadTime time.Duration
	var testCount int

	for _, testFile := range testFiles {
		testName := filepath.Base(testFile)

		for i := 0; i < iterations; i++ {
			// Measure session creation and data loading
			loadStart := time.Now()

			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")
			mapFile, err := os.Open(mapPath)
			if err != nil {
				t.Fatalf("Failed to open mapping file: %v", err)
			}

			m := mapping.NewMapping(sess)
			err = m.LoadMapping(mapFile)
			mapFile.Close()
			if err != nil {
				t.Fatalf("Failed to load mapping: %v", err)
			}

			err = m.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize mapping: %v", err)
			}

			testDataFile, err := os.Open(testFile)
			if err != nil {
				t.Fatalf("Failed to open test data file: %v", err)
			}
			err = m.LoadData(testDataFile)
			testDataFile.Close()
			if err != nil {
				t.Fatalf("Failed to load test data: %v", err)
			}

			loadTime := time.Since(loadStart)
			totalLoadTime += loadTime

			// Measure decision table execution
			execStart := time.Now()

			dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
			if err != nil {
				t.Fatalf("Failed to get decision table: %v", err)
			}

			err = dtObj.Execute(sess.GetState())
			if err != nil {
				t.Fatalf("%s: Decision table execution failed: %v", testName, err)
			}

			execTime := time.Since(execStart)
			totalExecTime += execTime
			testCount++
		}
	}

	avgLoadTime := totalLoadTime / time.Duration(testCount)
	avgExecTime := totalExecTime / time.Duration(testCount)

	t.Log("")
	t.Log("=== CHIP Performance Results (Go Runtime) ===")
	t.Logf("Test cases:        %d", len(testFiles))
	t.Logf("Iterations:        %d per test", iterations)
	t.Logf("Total executions:  %d", testCount)
	t.Log("")
	t.Logf("Average load time: %v", avgLoadTime)
	t.Logf("Average exec time: %v", avgExecTime)
	t.Logf("Total avg time:    %v", avgLoadTime+avgExecTime)
	t.Log("")
	t.Logf("Throughput:        %.0f executions/sec", float64(time.Second)/float64(avgExecTime))
}

// TestCHIPPerTestCase runs each test case individually with timing
func TestCHIPPerTestCase(t *testing.T) {
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

	t.Log("")
	t.Log("=== Per-Test-Case Results ===")
	t.Logf("%-40s %12s %12s", "Test Case", "Load", "Execute")
	t.Log("----------------------------------------------------------------")

	const iterations = 100

	for _, testFile := range testFiles {
		testName := filepath.Base(testFile)
		var loadTotal, execTotal time.Duration

		for i := 0; i < iterations; i++ {
			loadStart := time.Now()

			sess, _ := rs.NewSession()
			mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")
			mapFile, _ := os.Open(mapPath)
			m := mapping.NewMapping(sess)
			m.LoadMapping(mapFile)
			mapFile.Close()
			m.Initialize()

			testDataFile, _ := os.Open(testFile)
			m.LoadData(testDataFile)
			testDataFile.Close()

			loadTotal += time.Since(loadStart)

			execStart := time.Now()
			dtObj, _ := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
			dtObj.Execute(sess.GetState())
			execTotal += time.Since(execStart)
		}

		avgLoad := loadTotal / time.Duration(iterations)
		avgExec := execTotal / time.Duration(iterations)
		t.Logf("%-40s %12v %12v", testName, avgLoad, avgExec)
	}
}

// findCHIPDirB is the benchmark version of findCHIPDir
func findCHIPDirB(b *testing.B) string {
	paths := []string{
		"../../../../sampleprojects/CHIP",
		"../../../sampleprojects/CHIP",
		"../../sampleprojects/CHIP",
		"../sampleprojects/CHIP",
		"sampleprojects/CHIP",
	}

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "repository/xml/CHIP_edd.xml")); err == nil {
			return absPath
		}
	}
	return ""
}
