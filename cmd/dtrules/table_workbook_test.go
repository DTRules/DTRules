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
	"encoding/json"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// A table records the workbook it belongs in, and that decides where an export
// writes it. With no authoring surface for it, a table naming a workbook that
// does not exist could not be repointed at one that does by any sanctioned
// route: `get` did not return it, `put` could not set it, and hand-editing the
// XML is what the contract forbids. CHIP's seven tables named
// ChipEligibility_dt.xls long after the consolidation replaced it (#1068).

func TestTableJSONCarriesTheWorkbook(t *testing.T) {
	out := tableToJSON(&authoring.Table{Name: "T", Workbook: "CHIP.xlsx"})
	if out.Workbook != "CHIP.xlsx" {
		t.Errorf("workbook = %q, want CHIP.xlsx", out.Workbook)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"workbook":"CHIP.xlsx"`) {
		t.Errorf("workbook is missing from the emitted JSON: %s", data)
	}
}

func TestPutRepointsTheWorkbook(t *testing.T) {
	tbl := &authoring.Table{Name: "T", Workbook: "ChipEligibility_dt.xls"}
	tj := &TableJSON{Name: "T", Workbook: "CHIP.xlsx"}

	if err := tj.ApplyTo(tbl); err != nil {
		t.Fatal(err)
	}
	if tbl.Workbook != "CHIP.xlsx" {
		t.Errorf("workbook = %q, want CHIP.xlsx — a table could not be moved to a workbook that exists", tbl.Workbook)
	}
}

// Omitting the field is not a request to unset it. Every caller that predates
// this field omits it, and none of them mean "put this table in no workbook".
func TestOmittingTheWorkbookLeavesItAlone(t *testing.T) {
	tbl := &authoring.Table{Name: "T", Workbook: "CHIP.xlsx"}

	if err := (&TableJSON{Name: "T"}).ApplyTo(tbl); err != nil {
		t.Fatal(err)
	}
	if tbl.Workbook != "CHIP.xlsx" {
		t.Errorf("workbook was cleared to %q by a payload that never mentioned it", tbl.Workbook)
	}
}

func TestSchemaDocumentsTheWorkbook(t *testing.T) {
	if !strings.Contains(tableSchemaJSON, `"workbook"`) {
		t.Error("the schema does not mention workbook, so no caller can discover it")
	}
}
