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

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDTImporter_RoundTripPreservesFields exercises every DT XML element the
// loader cares about and asserts that read → write → re-read produces an
// identical in-memory model. Guards against regressions like the one that
// silently demoted <context_entity> directives to raw text inside a
// <context_dsl> on round-trip (and stripped postfix, action comments, and
// "-" condition columns).
func TestDTImporter_RoundTripPreservesFields(t *testing.T) {
	const sourceXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>

<!-- TABLE 1: Round_Trip_Test -->
<decision_table>
<table_name>Round_Trip_Test</table_name>
<xls_file>Test.xls</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS>Test table exercising every preserved field</COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts>
<context_entity>state_period</context_entity>
<context_details>
<context_number>1</context_number>
<context_comment>Iterate accounts</context_comment>
<context_dsl>for all accounts</context_dsl>
<context_postfix>
{ /Process_Account executetable } job.accounts forall
</context_postfix>
</context_details>
</contexts>
<initial_actions>
<initial_action>
<action_comment>Initialize totals</action_comment>
<initial_action_dsl>set result.total = 0</initial_action_dsl>
<initial_action_postfix>
0 /result.total xdef
</initial_action_postfix>
</initial_action>
</initial_actions>
<conditions>
<condition_details>
<condition_number>1</condition_number>
<condition_comment>Account is auth</condition_comment>
<condition_dsl>account.is_auth is equal to true</condition_dsl>
<condition_postfix>
account.is_auth true ==
</condition_postfix>
<condition_column column_number="1" column_value="Y" />
<condition_column column_number="2" column_value="N" />
<condition_column column_number="3" column_value="-" />
</condition_details>
</conditions>
<actions>
<action_details>
<action_number>1</action_number>
<action_comment>Add to total</action_comment>
<action_dsl>add account.balance to result.total</action_dsl>
<action_postfix>
account.balance result.total f+ /result.total xdef
</action_postfix>
<action_column column_number="1" column_value="X" />
</action_details>
</actions>
<policy_statements>
<policy_statement column="1">
<policy_description>Auth account balance counted</policy_description>
<policy_statement_postfix>"Auth account counted"</policy_statement_postfix>
</policy_statement>
</policy_statements>
</decision_table>
</decision_tables>
`

	dir := t.TempDir()
	src := filepath.Join(dir, "in.xml")
	if err := os.WriteFile(src, []byte(sourceXML), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Read.
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read src: %v", err)
	}
	var tables DecisionTablesXML
	if err := xml.Unmarshal(data, &tables); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(tables.Tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables.Tables))
	}
	t1 := tables.Tables[0]

	// Spot-check critical preserved fields after the first parse.
	if got, want := t1.TableName, "Round_Trip_Test"; got != want {
		t.Errorf("table_name = %q, want %q", got, want)
	}
	if len(t1.Contexts.Entities) != 1 || t1.Contexts.Entities[0] != "state_period" {
		t.Errorf("context_entity not preserved: %v", t1.Contexts.Entities)
	}
	if len(t1.Contexts.Details) != 1 {
		t.Fatalf("context details count = %d, want 1", len(t1.Contexts.Details))
	}
	cd := t1.Contexts.Details[0]
	if cd.Number != 1 || cd.Comment != "Iterate accounts" || cd.DSL != "for all accounts" {
		t.Errorf("context detail fields lost: %+v", cd)
	}
	if !strings.Contains(cd.Postfix, "executetable } job.accounts forall") {
		t.Errorf("context postfix lost: %q", cd.Postfix)
	}
	if len(t1.InitialActions) != 1 {
		t.Fatalf("initial_actions count = %d, want 1", len(t1.InitialActions))
	}
	ia := t1.InitialActions[0]
	if ia.Comment != "Initialize totals" || ia.DSL != "set result.total = 0" {
		t.Errorf("initial_action fields lost: %+v", ia)
	}
	if !strings.Contains(ia.Postfix, "0 /result.total xdef") {
		t.Errorf("initial_action_postfix lost: %q", ia.Postfix)
	}
	// "-" condition column must round-trip.
	cond := t1.Conditions[0]
	var sawDash bool
	for _, cv := range cond.Columns {
		if cv.Number == 3 && cv.Value == "-" {
			sawDash = true
		}
	}
	if !sawDash {
		t.Errorf("condition column 3 with value \"-\" not preserved: %+v", cond.Columns)
	}

	// Write.
	out := filepath.Join(dir, "out.xml")
	imp := NewDTImporter()
	if err := imp.WriteXML(&tables, out); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Re-read.
	data2, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	var tables2 DecisionTablesXML
	if err := xml.Unmarshal(data2, &tables2); err != nil {
		t.Fatalf("re-unmarshal: %v\nXML:\n%s", err, data2)
	}
	if len(tables2.Tables) != 1 {
		t.Fatalf("after round-trip: got %d tables", len(tables2.Tables))
	}
	t2 := tables2.Tables[0]

	// Every preserved field must still be there.
	if t2.TableName != t1.TableName {
		t.Errorf("round-trip lost table_name: %q vs %q", t2.TableName, t1.TableName)
	}
	if len(t2.Contexts.Entities) != 1 || t2.Contexts.Entities[0] != "state_period" {
		t.Errorf("round-trip lost context_entity: %v", t2.Contexts.Entities)
	}
	if len(t2.Contexts.Details) != 1 {
		t.Fatalf("round-trip lost context details (count = %d)", len(t2.Contexts.Details))
	}
	cd2 := t2.Contexts.Details[0]
	if cd2.Comment != cd.Comment || cd2.DSL != cd.DSL {
		t.Errorf("round-trip lost context detail content: %+v vs %+v", cd2, cd)
	}
	if strings.TrimSpace(cd2.Postfix) != strings.TrimSpace(cd.Postfix) {
		t.Errorf("round-trip changed context postfix\nbefore: %q\nafter:  %q", cd.Postfix, cd2.Postfix)
	}
	ia2 := t2.InitialActions[0]
	if ia2.Comment != ia.Comment {
		t.Errorf("round-trip lost initial_action comment: %q vs %q", ia2.Comment, ia.Comment)
	}
	if ia2.EffectiveDSL() != ia.EffectiveDSL() {
		t.Errorf("round-trip lost initial_action DSL: %q vs %q", ia2.EffectiveDSL(), ia.EffectiveDSL())
	}
	if strings.TrimSpace(ia2.EffectivePostfix()) != strings.TrimSpace(ia.EffectivePostfix()) {
		t.Errorf("round-trip lost initial_action postfix\nbefore: %q\nafter:  %q",
			ia.EffectivePostfix(), ia2.EffectivePostfix())
	}
	cond2 := t2.Conditions[0]
	if strings.TrimSpace(cond2.Postfix) != strings.TrimSpace(cond.Postfix) {
		t.Errorf("round-trip lost condition postfix\nbefore: %q\nafter:  %q", cond.Postfix, cond2.Postfix)
	}
	// Read through EffectiveColumns: rows are written dense now, so .Columns
	// is empty and the cells live in .Cells. The claim is unchanged -- a "-"
	// must survive the round trip -- and it is now structural rather than
	// something the writer has to remember, because a dense row has no way to
	// omit a cell (#1017, #1079).
	sawDash = false
	for _, cv := range cond2.EffectiveColumns() {
		if cv.Number == 3 && cv.Value == "-" {
			sawDash = true
		}
	}
	if !sawDash {
		t.Errorf("round-trip dropped \"-\" condition column: cells=%q cols=%+v",
			cond2.Cells, cond2.Columns)
	}
	act := t1.Actions[0]
	act2 := t2.Actions[0]
	if act2.DSL != act.DSL || act2.Comment != act.Comment {
		t.Errorf("round-trip lost action fields: %+v vs %+v", act2, act)
	}
	if strings.TrimSpace(act2.Postfix) != strings.TrimSpace(act.Postfix) {
		t.Errorf("round-trip lost action postfix\nbefore: %q\nafter:  %q", act.Postfix, act2.Postfix)
	}
	if len(t2.PolicyStatements) != 1 {
		t.Fatalf("round-trip lost policy statement (count = %d)", len(t2.PolicyStatements))
	}
	if t2.PolicyStatements[0].Description != t1.PolicyStatements[0].Description {
		t.Errorf("round-trip lost policy description")
	}
}

// TestDTImporter_RoundTripLegacyInitialActionTags verifies that initial
// actions authored with the legacy <action_dsl> / <action_postfix> tag pair
// (rather than <initial_action_dsl> / <initial_action_postfix>) are preserved
// on round-trip via the ActionDSL / ActionPostfix fields.
func TestDTImporter_RoundTripLegacyInitialActionTags(t *testing.T) {
	const sourceXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Legacy_Initial</table_name>
<xls_file>Test.xls</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS>Legacy tag form</COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions>
<initial_action>
<action_comment>Legacy tag form</action_comment>
<action_dsl>set result.x = 0</action_dsl>
<action_postfix>
0 /result.x xdef
</action_postfix>
</initial_action>
</initial_actions>
<conditions></conditions>
<actions></actions>
<policy_statements></policy_statements>
</decision_table>
</decision_tables>
`
	dir := t.TempDir()
	src := filepath.Join(dir, "in.xml")
	if err := os.WriteFile(src, []byte(sourceXML), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	data, _ := os.ReadFile(src)
	var tables DecisionTablesXML
	if err := xml.Unmarshal(data, &tables); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ia := tables.Tables[0].InitialActions[0]
	if ia.EffectiveDSL() != "set result.x = 0" {
		t.Errorf("EffectiveDSL did not read legacy <action_dsl>: %q", ia.EffectiveDSL())
	}
	if !strings.Contains(ia.EffectivePostfix(), "0 /result.x xdef") {
		t.Errorf("EffectivePostfix did not read legacy <action_postfix>: %q", ia.EffectivePostfix())
	}
	// Write + re-read; legacy tags should round-trip.
	out := filepath.Join(dir, "out.xml")
	if err := NewDTImporter().WriteXML(&tables, out); err != nil {
		t.Fatalf("write: %v", err)
	}
	data2, _ := os.ReadFile(out)
	var tables2 DecisionTablesXML
	if err := xml.Unmarshal(data2, &tables2); err != nil {
		t.Fatalf("re-unmarshal: %v\n%s", err, data2)
	}
	ia2 := tables2.Tables[0].InitialActions[0]
	if ia2.EffectiveDSL() != ia.EffectiveDSL() {
		t.Errorf("round-trip lost legacy DSL: %q vs %q", ia2.EffectiveDSL(), ia.EffectiveDSL())
	}
	if strings.TrimSpace(ia2.EffectivePostfix()) != strings.TrimSpace(ia.EffectivePostfix()) {
		t.Errorf("round-trip lost legacy postfix\nbefore: %q\nafter:  %q",
			ia.EffectivePostfix(), ia2.EffectivePostfix())
	}
}
