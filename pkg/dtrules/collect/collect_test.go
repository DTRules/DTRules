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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const collectEDD = `<?xml version="1.0" encoding="UTF-8"?>
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

func loadPatient(t *testing.T) *session.RuleSet {
	t.Helper()
	rs := session.NewRuleSet("collect-test")
	if rs == nil {
		t.Fatal("NewRuleSet nil")
	}
	if err := rs.LoadEDD(strings.NewReader(collectEDD)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	return rs
}

func TestCollector_AsksReachedCollectFieldOnce(t *testing.T) {
	rs := loadPatient(t)
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	if pe == nil {
		t.Fatal("patient entity not found")
	}
	age := dtrules.GetRName("age")

	var asked []Request
	c := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		asked = append(asked, req)
		return dtrules.GetRString("58"), true, nil // Put coerces "58" -> integer
	}))

	// First read: asks, and the answer is stored (coerced to the field type).
	if err := c.MaybeCollect(pe, age); err != nil {
		t.Fatalf("MaybeCollect: %v", err)
	}
	if len(asked) != 1 {
		t.Fatalf("want 1 ask, got %d", len(asked))
	}
	if asked[0].Field != "age" || asked[0].QType != "number" || asked[0].Text != "Patient age?" {
		t.Errorf("unexpected request: %+v", asked[0])
	}
	if v, _ := pe.Get(age); v == nil || v.StringValue() != "58" {
		t.Errorf("age not set to 58: %v", v)
	}
	if !pe.IsCollected(age) {
		t.Error("age not marked collected")
	}

	// Second read: already collected, must NOT ask again.
	if err := c.MaybeCollect(pe, age); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Errorf("re-asked an already-collected field: %d asks", len(asked))
	}

	// A non-collect field is never asked.
	if err := c.MaybeCollect(pe, dtrules.GetRName("notes")); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Errorf("asked a non-collect field")
	}
}

func TestCollector_MultipleChoiceOptionsAndSkip(t *testing.T) {
	rs := loadPatient(t)
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	allergic := dtrules.GetRName("allergic")

	var asked []Request
	// Asker declines (ok=false): keep the default, but mark collected.
	c := New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		asked = append(asked, req)
		return nil, false, nil
	}))

	if err := c.MaybeCollect(pe, allergic); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0].QType != "multiple_choice" || len(asked[0].Options) != 2 {
		t.Fatalf("multiple_choice request wrong: %+v", asked)
	}
	if asked[0].Options[0].Value != "true" || asked[0].Options[0].Label != "Yes" {
		t.Errorf("options not passed through: %+v", asked[0].Options)
	}
	// Default kept (false), and marked collected so it won't re-ask.
	if v, _ := pe.Get(allergic); v == nil || v.StringValue() != "false" {
		t.Errorf("default not kept on skip: %v", v)
	}
	if !pe.IsCollected(allergic) {
		t.Error("skip must still mark collected (avoid re-ask)")
	}
	if err := c.MaybeCollect(pe, allergic); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Errorf("re-asked after skip")
	}
}

// TestCollector_FindHookTriggersCollection confirms the runtime seam: a read
// of a collect field through DTState.Find invokes the attached collector, and
// the collected value is what Find returns. With no collector attached, Find
// returns the default unchanged (batch == today).
func TestCollector_FindHookTriggersCollection(t *testing.T) {
	rs := loadPatient(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Skip("state is not *interpreter.DTState")
	}
	pe := rs.GetEntityFactory().FindRefEntityByString("patient")
	if pe == nil {
		t.Fatal("patient entity not found")
	}
	if err := state.EntityPush(pe); err != nil {
		t.Fatalf("EntityPush: %v", err)
	}
	ageQ := dtrules.GetRName("patient.age")

	// Batch path: no collector -> Find returns the default (0), no asking.
	if v, err := state.Find(ageQ); err != nil || v == nil || v.StringValue() != "0" {
		t.Fatalf("batch Find should return default 0, got %v (err %v)", v, err)
	}

	// Interactive: attach a collector; Find now blocks-and-collects.
	asked := 0
	state.SetCollector(New(AskerFunc(func(req Request) (dtrules.Object, bool, error) {
		asked++
		return dtrules.GetRString("47"), true, nil
	})))
	v, err := state.Find(ageQ)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if asked != 1 {
		t.Errorf("collector not invoked by Find: asked=%d", asked)
	}
	if v == nil || v.StringValue() != "47" {
		t.Errorf("Find did not return the collected value: %v", v)
	}
}

// TestCollector_NilSafe confirms the batch invariant: a nil-asker collector
// and a non-REntity are both no-ops.
func TestCollector_NilSafe(t *testing.T) {
	c := New(nil)
	if err := c.MaybeCollect(nil, dtrules.GetRName("x")); err != nil {
		t.Errorf("nil entity should be a no-op, got %v", err)
	}
}
