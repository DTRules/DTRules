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
	"math/big"
	"strings"
)

// RBigInt represents a big integer value in the rules engine.
type RBigInt struct {
	BaseObject
	value *big.Int
}

// Cached common BigInt values
var (
	bigIntMOne = &RBigInt{value: big.NewInt(-1)}
	bigIntZero = &RBigInt{value: big.NewInt(0)}
	bigIntOne  = &RBigInt{value: big.NewInt(1)}
)

// NewRBigInt creates a new RBigInt, copying the provided value.
func NewRBigInt(v *big.Int) *RBigInt {
	if v == nil {
		return bigIntZero
	}
	return &RBigInt{value: new(big.Int).Set(v)}
}

// GetRBigIntValue returns an RBigInt for the given big.Int value.
// Common values (-1, 0, 1) are cached.
func GetRBigIntValue(v *big.Int) *RBigInt {
	if v == nil {
		return bigIntZero
	}
	if v.IsInt64() {
		switch v.Int64() {
		case -1:
			return bigIntMOne
		case 0:
			return bigIntZero
		case 1:
			return bigIntOne
		}
	}
	return &RBigInt{value: new(big.Int).Set(v)}
}

// GetRBigIntFromString parses a decimal string and returns an RBigInt.
func GetRBigIntFromString(s string) (*RBigInt, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "null") {
		return bigIntZero, nil
	}
	v := new(big.Int)
	_, ok := v.SetString(s, 10)
	if !ok {
		return nil, ConversionError("RBigInt.GetRBigIntFromString", "Could not convert the string '"+s+"' to a big integer")
	}
	return GetRBigIntValue(v), nil
}

// GetRBigIntFromInt64 converts an int64 to an RBigInt.
func GetRBigIntFromInt64(v int64) *RBigInt {
	switch v {
	case -1:
		return bigIntMOne
	case 0:
		return bigIntZero
	case 1:
		return bigIntOne
	default:
		return &RBigInt{value: big.NewInt(v)}
	}
}

// Type returns the type for this object.
func (r *RBigInt) Type() *RType {
	return TypeBigInt
}

// Execute pushes this object onto the data stack.
func (r *RBigInt) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RBigInt) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RBigInt) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RBigInt) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for big integers.
func (r *RBigInt) IsExecutable() bool {
	return false
}

// Equals compares this big integer with another object.
func (r *RBigInt) Equals(o Object) (bool, error) {
	// Null is never equal to a big integer
	if o.Type() == TypeNull {
		return false, nil
	}
	// Try to get the other value as a BigInt
	other, err := o.RBigIntValue()
	if err != nil {
		return false, err
	}
	return r.value.Cmp(other.value) == 0, nil
}

// Compare compares this big integer with another object.
func (r *RBigInt) Compare(o Object) (int, error) {
	other, err := o.RBigIntValue()
	if err != nil {
		return 0, err
	}
	return r.value.Cmp(other.value), nil
}

// StringValue returns the string representation of this big integer.
func (r *RBigInt) StringValue() string {
	return r.value.String()
}

// PostFix returns the postfix representation.
func (r *RBigInt) PostFix() string {
	return r.StringValue()
}

// Clone returns this object (big integers are immutable).
func (r *RBigInt) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RBigInt) RClone() Object {
	return r
}

// IntValue returns the int value, or an error if overflow.
func (r *RBigInt) IntValue() (int, error) {
	if !r.value.IsInt64() {
		return 0, ConversionError("RBigInt.IntValue", "BigInt value overflows int64")
	}
	v := r.value.Int64()
	// Check if it fits in an int (which may be 32-bit on some platforms)
	if int64(int(v)) != v {
		return 0, ConversionError("RBigInt.IntValue", "BigInt value overflows int")
	}
	return int(v), nil
}

// LongValue returns the int64 value, or an error if overflow.
func (r *RBigInt) LongValue() (int64, error) {
	if !r.value.IsInt64() {
		return 0, ConversionError("RBigInt.LongValue", "BigInt value overflows int64")
	}
	return r.value.Int64(), nil
}

// DoubleValue returns the float64 value.
func (r *RBigInt) DoubleValue() (float64, error) {
	f := new(big.Float).SetInt(r.value)
	result, _ := f.Float64()
	return result, nil
}

// BooleanValue returns true if the value is non-zero.
func (r *RBigInt) BooleanValue() (bool, error) {
	return r.value.Sign() != 0, nil
}

// RIntegerValue returns an RInteger for this value, or an error if overflow.
func (r *RBigInt) RIntegerValue() (*RInteger, error) {
	if !r.value.IsInt64() {
		return nil, ConversionError("RBigInt.RIntegerValue", "BigInt value overflows int64")
	}
	return GetRIntegerValue(r.value.Int64()), nil
}

// RDoubleValue returns an RDouble for this value.
func (r *RBigInt) RDoubleValue() (*RDouble, error) {
	f, err := r.DoubleValue()
	if err != nil {
		return nil, err
	}
	return GetRDoubleValue(f), nil
}

// RStringValue returns an RString for this value.
func (r *RBigInt) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// RBigIntValue returns this object.
func (r *RBigInt) RBigIntValue() (*RBigInt, error) {
	return r, nil
}

// BigIntValue returns a copy of the underlying big.Int value.
func (r *RBigInt) BigIntValue() *big.Int {
	return new(big.Int).Set(r.value)
}

// String implements the Stringer interface.
func (r *RBigInt) String() string {
	return r.StringValue()
}
