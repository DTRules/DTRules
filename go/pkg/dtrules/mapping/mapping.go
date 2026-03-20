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

// Package mapping implements the DTRules XML to Entity mapping system.
// It loads mapping definitions and uses them to parse input data XML
// into DTRules entities.
package mapping

import (
	"fmt"
	"io"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// Mapping holds the configuration for mapping XML data to entities.
type Mapping struct {
	session dtrules.Session
	state   dtrules.State

	// entities maps entity name to cardinality ("1", "*", or "+")
	entities map[string]string

	// requests maps XML tag to EntityInfo for entity creation
	requests map[string]*EntityInfo

	// setattributes maps XML tag to AttributeInfo for attribute setting
	setattributes map[string]*AttributeInfo

	// multiple maps XML tag to attribute name for polymorphic entity creation
	multiple map[string]string

	// entitystack is the list of entities to create at initialization
	entitystack []string

	// attribute2listPairs maps attributes to lists (for addalltolist)
	attribute2listPairs [][2]string
}

// EntityInfo holds information about an entity to create.
type EntityInfo struct {
	Name   string // entity name
	ID     string // attribute to use as ID
	List   string // list to add entity to
	Entity string // original entity name from mapping
}

// AttributeInfo holds information about setting an attribute.
type AttributeInfo struct {
	Tag     string
	Attribs map[string]*Attrib // keyed by enclosure entity name (empty string for default)
}

// Attrib represents a single attribute mapping.
type Attrib struct {
	Tag        string
	Entity     string
	RAttribute string
	Type       AttributeType
}

// AttributeType represents the type of an attribute.
type AttributeType int

const (
	TypeNone AttributeType = iota
	TypeString
	TypeInteger
	TypeDouble
	TypeBoolean
	TypeDate
	TypeEntity
	TypeArray
	TypeXMLValue
)

// ParseAttributeType converts a string type to AttributeType.
func ParseAttributeType(s string) AttributeType {
	switch strings.ToLower(s) {
	case "string":
		return TypeString
	case "integer", "int":
		return TypeInteger
	case "double", "float", "real":
		return TypeDouble
	case "boolean", "bool":
		return TypeBoolean
	case "date":
		return TypeDate
	case "entity":
		return TypeEntity
	case "array", "list":
		return TypeArray
	case "xmlvalue":
		return TypeXMLValue
	default:
		return TypeNone
	}
}

// NewMapping creates a new Mapping for the given session.
func NewMapping(session dtrules.Session) *Mapping {
	return &Mapping{
		session:             session,
		state:               session.GetState(),
		entities:            make(map[string]string),
		requests:            make(map[string]*EntityInfo),
		setattributes:       make(map[string]*AttributeInfo),
		multiple:            make(map[string]string),
		entitystack:         make([]string, 0),
		attribute2listPairs: make([][2]string, 0),
	}
}

// LoadMapping loads a mapping definition from an XML reader.
func (m *Mapping) LoadMapping(r io.Reader) error {
	loader := newMapLoader(m)
	return loader.Load(r)
}

// LoadData loads data from an XML reader according to the mapping.
func (m *Mapping) LoadData(r io.Reader) error {
	loader := newDataLoader(m)
	return loader.Load(r)
}

// LoadDataAndPush loads data and pushes singleton entities onto the entity stack.
// Use this for projects where singleton entities get their data from the input XML
// rather than being initialized with defaults. Entities are pushed in the order
// specified by pushOrder (e.g., []string{"state_config", "job", "taxpayer"}).
func (m *Mapping) LoadDataAndPush(r io.Reader, pushOrder []string) error {
	loader := newDataLoader(m)
	if err := loader.Load(r); err != nil {
		return err
	}
	// Push singleton entities onto the entity stack in the specified order
	for _, name := range pushOrder {
		if entity, ok := loader.entities[name]; ok {
			m.state.EntityPush(entity)
		}
	}
	return nil
}

// Initialize creates the initial entities and pushes them onto the entity stack.
func (m *Mapping) Initialize() error {
	for _, entityName := range m.entitystack {
		rname := dtrules.GetRName(entityName)
		if rname == nil {
			return fmt.Errorf("invalid entity name syntax in entity stack: %s", entityName)
		}
		entity, err := m.session.CreateEntity(rname)
		if err != nil {
			return fmt.Errorf("failed to create initial entity %s: %w", entityName, err)
		}
		m.state.EntityPush(entity)
	}
	return nil
}

// AddEntityInfo adds entity creation info for a tag.
func (m *Mapping) AddEntityInfo(tag string, info *EntityInfo) {
	m.requests[tag] = info
}

// AddAttributeInfo adds attribute mapping info for a tag.
func (m *Mapping) AddAttributeInfo(tag, enclosure, rAttribute, typeStr string) {
	aInfo := m.setattributes[tag]
	if aInfo == nil {
		aInfo = &AttributeInfo{
			Tag:     tag,
			Attribs: make(map[string]*Attrib),
		}
		m.setattributes[tag] = aInfo
	}

	attrib := &Attrib{
		Tag:        tag,
		Entity:     strings.ToLower(enclosure),
		RAttribute: strings.ToLower(rAttribute),
		Type:       ParseAttributeType(typeStr),
	}
	aInfo.Attribs[strings.ToLower(enclosure)] = attrib
}

// AddInitialEntity adds an entity to be created at initialization.
func (m *Mapping) AddInitialEntity(entityName string) {
	m.entitystack = append(m.entitystack, strings.ToLower(entityName))
}

// SetEntityCardinality sets the cardinality for an entity ("1", "*", or "+").
func (m *Mapping) SetEntityCardinality(entityName, number string) {
	m.entities[strings.ToLower(entityName)] = number
}

// AddAttribute2List adds an attribute to list mapping.
func (m *Mapping) AddAttribute2List(attribute, list string) {
	m.attribute2listPairs = append(m.attribute2listPairs,
		[2]string{strings.ToLower(attribute), strings.ToLower(list)})
}

// GetEntityInfo returns the EntityInfo for a tag.
func (m *Mapping) GetEntityInfo(tag string) *EntityInfo {
	return m.requests[tag]
}

// GetAttributeInfo returns the AttributeInfo for a tag.
func (m *Mapping) GetAttributeInfo(tag string) *AttributeInfo {
	return m.setattributes[tag]
}

// GetEntityCardinality returns the cardinality for an entity.
func (m *Mapping) GetEntityCardinality(entityName string) string {
	if c, ok := m.entities[strings.ToLower(entityName)]; ok {
		return c
	}
	return "*" // default
}

// Lookup finds the Attrib for a given enclosure, falling back to default.
func (a *AttributeInfo) Lookup(enclosure string) *Attrib {
	if attrib, ok := a.Attribs[strings.ToLower(enclosure)]; ok {
		return attrib
	}
	// Fall back to default (empty enclosure)
	return a.Attribs[""]
}
