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

package collect

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// recorderEDD mirrors the shape the issue's test vectors use: a ranged number
// question (SinusitisTherapy's patient.pcr), a multiple_choice, and a plain
// non-collect field.
const recorderEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="pcr" type="double" access="rw" default_value="0.0" collect="true">
      <question text="Plasma creatinine (mg/dL)?" type="number" ref_low="0.7" ref_high="1.3" units="mg/dL"></question>
    </field>
    <field name="allergic" type="boolean" access="rw" default_value="false" collect="true">
      <question text="Penicillin-allergic?" type="multiple_choice">
        <option value="true" label="Yes"></option>
        <option value="false" label="No"></option>
      </question>
    </field>
    <field name="unreached" type="integer" access="rw" default_value="0" collect="true">
      <question text="Never read by this run?" type="number"></question>
    </field>
    <field name="notes" type="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`

func loadRecorderRules(t *testing.T) *session.RuleSet {
	t.Helper()
	rs := session.NewRuleSet("recorder-test")
	if rs == nil {
		t.Fatal("NewRuleSet nil")
	}
	if err := rs.LoadEDD(strings.NewReader(recorderEDD)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	return rs
}

func newPatient(t *testing.T, rs *session.RuleSet) *entity.REntity {
	t.Helper()
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	e, err := sess.CreateEntity(dtrules.GetRName("patient"))
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	re, ok := e.(*entity.REntity)
	if !ok {
		t.Fatalf("CreateEntity returned %T, want *entity.REntity", e)
	}
	return re
}

// Vector 2 / 9: the recorded question carries the same content as the Request
// the CLI or web front end would have been asked — text, type, options,
// reference range, units — plus the default that was substituted for it.
func TestRecorder_RecordsRequestContentAndDefault(t *testing.T) {
	rs := loadRecorderRules(t)
	p := newPatient(t, rs)
	r := NewRecorder()

	if err := r.MaybeCollect(p, dtrules.GetRName("pcr")); err != nil {
		t.Fatalf("MaybeCollect: %v", err)
	}
	pending := r.Pending()
	if len(pending) != 1 {
		t.Fatalf("want 1 pending, got %d: %+v", len(pending), pending)
	}
	got := pending[0]
	if got.Entity != "patient" || got.Field != "pcr" {
		t.Errorf("identity: got %s.%s, want patient.pcr", got.Entity, got.Field)
	}
	if got.QuestionText != "Plasma creatinine (mg/dL)?" || got.QuestionType != "number" {
		t.Errorf("question: got %q/%q", got.QuestionText, got.QuestionType)
	}
	if got.RefLow != "0.7" || got.RefHigh != "1.3" || got.Units != "mg/dL" {
		t.Errorf("range: got %q-%q %q, want 0.7-1.3 mg/dL", got.RefLow, got.RefHigh, got.Units)
	}
	// The default is the live value the run went on to use, not the EDD's
	// declared spelling of it ("0.0" declares the double zero; the object
	// renders as "0").
	if got.Default != "0" {
		t.Errorf("default: got %q, want %q", got.Default, "0")
	}
	if got.Instance == 0 {
		t.Errorf("instance identity not recorded")
	}
	// The default stood: nothing was written, and the field is marked
	// collected so the run carries on without asking again.
	if v, _ := p.Get(dtrules.GetRName("pcr")); v == nil || v.StringValue() != "0" {
		t.Errorf("recorder must not change the value, got %v", v)
	}
	if !p.IsCollected(dtrules.GetRName("pcr")) {
		t.Errorf("field should be marked collected so it is not asked twice")
	}
}

// Vector 2: a multiple_choice question publishes its options.
func TestRecorder_RecordsOptions(t *testing.T) {
	rs := loadRecorderRules(t)
	p := newPatient(t, rs)
	r := NewRecorder()

	if err := r.MaybeCollect(p, dtrules.GetRName("allergic")); err != nil {
		t.Fatalf("MaybeCollect: %v", err)
	}
	pending := r.Pending()
	if len(pending) != 1 || pending[0].QuestionType != "multiple_choice" {
		t.Fatalf("unexpected pending: %+v", pending)
	}
	opts := pending[0].Options
	if len(opts) != 2 || opts[0].Value != "true" || opts[0].Label != "Yes" ||
		opts[1].Value != "false" || opts[1].Label != "No" {
		t.Errorf("options: %+v", opts)
	}
}

// Vectors 4 and 5: every *reached* collect field is listed once, in the order
// reached; a field the run never reads is not listed. (Which fields a run
// reaches is the interpreter's business; what the recorder guarantees is that
// it records exactly the ones it is offered, once each.)
func TestRecorder_OncePerFieldInOrderReached(t *testing.T) {
	rs := loadRecorderRules(t)
	p := newPatient(t, rs)
	r := NewRecorder()

	// allergic is read twice and pcr once, in that order; unreached is never
	// read at all.
	for _, f := range []string{"allergic", "pcr", "allergic", "pcr", "notes"} {
		if err := r.MaybeCollect(p, dtrules.GetRName(f)); err != nil {
			t.Fatalf("MaybeCollect %s: %v", f, err)
		}
	}
	pending := r.Pending()
	if len(pending) != 2 {
		t.Fatalf("want 2 pending, got %d: %+v", len(pending), pending)
	}
	if pending[0].Field != "allergic" || pending[1].Field != "pcr" {
		t.Errorf("order: got %s,%s want allergic,pcr", pending[0].Field, pending[1].Field)
	}
	for _, p := range pending {
		if p.Field == "unreached" {
			t.Errorf("an unreached field must not be pending")
		}
		if p.Field == "notes" {
			t.Errorf("a non-collect field must not be pending")
		}
	}
}

// Vector 10: two instances of one entity type, each missing the field, give
// one entry apiece, told apart by instance identity.
func TestRecorder_OneEntryPerEntityInstance(t *testing.T) {
	rs := loadRecorderRules(t)
	a := newPatient(t, rs)
	b := newPatient(t, rs)
	if a.GetID() == b.GetID() {
		t.Fatalf("test needs two distinct instances, both are %d", a.GetID())
	}
	r := NewRecorder()

	pcr := dtrules.GetRName("pcr")
	for _, e := range []*entity.REntity{a, b, a, b} {
		if err := r.MaybeCollect(e, pcr); err != nil {
			t.Fatalf("MaybeCollect: %v", err)
		}
	}
	pending := r.Pending()
	if len(pending) != 2 {
		t.Fatalf("want one entry per instance (2), got %d: %+v", len(pending), pending)
	}
	if pending[0].Instance != a.GetID() || pending[1].Instance != b.GetID() {
		t.Errorf("instances: got %d,%d want %d,%d",
			pending[0].Instance, pending[1].Instance, a.GetID(), b.GetID())
	}
}

// Vector 9: the recorder attaches straight to a DTState via SetCollector — it
// is a dtrules.Collector, not just an Asker.
func TestRecorder_AttachesAsCollector(t *testing.T) {
	rs := loadRecorderRules(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	dts, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Fatalf("state is %T, want *interpreter.DTState", sess.GetState())
	}
	r := NewRecorder()
	dts.SetCollector(r)
	if dts.Collector() != dtrules.Collector(r) {
		t.Errorf("SetCollector did not take the recorder")
	}
	// Batch execution — no collector attached — is unchanged: nothing to
	// record and nothing to pay for.
	dts.SetCollector(nil)
	if dts.Collector() != nil {
		t.Errorf("detaching the collector must restore batch execution")
	}
}

// An empty recording is `[]`, not `null`: "no question was reached" must be
// distinguishable from "the run never got that far", and a caller that parses
// the file should not have to special-case a null.
func TestRecorder_EmptyWritesJSONArray(t *testing.T) {
	r := NewRecorder()
	if got := r.Pending(); got == nil || len(got) != 0 {
		t.Fatalf("Pending on a fresh recorder: %+v", got)
	}
	var b strings.Builder
	if err := r.WritePending(&b); err != nil {
		t.Fatalf("WritePending: %v", err)
	}
	if strings.TrimSpace(b.String()) != "[]" {
		t.Errorf("empty recording wrote %q, want []", strings.TrimSpace(b.String()))
	}
}

// The published JSON uses the field names the issue names, so a front end can
// render the question without reaching back into DTRules.
func TestRecorder_JSONFieldNames(t *testing.T) {
	rs := loadRecorderRules(t)
	p := newPatient(t, rs)
	r := NewRecorder()
	if err := r.MaybeCollect(p, dtrules.GetRName("pcr")); err != nil {
		t.Fatalf("MaybeCollect: %v", err)
	}
	var b strings.Builder
	if err := r.WritePending(&b); err != nil {
		t.Fatalf("WritePending: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(b.String()), &out); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, b.String())
	}
	if len(out) != 1 {
		t.Fatalf("want 1 record, got %d", len(out))
	}
	for _, key := range []string{"entity", "instance", "field", "question_text",
		"question_type", "ref_low", "ref_high", "units", "default"} {
		if _, ok := out[0][key]; !ok {
			t.Errorf("published record has no %q: %v", key, out[0])
		}
	}
}
