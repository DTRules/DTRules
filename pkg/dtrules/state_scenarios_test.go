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
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
)

// stateScenarioStates are the states whose testfiles/TestScenarios/<XX>
// scenarios this test holds to their expected_state_tax: the #1200 states,
// whose tables computed nothing until they were rewritten.
var stateScenarioStates = []string{"AR", "LA"}

// TestStateScenarios runs each scenario under testfiles/TestScenarios/<XX>
// and compares that state's roster entry -- what Compute_Roster_State_Tax
// harvested from XX_Tax -- with the figure the scenario works out by hand.
// Validate_Summary does not check expected_state_tax (#1317), so this does.
func TestStateScenarios(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	expected := regexp.MustCompile(`<expected_state_tax>([0-9.]+)</expected_state_tax>`)
	for _, code := range stateScenarioStates {
		dir := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", code)
		files, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
		if len(files) < 3 {
			t.Errorf("want at least 3 %s scenarios, found %d", code, len(files))
		}
		for _, path := range files {
			t.Run(code+"/"+filepath.Base(path), func(t *testing.T) {
				src, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				m := expected.FindSubmatch(src)
				if m == nil {
					t.Fatal("scenario has no <expected_state_tax>")
				}
				want, _ := strconv.ParseFloat(string(m[1]), 64)

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
				if auditHasFailure(state) {
					t.Error("the rules' own validation reported a FAIL line")
				}

				got, found := stateRosterTax(state, code)
				if !found {
					t.Fatalf("no %s entry on job.state_tax_results", code)
				}
				if got != want {
					t.Errorf("%s tax $%.2f, scenario works out $%.2f", code, got, want)
				}
			})
		}
	}
}

// stateRosterTax returns the state_tax_before_credits of the roster entry for
// the given state code.
func stateRosterTax(state dtrules.State, stateCode string) (float64, bool) {
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
