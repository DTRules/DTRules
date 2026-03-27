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

package session

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

func TestNewRuleSet(t *testing.T) {
	rs := NewRuleSet("test")
	if rs == nil {
		t.Fatal("NewRuleSet returned nil")
	}

	if rs.GetName().StringValue() != "test" {
		t.Errorf("Expected name 'test', got '%s'", rs.GetName().StringValue())
	}
}

func TestLoadEDD(t *testing.T) {
	rs := NewRuleSet("test")

	// EDD XML in the actual DTRules format
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="person" access="rw" comment="">
		<field name="person" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="name" type="string" subtype="" access="rw" input="" default_value="" comment="Person name"></field>
		<field name="age" type="integer" subtype="" access="rw" input="" default_value="0" comment="Person age"></field>
		<field name="active" type="boolean" subtype="" access="rw" input="" default_value="false" comment="Is active"></field>
	</entity>
</entity_data_dictionary>`

	err := rs.LoadEDD(strings.NewReader(eddXML))
	if err != nil {
		t.Fatalf("LoadEDD failed: %v", err)
	}

	// Verify entity was created
	names := rs.GetEntityNames()
	if len(names) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(names))
	}

	found := false
	for _, name := range names {
		if name.StringValue() == "person" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'person' entity to be created")
	}
}

func TestNewSession(t *testing.T) {
	rs := NewRuleSet("test")

	session, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if session == nil {
		t.Fatal("Session is nil")
	}

	state := session.GetState()
	if state == nil {
		t.Fatal("State is nil")
	}
}

func TestSessionCreateEntity(t *testing.T) {
	rs := NewRuleSet("test")

	// Load EDD in the actual DTRules format
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="item" access="rw" comment="">
		<field name="item" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="value" type="integer" subtype="" access="rw" input="" default_value="0" comment="Item value"></field>
	</entity>
</entity_data_dictionary>`
	rs.LoadEDD(strings.NewReader(eddXML))

	// Create session
	session, _ := rs.NewSession()
	rsession := session.(*RSession)

	// Create entity
	entity, err := rsession.CreateEntity(dtrules.GetRName("item"))
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}
	if entity == nil {
		t.Fatal("Entity is nil")
	}

	// Set value
	err = entity.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(42))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get value
	value, err := entity.Get(dtrules.GetRName("value"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	val, _ := value.IntValue()
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestSessionGetEntityByID(t *testing.T) {
	rs := NewRuleSet("test")

	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="test" access="rw" comment="">
		<field name="test" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="data" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
	</entity>
</entity_data_dictionary>`
	rs.LoadEDD(strings.NewReader(eddXML))

	session, _ := rs.NewSession()
	rsession := session.(*RSession)

	entity, _ := rsession.CreateEntity(dtrules.GetRName("test"))
	id := entity.GetID()

	// Retrieve by ID
	retrieved := rsession.GetEntityByID(id)
	if retrieved == nil {
		t.Fatal("GetEntityByID returned nil")
	}
	if retrieved != entity {
		t.Error("Retrieved entity is not the same instance")
	}
}

func TestSessionAttributes(t *testing.T) {
	rs := NewRuleSet("test")
	session, _ := rs.NewSession()
	rsession := session.(*RSession)

	// Set attribute
	rsession.SetAttribute("key", "value")

	// Get attribute
	val := rsession.GetAttribute("key")
	if val != "value" {
		t.Errorf("Expected 'value', got '%v'", val)
	}

	// Get non-existent attribute
	val = rsession.GetAttribute("nonexistent")
	if val != nil {
		t.Errorf("Expected nil for non-existent attribute, got %v", val)
	}
}

func TestDateParser(t *testing.T) {
	parser := NewDateParser()

	tests := []struct {
		input    string
		expected time.Time
	}{
		{"01/15/2024", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"1/5/2024", time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)},
		{"2024-01-15", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"Jan 15, 2024", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"January 15, 2024", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parser.GetDate(tt.input)
			if err != nil {
				t.Fatalf("GetDate failed for '%s': %v", tt.input, err)
			}
			if !result.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDateParserInvalid(t *testing.T) {
	parser := NewDateParser()

	_, err := parser.GetDate("invalid date")
	if err == nil {
		t.Error("Expected error for invalid date")
	}
}

func TestDateParserAddFormat(t *testing.T) {
	parser := NewDateParser()

	// Add custom format
	parser.AddFormat("02-Jan-2006")

	result, err := parser.GetDate("15-Jan-2024")
	if err != nil {
		t.Fatalf("GetDate failed: %v", err)
	}

	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestRuleSetAttributes(t *testing.T) {
	rs := NewRuleSet("test")

	rs.SetAttribute("version", "1.0")
	val := rs.GetAttribute("version")
	if val != "1.0" {
		t.Errorf("Expected '1.0', got '%v'", val)
	}
}

func TestRuleSetEntryPoints(t *testing.T) {
	rs := NewRuleSet("test")

	rs.AddEntryPoint("main", "MainDecisionTable")
	tableName := rs.GetEntryPoint("main")
	if tableName != "MainDecisionTable" {
		t.Errorf("Expected 'MainDecisionTable', got '%s'", tableName)
	}

	// Non-existent entry point
	tableName = rs.GetEntryPoint("nonexistent")
	if tableName != "" {
		t.Errorf("Expected empty string for non-existent entry point, got '%s'", tableName)
	}
}

func TestSessionUniqueID(t *testing.T) {
	rs := NewRuleSet("test")
	session, _ := rs.NewSession()
	rsession := session.(*RSession)

	id1 := rsession.GetUniqueID()
	id2 := rsession.GetUniqueID()

	if id1 == id2 {
		t.Error("Unique IDs should be different")
	}
	if id2 != id1+1 {
		t.Error("Unique IDs should be sequential")
	}
}

func TestSessionDateParser(t *testing.T) {
	rs := NewRuleSet("test")
	session, _ := rs.NewSession()
	rsession := session.(*RSession)

	parser := rsession.GetDateParser()
	if parser == nil {
		t.Fatal("DateParser is nil")
	}

	// Test parsing with the session's date parser
	result, err := parser.GetDate("01/01/2024")
	if err != nil {
		t.Fatalf("GetDate failed: %v", err)
	}

	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestSessionGetRuleSet(t *testing.T) {
	rs := NewRuleSet("myruleset")
	session, _ := rs.NewSession()

	retrievedRS := session.GetRuleSet()
	if retrievedRS == nil {
		t.Fatal("GetRuleSet returned nil")
	}

	if retrievedRS.(*RuleSet) != rs {
		t.Error("GetRuleSet returned different rule set")
	}
}

func TestLoadEDDFile(t *testing.T) {
	// Create a temporary EDD file
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="testentity" access="rw" comment="">
		<field name="testentity" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="value" type="integer" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
</entity_data_dictionary>`

	tmpFile := "/tmp/test_edd.xml"
	err := os.WriteFile(tmpFile, []byte(eddXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	rs := NewRuleSet("test")
	err = rs.LoadEDDFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadEDDFile failed: %v", err)
	}

	// Verify entity was loaded
	names := rs.GetEntityNames()
	found := false
	for _, name := range names {
		if name.StringValue() == "testentity" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'testentity' to be loaded")
	}
}

func TestLoadDecisionTablesFile(t *testing.T) {
	// Create a simple decision table
	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>TestTable</table_name>
		<attribute_fields>
			<TABLE_NUMBER>1000</TABLE_NUMBER>
		</attribute_fields>
		<initial_actions></initial_actions>
		<conditions></conditions>
		<actions></actions>
	</decision_table>
</decision_tables>`

	tmpFile := "/tmp/test_dt.xml"
	err := os.WriteFile(tmpFile, []byte(dtXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	rs := NewRuleSet("test")

	// Need minimal EDD for DT to work
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="result" access="rw" comment="">
		<field name="result" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
	</entity>
</entity_data_dictionary>`
	rs.LoadEDD(strings.NewReader(eddXML))

	err = rs.LoadDecisionTablesFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadDecisionTablesFile failed: %v", err)
	}

	// Verify DT was loaded
	dtNames := rs.GetDecisionTableNames()
	if len(dtNames) == 0 {
		t.Error("No decision tables loaded")
		return
	}
	found := false
	for _, name := range dtNames {
		if name.StringValue() == "TestTable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'TestTable' to be loaded")
	}
}

func TestLoadRulesFromFiles(t *testing.T) {
	// Create EDD file
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="item" access="rw" comment="">
		<field name="item" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="count" type="integer" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
</entity_data_dictionary>`

	eddFile := "/tmp/test_edd2.xml"
	err := os.WriteFile(eddFile, []byte(eddXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create EDD file: %v", err)
	}
	defer os.Remove(eddFile)

	// Create DT file
	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>TestTable2</table_name>
		<attribute_fields>
			<TABLE_NUMBER>1000</TABLE_NUMBER>
		</attribute_fields>
		<initial_actions></initial_actions>
		<conditions></conditions>
		<actions></actions>
	</decision_table>
</decision_tables>`

	dtFile := "/tmp/test_dt2.xml"
	err = os.WriteFile(dtFile, []byte(dtXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create DT file: %v", err)
	}
	defer os.Remove(dtFile)

	// Load using convenience function
	rs, err := LoadRulesFromFiles("test", eddFile, dtFile)
	if err != nil {
		t.Fatalf("LoadRulesFromFiles failed: %v", err)
	}

	if rs == nil {
		t.Fatal("LoadRulesFromFiles returned nil")
	}

	// Verify name
	if rs.GetName().StringValue() != "test" {
		t.Errorf("Expected name 'test', got '%s'", rs.GetName().StringValue())
	}

	// Verify entity loaded
	names := rs.GetEntityNames()
	foundEntity := false
	for _, name := range names {
		if name.StringValue() == "item" {
			foundEntity = true
			break
		}
	}
	if !foundEntity {
		t.Error("Expected 'item' entity to be loaded")
	}

	// Verify DT loaded
	dtNames := rs.GetDecisionTableNames()
	if len(dtNames) == 0 {
		t.Error("No decision tables loaded")
		return
	}
	foundDT := false
	for _, name := range dtNames {
		if name.StringValue() == "TestTable2" {
			foundDT = true
			break
		}
	}
	if !foundDT {
		t.Error("Expected 'TestTable2' to be loaded")
	}
}
