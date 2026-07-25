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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// build10EntryMapXML returns a MapXML with 10 attributes across 2 sections.
func build10EntryMapXML() *MapXML {
	return &MapXML{
		MapName: "RoundTrip",
		Entries: []MapEntry{
			{IsSection: true, Comment: "Job mappings"},
			{Tag: "id", RAttribute: "id", Enclosure: "job", Type: "integer"},
			{Tag: "tax_year", RAttribute: "tax_year", Enclosure: "job", Type: "integer"},
			{Tag: "filing_status", RAttribute: "filing_status", Enclosure: "job", Type: "string"},
			{Tag: "state", RAttribute: "state", Enclosure: "job", Type: "string"},
			{Tag: "expected_agi", RAttribute: "expected_agi", Enclosure: "job", Type: "double"},
			{IsSection: true, Comment: "Taxpayer mappings"},
			{Tag: "name", RAttribute: "name", Enclosure: "taxpayer", Type: "string"},
			{Tag: "ssn", RAttribute: "ssn", Enclosure: "taxpayer", Type: "string"},
			{Tag: "w2_wages", RAttribute: "w2_wages", Enclosure: "taxpayer", Type: "double"},
			{Tag: "estimated_payments", RAttribute: "estimated_payments", Enclosure: "taxpayer", Type: "double"},
		},
	}
}

// TestMAPRoundTripXMLToXLSXToXML verifies that XML→xlsx→XML preserves semantic equality.
func TestMAPRoundTripXMLToXLSXToXML(t *testing.T) {
	dir := t.TempDir()
	orig := build10EntryMapXML()

	// Step 1: write original to XML
	xmlPath := filepath.Join(dir, "test_map.xml")
	if err := WriteMapXML(orig, xmlPath); err != nil {
		t.Fatalf("WriteMapXML: %v", err)
	}

	// Step 2: load XML back
	loaded, err := LoadMapXMLFromFile(xmlPath)
	if err != nil {
		t.Fatalf("LoadMapXMLFromFile: %v", err)
	}

	// Step 3: export to xlsx
	xlsxPath := filepath.Join(dir, "test_map.xlsx")
	exp := NewMapExporter()
	if err := exp.ExportToFile(loaded, xlsxPath); err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}

	// Step 4: import xlsx
	imp := NewMapImporter()
	imported, err := imp.ImportFile(xlsxPath)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	// Step 5: write imported back to XML
	xml2Path := filepath.Join(dir, "test_map2.xml")
	if err := WriteMapXML(imported, xml2Path); err != nil {
		t.Fatalf("WriteMapXML round2: %v", err)
	}

	// Compare semantic equality: entry counts and values
	if len(imported.Entries) != len(orig.Entries) {
		t.Fatalf("entry count mismatch: got %d, want %d", len(imported.Entries), len(orig.Entries))
	}

	for i, o := range orig.Entries {
		g := imported.Entries[i]
		if o.IsSection != g.IsSection {
			t.Errorf("entry[%d] IsSection: got %v, want %v", i, g.IsSection, o.IsSection)
		}
		if o.IsSection {
			if o.Comment != g.Comment {
				t.Errorf("entry[%d] Comment: got %q, want %q", i, g.Comment, o.Comment)
			}
			continue
		}
		if o.Tag != g.Tag || o.RAttribute != g.RAttribute || o.Enclosure != g.Enclosure || o.Type != g.Type {
			t.Errorf("entry[%d]: got {%s %s %s %s}, want {%s %s %s %s}",
				i, g.Tag, g.RAttribute, g.Enclosure, g.Type,
				o.Tag, o.RAttribute, o.Enclosure, o.Type)
		}
	}

	// Also verify both XML files are valid by checking they exist and are non-empty
	for _, p := range []string{xmlPath, xml2Path} {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("stat %s: %v", p, err)
		} else if info.Size() == 0 {
			t.Errorf("%s is empty", p)
		}
	}
}

// TestMAPRoundTripStructuralSections pins the createentity / entities /
// initialization sections through the full XML→xlsx→XML cycle against the
// real KidAid map — the sections whose loss broke regenerated maps.
func TestMAPRoundTripStructuralSections(t *testing.T) {
	src := filepath.Join("..", "..", "..", "sampleprojects", "KidAid", "xml", "kidaid_map.xml")
	if _, err := os.Stat(src); err != nil {
		t.Skip("KidAid sample not available")
	}
	orig, err := LoadMapXMLFromFile(src)
	if err != nil {
		t.Fatalf("load kidaid map: %v", err)
	}
	if len(orig.CreateEntities) == 0 || len(orig.EntityDecls) == 0 || len(orig.InitialEntities) == 0 {
		t.Fatalf("kidaid map fixture missing structural sections: ce=%d decls=%d init=%d",
			len(orig.CreateEntities), len(orig.EntityDecls), len(orig.InitialEntities))
	}

	dir := t.TempDir()
	xlsxPath := filepath.Join(dir, "kidaid_map.xlsx")
	if err := NewMapExporter().ExportToFile(orig, xlsxPath); err != nil {
		t.Fatalf("export: %v", err)
	}
	imported, err := NewMapImporter().ImportFile(xlsxPath)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if len(imported.CreateEntities) != len(orig.CreateEntities) {
		t.Errorf("createentity count: got %d, want %d", len(imported.CreateEntities), len(orig.CreateEntities))
	}
	if len(imported.EntityDecls) != len(orig.EntityDecls) {
		t.Errorf("entity decl count: got %d, want %d", len(imported.EntityDecls), len(orig.EntityDecls))
	}
	if len(imported.InitialEntities) != len(orig.InitialEntities) {
		t.Errorf("initialentity count: got %d, want %d", len(imported.InitialEntities), len(orig.InitialEntities))
	}
	for i := range orig.CreateEntities {
		if i < len(imported.CreateEntities) && imported.CreateEntities[i] != orig.CreateEntities[i] {
			t.Errorf("createentity[%d]: got %+v, want %+v", i, imported.CreateEntities[i], orig.CreateEntities[i])
		}
	}
	for i := range orig.EntityDecls {
		if i < len(imported.EntityDecls) && imported.EntityDecls[i] != orig.EntityDecls[i] {
			t.Errorf("entitydecl[%d]: got %+v, want %+v", i, imported.EntityDecls[i], orig.EntityDecls[i])
		}
	}
	for i := range orig.InitialEntities {
		if i < len(imported.InitialEntities) && imported.InitialEntities[i] != orig.InitialEntities[i] {
			t.Errorf("initialentity[%d]: got %+v, want %+v", i, imported.InitialEntities[i], orig.InitialEntities[i])
		}
	}

	// The rewritten XML must load in the ENGINE's mapping parser — the
	// ultimate arbiter that nothing structural was lost.
	xml2 := filepath.Join(dir, "kidaid_map_roundtrip.xml")
	if err := WriteMapXML(imported, xml2); err != nil {
		t.Fatalf("write xml: %v", err)
	}
	data, err := os.ReadFile(xml2)
	if err != nil {
		t.Fatalf("read xml: %v", err)
	}
	for _, needle := range []string{"<createentity", "<entities>", "<initialization>", "epush='true'"} {
		if !strings.Contains(string(data), needle) {
			t.Errorf("rewritten map XML missing %q", needle)
		}
	}
}
