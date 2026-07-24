package excel

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteXML_NormalizesProvenance pins the emission contract: every table
// in written XML carries XLSFile and a <source>. Existing provenance is
// preserved (round-trip contract); tables without any — the authoring SDK
// used to emit an empty xls_file and no source — are backfilled, with sheet
// numbers appended after the file's highest existing sheet in TABLE_NUMBER
// order. Editors and sync tooling can then locate every table's workbook.
func TestWriteXML_NormalizesProvenance(t *testing.T) {
	tables := &DecisionTablesXML{Tables: []DecisionTableXML{
		{ // Excel-era table: carries provenance, but a stale sheet number.
			TableName:       "Beta",
			XLSFile:         "staking.xlsx",
			Source:          &SourceXML{RelativePath: "staking.xlsx", FileName: "staking.xlsx", SheetNumber: 10},
			AttributeFields: AttributeFieldsXML{Type: "FIRST", TableNumber: "2000"},
		},
		{ // SDK-authored table: no provenance at all.
			TableName:       "Alpha",
			AttributeFields: AttributeFieldsXML{Type: "FIRST", TableNumber: "100"},
		},
		{ // Non-numeric table number sorts after numeric ones.
			TableName:       "Gamma",
			AttributeFields: AttributeFieldsXML{Type: "FIRST", TableNumber: ""},
		},
	}}

	path := filepath.Join(t.TempDir(), "staking_dt.xml")
	if err := NewDTImporter().WriteXML(tables, path); err != nil {
		t.Fatal(err)
	}

	// Read back through the same parser.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalDecisionTablesXML(data)
	if err != nil {
		t.Fatal(err)
	}

	// Beta keeps its imported sheet 10; Alpha and Gamma are backfilled after
	// it in TABLE_NUMBER order (numeric 100 first, non-numeric last).
	want := map[string]int{"Beta": 10, "Alpha": 11, "Gamma": 12}
	for _, tb := range got.Tables {
		if tb.XLSFile != "staking.xlsx" {
			t.Errorf("%s: xls_file = %q, want staking.xlsx (inherited from the sibling that had one)", tb.TableName, tb.XLSFile)
		}
		if tb.Source == nil {
			t.Fatalf("%s: no <source> emitted", tb.TableName)
		}
		if tb.Source.FileName != "staking.xlsx" || tb.Source.RelativePath != "staking.xlsx" {
			t.Errorf("%s: source paths = %q/%q, want staking.xlsx", tb.TableName, tb.Source.RelativePath, tb.Source.FileName)
		}
		if tb.Source.SheetNumber != want[tb.TableName] {
			t.Errorf("%s: sheet_number = %d, want %d (existing preserved, missing appended)", tb.TableName, tb.Source.SheetNumber, want[tb.TableName])
		}
	}
}

// TestWriteXML_DerivesWorkbookFromFilename: a file with NO provenance
// anywhere gets its workbook derived from the XML filename, so a from-scratch
// SDK project still emits locatable metadata.
func TestWriteXML_DerivesWorkbookFromFilename(t *testing.T) {
	tables := &DecisionTablesXML{Tables: []DecisionTableXML{
		{TableName: "Only", AttributeFields: AttributeFieldsXML{Type: "FIRST", TableNumber: "5"}},
	}}
	path := filepath.Join(t.TempDir(), "budget_dt.xml")
	if err := NewDTImporter().WriteXML(tables, path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	got, err := UnmarshalDecisionTablesXML(data)
	if err != nil {
		t.Fatal(err)
	}
	tb := got.Tables[0]
	if tb.XLSFile != "budget.xlsx" {
		t.Errorf("xls_file = %q, want budget.xlsx (derived from budget_dt.xml)", tb.XLSFile)
	}
	if tb.Source == nil || tb.Source.SheetNumber != 1 {
		t.Errorf("source = %+v, want sheet 1", tb.Source)
	}
}
