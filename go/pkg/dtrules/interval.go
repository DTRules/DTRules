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
)

// IntervalUnit represents the unit of time for an interval.
type IntervalUnit int

const (
	// IntervalDays represents an interval measured in days.
	IntervalDays IntervalUnit = iota
	// IntervalMonths represents an interval measured in months.
	IntervalMonths
	// IntervalYears represents an interval measured in years.
	IntervalYears
)

// String returns the string representation of the interval unit.
func (u IntervalUnit) String() string {
	switch u {
	case IntervalDays:
		return "days"
	case IntervalMonths:
		return "months"
	case IntervalYears:
		return "years"
	default:
		return "unknown"
	}
}

// RInterval represents a time interval in the rules engine.
type RInterval struct {
	BaseObject
	amount int
	unit   IntervalUnit
}

// NewRInterval creates a new RInterval with the given amount and unit.
func NewRInterval(amount int, unit IntervalUnit) *RInterval {
	return &RInterval{
		amount: amount,
		unit:   unit,
	}
}

// GetAmount returns the numeric amount of the interval.
func (r *RInterval) GetAmount() int {
	return r.amount
}

// GetUnit returns the unit of the interval.
func (r *RInterval) GetUnit() IntervalUnit {
	return r.unit
}

// Type returns the type for this object.
func (r *RInterval) Type() *RType {
	return TypeInterval
}

// Execute pushes this object onto the data stack.
func (r *RInterval) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RInterval) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RInterval) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RInterval) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for intervals.
func (r *RInterval) IsExecutable() bool {
	return false
}

// Equals compares this interval with another object.
func (r *RInterval) Equals(o Object) (bool, error) {
	other := AsInterval(o)
	if other == nil {
		return false, nil
	}
	return r.amount == other.amount && r.unit == other.unit, nil
}

// Compare compares this interval with another object.
// Intervals are compared by converting to a common unit (days).
func (r *RInterval) Compare(o Object) (int, error) {
	other := AsInterval(o)
	if other == nil {
		return 0, NewRulesError("Type Error", "Compare", "Cannot compare interval with non-interval")
	}

	// Convert both to days for comparison (approximate)
	days1 := r.toDays()
	days2 := other.toDays()

	if days1 < days2 {
		return -1, nil
	}
	if days1 > days2 {
		return 1, nil
	}
	return 0, nil
}

// toDays converts the interval to an approximate number of days.
func (r *RInterval) toDays() int {
	switch r.unit {
	case IntervalDays:
		return r.amount
	case IntervalMonths:
		return r.amount * 30 // Approximate
	case IntervalYears:
		return r.amount * 365 // Approximate
	default:
		return 0
	}
}

// StringValue returns the string representation of this interval.
func (r *RInterval) StringValue() string {
	return fmt.Sprintf("%d %s", r.amount, r.unit.String())
}

// PostFix returns the postfix representation.
func (r *RInterval) PostFix() string {
	unitName := ""
	switch r.unit {
	case IntervalDays:
		unitName = "days"
	case IntervalMonths:
		unitName = "months"
	case IntervalYears:
		unitName = "years"
	}
	return fmt.Sprintf("%d %s", r.amount, unitName)
}

// Clone returns a copy of this interval.
func (r *RInterval) Clone(session Session) (Object, error) {
	return &RInterval{
		amount: r.amount,
		unit:   r.unit,
	}, nil
}

// RClone returns a copy of this interval.
func (r *RInterval) RClone() Object {
	return &RInterval{
		amount: r.amount,
		unit:   r.unit,
	}
}

// IntValue returns the amount as an int.
func (r *RInterval) IntValue() (int, error) {
	return r.amount, nil
}

// LongValue returns the amount as an int64.
func (r *RInterval) LongValue() (int64, error) {
	return int64(r.amount), nil
}

// DoubleValue returns the amount as a float64.
func (r *RInterval) DoubleValue() (float64, error) {
	return float64(r.amount), nil
}

// RIntegerValue returns an RInteger for the amount.
func (r *RInterval) RIntegerValue() (*RInteger, error) {
	return GetRIntegerValue(int64(r.amount)), nil
}

// RDoubleValue returns an RDouble for the amount.
func (r *RInterval) RDoubleValue() (*RDouble, error) {
	return GetRDoubleValue(float64(r.amount)), nil
}

// RStringValue returns an RString for this interval.
func (r *RInterval) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// String implements the Stringer interface.
func (r *RInterval) String() string {
	return r.StringValue()
}

// AsInterval attempts to cast an Object to an *RInterval.
// Returns nil if the object is not an interval.
func AsInterval(o Object) *RInterval {
	if o == nil {
		return nil
	}
	interval, ok := o.(*RInterval)
	if !ok {
		return nil
	}
	return interval
}
