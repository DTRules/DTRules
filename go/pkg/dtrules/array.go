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

// RArray represents an array in the rules engine.
// Arrays come in executable (code blocks) and non-executable (data) forms.
// They share the same underlying data but have different execution behavior.
type RArray struct {
	BaseObject
	array      []Object // Dynamic array form
	code       []Object // Cached/compiled form for execution
	pair       *RArray  // Partner with opposite executable state
	executable bool     // Whether this is the executable version
	dups       bool     // Whether duplicates are allowed
	id         int      // Unique ID for this array
}

// NewArray creates a new RArray with the specified properties.
func NewArray(session Session, duplicates bool, executable bool) (*RArray, error) {
	id := session.GetUniqueID()
	ar := newArrayWithID(id, duplicates, executable)

	state := session.GetState()
	if state != nil && state.TestState(TraceFlag) {
		state.TraceInfo("newarray", "arrayId", intToStr(ar.id), "")
	}
	return ar, nil
}

// NewArrayWithElements creates a new RArray with initial elements.
func NewArrayWithElements(session Session, duplicates bool, elements []Object, executable bool) (*RArray, error) {
	id := session.GetUniqueID()
	ar := newArrayWithIDAndElements(id, duplicates, elements, executable)

	state := session.GetState()
	if state != nil && state.TestState(TraceFlag) {
		state.TraceInfo("newarray", "arrayId", intToStr(ar.id), "")
		for _, v := range elements {
			state.TraceInfo("addto", "arrayId", intToStr(ar.id), v.PostFix())
		}
	}
	return ar, nil
}

// NewArrayTraceInterface is for low-level debugging access only.
func NewArrayTraceInterface(id int, duplicates bool, executable bool) *RArray {
	return newArrayWithID(id, duplicates, executable)
}

// newArrayWithID creates an RArray with a specific ID.
func newArrayWithID(id int, duplicates bool, executable bool) *RArray {
	array := make([]Object, 0)
	ar := &RArray{
		id:         id,
		array:      array,
		executable: executable,
		dups:       duplicates,
	}
	ar.pair = &RArray{
		id:         id,
		array:      array, // Share the same backing array
		executable: !executable,
		dups:       duplicates,
		pair:       ar,
	}
	return ar
}

// newArrayWithIDAndElements creates an RArray with initial elements.
func newArrayWithIDAndElements(id int, duplicates bool, elements []Object, executable bool) *RArray {
	array := make([]Object, len(elements))
	copy(array, elements)
	ar := &RArray{
		id:         id,
		array:      array,
		executable: executable,
		dups:       duplicates,
	}
	ar.pair = &RArray{
		id:         id,
		array:      array, // Share the same backing array
		executable: !executable,
		dups:       duplicates,
		pair:       ar,
	}
	return ar
}

// TraceFlag is the flag for trace state - will be properly defined in interpreter
const TraceFlag = 1

// intToStr is a helper to convert int to string
func intToStr(i int) string {
	return strings.TrimPrefix(strings.TrimPrefix(string(rune('0'+i/100000000)), "0"), "0")
}

// uncache ensures the array is in mutable form.
func (r *RArray) uncache() {
	if r.array == nil && r.code != nil {
		r.array = make([]Object, len(r.code))
		copy(r.array, r.code)
		r.pair.array = r.array
		r.code = nil
		r.pair.code = nil
	}
}

// Type returns the type for this object.
func (r *RArray) Type() *RType {
	return TypeArray
}

// Execute runs each element in the array.
func (r *RArray) Execute(state State) error {
	// Cache to code form if needed
	if r.code == nil {
		r.code = make([]Object, len(r.array))
		copy(r.code, r.array)
		r.pair.code = r.code
		r.array = nil
		r.pair.array = nil
	}

	// Empty array is a no-op
	if len(r.code) == 0 {
		return nil
	}

	for i, obj := range r.code {
		err := obj.ArrayExecute(state)
		if err != nil {
			// Add postfix context to error
			if re, ok := err.(*RulesError); ok {
				re.SetPostFix(r.generatePostfix(i))
			}
			return err
		}
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RArray) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns the executable version of this array.
func (r *RArray) GetExecutable() Object {
	if r.executable {
		return r
	}
	return r.pair
}

// GetNonExecutable returns the non-executable version of this array.
func (r *RArray) GetNonExecutable() Object {
	if !r.executable {
		return r
	}
	return r.pair
}

// IsExecutable returns whether this array is executable.
func (r *RArray) IsExecutable() bool {
	return r.executable
}

// Equals compares this array with another object.
func (r *RArray) Equals(o Object) (bool, error) {
	r.uncache()
	if o.Type() != TypeArray {
		return false, nil
	}
	other, err := o.RArrayValue()
	if err != nil {
		return false, err
	}
	other.uncache()

	if len(r.array) != len(other.array) {
		return false, nil
	}
	for i, v := range r.array {
		eq, err := v.Equals(other.array[i])
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

// Compare is not supported for arrays.
func (r *RArray) Compare(o Object) (int, error) {
	return 0, UndefinedError("Compare", "Array Objects do not support Compare")
}

// StringValue returns the string representation of this array.
func (r *RArray) StringValue() string {
	var sb strings.Builder
	if r.executable {
		sb.WriteString("{ ")
	} else {
		sb.WriteString("[ ")
	}

	if r.array != nil {
		for _, obj := range r.array {
			sb.WriteString(obj.StringValue())
			sb.WriteString(" ")
		}
	} else if r.code != nil {
		for _, obj := range r.code {
			sb.WriteString(obj.StringValue())
			sb.WriteString(" ")
		}
	}

	if r.executable {
		sb.WriteString("}")
	} else {
		sb.WriteString("]")
	}
	return sb.String()
}

// PostFix returns the postfix representation.
func (r *RArray) PostFix() string {
	var sb strings.Builder
	if r.executable {
		sb.WriteString("{")
	} else {
		sb.WriteString("[")
	}

	if r.array != nil {
		for _, obj := range r.array {
			sb.WriteString(obj.PostFix())
			sb.WriteString(" ")
		}
	} else if r.code != nil {
		for _, obj := range r.code {
			sb.WriteString(obj.PostFix())
			sb.WriteString(" ")
		}
	}

	if r.executable {
		sb.WriteString("}")
	} else {
		sb.WriteString("]")
	}
	return sb.String()
}

// generatePostfix generates postfix with error marker.
func (r *RArray) generatePostfix(errorIndex int) string {
	r.uncache()
	var sb strings.Builder
	for i, obj := range r.array {
		if i == errorIndex {
			sb.WriteString(" ERROR==> ")
		}
		sb.WriteString(obj.PostFix())
		sb.WriteString(" ")
		if i == errorIndex {
			sb.WriteString(" <== ")
		}
	}
	return sb.String()
}

// Clone creates a shallow copy of this array.
func (r *RArray) Clone(session Session) (Object, error) {
	newElements := make([]Object, r.Size())
	if r.array != nil {
		copy(newElements, r.array)
	} else if r.code != nil {
		copy(newElements, r.code)
	}
	return NewArrayWithElements(session, r.dups, newElements, r.executable)
}

// DeepCopy creates a deep copy of this array.
func (r *RArray) DeepCopy(session Session) (Object, error) {
	newElements := make([]Object, r.Size())
	if r.array != nil {
		for i, elem := range r.array {
			cloned, err := elem.Clone(session)
			if err != nil {
				return nil, err
			}
			newElements[i] = cloned
		}
	} else if r.code != nil {
		for i, elem := range r.code {
			cloned, err := elem.Clone(session)
			if err != nil {
				return nil, err
			}
			newElements[i] = cloned
		}
	}
	return NewArrayWithElements(session, r.dups, newElements, r.executable)
}

// RClone returns this object.
func (r *RArray) RClone() Object {
	return r
}

// ArrayValue returns the slice of elements.
func (r *RArray) ArrayValue() ([]Object, error) {
	r.uncache()
	return r.array, nil
}

// RArrayValue returns this object.
func (r *RArray) RArrayValue() (*RArray, error) {
	return r, nil
}

// RStringValue returns an RString for this array.
func (r *RArray) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// Add adds an element to this array.
func (r *RArray) Add(v Object) bool {
	r.uncache()
	if !r.dups && r.Contains(v) {
		return false
	}
	r.array = append(r.array, v)
	r.pair.array = r.array
	return true
}

// AddAt adds an element at a specific index.
// Returns an error if index is negative.
func (r *RArray) AddAt(index int, v Object) error {
	if index < 0 {
		return OutOfBoundsError("RArray.AddAt", "negative index not allowed")
	}
	r.uncache()
	if index >= len(r.array) {
		r.array = append(r.array, v)
	} else {
		r.array = append(r.array[:index+1], r.array[index:]...)
		r.array[index] = v
	}
	r.pair.array = r.array
	return nil
}

// Get returns the element at the given index.
func (r *RArray) Get(index int) (Object, error) {
	r.uncache()
	if index < 0 || index >= len(r.array) {
		return nil, OutOfBoundsError("RArray", "Index out of bounds")
	}
	return r.array[index], nil
}

// Delete removes the element at the given index.
func (r *RArray) Delete(index int) {
	r.uncache()
	if index >= 0 && index < len(r.array) {
		r.array = append(r.array[:index], r.array[index+1:]...)
		r.pair.array = r.array
	}
}

// Remove removes the first occurrence of the given element.
func (r *RArray) Remove(v Object) {
	r.uncache()
	for i, elem := range r.array {
		eq, err := elem.Equals(v)
		if err == nil && eq {
			r.Delete(i)
			return
		}
	}
}

// Contains checks if the array contains the given element.
func (r *RArray) Contains(v Object) bool {
	r.uncache()
	for _, elem := range r.array {
		eq, err := elem.Equals(v)
		if err == nil && eq {
			return true
		}
	}
	return false
}

// Size returns the number of elements in the array.
func (r *RArray) Size() int {
	if r.array != nil {
		return len(r.array)
	}
	if r.code != nil {
		return len(r.code)
	}
	return 0
}

// IsEmpty returns true if the array is empty.
func (r *RArray) IsEmpty() bool {
	return r.Size() == 0
}

// Clear removes all elements from the array.
func (r *RArray) Clear() {
	r.array = make([]Object, 0)
	r.code = nil
	r.pair.array = r.array
	r.pair.code = nil
}

// GetID returns the unique ID for this array.
func (r *RArray) GetID() int {
	return r.id
}

// GetIterator returns a slice for iteration.
func (r *RArray) GetIterator() []Object {
	r.uncache()
	return r.array
}

// String implements the Stringer interface.
func (r *RArray) String() string {
	return r.StringValue()
}
