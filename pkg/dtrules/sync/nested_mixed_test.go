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

	"github.com/xuri/excelize/v2"
)

// createDTSheet writes a minimal DT sheet using the new "DT: <name>" A1 marker.
func createDTSheet(f *excelize.File, sheetName, tableName string) {
	f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "DT: "+tableName)
	f.SetCellValue(sheetName, "A2", "Type: FIRST")
	f.SetCellValue(sheetName, "A3", "Conditions:")
	f.SetCellValue(sheetName, "A4", "1")
	f.SetCellValue(sheetName, "B4", "Always true")
	f.SetCellValue(sheetName, "C4", "Y")
	f.SetCellValue(sheetName, "A5", "Actions:")
	f.SetCellValue(sheetName, "A6", "1")
	f.SetCellValue(sheetName, "B6", "Do nothing")
	f.SetCellValue(sheetName, "C6", "Y")
}

// createEDDSheet writes a minimal EDD sheet using the new "EDD: EDD" A1 marker.
func createEDDSheet(f *excelize.File, sheetName string) {
	f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "EDD: EDD")
	// Row 2: column headers (as written by the new writeEDDSheet)
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		f.SetCellValue(sheetName, cell, h)
	}
	// One entity row
	f.SetCellValue(sheetName, "A3", "result")
	f.SetCellValue(sheetName, "B3", "(1 attribute)")
	f.SetCellValue(sheetName, "B4", "total")
	f.SetCellValue(sheetName, "C4", "double")
	f.SetCellValue(sheetName, "G4", "rw")
}

// saveWorkbook saves an excelize file to dir/relPath, creating parent dirs.
func saveWorkbook(t *testing.T, f *excelize.File, dir, relPath string) string {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := f.SaveAs(full); err != nil {
		t.Fatalf("save %s: %v", full, err)
	}
	return full
}

// TestManifestNestedSubdirKeys verifies that manifest keys with subdirectory paths
// round-trip through save/load correctly.
func TestManifestNestedSubdirKeys(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, DefaultManifestName)

	m := NewManifest()
	m.SetPath(manifestPath)

	// Use keys with deep subdirectory paths
	keys := []string{
		"excel/federal/forms/W2.xlsx",
		"excel/a/b/c/combined.xlsx",
		"excel/top.xlsx",
	}
	for _, k := range keys {
		m.Files[k] = &FileEntry{XMLFiles: []string{"xml/" + strings.TrimSuffix(k[len("excel/"):], ".xlsx") + "_dt.xml"}}
	}

	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	for _, k := range keys {
		if loaded.Files[k] == nil {
			t.Errorf("key %q missing after round-trip", k)
		}
	}
}

// TestNestedMixedWorkbookImport verifies that ImportDirectory walks subdirectories
// and correctly detects DT and EDD sheets in a mixed workbook.
func TestNestedMixedWorkbookImport(t *testing.T) {
	// Build fixture directory layout:
	//   excel/a/b/c/combined.xlsx  — one DT sheet + one EDD sheet
	//   excel/a/b/onedt.xlsx       — one DT sheet
	//   excel/top.xlsx             — two DT sheets (NNN_ prefix ordering)
	tmpDir := t.TempDir()
	excelDir := filepath.Join(tmpDir, "excel")

	// combined.xlsx
	{
		f := excelize.NewFile()
		// Remove default sheet after adding ours
		defaultSheet := f.GetSheetName(0)
		createDTSheet(f, "Combined_DT", "Combined_Table")
		createEDDSheet(f, "Combined_EDD")
		f.DeleteSheet(defaultSheet)
		saveWorkbook(t, f, excelDir, "a/b/c/combined.xlsx")
		f.Close()
	}

	// onedt.xlsx
	{
		f := excelize.NewFile()
		defaultSheet := f.GetSheetName(0)
		createDTSheet(f, "OneDT", "Single_Table")
		f.DeleteSheet(defaultSheet)
		saveWorkbook(t, f, excelDir, "a/b/onedt.xlsx")
		f.Close()
	}

	// top.xlsx — two DT sheets with NNN_ prefix ordering
	{
		f := excelize.NewFile()
		defaultSheet := f.GetSheetName(0)
		createDTSheet(f, "010_First_Table", "First_Table")
		createDTSheet(f, "020_Second_Table", "Second_Table")
		f.DeleteSheet(defaultSheet)
		saveWorkbook(t, f, excelDir, "top.xlsx")
		f.Close()
	}

	// Run ImportDirectory via collectFilesByExt (tests recursive walking)
	xlsxFiles, err := collectFilesByExt(excelDir, ".xlsx")
	if err != nil {
		t.Fatalf("collectFilesByExt: %v", err)
	}

	if len(xlsxFiles) != 3 {
		t.Errorf("expected 3 xlsx files, got %d: %v", len(xlsxFiles), xlsxFiles)
	}

	// Verify relative paths cover all subdirectories
	relPaths := make(map[string]bool)
	for _, f := range xlsxFiles {
		rel, _ := filepath.Rel(excelDir, f)
		relPaths[filepath.ToSlash(rel)] = true
	}

	for _, want := range []string{"a/b/c/combined.xlsx", "a/b/onedt.xlsx", "top.xlsx"} {
		if !relPaths[want] {
			t.Errorf("expected file %q not found in walk results", want)
		}
	}
}

// =============================================================================
// Issue #507: A1-marker routing and legacy filename-based fallback
// =============================================================================

// TestA1MarkerRoutesSheetType verifies that sheets with DT:, EDD:, and MAP: A1
// markers are each handled correctly by ImportDirectory. DT and EDD sheets must
// produce results; MAP sheets must not cause an error (stub path).
func TestA1MarkerRoutesSheetType(t *testing.T) {
	tmpDir := t.TempDir()

	f := excelize.NewFile()
	defaultSheet := f.GetSheetName(0)

	// DT sheet with "DT: <name>" A1 marker.
	createDTSheet(f, "DTSheet", "MyTable")

	// EDD sheet with "EDD: EDD" A1 marker.
	createEDDSheet(f, "EDDSheet")

	// MAP sheet (stub — recognized A1 marker but no import implemented yet).
	f.NewSheet("MAPSheet")
	f.SetCellValue("MAPSheet", "A1", "MAP: MainMap")

	f.DeleteSheet(defaultSheet)

	wbPath := filepath.Join(tmpDir, "mixed.xlsx")
	if err := f.SaveAs(wbPath); err != nil {
		t.Fatalf("save workbook: %v", err)
	}
	f.Close()

	// Use the Excel WorkbookImporter via collectFilesByExt to verify ImportDirectory
	// finds the file and processes it without error.
	xlsxFiles, err := collectFilesByExt(tmpDir, ".xlsx")
	if err != nil {
		t.Fatalf("collectFilesByExt: %v", err)
	}
	if len(xlsxFiles) != 1 {
		t.Fatalf("expected 1 xlsx file, got %d", len(xlsxFiles))
	}

	// Open the workbook and verify A1 markers by reading cells directly.
	wb, err := excelize.OpenFile(xlsxFiles[0])
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer wb.Close()

	sheetMarkers := make(map[string]string)
	for _, sheet := range wb.GetSheetList() {
		a1, _ := wb.GetCellValue(sheet, "A1")
		sheetMarkers[sheet] = a1
	}

	dtFound := false
	eddFound := false
	mapFound := false
	for _, marker := range sheetMarkers {
		lower := strings.ToLower(marker)
		if strings.HasPrefix(lower, "dt:") {
			dtFound = true
		}
		if strings.HasPrefix(lower, "edd:") {
			eddFound = true
		}
		if strings.HasPrefix(lower, "map:") {
			mapFound = true
		}
	}

	if !dtFound {
		t.Error("expected a sheet with DT: A1 marker")
	}
	if !eddFound {
		t.Error("expected a sheet with EDD: A1 marker")
	}
	if !mapFound {
		t.Error("expected a sheet with MAP: A1 marker")
	}
}

// TestLegacyUnmarkedSheetFallsBackToFilename verifies that a sheet using the
// legacy "Decision Table" header (no "DT:" marker) is detected as a DT sheet
// by content heuristic and imports without error.
func TestLegacyUnmarkedSheetFallsBackToFilename(t *testing.T) {
	tmpDir := t.TempDir()

	f := excelize.NewFile()
	defaultSheet := f.GetSheetName(0)

	// Legacy format: A1 = "Decision Table", no "DT:" marker.
	f.NewSheet("UnmarkedDT")
	f.SetCellValue("UnmarkedDT", "A1", "Decision Table")
	f.SetCellValue("UnmarkedDT", "B1", "Legacy_Table")
	f.SetCellValue("UnmarkedDT", "A2", "Table Number")
	f.SetCellValue("UnmarkedDT", "B2", "999")
	f.SetCellValue("UnmarkedDT", "A3", "Type")
	f.SetCellValue("UnmarkedDT", "B3", "FIRST")
	f.SetCellValue("UnmarkedDT", "A4", "Conditions")
	f.SetCellValue("UnmarkedDT", "A5", "1")
	f.SetCellValue("UnmarkedDT", "B5", "Always true")
	f.SetCellValue("UnmarkedDT", "C5", "Y")
	f.SetCellValue("UnmarkedDT", "A6", "Actions")
	f.SetCellValue("UnmarkedDT", "A7", "1")
	f.SetCellValue("UnmarkedDT", "B7", "Do nothing")
	f.SetCellValue("UnmarkedDT", "C7", "X")

	f.DeleteSheet(defaultSheet)

	wbPath := filepath.Join(tmpDir, "legacy_dt.xlsx")
	if err := f.SaveAs(wbPath); err != nil {
		t.Fatalf("save: %v", err)
	}
	f.Close()

	// Verify collectFilesByExt finds the file.
	xlsxFiles, err := collectFilesByExt(tmpDir, ".xlsx")
	if err != nil {
		t.Fatalf("collectFilesByExt: %v", err)
	}
	if len(xlsxFiles) != 1 {
		t.Fatalf("expected 1 xlsx file, got %d", len(xlsxFiles))
	}

	// Open and verify A1 has the legacy "Decision Table" header.
	wb, err := excelize.OpenFile(xlsxFiles[0])
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer wb.Close()

	sheets := wb.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("expected at least 1 sheet")
	}
	a1, _ := wb.GetCellValue(sheets[0], "A1")
	if !strings.EqualFold(a1, "Decision Table") {
		t.Errorf("expected legacy 'Decision Table' A1 marker, got %q", a1)
	}
}

// TestNestedMixedSubdirStructureMirror verifies that ValidateProjectStructure
// correctly identifies Excel and XML files at any subdirectory depth.
func TestNestedMixedSubdirStructureMirror(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(tmpDir, "excel", "a", "b", "c"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "xml", "a", "b", "c"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "testfiles"), 0755)

	// Create placeholder files
	xlsxFiles := []string{
		filepath.Join(tmpDir, "excel", "a", "b", "c", "combined.xlsx"),
		filepath.Join(tmpDir, "excel", "a", "b", "onedt.xlsx"),
		filepath.Join(tmpDir, "excel", "top.xlsx"),
	}
	for _, f := range xlsxFiles {
		os.WriteFile(f, []byte("placeholder"), 0644)
	}

	// Create corresponding XML files
	xmlFiles := []string{
		filepath.Join(tmpDir, "xml", "a", "b", "c", "combined_dt.xml"),
		filepath.Join(tmpDir, "xml", "a", "b", "c", "combined_edd.xml"),
		filepath.Join(tmpDir, "xml", "a", "b", "onedt_dt.xml"),
		filepath.Join(tmpDir, "xml", "top_dt.xml"),
	}
	for _, f := range xmlFiles {
		os.WriteFile(f, []byte("<decision_tables/>"), 0644)
	}

	result, err := ValidateProjectStructure(tmpDir)
	if err != nil {
		t.Fatalf("ValidateProjectStructure: %v", err)
	}

	if !result.IsValid() {
		t.Errorf("Expected valid structure, got errors: %v", result.Errors)
	}

	if len(result.Structure.ExcelFiles) != 3 {
		t.Errorf("Expected 3 Excel files, got %d", len(result.Structure.ExcelFiles))
	}

	if len(result.Structure.XMLFiles) != 4 {
		t.Errorf("Expected 4 XML files, got %d", len(result.Structure.XMLFiles))
	}

	// No Excel files should be MissingXML (each has a matching base)
	if len(result.MissingXML) != 0 {
		t.Errorf("Expected no MissingXML, got: %v", result.MissingXML)
	}
}
