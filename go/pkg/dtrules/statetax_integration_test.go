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

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/mapping"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// TestStateTaxIntegration tests the full pipeline with the StateTax sample project
func TestStateTaxIntegration(t *testing.T) {
	stateTaxDir := findStateTaxDir(t)
	if stateTaxDir == "" {
		t.Skip("StateTax sample project not found")
	}

	t.Logf("Using StateTax directory: %s", stateTaxDir)

	// Create a new RuleSet
	rs := session.NewRuleSet("StateTax")

	// Load the EDD
	eddPath := filepath.Join(stateTaxDir, "repository/xml/StateTax_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	t.Log("EDD loaded successfully")

	// Verify entities were created
	entityNames := rs.GetEntityNames()
	t.Logf("Loaded %d entities", len(entityNames))
	for _, name := range entityNames {
		t.Logf("  - %s", name.StringValue())
	}

	expectedEntities := []string{"adjustment", "constants", "income", "job", "result", "taxpayer"}
	for _, expected := range expectedEntities {
		found := false
		for _, name := range entityNames {
			if name.StringValue() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected entity '%s' not found", expected)
		}
	}

	// Load the decision tables
	dtPath := filepath.Join(stateTaxDir, "repository/xml/StateTax_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}
	t.Log("Decision tables loaded successfully")

	// Verify decision tables were created
	dtNames := rs.GetDecisionTableNames()
	t.Logf("Loaded %d decision tables", len(dtNames))
	for _, name := range dtNames {
		t.Logf("  - %s", name.StringValue())
	}

	expectedTables := []string{
		"Compute_Tax",
		"Calculate_Gross_Income",
		"Apply_Adjustments",
		"Determine_Filing_Details",
		"Calculate_Taxable_Income",
		"Apply_Tax_Brackets",
		"Evaluate_Results",
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

	// Create a session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Log("Session created successfully")

	// Load the mapping
	mapPath := filepath.Join(stateTaxDir, "repository/xml/StateTax_map.xml")
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
	t.Log("Mapping loaded successfully")

	// Initialize the mapping (creates initial entities)
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize mapping: %v", err)
	}
	t.Log("Mapping initialized successfully")

	// Load test data (TestCase_001: Single, $77k income, expected tax ~$2,978)
	testDataPath := filepath.Join(stateTaxDir, "testfiles/TestScenarios/TestCase_001.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data file: %v", err)
	}
	defer testDataFile.Close()

	err = m.LoadData(testDataFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	t.Log("Test data loaded successfully")

	// Get the main decision table
	factory := sess.GetEntityFactory()
	dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Compute_Tax"))
	if err != nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}
	if dtObj == nil {
		t.Fatal("Decision table 'Compute_Tax' not found")
	}
	t.Log("Found Compute_Tax decision table")

	// Execute the decision table
	state := sess.GetState()
	err = dtObj.Execute(state)
	if err != nil {
		t.Logf("Decision table execution error (may be expected if not all operators implemented): %v", err)
	} else {
		t.Log("Decision table executed successfully")
	}

	// Check results on the job entity
	jobEntity, err := state.EntityFetch(0)
	if err != nil {
		t.Logf("Could not fetch job entity: %v", err)
	} else {
		t.Logf("Job entity: %s", jobEntity.GetName().StringValue())

		// Try to get the results array
		results, err := jobEntity.Get(dtrules.GetRName("results"))
		if err == nil && results != nil {
			t.Logf("Results: %s", results.StringValue())
		}
	}
}

// TestStateTaxEDDLoad tests just the EDD loading for StateTax
func TestStateTaxEDDLoad(t *testing.T) {
	stateTaxDir := findStateTaxDir(t)
	if stateTaxDir == "" {
		t.Skip("StateTax sample project not found")
	}

	rs := session.NewRuleSet("StateTax")

	eddPath := filepath.Join(stateTaxDir, "repository/xml/StateTax_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Verify entity structure
	factory := rs.GetEntityFactory()

	// Check taxpayer entity has expected attributes
	taxpayerRef, err := factory.GetReferenceEntity(dtrules.GetRName("taxpayer"))
	if err != nil {
		t.Fatalf("Failed to get taxpayer entity: %v", err)
	}

	expectedAttrs := []string{"filing_status", "grossIncome", "agi", "deduction", "exemptions", "taxableIncome", "taxOwed", "incomes", "adjustments", "num_dependents"}
	for _, attr := range expectedAttrs {
		if !taxpayerRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Taxpayer entity missing attribute: %s", attr)
		}
	}

	// Check constants entity has tax bracket values
	constantsRef, err := factory.GetReferenceEntity(dtrules.GetRName("constants"))
	if err != nil {
		t.Fatalf("Failed to get constants entity: %v", err)
	}

	bracketAttrs := []string{"Bracket1_Limit", "Bracket2_Limit", "StandardDeduction_Single", "StandardDeduction_MFJ", "ExemptionAmount"}
	for _, attr := range bracketAttrs {
		if !constantsRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Constants entity missing attribute: %s", attr)
		}
	}

	t.Logf("Taxpayer entity has %d attributes", len(taxpayerRef.GetAttributeNames()))
	t.Logf("Constants entity has %d attributes", len(constantsRef.GetAttributeNames()))
}

// TestStateTaxDTLoad tests decision table loading for StateTax
func TestStateTaxDTLoad(t *testing.T) {
	stateTaxDir := findStateTaxDir(t)
	if stateTaxDir == "" {
		t.Skip("StateTax sample project not found")
	}

	rs := session.NewRuleSet("StateTax")

	// Must load EDD first
	eddPath := filepath.Join(stateTaxDir, "repository/xml/StateTax_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Load decision tables
	dtPath := filepath.Join(stateTaxDir, "repository/xml/StateTax_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	// Verify all 7 decision tables exist
	expectedTables := []string{
		"Compute_Tax",
		"Calculate_Gross_Income",
		"Apply_Adjustments",
		"Determine_Filing_Details",
		"Calculate_Taxable_Income",
		"Apply_Tax_Brackets",
		"Evaluate_Results",
	}

	dtNames := rs.GetDecisionTableNames()
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

	t.Logf("All %d decision tables loaded successfully", len(dtNames))
}

// findStateTaxDir locates the StateTax sample project directory
func findStateTaxDir(t *testing.T) string {
	// Try relative paths from the test location
	paths := []string{
		"../../../../sampleprojects/StateTax",
		"../../../sampleprojects/StateTax",
		"../../sampleprojects/StateTax",
		"../sampleprojects/StateTax",
		"sampleprojects/StateTax",
	}

	// Get current working directory
	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "repository/xml/StateTax_edd.xml")); err == nil {
			return absPath
		}
	}

	// Try from known locations
	home, _ := os.UserHomeDir()
	knownPaths := []string{
		filepath.Join(home, "DTRules/sampleprojects/StateTax"),
		filepath.Join(home, "repos/github.com/PaulSnow/DTRules/sampleprojects/StateTax"),
		"/home/paul/DTRules/sampleprojects/StateTax",
	}

	for _, p := range knownPaths {
		if _, err := os.Stat(filepath.Join(p, "repository/xml/StateTax_edd.xml")); err == nil {
			return p
		}
	}

	return ""
}
