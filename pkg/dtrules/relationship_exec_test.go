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

package dtrules_test

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestRelationshipExecution (#890): the entity-relationship constructs
// `<name> of <entity>` (getrelationship) and `<entity> has a <name>`
// (hasrelationship) had no runtime op and crashed. They now resolve a
// relationship as an entity-typed attribute. This links a patient to a doctor
// and runs both constructs end to end.
func TestRelationshipExecution(t *testing.T) {
	rs := session.NewRuleSet("rel")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	// physician { dname: string }
	physName := dtrules.GetRName("physician")
	physRef, err := ef.FindCreateRefEntity(true, physName)
	if err != nil {
		t.Fatal(err)
	}
	physRef.AddAttribute(dtrules.GetRName("dname"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")

	// patient { doctor: physician }
	patName := dtrules.GetRName("patient")
	patRef, err := ef.FindCreateRefEntity(true, patName)
	if err != nil {
		t.Fatal(err)
	}
	patRef.AddAttribute(dtrules.GetRName("doctor"), "", nil, true, true, dtrules.TypeEntity, "physician", "", "", "")

	// root { patient, dr: entity; hasdoc: boolean }
	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	rootRef.AddAttribute(dtrules.GetRName("patient"), "", nil, true, true, dtrules.TypeEntity, "patient", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("dr"), "", nil, true, true, dtrules.TypeEntity, "physician", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("hasdoc"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")

	physician, err := ef.CreateEntity(sess, physName)
	if err != nil {
		t.Fatal(err)
	}
	physician.Put(dtrules.GetRName("dname"), dtrules.NewRString("Dr House"))
	patient, err := ef.CreateEntity(sess, patName)
	if err != nil {
		t.Fatal(err)
	}
	patient.Put(dtrules.GetRName("doctor"), physician)
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("patient"), patient)

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"patient": "entity", "dr": "entity", "hasdoc": "boolean"})

	runAction := func(expr string) {
		t.Helper()
		pf, err := elc.CompileAction(expr)
		if err != nil {
			t.Fatalf("%q compile: %v", expr, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", expr, pf, err)
		}
		state.EntityPush(root)
		if err := obj.Execute(state); err != nil {
			state.EntityPop()
			t.Fatalf("%q execute %q: %v", expr, pf, err)
		}
		state.EntityPop()
	}

	// getrelationship: `the "doctor" of patient` should be the physician.
	runAction(`set dr = the "doctor" of patient`)
	drObj, _ := root.Get(dtrules.GetRName("dr"))
	drEnt, err := drObj.REntityValue()
	if err != nil {
		t.Fatalf("dr is not an entity: %v (%T)", err, drObj)
	}
	name, _ := drEnt.Get(dtrules.GetRName("dname"))
	if name == nil || name.StringValue() != "Dr House" {
		t.Errorf("the doctor of patient: dname = %v, want \"Dr House\"", name)
	}

	// hasrelationship: patient has a doctor -> true.
	runAction(`set hasdoc = patient has a "doctor"`)
	hd, _ := root.Get(dtrules.GetRName("hasdoc"))
	if b, _ := hd.BooleanValue(); !b {
		t.Error(`patient has a "doctor" should be true`)
	}

	// Clear the relationship -> has a doctor is now false.
	patient.Put(dtrules.GetRName("doctor"), dtrules.GetRNull())
	runAction(`set hasdoc = patient has a "doctor"`)
	hd2, _ := root.Get(dtrules.GetRName("hasdoc"))
	if b, _ := hd2.BooleanValue(); b {
		t.Error(`patient has a "doctor" should be false when unset`)
	}
}
