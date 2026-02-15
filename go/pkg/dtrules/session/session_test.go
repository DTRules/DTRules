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
	"strings"
	"testing"
	"time"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
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

func TestLoadEDDJSON(t *testing.T) {
	rs := NewRuleSet("test")

	jsonData := `{
		"entities": [
			{
				"name": "customer",
				"access": "rw",
				"fields": [
					{"name": "name", "type": "string", "access": "rw"},
					{"name": "age", "type": "integer", "access": "rw"},
					{"name": "balance", "type": "double", "access": "rw"},
					{"name": "active", "type": "boolean", "access": "rw"}
				]
			}
		]
	}`

	err := rs.LoadEDDJSON(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("LoadEDDJSON failed: %v", err)
	}

	// Verify entity was created
	names := rs.GetEntityNames()
	found := false
	for _, name := range names {
		if name.StringValue() == "customer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'customer' entity to be created via JSON")
	}
}

func TestLoadEDDJSONMultipleEntities(t *testing.T) {
	rs := NewRuleSet("test")

	jsonData := `{
		"entities": [
			{
				"name": "order",
				"access": "rw",
				"fields": [
					{"name": "order_id", "type": "integer", "access": "rw"},
					{"name": "total", "type": "double", "access": "rw"}
				]
			},
			{
				"name": "item",
				"access": "rw",
				"fields": [
					{"name": "item_id", "type": "integer", "access": "rw"},
					{"name": "description", "type": "string", "access": "rw"}
				]
			}
		]
	}`

	err := rs.LoadEDDJSON(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("LoadEDDJSON failed: %v", err)
	}

	names := rs.GetEntityNames()
	if len(names) < 2 {
		t.Errorf("Expected at least 2 entities, got %d", len(names))
	}

	entityNames := make(map[string]bool)
	for _, name := range names {
		entityNames[name.StringValue()] = true
	}
	if !entityNames["order"] {
		t.Error("Expected 'order' entity")
	}
	if !entityNames["item"] {
		t.Error("Expected 'item' entity")
	}
}

func TestLoadEDDJSONInvalid(t *testing.T) {
	rs := NewRuleSet("test")

	err := rs.LoadEDDJSON(strings.NewReader(`{invalid json`))
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestLoadEDDJSONEmpty(t *testing.T) {
	rs := NewRuleSet("test")

	err := rs.LoadEDDJSON(strings.NewReader(`{"entities": []}`))
	if err != nil {
		t.Fatalf("Empty JSON EDD should succeed: %v", err)
	}
}

func TestLoadEDDJSONThenCreateSession(t *testing.T) {
	rs := NewRuleSet("test")

	jsonData := `{
		"entities": [
			{
				"name": "widget",
				"access": "rw",
				"fields": [
					{"name": "label", "type": "string", "access": "rw"},
					{"name": "count", "type": "integer", "access": "rw"}
				]
			}
		]
	}`

	err := rs.LoadEDDJSON(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("LoadEDDJSON failed: %v", err)
	}

	// Create session and entity from JSON-loaded EDD
	session, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	rsession := session.(*RSession)
	entity, err := rsession.CreateEntity(dtrules.GetRName("widget"))
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}
	if entity == nil {
		t.Fatal("Entity is nil")
	}

	// Set and get a value
	err = entity.Put(dtrules.GetRName("label"), dtrules.NewRString("test-widget"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, err := entity.Get(dtrules.GetRName("label"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value.StringValue() != "test-widget" {
		t.Errorf("Expected 'test-widget', got %s", value.StringValue())
	}
}

func TestLoadEDDJSONMatchesXMLEntities(t *testing.T) {
	// Load same entity structure via both XML and JSON, compare
	rsXML := NewRuleSet("xml")
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="person" access="rw" comment="">
		<field name="name" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="age" type="integer" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
</entity_data_dictionary>`

	if err := rsXML.LoadEDD(strings.NewReader(eddXML)); err != nil {
		t.Fatalf("LoadEDD failed: %v", err)
	}

	rsJSON := NewRuleSet("json")
	eddJSON := `{
		"entities": [
			{
				"name": "person",
				"access": "rw",
				"fields": [
					{"name": "name", "type": "string", "access": "rw"},
					{"name": "age", "type": "integer", "access": "rw", "defaultValue": "0"}
				]
			}
		]
	}`

	if err := rsJSON.LoadEDDJSON(strings.NewReader(eddJSON)); err != nil {
		t.Fatalf("LoadEDDJSON failed: %v", err)
	}

	xmlNames := rsXML.GetEntityNames()
	jsonNames := rsJSON.GetEntityNames()

	if len(xmlNames) != len(jsonNames) {
		t.Errorf("Entity count mismatch: XML=%d, JSON=%d", len(xmlNames), len(jsonNames))
	}

	// Both should have "person"
	xmlHasPerson := false
	jsonHasPerson := false
	for _, name := range xmlNames {
		if name.StringValue() == "person" {
			xmlHasPerson = true
		}
	}
	for _, name := range jsonNames {
		if name.StringValue() == "person" {
			jsonHasPerson = true
		}
	}
	if !xmlHasPerson || !jsonHasPerson {
		t.Error("Both XML and JSON should produce a 'person' entity")
	}
}

func TestNewRuleSetInvalidName(t *testing.T) {
	rs := NewRuleSet(".invalid")
	if rs != nil {
		t.Error("Expected nil for invalid rule set name")
	}
}
