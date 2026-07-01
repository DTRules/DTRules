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

// TestMixedCompareExecution (#889): integer-vs-double comparisons lower to the
// float comparison ops (f>, f<, f>=, f<=) with operands in source order. Order
// matters (`i > 1.5` must not become `1.5 > i`); this runs each direction and
// asserts the boolean, which a token test cannot verify.
func TestMixedCompareExecution(t *testing.T) {
	rs := session.NewRuleSet("mc")
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
	rootRef.AddAttribute(dtrules.GetRName("num"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("dbl"), "", dtrules.GetRDoubleValue(0), true, true, dtrules.TypeDouble, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("result"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"num": "integer", "dbl": "double", "result": "boolean"})

	run := func(expr string, iv int64, dv float64) bool {
		t.Helper()
		root.Put(dtrules.GetRName("num"), dtrules.GetRIntegerValue(iv))
		root.Put(dtrules.GetRName("dbl"), dtrules.GetRDoubleValue(dv))
		pf, err := elc.CompileAction("set result = " + expr)
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
		v, _ := root.Get(dtrules.GetRName("result"))
		b, _ := v.BooleanValue()
		return b
	}

	for _, c := range []struct {
		expr   string
		iv     int64
		dv     float64
		want   bool
	}{
		{"num > 1.5", 2, 0, true},   // 2 > 1.5
		{"num > 1.5", 1, 0, false},  // 1 > 1.5
		{"1.5 > num", 0, 0, true},   // 1.5 > 0 (order preserved)
		{"1.5 > num", 2, 0, false},  // 1.5 > 2
		{"dbl > 2", 0, 2.5, true},   // 2.5 > 2
		{"dbl > 2", 0, 1.5, false},  // 1.5 > 2
		{"2 > dbl", 0, 1.5, true},   // 2 > 1.5
		{"num <= 3.0", 3, 0, true},  // 3 <= 3.0
	} {
		if got := run(c.expr, c.iv, c.dv); got != c.want {
			t.Errorf("%q (i=%d,d=%v) = %v, want %v", c.expr, c.iv, c.dv, got, c.want)
		}
	}
}
