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
	"testing"
)

func TestRInteger(t *testing.T) {
	// Test creation
	i := GetRIntegerValue(42)
	if i == nil {
		t.Fatal("GetRIntegerValue returned nil")
	}

	// Test IntValue
	val, err := i.IntValue()
	if err != nil {
		t.Fatalf("IntValue failed: %v", err)
	}
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}

	// Test LongValue
	longVal, err := i.LongValue()
	if err != nil {
		t.Fatalf("LongValue failed: %v", err)
	}
	if longVal != 42 {
		t.Errorf("Expected 42, got %d", longVal)
	}

	// Test DoubleValue
	doubleVal, err := i.DoubleValue()
	if err != nil {
		t.Fatalf("DoubleValue failed: %v", err)
	}
	if doubleVal != 42.0 {
		t.Errorf("Expected 42.0, got %f", doubleVal)
	}

	// Test StringValue
	if i.StringValue() != "42" {
		t.Errorf("Expected '42', got '%s'", i.StringValue())
	}

	// Test Type
	if i.Type() != TypeInteger {
		t.Errorf("Expected TypeInteger, got %v", i.Type())
	}

	// Test cached values
	zero := GetRIntegerValue(0)
	zero2 := GetRIntegerValue(0)
	if zero != zero2 {
		t.Error("Expected cached zero values to be same instance")
	}

	one := GetRIntegerValue(1)
	one2 := GetRIntegerValue(1)
	if one != one2 {
		t.Error("Expected cached one values to be same instance")
	}

	negOne := GetRIntegerValue(-1)
	negOne2 := GetRIntegerValue(-1)
	if negOne != negOne2 {
		t.Error("Expected cached -1 values to be same instance")
	}
}

func TestRDouble(t *testing.T) {
	// Test creation
	d := GetRDoubleValue(3.14)
	if d == nil {
		t.Fatal("GetRDoubleValue returned nil")
	}

	// Test DoubleValue
	val, err := d.DoubleValue()
	if err != nil {
		t.Fatalf("DoubleValue failed: %v", err)
	}
	if val != 3.14 {
		t.Errorf("Expected 3.14, got %f", val)
	}

	// Test IntValue (truncation)
	intVal, err := d.IntValue()
	if err != nil {
		t.Fatalf("IntValue failed: %v", err)
	}
	if intVal != 3 {
		t.Errorf("Expected 3, got %d", intVal)
	}

	// Test Type
	if d.Type() != TypeDouble {
		t.Errorf("Expected TypeDouble, got %v", d.Type())
	}
}

func TestRBoolean(t *testing.T) {
	// Test True singleton
	if True == nil {
		t.Fatal("True is nil")
	}
	if False == nil {
		t.Fatal("False is nil")
	}

	// Test BooleanValue
	trueVal, err := True.BooleanValue()
	if err != nil {
		t.Fatalf("BooleanValue failed: %v", err)
	}
	if !trueVal {
		t.Error("Expected true")
	}

	falseVal, err := False.BooleanValue()
	if err != nil {
		t.Fatalf("BooleanValue failed: %v", err)
	}
	if falseVal {
		t.Error("Expected false")
	}

	// Test GetRBoolean
	if GetRBoolean(true) != True {
		t.Error("GetRBoolean(true) should return True singleton")
	}
	if GetRBoolean(false) != False {
		t.Error("GetRBoolean(false) should return False singleton")
	}

	// Test Type
	if True.Type() != TypeBoolean {
		t.Errorf("Expected TypeBoolean, got %v", True.Type())
	}
}

func TestRString(t *testing.T) {
	// Test creation
	s := NewRString("hello")
	if s == nil {
		t.Fatal("NewRString returned nil")
	}

	// Test StringValue
	if s.StringValue() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", s.StringValue())
	}

	// Test Type
	if s.Type() != TypeString {
		t.Errorf("Expected TypeString, got %v", s.Type())
	}

	// Test empty string
	empty := NewRString("")
	if empty.StringValue() != "" {
		t.Errorf("Expected empty string, got '%s'", empty.StringValue())
	}
}

func TestRName(t *testing.T) {
	// Test creation and interning
	name1 := GetRName("test")
	name2 := GetRName("test")
	if name1 != name2 {
		t.Error("Interned names should be same instance")
	}

	// Test case insensitivity
	name3 := GetRName("TEST")
	if name1 != name3 {
		t.Error("Names should be case-insensitive")
	}

	// Test StringValue
	if name1.StringValue() != "test" {
		t.Errorf("Expected 'test', got '%s'", name1.StringValue())
	}

	// Test Type
	if name1.Type() != TypeName {
		t.Errorf("Expected TypeName, got %v", name1.Type())
	}

	// Test entity.attribute syntax
	compoundName := GetRName("entity.attribute")
	if compoundName.StringValue() != "entity.attribute" {
		t.Errorf("Expected 'entity.attribute', got '%s'", compoundName.StringValue())
	}
}

func TestRNull(t *testing.T) {
	// Test singleton
	null1 := GetRNull()
	null2 := GetRNull()
	if null1 != null2 {
		t.Error("RNull should be singleton")
	}

	// Test Type
	if null1.Type() != TypeNull {
		t.Errorf("Expected TypeNull, got %v", null1.Type())
	}

	// Test StringValue (returns empty string per DTRules design)
	if null1.StringValue() != "" {
		t.Errorf("Expected '', got '%s'", null1.StringValue())
	}

	// Test PostFix
	if null1.PostFix() != "null" {
		t.Errorf("Expected 'null', got '%s'", null1.PostFix())
	}
}

func TestRMark(t *testing.T) {
	// Test singleton
	mark1 := GetMark()
	mark2 := GetMark()
	if mark1 != mark2 {
		t.Error("RMark should be singleton")
	}

	// Test Type
	if mark1.Type() != TypeMark {
		t.Errorf("Expected TypeMark, got %v", mark1.Type())
	}
}

func TestEquals(t *testing.T) {
	// Test integer equality
	i1 := GetRIntegerValue(42)
	i2 := GetRIntegerValue(42)
	i3 := GetRIntegerValue(43)

	eq, err := i1.Equals(i2)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if !eq {
		t.Error("42 should equal 42")
	}

	eq, err = i1.Equals(i3)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if eq {
		t.Error("42 should not equal 43")
	}

	// Test string equality
	s1 := NewRString("hello")
	s2 := NewRString("hello")
	s3 := NewRString("world")

	eq, err = s1.Equals(s2)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if !eq {
		t.Error("'hello' should equal 'hello'")
	}

	eq, err = s1.Equals(s3)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if eq {
		t.Error("'hello' should not equal 'world'")
	}
}

func TestCompare(t *testing.T) {
	// Test integer comparison
	i1 := GetRIntegerValue(10)
	i2 := GetRIntegerValue(20)
	i3 := GetRIntegerValue(10)

	cmp, err := i1.Compare(i2)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp >= 0 {
		t.Error("10 should be less than 20")
	}

	cmp, err = i2.Compare(i1)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp <= 0 {
		t.Error("20 should be greater than 10")
	}

	cmp, err = i1.Compare(i3)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp != 0 {
		t.Error("10 should equal 10")
	}

	// Test string comparison
	s1 := NewRString("apple")
	s2 := NewRString("banana")

	cmp, err = s1.Compare(s2)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if cmp >= 0 {
		t.Error("'apple' should be less than 'banana'")
	}
}

func TestParseFromString(t *testing.T) {
	// Test integer parsing
	i, err := GetRIntegerValueFromString("123")
	if err != nil {
		t.Fatalf("GetRIntegerValueFromString failed: %v", err)
	}
	val, _ := i.IntValue()
	if val != 123 {
		t.Errorf("Expected 123, got %d", val)
	}

	// Test negative integer
	i, err = GetRIntegerValueFromString("-456")
	if err != nil {
		t.Fatalf("GetRIntegerValueFromString failed: %v", err)
	}
	val, _ = i.IntValue()
	if val != -456 {
		t.Errorf("Expected -456, got %d", val)
	}

	// Test double parsing
	d, err := GetRDoubleValueFromString("3.14159")
	if err != nil {
		t.Fatalf("GetRDoubleValueFromString failed: %v", err)
	}
	dval, _ := d.DoubleValue()
	if dval != 3.14159 {
		t.Errorf("Expected 3.14159, got %f", dval)
	}

	// Test boolean parsing
	b, err := ParseBooleanValue("true")
	if err != nil {
		t.Fatalf("ParseBooleanValue failed: %v", err)
	}
	if !b {
		t.Error("Expected true")
	}

	b, err = ParseBooleanValue("false")
	if err != nil {
		t.Fatalf("ParseBooleanValue failed: %v", err)
	}
	if b {
		t.Error("Expected false")
	}
}

func TestArrayAddAtNegativeIndex(t *testing.T) {
	// Use NewArrayTraceInterface which doesn't require a session
	arr := NewArrayTraceInterface(1, true, false)

	// Add some elements
	arr.Add(GetRIntegerValue(1))
	arr.Add(GetRIntegerValue(2))

	// Try to add at negative index - should return error
	err := arr.AddAt(-1, GetRIntegerValue(99))
	if err == nil {
		t.Error("Expected error for negative index")
	}

	// Valid index should work
	err = arr.AddAt(0, GetRIntegerValue(0))
	if err != nil {
		t.Errorf("Unexpected error for valid index: %v", err)
	}

	// Verify the array has the expected elements
	if arr.Size() != 3 {
		t.Errorf("Expected size 3, got %d", arr.Size())
	}
}
