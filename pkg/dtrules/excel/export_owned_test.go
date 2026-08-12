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

package excel

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
	"github.com/xuri/excelize/v2"
)

// Refreshing a project's workbooks used to call ExportDecisionTables once per
// workbook, and that writes the whole rule set. Across N workbooks every one
// ended up holding every table: a no-op `dtrules table put` on
// SinusitisTherapy took service1_medication.xlsx from 3 sheets to 6 and
// therapy.xlsx from 1 to 6 (#1077). Ownership is decided by workbookKey.

func TestWorkbookKeyMatchesHowXlsFileIsActuallyRecorded(t *testing.T) {
	cases := []struct {
		a, b string
		same bool
	}{
		// The recorded xls_file is historically inconsistent about extension.
		{"/p/excel/Foo.xlsx", "Foo.xls", true},
		{"/p/excel/Foo.xlsx", "Foo.xlsx", true},
		{"/p/excel/Foo.xlsx", "Foo", true},
		// And about being a path rather than a name.
		{"/p/excel/states/NV_corp.xlsx", "NV_corp.xlsx", true},
		{"/p/excel/Foo.xlsx", "other/Foo.xls", true},
		// Case is display-only in DTRules; two spellings are one name.
		{"/p/excel/Foo.xlsx", "foo.XLS", true},
		// Different workbooks stay different.
		{"/p/excel/Foo.xlsx", "Bar.xlsx", false},
		{"/p/excel/CHIP.xlsx", "CHIP_map.xlsx", false},
	}
	for _, c := range cases {
		got := workbookKey(c.a) == workbookKey(c.b)
		if got != c.same {
			t.Errorf("workbookKey(%q)==workbookKey(%q) = %v, want %v (%q vs %q)",
				c.a, c.b, got, c.same, workbookKey(c.a), workbookKey(c.b))
		}
	}
}

// A workbook no table claims must be left alone. Overwriting it with an empty
// file would destroy a workbook whose tables merely record a different
// spelling than the one on disk -- which is the state CorporateTax and CHIP
// were both in.
func TestExportOwnedByLeavesAnUnclaimedWorkbookAlone(t *testing.T) {
	e := NewExporter(session.NewRuleSet("empty")) // nothing can claim anything

	n, err := e.ExportDecisionTablesOwnedBy("/nonexistent/dir/Unclaimed.xlsx")
	if err != nil {
		t.Fatalf("an unclaimed workbook should be a no-op, not an error: %v", err)
	}
	if n != 0 {
		t.Errorf("wrote %d tables to a workbook no table claims, want 0", n)
	}
	// The path does not exist; reaching SaveAs would have errored above. That
	// it did not is the assertion.
}

// A workbook that keeps its tables and loses its dictionary has no legitimate
// form. The entities come from the loaded rule set, so an empty set means the
// EDD did not load -- and exporting anyway deletes the dictionary from the
// system of record, after which the next build compiles every field as an
// integer (#1094).
func TestExportOwnedByRefusesToDropTheEDDSheet(t *testing.T) {
	// Decision tables but no EDD: the shape a project takes when its EDD files
	// were never loaded. The tables claim a workbook, so the export runs.
	rs := session.NewRuleSet("no-edd")
	if rs == nil {
		t.Skip("could not create rule set")
	}
	if err := rs.LoadDecisionTablesFile(kidAidDT); err != nil {
		t.Skipf("skip DT load: %v", err)
	}
	names := rs.GetDecisionTableNames()
	if len(names) == 0 {
		t.Skip("no decision tables loaded")
	}
	e := NewExporter(rs)
	claimed := ""
	for _, dt := range e.getAllDecisionTables() {
		if p := dt.GetFilePath(); p != "" {
			claimed = filepath.Base(p)
			break
		}
	}
	if claimed == "" {
		t.Skip("no table names a workbook")
	}

	// The workbook on disk has a dictionary.
	path := filepath.Join(t.TempDir(), claimed)
	f := excelize.NewFile()
	if _, err := f.NewSheet("EDD"); err != nil {
		t.Fatal(err)
	}
	f.DeleteSheet(f.GetSheetName(0))
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := e.ExportDecisionTablesOwnedBy(path); err == nil {
		t.Fatal("exported a workbook with no entities over one that had an EDD sheet; " +
			"that deletes the dictionary from the system of record")
	} else if !strings.Contains(err.Error(), "EDD") {
		t.Errorf("error does not say what was at stake: %v", err)
	}

	back, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("the workbook is gone: %v", err)
	}
	defer back.Close()
	for _, s := range back.GetSheetList() {
		if strings.EqualFold(s, "EDD") {
			return
		}
	}
	t.Error("the EDD sheet was deleted from the system of record")
}

// An entity is one thing at run time and may be declared in many files. The
// merged REntity keeps one xls_file, so ownership judged by that alone claimed
// only the workbook that won the merge: TaxReturn's 50 state EDDs each add
// fields to the shared `result` naming their own workbook, and 49 of them were
// claimed by nothing and lost their EDD sheet on every refresh (#1109).
func TestEntityOwnershipFollowsTheFieldDeclaration(t *testing.T) {
	shared := &entity.REntity{}
	// Stand in for the merged entity: one xls_file, fields from two workbooks.
	shared.SetXlsFile("TaxReturn.xlsx")

	e := NewExporter(session.NewRuleSet("ownership"))
	fromFL := &entity.EntityEntry{Entity: shared, SourceXlsFile: "FL.xlsx"}
	fromCore := &entity.EntityEntry{Entity: shared, SourceXlsFile: "TaxReturn.xlsx"}
	unsourced := &entity.EntityEntry{Entity: shared}

	// Scoped to FL: only what FL declared. Writing the whole merged entity here
	// would put all 50 states' fields in every state's workbook, and the next
	// build would write them back into every state's EDD file.
	e.scopedTo = workbookKey("FL.xlsx")
	if !e.includes(fromFL) {
		t.Error("FL's own field was left out of FL's workbook")
	}
	if e.includes(fromCore) {
		t.Error("a field the core EDD declared was written into FL's workbook")
	}
	if e.includes(unsourced) {
		t.Error("a field with no recorded source went to a workbook that does not own the entity")
	}

	// Unscoped exports are unchanged: everything goes.
	e.scopedTo = ""
	for _, entry := range []*entity.EntityEntry{fromFL, fromCore, unsourced} {
		if !e.includes(entry) {
			t.Error("a whole-project export dropped a field")
		}
	}
}
