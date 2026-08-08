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

// TestCombinatoricsFromEL compiles the combinatorial generators (#980) from
// authored EL — the exact statements the Cribbage sample's tables use — and
// scores real hands end to end. The expected numbers are the cribbage
// fifteens/pairs/runs values (no flush or nobs here; those are ordinary
// conditions, not combinatorics).
func TestCombinatoricsFromEL(t *testing.T) {
	rs := session.NewRuleSet("comb")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	intAttr := func(ref *entity.REntity, name string) {
		ref.AddAttribute(dtrules.GetRName(name), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")
	}
	arrAttr := func(ref *entity.REntity, name, subtype string) {
		ref.AddAttribute(dtrules.GetRName(name), "", nil, true, true, dtrules.TypeArray, subtype, "", "", "")
	}

	cardRef, err := ef.FindCreateRefEntity(true, dtrules.GetRName("card"))
	if err != nil {
		t.Fatal(err)
	}
	intAttr(cardRef, "rank")
	intAttr(cardRef, "value")

	comboRef, err := ef.FindCreateRefEntity(true, dtrules.GetRName("combo"))
	if err != nil {
		t.Fatal(err)
	}
	intAttr(comboRef, "count")
	intAttr(comboRef, "sum")
	arrAttr(comboRef, "members", "card")

	groupRef, err := ef.FindCreateRefEntity(true, dtrules.GetRName("grp"))
	if err != nil {
		t.Fatal(err)
	}
	intAttr(groupRef, "key")
	intAttr(groupRef, "count")
	arrAttr(groupRef, "members", "card")

	runRef, err := ef.FindCreateRefEntity(true, dtrules.GetRName("seq"))
	if err != nil {
		t.Fatal(err)
	}
	intAttr(runRef, "start")
	intAttr(runRef, "span")
	intAttr(runRef, "multiplicity")

	handRef, err := ef.FindCreateRefEntity(true, dtrules.GetRName("hand"))
	if err != nil {
		t.Fatal(err)
	}
	arrAttr(handRef, "cards", "card")
	arrAttr(handRef, "combos", "combo")
	arrAttr(handRef, "rank_groups", "grp")
	arrAttr(handRef, "run_list", "seq")
	intAttr(handRef, "fifteen_points")
	intAttr(handRef, "pair_points")
	intAttr(handRef, "run_points")

	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{
		"cards": "array", "combos": "array", "rank_groups": "array", "run_list": "array",
		"rank": "integer", "value": "integer",
		"count": "integer", "sum": "integer", "key": "integer",
		"start": "integer", "span": "integer", "multiplicity": "integer",
		"fifteen_points": "integer", "pair_points": "integer", "run_points": "integer",
	})

	state := sess.GetState().(*interpreter.DTState)

	// The action script a table's initial actions would carry. Pair points
	// fall out of group counts as count*(count-1): 2, 6, 12 for 2, 3, 4.
	actions := []string{
		`subsets(cards, "combo", "value", combos)`,
		`groupby(cards, "rank", "grp", rank_groups)`,
		`maximalruns(cards, "rank", 3, "seq", run_list)`,
		`set fifteen_points = 2 * number of combos where sum == 15`,
		`set pair_points = sum of count * (count - 1) in rank_groups`,
		`set run_points = sum of span * multiplicity in run_list`,
	}

	runHand := func(t *testing.T, ranks []int) (fifteens, pairs, runs int) {
		t.Helper()
		hand, err := ef.CreateEntity(sess, dtrules.GetRName("hand"))
		if err != nil {
			t.Fatal(err)
		}
		elems := make([]dtrules.Object, 0, len(ranks))
		for _, r := range ranks {
			c, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
			if err != nil {
				t.Fatal(err)
			}
			v := r
			if v > 10 {
				v = 10
			}
			c.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(r)))
			c.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(int64(v)))
			elems = append(elems, c)
		}
		cardsArr, err := dtrules.NewArrayWithElements(sess, true, elems, false)
		if err != nil {
			t.Fatal(err)
		}
		hand.Put(dtrules.GetRName("cards"), cardsArr)
		for _, work := range []string{"combos", "rank_groups", "run_list"} {
			arr, err := dtrules.NewArray(sess, true, false)
			if err != nil {
				t.Fatal(err)
			}
			hand.Put(dtrules.GetRName(work), arr)
		}

		state.EntityPush(hand)
		defer state.EntityPop()
		for _, dsl := range actions {
			postfix, err := elc.CompileAction(dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", dsl, err)
			}
			obj, err := sess.Compile(postfix)
			if err != nil {
				t.Fatalf("assemble %q (postfix %q): %v", dsl, postfix, err)
			}
			if err := obj.Execute(state); err != nil {
				t.Fatalf("execute %q (postfix %q): %v", dsl, postfix, err)
			}
		}
		get := func(name string) int {
			v, err := hand.Get(dtrules.GetRName(name))
			if err != nil || v == nil {
				t.Fatalf("get %s: %v", name, err)
			}
			n, _ := v.IntValue()
			return n
		}
		return get("fifteen_points"), get("pair_points"), get("run_points")
	}

	cases := []struct {
		name                  string
		ranks                 []int
		fifteens, pairs, runs int
	}{
		// The perfect 29 minus nobs: 8 fifteens, 6 pairs, no run.
		{"four fives and a jack", []int{5, 5, 5, 11, 5}, 16, 12, 0},
		// The double run: 7+8 twice, one pair, run of 4 doubled.
		{"double run", []int{7, 8, 8, 9, 10}, 4, 2, 8},
		// A-2-3-4-5: one fifteen (the whole hand), run of five.
		{"run of five", []int{1, 2, 3, 4, 5}, 2, 0, 5},
		// A zero hand stays zero.
		{"zero hand", []int{1, 2, 7, 9, 13}, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fifteens, pairs, runs := runHand(t, tc.ranks)
			if fifteens != tc.fifteens || pairs != tc.pairs || runs != tc.runs {
				t.Errorf("got fifteens %d, pairs %d, runs %d — want %d, %d, %d",
					fifteens, pairs, runs, tc.fifteens, tc.pairs, tc.runs)
			}
		})
	}
}
