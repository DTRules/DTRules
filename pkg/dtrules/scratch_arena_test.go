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

// TestScratchArenaScoresRepeatedly is the arena's (#1025) integration gate:
// the Cribbage sample's Score_Hand, executed repeatedly on one session with
// ResetScratch between executions, must produce the reference totals every
// time — recycled combo/group/run entities carrying a previous hand's
// values into the next would show up here immediately.
func TestScratchArenaScoresRepeatedly(t *testing.T) {
	rs := session.NewRuleSet("CribbageArena")
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

	sa, ok := sess.(dtrules.ScratchAllocator)
	if !ok {
		t.Fatal("session must implement dtrules.ScratchAllocator")
	}
	sa.EnableScratch()

	type card struct{ rank, suit int }
	makeCard := func(c card) dtrules.Entity {
		e, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
		if err != nil {
			t.Fatal(err)
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

	// Two hands with known totals, alternated so a leak from either would
	// corrupt the other: the perfect 29 and the double run worth 14.
	hands := []struct {
		kept    [4]card
		starter card
		total   int
	}{
		{[4]card{{5, 1}, {5, 2}, {5, 3}, {11, 0}}, card{5, 0}, 29},
		{[4]card{{8, 0}, {8, 1}, {9, 2}, {10, 3}}, card{7, 0}, 14},
	}

	// Durable state built once: the hand entity, its cards, and the work
	// arrays. Each iteration clears the work arrays (they hold last round's
	// scratch entities) and resets the arena — the steady-state loop shape.
	dt, err := ef.GetDecisionTable(dtrules.GetRName("Score_Hand"))
	if err != nil {
		t.Fatalf("GetDecisionTable: %v", err)
	}

	for round := 0; round < 6; round++ {
		h := hands[round%len(hands)]
		hand, err := ef.CreateEntity(sess, dtrules.GetRName("hand"))
		if err != nil {
			t.Fatal(err)
		}
		keptObjs := make([]dtrules.Object, 0, 4)
		allObjs := make([]dtrules.Object, 0, 5)
		for _, c := range h.kept {
			e := makeCard(c)
			keptObjs = append(keptObjs, e)
			allObjs = append(allObjs, e)
		}
		allObjs = append(allObjs, makeCard(h.starter))
		keptArr, _ := dtrules.NewArrayWithElements(sess, true, keptObjs, false)
		allArr, _ := dtrules.NewArrayWithElements(sess, true, allObjs, false)
		hand.Put(dtrules.GetRName("kept"), keptArr)
		hand.Put(dtrules.GetRName("cards"), allArr)
		hand.Put(dtrules.GetRName("starter_suit"), dtrules.GetRIntegerValue(int64(h.starter.suit)))
		hand.Put(dtrules.GetRName("is_crib"), dtrules.GetRBoolean(false))
		for _, work := range []string{"combos", "rank_groups", "suit_groups", "run_list"} {
			arr, _ := dtrules.NewArray(sess, true, false)
			hand.Put(dtrules.GetRName(work), arr)
		}

		state.EntityPush(hand)
		err = dt.Execute(state)
		state.EntityPop()
		if err != nil {
			t.Fatalf("round %d: Score_Hand: %v", round, err)
		}

		v, err := hand.Get(dtrules.GetRName("total"))
		if err != nil || v == nil {
			t.Fatalf("round %d: get total: %v", round, err)
		}
		if got, _ := v.IntValue(); got != h.total {
			t.Errorf("round %d: total = %d, want %d — a recycled scratch entity leaked state", round, got, h.total)
		}

		sa.ResetScratch()
	}
}
