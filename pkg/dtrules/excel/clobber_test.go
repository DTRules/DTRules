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

// Tests for issue #585: DT importer clobbering action_dsl with action_comment
// during combined-workbook (exporter format) parsing.

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildExporterFormatSheet writes an xlsx sheet in the exporter.go format with
// one condition and one action. The layout mirrors what exporter.go produces:
//
//	col A=number, col B=comment, col C=DSL, col D+=decision column values
//
// The header section uses "Name: <table>" as the first row so detectFormat
// identifies it as "exporter" format.
func buildExporterFormatXLSX(t *testing.T, tableName, condComment, condDSL, actionComment, actionDSL string) string {
	t.Helper()

	f := excelize.NewFile()
	sheet := "Sheet1"

	row := 1

	// Table name row (exporter format marker)
	f.SetCellValue(sheet, cell(1, row), "Name: "+tableName)
	row++

	// Type row
	f.SetCellValue(sheet, cell(1, row), "Type: FIRST")
	row++

	// Table number
	f.SetCellValue(sheet, cell(1, row), "TABLE_NUMBER: 1")
	row++

	// Conditions header
	f.SetCellValue(sheet, cell(1, row), "Conditions: Comments")
	f.SetCellValue(sheet, cell(3, row), "DSL: Conditions")
	row++

	// Conditions column header row: col A=#, col B=Comment, col C=DSL, col D=1
	f.SetCellValue(sheet, cell(1, row), "#")
	f.SetCellValue(sheet, cell(2, row), "Comment")
	f.SetCellValue(sheet, cell(3, row), "DSL")
	f.SetCellValue(sheet, cell(4, row), "1") // decision column 1
	row++

	// Condition data row
	f.SetCellValue(sheet, cell(1, row), "1")
	f.SetCellValue(sheet, cell(2, row), condComment)
	f.SetCellValue(sheet, cell(3, row), condDSL)
	f.SetCellValue(sheet, cell(4, row), "*") // column value
	row++

	// Blank row to end conditions section
	row++

	// Actions header
	f.SetCellValue(sheet, cell(1, row), "Actions: Comments")
	f.SetCellValue(sheet, cell(3, row), "DSL: Actions")
	row++

	// Actions column header row: col A=#, col B=Comment, col C=DSL, col D=1
	f.SetCellValue(sheet, cell(1, row), "#")
	f.SetCellValue(sheet, cell(2, row), "Comment")
	f.SetCellValue(sheet, cell(3, row), "DSL")
	f.SetCellValue(sheet, cell(4, row), "1")
	row++

	// Action data row
	f.SetCellValue(sheet, cell(1, row), "1")
	f.SetCellValue(sheet, cell(2, row), actionComment)
	f.SetCellValue(sheet, cell(3, row), actionDSL)
	f.SetCellValue(sheet, cell(4, row), "X")
	row++
	_ = row

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path
}

// cell converts 1-based column and row to an Excel cell address like "A1".
func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// TestActionDSLPreservedThroughRoundTrip verifies that a distinct action_dsl
// value survives import and is not overwritten by the action_comment text (#585).
func TestActionDSLPreservedThroughRoundTrip(t *testing.T) {
	const actionComment = "broken action"
	const actionDSL = "garbage ~ {{"

	path := buildExporterFormatXLSX(t, "Clobber",
		"default", "otherwise",
		actionComment, actionDSL,
	)

	imp := NewDTImporter()
	tables, err := imp.ImportDecisionTables(path)
	if err != nil {
		t.Fatalf("ImportDecisionTables: %v", err)
	}
	if len(tables.Tables) == 0 {
		t.Fatal("no tables parsed")
	}
	tbl := tables.Tables[0]
	if len(tbl.Actions) == 0 {
		t.Fatal("no actions parsed")
	}
	got := tbl.Actions[0].DSL
	if got != actionDSL {
		t.Errorf("action_dsl clobbered: want %q got %q (action_comment=%q)", actionDSL, got, actionComment)
	}
}

// TestActionCommentPreservedThroughRoundTrip verifies that action_comment is
// preserved as the comment field (not silently dropped or mixed with DSL).
func TestActionCommentPreservedThroughRoundTrip(t *testing.T) {
	const actionComment = "the human-readable label"
	const actionDSL = "job.value = 42"

	path := buildExporterFormatXLSX(t, "CommentCheck",
		"default", "otherwise",
		actionComment, actionDSL,
	)

	imp := NewDTImporter()
	tables, err := imp.ImportDecisionTables(path)
	if err != nil {
		t.Fatalf("ImportDecisionTables: %v", err)
	}
	if len(tables.Tables) == 0 {
		t.Fatal("no tables parsed")
	}
	tbl := tables.Tables[0]
	if len(tbl.Actions) == 0 {
		t.Fatal("no actions parsed")
	}
	action := tbl.Actions[0]
	if action.Comment != actionComment {
		t.Errorf("action_comment wrong: want %q got %q", actionComment, action.Comment)
	}
	if action.DSL != actionDSL {
		t.Errorf("action_dsl wrong: want %q got %q", actionDSL, action.DSL)
	}
}

// TestConditionDSLPreservedThroughRoundTrip verifies that condition_dsl is not
// overwritten by condition_comment during import (#585).
func TestConditionDSLPreservedThroughRoundTrip(t *testing.T) {
	const condComment = "value is positive"
	const condDSL = "job.value > 0"

	path := buildExporterFormatXLSX(t, "CondDSLCheck",
		condComment, condDSL,
		"setup", "job.tax = 0",
	)

	imp := NewDTImporter()
	tables, err := imp.ImportDecisionTables(path)
	if err != nil {
		t.Fatalf("ImportDecisionTables: %v", err)
	}
	if len(tables.Tables) == 0 {
		t.Fatal("no tables parsed")
	}
	tbl := tables.Tables[0]
	if len(tbl.Conditions) == 0 {
		t.Fatal("no conditions parsed")
	}
	cond := tbl.Conditions[0]
	if cond.DSL != condDSL {
		t.Errorf("condition_dsl clobbered: want %q got %q (comment=%q)", condDSL, cond.DSL, condComment)
	}
	if cond.Comment != condComment {
		t.Errorf("condition_comment wrong: want %q got %q", condComment, cond.Comment)
	}
}
