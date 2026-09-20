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

const constrainedEDDXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="Acute Sinusitis" max_length="40">
      <allowed_value value="Acute Sinusitis"></allowed_value>
      <allowed_value value="Chronic Sinusitis"></allowed_value>
    </field>
    <field name="summary" type="string" access="rw" max_words="5"></field>
    <field name="notes" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`

func fieldsByName(t *testing.T, edd *EDDXML, entityName string) map[string]*EDDXMLField {
	t.Helper()
	out := map[string]*EDDXMLField{}
	for _, e := range edd.Entities {
		if e.Name != entityName {
			continue
		}
		for _, f := range e.Fields {
			out[f.Name] = f
		}
	}
	if len(out) == 0 {
		t.Fatalf("entity %q not found after round-trip; got %d entities", entityName, len(edd.Entities))
	}
	return out
}

func assertConstraints(t *testing.T, got map[string]*EDDXMLField) {
	t.Helper()
	d := got["diagnosis"]
	if d == nil {
		t.Fatalf("diagnosis field lost")
	}
	if len(d.AllowedValues) != 2 ||
		d.AllowedValues[0].Value != "Acute Sinusitis" || d.AllowedValues[1].Value != "Chronic Sinusitis" {
		t.Errorf("allowed_values lost or restyled: %+v", d.AllowedValues)
	}
	if d.MaxLength != "40" {
		t.Errorf("max_length = %q, want \"40\"", d.MaxLength)
	}
	if s := got["summary"]; s == nil || s.MaxWords != "5" {
		t.Errorf("max_words lost: %+v", s)
	}
	if n := got["notes"]; n == nil || len(n.AllowedValues) != 0 || n.MaxLength != "" || n.MaxWords != "" {
		t.Errorf("an unconstrained field grew constraints: %+v", n)
	}
}

// TestEDDRoundTrip_Constraints_RuleSetPath is vector 11 on the chain
// `dtrules verify` uses: the loader reads the constraints onto the entity
// entries, the REntity exporter writes columns N–P, and the importer reads
// them back. If any link drops them, an Excel refresh silently deletes a
// field's declared vocabulary (#1209).
func TestEDDRoundTrip_Constraints_RuleSetPath(t *testing.T) {
	rs := session.NewRuleSet("constraints-rt")
	if rs == nil {
		t.Fatal("NewRuleSet returned nil")
	}
	if err := rs.LoadEDD(strings.NewReader(constrainedEDDXML)); err != nil {
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
	assertConstraints(t, fieldsByName(t, got, "patient"))
}

// TestEDDRoundTrip_Constraints_XMLPath is vector 11 on the `dtrules build`
// chain: XML → workbook → XML, and the second XML must be byte-identical to
// the first. That equality is exactly what `dtrules verify` compares.
func TestEDDRoundTrip_Constraints_XMLPath(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "patient_edd.xml")

	// Start from XML the writer itself produced, so the comparison is about
	// the constraints surviving and not about indentation.
	seed, err := UnmarshalEDDXML([]byte(constrainedEDDXML))
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}
	imp := NewEDDImporter()
	if err := imp.WriteXML(seed, xmlPath); err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	first, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), `max_length="40"`) ||
		!strings.Contains(string(first), `<allowed_value value="Acute Sinusitis">`) {
		t.Fatalf("constraints missing from the written XML:\n%s", first)
	}

	workbook := filepath.Join(dir, "patient.xlsx")
	if err := WriteEDDXMLToExcel(seed, workbook); err != nil {
		t.Fatalf("WriteEDDXMLToExcel: %v", err)
	}

	back, err := NewEDDImporter().ImportEDD(workbook)
	if err != nil {
		t.Fatalf("ImportEDD: %v", err)
	}
	assertConstraints(t, fieldsByName(t, back, "patient"))

	// Rebuild the XML from the workbook and compare, the way verify does.
	sub := filepath.Join(dir, "rebuilt")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	rebuilt := filepath.Join(sub, "patient_edd.xml")
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
