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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestEDDRoundTrip_CollectQuestion_RuleSetPath exercises the exact chain that
// `dtrules verify` uses for the EDD: the loader populates EntityEntry.Collect/
// Question (processField), the REntity exporter writes columns I–L
// (writeEDDEntities), and the importer reads them back. This is what makes the
// authoring-contract round-trip hold for a project that uses collect (#850/#851).
func TestEDDRoundTrip_CollectQuestion_RuleSetPath(t *testing.T) {
	const eddXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="age" type="integer" access="rw" default_value="0" collect="true">
      <question text="Patient age?" type="number"></question>
    </field>
    <field name="allergic" type="boolean" access="rw" default_value="false" collect="true">
      <question text="Penicillin-allergic?" type="multiple_choice">
        <option value="true" label="Yes"></option>
        <option value="false" label="No"></option>
      </question>
    </field>
    <field name="notes" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`

	rs := session.NewRuleSet("collect-rt")
	if rs == nil {
		t.Fatal("NewRuleSet returned nil")
	}
	if err := rs.LoadEDD(strings.NewReader(eddXML)); err != nil {
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

	byName := map[string]*EDDXMLField{}
	for _, e := range got.Entities {
		if e.Name == "patient" {
			for _, f := range e.Fields {
				byName[f.Name] = f
			}
		}
	}
	if len(byName) == 0 {
		t.Fatalf("patient entity not found after round-trip; got %d entities", len(got.Entities))
	}

	if a := byName["allergic"]; a == nil || a.Collect != "true" || a.Question == nil ||
		a.Question.Type != "multiple_choice" || len(a.Question.Options) != 2 {
		t.Errorf("allergic collect/question/options lost through RuleSet path: %+v", a)
	}
	if age := byName["age"]; age == nil || age.Collect != "true" || age.Question == nil || age.Question.Type != "number" {
		t.Errorf("age collect/number-question lost: %+v", age)
	}
	if n := byName["notes"]; n == nil || n.Collect != "" || n.Question != nil {
		t.Errorf("non-collected field gained metadata: %+v", n)
	}
}
