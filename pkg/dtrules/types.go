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
	"sync"
)

// RType represents type metadata for Rules Engine objects.
// Types are registered globally and can be looked up by name or ID.
type RType struct {
	typename string
	id       int
}

var (
	typesMu  sync.RWMutex
	types    = make(map[string]*RType)
	typeList = []*RType{nil} // Index 0 is nil, IDs start at 1
	subtypes = make(map[string]*RType)
)

// NewType defines a new type with the given name.
// If the type already exists, returns the existing type.
func NewType(name string) *RType {
	name = strings.ToLower(name)

	typesMu.Lock()
	defer typesMu.Unlock()

	if existing, ok := types[name]; ok {
		return existing
	}

	newType := &RType{
		typename: name,
		id:       len(typeList),
	}
	types[name] = newType
	typeList = append(typeList, newType)
	return newType
}

// IsType returns true if the given name is a defined type.
func IsType(name string) bool {
	name = strings.ToLower(name)

	typesMu.RLock()
	defer typesMu.RUnlock()

	_, ok := types[name]
	return ok
}

// GetType returns the RType for the given name, or nil if not defined.
func GetType(name string) *RType {
	name = strings.ToLower(name)

	typesMu.RLock()
	defer typesMu.RUnlock()

	return types[name]
}

// GetTypeByID returns the RType for the given ID, or nil if not defined.
func GetTypeByID(id int) *RType {
	typesMu.RLock()
	defer typesMu.RUnlock()

	if id < 0 || id >= len(typeList) {
		return nil
	}
	return typeList[id]
}

// GetID returns the unique ID for this type.
func (t *RType) GetID() int {
	return t.id
}

// GetName returns the RName for this type.
// Note: Type names are validated at registration, so this should never return nil.
func (t *RType) GetName() *RName {
	rname := GetRName(t.typename)
	if rname == nil {
		// This shouldn't happen since type names are validated,
		// but handle defensively
		return GetRName("unknown")
	}
	return rname
}

// String returns the string value for this type.
func (t *RType) String() string {
	return t.typename
}

// AddSubType adds a subtype for this type.
// Generally, subtypes are implementations of a given type.
func (t *RType) AddSubType(subtype *RType) {
	typesMu.Lock()
	defer typesMu.Unlock()

	if _, ok := subtypes[subtype.String()]; !ok {
		subtypes[subtype.String()] = subtype
	}
}

// Pre-defined type IDs for quick type checking
var (
	TypeOperator      *RType
	TypeString        *RType
	TypeInteger       *RType
	TypeDouble        *RType
	TypeName          *RType
	TypeDate          *RType
	TypeInterval      *RType
	TypeBoolean       *RType
	TypeNull          *RType
	TypeArray         *RType
	TypeTable         *RType
	TypeEntity        *RType
	TypeDecisionTable *RType
	TypeMark          *RType
)

func init() {
	// Initialize the pre-defined types in a specific order to get stable IDs
	TypeOperator = NewType("operator")
	TypeString = NewType("string")
	TypeInteger = NewType("integer")
	TypeDouble = NewType("double")
	TypeName = NewType("name")
	TypeDate = NewType("date")
	TypeInterval = NewType("interval")
	TypeBoolean = NewType("boolean")
	TypeNull = NewType("null")
	TypeArray = NewType("array")
	TypeTable = NewType("table")
	TypeEntity = NewType("entity")
	TypeDecisionTable = NewType("decisiontable")
	TypeMark = NewType("mark")

	// Register common type aliases
	registerTypeAlias("float", TypeDouble)
	registerTypeAlias("long", TypeInteger)
	registerTypeAlias("int", TypeInteger)
	registerTypeAlias("bool", TypeBoolean)
	registerTypeAlias("datetime", TypeDate)
	registerTypeAlias("time", TypeDate)
	registerTypeAlias("list", TypeArray)
	registerTypeAlias("map", TypeTable)
	registerTypeAlias("dict", TypeTable)
	registerTypeAlias("dictionary", TypeTable)
	registerTypeAlias("hash", TypeTable)
	registerTypeAlias("object", TypeEntity)
}

// registerTypeAlias registers an alias name that maps to an existing type.
func registerTypeAlias(alias string, target *RType) {
	alias = strings.ToLower(alias)
	typesMu.Lock()
	defer typesMu.Unlock()
	types[alias] = target
}
