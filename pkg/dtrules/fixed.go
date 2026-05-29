// Copyright 2004-2011 DTRules.com, Inc.
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
	"fmt"
	"math/big"
	"strings"
)

// FixedDecimals is the implicit decimal scale of the fixed-point type.
// Every RFixed value is stored as a signed 256-bit integer mantissa divided
// by 10^FixedDecimals. Matches the 8-decimal convention used by most
// blockchain token systems.
const FixedDecimals = 8

var (
	fixedScaleBig = new(big.Int).SetInt64(100_000_000)

	// fixedBound is 2^255. A mantissa m is in range iff |m| < fixedBound,
	// giving symmetric signed 256-bit bounds (we trade the single value
	// -2^255 for symmetric Neg/Abs semantics).
	fixedBound = new(big.Int).Lsh(big.NewInt(1), 255)
)

// RFixed represents a fixed-point decimal value with 8 decimals of precision,
// backed by a 256-bit signed mantissa.
//
// Every operation either yields a value on the 10^-8 grid with exact
// addition/subtraction and truncate-toward-zero on multiply/divide, or
// returns an error if the result overflows the 256-bit range.
type RFixed struct {
	BaseObject
	mantissa *big.Int // invariant: |mantissa| < 2^255
}

// Cached common fixed values. Built with direct struct literals (rather
// than routed through newRFixedFromMantissa) to avoid a package-init cycle
// with the nil guard in the constructor.
var (
	fixedMOne = &RFixed{mantissa: new(big.Int).Neg(fixedScaleBig)}
	fixedZero = &RFixed{mantissa: big.NewInt(0)}
	fixedOne  = &RFixed{mantissa: new(big.Int).Set(fixedScaleBig)}
)

// newRFixedFromMantissa wraps a raw 10^-8-scaled mantissa after bounds checking.
func newRFixedFromMantissa(m *big.Int) (*RFixed, error) {
	if m == nil {
		return &RFixed{mantissa: big.NewInt(0)}, nil
	}
	if err := checkFixedRange(m); err != nil {
		return nil, err
	}
	return &RFixed{mantissa: new(big.Int).Set(m)}, nil
}

func checkFixedRange(m *big.Int) error {
	abs := new(big.Int).Abs(m)
	if abs.Cmp(fixedBound) >= 0 {
		return ConversionError("RFixed", "fixed-point value exceeds 256-bit signed range")
	}
	return nil
}

// GetRFixedFromMantissa constructs an RFixed from a raw mantissa (value * 10^8).
func GetRFixedFromMantissa(m *big.Int) (*RFixed, error) {
	return newRFixedFromMantissa(m)
}

// GetRFixedFromInt64 constructs an RFixed representing the integer v exactly.
func GetRFixedFromInt64(v int64) (*RFixed, error) {
	switch v {
	case -1:
		return fixedMOne, nil
	case 0:
		return fixedZero, nil
	case 1:
		return fixedOne, nil
	}
	m := new(big.Int).Mul(big.NewInt(v), fixedScaleBig)
	return newRFixedFromMantissa(m)
}

// GetRFixedFromBigInt constructs an RFixed representing the integer v exactly.
// Returns an error if v * 10^8 would exceed the 256-bit mantissa range.
func GetRFixedFromBigInt(v *big.Int) (*RFixed, error) {
	if v == nil {
		return fixedZero, nil
	}
	m := new(big.Int).Mul(v, fixedScaleBig)
	return newRFixedFromMantissa(m)
}

// GetRFixedFromString parses a decimal string into an RFixed.
// Accepts forms like "1", "1.5", "-0.00000001", "1.50000000fp".
// Fractional digits beyond the 8-decimal grid are truncated toward zero.
func GetRFixedFromString(s string) (*RFixed, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "null") {
		return fixedZero, nil
	}
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "fp") {
		s = s[:len(s)-2]
		if s == "" {
			return nil, ConversionError("RFixed.GetRFixedFromString", "'fp' suffix with no numeric body")
		}
	}

	negative := false
	switch {
	case strings.HasPrefix(s, "-"):
		negative = true
		s = s[1:]
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	}

	wholeStr, fracStr := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		wholeStr = s[:dot]
		fracStr = s[dot+1:]
	}
	if wholeStr == "" {
		wholeStr = "0"
	}
	if !allDigits(wholeStr) || !allDigits(fracStr) {
		return nil, ConversionError("RFixed.GetRFixedFromString", "invalid fixed-point literal: "+s)
	}

	if len(fracStr) > FixedDecimals {
		fracStr = fracStr[:FixedDecimals]
	} else if len(fracStr) < FixedDecimals {
		fracStr += strings.Repeat("0", FixedDecimals-len(fracStr))
	}

	whole := new(big.Int)
	if _, ok := whole.SetString(wholeStr, 10); !ok {
		return nil, ConversionError("RFixed.GetRFixedFromString", "invalid fixed-point literal: "+s)
	}
	frac := new(big.Int)
	if _, ok := frac.SetString(fracStr, 10); !ok {
		return nil, ConversionError("RFixed.GetRFixedFromString", "invalid fixed-point literal: "+s)
	}

	mantissa := new(big.Int).Mul(whole, fixedScaleBig)
	mantissa.Add(mantissa, frac)
	if negative {
		mantissa.Neg(mantissa)
	}
	return newRFixedFromMantissa(mantissa)
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// PromoteToRFixed promotes an integer or bigint operand to RFixed for mixed
// arithmetic. Double values must be promoted via an explicit cvfp cast to
// avoid silently snapping non-representable floats (e.g. 0.1) onto the grid.
func PromoteToRFixed(o Object) (*RFixed, error) {
	switch v := o.(type) {
	case *RFixed:
		return v, nil
	case *RInteger:
		return GetRFixedFromInt64(v.Value())
	case *RBigInt:
		return GetRFixedFromBigInt(v.BigIntValue())
	case *RDouble:
		return nil, ConversionError("RFixed.promote",
			"cannot promote double to fixed implicitly; use cvfp to opt in")
	default:
		return nil, ConversionError("RFixed.promote",
			"cannot promote "+o.Type().String()+" to fixed")
	}
}

// Add returns r + other. Exact; errors only on 256-bit overflow.
func (r *RFixed) Add(other *RFixed) (*RFixed, error) {
	m := new(big.Int).Add(r.mantissa, other.mantissa)
	return newRFixedFromMantissa(m)
}

// Sub returns r - other. Exact; errors only on 256-bit overflow.
func (r *RFixed) Sub(other *RFixed) (*RFixed, error) {
	m := new(big.Int).Sub(r.mantissa, other.mantissa)
	return newRFixedFromMantissa(m)
}

// Mul returns r * other with truncation toward zero on the dropped 10^-8
// digits. Errors if the result overflows the 256-bit mantissa.
func (r *RFixed) Mul(other *RFixed) (*RFixed, error) {
	product := new(big.Int).Mul(r.mantissa, other.mantissa)
	m := new(big.Int).Quo(product, fixedScaleBig)
	return newRFixedFromMantissa(m)
}

// Div returns r / other with truncation toward zero. Errors on divide-by-zero
// or if the result overflows.
func (r *RFixed) Div(other *RFixed) (*RFixed, error) {
	if other.mantissa.Sign() == 0 {
		return nil, ConversionError("RFixed.Div", "division by zero")
	}
	scaled := new(big.Int).Mul(r.mantissa, fixedScaleBig)
	m := new(big.Int).Quo(scaled, other.mantissa)
	return newRFixedFromMantissa(m)
}

// DivRoundHalfUp returns r / other rounded to the nearest representable
// fp value on the 10^-8 grid, with halves rounding away from zero
// (commonly called "round half up").
//
// Why this exists: the pure-fp idiom (a*b + c/2)/c that produces
// round-half-up at integer precision adds c/2 in **value** units, which
// for fp is "half of the result's last representable digit" = 0.5 × 10^-8.
// That sub-grid offset cannot be expressed by adding any fp value to the
// numerator before dividing. This operator does the rounding at the
// mantissa level, where the grid is integer (1 mantissa unit = 10^-8
// value), so "half" is well-defined and resolvable.
//
// Implementation: with a = r.mantissa, b = other.mantissa, the scaled
// numerator is a*10^8 and the divisor is b. Round-half-away-from-zero of
// (a*10^8) / b is floor((2·|a|·10^8 + |b|) / (2·|b|)) with the sign
// recovered from sign(a)·sign(b). Truncating bigint.Quo on the positive
// numerator/divisor gives floor, so this is exact.
//
// Errors on divide-by-zero or if the rounded result overflows the 256-bit
// mantissa range.
func (r *RFixed) DivRoundHalfUp(other *RFixed) (*RFixed, error) {
	if other.mantissa.Sign() == 0 {
		return nil, ConversionError("RFixed.DivRoundHalfUp", "division by zero")
	}
	// Magnitude in nonneg arithmetic; sign reapplied at the end.
	absA := new(big.Int).Abs(r.mantissa)
	absB := new(big.Int).Abs(other.mantissa)

	// numerator = 2 * |a| * 10^8 + |b|; denominator = 2 * |b|.
	// floor(numerator / denominator) on nonneg ints = round-half-up
	// magnitude. For a half-tie, numerator == k * denom exactly, so floor
	// yields k — i.e., rounds away from zero by magnitude.
	num := new(big.Int).Mul(absA, fixedScaleBig)
	num.Lsh(num, 1) // *2
	num.Add(num, absB)
	den := new(big.Int).Lsh(absB, 1)
	m := new(big.Int).Quo(num, den)

	if r.mantissa.Sign()*other.mantissa.Sign() < 0 {
		m.Neg(m)
	}
	return newRFixedFromMantissa(m)
}

// Neg returns -r. Symmetric bounds guarantee success.
func (r *RFixed) Neg() *RFixed {
	result, _ := newRFixedFromMantissa(new(big.Int).Neg(r.mantissa))
	return result
}

// Abs returns |r|. Symmetric bounds guarantee success.
func (r *RFixed) Abs() *RFixed {
	result, _ := newRFixedFromMantissa(new(big.Int).Abs(r.mantissa))
	return result
}

// Trunc returns the integer part of r as an RFixed (truncated toward zero).
func (r *RFixed) Trunc() *RFixed {
	whole := new(big.Int).Quo(r.mantissa, fixedScaleBig)
	m := new(big.Int).Mul(whole, fixedScaleBig)
	result, _ := newRFixedFromMantissa(m)
	return result
}

// Sign returns -1, 0, or +1 for the mantissa.
func (r *RFixed) Sign() int {
	return r.mantissa.Sign()
}

// Mantissa returns a copy of the underlying scaled mantissa.
func (r *RFixed) Mantissa() *big.Int {
	return new(big.Int).Set(r.mantissa)
}

// Type returns the RType for fixed-point values.
func (r *RFixed) Type() *RType {
	return TypeFixed
}

// Execute pushes this object onto the data stack.
func (r *RFixed) Execute(state State) error {
	return state.DataPush(r)
}

// ArrayExecute pushes this object onto the data stack.
func (r *RFixed) ArrayExecute(state State) error {
	return state.DataPush(r)
}

// GetExecutable returns this object.
func (r *RFixed) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RFixed) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for fixed values.
func (r *RFixed) IsExecutable() bool {
	return false
}

// Equals compares this fixed value with another object. Integer and bigint
// are promoted to fixed exactly; double operands return an error (use cvfp).
func (r *RFixed) Equals(o Object) (bool, error) {
	if o.Type() == TypeNull {
		return false, nil
	}
	other, err := PromoteToRFixed(o)
	if err != nil {
		return false, err
	}
	return r.mantissa.Cmp(other.mantissa) == 0, nil
}

// Compare compares this fixed value with another object.
func (r *RFixed) Compare(o Object) (int, error) {
	other, err := PromoteToRFixed(o)
	if err != nil {
		return 0, err
	}
	return r.mantissa.Cmp(other.mantissa), nil
}

// StringValue renders the value with exactly FixedDecimals fractional digits.
func (r *RFixed) StringValue() string {
	sign := ""
	abs := new(big.Int).Abs(r.mantissa)
	if r.mantissa.Sign() < 0 {
		sign = "-"
	}
	whole := new(big.Int).Quo(abs, fixedScaleBig)
	frac := new(big.Int).Rem(abs, fixedScaleBig)
	return fmt.Sprintf("%s%s.%08d", sign, whole.String(), frac.Int64())
}

// PostFix returns the postfix literal form, suffixed with "fp" so the
// compiler round-trips the value back to RFixed on re-parse.
func (r *RFixed) PostFix() string {
	return r.StringValue() + "fp"
}

// Clone returns this object (RFixed is immutable).
func (r *RFixed) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RFixed) RClone() Object {
	return r
}

// IntValue returns the integer part truncated toward zero, errors on int overflow.
func (r *RFixed) IntValue() (int, error) {
	whole := new(big.Int).Quo(r.mantissa, fixedScaleBig)
	if !whole.IsInt64() {
		return 0, ConversionError("RFixed.IntValue", "fixed value overflows int64")
	}
	v := whole.Int64()
	if int64(int(v)) != v {
		return 0, ConversionError("RFixed.IntValue", "fixed value overflows int")
	}
	return int(v), nil
}

// LongValue returns the integer part truncated toward zero, errors on int64 overflow.
func (r *RFixed) LongValue() (int64, error) {
	whole := new(big.Int).Quo(r.mantissa, fixedScaleBig)
	if !whole.IsInt64() {
		return 0, ConversionError("RFixed.LongValue", "fixed value overflows int64")
	}
	return whole.Int64(), nil
}

// DoubleValue returns the value as float64. Lossy for large or
// high-precision values; provided for display/interop only.
func (r *RFixed) DoubleValue() (float64, error) {
	f := new(big.Float).SetInt(r.mantissa)
	scale := new(big.Float).SetInt(fixedScaleBig)
	f.Quo(f, scale)
	result, _ := f.Float64()
	return result, nil
}

// BooleanValue returns true if the value is non-zero.
func (r *RFixed) BooleanValue() (bool, error) {
	return r.mantissa.Sign() != 0, nil
}

// RIntegerValue returns the integer part as RInteger (truncates toward zero).
func (r *RFixed) RIntegerValue() (*RInteger, error) {
	v, err := r.LongValue()
	if err != nil {
		return nil, err
	}
	return GetRIntegerValue(v), nil
}

// RDoubleValue returns the value as RDouble.
func (r *RFixed) RDoubleValue() (*RDouble, error) {
	f, err := r.DoubleValue()
	if err != nil {
		return nil, err
	}
	return GetRDoubleValue(f), nil
}

// RBigIntValue returns the integer part as RBigInt (truncates toward zero).
func (r *RFixed) RBigIntValue() (*RBigInt, error) {
	whole := new(big.Int).Quo(r.mantissa, fixedScaleBig)
	return GetRBigIntValue(whole), nil
}

// RStringValue returns the decimal string form.
func (r *RFixed) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// String implements the Stringer interface.
func (r *RFixed) String() string {
	return r.StringValue() + "fp"
}
