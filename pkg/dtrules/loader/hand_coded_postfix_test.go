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

package loader_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/loader"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestLegacyPostfixTable_RefusesExecute is the end-to-end gate test: a table
// with hand-coded postfix and no EL DSL is loaded by the standard loader,
// the loader marks it, and the runtime Execute / ExecuteTable refuse to
// run it.
//
// This guards against:
//   - the loader forgetting to set the HandCodedPostfix flag,
//   - the runtime forgetting to check it,
//   - the analyzer's detection rule drifting away from the loader's.
func TestLegacyPostfixTable_RefusesExecute(t *testing.T) {
	tmpDir := t.TempDir()

	const eddContent = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="x" type="double" default_value="0.0"/>
  </entity>
</entity_data_dictionary>`

	const dtContent = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>Legacy_Hand_Coded</table_name>
    <xls_file>legacy.xls</xls_file>
    <attribute_fields>
      <TABLE_NUMBER>10001</TABLE_NUMBER>
      <Type>FIRST</Type>
    </attribute_fields>
    <contexts></contexts>
    <initial_actions></initial_actions>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl></condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="Y"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl></action_dsl>
        <action_postfix>1.0 /result.x xdef</action_postfix>
        <action_column column_number="1" column_value="X"/>
      </action_details>
    </actions>
    <policy_statements></policy_statements>
  </decision_table>
</decision_tables>`

	if err := os.WriteFile(filepath.Join(tmpDir, "test_edd.xml"), []byte(eddContent), 0o644); err != nil {
		t.Fatalf("write edd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test_dt.xml"), []byte(dtContent), 0o644); err != nil {
		t.Fatalf("write dt: %v", err)
	}

	rs := session.NewRuleSet("test")
	if err := loader.LoadRulesFromDirectory(rs, tmpDir); err != nil {
		t.Fatalf("LoadRulesFromDirectory: %v", err)
	}

	dt, err := rs.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Legacy_Hand_Coded"))
	if err != nil {
		t.Fatalf("GetDecisionTable: %v", err)
	}
	if dt == nil {
		t.Fatal("expected Legacy_Hand_Coded to be loaded")
	}

	rdt, ok := dt.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatalf("expected *RDecisionTable, got %T", dt)
	}
	if !rdt.HasHandCodedPostfix() {
		t.Fatal("loader did not set HandCodedPostfix flag on a postfix-only table")
	}

	// Runtime refusal: ExecuteTable must return an error mentioning the
	// hand-coded postfix block.
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	err = rdt.ExecuteTable(sess.GetState())
	if err == nil {
		t.Fatal("ExecuteTable should refuse a hand-coded-postfix table")
	}
	if !strings.Contains(err.Error(), "hand-coded postfix") {
		t.Errorf("error should explain the block: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Legacy_Hand_Coded") {
		t.Errorf("error should name the offending table: %q", err.Error())
	}
}

// TestPartiallyAuthoredTable_RefusesExecute: under the strict per-element
// rule, even a single hand-coded-postfix element in an otherwise
// EL-authored table is enough to block execution. The authoring API is
// the supported edit surface; any element bypassing it is rejected.
func TestPartiallyAuthoredTable_RefusesExecute(t *testing.T) {
	tmpDir := t.TempDir()

	const eddContent = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata><file_path>test/10000</file_path></file_metadata>
  <entity name="result">
    <field name="x" type="double" default_value="0.0"/>
    <field name="y" type="double" default_value="0.0"/>
  </entity>
</entity_data_dictionary>`

	const dtContent = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>Partial_DSL_Postfix</table_name>
    <xls_file>partial.xls</xls_file>
    <attribute_fields>
      <TABLE_NUMBER>10002</TABLE_NUMBER>
      <Type>FIRST</Type>
    </attribute_fields>
    <contexts></contexts>
    <initial_actions></initial_actions>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="Y"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>set result.x = 1</action_dsl>
        <action_postfix>1.0 /result.x xdef</action_postfix>
        <action_column column_number="1" column_value="X"/>
      </action_details>
      <action_details>
        <action_number>2</action_number>
        <action_dsl></action_dsl>
        <action_postfix>2.0 /result.y xdef</action_postfix>
        <action_column column_number="1" column_value="X"/>
      </action_details>
    </actions>
    <policy_statements></policy_statements>
  </decision_table>
</decision_tables>`

	if err := os.WriteFile(filepath.Join(tmpDir, "test_edd.xml"), []byte(eddContent), 0o644); err != nil {
		t.Fatalf("write edd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test_dt.xml"), []byte(dtContent), 0o644); err != nil {
		t.Fatalf("write dt: %v", err)
	}

	rs := session.NewRuleSet("test")
	if err := loader.LoadRulesFromDirectory(rs, tmpDir); err != nil {
		t.Fatalf("LoadRulesFromDirectory: %v", err)
	}
	dt, _ := rs.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Partial_DSL_Postfix"))
	rdt, ok := dt.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatalf("expected *RDecisionTable, got %T", dt)
	}
	if !rdt.HasHandCodedPostfix() {
		t.Fatal("strict rule should flag the table — action 2 has postfix without DSL")
	}
	if !strings.Contains(rdt.HandCodedPostfixReason(), "action 2") {
		t.Errorf("reason should name the offending element: %q", rdt.HandCodedPostfixReason())
	}

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := rdt.ExecuteTable(sess.GetState()); err == nil {
		t.Fatal("ExecuteTable should refuse a partially-authored table")
	}
}

// TestFullyAuthoredTable_ExecutesOK: every element has DSL — nothing
// hand-coded — so the table is allowed to execute.
func TestFullyAuthoredTable_ExecutesOK(t *testing.T) {
	tmpDir := t.TempDir()

	const eddContent = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata><file_path>test/10000</file_path></file_metadata>
  <entity name="result">
    <field name="x" type="double" default_value="0.0"/>
  </entity>
</entity_data_dictionary>`

	const dtContent = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>Fully_Authored</table_name>
    <xls_file>authored.xls</xls_file>
    <attribute_fields>
      <TABLE_NUMBER>10003</TABLE_NUMBER>
      <Type>FIRST</Type>
    </attribute_fields>
    <contexts></contexts>
    <initial_actions></initial_actions>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="Y"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>set result.x = 1</action_dsl>
        <action_postfix>1.0 /result.x xdef</action_postfix>
        <action_column column_number="1" column_value="X"/>
      </action_details>
    </actions>
    <policy_statements></policy_statements>
  </decision_table>
</decision_tables>`

	if err := os.WriteFile(filepath.Join(tmpDir, "test_edd.xml"), []byte(eddContent), 0o644); err != nil {
		t.Fatalf("write edd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test_dt.xml"), []byte(dtContent), 0o644); err != nil {
		t.Fatalf("write dt: %v", err)
	}

	rs := session.NewRuleSet("test")
	if err := loader.LoadRulesFromDirectory(rs, tmpDir); err != nil {
		t.Fatalf("LoadRulesFromDirectory: %v", err)
	}
	dt, _ := rs.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Fully_Authored"))
	rdt, ok := dt.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatalf("expected *RDecisionTable, got %T", dt)
	}
	if rdt.HasHandCodedPostfix() {
		t.Error("a fully-authored table should not be flagged")
	}
}
