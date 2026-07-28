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

// TestKidAid_Application tests the KidAid_Application project
// This project contains TWO independent rulesets: kidaid and sp2
func TestKidAid_Application(t *testing.T) {
	baseDir := findSampleProjectsDir(t)
	if baseDir == "" {
		t.Skip("Sample projects directory not found")
	}

	projectDir := filepath.Join(baseDir, "KidAid_Application")
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Skip("KidAid_Application project not found")
	}

	xmlDir := filepath.Join(projectDir, "repository/xml")
	if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
		t.Fatalf("KidAid_Application xml directory not found: %s", xmlDir)
	}

	expectedKidAidTables := []string{
		"Compute_Eligibility",
		"Calculate_Individual_Income",
		"Calculate_Group_Size",
		"Evaluate_KidAid_Eligibility",
		"Evaluate_MEDICAID_Eligibility",
		"Evaluate_FOODSTAMPS_Eligibility",
		"Evaluate_Results",
	}

	expectedSp2Tables := []string{
		"Compute_Eligibility",
		"Calculate_Individual_Income",
		"Calculate_Group_Size",
		"Evaluate_KidAid_Eligibility",
		"Evaluate_MEDICAID_Eligibility",
		"Evaluate_FOODSTAMPS_Eligibilty", // Note: typo in original
		"Evaluate_Results",
	}

	t.Run("LoadKidAidRuleset", func(t *testing.T) {
		rs := session.NewRuleSet("KidAid_Application_KidAid")

		eddPath := filepath.Join(xmlDir, "kidaid_edd.xml")
		eddFile, err := os.Open(eddPath)
		if err != nil {
			t.Fatalf("Failed to open kidaid EDD file: %v", err)
		}
		defer eddFile.Close()

		err = rs.LoadEDD(eddFile)
		if err != nil {
			t.Fatalf("Failed to load kidaid EDD: %v", err)
		}

		dtPath := filepath.Join(xmlDir, "kidaid_dt.xml")
		dtFile, err := os.Open(dtPath)
		if err != nil {
			t.Fatalf("Failed to open kidaid DT file: %v", err)
		}
		defer dtFile.Close()

		err = rs.LoadDecisionTables(dtFile)
		if err != nil {
			t.Fatalf("Failed to load kidaid decision tables: %v", err)
		}

		dtNames := rs.GetDecisionTableNames()
		t.Logf("Loaded %d kidaid decision tables", len(dtNames))

		for _, expected := range expectedKidAidTables {
			found := false
			for _, name := range dtNames {
				if name.StringValue() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected kidaid decision table '%s' not found", expected)
			}
		}

		t.Log("KidAid ruleset tables:")
		for _, name := range dtNames {
			t.Logf("  - %s", name.StringValue())
		}
	})

	t.Run("LoadSp2Ruleset", func(t *testing.T) {
		rs := session.NewRuleSet("KidAid_Application_Sp2")

		eddPath := filepath.Join(xmlDir, "sp2_edd.xml")
		eddFile, err := os.Open(eddPath)
		if err != nil {
			t.Fatalf("Failed to open sp2 EDD file: %v", err)
		}
		defer eddFile.Close()

		err = rs.LoadEDD(eddFile)
		if err != nil {
			t.Fatalf("Failed to load sp2 EDD: %v", err)
		}

		dtPath := filepath.Join(xmlDir, "sp2_dt.xml")
		dtFile, err := os.Open(dtPath)
		if err != nil {
			t.Fatalf("Failed to open sp2 DT file: %v", err)
		}
		defer dtFile.Close()

		err = rs.LoadDecisionTables(dtFile)
		if err != nil {
			t.Fatalf("Failed to load sp2 decision tables: %v", err)
		}

		dtNames := rs.GetDecisionTableNames()
		t.Logf("Loaded %d sp2 decision tables", len(dtNames))

		for _, expected := range expectedSp2Tables {
			found := false
			for _, name := range dtNames {
				if name.StringValue() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected sp2 decision table '%s' not found", expected)
			}
		}

		t.Log("Sp2 ruleset tables:")
		for _, name := range dtNames {
			t.Logf("  - %s", name.StringValue())
		}
	})

	t.Run("VerifyIndependence", func(t *testing.T) {
		// Verify both rulesets can coexist and are independent
		rs1 := session.NewRuleSet("KidAid_Application_KidAid")
		rs2 := session.NewRuleSet("KidAid_Application_Sp2")

		// Load kidaid
		eddFile1, _ := os.Open(filepath.Join(xmlDir, "kidaid_edd.xml"))
		defer eddFile1.Close()
		rs1.LoadEDD(eddFile1)
		dtFile1, _ := os.Open(filepath.Join(xmlDir, "kidaid_dt.xml"))
		defer dtFile1.Close()
		rs1.LoadDecisionTables(dtFile1)

		// Load sp2
		eddFile2, _ := os.Open(filepath.Join(xmlDir, "sp2_edd.xml"))
		defer eddFile2.Close()
		rs2.LoadEDD(eddFile2)
		dtFile2, _ := os.Open(filepath.Join(xmlDir, "sp2_dt.xml"))
		defer dtFile2.Close()
		rs2.LoadDecisionTables(dtFile2)

		// Verify different table counts
		dt1 := rs1.GetDecisionTableNames()
		dt2 := rs2.GetDecisionTableNames()

		t.Logf("KidAid ruleset has %d tables, Sp2 ruleset has %d tables", len(dt1), len(dt2))

		// Both should have the same table names but be independent instances
		if len(dt1) != len(expectedKidAidTables) {
			t.Errorf("Expected %d kidaid tables, got %d", len(expectedKidAidTables), len(dt1))
		}
		if len(dt2) != len(expectedSp2Tables) {
			t.Errorf("Expected %d sp2 tables, got %d", len(expectedSp2Tables), len(dt2))
		}

		// Create sessions from both
		sess1, err := rs1.NewSession()
		if err != nil {
			t.Fatalf("Failed to create kidaid session: %v", err)
		}
		sess2, err := rs2.NewSession()
		if err != nil {
			t.Fatalf("Failed to create sp2 session: %v", err)
		}

		t.Log("Successfully created independent sessions for both rulesets")

		// Verify both can access their entry tables
		factory1 := sess1.GetEntityFactory()
		dt1Obj, err := factory1.GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
		if err != nil || dt1Obj == nil {
			t.Fatal("Failed to get Compute_Eligibility from kidaid ruleset")
		}

		factory2 := sess2.GetEntityFactory()
		dt2Obj, err := factory2.GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
		if err != nil || dt2Obj == nil {
			t.Fatal("Failed to get Compute_Eligibility from sp2 ruleset")
		}

		t.Log("Both rulesets are independent and functional")
	})
}

// TestAllEligibilityProjects runs all eligibility tests in sequence
func TestAllEligibilityProjects(t *testing.T) {
	t.Run("KidAid", func(t *testing.T) {
		TestKidAid(t)
	})
	t.Run("KidAid_Application", func(t *testing.T) {
		TestKidAid_Application(t)
	})
}
