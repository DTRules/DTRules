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

package operators

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// Behavior tests for the entity-manipulation ops. These are the ops
// decision tables rely on every time they reference an attribute —
// a silent regression here would break every rule that uses `entityfetch`
// or attribute lookup. None of the entity ops had behavior tests before,
// only registration coverage.
//
// Covered: entitypush, entitypop, entityname, entityid, req,
// InContext, isdefined, def.
// Not covered (needs a real EntityFactory beyond the mockSession):
//   newentity, findcreateentity, entityfetch.

// newTestEntity builds a fresh REntity with a unique ID and an optional
// single writable attribute for def-test coverage.
func newTestEntity(name string, id int, attrs map[string]dtrules.Object) *entity.REntity {
	e := entity.NewREntity(id, false, dtrules.GetRName(name))
	for n, v := range attrs {
		e.AddAttribute(
			dtrules.GetRName(n), "",
			v, true, true, dtrules.TypeInteger, "",
			"test attr", "", "",
		)
	}
	return e
}

// TestEntityNameAndID — entityname and entityid extract metadata.
func TestEntityNameAndID(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Customer", 42, nil)

	// entityname: ( entity -- name )
	state.DataPush(e)
	o, _ := Get(dtrules.GetRName("entityname"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("entityname: %v", err)
	}
	top, _ := state.DataPop()
	rn, err := top.RNameValue()
	if err != nil {
		t.Fatalf("RNameValue: %v", err)
	}
	if rn.StringValue() != "Customer" {
		t.Errorf("entityname: got %q, want %q", rn.StringValue(), "Customer")
	}

	// entityid: ( entity -- int )
	state.DataPush(e)
	o, _ = Get(dtrules.GetRName("entityid"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("entityid: %v", err)
	}
	top, _ = state.DataPop()
	id, _ := top.LongValue()
	if id != 42 {
		t.Errorf("entityid: got %d, want 42", id)
	}
}

// TestEntityPushPop — push then pop should round-trip the entity.
func TestEntityPushPop(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Widget", 7, nil)

	state.DataPush(e)
	pushOp, _ := Get(dtrules.GetRName("entitypush"))
	if err := pushOp.Execute(state); err != nil {
		t.Fatalf("entitypush: %v", err)
	}

	if state.EntityDepth() < 1 {
		t.Errorf("entitypush should have pushed; depth=%d", state.EntityDepth())
	}

	popOp, _ := Get(dtrules.GetRName("entitypop"))
	if err := popOp.Execute(state); err != nil {
		t.Fatalf("entitypop: %v", err)
	}
	top, _ := state.DataPop()
	gotEntity, err := top.REntityValue()
	if err != nil {
		t.Fatalf("REntityValue: %v", err)
	}
	if gotEntity.GetID() != 7 {
		t.Errorf("round-tripped entity id=%d, want 7", gotEntity.GetID())
	}
}

// TestEntityReq — req is reference equality by ID.
func TestEntityReq(t *testing.T) {
	state := newTestState()
	a := newTestEntity("E", 1, nil)
	b := newTestEntity("E", 1, nil) // same ID, different instance
	c := newTestEntity("E", 2, nil) // different ID

	cases := []struct {
		name  string
		x, y  dtrules.Object
		want  bool
	}{
		{"same_instance", a, a, true},
		{"same_id", a, b, true},
		{"different_id", a, c, false},
		{"null_null", dtrules.GetRNull(), dtrules.GetRNull(), true},
		{"entity_vs_null", a, dtrules.GetRNull(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state.DataPush(tc.x)
			state.DataPush(tc.y)
			o, _ := Get(dtrules.GetRName("req"))
			if err := o.Execute(state); err != nil {
				t.Fatalf("req: %v", err)
			}
			top, _ := state.DataPop()
			got, _ := top.BooleanValue()
			if got != tc.want {
				t.Errorf("req %s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestInContext — InContext searches the entity stack for the named
// entity and returns a boolean. Must preserve stack order on success
// and failure.
func TestInContext(t *testing.T) {
	state := newTestState()
	cust := newTestEntity("Customer", 1, nil)
	order := newTestEntity("Order", 2, nil)

	state.EntityPush(cust)
	state.EntityPush(order)
	depthBefore := state.EntityDepth()

	// Positive case: Customer is on the stack.
	state.DataPush(dtrules.GetRName("Customer"))
	o, _ := Get(dtrules.GetRName("InContext"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("InContext: %v", err)
	}
	top, _ := state.DataPop()
	found, _ := top.BooleanValue()
	if !found {
		t.Errorf("InContext(Customer): got false, want true")
	}
	if state.EntityDepth() != depthBefore {
		t.Errorf("InContext must preserve entity stack depth; got %d, want %d",
			state.EntityDepth(), depthBefore)
	}

	// Negative case: no such entity.
	state.DataPush(dtrules.GetRName("Missing"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("InContext negative: %v", err)
	}
	top, _ = state.DataPop()
	found, _ = top.BooleanValue()
	if found {
		t.Errorf("InContext(Missing): got true, want false")
	}
	if state.EntityDepth() != depthBefore {
		t.Errorf("InContext must preserve entity stack depth after miss")
	}
}

// TestIsDefined — isdefined returns true if the name resolves on the
// entity stack, false otherwise. Unlike bare name lookup it doesn't
// error on miss.
func TestIsDefined(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Box",
		1,
		map[string]dtrules.Object{"width": dtrules.GetRIntegerValue(0)})
	state.EntityPush(e)

	// positive
	state.DataPush(dtrules.GetRName("width"))
	o, _ := Get(dtrules.GetRName("isdefined"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("isdefined: %v", err)
	}
	top, _ := state.DataPop()
	v, _ := top.BooleanValue()
	if !v {
		t.Errorf("isdefined(width): got false, want true")
	}

	// negative — must not error
	state.DataPush(dtrules.GetRName("notreal"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("isdefined missing: %v", err)
	}
	top, _ = state.DataPop()
	v, _ = top.BooleanValue()
	if v {
		t.Errorf("isdefined(notreal): got true, want false")
	}
}

// TestDefAssignsAttribute — def writes value to the named attribute
// on the entity stack. Validates the full cycle: push entity with
// attribute, def the value, verify via Find.
func TestDefAssignsAttribute(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Counter", 1,
		map[string]dtrules.Object{"count": dtrules.GetRIntegerValue(0)})
	state.EntityPush(e)

	// Stack shape for def: ( value name -- )
	state.DataPush(dtrules.GetRIntegerValue(99))
	state.DataPush(dtrules.GetRName("count"))
	o, _ := Get(dtrules.GetRName("def"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("def: %v", err)
	}

	got, err := state.Find(dtrules.GetRName("count"))
	if err != nil {
		t.Fatalf("Find after def: %v", err)
	}
	v, err := got.LongValue()
	if err != nil {
		t.Fatalf("LongValue: %v", err)
	}
	if v != 99 {
		t.Errorf("def did not set value: got %d, want 99", v)
	}
}

// runRelOp pushes (entity, name) and executes op, returning the top result.
func runRelOp(t *testing.T, op string, ent dtrules.Object, name string) dtrules.Object {
	t.Helper()
	state := newTestState()
	state.DataPush(ent)
	state.DataPush(dtrules.NewRString(name))
	o, ok := Get(dtrules.GetRName(op))
	if !ok {
		t.Fatalf("op %q not registered", op)
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	top, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	return top
}

// TestRelationshipOps (#890) covers getrelationship/hasrelationship directly,
// including the edge cases the end-to-end test doesn't: missing attribute and
// a non-entity attribute.
func TestRelationshipOps(t *testing.T) {
	physician := entity.NewREntity(1, false, dtrules.GetRName("physician"))
	patient := entity.NewREntity(2, false, dtrules.GetRName("patient"))
	patient.AddAttribute(dtrules.GetRName("doctor"), "", physician, true, true, dtrules.TypeEntity, "physician", "", "", "")
	patient.AddAttribute(dtrules.GetRName("pname"), "", dtrules.NewRString("Bob"), true, true, dtrules.TypeString, "", "", "", "")

	// getrelationship: existing entity attribute -> that entity.
	got := runRelOp(t, "getrelationship", patient, "doctor")
	if e, err := got.REntityValue(); err != nil || e.GetID() != 1 {
		t.Errorf("getrelationship(doctor) = %v (%T), want physician(id 1)", got, got)
	}
	// getrelationship: missing attribute -> null.
	if got := runRelOp(t, "getrelationship", patient, "nurse"); got.Type() != dtrules.TypeNull {
		t.Errorf("getrelationship(nurse) = %v, want null", got)
	}

	// hasrelationship: entity attribute set -> true.
	if b, _ := runRelOp(t, "hasrelationship", patient, "doctor").BooleanValue(); !b {
		t.Error("hasrelationship(doctor) = false, want true")
	}
	// hasrelationship: missing attribute -> false.
	if b, _ := runRelOp(t, "hasrelationship", patient, "nurse").BooleanValue(); b {
		t.Error("hasrelationship(nurse) = true, want false")
	}
	// hasrelationship: non-entity attribute -> false.
	if b, _ := runRelOp(t, "hasrelationship", patient, "pname").BooleanValue(); b {
		t.Error("hasrelationship(pname) = true, want false (string, not a relationship)")
	}
	// hasrelationship: entity attribute cleared to null -> false.
	patient.Put(dtrules.GetRName("doctor"), dtrules.GetRNull())
	if b, _ := runRelOp(t, "hasrelationship", patient, "doctor").BooleanValue(); b {
		t.Error("hasrelationship(doctor) = true after null, want false")
	}
}
