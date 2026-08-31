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

package main

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// Every mapping entry in every sample project, replayed through the check that
// now guards `map patch`. Two things are being held at once.
//
// First, no false positives: the check has to accept the 791 entries that are
// correct, or it would block legitimate authoring rather than bad authoring.
//
// Second, the entries it does reject are real and are counted. A mapping entry
// naming a field the EDD does not declare is silent at load -- the value is
// dropped and the run continues -- so these are live defects, not style. The
// ceiling can only come down:
//
//	CorporateTax (both map files)  job.id, job.tax_year, job.expected_taxable_income,
//	                               job.expected_federal_tax, job.expected_refund_or_owed,
//	                               expense.id, result.audit_trail, result.total_revenue
//	                               -- result.audit_trail moved to job in campaign #948
//	                               and the mapping still points at where it was
//	TaxReturn                      taxpayer.estimated_tax_payments (the field is
//	                               estimated_payments), unemployment_compensation,
//	                               gambling_winnings, alimony_received,
//	                               alimony_divorce_date, state_tax_refund,
//	                               long_term_capital_gains, qualified_dividends,
//	                               dependent.qualified_education_expenses, and
//	                               unreported_tips, whose array is
//	                               unreported_tips_records -- the entity name is
//	                               already plural, so the loader's name+"s"
//	                               fallback looks for "unreported_tipss"
//	SyntaxTests                    address, whose fallback looks for "addresss"
//
// Tracked in #1175's sibling; fixing them changes what loads, so each needs its
// scenarios reconciled rather than a blind rename.
const knownUndeclaredMappingEntries = 23

func TestSampleMappingsResolveAgainstTheirEDD(t *testing.T) {
	maps, _ := filepath.Glob("../../sampleprojects/*/xml/*_map.xml")
	if len(maps) == 0 {
		t.Skip("sample projects not present")
	}

	var rejected []string
	checked := 0
	for _, mp := range maps {
		m, err := excel.LoadMapXMLFromFile(mp)
		if err != nil {
			continue
		}
		edd := loadEDDModel(filepath.Dir(mp))
		if edd == nil {
			continue // no EDD declared; the check is a no-op by design
		}
		proj := filepath.Base(filepath.Dir(filepath.Dir(mp)))

		for _, e := range m.Entries {
			if e.IsSection {
				continue
			}
			checked++
			op := mapPatchOp{Op: "add-attribute", Attribute: &mapAttributeJSON{
				Tag: e.Tag, RAttribute: e.RAttribute, Enclosure: e.Enclosure, Type: e.Type}}
			if err := validateMapOp(m, op, edd); err != nil {
				rejected = append(rejected, proj+" "+e.Enclosure+"."+e.Tag)
			}
		}
		for _, c := range m.CreateEntities {
			checked++
			number := ""
			for _, d := range m.EntityDecls {
				if d.Name == c.Entity {
					number = d.Number
				}
			}
			op := mapPatchOp{Op: "add-entity", Entity: c.Entity, Number: number,
				Tag: c.Tag, ID: c.ID, List: c.List}
			if err := validateMapOp(m, op, edd); err != nil {
				rejected = append(rejected, proj+" createentity "+c.Entity)
			}
		}
	}

	sort.Strings(rejected)
	t.Logf("checked %d mapping entries, %d resolve against nothing", checked, len(rejected))

	if len(rejected) > knownUndeclaredMappingEntries {
		for _, r := range rejected {
			t.Logf("  %s", r)
		}
		t.Errorf("%d mapping entries name something the EDD does not declare, ceiling is %d — "+
			"a new one is a value that will be dropped at load without erroring",
			len(rejected), knownUndeclaredMappingEntries)
	}
	if len(rejected) < knownUndeclaredMappingEntries {
		t.Logf("only %d left of %d — lower knownUndeclaredMappingEntries to hold the gain",
			len(rejected), knownUndeclaredMappingEntries)
	}
}
