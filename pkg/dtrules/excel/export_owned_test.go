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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
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
