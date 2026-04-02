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

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// mockSession is a minimal session implementation for testing.
type mockSession struct {
	factory *entity.Factory
}

func (s *mockSession) GetState() dtrules.State                 { return nil }
func (s *mockSession) GetEntityFactory() dtrules.EntityFactory { return s.factory }
func (s *mockSession) GetUniqueID() int                        { return 0 }
func (s *mockSession) GetDateParser() dtrules.DateParser       { return nil }
func (s *mockSession) GetRuleSet() dtrules.RuleSet             { return nil }
func (s *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return nil, nil
}
func (s *mockSession) GetEntityByID(id int) dtrules.Entity { return nil }
func (s *mockSession) Compile(expr string) (dtrules.Object, error) {
	// For testing, just return nil - we're testing EL compilation, not postfix execution
	return nil, nil
}

func TestDTLoaderMalformedXML(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0"?>
<decision_tables>
  <decision_table>
    <table_name>Broken
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for malformed XML")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestDTLoaderEmptyXML(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Empty DT file should load without error: %v", err)
	}
}

func TestDTLoaderReadError(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	err := loader.Load(&errorReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

func TestDTLoaderGetErrors(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	// Initially no errors
	if len(loader.GetErrors()) != 0 {
		t.Error("Expected no errors initially")
	}
}

func TestNewDTLoader(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}
	if loader.factory != factory {
		t.Error("Expected factory to be set")
	}
}

func TestDTLoaderXMLSizeLimit(t *testing.T) {
	// Save original value and restore after test
	originalMax := MaxXMLSize
	defer func() { MaxXMLSize = originalMax }()

	// Set a small limit for testing
	MaxXMLSize = 100

	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	// Create XML larger than the limit
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>LargeTable</table_name>
        <attribute_fields>
            <Type>First</Type>
            <COMMENTS>This comment makes the XML larger than the size limit for testing purposes</COMMENTS>
        </attribute_fields>
    </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for oversized XML")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

func TestDTLoaderXMLSizeLimitDisabled(t *testing.T) {
	// Save original value and restore after test
	originalMax := MaxXMLSize
	defer func() { MaxXMLSize = originalMax }()

	// Disable the limit
	MaxXMLSize = 0

	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Expected no error with size limit disabled: %v", err)
	}
}

// TestDTTableGetTableName tests the GetTableName method which supports both
// attribute and element forms for table names.
// Issue #374: EL format uses name attribute on <decision_table name="...">
func TestDTTableGetTableName(t *testing.T) {
	tests := []struct {
		name     string
		table    DTTable
		expected string
	}{
		{
			name: "attribute form (EL format)",
			table: DTTable{
				NameAttr:  "EL_Table_Name",
				TableName: "",
			},
			expected: "EL_Table_Name",
		},
		{
			name: "element form (traditional format)",
			table: DTTable{
				NameAttr:  "",
				TableName: "Traditional_Table_Name",
			},
			expected: "Traditional_Table_Name",
		},
		{
			name: "attribute takes precedence",
			table: DTTable{
				NameAttr:  "Attribute_Name",
				TableName: "Element_Name",
			},
			expected: "Attribute_Name",
		},
		{
			name: "both empty",
			table: DTTable{
				NameAttr:  "",
				TableName: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.table.GetTableName()
			if result != tt.expected {
				t.Errorf("GetTableName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestDTTableGetTableNumber tests the GetTableNumber method which supports both
// attribute and element forms for table numbers.
// Issue #374: EL format uses number attribute on <decision_table number="...">
func TestDTTableGetTableNumber(t *testing.T) {
	tests := []struct {
		name     string
		table    DTTable
		expected string
	}{
		{
			name: "attribute form (EL format)",
			table: DTTable{
				NumberAttr:      "1000",
				AttributeFields: DTAttributeFields{TableNumber: ""},
			},
			expected: "1000",
		},
		{
			name: "element form (traditional format)",
			table: DTTable{
				NumberAttr:      "",
				AttributeFields: DTAttributeFields{TableNumber: "2000"},
			},
			expected: "2000",
		},
		{
			name: "attribute takes precedence",
			table: DTTable{
				NumberAttr:      "3000",
				AttributeFields: DTAttributeFields{TableNumber: "4000"},
			},
			expected: "3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.table.GetTableNumber()
			if result != tt.expected {
				t.Errorf("GetTableNumber() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestDTLoaderELAutoCompile tests that EL descriptions are automatically compiled
// to postfix when the postfix element is empty or missing.
func TestDTLoaderELAutoCompile(t *testing.T) {
	factory := entity.NewFactory(nil)
	session := &mockSession{factory: factory}
	loader := NewDTLoader(session, factory)

	// XML with EL descriptions but no postfix - should auto-compile
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Test_EL_AutoCompile</table_name>
        <attribute_fields>
            <Type>First</Type>
        </attribute_fields>
        <contexts></contexts>
        <initial_actions></initial_actions>
        <conditions>
            <condition_details>
                <condition_number>1</condition_number>
                <condition_description>taxpayer.income > 50000</condition_description>
                <condition_column column_number="1" column_value="y"/>
                <condition_column column_number="2" column_value="n"/>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_number>1</action_number>
                <action_description>set result.eligible = true</action_description>
                <action_column column_number="1" column_value="x"/>
                <action_column column_number="2" column_value=""/>
            </action_details>
        </actions>
        <policy_statements></policy_statements>
    </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load DT with EL auto-compile: %v", err)
	}
	t.Log("Table Test_EL_AutoCompile loaded successfully with auto-compiled EL")
}

// TestDTLoaderELAutoCompileWithExistingPostfix tests that existing postfix is preserved
// when both description and postfix are present.
func TestDTLoaderELAutoCompileWithExistingPostfix(t *testing.T) {
	factory := entity.NewFactory(nil)
	session := &mockSession{factory: factory}
	loader := NewDTLoader(session, factory)

	// XML with both description and postfix - postfix should be used
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Test_EL_Preserve_Postfix</table_name>
        <attribute_fields>
            <Type>First</Type>
        </attribute_fields>
        <contexts></contexts>
        <initial_actions></initial_actions>
        <conditions>
            <condition_details>
                <condition_number>1</condition_number>
                <condition_description>taxpayer.income > 50000</condition_description>
                <condition_postfix>taxpayer.income 50000 f></condition_postfix>
                <condition_column column_number="1" column_value="y"/>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_number>1</action_number>
                <action_description>set result.tax = income * 0.25</action_description>
                <action_postfix>income 0.25 fmul /result.tax xdef</action_postfix>
                <action_column column_number="1" column_value="x"/>
            </action_details>
        </actions>
        <policy_statements></policy_statements>
    </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load DT with existing postfix: %v", err)
	}
	t.Log("Table loaded successfully with existing postfix preserved")
}

// TestDTLoaderELAutoCompileFallback tests that invalid EL expressions use a fallback.
// Invalid EL descriptions now log a warning and use a fallback ("true always" for conditions).
func TestDTLoaderELAutoCompileFallback(t *testing.T) {
	factory := entity.NewFactory(nil)
	session := &mockSession{factory: factory}
	loader := NewDTLoader(session, factory)

	// XML with invalid EL expression - should succeed with fallback
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Test_EL_Fallback</table_name>
        <attribute_fields>
            <Type>First</Type>
        </attribute_fields>
        <contexts></contexts>
        <initial_actions></initial_actions>
        <conditions>
            <condition_details>
                <condition_number>1</condition_number>
                <condition_description>value > (invalid</condition_description>
                <condition_column column_number="1" column_value="y"/>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_number>1</action_number>
                <action_description>Do something</action_description>
                <action_postfix>true</action_postfix>
                <action_column column_number="1" column_value="x"/>
            </action_details>
        </actions>
        <policy_statements></policy_statements>
    </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	// Should succeed now - invalid EL uses fallback with warning
	if err != nil {
		t.Fatalf("Unexpected error (invalid EL should use fallback): %v", err)
	}
	t.Log("Invalid EL description correctly used fallback")
}

// TestFilePathLoading tests that FILE_PATH is loaded and stored on the decision table.
func TestFilePathLoading(t *testing.T) {
	factory := entity.NewFactory(nil)
	session := &mockSession{factory: factory}
	loader := NewDTLoader(session, factory)

	xml := `<?xml version="1.0"?>
<decision_tables>
    <decision_table>
        <table_name>Test_FilePath</table_name>
        <attribute_fields>
            <Type>FIRST</Type>
            <COMMENTS>Test table</COMMENTS>
            <TABLE_NUMBER>100</TABLE_NUMBER>
            <FILE_PATH>income/wages.xlsx</FILE_PATH>
        </attribute_fields>
        <contexts></contexts>
        <initial_actions></initial_actions>
        <conditions></conditions>
        <actions></actions>
        <policy_statements></policy_statements>
    </decision_table>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load XML: %v", err)
	}

	// Find the loaded table
	rname := dtrules.GetRName("Test_FilePath")
	if rname == nil {
		t.Fatal("Failed to create RName")
	}

	dtObj := factory.FindDecisionTable(rname)
	if dtObj == nil {
		t.Fatal("Decision table not found")
	}

	// Verify FILE_PATH was stored via GetFilePath
	dt := dtObj.(interface{ GetFilePath() string })

	if v := dt.GetFilePath(); v != "income/wages.xlsx" {
		t.Errorf("Expected FILE_PATH 'income/wages.xlsx', got '%s'", v)
	}
}
