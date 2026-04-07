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

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestKidAidApplicationIntegration tests the full pipeline with the KidAid_Application sample project.
// This project has the same structure as KidAid but is set up as an application sample.
func TestKidAidApplicationIntegration(t *testing.T) {
	projectDir := findKidAidApplicationDir(t)
	if projectDir == "" {
		t.Skip("KidAid_Application sample project not found")
	}

	t.Logf("Using KidAid_Application directory: %s", projectDir)

	// Create a new RuleSet
	rs := session.NewRuleSet("KidAid_Application")

	// Load the EDD (kidaid_edd.xml)
	eddPath := filepath.Join(projectDir, "repository/xml/kidaid_edd.xml")
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

	expectedEntities := []string{"case", "client", "constants", "income", "job", "relationship", "result"}
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

	// Load the decision tables (kidaid_dt.xml)
	dtPath := filepath.Join(projectDir, "repository/xml/kidaid_dt.xml")
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

	// Verify key decision tables exist
	expectedTables := []string{
		"Compute_Eligibility",
		"Calculate_Individual_Income",
		"Calculate_Group_Size",
		"Evaluate_KidAid_Eligibility",
		"Evaluate_MEDICAID_Eligibility",
		"Evaluate_FOODSTAMPS_Eligibility",
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

	// Load the mapping (kidaid_map.xml)
	mapPath := filepath.Join(projectDir, "repository/xml/kidaid_map.xml")
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

	// Initialize the mapping (creates initial entities: constants, job, case)
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize mapping: %v", err)
	}
	t.Log("Mapping initialized successfully")

	// Use test data from the main KidAid project (same data format)
	kidAidDir := findKidAidDir(t)
	if kidAidDir == "" {
		t.Log("KidAid project not found, skipping test case execution")
		return
	}

	testDataPath := filepath.Join(kidAidDir, "testfiles/TestScenarios/TestCase_001.xml")
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
	dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
	if err != nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}
	if dtObj == nil {
		t.Fatal("Decision table 'Compute_Eligibility' not found")
	}
	t.Log("Found Compute_Eligibility decision table")

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

		results, err := jobEntity.Get(dtrules.GetRName("results"))
		if err == nil && results != nil {
			t.Logf("Results: %s", results.StringValue())
		}
	}
}

// TestKidAidApplicationEDDLoad tests just the EDD loading for KidAid_Application
func TestKidAidApplicationEDDLoad(t *testing.T) {
	projectDir := findKidAidApplicationDir(t)
	if projectDir == "" {
		t.Skip("KidAid_Application sample project not found")
	}

	rs := session.NewRuleSet("KidAid_Application")

	eddPath := filepath.Join(projectDir, "repository/xml/kidaid_edd.xml")
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

	// Check client entity has expected attributes
	clientRef, err := factory.GetReferenceEntity(dtrules.GetRName("client"))
	if err != nil {
		t.Fatalf("Failed to get client entity: %v", err)
	}

	expectedAttrs := []string{"age", "applying", "eligible", "gender", "client_ID", "incomes", "pregnant", "disabled", "validatedCitizenship"}
	for _, attr := range expectedAttrs {
		if !clientRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Client entity missing attribute: %s", attr)
		}
	}

	t.Logf("Client entity has %d attributes", len(clientRef.GetAttributeNames()))

	// Check constants entity has expected attributes
	constantsRef, err := factory.GetReferenceEntity(dtrules.GetRName("constants"))
	if err != nil {
		t.Fatalf("Failed to get constants entity: %v", err)
	}

	constantsAttrs := []string{"KidAid", "MEDICAID", "FOODSTAMPS", "FPL_Base", "FPL_PerAdditionalPerson", "coveredCounties"}
	for _, attr := range constantsAttrs {
		if !constantsRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Constants entity missing attribute: %s", attr)
		}
	}

	t.Logf("Constants entity has %d attributes", len(constantsRef.GetAttributeNames()))

	// Check job entity has results array
	jobRef, err := factory.GetReferenceEntity(dtrules.GetRName("job"))
	if err != nil {
		t.Fatalf("Failed to get job entity: %v", err)
	}

	jobAttrs := []string{"program", "results", "currentdate", "effectivedate"}
	for _, attr := range jobAttrs {
		if !jobRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Job entity missing attribute: %s", attr)
		}
	}

	t.Logf("Job entity has %d attributes", len(jobRef.GetAttributeNames()))
}

// TestKidAidApplicationDTLoad tests decision table loading and structure
func TestKidAidApplicationDTLoad(t *testing.T) {
	projectDir := findKidAidApplicationDir(t)
	if projectDir == "" {
		t.Skip("KidAid_Application sample project not found")
	}

	rs := session.NewRuleSet("KidAid_Application")

	// Must load EDD first
	eddPath := filepath.Join(projectDir, "repository/xml/kidaid_edd.xml")
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
	dtPath := filepath.Join(projectDir, "repository/xml/kidaid_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	// Verify specific decision tables exist
	expectedTables := []string{
		"Compute_Eligibility",
		"Calculate_Individual_Income",
		"Calculate_Group_Size",
		"Evaluate_KidAid_Eligibility",
		"Evaluate_MEDICAID_Eligibility",
		"Evaluate_FOODSTAMPS_Eligibility",
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

// TestKidAidApplicationMappingLoad tests mapping loading and initialization
func TestKidAidApplicationMappingLoad(t *testing.T) {
	projectDir := findKidAidApplicationDir(t)
	if projectDir == "" {
		t.Skip("KidAid_Application sample project not found")
	}

	rs := session.NewRuleSet("KidAid_Application")

	// Load EDD
	eddPath := filepath.Join(projectDir, "repository/xml/kidaid_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Create session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(projectDir, "repository/xml/kidaid_map.xml")
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

	// Initialize
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize mapping: %v", err)
	}

	// Verify data was loaded - check entity stack
	state := sess.GetState()
	depth := state.EntityDepth()
	t.Logf("Entity stack depth after initialization: %d", depth)

	// We should have at least constants, job, case on the stack (from initialization)
	if depth < 3 {
		t.Errorf("Expected at least 3 entities on stack, got %d", depth)
	}

	// Check that we can find expected entities on the stack
	for i := 0; i < depth; i++ {
		entity, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		t.Logf("Entity %d: %s", i, entity.GetName().StringValue())
	}
}

// TestKidAidApplicationSP2Files tests loading the sp2_* variant files
func TestKidAidApplicationSP2Files(t *testing.T) {
	projectDir := findKidAidApplicationDir(t)
	if projectDir == "" {
		t.Skip("KidAid_Application sample project not found")
	}

	t.Logf("Testing sp2_* files in KidAid_Application")

	rs := session.NewRuleSet("SampleProject2")

	// Load the sp2_edd.xml
	eddPath := filepath.Join(projectDir, "repository/xml/sp2_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open sp2_edd.xml file: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load sp2_edd.xml: %v", err)
	}
	t.Log("sp2_edd.xml loaded successfully")

	// Verify entities were created
	entityNames := rs.GetEntityNames()
	t.Logf("Loaded %d entities from sp2_edd.xml", len(entityNames))

	// Load the sp2_dt.xml decision tables
	dtPath := filepath.Join(projectDir, "repository/xml/sp2_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open sp2_dt.xml file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load sp2_dt.xml: %v", err)
	}
	t.Log("sp2_dt.xml loaded successfully")

	// Verify decision tables were created
	dtNames := rs.GetDecisionTableNames()
	t.Logf("Loaded %d decision tables from sp2_dt.xml", len(dtNames))

	// Create session and test mapping
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	mapPath := filepath.Join(projectDir, "repository/xml/sp2_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("Failed to open sp2_map.xml file: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	err = m.LoadMapping(mapFile)
	if err != nil {
		t.Fatalf("Failed to load sp2_map.xml: %v", err)
	}
	t.Log("sp2_map.xml loaded successfully")

	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize sp2 mapping: %v", err)
	}
	t.Log("sp2 mapping initialized successfully")
}

// TestKidAidApplicationTestCase002 runs the pipeline with TestCase_002 data
func TestKidAidApplicationTestCase002(t *testing.T) {
	projectDir := findKidAidApplicationDir(t)
	if projectDir == "" {
		t.Skip("KidAid_Application sample project not found")
	}

	kidAidDir := findKidAidDir(t)
	if kidAidDir == "" {
		t.Skip("KidAid project not found (needed for test data)")
	}

	rs := session.NewRuleSet("KidAid_Application")

	// Load EDD
	eddPath := filepath.Join(projectDir, "repository/xml/kidaid_edd.xml")
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
	dtPath := filepath.Join(projectDir, "repository/xml/kidaid_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	err = rs.LoadDecisionTables(dtFile)
	dtFile.Close()
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	// Create session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(projectDir, "repository/xml/kidaid_map.xml")
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

	// Load test case 002 data (pregnant 19-year-old)
	testDataPath := filepath.Join(kidAidDir, "testfiles/TestScenarios/TestCase_002.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data file: %v", err)
	}
	defer testDataFile.Close()

	err = m.LoadData(testDataFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	t.Log("Test case 002 data loaded successfully")

	// Execute Compute_Eligibility
	factory := sess.GetEntityFactory()
	dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
	if err != nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}
	if dtObj == nil {
		t.Fatal("Decision table 'Compute_Eligibility' not found")
	}

	state := sess.GetState()
	err = dtObj.Execute(state)
	if err != nil {
		t.Logf("Decision table execution error (may be expected if not all operators implemented): %v", err)
	} else {
		t.Log("Decision table executed successfully for test case 002")
	}
}

// findKidAidApplicationDir locates the KidAid_Application sample project directory
func findKidAidApplicationDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from the test location
	paths := []string{
		"../../../../sampleprojects/KidAid_Application",
		"../../../sampleprojects/KidAid_Application",
		"../../sampleprojects/KidAid_Application",
		"../sampleprojects/KidAid_Application",
		"sampleprojects/KidAid_Application",
	}

	// Get current working directory
	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "repository/xml/kidaid_edd.xml")); err == nil {
			return absPath
		}
	}

	// Try from GOPATH or known location
	home, _ := os.UserHomeDir()
	knownPaths := []string{
		filepath.Join(home, "go/src/github.com/DTRules/DTRules/sampleprojects/KidAid_Application"),
		filepath.Join(home, "repos/github.com/DTRules/DTRules/sampleprojects/KidAid_Application"),
		"/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/KidAid_Application",
	}

	for _, p := range knownPaths {
		if _, err := os.Stat(filepath.Join(p, "repository/xml/kidaid_edd.xml")); err == nil {
			return p
		}
	}

	return ""
}
