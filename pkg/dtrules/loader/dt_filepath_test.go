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

package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/loader"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestDTLoaderWithFilePath tests DT loading when FILE_PATH attribute is present
func TestDTLoaderWithFilePath(t *testing.T) {
	// Create temp files for loading
	tmpDir := t.TempDir()

	// First create EDD to define result entity
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_edd</file_path>
  </file_metadata>
  <entity name="result">
    <field name="status" type="string" default_value=""/>
  </entity>
</entity_data_dictionary>`
	eddFile := filepath.Join(tmpDir, "10000_test_edd.xml")
	if err := os.WriteFile(eddFile, []byte(eddContent), 0644); err != nil {
		t.Fatalf("Failed to create EDD file: %v", err)
	}

	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>test_table</table_name>
    <attribute_fields>
      <Type>FIRST</Type>
      <TABLE_NUMBER>10001</TABLE_NUMBER>
      <FILE_PATH>states/CA_dt.xml</FILE_PATH>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>Always true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="X"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>Set status</action_dsl>
        <action_postfix>"ok" result status !</action_postfix>
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

	// Load using the directory loader which uses proper session
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("test_table")
	factory := rs.GetEntityFactory()
	tableObj := factory.FindDecisionTable(tableName)
	if tableObj == nil {
		t.Fatal("Expected test_table to be created")
	}

	// Cast to concrete type to access GetFilePath
	dt, ok := tableObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatal("Table is not *RDecisionTable")
	}

	// Verify FILE_PATH is set correctly
	filePath := dt.GetFilePath()
	if filePath != "states/CA_dt.xml" {
		t.Errorf("Expected FILE_PATH='states/CA_dt.xml', got: %s", filePath)
	}
}

// TestDTLoaderLegacyXlsFile tests DT loading with legacy xls_file (no FILE_PATH)
func TestDTLoaderLegacyXlsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create EDD
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_edd</file_path>
  </file_metadata>
  <entity name="result">
    <field name="status" type="string" default_value=""/>
  </entity>
</entity_data_dictionary>`
	eddFile := filepath.Join(tmpDir, "10000_test_edd.xml")
	if err := os.WriteFile(eddFile, []byte(eddContent), 0644); err != nil {
		t.Fatalf("Failed to create EDD file: %v", err)
	}

	// Create DT with xls_file as FILE_PATH fallback
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>legacy_table</table_name>
    <xls_file>LegacyFile.xls</xls_file>
    <attribute_fields>
      <Type>FIRST</Type>
      <TABLE_NUMBER>10002</TABLE_NUMBER>
      <FILE_PATH>test/10002_legacy_table</FILE_PATH>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>Always true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="X"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>Set status</action_dsl>
        <action_postfix>"ok" result status !</action_postfix>
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
	dtFile := filepath.Join(tmpDir, "10002_legacy_dt.xml")
	if err := os.WriteFile(dtFile, []byte(dtContent), 0644); err != nil {
		t.Fatalf("Failed to create DT file: %v", err)
	}

	// Load using the directory loader
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("legacy_table")
	factory := rs.GetEntityFactory()
	tableObj := factory.FindDecisionTable(tableName)
	if tableObj == nil {
		t.Fatal("Expected legacy_table to be created")
	}

	// Cast to concrete type to access GetFilePath
	dt, ok := tableObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatal("Table is not *RDecisionTable")
	}

	// FILE_PATH should be the explicit one, not xls_file
	filePath := dt.GetFilePath()
	if filePath != "test/10002_legacy_table" {
		t.Errorf("Expected FILE_PATH='test/10002_legacy_table', got: %s", filePath)
	}
}

// TestDTGetFilePathPriority tests that FILE_PATH takes precedence over xls_file
func TestDTGetFilePathPriority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create EDD
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_edd</file_path>
  </file_metadata>
  <entity name="result">
    <field name="status" type="string" default_value=""/>
  </entity>
</entity_data_dictionary>`
	eddFile := filepath.Join(tmpDir, "10000_test_edd.xml")
	if err := os.WriteFile(eddFile, []byte(eddContent), 0644); err != nil {
		t.Fatalf("Failed to create EDD file: %v", err)
	}

	// Create DT with both FILE_PATH and xls_file
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>priority_table</table_name>
    <xls_file>OldPath.xls</xls_file>
    <attribute_fields>
      <Type>FIRST</Type>
      <TABLE_NUMBER>10003</TABLE_NUMBER>
      <FILE_PATH>states/NY_dt.xml</FILE_PATH>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>Always true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="X"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>Set status</action_dsl>
        <action_postfix>"ok" result status !</action_postfix>
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
	dtFile := filepath.Join(tmpDir, "10003_priority_dt.xml")
	if err := os.WriteFile(dtFile, []byte(dtContent), 0644); err != nil {
		t.Fatalf("Failed to create DT file: %v", err)
	}

	// Load using the directory loader
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("priority_table")
	factory := rs.GetEntityFactory()
	tableObj := factory.FindDecisionTable(tableName)
	if tableObj == nil {
		t.Fatal("Expected priority_table to be created")
	}

	// Cast to concrete type to access GetFilePath
	dt, ok := tableObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatal("Table is not *RDecisionTable")
	}

	// FILE_PATH should take precedence over xls_file
	filePath := dt.GetFilePath()
	if filePath != "states/NY_dt.xml" {
		t.Errorf("Expected FILE_PATH to take precedence: 'states/NY_dt.xml', got: %s", filePath)
	}

	// Verify old xls_file value is not used
	if filePath == "OldPath.xls" {
		t.Error("FILE_PATH should override xls_file, but got xls_file value instead")
	}
}

// TestDTGetFilePathEmpty tests behavior when FILE_PATH is present but empty
func TestDTGetFilePathEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Create EDD
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_edd</file_path>
  </file_metadata>
  <entity name="result">
    <field name="status" type="string" default_value=""/>
  </entity>
</entity_data_dictionary>`
	eddFile := filepath.Join(tmpDir, "10000_test_edd.xml")
	if err := os.WriteFile(eddFile, []byte(eddContent), 0644); err != nil {
		t.Fatalf("Failed to create EDD file: %v", err)
	}

	// Create DT with FILE_PATH (required for directory loading)
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>no_path_table</table_name>
    <attribute_fields>
      <Type>FIRST</Type>
      <TABLE_NUMBER>10004</TABLE_NUMBER>
      <FILE_PATH>test/10004_no_path_table</FILE_PATH>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <condition_dsl>Always true</condition_dsl>
        <condition_postfix>true</condition_postfix>
        <condition_column column_number="1" column_value="X"/>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <action_dsl>Set status</action_dsl>
        <action_postfix>"ok" result status !</action_postfix>
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
	dtFile := filepath.Join(tmpDir, "10004_no_path_dt.xml")
	if err := os.WriteFile(dtFile, []byte(dtContent), 0644); err != nil {
		t.Fatalf("Failed to create DT file: %v", err)
	}

	// Load using the directory loader
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("no_path_table")
	factory := rs.GetEntityFactory()
	tableObj := factory.FindDecisionTable(tableName)
	if tableObj == nil {
		t.Fatal("Expected no_path_table to be created")
	}

	// Cast to concrete type to access GetFilePath
	dt, ok := tableObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatal("Table is not *RDecisionTable")
	}

	// GetFilePath should return the FILE_PATH we set
	filePath := dt.GetFilePath()
	if filePath != "test/10004_no_path_table" {
		t.Errorf("Expected FILE_PATH='test/10004_no_path_table', got: %s", filePath)
	}
}
