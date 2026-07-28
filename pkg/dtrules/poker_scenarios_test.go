// Copyright 2024 Paul Snow
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
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// pokerExpectation pairs a test-case comment with the player that follows it:
//
//	<!-- Test Case 1: TAG with strong hand (should RAISE) -->
//	<player>
//	  <id>tag_strong</id>
//
// The scenario file has documented what each archetype ought to do since it
// was written; this is what makes those comments load-bearing.
var pokerExpectation = regexp.MustCompile(`\(should ([A-Z]+)[^)]*\)[^<]*-->\s*<player>\s*<id>([^<]+)</id>`)

// TestPokerScenarios runs every player in the scenario file through
// Poker_Decision and checks the action against the expectation beside it.
func TestPokerScenarios(t *testing.T) {
	dir := findPokerDir(t)
	if dir == "" {
		t.Skip("Poker sample project not found")
	}

	scenario := filepath.Join(dir, "testfiles/TestScenarios/basic_scenarios.xml")
	raw, err := os.ReadFile(scenario)
	if err != nil {
		t.Fatalf("read scenario: %v", err)
	}
	matches := pokerExpectation.FindAllStringSubmatch(string(raw), -1)
	if len(matches) == 0 {
		t.Fatal("scenario file documents no `(should ACTION)` expectations")
	}
	want := make(map[string]string, len(matches))
	for _, m := range matches {
		want[m[2]] = m[1]
	}

	t.Logf("checking %d documented expectations", len(want))
	got := runPokerScenario(t, dir, scenario)

	for id, action := range want {
		switch decided, ok := got[id]; {
		case !ok:
			t.Errorf("player %s produced no decision", id)
		case decided != action:
			t.Errorf("player %s: action = %s, want %s", id, decided, action)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d decisions for %d players", len(got), len(want))
	}
}

// runPokerScenario returns each player's decided action, keyed by player id.
//
// Poker_Decision reads `player.*` off the entity stack and has no context of
// its own, so the caller drives the iteration — one push/run/pop per player.
func runPokerScenario(t *testing.T, dir, scenarioPath string) map[string]string {
	t.Helper()

	rs := session.NewRuleSet("Poker")
	if err := rs.LoadFromDirectory(filepath.Join(dir, "xml")); err != nil {
		t.Fatalf("LoadFromDirectory: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	mapFile, err := os.Open(filepath.Join(dir, "xml/Poker_map.xml"))
	if err != nil {
		t.Fatalf("open map: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if err := m.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	dataFile, err := os.Open(scenarioPath)
	if err != nil {
		t.Fatalf("open scenario: %v", err)
	}
	defer dataFile.Close()

	if err := m.LoadDataAndPush(dataFile, []string{"game"}); err != nil {
		t.Fatalf("LoadDataAndPush: %v", err)
	}

	state := sess.GetState()
	dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Poker_Decision"))
	if err != nil {
		t.Fatalf("GetDecisionTable: %v", err)
	}

	players := arrayField(t, state, "game", "players")
	if players.Size() == 0 {
		t.Fatal("game.players is empty — the scenario data never became entities")
	}

	for i := 0; i < players.Size(); i++ {
		elem, err := players.Get(i)
		if err != nil {
			t.Fatalf("game.players[%d]: %v", i, err)
		}
		player, ok := elem.(dtrules.Entity)
		if !ok {
			t.Fatalf("game.players[%d] is %T, want an entity", i, elem)
		}
		if err := state.EntityPush(player); err != nil {
			t.Fatalf("EntityPush: %v", err)
		}
		if err := dtObj.Execute(state); err != nil {
			t.Fatalf("Poker_Decision for player %d: %v", i, err)
		}
		if _, err := state.EntityPop(); err != nil {
			t.Fatalf("EntityPop: %v", err)
		}
	}

	decisions := arrayField(t, state, "game", "decisions")
	out := make(map[string]string, decisions.Size())
	for i := 0; i < decisions.Size(); i++ {
		elem, err := decisions.Get(i)
		if err != nil {
			t.Fatalf("game.decisions[%d]: %v", i, err)
		}
		decision, ok := elem.(dtrules.Entity)
		if !ok {
			t.Fatalf("game.decisions[%d] is %T, want an entity", i, elem)
		}
		id := entityString(t, decision, "player_id")
		out[id] = entityString(t, decision, "action")
	}
	return out
}

// arrayField fetches an array attribute off whichever entity on the stack
// carries it.
func arrayField(t *testing.T, state dtrules.State, entityName, field string) *dtrules.RArray {
	t.Helper()
	ent, err := state.FindEntity(dtrules.GetRName(field))
	if err != nil {
		t.Fatalf("no entity on the stack carries %q: %v", field, err)
	}
	obj, err := ent.Get(dtrules.GetRName(field))
	if err != nil {
		t.Fatalf("%s.%s: %v", entityName, field, err)
	}
	arr, err := obj.RArrayValue()
	if err != nil {
		t.Fatalf("%s.%s is not an array: %v", entityName, field, err)
	}
	return arr
}

func entityString(t *testing.T, e dtrules.Entity, field string) string {
	t.Helper()
	obj, err := e.Get(dtrules.GetRName(field))
	if err != nil {
		t.Fatalf("%s: %v", field, err)
	}
	return obj.StringValue()
}
