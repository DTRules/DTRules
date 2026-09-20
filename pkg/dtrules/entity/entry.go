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
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// EntityEntry holds attribute type and metadata for an entity attribute.
// Each attribute has an index into the entity's values slice.
type EntityEntry struct {
	Entity    *REntity       // The owning entity
	Attribute *dtrules.RName // The name of this attribute
	// AuthoredName is the field name exactly as written in the EDD.
	//
	// EL names are case-insensitive, so `Status` and `status` are ONE name —
	// not two that clash. Case carries no meaning to the engine at all; it is
	// for people, to make a field readable and to follow a project's naming
	// conventions. Resolution ignores it and always will.
	//
	// It still has to survive a write. The intern cache keeps the first
	// spelling it sees process-wide, so without this the exporter wrote back
	// whichever spelling happened to be interned first and silently restyled
	// the author's field. Nothing computed differently — but the XML no longer
	// matched what building from Excel produced, and byte-equality is what
	// `dtrules verify` compares (#1040).
	//
	// Empty for entries created by paths that record no spelling; callers fall
	// back to the interned name.
	AuthoredName string
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

	// SourceXlsFile is the workbook whose EDD declared this field.
	//
	// An entity is one thing at run time but may be declared in many files:
	// TaxReturn has 50 state EDDs that each add fields to the shared `result`,
	// and each names its own workbook. Those merge into one REntity, which can
	// hold only one xls_file — so 49 of the 50 workbooks were claimed by no
	// entity, and an Excel refresh wrote them with their decision tables and no
	// EDD sheet at all, deleting the dictionary from the system of record
	// (#1109).
	//
	// Recording the declaring workbook per field is what lets an export give
	// each workbook back exactly the fields its own EDD declared, so the round
	// trip through Excel reproduces the file it came from.
	//
	// Empty for fields created by paths that record no source — synthesized
	// entries, and any EDD whose entity declares no xls_file.
	SourceXlsFile string

	// Collect marks a field whose value is asked of the user rather than
	// taken from its default (#850). This is static metadata shared across
	// all instances of the entity (correct: the question is the same for
	// every instance); per-instance "has it been collected yet" state lives
	// elsewhere (#852).
	Collect bool
	// Question describes how to ask for a Collect field (nil when none).
	Question *QuestionMeta

	// Constraints bound the values this field may legally hold (#1209).
	// Nil when the field declares none, which is the common case — a field
	// with no constraint is never checked.
	Constraints *FieldConstraints
}

// FieldConstraints declares what values a field may take (#1209). Every
// member is optional; a zero-valued FieldConstraints constrains nothing.
//
// This is declaration only. Enforcement on the paths that write a field from
// outside the rules is separate work; what lives here is the metadata the EDD
// carries, round-trips through Excel, and exposes to the authoring API.
type FieldConstraints struct {
	// AllowedValues is the closed vocabulary for the field. Empty means the
	// field is not restricted to a vocabulary.
	//
	// Matching follows EL's rule for names: case-insensitive. "Acute" and
	// "acute" are one value, not two. The authored spelling is what is
	// written back out, so the list is stored exactly as declared.
	AllowedValues []string
	// MaxLength is the longest string the field may hold, in characters
	// (runes, not bytes). 0 means unlimited.
	MaxLength int
	// MaxWords is the most whitespace-separated words the field may hold.
	// 0 means unlimited.
	MaxWords int
}

// IsEmpty reports whether these constraints restrict nothing.
func (c *FieldConstraints) IsEmpty() bool {
	return c == nil || (len(c.AllowedValues) == 0 && c.MaxLength == 0 && c.MaxWords == 0)
}

// MatchAllowed looks value up in the vocabulary and returns the authored
// spelling of the entry it matched. Matching is case-insensitive, the same
// rule EL uses for names; the spelling returned is the one the EDD declares,
// never the caller's.
//
// Reports false when the value is outside the vocabulary. A field with no
// vocabulary admits everything, so the value is returned unchanged.
func (c *FieldConstraints) MatchAllowed(value string) (string, bool) {
	if c == nil || len(c.AllowedValues) == 0 {
		return value, true
	}
	for _, v := range c.AllowedValues {
		if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(value)) {
			return v, true
		}
	}
	return value, false
}

// WordCount counts the whitespace-separated words in s, the unit MaxWords
// bounds. Any run of whitespace separates, so tabs and newlines count the
// same as spaces.
func WordCount(s string) int { return len(strings.Fields(s)) }

// Check reports the first way value violates these constraints, or nil when
// it satisfies them all. The error names the field, the offending value and
// the limit it broke, so a caller can report it without composing its own
// message; fieldName is the qualified name to use ("patient.diagnosis").
func (c *FieldConstraints) Check(fieldName, value string) error {
	if c.IsEmpty() {
		return nil
	}
	if c.MaxLength > 0 {
		if n := len([]rune(value)); n > c.MaxLength {
			return fmt.Errorf("%s: value is %d characters, max_length is %d", fieldName, n, c.MaxLength)
		}
	}
	if c.MaxWords > 0 {
		if n := WordCount(value); n > c.MaxWords {
			return fmt.Errorf("%s: value is %d words, max_words is %d", fieldName, n, c.MaxWords)
		}
	}
	if _, ok := c.MatchAllowed(value); !ok {
		return fmt.Errorf("%s: %q is not one of the allowed values [%s]",
			fieldName, value, strings.Join(c.AllowedValues, ", "))
	}
	return nil
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
