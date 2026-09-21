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
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// #1233: a Go caller asks the state whether an execution changed entity
// data, through dtrules.ChangeTracker.

const changeEDD = `<entity_data_dictionary version="2">
<file_path>chg_edd</file_path>
<entity name="state" number="100" access="rw">
<field name="count" type="integer" subtype="" access="rw" input="" default_value="3" comment="a counter"></field>
<field name="items" type="array" subtype="string" access="rw" input="" default_value="" comment="a list"></field>
<field name="age" type="integer" subtype="" access="rw" input="" default_value="40" comment="asked" collect="true"><question text="Age?" type="number"></question></field>
</entity>
</entity_data_dictionary>`

func changeTable(name, dsl, postfix string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return `<decision_table><table_name>` + name + `</table_name>
<initial_actions><initial_action><action_comment>act</action_comment>
<initial_action_dsl>` + r.Replace(dsl) + `</initial_action_dsl>
<initial_action_postfix>` + r.Replace(postfix) + `</initial_action_postfix>
</initial_action></initial_actions>
<conditions></conditions><actions></actions></decision_table>
`
}

var changeDT = "<decision_tables>\n" +
	changeTable("Same_Count", "set state.count = 3", "3 cvi /state.count xdef") +
	changeTable("Change_Count", "set state.count = 7", "7 cvi /state.count xdef") +
	changeTable("Add_Item", `add "c" to state.items`, `"c" state.items swap addto`) +
	changeTable("Same_List", `set state.items = ["a", "b"]`, `newarray dup "a" addto dup "b" addto /state.items xdef`) +
	changeTable("Read_Age", "set state.age = state.age", "state.age cvi /state.age xdef") +
	"</decision_tables>"

// newChangeState loads the project, pushes a state entity holding
// count=3, items=[a b], and returns the session with its tracker reset.
func newChangeState(t *testing.T) (dtrules.Session, dtrules.ChangeTracker, dtrules.Entity) {
	t.Helper()
	rs := session.NewRuleSet("changes")
	if err := rs.LoadEDD(strings.NewReader(changeEDD)); err != nil {
		t.Fatal(err)
	}
	if err := rs.LoadDecisionTables(strings.NewReader(changeDT)); err != nil {
		t.Fatal(err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ent, err := sess.CreateEntity(dtrules.GetRName("state"))
	if err != nil {
		t.Fatal(err)
	}
	items, _ := dtrules.NewArray(sess, true, false)
	items.Add(dtrules.GetRString("a"))
	items.Add(dtrules.GetRString("b"))
	if err := ent.Put(dtrules.GetRName("items"), items); err != nil {
		t.Fatal(err)
	}
	sess.GetState().EntityPush(ent)
	tracker, ok := sess.GetState().(dtrules.ChangeTracker)
	if !ok {
		t.Fatal("the session state does not implement dtrules.ChangeTracker")
	}
	tracker.ResetChanged()
	return sess, tracker, ent
}

func TestChangeTracker_Execute(t *testing.T) {
	for _, tc := range []struct {
		table   string
		changed bool
	}{
		{"Same_Count", false},
		{"Same_List", false},
		{"Change_Count", true},
		{"Add_Item", true},
	} {
		t.Run(tc.table, func(t *testing.T) {
			sess, tracker, _ := newChangeState(t)
			if err := sess.(*session.RSession).Execute(tc.table); err != nil {
				t.Fatal(err)
			}
			if got := tracker.Changed(); got != tc.changed {
				t.Errorf("Changed() = %v, want %v", got, tc.changed)
			}
		})
	}
}

// The flag is sticky until reset, so a caller measures one execution by
// resetting before it. It records writes, not a diff: 3 -> 7 -> 3 reads
// changed, because each write changed the value.
func TestChangeTracker_StickyUntilReset(t *testing.T) {
	sess, tracker, _ := newChangeState(t)
	rsess := sess.(*session.RSession)
	if err := rsess.Execute("Change_Count"); err != nil {
		t.Fatal(err)
	}
	if err := rsess.Execute("Same_Count"); err != nil { // 7 -> 3
		t.Fatal(err)
	}
	if !tracker.Changed() {
		t.Fatal("two changes must read changed")
	}
	tracker.ResetChanged()
	if err := rsess.Execute("Same_Count"); err != nil { // count is 3 already
		t.Fatal(err)
	}
	if tracker.Changed() {
		t.Error("rewriting the value a field already holds is not a change")
	}
}

// Sorting, shuffling and array building: an in-place operation counts only
// when the elements come out different, and filling an array nothing holds
// yet is not a change at all.
func TestChangeTracker_ArrayOperators(t *testing.T) {
	run := func(t *testing.T, state dtrules.State, op string, args ...dtrules.Object) {
		t.Helper()
		for _, a := range args {
			state.DataPush(a)
		}
		o, ok := operators.GetByString(op)
		if !ok {
			t.Fatalf("%s not registered", op)
		}
		if err := o.Execute(state); err != nil {
			t.Fatal(err)
		}
	}
	items := func(t *testing.T, ent dtrules.Entity) *dtrules.RArray {
		v, _ := ent.Get(dtrules.GetRName("items"))
		a, err := v.RArrayValue()
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	t.Run("sort already sorted", func(t *testing.T) {
		sess, tracker, ent := newChangeState(t)
		run(t, sess.GetState(), "sortarray", items(t, ent), dtrules.GetRBoolean(true))
		if tracker.Changed() {
			t.Error("sorting [a b] ascending changed nothing")
		}
	})
	t.Run("sort reorders", func(t *testing.T) {
		sess, tracker, ent := newChangeState(t)
		run(t, sess.GetState(), "sortarray", items(t, ent), dtrules.GetRBoolean(false))
		if !tracker.Changed() {
			t.Error("sorting [a b] descending reordered it")
		}
	})
	t.Run("fill a new array nothing holds", func(t *testing.T) {
		sess, tracker, _ := newChangeState(t)
		state := sess.GetState()
		run(t, state, "newarray")
		arr, _ := state.DataPop()
		run(t, state, "addto", arr, dtrules.GetRString("x"))
		run(t, state, "cleararray", arr)
		if tracker.Changed() {
			t.Error("a scratch array is not entity data")
		}
	})
	t.Run("remove absent and clear empty", func(t *testing.T) {
		sess, tracker, ent := newChangeState(t)
		state := sess.GetState()
		run(t, state, "remove", items(t, ent), dtrules.GetRString("zz"))
		run(t, state, "removeat", items(t, ent), dtrules.GetRIntegerValue(9))
		if tracker.Changed() {
			t.Error("removing what is not there changed nothing")
		}
		run(t, state, "cleararray", items(t, ent))
		if !tracker.Changed() {
			t.Error("clearing [a b] is a change")
		}
	})
}

// An interactive answer that differs from what the field held is a change;
// accepting the value it already holds is not. Read_Age reads age (the
// collector answers) and writes it back, so the answer is the only thing
// that can change.
func TestChangeTracker_CollectedAnswer(t *testing.T) {
	for _, tc := range []struct {
		answer  int64
		changed bool
	}{{40, false}, {41, true}} {
		sess, tracker, _ := newChangeState(t)
		state := sess.GetState().(*interpreter.DTState)
		state.SetCollector(collect.New(collect.AskerFunc(func(collect.Request) (dtrules.Object, bool, error) {
			return dtrules.GetRIntegerValue(tc.answer), true, nil
		})))
		if err := sess.(*session.RSession).Execute("Read_Age"); err != nil {
			t.Fatal(err)
		}
		v, _ := state.Find(dtrules.GetRName("state.age"))
		if n, _ := v.IntValue(); int64(n) != tc.answer {
			t.Fatalf("age = %v, want the answer %d", v, tc.answer)
		}
		if got := tracker.Changed(); got != tc.changed {
			t.Errorf("answer %d over default 40: Changed() = %v, want %v", tc.answer, got, tc.changed)
		}
	}
}

// The entity stack decides which instance each name resolves to, and so
// what a save writes: leaving it different is a change; a balanced
// push/pop is not.
func TestChangeTracker_EntityStack(t *testing.T) {
	t.Run("push left on the stack", func(t *testing.T) {
		sess, tracker, _ := newChangeState(t)
		fresh, err := sess.CreateEntity(dtrules.GetRName("state"))
		if err != nil {
			t.Fatal(err)
		}
		sess.GetState().EntityPush(fresh)
		if !tracker.Changed() {
			t.Error("a new state instance now shadows the old one")
		}
	})
	t.Run("pop", func(t *testing.T) {
		sess, tracker, _ := newChangeState(t)
		sess.GetState().EntityPop()
		if !tracker.Changed() {
			t.Error("the state entity is no longer on the stack")
		}
	})
	t.Run("balanced", func(t *testing.T) {
		sess, tracker, ent := newChangeState(t)
		sess.GetState().EntityPush(ent)
		sess.GetState().EntityPop()
		if tracker.Changed() {
			t.Error("a balanced push/pop leaves the stack as it was")
		}
	})
}

// SameValue compares what a save would write, not the tolerant Equals:
// doubles 1e-11 apart are Equal but save differently.
func TestSameValue_IsStricterThanEquals(t *testing.T) {
	a, b := dtrules.GetRDoubleValue(3), dtrules.GetRDoubleValue(3.00000000001)
	if eq, _ := a.Equals(b); !eq {
		t.Skip("RDouble.Equals is no longer tolerant; the case below is moot")
	}
	if dtrules.SameValue(a, b) {
		t.Errorf("3 and 3.00000000001 save as %q and %q; SameValue must not call them the same", a.StringValue(), b.StringValue())
	}
	if !dtrules.SameValue(a, dtrules.GetRDoubleValue(3)) {
		t.Error("equal doubles are the same value")
	}
	if dtrules.SameValue(dtrules.GetRIntegerValue(3), a) {
		t.Error("an integer and a double are different types")
	}
}
