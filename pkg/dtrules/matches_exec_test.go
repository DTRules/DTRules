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

// TestMatchesSubjectThenPattern (#1228): `s matches p` is true when the
// string s matches the regular expression p — the left operand is the
// subject, the right the pattern, as the EL reference documents. The EL is
// compiled and executed, so the emitter's operand order and the operator's
// pop order are checked together.
func TestMatchesSubjectThenPattern(t *testing.T) {
	rs := session.NewRuleSet("matches")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	rootRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("b"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("id"), dtrules.NewRString("e11.4362a"))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"id": "string", "b": "boolean"})

	run := func(pf string) {
		t.Helper()
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("assemble %q: %v", pf, err)
		}
		state.EntityPush(root)
		defer state.EntityPop()
		if err := obj.Execute(state); err != nil {
			t.Fatalf("execute %q: %v", pf, err)
		}
	}
	action := func(src string) bool {
		t.Helper()
		pf, err := elc.CompileAction(src)
		if err != nil {
			t.Fatalf("%q compile: %v", src, err)
		}
		run(pf)
		v, _ := root.Get(dtrules.GetRName("b"))
		got, _ := v.BooleanValue()
		return got
	}
	condition := func(src string) bool {
		t.Helper()
		pf, err := elc.CompileCondition(src)
		if err != nil {
			t.Fatalf("%q compile: %v", src, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("assemble %q: %v", pf, err)
		}
		state.EntityPush(root)
		defer state.EntityPop()
		if err := obj.Execute(state); err != nil {
			t.Fatalf("execute %q: %v", pf, err)
		}
		v, err := state.DataPop()
		if err != nil {
			t.Fatalf("%q left nothing on the data stack: %v", src, err)
		}
		got, _ := v.BooleanValue()
		return got
	}

	for _, tc := range []struct {
		src  string
		want bool
	}{
		{`"abc" matches "^a"`, true},
		{`"^a" matches "abc"`, false},
		{`id matches "^[A-Za-z0-9._-]+$"`, true},
		{`"^[A-Za-z0-9._-]+$" matches id`, false},
		{`id matches "^e11\."`, true},
		{`id matches "^x"`, false},
	} {
		if got := action("set b = " + tc.src); got != tc.want {
			t.Errorf("action: set b = %s -> %v, want %v", tc.src, got, tc.want)
		}
		if got := condition(tc.src); got != tc.want {
			t.Errorf("condition: %s -> %v, want %v", tc.src, got, tc.want)
		}
	}
}
