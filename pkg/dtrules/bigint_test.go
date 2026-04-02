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
	"math/big"
	"testing"
)

// =============================================================================
// Basic RBigInt creation and string conversion tests
// =============================================================================

func TestBigIntCreation(t *testing.T) {
	t.Run("NewRBigInt creates new big int with copy of value", func(t *testing.T) {
		original := big.NewInt(12345)
		bi := NewRBigInt(original)

		// Verify value is correct
		if bi.StringValue() != "12345" {
			t.Errorf("Expected 12345, got %s", bi.StringValue())
		}

		// Verify it's a copy (mutating original should not affect RBigInt)
		original.SetInt64(99999)
		if bi.StringValue() != "12345" {
			t.Errorf("NewRBigInt should copy value, but got %s after mutating original", bi.StringValue())
		}
	})

	t.Run("NewRBigInt with nil returns zero", func(t *testing.T) {
		bi := NewRBigInt(nil)
		if bi.StringValue() != "0" {
			t.Errorf("Expected 0 for nil input, got %s", bi.StringValue())
		}
	})

	t.Run("GetRBigIntValue returns cached values", func(t *testing.T) {
		// Test -1 caching
		negOne1 := GetRBigIntValue(big.NewInt(-1))
		negOne2 := GetRBigIntValue(big.NewInt(-1))
		if negOne1 != negOne2 {
			t.Error("Expected cached -1 values to be same instance")
		}

		// Test 0 caching
		zero1 := GetRBigIntValue(big.NewInt(0))
		zero2 := GetRBigIntValue(big.NewInt(0))
		if zero1 != zero2 {
			t.Error("Expected cached 0 values to be same instance")
		}

		// Test 1 caching
		one1 := GetRBigIntValue(big.NewInt(1))
		one2 := GetRBigIntValue(big.NewInt(1))
		if one1 != one2 {
			t.Error("Expected cached 1 values to be same instance")
		}
	})

	t.Run("GetRBigIntValue with nil returns zero", func(t *testing.T) {
		bi := GetRBigIntValue(nil)
		if bi.StringValue() != "0" {
			t.Errorf("Expected 0 for nil input, got %s", bi.StringValue())
		}
	})

	t.Run("GetRBigIntValue creates copy for non-cached values", func(t *testing.T) {
		original := big.NewInt(42)
		bi := GetRBigIntValue(original)

		if bi.StringValue() != "42" {
			t.Errorf("Expected 42, got %s", bi.StringValue())
		}

		// Verify it's a copy
		original.SetInt64(100)
		if bi.StringValue() != "42" {
			t.Errorf("GetRBigIntValue should copy value, but got %s after mutating original", bi.StringValue())
		}
	})
}

func TestBigIntFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"positive", "12345", "12345", false},
		{"negative", "-9876", "-9876", false},
		{"zero", "0", "0", false},
		{"empty string", "", "0", false},
		{"whitespace", "  42  ", "42", false},
		{"null string", "null", "0", false},
		{"NULL string", "NULL", "0", false},
		{"large positive", "999999999999999999999999999999", "999999999999999999999999999999", false},
		{"large negative", "-888888888888888888888888888888", "-888888888888888888888888888888", false},
		{"invalid", "not-a-number", "", true},
		{"hex invalid", "0x123", "", true},
		{"float invalid", "3.14", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi, err := GetRBigIntFromString(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
		})
	}
}

func TestBigIntFromInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"negative one", -1, "-1"},
		{"positive", 42, "42"},
		{"negative", -100, "-100"},
		{"max int64", math.MaxInt64, "9223372036854775807"},
		{"min int64", math.MinInt64, "-9223372036854775808"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi := GetRBigIntFromInt64(tt.input)
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
		})
	}

	// Test caching
	t.Run("GetRBigIntFromInt64 caches -1, 0, 1", func(t *testing.T) {
		// -1 caching
		negOne1 := GetRBigIntFromInt64(-1)
		negOne2 := GetRBigIntFromInt64(-1)
		if negOne1 != negOne2 {
			t.Error("Expected cached -1 values to be same instance")
		}

		// 0 caching
		zero1 := GetRBigIntFromInt64(0)
		zero2 := GetRBigIntFromInt64(0)
		if zero1 != zero2 {
			t.Error("Expected cached 0 values to be same instance")
		}

		// 1 caching
		one1 := GetRBigIntFromInt64(1)
		one2 := GetRBigIntFromInt64(1)
		if one1 != one2 {
			t.Error("Expected cached 1 values to be same instance")
		}
	})
}

func TestBigIntStringValue(t *testing.T) {
	tests := []struct {
		name     string
		value    *big.Int
		expected string
	}{
		{"zero", big.NewInt(0), "0"},
		{"positive", big.NewInt(12345), "12345"},
		{"negative", big.NewInt(-9876), "-9876"},
		{"large positive", new(big.Int).Exp(big.NewInt(10), big.NewInt(50), nil), "100000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi := NewRBigInt(tt.value)
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
			// Also test String() method
			if bi.String() != tt.expected {
				t.Errorf("String(): Expected %s, got %s", tt.expected, bi.String())
			}
			// Also test PostFix() method
			if bi.PostFix() != tt.expected {
				t.Errorf("PostFix(): Expected %s, got %s", tt.expected, bi.PostFix())
			}
		})
	}
}

func TestBigIntType(t *testing.T) {
	bi := GetRBigIntFromInt64(42)
	if bi.Type() != TypeBigInt {
		t.Errorf("Expected TypeBigInt, got %v", bi.Type())
	}
}

// =============================================================================
// Overflow handling tests
// =============================================================================

func TestBigIntOverflow(t *testing.T) {
	// Create values larger than int64 max
	maxInt64Plus1 := new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
	minInt64Minus1 := new(big.Int).Sub(big.NewInt(math.MinInt64), big.NewInt(1))
	veryLarge := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil) // 10^30

	t.Run("LongValue returns error for overflow above max", func(t *testing.T) {
		bi := NewRBigInt(maxInt64Plus1)
		_, err := bi.LongValue()
		if err == nil {
			t.Error("Expected error for value larger than int64 max")
		}
	})

	t.Run("LongValue returns error for overflow below min", func(t *testing.T) {
		bi := NewRBigInt(minInt64Minus1)
		_, err := bi.LongValue()
		if err == nil {
			t.Error("Expected error for value smaller than int64 min")
		}
	})

	t.Run("LongValue returns error for very large value", func(t *testing.T) {
		bi := NewRBigInt(veryLarge)
		_, err := bi.LongValue()
		if err == nil {
			t.Error("Expected error for very large value")
		}
	})

	t.Run("IntValue returns error for overflow", func(t *testing.T) {
		bi := NewRBigInt(maxInt64Plus1)
		_, err := bi.IntValue()
		if err == nil {
			t.Error("Expected error for value larger than int64 max")
		}
	})

	t.Run("IntValue returns error for very large value", func(t *testing.T) {
		bi := NewRBigInt(veryLarge)
		_, err := bi.IntValue()
		if err == nil {
			t.Error("Expected error for very large value")
		}
	})

	t.Run("LongValue succeeds for values within int64 range", func(t *testing.T) {
		bi := GetRBigIntFromInt64(math.MaxInt64)
		val, err := bi.LongValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if val != math.MaxInt64 {
			t.Errorf("Expected %d, got %d", int64(math.MaxInt64), val)
		}
	})

	t.Run("IntValue succeeds for small values", func(t *testing.T) {
		bi := GetRBigIntFromInt64(42)
		val, err := bi.IntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}
	})
}

func TestBigIntRIntegerValue(t *testing.T) {
	t.Run("RIntegerValue succeeds for small values", func(t *testing.T) {
		bi := GetRBigIntFromInt64(42)
		ri, err := bi.RIntegerValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		val, _ := ri.LongValue()
		if val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}
	})

	t.Run("RIntegerValue returns error for overflow", func(t *testing.T) {
		veryLarge := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
		bi := NewRBigInt(veryLarge)
		_, err := bi.RIntegerValue()
		if err == nil {
			t.Error("Expected error for very large value")
		}
	})
}

func TestBigIntDoubleValue(t *testing.T) {
	t.Run("DoubleValue for small values", func(t *testing.T) {
		bi := GetRBigIntFromInt64(42)
		val, err := bi.DoubleValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if val != 42.0 {
			t.Errorf("Expected 42.0, got %f", val)
		}
	})

	t.Run("DoubleValue for large values", func(t *testing.T) {
		veryLarge := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
		bi := NewRBigInt(veryLarge)
		val, err := bi.DoubleValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// Should be close to 1e20
		expected := 1e20
		if math.Abs(val-expected)/expected > 1e-10 {
			t.Errorf("Expected approximately %e, got %e", expected, val)
		}
	})
}

func TestBigIntBooleanValue(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected bool
	}{
		{"zero is false", 0, false},
		{"one is true", 1, true},
		{"negative is true", -1, true},
		{"large positive is true", 12345, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi := GetRBigIntFromInt64(tt.value)
			val, err := bi.BooleanValue()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if val != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, val)
			}
		})
	}
}

// =============================================================================
// Type coercion tests - RBigIntValue() on other types
// =============================================================================

func TestBigIntTypeCoercion(t *testing.T) {
	t.Run("RInteger to RBigInt", func(t *testing.T) {
		ri := GetRIntegerValue(12345)
		bi, err := ri.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "12345" {
			t.Errorf("Expected 12345, got %s", bi.StringValue())
		}
	})

	t.Run("RDouble to RBigInt truncates", func(t *testing.T) {
		rd := GetRDoubleValue(3.99)
		bi, err := rd.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "3" {
			t.Errorf("Expected 3 (truncated), got %s", bi.StringValue())
		}
	})

	t.Run("RDouble negative to RBigInt truncates toward zero", func(t *testing.T) {
		rd := GetRDoubleValue(-3.99)
		bi, err := rd.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "-3" {
			t.Errorf("Expected -3 (truncated), got %s", bi.StringValue())
		}
	})

	t.Run("RString numeric to RBigInt", func(t *testing.T) {
		rs := NewRString("9876543210")
		bi, err := rs.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "9876543210" {
			t.Errorf("Expected 9876543210, got %s", bi.StringValue())
		}
	})

	t.Run("RString large number to RBigInt", func(t *testing.T) {
		largeNum := "123456789012345678901234567890"
		rs := NewRString(largeNum)
		bi, err := rs.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != largeNum {
			t.Errorf("Expected %s, got %s", largeNum, bi.StringValue())
		}
	})

	t.Run("RString non-numeric returns error", func(t *testing.T) {
		rs := NewRString("not-a-number")
		_, err := rs.RBigIntValue()
		if err == nil {
			t.Error("Expected error for non-numeric string")
		}
	})

	t.Run("RNull to RBigInt returns zero", func(t *testing.T) {
		rn := GetRNull()
		bi, err := rn.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "0" {
			t.Errorf("Expected 0, got %s", bi.StringValue())
		}
	})

	t.Run("RBoolean true to RBigInt returns 1", func(t *testing.T) {
		bi, err := True.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "1" {
			t.Errorf("Expected 1, got %s", bi.StringValue())
		}
	})

	t.Run("RBoolean false to RBigInt returns 0", func(t *testing.T) {
		bi, err := False.RBigIntValue()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if bi.StringValue() != "0" {
			t.Errorf("Expected 0, got %s", bi.StringValue())
		}
	})
}

// =============================================================================
// Equals and Compare tests
// =============================================================================

func TestBigIntEquals(t *testing.T) {
	t.Run("equal values", func(t *testing.T) {
		a := GetRBigIntFromInt64(42)
		b := GetRBigIntFromInt64(42)
		eq, err := a.Equals(b)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !eq {
			t.Error("Expected 42 to equal 42")
		}
	})

	t.Run("unequal values", func(t *testing.T) {
		a := GetRBigIntFromInt64(42)
		b := GetRBigIntFromInt64(43)
		eq, err := a.Equals(b)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if eq {
			t.Error("Expected 42 to not equal 43")
		}
	})

	t.Run("compare with RInteger", func(t *testing.T) {
		a := GetRBigIntFromInt64(42)
		b := GetRIntegerValue(42)
		eq, err := a.Equals(b)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !eq {
			t.Error("Expected BigInt(42) to equal Integer(42)")
		}
	})

	t.Run("compare with RNull returns false", func(t *testing.T) {
		a := GetRBigIntFromInt64(0)
		b := GetRNull()
		eq, err := a.Equals(b)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if eq {
			t.Error("Expected BigInt(0) to not equal null")
		}
	})

	t.Run("large values equal", func(t *testing.T) {
		largeNum := "123456789012345678901234567890"
		a, _ := GetRBigIntFromString(largeNum)
		b, _ := GetRBigIntFromString(largeNum)
		eq, err := a.Equals(b)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !eq {
			t.Error("Expected large values to be equal")
		}
	})
}

func TestBigIntCompare(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int64
		expected int
	}{
		{"a < b", 10, 20, -1},
		{"a > b", 20, 10, 1},
		{"a == b", 15, 15, 0},
		{"negative comparison", -5, -10, 1},
		{"mixed signs", -5, 5, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := GetRBigIntFromInt64(tt.a)
			b := GetRBigIntFromInt64(tt.b)
			cmp, err := a.Compare(b)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if cmp != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, cmp)
			}
		})
	}
}

// =============================================================================
// Clone and other Object methods tests
// =============================================================================

func TestBigIntClone(t *testing.T) {
	bi := GetRBigIntFromInt64(42)

	// RClone should return same instance (immutable)
	cloned := bi.RClone()
	if cloned != bi {
		t.Error("RClone should return same instance for immutable RBigInt")
	}

	// Clone should return same instance
	cloned2, err := bi.Clone(nil)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}
	if cloned2 != bi {
		t.Error("Clone should return same instance for immutable RBigInt")
	}
}

func TestBigIntExecutable(t *testing.T) {
	bi := GetRBigIntFromInt64(42)

	if bi.IsExecutable() {
		t.Error("BigInt should not be executable")
	}

	if bi.GetExecutable() != bi {
		t.Error("GetExecutable should return self")
	}

	if bi.GetNonExecutable() != bi {
		t.Error("GetNonExecutable should return self")
	}
}

func TestBigIntBigIntValue(t *testing.T) {
	original := big.NewInt(12345)
	bi := NewRBigInt(original)

	// BigIntValue should return a copy
	copy := bi.BigIntValue()
	if copy.Cmp(original) != 0 {
		t.Errorf("Expected %s, got %s", original.String(), copy.String())
	}

	// Verify it's a copy by mutating
	copy.SetInt64(99999)
	if bi.StringValue() != "12345" {
		t.Error("BigIntValue should return a copy, not the original")
	}
}

func TestBigIntRStringValue(t *testing.T) {
	bi := GetRBigIntFromInt64(42)
	rs := bi.RStringValue()
	if rs.StringValue() != "42" {
		t.Errorf("Expected '42', got '%s'", rs.StringValue())
	}
}

func TestBigIntRDoubleValue(t *testing.T) {
	bi := GetRBigIntFromInt64(42)
	rd, err := bi.RDoubleValue()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	val, _ := rd.DoubleValue()
	if val != 42.0 {
		t.Errorf("Expected 42.0, got %f", val)
	}
}
