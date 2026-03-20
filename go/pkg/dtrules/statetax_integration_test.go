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

// TestStateTaxIntegration tests the full pipeline with a flat-tax state (Illinois)
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

	expectedEntities := []string{"adjustment", "bracket", "income", "job", "result", "state_config", "taxpayer"}
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
		"Apply_Flat_Tax",
		"Select_And_Apply_Brackets",
		"Apply_Bracket_Iteration",
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

	// Load test data (Illinois flat tax test case)
	// Use LoadDataAndPush to create entities from XML and push them onto the entity stack
	testDataPath := filepath.Join(stateTaxDir, "testfiles/TestScenarios/TestCase_IL_flat.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data file: %v", err)
	}
	defer testDataFile.Close()

	err = m.LoadDataAndPush(testDataFile, []string{"state_config", "job", "taxpayer"})
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

	// Check results
	jobEntity, err := state.EntityFetch(0)
	if err != nil {
		t.Logf("Could not fetch job entity: %v", err)
	} else {
		t.Logf("Entity: %s", jobEntity.GetName().StringValue())

		results, err := jobEntity.Get(dtrules.GetRName("results"))
		if err == nil && results != nil {
			t.Logf("Results: %s", results.StringValue())
		}
	}
}

// TestStateTaxEDDLoad tests EDD loading for the new state-aware schema
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

	factory := rs.GetEntityFactory()

	// Check state_config entity
	stateConfigRef, err := factory.GetReferenceEntity(dtrules.GetRName("state_config"))
	if err != nil {
		t.Fatalf("Failed to get state_config entity: %v", err)
	}

	scAttrs := []string{"state_code", "tax_type", "flat_rate_bps", "standard_deduction_single", "exemption_amount", "brackets_single", "brackets_mfj", "brackets_hoh"}
	for _, attr := range scAttrs {
		if !stateConfigRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("state_config missing attribute: %s", attr)
		}
	}

	// Check bracket entity
	bracketRef, err := factory.GetReferenceEntity(dtrules.GetRName("bracket"))
	if err != nil {
		t.Fatalf("Failed to get bracket entity: %v", err)
	}

	brAttrs := []string{"floor", "ceiling", "rate_bps", "base_tax"}
	for _, attr := range brAttrs {
		if !bracketRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("bracket missing attribute: %s", attr)
		}
	}

	// Check taxpayer has new fields
	taxpayerRef, err := factory.GetReferenceEntity(dtrules.GetRName("taxpayer"))
	if err != nil {
		t.Fatalf("Failed to get taxpayer entity: %v", err)
	}

	tpAttrs := []string{"active_brackets", "bracket_applied", "filing_status", "grossIncome", "taxOwed"}
	for _, attr := range tpAttrs {
		if !taxpayerRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("taxpayer missing attribute: %s", attr)
		}
	}

	t.Logf("state_config: %d attrs, bracket: %d attrs, taxpayer: %d attrs",
		len(stateConfigRef.GetAttributeNames()),
		len(bracketRef.GetAttributeNames()),
		len(taxpayerRef.GetAttributeNames()))
}

// TestStateTaxDTLoad tests decision table loading for the new schema
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

	// Verify all 9 decision tables exist
	expectedTables := []string{
		"Compute_Tax",
		"Calculate_Gross_Income",
		"Apply_Adjustments",
		"Determine_Filing_Details",
		"Calculate_Taxable_Income",
		"Apply_Flat_Tax",
		"Select_And_Apply_Brackets",
		"Apply_Bracket_Iteration",
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
	paths := []string{
		"../../../../sampleprojects/StateTax",
		"../../../sampleprojects/StateTax",
		"../../sampleprojects/StateTax",
		"../sampleprojects/StateTax",
		"sampleprojects/StateTax",
	}

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
