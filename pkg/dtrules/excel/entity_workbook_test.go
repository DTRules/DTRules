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

package excel

import "testing"

// An entity declared in X_edd.xml belongs in X.xlsx unless it says otherwise.
// That is the pairing the sync layer already uses to match an EDD file to its
// workbook, and without it an entity created through the authoring API has no
// xls_file at all -- so nothing claims its workbook, the refresh writes no EDD
// sheet, and the definitions live only in XML (#1094).

func TestEntitiesInheritTheirEDDFilesWorkbook(t *testing.T) {
	edd := &EDDXML{Entities: []*EDDXMLEntity{
		{Name: "nv_result"},
		{Name: "nv_apportionment"},
	}}

	backfillEntityWorkbook(edd, "/p/xml/states/NV_corp_edd.xml")

	for _, e := range edd.Entities {
		if e.XlsFile != "NV_corp.xlsx" {
			t.Errorf("%s got xls_file %q, want NV_corp.xlsx", e.Name, e.XlsFile)
		}
	}
}

// An entity that already names a workbook keeps it, so a project that
// deliberately groups entities elsewhere is untouched.
func TestAnExplicitWorkbookIsNotOverwritten(t *testing.T) {
	edd := &EDDXML{Entities: []*EDDXMLEntity{
		{Name: "shared", XlsFile: "CorporateTax_core.xlsx"},
		{Name: "nv_result"},
	}}

	backfillEntityWorkbook(edd, "/p/xml/states/NV_corp_edd.xml")

	if edd.Entities[0].XlsFile != "CorporateTax_core.xlsx" {
		t.Errorf("an explicit workbook was overwritten: %q", edd.Entities[0].XlsFile)
	}
	if edd.Entities[1].XlsFile != "NV_corp.xlsx" {
		t.Errorf("the unset one should be filled: %q", edd.Entities[1].XlsFile)
	}
}

// The _edd suffix is dropped, not just the extension -- otherwise the entity
// would claim NV_corp_edd.xlsx, which is not a workbook any project has.
func TestTheEDDSuffixIsNotPartOfTheWorkbookName(t *testing.T) {
	edd := &EDDXML{Entities: []*EDDXMLEntity{{Name: "job"}}}

	backfillEntityWorkbook(edd, "CHIP_edd.xml")

	if edd.Entities[0].XlsFile != "CHIP.xlsx" {
		t.Errorf("got %q, want CHIP.xlsx", edd.Entities[0].XlsFile)
	}
}
