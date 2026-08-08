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

// TestFloatNegateExecution (#878): negating a float expression compiled to the
// unregistered op `neg`, which crashed at runtime with operator-not-found.
// VisitFloatNegate now emits `fnegate` (opFNegate, via DoubleValue) — `negate`
// would be wrong here because it truncates the double via IntValue. This
// compiles `-(db2 + 1.0)` and runs it, asserting the exact double result
// (which a truncating negate would not produce).
func TestFloatNegateExecution(t *testing.T) {
	rs := session.NewRuleSet("fneg")
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

	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("db2"), dtrules.GetRDoubleValue(3.5))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"db": "double", "db2": "double"})

	// `db2 * 2.0` lowers to the float `fmul`, so the only thing under test is
	// the negation: -(3.5 * 2.0) = -7.0. A truncating `negate` would yield -7.0
	// here only by luck, so a fractional case is also checked below.
	postfix, err := elc.CompileAction("set db = -(db2 * 2.0)")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	obj, err := sess.Compile(postfix)
	if err != nil {
		t.Fatalf("assemble postfix %q: %v", postfix, err)
	}
	state.EntityPush(root)
	if err := obj.Execute(state); err != nil {
		state.EntityPop()
		t.Fatalf("execute %q: %v", postfix, err)
	}
	state.EntityPop()

	val, err := root.Get(dtrules.GetRName("db"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := val.DoubleValue()
	if err != nil {
		t.Fatal(err)
	}
	if got != -7.0 {
		t.Errorf("-(db2 * 2.0) = %v, want -7.0", got)
	}

	// Negating a fractional float literal: -2.5. A truncating negate would
	// give -2.0, proving fnegate (not negate) is in effect.
	pf2, err := elc.CompileAction("set db = -2.5")
	if err != nil {
		t.Fatalf("compile -2.5: %v", err)
	}
	obj2, err := sess.Compile(pf2)
	if err != nil {
		t.Fatalf("assemble %q: %v", pf2, err)
	}
	state.EntityPush(root)
	if err := obj2.Execute(state); err != nil {
		state.EntityPop()
		t.Fatalf("execute %q: %v", pf2, err)
	}
	state.EntityPop()
	v2, _ := root.Get(dtrules.GetRName("db"))
	g2, _ := v2.DoubleValue()
	if g2 != -2.5 {
		t.Errorf("-2.5 = %v, want -2.5 (a truncating negate would give -2.0)", g2)
	}
}
