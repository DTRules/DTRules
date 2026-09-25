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

// Oklahoma (#1200). ok_tax had conditions and no actions. It now follows
// the 2025 Form 511, and OK_Tax_Table computes the 2025 Income Tax Table.

// okColumnLimits are what ok_tax sets for each table column: the five
// bracket tops, then the base of the $100,000-or-more computation.
var okColumnLimits = map[string][6]float64{
	"single": {1000, 2500, 3750, 4900, 7200, 4562},
	"joint":  {2000, 5000, 7500, 9800, 14400, 4373},
}

// TestOKTaxTableMatchesThePublishedTable runs OK_Tax_Table at both ends of
// every $50 band of the published 2025 table, in both columns, and on the
// $100,000-or-more computation.
func TestOKTaxTableMatchesThePublishedTable(t *testing.T) {
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
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("OK_Tax_Table"))
	if err != nil || dt == nil {
		t.Fatalf("OK_Tax_Table: %v", err)
	}
	tax := func(column string, income int) int {
		t.Helper()
		l := okColumnLimits[column]
		for i, f := range []string{"ok_limit_1", "ok_limit_2", "ok_limit_3", "ok_limit_4", "ok_limit_5", "ok_high_base"} {
			if err := res.Put(dtrules.GetRName(f), dtrules.GetRDoubleValue(l[i])); err != nil {
				t.Fatal(err)
			}
		}
		if err := res.Put(dtrules.GetRName("ok_taxable_income"), dtrules.GetRDoubleValue(float64(income))); err != nil {
			t.Fatal(err)
		}
		state.EntityPush(res)
		defer state.EntityPop()
		if err := dt.Execute(state); err != nil {
			t.Fatalf("%s $%d: %v", column, income, err)
		}
		v, _ := res.Get(dtrules.GetRName("ok_table_tax"))
		f, _ := v.DoubleValue()
		return int(f)
	}

	f, err := os.Open(filepath.Join("testdata", "ok_2025_income_tax_table.tsv"))
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
		rows++
		for c, column := range []string{"single", "joint"} {
			want, _ := strconv.Atoi(p[2+c])
			for _, income := range []int{lo, hi - 1} {
				if got := tax(column, income); got != want {
					t.Errorf("%s column, taxable income $%d (at least %d, less than %d): tax $%d, table says $%d",
						column, income, lo, hi, got, want)
				}
			}
		}
	}
	if rows != 2000 {
		t.Fatalf("read %d table rows, want all 2,000", rows)
	}
	for _, c := range []struct {
		column      string
		income, tax int
	}{
		{"single", 100000, 4562},
		{"single", 150000, 4562 + 2375}, // 4.75% of $50,000
		{"joint", 100000, 4373},
		{"joint", 108650, 4784}, // $4,373 + $410.875
	} {
		if got := tax(c.column, c.income); got != c.tax {
			t.Errorf("%s, taxable income $%d: tax $%d, the page 38 computation gives $%d", c.column, c.income, got, c.tax)
		}
	}
}
