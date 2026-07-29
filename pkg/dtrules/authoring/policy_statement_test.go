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

// TestContextLocalsAreVisibleToRows guards #965.
//
// A table can declare a local in a context row and refer to it from its
// conditions and actions. Compiling each row through its own EL compiler lost
// the slot, so the name was emitted bare and the rule died at execute with
// "The Name 'ApplyingClient' was not defined by any Entity on the Entity
// Stack". CHIP's Calculate_Group_Size had been dead that way for as long as
// the sample existed, and ChipApp, KidAid and SyntaxTests use the same idiom.
func TestContextLocalsAreVisibleToRows(t *testing.T) {
	symbols := map[string]string{
		"client":          "entity",
		"client.applying": "boolean",
		"clients":         "array",
	}
	tbl := newTable(&excel.DecisionTableXML{
		TableName: "Calculate_Group_Size",
		Contexts: excel.ContextsField{Details: []excel.ContextDetailXML{
			{Number: 1, DSL: "for all clients"},
			{Number: 2, DSL: "local entity ApplyingClient = client"},
		}},
		Conditions: []excel.ConditionXML{
			{Number: "1", DSL: "ApplyingClient == client"},
		},
	}, symbols)

	// Force a write-out, which is where postfix is generated.
	if err := tbl.UpdateCondition(1, Condition{
		Number:  1,
		DSL:     "ApplyingClient == client",
		Columns: map[int]string{1: "Y"},
	}); err != nil {
		t.Fatalf("UpdateCondition: %v", err)
	}

	got := strings.Join(strings.Fields(tbl.xml.Conditions[0].Postfix), " ")
	if strings.Contains(got, "ApplyingClient") {
		t.Errorf("the local is emitted as a bare name and will not resolve at run time: %s", got)
	}
	if !strings.Contains(got, "local@") {
		t.Errorf("condition postfix does not reference a local slot: %s", got)
	}
}

// TestLocalSlotsDoNotBleedBetweenTables: each table gets a fresh scope, so
// slot indices from one table's contexts cannot be reused by another's rows.
func TestLocalSlotsDoNotBleedBetweenTables(t *testing.T) {
	symbols := map[string]string{"client": "entity", "clients": "array"}
	build := func(name string) string {
		tbl := newTable(&excel.DecisionTableXML{
			TableName: name,
			Contexts: excel.ContextsField{Details: []excel.ContextDetailXML{
				{Number: 1, DSL: "for all clients"},
				{Number: 2, DSL: "local entity Held = client"},
			}},
			Conditions: []excel.ConditionXML{{Number: "1", DSL: "Held == client"}},
		}, symbols)
		if err := tbl.UpdateCondition(1, Condition{Number: 1, DSL: "Held == client", Columns: map[int]string{1: "Y"}}); err != nil {
			t.Fatalf("UpdateCondition: %v", err)
		}
		return strings.Join(strings.Fields(tbl.xml.Conditions[0].Postfix), " ")
	}

	first, second := build("First_Table"), build("Second_Table")
	if first != second {
		t.Errorf("slot indices differ between tables that declare the same locals:\n  %s\n  %s", first, second)
	}
}

// TestHandCodedRowsAreReportedBeforeARecompileEatsThem is the guard for the
// one operation in this package that destroys content.
//
// syncToXML regenerates every postfix from its DSL, so a row whose only
// content is postfix comes back empty — its logic gone, from an operation
// that reads like normalization. Four sample projects lost rows that way
// during the repair campaign, and no test caught it because the emptied rows
// were not covered by any scenario. HandCodedRows names them first.
func TestHandCodedRowsAreReportedBeforeARecompileEatsThem(t *testing.T) {
	tbl := newTable(&excel.DecisionTableXML{
		TableName: "Evaluate_MEDICAID_Eligibility",
		Conditions: []excel.ConditionXML{
			{Number: "1", DSL: "client.age > 18", Postfix: "client.age 18 >"},
		},
		Actions: []excel.ActionXML{
			{Number: "1", DSL: "set client.eligible = false", Postfix: "false cvb /client.eligible xdef"},
			// The shape that gets destroyed: postfix, no DSL.
			{Number: "2", Postfix: `"not supported" client.notes swap addto`},
		},
	}, map[string]string{})

	got := tbl.HandCodedRows()
	if len(got) != 1 || got[0] != "action 2" {
		t.Fatalf("HandCodedRows() = %v, want [\"action 2\"]", got)
	}

	// A table authored in EL throughout has nothing to report.
	clean := newTable(&excel.DecisionTableXML{
		TableName: "Clean",
		Actions: []excel.ActionXML{
			{Number: "1", DSL: "set client.eligible = true", Postfix: "true cvb /client.eligible xdef"},
		},
	}, map[string]string{})
	if rows := clean.HandCodedRows(); len(rows) != 0 {
		t.Errorf("a fully authored table reports %v, want nothing", rows)
	}
}
