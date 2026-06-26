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
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestDateBetweenExecution (#888): `<date> is between <a> and <b>` lowers to
// `d>= … d<= and`. The d<= and d>= ops were never registered, so this crashed
// at runtime with "RName 'd<=' was not defined" — yet token-presence tests
// passed because the postfix string was produced fine. This compiles the EL
// and RUNS it, asserting the boolean for in-range, out-of-range, and the
// inclusive endpoints.
func TestDateBetweenExecution(t *testing.T) {
	rs := session.NewRuleSet("dbtw")
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
	for _, f := range []string{"d1", "lo", "hi"} {
		rootRef.AddAttribute(dtrules.GetRName(f), "", nil, true, true, dtrules.TypeDate, "", "", "", "")
	}
	rootRef.AddAttribute(dtrules.GetRName("inrange"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")

	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	date := func(y, m, d int) dtrules.Object {
		return dtrules.GetRTime(time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC))
	}
	root.Put(dtrules.GetRName("lo"), date(2024, 1, 1))
	root.Put(dtrules.GetRName("hi"), date(2024, 12, 31))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"d1": "date", "lo": "date", "hi": "date", "inrange": "boolean"})
	postfix, err := elc.CompileAction("set inrange = d1 is between lo and hi")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	obj, err := sess.Compile(postfix)
	if err != nil {
		t.Fatalf("assemble %q: %v", postfix, err)
	}

	eval := func(y, m, d int) bool {
		t.Helper()
		root.Put(dtrules.GetRName("d1"), date(y, m, d))
		state.EntityPush(root)
		if err := obj.Execute(state); err != nil {
			state.EntityPop()
			t.Fatalf("execute %q: %v", postfix, err)
		}
		state.EntityPop()
		v, _ := root.Get(dtrules.GetRName("inrange"))
		b, _ := v.BooleanValue()
		return b
	}

	for _, c := range []struct {
		y, m, d int
		want    bool
	}{
		{2024, 6, 15, true},   // inside
		{2023, 12, 31, false}, // before lo
		{2025, 1, 1, false},   // after hi
		{2024, 1, 1, true},    // == lo (inclusive, exercises d>=)
		{2024, 12, 31, true},  // == hi (inclusive, exercises d<=)
	} {
		if got := eval(c.y, c.m, c.d); got != c.want {
			t.Errorf("%d-%02d-%02d between 2024-01-01 and 2024-12-31 = %v, want %v", c.y, c.m, c.d, got, c.want)
		}
	}
}
