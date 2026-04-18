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

package ruleset

import (
	"bytes"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func TestNewCompiledRuleSet(t *testing.T) {
	rs := NewCompiledRuleSet("test-ruleset")

	if rs.Name != "test-ruleset" {
		t.Errorf("expected name 'test-ruleset', got '%s'", rs.Name)
	}
	if rs.Tables == nil {
		t.Error("Tables should not be nil")
	}
	if rs.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if len(rs.EntityDefs) != 0 {
		t.Error("EntityDefs should be empty")
	}
}

func TestCompiledRuleSetAddTable(t *testing.T) {
	rs := NewCompiledRuleSet("test")

	table := NewCompiledTable("MyTable", TableTypeBalanced)
	rs.AddTable(table)

	if len(rs.Tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(rs.Tables))
	}

	retrieved := rs.GetTable("MyTable")
	if retrieved == nil {
		t.Error("GetTable returned nil")
	}
	if retrieved.Name != "MyTable" {
		t.Errorf("expected name 'MyTable', got '%s'", retrieved.Name)
	}
}

func TestCompiledRuleSetGetTableNotFound(t *testing.T) {
	rs := NewCompiledRuleSet("test")

	retrieved := rs.GetTable("NonExistent")
	if retrieved != nil {
		t.Error("expected nil for non-existent table")
	}
}

func TestCompiledRuleSetTableNames(t *testing.T) {
	rs := NewCompiledRuleSet("test")
	rs.AddTable(NewCompiledTable("Table1", TableTypeFirst))
	rs.AddTable(NewCompiledTable("Table2", TableTypeAll))
	rs.AddTable(NewCompiledTable("Table3", TableTypeBalanced))

	names := rs.TableNames()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Names should contain all table names (order not guaranteed)
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["Table1"] || !nameSet["Table2"] || !nameSet["Table3"] {
		t.Error("missing expected table names")
	}
}

func TestTableTypeString(t *testing.T) {
	tests := []struct {
		typ      TableType
		expected string
	}{
		{TableTypeBalanced, "BALANCED"},
		{TableTypeFirst, "FIRST"},
		{TableTypeAll, "ALL"},
		{TableType(99), "UNKNOWN"},
	}

	for _, tc := range tests {
		if tc.typ.String() != tc.expected {
			t.Errorf("expected '%s', got '%s'", tc.expected, tc.typ.String())
		}
	}
}

func TestNewCompiledTable(t *testing.T) {
	table := NewCompiledTable("TestTable", TableTypeFirst)

	if table.Name != "TestTable" {
		t.Errorf("expected name 'TestTable', got '%s'", table.Name)
	}
	if table.Type != TableTypeFirst {
		t.Errorf("expected type FIRST, got %v", table.Type)
	}
	if table.InitialActions == nil {
		t.Error("InitialActions should not be nil")
	}
	if table.Conditions == nil {
		t.Error("Conditions should not be nil")
	}
	if table.Actions == nil {
		t.Error("Actions should not be nil")
	}
}

func TestNewConditionNode(t *testing.T) {
	node := NewConditionNode(5)

	if node.Type != NodeTypeCondition {
		t.Errorf("expected NodeTypeCondition, got %v", node.Type)
	}
	if node.ConditionIndex != 5 {
		t.Errorf("expected ConditionIndex 5, got %d", node.ConditionIndex)
	}
	if node.TrueBranch != nil {
		t.Error("TrueBranch should be nil")
	}
	if node.FalseBranch != nil {
		t.Error("FalseBranch should be nil")
	}
}

func TestNewActionNode(t *testing.T) {
	actions := []int{1, 3, 5}
	node := NewActionNode(actions, 2)

	if node.Type != NodeTypeAction {
		t.Errorf("expected NodeTypeAction, got %v", node.Type)
	}
	if node.Column != 2 {
		t.Errorf("expected Column 2, got %d", node.Column)
	}
	if len(node.ActionIndices) != 3 {
		t.Errorf("expected 3 action indices, got %d", len(node.ActionIndices))
	}

	// Modify original to ensure copy was made
	actions[0] = 99
	if node.ActionIndices[0] != 1 {
		t.Error("ActionIndices should be a copy")
	}
}

func TestCompiledRuleSetSerializeDeserialize(t *testing.T) {
	// Create a ruleset
	rs := NewCompiledRuleSet("test-ruleset")
	rs.Version = "1.0"
	rs.Metadata["author"] = "test"

	// Add entity definition
	rs.EntityDefs = append(rs.EntityDefs, EntityDefinition{
		Name:     "Customer",
		Readonly: false,
		Attributes: []AttributeDefinition{
			{Name: "name", Type: AttrTypeString, Input: true},
			{Name: "age", Type: AttrTypeInteger, Output: true},
		},
	})

	// Add a table with bytecode
	table := NewCompiledTable("EligibilityCheck", TableTypeBalanced)

	// Add a simple condition
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(18))
	bc.EmitPushName(dtrules.GetRName("age"))
	bc.Emit(dtrules.OpLookup)
	bc.Emit(dtrules.OpGe)
	table.Conditions = append(table.Conditions, bc)

	// Add a simple action
	bcAction := dtrules.NewBytecodeChunk()
	bcAction.Emit(dtrules.OpPushTrue)
	bcAction.EmitPushName(dtrules.GetRName("eligible"))
	bcAction.Emit(dtrules.OpDef)
	table.Actions = append(table.Actions, bcAction)

	// Add decision tree
	condNode := NewConditionNode(0)
	condNode.TrueBranch = NewActionNode([]int{0}, 0)
	condNode.FalseBranch = NewActionNode([]int{}, 1)
	table.DecisionTree = condNode

	rs.AddTable(table)

	// Serialize
	var buf bytes.Buffer
	if err := rs.Serialize(&buf); err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Deserialize
	rs2, err := Deserialize(&buf)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify
	if rs2.Name != rs.Name {
		t.Errorf("name mismatch: expected '%s', got '%s'", rs.Name, rs2.Name)
	}
	if rs2.Version != rs.Version {
		t.Errorf("version mismatch: expected '%s', got '%s'", rs.Version, rs2.Version)
	}
	if rs2.Metadata["author"] != "test" {
		t.Errorf("metadata mismatch: expected 'test', got '%s'", rs2.Metadata["author"])
	}

	// Verify entity definitions
	if len(rs2.EntityDefs) != 1 {
		t.Fatalf("expected 1 entity def, got %d", len(rs2.EntityDefs))
	}
	if rs2.EntityDefs[0].Name != "Customer" {
		t.Errorf("entity name mismatch")
	}
	if len(rs2.EntityDefs[0].Attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(rs2.EntityDefs[0].Attributes))
	}

	// Verify table
	table2 := rs2.GetTable("EligibilityCheck")
	if table2 == nil {
		t.Fatal("table not found after deserialization")
	}
	if table2.Type != TableTypeBalanced {
		t.Errorf("table type mismatch")
	}
	if len(table2.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(table2.Conditions))
	}
	if len(table2.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(table2.Actions))
	}

	// Verify decision tree
	if table2.DecisionTree == nil {
		t.Fatal("decision tree not deserialized")
	}
	if table2.DecisionTree.Type != NodeTypeCondition {
		t.Error("root should be condition node")
	}
	if table2.DecisionTree.TrueBranch == nil || table2.DecisionTree.TrueBranch.Type != NodeTypeAction {
		t.Error("true branch should be action node")
	}
}

func TestDeserializeInvalidMagic(t *testing.T) {
	buf := bytes.NewBuffer([]byte("XXXX"))
	_, err := Deserialize(buf)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestDeserializeTooShort(t *testing.T) {
	buf := bytes.NewBuffer([]byte("DT"))
	_, err := Deserialize(buf)
	if err == nil {
		t.Error("expected error for too short input")
	}
}

func TestAttributeTypes(t *testing.T) {
	types := []struct {
		typ  AttributeType
		name string
	}{
		{AttrTypeUnknown, "unknown"},
		{AttrTypeInteger, "integer"},
		{AttrTypeDouble, "double"},
		{AttrTypeString, "string"},
		{AttrTypeBoolean, "boolean"},
		{AttrTypeDate, "date"},
		{AttrTypeArray, "array"},
		{AttrTypeEntity, "entity"},
		{AttrTypeName, "name"},
	}

	// Just verify the constants have distinct values
	seen := make(map[AttributeType]bool)
	for _, tc := range types {
		if seen[tc.typ] {
			t.Errorf("duplicate type value: %v", tc.typ)
		}
		seen[tc.typ] = true
	}
}
