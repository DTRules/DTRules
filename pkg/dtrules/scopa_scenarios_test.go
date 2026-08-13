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

// The Scopa sample (#1118) is the second-game validation of the
// combinatorial primitives (#980): capture resolution (subset-sum under the
// mandatory-single precedence rule), the scopa sweep, and primiera scoring
// (groupby + a mapping table + the max-fold idiom) — all from decision
// tables, with the operators exactly as merged for cribbage.

func loadScopa(t *testing.T) (dtrules.Session, *entity.Factory, dtrules.State) {
	t.Helper()
	rs := session.NewRuleSet("Scopa")
	eddFile, err := os.Open("../../sampleprojects/Scopa/xml/project_edd.xml")
	if err != nil {
		t.Fatalf("open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	dtFile, err := os.Open("../../sampleprojects/Scopa/xml/Scopa_dt.xml")
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
	return sess, sess.GetEntityFactory().(*entity.Factory), sess.GetState()
}

type scopaCard struct{ rank, suit int }

func makeScopaCards(t *testing.T, sess dtrules.Session, ef *entity.Factory, cards []scopaCard) []dtrules.Object {
	t.Helper()
	objs := make([]dtrules.Object, 0, len(cards))
	for _, sc := range cards {
		c, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
		if err != nil {
			t.Fatalf("CreateEntity(card): %v", err)
		}
		c.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(sc.rank)))
		c.Put(dtrules.GetRName("suit"), dtrules.GetRIntegerValue(int64(sc.suit)))
		c.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(int64(sc.rank)))
		objs = append(objs, c)
	}
	return objs
}

func TestScopaCaptureResolution(t *testing.T) {
	cases := []struct {
		name        string
		table       []int // ranks on the table (suits immaterial to capture)
		played      int
		wantFound   int
		wantSize    int
		wantIsScopa bool
	}{
		{
			// The precedence rule: with a 7 on the table, 3+4 is barred.
			name: "single capture is mandatory", table: []int{3, 4, 7}, played: 7,
			wantFound: 1, wantSize: 1, wantIsScopa: false,
		},
		{
			name: "sum capture sweeps the table", table: []int{3, 4}, played: 7,
			wantFound: 1, wantSize: 2, wantIsScopa: true,
		},
		{
			name: "three-card sum", table: []int{1, 2, 3}, played: 6,
			wantFound: 1, wantSize: 3, wantIsScopa: true,
		},
		{
			// A matching single among sum candidates: the single still wins,
			// and taking one card of four is no sweep.
			name: "single beats the sums without sweeping", table: []int{1, 2, 3, 6}, played: 6,
			wantFound: 1, wantSize: 1, wantIsScopa: false,
		},
		{
			name: "two competing sums resolve to one capture", table: []int{2, 5, 3, 4}, played: 7,
			wantFound: 1, wantSize: 2, wantIsScopa: false,
		},
		{
			name: "no capture", table: []int{8, 9}, played: 3,
			wantFound: 0, wantSize: 0, wantIsScopa: false,
		},
		{
			name: "empty table captures nothing", table: nil, played: 5,
			wantFound: 0, wantSize: 0, wantIsScopa: false,
		},
	}

	sess, ef, state := loadScopa(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			move, err := ef.CreateEntity(sess, dtrules.GetRName("move"))
			if err != nil {
				t.Fatalf("CreateEntity(move): %v", err)
			}
			cards := make([]scopaCard, 0, len(tc.table))
			for _, r := range tc.table {
				cards = append(cards, scopaCard{rank: r, suit: 1})
			}
			tableArr, err := dtrules.NewArrayWithElements(sess, true, makeScopaCards(t, sess, ef, cards), false)
			if err != nil {
				t.Fatal(err)
			}
			options, err := dtrules.NewArray(sess, true, false)
			if err != nil {
				t.Fatal(err)
			}
			move.Put(dtrules.GetRName("table_cards"), tableArr)
			move.Put(dtrules.GetRName("options"), options)
			move.Put(dtrules.GetRName("played_rank"), dtrules.GetRIntegerValue(int64(tc.played)))

			dt, err := ef.GetDecisionTable(dtrules.GetRName("Resolve_Capture"))
			if err != nil {
				t.Fatalf("GetDecisionTable: %v", err)
			}
			state.EntityPush(move)
			err = dt.Execute(state)
			state.EntityPop()
			if err != nil {
				t.Fatalf("Resolve_Capture: %v", err)
			}

			geti := func(name string) int {
				v, err := move.Get(dtrules.GetRName(name))
				if err != nil || v == nil {
					t.Fatalf("get %s: %v", name, err)
				}
				n, _ := v.IntValue()
				return n
			}
			scopaV, _ := move.Get(dtrules.GetRName("is_scopa"))
			gotScopa, _ := scopaV.BooleanValue()
			if geti("capture_found") != tc.wantFound || geti("capture_size") != tc.wantSize || gotScopa != tc.wantIsScopa {
				t.Errorf("found %d size %d scopa %v — want %d, %d, %v",
					geti("capture_found"), geti("capture_size"), gotScopa,
					tc.wantFound, tc.wantSize, tc.wantIsScopa)
			}
		})
	}
}

func TestScopaPrimiera(t *testing.T) {
	cases := []struct {
		name       string
		pile       []scopaCard
		wantTotal  int
		settebello bool
	}{
		{
			// Bests per suit: denari 7 (21), suit1 7 (21), suit2 A (16),
			// suit3 10 (10). The 6 of denari loses to its 7.
			name: "four suits with the settebello",
			pile: []scopaCard{{7, 0}, {6, 0}, {7, 1}, {1, 2}, {10, 3}},
			wantTotal: 68, settebello: true,
		},
		{
			name: "two suits, no settebello",
			pile: []scopaCard{{6, 0}, {2, 1}},
			wantTotal: 30, settebello: false,
		},
		{
			// A 7 that is not denari is worth 21 but is not the settebello.
			name: "seven of cups is no settebello",
			pile: []scopaCard{{7, 2}},
			wantTotal: 21, settebello: false,
		},
		{
			name: "empty pile", pile: nil, wantTotal: 0, settebello: false,
		},
	}

	sess, ef, state := loadScopa(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			player, err := ef.CreateEntity(sess, dtrules.GetRName("player"))
			if err != nil {
				t.Fatalf("CreateEntity(player): %v", err)
			}
			pileArr, err := dtrules.NewArrayWithElements(sess, true, makeScopaCards(t, sess, ef, tc.pile), false)
			if err != nil {
				t.Fatal(err)
			}
			groups, err := dtrules.NewArray(sess, true, false)
			if err != nil {
				t.Fatal(err)
			}
			player.Put(dtrules.GetRName("pile"), pileArr)
			player.Put(dtrules.GetRName("suit_groups"), groups)

			dt, err := ef.GetDecisionTable(dtrules.GetRName("Score_Primiera"))
			if err != nil {
				t.Fatalf("GetDecisionTable: %v", err)
			}
			state.EntityPush(player)
			err = dt.Execute(state)
			state.EntityPop()
			if err != nil {
				t.Fatalf("Score_Primiera: %v", err)
			}

			totalV, err := player.Get(dtrules.GetRName("primiera_total"))
			if err != nil || totalV == nil {
				t.Fatalf("get primiera_total: %v", err)
			}
			total, _ := totalV.IntValue()
			sbV, _ := player.Get(dtrules.GetRName("has_settebello"))
			sb, _ := sbV.BooleanValue()
			if total != tc.wantTotal || sb != tc.settebello {
				t.Errorf("primiera %d settebello %v — want %d, %v", total, sb, tc.wantTotal, tc.settebello)
			}
		})
	}
}
