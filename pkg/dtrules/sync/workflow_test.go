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

package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// WORKFLOW TESTS: Excel as Source of Truth
//
// These tests verify the complete workflow where:
// - Excel is the authoritative source (users edit Excel)
// - XML is the working copy (AI edits XML)
// - Exports to Excel are guarded to prevent overwriting user changes
// =============================================================================

// TestWorkflow_InitialExport tests the first export when no manifest exists.
func TestWorkflow_InitialExport(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create XML file (AI creates rules)
	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	if err := os.WriteFile(xmlFile, []byte("<rules>initial</rules>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create syncer with mock exporter
	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)

	// First export should succeed (no manifest entry = first time)
	pair := &FilePair{
		XMLPath:   xmlFile,
		ExcelPath: filepath.Join(excelDir, "rules_dt.xlsx"),
		XMLExists: true,
	}

	err := syncer.exportXMLToExcel(pair)
	if err != nil {
		t.Fatalf("Initial export should succeed: %v", err)
	}

	// Verify export was called
	if len(exporter.dtExports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(exporter.dtExports))
	}

	// The new contract (#1091): a project without a manifest never grows
	// one. Provenance in the XML carries the sync state; the export succeeds
	// and the directory stays clean.
	manifestPath := filepath.Join(excelDir, DefaultManifestName)
	if _, err := os.Stat(manifestPath); err == nil {
		t.Error("a fresh project must not grow a manifest on export (#1091)")
	}

	t.Log("SUCCESS: initial export creates no manifest")
}

// TestWorkflow_ConsecutiveExportsWithoutUserEdit tests multiple AI exports
// without any user edits in between. All should succeed.
func TestWorkflow_ConsecutiveExportsWithoutUserEdit(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	pair := &FilePair{
		XMLPath:   xmlFile,
		ExcelPath: excelFile,
		XMLExists: true,
	}

	// Export 1: AI creates initial rules
	if err := os.WriteFile(xmlFile, []byte("<rules>v1</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Export 1 failed: %v", err)
	}

	// Export 2: AI updates rules
	if err := os.WriteFile(xmlFile, []byte("<rules>v2</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	// Reload manifest
	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Export 2 failed: %v", err)
	}

	// Export 3: AI updates rules again
	if err := os.WriteFile(xmlFile, []byte("<rules>v3</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Export 3 failed: %v", err)
	}

	// All three exports should have succeeded
	if len(exporter.dtExports) != 3 {
		t.Errorf("Expected 3 exports, got %d", len(exporter.dtExports))
	}

	t.Log("SUCCESS: Multiple consecutive AI exports work without user edits")
}

// TestWorkflow_UserEditBlocksExport is the critical test:
// User edits Excel → AI tries to export → REJECTED
func TestWorkflow_UserEditBlocksExport(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	pair := &FilePair{
		XMLPath:   xmlFile,
		ExcelPath: excelFile,
		XMLExists: true,
	}

	// Step 1: AI does initial export. The project carries a manifest --
	// seeded explicitly, because a project without one never grows one now
	// (#1091), and this test is about the guard on a project that HAS one.
	m, err := LoadManifestFromDir(excelDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}
	if err := os.WriteFile(xmlFile, []byte("<rules>AI v1</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Initial export failed: %v", err)
	}

	// Step 2: USER modifies Excel (simulated)
	time.Sleep(100 * time.Millisecond)
	userEditTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile, []byte("USER EDITED TAX RATE"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile, userEditTime, userEditTime); err != nil {
		t.Fatal(err)
	}

	// Step 3: AI modifies XML and tries to export
	if err := os.WriteFile(xmlFile, []byte("<rules>AI v2</rules>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Reload manifest to pick up recorded export
	syncer.manifest = nil

	// Step 4: Export should be REJECTED
	err = syncer.exportXMLToExcel(pair)
	if err == nil {
		t.Fatal("CRITICAL FAILURE: Export should be REJECTED when user modified Excel")
	}

	if !IsExcelModifiedError(err) {
		t.Fatalf("Expected ExcelModifiedError, got %T: %v", err, err)
	}

	// Verify only 1 export happened (the initial one)
	if len(exporter.dtExports) != 1 {
		t.Errorf("Expected only 1 export (initial), got %d", len(exporter.dtExports))
	}

	t.Logf("SUCCESS: User edit blocks AI export: %v", err)
}

// TestWorkflow_RecoveryAfterUserEdit tests the recovery flow:
// User edits Excel → AI export rejected → AI imports Excel → AI can export again
func TestWorkflow_RecoveryAfterUserEdit(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	importer := &trackingImporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}
	syncer.SetImporter(importer)

	pair := &FilePair{
		XMLPath:     xmlFile,
		ExcelPath:   excelFile,
		XMLExists:   true,
		ExcelExists: true,
	}

	// Step 1: AI does initial export
	if err := os.WriteFile(xmlFile, []byte("<rules>AI v1</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Initial export failed: %v", err)
	}

	// Step 2: USER modifies Excel
	time.Sleep(100 * time.Millisecond)
	userEditTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile, []byte("USER EDITED"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile, userEditTime, userEditTime); err != nil {
		t.Fatal(err)
	}

	// Step 3: AI tries to export - REJECTED
	syncer.manifest = nil
	err := syncer.exportXMLToExcel(pair)
	if err == nil {
		t.Fatal("Export should be rejected")
	}
	t.Logf("Export correctly rejected: %v", err)

	// Step 4: AI imports Excel to XML (gets user's changes)
	if err := syncer.importExcelToXML(pair); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import was called
	if len(importer.dtImports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(importer.dtImports))
	}

	// Step 5: Now AI can export again (after incorporating user's changes)
	// The import updated the XML, and now we need to re-export
	// The key insight: after import, we "own" the Excel file again
	// because we've incorporated the user's changes.

	// Simulate: AI merges its changes with user's changes in XML
	if err := os.WriteFile(xmlFile, []byte("<rules>MERGED: user + AI</rules>"), 0644); err != nil {
		t.Fatal(err)
	}

	// After import, the manifest needs to be updated to reflect that
	// we've acknowledged the user's changes. We do this by touching
	// the Excel file to make it "ours" again (simulating the export).
	// In real code, the export would write new content to Excel.

	// Reset the Excel timestamp to "now" to simulate a fresh export
	// that incorporates the user's changes
	now := time.Now()
	if err := os.Chtimes(excelFile, now, now); err != nil {
		t.Fatal(err)
	}

	// Record that we've processed the user's changes
	manifest, _ := syncer.GetManifest()
	manifest.RecordExport(excelFile, []string{xmlFile})

	// Now export should work
	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Export after recovery should succeed: %v", err)
	}

	t.Log("SUCCESS: Recovery workflow works - import then export")
}

// TestWorkflow_RequireNoUserEditsCheck tests the pre-check function
func TestWorkflow_RequireNoUserEditsCheck(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	// Create files
	if err := os.WriteFile(xmlFile, []byte("<rules/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(excelFile, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	// Initial state: no manifest, so no user edits tracked
	err := syncer.RequireNoUserEdits()
	if err != nil {
		t.Fatalf("Should pass with no manifest: %v", err)
	}

	// Do an export to create manifest entry
	pair := &FilePair{
		XMLPath:   xmlFile,
		ExcelPath: excelFile,
		XMLExists: true,
	}
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatal(err)
	}

	// Check again - should still pass (no user edits)
	syncer.manifest = nil
	err = syncer.RequireNoUserEdits()
	if err != nil {
		t.Fatalf("Should pass after export: %v", err)
	}

	// Now simulate user edit
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile, []byte("USER EDIT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Check should now FAIL
	syncer.manifest = nil
	err = syncer.RequireNoUserEdits()
	if err == nil {
		t.Fatal("RequireNoUserEdits should fail after user edit")
	}

	t.Logf("SUCCESS: RequireNoUserEdits detects user changes: %v", err)
}

// TestWorkflow_MultipleFilesPartialUserEdit tests that user editing ONE file
// doesn't block exports to OTHER files.
func TestWorkflow_MultipleFilesPartialUserEdit(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create two separate rule files
	xmlFile1 := filepath.Join(xmlDir, "rules1_dt.xml")
	xmlFile2 := filepath.Join(xmlDir, "rules2_dt.xml")
	excelFile1 := filepath.Join(excelDir, "rules1_dt.xlsx")
	excelFile2 := filepath.Join(excelDir, "rules2_dt.xlsx")

	if err := os.WriteFile(xmlFile1, []byte("<rules>1</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xmlFile2, []byte("<rules>2</rules>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	pair1 := &FilePair{XMLPath: xmlFile1, ExcelPath: excelFile1, XMLExists: true}
	pair2 := &FilePair{XMLPath: xmlFile2, ExcelPath: excelFile2, XMLExists: true}

	// Export both files
	if err := syncer.exportXMLToExcel(pair1); err != nil {
		t.Fatal(err)
	}
	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair2); err != nil {
		t.Fatal(err)
	}

	// User edits only file 1
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile1, []byte("USER EDITED FILE 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile1, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Export to file 1 should be REJECTED
	syncer.manifest = nil
	err := syncer.exportXMLToExcel(pair1)
	if err == nil {
		t.Fatal("Export to user-edited file should be rejected")
	}
	if !IsExcelModifiedError(err) {
		t.Fatalf("Expected ExcelModifiedError, got %T", err)
	}

	// Export to file 2 should SUCCEED (user didn't touch it)
	syncer.manifest = nil
	err = syncer.exportXMLToExcel(pair2)
	if err != nil {
		t.Fatalf("Export to unmodified file should succeed: %v", err)
	}

	t.Log("SUCCESS: User edit blocks only the affected file, not others")
}

// TestWorkflow_CombinedWorkbookUserEdit tests combined workbook mode
// where one Excel file maps to multiple XML files.
func TestWorkflow_CombinedWorkbookUserEdit(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")
	statesXML := filepath.Join(xmlDir, "states")
	statesExcel := filepath.Join(excelDir, "states")

	if err := os.MkdirAll(statesXML, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(statesExcel, 0755); err != nil {
		t.Fatal(err)
	}

	// Combined workbook: CO.xlsx -> CO_dt.xml + CO_edd.xml
	dtXML := filepath.Join(statesXML, "CO_dt.xml")
	eddXML := filepath.Join(statesXML, "CO_edd.xml")
	excelWB := filepath.Join(statesExcel, "CO.xlsx")

	if err := os.WriteFile(dtXML, []byte("<dt/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eddXML, []byte("<edd/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	syncer.SetUseCombinedWorkbooks(true)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	wb := &CombinedWorkbook{
		ExcelPath:  excelWB,
		DTXMLPath:  dtXML,
		EDDXMLPath: eddXML,
		DTExists:   true,
		EDDExists:  true,
	}

	// Initial export
	if err := syncer.exportCombinedWorkbook(wb); err != nil {
		t.Fatalf("Initial export failed: %v", err)
	}

	// User edits the combined workbook
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelWB, []byte("USER EDITED CO TAXES"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelWB, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Second export should be REJECTED
	syncer.manifest = nil
	err := syncer.exportCombinedWorkbook(wb)
	if err == nil {
		t.Fatal("Export should be rejected after user edit")
	}

	if !IsExcelModifiedError(err) {
		t.Fatalf("Expected ExcelModifiedError, got %T: %v", err, err)
	}

	t.Logf("SUCCESS: Combined workbook user edit blocks export: %v", err)
}

// TestWorkflow_ManifestPersistence tests that the manifest persists across
// syncer instances (simulating AI sessions).
func TestWorkflow_ManifestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	if err := os.WriteFile(xmlFile, []byte("<rules/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Session 1: AI exports
	syncer1 := NewSyncer(xmlDir, excelDir)
	exporter1 := &trackingExporter{}
	syncer1.SetExporter(exporter1)
	// Persistence needs a manifest to persist; seeded explicitly (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	pair := &FilePair{XMLPath: xmlFile, ExcelPath: excelFile, XMLExists: true}
	if err := syncer1.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Session 1 export failed: %v", err)
	}

	// User edits between sessions
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile, []byte("USER EDIT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Session 2: New syncer instance (simulates new AI session)
	syncer2 := NewSyncer(xmlDir, excelDir)
	exporter2 := &trackingExporter{}
	syncer2.SetExporter(exporter2)

	// Export should be REJECTED (manifest persisted from session 1)
	err := syncer2.exportXMLToExcel(pair)
	if err == nil {
		t.Fatal("Session 2 export should be rejected - manifest should persist")
	}

	if !IsExcelModifiedError(err) {
		t.Fatalf("Expected ExcelModifiedError, got %T", err)
	}

	t.Log("SUCCESS: Manifest persists across syncer instances")
}

// TestWorkflow_ErrorMessageClarity tests that error messages are clear and actionable.
func TestWorkflow_ErrorMessageClarity(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "TaxReturn_dt.xml")
	excelFile := filepath.Join(excelDir, "TaxReturn_dt.xlsx")

	if err := os.WriteFile(xmlFile, []byte("<rules/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	pair := &FilePair{XMLPath: xmlFile, ExcelPath: excelFile, XMLExists: true}
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatal(err)
	}

	// User edits
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile, []byte("USER"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	syncer.manifest = nil
	err := syncer.exportXMLToExcel(pair)
	if err == nil {
		t.Fatal("Expected error")
	}

	errMsg := err.Error()

	// Error message should contain:
	// 1. The file path
	if !strings.Contains(errMsg, "TaxReturn_dt.xlsx") {
		t.Errorf("Error should mention file path, got: %s", errMsg)
	}

	// 2. Mention that it was modified
	if !strings.Contains(errMsg, "modified") {
		t.Errorf("Error should mention 'modified', got: %s", errMsg)
	}

	// 3. Mention the action to take
	if !strings.Contains(errMsg, "Import") {
		t.Errorf("Error should mention 'Import' as recovery action, got: %s", errMsg)
	}

	t.Logf("SUCCESS: Error message is clear: %s", errMsg)
}

// TestWorkflow_FullCycle tests a complete realistic workflow:
// 1. AI creates initial rules and exports
// 2. AI makes changes and exports again
// 3. User edits Excel
// 4. AI tries to export - REJECTED
// 5. AI imports Excel to get user changes
// 6. AI merges changes and exports successfully
func TestWorkflow_FullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	importer := &trackingImporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}
	syncer.SetImporter(importer)

	pair := &FilePair{
		XMLPath:   xmlFile,
		ExcelPath: excelFile,
		XMLExists: true,
	}

	// Phase 1: AI creates initial rules
	t.Log("Phase 1: AI creates initial rules")
	if err := os.WriteFile(xmlFile, []byte("<rules>AI_INITIAL</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Phase 1 failed: %v", err)
	}

	// Phase 2: AI makes more changes
	t.Log("Phase 2: AI makes additional changes")
	if err := os.WriteFile(xmlFile, []byte("<rules>AI_UPDATE_1</rules>"), 0644); err != nil {
		t.Fatal(err)
	}
	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Phase 2 failed: %v", err)
	}

	// Phase 3: User edits Excel
	t.Log("Phase 3: User edits Excel")
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelFile, []byte("USER_TAX_RATE_CHANGE"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Phase 4: AI tries to export - REJECTED
	t.Log("Phase 4: AI tries to export - expecting rejection")
	syncer.manifest = nil
	err := syncer.exportXMLToExcel(pair)
	if err == nil {
		t.Fatal("Phase 4: Export should be rejected")
	}
	if !IsExcelModifiedError(err) {
		t.Fatalf("Phase 4: Expected ExcelModifiedError, got %T", err)
	}
	t.Logf("Phase 4: Correctly rejected: %v", err)

	// Phase 5: AI imports Excel to get user's changes
	t.Log("Phase 5: AI imports Excel to get user changes")
	pair.ExcelExists = true
	if err := syncer.importExcelToXML(pair); err != nil {
		t.Fatalf("Phase 5 failed: %v", err)
	}

	// Phase 6: AI merges and exports
	t.Log("Phase 6: AI merges changes and exports")
	// In real scenario, AI would read the imported XML, merge with its pending changes
	if err := os.WriteFile(xmlFile, []byte("<rules>MERGED_USER_AND_AI</rules>"), 0644); err != nil {
		t.Fatal(err)
	}

	// After import, reset the Excel timestamp to "now" to simulate
	// acknowledging the user's changes (the export will write new content)
	now := time.Now()
	if err := os.Chtimes(excelFile, now, now); err != nil {
		t.Fatal(err)
	}

	// Record that we've processed the user's changes
	manifest, _ := syncer.GetManifest()
	manifest.RecordExport(excelFile, []string{xmlFile})

	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Phase 6 failed: %v", err)
	}

	// Verify the full cycle
	if len(exporter.dtExports) != 3 {
		t.Errorf("Expected 3 successful exports, got %d", len(exporter.dtExports))
	}
	if len(importer.dtImports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(importer.dtImports))
	}

	t.Log("SUCCESS: Full workflow cycle completed")
}

// TestWorkflow_TimestampPrecision tests that the grace period handles
// filesystem timestamp precision correctly.
func TestWorkflow_TimestampPrecision(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	xmlFile := filepath.Join(xmlDir, "rules_dt.xml")
	excelFile := filepath.Join(excelDir, "rules_dt.xlsx")

	if err := os.WriteFile(xmlFile, []byte("<rules/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &trackingExporter{}
	syncer.SetExporter(exporter)
	// The guards under test protect a project that HAS a manifest; seeded
	// explicitly, since a project without one never grows one (#1091).
	if m0, err := LoadManifestFromDir(excelDir); err != nil {
		t.Fatal(err)
	} else if err := m0.Save(); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	pair := &FilePair{XMLPath: xmlFile, ExcelPath: excelFile, XMLExists: true}

	// Export
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatal(err)
	}

	// Immediately try to export again (within grace period)
	// This should succeed because file wasn't modified externally
	syncer.manifest = nil
	if err := syncer.exportXMLToExcel(pair); err != nil {
		t.Fatalf("Rapid export should succeed (within grace period): %v", err)
	}

	t.Log("SUCCESS: Grace period handles timestamp precision correctly")
}
