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

// MAP sheet layout
//
// Row 1:  A1 = "MAP: <map_name>"  (merged across A–E for visibility)
// Row 2:  column headers — Tag | RAttribute | Enclosure | Type
// Row 3+: data rows
//         - Normal row: Tag, RAttribute, Enclosure, Type values
//         - Section row: col A = "# <comment text>", cols B–E blank
//
// Section comments from the XML (<!-- ... -->) become separator rows with
// the comment text prefixed with "# " in column A.  On re-import the "# "
// prefix is stripped to recover the original comment text.

package excel

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

const (
	mapColTag        = 1
	mapColRAttribute = 2
	mapColEnclosure  = 3
	mapColType       = 4
	mapColCount      = 4
)

// MapExporter writes a MapXML model to a MAP sheet in an Excel workbook.
type MapExporter struct{}

// NewMapExporter creates a MapExporter.
func NewMapExporter() *MapExporter { return &MapExporter{} }

// ExportToFile writes m to a standalone xlsx file at path.
func (e *MapExporter) ExportToFile(m *MapXML, path string) error {
	f := excelize.NewFile()
	defer f.Close()

	styler, err := NewStyler(f)
	if err != nil {
		return fmt.Errorf("create styles: %w", err)
	}

	sheetName := f.GetSheetName(0)
	if err := e.writeMapSheet(f, styler, sheetName, m); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return f.SaveAs(path)
}

// WriteMapSheet adds a MAP sheet named sheetName to an open workbook f.
// The sheet carries "MAP: <m.MapName>" in A1 so mixed-workbook importers
// can detect the sheet type from the A1 marker.
func (e *MapExporter) WriteMapSheet(f *excelize.File, styler *Styler, sheetName string, m *MapXML) error {
	if _, err := f.NewSheet(sheetName); err != nil {
		return err
	}
	return e.writeMapSheet(f, styler, sheetName, m)
}

func (e *MapExporter) writeMapSheet(f *excelize.File, styler *Styler, sheet string, m *MapXML) error {
	// Column widths
	f.SetColWidth(sheet, "A", "A", 35)
	f.SetColWidth(sheet, "B", "B", 35)
	f.SetColWidth(sheet, "C", "C", 20)
	f.SetColWidth(sheet, "D", "D", 12)

	// Row 1: MAP: marker merged across all columns
	markerCell := cellName(1, 1)
	endCell := cellName(mapColCount, 1)
	f.MergeCell(sheet, markerCell, endCell)
	f.SetCellValue(sheet, markerCell, "MAP: "+m.MapName)
	f.SetCellStyle(sheet, markerCell, endCell, styler.HeaderStyle)

	// Row 2: column headers
	headers := []string{"Tag", "RAttribute", "Enclosure", "Type"}
	for i, h := range headers {
		cell := cellName(i+1, 2)
		styler.ApplyHeader(f, sheet, cell, cell, cell, h)
	}

	FreezePaneAtRow2(f, sheet)

	// Rows 3+: data
	sectionStyle, err := f.NewStyle(&excelize.Style{
		Font:  &excelize.Font{Italic: true, Family: fontSansSerif, Size: 10, Color: "666666"},
		Fill:  excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8F8F8"}},
		Border: thinBorder(),
	})
	if err != nil {
		return err
	}

	row := 3
	for _, entry := range m.Entries {
		if entry.IsSection {
			// Section separator: "# <comment>" in A, blank in B–D
			f.SetCellValue(sheet, cellName(1, row), "# "+entry.Comment)
			f.SetCellStyle(sheet, cellName(1, row), cellName(mapColCount, row), sectionStyle)
		} else {
			f.SetCellValue(sheet, cellName(mapColTag, row), entry.Tag)
			f.SetCellValue(sheet, cellName(mapColRAttribute, row), entry.RAttribute)
			f.SetCellValue(sheet, cellName(mapColEnclosure, row), entry.Enclosure)
			f.SetCellValue(sheet, cellName(mapColType, row), entry.Type)
			f.SetCellStyle(sheet, cellName(1, row), cellName(mapColCount, row), styler.BodyStyle)
		}
		row++
	}

	return nil
}
