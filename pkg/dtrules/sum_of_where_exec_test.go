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

// TestSumOfWhereExecution runs the new `sum of … in … where …` (#864) over real
// entities and asserts the numeric result. Token-only tests can't catch the
// forall/if operand order; this throws at runtime if the order is wrong.
func TestSumOfWhereExecution(t *testing.T) {
	rs := session.NewRuleSet("sow")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	kidName := dtrules.GetRName("kid")
	kidRef, err := ef.FindCreateRefEntity(true, kidName)
	if err != nil {
		t.Fatal(err)
	}
	kidRef.AddAttribute(dtrules.GetRName("tax"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	rootRef.AddAttribute(dtrules.GetRName("kids"), "", nil, true, true, dtrules.TypeArray, "kid", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("total"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	// kids = [5, -3, 10, 0, 7]: positives sum to 22.
	elems := make([]dtrules.Object, 0, 5)
	for _, v := range []int{5, -3, 10, 0, 7} {
		k, err := ef.CreateEntity(sess, kidName)
		if err != nil {
			t.Fatal(err)
		}
		k.Put(dtrules.GetRName("tax"), dtrules.GetRIntegerValue(int64(v)))
		elems = append(elems, k)
	}
	arr, err := dtrules.NewArrayWithElements(sess, true, elems, false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("kids"), arr)

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"kids": "array", "tax": "integer", "total": "integer"})

	run := func(expr string) (int, error) {
		postfix, err := elc.CompileAction("set total = " + expr)
		if err != nil {
			return 0, err
		}
		obj, err := sess.Compile(postfix)
		if err != nil {
			return 0, err
		}
		state.EntityPush(root)
		if err := obj.Execute(state); err != nil {
			state.EntityPop()
			return 0, err
		}
		state.EntityPop()
		val, err := root.Get(dtrules.GetRName("total"))
		if err != nil || val == nil {
			return 0, err
		}
		return val.IntValue()
	}

	for _, c := range []struct {
		expr string
		want int
	}{
		{"sum of tax in kids where tax > 0", 22},   // 5+10+7
		{"sum of tax in kids where tax > 100", 0},  // predicate always false → balanced 0
	} {
		got, err := run(c.expr)
		if err != nil {
			t.Errorf("%q: exec error: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %d, want %d", c.expr, got, c.want)
		}
	}
}
