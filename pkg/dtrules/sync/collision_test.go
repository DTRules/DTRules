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

package sync

import (
	"strings"
	"testing"
)

// Two workbooks writing the same XML file means one of them is discarded, and
// which one depended on Go's map iteration order. `dtrules verify` returned 1,
// 1, 2, 0 and 2 findings on five passes over an unchanged TaxReturn, because
// TaxReturn.xlsx carries an EDD sheet and TaxReturn_edd.xlsx sat beside it and
// both produced xml/TaxReturn_edd.xml (#1089).

func TestCollidingWorkbooksAreRefused(t *testing.T) {
	err := assertNoOutputCollision([]CombinedWorkbook{
		{ExcelPath: "/p/excel/TaxReturn.xlsx",
			DTXMLPath:  "/p/xml/TaxReturn_dt.xml",
			EDDXMLPath: "/p/xml/TaxReturn_edd.xml"},
		{ExcelPath: "/p/excel/TaxReturn_edd.xlsx",
			EDDXMLPath: "/p/xml/TaxReturn_edd.xml"},
	})
	if err == nil {
		t.Fatal("two workbooks producing one XML file must be refused; one of " +
			"them is silently discarded otherwise")
	}
	// Both claimants have to be named -- knowing there is a collision without
	// knowing which files collided is not actionable.
	for _, want := range []string{"TaxReturn_edd.xml", "TaxReturn.xlsx", "TaxReturn_edd.xlsx"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error must name %q, got: %v", want, err)
		}
	}
}

func TestDistinctWorkbooksAreAccepted(t *testing.T) {
	err := assertNoOutputCollision([]CombinedWorkbook{
		{ExcelPath: "/p/excel/A.xlsx", DTXMLPath: "/p/xml/A_dt.xml", EDDXMLPath: "/p/xml/A_edd.xml"},
		{ExcelPath: "/p/excel/B.xlsx", DTXMLPath: "/p/xml/B_dt.xml", EDDXMLPath: "/p/xml/B_edd.xml"},
		{ExcelPath: "/p/excel/C_edd.xlsx", EDDXMLPath: "/p/xml/C_edd.xml"},
	})
	if err != nil {
		t.Fatalf("workbooks with their own outputs must be accepted: %v", err)
	}
}

// An unused half is empty, not a path, and two workbooks both leaving their DT
// side empty is not a collision.
func TestEmptyOutputPathsDoNotCollide(t *testing.T) {
	err := assertNoOutputCollision([]CombinedWorkbook{
		{ExcelPath: "/p/excel/A_edd.xlsx", EDDXMLPath: "/p/xml/A_edd.xml"},
		{ExcelPath: "/p/excel/B_edd.xlsx", EDDXMLPath: "/p/xml/B_edd.xml"},
	})
	if err != nil {
		t.Fatalf("two EDD-only workbooks with different outputs are fine: %v", err)
	}
}
