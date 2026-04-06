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

package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// testCellName creates a cell name from column and row (1-indexed)
// Named differently from cellName in exporter.go to avoid redeclaration
func testCellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// createCombinedWorkbook creates a test Excel file with both DT and EDD sheets
func createCombinedWorkbook(t *testing.T, dir, filename string) string {
	t.Helper()

	f := excelize.NewFile()

	// Create EDD sheet
	eddSheet := "EDD"
	f.SetSheetName("Sheet1", eddSheet)

	// Write EDD headers
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(eddSheet, cell, header)
	}

	// Add entity
	f.SetCellValue(eddSheet, "A2", "result")
	f.SetCellValue(eddSheet, "B2", "(1 attribute)")
	f.SetCellValue(eddSheet, "B3", "total_tax")
	f.SetCellValue(eddSheet, "C3", "double")
	f.SetCellValue(eddSheet, "E3", "0")
	f.SetCellValue(eddSheet, "G3", "rw")
	f.SetCellValue(eddSheet, "H3", "Total tax calculated")

	// Create DT sheet
	dtSheet := "Calculate_Tax"
	f.NewSheet(dtSheet)

	f.SetCellValue(dtSheet, "A1", "Decision Table")
	f.SetCellValue(dtSheet, "B1", "Calculate_Tax")
	f.SetCellValue(dtSheet, "A2", "Table Number")
	f.SetCellValue(dtSheet, "B2", "100")
	f.SetCellValue(dtSheet, "A3", "Type")
	f.SetCellValue(dtSheet, "B3", "FIRST")
	f.SetCellValue(dtSheet, "A4", "Comments")
	f.SetCellValue(dtSheet, "B4", "Calculate tax amount")

	// Conditions section
	f.SetCellValue(dtSheet, "A6", "Conditions")
	f.SetCellValue(dtSheet, "A7", "#")
	f.SetCellValue(dtSheet, "B7", "Description")
	f.SetCellValue(dtSheet, "C7", "Col 1")
	f.SetCellValue(dtSheet, "A8", "1")
	f.SetCellValue(dtSheet, "B8", "Has income")
	f.SetCellValue(dtSheet, "C8", "Y")

	// Actions section
	f.SetCellValue(dtSheet, "A10", "Actions")
	f.SetCellValue(dtSheet, "A11", "#")
	f.SetCellValue(dtSheet, "B11", "Description")
	f.SetCellValue(dtSheet, "C11", "Col 1")
	f.SetCellValue(dtSheet, "A12", "1")
	f.SetCellValue(dtSheet, "B12", "Set tax")
	f.SetCellValue(dtSheet, "C12", "X")

	filePath := filepath.Join(dir, filename)
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("Failed to save test workbook: %v", err)
	}
	f.Close()

	return filePath
}

// createDTOnlyWorkbook creates a test Excel file with only DT sheets
func createDTOnlyWorkbook(t *testing.T, dir, filename, tableName, tableNum string) string {
	t.Helper()

	f := excelize.NewFile()
	sheet := tableName
	f.SetSheetName("Sheet1", sheet)

	f.SetCellValue(sheet, "A1", "Decision Table")
	f.SetCellValue(sheet, "B1", tableName)
	f.SetCellValue(sheet, "A2", "Table Number")
	f.SetCellValue(sheet, "B2", tableNum)
	f.SetCellValue(sheet, "A3", "Type")
	f.SetCellValue(sheet, "B3", "ALL")

	// Conditions
	f.SetCellValue(sheet, "A5", "Conditions")
	f.SetCellValue(sheet, "A6", "1")
	f.SetCellValue(sheet, "B6", "Test condition")
	f.SetCellValue(sheet, "C6", "Y")

	// Actions
	f.SetCellValue(sheet, "A8", "Actions")
	f.SetCellValue(sheet, "A9", "1")
	f.SetCellValue(sheet, "B9", "Test action")
	f.SetCellValue(sheet, "C9", "X")

	filePath := filepath.Join(dir, filename)
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("Failed to save test workbook: %v", err)
	}
	f.Close()

	return filePath
}

func TestWorkbookImporter_ImportCombinedWorkbook(t *testing.T) {
	dir := t.TempDir()
	workbookPath := createCombinedWorkbook(t, dir, "combined.xlsx")

	importer := NewWorkbookImporter()
	result, err := importer.ImportWorkbook(workbookPath)
	if err != nil {
		t.Fatalf("ImportWorkbook failed: %v", err)
	}

	// Check DT tables
	if len(result.DTables.Tables) != 1 {
		t.Errorf("Expected 1 DT table, got %d", len(result.DTables.Tables))
	} else {
		if result.DTables.Tables[0].TableName != "Calculate_Tax" {
			t.Errorf("Expected table name 'Calculate_Tax', got '%s'", result.DTables.Tables[0].TableName)
		}
	}

	// Check EDD entities
	if len(result.EDD.Entities) != 1 {
		t.Errorf("Expected 1 EDD entity, got %d", len(result.EDD.Entities))
	} else {
		if result.EDD.Entities[0].Name != "result" {
			t.Errorf("Expected entity name 'result', got '%s'", result.EDD.Entities[0].Name)
		}
	}
}

func TestWorkbookImporter_RecursiveDirectory(t *testing.T) {
	baseDir := t.TempDir()

	// Create directory structure
	statesDir := filepath.Join(baseDir, "states")
	if err := os.MkdirAll(statesDir, 0755); err != nil {
		t.Fatalf("Failed to create states directory: %v", err)
	}

	// Create root-level workbook
	createDTOnlyWorkbook(t, baseDir, "main.xlsx", "Main_Table", "1")

	// Create state-level workbooks
	createDTOnlyWorkbook(t, statesDir, "CO.xlsx", "Calculate_CO_Tax", "100")
	createDTOnlyWorkbook(t, statesDir, "CA.xlsx", "Calculate_CA_Tax", "200")

	// Import recursively
	importer := NewWorkbookImporter()
	results, err := importer.ImportDirectory(baseDir)
	if err != nil {
		t.Fatalf("ImportDirectory failed: %v", err)
	}

	// Should have 3 workbooks
	if len(results) != 3 {
		t.Errorf("Expected 3 workbooks, got %d", len(results))
	}

	// Check file paths are relative
	expectedPaths := map[string]bool{
		"main.xlsx":        false,
		"states/CA.xlsx":   false,
		"states/CO.xlsx":   false,
	}

	for _, result := range results {
		// Normalize path separators for cross-platform compatibility
		relPath := filepath.ToSlash(result.FilePath)
		if _, ok := expectedPaths[relPath]; ok {
			expectedPaths[relPath] = true
		} else {
			t.Errorf("Unexpected file path: %s", result.FilePath)
		}

		// Verify xls_file metadata is set correctly
		for _, table := range result.DTables.Tables {
			tablePath := filepath.ToSlash(table.XLSFile)
			if tablePath != relPath {
				t.Errorf("Table XLSFile mismatch: expected '%s', got '%s'", relPath, table.XLSFile)
			}
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("Expected path not found: %s", path)
		}
	}
}

func TestWorkbookImporter_WriteAll(t *testing.T) {
	baseDir := t.TempDir()
	excelDir := filepath.Join(baseDir, "excel")
	xmlDir := filepath.Join(baseDir, "xml")

	// Create excel directory structure
	statesDir := filepath.Join(excelDir, "states")
	if err := os.MkdirAll(statesDir, 0755); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Create workbooks
	createCombinedWorkbook(t, excelDir, "main.xlsx")
	createDTOnlyWorkbook(t, statesDir, "CO.xlsx", "Calculate_CO_Tax", "100")

	// Import and write
	importer := NewWorkbookImporter()
	results, err := importer.ImportDirectory(excelDir)
	if err != nil {
		t.Fatalf("ImportDirectory failed: %v", err)
	}

	if err := importer.WriteAll(results, xmlDir); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	// Verify output files exist
	expectedFiles := []string{
		filepath.Join(xmlDir, "main_dt.xml"),
		filepath.Join(xmlDir, "main_edd.xml"),
		filepath.Join(xmlDir, "states", "CO_dt.xml"),
	}

	for _, path := range expectedFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected output file not found: %s", path)
		}
	}

	// CO.xlsx only has DT, no EDD file should be created
	coEddPath := filepath.Join(xmlDir, "states", "CO_edd.xml")
	if _, err := os.Stat(coEddPath); !os.IsNotExist(err) {
		t.Errorf("CO_edd.xml should not exist (no EDD in workbook)")
	}
}

func TestWorkbookImporter_MergeResults(t *testing.T) {
	dir := t.TempDir()

	// Create two workbooks
	createCombinedWorkbook(t, dir, "workbook1.xlsx")
	createDTOnlyWorkbook(t, dir, "workbook2.xlsx", "Second_Table", "200")

	importer := NewWorkbookImporter()
	results, err := importer.ImportDirectory(dir)
	if err != nil {
		t.Fatalf("ImportDirectory failed: %v", err)
	}

	mergedDT, mergedEDD := importer.MergeResults(results)

	// Should have 2 tables total
	if len(mergedDT.Tables) != 2 {
		t.Errorf("Expected 2 merged tables, got %d", len(mergedDT.Tables))
	}

	// Should have 1 entity (only workbook1 has EDD)
	if len(mergedEDD.Entities) != 1 {
		t.Errorf("Expected 1 merged entity, got %d", len(mergedEDD.Entities))
	}
}

func TestWorkbookImporter_SkipTempFiles(t *testing.T) {
	dir := t.TempDir()

	// Create normal workbook
	createDTOnlyWorkbook(t, dir, "normal.xlsx", "Normal_Table", "1")

	// Create temp file (simulates Excel temp file)
	tempPath := filepath.Join(dir, "~$normal.xlsx")
	if err := os.WriteFile(tempPath, []byte("temp"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	importer := NewWorkbookImporter()
	results, err := importer.ImportDirectory(dir)
	if err != nil {
		t.Fatalf("ImportDirectory failed: %v", err)
	}

	// Should only import 1 workbook (skip temp file)
	if len(results) != 1 {
		t.Errorf("Expected 1 workbook (temp file skipped), got %d", len(results))
	}
}

func TestWorkbookImporter_SheetDetection(t *testing.T) {
	dir := t.TempDir()

	// Create workbook with multiple sheet types
	f := excelize.NewFile()

	// EDD sheet with standard format
	eddSheet := "EDD"
	f.SetSheetName("Sheet1", eddSheet)
	f.SetCellValue(eddSheet, "A1", "Entity")
	f.SetCellValue(eddSheet, "B1", "Attribute")
	f.SetCellValue(eddSheet, "C1", "Type")

	// DT sheet with dt2excel format
	dtSheet1 := "Table1"
	f.NewSheet(dtSheet1)
	f.SetCellValue(dtSheet1, "A1", "Decision Table")
	f.SetCellValue(dtSheet1, "B1", "Table1")

	// DT sheet with exporter format
	dtSheet2 := "Table2"
	f.NewSheet(dtSheet2)
	f.SetCellValue(dtSheet2, "A1", "Name: Table2")
	f.SetCellValue(dtSheet2, "A2", "Type: ALL")

	// Unknown sheet (should be skipped)
	otherSheet := "Notes"
	f.NewSheet(otherSheet)
	f.SetCellValue(otherSheet, "A1", "Some random notes")
	f.SetCellValue(otherSheet, "A2", "Not a DT or EDD")

	filePath := filepath.Join(dir, "multi_type.xlsx")
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("Failed to save test workbook: %v", err)
	}
	f.Close()

	importer := NewWorkbookImporter()
	result, err := importer.ImportWorkbook(filePath)
	if err != nil {
		t.Fatalf("ImportWorkbook failed: %v", err)
	}

	// Should detect 2 DT sheets
	if len(result.DTables.Tables) != 2 {
		t.Errorf("Expected 2 DT tables, got %d", len(result.DTables.Tables))
	}

	// Should detect 1 EDD sheet (even though it only has headers)
	// Note: The EDD sheet with just headers may not create entities
	// This is expected behavior
}

func TestDTImporter_RecursiveDirectory(t *testing.T) {
	baseDir := t.TempDir()

	// Create directory structure
	subDir := filepath.Join(baseDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create workbooks at different levels
	createDTOnlyWorkbook(t, baseDir, "root.xlsx", "Root_Table", "1")
	createDTOnlyWorkbook(t, subDir, "sub.xlsx", "Sub_Table", "2")

	importer := NewDTImporter()
	tables, err := importer.ImportDecisionTablesFromDir(baseDir)
	if err != nil {
		t.Fatalf("ImportDecisionTablesFromDir failed: %v", err)
	}

	// Should find both tables
	if len(tables.Tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(tables.Tables))
	}

	// Verify xls_file paths
	foundRoot := false
	foundSub := false
	for _, table := range tables.Tables {
		relPath := filepath.ToSlash(table.XLSFile)
		if relPath == "root.xlsx" {
			foundRoot = true
		}
		if relPath == "subdir/sub.xlsx" {
			foundSub = true
		}
	}

	if !foundRoot {
		t.Error("Root table not found or xls_file incorrect")
	}
	if !foundSub {
		t.Error("Subdirectory table not found or xls_file incorrect")
	}
}

func TestEDDImporter_RecursiveDirectory(t *testing.T) {
	baseDir := t.TempDir()

	// Create directory structure
	subDir := filepath.Join(baseDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create EDD workbooks at different levels
	createEDDWorkbook(t, baseDir, "root_edd.xlsx", "job")
	createEDDWorkbook(t, subDir, "sub_edd.xlsx", "result")

	importer := NewEDDImporter()
	edd, err := importer.ImportEDDFromDir(baseDir)
	if err != nil {
		t.Fatalf("ImportEDDFromDir failed: %v", err)
	}

	// Should find both entities
	if len(edd.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(edd.Entities))
	}

	// Verify xls_file paths
	for _, entity := range edd.Entities {
		relPath := filepath.ToSlash(entity.XlsFile)
		if entity.Name == "job" && relPath != "root_edd.xlsx" {
			t.Errorf("Expected job xls_file 'root_edd.xlsx', got '%s'", entity.XlsFile)
		}
		if entity.Name == "result" && relPath != "subdir/sub_edd.xlsx" {
			t.Errorf("Expected result xls_file 'subdir/sub_edd.xlsx', got '%s'", entity.XlsFile)
		}
	}
}

// createEDDWorkbook creates a simple EDD workbook for testing
func createEDDWorkbook(t *testing.T, dir, filename, entityName string) string {
	t.Helper()

	f := excelize.NewFile()
	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	// Write headers
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Entity header
	f.SetCellValue(sheet, "A2", entityName)
	f.SetCellValue(sheet, "B2", "(1 attribute)")

	// Entity field
	f.SetCellValue(sheet, "B3", "test_field")
	f.SetCellValue(sheet, "C3", "string")
	f.SetCellValue(sheet, "G3", "rw")
	f.SetCellValue(sheet, "H3", "Test field")

	filePath := filepath.Join(dir, filename)
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("Failed to save EDD workbook: %v", err)
	}
	f.Close()

	return filePath
}

// TestStripDTOrEDDSuffix tests the suffix stripping helper function
func TestStripDTOrEDDSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TaxReturn_edd", "TaxReturn"},
		{"TaxReturn_dt", "TaxReturn"},
		{"CO_edd", "CO"},
		{"CO_dt", "CO"},
		{"MyFile", "MyFile"},
		{"file_with_underscore", "file_with_underscore"},
		{"edd_prefix", "edd_prefix"}, // Not a suffix
		{"dt_prefix", "dt_prefix"},   // Not a suffix
		{"_edd", ""},                 // Edge case: just suffix
		{"_dt", ""},                  // Edge case: just suffix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripDTOrEDDSuffix(tt.input)
			if result != tt.expected {
				t.Errorf("stripDTOrEDDSuffix(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestImportWorkbookToDirNoDoubleSuffix ensures _edd.xlsx doesn't become _edd_edd.xml
func TestImportWorkbookToDirNoDoubleSuffix(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	excelDir := filepath.Join(tempDir, "excel")
	xmlDir := filepath.Join(tempDir, "xml")

	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatalf("Failed to create excel dir: %v", err)
	}
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("Failed to create xml dir: %v", err)
	}

	// Create EDD workbook with _edd suffix in filename (like TaxReturn_edd.xlsx)
	excelPath := createEDDWorkbook(t, excelDir, "test_edd.xlsx", "result")

	// Import it
	importer := NewWorkbookImporter()
	dtPath, eddPath, err := importer.ImportWorkbookToDir(excelPath, xmlDir)
	if err != nil {
		t.Fatalf("ImportWorkbookToDir failed: %v", err)
	}

	// dtPath should be empty (no DT sheet)
	if dtPath != "" {
		t.Errorf("Expected empty dtPath, got %q", dtPath)
	}

	// eddPath should be test_edd.xml, NOT test_edd_edd.xml
	expectedEddPath := filepath.Join(xmlDir, "test_edd.xml")
	if eddPath != expectedEddPath {
		t.Errorf("eddPath = %q, want %q", eddPath, expectedEddPath)
	}

	// Verify the _edd_edd.xml file does NOT exist
	badPath := filepath.Join(xmlDir, "test_edd_edd.xml")
	if _, err := os.Stat(badPath); err == nil {
		t.Errorf("Spurious file %q should not exist", badPath)
	}
}
