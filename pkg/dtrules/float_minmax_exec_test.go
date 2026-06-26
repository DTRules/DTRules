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

// TestFloatMinMaxExecution (#889) fills the execution gap for the float
// min/max family: `the minimum/maximum of <double> and <double>` lowers to
// fmin/fmax. The fractional values verify a double-precision result (an
// integer min/max would truncate).
func TestFloatMinMaxExecution(t *testing.T) {
	rs := session.NewRuleSet("fmm")
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
	for _, f := range []string{"d1", "d2", "res"} {
		rootRef.AddAttribute(dtrules.GetRName(f), "", dtrules.GetRDoubleValue(0), true, true, dtrules.TypeDouble, "", "", "", "")
	}
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"d1": "double", "d2": "double", "res": "double"})

	run := func(expr string, a, b float64) float64 {
		t.Helper()
		root.Put(dtrules.GetRName("d1"), dtrules.GetRDoubleValue(a))
		root.Put(dtrules.GetRName("d2"), dtrules.GetRDoubleValue(b))
		pf, err := elc.CompileAction("set res = " + expr)
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
		v, _ := root.Get(dtrules.GetRName("res"))
		f, _ := v.DoubleValue()
		return f
	}

	for _, c := range []struct {
		expr    string
		a, b    float64
		want    float64
	}{
		{"the minimum of d1 and d2", 1.5, 2.5, 1.5},
		{"the minimum of d1 and d2", 2.5, 1.5, 1.5},
		{"the maximum of d1 and d2", 1.5, 2.5, 2.5},
		{"the maximum of d1 and d2", 2.5, 1.5, 2.5},
		{"the minimum of d1 and d2", -0.25, 0.25, -0.25},
	} {
		if got := run(c.expr, c.a, c.b); got != c.want {
			t.Errorf("%q (%v,%v) = %v, want %v", c.expr, c.a, c.b, got, c.want)
		}
	}
}
