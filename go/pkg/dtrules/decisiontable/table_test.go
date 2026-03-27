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

package decisiontable

import (
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

func TestNewRDecisionTable(t *testing.T) {
	name := dtrules.GetRName("TestTable")
	dt := NewRDecisionTable(name, nil)

	if dt == nil {
		t.Fatal("Expected non-nil decision table")
	}

	if dt.GetName() != "TestTable" {
		t.Errorf("Expected name 'TestTable', got '%s'", dt.GetName())
	}

	if dt.GetTableType() != BALANCED {
		t.Errorf("Expected default type BALANCED, got %v", dt.GetTableType())
	}
}

func TestTableTypeString(t *testing.T) {
	tests := []struct {
		tableType TableType
		expected  string
	}{
		{BALANCED, "BALANCED"},
		{FIRST, "FIRST"},
		{ALL, "ALL"},
		{TableType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		got := tt.tableType.String()
		if got != tt.expected {
			t.Errorf("TableType(%d).String() = %s, want %s", tt.tableType, got, tt.expected)
		}
	}
}

func TestSetTableType(t *testing.T) {
	name := dtrules.GetRName("TestTable")
	dt := NewRDecisionTable(name, nil)

	dt.SetTableType(FIRST)
	if dt.GetTableType() != FIRST {
		t.Errorf("Expected FIRST, got %v", dt.GetTableType())
	}

	dt.SetTableType(ALL)
	if dt.GetTableType() != ALL {
		t.Errorf("Expected ALL, got %v", dt.GetTableType())
	}
}

func TestRDecisionTableType(t *testing.T) {
	name := dtrules.GetRName("TestTable")
	dt := NewRDecisionTable(name, nil)

	if dt.Type() != dtrules.TypeDecisionTable {
		t.Errorf("Expected TypeDecisionTable, got %v", dt.Type())
	}
}

func TestRDecisionTableStringValue(t *testing.T) {
	name := dtrules.GetRName("MyTable")
	dt := NewRDecisionTable(name, nil)

	if dt.StringValue() != "MyTable" {
		t.Errorf("Expected 'MyTable', got '%s'", dt.StringValue())
	}
}

func TestRDecisionTableClone(t *testing.T) {
	name := dtrules.GetRName("TestTable")
	dt := NewRDecisionTable(name, nil)

	cloned, err := dt.Clone(nil)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Decision tables are shared, not deep-cloned
	if cloned != dt {
		t.Error("Expected clone to return same instance")
	}
}

func TestSetField(t *testing.T) {
	name := dtrules.GetRName("TestTable")
	dt := NewRDecisionTable(name, nil)

	dt.SetField("Type", "FIRST")
	dt.SetField("Comments", "Test table")

	if dt.GetField("Type") != "FIRST" {
		t.Errorf("Expected field 'Type' to be 'FIRST'")
	}

	if dt.GetField("Comments") != "Test table" {
		t.Errorf("Expected field 'Comments' to be 'Test table'")
	}

	if dt.GetField("NonExistent") != "" {
		t.Error("Expected empty string for non-existent field")
	}
}

func TestGetErrors(t *testing.T) {
	name := dtrules.GetRName("TestTable")
	dt := NewRDecisionTable(name, nil)

	// Initially no errors
	if len(dt.GetErrors()) != 0 {
		t.Error("Expected no errors initially")
	}
}

func TestDASHConstant(t *testing.T) {
	if DASH != "-" {
		t.Errorf("Expected DASH to be '-', got '%s'", DASH)
	}
}

func TestMAXCOLConstant(t *testing.T) {
	if MAXCOL != 16 {
		t.Errorf("Expected MAXCOL to be 16, got %d", MAXCOL)
	}
}

func TestRDecisionTableGetRName(t *testing.T) {
	name := dtrules.GetRName("NameTest")
	dt := NewRDecisionTable(name, nil)

	rname := dt.GetRName()
	if rname != name {
		t.Error("Expected GetRName to return the original name")
	}
}

func TestRDecisionTableIsExecutable(t *testing.T) {
	name := dtrules.GetRName("ExecTest")
	dt := NewRDecisionTable(name, nil)

	// Decision tables are executable
	if !dt.IsExecutable() {
		t.Error("Expected decision table to be executable")
	}
}

func TestSetMaxColValidation(t *testing.T) {
	name := dtrules.GetRName("MaxColTest")
	dt := NewRDecisionTable(name, nil)

	// Test clamping to minimum
	dt.SetMaxCol(0)
	if dt.GetMaxCol() != 1 {
		t.Errorf("Expected MaxCol to be clamped to 1, got %d", dt.GetMaxCol())
	}

	dt.SetMaxCol(-5)
	if dt.GetMaxCol() != 1 {
		t.Errorf("Expected MaxCol to be clamped to 1, got %d", dt.GetMaxCol())
	}

	// Test clamping to maximum
	dt.SetMaxCol(100)
	if dt.GetMaxCol() != MAXCOL {
		t.Errorf("Expected MaxCol to be clamped to MAXCOL (%d), got %d", MAXCOL, dt.GetMaxCol())
	}

	// Test valid value
	dt.SetMaxCol(8)
	if dt.GetMaxCol() != 8 {
		t.Errorf("Expected MaxCol to be 8, got %d", dt.GetMaxCol())
	}
}
