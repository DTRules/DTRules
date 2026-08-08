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
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestCHIPSimpleExecution loads CHIP and executes it end to end.
//
// The long road here (#962): this test used to assert a successful run while
// the mapping loaded nothing into the entities it read, so it passed on a run
// that did nothing. Fixing the mapping made three real defects visible in
// turn — context locals invisible to their own table's rows (#965), the
// removed `is the <R> of` form (#927), and finally the relationship endpoints
// never loading at all, because a tag that both creates an entity and sets an
// attribute resolved its enclosure against the entity it had just created.
func TestCHIPSimpleExecution(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	t.Log("=== CHIP Simple Execution Test ===")
	t.Logf("CHIP directory: %s", chipDir)

	// Load rule set
	rs := session.NewRuleSet("CHIP")

	// Load EDD
	eddPath := filepath.Join(chipDir, "xml/CHIP_edd.xml")
	t.Logf("Loading EDD: %s", eddPath)
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	t.Log("✓ EDD loaded")

	// Load decision tables
	dtPath := filepath.Join(chipDir, "xml/CHIP_dt.xml")
	t.Logf("Loading decision tables: %s", dtPath)
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	err = rs.LoadDecisionTables(dtFile)
	dtFile.Close()
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}
	t.Log("✓ Decision tables loaded")

	// List all decision tables that were actually loaded
	t.Log("\nLoaded decision tables:")
	ef := rs.GetEntityFactory()
	dtList := ef.GetDecisionTableNames()
	t.Logf("Total decision tables loaded: %d", len(dtList))

	for i, dtName := range dtList {
		dt, err := ef.GetDecisionTable(dtName)
		if err != nil {
			t.Logf("  [%d] %s - ERROR: %v", i+1, dtName, err)
			continue
		}
		if dt == nil {
			t.Logf("  [%d] %s - NIL", i+1, dtName)
			continue
		}

		dtObj := dt.(*decisiontable.RDecisionTable)
		filePath := dtObj.GetFilePath()
		t.Logf("  [%d] %s - FILE_PATH: %s", i+1, dtName, filePath)
	}

	// Create session and test execution
	t.Log("\nTesting execution...")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(chipDir, "xml/CHIP_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("Failed to open mapping: %v", err)
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

	// Load test data
	testDataPath := filepath.Join(chipDir, "testfiles/TestScenarios/TestCase_001.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data: %v", err)
	}
	err = m.LoadData(testDataFile)
	testDataFile.Close()
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// Execute
	dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
	if err != nil {
		t.Fatalf("Failed to get Compute_Eligibility: %v", err)
	}
	err = dtObj.Execute(sess.GetState())
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	t.Log("✓ Execution completed successfully")
	t.Log("\n=== CHIP Test PASSED ===")
}

// TestChipAppSimpleExecution performs a simple load test for ChipApp
func TestChipAppSimpleExecution(t *testing.T) {
	chipAppDir := findChipAppDir(t)
	if chipAppDir == "" {
		t.Skip("ChipApp sample project not found")
	}

	t.Log("=== ChipApp Simple Execution Test ===")
	t.Logf("ChipApp directory: %s", chipAppDir)

	// Load rule set
	rs := session.NewRuleSet("ChipApp")

	// Load EDD
	eddPath := filepath.Join(chipAppDir, "xml/CHIP_edd.xml")
	t.Logf("Loading EDD: %s", eddPath)
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	t.Log("✓ EDD loaded")

	// Load decision tables
	dtPath := filepath.Join(chipAppDir, "xml/CHIP_dt.xml")
	t.Logf("Loading decision tables: %s", dtPath)
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	err = rs.LoadDecisionTables(dtFile)
	dtFile.Close()
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}
	t.Log("✓ Decision tables loaded")

	// List all decision tables
	t.Log("\nLoaded decision tables:")
	ef := rs.GetEntityFactory()
	dtList := ef.GetDecisionTableNames()
	t.Logf("Total decision tables loaded: %d", len(dtList))

	for i, dtName := range dtList {
		dt, err := ef.GetDecisionTable(dtName)
		if err != nil {
			t.Logf("  [%d] %s - ERROR: %v", i+1, dtName, err)
			continue
		}
		if dt == nil {
			t.Logf("  [%d] %s - NIL", i+1, dtName)
			continue
		}

		dtObj := dt.(*decisiontable.RDecisionTable)
		filePath := dtObj.GetFilePath()
		t.Logf("  [%d] %s - FILE_PATH: %s", i+1, dtName, filePath)
	}

	t.Log("\n=== ChipApp Test PASSED ===")
}

// Helper function to find ChipApp directory
func findChipAppDir(t *testing.T) string {
	paths := []string{
		"../../../sampleprojects/ChipApp",
		"../../sampleprojects/ChipApp",
		"sampleprojects/ChipApp",
	}

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	return ""
}

// TestCHIPCountsTheHousehold pins the outcome of #962: CHIP's household
// size must include relatives found through the relationship entities, not
// just the applicant.
//
// The scenario has clients 1001 and 1002 with a parent/child relationship
// between them, encoded as <source id="1001"/> / <target id="1002"/> — a tag
// that both creates the client entity and stores it on the enclosing
// relationship. Until the loader resolved that enclosure correctly, source
// and target loaded as null, every relationship row evaluated false, and the
// applicant was a household of one.
func TestCHIPCountsTheHousehold(t *testing.T) {
	chipDir := findCHIPDir(t)
	if chipDir == "" {
		t.Skip("CHIP sample project not found")
	}

	rs := session.NewRuleSet("CHIP")
	if err := rs.LoadFromDirectory(filepath.Join(chipDir, "xml")); err != nil {
		t.Fatalf("load: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	mf, err := os.Open(filepath.Join(chipDir, "xml", "CHIP_map.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer mf.Close()
	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mf); err != nil {
		t.Fatalf("mapping: %v", err)
	}
	df, err := os.Open(filepath.Join(chipDir, "testfiles", "TestScenarios", "TestCase_001.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer df.Close()
	if err := m.LoadDataAndPushSingletons(df); err != nil {
		t.Fatalf("data: %v", err)
	}

	state := sess.GetState()

	// The relationship endpoints must be entities, not null.
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != "case" {
			continue
		}
		ro, err := e.Get(dtrules.GetRName("relationships"))
		if err != nil || ro == nil || ro.Type() != dtrules.TypeArray {
			t.Fatal("case.relationships did not load")
		}
		arr, _ := ro.RArrayValue()
		if len(arr.GetIterator()) == 0 {
			t.Fatal("case.relationships is empty")
		}
		for _, r := range arr.GetIterator() {
			re, _ := r.REntityValue()
			for _, f := range []string{"source", "target"} {
				v, err := re.Get(dtrules.GetRName(f))
				if err != nil || v == nil || v.Type() != dtrules.TypeEntity {
					t.Errorf("relationship.%s is %v, want an entity — the endpoint did not resolve (#962)", f, v)
				}
			}
		}
	}

	entry, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Eligibility"))
	if err != nil || entry == nil {
		t.Fatalf("entry table: %v", err)
	}
	if err := entry.Execute(state); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// 1001 (applying, age 50) and 1002 (age 4) are parent and child, so the
	// household is 2. A count of 1 means the relationship rows found nothing.
	found := false
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != "client" {
			continue
		}
		v, err := e.Get(dtrules.GetRName("incomeGroupCount"))
		if err != nil || v == nil {
			continue
		}
		n, err := v.IntValue()
		if err != nil {
			continue
		}
		found = true
		if n != 2 {
			t.Errorf("incomeGroupCount = %d, want 2 (applicant plus the related child)", n)
		}
	}
	if !found {
		t.Error("no client carried an incomeGroupCount")
	}
}
