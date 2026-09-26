// Copyright 2026 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
)

// #1316: the tables compare job.filing_status with "MFJ", "HOH", "MFS",
// "QSS" and "Single", and nothing normalised the input, so a joint return
// spelled married_filing_jointly was taxed as Single. Normalize_Filing_Status
// runs first in Compute_Tax_Return.
func TestNormalizeFilingStatus(t *testing.T) {
	rs, _ := loadTaxReturn(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	job, err := sess.CreateEntity(dtrules.GetRName("job"))
	if err != nil {
		t.Fatal(err)
	}
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Normalize_Filing_Status"))
	if err != nil || dt == nil {
		t.Fatalf("Normalize_Filing_Status: %v", err)
	}
	normalize := func(in string) string {
		t.Helper()
		if err := job.Put(dtrules.GetRName("filing_status"), dtrules.NewRString(in)); err != nil {
			t.Fatal(err)
		}
		state.EntityPush(job)
		defer state.EntityPop()
		if err := dt.Execute(state); err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		v, _ := job.Get(dtrules.GetRName("filing_status"))
		return v.StringValue()
	}
	for want, spellings := range map[string][]string{
		// Every spelling in the scenario corpus, and the canonical codes.
		"MFJ":    {"MFJ", "mfj", "M", "married_filing_jointly", "Married Filing Jointly", "Married_Filing_Jointly", "married_joint"},
		"MFS":    {"MFS", "married_filing_separately", "Married Filing Separately"},
		"HOH":    {"HOH", "head_of_household", "Head of Household"},
		"QSS":    {"QSS", "QW", "qualifying_surviving_spouse", "Qualifying Widow(er)"},
		"Single": {"Single", "single", "SINGLE", "S"},
	} {
		for _, in := range spellings {
			if got := normalize(in); got != want {
				t.Errorf("%q normalised to %q, want %q", in, got, want)
			}
		}
	}
	if got := normalize("common_law"); got != "common_law" {
		t.Errorf("an unrecognised status was rewritten to %q; it should be left alone and reported", got)
	}
}

// End to end: a joint return spelled married_filing_jointly gets the MFJ
// standard deduction, where it used to get Single's.
func TestMisspelledJointReturnIsJoint(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	path := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", "TestCase_DE_02_MFJ_Mid_Income.xml")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "<filing_status>married_filing_jointly</filing_status>") {
		t.Fatal("fixture no longer spells its status married_filing_jointly; pick another")
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	m := mapping.NewMapping(sess)
	mf, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer mf.Close()
	if err := m.LoadMapping(mf); err != nil {
		t.Fatal(err)
	}
	if err := m.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := m.LoadData(strings.NewReader(string(src))); err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		t.Fatalf("Compute_Tax_Return: %v", err)
	}
	if err := dt.Execute(state); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != "result" {
			continue
		}
		v, _ := e.Get(dtrules.GetRName("standard_deduction"))
		if f, _ := v.DoubleValue(); f != 31500 {
			t.Errorf("standard deduction $%.0f, want the 2025 MFJ $31,500", f)
		}
		return
	}
	t.Fatal("no result entity on the stack")
}
