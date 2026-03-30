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

	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

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
