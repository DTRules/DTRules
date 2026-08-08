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
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestOrderSensitiveExecution (#897) exercises two-operand / order-sensitive EL
// constructs that had no execution test — a reversed operand order gives wrong
// results a token-presence test can't catch (the substring-bug class).
func TestOrderSensitiveExecution(t *testing.T) {
	rs := session.NewRuleSet("ord")
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
	add := func(n string, ty *dtrules.RType, v dtrules.Object) {
		rootRef.AddAttribute(dtrules.GetRName(n), "", v, true, true, ty, "", "", "", "")
	}
	add("d1", dtrules.TypeDate, nil)
	add("d2", dtrules.TypeDate, nil)
	add("s", dtrules.TypeString, dtrules.NewRString(""))
	add("n", dtrules.TypeInteger, dtrules.GetRIntegerValue(0))
	add("b", dtrules.TypeBoolean, dtrules.GetRBoolean(false))
	add("dout", dtrules.TypeDate, nil)
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("d1"), dtrules.GetRTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))
	root.Put(dtrules.GetRName("d2"), dtrules.GetRTime(time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC)))
	root.Put(dtrules.GetRName("s"), dtrules.NewRString("hello"))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"d1": "date", "d2": "date", "s": "string", "n": "integer", "b": "boolean", "dout": "date"})

	exec := func(action string) {
		t.Helper()
		pf, err := elc.CompileAction(action)
		if err != nil {
			t.Fatalf("%q compile: %v", action, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", action, pf, err)
		}
		state.EntityPush(root)
		if err := obj.Execute(state); err != nil {
			state.EntityPop()
			t.Fatalf("%q execute %q: %v", action, pf, err)
		}
		state.EntityPop()
	}
	getInt := func() int { v, _ := root.Get(dtrules.GetRName("n")); i, _ := v.IntValue(); return i }
	getBool := func() bool { v, _ := root.Get(dtrules.GetRName("b")); x, _ := v.BooleanValue(); return x }
	getDate := func() time.Time { v, _ := root.Get(dtrules.GetRName("dout")); x, _ := v.TimeValue(); return x }

	// days between: d2 - d1 = 14 (order matters: from d1 to d2).
	exec("set n = days from d1 to d2")
	if getInt() != 14 {
		t.Errorf("days from d1 to d2 = %d, want 14", getInt())
	}
	// reverse: from d2 to d1 = -14.
	exec("set n = days from d2 to d1")
	if getInt() != -14 {
		t.Errorf("days from d2 to d1 = %d, want -14", getInt())
	}

	// index of "l" in "hello" = 2 (0-based). Order: needle in haystack.
	exec(`set n = index of "l" in s`)
	if getInt() != 2 {
		t.Errorf(`index of "l" in "hello" = %d, want 2`, getInt())
	}
	// index of missing substring = -1.
	exec(`set n = index of "z" in s`)
	if getInt() != -1 {
		t.Errorf(`index of "z" in "hello" = %d, want -1`, getInt())
	}

	// starts with.
	exec(`set b = s starts with "he"`)
	if !getBool() {
		t.Error(`"hello" starts with "he" should be true`)
	}
	exec(`set b = s starts with "lo"`)
	if getBool() {
		t.Error(`"hello" starts with "lo" should be false`)
	}

	// first/end of month/year on d2 (2024-06-15).
	exec("set dout = first of months of d2")
	if g := getDate(); g.Year() != 2024 || g.Month() != 6 || g.Day() != 1 {
		t.Errorf("first of months = %v, want 2024-06-01", g.Format("2006-01-02"))
	}
	exec("set dout = end of months of d2")
	if g := getDate(); g.Year() != 2024 || g.Month() != 6 || g.Day() != 30 {
		t.Errorf("end of months = %v, want 2024-06-30", g.Format("2006-01-02"))
	}
	exec("set dout = first of years of d2")
	if g := getDate(); g.Year() != 2024 || g.Month() != 1 || g.Day() != 1 {
		t.Errorf("first of years = %v, want 2024-01-01", g.Format("2006-01-02"))
	}

	// in-zone hour: 2024-06-15 14:00 UTC in America/New_York (EDT, UTC-4) = 10.
	exec(`set n = get hourof d2 in zone "America/New_York"`)
	if getInt() != 10 {
		t.Errorf(`hourof d2 in America/New_York = %d, want 10 (14:00 UTC - 4)`, getInt())
	}
}
