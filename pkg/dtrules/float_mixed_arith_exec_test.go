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

// TestFloatMixedArithExecution (#884): adding/subtracting a float operand
// compiled to the integer `+`/`-`, truncating the fraction at runtime
// (`db2 + 1.0` with db2=3.5 gave 4, not 4.5). The mixed add/sub visitors now
// promote through the float, emitting f+/f-. This runs each form and asserts
// the exact double — a truncating op would fail the fractional results.
func TestFloatMixedArithExecution(t *testing.T) {
	rs := session.NewRuleSet("fmix")
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
	rootRef.AddAttribute(dtrules.GetRName("db"), "", dtrules.GetRDoubleValue(0), true, true, dtrules.TypeDouble, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("db2"), "", dtrules.GetRDoubleValue(0), true, true, dtrules.TypeDouble, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("it"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("db2"), dtrules.GetRDoubleValue(3.5))
	root.Put(dtrules.GetRName("it"), dtrules.GetRIntegerValue(2))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"db": "double", "db2": "double", "it": "integer"})

	run := func(expr string) float64 {
		t.Helper()
		postfix, err := elc.CompileAction("set db = " + expr)
		if err != nil {
			t.Fatalf("%q compile: %v", expr, err)
		}
		obj, err := sess.Compile(postfix)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", expr, postfix, err)
		}
		state.EntityPush(root)
		if err := obj.Execute(state); err != nil {
			state.EntityPop()
			t.Fatalf("%q execute %q: %v", expr, postfix, err)
		}
		state.EntityPop()
		v, _ := root.Get(dtrules.GetRName("db"))
		f, _ := v.DoubleValue()
		return f
	}

	for _, c := range []struct {
		expr string
		want float64
	}{
		{"db2 + 1.0", 4.5},  // double + float-literal (was 4 via integer +)
		{"db2 - 1.0", 2.5},  // double - float-literal
		{"1.0 + db2", 4.5},  // float-literal + double
		{"it + 1.5", 3.5},   // integer + float-literal (int promoted)
		{"2.5 + 1.25", 3.75}, // float + float
	} {
		if got := run(c.expr); got != c.want {
			t.Errorf("%q = %v, want %v", c.expr, got, c.want)
		}
	}
}
