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

// TestCribbagePegging scores play (pegging) events entirely from the
// Cribbage sample's decision tables over the suffixes primitive (#1023):
// fifteen, thirty-one, trailing pairs, longest-run-only runs, go, and last
// card. Each case is the state after the newest card is laid (or a bare
// go/last award with an empty stack — the stack's own points were scored
// when its cards were laid). Expected values mirror the reference
// implementation's peg tests (github.com/paulsnow/cribbage peg_test.go).
func TestCribbagePegging(t *testing.T) {
	cases := []struct {
		name   string
		stack  []int // ranks in lay order, newest last
		count  int
		isGo   bool
		isLast bool
		total  int
	}{
		{name: "fifteen", stack: []int{5, 10}, count: 15, total: 2},
		{name: "pair", stack: []int{8, 8}, count: 16, total: 2},
		{name: "trips", stack: []int{8, 8, 8}, count: 24, total: 6},
		{name: "quads", stack: []int{7, 7, 7, 7}, count: 28, total: 12},
		{name: "run out of order with fifteen", stack: []int{4, 6, 5}, count: 15, total: 5},
		{name: "thirty-one", stack: []int{10, 10, 6, 5}, count: 31, total: 2},
		{name: "run of four scores only once", stack: []int{2, 3, 4, 5}, count: 14, total: 4},
		{name: "seven-card run", stack: []int{1, 2, 3, 4, 5, 6, 7}, count: 28, total: 7},
		{name: "no score", stack: []int{9, 10}, count: 19, total: 0},
		{name: "pair broken by an intervening card", stack: []int{7, 2, 7}, count: 16, total: 0},
		{name: "go point", stack: nil, count: 27, isGo: true, total: 1},
		{name: "last card", stack: []int{9, 13}, count: 19, isLast: true, total: 1},
		{name: "last card making thirty-one stays 2", stack: []int{10, 10, 6, 5}, count: 31, isLast: true, total: 2},
		{name: "go never granted on thirty-one", stack: nil, count: 31, isGo: true, total: 0},
	}

	rs := session.NewRuleSet("CribbagePegging")
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			play, err := ef.CreateEntity(sess, dtrules.GetRName("play"))
			if err != nil {
				t.Fatalf("CreateEntity(play): %v", err)
			}
			cardObjs := make([]dtrules.Object, 0, len(tc.stack))
			for _, r := range tc.stack {
				c, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
				if err != nil {
					t.Fatalf("CreateEntity(card): %v", err)
				}
				v := r
				if v > 10 {
					v = 10
				}
				c.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(r)))
				c.Put(dtrules.GetRName("suit"), dtrules.GetRIntegerValue(0))
				c.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(int64(v)))
				cardObjs = append(cardObjs, c)
			}
			stackArr, err := dtrules.NewArrayWithElements(sess, true, cardObjs, false)
			if err != nil {
				t.Fatal(err)
			}
			windows, err := dtrules.NewArray(sess, true, false)
			if err != nil {
				t.Fatal(err)
			}
			play.Put(dtrules.GetRName("cards"), stackArr)
			play.Put(dtrules.GetRName("windows"), windows)
			play.Put(dtrules.GetRName("count"), dtrules.GetRIntegerValue(int64(tc.count)))
			play.Put(dtrules.GetRName("is_go"), dtrules.GetRBoolean(tc.isGo))
			play.Put(dtrules.GetRName("is_last"), dtrules.GetRBoolean(tc.isLast))

			dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Score_Play"))
			if err != nil {
				t.Fatalf("GetDecisionTable(Score_Play): %v", err)
			}
			state.EntityPush(play)
			err = dt.Execute(state)
			state.EntityPop()
			if err != nil {
				t.Fatalf("Score_Play: %v", err)
			}

			get := func(name string) int {
				v, err := play.Get(dtrules.GetRName(name))
				if err != nil || v == nil {
					t.Fatalf("get %s: %v", name, err)
				}
				n, _ := v.IntValue()
				return n
			}
			if got := get("total"); got != tc.total {
				t.Errorf("total = %d, want %d (count %d, pairs %d, runs %d, go %d)",
					got, tc.total, get("count_points"), get("pair_points"),
					get("run_points"), get("go_points"))
			}
		})
	}
}
