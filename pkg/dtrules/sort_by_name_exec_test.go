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
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestSortByTheNameExecution: the form the #1227 compile error points at,
// `by the name "<field>"`, sorts entities by a non-keyword string field and
// by a date field when run. The hint is only worth giving if it works.
func TestSortByTheNameExecution(t *testing.T) {
	rs := session.NewRuleSet("i1227")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	entryName := dtrules.GetRName("entry")
	entryRef, err := ef.FindCreateRefEntity(true, entryName)
	if err != nil {
		t.Fatal(err)
	}
	entryRef.AddAttribute(dtrules.GetRName("key"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	entryRef.AddAttribute(dtrules.GetRName("when"), "", nil, true, true, dtrules.TypeDate, "", "", "", "")

	stateName := dtrules.GetRName("state")
	stateRef, err := ef.FindCreateRefEntity(true, stateName)
	if err != nil {
		t.Fatal(err)
	}
	stateRef.AddAttribute(dtrules.GetRName("entries"), "", nil, true, true, dtrules.TypeArray, "entry", "", "", "")

	root, err := ef.CreateEntity(sess, stateName)
	if err != nil {
		t.Fatal(err)
	}
	arr, err := dtrules.NewArrayWithElements(sess, true, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// Unsorted on both keys, and the two orders differ.
	for _, e := range []struct {
		key  string
		when time.Time
	}{
		{"pear", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
		{"apple", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"fig", time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
	} {
		ent, err := ef.CreateEntity(sess, entryName)
		if err != nil {
			t.Fatal(err)
		}
		ent.Put(dtrules.GetRName("key"), dtrules.NewRString(e.key))
		ent.Put(dtrules.GetRName("when"), dtrules.GetRTime(e.when))
		arr.Add(ent)
	}
	root.Put(dtrules.GetRName("entries"), arr)

	state := sess.GetState().(*interpreter.DTState)
	symbols := map[string]string{
		"state.entries": "array", "entries": "array",
		"entry.key": "string", "key": "string",
		"entry.when": "date", "when": "date",
	}
	keys := func() string {
		v, _ := root.Get(dtrules.GetRName("entries"))
		a, _ := v.RArrayValue()
		var out []string
		for _, o := range a.GetIterator() {
			e, _ := o.REntityValue()
			k, _ := e.Get(dtrules.GetRName("key"))
			out = append(out, k.StringValue())
		}
		return strings.Join(out, ",")
	}

	for _, tc := range []struct{ action, want string }{
		{`sort state.entries in ascending order by the name "key"`, "apple,fig,pear"},
		{`sort state.entries in descending order by the name "key"`, "pear,fig,apple"},
		{`sort state.entries in descending order by the name "when"`, "fig,pear,apple"},
		{`sort state.entries in ascending order by (name) "when"`, "apple,pear,fig"},
	} {
		elc := el.NewCompiler()
		elc.SetSymbols(symbols)
		pf, err := elc.CompileAction(tc.action)
		if err != nil {
			t.Fatalf("%q compile: %v", tc.action, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", tc.action, pf, err)
		}
		state.EntityPush(root)
		err = obj.Execute(state)
		state.EntityPop()
		if err != nil {
			t.Fatalf("%q execute %q: %v", tc.action, pf, err)
		}
		if got := keys(); got != tc.want {
			t.Errorf("%q: order %s, want %s", tc.action, got, tc.want)
		}
	}
}
