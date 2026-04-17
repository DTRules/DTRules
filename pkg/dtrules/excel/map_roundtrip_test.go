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
