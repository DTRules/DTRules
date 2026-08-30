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

// Reciprocal agreements let a resident of state A who works in state B pay
// only A. B does not tax the wages and, on a filed certificate, does not
// withhold on them either (#234).
//
// Two properties matter and both are tested here. Coverage: every one of the
// 30 bilateral pairs, from both sides, is exercised by TestReciprocityMatrix
// -- writing 30 scenario XML files by hand would cover the same ground far
// more slowly and leave the reverse direction of each pair untested.
//
// The pairs are as published by each state's own revenue department for tax
// year 2025. Two absences are deliberate and are asserted as negatives in
// TestReciprocityRejectsNonAgreements:
//
//   - Minnesota-Wisconsin ended 2010-01-01. Wisconsin Pub 121 (01/26) lists
//     only IL, IN, KY and MI; Minnesota lists only MI and ND. A 2024 joint
//     study produced no reinstatement.
//   - Arizona is not a reciprocity state, though it is widely listed as one.
//     A.R.S. 43-1096 is a withholding exception backed by a credit: the wages
//     stay Arizona-source and Arizona-taxable and a Form 140NR is still
//     required. Modeling it as reciprocity would exempt income that is owed.
var reciprocalPairs = map[string][]string{
	"DC": {}, // barred from taxing any nonresident; see TestReciprocityDCExemptsEveryNonResident
	"IL": {"IA", "KY", "MI", "WI"},
	"IN": {"KY", "MI", "OH", "PA", "WI"},
	"IA": {"IL"},
	"KY": {"IL", "IN", "MI", "OH", "VA", "WV", "WI"},
	"MD": {"DC", "PA", "VA", "WV"},
	"MI": {"IL", "IN", "KY", "MN", "OH", "WI"},
	"MN": {"MI", "ND"},
	"MT": {"ND"},
	"NJ": {"PA"},
	"ND": {"MN", "MT"},
	"OH": {"IN", "KY", "MI", "PA", "WV"},
	"PA": {"IN", "MD", "NJ", "OH", "VA", "WV"},
	"VA": {"DC", "KY", "MD", "PA", "WV"},
	"WV": {"KY", "MD", "OH", "PA", "VA"},
	"WI": {"IL", "IN", "KY", "MI"},
}

// stateResult is the slice of a state_tax_result the reciprocity rules decide.
type stateResult struct {
	found        bool
	applies      bool
	partner      string
	wageIncome   float64
	sourceIncome float64
}

// runReciprocityCase files a return for a resident of residentState whose only
// work is in workState, and reports what the rules decided for workState.
// wages are sourced to workState; nonWage (rental income, when non-zero) is
// sourced there too, so the wage-only limit can be observed.
func runReciprocityCase(t *testing.T, rs *session.RuleSet, xmlDir,
	residentState, workState string, wages, nonWage float64) stateResult {
	t.Helper()

	extraIncome := ""
	if nonWage != 0 {
		extraIncome = fmt.Sprintf(`
      <income id="2">
         <id>2</id>
         <taxpayer_id>1</taxpayer_id>
         <type>rental</type>
         <source>Rental property</source>
         <gross_amount>%.0f</gross_amount>
         <state_code>%s</state_code>
      </income>`, nonWage, workState)
	}

	doc := fmt.Sprintf(`<job>
   <id>1</id>
   <tax_year>2025</tax_year>
   <filing_status>Single</filing_status>
   <state>%s</state>
   <state_periods>
      <state_period id="1">
         <id>1</id>
         <state_code>%s</state_code>
         <start_date>1/1/2025</start_date>
         <end_date>12/31/2025</end_date>
         <resident_status>resident</resident_status>
      </state_period>
      <state_period id="2">
         <id>2</id>
         <state_code>%s</state_code>
         <start_date>1/1/2025</start_date>
         <end_date>12/31/2025</end_date>
         <resident_status>nonresident</resident_status>
      </state_period>
   </state_periods>
   <taxpayers>
      <taxpayer id="1">
         <id>1</id>
         <name>Test Taxpayer</name>
         <is_primary>true</is_primary>
         <w2_wages>%.0f</w2_wages>
      </taxpayer>
   </taxpayers>
   <incomes>
      <income id="1">
         <id>1</id>
         <taxpayer_id>1</taxpayer_id>
         <type>w2_wages</type>
         <source>Employer</source>
         <gross_amount>%.0f</gross_amount>
         <state_code>%s</state_code>
      </income>%s
   </incomes>
</job>`, residentState, residentState, workState, wages, wages, workState, extraIncome)

	path := filepath.Join(t.TempDir(), "case.xml")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write scenario: %v", err)
	}

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
	if err := m.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open scenario: %v", err)
	}
	defer f.Close()
	if err := m.LoadData(f); err != nil {
		t.Fatalf("load data: %v", err)
	}

	state := sess.GetState()
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		t.Fatalf("Compute_Tax_Return: %v", err)
	}
	if err := dt.Execute(state); err != nil {
		t.Fatalf("execute %s/%s: %v", residentState, workState, err)
	}

	return readStateResult(t, state, workState)
}

// readStateResult pulls the state_tax_result for one state code off the job.
func readStateResult(t *testing.T, state dtrules.State, stateCode string) stateResult {
	t.Helper()
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
		for _, item := range arr {
			ent, err := item.REntityValue()
			if err != nil || ent == nil {
				continue
			}
			code, _ := ent.Get(dtrules.GetRName("state_code"))
			if code == nil || code.StringValue() != stateCode {
				continue
			}
			out := stateResult{found: true}
			if o, _ := ent.Get(dtrules.GetRName("reciprocity_applies")); o != nil {
				out.applies, _ = o.BooleanValue()
			}
			if o, _ := ent.Get(dtrules.GetRName("reciprocity_partner")); o != nil {
				out.partner = o.StringValue()
			}
			if o, _ := ent.Get(dtrules.GetRName("state_wage_income")); o != nil {
				out.wageIncome, _ = o.DoubleValue()
			}
			if o, _ := ent.Get(dtrules.GetRName("state_source_income")); o != nil {
				out.sourceIncome, _ = o.DoubleValue()
			}
			return out
		}
	}
	return stateResult{}
}

func loadTaxReturn(t *testing.T) (*session.RuleSet, string) {
	t.Helper()
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")
	if _, err := os.Stat(xmlDir); err != nil {
		t.Skip("TaxReturn sample not present")
	}
	rs := session.NewRuleSet("TaxReturn")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("load rules: %v", err)
	}
	return rs, xmlDir
}

// Every published pair, from both sides: a resident of each partner working in
// the work state has their wages exempted there.
func TestReciprocityMatrix(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	for workState, partners := range reciprocalPairs {
		for _, residentState := range partners {
			name := residentState + "_resident_working_in_" + workState
			t.Run(name, func(t *testing.T) {
				got := runReciprocityCase(t, rs, xmlDir, residentState, workState, 80000, 0)
				if !got.found {
					t.Fatalf("no state_tax_result for the work state %s", workState)
				}
				if !got.applies {
					t.Errorf("%s should exempt wages of %s residents, but no agreement was found",
						workState, residentState)
				}
				if got.partner != residentState {
					t.Errorf("reciprocity_partner = %q, want %q", got.partner, residentState)
				}
				if got.wageIncome != 0 {
					t.Errorf("%s wages still taxable by %s: $%.2f, want 0",
						residentState, workState, got.wageIncome)
				}
				if got.sourceIncome != 0 {
					t.Errorf("wages were the only %s income, so nothing should remain sourced there; got $%.2f",
						workState, got.sourceIncome)
				}
			})
		}
	}
}

// DC cannot tax any nonresident -- the Home Rule Act bars it, DC Code
// 1-206.02(a)(5) -- so it has no partner list. The relief must reach a
// resident of a state with no DC pact at all.
func TestReciprocityDCExemptsEveryNonResident(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	for _, residentState := range []string{"CA", "NY", "TX", "MD", "VA"} {
		t.Run(residentState, func(t *testing.T) {
			got := runReciprocityCase(t, rs, xmlDir, residentState, "DC", 80000, 0)
			if !got.applies {
				t.Errorf("DC may not tax a %s resident's wages", residentState)
			}
			if got.wageIncome != 0 {
				t.Errorf("DC left $%.2f of wages taxable", got.wageIncome)
			}
		})
	}
}

// A DC resident working in DC is not a nonresident, so the DC row -- the only
// one with no partner list -- must not exempt their own wages.
func TestReciprocityDoesNotExemptResidentsOwnState(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	got := runReciprocityCase(t, rs, xmlDir, "DC", "DC", 80000, 0)
	if got.applies {
		t.Error("a DC resident's own DC wages were exempted; the non-resident guard on the DC row is not holding")
	}
}

// Pairs that are widely believed to exist and do not. Getting these wrong
// exempts income that is actually owed.
func TestReciprocityRejectsNonAgreements(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	cases := []struct {
		resident, work, why string
	}{
		{"WI", "MN", "Minnesota ended the Wisconsin agreement on 2010-01-01"},
		{"MN", "WI", "the Wisconsin-Minnesota agreement is terminated in both directions"},
		{"CA", "AZ", "A.R.S. 43-1096 is a withholding exception backed by a credit, not reciprocity; a 140NR is still required"},
		{"IN", "AZ", "Arizona is not a reciprocity state for Indiana residents either"},
		{"NY", "CT", "New York and Connecticut have no agreement"},
		{"CA", "KY", "Kentucky has seven agreements and California is not among them"},
		{"IL", "IN", "Illinois and Indiana are not reciprocal in either direction"},
		{"IN", "IL", "Illinois and Indiana are not reciprocal in either direction"},
		{"MD", "MI", "Maryland's pre-1992 statutory reciprocity with Michigan ended with Chapter 1, Acts of 1992"},
	}

	for _, c := range cases {
		t.Run(c.resident+"_in_"+c.work, func(t *testing.T) {
			got := runReciprocityCase(t, rs, xmlDir, c.resident, c.work, 80000, 0)
			if got.applies {
				t.Errorf("%s exempted a %s resident's wages, but %s", c.work, c.resident, c.why)
			}
			if got.found && got.wageIncome != 80000 {
				t.Errorf("%s should still be taxing $80000 of wages, has $%.2f", c.work, got.wageIncome)
			}
		})
	}
}

// Reciprocity reaches employee compensation and nothing else. Wisconsin Pub
// 121: "It does not apply to other types of income, such as gains on the sale
// of property, rental income, lottery winnings, self-employment income ..."
// So a taxpayer with both wages and rental income in the work state keeps the
// rental income taxable there.
func TestReciprocityExemptsWagesOnly(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)

	got := runReciprocityCase(t, rs, xmlDir, "OH", "KY", 95000, 18000)
	if !got.applies {
		t.Fatal("KY should exempt the wages of an OH resident")
	}
	if got.wageIncome != 0 {
		t.Errorf("wages still taxable by KY: $%.2f, want 0", got.wageIncome)
	}
	if got.sourceIncome != 18000 {
		t.Errorf("KY-source income after the exemption = $%.2f, want the $18000 of rental income "+
			"-- reciprocity covers compensation only", got.sourceIncome)
	}
}
