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

// RTable represents a key-value map in the rules engine.
type RTable struct {
	BaseObject
	id          int
	tablename   *RName
	description *RString
	table       map[tableKey]Object
}

// tableKey wraps an Object for use as a map key.
// We use StringValue() as the key since Go maps require comparable keys.
type tableKey struct {
	key    Object
	strKey string
}

func makeTableKey(obj Object) tableKey {
	return tableKey{
		key:    obj,
		strKey: obj.StringValue(),
	}
}

// NewRTable creates a new RTable.
func NewRTable(ef EntityFactory, tablename *RName, description string) (*RTable, error) {
	return &RTable{
		id:          ef.GetUniqueID(),
		tablename:   tablename,
		description: NewRString(description),
		table:       make(map[tableKey]Object),
	}, nil
}

// Type returns the type for this object.
func (r *RTable) Type() *RType {
	return TypeTable
}

// Execute pushes this object onto the data stack.
func (r *RTable) Execute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// ArrayExecute pushes this object onto the data stack.
func (r *RTable) ArrayExecute(state State) error {
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns this object.
func (r *RTable) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RTable) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for tables.
func (r *RTable) IsExecutable() bool {
	return false
}

// Equals compares this table with another object.
func (r *RTable) Equals(o Object) (bool, error) {
	if o.Type() != TypeTable {
		return false, nil
	}
	// Tables are equal if they are the same object
	return r == o, nil
}

// Compare is not supported for tables.
func (r *RTable) Compare(o Object) (int, error) {
	return 0, UndefinedError("Compare", "Table Objects do not support Compare")
}

// StringValue returns the table name.
func (r *RTable) StringValue() string {
	if r.tablename != nil {
		return r.tablename.StringValue()
	}
	return ""
}

// PostFix returns the postfix representation.
func (r *RTable) PostFix() string {
	return r.StringValue()
}

// Clone creates a shallow copy of this table.
func (r *RTable) Clone(session Session) (Object, error) {
	newTable, err := NewRTable(session.GetEntityFactory(), r.tablename, r.description.StringValue())
	if err != nil {
		return nil, err
	}
	for k, v := range r.table {
		newTable.table[k] = v
	}
	return newTable, nil
}

// RClone returns this object.
func (r *RTable) RClone() Object {
	return r
}

// RTableValue returns this object.
func (r *RTable) RTableValue() (*RTable, error) {
	return r, nil
}

// TableValue returns the underlying map.
func (r *RTable) TableValue() (map[Object]Object, error) {
	result := make(map[Object]Object)
	for k, v := range r.table {
		result[k.key] = v
	}
	return result, nil
}

// RStringValue returns an RString for this table.
func (r *RTable) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// GetID returns the unique ID for this table.
func (r *RTable) GetID() int {
	return r.id
}

// GetTablename returns the table name.
func (r *RTable) GetTablename() *RName {
	return r.tablename
}

// GetDescription returns the table description.
func (r *RTable) GetDescription() *RString {
	return r.description
}

// SetDescription sets the table description.
func (r *RTable) SetDescription(description *RString) {
	r.description = description
}

// SetValue sets a key-value pair.
func (r *RTable) SetValue(key Object, value Object) error {
	r.table[makeTableKey(key)] = value
	return nil
}

// GetValue gets a value by key.
// Returns RNull if the key is not found.
func (r *RTable) GetValue(key Object) (Object, error) {
	v, ok := r.table[makeTableKey(key)]
	if !ok {
		return GetRNull(), nil
	}
	return v, nil
}

// SetValueMultiKey sets a value using multiple keys (for multi-dimensional tables).
func (r *RTable) SetValueMultiKey(state State, keys []Object, value Object) error {
	var v Object = r
	for i := 0; i < len(keys)-1; i++ {
		if v.Type() != TypeTable {
			return OutOfBoundsError("RTable", "Invalid Number of Keys used with Table "+r.StringValue())
		}
		table, _ := v.RTableValue()
		next, _ := table.GetValue(keys[i])
		if next.Type() == TypeNull {
			// Create intermediate table
			newTable, err := NewRTable(state.GetSession().GetEntityFactory(), r.tablename, r.description.StringValue())
			if err != nil {
				return err
			}
			table.SetValue(keys[i], newTable)
			next = newTable
		}
		v = next
	}
	table, _ := v.RTableValue()
	return table.SetValue(keys[len(keys)-1], value)
}

// GetValueMultiKey gets a value using multiple keys.
func (r *RTable) GetValueMultiKey(keys []Object) (Object, error) {
	var v Object = r
	for i := 0; i < len(keys); i++ {
		if v.Type() != TypeTable {
			return nil, OutOfBoundsError("RTable", "Invalid Number of Keys used with Table "+r.StringValue())
		}
		table, _ := v.RTableValue()
		next, _ := table.GetValue(keys[i])
		if next.Type() == TypeNull {
			return GetRNull(), nil
		}
		v = next
	}
	return v, nil
}

// ContainsKey returns true if the key exists in the table.
func (r *RTable) ContainsKey(key Object) bool {
	_, ok := r.table[makeTableKey(key)]
	return ok
}

// ContainsValue returns true if the value exists in the table.
func (r *RTable) ContainsValue(value Object) bool {
	for _, v := range r.table {
		eq, err := v.Equals(value)
		if err == nil && eq {
			return true
		}
	}
	return false
}

// GetKeys returns an array of all keys.
func (r *RTable) GetKeys(session Session) (*RArray, error) {
	keys := make([]Object, 0, len(r.table))
	for k := range r.table {
		keys = append(keys, k.key)
	}
	return NewArrayWithElements(session, true, keys, false)
}

// Size returns the number of entries in the table.
func (r *RTable) Size() int {
	return len(r.table)
}

// String implements the Stringer interface.
func (r *RTable) String() string {
	return r.StringValue()
}
