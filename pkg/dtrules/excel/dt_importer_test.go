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
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestDTImporter_ParseDT2ExcelFormat tests parsing the dt2excel format.
func TestDTImporter_ParseDT2ExcelFormat(t *testing.T) {
	// Create a test Excel file in dt2excel format
	f := excelize.NewFile()
	sheet := "Test Table"
	f.SetSheetName("Sheet1", sheet)

	// Write test data in dt2excel format
	f.SetCellValue(sheet, "A1", "Decision Table")
	f.SetCellValue(sheet, "B1", "Test_Table")
	f.SetCellValue(sheet, "A2", "Table Number")
	f.SetCellValue(sheet, "B2", "1000")
	f.SetCellValue(sheet, "A3", "Type")
	f.SetCellValue(sheet, "B3", "FIRST")
	f.SetCellValue(sheet, "A4", "Comments")
	f.SetCellValue(sheet, "B4", "Test table for unit testing")

	// Initial actions
	f.SetCellValue(sheet, "A6", "Initial Actions")
	f.SetCellValue(sheet, "A7", "Initialize variables")
	f.SetCellValue(sheet, "B7", "0 /x xdef")

	// Conditions
	f.SetCellValue(sheet, "A9", "Conditions")
	f.SetCellValue(sheet, "A10", "#")
	f.SetCellValue(sheet, "B10", "Description")
	f.SetCellValue(sheet, "C10", "Col 1")
	f.SetCellValue(sheet, "D10", "Col 2")
	f.SetCellValue(sheet, "A11", "1")
	f.SetCellValue(sheet, "B11", "First condition")
	f.SetCellValue(sheet, "C11", "Y")
	f.SetCellValue(sheet, "A12", "2")
	f.SetCellValue(sheet, "B12", "Second condition")
	f.SetCellValue(sheet, "D12", "Y")

	// Actions
	f.SetCellValue(sheet, "A14", "Actions")
	f.SetCellValue(sheet, "A15", "#")
	f.SetCellValue(sheet, "B15", "Description")
	f.SetCellValue(sheet, "C15", "Col 1")
	f.SetCellValue(sheet, "D15", "Col 2")
	f.SetCellValue(sheet, "A16", "1")
	f.SetCellValue(sheet, "B16", "Action one")
	f.SetCellValue(sheet, "C16", "X")
	f.SetCellValue(sheet, "A17", "2")
	f.SetCellValue(sheet, "B17", "Action two")
	f.SetCellValue(sheet, "D17", "X")

	// Policy statements
	f.SetCellValue(sheet, "A19", "Policy Statements")
	f.SetCellValue(sheet, "A20", "Column 1")
	f.SetCellValue(sheet, "B20", "First column policy")
	f.SetCellValue(sheet, "A21", "Column 2")
	f.SetCellValue(sheet, "B21", "Second column policy")

	// Save to temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_dt2excel.xlsx")
	if err := f.SaveAs(testFile); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Import the file
	importer := NewDTImporter()
	tables, err := importer.ImportDecisionTables(testFile)
	if err != nil {
		t.Fatalf("ImportDecisionTables failed: %v", err)
	}

	// Verify results
	if len(tables.Tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables.Tables))
	}

	table := tables.Tables[0]

	if table.TableName != "Test_Table" {
		t.Errorf("Expected table name 'Test_Table', got '%s'", table.TableName)
	}

	if table.AttributeFields.Type != "FIRST" {
		t.Errorf("Expected type 'FIRST', got '%s'", table.AttributeFields.Type)
	}

	if table.AttributeFields.TableNumber != "1000" {
		t.Errorf("Expected table number '1000', got '%s'", table.AttributeFields.TableNumber)
	}

	if len(table.InitialActions) != 1 {
		t.Errorf("Expected 1 initial action, got %d", len(table.InitialActions))
	}

	if len(table.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(table.Conditions))
	} else {
		if len(table.Conditions[0].Columns) != 1 {
			t.Errorf("Expected 1 column for condition 1, got %d", len(table.Conditions[0].Columns))
		}
		if table.Conditions[0].Columns[0].Number != 1 {
			t.Errorf("Expected column number 1, got %d", table.Conditions[0].Columns[0].Number)
		}
		if table.Conditions[0].Columns[0].Value != "Y" {
			t.Errorf("Expected column value 'Y', got '%s'", table.Conditions[0].Columns[0].Value)
		}
	}

	if len(table.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(table.Actions))
	}

	if len(table.PolicyStatements) != 2 {
		t.Errorf("Expected 2 policy statements, got %d", len(table.PolicyStatements))
	}
}

// TestDTImporter_WriteXML tests XML output.
func TestDTImporter_WriteXML(t *testing.T) {
	tables := &DecisionTablesXML{
		Tables: []DecisionTableXML{
			{
				TableName: "Test_Write",
				XLSFile:   "test.xlsx",
				AttributeFields: AttributeFieldsXML{
					Type:        "ALL",
					Comments:    "Test <comment> with special & chars",
					TableNumber: "2000",
				},
				Conditions: []ConditionXML{
					{
						Number:  "1",
						Comment: "Test condition",
						Columns: []ColumnValueXML{
							{Number: 1, Value: "Y"},
						},
					},
				},
				Actions: []ActionXML{
					{
						Number:  "1",
						Comment: "Test action",
						Columns: []ColumnValueXML{
							{Number: 1, Value: "X"},
						},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test_output.xml")

	importer := NewDTImporter()
	if err := importer.WriteXML(tables, outputFile); err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	// Read and verify output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	xmlContent := string(content)

	// Check for proper XML structure
	if !strings.Contains(xmlContent, "<?xml version=") {
		t.Error("Missing XML declaration")
	}

	if !strings.Contains(xmlContent, "<decision_tables>") {
		t.Error("Missing root element")
	}

	if !strings.Contains(xmlContent, "<table_name>Test_Write</table_name>") {
		t.Error("Missing table name")
	}

	// Check XML escaping
	if !strings.Contains(xmlContent, "&lt;comment&gt;") {
		t.Error("XML special characters not escaped properly")
	}

	if !strings.Contains(xmlContent, "&amp;") {
		t.Error("Ampersand not escaped properly")
	}
}

// TestDTImporter_ImportFromDir tests directory import.
func TestDTImporter_ImportFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two test files
	for i := 1; i <= 2; i++ {
		f := excelize.NewFile()
		sheet := "Sheet1"

		f.SetCellValue(sheet, "A1", "Decision Table")
		f.SetCellValue(sheet, "B1", "Table_"+string(rune('0'+i)))
		f.SetCellValue(sheet, "A2", "Table Number")
		f.SetCellValue(sheet, "B2", i*1000)
		f.SetCellValue(sheet, "A3", "Type")
		f.SetCellValue(sheet, "B3", "ALL")

		testFile := filepath.Join(tmpDir, "test_"+string(rune('0'+i))+".xlsx")
		if err := f.SaveAs(testFile); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	importer := NewDTImporter()
	tables, err := importer.ImportDecisionTablesFromDir(tmpDir)
	if err != nil {
		t.Fatalf("ImportDecisionTablesFromDir failed: %v", err)
	}

	if len(tables.Tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(tables.Tables))
	}
}

// TestSortTablesByNumber tests table sorting.
func TestSortTablesByNumber(t *testing.T) {
	tables := &DecisionTablesXML{
		Tables: []DecisionTableXML{
			{TableName: "Third", AttributeFields: AttributeFieldsXML{TableNumber: "3000"}},
			{TableName: "First", AttributeFields: AttributeFieldsXML{TableNumber: "1000"}},
			{TableName: "Second", AttributeFields: AttributeFieldsXML{TableNumber: "2000"}},
		},
	}

	SortTablesByNumber(tables)

	expected := []string{"First", "Second", "Third"}
	for i, name := range expected {
		if tables.Tables[i].TableName != name {
			t.Errorf("Expected table %d to be '%s', got '%s'", i, name, tables.Tables[i].TableName)
		}
	}
}

// TestMergeTables tests table merging.
func TestMergeTables(t *testing.T) {
	t1 := &DecisionTablesXML{
		Tables: []DecisionTableXML{
			{TableName: "Table1"},
		},
	}
	t2 := &DecisionTablesXML{
		Tables: []DecisionTableXML{
			{TableName: "Table2"},
			{TableName: "Table3"},
		},
	}

	merged := MergeTables(t1, t2)

	if len(merged.Tables) != 3 {
		t.Errorf("Expected 3 tables, got %d", len(merged.Tables))
	}
}
