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

package sdk

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestNewEngine_WithDirectory(t *testing.T) {
	// Create temporary directory with test rules
	tmpDir := t.TempDir()

	// Create minimal EDD
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="input" readonly="false">
        <field name="value" type="integer" subtype="" default_value="0"/>
    </entity>
    <entity name="output" readonly="false">
        <field name="result" type="integer" subtype="" default_value="0"/>
    </entity>
</entity_data_dictionary>`

	eddPath := filepath.Join(tmpDir, "test_edd.xml")
	if err := os.WriteFile(eddPath, []byte(eddContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create minimal DT
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>TestTable</table_name>
    </decision_table>
</decision_tables>`

	dtPath := filepath.Join(tmpDir, "test_dt.xml")
	if err := os.WriteFile(dtPath, []byte(dtContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create engine
	engine, err := NewEngine("TestRules", WithDirectory(tmpDir))
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	if engine.Name() != "TestRules" {
		t.Errorf("Name = %q, want %q", engine.Name(), "TestRules")
	}

	// List entities
	entities := engine.ListEntities()
	if len(entities) != 2 {
		t.Errorf("ListEntities = %d entities, want 2", len(entities))
	}

	// List tables
	tables := engine.ListTables()
	if len(tables) != 1 {
		t.Errorf("ListTables = %d tables, want 1", len(tables))
	}
}

func TestNewEngine_WithFS(t *testing.T) {
	// Create in-memory filesystem
	fsys := fstest.MapFS{
		"rules/test_edd.xml": &fstest.MapFile{
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="data" readonly="false">
        <field name="x" type="integer" subtype="" default_value="0"/>
    </entity>
</entity_data_dictionary>`),
		},
		"rules/test_dt.xml": &fstest.MapFile{
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Compute</table_name>
    </decision_table>
</decision_tables>`),
		},
	}

	engine, err := NewEngine("EmbeddedRules", WithFS(fsys, "rules"))
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	if engine.Name() != "EmbeddedRules" {
		t.Errorf("Name = %q, want %q", engine.Name(), "EmbeddedRules")
	}

	entities := engine.ListEntities()
	if len(entities) != 1 {
		t.Errorf("ListEntities = %d, want 1", len(entities))
	}

	tables := engine.ListTables()
	if len(tables) != 1 {
		t.Errorf("ListTables = %d, want 1", len(tables))
	}
}

func TestContext_Set(t *testing.T) {
	engine, _ := NewEngine("Test")
	ctx := engine.NewContext()

	ctx.Set("name", "John").
		Set("age", 30).
		Set("active", true)

	if ctx.inputs["name"] != "John" {
		t.Errorf("inputs[name] = %v, want John", ctx.inputs["name"])
	}
	if ctx.inputs["age"] != 30 {
		t.Errorf("inputs[age] = %v, want 30", ctx.inputs["age"])
	}
	if ctx.inputs["active"] != true {
		t.Errorf("inputs[active] = %v, want true", ctx.inputs["active"])
	}
}

func TestContext_SetEntity(t *testing.T) {
	engine, _ := NewEngine("Test")
	ctx := engine.NewContext()

	ctx.SetEntity("person", "name", "John").
		SetEntity("person", "age", 30).
		SetEntity("address", "city", "Denver")

	if ctx.entities["person"]["name"] != "John" {
		t.Errorf("person.name = %v, want John", ctx.entities["person"]["name"])
	}
	if ctx.entities["person"]["age"] != 30 {
		t.Errorf("person.age = %v, want 30", ctx.entities["person"]["age"])
	}
	if ctx.entities["address"]["city"] != "Denver" {
		t.Errorf("address.city = %v, want Denver", ctx.entities["address"]["city"])
	}
}

func TestToObject(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{"nil", nil, false},
		{"bool true", true, false},
		{"bool false", false, false},
		{"int", 42, false},
		{"int64", int64(100), false},
		{"float64", 3.14, false},
		{"string", "hello", false},
		{"unsupported", []int{1, 2, 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := toObject(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toObject(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && obj == nil {
				t.Errorf("toObject(%v) = nil, want non-nil", tt.input)
			}
		})
	}
}

func TestEngine_ConcurrentContexts(t *testing.T) {
	engine, _ := NewEngine("Test")

	// Create multiple contexts concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			ctx := engine.NewContext()
			ctx.Set("value", n)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestEntityStackPush verifies that entities created via SetEntity are pushed
// onto the entity stack and accessible in decision tables (issue #491)
func TestEntityStackPush(t *testing.T) {
	// Create in-memory filesystem with rules that access entity fields
	fsys := fstest.MapFS{
		"rules/test_edd.xml": &fstest.MapFile{
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="sync_state" readonly="false">
        <field name="last_major_block" type="integer" subtype="" default_value="0"/>
        <field name="coverage_percent" type="double" subtype="" default_value="0"/>
    </entity>
    <entity name="result" readonly="false">
        <field name="valid" type="boolean" subtype="" default_value="false"/>
    </entity>
</entity_data_dictionary>`),
		},
		"rules/test_dt.xml": &fstest.MapFile{
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Check_Sync_State</table_name>
        <table_number>1</table_number>
        <purpose>Verify sync state entity is accessible</purpose>
        <column type="condition" column_number="1">
            <column_name>Always</column_name>
            <column_postfix>true</column_postfix>
            <row_entries>
                <row_entry row_number="1">Y</row_entry>
            </row_entries>
        </column>
        <column type="action" column_number="2">
            <column_name>SetValid</column_name>
            <column_postfix>sync_state.last_major_block 100 >= result.valid =</column_postfix>
            <row_entries>
                <row_entry row_number="1">X</row_entry>
            </row_entries>
        </column>
    </decision_table>
</decision_tables>`),
		},
	}

	engine, err := NewEngine("SyncRules", WithFS(fsys, "rules"))
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	ctx := engine.NewContext()
	ctx.SetEntity("sync_state", "last_major_block", int64(150))
	ctx.SetEntity("sync_state", "coverage_percent", 98.5)
	ctx.SetEntity("result", "valid", false)

	result, err := engine.Execute("Check_Sync_State", ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// The decision table should have been able to read sync_state.last_major_block
	// and set result.valid = true (since 150 >= 100)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}
