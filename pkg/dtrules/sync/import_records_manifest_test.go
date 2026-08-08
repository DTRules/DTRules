// Copyright 2026 Paul Snow
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
	"testing"
	"time"
)

// The export guard refuses to write when a workbook is newer than the last
// recorded export, and tells the caller to "Import Excel to XML first, then
// re-apply your changes". The manifest was only ever written on export, so
// doing exactly that changed nothing and the guard blocked forever -- which is
// every fresh clone, since checkout stamps each file with the checkout time.
// The tool's own instructions have to work.

// guardedProject builds a syncer over a workbook that is newer than the
// manifest's recorded export, i.e. one the guard will refuse.
func guardedProject(t *testing.T) (*Syncer, *CombinedWorkbook) {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	excelDir := filepath.Join(root, "excel")
	for _, d := range []string{xmlDir, excelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	excelPath := filepath.Join(excelDir, "Thing.xlsx")
	if err := os.WriteFile(excelPath, []byte("workbook"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewSyncerWithOptions(xmlDir, excelDir, DefaultOptions())
	wb := &CombinedWorkbook{
		ExcelPath:  excelPath,
		DTXMLPath:  filepath.Join(xmlDir, "Thing_dt.xml"),
		EDDXMLPath: filepath.Join(xmlDir, "Thing_edd.xml"),
		RelPath:    "Thing",
	}

	// Put the pair in the state a checkout leaves behind: the workbook was
	// touched after the recorded export, but both are in the past, so a
	// fresh RecordExport (stamped now) will clear it. Backdating rather than
	// pushing the workbook into the future matters -- a future mtime is
	// genuinely unresolvable and the guard should keep refusing it.
	m, err := s.GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if err := m.RecordExport(excelPath, []string{wb.DTXMLPath}); err != nil {
		t.Fatalf("RecordExport: %v", err)
	}
	touched := time.Now().Add(-time.Hour)
	if err := os.Chtimes(excelPath, touched, touched); err != nil {
		t.Fatal(err)
	}
	entry := m.GetEntry(excelPath)
	if entry == nil {
		t.Fatal("RecordExport left no entry")
	}
	entry.LastExportTime = touched.Add(-time.Hour)
	entry.ExcelModTimeAtExport = entry.LastExportTime
	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := m.ExportGuard(excelPath); err == nil {
		t.Fatal("setup is wrong: the guard should be blocking before the import")
	}
	return s, wb
}

func TestImportClearsTheExportGuard(t *testing.T) {
	s, wb := guardedProject(t)

	// The separate-importer path, reached with no workbook importer set: the
	// imports themselves no-op on a stub workbook, which is fine -- what is
	// under test is whether the manifest gets written.
	s.importer = stubImporter{}
	if err := s.importCombinedWorkbook(wb); err != nil {
		t.Fatalf("importCombinedWorkbook: %v", err)
	}

	m, err := s.GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if err := m.ExportGuard(wb.ExcelPath); err != nil {
		t.Fatalf("the guard still blocks after an import, so its own advice "+
			"cannot be followed: %v", err)
	}
}

// recordSynced must not lose the EDD when a workbook has no DT half, or the
// guard stays armed for EDD-only workbooks.
func TestRecordSyncedCoversEDDOnlyWorkbooks(t *testing.T) {
	s, wb := guardedProject(t)
	wb.DTXMLPath = ""

	s.recordSynced(wb)

	m, err := s.GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if err := m.ExportGuard(wb.ExcelPath); err != nil {
		t.Fatalf("EDD-only workbook left the guard armed: %v", err)
	}
}

// stubImporter satisfies the separate-importer interface without touching
// Excel; the import result is not what these tests are about.
type stubImporter struct{}

func (stubImporter) ImportDecisionTable(_, _ string) error { return nil }
func (stubImporter) ImportEDD(_, _ string) error           { return nil }
