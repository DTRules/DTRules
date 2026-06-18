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

package entity

import (
	"fmt"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// MappingKeyName is the attribute name used for mapping keys
var MappingKeyName = dtrules.GetRName("mapping*key")

// REntity represents an entity in the rules engine.
// Entities are typed dictionaries - you can only put objects
// with appropriate name and type into an entity.
type REntity struct {
	dtrules.BaseObject
	name       *dtrules.RName
	id         int
	readonly   bool
	attributes map[*dtrules.RName]*EntityEntry
	values     []dtrules.Object
	comment    string
	xlsFile    string // Export grouping for EDD spreadsheets (legacy)
	filePath   string // Canonical file path for Excel generation

	// collected tracks, per attribute index, whether a `collect` field's
	// value is authoritative (true) vs still defaulted (false) — the
	// per-instance state for interactive collection (#852). nil means
	// tracking is off (batch execution): zero overhead, behaves as today.
	collected []bool
}

// EnableCollectTracking turns on per-instance collected/defaulted tracking
// for interactive collection (#852). It is a no-op in batch execution (no
// resolver attached), so the values slice and read path stay untouched.
func (e *REntity) EnableCollectTracking() {
	if e.collected == nil {
		e.collected = make([]bool, len(e.values))
	} else if len(e.collected) < len(e.values) {
		grown := make([]bool, len(e.values))
		copy(grown, e.collected)
		e.collected = grown
	}
}

// IsCollected reports whether the named attribute has been collected
// (authoritative). False when tracking is off or the field is still
// defaulted.
func (e *REntity) IsCollected(name *dtrules.RName) bool {
	if e.collected == nil {
		return false
	}
	entry := e.attributes[name]
	if entry == nil || entry.Index >= len(e.collected) {
		return false
	}
	return e.collected[entry.Index]
}

// MarkCollected marks the named attribute as collected (authoritative).
// No-op when tracking is off.
func (e *REntity) MarkCollected(name *dtrules.RName) {
	if e.collected == nil {
		return
	}
	entry := e.attributes[name]
	if entry != nil && entry.Index < len(e.collected) {
		e.collected[entry.Index] = true
	}
}

// NewREntity creates a new reference entity (id=0 for references).
func NewREntity(id int, readonly bool, name *dtrules.RName) *REntity {
	e := &REntity{
		name:       name,
		id:         id,
		readonly:   readonly,
		attributes: make(map[*dtrules.RName]*EntityEntry),
		values:     make([]dtrules.Object, 0),
	}
	// Add self-reference attribute
	e.AddAttribute(name, "", e, false, true, dtrules.TypeEntity, "", "Self Reference", "", "")
	// Add mapping key attribute
	e.AddAttribute(MappingKeyName, "", dtrules.GetRNull(), false, true, dtrules.TypeString, "", "Mapping Key", "", "")
	return e
}

// CloneEntity creates a new instance from a reference entity.
func CloneEntity(readonly bool, source *REntity, session dtrules.Session) (*REntity, error) {
	e := &REntity{
		name:       source.name,
		id:         session.GetUniqueID(),
		readonly:   readonly,
		attributes: source.attributes, // Share attribute definitions
		values:     make([]dtrules.Object, len(source.values)),
		comment:    source.comment,
	}
	copy(e.values, source.values)

	// Patch self-reference
	e.Put(e.name, e)
	// Clear mapping key
	e.Put(MappingKeyName, dtrules.GetRNull())

	// Clone values (shallow for entities, deep for primitives)
	for i, value := range e.values {
		if value == source {
			e.values[i] = e // Self-reference
		} else if value != nil {
			// Don't deep-clone entity values - just reference copy
			// Entities are shared, not owned by attributes
			if value.Type() == dtrules.TypeEntity {
				// Keep the reference as-is (already copied above)
				continue
			}
			cloned, err := value.Clone(session)
			if err != nil {
				return nil, err
			}
			e.values[i] = cloned
		}
	}

	// A cloned instance starts fully defaulted: its collect fields must be
	// (re-)collected. If the template tracks collection, give the clone a
	// fresh all-false slice rather than copying the template's state (#852).
	if source.collected != nil {
		e.collected = make([]bool, len(e.values))
	}

	return e, nil
}

// Type returns the type for this object.
func (e *REntity) Type() *dtrules.RType {
	return dtrules.TypeEntity
}

// Execute pushes this entity onto the data stack.
func (e *REntity) Execute(state dtrules.State) error {
	if err := state.DataPush(e); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this entity onto the data stack.
func (e *REntity) ArrayExecute(state dtrules.State) error {
	if err := state.DataPush(e); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this entity.
func (e *REntity) GetExecutable() dtrules.Object {
	return e
}

// GetNonExecutable returns this entity.
func (e *REntity) GetNonExecutable() dtrules.Object {
	return e
}

// IsExecutable returns false for entities.
func (e *REntity) IsExecutable() bool {
	return false
}

// Equals compares this entity with another object.
func (e *REntity) Equals(o dtrules.Object) (bool, error) {
	if o.Type() != dtrules.TypeEntity {
		return false, nil
	}
	other, err := o.REntityValue()
	if err != nil {
		return false, err
	}
	// Entities are equal if they are the same object
	return e == other.(*REntity), nil
}

// Compare is not supported for entities.
func (e *REntity) Compare(o dtrules.Object) (int, error) {
	return 0, dtrules.UndefinedError("Compare", "Entity Objects do not support Compare")
}

// StringValue returns the entity name.
func (e *REntity) StringValue() string {
	return e.name.StringValue()
}

// PostFix returns the postfix representation.
func (e *REntity) PostFix() string {
	return fmt.Sprintf("/%s %d fce ", e.name.StringValue(), e.id)
}

// Clone creates a copy of this entity.
func (e *REntity) Clone(session dtrules.Session) (dtrules.Object, error) {
	if e.readonly {
		return e, nil
	}
	return CloneEntity(false, e, session)
}

// RClone returns this entity.
func (e *REntity) RClone() dtrules.Object {
	return e
}

// REntityValue returns this entity.
func (e *REntity) REntityValue() (dtrules.Entity, error) {
	return e, nil
}

// RStringValue returns an RString for this entity's name.
func (e *REntity) RStringValue() *dtrules.RString {
	return dtrules.NewRString(e.StringValue())
}

// GetName returns the entity's name.
func (e *REntity) GetName() *dtrules.RName {
	return e.name
}

// GetID returns the unique instance ID.
func (e *REntity) GetID() int {
	return e.id
}

// SetID sets the ID (for debugging tools only).
func (e *REntity) SetID(id int) {
	e.id = id
}

// IsReadOnly returns whether this entity is read-only.
func (e *REntity) IsReadOnly() bool {
	return e.readonly
}

// GetComment returns the entity comment.
func (e *REntity) GetComment() string {
	return e.comment
}

// SetComment sets the entity comment.
func (e *REntity) SetComment(comment string) {
	e.comment = comment
}

// GetXlsFile returns the export grouping file for EDD spreadsheets.
func (e *REntity) GetXlsFile() string {
	return e.xlsFile
}

// SetXlsFile sets the export grouping file for EDD spreadsheets.
func (e *REntity) SetXlsFile(xlsFile string) {
	e.xlsFile = xlsFile
}

// GetFilePath returns the canonical file path for Excel generation.
// Falls back to legacy xls_file if filePath is not set.
func (e *REntity) GetFilePath() string {
	if e.filePath != "" {
		return e.filePath
	}
	return e.xlsFile // Fallback to legacy xls_file
}

// SetFilePath sets the canonical file path for Excel generation.
func (e *REntity) SetFilePath(path string) {
	e.filePath = path
}

// ContainsAttribute checks if entity has an attribute.
func (e *REntity) ContainsAttribute(name *dtrules.RName) bool {
	_, ok := e.attributes[name]
	return ok
}

// GetEntry returns the entry for an attribute.
func (e *REntity) GetEntry(name *dtrules.RName) *EntityEntry {
	return e.attributes[name]
}

// GetEntries returns all entries.
func (e *REntity) GetEntries() []*EntityEntry {
	entries := make([]*EntityEntry, 0, len(e.attributes))
	for _, entry := range e.attributes {
		entries = append(entries, entry)
	}
	return entries
}

// GetAttributeNames returns all attribute names.
func (e *REntity) GetAttributeNames() []*dtrules.RName {
	names := make([]*dtrules.RName, 0, len(e.attributes))
	for name := range e.attributes {
		names = append(names, name)
	}
	return names
}

// Get retrieves an attribute value by name.
func (e *REntity) Get(name *dtrules.RName) (dtrules.Object, error) {
	entry := e.attributes[name]
	if entry == nil {
		return nil, nil
	}
	return e.values[entry.Index], nil
}

// GetByIndex retrieves an attribute value by index.
func (e *REntity) GetByIndex(i int) dtrules.Object {
	if i < 0 || i >= len(e.values) {
		return nil
	}
	return e.values[i]
}

// Put sets an attribute value.
func (e *REntity) Put(name *dtrules.RName, value dtrules.Object) error {
	entry := e.attributes[name]
	if entry == nil {
		return dtrules.UndefinedError("REntity.Put", "Undefined Attribute "+name.StringValue()+" in Entity: "+e.name.StringValue())
	}

	// Type conversion if needed
	if value.Type() != dtrules.TypeNull && entry.Type.GetID() != value.Type().GetID() {
		var err error
		switch entry.Type.GetID() {
		case dtrules.TypeInteger.GetID():
			value, err = value.RIntegerValue()
		case dtrules.TypeDouble.GetID():
			value, err = value.RDoubleValue()
		case dtrules.TypeBoolean.GetID():
			value, err = value.RBooleanValue()
		case dtrules.TypeEntity.GetID():
			_, err = value.REntityValue()
		case dtrules.TypeName.GetID():
			value, err = value.RNameValue()
		case dtrules.TypeString.GetID():
			value = value.RStringValue()
		case dtrules.TypeDate.GetID():
			value, err = value.RTimeValue()
		case dtrules.TypeBigInt.GetID():
			value, err = value.RBigIntValue()
		case dtrules.TypeFixed.GetID():
			value, err = dtrules.PromoteToRFixed(value)
		}
		if err != nil {
			return err
		}
	}

	e.values[entry.Index] = value
	// A write makes a collected field authoritative, so a later read of it
	// during interactive collection won't re-ask. No-op in batch (tracking
	// off): one nil-check.
	if e.collected != nil && entry.Index < len(e.collected) && entry.Collect {
		e.collected[entry.Index] = true
	}
	return nil
}

// SetByIndex sets an attribute value by index.
func (e *REntity) SetByIndex(i int, v dtrules.Object) {
	if i >= 0 && i < len(e.values) {
		e.values[i] = v
	}
}

// AddAttribute adds an attribute to this entity.
func (e *REntity) AddAttribute(
	attributeName *dtrules.RName,
	defaultTxt string,
	defaultValue dtrules.Object,
	writable bool,
	readable bool,
	typ *dtrules.RType,
	subtype string,
	comment string,
	input string,
	output string,
) string {
	existing := e.GetEntry(attributeName)
	if existing == nil {
		// New attribute
		index := len(e.values)
		if defaultValue == nil {
			e.values = append(e.values, dtrules.GetRNull())
		} else {
			e.values = append(e.values, defaultValue)
		}
		if e.collected != nil {
			e.collected = append(e.collected, false)
		}
		entry := NewEntityEntry(
			e,
			attributeName,
			defaultTxt,
			defaultValue,
			writable,
			readable,
			typ,
			subtype,
			index,
			comment,
			input,
			output,
		)
		e.attributes[attributeName] = entry
		return ""
	}

	// Check for type conflict
	if existing.Type != typ {
		return fmt.Sprintf("The entity '%s' has an attribute '%s' with two types: (%s) and (%s)",
			e.name.StringValue(), attributeName.StringValue(),
			existing.Type.String(), typ.String())
	}
	return ""
}

// AddEntry adds an EntityEntry to this entity.
func (e *REntity) AddEntry(entry *EntityEntry) {
	oldEntry := e.GetEntry(entry.Attribute)
	if oldEntry != nil {
		// Replace existing - keep index
		entry.Index = oldEntry.Index
	} else {
		// New attribute
		entry.Index = len(e.values)
		e.values = append(e.values, dtrules.GetRNull())
		// Keep the collected slice parallel to values when tracking is on,
		// or IsCollected/MarkCollected silently no-op for this field (#852).
		if e.collected != nil {
			e.collected = append(e.collected, false)
		}
	}

	// Set default value
	if entry.DefaultValue == nil {
		e.values[entry.Index] = dtrules.GetRNull()
	} else {
		e.values[entry.Index] = entry.DefaultValue
	}

	e.attributes[entry.Attribute] = entry
}

// RemoveAttribute removes an attribute.
func (e *REntity) RemoveAttribute(name *dtrules.RName) {
	delete(e.attributes, name)
}

// String returns a string representation.
func (e *REntity) String() string {
	return fmt.Sprintf("%s[%d]", e.name.StringValue(), e.id)
}

// Implement remaining Object interface methods that delegate to BaseObject

// IntValue is not supported for entities.
func (e *REntity) IntValue() (int, error) {
	return 0, dtrules.ConversionError("IntValue", "No Integer value exists for entity")
}

// LongValue is not supported for entities.
func (e *REntity) LongValue() (int64, error) {
	return 0, dtrules.ConversionError("LongValue", "No Long value exists for entity")
}

// DoubleValue is not supported for entities.
func (e *REntity) DoubleValue() (float64, error) {
	return 0, dtrules.ConversionError("DoubleValue", "No Double value exists for entity")
}

// BooleanValue is not supported for entities.
func (e *REntity) BooleanValue() (bool, error) {
	return false, dtrules.ConversionError("BooleanValue", "No Boolean value exists for entity")
}

// TimeValue is not supported for entities.
func (e *REntity) TimeValue() (time.Time, error) {
	return time.Time{}, dtrules.ConversionError("TimeValue", "No Time value exists for entity")
}

// ArrayValue is not supported for entities.
func (e *REntity) ArrayValue() ([]dtrules.Object, error) {
	return nil, dtrules.ConversionError("ArrayValue", "No Array value exists for entity")
}

// TableValue is not supported for entities.
func (e *REntity) TableValue() (map[dtrules.Object]dtrules.Object, error) {
	return nil, dtrules.ConversionError("TableValue", "No Table value exists for entity")
}

// RIntegerValue is not supported for entities.
func (e *REntity) RIntegerValue() (*dtrules.RInteger, error) {
	return nil, dtrules.ConversionError("RIntegerValue", "No Integer value exists for entity")
}

// RDoubleValue is not supported for entities.
func (e *REntity) RDoubleValue() (*dtrules.RDouble, error) {
	return nil, dtrules.ConversionError("RDoubleValue", "No Double value exists for entity")
}

// RBooleanValue is not supported for entities.
func (e *REntity) RBooleanValue() (*dtrules.RBoolean, error) {
	return nil, dtrules.ConversionError("RBooleanValue", "No Boolean value exists for entity")
}

// RNameValue is not supported for entities.
func (e *REntity) RNameValue() (*dtrules.RName, error) {
	return nil, dtrules.ConversionError("RNameValue", "No Name value exists for entity")
}

// RArrayValue is not supported for entities.
func (e *REntity) RArrayValue() (*dtrules.RArray, error) {
	return nil, dtrules.ConversionError("RArrayValue", "No Array value exists for entity")
}

// RTimeValue is not supported for entities.
func (e *REntity) RTimeValue() (*dtrules.RDate, error) {
	return nil, dtrules.ConversionError("RTimeValue", "No Time value exists for entity")
}
