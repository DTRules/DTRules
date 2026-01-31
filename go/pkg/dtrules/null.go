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

// RNull represents a null value in the rules engine.
// There is only one instance of RNull (singleton).
type RNull struct {
	BaseObject
}

// Singleton null value
var theNullObject = &RNull{}

// GetRNull returns the singleton null object.
func GetRNull() *RNull {
	return theNullObject
}

// Type returns the type for this object.
func (r *RNull) Type() *RType {
	return TypeNull
}

// Execute pushes this object onto the data stack.
func (r *RNull) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RNull) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RNull) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RNull) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for null.
func (r *RNull) IsExecutable() bool {
	return false
}

// Equals returns true only for other null objects.
func (r *RNull) Equals(o Object) (bool, error) {
	return o.Type() == TypeNull, nil
}

// Compare returns 0 for other nulls, -1 for anything else.
// Null is considered less than anything else (for sorting).
func (r *RNull) Compare(o Object) (int, error) {
	eq, _ := r.Equals(o)
	if eq {
		return 0, nil
	}
	return -1, nil
}

// StringValue returns an empty string.
func (r *RNull) StringValue() string {
	return ""
}

// PostFix returns "null".
func (r *RNull) PostFix() string {
	return "null"
}

// Clone returns this object (null is a singleton).
func (r *RNull) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RNull) RClone() Object {
	return r
}

// RStringValue returns an empty RString.
func (r *RNull) RStringValue() *RString {
	return NewRString("")
}

// String implements the Stringer interface.
func (r *RNull) String() string {
	return ""
}

// HashCode returns 0 for null.
func (r *RNull) HashCode() int {
	return 0
}
