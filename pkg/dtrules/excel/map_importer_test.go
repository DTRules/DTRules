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
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildSmallMAPXlsx creates an in-memory xlsx with a MAP sheet for testing.
func buildSmallMAPXlsx(t *testing.T, dir string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	// Row 1: MAP: marker
	f.SetCellValue(sheet, "A1", "MAP: TestProject")
	// Row 2: headers
	f.SetCellValue(sheet, "A2", "Tag")
	f.SetCellValue(sheet, "B2", "RAttribute")
	f.SetCellValue(sheet, "C2", "Enclosure")
	f.SetCellValue(sheet, "D2", "Type")
	// Row 3: section separator
	f.SetCellValue(sheet, "A3", "# Job mappings")
	// Row 4: first attribute
	f.SetCellValue(sheet, "A4", "id")
	f.SetCellValue(sheet, "B4", "id")
	f.SetCellValue(sheet, "C4", "job")
	f.SetCellValue(sheet, "D4", "integer")
	// Row 5: second attribute
	f.SetCellValue(sheet, "A5", "tax_year")
	f.SetCellValue(sheet, "B5", "tax_year")
	f.SetCellValue(sheet, "C5", "job")
	f.SetCellValue(sheet, "D5", "integer")
	// Row 6: section separator
	f.SetCellValue(sheet, "A6", "# Taxpayer mappings")
	// Row 7: third attribute
	f.SetCellValue(sheet, "A7", "name")
	f.SetCellValue(sheet, "B7", "name")
	f.SetCellValue(sheet, "C7", "taxpayer")
	f.SetCellValue(sheet, "D7", "string")

	path := filepath.Join(dir, "test_map.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save test MAP xlsx: %v", err)
	}
	return path
}

func TestMapImporterImportFile(t *testing.T) {
	dir := t.TempDir()
	xlsxPath := buildSmallMAPXlsx(t, dir)

	imp := NewMapImporter()
	m, err := imp.ImportFile(xlsxPath)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	if m.MapName != "TestProject" {
		t.Errorf("MapName = %q, want TestProject", m.MapName)
	}

	// Count entries: 2 sections + 3 attributes = 5
	if len(m.Entries) != 5 {
		t.Fatalf("entry count = %d, want 5", len(m.Entries))
	}

	// Entry 0: section "Job mappings"
	if !m.Entries[0].IsSection || m.Entries[0].Comment != "Job mappings" {
		t.Errorf("entry[0] = %+v, want section 'Job mappings'", m.Entries[0])
	}

	// Entry 1: tag=id, enclosure=job, type=integer
	e1 := m.Entries[1]
	if e1.Tag != "id" || e1.RAttribute != "id" || e1.Enclosure != "job" || e1.Type != "integer" {
		t.Errorf("entry[1] = %+v", e1)
	}

	// Entry 2: tag=tax_year
	e2 := m.Entries[2]
	if e2.Tag != "tax_year" || e2.Enclosure != "job" {
		t.Errorf("entry[2] = %+v", e2)
	}

	// Entry 3: section "Taxpayer mappings"
	if !m.Entries[3].IsSection || m.Entries[3].Comment != "Taxpayer mappings" {
		t.Errorf("entry[3] = %+v, want section 'Taxpayer mappings'", m.Entries[3])
	}

	// Entry 4: tag=name, enclosure=taxpayer
	e4 := m.Entries[4]
	if e4.Tag != "name" || e4.Enclosure != "taxpayer" || e4.Type != "string" {
		t.Errorf("entry[4] = %+v", e4)
	}
}
