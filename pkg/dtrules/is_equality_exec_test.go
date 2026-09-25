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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const isEqualityEDD = `<entity_data_dictionary version="2">
<file_path>is_edd</file_path>
<entity name="v" number="100" access="rw">
<field name="b1" type="bytes" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="b2" type="bytes" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="b3" type="bytes" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="f1" type="fixed" subtype="" access="rw" input="" default_value="0" comment=""></field>
<field name="f2" type="fixed" subtype="" access="rw" input="" default_value="0" comment=""></field>
<field name="s1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="s2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
<field name="flag" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
</entity>
</entity_data_dictionary>`

// TestIsComparesLikeEquals (#1310): `a is b` and `a is not b` on two
// identifiers are the comparison `a == b` and `a != b` are, typed from the
// EDD. They parsed as a string comparison and emitted streq for every type,
// so bytes lost the constant-time bytes== the `==` form gets. Compiled as
// `dtrules build` compiles (EDD symbols, CheckAction) and executed.
func TestIsComparesLikeEquals(t *testing.T) {
	dir := t.TempDir()
	eddPath := filepath.Join(dir, "is_edd.xml")
	if err := os.WriteFile(eddPath, []byte(isEqualityEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	symbols := authoring.LoadEDDSymbols(dir)

	rs := session.NewRuleSet("is")
	if err := rs.LoadEDDFile(eddPath); err != nil {
		t.Fatalf("load edd: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	v, err := sess.CreateEntity(dtrules.GetRName("v"))
	if err != nil {
		t.Fatal(err)
	}
	put := func(field string, val dtrules.Object) {
		t.Helper()
		if err := v.Put(dtrules.GetRName(field), val); err != nil {
			t.Fatal(err)
		}
	}
	fixed := func(s string) dtrules.Object {
		t.Helper()
		f, err := dtrules.GetRFixedFromString(s)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	put("b1", dtrules.NewRBytes([]byte{0xde, 0xad}))
	put("b2", dtrules.NewRBytes([]byte{0xde, 0xad}))
	put("b3", dtrules.NewRBytes([]byte{0xbe, 0xef}))
	put("f1", fixed("1.50"))
	put("f2", fixed("1.5"))
	put("s1", dtrules.NewRString("x"))
	put("s2", dtrules.NewRString("y"))

	run := func(cond string) (bool, string) {
		t.Helper()
		postfix, err := authoring.CheckAction("set flag = "+cond, symbols)
		if err != nil {
			t.Fatalf("%q compile: %v", cond, err)
		}
		obj, err := sess.Compile(postfix)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", cond, postfix, err)
		}
		state.EntityPush(v)
		defer state.EntityPop()
		if err := obj.Execute(state); err != nil {
			t.Fatalf("%q execute %q: %v", cond, postfix, err)
		}
		got, err := v.Get(dtrules.GetRName("flag"))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := got.BooleanValue()
		return b, postfix
	}

	for _, c := range []struct {
		cond   string
		want   bool
		wantOp string
	}{
		{"b1 is b2", true, "bytes=="},
		{"b1 is b3", false, "bytes=="},
		{"b1 is not b3", true, "bytes!="},
		{"b1 is not b2", false, "bytes!="},
		{"f1 is f2", true, "fp=="},
		{"f1 is not f2", false, "fp!="},
		{"s1 is s2", false, "streq"},
		{"s1 is not s2", true, "streq"},
	} {
		got, postfix := run(c.cond)
		if got != c.want {
			t.Errorf("%q = %v, want %v (postfix %q)", c.cond, got, c.want, postfix)
		}
		if !strings.Contains(postfix, c.wantOp) {
			t.Errorf("%q compiled to %q, want %s", c.cond, postfix, c.wantOp)
		}
		eq := strings.Replace(strings.Replace(c.cond, " is not ", " != ", 1), " is ", " == ", 1)
		if _, eqPostfix := run(eq); eqPostfix != postfix {
			t.Errorf("%q compiled to %q but %q to %q; they are one comparison", c.cond, postfix, eq, eqPostfix)
		}
	}
}
