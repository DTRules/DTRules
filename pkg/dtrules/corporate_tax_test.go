// Copyright 2024 Paul Snow
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package dtrules_test

// CorporateTax tests.
//
// The predecessors of these tests read a Phase-1-era repository/ mirror,
// swallowed 23 decision-table load errors as "simplified format", and ran
// zero scenarios — Total: 0, Passed: 0, Failed: 0, passing — attesting to a
// project whose primary artifact had never parsed in any commit (#948).
//
// These load the real project strictly, execute a real scenario through the
// real entry table, and check the arithmetic.

import (
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

func corporateTaxDir(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		"../../sampleprojects/CorporateTax",
		"../sampleprojects/CorporateTax",
		"sampleprojects/CorporateTax",
	} {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, "xml", "CorporateTax_core_edd.xml")); err == nil {
			return abs
		}
	}
	t.Skip("CorporateTax sample project not found")
	return ""
}

// loadCorporateTax loads the project the way `dtrules run` does — strictly.
// A load error here is a test failure, never a "note".
func loadCorporateTax(t *testing.T, dir string) *session.RuleSet {
	t.Helper()
	rs := session.NewRuleSet("CorporateTax")
	if err := rs.LoadFromDirectory(filepath.Join(dir, "xml")); err != nil {
		t.Fatalf("load: %v", err)
	}
	return rs
}

func TestCorporateTaxLoadsStrict(t *testing.T) {
	dir := corporateTaxDir(t)
	rs := loadCorporateTax(t, dir)

	if n := len(rs.GetEntityNames()); n < 19 {
		t.Errorf("entities = %d, want >= 19", n)
	}
	tables := rs.GetDecisionTableNames()
	if len(tables) < 170 {
		t.Errorf("decision tables = %d, want >= 170 (52 states x 3 + specials + orchestrator)", len(tables))
	}

	// The dispatch contract: every state resolves the full trio by the one
	// naming convention Run_Corporate_Tax constructs at runtime. EL names are
	// case-insensitive, so compare folded.
	byName := map[string]bool{}
	states := map[string]bool{}
	for _, n := range tables {
		// EL names are case-insensitive and RName interns the first spelling
		// it sees — compare folded, never by exact case.
		s := strings.ToLower(n.StringValue())
		byName[s] = true
		if strings.HasPrefix(s, "determine_") && strings.HasSuffix(s, "_filing_requirement") {
			st := strings.TrimSuffix(strings.TrimPrefix(s, "determine_"), "_filing_requirement")
			if len(st) == 2 {
				states[strings.ToUpper(st)] = true
			}
		}
	}
	if len(states) < 51 {
		t.Errorf("states with a Determine table = %d, want 51", len(states))
	}
	for st := range states {
		for _, pat := range []string{"calculate_%s_income_adjustments", "calculate_%s_state_tax"} {
			name := strings.ToLower(strings.Replace(pat, "%s", strings.ToLower(st), 1))
			if !byName[name] {
				t.Errorf("dispatch cannot resolve %s for %s", pat, st)
			}
		}
	}
}

// TestCorporateTaxScenarios runs each checked-in scenario through the entry
// table and checks the arithmetic against the state's own published form.
// Every expected figure below is traceable to a document under
// reference/forms/<STATE>/ — that is what those were downloaded for.
func TestCorporateTaxScenarios(t *testing.T) {
	cases := []struct {
		name string
		file string
		// stateFile is that state's own data, loaded as a second document.
		stateFile string
		entity    string
		expected  map[string]float64
		why       string
	}{
		{
			name:   "CA flat rate",
			file:   "TestCase_CA_flat_rate.xml",
			entity: "apportionment",
			expected: map[string]float64{
				"state_tax":            88400,
				"state_refund_or_owed": 1600,
			},
			why: "1,000,000 x 8.84%; 90,000 paid",
		},
		{
			// The bracket path. Until this scenario existed the graduated-rate
			// rows were verified by compilation only — they were hand-authored
			// from postfix that used operators which never existed, so nothing
			// had ever executed them.
			name:      "ME graduated brackets",
			file:      "TestCase_ME_graduated.xml",
			stateFile: "TestCase_ME_graduated.me.xml",
			// Maine's fields live on its own entity now: result.me_tier1_tax
			// became me_result.tier1_tax, the prefix moving off the field and
			// onto the entity.
			entity: "me_result",
			expected: map[string]float64{
				"tier1_tax":      12250,  // 350,000 x 3.5%
				"tier2_tax":      55510,  // 700,000 x 7.93%
				"tier3_tax":      204085, // 2,450,000 x 8.33%
				"tier4_tax":      44650,  // 500,000 x 8.93%
				"tax_liability":  316495, // = 271,845 + 8.93% over 3.5M
				"refund_or_owed": -16495,
			},
			why: "1120ME instructions: $271,845 plus 8.93% of the excess over $3,500,000",
		},
		{
			// A renamed _Corporate_ state, driven through its mt_result.*
			// inputs — the map tags those states needed.
			name:      "MT flat rate",
			file:      "TestCase_MT_flat_rate.xml",
			stateFile: "TestCase_MT_flat_rate.mt.xml",
			entity:    "mt_result",
			expected: map[string]float64{
				"taxable_income": 800000,
				"tax_liability":  54000,
				"refund_or_owed": -4000,
			},
			why: "Form CIT instructions: a tax of 6.75 percent on total Montana net income",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := corporateTaxDir(t)
			rs := loadCorporateTax(t, dir)

			sess, err := rs.NewSession()
			if err != nil {
				t.Fatal(err)
			}
			m := mapping.NewMapping(sess)
			mf, err := os.Open(filepath.Join(dir, "xml", "CorporateTax_map.xml"))
			if err != nil {
				t.Fatal(err)
			}
			defer mf.Close()
			if err := m.LoadMapping(mf); err != nil {
				t.Fatalf("mapping: %v", err)
			}
			df, err := os.Open(filepath.Join(dir, "testfiles", "TestScenarios", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			defer df.Close()
			// Initialize first, then load each document. Initialize creates
			// and pushes the singletons and records them, and newDataLoader
			// seeds from that record -- so a second document binds to the
			// instances already on the stack instead of building its own.
			//
			// LoadDataAndPushSingletons is the single-document shortcut: it
			// pushes whatever *its* load built, so calling it twice puts a
			// second set of instances on the stack and the federal values are
			// left on the pair nobody is reading (#1094).
			if err := m.Initialize(); err != nil {
				t.Fatalf("initialize: %v", err)
			}
			if err := m.LoadData(df); err != nil {
				t.Fatalf("data: %v", err)
			}
			if tc.stateFile != "" {
				sf, err := os.Open(filepath.Join(dir, "testfiles", "TestScenarios", tc.stateFile))
				if err != nil {
					t.Fatal(err)
				}
				defer sf.Close()
				if err := m.LoadData(sf); err != nil {
					t.Fatalf("state data: %v", err)
				}
			}

			state := sess.GetState()
			entry, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Run_Corporate_Tax"))
			if err != nil || entry == nil {
				t.Fatalf("entry table: %v", err)
			}
			if err := entry.Execute(state); err != nil {
				t.Fatalf("execute: %v", err)
			}

			for field, want := range tc.expected {
				got, ok := entityDouble(state, tc.entity, field)
				if !ok {
					t.Errorf("%s.%s not readable", tc.entity, field)
					continue
				}
				if math.Abs(got-want) > 0.005 {
					t.Errorf("%s.%s = %v, want %v (%s)", tc.entity, field, got, want, tc.why)
				}
			}
		})
	}
}

// entityDouble reads a double field off a named entity on the stack.
func entityDouble(state dtrules.State, entity, field string) (float64, bool) {
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != entity {
			continue
		}
		v, err := e.Get(dtrules.GetRName(field))
		if err != nil || v == nil {
			return 0, false
		}
		f, err := v.DoubleValue()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// TestCorporateTaxNoHandCodedPostfix is the campaign gate (#948): postfix is
// a compiled artifact of the EL DSL, never authored. A row carrying postfix
// with no DSL is deleted by the next authoring write, silently — this must
// never regress.
func TestCorporateTaxNoHandCodedPostfix(t *testing.T) {
	dir := corporateTaxDir(t)

	files, err := filepath.Glob(filepath.Join(dir, "xml", "states", "*_corp_dt.xml"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, filepath.Join(dir, "xml", "orchestrator_dt.xml"))

	var hand []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Tables []struct {
				Name     string `xml:"table_name"`
				Contexts []struct {
					DSL string `xml:"context_dsl"`
					PF  string `xml:"context_postfix"`
				} `xml:"contexts>context_details"`
				Inits []struct {
					DSL string `xml:"initial_action_dsl"`
					PF  string `xml:"initial_action_postfix"`
				} `xml:"initial_actions>initial_action"`
				Conds []struct {
					DSL string `xml:"condition_dsl"`
					PF  string `xml:"condition_postfix"`
				} `xml:"conditions>condition_details"`
				Acts []struct {
					DSL string `xml:"action_dsl"`
					PF  string `xml:"action_postfix"`
				} `xml:"actions>action_details"`
			} `xml:"decision_table"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		check := func(tn, kind, dsl, pf string) {
			if strings.TrimSpace(dsl) == "" && strings.TrimSpace(pf) != "" {
				hand = append(hand, tn+" ("+kind+")")
			}
		}
		for _, tb := range doc.Tables {
			for _, r := range tb.Contexts {
				check(tb.Name, "context", r.DSL, r.PF)
			}
			for _, r := range tb.Inits {
				check(tb.Name, "initial action", r.DSL, r.PF)
			}
			for _, r := range tb.Conds {
				check(tb.Name, "condition", r.DSL, r.PF)
			}
			for _, r := range tb.Acts {
				check(tb.Name, "action", r.DSL, r.PF)
			}
		}
	}
	if len(hand) > 0 {
		t.Errorf("%d rows carry postfix without DSL — the next authoring write deletes them:\n  %s",
			len(hand), strings.Join(hand, "\n  "))
	}
}
