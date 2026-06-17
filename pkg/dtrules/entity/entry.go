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

// Package entity implements the entity system for DTRules.
package entity

import (
	"fmt"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// EntityEntry holds attribute type and metadata for an entity attribute.
// Each attribute has an index into the entity's values slice.
type EntityEntry struct {
	Entity       *REntity       // The owning entity
	Attribute    *dtrules.RName // The name of this attribute
	DefaultTxt   string         // Text representation of default value
	DefaultValue dtrules.Object // Default value (may be nil)
	Writable     bool           // Whether attribute is writable by decision tables
	Readable     bool           // Whether attribute is readable by decision tables
	Type         *dtrules.RType // The type (integer, double, etc.)
	SubType      string         // Subtype for some attributes
	Index        int            // Index into values slice
	Comment      string         // Comment describing this attribute
	Input        string         // Mapping sources that populate this attribute
	Output       string         // Entries to auto-update in source objects

	// Collect marks a field whose value is asked of the user rather than
	// taken from its default (#850). This is static metadata shared across
	// all instances of the entity (correct: the question is the same for
	// every instance); per-instance "has it been collected yet" state lives
	// elsewhere (#852).
	Collect bool
	// Question describes how to ask for a Collect field (nil when none).
	Question *QuestionMeta
}

// QuestionMeta is the metadata used to ask the user for a Collect field (#850).
type QuestionMeta struct {
	Text    string           // prompt shown to the user
	Type    string           // multiple_choice | ascii | number | date
	Options []QuestionOption // choices, for multiple_choice

	// Reference range for a number question, in the style of a lab report
	// (#850): RefLow/RefHigh bound the expected ("normal") range and Units
	// labels the value (e.g. "mg/dL"). Either bound may be empty (one-sided).
	// Guidance only — values outside the range are flagged High/Low, never
	// rejected.
	RefLow  string
	RefHigh string
	Units   string
}

// QuestionOption is one choice for a multiple_choice question.
type QuestionOption struct {
	Value string
	Label string
}

// NewEntityEntry creates a new EntityEntry.
func NewEntityEntry(
	entity *REntity,
	attribute *dtrules.RName,
	defaultTxt string,
	defaultValue dtrules.Object,
	writable bool,
	readable bool,
	typ *dtrules.RType,
	subtype string,
	index int,
	comment string,
	input string,
	output string,
) *EntityEntry {
	return &EntityEntry{
		Entity:       entity,
		Attribute:    attribute,
		DefaultTxt:   defaultTxt,
		DefaultValue: defaultValue,
		Writable:     writable,
		Readable:     readable,
		Type:         typ,
		SubType:      subtype,
		Index:        index,
		Comment:      comment,
		Input:        input,
		Output:       output,
	}
}

// GetAttribute returns the attribute name.
func (e *EntityEntry) GetAttribute() *dtrules.RName {
	return e.Attribute
}

// GetDefaultTxt returns the default value text.
func (e *EntityEntry) GetDefaultTxt() string {
	return e.DefaultTxt
}

// GetDefaultValue returns the default value.
func (e *EntityEntry) GetDefaultValue() dtrules.Object {
	return e.DefaultValue
}

// GetType returns the attribute type.
func (e *EntityEntry) GetType() *dtrules.RType {
	return e.Type
}

// IsWritable returns whether the attribute is writable.
func (e *EntityEntry) IsWritable() bool {
	return e.Writable
}

// IsReadable returns whether the attribute is readable.
func (e *EntityEntry) IsReadable() bool {
	return e.Readable
}

// GetSubType returns the subtype.
func (e *EntityEntry) GetSubType() string {
	return e.SubType
}

// GetIndex returns the index into values slice.
func (e *EntityEntry) GetIndex() int {
	return e.Index
}

// GetComment returns the comment.
func (e *EntityEntry) GetComment() string {
	return e.Comment
}

// GetInput returns the input mapping.
func (e *EntityEntry) GetInput() string {
	return e.Input
}

// GetOutput returns the output mapping.
func (e *EntityEntry) GetOutput() string {
	return e.Output
}

// String returns a string representation of this entry.
func (e *EntityEntry) String() string {
	return fmt.Sprintf("(%s) default: %s", e.Type.String(), e.DefaultTxt)
}
