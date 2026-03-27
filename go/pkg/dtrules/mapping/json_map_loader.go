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

package mapping

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// JSONMapping represents the top-level JSON mapping structure.
type JSONMapping struct {
	SetAttributes   []JSONSetAttribute  `json:"setattributes"`
	CreateEntities  []JSONCreateEntity  `json:"createentities"`
	Entities        []JSONEntity        `json:"entities"`
	InitialEntities []JSONInitialEntity `json:"initialentities"`
	AddAllToList    []JSONAddAllToList  `json:"addalltolist,omitempty"`
}

// JSONSetAttribute represents a setattribute mapping in JSON.
type JSONSetAttribute struct {
	Tag        string `json:"tag"`
	RAttribute string `json:"rattribute,omitempty"`
	Enclosure  string `json:"enclosure,omitempty"`
	Entity     string `json:"entity,omitempty"`
	Type       string `json:"type,omitempty"`
}

// JSONCreateEntity represents a createentity mapping in JSON.
type JSONCreateEntity struct {
	Entity    string `json:"entity"`
	Tag       string `json:"tag"`
	ID        string `json:"id,omitempty"`
	List      string `json:"list,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Value     string `json:"value,omitempty"`
}

// JSONEntity represents an entity cardinality definition in JSON.
type JSONEntity struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

// JSONInitialEntity represents an initial entity definition in JSON.
type JSONInitialEntity struct {
	Entity string `json:"entity"`
}

// JSONAddAllToList represents an addalltolist mapping in JSON.
type JSONAddAllToList struct {
	WithAttribute string `json:"withattribute"`
	ToList        string `json:"tolist"`
}

// jsonMapLoader loads mapping definition from JSON.
type jsonMapLoader struct {
	mapping *Mapping
	errors  []string
}

// newJSONMapLoader creates a new JSON map loader.
func newJSONMapLoader(m *Mapping) *jsonMapLoader {
	return &jsonMapLoader{
		mapping: m,
		errors:  make([]string, 0),
	}
}

// Load parses the mapping definition from JSON.
func (l *jsonMapLoader) Load(r io.Reader) error {
	var jm JSONMapping
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&jm); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	// Process setattributes
	for _, sa := range jm.SetAttributes {
		if err := l.handleSetAttribute(sa); err != nil {
			l.errors = append(l.errors, err.Error())
		}
	}

	// Process createentities
	for _, ce := range jm.CreateEntities {
		if err := l.handleCreateEntity(ce); err != nil {
			l.errors = append(l.errors, err.Error())
		}
	}

	// Process entities
	for _, e := range jm.Entities {
		if err := l.handleEntity(e); err != nil {
			l.errors = append(l.errors, err.Error())
		}
	}

	// Process initial entities
	for _, ie := range jm.InitialEntities {
		if err := l.handleInitialEntity(ie); err != nil {
			l.errors = append(l.errors, err.Error())
		}
	}

	// Process addalltolist
	for _, atl := range jm.AddAllToList {
		if err := l.handleAddAllToList(atl); err != nil {
			l.errors = append(l.errors, err.Error())
		}
	}

	if len(l.errors) > 0 {
		return fmt.Errorf("mapping load errors: %s", strings.Join(l.errors, "; "))
	}
	return nil
}

// handleSetAttribute processes a JSON setattribute entry.
func (l *jsonMapLoader) handleSetAttribute(sa JSONSetAttribute) error {
	tag := sa.Tag
	typeStr := sa.Type
	rAttribute := sa.RAttribute
	if rAttribute == "" {
		rAttribute = tag // default to tag name
	}

	// enclosure can be specified as "enclosure" or "entity"
	enclosure := sa.Enclosure
	if enclosure == "" {
		enclosure = sa.Entity
	}

	if tag == "" {
		return fmt.Errorf("setattribute requires tag")
	}

	l.mapping.AddAttributeInfo(tag, enclosure, rAttribute, typeStr)
	return nil
}

// handleCreateEntity processes a JSON createentity entry.
func (l *jsonMapLoader) handleCreateEntity(ce JSONCreateEntity) error {
	entity := strings.ToLower(ce.Entity)
	tag := ce.Tag
	id := ce.ID
	list := strings.ToLower(ce.List)
	attribute := ce.Attribute
	value := ce.Value

	if entity == "" || tag == "" {
		return fmt.Errorf("createentity requires entity and tag")
	}

	info := &EntityInfo{
		Name:   entity,
		ID:     id,
		List:   list,
		Entity: entity,
	}

	// If attribute/value specified, it's a polymorphic mapping
	if attribute != "" && value != "" {
		l.mapping.multiple[tag] = attribute
		l.mapping.AddEntityInfo(value, info)
	} else {
		l.mapping.AddEntityInfo(tag, info)
	}

	return nil
}

// handleEntity processes a JSON entity cardinality entry.
func (l *jsonMapLoader) handleEntity(e JSONEntity) error {
	name := strings.ToLower(e.Name)
	number := e.Number

	if name == "" {
		return fmt.Errorf("entity requires name")
	}

	if number != "1" && number != "*" && number != "+" {
		return fmt.Errorf("entity number must be '1', '*', or '+', got: %s", number)
	}

	l.mapping.SetEntityCardinality(name, number)
	return nil
}

// handleInitialEntity processes a JSON initial entity entry.
func (l *jsonMapLoader) handleInitialEntity(ie JSONInitialEntity) error {
	entity := ie.Entity
	if entity == "" {
		return fmt.Errorf("initialentity requires entity")
	}

	l.mapping.AddInitialEntity(entity)
	return nil
}

// handleAddAllToList processes a JSON addalltolist entry.
func (l *jsonMapLoader) handleAddAllToList(atl JSONAddAllToList) error {
	withAttribute := atl.WithAttribute
	toList := atl.ToList

	if withAttribute == "" || toList == "" {
		return fmt.Errorf("addalltolist requires withattribute and tolist")
	}

	l.mapping.AddAttribute2List(withAttribute, toList)
	return nil
}
