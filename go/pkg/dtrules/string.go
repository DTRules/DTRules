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
	"sync"
)

// String interning table for reusing common strings
var (
	stringInternMu    sync.RWMutex
	stringInternTable = make(map[string]*RString)
	// Pre-intern common strings
	emptyString *RString
)

func init() {
	// Pre-intern the empty string
	emptyString = newRStringInternal("", false)
	stringInternTable[""] = emptyString
}

// RString represents a string value in the rules engine.
// Strings come in executable and non-executable pairs.
type RString struct {
	BaseObject
	value      string
	executable bool
	pair       *RString // Partner with opposite executable state
	interned   bool     // True if this string is in the intern table
}

// MaxInternLength is the maximum string length to intern.
// Longer strings are not interned to avoid memory bloat.
const MaxInternLength = 64

// NewRString creates a new non-executable RString.
// Short strings (<=64 chars) are interned for reuse.
func NewRString(v string) *RString {
	return GetRString(v)
}

// GetRString returns an interned string if available, otherwise creates a new one.
func GetRString(v string) *RString {
	// Fast path for empty string
	if v == "" {
		return emptyString
	}

	// Only intern short strings
	if len(v) <= MaxInternLength {
		// Check cache first (read lock)
		stringInternMu.RLock()
		if s, ok := stringInternTable[v]; ok {
			stringInternMu.RUnlock()
			return s
		}
		stringInternMu.RUnlock()

		// Create and cache (write lock)
		stringInternMu.Lock()
		// Double-check after acquiring write lock
		if s, ok := stringInternTable[v]; ok {
			stringInternMu.Unlock()
			return s
		}
		s := newRStringInternal(v, false)
		s.interned = true
		s.pair.interned = true
		stringInternTable[v] = s
		stringInternMu.Unlock()
		return s
	}

	// Long strings are not interned
	return newRStringInternal(v, false)
}

// NewRStringExecutable creates a new RString with the specified executable state.
func NewRStringExecutable(v string, executable bool) *RString {
	s := GetRString(v)
	if executable {
		return s.pair // Return the executable form
	}
	return s
}

// newRString creates an RString pair (executable and non-executable).
// Use GetRString instead for interning support.
func newRString(v string, executable bool) *RString {
	return newRStringInternal(v, executable)
}

// newRStringInternal creates an RString pair without interning check.
func newRStringInternal(v string, executable bool) *RString {
	s := &RString{
		value:      v,
		executable: executable,
	}
	s.pair = &RString{
		value:      v,
		executable: !executable,
		pair:       s,
	}
	return s
}

// Type returns the type for this object.
func (r *RString) Type() *RType {
	return TypeString
}

// Execute handles the execution of this string.
// If executable, it compiles and executes the string content.
// Otherwise, it pushes itself onto the data stack.
func (r *RString) Execute(state State) error {
	if r.IsExecutable() {
		// Compile the string content using the session's compiler
		session := state.GetSession()
		compiled, err := session.Compile(r.value)
		if err != nil {
			return err
		}
		// Execute the compiled code
		return state.Evaluate(compiled)
	}
	return state.DataPush(r)
}

// ArrayExecute pushes this object onto the data stack.
func (r *RString) ArrayExecute(state State) error {
	return state.DataPush(r)
}

// GetExecutable returns the executable version of this string.
func (r *RString) GetExecutable() Object {
	if r.executable {
		return r
	}
	return r.pair
}

// GetNonExecutable returns the non-executable version of this string.
func (r *RString) GetNonExecutable() Object {
	if !r.executable {
		return r
	}
	return r.pair
}

// IsExecutable returns whether this string is executable.
func (r *RString) IsExecutable() bool {
	return r.executable
}

// Equals compares this string with another object.
func (r *RString) Equals(o Object) (bool, error) {
	return r.value == o.StringValue(), nil
}

// Compare compares this string with another object.
func (r *RString) Compare(o Object) (int, error) {
	f := strings.Compare(r.value, o.StringValue())
	if f < 0 {
		return -1, nil
	}
	if f > 0 {
		return 1, nil
	}
	return 0, nil
}

// StringValue returns the string value.
func (r *RString) StringValue() string {
	return r.value
}

// PostFix returns the postfix representation.
func (r *RString) PostFix() string {
	return r.String()
}

// Clone returns this object (strings are immutable).
func (r *RString) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RString) RClone() Object {
	return r
}

// IntValue tries to parse the string as an integer.
func (r *RString) IntValue() (int, error) {
	i, err := strconv.Atoi(r.value)
	if err != nil {
		return 0, ConversionError("RString.IntValue", "Could not convert '"+r.value+"' to integer")
	}
	return i, nil
}

// LongValue tries to parse the string as a long.
func (r *RString) LongValue() (int64, error) {
	l, err := strconv.ParseInt(r.value, 10, 64)
	if err != nil {
		return 0, ConversionError("RString.LongValue", "Could not convert '"+r.value+"' to long")
	}
	return l, nil
}

// DoubleValue tries to parse the string as a double.
func (r *RString) DoubleValue() (float64, error) {
	d, err := strconv.ParseFloat(r.value, 64)
	if err != nil {
		return 0, ConversionError("RString.DoubleValue", "Could not convert '"+r.value+"' to double")
	}
	return d, nil
}

// BooleanValue tries to parse the string as a boolean.
func (r *RString) BooleanValue() (bool, error) {
	return ParseBooleanValue(r.value)
}

// RIntegerValue returns an RInteger parsed from this string.
func (r *RString) RIntegerValue() (*RInteger, error) {
	v, err := r.LongValue()
	if err != nil {
		return nil, err
	}
	return GetRIntegerValue(v), nil
}

// RDoubleValue returns an RDouble parsed from this string.
func (r *RString) RDoubleValue() (*RDouble, error) {
	v, err := r.DoubleValue()
	if err != nil {
		return nil, err
	}
	return GetRDoubleValue(v), nil
}

// RBooleanValue returns an RBoolean parsed from this string.
func (r *RString) RBooleanValue() (*RBoolean, error) {
	v, err := r.BooleanValue()
	if err != nil {
		return nil, err
	}
	return GetRBoolean(v), nil
}

// RStringValue returns this object.
func (r *RString) RStringValue() *RString {
	return r
}

// RNameValue returns an RName for this string value.
func (r *RString) RNameValue() (*RName, error) {
	return GetRName(r.value), nil
}

// Value returns the underlying string value.
func (r *RString) Value() string {
	return r.value
}

// String implements the Stringer interface.
// Returns the string quoted.
func (r *RString) String() string {
	return "\"" + r.value + "\""
}

// HashCode returns a hash code for this string.
func (r *RString) HashCode() int {
	h := 0
	for _, c := range r.value {
		h = 31*h + int(c)
	}
	return h
}
