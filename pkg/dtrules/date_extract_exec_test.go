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

// TestDateExtractExecution (#889) gives the date/time extraction ops their
// first execution coverage (the audit found ~20 with zero tests). Each `get …`
// form lowers to a single-operand op; this runs them over a fixed UTC datetime
// (2024-06-15 14:30:45, a Saturday in a leap year) and asserts the value.
func TestDateExtractExecution(t *testing.T) {
	rs := session.NewRuleSet("dx")
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
	rootRef.AddAttribute(dtrules.GetRName("n"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("dt"), dtrules.GetRTime(time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"dt": "date", "n": "integer"})

	run := func(expr string) int {
		t.Helper()
		pf, err := elc.CompileAction("set n = " + expr)
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
		v, _ := root.Get(dtrules.GetRName("n"))
		got, _ := v.IntValue()
		return got
	}

	// Unambiguous fields.
	for _, c := range []struct {
		expr string
		want int
	}{
		{"get yearof dt", 2024},
		{"get days of months for dt", 15}, // day-of-month
		{"get hourof dt", 14},
		{"get minuteof dt", 30},
		{"get secondof dt", 45},
		{"get days in months for dt", 30},  // June has 30 days
		{"get days in yearof dt", 366},     // 2024 is a leap year
	} {
		if got := run(c.expr); got != c.want {
			t.Errorf("%q = %d, want %d", c.expr, got, c.want)
		}
	}

	// Convention-dependent fields: assert sane ranges (must at least run, not crash).
	if dow := run("get dayofweek of dt"); dow < 0 || dow > 7 {
		t.Errorf("day of week = %d, out of range", dow)
	}
	if woy := run("get weekofyear of dt"); woy < 1 || woy > 53 {
		t.Errorf("week of year = %d, out of range", woy)
	}
}
