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

// TestBigIntMinMaxExecution (#899): `the minimum/maximum of <bigint> and
// <bigint>` used to emit the integer min/max, truncating operands beyond int64
// via IntValue. It now emits bmin/bmax. This runs it over values far past
// int64 and asserts the exact big value survives.
func TestBigIntMinMaxExecution(t *testing.T) {
	rs := session.NewRuleSet("bmm")
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
	zero, _ := dtrules.GetRBigIntFromString("0")
	for _, f := range []string{"big1", "big2", "res"} {
		rootRef.AddAttribute(dtrules.GetRName(f), "", zero, true, true, dtrules.TypeBigInt, "", "", "", "")
	}
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	smallStr := "100000000000000000000000000000"  // 1e29, far beyond int64
	largeStr := "999999999999999999999999999999" // ~1e30
	small, _ := dtrules.GetRBigIntFromString(smallStr)
	large, _ := dtrules.GetRBigIntFromString(largeStr)
	root.Put(dtrules.GetRName("big1"), large)
	root.Put(dtrules.GetRName("big2"), small)

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"big1": "bigint", "big2": "bigint", "res": "bigint"})

	run := func(expr string) string {
		t.Helper()
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
		bi, err := v.RBigIntValue()
		if err != nil {
			t.Fatalf("res not bigint: %v", err)
		}
		return bi.BigIntValue().String()
	}

	if got := run("the minimum of big1 and big2"); got != smallStr {
		t.Errorf("minimum = %s, want %s (truncation would give a different value)", got, smallStr)
	}
	if got := run("the maximum of big1 and big2"); got != largeStr {
		t.Errorf("maximum = %s, want %s", got, largeStr)
	}
}
