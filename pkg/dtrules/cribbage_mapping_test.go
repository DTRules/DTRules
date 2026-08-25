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

// TestCribbageScenariosThroughTheMapping runs the sample the way a person
// with only the CLI runs it — `dtrules run --input <scenario>` — rather than
// the way its other tests do, which build entities in Go.
//
// Until the mapping existed the sample could not be executed at all without
// host code: `dtrules run` failed with "hand.fifteen_points is undefined"
// because nothing instantiated the hand. That made the whole #980 primitive
// family demonstrable only to someone willing to write Go.
//
// The scenarios state only what a player can see — rank, suit, and which
// card was cut. Counting value, the kept four, and the starter's suit are
// derived by Score_Hand's initial actions, so a scenario cannot be made
// internally inconsistent (a "value" that disagrees with its rank, a kept
// list that disagrees with the cut).
func TestCribbageScenariosThroughTheMapping(t *testing.T) {
	cases := []struct {
		file  string
		total int
		why   string
	}{
		{"perfect_29.xml", 29, "the highest hand in cribbage"},
		{"double_run.xml", 14, "double run of four, a pair, two fifteens"},
		{"crib_flush_denied.xml", 0, "four kept hearts, spade cut, in the crib: no flush"},
		{"nobs.xml", 5, "his nobs plus two fifteens"},
	}

	sampleDir := filepath.Join("..", "..", "sampleprojects", "Cribbage")
	xmlDir := filepath.Join(sampleDir, "xml")

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			rs := session.NewRuleSet("CribbageMapped")
			for _, f := range []string{"project_edd.xml", "Cribbage_dt.xml"} {
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

			mapFile, err := os.Open(filepath.Join(xmlDir, "Cribbage_map.xml"))
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
			// This is the step that did not exist before the mapping: the
			// scenario's XML becomes a hand entity, its cards, and a pushed
			// singleton — with no Go code describing the shape.
			if err := m.LoadDataAndPushSingletons(input); err != nil {
				t.Fatalf("LoadDataAndPushSingletons: %v", err)
			}

			dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Score_Hand"))
			if err != nil {
				t.Fatalf("GetDecisionTable: %v", err)
			}
			if err := dt.Execute(sess.GetState()); err != nil {
				t.Fatalf("Score_Hand: %v", err)
			}

			hand, err := sess.GetState().FindEntity(dtrules.GetRName("hand"))
			if err != nil || hand == nil {
				t.Fatalf("no hand entity after the run: %v", err)
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
				t.Errorf("%s (%s): total = %d, want %d — fifteens %d, pairs %d, runs %d, flush %d, nobs %d",
					tc.file, tc.why, got, tc.total,
					get("fifteen_points"), get("pair_points"), get("run_points"),
					get("flush_points"), get("nobs_points"))
			}
		})
	}
}
