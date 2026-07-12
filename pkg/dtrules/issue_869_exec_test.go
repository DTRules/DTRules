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

// TestIssue869ThereIsInArrayExecution (#869) covers the `there is <x> in
// <array> where <p>` surface end to end. The grammar's entity alternative
// shadows boolThereIsInArrayWhere (typedEntity and typedArray both come from
// IDENT), so an array operand used to compile to `<arr> entitypush …` and
// crash at runtime when entitypush called REntityValue on the array. The
// emitter now routes by declared type to the OR-accumulator fold.
func TestIssue869ThereIsInArrayExecution(t *testing.T) {
	rs := session.NewRuleSet("i869")
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
		// Positive form, all three inthe spellings (in/for/on).
		{"there is k in kids where tax > 5", true},    // 10, 7
		{"there is k in kids where tax > 100", false}, // none
		{"there is k for kids where tax < 0", true},   // -3
		{"there is k on kids where tax == 0", true},   // 0
		{"is there k in kids where tax > 9", true},    // 10
		// Negative form.
		{"there is no k in kids where tax > 100", true},
		{"there is no k in kids where tax > 5", false},
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
