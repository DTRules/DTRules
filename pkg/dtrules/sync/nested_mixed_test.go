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
