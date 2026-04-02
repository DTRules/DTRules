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
	"strings"
)

// RBoolean represents a boolean value in the rules engine.
// Only two instances exist: True and False.
type RBoolean struct {
	BaseObject
	value bool
}

// Singleton boolean values
var (
	True  = &RBoolean{value: true}
	False = &RBoolean{value: false}
)

// GetRBoolean returns the RBoolean for the given boolean value.
func GetRBoolean(value bool) *RBoolean {
	if value {
		return True
	}
	return False
}

// GetRBooleanFromString parses a string and returns an RBoolean.
func GetRBooleanFromString(value string) (*RBoolean, error) {
	b, err := ParseBooleanValue(value)
	if err != nil {
		return nil, err
	}
	return GetRBoolean(b), nil
}

// ParseBooleanValue provides all supported conversions of a string to boolean.
func ParseBooleanValue(value string) (bool, error) {
	v := strings.TrimSpace(value)
	switch strings.ToLower(v) {
	case "true", "y", "t", "yes":
		return true, nil
	case "false", "n", "f", "no":
		return false, nil
	default:
		return false, TypeCheckError("String Conversion to Boolean", "No boolean value for this string: "+value)
	}
}

// Type returns the type for this object.
func (r *RBoolean) Type() *RType {
	return TypeBoolean
}

// Execute pushes this object onto the data stack.
func (r *RBoolean) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RBoolean) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RBoolean) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RBoolean) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for booleans.
func (r *RBoolean) IsExecutable() bool {
	return false
}

// Equals compares this boolean with another object.
func (r *RBoolean) Equals(o Object) (bool, error) {
	v, err := o.BooleanValue()
	if err != nil {
		return false, err
	}
	return r.value == v, nil
}

// Compare is not supported for booleans.
func (r *RBoolean) Compare(o Object) (int, error) {
	return 0, UndefinedError("Compare", "Boolean Objects do not support Compare")
}

// StringValue returns the string representation of this boolean.
func (r *RBoolean) StringValue() string {
	if r.value {
		return "true"
	}
	return "false"
}

// PostFix returns the postfix representation.
func (r *RBoolean) PostFix() string {
	return r.StringValue()
}

// Clone returns this object (booleans are singletons).
func (r *RBoolean) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RBoolean) RClone() Object {
	return r
}

// BooleanValue returns the boolean value.
func (r *RBoolean) BooleanValue() (bool, error) {
	return r.value, nil
}

// RBooleanValue returns this object.
func (r *RBoolean) RBooleanValue() (*RBoolean, error) {
	return r, nil
}

// RStringValue returns an RString for this value.
func (r *RBoolean) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// RBigIntValue returns an RBigInt (1 for true, 0 for false).
func (r *RBoolean) RBigIntValue() (*RBigInt, error) {
	if r.value {
		return GetRBigIntFromInt64(1), nil
	}
	return GetRBigIntFromInt64(0), nil
}

// Value returns the underlying boolean value.
func (r *RBoolean) Value() bool {
	return r.value
}

// String implements the Stringer interface.
func (r *RBoolean) String() string {
	return r.StringValue()
}
