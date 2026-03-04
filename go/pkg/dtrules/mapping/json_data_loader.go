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
	"strconv"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// jsonDataLoader loads JSON data according to a mapping.
type jsonDataLoader struct {
	mapping *Mapping
	session dtrules.Session
	state   dtrules.State

	// entities maps "entityname$id" to created entities
	entities map[string]dtrules.Entity

	// codeCnt generates unique IDs
	codeCnt int
}

// newJSONDataLoader creates a new JSON data loader.
func newJSONDataLoader(m *Mapping) *jsonDataLoader {
	return &jsonDataLoader{
		mapping:  m,
		session:  m.session,
		state:    m.state,
		entities: make(map[string]dtrules.Entity),
	}
}

// Load parses JSON data according to the mapping.
func (l *jsonDataLoader) Load(r io.Reader) error {
	var data map[string]interface{}
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	return l.processObject(data)
}

// processObject processes a JSON object, mapping keys to entity creation and attribute setting.
func (l *jsonDataLoader) processObject(obj map[string]interface{}) error {
	for key, value := range obj {
		if err := l.processKeyValue(key, value); err != nil {
			return err
		}
	}
	return nil
}

// processKeyValue processes a single key-value pair from JSON data.
func (l *jsonDataLoader) processKeyValue(key string, value interface{}) error {
	// Check if this key maps to entity creation
	info := l.mapping.GetEntityInfo(key)
	if info != nil {
		return l.processEntityValue(key, info, value)
	}

	// Check if this key maps to an attribute
	aInfo := l.mapping.GetAttributeInfo(key)
	if aInfo != nil {
		return l.setAttributeFromJSON(key, aInfo, value)
	}

	// If it's an object, recurse into it (handles wrapper elements)
	if obj, ok := value.(map[string]interface{}); ok {
		return l.processObject(obj)
	}

	// If it's an array, process each element
	if arr, ok := value.([]interface{}); ok {
		for _, item := range arr {
			if obj, ok := item.(map[string]interface{}); ok {
				if err := l.processObject(obj); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// processEntityValue processes a value that should create an entity.
func (l *jsonDataLoader) processEntityValue(key string, info *EntityInfo, value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		return l.createAndLoadEntity(key, info, v)
	case []interface{}:
		for _, item := range v {
			if obj, ok := item.(map[string]interface{}); ok {
				if err := l.createAndLoadEntity(key, info, obj); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// createAndLoadEntity creates an entity and loads its attributes from JSON data.
func (l *jsonDataLoader) createAndLoadEntity(tag string, info *EntityInfo, data map[string]interface{}) error {
	// Extract ID from data
	idStr := ""
	if info.ID != "" {
		if idVal, ok := data[info.ID]; ok {
			idStr = fmt.Sprintf("%v", idVal)
		}
	}

	entity, err := l.findOrCreateEntity(info, idStr)
	if err != nil {
		return err
	}
	if entity == nil {
		return nil
	}

	// Set the mapping key
	code := idStr
	if code == "" {
		l.codeCnt++
		code = fmt.Sprintf("v%d", l.codeCnt)
	}
	mappingKey := dtrules.GetRName("mapping*key")
	if mappingKey != nil {
		entity.Put(mappingKey, dtrules.NewRString(code))
	}

	// Push entity onto stack
	l.state.EntityPush(entity)

	// Update references (add to lists, set references)
	l.updateReferences(entity, info)

	// Process data fields
	for fieldKey, fieldValue := range data {
		// Check for nested entity creation
		nestedInfo := l.mapping.GetEntityInfo(fieldKey)
		if nestedInfo != nil {
			if err := l.processEntityValue(fieldKey, nestedInfo, fieldValue); err != nil {
				// Pop the entity we pushed before returning error
				l.state.EntityPop()
				return err
			}
			continue
		}

		// Check for attribute mapping
		aInfo := l.mapping.GetAttributeInfo(fieldKey)
		if aInfo != nil {
			if err := l.setAttributeFromJSON(fieldKey, aInfo, fieldValue); err != nil {
				// Pop the entity we pushed before returning error
				l.state.EntityPop()
				return err
			}
		}
	}

	// Pop entity from stack
	createdEntity, err := l.state.EntityPop()
	if err != nil {
		return err
	}

	// Check if this entity was used as an attribute value (entity type)
	l.setEntityReferenceIfNeeded(tag, createdEntity)

	return nil
}

// setEntityReferenceIfNeeded checks if the tag maps to an entity-type attribute and sets the reference.
func (l *jsonDataLoader) setEntityReferenceIfNeeded(tag string, entity dtrules.Entity) {
	aInfo := l.mapping.GetAttributeInfo(tag)
	if aInfo == nil {
		return
	}

	// Find the enclosure entity on the stack
	topEntity, err := l.state.EntityFetch(0)
	if err != nil || topEntity == nil {
		return
	}

	enclosure := topEntity.GetName().StringValue()
	attrib := aInfo.Lookup(enclosure)
	if attrib == nil {
		attrib = aInfo.Lookup("")
	}
	if attrib == nil || attrib.Type != TypeEntity {
		return
	}

	attrName := dtrules.GetRName(attrib.RAttribute)
	if attrName == nil {
		return
	}

	foundEntity, err := l.state.FindEntity(attrName)
	if err != nil || foundEntity == nil {
		return
	}

	l.state.Def(attrName, entity, false)
}

// setAttributeFromJSON sets an attribute value from a JSON value.
func (l *jsonDataLoader) setAttributeFromJSON(key string, aInfo *AttributeInfo, value interface{}) error {
	// Get the top entity from stack to find enclosure
	topEntity, err := l.state.EntityFetch(0)
	if err != nil || topEntity == nil {
		return nil
	}

	enclosure := topEntity.GetName().StringValue()
	attrib := aInfo.Lookup(enclosure)
	if attrib == nil {
		attrib = aInfo.Lookup("")
	}
	if attrib == nil {
		return nil
	}

	attrName := dtrules.GetRName(attrib.RAttribute)
	if attrName == nil {
		return nil
	}

	// Find the entity that has this attribute
	entity, err := l.state.FindEntity(attrName)
	if err != nil || entity == nil {
		return nil
	}

	var dtValue dtrules.Object

	switch attrib.Type {
	case TypeInteger:
		dtValue = l.convertToInteger(value)
	case TypeDouble:
		dtValue = l.convertToDouble(value)
	case TypeBoolean:
		dtValue = l.convertToBoolean(value)
	case TypeDate:
		dtValue = l.convertToDate(value)
	case TypeEntity:
		// Entity types are handled in createAndLoadEntity
		return nil
	default:
		dtValue = l.convertToString(value)
	}

	_, err = l.state.Def(attrName, dtValue, false)
	return err
}

// convertToInteger converts a JSON value to an RInteger.
func (l *jsonDataLoader) convertToInteger(value interface{}) dtrules.Object {
	switch v := value.(type) {
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			n = 0
		}
		return dtrules.GetRIntegerValue(n)
	case float64:
		return dtrules.GetRIntegerValue(int64(v))
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			n = 0
		}
		return dtrules.GetRIntegerValue(n)
	case nil:
		return dtrules.GetRIntegerValue(0)
	default:
		return dtrules.GetRIntegerValue(0)
	}
}

// convertToDouble converts a JSON value to an RDouble.
func (l *jsonDataLoader) convertToDouble(value interface{}) dtrules.Object {
	switch v := value.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			f = 0
		}
		return dtrules.GetRDoubleValue(f)
	case float64:
		return dtrules.GetRDoubleValue(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			f = 0
		}
		return dtrules.GetRDoubleValue(f)
	case nil:
		return dtrules.GetRDoubleValue(0)
	default:
		return dtrules.GetRDoubleValue(0)
	}
}

// convertToBoolean converts a JSON value to a boolean Object.
func (l *jsonDataLoader) convertToBoolean(value interface{}) dtrules.Object {
	switch v := value.(type) {
	case bool:
		if v {
			return dtrules.True
		}
		return dtrules.False
	case string:
		if strings.EqualFold(v, "true") {
			return dtrules.True
		}
		return dtrules.False
	case nil:
		return dtrules.False
	default:
		return dtrules.False
	}
}

// convertToDate converts a JSON value to a date Object.
func (l *jsonDataLoader) convertToDate(value interface{}) dtrules.Object {
	str := ""
	switch v := value.(type) {
	case string:
		str = v
	case nil:
		return dtrules.GetRNull()
	default:
		str = fmt.Sprintf("%v", v)
	}

	if str == "" {
		return dtrules.GetRNull()
	}

	dateParser := l.session.GetDateParser()
	if dateParser != nil {
		parsedDate, err := dateParser.Parse(str)
		if err != nil {
			return dtrules.NewRString(str)
		}
		return dtrules.GetRTime(parsedDate)
	}
	return dtrules.NewRString(str)
}

// convertToString converts a JSON value to a string Object.
func (l *jsonDataLoader) convertToString(value interface{}) dtrules.Object {
	switch v := value.(type) {
	case string:
		return dtrules.NewRString(v)
	case nil:
		return dtrules.NewRString("")
	default:
		return dtrules.NewRString(fmt.Sprintf("%v", v))
	}
}

// findOrCreateEntity finds or creates an entity based on the info and ID.
func (l *jsonDataLoader) findOrCreateEntity(info *EntityInfo, idStr string) (dtrules.Entity, error) {
	entityName := info.Name
	cardinality := l.mapping.GetEntityCardinality(entityName)

	entityRName := dtrules.GetRName(entityName)
	if entityRName == nil {
		return nil, fmt.Errorf("invalid entity name syntax: %s", entityName)
	}

	var entity dtrules.Entity
	var err error

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
		if idStr != "" {
			key = entityName + "$" + idStr
			if e, ok := l.entities[key]; ok {
				return e, nil
			}
		}

		entity, err = l.session.CreateEntity(entityRName)
		if err != nil {
			return nil, err
		}

		if idStr != "" {
			l.entities[key] = entity
		}
	}

	return entity, nil
}

// updateReferences adds the entity to any matching lists on the entity stack.
func (l *jsonDataLoader) updateReferences(entity dtrules.Entity, info *EntityInfo) {
	entityName := entity.GetName().StringValue()

	// Determine list name
	listName := info.List
	if listName == "" {
		listName = entityName + "s" // default pluralization
	}
	listRName := dtrules.GetRName(listName)
	if listRName == nil {
		return
	}

	// Look for arrays on the entity stack
	depth := l.state.EntityDepth()
	for i := 0; i < depth; i++ {
		stackEntity, err := l.state.EntityFetch(i)
		if err != nil {
			continue
		}

		listObj, err := stackEntity.Get(listRName)
		if err != nil || listObj == nil {
			continue
		}

		if listObj.Type() == dtrules.TypeArray {
			arr, err := listObj.RArrayValue()
			if err != nil {
				continue
			}
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
			if topEntity.ContainsAttribute(entityRName) {
				isSame, _ := topEntity.GetName().Equals(entityRName)
				if !isSame {
					topEntity.Put(entityRName, entity)
				}
			}
		}
	}
}
