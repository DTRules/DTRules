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
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// mapLoader loads mapping definition XML.
type mapLoader struct {
	mapping *Mapping
	errors  []string
}

// newMapLoader creates a new map loader.
func newMapLoader(m *Mapping) *mapLoader {
	return &mapLoader{
		mapping: m,
		errors:  make([]string, 0),
	}
}

// Load parses the mapping definition XML.
func (l *mapLoader) Load(r io.Reader) error {
	decoder := xml.NewDecoder(r)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("XML parse error: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			if err := l.handleStartElement(t); err != nil {
				l.errors = append(l.errors, err.Error())
			}
		}
	}

	if len(l.errors) > 0 {
		return fmt.Errorf("mapping load errors: %v", l.errors)
	}
	return nil
}

// handleStartElement processes an XML start element.
func (l *mapLoader) handleStartElement(elem xml.StartElement) error {
	attrs := l.attrsToMap(elem.Attr)

	switch strings.ToLower(elem.Name.Local) {
	case "createentity":
		return l.handleCreateEntity(attrs)
	case "setattribute":
		return l.handleSetAttribute(attrs)
	case "entity":
		return l.handleEntity(attrs)
	case "initialentity":
		return l.handleInitialEntity(attrs)
	case "addalltolist":
		return l.handleAddAllToList(attrs)
	case "do2entitymap":
		// Skip Java-specific object mapping
		return nil
	case "mapping", "xmltoedd", "map", "entities", "initialization", "dataobjects":
		// Container elements, no action needed
		return nil
	}
	return nil
}

// handleCreateEntity processes a <createentity> element.
func (l *mapLoader) handleCreateEntity(attrs map[string]string) error {
	entity := strings.ToLower(attrs["entity"])
	tag := attrs["tag"]
	id := attrs["id"]
	list := strings.ToLower(attrs["list"])
	attribute := attrs["attribute"]
	value := attrs["value"]

	if entity == "" || tag == "" {
		return fmt.Errorf("createentity requires entity and tag attributes")
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

// handleSetAttribute processes a <setattribute> element.
func (l *mapLoader) handleSetAttribute(attrs map[string]string) error {
	tag := attrs["tag"]
	typeStr := attrs["type"]
	rAttribute := attrs["rattribute"]
	if rAttribute == "" {
		rAttribute = tag // default to tag name
	}

	// enclosure can be specified as "enclosure" or "entity"
	enclosure := attrs["enclosure"]
	if enclosure == "" {
		enclosure = attrs["entity"]
	}

	if tag == "" {
		return fmt.Errorf("setattribute requires tag attribute")
	}

	l.mapping.AddAttributeInfo(tag, enclosure, rAttribute, typeStr)
	return nil
}

// handleEntity processes an <entity> element.
func (l *mapLoader) handleEntity(attrs map[string]string) error {
	name := strings.ToLower(attrs["name"])
	number := attrs["number"]

	if name == "" {
		return fmt.Errorf("entity requires name attribute")
	}

	// Validate number
	if number != "1" && number != "*" && number != "+" {
		return fmt.Errorf("entity number must be '1', '*', or '+', got: %s", number)
	}

	l.mapping.SetEntityCardinality(name, number)
	return nil
}

// handleInitialEntity processes an <initialentity> element.
// epush='false' opts the entity out of being pushed; absent or 'true' includes it.
func (l *mapLoader) handleInitialEntity(attrs map[string]string) error {
	entity := attrs["entity"]
	if entity == "" {
		return fmt.Errorf("initialentity requires entity attribute")
	}

	if epushVal, ok := attrs["epush"]; ok && strings.ToLower(epushVal) == "false" {
		return nil
	}
	l.mapping.AddInitialEntity(entity)
	return nil
}

// handleAddAllToList processes an <addalltolist> element.
func (l *mapLoader) handleAddAllToList(attrs map[string]string) error {
	withAttribute := attrs["withattribute"]
	toList := attrs["tolist"]

	if withAttribute == "" || toList == "" {
		return fmt.Errorf("addalltolist requires withAttribute and toList attributes")
	}

	l.mapping.AddAttribute2List(withAttribute, toList)
	return nil
}

// attrsToMap converts XML attributes to a map.
func (l *mapLoader) attrsToMap(attrs []xml.Attr) map[string]string {
	result := make(map[string]string)
	for _, attr := range attrs {
		result[strings.ToLower(attr.Name.Local)] = attr.Value
	}
	return result
}
