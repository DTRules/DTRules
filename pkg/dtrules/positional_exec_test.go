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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestPositionalEntityAccess pins #1022's two smaller holes end to end:
// `first of <array>` (parity with the accidental `last of`) and
// `<array>[<i>]` in an entity context, which used to lower to `bytesidx` —
// a bytes accessor — and, with an appended conversion, could assign an
// integer where an entity was meant. Both must now yield the actual
// element entity, by identity.
func TestPositionalEntityAccess(t *testing.T) {
	rs := session.NewRuleSet("positional")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	cardName := dtrules.GetRName("card")
	cardRef, err := ef.FindCreateRefEntity(true, cardName)
	if err != nil {
		t.Fatal(err)
	}
	cardRef.AddAttribute(dtrules.GetRName("rank"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	rootRef.AddAttribute(dtrules.GetRName("cards"), "", nil, true, true, dtrules.TypeArray, "card", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("top"), "", nil, true, true, dtrules.TypeEntity, "card", "", "", "")

	ids := make([]int, 0, 3)
	elems := make([]dtrules.Object, 0, 3)
	for i, r := range []int{7, 11, 3} {
		c, err := ef.CreateEntity(sess, cardName)
		if err != nil {
			t.Fatal(err)
		}
		c.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(r)))
		ids = append(ids, c.GetID())
		elems = append(elems, c)
		_ = i
	}
	arr, err := dtrules.NewArrayWithElements(sess, true, elems, false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("cards"), arr)

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"cards": "array", "top": "entity", "rank": "integer"})

	run := func(dsl string) int {
		t.Helper()
		postfix, err := elc.CompileAction(dsl)
		if err != nil {
			t.Fatalf("compile %q: %v", dsl, err)
		}
		if strings.Contains(postfix, "bytesidx") {
			t.Fatalf("%q lowered to the bytes accessor: %q", dsl, postfix)
		}
		obj, err := sess.Compile(postfix)
		if err != nil {
			t.Fatalf("assemble %q (postfix %q): %v", dsl, postfix, err)
		}
		state.EntityPush(root)
		defer state.EntityPop()
		if err := obj.Execute(state); err != nil {
			t.Fatalf("execute %q (postfix %q): %v", dsl, postfix, err)
		}
		v, err := root.Get(dtrules.GetRName("top"))
		if err != nil || v == nil {
			t.Fatalf("get top after %q: %v", dsl, err)
		}
		ent, err := v.REntityValue()
		if err != nil {
			t.Fatalf("top is not an entity after %q: %v (value %v)", dsl, err, v)
		}
		return ent.GetID()
	}

	if got := run(`set top = first of cards`); got != ids[0] {
		t.Errorf("first of cards: got entity id %d, want %d", got, ids[0])
	}
	if got := run(`set top = cards[1]`); got != ids[1] {
		t.Errorf("cards[1]: got entity id %d, want %d", got, ids[1])
	}
	if got := run(`set top = cards[0]`); got != ids[0] {
		t.Errorf("cards[0]: got entity id %d, want %d", got, ids[0])
	}
}
