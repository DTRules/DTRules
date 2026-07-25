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

// MAP sheet import layout (see map_exporter.go for the authoritative schema):
//
// Row 1:  "MAP: <name>"  (A1 marker)
// Row 2:  headers — Tag | RAttribute | Enclosure | Type
// Row 3+: data or section rows
//         Section row:  col A starts with "# " (the "# " prefix is stripped)
//         Normal row:   cols A–D contain attribute data

package excel

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// MapImporter parses a MAP sheet from an Excel workbook into a MapXML model.
type MapImporter struct{}

// NewMapImporter creates a MapImporter.
func NewMapImporter() *MapImporter { return &MapImporter{} }

// ImportSheet reads a MAP sheet from an open excelize.File.
// sheetName is the name of the sheet to read. The A1 cell must start with "MAP:".
func (imp *MapImporter) ImportSheet(f *excelize.File, sheetName string) (*MapXML, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("read MAP sheet %q: %w", sheetName, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("MAP sheet %q is empty", sheetName)
	}

	// Row 0: "MAP: <name>"
	a1 := strings.TrimSpace(getCellValue(rows[0], 0))
	if !strings.HasPrefix(strings.ToLower(a1), "map:") {
		return nil, fmt.Errorf("MAP sheet %q: A1 does not start with MAP:", sheetName)
	}
	mapName := strings.TrimSpace(a1[4:])

	m := &MapXML{MapName: mapName}

	// Row 1: headers (skip)
	// Rows 2+: data. "## <label>" rows switch the parsing mode to one of
	// the structural sections (create entities / entities / initialization).
	section := ""
	for rowIdx := 2; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		colA := strings.TrimSpace(getCellValue(row, 0))

		if colA == "" {
			// Completely blank row — skip
			continue
		}

		if strings.HasPrefix(colA, "## ") {
			section = strings.TrimPrefix(colA, "## ")
			continue
		}

		if strings.HasPrefix(colA, "# ") {
			// Section separator (comment)
			comment := strings.TrimPrefix(colA, "# ")
			m.Entries = append(m.Entries, MapEntry{IsSection: true, Comment: comment})
			continue
		}

		colB := strings.TrimSpace(getCellValue(row, 1))
		switch section {
		case mapSectionCreateEntities:
			m.CreateEntities = append(m.CreateEntities, MapCreateEntity{
				Entity: colA,
				Tag:    colB,
				ID:     strings.TrimSpace(getCellValue(row, 2)),
			})
		case mapSectionEntities:
			m.EntityDecls = append(m.EntityDecls, MapEntityDecl{Name: colA, Number: colB})
		case mapSectionInitialization:
			m.InitialEntities = append(m.InitialEntities, MapInitialEntity{
				Entity: colA,
				EPush:  strings.EqualFold(colB, "true"),
			})
		default:
			// Normal attribute row
			entry := MapEntry{
				Tag:        colA,
				RAttribute: colB,
				Enclosure:  strings.TrimSpace(getCellValue(row, 2)),
				Type:       strings.TrimSpace(getCellValue(row, 3)),
			}
			if entry.RAttribute == "" {
				entry.RAttribute = entry.Tag
			}
			m.Entries = append(m.Entries, entry)
		}
	}

	return m, nil
}

// ImportFile opens an xlsx file and reads its first (or only) MAP sheet.
func (imp *MapImporter) ImportFile(path string) (*MapXML, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	for _, sheet := range f.GetSheetList() {
		rows, _ := f.GetRows(sheet)
		if len(rows) > 0 && len(rows[0]) > 0 {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(rows[0][0])), "map:") {
				return imp.ImportSheet(f, sheet)
			}
		}
	}
	return nil, fmt.Errorf("%s: no MAP sheet found", path)
}
