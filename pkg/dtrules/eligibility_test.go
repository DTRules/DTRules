// Copyright 2024 Paul Snow
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package dtrules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestKidAid tests the KidAid project
// This is the original KidAid eligibility system with 7 decision tables
func TestKidAid(t *testing.T) {
	baseDir := findSampleProjectsDir(t)
	if baseDir == "" {
		t.Skip("Sample projects directory not found")
	}

	projectDir := filepath.Join(baseDir, "KidAid")
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Skip("KidAid project not found")
	}

	// Try both repository/xml and xml directories
	xmlDir := filepath.Join(projectDir, "repository/xml")
	if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
		xmlDir = filepath.Join(projectDir, "xml")
		if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
			t.Fatalf("KidAid xml directory not found")
		}
	}

	expectedTables := []string{
		"Compute_Eligibility",
		"Calculate_Individual_Income",
		"Calculate_Group_Size",
		"Evaluate_KidAid_Eligibility",
		"Evaluate_MEDICAID_Eligibility",
		"Evaluate_FOODSTAMPS_Eligibility",
		"Evaluate_Results",
	}

	t.Run("LoadRules", func(t *testing.T) {
		rs := session.NewRuleSet("KidAid")

		// Try directory loading first
		err := rs.LoadFromDirectory(xmlDir)
		if err != nil {
			// Fall back to individual files
			eddPath := filepath.Join(xmlDir, "kidaid_edd.xml")
			eddFile, err := os.Open(eddPath)
			if err != nil {
				t.Fatalf("Failed to open EDD file: %v", err)
			}
			defer eddFile.Close()

			err = rs.LoadEDD(eddFile)
			if err != nil {
				t.Fatalf("Failed to load EDD: %v", err)
			}

			dtPath := filepath.Join(xmlDir, "kidaid_dt.xml")
			dtFile, err := os.Open(dtPath)
			if err != nil {
				t.Fatalf("Failed to open DT file: %v", err)
			}
			defer dtFile.Close()

			err = rs.LoadDecisionTables(dtFile)
			if err != nil {
				t.Fatalf("Failed to load decision tables: %v", err)
			}
		}

		dtNames := rs.GetDecisionTableNames()
		t.Logf("Loaded %d decision tables", len(dtNames))

		if len(dtNames) != len(expectedTables) {
			t.Errorf("Expected %d decision tables, got %d", len(expectedTables), len(dtNames))
		}

		for _, expected := range expectedTables {
			found := false
			for _, name := range dtNames {
				if name.StringValue() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected decision table '%s' not found", expected)
			}
		}

		t.Log("Decision tables loaded:")
		for _, name := range dtNames {
			t.Logf("  - %s", name.StringValue())
		}
	})

	t.Run("ExecuteWithTestData", func(t *testing.T) {
		rs := session.NewRuleSet("KidAid")
		err := rs.LoadFromDirectory(xmlDir)
		if err != nil {
			t.Logf("LoadFromDirectory failed (expected for files without FILE_PATH): %v", err)
			t.Log("Falling back to individual file loading...")

			eddPath := filepath.Join(xmlDir, "kidaid_edd.xml")
			eddFile, err := os.Open(eddPath)
			if err != nil {
				t.Fatalf("Failed to open EDD: %v", err)
			}
			defer eddFile.Close()
			if err := rs.LoadEDD(eddFile); err != nil {
				t.Fatalf("Failed to load EDD: %v", err)
			}
			t.Log("EDD loaded successfully")

			dtPath := filepath.Join(xmlDir, "kidaid_dt.xml")
			dtFile, err := os.Open(dtPath)
			if err != nil {
				t.Fatalf("Failed to open DT: %v", err)
			}
			defer dtFile.Close()
			if err := rs.LoadDecisionTables(dtFile); err != nil {
				t.Fatalf("Failed to load decision tables: %v", err)
			}
			t.Log("Decision tables loaded successfully")
		} else {
			t.Log("LoadFromDirectory succeeded")
		}

		sess, err := rs.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Load mapping
		mapPath := filepath.Join(xmlDir, "kidaid_map.xml")
		mapFile, err := os.Open(mapPath)
		if err != nil {
			t.Fatalf("Failed to open mapping file: %v", err)
		}
		defer mapFile.Close()

		m := mapping.NewMapping(sess)
		err = m.LoadMapping(mapFile)
		if err != nil {
			t.Fatalf("Failed to load mapping: %v", err)
		}

		err = m.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize mapping: %v", err)
		}

		// Try to load and execute a test case
		testFilesDir := filepath.Join(projectDir, "testfiles")
		testCasePath := filepath.Join(testFilesDir, "TestScenarios/TestCase_001.xml")

		if _, err := os.Stat(testCasePath); err == nil {
			dataFile, err := os.Open(testCasePath)
			if err != nil {
				t.Fatalf("Failed to open test case: %v", err)
			}
			defer dataFile.Close()

			err = m.LoadData(dataFile)
			if err != nil {
				t.Fatalf("Failed to load test data: %v", err)
			}

			// Execute the main decision table
			factory := sess.GetEntityFactory()
			dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
			if err != nil {
				t.Fatalf("Failed to get Compute_Eligibility table: %v", err)
			}
			if dtObj == nil {
				t.Fatal("Compute_Eligibility decision table is nil")
			}

			state := sess.GetState()
			if state == nil {
				t.Fatal("Session state is nil")
			}

			err = dtObj.Execute(state)
			if err != nil {
				t.Fatalf("Failed to execute Compute_Eligibility: %v", err)
			}

			t.Log("Successfully executed TestCase_001")
		} else {
			t.Log("No test cases found, compile-only test passed")
		}
	})
}

// TestAllEligibilityProjects runs all eligibility tests in sequence
func TestAllEligibilityProjects(t *testing.T) {
	t.Run("KidAid", func(t *testing.T) {
		TestKidAid(t)
	})
}
