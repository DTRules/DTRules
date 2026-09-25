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
	"strconv"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// New Mexico (#1200). nm_tax had conditions and no actions. It now follows
// the 2025 PIT-1, and NM_Tax_Table computes the 2025 Tax Rate Table.

// nmStatusLimits are what nm_tax sets for each column of the table: the four
// bracket tops through $100,000, then the page T-7 schedule (base over
// $100,000, the 5.9% threshold, and the base over it).
var nmStatusLimits = map[string][7]float64{
	"single": {5500, 16500, 33500, 66500, 4356, 210000, 9746},
	"joint":  {8000, 25000, 50000, 100000, 4087, 315000, 14622},
	"mfs":    {4000, 12500, 25000, 50000, 4492, 157500, 7310},
}

// TestNMTaxTableMatchesThePublishedTable runs NM_Tax_Table at both ends of
// every band of the published 2025 table, for all four columns, and on the
// page T-7 schedule above it.
func TestNMTaxTableMatchesThePublishedTable(t *testing.T) {
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
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("NM_Tax_Table"))
	if err != nil || dt == nil {
		t.Fatalf("NM_Tax_Table: %v", err)
	}
	put := func(field string, v float64) {
		t.Helper()
		if err := res.Put(dtrules.GetRName(field), dtrules.GetRDoubleValue(v)); err != nil {
			t.Fatal(err)
		}
	}
	tax := func(status string, income int) int {
		t.Helper()
		l := nmStatusLimits[status]
		for i, f := range []string{"nm_limit_1", "nm_limit_2", "nm_limit_3", "nm_limit_4", "nm_high_base", "nm_top_threshold", "nm_top_base"} {
			put(f, l[i])
		}
		put("nm_taxable_income", float64(income))
		state.EntityPush(res)
		defer state.EntityPop()
		if err := dt.Execute(state); err != nil {
			t.Fatalf("%s $%d: %v", status, income, err)
		}
		v, _ := res.Get(dtrules.GetRName("nm_table_tax"))
		f, _ := v.DoubleValue()
		return int(f)
	}

	f, err := os.Open(filepath.Join("testdata", "nm_2025_tax_rate_table.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Head of Household shares the joint column's brackets.
	columns := []string{"single", "joint", "mfs", "joint"}
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
		rows++
		for c, status := range columns {
			want, _ := strconv.Atoi(p[2+c])
			for _, income := range []int{lo + 1, hi} {
				if got := tax(status, income); got != want {
					t.Errorf("column %d (%s), taxable income $%d (more than %d, not over %d): tax $%d, table says $%d",
						c+1, status, income, lo, hi, got, want)
				}
			}
		}
	}
	if rows < 1000 {
		t.Fatalf("read %d table rows; the test data is not the whole table", rows)
	}
	for _, c := range []struct {
		status      string
		income, tax int
	}{
		{"single", 150000, 4356 + 2450}, // 4.9% of $50,000 over $100,000
		{"single", 250000, 9746 + 2360}, // 5.9% of $40,000 over $210,000
		{"joint", 200000, 4087 + 4900},  // 4.9% of $100,000 over $100,000
		{"joint", 400000, 14622 + 5015}, // 5.9% of $85,000 over $315,000
		{"mfs", 200000, 7310 + 2508},    // 5.9% of $42,500 = $2,507.50, rounded
	} {
		if got := tax(c.status, c.income); got != c.tax {
			t.Errorf("%s, taxable income $%d: tax $%d, schedule gives $%d", c.status, c.income, got, c.tax)
		}
	}
}
