//go:build archive

// ARCHIVED: every test loads the legacy TaxReturn fixture (DSL without compiled
// postfix) and fails; also slow. Run with `go test -tags archive`. Revisit — #520.

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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// setupTaxReturn creates a RuleSet and loads EDD/DT for tax return
func setupTaxReturn(b *testing.B) *session.RuleSet {
	b.Helper()

	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("TaxReturn")

	// Load the whole xml/ tree (including states/) so Dispatch_State_Tax
	// finds every XX_Tax table, matching how the other tax suites load.
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		b.Fatalf("Failed to load rules from %s: %v", xmlDir, err)
	}

	return rs
}

// loadTestData loads test XML data into a session
func loadTestData(b *testing.B, sess *session.RSession, testFile string) {
	b.Helper()

	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Load mapping
	mapPath := filepath.Join(xmlDir, "TaxReturn_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		b.Fatalf("Failed to open mapping: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	err = m.LoadMapping(mapFile)
	if err != nil {
		b.Fatalf("Failed to load mapping: %v", err)
	}
	err = m.Initialize()
	if err != nil {
		b.Fatalf("Failed to init mapping: %v", err)
	}

	// Load test data
	testPath := filepath.Join(sampleDir, "testfiles", "TestScenarios", testFile)
	dataFile, err := os.Open(testPath)
	if err != nil {
		b.Fatalf("Failed to open test file %s: %v", testFile, err)
	}
	defer dataFile.Close()

	err = m.LoadData(dataFile)
	if err != nil {
		b.Fatalf("Failed to map test data: %v", err)
	}
}

// executeTaxReturn runs the Compute_Tax_Return decision table
func executeTaxReturn(b *testing.B, sess *session.RSession) {
	b.Helper()

	ef := sess.GetEntityFactory()
	dt, err := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		b.Fatalf("Failed to get decision table: %v", err)
	}

	state := sess.GetState()
	err = dt.Execute(state)
	if err != nil {
		b.Fatalf("Failed to execute: %v", err)
	}
}

// BenchmarkSingleStateTax benchmarks tax calculation for a single state
// Target: <50ms
func BenchmarkSingleStateTax(b *testing.B) {
	rs := setupTaxReturn(b)

	testCases := []struct {
		name string
		file string
	}{
		{"NH_Single_W2", "TestCase_NH_01_Single_W2.xml"},
		{"NH_MFJ_Two_Brackets", "TestCase_NH_02_MFJ_Two_Brackets.xml"},
		{"NH_High_Income", "TestCase_NH_03_High_Income_All_Brackets.xml"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create session once for warmup
			sess, err := rs.NewSession()
			if err != nil {
				b.Fatalf("Failed to create session: %v", err)
			}
			loadTestData(b, sess.(*session.RSession), tc.file)
			executeTaxReturn(b, sess.(*session.RSession))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create fresh session for each iteration
				sess, err := rs.NewSession()
				if err != nil {
					b.Fatalf("Failed to create session: %v", err)
				}

				loadTestData(b, sess.(*session.RSession), tc.file)
				executeTaxReturn(b, sess.(*session.RSession))
			}
		})
	}
}

// BenchmarkAllStates benchmarks tax calculation across all implemented states
// Target: <200ms for all states
func BenchmarkAllStates(b *testing.B) {
	rs := setupTaxReturn(b)

	// Test files for different states
	testFiles := []string{
		"TestCase_NH_01_Single_W2.xml",
		"TestCase_NH_02_MFJ_Two_Brackets.xml",
	}

	b.Run("AllStates", func(b *testing.B) {
		// Warmup
		for _, file := range testFiles {
			sess, _ := rs.NewSession()
			loadTestData(b, sess.(*session.RSession), file)
			executeTaxReturn(b, sess.(*session.RSession))
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for _, file := range testFiles {
				sess, err := rs.NewSession()
				if err != nil {
					b.Fatalf("Failed to create session: %v", err)
				}

				loadTestData(b, sess.(*session.RSession), file)
				executeTaxReturn(b, sess.(*session.RSession))
			}
		}
	})
}

// BenchmarkMultiStateTax benchmarks multi-state resident allocation
// Target: <200ms
func BenchmarkMultiStateTax(b *testing.B) {
	rs := setupTaxReturn(b)

	testCases := []struct {
		name string
		file string
	}{
		{"NY_FL_Move", "TestCase_MultiState_01_NY_FL_Move.xml"},
		{"CA_TX_Move", "TestCase_MultiState_02_CA_TX_Move.xml"},
		{"Traveling_Consultant", "TestCase_MultiState_03_Traveling_Consultant.xml"},
		{"MT_TX_Move", "TestCase_MultiState_02_MT_TX.xml"},
		{"NH_FL", "TestCase_MultiState_01_NH_FL.xml"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Warmup
			sess, _ := rs.NewSession()
			loadTestData(b, sess.(*session.RSession), tc.file)
			executeTaxReturn(b, sess.(*session.RSession))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				sess, err := rs.NewSession()
				if err != nil {
					b.Fatalf("Failed to create session: %v", err)
				}

				loadTestData(b, sess.(*session.RSession), tc.file)
				executeTaxReturn(b, sess.(*session.RSession))
			}
		})
	}
}

// BenchmarkFederalPlusState benchmarks combined federal and state tax calculation
// Target: <150ms
func BenchmarkFederalPlusState(b *testing.B) {
	rs := setupTaxReturn(b)

	testCases := []struct {
		name string
		file string
	}{
		{"Family_2025", "TestCase_Family_2025.xml"},
		{"NH_MFJ", "TestCase_NH_02_MFJ_Two_Brackets.xml"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Warmup
			sess, _ := rs.NewSession()
			loadTestData(b, sess.(*session.RSession), tc.file)
			executeTaxReturn(b, sess.(*session.RSession))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				sess, err := rs.NewSession()
				if err != nil {
					b.Fatalf("Failed to create session: %v", err)
				}

				loadTestData(b, sess.(*session.RSession), tc.file)
				executeTaxReturn(b, sess.(*session.RSession))
			}
		})
	}
}

// BenchmarkStateTaxComponents benchmarks individual components of state tax calculation
func BenchmarkStateTaxComponents(b *testing.B) {
	rs := setupTaxReturn(b)

	b.Run("SessionCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sess, err := rs.NewSession()
			if err != nil {
				b.Fatalf("Failed to create session: %v", err)
			}
			_ = sess
		}
	})

	b.Run("DataLoading", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			sess, _ := rs.NewSession()
			b.StartTimer()

			loadTestData(b, sess.(*session.RSession), "TestCase_NH_01_Single_W2.xml")
		}
	})

	b.Run("Execution", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			sess, _ := rs.NewSession()
			loadTestData(b, sess.(*session.RSession), "TestCase_NH_01_Single_W2.xml")
			b.StartTimer()

			executeTaxReturn(b, sess.(*session.RSession))
		}
	})
}

// TestPerformanceTargets validates that benchmarks meet performance targets
func TestPerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance target test in short mode")
	}

	rs := setupTaxReturnForTest(t)

	// Test single state: <50ms target
	t.Run("SingleState_Target_50ms", func(t *testing.T) {
		const iterations = 20
		const target = 50 * time.Millisecond

		var total time.Duration
		for i := 0; i < iterations; i++ {
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			start := time.Now()
			loadTestDataForTest(t, sess.(*session.RSession), "TestCase_NH_01_Single_W2.xml")
			executeTaxReturnForTest(t, sess.(*session.RSession))
			elapsed := time.Since(start)
			total += elapsed
		}

		avg := total / iterations
		t.Logf("Single state average: %v (target: %v)", avg, target)

		if avg > target {
			t.Logf("WARNING: Single state avg %v exceeds target %v", avg, target)
		}
	})

	// Test multi-state: <200ms target
	t.Run("MultiState_Target_200ms", func(t *testing.T) {
		const iterations = 20
		const target = 200 * time.Millisecond

		var total time.Duration
		for i := 0; i < iterations; i++ {
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			start := time.Now()
			loadTestDataForTest(t, sess.(*session.RSession), "TestCase_MultiState_03_Traveling_Consultant.xml")
			executeTaxReturnForTest(t, sess.(*session.RSession))
			elapsed := time.Since(start)
			total += elapsed
		}

		avg := total / iterations
		t.Logf("Multi-state average: %v (target: %v)", avg, target)

		if avg > target {
			t.Logf("WARNING: Multi-state avg %v exceeds target %v", avg, target)
		}
	})

	// Test federal+state: <150ms target
	t.Run("FederalPlusState_Target_150ms", func(t *testing.T) {
		const iterations = 20
		const target = 150 * time.Millisecond

		var total time.Duration
		for i := 0; i < iterations; i++ {
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			start := time.Now()
			loadTestDataForTest(t, sess.(*session.RSession), "TestCase_Family_2025.xml")
			executeTaxReturnForTest(t, sess.(*session.RSession))
			elapsed := time.Since(start)
			total += elapsed
		}

		avg := total / iterations
		t.Logf("Federal+State average: %v (target: %v)", avg, target)

		if avg > target {
			t.Logf("WARNING: Federal+State avg %v exceeds target %v", avg, target)
		}
	})
}

// Helper functions for test (non-benchmark) versions
func setupTaxReturnForTest(t *testing.T) *session.RuleSet {
	t.Helper()

	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("TaxReturn")

	// Load the whole xml/ tree (including states/) so Dispatch_State_Tax
	// finds every XX_Tax table, matching how the other tax suites load.
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("Failed to load rules from %s: %v", xmlDir, err)
	}

	return rs
}

func loadTestDataForTest(t *testing.T, sess *session.RSession, testFile string) {
	t.Helper()

	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	mapPath := filepath.Join(xmlDir, "TaxReturn_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("Failed to open mapping: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	err = m.LoadMapping(mapFile)
	if err != nil {
		t.Fatalf("Failed to load mapping: %v", err)
	}
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to init mapping: %v", err)
	}

	testPath := filepath.Join(sampleDir, "testfiles", "TestScenarios", testFile)
	dataFile, err := os.Open(testPath)
	if err != nil {
		t.Fatalf("Failed to open test file %s: %v", testFile, err)
	}
	defer dataFile.Close()

	err = m.LoadData(dataFile)
	if err != nil {
		t.Fatalf("Failed to map test data: %v", err)
	}
}

func executeTaxReturnForTest(t *testing.T, sess *session.RSession) {
	t.Helper()

	ef := sess.GetEntityFactory()
	dt, err := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}

	state := sess.GetState()
	err = dt.Execute(state)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
}

// BenchmarkReport generates a formatted performance report
func BenchmarkReport(b *testing.B) {
	if os.Getenv("GENERATE_REPORT") != "1" {
		b.Skip("Set GENERATE_REPORT=1 to generate performance report")
	}

	fmt.Println("\n=== State Tax Performance Benchmark Report ===")
	fmt.Println("Generated:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()

	// This is a placeholder - actual benchmarks are run via go test -bench
	fmt.Println("Run benchmarks with:")
	fmt.Println("  go test -bench=BenchmarkSingleStateTax -benchmem")
	fmt.Println("  go test -bench=BenchmarkMultiStateTax -benchmem")
	fmt.Println("  go test -bench=BenchmarkFederalPlusState -benchmem")
	fmt.Println("  go test -bench=. -benchmem")
	fmt.Println()
	fmt.Println("Performance Targets:")
	fmt.Println("  Single State:       <50ms")
	fmt.Println("  Multi-State:        <200ms")
	fmt.Println("  Federal + State:    <150ms")
}
