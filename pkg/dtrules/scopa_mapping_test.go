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
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestScopaScenariosThroughTheMapping runs the Scopa sample the way the CLI
// does — scenario XML through the mapping — rather than by building entities
// in Go, matching what the Cribbage sample gained in #1163.
//
// Scopa needs two singletons where Cribbage needed one: a capture is asked
// of a move, primiera is asked of a player, and both entry tables must find
// their root. The same card entity arrives on two streams, told apart by
// tag: <table_card> lands on move.table_cards, <pile_card> on player.pile.
func TestScopaScenariosThroughTheMapping(t *testing.T) {
	cases := []struct {
		file, entry, root string
		want              map[string]int
		why               string
	}{
		{
			file: "single_beats_sum.xml", entry: "Resolve_Capture", root: "move",
			want: map[string]int{"capture_found": 1, "capture_size": 1},
			why:  "a lone 7 on the table bars the 3+4 sum",
		},
		{
			file: "sweep.xml", entry: "Resolve_Capture", root: "move",
			want: map[string]int{"capture_found": 1, "capture_size": 2},
			why:  "no single 7, so 3+4 takes both and clears the table",
		},
		{
			file: "primiera.xml", entry: "Score_Primiera", root: "player",
			want: map[string]int{"primiera_total": 68},
			why:  "best card per suit: 21 + 21 + 16 + 10",
		},
	}

	sampleDir := filepath.Join("..", "..", "sampleprojects", "Scopa")
	xmlDir := filepath.Join(sampleDir, "xml")

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			rs := session.NewRuleSet("ScopaMapped")
			for _, f := range []string{"project_edd.xml", "Scopa_dt.xml"} {
				fh, err := os.Open(filepath.Join(xmlDir, f))
				if err != nil {
					t.Fatalf("open %s: %v", f, err)
				}
				if f == "project_edd.xml" {
					err = rs.LoadEDD(fh)
				} else {
					err = rs.LoadDecisionTables(fh)
				}
				fh.Close()
				if err != nil {
					t.Fatalf("load %s: %v", f, err)
				}
			}
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("NewSession: %v", err)
			}

			mapFile, err := os.Open(filepath.Join(xmlDir, "Scopa_map.xml"))
			if err != nil {
				t.Fatalf("open mapping: %v", err)
			}
			defer mapFile.Close()
			m := mapping.NewMapping(sess)
			if err := m.LoadMapping(mapFile); err != nil {
				t.Fatalf("LoadMapping: %v", err)
			}
			input, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", tc.file))
			if err != nil {
				t.Fatalf("open scenario: %v", err)
			}
			defer input.Close()
			if err := m.LoadDataAndPushSingletons(input); err != nil {
				t.Fatalf("LoadDataAndPushSingletons: %v", err)
			}

			dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(tc.entry))
			if err != nil {
				t.Fatalf("GetDecisionTable(%s): %v", tc.entry, err)
			}
			if err := dt.Execute(sess.GetState()); err != nil {
				t.Fatalf("%s: %v", tc.entry, err)
			}

			root, err := sess.GetState().FindEntity(dtrules.GetRName(tc.root))
			if err != nil || root == nil {
				t.Fatalf("no %s entity after the run: %v", tc.root, err)
			}
			for field, want := range tc.want {
				v, err := root.Get(dtrules.GetRName(field))
				if err != nil || v == nil {
					t.Fatalf("get %s: %v", field, err)
				}
				got, _ := v.IntValue()
				if got != want {
					t.Errorf("%s (%s): %s = %d, want %d", tc.file, tc.why, field, got, want)
				}
			}
		})
	}

	t.Run("scopa flag and settebello", func(t *testing.T) {
		// Booleans, checked separately from the integer fields above.
		for _, c := range []struct {
			file, entry, root, field string
			want                     bool
		}{
			{"sweep.xml", "Resolve_Capture", "move", "is_scopa", true},
			{"single_beats_sum.xml", "Resolve_Capture", "move", "is_scopa", false},
			{"primiera.xml", "Score_Primiera", "player", "has_settebello", true},
		} {
			rs := session.NewRuleSet("ScopaMappedBool")
			for _, f := range []string{"project_edd.xml", "Scopa_dt.xml"} {
				fh, _ := os.Open(filepath.Join(xmlDir, f))
				if f == "project_edd.xml" {
					rs.LoadEDD(fh)
				} else {
					rs.LoadDecisionTables(fh)
				}
				fh.Close()
			}
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatal(err)
			}
			mapFile, _ := os.Open(filepath.Join(xmlDir, "Scopa_map.xml"))
			m := mapping.NewMapping(sess)
			if err := m.LoadMapping(mapFile); err != nil {
				t.Fatal(err)
			}
			mapFile.Close()
			input, _ := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", c.file))
			if err := m.LoadDataAndPushSingletons(input); err != nil {
				t.Fatal(err)
			}
			input.Close()
			dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(c.entry))
			if err != nil {
				t.Fatal(err)
			}
			if err := dt.Execute(sess.GetState()); err != nil {
				t.Fatal(err)
			}
			root, _ := sess.GetState().FindEntity(dtrules.GetRName(c.root))
			v, err := root.Get(dtrules.GetRName(c.field))
			if err != nil || v == nil {
				t.Fatalf("get %s: %v", c.field, err)
			}
			got, _ := v.BooleanValue()
			if got != c.want {
				t.Errorf("%s: %s = %v, want %v", c.file, c.field, got, c.want)
			}
		}
	})
}
