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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Residency had two vocabularies for one idea: state_period.resident_status
// carried full_year|part_year|nonresident and was what scenarios supplied,
// while state_tax_result.residency_status carried
// resident|part_year_resident|nonresident and was read by
// Calculate_Part_Year_Allocation and written by nothing at all. Declared
// access="r" with default "resident", it could never be anything else, so the
// part-year branch was unreachable and every state took the full-year one --
// which also overwrote state_agi with the federal AGI for *non-resident*
// states, taxing them on every dollar earned anywhere (#1177).
//
// One vocabulary now, carried from the period to the result.

// residencyOf runs a two-period return and reports what the roster recorded
// for the work state: its resident_status and the income the state is left
// taxing.
func residencyOf(t *testing.T, rs *session.RuleSet, xmlDir,
	residentState, workState, workStatus string, wages float64) (status string, stateAGI float64) {
	return residencyOfState(t, rs, xmlDir, residentState, workState, workStatus, wages, workState)
}

// residencyOfState is residencyOf with the roster entry to read named
// explicitly -- the resident state's entry is the interesting one when the
// income is sourced somewhere else.
func residencyOfState(t *testing.T, rs *session.RuleSet, xmlDir,
	residentState, workState, workStatus string, wages float64, readState string) (status string, stateAGI float64) {
	t.Helper()

	doc := fmt.Sprintf(`<job>
   <id>1</id>
   <tax_year>2025</tax_year>
   <filing_status>Single</filing_status>
   <state>%s</state>
   <state_periods>
      <state_period id="1">
         <id>1</id><state_code>%s</state_code>
         <start_date>1/1/2025</start_date><end_date>12/31/2025</end_date>
         <resident_status>full_year</resident_status>
      </state_period>
      <state_period id="2">
         <id>2</id><state_code>%s</state_code>
         <start_date>1/1/2025</start_date><end_date>12/31/2025</end_date>
         <resident_status>%s</resident_status>
      </state_period>
   </state_periods>
   <taxpayers><taxpayer id="1"><id>1</id><name>T</name><is_primary>true</is_primary>
      <w2_wages>%.0f</w2_wages></taxpayer></taxpayers>
   <incomes><income id="1"><id>1</id><taxpayer_id>1</taxpayer_id><type>w2_wages</type>
      <source>E</source><gross_amount>%.0f</gross_amount><state_code>%s</state_code></income></incomes>
</job>`, residentState, residentState, workState, workStatus, wages, wages, workState)

	path := filepath.Join(t.TempDir(), "case.xml")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	m := mapping.NewMapping(sess)
	mf, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	defer mf.Close()
	if err := m.LoadMapping(mf); err != nil {
		t.Fatalf("mapping: %v", err)
	}
	f, _ := os.Open(path)
	defer f.Close()
	if err := m.LoadDataAndPushSingletons(f); err != nil {
		t.Fatalf("load: %v", err)
	}
	st := sess.GetState()
	dt, _ := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err := dt.Execute(st); err != nil {
		t.Fatalf("execute: %v", err)
	}

	for i := 0; i < st.EntityDepth(); i++ {
		e, _ := st.EntityFetch(i)
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
		for _, item := range arr {
			ent, err := item.REntityValue()
			if err != nil || ent == nil {
				continue
			}
			code, _ := ent.Get(dtrules.GetRName("state_code"))
			if code == nil || code.StringValue() != readState {
				continue
			}
			if o, _ := ent.Get(dtrules.GetRName("resident_status")); o != nil {
				status = o.StringValue()
			}
			if o, _ := ent.Get(dtrules.GetRName("state_agi")); o != nil {
				stateAGI, _ = o.DoubleValue()
			}
			return status, stateAGI
		}
	}
	return "", 0
}

// The status a scenario supplies on the period has to reach the result, or the
// part-year branch is unreachable no matter what the data says.
func TestResidencyStatusReachesTheRoster(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	for _, want := range []string{"part_year", "nonresident", "full_year"} {
		t.Run(want, func(t *testing.T) {
			got, _ := residencyOf(t, rs, xmlDir, "OH", "PA", want, 80000)
			if got != want {
				t.Errorf("state_tax_result.resident_status = %q, want %q — the period's status "+
					"never reached the roster, so the part-year branch cannot be selected", got, want)
			}
		})
	}
}

// The full-year branch used to run for every state, overwriting state_agi with
// the federal AGI. On a non-resident state that taxes it on every dollar the
// taxpayer earned anywhere, rather than the income sourced there.
func TestNonResidentKeepsItsSourcedIncome(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	// California, deliberately: Ohio has reciprocal agreements with IN, KY,
	// MI, PA and WV, and in those the work state's wages are exempted to zero
	// on purpose (#234). CA has no agreement with Ohio, so the full 80,000
	// stays sourced there and the figure is the one this test is about.
	_, caAGI := residencyOf(t, rs, xmlDir, "OH", "CA", "nonresident", 80000)
	if caAGI != 80000 {
		t.Errorf("CA state_agi = %.2f, want the 80000 sourced there", caAGI)
	}

	// A non-resident state with no income sourced to it must be left at zero,
	// not handed the federal AGI.
	_, coAGI := residencyOf(t, rs, xmlDir, "OH", "CO", "nonresident", 0)
	if coAGI != 0 {
		t.Errorf("a non-resident state with nothing sourced to it has state_agi = %.2f, want 0 — "+
			"the full-year branch is still overwriting it with the federal AGI", coAGI)
	}
}

// The order the dispatcher performs its steps in is load-bearing, and it was
// wrong. Apply_Part_Year_Allocations ran before Calculate_State_Source_Income
// built the roster, so it iterated an empty list and the residency branches
// never executed on a real result -- the same "reached by nothing" shape as
// the field they branch on, one level up (#1177).
//
// The visible symptom: a full-year resident state came out with state_agi 0
// whenever the income was sourced elsewhere, because the branch that gives a
// resident state the federal AGI never ran.
func TestResidentStateGetsTheFederalAGI(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	// An Ohio resident whose wages are all sourced to California. Ohio taxes
	// every dollar wherever earned; California taxes what is sourced there.
	// CA deliberately: Ohio's reciprocal partners are IN, KY, MI, PA and WV,
	// and in those the wages would be exempted to zero on purpose (#234).
	status, ohAGI := residencyOfState(t, rs, xmlDir, "OH", "CA", "nonresident", 90000, "OH")
	if status != "full_year" {
		t.Fatalf("resident status = %q, want full_year", status)
	}
	if ohAGI != 90000 {
		t.Errorf("the resident state's state_agi = %.2f, want the federal AGI 90000 — "+
			"the full-year branch did not run, which happens when the roster is built "+
			"after the allocation pass rather than before it", ohAGI)
	}
}
