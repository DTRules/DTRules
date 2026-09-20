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

package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAdvisoryProject(t *testing.T, edd, dt string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const advisoryEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="Acute Sinusitis">
      <allowed_value value="Acute Sinusitis"></allowed_value>
      <allowed_value value="Chronic Sinusitis"></allowed_value>
    </field>
    <field name="notes" type="string" access="rw" default_value=""></field>
  </entity>
</entity_data_dictionary>`

func dtWith(actions ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>Probe</table_name>
<actions>
`)
	for i, a := range actions {
		b.WriteString("<action_details><action_number>")
		b.WriteString(string(rune('1' + i)))
		b.WriteString("</action_number><action_dsl>")
		b.WriteString(a)
		b.WriteString("</action_dsl></action_details>\n")
	}
	b.WriteString("</actions>\n</decision_table>\n</decision_tables>\n")
	return b.String()
}

// TestConstrainedAssignments_FlagsTypo is the shape the advisory exists for:
// a literal one letter away from a member of the vocabulary.
func TestConstrainedAssignments_FlagsTypo(t *testing.T) {
	dir := writeAdvisoryProject(t, advisoryEDD, dtWith(`set patient.diagnosis = "Bannana"`))
	adv, err := AnalyzeConstrainedAssignments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(adv) != 1 {
		t.Fatalf("want 1 advisory, got %d (%v)", len(adv), adv)
	}
	a := adv[0]
	if a.Table != "Probe" || a.ActionNumber != "1" || a.Field != "patient.diagnosis" || a.Literal != "Bannana" {
		t.Errorf("advisory does not name the table, action, field and literal: %+v", a)
	}
	if !strings.Contains(a.String(), "Chronic Sinusitis") {
		t.Errorf("advisory text omits the allowed set: %s", a)
	}
}

// TestConstrainedAssignments_Quiet covers everything that must NOT be flagged:
// a member of the set (case-insensitively), an unconstrained field, and a
// computed value, which no static check can judge.
func TestConstrainedAssignments_Quiet(t *testing.T) {
	dir := writeAdvisoryProject(t, advisoryEDD, dtWith(
		`set patient.diagnosis = "chronic sinusitis"`,
		`set diagnosis = "Acute Sinusitis"`,
		`set patient.notes = "Banana"`,
		`set patient.diagnosis = patient.notes`,
	))
	adv, err := AnalyzeConstrainedAssignments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(adv) != 0 {
		t.Fatalf("no advisory expected, got %v", adv)
	}
}

// TestConstrainedAssignments_UnqualifiedResolves: a table with the entity on
// its context writes the field unqualified, and the bare name still resolves
// as long as only one entity declares it.
func TestConstrainedAssignments_UnqualifiedResolves(t *testing.T) {
	dir := writeAdvisoryProject(t, advisoryEDD, dtWith(`set diagnosis = "Banana"`))
	adv, err := AnalyzeConstrainedAssignments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(adv) != 1 || adv[0].Field != "patient.diagnosis" {
		t.Fatalf("bare field name did not resolve: %v", adv)
	}
}

// TestConstrainedAssignments_AmbiguousBareNameIsSilent: when two entities
// declare a vocabulary for the same field name, a bare reference cannot be
// resolved statically, and guessing would produce a false advisory.
func TestConstrainedAssignments_AmbiguousBareNameIsSilent(t *testing.T) {
	twoEntities := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="">
      <allowed_value value="Acute Sinusitis"></allowed_value>
    </field>
  </entity>
  <entity name="referral" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="">
      <allowed_value value="Banana"></allowed_value>
    </field>
  </entity>
</entity_data_dictionary>`
	dir := writeAdvisoryProject(t, twoEntities, dtWith(`set diagnosis = "Banana"`))
	adv, err := AnalyzeConstrainedAssignments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(adv) != 0 {
		t.Fatalf("an ambiguous bare name must not be guessed at: %v", adv)
	}
}
