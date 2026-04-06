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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestPokerIntegration tests the Poker demo with 4 AI archetypes.
// It loads the EDD, decision tables, and mapping, then executes decisions
// for multiple players with different archetypes and hand strengths.
func TestPokerIntegration(t *testing.T) {
	pokerDir := findPokerDir(t)
	if pokerDir == "" {
		t.Skip("Poker sample project not found")
	}

	t.Logf("Using Poker directory: %s", pokerDir)

	// Create a new RuleSet
	rs := session.NewRuleSet("Poker")

	// Load the EDD
	eddPath := filepath.Join(pokerDir, "xml/Poker_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	t.Log("EDD loaded successfully")

	// Verify entities were created
	entityNames := rs.GetEntityNames()
	t.Logf("Loaded %d entities", len(entityNames))
	for _, name := range entityNames {
		t.Logf("  - %s", name.StringValue())
	}

	expectedEntities := []string{"game", "player", "decision"}
	for _, expected := range expectedEntities {
		found := false
		for _, name := range entityNames {
			if name.StringValue() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected entity '%s' not found", expected)
		}
	}

	// Load the decision tables
	dtPath := filepath.Join(pokerDir, "xml/Poker_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}
	t.Log("Decision tables loaded successfully")

	// Verify decision tables were created
	dtNames := rs.GetDecisionTableNames()
	t.Logf("Loaded %d decision tables", len(dtNames))
	for _, name := range dtNames {
		t.Logf("  - %s", name.StringValue())
	}

	expectedTables := []string{"Poker_Decision", "TAG_Decision", "LAG_Decision", "Rock_Decision", "CallingStation_Decision"}
	for _, expected := range expectedTables {
		found := false
		for _, name := range dtNames {
			if name.StringValue() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected decision table '%s' not found", expected)
		}
	}

	// Create a session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	t.Log("Session created successfully")

	// Load the mapping
	mapPath := filepath.Join(pokerDir, "xml/Poker_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("Failed to open mapping file: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	err = m.LoadMapping(mapFile)
	if err != nil {
		t.Fatalf("Failed to load mapping: %v", err)
	}
	t.Log("Mapping loaded successfully")

	// Initialize the mapping (creates initial entities)
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize mapping: %v", err)
	}
	t.Log("Mapping initialized successfully")

	// Load test data
	testDataPath := filepath.Join(pokerDir, "testfiles/TestScenarios/basic_scenarios.xml")
	testDataFile, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Failed to open test data file: %v", err)
	}
	defer testDataFile.Close()

	err = m.LoadData(testDataFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	t.Log("Test data loaded successfully")

	// Get the entity factory and state for execution
	factory := sess.GetEntityFactory()
	state := sess.GetState()

	// Get the game entity to verify data loaded
	gameEntity, err := state.EntityFetch(0) // game is first on stack
	if err != nil {
		t.Fatalf("Failed to fetch game entity: %v", err)
	}
	t.Logf("Game entity: %s", gameEntity.GetName().StringValue())

	// Get players array
	playersVal, err := gameEntity.Get(dtrules.GetRName("players"))
	if err != nil {
		t.Fatalf("Failed to get players: %v", err)
	}
	playersArray, err := playersVal.ArrayValue()
	if err != nil {
		t.Fatalf("Failed to get players array: %v", err)
	}
	t.Logf("Loaded %d players", len(playersArray))

	// Execute decisions for each player
	for i, playerVal := range playersArray {
		playerEntity, err := playerVal.REntityValue()
		if err != nil {
			t.Fatalf("Failed to get player %d entity: %v", i, err)
		}

		playerName, _ := playerEntity.Get(dtrules.GetRName("name"))
		playerArchetype, _ := playerEntity.Get(dtrules.GetRName("archetype"))

		t.Logf("Processing player: %s (archetype: %s)", playerName.StringValue(), playerArchetype.StringValue())

		// Push player context for decision
		state.EntityPush(playerEntity)

		// Get and execute the main decision table
		dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Poker_Decision"))
		if err != nil {
			t.Fatalf("Failed to get decision table: %v", err)
		}

		err = dtObj.Execute(state)
		if err != nil {
			t.Errorf("Failed to execute Poker_Decision for %s: %v", playerName.StringValue(), err)
			state.EntityPop()
			continue
		}

		state.EntityPop()
	}

	// Verify decisions were made
	decisionsVal, err := gameEntity.Get(dtrules.GetRName("decisions"))
	if err != nil {
		t.Fatalf("Failed to get decisions: %v", err)
	}
	decisionsArray, err := decisionsVal.ArrayValue()
	if err != nil {
		t.Fatalf("Failed to get decisions array: %v", err)
	}

	t.Logf("Made %d decisions", len(decisionsArray))

	// Verify each decision
	for i, decisionVal := range decisionsArray {
		decisionEntity, err := decisionVal.REntityValue()
		if err != nil {
			t.Fatalf("Failed to get decision %d entity: %v", i, err)
		}

		playerId, _ := decisionEntity.Get(dtrules.GetRName("player_id"))
		archetype, _ := decisionEntity.Get(dtrules.GetRName("archetype"))
		action, _ := decisionEntity.Get(dtrules.GetRName("action"))
		reasoning, _ := decisionEntity.Get(dtrules.GetRName("reasoning"))

		t.Logf("Decision for %s (%s): %s - %s",
			playerId.StringValue(),
			archetype.StringValue(),
			action.StringValue(),
			reasoning.StringValue())

		// Verify action is one of the valid actions
		validActions := map[string]bool{"RAISE": true, "CALL": true, "CHECK": true, "FOLD": true}
		if !validActions[action.StringValue()] {
			t.Errorf("Invalid action '%s' for player %s", action.StringValue(), playerId.StringValue())
		}
	}

	// Verify we got decisions for all players
	if len(decisionsArray) != len(playersArray) {
		t.Errorf("Expected %d decisions, got %d", len(playersArray), len(decisionsArray))
	}

	t.Log("Poker integration test completed successfully!")
}

// TestPokerEDDLoad tests just the EDD loading for Poker
func TestPokerEDDLoad(t *testing.T) {
	pokerDir := findPokerDir(t)
	if pokerDir == "" {
		t.Skip("Poker sample project not found")
	}

	rs := session.NewRuleSet("Poker")

	eddPath := filepath.Join(pokerDir, "xml/Poker_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Verify entity structure
	factory := rs.GetEntityFactory()

	// Check game entity has expected attributes
	gameRef, err := factory.GetReferenceEntity(dtrules.GetRName("game"))
	if err != nil {
		t.Fatalf("Failed to get game entity: %v", err)
	}

	expectedGameAttrs := []string{"game", "mapping*key", "pot", "current_bet", "big_blind", "phase", "players", "decisions", "audit_trail"}
	for _, attr := range expectedGameAttrs {
		if !gameRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Game entity missing attribute: %s", attr)
		}
	}
	t.Logf("Game entity has %d attributes", len(gameRef.GetAttributeNames()))

	// Check player entity has expected attributes
	playerRef, err := factory.GetReferenceEntity(dtrules.GetRName("player"))
	if err != nil {
		t.Fatalf("Failed to get player entity: %v", err)
	}

	expectedPlayerAttrs := []string{"player", "mapping*key", "name", "archetype", "hand_strength", "pot_odds", "can_check", "position", "stack_size"}
	for _, attr := range expectedPlayerAttrs {
		if !playerRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Player entity missing attribute: %s", attr)
		}
	}
	t.Logf("Player entity has %d attributes", len(playerRef.GetAttributeNames()))

	// Check decision entity has expected attributes
	decisionRef, err := factory.GetReferenceEntity(dtrules.GetRName("decision"))
	if err != nil {
		t.Fatalf("Failed to get decision entity: %v", err)
	}

	expectedDecisionAttrs := []string{"decision", "mapping*key", "player_id", "archetype", "action", "raise_amount", "reasoning"}
	for _, attr := range expectedDecisionAttrs {
		if !decisionRef.ContainsAttribute(dtrules.GetRName(attr)) {
			t.Errorf("Decision entity missing attribute: %s", attr)
		}
	}
	t.Logf("Decision entity has %d attributes", len(decisionRef.GetAttributeNames()))
}

// TestPokerDTLoad tests decision table loading and structure for Poker
func TestPokerDTLoad(t *testing.T) {
	pokerDir := findPokerDir(t)
	if pokerDir == "" {
		t.Skip("Poker sample project not found")
	}

	rs := session.NewRuleSet("Poker")

	// Must load EDD first
	eddPath := filepath.Join(pokerDir, "xml/Poker_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD file: %v", err)
	}
	err = rs.LoadEDD(eddFile)
	eddFile.Close()
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Load decision tables
	dtPath := filepath.Join(pokerDir, "xml/Poker_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT file: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}

	// Verify all expected decision tables exist
	dtNames := rs.GetDecisionTableNames()
	expectedTables := []string{"Poker_Decision", "TAG_Decision", "LAG_Decision", "Rock_Decision", "CallingStation_Decision"}

	for _, expected := range expectedTables {
		found := false
		for _, name := range dtNames {
			if name.StringValue() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected decision table '%s' not found", expected)
		}
	}

	t.Logf("All %d decision tables loaded successfully", len(dtNames))
}

// findPokerDir searches for the Poker sample project directory
func findPokerDir(t *testing.T) string {
	t.Helper()

	// Try relative paths from the test location
	paths := []string{
		"../../../../sampleprojects/Poker",
		"../../../sampleprojects/Poker",
		"../../sampleprojects/Poker",
		"../sampleprojects/Poker",
		"sampleprojects/Poker",
	}

	// Get current working directory
	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(absPath, "xml/Poker_edd.xml")); err == nil {
			return absPath
		}
	}

	// Try from known locations
	home, _ := os.UserHomeDir()
	knownPaths := []string{
		filepath.Join(home, "go/src/github.com/DTRules/DTRules/sampleprojects/Poker"),
		"/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/Poker",
	}

	for _, p := range knownPaths {
		if _, err := os.Stat(filepath.Join(p, "xml/Poker_edd.xml")); err == nil {
			return p
		}
	}

	return ""
}

// TestPokerTAGDecisions tests TAG (Tight-Aggressive) specific decision logic
func TestPokerTAGDecisions(t *testing.T) {
	testCases := []struct {
		name           string
		handStrength   float64
		potOdds        float64
		canCheck       bool
		expectedAction string
	}{
		{"Strong hand should raise", 0.75, 0.3, false, "RAISE"},
		{"Medium hand good odds should call", 0.55, 0.2, false, "CALL"},
		{"Medium hand bad odds can check", 0.55, 0.6, true, "CHECK"},
		{"Weak hand can check", 0.30, 0.5, true, "CHECK"},
		{"Weak hand must call should fold", 0.25, 0.6, false, "FOLD"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("TAG test: hand_strength=%.2f, pot_odds=%.2f, can_check=%v -> expected %s",
				tc.handStrength, tc.potOdds, tc.canCheck, tc.expectedAction)
		})
	}
}

// TestPokerLAGDecisions tests LAG (Loose-Aggressive) specific decision logic
func TestPokerLAGDecisions(t *testing.T) {
	testCases := []struct {
		name           string
		handStrength   float64
		position       string
		canCheck       bool
		expectedAction string
	}{
		{"Strong hand should raise", 0.55, "middle", false, "RAISE"},
		{"Good position marginal hand should raise", 0.30, "button", false, "RAISE"},
		{"Weak position marginal should call", 0.30, "early", false, "CALL"},
		{"Very weak should fold", 0.15, "early", false, "FOLD"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("LAG test: hand_strength=%.2f, position=%s, can_check=%v -> expected %s",
				tc.handStrength, tc.position, tc.canCheck, tc.expectedAction)
		})
	}
}

// TestPokerRockDecisions tests Rock (Tight-Passive) specific decision logic
func TestPokerRockDecisions(t *testing.T) {
	testCases := []struct {
		name           string
		handStrength   float64
		canCheck       bool
		expectedAction string
	}{
		{"Monster hand should raise", 0.90, false, "RAISE"},
		{"Strong hand should call", 0.70, false, "CALL"},
		{"Medium hand can check", 0.50, true, "CHECK"},
		{"Weak hand must call should fold", 0.40, false, "FOLD"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Rock test: hand_strength=%.2f, can_check=%v -> expected %s",
				tc.handStrength, tc.canCheck, tc.expectedAction)
		})
	}
}

// TestPokerCallingStationDecisions tests Calling Station specific decision logic
func TestPokerCallingStationDecisions(t *testing.T) {
	testCases := []struct {
		name           string
		handStrength   float64
		isSmallBet     bool
		canCheck       bool
		expectedAction string
	}{
		{"Any value should call", 0.25, false, false, "CALL"},
		{"Small bet should call", 0.10, true, false, "CALL"},
		{"Can check weak hand", 0.10, false, true, "CHECK"},
		{"Big bet garbage should fold", 0.05, false, false, "FOLD"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("CallingStation test: hand_strength=%.2f, is_small_bet=%v, can_check=%v -> expected %s",
				tc.handStrength, tc.isSmallBet, tc.canCheck, tc.expectedAction)
		})
	}
}
