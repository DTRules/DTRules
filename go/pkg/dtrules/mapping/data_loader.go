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
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

// dataLoader loads XML data according to a mapping.
type dataLoader struct {
	mapping *Mapping
	session dtrules.Session
	state   dtrules.State

	// entities maps "entityname$id" to created entities
	entities map[string]dtrules.Entity

	// codeCnt generates unique IDs
	codeCnt int

	// tagStack tracks the current tag hierarchy
	tagStack []string

	// pendingAttribs tracks attributes to set on end tag
	pendingAttribs []pendingAttrib

	// entityCreatedStack tracks which tags created entities (for cleanup)
	entityCreatedStack []bool

	// charDataStack accumulates character data for each tag level
	charDataStack []*strings.Builder
}

// pendingAttrib represents an attribute to set on end tag.
type pendingAttrib struct {
	rAttribute string
	attrType   AttributeType
	isDate     bool
}

// newDataLoader creates a new data loader.
func newDataLoader(m *Mapping) *dataLoader {
	return &dataLoader{
		mapping:            m,
		session:            m.session,
		state:              m.state,
		entities:           make(map[string]dtrules.Entity),
		tagStack:           make([]string, 0),
		pendingAttribs:     make([]pendingAttrib, 0),
		entityCreatedStack: make([]bool, 0),
		charDataStack:      make([]*strings.Builder, 0),
	}
}

// Load parses data XML according to the mapping.
func (l *dataLoader) Load(r io.Reader) error {
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
				return err
			}
		case xml.EndElement:
			if err := l.handleEndElement(t); err != nil {
				return err
			}
		case xml.CharData:
			l.handleCharData(t)
		}
	}

	return nil
}

// handleStartElement processes an XML start element.
func (l *dataLoader) handleStartElement(elem xml.StartElement) error {
	tag := elem.Name.Local
	attrs := l.attrsToMap(elem.Attr)

	// Push onto tag stack
	l.tagStack = append(l.tagStack, tag)

	// Start new char data accumulator
	l.charDataStack = append(l.charDataStack, &strings.Builder{})

	// Track if we created an entity
	entityCreated := false

	// Check if this tag should create an entity
	info := l.mapping.GetEntityInfo(tag)
	if info != nil {
		entity, err := l.findOrCreateEntity(info, attrs)
		if err != nil {
			return err
		}
		if entity != nil {
			entityCreated = true

			// Set the mapping key
			code := attrs[info.ID]
			if code == "" {
				l.codeCnt++
				code = fmt.Sprintf("v%d", l.codeCnt)
			}
			mappingKey := dtrules.GetRName("mapping*key")
			if mappingKey != nil {
				entity.Put(mappingKey, dtrules.NewRString(code))
			}

			// Push onto entity stack
			l.state.EntityPush(entity)

			// Update references (add to lists, set references)
			l.updateReferences(entity, info)
		}
	}

	l.entityCreatedStack = append(l.entityCreatedStack, entityCreated)

	// Check if this tag sets an attribute
	aInfo := l.mapping.GetAttributeInfo(tag)
	if aInfo != nil {
		// Get the top entity from stack to find enclosure
		topEntity, err := l.state.EntityFetch(0)
		if err == nil && topEntity != nil {
			enclosure := topEntity.GetName().StringValue()
			attrib := aInfo.Lookup(enclosure)
			if attrib != nil {
				l.pendingAttribs = append(l.pendingAttribs, pendingAttrib{
					rAttribute: attrib.RAttribute,
					attrType:   attrib.Type,
					isDate:     attrib.Type == TypeDate,
				})
			} else {
				// Try default (empty enclosure)
				attrib = aInfo.Lookup("")
				if attrib != nil {
					l.pendingAttribs = append(l.pendingAttribs, pendingAttrib{
						rAttribute: attrib.RAttribute,
						attrType:   attrib.Type,
						isDate:     attrib.Type == TypeDate,
					})
				} else {
					l.pendingAttribs = append(l.pendingAttribs, pendingAttrib{})
				}
			}
		} else {
			l.pendingAttribs = append(l.pendingAttribs, pendingAttrib{})
		}
	} else {
		l.pendingAttribs = append(l.pendingAttribs, pendingAttrib{})
	}

	return nil
}

// handleEndElement processes an XML end element.
func (l *dataLoader) handleEndElement(elem xml.EndElement) error {
	if len(l.tagStack) == 0 {
		return nil
	}

	// Get character data for this element
	body := ""
	if len(l.charDataStack) > 0 {
		body = strings.TrimSpace(l.charDataStack[len(l.charDataStack)-1].String())
		l.charDataStack = l.charDataStack[:len(l.charDataStack)-1]
	}

	// Check if we need to pop an entity
	var createdEntity dtrules.Entity
	if len(l.entityCreatedStack) > 0 {
		created := l.entityCreatedStack[len(l.entityCreatedStack)-1]
		l.entityCreatedStack = l.entityCreatedStack[:len(l.entityCreatedStack)-1]

		if created {
			var err error
			createdEntity, err = l.state.EntityPop()
			if err != nil {
				return err
			}
		}
	}

	// Check if we need to set an attribute
	if len(l.pendingAttribs) > 0 {
		pending := l.pendingAttribs[len(l.pendingAttribs)-1]
		l.pendingAttribs = l.pendingAttribs[:len(l.pendingAttribs)-1]

		if pending.rAttribute != "" {
			if err := l.setAttribute(pending, body, createdEntity); err != nil {
				return err
			}
		}
	}

	// Pop tag stack
	l.tagStack = l.tagStack[:len(l.tagStack)-1]

	return nil
}

// handleCharData accumulates character data.
func (l *dataLoader) handleCharData(data xml.CharData) {
	if len(l.charDataStack) > 0 {
		l.charDataStack[len(l.charDataStack)-1].Write(data)
	}
}

// findOrCreateEntity finds or creates an entity based on the info and attributes.
func (l *dataLoader) findOrCreateEntity(info *EntityInfo, attrs map[string]string) (dtrules.Entity, error) {
	entityName := info.Name
	code := attrs[info.ID]

	cardinality := l.mapping.GetEntityCardinality(entityName)

	var entity dtrules.Entity
	var err error

	entityRName := dtrules.GetRName(entityName)
	if entityRName == nil {
		return nil, fmt.Errorf("invalid entity name syntax: %s", entityName)
	}

	if cardinality == "1" {
		// Singleton - reuse if exists
		if e, ok := l.entities[entityName]; ok {
			return e, nil
		}
		entity, err = l.session.CreateEntity(entityRName)
		if err != nil {
			return nil, err
		}
		l.entities[entityName] = entity
	} else {
		// Multiple instances
		key := ""
		if code != "" {
			key = entityName + "$" + code
			if e, ok := l.entities[key]; ok {
				return e, nil
			}
		}

		entity, err = l.session.CreateEntity(entityRName)
		if err != nil {
			return nil, err
		}

		if code != "" {
			l.entities[key] = entity
		}
	}

	return entity, nil
}

// setAttribute sets an attribute value on the current entity.
func (l *dataLoader) setAttribute(pending pendingAttrib, body string, createdEntity dtrules.Entity) error {
	attrName := dtrules.GetRName(pending.rAttribute)
	if attrName == nil {
		return nil // Invalid attribute name, skip
	}

	// Find the entity that has this attribute
	entity, err := l.state.FindEntity(attrName)
	if err != nil || entity == nil {
		return nil // Attribute not found, skip
	}

	var value dtrules.Object

	switch pending.attrType {
	case TypeInteger:
		if body == "" {
			body = "0"
		}
		val, err := strconv.ParseInt(body, 10, 64)
		if err != nil {
			val = 0
		}
		value = dtrules.GetRIntegerValue(val)

	case TypeDouble:
		if body == "" {
			body = "0"
		}
		val, err := strconv.ParseFloat(body, 64)
		if err != nil {
			val = 0
		}
		value = dtrules.GetRDoubleValue(val)

	case TypeBoolean:
		if body == "" {
			body = "false"
		}
		val := strings.ToLower(body) == "true"
		if val {
			value = dtrules.True
		} else {
			value = dtrules.False
		}

	case TypeDate:
		if body == "" {
			value = dtrules.GetRNull()
		} else {
			dateParser := l.session.GetDateParser()
			if dateParser != nil {
				parsedDate, err := dateParser.Parse(body)
				if err != nil {
					return fmt.Errorf("failed to parse date '%s': %w", body, err)
				}
				value = dtrules.GetRTime(parsedDate)
			} else {
				value = dtrules.NewRString(body)
			}
		}

	case TypeEntity:
		if createdEntity != nil {
			value = createdEntity
		} else {
			return fmt.Errorf("entity type attribute requires an entity value")
		}

	default:
		value = dtrules.NewRString(body)
	}

	// Use def to set the attribute in the current context
	_, err = l.state.Def(attrName, value, false)
	return err
}

// updateReferences adds the entity to any matching lists on the entity stack.
func (l *dataLoader) updateReferences(entity dtrules.Entity, info *EntityInfo) error {
	entityName := entity.GetName().StringValue()

	// Determine list name
	listName := info.List
	if listName == "" {
		listName = entityName + "s" // default pluralization
	}
	listRName := dtrules.GetRName(listName)
	if listRName == nil {
		return nil // Invalid list name, skip
	}

	// Look for arrays on the entity stack
	depth := l.state.EntityDepth()
	for i := 0; i < depth; i++ {
		stackEntity, err := l.state.EntityFetch(i)
		if err != nil {
			continue
		}

		// Check if this entity has a list attribute
		listObj, err := stackEntity.Get(listRName)
		if err != nil || listObj == nil {
			continue
		}

		// Check if it's an array
		if listObj.Type() == dtrules.TypeArray {
			arr, err := listObj.RArrayValue()
			if err != nil {
				continue
			}
			// Add entity to array if not already present
			if !arr.Contains(entity) {
				arr.Add(entity)
			}
		}
	}

	// Update references to this entity type on the entity stack
	if depth > 0 {
		topEntity, err := l.state.EntityFetch(depth - 1)
		if err == nil && topEntity != nil {
			entityRName := entity.GetName()
			// Check if top entity has an attribute with this name
			if topEntity.ContainsAttribute(entityRName) {
				// Don't overwrite self-reference
				isSame, _ := topEntity.GetName().Equals(entityRName)
				if !isSame {
					topEntity.Put(entityRName, entity)
				}
			}
		}
	}

	return nil
}

// attrsToMap converts XML attributes to a map.
func (l *dataLoader) attrsToMap(attrs []xml.Attr) map[string]string {
	result := make(map[string]string)
	for _, attr := range attrs {
		result[strings.ToLower(attr.Name.Local)] = attr.Value
	}
	return result
}
