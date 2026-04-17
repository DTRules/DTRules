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

package session_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const minimalEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="test_value" type="double" default_value="42.0"/>
  </entity>
</entity_data_dictionary>`

const minimalDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>Test_Table</table_name>
    <xls_file>test.xls</xls_file>
    <attribute_fields>
      <TABLE_NUMBER>10001</TABLE_NUMBER>
      <FILE_PATH>test/10001_Test_Table</FILE_PATH>
      <Type>FIRST</Type>
    </attribute_fields>
    <contexts></contexts>
    <initial_actions></initial_actions>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="X"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>Set result.test_value = 100</action_dsl>
        <action_postfix>100.0 result test_value !</action_postfix>
        <action_column column_number="1" column_value="X"/>
      </action_details>
    </actions>
    <policy_statements>
      <policy_statement column="1">
        <policy_description>Test case</policy_description>
        <policy_statement_postfix></policy_statement_postfix>
      </policy_statement>
    </policy_statements>
  </decision_table>
</decision_tables>`

// TestLoadRulesFromFS_WithEmbed simulates loading from an embed.FS using fstest.MapFS.
func TestLoadRulesFromFS_WithEmbed(t *testing.T) {
	mapFS := fstest.MapFS{
		"test_edd.xml": {Data: []byte(minimalEDD)},
		"test_dt.xml":  {Data: []byte(minimalDT)},
	}

	rs, err := session.LoadRulesFromFS("TestFS", mapFS, ".")
	if err != nil {
		t.Fatalf("LoadRulesFromFS failed: %v", err)
	}

	factory := rs.GetEntityFactory()
	dtName := dtrules.GetRName("Test_Table")
	dt := factory.FindDecisionTable(dtName)
	if dt == nil {
		t.Error("expected decision table 'Test_Table' to be loaded")
	}

	entityNames := rs.GetEntityNames()
	found := false
	for _, n := range entityNames {
		if n.GetName() == "result" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected entity 'result' to be loaded")
	}
}

// TestLoadRulesFromFS_NestedRoot verifies that root can be a subdirectory inside the FS.
func TestLoadRulesFromFS_NestedRoot(t *testing.T) {
	mapFS := fstest.MapFS{
		"rules/test_edd.xml": {Data: []byte(minimalEDD)},
		"rules/test_dt.xml":  {Data: []byte(minimalDT)},
		// file outside the root should be ignored
		"other/ignored.xml": {Data: []byte("<root/>")},
	}

	rs, err := session.LoadRulesFromFS("TestFSNested", mapFS, "rules")
	if err != nil {
		t.Fatalf("LoadRulesFromFS (nested root) failed: %v", err)
	}

	factory := rs.GetEntityFactory()
	dtName := dtrules.GetRName("Test_Table")
	if factory.FindDecisionTable(dtName) == nil {
		t.Error("expected decision table 'Test_Table' to be loaded from nested root")
	}
}

// TestLoadRulesFromDirectory_StillWorks is a regression test ensuring the path-based API is unchanged.
func TestLoadRulesFromDirectory_StillWorks(t *testing.T) {
	dirPath := t.TempDir()

	if err := os.WriteFile(dirPath+"/test_edd.xml", []byte(minimalEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dirPath+"/test_dt.xml", []byte(minimalDT), 0644); err != nil {
		t.Fatal(err)
	}

	rs, err := session.LoadRulesFromDirectory("TestDir", dirPath)
	if err != nil {
		t.Fatalf("LoadRulesFromDirectory failed: %v", err)
	}

	factory := rs.GetEntityFactory()
	if factory.FindDecisionTable(dtrules.GetRName("Test_Table")) == nil {
		t.Error("expected decision table to be loaded via directory path")
	}
}

// TestLoadRulesFromFS_MissingFile verifies a useful error when the FS root contains no XML.
func TestLoadRulesFromFS_MissingFile(t *testing.T) {
	mapFS := fstest.MapFS{
		"readme.txt": {Data: []byte("no xml here")},
	}

	_, err := session.LoadRulesFromFS("TestMissing", mapFS, ".")
	if err == nil {
		t.Fatal("expected error when no XML files present, got nil")
	}
}
