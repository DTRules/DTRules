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

// DateFormat is the default format for date strings.
const DateFormat = "01/02/2006 15:04:05.0000"

// RDate represents a date/time value in the rules engine.
type RDate struct {
	BaseObject
	time time.Time
}

// GetRTime returns an RDate for the given time.Time value.
func GetRTime(t time.Time) *RDate {
	return &RDate{time: t}
}

// GetRDate parses a string and returns an RDate.
// Uses the session's date parser for parsing.
func GetRDate(session Session, s string) (*RDate, error) {
	d, err := session.GetDateParser().GetDate(s)
	if err != nil {
		return nil, NewRulesError("Bad Date Format", "getRDate", "Could not parse: '"+s+"' as a Date or Time value")
	}
	return &RDate{time: d}, nil
}

// Type returns the type for this object.
func (r *RDate) Type() *RType {
	return TypeDate
}

// Execute pushes this object onto the data stack.
func (r *RDate) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RDate) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RDate) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RDate) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for dates.
func (r *RDate) IsExecutable() bool {
	return false
}

// Equals compares this date with another object.
func (r *RDate) Equals(o Object) (bool, error) {
	t, err := o.TimeValue()
	if err != nil {
		return false, err
	}
	return r.time.Equal(t), nil
}

// Compare compares this date with another object.
func (r *RDate) Compare(o Object) (int, error) {
	t, err := o.TimeValue()
	if err != nil {
		return 0, err
	}
	if r.time.Before(t) {
		return -1, nil
	}
	if r.time.After(t) {
		return 1, nil
	}
	return 0, nil
}

// StringValue returns the string representation of this date.
// Pure dates (midnight UTC) are serialized as YYYY-MM-DD to preserve existing
// XML/Excel output. Timestamps with a non-zero time component use RFC3339Nano.
func (r *RDate) StringValue() string {
	t := r.time.UTC()
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339Nano)
}

// PostFix returns the postfix representation.
func (r *RDate) PostFix() string {
	return "\"" + r.StringValue() + "\" cvd"
}

// Clone returns this object (dates are immutable).
func (r *RDate) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RDate) RClone() Object {
	return r
}

// LongValue returns the Unix timestamp in milliseconds.
func (r *RDate) LongValue() (int64, error) {
	return r.time.UnixMilli(), nil
}

// DoubleValue returns the Unix timestamp in milliseconds as float64.
func (r *RDate) DoubleValue() (float64, error) {
	return float64(r.time.UnixMilli()), nil
}

// TimeValue returns the time.Time value.
func (r *RDate) TimeValue() (time.Time, error) {
	return r.time, nil
}

// RTimeValue returns this object.
func (r *RDate) RTimeValue() (*RDate, error) {
	return r, nil
}

// RIntegerValue returns an RInteger for the Unix timestamp.
func (r *RDate) RIntegerValue() (*RInteger, error) {
	return GetRIntegerValue(r.time.UnixMilli()), nil
}

// RDoubleValue returns an RDouble for the Unix timestamp.
func (r *RDate) RDoubleValue() (*RDouble, error) {
	return GetRDoubleValue(float64(r.time.UnixMilli())), nil
}

// RStringValue returns an RString for this date.
func (r *RDate) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// RBigIntValue returns an RBigInt for the Unix timestamp in milliseconds.
func (r *RDate) RBigIntValue() (*RBigInt, error) {
	return GetRBigIntFromInt64(r.time.UnixMilli()), nil
}

// Time returns the underlying time.Time value.
func (r *RDate) Time() time.Time {
	return r.time
}

// GetYear returns the year component.
func (r *RDate) GetYear() int {
	return r.time.Year()
}

// GetMonth returns the month component (1-12).
func (r *RDate) GetMonth() int {
	return int(r.time.Month())
}

// GetDay returns the day of month component.
func (r *RDate) GetDay() int {
	return r.time.Day()
}

// String implements the Stringer interface.
func (r *RDate) String() string {
	return r.StringValue()
}
