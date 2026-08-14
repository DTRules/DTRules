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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// The failure this replaces: checkout stamps every file with the same time, so
// a fresh clone had no usable timestamp signal at all — and the export guard,
// comparing mtime against a recorded export time, started every clone locked
// (#1061). Provenance travels with the artifact, so a clone answers correctly.

func writeStampedProject(t *testing.T, dir, hash string) *CombinedWorkbook {
	t.Helper()
	wb := filepath.Join(dir, "book.xlsx")
	dt := filepath.Join(dir, "book_dt.xml")
	if err := os.WriteFile(wb, []byte("workbook bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		hash = excel.WorkbookHash(wb)
	}
	body := fmt.Sprintf(`<decision_tables><decision_table><table_name>T</table_name>`+
		`<source><relative_path>book.xlsx</relative_path><file_name>book.xlsx</file_name>`+
		`<sheet_number>1</sheet_number><source_hash>%s</source_hash></source>`+
		`</decision_table></decision_tables>`, hash)
	if err := os.WriteFile(dt, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return &CombinedWorkbook{
		ExcelPath: wb, DTXMLPath: dt,
		ExcelExists: true, DTExists: true,
	}
}

func TestProvenanceSurvivesEveryFileBeingTouched(t *testing.T) {
	dir := t.TempDir()
	wb := writeStampedProject(t, dir, "")

	// What a checkout does: one timestamp on everything.
	stamp := time.Now()
	for _, p := range []string{wb.ExcelPath, wb.DTXMLPath} {
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	wb.ExcelMod, wb.DTMod = stamp, stamp

	s := NewSyncerWithOptions(dir, dir, DefaultOptions())
	if got := s.determineWorkbookDirection(wb); got != NoSync {
		t.Errorf("direction = %v, want NoSync — the workbook is the one the XML was compiled from", got)
	}
}

func TestChangedWorkbookIsDetectedWhateverTheClockSays(t *testing.T) {
	dir := t.TempDir()
	wb := writeStampedProject(t, dir, "sha256:stale")

	// Make the XML look far newer, which timestamps would read as XMLToExcel.
	future := time.Now().Add(24 * time.Hour)
	if err := os.Chtimes(wb.DTXMLPath, future, future); err != nil {
		t.Fatal(err)
	}
	wb.DTMod, wb.ExcelMod = future, time.Now()

	s := NewSyncerWithOptions(dir, dir, DefaultOptions())
	if got := s.determineWorkbookDirection(wb); got != ExcelToXML {
		t.Errorf("direction = %v, want ExcelToXML — the recorded provenance does not match the workbook", got)
	}
}

// XML written before the stamp existed must still work.
func TestUnstampedXMLFallsBackToTimestamps(t *testing.T) {
	dir := t.TempDir()
	wb := filepath.Join(dir, "book.xlsx")
	dt := filepath.Join(dir, "book_dt.xml")
	if err := os.WriteFile(wb, []byte("workbook"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dt, []byte(`<decision_tables><decision_table><table_name>T</table_name></decision_table></decision_tables>`), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	cw := &CombinedWorkbook{
		ExcelPath: wb, DTXMLPath: dt, ExcelExists: true, DTExists: true,
		ExcelMod: now, DTMod: now.Add(-48 * time.Hour),
	}

	s := NewSyncerWithOptions(dir, dir, DefaultOptions())
	if got := s.determineWorkbookDirection(cw); got != ExcelToXML {
		t.Errorf("direction = %v, want ExcelToXML from the timestamp fallback", got)
	}
}
