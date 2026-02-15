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
	"math"
	"testing"
)

// =============================================================================
// Tag and type check tests
// =============================================================================

func TestValueTag(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want uint8
	}{
		{"null", ValueNull, VTagNull},
		{"integer", NewValueInteger(42), VTagInteger},
		{"double", NewValueDouble(3.14), VTagDouble},
		{"true", ValueTrue, VTagBoolean},
		{"false", ValueFalse, VTagBoolean},
		{"string", NewValueString("hello"), VTagString},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Tag(); got != tt.want {
				t.Errorf("Tag() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValueTypeChecksExclusive(t *testing.T) {
	// Each value should return true for exactly one Is* method
	tests := []struct {
		name    string
		v       Value
		isNull  bool
		isInt   bool
		isDbl   bool
		isBool  bool
		isStr   bool
		isName  bool
		isNum   bool
	}{
		{"null", ValueNull, true, false, false, false, false, false, false},
		{"integer", NewValueInteger(1), false, true, false, false, false, false, true},
		{"double", NewValueDouble(1.0), false, false, true, false, false, false, true},
		{"boolean", NewValueBoolean(true), false, false, false, true, false, false, false},
		{"string", NewValueString("x"), false, false, false, false, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.v.IsNull() != tt.isNull {
				t.Errorf("IsNull() = %v, want %v", tt.v.IsNull(), tt.isNull)
			}
			if tt.v.IsInteger() != tt.isInt {
				t.Errorf("IsInteger() = %v, want %v", tt.v.IsInteger(), tt.isInt)
			}
			if tt.v.IsDouble() != tt.isDbl {
				t.Errorf("IsDouble() = %v, want %v", tt.v.IsDouble(), tt.isDbl)
			}
			if tt.v.IsBoolean() != tt.isBool {
				t.Errorf("IsBoolean() = %v, want %v", tt.v.IsBoolean(), tt.isBool)
			}
			if tt.v.IsString() != tt.isStr {
				t.Errorf("IsString() = %v, want %v", tt.v.IsString(), tt.isStr)
			}
			if tt.v.IsNumeric() != tt.isNum {
				t.Errorf("IsNumeric() = %v, want %v", tt.v.IsNumeric(), tt.isNum)
			}
		})
	}
}

// =============================================================================
// Name value tests
// =============================================================================

func TestValueName(t *testing.T) {
	name := GetRName("testattr")
	v := NewValueName(name)

	if !v.IsName() {
		t.Error("expected IsName() to return true")
	}
	if v.Tag() != VTagName {
		t.Errorf("Tag() = %d, want %d", v.Tag(), VTagName)
	}
	if v.AsName() != name {
		t.Error("AsName() should return the original RName pointer")
	}
	// String should include the name value
	str := v.String()
	if str == "" {
		t.Error("String() should not be empty for name value")
	}
}

// =============================================================================
// Array value tests
// =============================================================================

func TestValueArray(t *testing.T) {
	arr := NewArrayTraceInterface(1, true, false)
	arr.Add(GetRIntegerValue(10))
	arr.Add(GetRIntegerValue(20))

	v := NewValueArray(arr)
	if v.Tag() != VTagArray {
		t.Errorf("Tag() = %d, want %d", v.Tag(), VTagArray)
	}
	if v.AsArray() != arr {
		t.Error("AsArray() should return the original array pointer")
	}
	str := v.String()
	if str == "" {
		t.Error("String() should not be empty for array value")
	}
}

// =============================================================================
// Object value tests
// =============================================================================

func TestValueObject(t *testing.T) {
	obj := GetRIntegerValue(99)
	v := NewValueObject(obj)

	if v.Tag() != VTagObject {
		t.Errorf("Tag() = %d, want %d", v.Tag(), VTagObject)
	}

	// AsObject should reconstruct the original interface
	recovered := v.AsObject()
	if recovered == nil {
		t.Fatal("AsObject() returned nil")
	}
	val, err := recovered.LongValue()
	if err != nil {
		t.Fatalf("LongValue() error: %v", err)
	}
	if val != 99 {
		t.Errorf("expected 99, got %d", val)
	}
}

func TestNewValueObjectNil(t *testing.T) {
	v := NewValueObject(nil)
	if !v.IsNull() {
		t.Error("NewValueObject(nil) should return a null value")
	}
}

// =============================================================================
// Entity value tests
// =============================================================================

// TestNewValueEntity is skipped since it requires a concrete Entity implementation
// which depends on the session package. This is covered by integration tests.

// =============================================================================
// AsDouble for integer values (implicit promotion)
// =============================================================================

func TestValueAsDoubleFromInteger(t *testing.T) {
	v := NewValueInteger(42)
	d := v.AsDouble()
	if d != 42.0 {
		t.Errorf("AsDouble() on integer = %f, want 42.0", d)
	}
}

// =============================================================================
// AsString for non-string types (falls through to String())
// =============================================================================

func TestValueAsStringFallback(t *testing.T) {
	tests := []struct {
		name     string
		v        Value
		expected string
	}{
		{"integer", NewValueInteger(42), "42"},
		{"null", ValueNull, "null"},
		{"bool_true", ValueTrue, "true"},
		{"bool_false", ValueFalse, "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.AsString(); got != tt.expected {
				t.Errorf("AsString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Cross-type arithmetic
// =============================================================================

func TestValueArithmeticMixed(t *testing.T) {
	intVal := NewValueInteger(10)
	dblVal := NewValueDouble(2.5)

	// int + double → double
	sum := intVal.Add(dblVal)
	if !sum.IsDouble() {
		t.Error("expected double result from int + double")
	}
	if sum.AsDouble() != 12.5 {
		t.Errorf("Add: expected 12.5, got %f", sum.AsDouble())
	}

	// int - double → double
	diff := intVal.Sub(dblVal)
	if diff.AsDouble() != 7.5 {
		t.Errorf("Sub: expected 7.5, got %f", diff.AsDouble())
	}

	// int * double → double
	prod := intVal.Mul(dblVal)
	if prod.AsDouble() != 25.0 {
		t.Errorf("Mul: expected 25.0, got %f", prod.AsDouble())
	}
}

// =============================================================================
// Division edge cases
// =============================================================================

func TestValueDivReturnsDouble(t *testing.T) {
	// Division of two integers should return double
	a := NewValueInteger(7)
	b := NewValueInteger(2)
	result := a.Div(b)
	if !result.IsDouble() {
		t.Error("Div should always return double")
	}
	if result.AsDouble() != 3.5 {
		t.Errorf("expected 3.5, got %f", result.AsDouble())
	}
}

func TestValueDivByZeroInf(t *testing.T) {
	a := NewValueDouble(1.0)
	b := NewValueDouble(0.0)
	result := a.Div(b)
	if !math.IsInf(result.AsDouble(), 1) {
		t.Errorf("expected +Inf, got %f", result.AsDouble())
	}
}

func TestValueTryDivByZeroDouble(t *testing.T) {
	a := NewValueDouble(1.0)
	b := NewValueDouble(0.0)
	_, err := a.TryDiv(b)
	if err == nil {
		t.Error("expected error for TryDiv by zero")
	}
}

// =============================================================================
// Equality edge cases
// =============================================================================

func TestValueEqualCrossType(t *testing.T) {
	// int 3 == double 3.0
	intV := NewValueInteger(3)
	dblV := NewValueDouble(3.0)
	if !intV.Equal(dblV) {
		t.Error("integer 3 should equal double 3.0")
	}
	if !dblV.Equal(intV) {
		t.Error("double 3.0 should equal integer 3")
	}
}

func TestValueEqualDifferentTypes(t *testing.T) {
	// Different non-numeric types should not be equal
	intV := NewValueInteger(1)
	boolV := NewValueBoolean(true) // num=1 but different tag
	if intV.Equal(boolV) {
		t.Error("integer 1 should not equal boolean true")
	}

	strV := NewValueString("hello")
	intV2 := NewValueInteger(0)
	if strV.Equal(intV2) {
		t.Error("string should not equal integer")
	}
}

func TestValueEqualNull(t *testing.T) {
	if !ValueNull.Equal(ValueNull) {
		t.Error("null should equal null")
	}
}

func TestValueEqualStrings(t *testing.T) {
	a := NewValueString("hello")
	b := NewValueString("hello")
	c := NewValueString("world")
	if !a.Equal(b) {
		t.Error("same strings should be equal")
	}
	if a.Equal(c) {
		t.Error("different strings should not be equal")
	}
}

func TestValueEqualBooleans(t *testing.T) {
	if !ValueTrue.Equal(ValueTrue) {
		t.Error("true should equal true")
	}
	if !ValueFalse.Equal(ValueFalse) {
		t.Error("false should equal false")
	}
	if ValueTrue.Equal(ValueFalse) {
		t.Error("true should not equal false")
	}
}

func TestValueEqualDoubles(t *testing.T) {
	a := NewValueDouble(3.14)
	b := NewValueDouble(3.14)
	c := NewValueDouble(2.71)
	if !a.Equal(b) {
		t.Error("same doubles should be equal")
	}
	if a.Equal(c) {
		t.Error("different doubles should not be equal")
	}
}

// =============================================================================
// Less comparison edge cases
// =============================================================================

func TestValueLessMixed(t *testing.T) {
	// Cross-type numeric comparison
	intV := NewValueInteger(2)
	dblV := NewValueDouble(3.0)
	if !intV.Less(dblV) {
		t.Error("integer 2 should be less than double 3.0")
	}
	if dblV.Less(intV) {
		t.Error("double 3.0 should not be less than integer 2")
	}
}

// =============================================================================
// AsObject for all types
// =============================================================================

func TestValueAsObjectAllTypes(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		typ  *RType
	}{
		{"null", ValueNull, TypeNull},
		{"integer", NewValueInteger(42), TypeInteger},
		{"double", NewValueDouble(3.14), TypeDouble},
		{"boolean_true", ValueTrue, TypeBoolean},
		{"boolean_false", ValueFalse, TypeBoolean},
		{"string", NewValueString("hello"), TypeString},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := tt.v.AsObject()
			if obj == nil {
				t.Fatal("AsObject() returned nil")
			}
			if obj.Type() != tt.typ {
				t.Errorf("AsObject().Type() = %v, want %v", obj.Type(), tt.typ)
			}
		})
	}
}

func TestValueAsObjectName(t *testing.T) {
	name := GetRName("testattr")
	v := NewValueName(name)
	obj := v.AsObject()
	if obj == nil {
		t.Fatal("AsObject() returned nil for name value")
	}
	// The object should be the same RName
	rn, err := obj.RNameValue()
	if err != nil {
		t.Fatalf("RNameValue() error: %v", err)
	}
	if rn != name {
		t.Error("AsObject().RNameValue() should return the original name")
	}
}

func TestValueAsObjectArray(t *testing.T) {
	arr := NewArrayTraceInterface(1, true, false)
	v := NewValueArray(arr)
	obj := v.AsObject()
	if obj == nil {
		t.Fatal("AsObject() returned nil for array value")
	}
	ra, err := obj.RArrayValue()
	if err != nil {
		t.Fatalf("RArrayValue() error: %v", err)
	}
	if ra != arr {
		t.Error("AsObject().RArrayValue() should return the original array")
	}
}

// =============================================================================
// ValueFromObject comprehensive tests
// =============================================================================

func TestValueFromObjectNil(t *testing.T) {
	v := ValueFromObject(nil)
	if !v.IsNull() {
		t.Error("ValueFromObject(nil) should return null value")
	}
}

func TestValueFromObjectString(t *testing.T) {
	obj := NewRString("hello")
	v := ValueFromObject(obj)
	if !v.IsString() {
		t.Error("expected string value")
	}
	if v.AsString() != "hello" {
		t.Errorf("expected 'hello', got %q", v.AsString())
	}
}

func TestValueFromObjectName(t *testing.T) {
	name := GetRName("attr")
	v := ValueFromObject(name)
	if !v.IsName() {
		t.Error("expected name value")
	}
	if v.AsName() != name {
		t.Error("expected same RName pointer")
	}
}

func TestValueFromObjectArray(t *testing.T) {
	arr := NewArrayTraceInterface(1, true, false)
	v := ValueFromObject(arr)
	if v.Tag() != VTagArray {
		t.Errorf("expected VTagArray, got %d", v.Tag())
	}
	if v.AsArray() != arr {
		t.Error("expected same array pointer")
	}
}

func TestValueFromObjectNull(t *testing.T) {
	v := ValueFromObject(GetRNull())
	if !v.IsNull() {
		t.Error("expected null value from RNull object")
	}
}

// =============================================================================
// String formatting
// =============================================================================

func TestValueStringFormatString(t *testing.T) {
	v := NewValueString("hello world")
	if v.String() != "hello world" {
		t.Errorf("String() = %q, want 'hello world'", v.String())
	}
}

func TestValueStringFormatName(t *testing.T) {
	name := GetRName("myattr")
	v := NewValueName(name)
	str := v.String()
	if str == "" {
		t.Error("String() should not be empty for name value")
	}
}

func TestValueStringFormatObject(t *testing.T) {
	obj := GetRIntegerValue(42)
	v := NewValueObject(obj)
	str := v.String()
	if str != "42" {
		t.Errorf("String() = %q, want '42'", str)
	}
}

// =============================================================================
// Boolean singleton tests
// =============================================================================

func TestValueBooleanSingletons(t *testing.T) {
	// NewValueBoolean should return the same singleton objects
	trueVal := NewValueBoolean(true)
	if trueVal != ValueTrue {
		t.Error("NewValueBoolean(true) should return ValueTrue singleton")
	}
	falseVal := NewValueBoolean(false)
	if falseVal != ValueFalse {
		t.Error("NewValueBoolean(false) should return ValueFalse singleton")
	}
}

// =============================================================================
// Integer overflow / boundary tests
// =============================================================================

func TestValueIntegerBoundary(t *testing.T) {
	maxVal := NewValueInteger(math.MaxInt64)
	minVal := NewValueInteger(math.MinInt64)

	if maxVal.AsInteger() != math.MaxInt64 {
		t.Error("MaxInt64 round-trip failed")
	}
	if minVal.AsInteger() != math.MinInt64 {
		t.Error("MinInt64 round-trip failed")
	}
}

// =============================================================================
// Double special values
// =============================================================================

func TestValueDoubleSpecialValues(t *testing.T) {
	inf := NewValueDouble(math.Inf(1))
	ninf := NewValueDouble(math.Inf(-1))
	nan := NewValueDouble(math.NaN())

	if !math.IsInf(inf.AsDouble(), 1) {
		t.Error("+Inf round-trip failed")
	}
	if !math.IsInf(ninf.AsDouble(), -1) {
		t.Error("-Inf round-trip failed")
	}
	if !math.IsNaN(nan.AsDouble()) {
		t.Error("NaN round-trip failed")
	}
}
