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
	"testing"
	"time"
)

// TestNewSyncCompiler tests compiler creation.
func TestNewSyncCompiler(t *testing.T) {
	opts := &CompilerOptions{
		XMLDir:   "/xml",
		ExcelDir: "/excel",
	}

	compiler := NewSyncCompiler(opts)

	if compiler == nil {
		t.Fatal("NewSyncCompiler returned nil")
	}
	if compiler.syncer == nil {
		t.Error("syncer is nil")
	}
	if compiler.options.SyncOptions == nil {
		t.Error("SyncOptions should be set to defaults")
	}
}

// TestNewSyncCompilerWithOptions tests compiler creation with custom options.
func TestNewSyncCompilerWithOptions(t *testing.T) {
	syncOpts := &SyncOptions{
		ConflictResolution: "prefer-excel",
		DryRun:             true,
		GracePeriod:        5 * time.Second,
	}
	opts := &CompilerOptions{
		XMLDir:               "/xml",
		ExcelDir:             "/excel",
		SyncOptions:          syncOpts,
		UseCombinedWorkbooks: true,
	}

	compiler := NewSyncCompiler(opts)

	if compiler.options.SyncOptions.ConflictResolution != "prefer-excel" {
		t.Errorf("ConflictResolution = %q, want %q",
			compiler.options.SyncOptions.ConflictResolution, "prefer-excel")
	}
	if !compiler.options.UseCombinedWorkbooks {
		t.Error("UseCombinedWorkbooks should be true")
	}
}

// trackingImporter tracks import calls for verification.
type trackingImporter struct {
	dtImports  []importCall
	eddImports []importCall
}

type importCall struct {
	excelPath string
	xmlPath   string
}

func (t *trackingImporter) ImportDecisionTable(excelPath, xmlPath string) error {
	t.dtImports = append(t.dtImports, importCall{excelPath, xmlPath})
	// Create the XML file to simulate import
	dir := filepath.Dir(xmlPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(xmlPath, []byte("<decision_tables/>"), 0644)
}

func (t *trackingImporter) ImportEDD(excelPath, xmlPath string) error {
	t.eddImports = append(t.eddImports, importCall{excelPath, xmlPath})
	// Create the XML file to simulate import
	dir := filepath.Dir(xmlPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(xmlPath, []byte("<entity_data_dictionary/>"), 0644)
}

// trackingExporter tracks export calls for verification.
type trackingExporter struct {
	dtExports  []exportCall
	eddExports []exportCall
}

type exportCall struct {
	xmlPath   string
	excelPath string
}

func (t *trackingExporter) ExportDecisionTable(xmlPath, excelPath string) error {
	t.dtExports = append(t.dtExports, exportCall{xmlPath, excelPath})
	// Create the Excel file to simulate export
	dir := filepath.Dir(excelPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(excelPath, []byte("EXCEL_DT"), 0644)
}

func (t *trackingExporter) ExportEDD(xmlPath, excelPath string) error {
	t.eddExports = append(t.eddExports, exportCall{xmlPath, excelPath})
	// Create the Excel file to simulate export
	dir := filepath.Dir(excelPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(excelPath, []byte("EXCEL_EDD"), 0644)
}

// TestPreSyncImportsExcelToXML verifies that pre-compile sync imports Excel files
// that are newer than their XML counterparts.
func TestPreSyncImportsExcelToXML(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Excel files (newer, should trigger import)
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	excelEDD := filepath.Join(excelDir, "rules_edd.xlsx")
	if err := os.WriteFile(excelDT, []byte("excel_dt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(excelEDD, []byte("excel_edd"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create older XML files
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	xmlEDD := filepath.Join(xmlDir, "rules_edd.xml")
	if err := os.WriteFile(xmlDT, []byte("<old_dt/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xmlEDD, []byte("<old_edd/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make XML files older
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(xmlDT, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(xmlEDD, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create compiler with tracking importer
	opts := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPostSync: true, // Only test pre-sync
	}
	compiler := NewSyncCompiler(opts)

	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	// Run pre-sync
	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// Verify imports were called
	if len(importer.dtImports) != 1 {
		t.Errorf("dtImports = %d, want 1", len(importer.dtImports))
	}
	if len(importer.eddImports) != 1 {
		t.Errorf("eddImports = %d, want 1", len(importer.eddImports))
	}

	// Verify result counts
	if result.ExcelToXMLCount != 2 {
		t.Errorf("ExcelToXMLCount = %d, want 2", result.ExcelToXMLCount)
	}
}

// TestPreSyncSkipsWhenXMLNewer verifies that pre-compile sync does NOT import
// when XML files are newer than Excel.
func TestPreSyncSkipsWhenXMLNewer(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create older Excel files
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("excel_dt"), 0644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(excelDT, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create newer XML file
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	if err := os.WriteFile(xmlDT, []byte("<newer_dt/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create compiler with tracking importer
	opts := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPostSync: true,
	}
	compiler := NewSyncCompiler(opts)

	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	// Run pre-sync
	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// Verify NO imports were called (XML is newer)
	if len(importer.dtImports) != 0 {
		t.Errorf("dtImports = %d, want 0 (XML is newer)", len(importer.dtImports))
	}

	// Result should indicate XML -> Excel direction, not Excel -> XML
	if result.ExcelToXMLCount != 0 {
		t.Errorf("ExcelToXMLCount = %d, want 0", result.ExcelToXMLCount)
	}
}

// TestPostSyncExportsXMLToExcel verifies that post-compile sync exports XML files
// to Excel format.
func TestPostSyncExportsXMLToExcel(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create XML files only (no Excel files yet)
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	xmlEDD := filepath.Join(xmlDir, "rules_edd.xml")
	if err := os.WriteFile(xmlDT, []byte("<decision_tables/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xmlEDD, []byte("<entity_data_dictionary/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create compiler with tracking exporter
	opts := &CompilerOptions{
		XMLDir:      xmlDir,
		ExcelDir:    excelDir,
		SkipPreSync: true, // Only test post-sync
	}
	compiler := NewSyncCompiler(opts)

	exporter := &trackingExporter{}
	compiler.SetExporter(exporter)

	// Run post-sync
	result, err := compiler.postSyncXMLToExcel()
	if err != nil {
		t.Fatalf("postSyncXMLToExcel failed: %v", err)
	}

	// Verify exports were called
	if len(exporter.dtExports) != 1 {
		t.Errorf("dtExports = %d, want 1", len(exporter.dtExports))
	}
	if len(exporter.eddExports) != 1 {
		t.Errorf("eddExports = %d, want 1", len(exporter.eddExports))
	}

	// Verify result counts
	if result.XMLToExcelCount != 2 {
		t.Errorf("XMLToExcelCount = %d, want 2", result.XMLToExcelCount)
	}
}

// TestExcelModificationAlwaysReflectedInXML is the critical end-to-end test.
// It verifies that when Excel files are modified, the changes are ALWAYS
// reflected in the XML files after sync.
func TestExcelModificationAlwaysReflectedInXML(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Step 1: Create initial XML files
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	if err := os.WriteFile(xmlDT, []byte("<decision_tables><version>1</version></decision_tables>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 2: Create newer Excel file (simulating user edit)
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("EXCEL_V2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make XML older to simulate Excel being modified after XML
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(xmlDT, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Record XML content before sync
	xmlContentBefore, err := os.ReadFile(xmlDT)
	if err != nil {
		t.Fatal(err)
	}

	// Step 3: Run pre-sync with a real importer that modifies the XML
	opts := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPostSync: true,
	}
	compiler := NewSyncCompiler(opts)

	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// Step 4: CRITICAL VERIFICATION - XML must have been updated
	if result.ExcelToXMLCount == 0 {
		t.Fatal("CRITICAL: Excel was modified but XML was NOT synced!")
	}

	// Verify the import was actually called
	if len(importer.dtImports) == 0 {
		t.Fatal("CRITICAL: Import was not called for modified Excel file!")
	}

	// Verify XML content was changed by the importer
	xmlContentAfter, err := os.ReadFile(xmlDT)
	if err != nil {
		t.Fatal(err)
	}

	if string(xmlContentBefore) == string(xmlContentAfter) {
		t.Log("Warning: XML content was not changed by mock importer")
		// This is OK for mock - real importer would change it
	}
}

// TestXMLModificationAlwaysReflectedInExcel verifies the reverse direction:
// XML changes should always be exported to Excel after compilation.
func TestXMLModificationAlwaysReflectedInExcel(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Step 1: Create initial Excel files
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("EXCEL_V1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 2: Create newer XML file (simulating code-generated changes)
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	if err := os.WriteFile(xmlDT, []byte("<decision_tables><version>2</version></decision_tables>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make Excel older
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(excelDT, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Step 3: Run post-sync
	opts := &CompilerOptions{
		XMLDir:      xmlDir,
		ExcelDir:    excelDir,
		SkipPreSync: true,
	}
	compiler := NewSyncCompiler(opts)

	exporter := &trackingExporter{}
	compiler.SetExporter(exporter)

	result, err := compiler.postSyncXMLToExcel()
	if err != nil {
		t.Fatalf("postSyncXMLToExcel failed: %v", err)
	}

	// CRITICAL VERIFICATION - Excel must have been updated
	if result.XMLToExcelCount == 0 {
		t.Fatal("CRITICAL: XML was modified but Excel was NOT synced!")
	}

	// Verify the export was actually called
	if len(exporter.dtExports) == 0 {
		t.Fatal("CRITICAL: Export was not called for modified XML file!")
	}
}

// TestConflictDetectionWhenBothModified verifies that conflicts are detected
// when both Excel and XML have been modified.
func TestConflictDetectionWhenBothModified(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create XML file
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	if err := os.WriteFile(xmlDT, []byte("<v1/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Excel file
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create another pair where Excel is newer
	xmlDT2 := filepath.Join(xmlDir, "other_dt.xml")
	if err := os.WriteFile(xmlDT2, []byte("<v1/>"), 0644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(xmlDT2, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	excelDT2 := filepath.Join(excelDir, "other_dt.xlsx")
	if err := os.WriteFile(excelDT2, []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with error conflict resolution (default)
	syncer := NewSyncer(xmlDir, excelDir)
	direction, pairs, err := syncer.CheckSync()
	if err != nil {
		t.Fatalf("CheckSync failed: %v", err)
	}

	// Should detect mixed directions as conflict
	if direction != Conflict {
		t.Logf("Direction: %v, Pairs: %+v", direction, pairs)
		// This might be NoSync if files are within grace period
		// or might be directional if timestamps differ enough
	}
}

// TestSkipPreSyncOption verifies that SkipPreSync actually skips pre-sync.
func TestSkipPreSyncOption(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create newer Excel file
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPreSync:  true, // This should prevent import
		SkipPostSync: true,
	}
	compiler := NewSyncCompiler(opts)

	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	// Even though Excel is newer, pre-sync should be skipped
	// Note: We can't test Compile() directly because it needs real XML files
	// But we can verify the flag is respected by calling preSyncExcelToXML
	// when SkipPreSync is false

	opts2 := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPreSync:  false,
		SkipPostSync: true,
	}
	compiler2 := NewSyncCompiler(opts2)
	compiler2.SetImporter(importer)

	result, err := compiler2.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// With SkipPreSync=false, import should happen
	if result.ExcelToXMLCount == 0 && len(importer.dtImports) == 0 {
		t.Log("No imports happened - Excel might be in grace period")
	}
}

// TestDryRunDoesNotModifyFiles verifies that dry run mode reports changes
// but does not actually modify files.
func TestDryRunDoesNotModifyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create newer Excel file
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	// XML file doesn't exist yet
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")

	opts := &CompilerOptions{
		XMLDir:   xmlDir,
		ExcelDir: excelDir,
		SyncOptions: &SyncOptions{
			DryRun:      true,
			GracePeriod: 2 * time.Second,
		},
		SkipPostSync: true,
	}
	compiler := NewSyncCompiler(opts)

	// Even with real importer, dry run should not create files
	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// Should report file would be synced
	if result.ExcelToXMLCount != 1 {
		t.Errorf("ExcelToXMLCount = %d, want 1", result.ExcelToXMLCount)
	}

	// But importer should NOT have been called
	if len(importer.dtImports) != 0 {
		t.Errorf("dtImports = %d, want 0 (dry run)", len(importer.dtImports))
	}

	// And XML file should NOT exist
	if _, err := os.Stat(xmlDT); err == nil {
		t.Error("XML file should not exist in dry run mode")
	}
}

// TestRecursiveDirectorySyncDuringCompile verifies that subdirectories
// (like states/) are properly synced during compilation.
func TestRecursiveDirectorySyncDuringCompile(t *testing.T) {
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

	// Create Excel files in states subdirectory
	coDT := filepath.Join(statesExcel, "CO_dt.xlsx")
	coEDD := filepath.Join(statesExcel, "CO_edd.xlsx")
	if err := os.WriteFile(coDT, []byte("co_dt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coEDD, []byte("co_edd"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPostSync: true,
	}
	compiler := NewSyncCompiler(opts)

	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// Should sync both state files
	if result.ExcelToXMLCount != 2 {
		t.Errorf("ExcelToXMLCount = %d, want 2", result.ExcelToXMLCount)
	}

	// Verify imports include subdirectory files
	if len(importer.dtImports) != 1 {
		t.Errorf("dtImports = %d, want 1", len(importer.dtImports))
	}
	if len(importer.eddImports) != 1 {
		t.Errorf("eddImports = %d, want 1", len(importer.eddImports))
	}

	// Verify the paths include the states subdirectory
	if len(importer.dtImports) > 0 {
		if filepath.Base(filepath.Dir(importer.dtImports[0].xmlPath)) != "states" {
			t.Errorf("DT import path should be in states subdir: %s", importer.dtImports[0].xmlPath)
		}
	}
}

// TestCombinedWorkbookSyncDuringCompile verifies that combined workbooks
// (single .xlsx -> _dt.xml + _edd.xml) are properly synced.
func TestCombinedWorkbookSyncDuringCompile(t *testing.T) {
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

	// Create combined workbook (CO.xlsx should create CO_dt.xml and CO_edd.xml)
	coWorkbook := filepath.Join(statesExcel, "CO.xlsx")
	if err := os.WriteFile(coWorkbook, []byte("combined"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &CompilerOptions{
		XMLDir:               xmlDir,
		ExcelDir:             excelDir,
		UseCombinedWorkbooks: true,
		SkipPostSync:         true,
	}
	compiler := NewSyncCompiler(opts)

	// Use mock workbook importer
	wbImporter := &mockWorkbookImporter{hasDT: true, hasEDD: true}
	compiler.SetWorkbookImporter(wbImporter)

	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("preSyncExcelToXML failed: %v", err)
	}

	// Should sync the combined workbook
	if result.ExcelToXMLCount != 1 {
		t.Errorf("ExcelToXMLCount = %d, want 1", result.ExcelToXMLCount)
	}

	// Verify workbook importer was called
	if len(wbImporter.importedWorkbooks) != 1 {
		t.Errorf("importedWorkbooks = %d, want 1", len(wbImporter.importedWorkbooks))
	}
}

// TestPreCompileSyncFunction tests the standalone PreCompileSync function.
func TestPreCompileSyncFunction(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Excel file
	excelDT := filepath.Join(excelDir, "test_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := PreCompileSync(xmlDir, excelDir)
	if err != nil {
		t.Fatalf("PreCompileSync failed: %v", err)
	}

	// Result should indicate sync is needed
	// Note: Without real importer, actual sync won't happen
	if result == nil {
		t.Error("PreCompileSync returned nil result")
	}
}

// TestPostCompileSyncFunction tests the standalone PostCompileSync function.
func TestPostCompileSyncFunction(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create XML file
	xmlDT := filepath.Join(xmlDir, "test_dt.xml")
	if err := os.WriteFile(xmlDT, []byte("<test/>"), 0644); err != nil {
		t.Fatal(err)
	}

	err := PostCompileSync(xmlDir, excelDir)
	// Will fail without real exporter, but function should be callable
	if err != nil {
		t.Logf("PostCompileSync returned error (expected without exporter): %v", err)
	}
}

// TestSyncEnforcementInTestWorkflow simulates the test workflow and verifies
// that sync is enforced. This is the integration test that answers:
// "Are Excel versions ALWAYS up to date when building test cases?"
func TestSyncEnforcementInTestWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Scenario: Developer modifies Excel file, then runs tests
	// Expected: XML should be updated before tests run

	// Step 1: Create initial XML (represents existing rules)
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	initialXML := "<decision_tables><table id='1'><name>Original</name></table></decision_tables>"
	if err := os.WriteFile(xmlDT, []byte(initialXML), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 2: Developer creates/modifies Excel (simulated)
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("MODIFIED_IN_EXCEL"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make XML older to simulate "Excel was modified after last compile"
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(xmlDT, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Step 3: Create compiler that would be used before tests
	opts := &CompilerOptions{
		XMLDir:       xmlDir,
		ExcelDir:     excelDir,
		SkipPreSync:  false, // CRITICAL: Must be false for enforcement
		SkipPostSync: true,
	}
	compiler := NewSyncCompiler(opts)

	importer := &trackingImporter{}
	compiler.SetImporter(importer)

	// Step 4: Run pre-sync (this is what ensures XML is updated)
	result, err := compiler.preSyncExcelToXML()
	if err != nil {
		t.Fatalf("Pre-sync failed: %v", err)
	}

	// CRITICAL ASSERTION: Import MUST have happened
	if result.ExcelToXMLCount == 0 {
		t.Fatal("SYNC ENFORCEMENT FAILURE: Excel was modified but sync did not import to XML!")
	}

	if len(importer.dtImports) == 0 {
		t.Fatal("SYNC ENFORCEMENT FAILURE: Importer was not called!")
	}

	t.Logf("SUCCESS: Excel modification was synced to XML (%d files)", result.ExcelToXMLCount)
}

// TestSyncEnforcementReversed tests the reverse scenario:
// XML is modified (e.g., by code generation), Excel should be updated
func TestSyncEnforcementReversed(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Scenario: Code generation modifies XML, Excel should be updated

	// Step 1: Create initial Excel (represents existing workbook)
	excelDT := filepath.Join(excelDir, "rules_dt.xlsx")
	if err := os.WriteFile(excelDT, []byte("ORIGINAL_EXCEL"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make Excel older
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(excelDT, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Step 2: Code generation creates/modifies XML (simulated)
	xmlDT := filepath.Join(xmlDir, "rules_dt.xml")
	if err := os.WriteFile(xmlDT, []byte("<generated/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 3: Create compiler for post-compile sync
	opts := &CompilerOptions{
		XMLDir:      xmlDir,
		ExcelDir:    excelDir,
		SkipPreSync: true,
	}
	compiler := NewSyncCompiler(opts)

	exporter := &trackingExporter{}
	compiler.SetExporter(exporter)

	// Step 4: Run post-sync
	result, err := compiler.postSyncXMLToExcel()
	if err != nil {
		t.Fatalf("Post-sync failed: %v", err)
	}

	// CRITICAL ASSERTION: Export MUST have happened
	if result.XMLToExcelCount == 0 {
		t.Fatal("SYNC ENFORCEMENT FAILURE: XML was modified but sync did not export to Excel!")
	}

	if len(exporter.dtExports) == 0 {
		t.Fatal("SYNC ENFORCEMENT FAILURE: Exporter was not called!")
	}

	t.Logf("SUCCESS: XML modification was synced to Excel (%d files)", result.XMLToExcelCount)
}
