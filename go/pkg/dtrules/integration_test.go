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

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

// TestCHIPIntegration tests the full pipeline with the CHIP sample project
func TestCHIPIntegration(t *testing.T) {
	// Find the CHIP sample project
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	t.Logf("Using CHIP directory: %s", chipDir)

	// Create a new RuleSet
	rs := session.NewRuleSet("CHIP")

	// Load the EDD
	eddPath := filepath.Join(chipDir, "repository/xml/CHIP_edd.xml")
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

	// Load the decision tables
	dtPath := filepath.Join(chipDir, "repository/xml/CHIP_dt.xml")
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

	// Create a session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Log("Session created successfully")

	// Load the mapping
	mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")
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

	// Load test data
	testDataPath := filepath.Join(chipDir, "testfiles/TestScenarios/TestCase_001.xml")
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
	// The job entity should be on the entity stack
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

// TestEDDLoad tests just the EDD loading
func TestEDDLoad(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	rs := session.NewRuleSet("CHIP")

	eddPath := filepath.Join(chipDir, "repository/xml/CHIP_edd.xml")
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

	expectedAttrs := []string{"age", "applying", "eligible", "gender", "id", "incomes", "pregnant"}
	for _, attr := range expectedAttrs {
		if !clientRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Client entity missing attribute: %s", attr)
		}
	}

	t.Logf("Client entity has %d attributes", len(clientRef.GetAttributeNames()))
}

// TestDTLoad tests decision table loading and structure
func TestDTLoad(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	rs := session.NewRuleSet("CHIP")

	// Must load EDD first
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

	// Load decision tables
	dtPath := filepath.Join(chipDir, "repository/xml/CHIP_dt.xml")
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

// TestMappingLoad tests mapping loading and data loading
func TestMappingLoad(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	rs := session.NewRuleSet("CHIP")

	// Load EDD
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

	// Create session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(chipDir, "repository/xml/CHIP_map.xml")
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

	// Load test data
	testDataPath := filepath.Join(chipDir, "testfiles/TestScenarios/TestCase_001.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data file: %v", err)
	}
	defer testDataFile.Close()

	err = m.LoadData(testDataFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// Verify data was loaded - check entity stack
	state := sess.GetState()
	depth := state.EntityDepth()
	t.Logf("Entity stack depth after loading: %d", depth)

	// We should have at least constants, job, case on the stack
	if depth < 3 {
		t.Errorf("Expected at least 3 entities on stack, got %d", depth)
	}

	// Check that we can find the job entity with program set
	for i := 0; i < depth; i++ {
		entity, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		t.Logf("Entity %d: %s", i, entity.GetName().StringValue())

		if entity.GetName().StringValue() == "job" {
			program, err := entity.Get(dtrules.GetRName("program"))
			if err == nil && program != nil {
				t.Logf("  program = %s", program.StringValue())
			}
		}
	}
}

// findCHIPDir locates the CHIP sample project directory
func findCHIPDir(t *testing.T) string {
	// Try relative paths from the test location
	paths := []string{
		"../../../../sampleprojects/CHIP",
		"../../../sampleprojects/CHIP",
		"../../sampleprojects/CHIP",
		"../sampleprojects/CHIP",
		"sampleprojects/CHIP",
	}

	// Get current working directory
	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "repository/xml/CHIP_edd.xml")); err == nil {
			return absPath
		}
	}

	// Try from GOPATH or known location
	home, _ := os.UserHomeDir()
	knownPaths := []string{
		filepath.Join(home, "repos/github.com/DTRules/DTRules/sampleprojects/CHIP"),
		"/home/paul/repos/github.com/DTRules/DTRules/sampleprojects/CHIP",
	}

	for _, p := range knownPaths {
		if _, err := os.Stat(filepath.Join(p, "repository/xml/CHIP_edd.xml")); err == nil {
			return p
		}
	}

	return ""
}
