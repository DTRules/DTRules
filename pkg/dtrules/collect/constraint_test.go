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

package collect

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// The collect resolver is the one outside write path the CLI vectors cannot
// reach: it needs an Asker, which only a UI supplies. #1209 vector 5.
const constrainedEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="Acute Sinusitis" collect="true" max_length="40">
      <question text="Working diagnosis?" type="ascii"></question>
      <allowed_value value="Acute Sinusitis"></allowed_value>
      <allowed_value value="Chronic Sinusitis"></allowed_value>
    </field>
    <field name="summary" type="string" access="rw" default_value="" collect="true" max_words="5">
      <question text="Summary?" type="ascii"></question>
    </field>
    <field name="notes" type="string" access="rw" default_value="" collect="true">
      <question text="Notes?" type="ascii"></question>
    </field>
  </entity>
</entity_data_dictionary>`

func loadConstrainedPatient(t *testing.T) *session.RuleSet {
	t.Helper()
	rs := session.NewRuleSet("collect-constraint-test")
	if rs == nil {
		t.Fatal("NewRuleSet nil")
	}
	if err := rs.LoadEDD(strings.NewReader(constrainedEDD)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	return rs
}

// TestCollector_RefusesValueOutsideVocabulary is issue vector 5: an Asker that
// answers with a value the field does not admit makes collection fail, names
// the field, the value and the set, and leaves the field UNCOLLECTED holding
// its default — not marked collected with a value that was rejected.
func TestCollector_RefusesValueOutsideVocabulary(t *testing.T) {
	rs := loadConstrainedPatient(t)
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	if pe == nil {
		t.Fatal("patient entity not found")
	}
	diagnosis := dtrules.GetRName("diagnosis")

	c := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		return dtrules.GetRString("Banana"), true, nil
	}))

	err := c.MaybeCollect(pe, diagnosis)
	if err == nil {
		t.Fatal("collecting a value outside the vocabulary must fail")
	}
	for _, want := range []string{"patient.diagnosis", "Banana", "Chronic Sinusitis"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	if pe.IsCollected(diagnosis) {
		t.Error("a refused answer must not leave the field marked collected")
	}
	if v, _ := pe.Get(diagnosis); v == nil || v.StringValue() != "Acute Sinusitis" {
		t.Errorf("refused answer must not be stored; got %v", v)
	}
}

// TestCollector_AcceptsCaseInsensitiveMatch: matching follows EL's rule for
// names. "acute sinusitis" and "Acute Sinusitis" are one value.
func TestCollector_AcceptsCaseInsensitiveMatch(t *testing.T) {
	rs := loadConstrainedPatient(t)
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	diagnosis := dtrules.GetRName("diagnosis")

	c := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		return dtrules.GetRString("chronic sinusitis"), true, nil
	}))
	if err := c.MaybeCollect(pe, diagnosis); err != nil {
		t.Fatalf("case-insensitive match must be accepted: %v", err)
	}
	if !pe.IsCollected(diagnosis) {
		t.Error("an accepted answer must mark the field collected")
	}
	if v, _ := pe.Get(diagnosis); v == nil || v.StringValue() != "chronic sinusitis" {
		t.Errorf("accepted answer not stored: %v", v)
	}
}

// TestCollector_RefusesOverlongAnswer covers the size limits: the error names
// the limit and the actual size, for both max_words and max_length.
func TestCollector_RefusesOverlongAnswer(t *testing.T) {
	rs := loadConstrainedPatient(t)
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")

	c := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		return dtrules.GetRString("one two  three\tfour five six"), true, nil
	}))
	err := c.MaybeCollect(pe, dtrules.GetRName("summary"))
	if err == nil {
		t.Fatal("a six-word answer must be refused when max_words is 5")
	}
	for _, want := range []string{"patient.summary", "6 words", "max_words is 5"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}

	c2 := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		return dtrules.GetRString(strings.Repeat("x", 41)), true, nil
	}))
	err = c2.MaybeCollect(pe, dtrules.GetRName("diagnosis"))
	if err == nil {
		t.Fatal("a 41-character answer must be refused when max_length is 40")
	}
	for _, want := range []string{"patient.diagnosis", "41 characters", "max_length is 40"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

// TestCollector_UnconstrainedFieldTakesAnything: a field declaring no
// constraint is not checked at all, so any answer stands.
func TestCollector_UnconstrainedFieldTakesAnything(t *testing.T) {
	rs := loadConstrainedPatient(t)
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	notes := dtrules.GetRName("notes")

	c := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		return dtrules.GetRString("Banana, and a great many words besides"), true, nil
	}))
	if err := c.MaybeCollect(pe, notes); err != nil {
		t.Fatalf("an unconstrained field must accept anything: %v", err)
	}
}

// TestCollector_RefusalAbortsTheRun: the refusal is not swallowed at the
// runtime seam. A read of the field through DTState.Find surfaces the error,
// which is what aborts an executing table.
func TestCollector_RefusalAbortsTheRun(t *testing.T) {
	rs := loadConstrainedPatient(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Skip("state is not *interpreter.DTState")
	}
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	if err := state.EntityPush(pe); err != nil {
		t.Fatalf("EntityPush: %v", err)
	}
	state.SetCollector(New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		return dtrules.GetRString("Banana"), true, nil
	})))

	_, err = state.Find(dtrules.GetRName("patient.diagnosis"))
	if err == nil {
		t.Fatal("a refused answer must surface as an error from Find, aborting the run")
	}
	if !strings.Contains(err.Error(), "patient.diagnosis") {
		t.Errorf("error %q does not name the field", err)
	}
}
