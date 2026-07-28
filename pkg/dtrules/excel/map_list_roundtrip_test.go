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

const listMapXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='floor' RAttribute='floor' enclosure='bracket' type='integer'></setattribute>
			<createentity entity='job' tag='job' id='id'></createentity>
			<createentity entity='bracket' tag='bracket_s' id='id' list='brackets_single'></createentity>
		</map>
		<entities>
			<entity name='job' number='1'></entity>
			<entity name='bracket' number='*'></entity>
		</entities>
	</XMLtoEDD>
</mapping>
`

// TestMapListAttributeSurvivesXMLRoundTrip guards the `list=` attribute on
// <createentity>. It is what appends a created entity to an array field on its
// enclosing entity; dropping it leaves the array empty at runtime and every
// rule that iterates it computes nothing — quietly. StateTax lost its tax
// brackets exactly this way, and its 51 scenarios still "passed" because
// nothing executed them.
func TestMapListAttributeSurvivesXMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_map.xml")
	if err := os.WriteFile(path, []byte(listMapXML), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadMapXMLFromFile(path)
	if err != nil {
		t.Fatalf("LoadMapXMLFromFile: %v", err)
	}
	if len(m.CreateEntities) != 2 {
		t.Fatalf("got %d createentity rows, want 2", len(m.CreateEntities))
	}
	if got := m.CreateEntities[1].List; got != "brackets_single" {
		t.Errorf("list attribute = %q, want %q", got, "brackets_single")
	}
	if got := m.CreateEntities[0].List; got != "" {
		t.Errorf("createentity without list got %q, want empty", got)
	}

	out := filepath.Join(dir, "out_map.xml")
	if err := WriteMapXML(m, out); err != nil {
		t.Fatalf("WriteMapXML: %v", err)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), `list='brackets_single'`) {
		t.Errorf("written XML lost the list attribute:\n%s", written)
	}
	if strings.Contains(string(written), `tag='job' id='id' list=`) {
		t.Errorf("written XML invented a list attribute for a row that had none:\n%s", written)
	}
}

// TestMapListAttributeSurvivesExcelRoundTrip covers the other half of the
// path: XML → xlsx → XML, which is what `dtrules build` runs.
func TestMapListAttributeSurvivesExcelRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_map.xml")
	if err := os.WriteFile(path, []byte(listMapXML), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadMapXMLFromFile(path)
	if err != nil {
		t.Fatalf("LoadMapXMLFromFile: %v", err)
	}

	xlsx := filepath.Join(dir, "test_map.xlsx")
	if err := NewMapExporter().ExportToFile(m, xlsx); err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}
	back, err := NewMapImporter().ImportFile(xlsx)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	if len(back.CreateEntities) != 2 {
		t.Fatalf("got %d createentity rows back, want 2", len(back.CreateEntities))
	}
	if got := back.CreateEntities[1].List; got != "brackets_single" {
		t.Errorf("list attribute after Excel round trip = %q, want %q", got, "brackets_single")
	}
}

// TestMapSectionKindToleratesOlderLegend confirms a workbook whose section
// marker predates a column being added still imports as that section. The
// legend is documentation for the human reading the sheet, not a format
// version.
func TestMapSectionKindToleratesOlderLegend(t *testing.T) {
	tests := map[string]string{
		"CREATE ENTITIES (entity | tag | id)":        mapSectionCreateEntities,
		"CREATE ENTITIES (entity | tag | id | list)": mapSectionCreateEntities,
		"ENTITIES (name | number: 1 or *)":           mapSectionEntities,
		"INITIALIZATION (entity | push)":             mapSectionInitialization,
		"SOMETHING ELSE":                             "SOMETHING ELSE",
	}
	for marker, want := range tests {
		if got := sectionKind(marker); got != want {
			t.Errorf("sectionKind(%q) = %q, want %q", marker, got, want)
		}
	}
}
