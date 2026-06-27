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

// TestStreqExecution (#889): name-path string equality `s == t` lowers to the
// `streq` op. The audit found it had no direct runtime test. This runs string
// == string and == literal, asserting the boolean.
func TestStreqExecution(t *testing.T) {
	rs := session.NewRuleSet("streq")
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
	rootRef.AddAttribute(dtrules.GetRName("s1"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("s2"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("result"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"s1": "string", "s2": "string", "result": "boolean"})

	run := func(expr, v1, v2 string) bool {
		t.Helper()
		root.Put(dtrules.GetRName("s1"), dtrules.NewRString(v1))
		root.Put(dtrules.GetRName("s2"), dtrules.NewRString(v2))
		pf, err := elc.CompileAction("set result = " + expr)
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
		v, _ := root.Get(dtrules.GetRName("result"))
		b, _ := v.BooleanValue()
		return b
	}

	if !run("s1 == s2", "hello", "hello") {
		t.Error(`"hello" == "hello" should be true`)
	}
	if run("s1 == s2", "hello", "world") {
		t.Error(`"hello" == "world" should be false`)
	}
	if !run(`s1 == "match"`, "match", "") {
		t.Error(`s1=="match" should be true`)
	}
	if run(`s1 != s2`, "a", "a") {
		t.Error(`"a" != "a" should be false`)
	}
}
