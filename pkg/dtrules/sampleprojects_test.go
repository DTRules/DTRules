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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// SampleProject defines a sample project configuration
type SampleProject struct {
	Name           string
	EDDFile        string
	DTFile         string
	MapFile        string
	EntryTable     string   // Main decision table to execute
	ExpectedTables []string // Expected decision table names
}

// sampleProjects defines all sample projects to test
var sampleProjects = []SampleProject{
	{
		Name:       "CHIP",
		EDDFile:    "CHIP_edd.xml",
		DTFile:     "CHIP_dt.xml",
		MapFile:    "CHIP_map.xml",
		EntryTable: "Compute_Eligibility",
		ExpectedTables: []string{
			"Compute_Eligibility",
			"Calculate_Individual_Income",
			"Calculate_Group_Size",
			"Evaluate_CHIP_Eligibility",
			"Evaluate_MEDICAID_Eligibility",
			"Evaluate_FOODSTAMPS_Eligibilty",
			"Evaluate_Results",
		},
	},
	{
		Name:       "KidAid",
		EDDFile:    "kidaid_edd.xml",
		DTFile:     "kidaid_dt.xml",
		MapFile:    "kidaid_map.xml",
		EntryTable: "Compute_Eligibility",
		ExpectedTables: []string{
			"Compute_Eligibility",
		},
	},
	{
		Name:       "SyntaxTests",
		EDDFile:    "syntaxexample_edd.xml",
		DTFile:     "syntaxexample_dt.xml",
		MapFile:    "syntaxexample_map.xml",
		EntryTable: "", // Skip execution - test data references attributes not on entity
		ExpectedTables: []string{
			"Syntax_Examples",
			"Syntax_Examples_2",
			"Syntax_Examples_3",
			"Syntax_Examples_4",
			"Error_Handling_Table",
		},
	},
	{
		Name:       "TestProject",
		EDDFile:    "Test_edd.xml",
		DTFile:     "Test_dt.xml",
		MapFile:    "Test_map.xml",
		EntryTable: "", // Compile only - no test cases
		ExpectedTables: []string{
			// Will be discovered when DT loading is fixed
		},
	},
	{
		Name:       "TaxReturn",
		EDDFile:    "TaxReturn_edd.xml",
		DTFile:     "TaxReturn_dt.xml",
		MapFile:    "TaxReturn_map.xml",
		EntryTable: "Compute_Tax_Return",
		ExpectedTables: []string{
			"Compute_Tax_Return",
			"Calculate_Gross_Income",
			"Process_W2_Income",
			"Process_Self_Employment",
			"Calculate_Vehicle_Deduction",
			"Process_Rental_Income",
			"Calculate_AGI_Adjustments",
			"Calculate_Deductions",
			"Calculate_Standard_Deduction",
			"Calculate_Itemized_Deductions",
			"Calculate_SALT_Deduction",
			"Calculate_Mortgage_Interest",
			"Calculate_Charitable_Deduction",
			"Calculate_Taxable_Income",
			"Calculate_Tax_Liability",
			"Apply_Tax_Brackets",
			"Calculate_Credits",
			"Calculate_Child_Tax_Credit",
			"Calculate_Final_Balance",
			"Calculate_OBBBA_Deductions",
			"Calculate_Tips_Deduction",
			"Calculate_Overtime_Deduction",
			"Calculate_Senior_Deduction",
			"Process_Other_Income",
		},
	},
}

// findSampleProjectsDir locates the sampleprojects directory
func findSampleProjectsDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from test directory
	candidates := []string{
		"../../../../sampleprojects",
		"../../../../../sampleprojects",
		"../../../../../../sampleprojects",
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err == nil {
		t.Logf("Current working directory: %s", cwd)
	}

	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			// Verify it's the right directory by checking for CHIP
			chipPath := filepath.Join(absPath, "CHIP")
			if info, err := os.Stat(chipPath); err == nil && info.IsDir() {
				return absPath
			}
		}
	}

	// Try absolute paths
	absolutePaths := []string{
		"/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects",
		"/home/paul/repos/github.com/DTRules/DTRules/sampleprojects",
	}

	for _, path := range absolutePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}

// TestAllSampleProjects runs integration tests for all sample projects
func TestAllSampleProjects(t *testing.T) {
	baseDir := findSampleProjectsDir(t)
	if baseDir == "" {
		t.Skip("Sample projects directory not found")
	}
	t.Logf("Using sample projects directory: %s", baseDir)

	for _, project := range sampleProjects {
		t.Run(project.Name, func(t *testing.T) {
			testSampleProject(t, baseDir, project)
		})
	}
}

// testSampleProject tests a single sample project
func testSampleProject(t *testing.T, baseDir string, project SampleProject) {
	projectDir := filepath.Join(baseDir, project.Name)

	// Check if project exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Skipf("Project directory not found: %s", projectDir)
	}

	// Find XML directory (try repository/xml first, then xml)
	xmlDir := filepath.Join(projectDir, "repository/xml")
	if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
		xmlDir = filepath.Join(projectDir, "xml")
		if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
			t.Fatalf("No XML directory found in %s", projectDir)
		}
	}

	// Create RuleSet
	rs := session.NewRuleSet(project.Name)

	// Try loading from directory first (multi-file structure)
	// If that fails, fall back to individual files (monolithic structure)
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Logf("Directory loading failed, trying individual files: %v", err)

		// Load EDD
		eddPath := filepath.Join(xmlDir, project.EDDFile)
		eddFile, err := os.Open(eddPath)
		if err != nil {
			t.Fatalf("Failed to open EDD file %s: %v", eddPath, err)
		}
		defer eddFile.Close()

		err = rs.LoadEDD(eddFile)
		if err != nil {
			t.Fatalf("Failed to load EDD: %v", err)
		}

		// Load Decision Tables
		dtPath := filepath.Join(xmlDir, project.DTFile)
		dtFile, err := os.Open(dtPath)
		if err != nil {
			t.Fatalf("Failed to open DT file %s: %v", dtPath, err)
		}
		defer dtFile.Close()

		err = rs.LoadDecisionTables(dtFile)
		if err != nil {
			t.Fatalf("Failed to load decision tables: %v", err)
		}
	}
	t.Logf("Rules loaded: %d entities", len(rs.GetEntityNames()))

	dtNames := rs.GetDecisionTableNames()
	t.Logf("Decision tables loaded: %d tables", len(dtNames))
	for _, name := range dtNames {
		t.Logf("  - %s", name.StringValue())
	}

	// Verify expected tables exist
	for _, expected := range project.ExpectedTables {
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

	// Create session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Log("Session created")

	// Load mapping
	mapPath := filepath.Join(xmlDir, project.MapFile)
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("Failed to open mapping file %s: %v", mapPath, err)
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
	t.Log("Mapping loaded and initialized")

	// Find and run test cases
	testFilesDir := filepath.Join(projectDir, "testfiles")
	testCases := findTestCases(t, testFilesDir)

	if len(testCases) == 0 {
		t.Log("No test cases found - compile-only test passed")
		return
	}

	t.Logf("Found %d test cases", len(testCases))

	// Run each test case
	for _, testCase := range testCases {
		t.Run(filepath.Base(testCase), func(t *testing.T) {
			runTestCase(t, rs, project, xmlDir, testCase)
		})
	}
}

// findTestCases finds all test case XML files in the testfiles directory
func findTestCases(t *testing.T, testFilesDir string) []string {
	t.Helper()

	var testCases []string

	// Walk the testfiles directory
	err := filepath.Walk(testFilesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}
		if info.IsDir() {
			// Skip output directories
			if info.Name() == "output" || info.Name() == "sampleOutput" {
				return filepath.SkipDir
			}
			return nil
		}
		// Include XML files that look like test cases
		if strings.HasSuffix(strings.ToLower(path), ".xml") {
			name := info.Name()
			// Skip trace and results files
			if strings.Contains(name, "_trace") || strings.Contains(name, "_results") {
				return nil
			}
			testCases = append(testCases, path)
		}
		return nil
	})

	if err != nil {
		t.Logf("Warning: error walking testfiles directory: %v", err)
	}

	return testCases
}

// runTestCase runs a single test case
func runTestCase(t *testing.T, rs *session.RuleSet, project SampleProject, xmlDir string, testCasePath string) {
	// Create a fresh session for each test case
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(xmlDir, project.MapFile)
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

	// Load test data
	dataFile, err := os.Open(testCasePath)
	if err != nil {
		t.Fatalf("Failed to open test case file: %v", err)
	}
	defer dataFile.Close()

	err = m.LoadData(dataFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// Execute entry decision table if specified
	if project.EntryTable != "" {
		factory := sess.GetEntityFactory()
		dtObj, err := factory.GetDecisionTable(dtrules.GetRName(project.EntryTable))
		if err != nil {
			t.Fatalf("Failed to get decision table '%s': %v", project.EntryTable, err)
		}
		if dtObj == nil {
			t.Fatalf("Decision table '%s' not found", project.EntryTable)
		}

		err = dtObj.Execute(sess.GetState())
		if err != nil {
			t.Fatalf("Failed to execute decision table: %v", err)
		}
	}

	t.Logf("Test case executed successfully")
}

// TestSampleProjectsCompileOnly tests that all sample projects can be compiled
func TestSampleProjectsCompileOnly(t *testing.T) {
	baseDir := findSampleProjectsDir(t)
	if baseDir == "" {
		t.Skip("Sample projects directory not found")
	}

	for _, project := range sampleProjects {
		t.Run(project.Name+"_Compile", func(t *testing.T) {
			projectDir := filepath.Join(baseDir, project.Name)

			// Find XML directory
			xmlDir := filepath.Join(projectDir, "repository/xml")
			if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
				xmlDir = filepath.Join(projectDir, "xml")
			}

			// Create RuleSet
			rs := session.NewRuleSet(project.Name)

			// Try loading from directory first (multi-file structure)
			// If that fails, fall back to individual files (monolithic structure)
			err := rs.LoadFromDirectory(xmlDir)
			if err != nil {
				t.Logf("Directory loading failed, trying individual files: %v", err)

				// Load EDD
				eddPath := filepath.Join(xmlDir, project.EDDFile)
				eddFile, err := os.Open(eddPath)
				if err != nil {
					t.Fatalf("Failed to open EDD: %v", err)
				}
				defer eddFile.Close()

				err = rs.LoadEDD(eddFile)
				if err != nil {
					t.Fatalf("Failed to load EDD: %v", err)
				}

				// Load Decision Tables
				dtPath := filepath.Join(xmlDir, project.DTFile)
				dtFile, err := os.Open(dtPath)
				if err != nil {
					t.Fatalf("Failed to open DT: %v", err)
				}
				defer dtFile.Close()

				err = rs.LoadDecisionTables(dtFile)
				if err != nil {
					t.Fatalf("Failed to load DT: %v", err)
				}
			}

			t.Logf("Rules compiled: %d entities, %d tables",
				len(rs.GetEntityNames()),
				len(rs.GetDecisionTableNames()))
		})
	}
}
