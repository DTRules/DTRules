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
	"os"
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

// The `dtrules build` chain: XML → workbook → XML, byte-identical, which is
// exactly what verify's [build] check compares.
func TestEDDRoundTrip_EntityComment_XMLPath(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "v_edd.xml")

	seed, err := UnmarshalEDDXML([]byte(commentedEDDXML))
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}
	if err := NewEDDImporter().WriteXML(seed, xmlPath); err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	first, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), `comment="the case under test"`) {
		t.Fatalf("entity comment missing from the written XML:\n%s", first)
	}

	workbook := filepath.Join(dir, "v.xlsx")
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

	rebuilt := filepath.Join(dir, "rebuilt", "v_edd.xml")
	if err := os.MkdirAll(filepath.Dir(rebuilt), 0o755); err != nil {
		t.Fatal(err)
	}
	back.FileMetadata = seed.FileMetadata
	for i, e := range back.Entities {
		if i < len(seed.Entities) {
			e.Number = seed.Entities[i].Number
			e.Source = seed.Entities[i].Source
			e.XlsFile = seed.Entities[i].XlsFile
		}
	}
	if err := NewEDDImporter().WriteXML(back, rebuilt); err != nil {
		t.Fatalf("WriteXML (rebuilt): %v", err)
	}
	second, err := os.ReadFile(rebuilt)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("the Excel round trip changed the EDD XML\n--- from XML ---\n%s\n--- rebuilt from Excel ---\n%s", first, second)
	}
}

// An entity declared in several EDD files is one REntity with one comment.
// A workbook-scoped export must write that comment only into the workbook
// whose EDD declared it, or the next build copies it into every other file
// that declares the entity -- TaxReturn's `result` is declared by 51 files
// and commented by one.
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
