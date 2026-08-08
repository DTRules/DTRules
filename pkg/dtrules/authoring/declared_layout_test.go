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
	"testing"
)

const declaredEDD = `<entity_data_dictionary>
	<entity name="job" number="100" access="rw">
		<field name="age" type="integer" subtype="" access="rw" input="" default_value="" comment=""></field>
	</entity>
</entity_data_dictionary>`

const declaredDT = `<decision_tables><decision_table>
<table_name>Only</table_name>
<attribute_fields><Type>FIRST</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
<conditions></conditions><actions></actions>
</decision_table></decision_tables>`

// declaredLayoutProject writes a project whose DTRules.xml points somewhere
// other than the conventional xml/ + excel/, and puts a decoy workbook
// directory next to the rules — the shape that made this silently wrong.
func declaredLayoutProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	rules := filepath.Join(root, "pkg", "rules")
	books := filepath.Join(root, "books")
	decoy := filepath.Join(root, "pkg", "excel") // adjacent to the rules
	for _, d := range []string{rules, books, decoy} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(rules, "P_edd.xml"), declaredEDD)
	write(filepath.Join(rules, "P_dt.xml"), declaredDT)
	write(filepath.Join(root, "DTRules.xml"),
		"<DTRules><xml_dir>pkg/rules</xml_dir><excel_dir>books</excel_dir></DTRules>")
	return root
}

// TestOpenProjectHonoursDeclaredLayout pins #1049.
//
// OpenProject hardcoded `<path>/xml` and never read DTRules.xml, so the
// authoring API — the sanctioned way to change a rule without hand-editing XML
// — could not open a project that declares its own layout at all. The obvious
// workaround, pointing --project at whatever directory happens to contain an
// xml/, silently retargets the Excel side too.
func TestOpenProjectHonoursDeclaredLayout(t *testing.T) {
	root := declaredLayoutProject(t)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject on a project that declares its layout: %v", err)
	}
	if want := filepath.Join(root, "pkg", "rules"); p.xmlDir != want {
		t.Errorf("xmlDir = %s, want the declared %s", p.xmlDir, want)
	}
	if got, want := p.projectRoot(), root; got != want {
		t.Errorf("projectRoot() = %s, want %s — inferring it by stripping \"xml\" is wrong here", got, want)
	}
	if got, want := p.excelDirectory(), filepath.Join(root, "books"); got != want {
		t.Errorf("excelDirectory() = %s, want the declared %s", got, want)
	}
	// The decoy sits beside the rules and is what adjacency-based search finds.
	if p.excelDirectory() == filepath.Join(root, "pkg", "excel") {
		t.Error("resolved the workbook directory next to the rules rather than the declared one")
	}
}

// TestOpenProjectStillAcceptsConventionalLayout keeps the fix narrow: a project
// with no DTRules.xml must still open by convention.
func TestOpenProjectStillAcceptsConventionalLayout(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "P_dt.xml"), []byte(declaredDT), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject on a conventional project: %v", err)
	}
	if p.xmlDir != xmlDir {
		t.Errorf("xmlDir = %s, want %s", p.xmlDir, xmlDir)
	}
	if got, want := p.excelDirectory(), filepath.Join(root, "excel"); got != want {
		t.Errorf("excelDirectory() = %s, want the conventional %s", got, want)
	}
}
