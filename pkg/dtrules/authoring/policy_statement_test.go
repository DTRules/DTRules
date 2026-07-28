// Copyright 2024 Paul Snow
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

package authoring

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

func policyTestTable() *Table {
	return newTable(&excel.DecisionTableXML{
		TableName: "Compute_Tax",
		PolicyStatements: []excel.PolicyStatementXML{
			{Column: "2", Description: "second", Postfix: `"stale text nobody rendered"`},
			{Column: "1", Description: "first", Postfix: `"first"`},
		},
	}, map[string]string{})
}

// TestPolicyStatementsSurfaceSorted checks the typed view exposes what the XML
// carries, ordered by column rather than by file order.
func TestPolicyStatementsSurfaceSorted(t *testing.T) {
	tbl := policyTestTable()
	if len(tbl.PolicyStatements) != 2 {
		t.Fatalf("got %d policy statements, want 2", len(tbl.PolicyStatements))
	}
	if tbl.PolicyStatements[0].Column != 1 || tbl.PolicyStatements[1].Column != 2 {
		t.Errorf("statements not sorted by column: %+v", tbl.PolicyStatements)
	}
}

// TestPolicyStatementPostfixIsRecompiled is the #817 rule applied to policy
// statements: the stored postfix is a compiled artifact, so a write-out
// replaces it rather than carrying forward a hand-written string that no
// longer matches the description it claims to render.
func TestPolicyStatementPostfixIsRecompiled(t *testing.T) {
	tbl := policyTestTable()
	if err := tbl.SetPolicyStatement(1, "first"); err != nil {
		t.Fatalf("SetPolicyStatement: %v", err)
	}

	for _, ps := range tbl.xml.PolicyStatements {
		if strings.Contains(ps.Postfix, "stale text nobody rendered") {
			t.Errorf("column %s kept its hand-written postfix: %s", ps.Column, ps.Postfix)
		}
	}
	if got, want := postfixFor(tbl, "2"), `"second"`; got != want {
		t.Errorf("column 2 postfix = %s, want %s", got, want)
	}
}

// TestSetPolicyStatementCompilesInterpolation covers the authoring path the
// sample repair campaign needs: rewriting a plain description as a template so
// the compiled statement reports live values.
func TestSetPolicyStatementCompilesInterpolation(t *testing.T) {
	tbl := policyTestTable()
	if err := tbl.SetPolicyStatement(3, "State {state_config.state_code} has no income tax"); err != nil {
		t.Fatalf("SetPolicyStatement: %v", err)
	}

	want := `"State " state_config.state_code cvs strconcat " has no income tax" strconcat`
	if got := postfixFor(tbl, "3"); got != want {
		t.Errorf("compiled statement:\n got %s\nwant %s", got, want)
	}
	if len(tbl.PolicyStatements) != 3 || tbl.PolicyStatements[2].Column != 3 {
		t.Errorf("new statement not appended in column order: %+v", tbl.PolicyStatements)
	}
}

// TestSetPolicyStatementReplaces confirms a second write to the same column
// updates rather than duplicates.
func TestSetPolicyStatementReplaces(t *testing.T) {
	tbl := policyTestTable()
	if err := tbl.SetPolicyStatement(1, "replaced"); err != nil {
		t.Fatalf("SetPolicyStatement: %v", err)
	}
	if len(tbl.PolicyStatements) != 2 {
		t.Fatalf("got %d statements, want 2", len(tbl.PolicyStatements))
	}
	if got := postfixFor(tbl, "1"); got != `"replaced"` {
		t.Errorf("column 1 postfix = %s", got)
	}
}

func TestDeletePolicyStatement(t *testing.T) {
	tbl := policyTestTable()
	if err := tbl.DeletePolicyStatement(1); err != nil {
		t.Fatalf("DeletePolicyStatement: %v", err)
	}
	if len(tbl.PolicyStatements) != 1 || tbl.PolicyStatements[0].Column != 2 {
		t.Errorf("after delete: %+v", tbl.PolicyStatements)
	}
	if len(tbl.xml.PolicyStatements) != 1 {
		t.Errorf("XML still holds %d statements", len(tbl.xml.PolicyStatements))
	}
	if err := tbl.DeletePolicyStatement(9); err == nil {
		t.Error("deleting a column with no statement should fail")
	}
}

func TestSetPolicyStatementRejectsBadColumn(t *testing.T) {
	tbl := policyTestTable()
	if err := tbl.SetPolicyStatement(0, "nope"); err == nil {
		t.Error("column 0 should be rejected")
	}
}

func postfixFor(t *Table, column string) string {
	for _, ps := range t.xml.PolicyStatements {
		if ps.Column == column {
			return ps.Postfix
		}
	}
	return ""
}
