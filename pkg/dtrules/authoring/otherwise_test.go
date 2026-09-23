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

package authoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// The otherwise column in the authoring model (#1215).
//
// '*' does not mean "don't care" -- that is '-'. It marks the otherwise
// column: the last column, holding no Y or N, which executes iff no other
// column executed. The model accepted it anywhere with no check at all, so a
// table the engine would refuse to load could be written through the API and
// only fail later, with the XML already on disk.

const otherwiseEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="policy" number="100" xls_file="p.xlsx" access="rw">
		<field name="flag" type="integer" subtype="" access="rw" input="" default_value="1" comment=""></field>
	</entity>
	<entity name="result" number="200" xls_file="p.xlsx" access="rw">
		<field name="fired" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
	</entity>
</entity_data_dictionary>
`

const otherwiseDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>Probe</table_name>
<xls_file>p.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
<contexts></contexts><initial_actions></initial_actions>
<conditions>
<condition_details><condition_number>1</condition_number><condition_dsl>policy.flag == 7</condition_dsl><condition_postfix>policy.flag 7 ==</condition_postfix><columns>Y--</columns></condition_details>
<condition_details><condition_number>2</condition_number><condition_dsl>policy.flag &gt; 100</condition_dsl><condition_postfix>policy.flag 100 &gt;</condition_postfix><columns>-Y-</columns></condition_details>
</conditions>
<actions>
<action_details><action_number>1</action_number><action_dsl>set result.fired = "1"</action_dsl><action_postfix>"1" cvs /result.fired xdef</action_postfix><columns>X--</columns></action_details>
<action_details><action_number>2</action_number><action_dsl>set result.fired = "2"</action_dsl><action_postfix>"2" cvs /result.fired xdef</action_postfix><columns>-X-</columns></action_details>
<action_details><action_number>3</action_number><action_dsl>set result.fired = "O"</action_dsl><action_postfix>"O" cvs /result.fired xdef</action_postfix><columns>--X</columns></action_details>
</actions>
</decision_table>
</decision_tables>
`

// otherwiseProject writes a three-column project whose third column is empty,
// ready to be made the otherwise column.
func otherwiseProject(t *testing.T) *Project {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "p_edd.xml"), []byte(otherwiseEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "p_dt.xml"), []byte(otherwiseDT), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// setCell is what `table patch set-condition-cell` does: one cell, through
// UpdateCondition.
func setCell(t *Table, num, col int, value string) error {
	for _, c := range t.Conditions {
		if c.Number != num {
			continue
		}
		cols := make(map[int]string, len(c.Columns)+1)
		for k, v := range c.Columns {
			cols[k] = v
		}
		cols[col] = value
		return t.UpdateCondition(num, Condition{Comment: c.Comment, DSL: c.DSL, Columns: cols})
	}
	return nil
}

// TestOtherwiseCellAccepted: the last column holds no Y or N, so '*' belongs
// there -- and the cell survives the write.
func TestOtherwiseCellAccepted(t *testing.T) {
	p := otherwiseProject(t)
	tbl := p.Table("Probe")
	if err := setCell(tbl, 1, 3, "*"); err != nil {
		t.Fatalf("refused a legal otherwise cell: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	reopened := mustReopen(t, p)
	if got := reopened.Table("Probe").Conditions[0].Columns[3]; got != "*" {
		t.Errorf("otherwise cell came back %q, want \"*\"", got)
	}
}

// TestOtherwiseCellRefusedOutsideLastColumn: '*' in an ordinary column is the
// "don't care" reading, and it is not what '*' means.
func TestOtherwiseCellRefusedOutsideLastColumn(t *testing.T) {
	p := otherwiseProject(t)
	tbl := p.Table("Probe")
	err := setCell(tbl, 1, 2, "*")
	if err == nil {
		t.Fatal("accepted '*' outside the last column")
	}
	if !strings.Contains(err.Error(), "last column") {
		t.Errorf("error does not state the rule: %v", err)
	}
	// Refused means unchanged: the cell the caller tried to overwrite is
	// still what it was.
	if got := tbl.Conditions[1].Columns[2]; got != "Y" {
		t.Errorf("a refused edit changed the table: column 2 of condition 2 is %q", got)
	}
}

// TestOtherwiseColumnRefusesYOrN: the rule read the other way round -- a
// column marked '*' may not also test something.
func TestOtherwiseColumnRefusesYOrN(t *testing.T) {
	p := otherwiseProject(t)
	tbl := p.Table("Probe")
	if err := setCell(tbl, 1, 3, "*"); err != nil {
		t.Fatalf("refused a legal otherwise cell: %v", err)
	}
	err := setCell(tbl, 2, 3, "N")
	if err == nil {
		t.Fatal("accepted an N in the otherwise column")
	}
	if !strings.Contains(err.Error(), "otherwise column") {
		t.Errorf("error does not state the rule: %v", err)
	}
}

// TestAddColumnGoesBeforeOtherwise: appending at max+1 would leave the
// otherwise column in the middle. #1215 refused a column with content there,
// but let an empty one through as padding (#1221); now the new column takes
// the otherwise column's place and the otherwise column moves one right.
func TestAddColumnGoesBeforeOtherwise(t *testing.T) {
	p := otherwiseProject(t)
	tbl := p.Table("Probe")
	if err := setCell(tbl, 1, 3, "*"); err != nil {
		t.Fatalf("refused a legal otherwise cell: %v", err)
	}
	for _, conds := range []map[int]string{{1: "N", 2: "N"}, {}} {
		want := tbl.Columns() // the otherwise column's current place
		col, err := tbl.InsertColumn(conds, []int{1})
		if err != nil {
			t.Fatalf("InsertColumn(%v): %v", conds, err)
		}
		if col != want {
			t.Errorf("new column is %d, want %d (the otherwise column's place)", col, want)
		}
		if got := tbl.otherwiseColumn(); got != tbl.Columns() {
			t.Errorf("otherwise column is %d of %d: no longer last", got, tbl.Columns())
		}
	}
}

// TestOtherwiseSurvivesExcelRoundTrip: Excel is the system of record, so a
// '*' that cannot make the trip is a '*' that cannot be authored. Save
// bootstraps the workbook from the XML; importing it back must give the same
// cells.
func TestOtherwiseSurvivesExcelRoundTrip(t *testing.T) {
	p := otherwiseProject(t)
	tbl := p.Table("Probe")
	if err := setCell(tbl, 1, 3, "*"); err != nil {
		t.Fatalf("refused a legal otherwise cell: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	workbook := filepath.Join(p.excelDirectory(), "p.xlsx")
	if _, err := os.Stat(workbook); err != nil {
		t.Fatalf("save wrote no workbook to bootstrap from: %v", err)
	}

	imp := excel.NewWorkbookImporter()
	result, err := imp.ImportWorkbook(workbook)
	if err != nil {
		t.Fatalf("import workbook: %v", err)
	}
	var found bool
	for _, table := range result.DTables.Tables {
		if table.TableName != "Probe" {
			continue
		}
		found = true
		cells := table.Conditions[0].Row(3)
		if cells[2] != "*" {
			t.Errorf("column 3 came back from Excel as %q, want \"*\"", cells[2])
		}
		if cells[0] != "Y" {
			t.Errorf("column 1 came back from Excel as %q, want \"Y\"", cells[0])
		}
	}
	if !found {
		t.Fatal("Probe is not in the exported workbook")
	}
}

// mustReopen re-reads the project from disk, which is the only way to see
// what a Save actually wrote.
func mustReopen(t *testing.T, p *Project) *Project {
	t.Helper()
	reopened, err := OpenProject(p.projectRoot())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return reopened
}
