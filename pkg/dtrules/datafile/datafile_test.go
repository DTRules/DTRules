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

package datafile

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const edd = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="medication" access="rw">
    <field name="name" type="string" access="rw"></field>
  </entity>
  <entity name="patient" access="rw">
    <field name="age" type="integer" access="rw" default_value="0" collect="true">
      <question text="Age?" type="number"></question>
    </field>
    <field name="active_medications" type="array" subtype="medication" access="rw"></field>
  </entity>
  <entity name="result" access="rw">
    <field name="warnings" type="array" subtype="string" access="rw"></field>
  </entity>
</entity_data_dictionary>`

func newRS(t *testing.T) *session.RuleSet {
	t.Helper()
	rs := session.NewRuleSet("datafile-test")
	if err := rs.LoadEDD(strings.NewReader(edd)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	return rs
}

func ref(rs *session.RuleSet, name string) *entity.REntity {
	return rs.GetEntityFactory().FindRefEntityByString(name)
}

func TestDatafile_RoundTrip(t *testing.T) {
	// --- source session: set patient.age, add a medication, a warning ---
	src := newRS(t)
	sess, err := src.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	patient := ref(src, "patient")
	patient.Put(dtrules.GetRName("age"), dtrules.GetRString("58"))

	med, err := sess.CreateEntity(dtrules.GetRName("medication"))
	if err != nil {
		t.Fatal(err)
	}
	med.Put(dtrules.GetRName("name"), dtrules.GetRString("Coumadin"))
	medsObj, _ := patient.Get(dtrules.GetRName("active_medications"))
	if arr, ok := medsObj.(*dtrules.RArray); ok {
		arr.Add(med)
	} else {
		t.Fatalf("active_medications is not an array: %T", medsObj)
	}
	result := ref(src, "result")
	if w, ok := mustArr(t, result, "warnings"); ok {
		w.Add(dtrules.GetRString("interaction warning"))
	}

	var buf strings.Builder
	if err := Write(&buf, src.GetEntityFactory().GetRefEntities()); err != nil {
		t.Fatalf("Write: %v", err)
	}
	xmlOut := buf.String()
	for _, want := range []string{"<patient>", "<age>58</age>", "<medication>", "<name>Coumadin</name>", "<item>interaction warning</item>"} {
		if !strings.Contains(xmlOut, want) {
			t.Errorf("canonical XML missing %q\n%s", want, xmlOut)
		}
	}

	// --- authoritative load into a fresh session ---
	dst := newRS(t)
	dstSess, _ := dst.NewSession()
	creator := func(sub string) (*entity.REntity, error) {
		e, err := dstSess.CreateEntity(dtrules.GetRName(sub))
		if err != nil {
			return nil, err
		}
		re, _ := e.(*entity.REntity)
		return re, nil
	}
	if err := Read(strings.NewReader(xmlOut), func(n string) *entity.REntity { return ref(dst, n) }, creator, Authoritative); err != nil {
		t.Fatalf("Read authoritative: %v", err)
	}
	dp := ref(dst, "patient")
	if v, _ := dp.Get(dtrules.GetRName("age")); v == nil || v.StringValue() != "58" {
		t.Errorf("age not loaded: %v", v)
	}
	if !dp.IsCollected(dtrules.GetRName("age")) {
		t.Error("authoritative load must mark age collected")
	}
	if arr, ok := mustArr(t, dp, "active_medications"); ok {
		els, _ := arr.ArrayValue()
		if len(els) != 1 {
			t.Fatalf("want 1 medication, got %d", len(els))
		}
		if me, ok := els[0].(*entity.REntity); ok {
			if nm, _ := me.Get(dtrules.GetRName("name")); nm == nil || nm.StringValue() != "Coumadin" {
				t.Errorf("medication name not loaded: %v", nm)
			}
		} else {
			t.Errorf("array element is not an entity: %T", els[0])
		}
	}

	// --- review load: value set, but NOT collected (so it gets re-asked) ---
	rv := newRS(t)
	if err := Read(strings.NewReader(xmlOut), func(n string) *entity.REntity { return ref(rv, n) }, nil, Review); err != nil {
		t.Fatalf("Read review: %v", err)
	}
	rp := ref(rv, "patient")
	if v, _ := rp.Get(dtrules.GetRName("age")); v == nil || v.StringValue() != "58" {
		t.Errorf("review: age value not pre-filled: %v", v)
	}
	if rp.IsCollected(dtrules.GetRName("age")) {
		t.Error("review load must leave age defaulted (not collected)")
	}
}

func mustArr(t *testing.T, e *entity.REntity, field string) (*dtrules.RArray, bool) {
	t.Helper()
	v, _ := e.Get(dtrules.GetRName(field))
	arr, ok := v.(*dtrules.RArray)
	if !ok {
		t.Errorf("%s is not an array: %T", field, v)
	}
	return arr, ok
}
