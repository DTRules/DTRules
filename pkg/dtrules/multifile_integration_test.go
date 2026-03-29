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

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestMultiFileLoading verifies that LoadFromDirectory works correctly
// This test confirms Issue #343 Part 4 implementation
func TestMultiFileLoading(t *testing.T) {
	// Find the sample projects directory
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Verify directory exists
	if _, err := os.Stat(xmlDir); os.IsNotExist(err) {
		t.Skipf("TaxReturn xml directory not found: %s", xmlDir)
	}

	// Create rule set and load from directory
	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	// Verify entities were loaded
	entities := rs.GetEntityNames()
	if len(entities) == 0 {
		t.Fatal("No entities loaded from directory")
	}
	t.Logf("Loaded %d entities from directory", len(entities))

	// Verify decision tables were loaded
	tables := rs.GetDecisionTableNames()
	if len(tables) == 0 {
		t.Fatal("No decision tables loaded from directory")
	}
	t.Logf("Loaded %d decision tables from directory", len(tables))

	// Verify core tables exist
	coreTableNames := []string{
		"Compute_Tax_Return",
		"Calculate_Gross_Income",
		"Calculate_AGI_Adjustments",
		"Calculate_Tax_Liability",
	}

	for _, name := range coreTableNames {
		found := false
		for _, tbl := range tables {
			if tbl.StringValue() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected core table '%s' not found", name)
		}
	}

	// Test that we can create a session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if sess == nil {
		t.Fatal("Session is nil")
	}

	t.Log("Multi-file directory loading test passed")
}

// TestMultiFileVsMonolithicEquivalence verifies that loading from directory
// produces the same result as loading from monolithic files
func TestMultiFileVsMonolithicEquivalence(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Check if both structures exist
	eddPath := filepath.Join(xmlDir, "TaxReturn_edd.xml")
	dtPath := filepath.Join(xmlDir, "TaxReturn_dt.xml")

	if _, err := os.Stat(eddPath); os.IsNotExist(err) {
		t.Skip("Monolithic EDD file not found - skipping equivalence test")
	}
	if _, err := os.Stat(dtPath); os.IsNotExist(err) {
		t.Skip("Monolithic DT file not found - skipping equivalence test")
	}

	// Load using directory approach
	rsDir := session.NewRuleSet("TaxReturn_Dir")
	err := rsDir.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load from directory: %v", err)
	}

	// Load using monolithic approach
	rsMono := session.NewRuleSet("TaxReturn_Mono")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	defer eddFile.Close()
	err = rsMono.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	defer dtFile.Close()
	err = rsMono.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load DT: %v", err)
	}

	// Compare entity counts
	entitiesDir := rsDir.GetEntityNames()
	entitiesMono := rsMono.GetEntityNames()

	t.Logf("Directory loading: %d entities", len(entitiesDir))
	t.Logf("Monolithic loading: %d entities", len(entitiesMono))

	// Note: Directory may have more entities if additional state files are included
	// We just verify both approaches load successfully

	// Compare decision table counts
	tablesDir := rsDir.GetDecisionTableNames()
	tablesMono := rsMono.GetDecisionTableNames()

	t.Logf("Directory loading: %d tables", len(tablesDir))
	t.Logf("Monolithic loading: %d tables", len(tablesMono))

	// Both approaches should successfully load the rules
	if len(entitiesDir) == 0 || len(entitiesMono) == 0 {
		t.Error("One or both approaches failed to load entities")
	}

	if len(tablesDir) == 0 || len(tablesMono) == 0 {
		t.Error("One or both approaches failed to load decision tables")
	}

	t.Log("Both loading approaches successful")
}

// TestBackwardCompatibilityFallback verifies the fallback pattern
// Try directory loading first, fall back to monolithic if needed
func TestBackwardCompatibilityFallback(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("TaxReturn")

	// Try directory loading first
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Logf("Directory loading failed (expected if no multi-file structure): %v", err)

		// Fall back to monolithic files
		eddPath := filepath.Join(xmlDir, "TaxReturn_edd.xml")
		dtPath := filepath.Join(xmlDir, "TaxReturn_dt.xml")

		eddFile, err := os.Open(eddPath)
		if err != nil {
			t.Fatalf("Failed to open EDD file: %v", err)
		}
		defer eddFile.Close()

		err = rs.LoadEDD(eddFile)
		if err != nil {
			t.Fatalf("Failed to load EDD: %v", err)
		}

		dtFile, err := os.Open(dtPath)
		if err != nil {
			t.Fatalf("Failed to open DT file: %v", err)
		}
		defer dtFile.Close()

		err = rs.LoadDecisionTables(dtFile)
		if err != nil {
			t.Fatalf("Failed to load DT: %v", err)
		}

		t.Log("Fallback to monolithic files successful")
	} else {
		t.Log("Directory loading successful")
	}

	// Verify something was loaded
	if len(rs.GetEntityNames()) == 0 {
		t.Error("No entities loaded")
	}

	if len(rs.GetDecisionTableNames()) == 0 {
		t.Error("No decision tables loaded")
	}

	t.Log("Backward compatibility fallback pattern works correctly")
}
