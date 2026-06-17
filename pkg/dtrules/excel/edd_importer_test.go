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
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createTestEDDExcel creates a test Excel file with EDD data
func createTestEDDExcel(t *testing.T, dir string) string {
	t.Helper()

	f := excelize.NewFile()
	defer f.Close()

	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	// Write headers
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	row := 2

	// Entity: job
	f.SetCellValue(sheet, cellName(1, row), "job")
	f.SetCellValue(sheet, cellName(2, row), "(3 attributes)")
	row++

	// job attributes
	f.SetCellValue(sheet, cellName(2, row), "id")
	f.SetCellValue(sheet, cellName(3, row), "integer")
	f.SetCellValue(sheet, cellName(4, row), "")
	f.SetCellValue(sheet, cellName(5, row), "")
	f.SetCellValue(sheet, cellName(6, row), "main")
	f.SetCellValue(sheet, cellName(7, row), "r")
	f.SetCellValue(sheet, cellName(8, row), "Job ID")
	row++

	f.SetCellValue(sheet, cellName(2, row), "tax_year")
	f.SetCellValue(sheet, cellName(3, row), "integer")
	f.SetCellValue(sheet, cellName(4, row), "")
	f.SetCellValue(sheet, cellName(5, row), "2025")
	f.SetCellValue(sheet, cellName(6, row), "main")
	f.SetCellValue(sheet, cellName(7, row), "r")
	f.SetCellValue(sheet, cellName(8, row), "Tax year being processed")
	row++

	f.SetCellValue(sheet, cellName(2, row), "taxpayers")
	f.SetCellValue(sheet, cellName(3, row), "array")
	f.SetCellValue(sheet, cellName(4, row), "taxpayer")
	f.SetCellValue(sheet, cellName(5, row), "")
	f.SetCellValue(sheet, cellName(6, row), "")
	f.SetCellValue(sheet, cellName(7, row), "r")
	f.SetCellValue(sheet, cellName(8, row), "Taxpayers on this return")
	row++

	// Entity: taxpayer
	f.SetCellValue(sheet, cellName(1, row), "taxpayer")
	f.SetCellValue(sheet, cellName(2, row), "(2 attributes)")
	row++

	f.SetCellValue(sheet, cellName(2, row), "name")
	f.SetCellValue(sheet, cellName(3, row), "string")
	f.SetCellValue(sheet, cellName(4, row), "")
	f.SetCellValue(sheet, cellName(5, row), "")
	f.SetCellValue(sheet, cellName(6, row), "main")
	f.SetCellValue(sheet, cellName(7, row), "r")
	f.SetCellValue(sheet, cellName(8, row), "Full name")
	row++

	f.SetCellValue(sheet, cellName(2, row), "is_primary")
	f.SetCellValue(sheet, cellName(3, row), "boolean")
	f.SetCellValue(sheet, cellName(4, row), "")
	f.SetCellValue(sheet, cellName(5, row), "false")
	f.SetCellValue(sheet, cellName(6, row), "main")
	f.SetCellValue(sheet, cellName(7, row), "r")
	f.SetCellValue(sheet, cellName(8, row), "Primary taxpayer on return")
	row++

	filename := filepath.Join(dir, "test_edd.xlsx")
	if err := f.SaveAs(filename); err != nil {
		t.Fatalf("Failed to save test Excel file: %v", err)
	}

	return filename
}

func TestImportEDD(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	// Create test Excel file
	excelFile := createTestEDDExcel(t, dir)

	// Import
	importer := NewEDDImporter()
	edd, err := importer.ImportEDD(excelFile)
	if err != nil {
		t.Fatalf("ImportEDD failed: %v", err)
	}

	// Validate
	if len(edd.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(edd.Entities))
	}

	// Check job entity
	var jobEntity *EDDXMLEntity
	for _, e := range edd.Entities {
		if e.Name == "job" {
			jobEntity = e
			break
		}
	}
	if jobEntity == nil {
		t.Fatal("job entity not found")
	}
	if len(jobEntity.Fields) != 3 {
		t.Errorf("Expected 3 fields in job entity, got %d", len(jobEntity.Fields))
	}

	// Check taxpayer entity
	var taxpayerEntity *EDDXMLEntity
	for _, e := range edd.Entities {
		if e.Name == "taxpayer" {
			taxpayerEntity = e
			break
		}
	}
	if taxpayerEntity == nil {
		t.Fatal("taxpayer entity not found")
	}
	if len(taxpayerEntity.Fields) != 2 {
		t.Errorf("Expected 2 fields in taxpayer entity, got %d", len(taxpayerEntity.Fields))
	}

	// Check field details
	for _, field := range jobEntity.Fields {
		if field.Name == "tax_year" {
			if field.Type != "integer" {
				t.Errorf("Expected tax_year type 'integer', got '%s'", field.Type)
			}
			if field.DefaultValue != "2025" {
				t.Errorf("Expected tax_year default '2025', got '%s'", field.DefaultValue)
			}
		}
		if field.Name == "taxpayers" {
			if field.Type != "array" {
				t.Errorf("Expected taxpayers type 'array', got '%s'", field.Type)
			}
			if field.SubType != "taxpayer" {
				t.Errorf("Expected taxpayers subtype 'taxpayer', got '%s'", field.SubType)
			}
		}
	}
}

func TestWriteXML(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	// Create test EDD
	edd := &EDDXML{
		Version: "2",
		Entities: []*EDDXMLEntity{
			{
				Name:   "result",
				Access: "rw",
				Fields: []*EDDXMLField{
					{
						Name:         "total_tax",
						Type:         "double",
						SubType:      "",
						DefaultValue: "0",
						Input:        "",
						Access:       "rw",
						Comment:      "Total tax calculated",
					},
				},
			},
		},
	}

	// Write XML
	importer := NewEDDImporter()
	outputFile := filepath.Join(dir, "output_edd.xml")
	if err := importer.WriteXML(edd, outputFile); err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	// Read and validate
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(data)

	// Check XML header
	if !strings.HasPrefix(content, "<?xml version=") {
		t.Error("XML header missing")
	}

	// Check structure
	if !strings.Contains(content, "<entity_data_dictionary") {
		t.Error("entity_data_dictionary element missing")
	}
	if !strings.Contains(content, `version="2"`) {
		t.Error("version attribute missing")
	}
	if !strings.Contains(content, `name="result"`) {
		t.Error("entity name missing")
	}
	if !strings.Contains(content, `name="total_tax"`) {
		t.Error("field name missing")
	}
}

func TestImportEDDFromDir(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	// Create first test Excel file
	f1 := excelize.NewFile()
	sheet := "EDD"
	f1.SetSheetName("Sheet1", sheet)

	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f1.SetCellValue(sheet, cell, header)
	}

	f1.SetCellValue(sheet, cellName(1, 2), "job")
	f1.SetCellValue(sheet, cellName(2, 2), "(1 attribute)")
	f1.SetCellValue(sheet, cellName(2, 3), "id")
	f1.SetCellValue(sheet, cellName(3, 3), "integer")
	f1.SetCellValue(sheet, cellName(7, 3), "r")
	f1.SetCellValue(sheet, cellName(8, 3), "Job ID")

	if err := f1.SaveAs(filepath.Join(dir, "file1.xlsx")); err != nil {
		t.Fatalf("Failed to save file1: %v", err)
	}
	f1.Close()

	// Create second test Excel file
	f2 := excelize.NewFile()
	f2.SetSheetName("Sheet1", sheet)

	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f2.SetCellValue(sheet, cell, header)
	}

	f2.SetCellValue(sheet, cellName(1, 2), "result")
	f2.SetCellValue(sheet, cellName(2, 2), "(1 attribute)")
	f2.SetCellValue(sheet, cellName(2, 3), "total")
	f2.SetCellValue(sheet, cellName(3, 3), "double")
	f2.SetCellValue(sheet, cellName(7, 3), "rw")
	f2.SetCellValue(sheet, cellName(8, 3), "Total amount")

	if err := f2.SaveAs(filepath.Join(dir, "file2.xlsx")); err != nil {
		t.Fatalf("Failed to save file2: %v", err)
	}
	f2.Close()

	// Import from directory
	importer := NewEDDImporter()
	edd, err := importer.ImportEDDFromDir(dir)
	if err != nil {
		t.Fatalf("ImportEDDFromDir failed: %v", err)
	}

	// Validate - should have 2 entities
	if len(edd.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(edd.Entities))
	}

	// Check that xls_file is set
	for _, e := range edd.Entities {
		if e.XlsFile == "" {
			t.Errorf("Entity %s missing xls_file", e.Name)
		}
	}
}

func TestMergeEDD(t *testing.T) {
	edd1 := &EDDXML{
		Version: "2",
		Entities: []*EDDXMLEntity{
			{
				Name:   "job",
				Access: "rw",
				Fields: []*EDDXMLField{
					{Name: "id", Type: "integer", Access: "r"},
				},
			},
		},
	}

	edd2 := &EDDXML{
		Version: "2",
		Entities: []*EDDXMLEntity{
			{
				Name:   "job",
				Access: "rw",
				Fields: []*EDDXMLField{
					{Name: "tax_year", Type: "integer", Access: "r"},
				},
			},
			{
				Name:   "result",
				Access: "rw",
				Fields: []*EDDXMLField{
					{Name: "total", Type: "double", Access: "rw"},
				},
			},
		},
	}

	merged := MergeEDD(edd1, edd2)

	// Should have 2 entities (job merged, result added)
	if len(merged.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(merged.Entities))
	}

	// Find job entity and check it has both fields
	for _, e := range merged.Entities {
		if e.Name == "job" {
			if len(e.Fields) != 2 {
				t.Errorf("Expected job entity to have 2 fields, got %d", len(e.Fields))
			}
		}
	}
}

func TestIsEntityHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"(5 attributes)", true},
		{"(1 attribute)", true},
		{"(12 attributes)", true},
		{"(0 attributes)", true},
		{"some field name", false},
		{"(attributes)", false},
		{"5 attributes", false},
		{"", false},
	}

	for _, tc := range tests {
		result := isEntityHeader(tc.input)
		if result != tc.expected {
			t.Errorf("isEntityHeader(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestImportMultiSheetFormat(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	// Create test Excel file in multi-sheet format (one sheet per entity)
	f := excelize.NewFile()

	// First entity: job
	sheet1 := "job"
	f.SetSheetName("Sheet1", sheet1)

	f.SetCellValue(sheet1, "A1", "Entity")
	f.SetCellValue(sheet1, "B1", "job")
	f.SetCellValue(sheet1, "A2", "Access")
	f.SetCellValue(sheet1, "B2", "rw")
	f.SetCellValue(sheet1, "A3", "Comment")
	f.SetCellValue(sheet1, "B3", "Root entity for processing")

	// Headers in row 5
	f.SetCellValue(sheet1, "A5", "Field Name")
	f.SetCellValue(sheet1, "B5", "Type")
	f.SetCellValue(sheet1, "C5", "Subtype")
	f.SetCellValue(sheet1, "D5", "Access")
	f.SetCellValue(sheet1, "E5", "Input")
	f.SetCellValue(sheet1, "F5", "Default")
	f.SetCellValue(sheet1, "G5", "Comment")

	// Field data starting row 6
	f.SetCellValue(sheet1, "A6", "id")
	f.SetCellValue(sheet1, "B6", "integer")
	f.SetCellValue(sheet1, "C6", "")
	f.SetCellValue(sheet1, "D6", "r")
	f.SetCellValue(sheet1, "E6", "main")
	f.SetCellValue(sheet1, "F6", "")
	f.SetCellValue(sheet1, "G6", "Job ID")

	f.SetCellValue(sheet1, "A7", "tax_year")
	f.SetCellValue(sheet1, "B7", "integer")
	f.SetCellValue(sheet1, "C7", "")
	f.SetCellValue(sheet1, "D7", "r")
	f.SetCellValue(sheet1, "E7", "main")
	f.SetCellValue(sheet1, "F7", "2025")
	f.SetCellValue(sheet1, "G7", "Tax year")

	// Second entity: result
	sheet2 := "result"
	f.NewSheet(sheet2)

	f.SetCellValue(sheet2, "A1", "Entity")
	f.SetCellValue(sheet2, "B1", "result")
	f.SetCellValue(sheet2, "A2", "Access")
	f.SetCellValue(sheet2, "B2", "rw")
	f.SetCellValue(sheet2, "A3", "Comment")
	f.SetCellValue(sheet2, "B3", "Result entity")

	f.SetCellValue(sheet2, "A5", "Field Name")
	f.SetCellValue(sheet2, "B5", "Type")
	f.SetCellValue(sheet2, "C5", "Subtype")
	f.SetCellValue(sheet2, "D5", "Access")
	f.SetCellValue(sheet2, "E5", "Input")
	f.SetCellValue(sheet2, "F5", "Default")
	f.SetCellValue(sheet2, "G5", "Comment")

	f.SetCellValue(sheet2, "A6", "total_tax")
	f.SetCellValue(sheet2, "B6", "double")
	f.SetCellValue(sheet2, "C6", "")
	f.SetCellValue(sheet2, "D6", "rw")
	f.SetCellValue(sheet2, "E6", "")
	f.SetCellValue(sheet2, "F6", "0")
	f.SetCellValue(sheet2, "G6", "Total tax calculated")

	filename := filepath.Join(dir, "multi_sheet_edd.xlsx")
	if err := f.SaveAs(filename); err != nil {
		t.Fatalf("Failed to save test Excel file: %v", err)
	}
	f.Close()

	// Import
	importer := NewEDDImporter()
	edd, err := importer.ImportEDD(filename)
	if err != nil {
		t.Fatalf("ImportEDD failed: %v", err)
	}

	// Validate
	if len(edd.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(edd.Entities))
	}

	// Check job entity
	var jobEntity *EDDXMLEntity
	for _, e := range edd.Entities {
		if e.Name == "job" {
			jobEntity = e
			break
		}
	}
	if jobEntity == nil {
		t.Fatal("job entity not found")
	}
	if len(jobEntity.Fields) != 2 {
		t.Errorf("Expected 2 fields in job entity, got %d", len(jobEntity.Fields))
	}
	if jobEntity.Comment != "Root entity for processing" {
		t.Errorf("Expected job comment 'Root entity for processing', got '%s'", jobEntity.Comment)
	}

	// Check field details
	for _, field := range jobEntity.Fields {
		if field.Name == "tax_year" {
			if field.Type != "integer" {
				t.Errorf("Expected tax_year type 'integer', got '%s'", field.Type)
			}
			if field.DefaultValue != "2025" {
				t.Errorf("Expected tax_year default '2025', got '%s'", field.DefaultValue)
			}
		}
	}

	// Check result entity
	var resultEntity *EDDXMLEntity
	for _, e := range edd.Entities {
		if e.Name == "result" {
			resultEntity = e
			break
		}
	}
	if resultEntity == nil {
		t.Fatal("result entity not found")
	}
	if len(resultEntity.Fields) != 1 {
		t.Errorf("Expected 1 field in result entity, got %d", len(resultEntity.Fields))
	}
}

func TestRoundTrip(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	// Create test Excel file
	excelFile := createTestEDDExcel(t, dir)

	// Import
	importer := NewEDDImporter()
	edd, err := importer.ImportEDD(excelFile)
	if err != nil {
		t.Fatalf("ImportEDD failed: %v", err)
	}

	// Write XML
	xmlFile := filepath.Join(dir, "roundtrip.xml")
	if err := importer.WriteXML(edd, xmlFile); err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	// Read XML back and validate
	data, err := os.ReadFile(xmlFile)
	if err != nil {
		t.Fatalf("Failed to read XML file: %v", err)
	}

	var parsedEDD EDDXML
	if err := xml.Unmarshal(data, &parsedEDD); err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Validate
	if len(parsedEDD.Entities) != len(edd.Entities) {
		t.Errorf("Entity count mismatch: got %d, expected %d", len(parsedEDD.Entities), len(edd.Entities))
	}

	for i, entity := range parsedEDD.Entities {
		if entity.Name != edd.Entities[i].Name {
			t.Errorf("Entity name mismatch at %d: got %s, expected %s", i, entity.Name, edd.Entities[i].Name)
		}
		if len(entity.Fields) != len(edd.Entities[i].Fields) {
			t.Errorf("Field count mismatch for %s: got %d, expected %d", entity.Name, len(entity.Fields), len(edd.Entities[i].Fields))
		}
	}
}

// TestEDDRoundTrip_CollectQuestion checks that the #850 collect flag and
// question metadata (text/type/options) survive an EDD Excel round-trip:
// EDDXML -> xlsx (writeEDDXMLEntities) -> EDDXML (ImportEDD / parseEDDSheet).
func TestEDDRoundTrip_CollectQuestion(t *testing.T) {
	edd := &EDDXML{Version: "2", Entities: []*EDDXMLEntity{{
		Name:   "patient",
		Access: "rw",
		Fields: []*EDDXMLField{
			{
				Name: "penicillin_allergic", Type: "boolean", Access: "rw", DefaultValue: "false",
				Collect: "true",
				Question: &EDDXMLQuestion{Text: "Penicillin-allergic?", Type: "multiple_choice",
					Options: []*EDDXMLOption{{Value: "true", Label: "Yes"}, {Value: "false", Label: "No"}}},
			},
			{Name: "age", Type: "integer", Access: "rw", DefaultValue: "0",
				Collect: "true", Question: &EDDXMLQuestion{Text: "Age?", Type: "number",
					RefLow: "0", RefHigh: "120", Units: "years"}},
			{Name: "notes", Type: "string", Access: "rw"}, // not collected
		},
	}}}

	file := filepath.Join(t.TempDir(), "rt.xlsx")
	if err := WriteEDDXMLToExcel(edd, file); err != nil {
		t.Fatalf("WriteEDDXMLToExcel: %v", err)
	}
	got, err := NewEDDImporter().ImportEDD(file)
	if err != nil {
		t.Fatalf("ImportEDD: %v", err)
	}
	if len(got.Entities) != 1 {
		t.Fatalf("want 1 entity, got %d", len(got.Entities))
	}
	byName := map[string]*EDDXMLField{}
	for _, f := range got.Entities[0].Fields {
		byName[f.Name] = f
	}

	pa := byName["penicillin_allergic"]
	if pa == nil || pa.Collect != "true" || pa.Question == nil {
		t.Fatalf("penicillin_allergic collect/question lost: %+v", pa)
	}
	if pa.Question.Type != "multiple_choice" || pa.Question.Text != "Penicillin-allergic?" {
		t.Errorf("question text/type lost: %+v", pa.Question)
	}
	if len(pa.Question.Options) != 2 || pa.Question.Options[0].Value != "true" || pa.Question.Options[0].Label != "Yes" {
		t.Errorf("options lost: %+v", pa.Question.Options)
	}

	if age := byName["age"]; age == nil || age.Collect != "true" || age.Question == nil || age.Question.Type != "number" {
		t.Errorf("age collect/number-question lost: %+v", age)
	} else if age.Question.RefLow != "0" || age.Question.RefHigh != "120" || age.Question.Units != "years" {
		t.Errorf("age reference range lost through Excel round-trip: %+v", age.Question)
	}
	if notes := byName["notes"]; notes == nil || notes.Collect != "" || notes.Question != nil {
		t.Errorf("non-collected field gained metadata: %+v", notes)
	}
}
