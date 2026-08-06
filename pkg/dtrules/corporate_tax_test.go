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

// TestCorporateTaxCAScenario runs the checked-in California scenario through
// the entry table and checks the arithmetic: 1,000,000 x 8.84% = 88,400 tax,
// 90,000 of payments leaves a 1,600 refund.
func TestCorporateTaxCAScenario(t *testing.T) {
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
	df, err := os.Open(filepath.Join(dir, "testfiles", "TestScenarios", "TestCase_CA_flat_rate.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer df.Close()
	if err := m.LoadDataAndPushSingletons(df); err != nil {
		t.Fatalf("data: %v", err)
	}

	state := sess.GetState()
	entry, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Run_Corporate_Tax"))
	if err != nil || entry == nil {
		t.Fatalf("entry table: %v", err)
	}
	if err := entry.Execute(state); err != nil {
		t.Fatalf("execute: %v", err)
	}

	get := func(field string) float64 {
		t.Helper()
		for i := 0; i < state.EntityDepth(); i++ {
			e, _ := state.EntityFetch(i)
			if e == nil || e.GetName().StringValue() != "apportionment" {
				continue
			}
			v, err := e.Get(dtrules.GetRName(field))
			if err != nil || v == nil {
				t.Fatalf("apportionment.%s: %v", field, err)
			}
			f, err := v.DoubleValue()
			if err != nil {
				t.Fatalf("apportionment.%s not a double: %v", field, err)
			}
			return f
		}
		t.Fatalf("apportionment not on the entity stack")
		return 0
	}

	if tax := get("state_tax"); math.Abs(tax-88400.0) > 0.005 {
		t.Errorf("state_tax = %v, want 88400 (1,000,000 x 8.84%%)", tax)
	}
	if refund := get("state_refund_or_owed"); math.Abs(refund-1600.0) > 0.005 {
		t.Errorf("state_refund_or_owed = %v, want 1600 (90,000 - 88,400)", refund)
	}
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
