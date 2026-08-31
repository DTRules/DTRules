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
	"sort"
	"strings"
	"testing"
)

// A deduction whose category no rule reads is not an error anywhere: it loads,
// it sits on the entity, and every table that would have used it passes it by.
// The return comes out wrong and nothing says so. That is how SALT summed
// property taxes alone while `state_income_tax` sat beside it unread (#1161),
// and it is the shape #1175 was filed about.
//
// So the vocabulary is closed and held here. Adding a spelling means adding it
// to this set *and* to the rule that reads it — which is the point, because
// the second half is the one that gets forgotten.
var canonicalDeductionCategories = map[string]string{
	"medical":               "Schedule A medical and dental, subject to the AGI floor",
	"state_tax":             "state and local income tax — SALT, with property_tax",
	"property_tax":          "real-estate tax — SALT, with state_tax",
	"mortgage_interest":     "Schedule A interest (the acquisition-debt limit applies)",
	"charity":               "Schedule A cash contributions",
	"charity_noncash":       "Schedule A non-cash contributions (Form 8283)",
	"student_loan_interest": "above-the-line adjustment, not Schedule A",
	"alimony_paid":          "above-the-line adjustment for pre-2019 divorces",
	"educator_expense":      "above-the-line adjustment",
	"other":                 "carried for completeness; read by no rule",
}

var deductionBlock = regexp.MustCompile(`(?s)<deduction\b.*?</deduction>`)
var categoryTag = regexp.MustCompile(`<category>([a-z_]+)</category>`)

func TestDeductionCategoriesAreCanonical(t *testing.T) {
	cwd, _ := os.Getwd()
	root := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn",
		"testfiles", "TestScenarios")
	if _, err := os.Stat(root); err != nil {
		t.Skip("TaxReturn scenarios not present")
	}

	offenders := map[string][]string{}
	seen := map[string]bool{}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".xml") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		// Only <deduction> elements: <category> also appears on expense,
		// business_expense, medical_expense, itemized_deduction and
		// adjustment, none of which the EDD declares or the mapping maps, so
		// they are dropped at load and are a separate problem.
		for _, blk := range deductionBlock.FindAll(data, -1) {
			m := categoryTag.FindSubmatch(blk)
			if m == nil {
				continue
			}
			cat := string(m[1])
			seen[cat] = true
			if _, ok := canonicalDeductionCategories[cat]; !ok {
				rel, _ := filepath.Rel(root, p)
				offenders[cat] = append(offenders[cat], rel)
			}
		}
		return nil
	})

	if len(offenders) > 0 {
		cats := make([]string, 0, len(offenders))
		for c := range offenders {
			cats = append(cats, c)
		}
		sort.Strings(cats)
		for _, c := range cats {
			t.Errorf("deduction category %q is not in the canonical set, so no rule reads it "+
				"and the deduction is silently ignored — %d scenario(s), e.g. %s",
				c, len(offenders[c]), offenders[c][0])
		}
	}

	if len(seen) == 0 {
		t.Fatal("no deduction categories found at all; the fixture or the regex is wrong")
	}
	t.Logf("%d distinct deduction categories in the corpus, all canonical", len(seen))
}

// The set is only worth holding if the rules actually read it. A canonical
// value nothing reads is a deduction that loads and is then ignored, which is
// the failure this whole vocabulary exists to prevent -- so the ones that are
// meant to be live are checked against the rules that consume them.
func TestCanonicalCategoriesTheRulesClaimToRead(t *testing.T) {
	cwd, _ := os.Getwd()
	dt := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn", "xml", "TaxReturn_dt.xml")
	data, err := os.ReadFile(dt)
	if err != nil {
		t.Skip("TaxReturn rules not present")
	}
	rules := string(data)

	// Categories the rules are expected to consume today. The above-the-line
	// adjustments and "other" are deliberately absent: they are declared so
	// the data can say what it means, and are not Schedule A items.
	for _, cat := range []string{"state_tax", "property_tax", "charity"} {
		if !strings.Contains(rules, `deduction.category is equal to "`+cat+`"`) {
			t.Errorf("no rule reads deduction category %q, so every deduction carrying it "+
				"is loaded and then ignored", cat)
		}
	}

	// And nothing should still be matching a spelling the corpus no longer uses.
	for _, gone := range []string{"state_income_tax", "state_local_taxes", "charitable_cash", "local_tax"} {
		if strings.Contains(rules, `deduction.category is equal to "`+gone+`"`) {
			t.Errorf("a rule still matches %q, which is not in the canonical set — "+
				"either the set is wrong or the rule is dead", gone)
		}
	}
}
