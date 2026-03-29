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

// RMark represents a marker on the data stack.
// Used by arraytomark to collect elements into an array.
// There is only one instance of RMark (singleton).
type RMark struct {
	BaseObject
}

// Singleton mark value
var theMark = &RMark{}

// GetMark returns the singleton mark object.
func GetMark() *RMark {
	return theMark
}

// Type returns the type for this object.
func (r *RMark) Type() *RType {
	return TypeMark
}

// Execute pushes this object onto the data stack.
func (r *RMark) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RMark) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RMark) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RMark) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for mark.
func (r *RMark) IsExecutable() bool {
	return false
}

// Equals returns true only for the mark singleton.
func (r *RMark) Equals(o Object) (bool, error) {
	return o == theMark, nil
}

// Compare is not supported for marks.
func (r *RMark) Compare(o Object) (int, error) {
	return 0, UndefinedError("Compare", "Mark Objects do not support Compare")
}

// StringValue returns "Mark".
func (r *RMark) StringValue() string {
	return "Mark"
}

// PostFix returns "mark".
func (r *RMark) PostFix() string {
	return "mark"
}

// Clone returns this object (mark is a singleton).
func (r *RMark) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RMark) RClone() Object {
	return r
}

// RStringValue returns an RString for this value.
func (r *RMark) RStringValue() *RString {
	return NewRString("Mark")
}

// String implements the Stringer interface.
func (r *RMark) String() string {
	return r.StringValue()
}
