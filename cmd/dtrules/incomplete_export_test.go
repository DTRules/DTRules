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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// The export builds from a tolerantly-loaded rule set, so a table that will
// not load is skipped and the workbook is written without it -- reporting
// success. Bootstrapping Cribbage produced a six-sheet workbook for an
// eleven-table project and the five missing tables were only noticed after
// being committed (#1081). verify cannot catch it: it compares the XML against
// a rebuild from that same workbook, so the missing tables are never part of
// the comparison.

// exportedProject writes a _dt.xml declaring tables and a workbook holding
// sheets, in whatever numbers the caller asks for.
func exportedProject(t *testing.T, tables, sheets int) *CLI {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	excelDir := filepath.Join(root, "excel")
	for _, d := range []string{xmlDir, excelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var sb strings.Builder
	sb.WriteString("<decision_tables>")
	for i := 0; i < tables; i++ {
		sb.WriteString("<decision_table><table_name>T")
		sb.WriteByte(byte('A' + i))
		sb.WriteString("</table_name></decision_table>")
	}
	sb.WriteString("</decision_tables>")
	if err := os.WriteFile(filepath.Join(xmlDir, "Proj_dt.xml"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	f := excelize.NewFile()
	defer f.Close()
	for i := 1; i < sheets; i++ { // NewFile already has one sheet
		if _, err := f.NewSheet("S" + string(rune('A'+i))); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(filepath.Join(excelDir, "Proj.xlsx")); err != nil {
		t.Fatal(err)
	}

	return &CLI{xmlDir: xmlDir, excelDir: excelDir}
}

func TestIncompleteExportIsReported(t *testing.T) {
	c := exportedProject(t, 11, 6) // the Cribbage shape

	found := c.incompleteExports()
	if len(found) != 1 {
		t.Fatalf("want 1 warning for a 6-sheet workbook covering 11 tables, got %d: %v",
			len(found), found)
	}
	// The counts are what make it actionable -- "something is wrong" is not
	// enough to act on before committing.
	for _, want := range []string{"6 sheet", "11 table", "Proj.xlsx", "Proj_dt.xml"} {
		if !strings.Contains(found[0], want) {
			t.Errorf("warning should mention %q, got: %s", want, found[0])
		}
	}
}

func TestCompleteExportIsSilent(t *testing.T) {
	c := exportedProject(t, 6, 6)

	if found := c.incompleteExports(); len(found) != 0 {
		t.Errorf("a workbook covering every table must be silent, got: %v", found)
	}
}

// More sheets than tables is normal -- a workbook can carry an EDD sheet or
// other content alongside. Only a shortfall means rules went missing.
func TestExtraSheetsAreNotAShortfall(t *testing.T) {
	c := exportedProject(t, 3, 8)

	if found := c.incompleteExports(); len(found) != 0 {
		t.Errorf("extra sheets are not missing rules, got: %v", found)
	}
}

// No workbook at all is verify's business, not this check's; reporting it here
// too would double up on every project mid-bootstrap.
func TestMissingWorkbookIsNotReportedHere(t *testing.T) {
	c := exportedProject(t, 4, 4)
	if err := os.Remove(filepath.Join(c.excelDir, "Proj.xlsx")); err != nil {
		t.Fatal(err)
	}

	if found := c.incompleteExports(); len(found) != 0 {
		t.Errorf("a missing workbook is reported by verify, not here, got: %v", found)
	}
}
