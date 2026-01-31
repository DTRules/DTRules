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

func TestValueInteger(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected int64
	}{
		{"zero", 0, 0},
		{"positive", 42, 42},
		{"negative", -100, -100},
		{"max", math.MaxInt64, math.MaxInt64},
		{"min", math.MinInt64, math.MinInt64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValueInteger(tt.value)
			if !v.IsInteger() {
				t.Error("expected IsInteger() to return true")
			}
			if v.AsInteger() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, v.AsInteger())
			}
		})
	}
}

func TestValueDouble(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{"zero", 0.0, 0.0},
		{"positive", 3.14159, 3.14159},
		{"negative", -2.71828, -2.71828},
		{"small", 1e-10, 1e-10},
		{"large", 1e100, 1e100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValueDouble(tt.value)
			if !v.IsDouble() {
				t.Error("expected IsDouble() to return true")
			}
			if v.AsDouble() != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, v.AsDouble())
			}
		})
	}
}

func TestValueBoolean(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected bool
	}{
		{"true", true, true},
		{"false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValueBoolean(tt.value)
			if !v.IsBoolean() {
				t.Error("expected IsBoolean() to return true")
			}
			if v.AsBoolean() != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, v.AsBoolean())
			}
		})
	}
}

func TestValueString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"empty", "", ""},
		{"simple", "hello", "hello"},
		{"unicode", "日本語", "日本語"},
		{"spaces", "hello world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValueString(tt.value)
			if !v.IsString() {
				t.Error("expected IsString() to return true")
			}
			if v.AsString() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, v.AsString())
			}
		})
	}
}

func TestValueNull(t *testing.T) {
	v := ValueNull
	if !v.IsNull() {
		t.Error("expected IsNull() to return true")
	}
}

func TestValueArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Value
		op       string
		expected int64
	}{
		{"add", NewValueInteger(10), NewValueInteger(5), "add", 15},
		{"sub", NewValueInteger(10), NewValueInteger(5), "sub", 5},
		{"mul", NewValueInteger(10), NewValueInteger(5), "mul", 50},
		{"add_negative", NewValueInteger(10), NewValueInteger(-5), "add", 5},
		{"sub_negative", NewValueInteger(10), NewValueInteger(-5), "sub", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result Value
			switch tt.op {
			case "add":
				result = tt.a.Add(tt.b)
			case "sub":
				result = tt.a.Sub(tt.b)
			case "mul":
				result = tt.a.Mul(tt.b)
			}
			if result.AsInteger() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result.AsInteger())
			}
		})
	}
}

func TestValueDivision(t *testing.T) {
	// Division always returns double
	a := NewValueInteger(10)
	b := NewValueInteger(3)
	result := a.Div(b)

	expected := 10.0 / 3.0
	if math.Abs(result.AsDouble()-expected) > 1e-10 {
		t.Errorf("expected %f, got %f", expected, result.AsDouble())
	}
}

func TestValueComparison(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Value
		less     bool
		equal    bool
	}{
		{"less", NewValueInteger(5), NewValueInteger(10), true, false},
		{"equal", NewValueInteger(10), NewValueInteger(10), false, true},
		{"greater", NewValueInteger(15), NewValueInteger(10), false, false},
		{"double_less", NewValueDouble(3.14), NewValueDouble(3.15), true, false},
		{"mixed", NewValueInteger(3), NewValueDouble(3.0), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.a.Less(tt.b) != tt.less {
				t.Errorf("Less: expected %t, got %t", tt.less, tt.a.Less(tt.b))
			}
			if tt.a.Equal(tt.b) != tt.equal {
				t.Errorf("Equal: expected %t, got %t", tt.equal, tt.a.Equal(tt.b))
			}
		})
	}
}

func TestValueToObject(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		check func(Object) bool
	}{
		{"integer", NewValueInteger(42), func(o Object) bool { v, _ := o.LongValue(); return v == 42 }},
		{"double", NewValueDouble(3.14), func(o Object) bool { v, _ := o.DoubleValue(); return math.Abs(v-3.14) < 1e-10 }},
		{"boolean", NewValueBoolean(true), func(o Object) bool { v, _ := o.BooleanValue(); return v }},
		{"null", ValueNull, func(o Object) bool { return o.Type() == TypeNull }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := tt.value.AsObject()
			if !tt.check(obj) {
				t.Errorf("AsObject() returned unexpected value")
			}
		})
	}
}

func TestValueFromObject(t *testing.T) {
	tests := []struct {
		name  string
		obj   Object
		check func(Value) bool
	}{
		{"integer", GetRIntegerValue(42), func(v Value) bool { return v.IsInteger() && v.AsInteger() == 42 }},
		{"double", GetRDoubleValue(3.14), func(v Value) bool { return v.IsDouble() && math.Abs(v.AsDouble()-3.14) < 1e-10 }},
		{"boolean", True, func(v Value) bool { return v.IsBoolean() && v.AsBoolean() }},
		{"null", GetRNull(), func(v Value) bool { return v.IsNull() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ValueFromObject(tt.obj)
			if !tt.check(v) {
				t.Errorf("ValueFromObject() returned unexpected value")
			}
		})
	}
}

func TestValueString_Format(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected string
	}{
		{"integer", NewValueInteger(42), "42"},
		{"negative", NewValueInteger(-100), "-100"},
		{"double", NewValueDouble(3.14), "3.14"},
		{"true", ValueTrue, "true"},
		{"false", ValueFalse, "false"},
		{"null", ValueNull, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.value.String())
			}
		})
	}
}
