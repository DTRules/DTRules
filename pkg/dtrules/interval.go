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
	"time"
)

// IntervalUnit represents the unit of time for an interval (days, months, or years).
type IntervalUnit int

const (
	IntervalDays IntervalUnit = iota
	IntervalMonths
	IntervalYears
)

// RInterval represents a time interval with an amount and unit.
type RInterval struct {
	BaseObject
	amount int
	unit   IntervalUnit
}

// NewRInterval creates a new interval with the given amount and unit.
func NewRInterval(amount int, unit IntervalUnit) *RInterval {
	return &RInterval{
		amount: amount,
		unit:   unit,
	}
}

// GetAmount returns the interval amount.
func (i *RInterval) GetAmount() int {
	return i.amount
}

// GetUnit returns the interval unit.
func (i *RInterval) GetUnit() IntervalUnit {
	return i.unit
}

// Type returns the object type.
func (i *RInterval) Type() *RType {
	return TypeInterval
}

// Execute pushes this object onto the data stack.
func (i *RInterval) Execute(state State) error {
	return state.DataPush(i)
}

// ArrayExecute pushes this object onto the data stack.
func (i *RInterval) ArrayExecute(state State) error {
	return state.DataPush(i)
}

// GetExecutable returns this object.
func (i *RInterval) GetExecutable() Object {
	return i
}

// GetNonExecutable returns this object.
func (i *RInterval) GetNonExecutable() Object {
	return i
}

// IsExecutable returns false for intervals.
func (i *RInterval) IsExecutable() bool {
	return false
}

// Equals compares this interval to another object.
func (i *RInterval) Equals(other Object) (bool, error) {
	if other == nil {
		return false, nil
	}
	otherInterval, ok := other.(*RInterval)
	if !ok {
		return false, nil
	}
	return i.amount == otherInterval.amount && i.unit == otherInterval.unit, nil
}

// Compare compares this interval with another object.
func (i *RInterval) Compare(other Object) (int, error) {
	if other == nil {
		return 1, nil
	}
	otherInterval, ok := other.(*RInterval)
	if !ok {
		return 0, UndefinedError("Compare", "Cannot compare interval with non-interval")
	}
	// Convert both to days for comparison
	days1 := i.toDays()
	days2 := otherInterval.toDays()
	if days1 < days2 {
		return -1, nil
	}
	if days1 > days2 {
		return 1, nil
	}
	return 0, nil
}

// toDays converts an interval to approximate days for comparison.
func (i *RInterval) toDays() int {
	switch i.unit {
	case IntervalDays:
		return i.amount
	case IntervalMonths:
		return i.amount * 30 // Approximate
	case IntervalYears:
		return i.amount * 365 // Approximate
	default:
		return 0
	}
}

// StringValue returns the string representation of the interval.
func (i *RInterval) StringValue() string {
	unitStr := "days"
	switch i.unit {
	case IntervalMonths:
		unitStr = "months"
	case IntervalYears:
		unitStr = "years"
	}
	return fmt.Sprintf("%d %s", i.amount, unitStr)
}

// PostFix returns the postfix representation.
func (i *RInterval) PostFix() string {
	return i.StringValue()
}

// Clone returns this object (intervals are immutable).
func (i *RInterval) Clone(session Session) (Object, error) {
	return i, nil
}

// RClone returns this object.
func (i *RInterval) RClone() Object {
	return i
}

// FormattedValue returns the formatted string representation.
func (i *RInterval) FormattedValue(state State) string {
	return i.StringValue()
}

// IsNull returns false since intervals are never null.
func (i *RInterval) IsNull() bool {
	return false
}

// IntValue returns the amount as an integer.
func (i *RInterval) IntValue() (int, error) {
	return i.amount, nil
}

// LongValue returns the amount as an int64.
func (i *RInterval) LongValue() (int64, error) {
	return int64(i.amount), nil
}

// DoubleValue returns the amount as a double.
func (i *RInterval) DoubleValue() (float64, error) {
	return float64(i.amount), nil
}

// BooleanValue returns false for intervals.
func (i *RInterval) BooleanValue() (bool, error) {
	return false, nil
}

// TimeValue is not supported for intervals.
func (i *RInterval) TimeValue() (time.Time, error) {
	return time.Time{}, UndefinedError("TimeValue", "Intervals do not support TimeValue")
}

// ArrayValue is not supported for intervals.
func (i *RInterval) ArrayValue() ([]Object, error) {
	return nil, UndefinedError("ArrayValue", "Intervals do not support ArrayValue")
}

// TableValue is not supported for intervals.
func (i *RInterval) TableValue() (map[Object]Object, error) {
	return nil, UndefinedError("TableValue", "Intervals do not support TableValue")
}

// RIntegerValue returns the amount as an RInteger.
func (i *RInterval) RIntegerValue() (*RInteger, error) {
	return GetRIntegerValueFromInt(i.amount), nil
}

// RDoubleValue returns the amount as an RDouble.
func (i *RInterval) RDoubleValue() (*RDouble, error) {
	return GetRDoubleValue(float64(i.amount)), nil
}

// RBigIntValue returns the amount as an RBigInt.
func (i *RInterval) RBigIntValue() (*RBigInt, error) {
	return GetRBigIntFromInt64(int64(i.amount)), nil
}

// RBooleanValue returns False.
func (i *RInterval) RBooleanValue() (*RBoolean, error) {
	return False, nil
}

// RStringValue returns the string representation as an RString.
func (i *RInterval) RStringValue() *RString {
	return NewRString(i.StringValue())
}

// RNameValue is not supported for intervals.
func (i *RInterval) RNameValue() (*RName, error) {
	return nil, UndefinedError("RNameValue", "Intervals do not support RNameValue")
}

// RArrayValue is not supported for intervals.
func (i *RInterval) RArrayValue() (*RArray, error) {
	return nil, UndefinedError("RArrayValue", "Intervals do not support RArrayValue")
}

// RTableValue is not supported for intervals.
func (i *RInterval) RTableValue() (*RTable, error) {
	return nil, UndefinedError("RTableValue", "Intervals do not support RTableValue")
}

// RTimeValue is not supported for intervals.
func (i *RInterval) RTimeValue() (*RDate, error) {
	return nil, UndefinedError("RTimeValue", "Intervals do not support RTimeValue")
}

// REntityValue is not supported for intervals.
func (i *RInterval) REntityValue() (Entity, error) {
	return nil, UndefinedError("REntityValue", "Intervals do not support REntityValue")
}

// AsInterval attempts to convert an Object to an RInterval.
// Returns the interval and true if successful, nil and false otherwise.
func AsInterval(obj Object) (*RInterval, bool) {
	if obj == nil {
		return nil, false
	}
	interval, ok := obj.(*RInterval)
	return interval, ok
}
