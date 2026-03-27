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
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
)

// TestEDDXMLAndJSONProduceIdenticalResults verifies that loading an EDD from XML
// produces identical results to loading the same EDD from JSON.
func TestEDDXMLAndJSONProduceIdenticalResults(t *testing.T) {
	// Test with an inline EDD definition
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="person" access="rw" comment="A person entity">
		<field name="name" type="string" access="rw" comment="Person name"/>
		<field name="age" type="integer" access="rw" default_value="0"/>
		<field name="active" type="boolean" access="r" default_value="true"/>
		<field name="score" type="double" access="rw"/>
		<field name="tags" type="array" subtype="string" access="rw"/>
	</entity>
	<entity name="order" access="rw">
		<field name="order_id" type="integer" access="r"/>
		<field name="total" type="double" access="rw"/>
		<field name="customer" type="entity" subtype="person" access="rw"/>
	</entity>
</entity_data_dictionary>`

	jsonData := `{
	"version": "2",
	"entities": [
		{
			"name": "person",
			"access": "rw",
			"comment": "A person entity",
			"fields": [
				{"name": "name", "type": "string", "access": "rw", "comment": "Person name"},
				{"name": "age", "type": "integer", "access": "rw", "default_value": "0"},
				{"name": "active", "type": "boolean", "access": "r", "default_value": "true"},
				{"name": "score", "type": "double", "access": "rw"},
				{"name": "tags", "type": "array", "subtype": "string", "access": "rw"}
			]
		},
		{
			"name": "order",
			"access": "rw",
			"fields": [
				{"name": "order_id", "type": "integer", "access": "r"},
				{"name": "total", "type": "double", "access": "rw"},
				{"name": "customer", "type": "entity", "subtype": "person", "access": "rw"}
			]
		}
	]
}`

	// Load from XML
	factoryXML := entity.NewFactory(nil)
	loaderXML := NewEDDLoader(nil, factoryXML)
	err := loaderXML.Load(strings.NewReader(xmlData))
	if err != nil {
		t.Fatalf("Failed to load XML EDD: %v", err)
	}

	// Load from JSON
	factoryJSON := entity.NewFactory(nil)
	loaderJSON := NewEDDLoader(nil, factoryJSON)
	err = loaderJSON.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	// Compare the two factories
	compareFactories(t, factoryXML, factoryJSON)
}

// TestFormatDetection verifies that the format detection works correctly.
func TestFormatDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
	}{
		{"XML with declaration", `<?xml version="1.0"?><root/>`, FormatXML},
		{"XML without declaration", `<root/>`, FormatXML},
		{"XML with whitespace", `   <root/>`, FormatXML},
		{"JSON object", `{"key": "value"}`, FormatJSON},
		{"JSON array", `[1, 2, 3]`, FormatJSON},
		{"JSON with whitespace", `   {"key": "value"}`, FormatJSON},
		{"XML with BOM", "\xEF\xBB\xBF<root/>", FormatXML},
		{"JSON with BOM", "\xEF\xBB\xBF{\"key\": \"value\"}", FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, _, err := DetectFormat(strings.NewReader(tt.input))
			if err != nil {
				t.Errorf("DetectFormat returned error: %v", err)
				return
			}
			if format != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, format)
			}
		})
	}
}

// TestCHIPEDDFilesXMLAndJSONIdentical tests that loading the CHIP EDD from XML and JSON
// produces identical entity definitions.
func TestCHIPEDDFilesXMLAndJSONIdentical(t *testing.T) {
	// Check if sample project files exist
	baseDir := findCHIPDir(t)
	if baseDir == "" {
		t.Skip("CHIP sample project not found, skipping integration test")
	}

	xmlDir := filepath.Join(baseDir, "xml")
	jsonDir := filepath.Join(baseDir, "json")

	// Check if JSON files exist
	if _, err := os.Stat(jsonDir); os.IsNotExist(err) {
		t.Skip("JSON directory not found, skipping integration test")
	}

	xmlPath := filepath.Join(xmlDir, "CHIP_edd.xml")
	jsonPath := filepath.Join(jsonDir, "CHIP_edd.json")

	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Skip("CHIP_edd.json not found")
	}

	testEDDFilesIdentical(t, xmlPath, jsonPath)
}

// findCHIPDir looks for the CHIP sample project directory.
func findCHIPDir(t *testing.T) string {
	// Try relative paths from test location
	possiblePaths := []string{
		"../../../../sampleprojects/CHIP",
		"../../../../../sampleprojects/CHIP",
		"../../../../../../sampleprojects/CHIP",
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	return ""
}

// testEDDFilesIdentical loads EDD from both files and compares them.
func testEDDFilesIdentical(t *testing.T, xmlPath, jsonPath string) {
	// Load XML
	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		t.Fatalf("Failed to open XML file: %v", err)
	}
	defer xmlFile.Close()

	factoryXML := entity.NewFactory(nil)
	loaderXML := NewEDDLoader(nil, factoryXML)
	if err := loaderXML.Load(xmlFile); err != nil {
		t.Fatalf("Failed to load XML EDD: %v", err)
	}

	// Load JSON
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		t.Fatalf("Failed to open JSON file: %v", err)
	}
	defer jsonFile.Close()

	factoryJSON := entity.NewFactory(nil)
	loaderJSON := NewEDDLoader(nil, factoryJSON)
	if err := loaderJSON.Load(jsonFile); err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	// Compare
	compareFactories(t, factoryXML, factoryJSON)
}

// compareFactories compares two entity factories to ensure they have the same entities and attributes.
func compareFactories(t *testing.T, f1, f2 *entity.Factory) {
	t.Helper()

	// Get all entity names from both factories
	names1 := f1.GetEntityNames()
	names2 := f2.GetEntityNames()

	if len(names1) != len(names2) {
		t.Errorf("Entity count differs: %d vs %d", len(names1), len(names2))
	}

	// Compare each entity
	for _, name := range names1 {
		e1, err1 := f1.GetReferenceEntity(name)
		e2, err2 := f2.GetReferenceEntity(name)

		if err1 != nil || err2 != nil {
			t.Errorf("Failed to get entity %s: err1=%v, err2=%v", name.StringValue(), err1, err2)
			continue
		}

		if e1 == nil && e2 == nil {
			continue
		}
		if e1 == nil || e2 == nil {
			t.Errorf("Entity %s: one is nil, other is not", name.StringValue())
			continue
		}

		// Compare attribute counts
		attrs1 := e1.GetAttributeNames()
		attrs2 := e2.GetAttributeNames()

		if len(attrs1) != len(attrs2) {
			t.Errorf("Entity %s: attribute count differs: %d vs %d",
				name.StringValue(), len(attrs1), len(attrs2))
		}

		// Compare each attribute exists
		for _, attrName := range attrs1 {
			if !e2.ContainsAttribute(attrName) {
				t.Errorf("Entity %s: attribute %s missing in second factory",
					name.StringValue(), attrName.StringValue())
			}
		}
	}
}

// TestEDDStructXMLJSONRoundTrip verifies that EDD structures can round-trip through XML and JSON.
func TestEDDStructXMLJSONRoundTrip(t *testing.T) {
	original := EDDFile{
		Version: "2",
		Entities: []EDDEntity{
			{
				Name:    "test",
				Access:  "rw",
				Comment: "Test entity",
				Fields: []EDDField{
					{Name: "field1", Type: "string", Access: "rw"},
					{Name: "field2", Type: "integer", DefaultValue: "42"},
				},
			},
		},
	}

	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to XML: %v", err)
	}

	// Unmarshal from XML
	var fromXML EDDFile
	if err := xml.Unmarshal(xmlBytes, &fromXML); err != nil {
		t.Fatalf("Failed to unmarshal from XML: %v", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Unmarshal from JSON
	var fromJSON EDDFile
	if err := json.Unmarshal(jsonBytes, &fromJSON); err != nil {
		t.Fatalf("Failed to unmarshal from JSON: %v", err)
	}

	// Compare
	if fromXML.Version != fromJSON.Version {
		t.Errorf("Version differs: XML=%s, JSON=%s", fromXML.Version, fromJSON.Version)
	}

	if len(fromXML.Entities) != len(fromJSON.Entities) {
		t.Errorf("Entity count differs: XML=%d, JSON=%d", len(fromXML.Entities), len(fromJSON.Entities))
	}

	for i := range fromXML.Entities {
		if fromXML.Entities[i].Name != fromJSON.Entities[i].Name {
			t.Errorf("Entity %d name differs: XML=%s, JSON=%s",
				i, fromXML.Entities[i].Name, fromJSON.Entities[i].Name)
		}

		if len(fromXML.Entities[i].Fields) != len(fromJSON.Entities[i].Fields) {
			t.Errorf("Entity %s field count differs: XML=%d, JSON=%d",
				fromXML.Entities[i].Name,
				len(fromXML.Entities[i].Fields),
				len(fromJSON.Entities[i].Fields))
		}
	}
}

// TestDTStructXMLJSONRoundTrip verifies that DT structures can round-trip through XML and JSON.
func TestDTStructXMLJSONRoundTrip(t *testing.T) {
	original := DTFile{
		Tables: []DTTable{
			{
				TableName: "Test_Table",
				XlsFile:   "test.xls",
				AttributeFields: DTAttributeFields{
					Type:        "FIRST",
					Comments:    "Test comment",
					FileName:    "test.xls",
					TableNumber: "100",
				},
				Conditions: DTConditions{
					Conditions: []DTConditionDetail{
						{
							Number:      1,
							Comment:     "Test condition",
							Description: "x == y",
							Postfix:     "x y eq ",
							Columns: []DTConditionCol{
								{ColumnNumber: 1, ColumnValue: "Y"},
							},
						},
					},
				},
				Actions: DTActions{
					Actions: []DTActionDetail{
						{
							Number:      1,
							Comment:     "Test action",
							Description: "do something",
							Postfix:     "something ",
							Columns: []DTActionCol{
								{ColumnNumber: 1, ColumnValue: "X"},
							},
						},
					},
				},
			},
		},
	}

	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to XML: %v", err)
	}

	// Unmarshal from XML
	var fromXML DTFile
	if err := xml.Unmarshal(xmlBytes, &fromXML); err != nil {
		t.Fatalf("Failed to unmarshal from XML: %v", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Unmarshal from JSON
	var fromJSON DTFile
	if err := json.Unmarshal(jsonBytes, &fromJSON); err != nil {
		t.Fatalf("Failed to unmarshal from JSON: %v", err)
	}

	// Compare
	if len(fromXML.Tables) != len(fromJSON.Tables) {
		t.Errorf("Table count differs: XML=%d, JSON=%d", len(fromXML.Tables), len(fromJSON.Tables))
	}

	for i := range fromXML.Tables {
		if fromXML.Tables[i].TableName != fromJSON.Tables[i].TableName {
			t.Errorf("Table %d name differs: XML=%s, JSON=%s",
				i, fromXML.Tables[i].TableName, fromJSON.Tables[i].TableName)
		}

		if fromXML.Tables[i].AttributeFields.Type != fromJSON.Tables[i].AttributeFields.Type {
			t.Errorf("Table %s type differs: XML=%s, JSON=%s",
				fromXML.Tables[i].TableName,
				fromXML.Tables[i].AttributeFields.Type,
				fromJSON.Tables[i].AttributeFields.Type)
		}

		// Compare conditions
		if len(fromXML.Tables[i].Conditions.Conditions) != len(fromJSON.Tables[i].Conditions.Conditions) {
			t.Errorf("Table %s condition count differs: XML=%d, JSON=%d",
				fromXML.Tables[i].TableName,
				len(fromXML.Tables[i].Conditions.Conditions),
				len(fromJSON.Tables[i].Conditions.Conditions))
		}

		// Compare actions
		if len(fromXML.Tables[i].Actions.Actions) != len(fromJSON.Tables[i].Actions.Actions) {
			t.Errorf("Table %s action count differs: XML=%d, JSON=%d",
				fromXML.Tables[i].TableName,
				len(fromXML.Tables[i].Actions.Actions),
				len(fromJSON.Tables[i].Actions.Actions))
		}
	}
}

// TestMappingJSONAndXMLProduceIdenticalResults tests the mapping structures.
func TestMappingJSONStructures(t *testing.T) {
	// This tests the JSON mapping structures only (mapping package has its own tests)
	// Test that JSON map file can be parsed correctly
	jsonPath := findCHIPDir(t)
	if jsonPath == "" {
		t.Skip("CHIP sample project not found")
	}

	mapPath := filepath.Join(jsonPath, "json", "CHIP_map.json")
	if _, err := os.Stat(mapPath); os.IsNotExist(err) {
		t.Skip("CHIP_map.json not found")
	}

	// Just verify the file is valid JSON
	data, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatalf("Failed to read map file: %v", err)
	}

	var mapData map[string]interface{}
	if err := json.Unmarshal(data, &mapData); err != nil {
		t.Fatalf("Failed to parse map JSON: %v", err)
	}

	// Verify expected fields exist
	expectedFields := []string{"setattributes", "createentities", "entities", "initialentities"}
	for _, field := range expectedFields {
		if _, ok := mapData[field]; !ok {
			t.Errorf("Expected field %s not found in map JSON", field)
		}
	}
}

// TestFormatString verifies the Format.String() method.
func TestFormatString(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatXML, "XML"},
		{FormatJSON, "JSON"},
		{FormatUnknown, "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.format.String(); got != tt.expected {
			t.Errorf("Format(%d).String() = %s, want %s", tt.format, got, tt.expected)
		}
	}
}

// TestEDDAutoDetectFormat verifies that the EDD loader auto-detects format correctly.
func TestEDDAutoDetectFormat(t *testing.T) {
	xmlData := `<entity_data_dictionary version="1">
		<entity name="test" access="rw">
			<field name="value" type="string" access="rw"/>
		</entity>
	</entity_data_dictionary>`

	jsonData := `{"version":"1","entities":[{"name":"test","access":"rw","fields":[{"name":"value","type":"string","access":"rw"}]}]}`

	// Test XML auto-detection
	factoryXML := entity.NewFactory(nil)
	loaderXML := NewEDDLoader(nil, factoryXML)
	if err := loaderXML.Load(strings.NewReader(xmlData)); err != nil {
		t.Fatalf("Failed to load XML EDD: %v", err)
	}

	testName := dtrules.GetRName("test")
	if _, err := factoryXML.GetReferenceEntity(testName); err != nil {
		t.Error("Failed to get entity from XML-loaded factory")
	}

	// Test JSON auto-detection
	factoryJSON := entity.NewFactory(nil)
	loaderJSON := NewEDDLoader(nil, factoryJSON)
	if err := loaderJSON.Load(strings.NewReader(jsonData)); err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	if _, err := factoryJSON.GetReferenceEntity(testName); err != nil {
		t.Error("Failed to get entity from JSON-loaded factory")
	}
}
