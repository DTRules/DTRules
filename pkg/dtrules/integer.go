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
	"strconv"
	"strings"
)

// RInteger represents an integer value (int64) in the rules engine.
type RInteger struct {
	BaseObject
	value int64
}

// Cached common integer values
var (
	intMOne = &RInteger{value: -1}
	intZero = &RInteger{value: 0}
	intOne  = &RInteger{value: 1}
)

// GetRIntegerValue returns an RInteger for the given int64 value.
// Common values (-1, 0, 1) are cached.
func GetRIntegerValue(i int64) *RInteger {
	switch i {
	case -1:
		return intMOne
	case 0:
		return intZero
	case 1:
		return intOne
	default:
		return &RInteger{value: i}
	}
}

// GetRIntegerValueFromInt returns an RInteger for the given int value.
func GetRIntegerValueFromInt(i int) *RInteger {
	return GetRIntegerValue(int64(i))
}

// GetRIntegerValueFromFloat returns an RInteger for the given float64 value.
func GetRIntegerValueFromFloat(f float64) *RInteger {
	return GetRIntegerValue(int64(f))
}

// GetRIntegerValueFromString parses a string and returns an RInteger.
func GetRIntegerValueFromString(s string) (*RInteger, error) {
	v, err := ParseIntegerValue(s)
	if err != nil {
		return nil, err
	}
	return GetRIntegerValue(v), nil
}

// ParseIntegerValue parses a string to int64.
func ParseIntegerValue(s string) (int64, error) {
	if s == "" || strings.EqualFold(strings.TrimSpace(s), "null") {
		return 0, nil
	}
	s = strings.TrimSpace(s)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ConversionError("RInteger.ParseIntegerValue", "Could not convert the string '"+s+"' to an integer: "+err.Error())
	}
	return v, nil
}

// Type returns the type for this object.
func (r *RInteger) Type() *RType {
	return TypeInteger
}

// Execute pushes this object onto the data stack.
func (r *RInteger) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RInteger) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RInteger) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RInteger) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for integers.
func (r *RInteger) IsExecutable() bool {
	return false
}

// Equals compares this integer with another object.
func (r *RInteger) Equals(o Object) (bool, error) {
	// Null is never equal to an integer
	if o.Type() == TypeNull {
		return false, nil
	}
	v, err := o.IntValue()
	if err != nil {
		return false, err
	}
	return r.value == int64(v), nil
}

// Compare compares this integer with another object.
func (r *RInteger) Compare(o Object) (int, error) {
	v, err := o.IntValue()
	if err != nil {
		return 0, err
	}
	if r.value == int64(v) {
		return 0, nil
	}
	if r.value < int64(v) {
		return -1, nil
	}
	return 1, nil
}

// StringValue returns the string representation of this integer.
func (r *RInteger) StringValue() string {
	return strconv.FormatInt(r.value, 10)
}

// PostFix returns the postfix representation.
func (r *RInteger) PostFix() string {
	return r.StringValue()
}

// Clone returns this object (integers are immutable).
func (r *RInteger) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RInteger) RClone() Object {
	return r
}

// IntValue returns the int value.
func (r *RInteger) IntValue() (int, error) {
	return int(r.value), nil
}

// LongValue returns the int64 value.
func (r *RInteger) LongValue() (int64, error) {
	return r.value, nil
}

// DoubleValue returns the float64 value.
func (r *RInteger) DoubleValue() (float64, error) {
	return float64(r.value), nil
}

// RIntegerValue returns this object.
func (r *RInteger) RIntegerValue() (*RInteger, error) {
	return r, nil
}

// RDoubleValue returns an RDouble for this value.
func (r *RInteger) RDoubleValue() (*RDouble, error) {
	return GetRDoubleValue(float64(r.value)), nil
}

// RStringValue returns an RString for this value.
func (r *RInteger) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// Value returns the underlying int64 value.
func (r *RInteger) Value() int64 {
	return r.value
}

// String implements the Stringer interface.
func (r *RInteger) String() string {
	return r.StringValue()
}
