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

package loader

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
)

// TestDTLoaderWithFilePath tests DT loading when FILE_PATH attribute is present
func TestDTLoaderWithFilePath(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>test_table</table_name>
    <attribute_fields>
      <Type>First</Type>
      <TABLE_NUMBER>1</TABLE_NUMBER>
      <FILE_PATH>states/CA_dt.xml</FILE_PATH>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <postfix>true</postfix>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <postfix>result.status = "ok"</postfix>
      </action_details>
    </actions>
    <policy_statements/>
  </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load DT with FILE_PATH: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("test_table")
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
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>legacy_table</table_name>
    <xls_file>LegacyFile.xls</xls_file>
    <attribute_fields>
      <Type>First</Type>
      <TABLE_NUMBER>2</TABLE_NUMBER>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <postfix>true</postfix>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <postfix>result.status = "ok"</postfix>
      </action_details>
    </actions>
    <policy_statements/>
  </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load DT with xls_file: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("legacy_table")
	tableObj := factory.FindDecisionTable(tableName)
	if tableObj == nil {
		t.Fatal("Expected legacy_table to be created")
	}

	// Cast to concrete type to access GetFilePath
	dt, ok := tableObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatal("Table is not *RDecisionTable")
	}

	// GetFilePath should fall back to xls_file value
	filePath := dt.GetFilePath()
	if filePath != "LegacyFile.xls" {
		t.Errorf("Expected GetFilePath fallback to 'LegacyFile.xls', got: %s", filePath)
	}
}

// TestDTGetFilePathPriority tests that FILE_PATH takes precedence over xls_file
func TestDTGetFilePathPriority(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>priority_table</table_name>
    <xls_file>OldPath.xls</xls_file>
    <attribute_fields>
      <Type>First</Type>
      <TABLE_NUMBER>3</TABLE_NUMBER>
      <FILE_PATH>states/NY_dt.xml</FILE_PATH>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <postfix>true</postfix>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <postfix>result.status = "ok"</postfix>
      </action_details>
    </actions>
    <policy_statements/>
  </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load DT with both FILE_PATH and xls_file: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("priority_table")
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

// TestDTGetFilePathEmpty tests behavior when neither FILE_PATH nor xls_file is present
func TestDTGetFilePathEmpty(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>no_path_table</table_name>
    <attribute_fields>
      <Type>First</Type>
      <TABLE_NUMBER>4</TABLE_NUMBER>
    </attribute_fields>
    <contexts/>
    <initial_actions/>
    <conditions>
      <condition_details>
        <condition_number>1</condition_number>
        <postfix>true</postfix>
      </condition_details>
    </conditions>
    <actions>
      <action_details>
        <action_number>1</action_number>
        <postfix>result.status = "ok"</postfix>
      </action_details>
    </actions>
    <policy_statements/>
  </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load DT without FILE_PATH or xls_file: %v", err)
	}

	// Verify the table was created
	tableName := dtrules.GetRName("no_path_table")
	tableObj := factory.FindDecisionTable(tableName)
	if tableObj == nil {
		t.Fatal("Expected no_path_table to be created")
	}

	// Cast to concrete type to access GetFilePath
	dt, ok := tableObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatal("Table is not *RDecisionTable")
	}

	// GetFilePath should return empty string
	filePath := dt.GetFilePath()
	if filePath != "" {
		t.Errorf("Expected empty file path, got: %s", filePath)
	}
}
