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

// TestDoubleFieldNegateExecution (#894): negating a bare double field used to
// emit integer `negate`, truncating the fraction. It now emits `fnegate`.
// -3.5 must stay -3.5 (a truncating negate gives -3).
func TestDoubleFieldNegateExecution(t *testing.T) {
	rs := session.NewRuleSet("dfn")
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
	rootRef.AddAttribute(dtrules.GetRName("out"), "", dtrules.GetRDoubleValue(0), true, true, dtrules.TypeDouble, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("db"), dtrules.GetRDoubleValue(3.5))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"db": "double", "out": "double"})
	pf, err := elc.CompileAction("set out = - db")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	obj, err := sess.Compile(pf)
	if err != nil {
		t.Fatalf("assemble %q: %v", pf, err)
	}
	state.EntityPush(root)
	if err := obj.Execute(state); err != nil {
		state.EntityPop()
		t.Fatalf("execute %q: %v", pf, err)
	}
	state.EntityPop()
	v, _ := root.Get(dtrules.GetRName("out"))
	got, _ := v.DoubleValue()
	if got != -3.5 {
		t.Errorf("- db (db=3.5) = %v, want -3.5 (a truncating negate gives -3)", got)
	}
}
