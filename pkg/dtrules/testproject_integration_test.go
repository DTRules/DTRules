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

// TestTestProjectIntegration tests the full pipeline with the TestProject sample project.
// It loads the EDD, decision tables, and mapping, then feeds in test data and
// executes the Test_Entry_Point decision table.
func TestTestProjectIntegration(t *testing.T) {
	testProjectDir := findTestProjectDir(t)
	if testProjectDir == "" {
		t.Skip("TestProject sample project not found")
	}

	t.Logf("Using TestProject directory: %s", testProjectDir)

	// Create a new RuleSet
	rs := session.NewRuleSet("TestProject")

	// Load the EDD
	eddPath := filepath.Join(testProjectDir, "xml/Test_edd.xml")
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

	expectedEntities := []string{"job", "thing"}
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
	dtPath := filepath.Join(testProjectDir, "xml/Test_dt.xml")
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

	// Verify the entry point table exists
	foundEntryPoint := false
	for _, name := range dtNames {
		if name.StringValue() == "Test_Entry_Point" {
			foundEntryPoint = true
			break
		}
	}
	if !foundEntryPoint {
		t.Fatal("Expected decision table 'Test_Entry_Point' not found")
	}

	// Create a session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Log("Session created successfully")

	// Load the mapping
	mapPath := filepath.Join(testProjectDir, "xml/Test_map.xml")
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
	testDataPath := filepath.Join(testProjectDir, "testfiles/TestScenarios/TestCase_001.xml")
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
	dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Test_Entry_Point"))
	if err != nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}
	if dtObj == nil {
		t.Fatal("Decision table 'Test_Entry_Point' not found")
	}
	t.Log("Found Test_Entry_Point decision table")

	// Execute the decision table
	state := sess.GetState()
	err = dtObj.Execute(state)
	if err != nil {
		t.Fatalf("Decision table execution failed: %v", err)
	}
	t.Log("Decision table executed successfully")

	// job.notes is the report the table builds: `clear the policy statements`
	// runs once per `for all things` pass, the fired column records what it
	// concluded about that thing, and `add the policy statements to the
	// job.notes` copies it in. So the report is one statement per thing, in
	// order, with `{thing.value}` resolved against that thing (#956).
	jobEntity, err := state.EntityFetch(0)
	if err != nil {
		t.Fatalf("Could not fetch job entity: %v", err)
	}

	notesObj, err := jobEntity.Get(dtrules.GetRName("notes"))
	if err != nil {
		t.Fatalf("job.notes: %v", err)
	}
	notes, err := notesObj.RArrayValue()
	if err != nil {
		t.Fatalf("job.notes is not an array: %v", err)
	}

	want := []string{
		"thing.value == 1",
		"thing.value == 2",
		"thing.value == 3",
		"thing.value is out of range, i.e.  99",
	}
	if notes.Size() != len(want) {
		t.Fatalf("job.notes holds %d statements, want %d: %s", notes.Size(), len(want), notesObj.StringValue())
	}
	for i, expected := range want {
		entry, err := notes.Get(i)
		if err != nil {
			t.Fatalf("job.notes[%d]: %v", i, err)
		}
		if got := entry.StringValue(); got != expected {
			t.Errorf("job.notes[%d] = %q, want %q", i, got, expected)
		}
	}

	things, err := jobEntity.Get(dtrules.GetRName("things"))
	if err != nil {
		t.Fatalf("job.things: %v", err)
	}
	thingArr, err := things.RArrayValue()
	if err != nil {
		t.Fatalf("job.things is not an array: %v", err)
	}
	if thingArr.Size() != 4 {
		t.Errorf("job.things holds %d entities, want 4 — the mapping did not build the test data", thingArr.Size())
	}
}

// TestTestProjectEDDLoad tests just the EDD loading for TestProject
func TestTestProjectEDDLoad(t *testing.T) {
	testProjectDir := findTestProjectDir(t)
	if testProjectDir == "" {
		t.Skip("TestProject sample project not found")
	}

	rs := session.NewRuleSet("TestProject")

	eddPath := filepath.Join(testProjectDir, "xml/Test_edd.xml")
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

	// Check job entity has expected attributes
	jobRef, err := factory.GetReferenceEntity(dtrules.GetRName("job"))
	if err != nil {
		t.Fatalf("Failed to get job entity: %v", err)
	}

	expectedJobAttrs := []string{"job", "mapping*key", "notes", "things"}
	for _, attr := range expectedJobAttrs {
		if !jobRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Job entity missing attribute: %s", attr)
		}
	}
	t.Logf("Job entity has %d attributes", len(jobRef.GetAttributeNames()))

	// Check thing entity has expected attributes
	thingRef, err := factory.GetReferenceEntity(dtrules.GetRName("thing"))
	if err != nil {
		t.Fatalf("Failed to get thing entity: %v", err)
	}

	expectedThingAttrs := []string{"thing", "mapping*key", "value"}
	for _, attr := range expectedThingAttrs {
		if !thingRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Thing entity missing attribute: %s", attr)
		}
	}
	t.Logf("Thing entity has %d attributes", len(thingRef.GetAttributeNames()))
}

// TestTestProjectDTLoad tests decision table loading and structure for TestProject
func TestTestProjectDTLoad(t *testing.T) {
	testProjectDir := findTestProjectDir(t)
	if testProjectDir == "" {
		t.Skip("TestProject sample project not found")
	}

	rs := session.NewRuleSet("TestProject")

	// Must load EDD first
	eddPath := filepath.Join(testProjectDir, "xml/Test_edd.xml")
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
	dtPath := filepath.Join(testProjectDir, "xml/Test_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	// Verify Test_Entry_Point decision table exists
	dtNames := rs.GetDecisionTableNames()
	found := false
	for _, name := range dtNames {
		if name.StringValue() == "Test_Entry_Point" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected decision table 'Test_Entry_Point' not found")
	}

	t.Logf("All %d decision tables loaded successfully", len(dtNames))
}

// findTestProjectDir locates the TestProject sample project directory
func findTestProjectDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from the test location
	paths := []string{
		"../../../../sampleprojects/TestProject",
		"../../../sampleprojects/TestProject",
		"../../sampleprojects/TestProject",
		"../sampleprojects/TestProject",
		"sampleprojects/TestProject",
	}

	// Get current working directory
	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "xml/Test_edd.xml")); err == nil {
			return absPath
		}
	}

	// Try from known locations
	home, _ := os.UserHomeDir()
	knownPaths := []string{
		filepath.Join(home, "repos/github.com/DTRules/DTRules/sampleprojects/TestProject"),
		"/home/paul/repos/github.com/DTRules/DTRules/sampleprojects/TestProject",
	}

	for _, p := range knownPaths {
		if _, err := os.Stat(filepath.Join(p, "xml/Test_edd.xml")); err == nil {
			return p
		}
	}

	return ""
}
