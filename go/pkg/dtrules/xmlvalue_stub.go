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

// RXmlValue is a stub type for XML values.
// TODO: Implement full XML value support
type RXmlValue struct {
	BaseObject
	name       string
	attributes map[string]string
}

// NewRXmlValue creates a new RXmlValue stub.
func NewRXmlValue(name string) *RXmlValue {
	return &RXmlValue{
		name:       name,
		attributes: make(map[string]string),
	}
}

// Type returns the type for this object.
func (r *RXmlValue) Type() *RType {
	return TypeEntity // Placeholder
}

// GetName returns the element name.
func (r *RXmlValue) GetName() string {
	return r.name
}

// SetAttribute sets an XML attribute.
func (r *RXmlValue) SetAttribute(key, value string) {
	r.attributes[key] = value
}

// GetAttribute gets an XML attribute.
func (r *RXmlValue) GetAttribute(key string) string {
	return r.attributes[key]
}

// StringValue returns the string representation.
func (r *RXmlValue) StringValue() string {
	return r.name
}

// PostFix returns the postfix representation.
func (r *RXmlValue) PostFix() string {
	return r.name
}

// Clone returns a copy.
func (r *RXmlValue) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns a copy.
func (r *RXmlValue) RClone() Object {
	return r
}

// Execute pushes this onto the data stack.
func (r *RXmlValue) Execute(state State) error {
	return state.DataPush(r)
}

// ArrayExecute pushes this onto the data stack.
func (r *RXmlValue) ArrayExecute(state State) error {
	return state.DataPush(r)
}

// GetExecutable returns this object.
func (r *RXmlValue) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RXmlValue) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false.
func (r *RXmlValue) IsExecutable() bool {
	return false
}

// Equals compares this with another object.
func (r *RXmlValue) Equals(o Object) (bool, error) {
	other, ok := o.(*RXmlValue)
	if !ok {
		return false, nil
	}
	return r.name == other.name, nil
}

// RStringValue returns an RString.
func (r *RXmlValue) RStringValue() *RString {
	return NewRString(r.StringValue())
}
