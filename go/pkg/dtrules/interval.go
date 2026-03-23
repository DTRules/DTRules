// Copyright 2026 Paul Snow
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

// IntervalUnit represents the unit of an interval (days, months, years)
type IntervalUnit int

const (
	IntervalDays IntervalUnit = iota
	IntervalMonths
	IntervalYears
)

// Interval represents a time interval
type Interval interface {
	Object
	GetAmount() int
	GetUnit() IntervalUnit
}

// RInterval implements the Interval interface
type RInterval struct {
	amount int
	unit   IntervalUnit
}

// NewRInterval creates a new interval
func NewRInterval(amount int, unit IntervalUnit) *RInterval {
	return &RInterval{
		amount: amount,
		unit:   unit,
	}
}

// GetAmount returns the interval amount
func (i *RInterval) GetAmount() int {
	return i.amount
}

// GetUnit returns the interval unit
func (i *RInterval) GetUnit() IntervalUnit {
	return i.unit
}

// AsInterval converts an Object to an Interval if possible
func AsInterval(obj Object) Interval {
	if interval, ok := obj.(Interval); ok {
		return interval
	}
	return nil
}

// Implement Object interface
func (i *RInterval) StringValue() string {
	units := []string{"days", "months", "years"}
	unitStr := "unknown"
	if int(i.unit) < len(units) {
		unitStr = units[i.unit]
	}
	return fmt.Sprintf("%d %s", i.amount, unitStr)
}

func (i *RInterval) PostFix() string {
	return i.StringValue()
}

func (i *RInterval) Type() *RType {
	return TypeInterval
}

func (i *RInterval) TypeId() int {
	return int(TypeInterval.GetID())
}

func (i *RInterval) BooleanValue() (bool, error) {
	return false, ConversionError("RInterval.BooleanValue", "Cannot convert interval to boolean")
}

func (i *RInterval) IntValue() (int, error) {
	return i.amount, nil
}

func (i *RInterval) LongValue() (int64, error) {
	return int64(i.amount), nil
}

func (i *RInterval) IntegerValue() (int64, error) {
	return int64(i.amount), nil
}

func (i *RInterval) DoubleValue() (float64, error) {
	return float64(i.amount), nil
}

func (i *RInterval) TableValue() (map[Object]Object, error) {
	return nil, ConversionError("RInterval.TableValue", "Cannot convert interval to table")
}

func (i *RInterval) RIntegerValue() (*RInteger, error) {
	return GetRIntegerValue(int64(i.amount)), nil
}

func (i *RInterval) RDoubleValue() (*RDouble, error) {
	return GetRDoubleValue(float64(i.amount)), nil
}

func (i *RInterval) RBooleanValue() (*RBoolean, error) {
	return nil, ConversionError("RInterval.RBooleanValue", "Cannot convert interval to boolean")
}

func (i *RInterval) RStringValue() *RString {
	return GetRString(i.StringValue())
}

func (i *RInterval) RNameValue() (*RName, error) {
	return nil, ConversionError("RInterval.RNameValue", "Cannot convert interval to name")
}

func (i *RInterval) RArrayValue() (*RArray, error) {
	return nil, ConversionError("RInterval.RArrayValue", "Cannot convert interval to array")
}

func (i *RInterval) RTableValue() (*RTable, error) {
	return nil, ConversionError("RInterval.RTableValue", "Cannot convert interval to table")
}

func (i *RInterval) ArrayValue() ([]Object, error) {
	return nil, ConversionError("RInterval.ArrayValue", "Cannot convert interval to array")
}

func (i *RInterval) REntityValue() (Entity, error) {
	return nil, ConversionError("RInterval.REntityValue", "Cannot convert interval to entity")
}

func (i *RInterval) TimeValue() (time.Time, error) {
	return time.Time{}, ConversionError("RInterval.TimeValue", "Cannot convert interval to time")
}

func (i *RInterval) RTimeValue() (*RDate, error) {
	return nil, ConversionError("RInterval.RTimeValue", "Cannot convert interval to time")
}

func (i *RInterval) Execute(state State) error {
	return state.DataPush(i)
}

func (i *RInterval) ArrayExecute(state State) error {
	return state.DataPush(i)
}

func (i *RInterval) GetExecutable() Object {
	return i
}

func (i *RInterval) GetNonExecutable() Object {
	return i
}

func (i *RInterval) IsExecutable() bool {
	return false
}

func (i *RInterval) Clone(session Session) (Object, error) {
	return i, nil
}

func (i *RInterval) RClone() Object {
	return i
}

func (i *RInterval) Equals(other Object) (bool, error) {
	if other == nil {
		return false, nil
	}
	if otherInterval, ok := other.(*RInterval); ok {
		return i.amount == otherInterval.amount && i.unit == otherInterval.unit, nil
	}
	return false, nil
}

func (i *RInterval) Compare(other Object) (int, error) {
	if otherInterval, ok := other.(*RInterval); ok {
		// Compare intervals by converting to days (approximate)
		days1 := i.amount
		if i.unit == IntervalMonths {
			days1 *= 30
		} else if i.unit == IntervalYears {
			days1 *= 365
		}

		days2 := otherInterval.amount
		if otherInterval.unit == IntervalMonths {
			days2 *= 30
		} else if otherInterval.unit == IntervalYears {
			days2 *= 365
		}

		if days1 < days2 {
			return -1, nil
		} else if days1 > days2 {
			return 1, nil
		}
		return 0, nil
	}
	return 0, ConversionError("RInterval.Compare", "Cannot compare interval with non-interval")
}
