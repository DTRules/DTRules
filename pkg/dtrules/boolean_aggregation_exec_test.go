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

// TestBooleanAggregationExecution exercises the forall boolean-aggregation
// constructs (#877, the follow-up #867 explicitly flagged) end to end. Like the
// numeric folds they lower to `seed { body op } arr forall`. Token-only tests
// cannot catch a reversed forall (the tokens are identical) — this compiles the
// EL, runs it over real entities, and asserts the boolean result. Before the
// operand-order fix these throw "non-Entity entry in array" at runtime because
// forall iterates the body block instead of the array.
func TestBooleanAggregationExecution(t *testing.T) {
	rs := session.NewRuleSet("boolagg")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	// kid { tax: integer }
	kidName := dtrules.GetRName("kid")
	kidRef, err := ef.FindCreateRefEntity(true, kidName)
	if err != nil {
		t.Fatal(err)
	}
	kidRef.AddAttribute(dtrules.GetRName("tax"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	// root { kids: array of kid; flag: boolean }
	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	rootRef.AddAttribute(dtrules.GetRName("kids"), "", nil, true, true, dtrules.TypeArray, "kid", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("flag"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")

	// kids = [5, -3, 10, 0, 7].
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
	elc.SetSymbols(map[string]string{"kids": "array", "tax": "integer", "flag": "boolean"})

	run := func(expr string) (bool, error) {
		postfix, err := elc.CompileAction("set flag = " + expr)
		if err != nil {
			return false, err
		}
		obj, err := sess.Compile(postfix)
		if err != nil {
			return false, err
		}
		state.EntityPush(root)
		if err := obj.Execute(state); err != nil {
			state.EntityPop()
			return false, err
		}
		state.EntityPop()
		val, err := root.Get(dtrules.GetRName("flag"))
		if err != nil || val == nil {
			return false, err
		}
		return val.BooleanValue()
	}

	for _, c := range []struct {
		expr string
		want bool
	}{
		{"all kids have tax > 0", false},        // -3 and 0 fail
		{"all kids have tax >= -3", true},       // all in range
		{"one of kids has a tax > 5", true},     // 10, 7
		{"one of kids has a tax > 100", false},  // none
	} {
		got, err := run(c.expr)
		if err != nil {
			t.Errorf("%q: exec error: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %v, want %v", c.expr, got, c.want)
		}
	}
}
