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
	"path/filepath"
	"testing"
)

// buildSmallMAPXML returns a tiny in-memory MapXML for testing.
func buildSmallMAPXML() *MapXML {
	return &MapXML{
		MapName: "Tiny",
		Entries: []MapEntry{
			{IsSection: true, Comment: "Job section"},
			{Tag: "id", RAttribute: "id", Enclosure: "job", Type: "integer"},
			{Tag: "name", RAttribute: "name", Enclosure: "job", Type: "string"},
			{IsSection: true, Comment: "Person section"},
			{Tag: "dob", RAttribute: "date_of_birth", Enclosure: "person", Type: "date"},
		},
	}
}

func TestMapExporterRoundTrip(t *testing.T) {
	dir := t.TempDir()
	xlsxPath := filepath.Join(dir, "tiny_map.xlsx")

	m := buildSmallMAPXML()
	exp := NewMapExporter()
	if err := exp.ExportToFile(m, xlsxPath); err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}

	// Re-import and assert entries match.
	imp := NewMapImporter()
	m2, err := imp.ImportFile(xlsxPath)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	if m2.MapName != m.MapName {
		t.Errorf("MapName = %q, want %q", m2.MapName, m.MapName)
	}
	if len(m2.Entries) != len(m.Entries) {
		t.Fatalf("entry count = %d, want %d", len(m2.Entries), len(m.Entries))
	}

	for i, orig := range m.Entries {
		got := m2.Entries[i]
		if orig.IsSection != got.IsSection {
			t.Errorf("entry[%d] IsSection mismatch: got %v", i, got.IsSection)
		}
		if orig.IsSection {
			if orig.Comment != got.Comment {
				t.Errorf("entry[%d] Comment = %q, want %q", i, got.Comment, orig.Comment)
			}
			continue
		}
		if orig.Tag != got.Tag {
			t.Errorf("entry[%d] Tag = %q, want %q", i, got.Tag, orig.Tag)
		}
		if orig.RAttribute != got.RAttribute {
			t.Errorf("entry[%d] RAttribute = %q, want %q", i, got.RAttribute, orig.RAttribute)
		}
		if orig.Enclosure != got.Enclosure {
			t.Errorf("entry[%d] Enclosure = %q, want %q", i, got.Enclosure, orig.Enclosure)
		}
		if orig.Type != got.Type {
			t.Errorf("entry[%d] Type = %q, want %q", i, got.Type, orig.Type)
		}
	}
}
