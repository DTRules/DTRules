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
	"os"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestCribbageScenarios runs the Cribbage sample project (#984): every point
// comes from a decision table reading the entities the combinatorial
// primitives (#980) materialized — no Go scoring code anywhere. Expected
// totals are the classic values from the reference implementation
// (github.com/paulsnow/cribbage score_test.go).
//
// Suits: 0=spades, 1=hearts, 2=diamonds, 3=clubs.
func TestCribbageScenarios(t *testing.T) {
	type card struct{ rank, suit int }
	cases := []struct {
		name    string
		kept    [4]card
		starter card
		isCrib  bool
		total   int
	}{
		{
			name:    "perfect 29",
			kept:    [4]card{{5, 1}, {5, 2}, {5, 3}, {11, 0}},
			starter: card{5, 0},
			total:   29,
		},
		{
			name:    "run of five with one fifteen",
			kept:    [4]card{{7, 3}, {8, 2}, {9, 1}, {10, 0}},
			starter: card{11, 2},
			total:   7,
		},
		{
			name:    "double run with pair and fifteens",
			kept:    [4]card{{8, 0}, {8, 1}, {9, 2}, {10, 3}},
			starter: card{7, 0},
			total:   14,
		},
		{
			name:    "five-card flush",
			kept:    [4]card{{2, 1}, {4, 1}, {6, 1}, {8, 1}},
			starter: card{13, 1},
			total:   5,
		},
		{
			name:    "four-card flush, off-suit starter",
			kept:    [4]card{{2, 1}, {4, 1}, {6, 1}, {8, 1}},
			starter: card{13, 0},
			total:   4,
		},
		{
			name:    "crib flush requires all five",
			kept:    [4]card{{2, 1}, {4, 1}, {6, 1}, {8, 1}},
			starter: card{13, 0},
			isCrib:  true,
			total:   0,
		},
		{
			name:    "nobs plus fifteens",
			kept:    [4]card{{11, 1}, {2, 3}, {3, 2}, {13, 0}},
			starter: card{9, 1},
			total:   5,
		},
		{
			name:    "zero hand",
			kept:    [4]card{{1, 0}, {2, 1}, {7, 2}, {9, 3}},
			starter: card{13, 0},
			total:   0,
		},
	}

	rs := session.NewRuleSet("Cribbage")
	eddFile, err := os.Open("../../sampleprojects/Cribbage/xml/project_edd.xml")
	if err != nil {
		t.Fatalf("open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	dtFile, err := os.Open("../../sampleprojects/Cribbage/xml/Cribbage_dt.xml")
	if err != nil {
		t.Fatalf("open decision tables: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Fatalf("LoadDecisionTables: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	state := sess.GetState()

	makeCard := func(t *testing.T, c card) dtrules.Entity {
		t.Helper()
		e, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
		if err != nil {
			t.Fatalf("CreateEntity(card): %v", err)
		}
		v := c.rank
		if v > 10 {
			v = 10
		}
		e.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(c.rank)))
		e.Put(dtrules.GetRName("suit"), dtrules.GetRIntegerValue(int64(c.suit)))
		e.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(int64(v)))
		return e
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hand, err := ef.CreateEntity(sess, dtrules.GetRName("hand"))
			if err != nil {
				t.Fatalf("CreateEntity(hand): %v", err)
			}
			keptObjs := make([]dtrules.Object, 0, 4)
			allObjs := make([]dtrules.Object, 0, 5)
			for _, c := range tc.kept {
				e := makeCard(t, c)
				keptObjs = append(keptObjs, e)
				allObjs = append(allObjs, e)
			}
			allObjs = append(allObjs, makeCard(t, tc.starter))

			keptArr, err := dtrules.NewArrayWithElements(sess, true, keptObjs, false)
			if err != nil {
				t.Fatal(err)
			}
			allArr, err := dtrules.NewArrayWithElements(sess, true, allObjs, false)
			if err != nil {
				t.Fatal(err)
			}
			hand.Put(dtrules.GetRName("kept"), keptArr)
			hand.Put(dtrules.GetRName("cards"), allArr)
			hand.Put(dtrules.GetRName("starter_suit"), dtrules.GetRIntegerValue(int64(tc.starter.suit)))
			hand.Put(dtrules.GetRName("is_crib"), dtrules.GetRBoolean(tc.isCrib))
			for _, work := range []string{"combos", "rank_groups", "suit_groups", "run_list"} {
				arr, err := dtrules.NewArray(sess, true, false)
				if err != nil {
					t.Fatal(err)
				}
				hand.Put(dtrules.GetRName(work), arr)
			}

			dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Score_Hand"))
			if err != nil {
				t.Fatalf("GetDecisionTable(Score_Hand): %v", err)
			}
			state.EntityPush(hand)
			err = dt.Execute(state)
			state.EntityPop()
			if err != nil {
				t.Fatalf("Score_Hand: %v", err)
			}

			get := func(name string) int {
				v, err := hand.Get(dtrules.GetRName(name))
				if err != nil || v == nil {
					t.Fatalf("get %s: %v", name, err)
				}
				n, _ := v.IntValue()
				return n
			}
			if got := get("total"); got != tc.total {
				t.Errorf("total = %d, want %d (fifteens %d, pairs %d, runs %d, flush %d, nobs %d)",
					got, tc.total, get("fifteen_points"), get("pair_points"),
					get("run_points"), get("flush_points"), get("nobs_points"))
			}
		})
	}
}
