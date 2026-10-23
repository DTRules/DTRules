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
)

// TestMilitaryRetirementReachesRules proves that a taxpayer's
// military_retirement_pay is read by the rules (#1330). Before the mapping
// carried the tag the pension never reached the taxpayer entity, so the 20
// military-retirement scenarios computed AGI from wages alone: IL_01's
// $45,000 pension on top of $30,000 of wages gave an AGI of $30,000.
//
// Military retired pay based on age or length of service is taxable federal
// income, reported on Form 1040 line 5b (IRC 61(a)(10); Pub. 525, "Military
// retirement pay"), so the federal AGI is the two together.
func TestMilitaryRetirementReachesRules(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	path := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", "State", "FullExemption",
		"TestCase_State_IL_01_Military_Retirement_Full_Exemption.xml")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<w2_wages>30000</w2_wages>", "<military_retirement_pay>45000</military_retirement_pay>"} {
		if strings.Count(string(src), want) != 1 {
			t.Fatalf("IL_01 should carry %s once", want)
		}
	}

	state := runScenarioForStateTaxCheck(t, rs, xmlDir, src)
	agi, ok := resultDouble(state, "agi")
	if !ok {
		t.Fatal("no result entity on job.results")
	}
	if math.Abs(agi-75000) > 0.5 {
		t.Errorf("IL_01 AGI = $%.2f, want $75,000 ($30,000 wages + $45,000 military retirement pay)", agi)
	}
	pension, ok := resultDouble(state, "military_retirement_income")
	if !ok || math.Abs(pension-45000) > 0.5 {
		t.Errorf("result.military_retirement_income = $%.2f, want $45,000", pension)
	}
	lines := auditTrailLines(state)
	if anyLineContains(lines, "FAIL:") {
		t.Errorf("IL_01 should validate cleanly; audit trail:\n%s", strings.Join(lines, "\n"))
	}
}

// resultDouble reads a double field off the result entity that
// Compute_Tax_Return pushes onto job.results.
func resultDouble(state dtrules.State, field string) (float64, bool) {
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != "job" {
			continue
		}
		v, _ := e.Get(dtrules.GetRName("results"))
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
			val, _ := entry.Get(dtrules.GetRName(field))
			if val == nil {
				return 0, false
			}
			f, err := val.DoubleValue()
			return f, err == nil
		}
	}
	return 0, false
}

// militaryRetirementScenarios are the military-retirement scenarios whose
// state table computes its 2025 form (#1330). The others fail only their state
// check, against #1200 placeholder tables, and the coverage ratchet counts
// them; this holds the ones that are right so they cannot drift.
var militaryRetirementScenarios = []struct{ dir, file, state string }{
	{"FullExemption", "TestCase_State_IL_01_Military_Retirement_Full_Exemption.xml", "IL"},
	{"PartialExemption", "TestCase_State_CO_01_Military_Retirement.xml", "CO"},
	{"PartialExemption", "TestCase_State_GA_01_Military_Retirement.xml", "GA"},
	{"PartialExemption", "TestCase_State_ID_01_Military_Retirement.xml", "ID"},
	{"PartialExemption", "TestCase_State_KY_01_Military_Retirement.xml", "KY"},
	{"PartialExemption", "TestCase_State_NM_01_Military_Retirement.xml", "NM"},
	{"PartialExemption", "TestCase_State_UT_01_Military_Retirement.xml", "UT"},
	{"PartialExemption", "TestCase_State_UT_02_Military_Retirement.xml", "UT"},
}

// TestMilitaryRetirementScenarios holds each listed scenario to the state tax
// its file works out from the state's 2025 form, within the $1 that
// Validate_State_Tax allows: the forms print whole dollars and the flat-rate
// tables do not round (IL $1,343.925 against the form's $1,344).
func TestMilitaryRetirementScenarios(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	expected := regexp.MustCompile(`<expected_state_tax>([0-9.]+)</expected_state_tax>`)
	for _, sc := range militaryRetirementScenarios {
		t.Run(sc.file, func(t *testing.T) {
			path := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", "State", sc.dir, sc.file)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			m := expected.FindSubmatch(src)
			if m == nil {
				t.Fatal("scenario has no <expected_state_tax>")
			}
			want, _ := strconv.ParseFloat(string(m[1]), 64)
			state := runScenarioForStateTaxCheck(t, rs, xmlDir, src)
			lines := auditTrailLines(state)
			if anyLineContains(lines, "FAIL:") {
				t.Errorf("the rules' own validation reported a FAIL line:\n%s", strings.Join(lines, "\n"))
			}
			got, found := stateRosterTax(state, sc.state)
			if !found {
				t.Fatalf("no %s entry on job.state_tax_results", sc.state)
			}
			if math.Abs(got-want) > 1.0 {
				t.Errorf("%s tax $%.3f, the scenario works out $%.2f from the 2025 form", sc.state, got, want)
			}
		})
	}
}
