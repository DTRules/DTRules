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
	"math/big"
	"strings"
	"testing"
)

// mustFp is a test helper that parses a decimal literal into RFixed or fails.
func mustFp(t *testing.T, s string) *RFixed {
	t.Helper()
	r, err := GetRFixedFromString(s)
	if err != nil {
		t.Fatalf("GetRFixedFromString(%q) failed: %v", s, err)
	}
	return r
}

// =============================================================================
// Parsing and string round-trip
// =============================================================================

func TestFixedParsing(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0", "0.00000000"},
		{"1", "1.00000000"},
		{"-1", "-1.00000000"},
		{"1.5", "1.50000000"},
		{"-1.5", "-1.50000000"},
		{"1.50000000fp", "1.50000000"},
		{"1.50000000FP", "1.50000000"},
		{"+1.25", "1.25000000"},
		{"-0.00000001", "-0.00000001"},
		{"0.00000001", "0.00000001"},
		{"12.34567890", "12.34567890"},
		{"1.500000009", "1.50000000"}, // truncate extra digits toward zero
		{"-1.500000009", "-1.50000000"},
		{"1000000", "1000000.00000000"},
		{"", "0.00000000"},
		{"null", "0.00000000"},
		{" 1.5 ", "1.50000000"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			r, err := GetRFixedFromString(c.in)
			if err != nil {
				t.Fatalf("parse %q: %v", c.in, err)
			}
			if got := r.StringValue(); got != c.want {
				t.Errorf("StringValue(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestFixedParseInvalid(t *testing.T) {
	for _, in := range []string{"abc", "1.2.3", "--1", "1e5"} {
		if _, err := GetRFixedFromString(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestFixedStringRoundTrip(t *testing.T) {
	r := mustFp(t, "1234567890.12345678")
	s := r.StringValue()
	r2 := mustFp(t, s)
	if r.StringValue() != r2.StringValue() {
		t.Errorf("round-trip: %q != %q", r.StringValue(), r2.StringValue())
	}
}

func TestFixedPostFixSuffix(t *testing.T) {
	r := mustFp(t, "1.5")
	if got, want := r.PostFix(), "1.50000000fp"; got != want {
		t.Errorf("PostFix = %q, want %q", got, want)
	}
}

// =============================================================================
// Construction from numeric inputs
// =============================================================================

func TestFixedFromInt64(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0.00000000"},
		{1, "1.00000000"},
		{-1, "-1.00000000"},
		{1_000_000, "1000000.00000000"},
	}
	for _, c := range cases {
		r, err := GetRFixedFromInt64(c.in)
		if err != nil {
			t.Fatalf("GetRFixedFromInt64(%d): %v", c.in, err)
		}
		if got := r.StringValue(); got != c.want {
			t.Errorf("GetRFixedFromInt64(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFixedFromBigInt(t *testing.T) {
	v := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil) // 10^20
	r, err := GetRFixedFromBigInt(v)
	if err != nil {
		t.Fatalf("GetRFixedFromBigInt(10^20): %v", err)
	}
	if got := r.StringValue(); got != "100000000000000000000.00000000" {
		t.Errorf("got %q", got)
	}
}

func TestFixedFromBigIntOverflow(t *testing.T) {
	// 2^255 * 10^8 comfortably exceeds the 256-bit signed mantissa bound.
	huge := new(big.Int).Lsh(big.NewInt(1), 255)
	if _, err := GetRFixedFromBigInt(huge); err == nil {
		t.Error("expected overflow error for 2^255")
	}
}

func TestFixedCachedSingletons(t *testing.T) {
	a, _ := GetRFixedFromInt64(0)
	b, _ := GetRFixedFromInt64(0)
	if a != b {
		t.Error("expected cached zero to be the same pointer")
	}
	c, _ := GetRFixedFromInt64(1)
	d, _ := GetRFixedFromInt64(1)
	if c != d {
		t.Error("expected cached one to be the same pointer")
	}
}

// =============================================================================
// Arithmetic — exactness and truncation
// =============================================================================

func TestFixedAddSub(t *testing.T) {
	a := mustFp(t, "1.23456789")
	b := mustFp(t, "2.11111111")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if got := sum.StringValue(); got != "3.34567900" {
		t.Errorf("Add: got %q, want 3.34567900", got)
	}
	diff, err := sum.Sub(b)
	if err != nil {
		t.Fatal(err)
	}
	if got := diff.StringValue(); got != a.StringValue() {
		t.Errorf("(a+b)-b: got %q, want %q", got, a.StringValue())
	}
}

func TestFixedMulExact(t *testing.T) {
	a := mustFp(t, "1.5")
	b := mustFp(t, "1.5")
	r, err := a.Mul(b)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.StringValue(); got != "2.25000000" {
		t.Errorf("1.5 * 1.5: got %q, want 2.25000000", got)
	}
}

func TestFixedMulTruncateTowardZero(t *testing.T) {
	// 1.00000001 * 1.00000001 = 1.0000000200000001 → truncated to 1.00000002
	a := mustFp(t, "1.00000001")
	r, err := a.Mul(a)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.StringValue(); got != "1.00000002" {
		t.Errorf("truncation: got %q, want 1.00000002", got)
	}

	// Negative truncation must also move toward zero (not toward -inf).
	// -1.00000001 * 1.00000001 → -1.00000002 (same magnitude, negative)
	neg := mustFp(t, "-1.00000001")
	r2, err := neg.Mul(a)
	if err != nil {
		t.Fatal(err)
	}
	if got := r2.StringValue(); got != "-1.00000002" {
		t.Errorf("negative truncation: got %q, want -1.00000002", got)
	}
}

func TestFixedDivTruncate(t *testing.T) {
	one := mustFp(t, "1")
	three := mustFp(t, "3")
	r, err := one.Div(three)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.StringValue(); got != "0.33333333" {
		t.Errorf("1/3: got %q, want 0.33333333", got)
	}

	// -1/3 must truncate toward zero, not toward -inf.
	negOne := mustFp(t, "-1")
	r2, err := negOne.Div(three)
	if err != nil {
		t.Fatal(err)
	}
	if got := r2.StringValue(); got != "-0.33333333" {
		t.Errorf("-1/3: got %q, want -0.33333333 (truncate toward zero)", got)
	}
}

func TestFixedDivByZero(t *testing.T) {
	a := mustFp(t, "1")
	z := mustFp(t, "0")
	if _, err := a.Div(z); err == nil {
		t.Error("expected divide-by-zero error")
	}
}

func TestFixedNegAbsSign(t *testing.T) {
	a := mustFp(t, "1.5")
	if got := a.Neg().StringValue(); got != "-1.50000000" {
		t.Errorf("Neg: got %q", got)
	}
	b := mustFp(t, "-1.5")
	if got := b.Abs().StringValue(); got != "1.50000000" {
		t.Errorf("Abs: got %q", got)
	}
	if mustFp(t, "0").Sign() != 0 {
		t.Error("Sign(0) must be 0")
	}
	if mustFp(t, "1").Sign() != 1 {
		t.Error("Sign(1) must be 1")
	}
	if mustFp(t, "-1").Sign() != -1 {
		t.Error("Sign(-1) must be -1")
	}
}

func TestFixedTrunc(t *testing.T) {
	if got := mustFp(t, "1.99999999").Trunc().StringValue(); got != "1.00000000" {
		t.Errorf("Trunc(1.99999999): got %q", got)
	}
	if got := mustFp(t, "-1.99999999").Trunc().StringValue(); got != "-1.00000000" {
		t.Errorf("Trunc(-1.99999999) must truncate toward zero: got %q", got)
	}
}

// =============================================================================
// Overflow on arithmetic
// =============================================================================

func TestFixedOverflowOnMul(t *testing.T) {
	// Mul result is (a_m * b_m) / 10^8. With 10^8 ≈ 2^26.575, overflow
	// (result ≥ 2^255) requires a_m * b_m ≥ 2^281.5. For a = b, that means
	// a_m ≥ 2^141; use 2^200 to stay well above the bound.
	mantissa := new(big.Int).Lsh(big.NewInt(1), 200)
	a, err := GetRFixedFromMantissa(mantissa)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Mul(a); err == nil {
		t.Error("expected overflow error on huge mul")
	}
}

func TestFixedOverflowOnAdd(t *testing.T) {
	// Two mantissas each just below 2^255 → sum exceeds the bound.
	near := new(big.Int).Sub(fixedBound, big.NewInt(1))
	a, err := GetRFixedFromMantissa(near)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Add(a); err == nil {
		t.Error("expected overflow error on Add")
	}
}

func TestFixedOverflowOnSub(t *testing.T) {
	// Near-max-positive minus near-max-negative exceeds the bound: subtraction
	// needs its own overflow check because it can't be reached via Add alone.
	near := new(big.Int).Sub(fixedBound, big.NewInt(1))
	pos, err := GetRFixedFromMantissa(near)
	if err != nil {
		t.Fatal(err)
	}
	neg, err := GetRFixedFromMantissa(new(big.Int).Neg(near))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pos.Sub(neg); err == nil {
		t.Error("expected overflow error on Sub of near-max-pos − near-max-neg")
	}
}

func TestFixedMulTruncatesToZero(t *testing.T) {
	// 0.00000001 × 0.00000001 = 0.0000000000000001 → truncated to 0 on the
	// 10^-8 grid. Guards the "small but non-zero operands yield exactly zero"
	// boundary where sub-satoshi multiplication silently vanishes.
	a := mustFp(t, "0.00000001")
	r, err := a.Mul(a)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.StringValue(); got != "0.00000000" {
		t.Errorf("0.00000001² on 10^-8 grid: got %q, want 0.00000000", got)
	}
	if r.Sign() != 0 {
		t.Errorf("expected Sign 0 after truncate-to-zero, got %d", r.Sign())
	}
}

// =============================================================================
// Compare and Equals — mixed-type promotion
// =============================================================================

func TestFixedEqualsIntegerPromotion(t *testing.T) {
	two := mustFp(t, "2")
	if eq, err := two.Equals(GetRIntegerValue(2)); err != nil || !eq {
		t.Errorf("2fp == RInteger(2): eq=%v err=%v", eq, err)
	}
	if eq, err := two.Equals(GetRIntegerValue(3)); err != nil || eq {
		t.Errorf("2fp == RInteger(3): eq=%v err=%v", eq, err)
	}
}

func TestFixedEqualsBigIntPromotion(t *testing.T) {
	five := mustFp(t, "5")
	if eq, err := five.Equals(GetRBigIntFromInt64(5)); err != nil || !eq {
		t.Errorf("5fp == RBigInt(5): eq=%v err=%v", eq, err)
	}
}

func TestFixedEqualsDoubleRejected(t *testing.T) {
	one := mustFp(t, "1")
	_, err := one.Equals(GetRDoubleValue(1.0))
	if err == nil {
		t.Fatal("expected error comparing fixed to double without cvfp")
	}
	if !strings.Contains(err.Error(), "cvfp") {
		t.Errorf("error should mention cvfp: %v", err)
	}
}

func TestFixedCompare(t *testing.T) {
	a := mustFp(t, "1.5")
	b := mustFp(t, "2.5")
	if c, _ := a.Compare(b); c != -1 {
		t.Errorf("1.5 vs 2.5: got %d, want -1", c)
	}
	if c, _ := b.Compare(a); c != 1 {
		t.Errorf("2.5 vs 1.5: got %d, want 1", c)
	}
	if c, _ := a.Compare(a); c != 0 {
		t.Errorf("1.5 vs 1.5: got %d, want 0", c)
	}
}

// =============================================================================
// Conversions out of fixed
// =============================================================================

func TestFixedIntValueTruncate(t *testing.T) {
	if v, _ := mustFp(t, "1.99999999").IntValue(); v != 1 {
		t.Errorf("IntValue(1.99999999): got %d, want 1", v)
	}
	if v, _ := mustFp(t, "-1.99999999").IntValue(); v != -1 {
		t.Errorf("IntValue(-1.99999999): got %d, want -1 (truncate toward zero)", v)
	}
}

func TestFixedLongValueOverflow(t *testing.T) {
	// int64 max is 9.22e18; a 10^19 fixed value overflows.
	big10pow19 := new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)
	r, err := GetRFixedFromBigInt(big10pow19)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.LongValue(); err == nil {
		t.Error("expected int64 overflow error on LongValue")
	}
}

func TestFixedDoubleValue(t *testing.T) {
	f, _ := mustFp(t, "1.5").DoubleValue()
	if f != 1.5 {
		t.Errorf("DoubleValue(1.5): got %v, want 1.5", f)
	}
}

func TestFixedRBigIntValueTruncates(t *testing.T) {
	r := mustFp(t, "1234567890.87654321")
	bi, err := r.RBigIntValue()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := bi.StringValue(), "1234567890"; got != want {
		t.Errorf("RBigIntValue truncated: got %q, want %q", got, want)
	}
}

func TestFixedBooleanValue(t *testing.T) {
	if b, _ := mustFp(t, "0").BooleanValue(); b {
		t.Error("BooleanValue(0) must be false")
	}
	if b, _ := mustFp(t, "0.00000001").BooleanValue(); !b {
		t.Error("BooleanValue(0.00000001) must be true")
	}
}

// =============================================================================
// Type registration sanity
// =============================================================================

func TestFixedTypeRegistration(t *testing.T) {
	if TypeFixed == nil {
		t.Fatal("TypeFixed not registered")
	}
	if TypeFixed.String() != "fixed" {
		t.Errorf("TypeFixed name: got %q, want fixed", TypeFixed.String())
	}
	r := mustFp(t, "1")
	if r.Type() != TypeFixed {
		t.Error("RFixed.Type() must return TypeFixed")
	}
}
