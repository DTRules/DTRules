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
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const arrayIndexEDD = `<entity_data_dictionary version="2">
<file_path>idx_edd</file_path>
<entity name="node" number="100" access="rw">
<field name="ancestors" type="array" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="counts" type="array" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="dates" type="array" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="flags" type="array" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="children" type="array" subtype="child" access="rw" input="" default_value="" comment=""></field>
<field name="kin" type="array" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="digest" type="bytes" subtype="" access="rw" input="" default_value="" comment=""></field>
</entity>
<entity name="child" number="101" access="rw">
<field name="label" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
</entity>
<entity name="elder" number="103" access="rw">
<field name="label" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
</entity>
<entity name="result" number="102" access="rw">
<field name="table" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="count" type="integer" subtype="" access="rw" input="" default_value="0" comment=""></field>
<field name="amount" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
<field name="when" type="date" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="flag" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
<field name="kid" type="entity" subtype="child" access="rw" input="" default_value="" comment=""></field>
</entity>
</entity_data_dictionary>`

// TestCastIndexedArrayElement (#1229): `(string) arr[i]` — and every other
// cast of an indexed array — gives the element at i, whatever the element
// type, while indexing a bytes value still gives the byte. The EL is compiled
// the way `dtrules build` compiles it: symbols from the EDD on disk
// (authoring.LoadEDDSymbols) and the authoring compiler (CheckAction); the
// postfix then runs against entities created from the same EDD.
func TestCastIndexedArrayElement(t *testing.T) {
	dir := t.TempDir()
	eddPath := filepath.Join(dir, "idx_edd.xml")
	if err := os.WriteFile(eddPath, []byte(arrayIndexEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	symbols := authoring.LoadEDDSymbols(dir)
	if symbols["node.ancestors"] != "array" || symbols["node.digest"] != "bytes" {
		t.Fatalf("EDD symbols not loaded as expected: %v", symbols)
	}

	rs := session.NewRuleSet("idx")
	if err := rs.LoadEDDFile(eddPath); err != nil {
		t.Fatalf("load edd: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()

	node, err := sess.CreateEntity(dtrules.GetRName("node"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := sess.CreateEntity(dtrules.GetRName("result"))
	if err != nil {
		t.Fatal(err)
	}
	child0, err := sess.CreateEntity(dtrules.GetRName("child"))
	if err != nil {
		t.Fatal(err)
	}
	child1, err := sess.CreateEntity(dtrules.GetRName("child"))
	if err != nil {
		t.Fatal(err)
	}
	child1.Put(dtrules.GetRName("label"), dtrules.NewRString("second"))
	elder, err := sess.CreateEntity(dtrules.GetRName("elder"))
	if err != nil {
		t.Fatal(err)
	}

	arr := func(elems ...dtrules.Object) *dtrules.RArray {
		t.Helper()
		a, err := dtrules.NewArrayWithElements(sess, true, elems, false)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	node.Put(dtrules.GetRName("ancestors"), arr(dtrules.NewRString("e11"), dtrules.NewRString("e12")))
	node.Put(dtrules.GetRName("counts"), arr(dtrules.GetRIntegerValue(300), dtrules.GetRIntegerValue(41)))
	node.Put(dtrules.GetRName("dates"), arr(dtrules.NewRString("2026-01-15")))
	node.Put(dtrules.GetRName("flags"), arr(dtrules.GetRBoolean(false), dtrules.GetRBoolean(true)))
	node.Put(dtrules.GetRName("children"), arr(child0, child1))
	node.Put(dtrules.GetRName("kin"), arr(child0, elder))
	node.Put(dtrules.GetRName("digest"), dtrules.NewRBytes([]byte{0xde, 0xad, 0xbe, 0xef}))

	var syms map[string]string = symbols
	exec := func(action string) {
		t.Helper()
		postfix, err := authoring.CheckAction(action, syms)
		if err != nil {
			t.Fatalf("%q compile: %v", action, err)
		}
		obj, err := sess.Compile(postfix)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", action, postfix, err)
		}
		state.EntityPush(node)
		state.EntityPush(result)
		defer func() {
			state.EntityPop()
			state.EntityPop()
		}()
		if err := obj.Execute(state); err != nil {
			t.Errorf("%q execute %q: %v", action, postfix, err)
		}
	}
	get := func(field string) dtrules.Object {
		t.Helper()
		v, err := result.Get(dtrules.GetRName(field))
		if err != nil {
			t.Fatalf("get result.%s: %v", field, err)
		}
		return v
	}
	str := func(field string) string { return get(field).StringValue() }
	num := func(field string) int {
		i, _ := get(field).IntValue()
		return i
	}

	// Arrays of strings: the issue's statement, and a non-zero index.
	exec(`set result.table = (string) node.ancestors[0]`)
	if got := str("table"); got != "e11" {
		t.Errorf(`(string) node.ancestors[0] = %q, want "e11"`, got)
	}
	exec(`set result.table = (string) ancestors[1]`)
	if got := str("table"); got != "e12" {
		t.Errorf(`(string) ancestors[1] = %q, want "e12"`, got)
	}

	// Arrays of numbers, cast to each numeric type and to a string. 300 is
	// not a byte, so a byte read could not produce it.
	exec(`set result.count = (long) node.counts[0]`)
	if got := num("count"); got != 300 {
		t.Errorf("(long) node.counts[0] = %d, want 300", got)
	}
	exec(`set result.amount = (double) node.counts[1]`)
	if f, _ := get("amount").DoubleValue(); f != 41 {
		t.Errorf("(double) node.counts[1] = %v, want 41", f)
	}
	exec(`set result.table = (string) node.counts[0]`)
	if got := str("table"); got != "300" {
		t.Errorf(`(string) node.counts[0] = %q, want "300"`, got)
	}

	// Arrays of booleans and of date strings.
	exec(`set result.flag = (boolean) node.flags[1]`)
	if b, _ := get("flag").BooleanValue(); !b {
		t.Error("(boolean) node.flags[1] = false, want true")
	}
	exec(`set result.when = (date) node.dates[0]`)
	if d, err := get("when").TimeValue(); err != nil || d.Format("2006-01-02") != "2026-01-15" {
		t.Errorf("(date) node.dates[0] = %v (%v), want 2026-01-15", d, err)
	}

	// Arrays of entities: the element itself, and a cast of it.
	exec(`set result.kid = node.children[1]`)
	if k, err := get("kid").REntityValue(); err != nil || k.GetID() != child1.GetID() {
		t.Errorf("node.children[1] = %v (%v), want the second child", get("kid"), err)
	}
	// A cast of an entity element reads its entity type, so index an array
	// whose two elements differ in type: an off-by-one gives the other one.
	exec(`set result.table = (string) node.kin[1]`)
	if got := str("table"); got != "elder" {
		t.Errorf(`(string) node.kin[1] = %q, want "elder"`, got)
	}
	exec(`set result.table = (string) node.kin[0]`)
	if got := str("table"); got != "child" {
		t.Errorf(`(string) node.kin[0] = %q, want "child"`, got)
	}

	// A bytes value still indexes to the byte.
	exec(`set result.count = (long) node.digest[1]`)
	if got := num("count"); got != 0xad {
		t.Errorf("(long) node.digest[1] = %d, want %d", got, 0xad)
	}
	exec(`set result.count = node.digest[3]`)
	if got := num("count"); got != 0xef {
		t.Errorf("node.digest[3] = %d, want %d", got, 0xef)
	}

	// With no symbol table (CheckAction(el, nil), and `dtrules build` on a
	// project without an EDD) an undeclared name is indexed as an array, by
	// the cast form and the bare form alike.
	syms = nil
	exec(`set result.count = (long) node.counts[0]`)
	if got := num("count"); got != 300 {
		t.Errorf("no symbols: (long) node.counts[0] = %d, want 300", got)
	}
	exec(`set result.count = node.counts[1]`)
	if got := num("count"); got != 41 {
		t.Errorf("no symbols: node.counts[1] = %d, want 41", got)
	}
}
