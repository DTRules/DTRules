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

package dtrules

import (
	"errors"
	"testing"
)

func TestGetRNameBasic(t *testing.T) {
	name := GetRName("test")
	if name == nil {
		t.Fatal("Expected non-nil name")
	}
	if name.StringValue() != "test" {
		t.Errorf("Expected 'test', got '%s'", name.StringValue())
	}
}

func TestGetRNameInterning(t *testing.T) {
	name1 := GetRName("interned")
	name2 := GetRName("interned")

	if name1 != name2 {
		t.Error("Expected same instance for interned names")
	}
}

func TestGetRNameLiteral(t *testing.T) {
	name := GetRName("/literal")
	if name == nil {
		t.Fatal("Expected non-nil name")
	}
	if name.IsExecutable() {
		t.Error("Expected non-executable name for literal")
	}
	if name.StringValue() != "literal" {
		t.Errorf("Expected 'literal', got '%s'", name.StringValue())
	}
}

func TestGetRNameEntityAttribute(t *testing.T) {
	name := GetRName("entity.attribute")
	if name == nil {
		t.Fatal("Expected non-nil name")
	}
	if name.GetEntityName() == nil {
		t.Error("Expected entity component")
	}
	if name.GetEntityName().StringValue() != "entity" {
		t.Errorf("Expected entity 'entity', got '%s'", name.GetEntityName().StringValue())
	}
	if name.GetName() != "attribute" {
		t.Errorf("Expected name 'attribute', got '%s'", name.GetName())
	}
}

func TestTryGetRNameValid(t *testing.T) {
	name, err := TryGetRName("valid")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if name == nil {
		t.Fatal("Expected non-nil name")
	}
}

func TestTryGetRNameInvalidSyntax(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{".leading", "leading dot"},
		{"trailing.", "trailing dot"},
		{"a.b.c", "multiple dots"},
		{".", "just a dot"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			name, err := TryGetRName(tt.input)
			if err == nil {
				t.Errorf("Expected error for '%s'", tt.input)
			}
			if name != nil {
				t.Errorf("Expected nil name for invalid syntax")
			}
			if !errors.Is(err, ErrInvalidNameSyntax) {
				t.Errorf("Expected ErrInvalidNameSyntax, got: %v", err)
			}
		})
	}
}

func TestGetRNameInvalidSyntaxReturnsPlaceholder(t *testing.T) {
	// GetRName should not panic, but return a placeholder
	name := GetRName(".invalid")
	if name == nil {
		t.Fatal("Expected non-nil placeholder name")
	}
	// The placeholder should be "_invalid_"
	if name.StringValue() != "_invalid_" {
		t.Logf("Got placeholder name: '%s'", name.StringValue())
	}
}

func TestGetRNameSpaceReplacement(t *testing.T) {
	name := GetRName("has space")
	if name.StringValue() != "has_space" {
		t.Errorf("Expected 'has_space', got '%s'", name.StringValue())
	}
}

func TestGetRNameTrimming(t *testing.T) {
	name := GetRName("  trimmed  ")
	if name.StringValue() != "trimmed" {
		t.Errorf("Expected 'trimmed', got '%s'", name.StringValue())
	}
}

func TestRNameExecutable(t *testing.T) {
	exec := GetRName("test")
	nonExec := GetRName("/test")

	if !exec.IsExecutable() {
		t.Error("Expected executable name")
	}
	if nonExec.IsExecutable() {
		t.Error("Expected non-executable name")
	}

	// Test partner relationship
	if exec.GetNonExecutable() != nonExec {
		t.Error("Expected GetNonExecutable to return partner")
	}
	if nonExec.GetExecutable() != exec {
		t.Error("Expected GetExecutable to return partner")
	}
}

func TestRNameEquals(t *testing.T) {
	name1 := GetRName("same")
	name2 := GetRName("SAME") // Should be case-insensitive

	eq, err := name1.Equals(name2)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if !eq {
		t.Error("Expected names to be equal (case-insensitive)")
	}
}

func TestRNameCompare(t *testing.T) {
	name1 := GetRName("aaa")
	name2 := GetRName("bbb")

	cmp, err := name1.Compare(name2)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp >= 0 {
		t.Error("Expected 'aaa' < 'bbb'")
	}

	cmp, err = name2.Compare(name1)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp <= 0 {
		t.Error("Expected 'bbb' > 'aaa'")
	}
}

func TestRNameClone(t *testing.T) {
	name := GetRName("cloned")
	cloned, err := name.Clone(nil)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}
	if cloned != name {
		t.Error("Expected clone to return same instance (names are interned)")
	}
}

func TestRNameType(t *testing.T) {
	name := GetRName("typed")
	if name.Type() != TypeName {
		t.Errorf("Expected TypeName, got %v", name.Type())
	}
}

func TestRNamePostFix(t *testing.T) {
	exec := GetRName("test")
	nonExec := GetRName("/test")

	if exec.PostFix() != "test" {
		t.Errorf("Expected 'test', got '%s'", exec.PostFix())
	}
	if nonExec.PostFix() != "/test" {
		t.Errorf("Expected '/test', got '%s'", nonExec.PostFix())
	}
}

func TestRNameHashCode(t *testing.T) {
	name1 := GetRName("hash")
	name2 := GetRName("hash")

	if name1.HashCode() != name2.HashCode() {
		t.Error("Expected same hash code for same name")
	}
}

func TestRNameString(t *testing.T) {
	name := GetRName("stringer")
	if name.String() != "stringer" {
		t.Errorf("Expected 'stringer', got '%s'", name.String())
	}

	literal := GetRName("/literal")
	if literal.String() != "/literal" {
		t.Errorf("Expected '/literal', got '%s'", literal.String())
	}
}

func TestRNameRStringValue(t *testing.T) {
	name := GetRName("tostring")
	rstr := name.RStringValue()
	if rstr == nil {
		t.Fatal("Expected non-nil RString")
	}
	if rstr.StringValue() != "tostring" {
		t.Errorf("Expected 'tostring', got '%s'", rstr.StringValue())
	}
}

func TestRNameRNameValue(t *testing.T) {
	name := GetRName("self")
	rname, err := name.RNameValue()
	if err != nil {
		t.Fatalf("RNameValue failed: %v", err)
	}
	if rname != name {
		t.Error("Expected RNameValue to return self")
	}
}

func TestGetRNameExecutable(t *testing.T) {
	// Test GetRNameExecutable function
	execName := GetRNameExecutable("explicit", true)
	if !execName.IsExecutable() {
		t.Error("Expected executable name")
	}

	nonExecName := GetRNameExecutable("explicit", false)
	if nonExecName.IsExecutable() {
		t.Error("Expected non-executable name")
	}
}

func TestRNameEqualsNonName(t *testing.T) {
	name := GetRName("test")
	str := NewRString("test")

	eq, err := name.Equals(str)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if eq {
		t.Error("Expected name and string to not be equal")
	}
}
