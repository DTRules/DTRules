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

// findSyntaxTestsDir locates the SyntaxTests sample project directory.
func findSyntaxTestsDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from the test location
	paths := []string{
		"../../../../sampleprojects/SyntaxTests",
		"../../../sampleprojects/SyntaxTests",
		"../../sampleprojects/SyntaxTests",
		"../sampleprojects/SyntaxTests",
		"sampleprojects/SyntaxTests",
	}

	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "repository/xml/syntaxexample_edd.xml")); err == nil {
			return absPath
		}
	}

	// Try absolute paths
	home, _ := os.UserHomeDir()
	knownPaths := []string{
		filepath.Join(home, "repos/github.com/DTRules/DTRules/sampleprojects/SyntaxTests"),
		"/home/paul/repos/github.com/DTRules/DTRules/sampleprojects/SyntaxTests",
	}

	for _, p := range knownPaths {
		if _, err := os.Stat(filepath.Join(p, "repository/xml/syntaxexample_edd.xml")); err == nil {
			return p
		}
	}

	return ""
}

// loadSyntaxTestsRuleSet creates and loads the SyntaxTests rule set (EDD + decision tables).
func loadSyntaxTestsRuleSet(t *testing.T, syntaxDir string) *session.RuleSet {
	t.Helper()

	rs := session.NewRuleSet("SyntaxExamples")

	// Load EDD
	eddPath := filepath.Join(syntaxDir, "repository/xml/syntaxexample_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Load decision tables
	dtPath := filepath.Join(syntaxDir, "repository/xml/syntaxexample_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	return rs
}

// setupSyntaxTestsSession creates a session with mapping initialized and test data loaded.
func setupSyntaxTestsSession(t *testing.T, rs *session.RuleSet, syntaxDir string) dtrules.Session {
	t.Helper()

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(syntaxDir, "repository/xml/syntaxexample_map.xml")
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
	testDataPath := filepath.Join(syntaxDir, "testfiles/test.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data file: %v", err)
	}
	defer testDataFile.Close()

	err = m.LoadData(testDataFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	return sess
}

// allSyntaxTables lists all decision tables in the SyntaxTests project.
var allSyntaxTables = []string{
	"Syntax_Examples",
	"Syntax_Examples_2",
	"Syntax_Examples_3",
	"Syntax_Examples_4",
	"Error_Handling_Table",
}

// executableSyntaxTables lists the tables whose contexts and actions are compatible
// with the provided test data. Syntax_Examples and Syntax_Examples_2 have contexts
// that reference attributes across entity boundaries (e.g., 'street' from address
// during client iteration, 'validatedCitizenship' from client during case iteration),
// which causes runtime errors with the test data. These tables are still valuable
// for testing EDD/DT compilation -- see TestSyntaxTestsCompile.
var executableSyntaxTables = []string{
	"Syntax_Examples_3",
	"Syntax_Examples_4",
	"Error_Handling_Table",
}

// TestSyntaxTestsIntegration is the main integration test for the SyntaxTests sample project.
// SyntaxTests exercises comprehensive EL (Expression Language) features including:
//   - Entity traversal and attribute access
//   - Array operations (forall, forfirst, memberof, sortarray)
//   - Local variable allocation (allocate/deallocate)
//   - Type conversions (cvi, cvr, cvs, cvb, cvd, cve)
//   - Arithmetic operations (+, -, *, /)
//   - Boolean logic and comparisons
//   - String operations (streq, strconcat)
//   - Entity stack operations (entitypush, entitypop)
//   - Control flow (if, for, forall, forfirst, forallr)
//   - Error handling constructs
//
// This makes it the most valuable test for catching Go/Java behavioral differences.
func TestSyntaxTestsIntegration(t *testing.T) {
	syntaxDir := findSyntaxTestsDir(t)
	if syntaxDir == "" {
		t.Skip("SyntaxTests sample project not found")
	}
	t.Logf("Using SyntaxTests directory: %s", syntaxDir)

	rs := loadSyntaxTestsRuleSet(t, syntaxDir)
	t.Log("SyntaxTests rules loaded successfully")

	// Verify entities
	entityNames := rs.GetEntityNames()
	t.Logf("Loaded %d entities", len(entityNames))

	expectedEntities := []string{"address", "case", "client", "constants", "ErrorDetails", "income", "job", "relationship", "result"}
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

	// Verify all decision tables compiled and loaded
	dtNames := rs.GetDecisionTableNames()
	t.Logf("Loaded %d decision tables", len(dtNames))
	for _, name := range dtNames {
		t.Logf("  - %s", name.StringValue())
	}
	if len(dtNames) != len(allSyntaxTables) {
		t.Errorf("Expected %d decision tables, got %d", len(allSyntaxTables), len(dtNames))
	}
	for _, expected := range allSyntaxTables {
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

	// Setup session with mapping and test data
	sess := setupSyntaxTestsSession(t, rs, syntaxDir)
	t.Log("Session created, mapping loaded, test data loaded")

	// Verify entity stack was populated by mapping
	state := sess.GetState()
	depth := state.EntityDepth()
	t.Logf("Entity stack depth after data load: %d", depth)
	if depth < 3 {
		t.Errorf("Expected at least 3 entities on stack (constants, job, case), got %d", depth)
	}

	// Execute tables whose contexts are compatible with the test data
	factory := sess.GetEntityFactory()
	for _, tableName := range executableSyntaxTables {
		t.Run("Execute_"+tableName, func(t *testing.T) {
			dtObj, err := factory.GetDecisionTable(dtrules.GetRName(tableName))
			if err != nil {
				t.Fatalf("Failed to get decision table '%s': %v", tableName, err)
			}
			if dtObj == nil {
				t.Fatalf("Decision table '%s' not found", tableName)
			}

			err = dtObj.Execute(state)
			if err != nil {
				t.Fatalf("Failed to execute decision table '%s': %v", tableName, err)
			}
			t.Logf("Decision table '%s' executed successfully", tableName)
		})
	}
}

// TestSyntaxTestsCompile verifies that all 5 SyntaxTests decision tables can be
// compiled from their postfix expressions. This is the core value of SyntaxTests:
// ensuring the Go EL compiler handles the same syntax constructs as the Java
// implementation.
func TestSyntaxTestsCompile(t *testing.T) {
	syntaxDir := findSyntaxTestsDir(t)
	if syntaxDir == "" {
		t.Skip("SyntaxTests sample project not found")
	}

	rs := loadSyntaxTestsRuleSet(t, syntaxDir)

	dtNames := rs.GetDecisionTableNames()
	if len(dtNames) != 5 {
		t.Fatalf("Expected 5 decision tables, got %d", len(dtNames))
	}

	// Verify every table is retrievable (compilation succeeded)
	factory := rs.GetEntityFactory()
	for _, expected := range allSyntaxTables {
		dt, err := factory.GetDecisionTable(dtrules.GetRName(expected))
		if err != nil {
			t.Errorf("Failed to get decision table '%s': %v", expected, err)
			continue
		}
		if dt == nil {
			t.Errorf("Decision table '%s' is nil after compilation", expected)
			continue
		}
		t.Logf("Compiled successfully: %s", expected)
	}
}

// TestSyntaxTestsLoadEDD tests EDD loading and entity structure for SyntaxTests.
func TestSyntaxTestsLoadEDD(t *testing.T) {
	syntaxDir := findSyntaxTestsDir(t)
	if syntaxDir == "" {
		t.Skip("SyntaxTests sample project not found")
	}

	rs := session.NewRuleSet("SyntaxExamples")

	eddPath := filepath.Join(syntaxDir, "repository/xml/syntaxexample_edd.xml")
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

	// Verify client entity attributes -- these exercise many types (boolean, integer,
	// float, string, time, entity, array) which are critical for EL type conversion tests
	clientRef, err := factory.GetReferenceEntity(dtrules.GetRName("client"))
	if err != nil {
		t.Fatalf("Failed to get client reference entity: %v", err)
	}

	expectedClientAttrs := []string{
		"age", "eligible", "validatedCitizenship", "totalIncome",
		"additionalIncome", "birthday", "dateofBirth", "incomes",
		"interestingClients", "interestingDates", "notes", "newClient",
	}
	for _, attr := range expectedClientAttrs {
		if !clientRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Client entity missing attribute: %s", attr)
		}
	}

	t.Logf("Client entity has %d attributes", len(clientRef.GetAttributeNames()))

	// Verify constants entity has expected default values for EL expression testing
	constantsRef, err := factory.GetReferenceEntity(dtrules.GetRName("constants"))
	if err != nil {
		t.Fatalf("Failed to get constants reference entity: %v", err)
	}

	expectedConstantsAttrs := []string{
		"FPL_Base", "FPL_PerAdditionalPerson", "coveredCounties",
		"ExcludedIncomeTypes", "diagCodes",
	}
	// `translationTable` was declared with the unsupported `type='table'`
	// and used nowhere in the project; removed in #783's narrow cleanup.
	for _, attr := range expectedConstantsAttrs {
		if !constantsRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Constants entity missing attribute: %s", attr)
		}
	}

	t.Logf("Constants entity has %d attributes", len(constantsRef.GetAttributeNames()))
}

// TestSyntaxTestsMapping tests mapping loading and data loading for SyntaxTests.
func TestSyntaxTestsMapping(t *testing.T) {
	syntaxDir := findSyntaxTestsDir(t)
	if syntaxDir == "" {
		t.Skip("SyntaxTests sample project not found")
	}

	rs := loadSyntaxTestsRuleSet(t, syntaxDir)
	sess := setupSyntaxTestsSession(t, rs, syntaxDir)

	// Verify entity stack state after data load
	state := sess.GetState()
	depth := state.EntityDepth()
	t.Logf("Entity stack depth: %d", depth)

	// The mapping initializes constants, job, and case on the entity stack
	if depth < 3 {
		t.Errorf("Expected at least 3 entities on stack, got %d", depth)
	}

	// Log entity stack contents
	for i := 0; i < depth; i++ {
		entity, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		t.Logf("Entity %d: %s", i, entity.GetName().StringValue())
	}
}

// TestSyntaxTestsExecuteEachTable tests executing each compatible decision table
// individually with a fresh session. This ensures each table can run independently
// without relying on side effects from other tables.
func TestSyntaxTestsExecuteEachTable(t *testing.T) {
	syntaxDir := findSyntaxTestsDir(t)
	if syntaxDir == "" {
		t.Skip("SyntaxTests sample project not found")
	}

	rs := loadSyntaxTestsRuleSet(t, syntaxDir)

	for _, tableName := range executableSyntaxTables {
		t.Run(tableName, func(t *testing.T) {
			// Create a fresh session for each table to test independence
			sess := setupSyntaxTestsSession(t, rs, syntaxDir)

			factory := sess.GetEntityFactory()
			dtObj, err := factory.GetDecisionTable(dtrules.GetRName(tableName))
			if err != nil {
				t.Fatalf("Failed to get decision table '%s': %v", tableName, err)
			}
			if dtObj == nil {
				t.Fatalf("Decision table '%s' not found", tableName)
			}

			state := sess.GetState()
			err = dtObj.Execute(state)
			if err != nil {
				t.Fatalf("Failed to execute '%s': %v", tableName, err)
			}
			t.Logf("Decision table '%s' executed successfully with fresh session", tableName)
		})
	}
}
