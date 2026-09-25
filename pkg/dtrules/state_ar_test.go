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
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
)

// Arkansas (#1200). AR_Tax computed nothing: its actions were wired to no
// column. It now follows the 2025 AR1000F, and AR_Tax_Table computes the
// Regular Income Tax Table rather than looking it up.

// TestARTaxTableMatchesThePublishedTable runs AR_Tax_Table at both ends of
// every $100 band of the published 2025 Regular Income Tax Table. DFA: "If you
// use a formula to calculate Arkansas income tax, the results must match the
// table exactly."
func TestARTaxTableMatchesThePublishedTable(t *testing.T) {
	rs, _ := loadTaxReturn(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	res, err := sess.CreateEntity(dtrules.GetRName("result"))
	if err != nil {
		t.Fatal(err)
	}
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("AR_Tax_Table"))
	if err != nil || dt == nil {
		t.Fatalf("AR_Tax_Table: %v", err)
	}
	tax := func(nti int) int {
		t.Helper()
		if err := res.Put(dtrules.GetRName("ar_taxable_income"), dtrules.GetRDoubleValue(float64(nti))); err != nil {
			t.Fatal(err)
		}
		state.EntityPush(res)
		defer state.EntityPop()
		if err := dt.Execute(state); err != nil {
			t.Fatalf("NTI %d: %v", nti, err)
		}
		v, _ := res.Get(dtrules.GetRName("ar_table_tax"))
		f, _ := v.DoubleValue()
		return int(f)
	}

	f, err := os.Open(filepath.Join("testdata", "ar_2025_regular_tax_table.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Split(line, "\t")
		lo, _ := strconv.Atoi(p[0])
		hi, _ := strconv.Atoi(p[1])
		want, _ := strconv.Atoi(p[2])
		rows++
		for _, nti := range []int{lo, hi - 1} {
			if got := tax(nti); got != want {
				t.Errorf("net taxable income $%d (table row %d-%d): tax $%d, table says $%d", nti, lo, hi, got, want)
			}
		}
	}
	if rows < 900 {
		t.Fatalf("read %d table rows; the test data is not the whole table", rows)
	}
	// Below the table, and the formula above it: $3,809 + 3.9% of the excess
	// over $100,000.
	for _, c := range [][2]int{{0, 0}, {5099, 0}, {100001, 3809}, {100500, 3829}, {150000, 5759}} {
		if got := tax(c[0]); got != c[1] {
			t.Errorf("net taxable income $%d: tax $%d, want $%d", c[0], got, c[1])
		}
	}
}

// TestARScenarios runs each testfiles/TestScenarios/AR scenario and compares
// the Arkansas roster entry's tax -- what Compute_Roster_State_Tax harvested
// from AR_Tax -- with the figure the scenario works out by hand.
// Validate_Summary does not check expected_state_tax, so this does.
func TestARScenarios(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	dir := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", "AR")
	files, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
	if len(files) < 3 {
		t.Fatalf("want at least 3 AR scenarios, found %d", len(files))
	}
	expected := regexp.MustCompile(`<expected_state_tax>([0-9.]+)</expected_state_tax>`)
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
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

			got, found := arRosterTax(state)
			if !found {
				t.Fatal("no Arkansas entry on job.state_tax_results")
			}
			if got != want {
				t.Errorf("Arkansas tax $%.2f, scenario works out $%.2f", got, want)
			}
		})
	}
}

// arRosterTax returns the state_tax_before_credits of the AR roster entry.
func arRosterTax(state dtrules.State) (float64, bool) {
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
			if code == nil || code.StringValue() != "AR" {
				continue
			}
			tax, _ := entry.Get(dtrules.GetRName("state_tax_before_credits"))
			f, _ := tax.DoubleValue()
			return f, true
		}
	}
	return 0, false
}
