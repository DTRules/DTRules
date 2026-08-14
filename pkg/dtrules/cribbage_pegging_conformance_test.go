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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// referencePegScore is an independent oracle for pegging points on a legal
// stack — a direct transcription of the reference implementation's rules
// (github.com/paulsnow/cribbage peg.go), NOT of the decision tables:
// fifteen/thirty-one on the running count, the trailing equal-rank block,
// and the longest trailing run of distinct consecutive ranks in any order.
func referencePegScore(ranks []int) int {
	count, pts := 0, 0
	for _, r := range ranks {
		v := r
		if v > 10 {
			v = 10
		}
		count += v
	}
	if count == 15 || count == 31 {
		pts += 2
	}

	k := 1
	for i := len(ranks) - 2; i >= 0 && ranks[i] == ranks[len(ranks)-1]; i-- {
		k++
	}
	switch k {
	case 2:
		pts += 2
	case 3:
		pts += 6
	case 4:
		pts += 12
	}

	for l := len(ranks); l >= 3; l-- {
		window := ranks[len(ranks)-l:]
		seen := map[int]bool{}
		mn, mx, ok := 99, 0, true
		for _, r := range window {
			if seen[r] {
				ok = false
				break
			}
			seen[r] = true
			if r < mn {
				mn = r
			}
			if r > mx {
				mx = r
			}
		}
		if ok && mx-mn == l-1 {
			pts += l
			break
		}
	}
	return pts
}

// TestCribbagePeggingConformance throws randomized legal stacks at the
// Cribbage sample's Score_Play tables and requires exact agreement with the
// independent oracle above. Deck-legal draws (at most four of a rank),
// count capped at 31, fixed seed for reproducibility.
func TestCribbagePeggingConformance(t *testing.T) {
	rs := session.NewRuleSet("CribbagePeggingConformance")
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
	dt, err := ef.GetDecisionTable(dtrules.GetRName("Score_Play"))
	if err != nil {
		t.Fatalf("GetDecisionTable(Score_Play): %v", err)
	}

	scoreViaTables := func(ranks []int) (int, error) {
		play, err := ef.CreateEntity(sess, dtrules.GetRName("play"))
		if err != nil {
			return 0, err
		}
		count := 0
		cardObjs := make([]dtrules.Object, 0, len(ranks))
		for _, r := range ranks {
			c, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
			if err != nil {
				return 0, err
			}
			v := r
			if v > 10 {
				v = 10
			}
			count += v
			c.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(r)))
			c.Put(dtrules.GetRName("suit"), dtrules.GetRIntegerValue(0))
			c.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(int64(v)))
			cardObjs = append(cardObjs, c)
		}
		stackArr, err := dtrules.NewArrayWithElements(sess, true, cardObjs, false)
		if err != nil {
			return 0, err
		}
		windows, err := dtrules.NewArray(sess, true, false)
		if err != nil {
			return 0, err
		}
		play.Put(dtrules.GetRName("cards"), stackArr)
		play.Put(dtrules.GetRName("windows"), windows)
		play.Put(dtrules.GetRName("count"), dtrules.GetRIntegerValue(int64(count)))
		play.Put(dtrules.GetRName("is_go"), dtrules.GetRBoolean(false))
		play.Put(dtrules.GetRName("is_last"), dtrules.GetRBoolean(false))

		state.EntityPush(play)
		err = dt.Execute(state)
		state.EntityPop()
		if err != nil {
			return 0, err
		}
		v, err := play.Get(dtrules.GetRName("total"))
		if err != nil || v == nil {
			return 0, fmt.Errorf("get total: %v", err)
		}
		return v.IntValue()
	}

	rng := rand.New(rand.NewSource(12345))
	deck := make([]int, 0, 52)
	for r := 1; r <= 13; r++ {
		for s := 0; s < 4; s++ {
			deck = append(deck, r)
		}
	}

	const iterations = 1500
	mismatches := 0
	for i := 0; i < iterations; i++ {
		rng.Shuffle(len(deck), func(a, b int) { deck[a], deck[b] = deck[b], deck[a] })
		stack := make([]int, 0, 13)
		count := 0
		for _, r := range deck {
			v := r
			if v > 10 {
				v = 10
			}
			if count+v > 31 {
				break
			}
			stack = append(stack, r)
			count += v
		}
		if len(stack) == 0 {
			continue
		}

		want := referencePegScore(stack)
		got, err := scoreViaTables(stack)
		if err != nil {
			t.Fatalf("stack %v: %v", stack, err)
		}
		if got != want {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("stack %v (count %d): tables scored %d, oracle says %d", stack, count, got, want)
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d of %d random stacks disagree with the oracle", mismatches, iterations)
	}
}
