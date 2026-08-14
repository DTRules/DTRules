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

// What one small decision table costs to execute (#1025).
//
// The Cribbage sample is the right subject for this question: Score_Hand is
// about as small as a real table gets — five performed tables over entities
// three combinatorial generators materialize — and its Go equivalent is
// twenty lines, so the comparison is honest rather than rigged by an
// expensive reference implementation.
//
// The number matters beyond card scoring. Anything shaped like
// enumerate-and-average — the `expect` primitive #980 named as a later
// candidate, exhaustive expected-value analysis, Monte-Carlo over a rule set
// — pays this cost on every inner call, so it is the ceiling on how much of
// that class of work can move from code into tables. Before #1025 it had only
// ever been measured in a scratch harness that no longer exists, which is the
// reason this file is here: a figure nobody can reproduce is not a figure.
//
// Run:
//
//	go test ./pkg/dtrules/ -run '^$' -bench Cribbage -benchmem
//
// Read ScoreHand as the cost a host pays per scoring, and ScoreHandSetup as
// how much of that is building the entities rather than executing the table —
// the difference is the engine's own share.

// benchCard is a rank/suit pair. Suits are 0-3; rank 1 is the ace.
type benchCard struct{ rank, suit int }

// newCribbageBench loads the Cribbage sample and returns a session ready to
// score. The ruleset load is deliberately outside the measured loop: it
// happens once per process in any real embedding.
func newCribbageBench(b *testing.B) (dtrules.Session, *entity.Factory, dtrules.State) {
	b.Helper()
	rs := session.NewRuleSet("CribbageBench")
	eddFile, err := os.Open("../../sampleprojects/Cribbage/xml/project_edd.xml")
	if err != nil {
		b.Fatalf("open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		b.Fatalf("LoadEDD: %v", err)
	}
	dtFile, err := os.Open("../../sampleprojects/Cribbage/xml/Cribbage_dt.xml")
	if err != nil {
		b.Fatalf("open decision tables: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		b.Fatalf("LoadDecisionTables: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		b.Fatalf("NewSession: %v", err)
	}
	return sess, sess.GetEntityFactory().(*entity.Factory), sess.GetState()
}

// makeBenchCard materializes one card entity.
func makeBenchCard(b *testing.B, sess dtrules.Session, ef *entity.Factory, c benchCard) dtrules.Entity {
	e, err := ef.CreateEntity(sess, dtrules.GetRName("card"))
	if err != nil {
		b.Fatalf("CreateEntity(card): %v", err)
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

// buildBenchHand creates the hand entity and its work arrays for one scoring.
//
// A fresh hand every iteration is not an artefact of benchmarking — the
// generators append to the work arrays, so scoring a second hand into the
// same entity would accumulate the first hand's combos. Every real caller
// pays this too.
func buildBenchHand(b *testing.B, sess dtrules.Session, ef *entity.Factory,
	kept [4]benchCard, starter benchCard, isCrib bool) dtrules.Entity {

	hand, err := ef.CreateEntity(sess, dtrules.GetRName("hand"))
	if err != nil {
		b.Fatalf("CreateEntity(hand): %v", err)
	}
	keptObjs := make([]dtrules.Object, 0, 4)
	allObjs := make([]dtrules.Object, 0, 5)
	for _, c := range kept {
		e := makeBenchCard(b, sess, ef, c)
		keptObjs = append(keptObjs, e)
		allObjs = append(allObjs, e)
	}
	allObjs = append(allObjs, makeBenchCard(b, sess, ef, starter))

	keptArr, err := dtrules.NewArrayWithElements(sess, true, keptObjs, false)
	if err != nil {
		b.Fatal(err)
	}
	allArr, err := dtrules.NewArrayWithElements(sess, true, allObjs, false)
	if err != nil {
		b.Fatal(err)
	}
	hand.Put(dtrules.GetRName("kept"), keptArr)
	hand.Put(dtrules.GetRName("cards"), allArr)
	hand.Put(dtrules.GetRName("starter_suit"), dtrules.GetRIntegerValue(int64(starter.suit)))
	hand.Put(dtrules.GetRName("is_crib"), dtrules.GetRBoolean(isCrib))
	for _, work := range []string{"combos", "rank_groups", "suit_groups", "run_list"} {
		arr, err := dtrules.NewArray(sess, true, false)
		if err != nil {
			b.Fatal(err)
		}
		hand.Put(dtrules.GetRName(work), arr)
	}
	return hand
}

// A middling hand: two fifteens, a pair and a run, so no scoring path is
// skipped and none of them blows up. A zero hand would flatter the table by
// firing almost nothing.
var (
	benchKept    = [4]benchCard{{5, 0}, {6, 1}, {7, 2}, {8, 3}}
	benchStarter = benchCard{5, 1}
)

// BenchmarkCribbageScoreHand is the whole cost of one scoring as a host pays
// it: build the five card entities and the hand, then execute Score_Hand.
func BenchmarkCribbageScoreHand(b *testing.B) {
	sess, ef, state := newCribbageBench(b)
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Score_Hand"))
	if err != nil {
		b.Fatalf("GetDecisionTable(Score_Hand): %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hand := buildBenchHand(b, sess, ef, benchKept, benchStarter, false)
		state.EntityPush(hand)
		err := dt.Execute(state)
		state.EntityPop()
		if err != nil {
			b.Fatalf("Score_Hand: %v", err)
		}
	}
}

// BenchmarkCribbageScoreHandSetup measures only the entity and array
// construction, so the engine's own share is the difference between this and
// BenchmarkCribbageScoreHand. Keeping them separate is what makes the split
// reproducible; a single number cannot say whether the cost is the embedding
// boundary or the table.
func BenchmarkCribbageScoreHandSetup(b *testing.B) {
	sess, ef, _ := newCribbageBench(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildBenchHand(b, sess, ef, benchKept, benchStarter, false)
	}
}

// BenchmarkCribbageScorePlay measures the pegging table (#1023), which reaches
// order-dependent structure through `suffixes` rather than the subset
// enumeration Score_Hand uses. The two together say whether the cost tracks
// the generator or the table machinery around it.
func BenchmarkCribbageScorePlay(b *testing.B) {
	sess, ef, state := newCribbageBench(b)
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Score_Play"))
	if err != nil {
		b.Fatalf("GetDecisionTable(Score_Play): %v", err)
	}
	stack := []int{4, 6, 5} // a run of three, laid out of order, making fifteen

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		play, err := ef.CreateEntity(sess, dtrules.GetRName("play"))
		if err != nil {
			b.Fatalf("CreateEntity(play): %v", err)
		}
		cards := make([]dtrules.Object, 0, len(stack))
		for _, r := range stack {
			cards = append(cards, makeBenchCard(b, sess, ef, benchCard{r, 0}))
		}
		stackArr, err := dtrules.NewArrayWithElements(sess, true, cards, false)
		if err != nil {
			b.Fatal(err)
		}
		windows, err := dtrules.NewArray(sess, true, false)
		if err != nil {
			b.Fatal(err)
		}
		play.Put(dtrules.GetRName("cards"), stackArr)
		play.Put(dtrules.GetRName("windows"), windows)
		play.Put(dtrules.GetRName("count"), dtrules.GetRIntegerValue(15))
		play.Put(dtrules.GetRName("is_go"), dtrules.GetRBoolean(false))
		play.Put(dtrules.GetRName("is_last"), dtrules.GetRBoolean(false))

		state.EntityPush(play)
		err = dt.Execute(state)
		state.EntityPop()
		if err != nil {
			b.Fatalf("Score_Play: %v", err)
		}
	}
}
