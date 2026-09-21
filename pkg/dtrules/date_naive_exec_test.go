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

// TestNaiveTimestampCastExecution (#1275): `(date) "2020-01-01T10:00:00"`
// compiled and then cast to null at run time, because the session's date
// parser did not know the form. It is 10:00 UTC.
func TestNaiveTimestampCastExecution(t *testing.T) {
	rs := session.NewRuleSet("t1275")
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
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"d": "date"})
	pf, err := elc.CompileAction(`set d = (date) "2020-01-01T10:00:00"`)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := sess.Compile(pf)
	if err != nil {
		t.Fatalf("assemble %q: %v", pf, err)
	}
	state := sess.GetState().(*interpreter.DTState)
	state.EntityPush(root)
	err = obj.Execute(state)
	state.EntityPop()
	if err != nil {
		t.Fatalf("execute %q: %v", pf, err)
	}
	v, _ := root.Get(dtrules.GetRName("d"))
	got, err := v.TimeValue()
	if err != nil {
		t.Fatalf("d is %T %q, not a date: %v", v, v.StringValue(), err)
	}
	if want := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("d = %s, want %s", got, want)
	}
}
