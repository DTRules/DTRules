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

package dtrules_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Pins the multi-entry-point contract: load a project once, populate a
// session with input data, then execute different decision tables as
// separate entry points against the same session. State written by
// the first table must be visible to the second.
//
// The pattern is supported directly by `RSession.Execute(tableName)`
// — every loaded decision table is callable by name; there is no
// special "entry point" registration. See:
//   - docs/multi-entry-points.md (long-form companion)
//   - `dtrules docs entry-points` (embedded topic)
//
// Originally surfaced as a documentation/test gap during the v1.15.0
// review: the engine supported the pattern but no focused regression
// test existed to pin it. The closest test (`TestSyntaxTestsIntegration`)
// exercises a similar shape but is currently failing on unrelated
// sample-project content issues, which hides the validation.

const multiEntryEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="client" access="rw">
    <field name="age" type="integer" subtype="" access="rw" input="" default_value="0" comment=""/>
    <field name="eligible" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
    <field name="risk_score" type="integer" subtype="" access="rw" input="" default_value="0" comment=""/>
    <field name="audit_count" type="integer" subtype="" access="rw" input="" default_value="0" comment=""/>
  </entity>
</entity_data_dictionary>
`

const multiEntryDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Check_Eligibility</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>client.age &gt;= 18</condition_dsl>
    <condition_postfix>client.age 18 &gt;=</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>set client.eligible = true</action_dsl>
    <action_postfix>true cvb /client.eligible xdef</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
<decision_table>
<table_name>Compute_Risk</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>2</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>client.age &lt; 25</condition_dsl>
    <condition_postfix>client.age 25 &lt;</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>set client.risk_score = 80</action_dsl>
    <action_postfix>80 /client.risk_score xdef</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
<decision_table>
<table_name>Generate_Audit</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>3</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>client.eligible</condition_dsl>
    <condition_postfix>client.eligible</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>set client.audit_count = 1</action_dsl>
    <action_postfix>1 /client.audit_count xdef</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

// loadMultiEntryProject is a test helper: load EDD + DT, create a
// session, populate a single `client` entity with age=age, and return
// the session and entity for the test body to drive.
func loadMultiEntryProject(t *testing.T, age int64) (*session.RSession, dtrules.Entity) {
	t.Helper()
	rs := session.NewRuleSet("MultiEntryTest")
	if err := rs.LoadEDD(strings.NewReader(multiEntryEDD)); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}
	if err := rs.LoadDecisionTables(strings.NewReader(multiEntryDT)); err != nil {
		t.Fatalf("LoadDecisionTables: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	rsess := sess.(*session.RSession)
	entity, err := rsess.CreateEntity(dtrules.GetRName("client"))
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	entity.Put(dtrules.GetRName("age"), dtrules.GetRIntegerValueFromInt(int(age)))
	rsess.GetState().EntityPush(entity)
	t.Cleanup(func() { rsess.GetState().EntityPop() })
	return rsess, entity
}

func getInt(t *testing.T, e dtrules.Entity, field string) int {
	t.Helper()
	obj, _ := e.Get(dtrules.GetRName(field))
	ri, err := obj.RIntegerValue()
	if err != nil {
		t.Fatalf("RIntegerValue(%s): %v", field, err)
	}
	v, _ := ri.IntValue()
	return v
}

func getBool(t *testing.T, e dtrules.Entity, field string) bool {
	t.Helper()
	obj, _ := e.Get(dtrules.GetRName(field))
	rb, err := obj.RBooleanValue()
	if err != nil {
		t.Fatalf("RBooleanValue(%s): %v", field, err)
	}
	return rb.Value()
}

// TestMultipleEntryPoints_TwoTablesSeeSharedState is the headline
// contract: two Execute calls on the same session see each other's
// mutations.
func TestMultipleEntryPoints_TwoTablesSeeSharedState(t *testing.T) {
	rsess, entity := loadMultiEntryProject(t, 20)

	// Entry point #1: sets client.eligible = true.
	if err := rsess.Execute("Check_Eligibility"); err != nil {
		t.Fatalf("Execute(Check_Eligibility): %v", err)
	}
	if !getBool(t, entity, "eligible") {
		t.Fatalf("after Check_Eligibility: expected eligible=true")
	}

	// Entry point #2: sees the entity from #1, sets risk_score.
	if err := rsess.Execute("Compute_Risk"); err != nil {
		t.Fatalf("Execute(Compute_Risk): %v", err)
	}
	if got := getInt(t, entity, "risk_score"); got != 80 {
		t.Errorf("after Compute_Risk: expected risk_score=80, got %d", got)
	}

	// And #1's mutation is still visible after #2 — neither call
	// rolled back the other.
	if !getBool(t, entity, "eligible") {
		t.Errorf("eligible regressed across calls — second Execute must not have rolled back first")
	}
}

// TestMultipleEntryPoints_TableTwoReadsTableOnesWrite pins the
// dependency direction: table B's condition reads a field that table
// A wrote. Without state persistence across Execute calls, B's
// condition would see the EDD default (false) and the audit row
// would not fire.
func TestMultipleEntryPoints_TableTwoReadsTableOnesWrite(t *testing.T) {
	rsess, entity := loadMultiEntryProject(t, 20)

	// Pre-condition: audit_count starts at EDD default (0).
	if got := getInt(t, entity, "audit_count"); got != 0 {
		t.Fatalf("audit_count should start at 0, got %d", got)
	}

	// Run #1 (sets eligible=true).
	if err := rsess.Execute("Check_Eligibility"); err != nil {
		t.Fatalf("Execute(Check_Eligibility): %v", err)
	}

	// Run #2 (Generate_Audit; condition is `client.eligible`).
	if err := rsess.Execute("Generate_Audit"); err != nil {
		t.Fatalf("Execute(Generate_Audit): %v", err)
	}

	if got := getInt(t, entity, "audit_count"); got != 1 {
		t.Errorf("after Generate_Audit: expected audit_count=1 (proves the table read client.eligible written by Check_Eligibility), got %d", got)
	}
}

// TestMultipleEntryPoints_ConditionMissCarriesForward pins the
// negative case: an entity that fails the first table's condition
// (age=12 < 18 → eligible stays false) means the second table sees
// the default state — confirming the entity persists in both
// directions.
func TestMultipleEntryPoints_ConditionMissCarriesForward(t *testing.T) {
	rsess, entity := loadMultiEntryProject(t, 12)

	if err := rsess.Execute("Check_Eligibility"); err != nil {
		t.Fatalf("Execute(Check_Eligibility): %v", err)
	}
	// FIRST policy with a single column matching age >= 18; age=12
	// misses → no action → eligible stays at the EDD default (false).
	if getBool(t, entity, "eligible") {
		t.Fatalf("expected eligible=false for age=12, got true")
	}

	// Generate_Audit's condition reads eligible (still false) →
	// audit_count stays at 0.
	if err := rsess.Execute("Generate_Audit"); err != nil {
		t.Fatalf("Execute(Generate_Audit): %v", err)
	}
	if got := getInt(t, entity, "audit_count"); got != 0 {
		t.Errorf("expected audit_count=0 when eligible was false across both calls, got %d", got)
	}
}

// TestMultipleEntryPoints_UnknownTableErrors pins the configuration-
// bug contract: Execute with a name that isn't a loaded table returns
// an error, doesn't panic, and doesn't mutate session state.
func TestMultipleEntryPoints_UnknownTableErrors(t *testing.T) {
	rsess, entity := loadMultiEntryProject(t, 20)

	err := rsess.Execute("Definitely_Not_A_Table")
	if err == nil {
		t.Fatal("Execute on an unknown table must return an error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should name the failure mode, got: %v", err)
	}
	// Session state must be untouched by a failed lookup.
	if getBool(t, entity, "eligible") {
		t.Errorf("eligible should still be the default false after a failed Execute, got true")
	}
}

// TestMultipleEntryPoints_ReExecuteSameTableIsIdempotent pins that
// calling Execute twice on the SAME table is allowed and produces
// the same result — useful for retry / replay workflows.
func TestMultipleEntryPoints_ReExecuteSameTableIsIdempotent(t *testing.T) {
	rsess, entity := loadMultiEntryProject(t, 20)

	if err := rsess.Execute("Check_Eligibility"); err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	if !getBool(t, entity, "eligible") {
		t.Fatalf("after first Execute: expected eligible=true")
	}

	if err := rsess.Execute("Check_Eligibility"); err != nil {
		t.Fatalf("second Execute on same table: %v", err)
	}
	if !getBool(t, entity, "eligible") {
		t.Errorf("after second Execute on same table: expected eligible still true")
	}
}
