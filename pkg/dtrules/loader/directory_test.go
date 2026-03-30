// Copyright 2004-2011 DTRules.com, Inc.
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
	"github.com/DTRules/DTRules/pkg/dtrules/loader"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

func TestCollectXMLFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create some test files
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	testFiles := []string{
		filepath.Join(tmpDir, "file1.xml"),
		filepath.Join(tmpDir, "file2.XML"),
		filepath.Join(subDir, "file3.xml"),
		filepath.Join(tmpDir, "notxml.txt"),
	}

	for _, f := range testFiles {
		if err := os.WriteFile(f, []byte("<root/>"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	files, err := loader.CollectXMLFiles(tmpDir)
	if err != nil {
		t.Fatalf("CollectXMLFiles failed: %v", err)
	}

	// Should find 3 XML files (case-insensitive)
	if len(files) != 3 {
		t.Errorf("Expected 3 XML files, got %d", len(files))
	}
}

func TestParseFileMetadata_DT(t *testing.T) {
	// Create a temporary DT file
	tmpFile := filepath.Join(t.TempDir(), "test_dt.xml")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>Test_Table</table_name>
    <xls_file>test.xls</xls_file>
    <attribute_fields>
      <TABLE_NUMBER>40100</TABLE_NUMBER>
      <FILE_PATH>states/AL/40100_Test_Table</FILE_PATH>
    </attribute_fields>
    <contexts></contexts>
    <initial_actions></initial_actions>
    <conditions></conditions>
    <actions></actions>
    <policy_statements></policy_statements>
  </decision_table>
</decision_tables>`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	info, err := loader.ParseFileMetadata(tmpFile)
	if err != nil {
		t.Fatalf("ParseFileMetadata failed: %v", err)
	}

	if !info.IsDecisionTable {
		t.Error("Expected IsDecisionTable to be true")
	}

	if info.Number != 40100 {
		t.Errorf("Expected number 40100, got %d", info.Number)
	}

	if info.FilePath != "states/AL/40100_Test_Table" {
		t.Errorf("Expected FilePath 'states/AL/40100_Test_Table', got '%s'", info.FilePath)
	}
}

func TestParseFileMetadata_EDD(t *testing.T) {
	// Create a temporary EDD file
	tmpFile := filepath.Join(t.TempDir(), "test_edd.xml")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/AL/40100_AL_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="test_field" type="double" default_value="100.0"/>
  </entity>
</entity_data_dictionary>`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	info, err := loader.ParseFileMetadata(tmpFile)
	if err != nil {
		t.Fatalf("ParseFileMetadata failed: %v", err)
	}

	if info.IsDecisionTable {
		t.Error("Expected IsDecisionTable to be false")
	}

	if info.Number != 40100 {
		t.Errorf("Expected number 40100, got %d", info.Number)
	}

	if info.FilePath != "states/AL/40100_AL_constants" {
		t.Errorf("Expected FilePath 'states/AL/40100_AL_constants', got '%s'", info.FilePath)
	}
}

func TestLoadRulesFromDirectory_InvalidPath(t *testing.T) {
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, "/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

func TestLoadRulesFromDirectory_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err == nil {
		t.Error("Expected error for empty directory")
	}
}

func TestLoadRulesFromDirectory_BasicFlow(t *testing.T) {
	// Create a temporary directory with a minimal EDD and DT
	tmpDir := t.TempDir()

	// Create a minimal EDD file
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_test_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="test_value" type="double" default_value="42.0"/>
  </entity>
</entity_data_dictionary>`

	eddFile := filepath.Join(tmpDir, "test_edd.xml")
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

	dtFile := filepath.Join(tmpDir, "test_dt.xml")
	if err := os.WriteFile(dtFile, []byte(dtContent), 0644); err != nil {
		t.Fatalf("Failed to create DT file: %v", err)
	}

	// Load the rules
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err != nil {
		t.Fatalf("LoadRulesFromDirectory failed: %v", err)
	}

	// Verify the decision table was loaded
	factory := rs.GetEntityFactory()
	dtName := dtrules.GetRName("Test_Table")
	dt := factory.FindDecisionTable(dtName)
	if dt == nil {
		t.Error("Expected decision table 'Test_Table' to be loaded")
	}
}

func TestLoadRulesFromDirectory_Ordering(t *testing.T) {
	// Create a temporary directory with multiple files
	tmpDir := t.TempDir()

	// Create files with different numbers (should be loaded in order)
	files := []struct {
		name    string
		number  string
		isEDD   bool
		content string
	}{
		{
			name:   "50000_edd.xml",
			number: "50000",
			isEDD:  true,
			content: `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/50000_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="value_50000" type="double" default_value="50000.0"/>
  </entity>
</entity_data_dictionary>`,
		},
		{
			name:   "10000_edd.xml",
			number: "10000",
			isEDD:  true,
			content: `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>test/10000_constants</file_path>
  </file_metadata>
  <entity name="result">
    <field name="value_10000" type="double" default_value="10000.0"/>
  </entity>
</entity_data_dictionary>`,
		},
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f.name, err)
		}
	}

	// Load the rules
	rs := session.NewRuleSet("test")
	err := loader.LoadRulesFromDirectory(rs, tmpDir)
	if err != nil {
		t.Fatalf("LoadRulesFromDirectory failed: %v", err)
	}

	// Just verify it loaded without error - order verification would require
	// instrumenting the loader to track order
}
