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
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// nonresidentTax matches the figure a MultiState scenario works out from a
// state's non-resident or part-year form, e.g.
// <expected_ky_nonresident_tax>589.20</...> or
// <expected_ma_part_year_tax>1668.00</...>.
var nonresidentTax = regexp.MustCompile(`<expected_([a-z]{2})_(nonresident|part_year)_tax>([0-9.]+)</expected_[a-z]{2}_(?:nonresident|part_year)_tax>`)

// TestNonresidentScenarios runs every MultiState scenario that carries an
// expected_XX_nonresident_tax or expected_XX_part_year_tax and compares the
// XX roster entry -- what Compute_Roster_State_Tax harvested from XX_Tax for
// that entry -- with the figure the scenario works out from that state's
// non-resident / part-year worksheet (#1201). Nothing in the rules checks
// these figures (#1317), so this does.
//
// This file stands on its own (its helpers are not shared with
// state_scenarios_test.go, which #1329 also changes) so that it survives
// either merge order.
func TestNonresidentScenarios(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	dir := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", "MultiState")
	files, _ := filepath.Glob(filepath.Join(dir, "TestCase_MultiState_*.xml"))
	var checked int
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		matches := nonresidentTax.FindAllSubmatch(src, -1)
		if len(matches) == 0 {
			continue
		}
		t.Run(filepath.Base(path), func(t *testing.T) {
			state := runMultiStateScenario(t, rs, xmlDir, src)
			if auditHasFailure(state) {
				t.Error("the rules' own validation reported a FAIL line")
			}
			for _, m := range matches {
				code := strings.ToUpper(string(m[1]))
				want, _ := strconv.ParseFloat(string(m[3]), 64)
				got, found := nonresidentRosterTax(state, code)
				if !found {
					t.Errorf("no %s entry on job.state_tax_results", code)
					continue
				}
				// Half a cent: the forms work in dollars and cents, the
				// engine in float64.
				if math.Abs(got-want) > 0.005 {
					t.Errorf("%s %s tax $%.2f, scenario works out $%.2f", code, m[2], got, want)
				}
				checked++
			}
		})
	}
	if checked < 7 {
		t.Errorf("checked %d non-resident and part-year figures; the KY and six MA scenarios make at least 7", checked)
	}
}

// runMultiStateScenario loads one scenario through the TaxReturn mapping and runs
// Compute_Tax_Return on it, returning the finished state.
func runMultiStateScenario(t *testing.T, rs *session.RuleSet, xmlDir string, src []byte) dtrules.State {
	t.Helper()
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	mp := mapping.NewMapping(sess)
	mf, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer mf.Close()
	if err := mp.LoadMapping(mf); err != nil {
		t.Fatal(err)
	}
	if err := mp.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := mp.LoadData(strings.NewReader(string(src))); err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		t.Fatalf("Compute_Tax_Return: %v", err)
	}
	if err := dt.Execute(state); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return state
}

// nonresidentRosterTax returns the state_tax_before_credits of the roster entry for
// the given state code.
func nonresidentRosterTax(state dtrules.State, stateCode string) (float64, bool) {
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != "job" {
			continue
		}
		v, _ := e.Get(dtrules.GetRName("state_tax_results"))
		if v == nil {
			continue
		}
		arr, err := v.ArrayValue()
		if err != nil {
			continue
		}
		for _, o := range arr {
			entry, ok := o.(dtrules.Entity)
			if !ok {
				continue
			}
			code, _ := entry.Get(dtrules.GetRName("state_code"))
			if code == nil || code.StringValue() != stateCode {
				continue
			}
			tax, _ := entry.Get(dtrules.GetRName("state_tax_before_credits"))
			f, _ := tax.DoubleValue()
			return f, true
		}
	}
	return 0, false
}
