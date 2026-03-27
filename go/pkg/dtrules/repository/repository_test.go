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

package repository

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules/ruleset"
)

func TestNew(t *testing.T) {
	repo := New()
	if repo == nil {
		t.Fatal("New() returned nil")
	}
	if repo.strictMode {
		t.Error("expected strictMode false")
	}
}

func TestNewStrict(t *testing.T) {
	repo := NewStrict()
	if repo == nil {
		t.Fatal("NewStrict() returned nil")
	}
	if !repo.strictMode {
		t.Error("expected strictMode true")
	}
}

func TestLoadFromXML(t *testing.T) {
	repo := New()

	edd := `<?xml version="1.0"?>
<edd name="test">
	<entity entityname="Customer" readonly="false">
		<field fieldname="name" type="string" input="true"/>
		<field fieldname="age" type="integer" output="true"/>
	</entity>
</edd>`

	dt := `<?xml version="1.0"?>
<decision_tables name="test">
	<decision_table name="CheckAge" type="BALANCED">
		<conditions>
			<item num="1">/age 18 >=</item>
		</conditions>
		<actions>
			<item num="1">true /eligible def</item>
		</actions>
	</decision_table>
</decision_tables>`

	rs, err := repo.LoadFromXML(strings.NewReader(edd), strings.NewReader(dt))
	if err != nil {
		t.Fatalf("LoadFromXML failed: %v", err)
	}

	if rs.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", rs.Name)
	}

	// Check entity definitions
	if len(rs.EntityDefs) != 1 {
		t.Fatalf("expected 1 entity def, got %d", len(rs.EntityDefs))
	}
	if rs.EntityDefs[0].Name != "Customer" {
		t.Errorf("expected entity name 'Customer', got '%s'", rs.EntityDefs[0].Name)
	}
	if len(rs.EntityDefs[0].Attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(rs.EntityDefs[0].Attributes))
	}

	// Check table
	table := rs.GetTable("CheckAge")
	if table == nil {
		t.Fatal("table 'CheckAge' not found")
	}
	if table.Type != ruleset.TableTypeBalanced {
		t.Errorf("expected BALANCED type, got %v", table.Type)
	}
	if len(table.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(table.Conditions))
	}
	if len(table.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(table.Actions))
	}
}

func TestLoadFromXMLInvalidEDD(t *testing.T) {
	repo := New()

	edd := `not valid xml`
	dt := `<?xml version="1.0"?><decision_tables name="test"></decision_tables>`

	_, err := repo.LoadFromXML(strings.NewReader(edd), strings.NewReader(dt))
	if err == nil {
		t.Error("expected error for invalid EDD")
	}
}

func TestLoadFromXMLInvalidDT(t *testing.T) {
	repo := New()

	edd := `<?xml version="1.0"?><edd name="test"></edd>`
	dt := `not valid xml`

	_, err := repo.LoadFromXML(strings.NewReader(edd), strings.NewReader(dt))
	if err == nil {
		t.Error("expected error for invalid DT")
	}
}

func TestSaveAndLoadCompiled(t *testing.T) {
	repo := New()

	// Create a ruleset
	rs := ruleset.NewCompiledRuleSet("test-ruleset")
	rs.Version = "1.0"
	rs.AddTable(ruleset.NewCompiledTable("TestTable", ruleset.TableTypeFirst))

	// Save
	var buf bytes.Buffer
	if err := repo.Save(rs, &buf); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	rs2, err := repo.LoadCompiled(&buf)
	if err != nil {
		t.Fatalf("LoadCompiled failed: %v", err)
	}

	if rs2.Name != rs.Name {
		t.Errorf("name mismatch: expected '%s', got '%s'", rs.Name, rs2.Name)
	}
	if rs2.Version != rs.Version {
		t.Errorf("version mismatch: expected '%s', got '%s'", rs.Version, rs2.Version)
	}
	if rs2.GetTable("TestTable") == nil {
		t.Error("table not found after round-trip")
	}
}

func TestValidateEmptyName(t *testing.T) {
	repo := New()
	rs := &ruleset.CompiledRuleSet{
		Name:   "",
		Tables: make(map[string]*ruleset.CompiledTable),
	}

	errs := repo.Validate(rs)
	if len(errs) == 0 {
		t.Error("expected validation error for empty name")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "name is empty") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'name is empty' error")
	}
}

func TestValidateValidRuleSet(t *testing.T) {
	repo := New()
	rs := ruleset.NewCompiledRuleSet("valid-ruleset")
	rs.AddTable(ruleset.NewCompiledTable("Table1", ruleset.TableTypeBalanced))

	errs := repo.Validate(rs)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		expr     string
		expected []string
	}{
		{"1 2 +", []string{"1", "2", "+"}},
		{"  1   2   +  ", []string{"1", "2", "+"}},
		{`"hello world"`, []string{`"hello world"`}},
		{`'single quotes'`, []string{`'single quotes'`}},
		{"{1 2 +}", []string{"{1 2 +}"}},
		{"1 {2 3 +} 4", []string{"1", "{2 3 +}", "4"}},
		{"/name def", []string{"/name", "def"}},
		{"", []string{}},
		{"   ", []string{}},
	}

	for _, tc := range tests {
		result := tokenize(tc.expr)
		if len(result) != len(tc.expected) {
			t.Errorf("tokenize(%q): expected %d tokens, got %d: %v",
				tc.expr, len(tc.expected), len(result), result)
			continue
		}
		for i, tok := range result {
			if tok != tc.expected[i] {
				t.Errorf("tokenize(%q)[%d]: expected %q, got %q",
					tc.expr, i, tc.expected[i], tok)
			}
		}
	}
}

func TestCompileExpressionSimple(t *testing.T) {
	bc, err := compileExpression("1 2 +")
	if err != nil {
		t.Fatalf("compileExpression failed: %v", err)
	}
	if bc == nil {
		t.Fatal("expected bytecode, got nil")
	}
	if len(bc.Code()) == 0 {
		t.Error("expected non-empty bytecode")
	}
}

func TestCompileExpressionEmpty(t *testing.T) {
	bc, err := compileExpression("")
	if err != nil {
		t.Fatalf("compileExpression failed: %v", err)
	}
	if bc != nil {
		t.Error("expected nil for empty expression")
	}
}

func TestCompileExpressionWithString(t *testing.T) {
	bc, err := compileExpression(`"hello" print`)
	if err != nil {
		t.Fatalf("compileExpression failed: %v", err)
	}
	if bc == nil {
		t.Fatal("expected bytecode, got nil")
	}
}

func TestCompileExpressionWithLiteralName(t *testing.T) {
	bc, err := compileExpression("/myvar 42 def")
	if err != nil {
		t.Fatalf("compileExpression failed: %v", err)
	}
	if bc == nil {
		t.Fatal("expected bytecode, got nil")
	}
}

func TestCompileExpressionWithBooleans(t *testing.T) {
	bc, err := compileExpression("true false and")
	if err != nil {
		t.Fatalf("compileExpression failed: %v", err)
	}
	if bc == nil {
		t.Fatal("expected bytecode, got nil")
	}
}

func TestParseAttributeType(t *testing.T) {
	tests := []struct {
		input    string
		expected ruleset.AttributeType
	}{
		{"integer", ruleset.AttrTypeInteger},
		{"INT", ruleset.AttrTypeInteger},
		{"long", ruleset.AttrTypeInteger},
		{"double", ruleset.AttrTypeDouble},
		{"FLOAT", ruleset.AttrTypeDouble},
		{"number", ruleset.AttrTypeDouble},
		{"string", ruleset.AttrTypeString},
		{"TEXT", ruleset.AttrTypeString},
		{"boolean", ruleset.AttrTypeBoolean},
		{"bool", ruleset.AttrTypeBoolean},
		{"date", ruleset.AttrTypeDate},
		{"array", ruleset.AttrTypeArray},
		{"list", ruleset.AttrTypeArray},
		{"entity", ruleset.AttrTypeEntity},
		{"table", ruleset.AttrTypeTable},
		{"map", ruleset.AttrTypeTable},
		{"hashtable", ruleset.AttrTypeTable},
		{"name", ruleset.AttrTypeName},
		{"unknown", ruleset.AttrTypeUnknown},
		{"xyz", ruleset.AttrTypeUnknown},
	}

	for _, tc := range tests {
		result := parseAttributeType(tc.input)
		if result != tc.expected {
			t.Errorf("parseAttributeType(%q): expected %v, got %v",
				tc.input, tc.expected, result)
		}
	}
}

func TestParseTableType(t *testing.T) {
	tests := []struct {
		input    string
		expected ruleset.TableType
	}{
		{"BALANCED", ruleset.TableTypeBalanced},
		{"balanced", ruleset.TableTypeBalanced},
		{"FIRST", ruleset.TableTypeFirst},
		{"first", ruleset.TableTypeFirst},
		{"ALL", ruleset.TableTypeAll},
		{"all", ruleset.TableTypeAll},
		{"", ruleset.TableTypeBalanced},
		{"unknown", ruleset.TableTypeBalanced},
	}

	for _, tc := range tests {
		result := parseTableType(tc.input)
		if result != tc.expected {
			t.Errorf("parseTableType(%q): expected %v, got %v",
				tc.input, tc.expected, result)
		}
	}
}
