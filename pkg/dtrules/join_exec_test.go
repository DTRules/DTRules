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

// TestJoinExecution (#1234): `join <array> by <sep>` is the inverse of
// `tokenize`. Joining used to need a loop that left a trailing separator.
func TestJoinExecution(t *testing.T) {
	rs := session.NewRuleSet("i1234")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	name := dtrules.GetRName("doc")
	ref, err := ef.FindCreateRefEntity(true, name)
	if err != nil {
		t.Fatal(err)
	}
	ref.AddAttribute(dtrules.GetRName("words"), "", nil, true, true, dtrules.TypeArray, "", "", "", "")
	ref.AddAttribute(dtrules.GetRName("nums"), "", nil, true, true, dtrules.TypeArray, "", "", "", "")
	ref.AddAttribute(dtrules.GetRName("line"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	ref.AddAttribute(dtrules.GetRName("out"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	root, err := ef.CreateEntity(sess, name)
	if err != nil {
		t.Fatal(err)
	}
	words, _ := dtrules.NewArrayWithElements(sess, false, nil, false)
	root.Put(dtrules.GetRName("words"), words)
	nums, _ := dtrules.NewArrayWithElements(sess, false, []dtrules.Object{
		dtrules.GetRIntegerValue(3), dtrules.GetRIntegerValue(1), dtrules.GetRIntegerValue(4),
	}, false)
	root.Put(dtrules.GetRName("nums"), nums)

	state := sess.GetState().(*interpreter.DTState)
	symbols := map[string]string{"words": "array", "nums": "array", "line": "string", "out": "string"}
	exec := func(action string) string {
		t.Helper()
		elc := el.NewCompiler()
		elc.SetSymbols(symbols)
		pf, err := elc.CompileAction(action)
		if err != nil {
			t.Fatalf("%q compile: %v", action, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", action, pf, err)
		}
		state.EntityPush(root)
		err = obj.Execute(state)
		state.EntityPop()
		if err != nil {
			t.Fatalf("%q execute %q: %v", action, pf, err)
		}
		v, _ := root.Get(dtrules.GetRName("out"))
		return v.StringValue()
	}

	cases := []struct{ action, want string }{
		{`set out = join words by ", "`, ""}, // empty array
		{`add "a" to words; set out = join words by ", "`, "a"},
		{`add "b" to words; add "c" to words; set out = join words by ", "`, "a, b, c"},
		{`set out = join words by ""`, "abc"},
		{`set out = join nums by "-"`, "3-1-4"},
		{`set out = "[" + (join words by "|") + "]"`, "[a|b|c]"},
		{`set out = join tokenize "x;y;z" by ";" by "/"`, "x/y/z"},
		// Precedence: the separator binds as a primary, so `+` after it
		// concatenates onto the joined string, and a compound separator
		// needs parentheses.
		{`set out = "[" + join words by "|" + "]"`, "[a|b|c]"},
		{`set out = join words by "|" + "]"`, "a|b|c]"},
		{`set out = join words by ("|" + "-")`, "a|-b|-c"},
	}
	for _, c := range cases {
		if got := exec(c.action); got != c.want {
			t.Errorf("%s: got %q, want %q", c.action, got, c.want)
		}
	}

	// The inverse of tokenize, including empty fields and a separator at
	// either end.
	for _, line := range []string{"a,b,c", "a,,b", ",a,", "", "no separator"} {
		root.Put(dtrules.GetRName("line"), dtrules.NewRString(line))
		if got := exec(`set out = join (tokenize line by ",") by ","`); got != line {
			t.Errorf("join(tokenize(%q)) = %q", line, got)
		}
	}
}
