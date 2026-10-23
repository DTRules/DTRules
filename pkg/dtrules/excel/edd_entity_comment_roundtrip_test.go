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

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// An entity's comment lives in the Description cell of its header row on the
// EDD sheet (#1306). Before that cell was read and written, exporting a
// dictionary and rebuilding it dropped every entity comment, and the
// bootstrap had to keep such an EDD as XML no workbook reproduces.

const commentedEDDXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="policy" access="rw" comment="the case under test">
    <field name="flag" type="integer" access="rw" default_value="1" comment="always 1"></field>
  </entity>
  <entity name="result" access="rw">
    <field name="fired" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`

func entityComment(t *testing.T, edd *EDDXML, name string) string {
	t.Helper()
	for _, e := range edd.Entities {
		if e != nil && strings.EqualFold(e.Name, name) {
			return e.Comment
		}
	}
	t.Fatalf("entity %q not found after round-trip; got %d entities", name, len(edd.Entities))
	return ""
}

// The chain `dtrules verify` and every authoring refresh use: the loader puts
// the comment on the REntity, the exporter writes the sheet, the importer
// reads it back.
func TestEDDRoundTrip_EntityComment_RuleSetPath(t *testing.T) {
	rs := session.NewRuleSet("entity-comment-rt")
	if err := rs.LoadEDD(strings.NewReader(commentedEDDXML)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}

	file := filepath.Join(t.TempDir(), "edd.xlsx")
	if err := NewExporter(rs).ExportEDD(file); err != nil {
		t.Fatalf("ExportEDD: %v", err)
	}
	got, err := NewEDDImporter().ImportEDD(file)
	if err != nil {
		t.Fatalf("ImportEDD: %v", err)
	}
	if c := entityComment(t, got, "policy"); c != "the case under test" {
		t.Errorf("policy comment = %q, want %q", c, "the case under test")
	}
	if c := entityComment(t, got, "result"); c != "" {
		t.Errorf("result grew a comment: %q", c)
	}
	// The field's own comment still travels in its row.
	if f := fieldsByName(t, got, "policy")["flag"]; f == nil || f.Comment != "always 1" {
		t.Errorf("field comment lost: %+v", f)
	}
}

// WriteEDDXMLToExcel is the other writer of the sheet, from the XML structs
// rather than a loaded rule set; it writes the same cell. The build chain
// itself, XML → workbook → XML and verify, is TestBootstrapRoundTripsAnEntityComment.
func TestEDDRoundTrip_EntityComment_XMLPath(t *testing.T) {
	seed, err := UnmarshalEDDXML([]byte(commentedEDDXML))
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}
	workbook := filepath.Join(t.TempDir(), "v.xlsx")
	if err := WriteEDDXMLToExcel(seed, workbook); err != nil {
		t.Fatalf("WriteEDDXMLToExcel: %v", err)
	}
	back, err := NewEDDImporter().ImportEDD(workbook)
	if err != nil {
		t.Fatalf("ImportEDD: %v", err)
	}
	if c := entityComment(t, back, "policy"); c != "the case under test" {
		t.Errorf("policy comment = %q after the workbook, want %q", c, "the case under test")
	}
	if c := entityComment(t, back, "result"); c != "" {
		t.Errorf("result grew a comment: %q", c)
	}
}

// An entity declared in several EDD files is one REntity. A workbook-scoped
// export must write only the comment the workbook's own EDD declared, or the
// next build copies it into every other file that declares the entity --
// TaxReturn's `result` is declared by 51 files and commented by one.
func TestEDDExport_EntityComment_StaysWithItsWorkbook(t *testing.T) {
	const declaring = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="result" xls_file="A.xlsx" access="rw" comment="Tax calculation results">
    <field name="total" type="double" access="rw"></field>
  </entity>
</entity_data_dictionary>`
	const extending = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="result" xls_file="B.xlsx" access="rw">
    <field name="co_tax" type="double" access="rw"></field>
  </entity>
</entity_data_dictionary>`

	rs := session.NewRuleSet("entity-comment-scope")
	if err := rs.LoadEDD(strings.NewReader(declaring)); err != nil {
		t.Fatalf("LoadEDD A: %v", err)
	}
	if err := rs.LoadEDD(strings.NewReader(extending)); err != nil {
		t.Fatalf("LoadEDD B: %v", err)
	}

	dir := t.TempDir()
	exp := NewExporter(rs)
	for _, tc := range []struct{ workbook, want string }{
		{"A.xlsx", "Tax calculation results"},
		{"B.xlsx", ""},
	} {
		path := filepath.Join(dir, tc.workbook)
		if n, err := exp.ExportDecisionTablesOwnedBy(path); err != nil || n == 0 {
			t.Fatalf("export %s: n=%d err=%v", tc.workbook, n, err)
		}
		got, err := NewEDDImporter().ImportEDD(path)
		if err != nil {
			t.Fatalf("ImportEDD %s: %v", tc.workbook, err)
		}
		if c := entityComment(t, got, "result"); c != tc.want {
			t.Errorf("%s: result comment = %q, want %q", tc.workbook, c, tc.want)
		}
	}
}

// Two EDD files that both comment the same entity each keep their own: the
// loader holds one comment per declaring workbook, not one per entity, so
// neither file's build overwrites the other's. TaxReturn_edd_forms.xml
// declares `household_employee` twice with two comments.
func TestEDDExport_EntityComment_EachWorkbookKeepsItsOwn(t *testing.T) {
	const first = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="household_employee" xls_file="Core.xlsx" access="rw" comment="Household employee (nanny, housekeeper, etc.)">
    <field name="name" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`
	const second = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="household_employee" xls_file="Forms.xlsx" access="rw" comment="Schedule H Household Employment">
    <field name="wages" type="double" access="rw"></field>
  </entity>
</entity_data_dictionary>`

	rs := session.NewRuleSet("entity-comment-two-files")
	if err := rs.LoadEDD(strings.NewReader(first)); err != nil {
		t.Fatalf("LoadEDD Core: %v", err)
	}
	if err := rs.LoadEDD(strings.NewReader(second)); err != nil {
		t.Fatalf("LoadEDD Forms: %v", err)
	}

	dir := t.TempDir()
	exp := NewExporter(rs)
	for _, tc := range []struct{ workbook, want string }{
		{"Core.xlsx", "Household employee (nanny, housekeeper, etc.)"},
		{"Forms.xlsx", "Schedule H Household Employment"},
	} {
		path := filepath.Join(dir, tc.workbook)
		if n, err := exp.ExportDecisionTablesOwnedBy(path); err != nil || n == 0 {
			t.Fatalf("export %s: n=%d err=%v", tc.workbook, n, err)
		}
		got, err := NewEDDImporter().ImportEDD(path)
		if err != nil {
			t.Fatalf("ImportEDD %s: %v", tc.workbook, err)
		}
		if c := entityComment(t, got, "household_employee"); c != tc.want {
			t.Errorf("%s: household_employee comment = %q, want %q", tc.workbook, c, tc.want)
		}
	}
}

// An entity declared once with no workbook and once naming one: the
// workbook gets the comment that named it, whichever order the files were
// loaded in. The unsourced key sorted first and fell back to the entity's
// own workbook -- the last one loaded -- so Forms.xlsx got the unsourced
// comment. TaxReturn_edd_forms.xml declares `household_employee` in exactly
// this shape (lines 13 and 84).
func TestEDDExport_EntityComment_WorkbookBeatsUnsourced(t *testing.T) {
	const unsourced = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="household_employee" access="rw" comment="A-unsourced">
    <field name="name" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`
	const forms = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="household_employee" xls_file="Forms.xlsx" access="rw" comment="B-forms">
    <field name="wages" type="double" access="rw"></field>
  </entity>
</entity_data_dictionary>`

	for _, order := range []struct {
		name  string
		files []string
	}{
		{"unsourced then Forms", []string{unsourced, forms}},
		{"Forms then unsourced", []string{forms, unsourced}},
	} {
		t.Run(order.name, func(t *testing.T) {
			rs := session.NewRuleSet("entity-comment-unsourced")
			for i, f := range order.files {
				if err := rs.LoadEDD(strings.NewReader(f)); err != nil {
					t.Fatalf("LoadEDD %d: %v", i, err)
				}
			}
			path := filepath.Join(t.TempDir(), "Forms.xlsx")
			if n, err := NewExporter(rs).ExportDecisionTablesOwnedBy(path); err != nil || n == 0 {
				t.Fatalf("export Forms.xlsx: n=%d err=%v", n, err)
			}
			got, err := NewEDDImporter().ImportEDD(path)
			if err != nil {
				t.Fatalf("ImportEDD: %v", err)
			}
			if c := entityComment(t, got, "household_employee"); c != "B-forms" {
				t.Errorf("Forms.xlsx: household_employee comment = %q, want %q", c, "B-forms")
			}
		})
	}
}

// With no declaration naming the entity's workbook, an unsourced comment
// still falls back to it, as an unsourced field does.
func TestEDDExport_EntityComment_UnsourcedFallsBackToOwnWorkbook(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="policy" xls_file="v.xlsx" access="rw">
    <field name="flag" type="integer" access="rw"></field>
  </entity>
  <entity name="policy" access="rw" comment="A-unsourced">
    <field name="other" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`
	rs := session.NewRuleSet("entity-comment-fallback")
	if err := rs.LoadEDD(strings.NewReader(xmlDoc)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	path := filepath.Join(t.TempDir(), "v.xlsx")
	if n, err := NewExporter(rs).ExportDecisionTablesOwnedBy(path); err != nil || n == 0 {
		t.Fatalf("export v.xlsx: n=%d err=%v", n, err)
	}
	got, err := NewEDDImporter().ImportEDD(path)
	if err != nil {
		t.Fatalf("ImportEDD: %v", err)
	}
	if c := entityComment(t, got, "policy"); c != "A-unsourced" {
		t.Errorf("v.xlsx: policy comment = %q, want %q", c, "A-unsourced")
	}
}
