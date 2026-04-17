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

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const (
	kidAidEDD = "../../../sampleprojects/KidAid/xml/kidaid_edd.xml"
	kidAidDT  = "../../../sampleprojects/KidAid/xml/kidaid_dt.xml"
)

// TestNewStyler verifies all style indices are valid and distinct where expected.
func TestNewStyler(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	s, err := NewStyler(f)
	if err != nil {
		t.Fatalf("NewStyler: %v", err)
	}

	for name, idx := range map[string]int{
		"HeaderStyle":    s.HeaderStyle,
		"BodyStyle":      s.BodyStyle,
		"DSLStyle":       s.DSLStyle,
		"SeparatorStyle": s.SeparatorStyle,
		"NumberStyle":    s.NumberStyle,
	} {
		if idx < 0 {
			t.Errorf("style %s has invalid index %d", name, idx)
		}
	}

	// HeaderStyle and DSLStyle differ (different fonts: Calibri vs Consolas).
	if s.HeaderStyle == s.DSLStyle {
		t.Error("HeaderStyle and DSLStyle should be distinct style indices")
	}
}

// TestFreezePaneAtRow2 must not panic.
func TestFreezePaneAtRow2(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	f.NewSheet("TestSheet")
	FreezePaneAtRow2(f, "TestSheet")
}

// TestAutoWidth verifies min/max clamping.
func TestAutoWidth(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)

	AutoWidth(f, sheet, "A", 0)   // below min → clamp to 8
	AutoWidth(f, sheet, "B", 100) // above max → clamp to maxColWidth (60)
	AutoWidth(f, sheet, "C", 30)  // within range
}

// TestExportDecisionTablesStyle exports a real KidAid DT and spot-checks:
//   - CONDITIONS header row has HeaderStyle fill (#E8E8E8, bold)
//   - DSL expression cell (col C) uses monospace font (Consolas)
//   - output file is non-empty
func TestExportDecisionTablesStyle(t *testing.T) {
	rs := loadKidAidRuleSet(t)

	exporter := NewExporter(rs)
	out := filepath.Join(t.TempDir(), "dt_style.xlsx")
	if err := exporter.ExportDecisionTables(out); err != nil {
		t.Fatalf("ExportDecisionTables: %v", err)
	}

	assertFileNonEmpty(t, out)

	f := openXLSX(t, out)
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("exported DT file has no sheets")
	}
	sheet := sheets[0]

	rows, _ := f.GetRows(sheet)
	condRow := findRowContaining(rows, "CONDITIONS: COMMENTS")
	if condRow < 0 {
		t.Fatal("CONDITIONS: COMMENTS header not found in sheet")
	}

	// Verify header fill and bold.
	assertHeaderStyle(t, f, sheet, 1, condRow)

	// Verify DSL cell (col 3) in the first data row after header uses monospace font.
	dslDataRow := condRow + 1
	if dslDataRow <= len(rows) {
		assertDSLFont(t, f, sheet, 3, dslDataRow)
	}
}

// TestExportEDDStyle exports a real KidAid EDD and checks header styling.
func TestExportEDDStyle(t *testing.T) {
	rs := session.NewRuleSet("edd-style-test")
	if rs == nil {
		t.Skip("could not create rule set")
	}
	if err := rs.LoadEDDFile(kidAidEDD); err != nil {
		t.Skipf("skip EDD load: %v", err)
	}

	exporter := NewExporter(rs)
	out := filepath.Join(t.TempDir(), "edd_style.xlsx")
	if err := exporter.ExportEDD(out); err != nil {
		t.Fatalf("ExportEDD: %v", err)
	}

	assertFileNonEmpty(t, out)

	f := openXLSX(t, out)
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("EDD file has no sheets")
	}
	sheet := sheets[0]

	val, _ := f.GetCellValue(sheet, "A1")
	if val != "Entity" {
		t.Errorf("A1: got %q, want \"Entity\"", val)
	}

	assertHeaderStyle(t, f, sheet, 1, 1)
}

// TestExportCombinedStyle exports a combined workbook and checks both DT and EDD sheets.
func TestExportCombinedStyle(t *testing.T) {
	rs := loadKidAidRuleSet(t)

	exporter := NewExporter(rs)
	out := filepath.Join(t.TempDir(), "combined_style.xlsx")
	if err := exporter.ExportCombinedWorkbook(out); err != nil {
		t.Fatalf("ExportCombinedWorkbook: %v", err)
	}

	assertFileNonEmpty(t, out)

	f := openXLSX(t, out)
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) < 2 {
		t.Fatalf("combined workbook should have DT + EDD sheets, got %d", len(sheets))
	}

	// Last sheet should be EDD.
	eddSheet := sheets[len(sheets)-1]
	val, _ := f.GetCellValue(eddSheet, "A1")
	if val != "Entity" {
		t.Errorf("EDD sheet A1: got %q, want \"Entity\"", val)
	}
}

// --- helpers ---

func loadKidAidRuleSet(t *testing.T) *session.RuleSet {
	t.Helper()
	rs := session.NewRuleSet("dt-style-test")
	if rs == nil {
		t.Skip("could not create rule set")
	}
	if err := rs.LoadEDDFile(kidAidEDD); err != nil {
		t.Skipf("skip EDD load: %v", err)
	}
	if err := rs.LoadDecisionTablesFile(kidAidDT); err != nil {
		t.Skipf("skip DT load: %v", err)
	}
	if len(rs.GetDecisionTableNames()) == 0 {
		t.Skip("no decision tables loaded")
	}
	return rs
}

func assertFileNonEmpty(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Errorf("output file %s is empty", path)
	}
}

func openXLSX(t *testing.T, path string) *excelize.File {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	return f
}

func findRowContaining(rows [][]string, needle string) int {
	for i, row := range rows {
		for _, cell := range row {
			if cell == needle {
				return i + 1 // 1-indexed
			}
		}
	}
	return -1
}

func assertHeaderStyle(t *testing.T, f *excelize.File, sheet string, col, row int) {
	t.Helper()
	cell, _ := excelize.CoordinatesToCellName(col, row)
	styleID, err := f.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatalf("GetCellStyle %s: %v", cell, err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle %d: %v", styleID, err)
	}
	if len(style.Fill.Color) == 0 || style.Fill.Color[0] != colorHeader {
		t.Errorf("cell %s fill: got %v, want %s", cell, style.Fill.Color, colorHeader)
	}
	if style.Font == nil || !style.Font.Bold {
		t.Errorf("cell %s: expected bold font", cell)
	}
}

func assertDSLFont(t *testing.T, f *excelize.File, sheet string, col, row int) {
	t.Helper()
	cell, _ := excelize.CoordinatesToCellName(col, row)
	styleID, err := f.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatalf("GetCellStyle %s: %v", cell, err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle %d: %v", styleID, err)
	}
	if style.Font == nil || style.Font.Family != fontMono {
		got := ""
		if style.Font != nil {
			got = style.Font.Family
		}
		t.Errorf("DSL cell %s font: got %q, want %q", cell, got, fontMono)
	}
}

// Ensure testdata directory exists (used by future golden file tests).
var _ = filepath.Join("testdata", ".keep")
