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

// A setattribute resolves its enclosure against the entity stack while the
// document is being read. An entity that is only an <initialentity> -- nothing
// names it in a createentity -- was pushed only after the read, so the lookup
// missed and every attribute it enclosed was dropped in silence. A
// default-valued instance was then pushed on top and that is what executed.
//
// TaxReturn is the shape that exposes it: `job` is an initialentity carrying
// 20-odd mapped attributes and no createentity. `dtrules run --input` reported
// job.state as the EDD default "TX" on a scenario that says "OH", with an
// empty filing status and a return that computed to zero -- looking like a
// plausible answer the whole way (#1168).
//
// The two load paths must agree: Initialize + LoadData is what the scenario
// corpus uses, LoadDataAndPushSingletons is what `dtrules run` and the trace
// tooling use, and a project cannot mean two different things depending on
// which one read it.
func TestBothLoadPathsSeeTheSameSingletonData(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")
	if _, err := os.Stat(xmlDir); err != nil {
		t.Skip("TaxReturn sample not present")
	}
	scenario := filepath.Join(sampleDir, "testfiles", "TestScenarios",
		"Integration", "TestCase_Integration_01_Family_All_Credits.xml")

	rs := session.NewRuleSet("TaxReturn")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("load rules: %v", err)
	}

	// load reads the scenario through one of the two paths and reports the
	// job attributes that arrive only as enclosure='job' setattributes.
	load := func(pushSingletons bool) (state, filing, expectedAGI string) {
		sess, err := rs.NewSession()
		if err != nil {
			t.Fatalf("session: %v", err)
		}
		m := mapping.NewMapping(sess)
		mf, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
		if err != nil {
			t.Fatalf("open mapping: %v", err)
		}
		defer mf.Close()
		if err := m.LoadMapping(mf); err != nil {
			t.Fatalf("load mapping: %v", err)
		}
		f, err := os.Open(scenario)
		if err != nil {
			t.Fatalf("open scenario: %v", err)
		}
		defer f.Close()

		if pushSingletons {
			if err := m.LoadDataAndPushSingletons(f); err != nil {
				t.Fatalf("LoadDataAndPushSingletons: %v", err)
			}
		} else {
			if err := m.Initialize(); err != nil {
				t.Fatalf("initialize: %v", err)
			}
			if err := m.LoadData(f); err != nil {
				t.Fatalf("LoadData: %v", err)
			}
		}

		st := sess.GetState()
		get := func(attr string) string {
			for i := 0; i < st.EntityDepth(); i++ {
				e, _ := st.EntityFetch(i)
				if e == nil || e.GetName().StringValue() != "job" {
					continue
				}
				if v, _ := e.Get(dtrules.GetRName(attr)); v != nil {
					return v.StringValue()
				}
			}
			return ""
		}
		return get("state"), get("filing_status"), get("expected_agi")
	}

	wantState, wantFiling, wantAGI := "OH", "MFJ", "73200"

	for _, tc := range []struct {
		name           string
		pushSingletons bool
	}{
		{"Initialize+LoadData", false},
		{"LoadDataAndPushSingletons", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotState, gotFiling, gotAGI := load(tc.pushSingletons)
			if gotState != wantState {
				t.Errorf("job.state = %q, want %q — the scenario's value was dropped and "+
					"the EDD default is what executed", gotState, wantState)
			}
			if gotFiling != wantFiling {
				t.Errorf("job.filing_status = %q, want %q", gotFiling, wantFiling)
			}
			if gotAGI != wantAGI {
				t.Errorf("job.expected_agi = %q, want %q — an expectation that does not load "+
					"asserts nothing", gotAGI, wantAGI)
			}
		})
	}
}
