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

// TestTodayInZoneExecution (#1273): `today in zone "America/Chicago"` is
// midnight of today's Chicago date, in Chicago's zone. It used to be UTC
// midnight rewrapped, which in Chicago reads as 18:00 or 19:00 the day
// before.
func TestTodayInZoneExecution(t *testing.T) {
	rs := session.NewRuleSet("t1273")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	name := dtrules.GetRName("clock")
	ref, err := ef.FindCreateRefEntity(true, name)
	if err != nil {
		t.Fatal(err)
	}
	ref.AddAttribute(dtrules.GetRName("d"), "", nil, true, true, dtrules.TypeDate, "", "", "", "")
	root, err := ef.CreateEntity(sess, name)
	if err != nil {
		t.Fatal(err)
	}
	loc, err := dtrules.ResolveZone("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}

	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"d": "date"})
	pf, err := elc.CompileAction(`set d = today in zone "America/Chicago"`)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := sess.Compile(pf)
	if err != nil {
		t.Fatalf("assemble %q: %v", pf, err)
	}
	before := time.Now().In(loc)
	state := sess.GetState().(*interpreter.DTState)
	state.EntityPush(root)
	err = obj.Execute(state)
	state.EntityPop()
	after := time.Now().In(loc)
	if err != nil {
		t.Fatalf("execute %q: %v", pf, err)
	}
	v, _ := root.Get(dtrules.GetRName("d"))
	got, err := v.TimeValue()
	if err != nil {
		t.Fatal(err)
	}
	local := got.In(loc)
	if local.Hour() != 0 || local.Minute() != 0 || local.Second() != 0 {
		t.Errorf("today in zone Chicago = %s, want midnight in Chicago", local.Format(time.RFC3339))
	}
	y, m, d := local.Date()
	by, bm, bd := before.Date()
	ay, am, ad := after.Date()
	if (y != by || m != bm || d != bd) && (y != ay || m != am || d != ad) {
		t.Errorf("today in zone Chicago = %s, want Chicago's date %s", local.Format("2006-01-02"), before.Format("2006-01-02"))
	}
}
