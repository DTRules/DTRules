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
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

// TestLoadFromDirectoryQuick is a quick test that verifies directory loading works
// without loading the large TaxReturn project
func TestLoadFromDirectoryQuick(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create a minimal EDD file
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="test_value" type="double" default_value="42.0"/>
    <field name="state_tax" type="double" default_value="0.0"/>
  </entity>
</entity_data_dictionary>`

	eddFile := filepath.Join(tmpDir, "10000_test_edd.xml")
	if err := os.WriteFile(eddFile, []byte(eddContent), 0644); err != nil {
		t.Fatalf("Failed to create EDD file: %v", err)
	}

	// Create a minimal DT file
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
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
        <condition_description>true</condition_description>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="X"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_description>Set result.test_value = 100</action_description>
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

	dtFile := filepath.Join(tmpDir, "10001_test_dt.xml")
	if err := os.WriteFile(dtFile, []byte(dtContent), 0644); err != nil {
		t.Fatalf("Failed to create DT file: %v", err)
	}

	// Test LoadFromDirectory
	rs := session.NewRuleSet("QuickTest")
	err := rs.LoadFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDirectory failed: %v", err)
	}

	// Verify entities loaded
	entities := rs.GetEntityNames()
	if len(entities) == 0 {
		t.Error("No entities loaded")
	}
	t.Logf("Loaded %d entities", len(entities))

	// Verify decision tables loaded
	tables := rs.GetDecisionTableNames()
	if len(tables) == 0 {
		t.Error("No decision tables loaded")
	}
	t.Logf("Loaded %d decision tables", len(tables))

	// Verify specific table exists
	found := false
	for _, tbl := range tables {
		if tbl.StringValue() == "Test_Table" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected table 'Test_Table' not found")
	}

	t.Log("Quick multi-file loading test PASSED")
}
