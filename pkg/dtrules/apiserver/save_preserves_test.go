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

package apiserver

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// A project carrying everything the editor's model does NOT edit: a context,
// a policy statement, and collect/question metadata on an EDD field. The
// collapse onto authoring.Project (#1084) must round-trip all of it.
const preserveEDD = `<entity_data_dictionary version="2">
  <entity name="person" number="100" access="rw">
    <field name="income" type="integer" access="rw" collect="true"><question text="Annual income?" type="number"></question></field>
    <field name="qualified" type="boolean" access="rw"></field>
  </entity>
  <entity name="job" number="200" access="rw">
    <field name="salary" type="integer" access="rw"></field>
  </entity>
</entity_data_dictionary>`

const preserveDT = `<decision_tables>
<decision_table>
  <table_name>Qualify</table_name>
  <attribute_fields><Type>FIRST</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
  <contexts><context_details>
    <context_number>1</context_number>
    <context_dsl>for all person.jobs</context_dsl>
    <context_postfix>dup person.jobs forall pop</context_postfix>
  </context_details></contexts>
  <conditions><condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>person.income &gt; 1000</condition_dsl>
    <condition_postfix>person.income 1000 i&gt;</condition_postfix>
    <condition_column column_number="1" column_value="Y" />
  </condition_details></conditions>
  <actions><action_details>
    <action_number>1</action_number>
    <action_dsl>set person.qualified = true</action_dsl>
    <action_postfix>true person.qualified bset</action_postfix>
    <action_column column_number="1" column_value="X" />
  </action_details></actions>
  <policy_statements><policy_statement column="1">
    <policy_description>Qualified on income</policy_description>
    <policy_statement_postfix>"Qualified on income"</policy_statement_postfix>
  </policy_statement></policy_statements>
</decision_table>
</decision_tables>`

// TestSavePreservesWhatTheEditorCannotEdit pins the round-trip half of
// #1084: contexts, policy statements, and collect/question metadata are
// read-only in the editor's model, and a save that reconciles through
// authoring.Project must carry them through untouched — the old serializer
// preserved them by merging, the new path by mutating only what the editor
// can express.
func TestSavePreservesWhatTheEditorCannotEdit(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "p_edd.xml"), []byte(preserveEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(xmlDir, "p_dt.xml")
	if err := os.WriteFile(dtPath, []byte(preserveDT), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{modified: map[string]bool{}}
	if err := s.LoadProject(dir); err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	// The editor touches one condition comment and one entity comment — the
	// smallest possible edits — and marks both files dirty.
	for _, td := range s.tables {
		if td.TableName == "Qualify" {
			td.Conditions[0].Comment = "edited in the browser"
		}
	}
	for _, e := range s.entities {
		if e.Name == "person" {
			e.Comment = "edited entity comment"
		}
	}
	for f := range map[string]bool{"xml/p_edd.xml": true, "xml/p_dt.xml": true} {
		s.modified[f] = true
	}
	s.mu.Lock()
	saved, err := s.saveViaProject()
	s.mu.Unlock()
	if err != nil {
		t.Fatalf("saveViaProject: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("saved %v, want both files", saved)
	}

	// The table's untouchables survived.
	got := tableFromFile(t, dtPath)
	if len(got.Contexts.Details) != 1 || !strings.Contains(got.Contexts.Details[0].DSL, "for all person.jobs") {
		t.Errorf("context did not survive the save: %+v", got.Contexts.Details)
	}
	if len(got.PolicyStatements) != 1 || got.PolicyStatements[0].Description != "Qualified on income" {
		t.Errorf("policy statement did not survive the save: %+v", got.PolicyStatements)
	}
	if got.Conditions[0].Comment != "edited in the browser" {
		t.Errorf("the edit itself did not land: %q", got.Conditions[0].Comment)
	}
	if !strings.Contains(got.Conditions[0].Postfix, "1000") {
		t.Errorf("condition postfix lost: %q", got.Conditions[0].Postfix)
	}

	// The EDD's collect metadata survived, keyed by name not by index.
	eddData, err := os.ReadFile(filepath.Join(xmlDir, "p_edd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var edd excel.EDDXML
	if err := xml.Unmarshal(eddData, &edd); err != nil {
		t.Fatalf("saved EDD does not parse: %v", err)
	}
	var person *excel.EDDXMLEntity
	for _, e := range edd.Entities {
		if e.Name == "person" {
			person = e
		}
	}
	if person == nil {
		t.Fatal("person entity missing after save")
	}
	if person.Comment != "edited entity comment" {
		t.Errorf("entity comment edit did not land: %q", person.Comment)
	}
	foundCollect := false
	for _, f := range person.Fields {
		if f.Name == "income" && f.Collect == "true" &&
			f.Question != nil && f.Question.Text == "Annual income?" {
			foundCollect = true
		}
	}
	if !foundCollect {
		t.Error("collect/question metadata did not survive the save")
	}
}
