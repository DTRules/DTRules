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

// RDouble represents a double (float64) value in the rules engine.
type RDouble struct {
	BaseObject
	value float64
}

// Cached common double values
var (
	dblMOne = &RDouble{value: -1.0}
	dblZero = &RDouble{value: 0.0}
	dblOne  = &RDouble{value: 1.0}
)

// GetRDoubleValue returns an RDouble for the given float64 value.
// Common values (-1.0, 0.0, 1.0) are cached.
func GetRDoubleValue(v float64) *RDouble {
	switch v {
	case -1.0:
		return dblMOne
	case 0.0:
		return dblZero
	case 1.0:
		return dblOne
	default:
		return &RDouble{value: v}
	}
}

// GetRDoubleValueFromInt returns an RDouble for the given int value.
func GetRDoubleValueFromInt(v int) *RDouble {
	return GetRDoubleValue(float64(v))
}

// GetRDoubleValueFromLong returns an RDouble for the given int64 value.
func GetRDoubleValueFromLong(v int64) *RDouble {
	return GetRDoubleValue(float64(v))
}

// GetRDoubleValueFromString parses a string and returns an RDouble.
func GetRDoubleValueFromString(s string) (*RDouble, error) {
	v, err := ParseDoubleValue(s)
	if err != nil {
		return nil, err
	}
	return GetRDoubleValue(v), nil
}

// ParseDoubleValue parses a string to float64.
func ParseDoubleValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0.0, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, ConversionError("RDouble.ParseDoubleValue", "Could not convert the string '"+s+"' to a double: "+err.Error())
	}
	return v, nil
}

// Type returns the type for this object.
func (r *RDouble) Type() *RType {
	return TypeDouble
}

// Execute pushes this object onto the data stack.
func (r *RDouble) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RDouble) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RDouble) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RDouble) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for doubles.
func (r *RDouble) IsExecutable() bool {
	return false
}

// Equals compares this double with another object.
func (r *RDouble) Equals(o Object) (bool, error) {
	v, err := o.DoubleValue()
	if err != nil {
		return false, err
	}
	return r.value == v, nil
}

// Compare compares this double with another object.
func (r *RDouble) Compare(o Object) (int, error) {
	v, err := o.DoubleValue()
	if err != nil {
		return 0, err
	}
	if r.value == v {
		return 0, nil
	}
	if r.value < v {
		return -1, nil
	}
	return 1, nil
}

// StringValue returns the string representation of this double.
func (r *RDouble) StringValue() string {
	return strconv.FormatFloat(r.value, 'f', -1, 64)
}

// PostFix returns the postfix representation.
func (r *RDouble) PostFix() string {
	return r.StringValue()
}

// Clone returns this object (doubles are immutable).
func (r *RDouble) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RDouble) RClone() Object {
	return r
}

// IntValue returns the int value.
func (r *RDouble) IntValue() (int, error) {
	return int(r.value), nil
}

// LongValue returns the int64 value.
func (r *RDouble) LongValue() (int64, error) {
	return int64(r.value), nil
}

// DoubleValue returns the float64 value.
func (r *RDouble) DoubleValue() (float64, error) {
	return r.value, nil
}

// RDoubleValue returns this object.
func (r *RDouble) RDoubleValue() (*RDouble, error) {
	return r, nil
}

// RIntegerValue returns an RInteger for this value.
func (r *RDouble) RIntegerValue() (*RInteger, error) {
	return GetRIntegerValue(int64(r.value)), nil
}

// RStringValue returns an RString for this value.
func (r *RDouble) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// Value returns the underlying float64 value.
func (r *RDouble) Value() float64 {
	return r.value
}

// String implements the Stringer interface.
func (r *RDouble) String() string {
	return r.StringValue()
}
