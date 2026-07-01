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

// TestDateArithExecution (#888/#889): `<date> - N days/months/years` emitted
// nonexistent `subdays/submonths/subyears` ops (crash); the fix emits
// `negate add*`. The `+ N` forms use add* directly. This runs both directions
// and asserts the resulting date components — the subtraction path had no
// execution test.
func TestDateArithExecution(t *testing.T) {
	rs := session.NewRuleSet("da")
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
	rootRef.AddAttribute(dtrules.GetRName("dt"), "", nil, true, true, dtrules.TypeDate, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("out"), "", nil, true, true, dtrules.TypeDate, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"dt": "date", "out": "date"})

	run := func(expr string) time.Time {
		t.Helper()
		root.Put(dtrules.GetRName("dt"), dtrules.GetRTime(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)))
		pf, err := elc.CompileAction("set out = " + expr)
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
		v, _ := root.Get(dtrules.GetRName("out"))
		got, err := v.TimeValue()
		if err != nil {
			t.Fatalf("%q TimeValue: %v", expr, err)
		}
		return got
	}

	for _, c := range []struct {
		expr             string
		wy, wm, wd       int
	}{
		{"dt - 5 days", 2024, 6, 10},   // subtraction (was a crash: subdays)
		{"dt + 5 days", 2024, 6, 20},   // addition (control)
		{"dt - 1 months", 2024, 5, 15}, // submonths -> negate addmonths
		{"dt - 1 years", 2023, 6, 15},  // subyears -> negate addyears
	} {
		got := run(c.expr)
		if got.Year() != c.wy || int(got.Month()) != c.wm || got.Day() != c.wd {
			t.Errorf("%q = %04d-%02d-%02d, want %04d-%02d-%02d",
				c.expr, got.Year(), int(got.Month()), got.Day(), c.wy, c.wm, c.wd)
		}
	}
}
