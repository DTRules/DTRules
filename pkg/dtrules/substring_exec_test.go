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

// TestSubstringExecution (#889): `substring of s from <start> to <end>` is
// documented as "extract characters start..end-1" (end exclusive, by character
// index). The emitter passed the END index straight to opSubstring, whose
// third operand is a LENGTH — so the result was only correct when start==0
// (where end == length). A token test can't catch it: the tokens are identical.
// This runs the construct and asserts the extracted string for start>0.
func TestSubstringExecution(t *testing.T) {
	rs := session.NewRuleSet("sub")
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
	rootRef.AddAttribute(dtrules.GetRName("src"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("dst"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("src"), dtrules.NewRString("hello"))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"src": "string", "dst": "string"})

	run := func(expr string) string {
		t.Helper()
		pf, err := elc.CompileAction("set dst = " + expr)
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
		v, _ := root.Get(dtrules.GetRName("dst"))
		return v.StringValue()
	}

	for _, c := range []struct {
		expr, want string
	}{
		{"substring of src from 0 to 5", "hello"}, // full (masks the bug)
		{"substring of src from 0 to 2", "he"},
		{"substring of src from 2 to 4", "ll"},  // was "llo" (end used as length)
		{"substring of src from 1 to 3", "el"},  // was "ell"
		{"substring of src from 3 to 5", "lo"},
	} {
		if got := run(c.expr); got != c.want {
			t.Errorf("%q = %q, want %q", c.expr, got, c.want)
		}
	}
}
