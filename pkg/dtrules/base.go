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
	"time"
)

// BaseObject provides default implementations for the Object interface.
// Embed this struct in concrete types to get default behavior.
type BaseObject struct{}

// Execute pushes this object onto the data stack by default.
func (b *BaseObject) Execute(state State) error {
	// This will be overridden - we can't push BaseObject itself
	return nil
}

// ArrayExecute pushes this object onto the data stack by default.
func (b *BaseObject) ArrayExecute(state State) error {
	// This will be overridden - we can't push BaseObject itself
	return nil
}

// GetExecutable returns this object by default (for non-executable objects).
func (b *BaseObject) GetExecutable() Object {
	return nil // Must be overridden
}

// GetNonExecutable returns this object by default.
func (b *BaseObject) GetNonExecutable() Object {
	return nil // Must be overridden
}

// IsExecutable returns false by default.
func (b *BaseObject) IsExecutable() bool {
	return false
}

// Equals returns true if objects are identical by default.
func (b *BaseObject) Equals(o Object) (bool, error) {
	return false, nil // Must be overridden
}

// Compare throws an error by default - not all types support comparison.
func (b *BaseObject) Compare(o Object) (int, error) {
	return 0, UndefinedError("Compare", "Objects do not support Compare")
}

// PostFix returns StringValue by default.
func (b *BaseObject) PostFix() string {
	return "" // Must be overridden
}

// Clone returns the object itself by default (for immutable types).
func (b *BaseObject) Clone(session Session) (Object, error) {
	return nil, nil // Must be overridden
}

// RClone returns the object itself by default.
func (b *BaseObject) RClone() Object {
	return nil // Must be overridden
}

// Default type conversion methods - all return errors

// IntValue returns an error by default.
func (b *BaseObject) IntValue() (int, error) {
	return 0, ConversionError("IntValue", "No Integer value exists for this type")
}

// LongValue returns an error by default.
func (b *BaseObject) LongValue() (int64, error) {
	return 0, ConversionError("LongValue", "No Long value exists for this type")
}

// DoubleValue returns an error by default.
func (b *BaseObject) DoubleValue() (float64, error) {
	return 0, ConversionError("DoubleValue", "No double value exists for this type")
}

// BooleanValue returns an error by default.
func (b *BaseObject) BooleanValue() (bool, error) {
	return false, ConversionError("BooleanValue", "No Boolean value exists for this type")
}

// TimeValue returns an error by default.
func (b *BaseObject) TimeValue() (time.Time, error) {
	return time.Time{}, ConversionError("TimeValue", "No Time value exists for this type")
}

// ArrayValue returns an error by default.
func (b *BaseObject) ArrayValue() ([]Object, error) {
	return nil, ConversionError("ArrayValue", "No Array value exists for this type")
}

// TableValue returns an error by default.
func (b *BaseObject) TableValue() (map[Object]Object, error) {
	return nil, ConversionError("TableValue", "No Table value exists for this type")
}

// RIntegerValue returns an error by default.
func (b *BaseObject) RIntegerValue() (*RInteger, error) {
	return nil, ConversionError("RIntegerValue", "No Integer value exists for this type")
}

// RDoubleValue returns an error by default.
func (b *BaseObject) RDoubleValue() (*RDouble, error) {
	return nil, ConversionError("RDoubleValue", "No Double value exists for this type")
}

// RBooleanValue tries to convert via BooleanValue by default.
func (b *BaseObject) RBooleanValue() (*RBoolean, error) {
	return nil, ConversionError("RBooleanValue", "No Boolean value exists for this type")
}

// RStringValue returns nil by default - must be overridden.
func (b *BaseObject) RStringValue() *RString {
	return nil // Must be overridden
}

// RNameValue returns an error by default.
func (b *BaseObject) RNameValue() (*RName, error) {
	return nil, ConversionError("RNameValue", "No Name value exists for this type")
}

// RArrayValue returns an error by default.
func (b *BaseObject) RArrayValue() (*RArray, error) {
	return nil, ConversionError("RArrayValue", "No Array value exists for this type")
}

// RTableValue returns an error by default.
func (b *BaseObject) RTableValue() (*RTable, error) {
	return nil, ConversionError("RTableValue", "No Table value exists for this type")
}

// RTimeValue returns an error by default.
func (b *BaseObject) RTimeValue() (*RDate, error) {
	return nil, ConversionError("RTimeValue", "No Time value exists for this type")
}

// REntityValue returns an error by default.
func (b *BaseObject) REntityValue() (Entity, error) {
	return nil, ConversionError("REntityValue", "No Entity value exists for this type")
}

// RBigIntValue returns an error by default.
func (b *BaseObject) RBigIntValue() (*RBigInt, error) {
	return nil, ConversionError("RBigIntValue", "cannot convert to bigint")
}

// RBytesValue returns an error by default.
func (b *BaseObject) RBytesValue() (*RBytes, error) {
	return nil, ConversionError("RBytesValue", "cannot convert to bytes")
}
