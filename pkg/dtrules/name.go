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
	"fmt"
	"log"
	"strings"
	"sync"
)

// RName represents an interned name in the rules engine.
// Names come in executable and non-executable pairs.
// Names can optionally have an entity prefix (entity.attribute syntax).
type RName struct {
	BaseObject
	entity     *RName // Entity component (nil if none)
	name       string // The attribute/name part
	executable bool
	hashcode   int
	partner    *RName // Partner with opposite executable state
	postfix    string // Cached postfix representation
}

var (
	namesMu sync.RWMutex
	names   = make(map[string]*RName)
)

// ErrInvalidNameSyntax is returned when a name has invalid syntax.
var ErrInvalidNameSyntax = fmt.Errorf("invalid name syntax")

// TryGetRName returns an RName for the given string, or an error if invalid.
// Parses the string for leading slash (literal) and dot syntax (entity.attribute).
// Names are cached and interned.
func TryGetRName(name string) (*RName, error) {
	namesMu.RLock()
	if rn, ok := names[name]; ok {
		namesMu.RUnlock()
		return rn, nil
	}
	namesMu.RUnlock()

	cache := name

	// Fix the name: trim and replace internal spaces with '_'
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "_")

	// Check for leading slash (non-executable/literal)
	executable := name == "" || name[0] != '/'
	if !executable {
		name = name[1:] // Remove the slash
	}

	// Check for dot syntax (entity.attribute) - handle BEFORE acquiring lock
	// to avoid recursive deadlock
	var entityRName *RName
	dot := strings.Index(name, ".")
	if dot >= 0 {
		if dot == 0 || dot+1 == len(name) || strings.Index(name[dot+1:], ".") >= 0 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidNameSyntax, name)
		}
		entityPart := name[:dot]
		// Recursively get entity name BEFORE acquiring the write lock
		var err error
		entityRName, err = TryGetRName(entityPart)
		if err != nil {
			return nil, err
		}
		name = name[dot+1:] // attrPart
	}

	// Now acquire write lock for creating the name
	namesMu.Lock()
	defer namesMu.Unlock()

	// Double-check after getting write lock
	if rn, ok := names[cache]; ok {
		return rn, nil
	}

	rname := getRNameWithEntity(entityRName, name, executable)
	names[cache] = rname
	return rname, nil
}

// GetRName returns an RName for the given string.
// Parses the string for leading slash (literal) and dot syntax (entity.attribute).
// Names are cached and interned.
//
// WARNING: If the name has invalid syntax (e.g., ".foo", "foo.", "a.b.c"),
// this function logs a warning and returns nil. Callers should use TryGetRName
// for explicit error handling, or check the return value for nil.
func GetRName(name string) *RName {
	rn, err := TryGetRName(name)
	if err != nil {
		log.Printf("WARNING: GetRName called with invalid name syntax %q: %v", name, err)
		return nil
	}
	return rn
}

// GetRNameExecutable returns an RName with the specified executable state.
func GetRNameExecutable(name string, executable bool) *RName {
	// Fix the name: trim and replace internal spaces with '_'
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "_")
	return getRNameWithEntity(nil, name, executable)
}

// getRNameWithEntity creates or retrieves an RName with an optional entity prefix.
func getRNameWithEntity(entity *RName, name string, executable bool) *RName {
	// Fix the name: trim and replace internal spaces with '_'
	name = strings.ReplaceAll(name, " ", "_")

	// Build cache key
	lname := strings.ToLower(name)
	cname := lname
	if entity != nil {
		cname = strings.ToLower(entity.StringValue()) + "." + lname
	}

	// Check cache
	if rn, ok := names[cname]; ok {
		if executable {
			return rn.GetExecutable().(*RName)
		}
		return rn.GetNonExecutable().(*RName)
	}

	// Create new RName pair
	hashcode := hashString(lname)
	rn := newRName(entity, name, executable, hashcode)
	names[cname] = rn.GetExecutable().(*RName)

	if executable {
		return rn.GetExecutable().(*RName)
	}
	return rn.GetNonExecutable().(*RName)
}

// newRName creates an RName pair.
func newRName(entity *RName, name string, executable bool, hashcode int) *RName {
	// Build postfix with entity prefix if present
	postfix := name
	if entity != nil {
		postfix = entity.name + "." + name
	}
	if !executable {
		postfix = "/" + postfix
	}

	rn := &RName{
		entity:     entity,
		name:       name,
		executable: executable,
		hashcode:   hashcode,
		postfix:    postfix,
	}

	// Create partner with opposite executable state
	partnerPostfix := name
	if entity != nil {
		partnerPostfix = entity.name + "." + name
	}
	if executable {
		partnerPostfix = "/" + partnerPostfix
	}
	rn.partner = &RName{
		entity:     entity,
		name:       name,
		executable: !executable,
		hashcode:   hashcode,
		partner:    rn,
		postfix:    partnerPostfix,
	}

	return rn
}

// hashString computes a hash for a string.
func hashString(s string) int {
	h := 0
	for _, c := range strings.ToLower(s) {
		h = 31*h + int(c)
	}
	return h
}

// Type returns the type for this object.
func (r *RName) Type() *RType {
	return TypeName
}

// Execute looks up and executes this name.
// For executable names, it looks up the value and executes/pushes it.
// For non-executable names (literal /name), it pushes the name itself to the stack.
func (r *RName) Execute(state State) error {
	return r.ArrayExecute(state)
}

// ArrayExecute looks up this name on the entity stack and executes or pushes the result.
func (r *RName) ArrayExecute(state State) error {
	if r.executable {
		o, err := state.Find(r)
		if err != nil {
			return err
		}
		if o == nil {
			return UndefinedError("RName", "The Name '"+r.name+"' was not defined by any Entity on the Entity Stack")
		}
		if o.IsExecutable() {
			return o.Execute(state)
		}
		if err := state.DataPush(o); err != nil {
			return err
		}
		return nil
	}
	if err := state.DataPush(r); err != nil {
		return err
	}
	return nil
}

// GetExecutable returns the executable version of this name.
func (r *RName) GetExecutable() Object {
	if r.executable {
		return r
	}
	return r.partner
}

// GetNonExecutable returns the non-executable version of this name.
func (r *RName) GetNonExecutable() Object {
	if !r.executable {
		return r
	}
	return r.partner
}

// IsExecutable returns whether this name is executable.
func (r *RName) IsExecutable() bool {
	return r.executable
}

// Equals compares this name with another object.
func (r *RName) Equals(o Object) (bool, error) {
	if o.Type() != TypeName {
		return false, nil
	}
	rn, ok := o.(*RName)
	if !ok {
		return false, nil
	}
	return strings.EqualFold(r.name, rn.name), nil
}

// Compare compares this name with another object.
func (r *RName) Compare(o Object) (int, error) {
	return strings.Compare(strings.ToLower(r.name), strings.ToLower(o.StringValue())), nil
}

// StringValue returns the name value (including entity prefix if present).
func (r *RName) StringValue() string {
	if r.entity != nil {
		return r.entity.StringValue() + "." + r.name
	}
	return r.name
}

// PostFix returns the postfix representation.
func (r *RName) PostFix() string {
	return r.postfix
}

// Clone returns this object (names are interned).
func (r *RName) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RName) RClone() Object {
	return r
}

// RNameValue returns this object.
func (r *RName) RNameValue() (*RName, error) {
	return r, nil
}

// RStringValue returns an RString for this name.
func (r *RName) RStringValue() *RString {
	return NewRString(r.name)
}

// GetEntityName returns the entity component of the name (nil if none).
func (r *RName) GetEntityName() *RName {
	return r.entity
}

// GetName returns just the attribute component of the name.
func (r *RName) GetName() string {
	return r.name
}

// HashCode returns the hash code for this name.
func (r *RName) HashCode() int {
	return r.hashcode
}

// String implements the Stringer interface.
func (r *RName) String() string {
	return r.PostFix()
}
