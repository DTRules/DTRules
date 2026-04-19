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

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// jsonDataLoader loads JSON data into entities using the mapping configuration.
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

// Load parses JSON data according to the mapping configuration.
func (l *jsonDataLoader) Load(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("JSON read error: %w", err)
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	for tag, raw := range rawData {
		if err := l.processTag(tag, raw); err != nil {
			return err
		}
	}

	return nil
}

// processTag handles a top-level JSON key, checking both entity and attribute mappings.
func (l *jsonDataLoader) processTag(tag string, raw json.RawMessage) error {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil
	}

	// Check if this tag creates an entity
	info := l.mapping.GetEntityInfo(tag)
	if info != nil {
		switch trimmed[0] {
		case '{':
			var obj map[string]interface{}
			if err := json.Unmarshal(raw, &obj); err != nil {
				return fmt.Errorf("failed to parse entity '%s': %w", tag, err)
			}
			return l.processEntityObject(tag, info, obj)

		case '[':
			var arr []json.RawMessage
			if err := json.Unmarshal(raw, &arr); err != nil {
				return fmt.Errorf("failed to parse array '%s': %w", tag, err)
			}
			for i, item := range arr {
				var obj map[string]interface{}
				if err := json.Unmarshal(item, &obj); err != nil {
					return fmt.Errorf("failed to parse %s[%d]: %w", tag, i, err)
				}
				if err := l.processEntityObject(tag, info, obj); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// Not a mapped entity — check if it's a simple attribute value
	aInfo := l.mapping.GetAttributeInfo(tag)
	if aInfo != nil {
		return l.processAttributeValue(tag, aInfo, raw)
	}

	return nil
}

// processEntityObject creates an entity from a JSON object and processes its fields.
func (l *jsonDataLoader) processEntityObject(tag string, info *EntityInfo, obj map[string]interface{}) error {
	entity, err := l.findOrCreateEntity(info, obj)
	if err != nil {
		return err
	}
	if entity == nil {
		return nil
	}

	// Set mapping key
	code := ""
	if idField, ok := obj[info.ID]; ok {
		code = fmt.Sprintf("%v", idField)
	}
	if code == "" {
		l.codeCnt++
		code = fmt.Sprintf("v%d", l.codeCnt)
	}
	mappingKey := dtrules.GetRName("mapping*key")
	if mappingKey != nil {
		entity.Put(mappingKey, dtrules.NewRString(code))
	}

	// Push entity onto the entity stack
	l.state.EntityPush(entity)

	// Process child fields
	for childTag, childValue := range obj {
		if childTag == info.ID {
			continue // skip the ID field
		}

		// Check if child tag creates a nested entity
		childInfo := l.mapping.GetEntityInfo(childTag)
		if childInfo != nil {
			if err := l.processNestedEntity(childTag, childInfo, childValue); err != nil {
				return err
			}
			continue
		}

		// Check if child tag sets an attribute
		aInfo := l.mapping.GetAttributeInfo(childTag)
		if aInfo != nil {
			raw, err := json.Marshal(childValue)
			if err != nil {
				continue
			}
			if err := l.processAttributeValue(childTag, aInfo, raw); err != nil {
				return err
			}
			continue
		}

		// Direct attribute assignment — try setting the value on the entity stack
		attrName := dtrules.GetRName(childTag)
		if attrName == nil {
			continue
		}
		foundEntity, err := l.state.FindEntity(attrName)
		if err != nil || foundEntity == nil {
			continue
		}
		value := l.goValueToDTRules(childValue)
		if value != nil {
			l.state.Def(attrName, value, false)
		}
	}

	// Pop entity from the entity stack
	l.state.EntityPop()

	return nil
}

// processNestedEntity handles a nested entity within a parent entity.
func (l *jsonDataLoader) processNestedEntity(tag string, info *EntityInfo, value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		return l.processEntityObject(tag, info, v)
	case []interface{}:
		for i, item := range v {
			if itemObj, ok := item.(map[string]interface{}); ok {
				if err := l.processEntityObject(tag, info, itemObj); err != nil {
					return fmt.Errorf("nested entity %s[%d]: %w", tag, i, err)
				}
			}
		}
	}
	return nil
}

// processAttributeValue sets an attribute from a JSON value.
func (l *jsonDataLoader) processAttributeValue(tag string, aInfo *AttributeInfo, raw json.RawMessage) error {
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

	// Unmarshal the raw JSON to a Go value
	var goValue interface{}
	if err := json.Unmarshal(raw, &goValue); err != nil {
		return nil
	}

	body := fmt.Sprintf("%v", goValue)

	attrName := dtrules.GetRName(attrib.RAttribute)
	if attrName == nil {
		return nil
	}

	value := l.convertToAttributeType(attrib.Type, body, goValue)
	if value != nil {
		l.state.Def(attrName, value, false)
	}

	return nil
}

// convertToAttributeType converts a value to the specified attribute type.
func (l *jsonDataLoader) convertToAttributeType(attrType AttributeType, body string, goValue interface{}) dtrules.Object {
	switch attrType {
	case TypeInteger:
		if f, ok := goValue.(float64); ok {
			return dtrules.GetRIntegerValue(int64(f))
		}
		if v, err := dtrules.GetRIntegerValueFromString(body); err == nil {
			return v
		}
		return dtrules.GetRIntegerValue(0)

	case TypeDouble:
		if f, ok := goValue.(float64); ok {
			return dtrules.GetRDoubleValue(f)
		}
		if v, err := dtrules.GetRDoubleValueFromString(body); err == nil {
			return v
		}
		return dtrules.GetRDoubleValue(0)

	case TypeBoolean:
		if b, ok := goValue.(bool); ok {
			return dtrules.GetRBoolean(b)
		}
		val := strings.ToLower(body) == "true"
		return dtrules.GetRBoolean(val)

	case TypeDate:
		if body == "" {
			return dtrules.GetRNull()
		}
		dateParser := l.session.GetDateParser()
		if dateParser != nil {
			parsedDate, err := dateParser.Parse(body)
			if err == nil {
				return dtrules.GetRTime(parsedDate)
			}
		}
		return dtrules.NewRString(body)

	case TypeBigInt:
		// BigInt values should be passed as strings in JSON to preserve precision
		if v, err := dtrules.GetRBigIntFromString(body); err == nil {
			return v
		}
		// Try parsing from float64 if it was a JSON number
		if f, ok := goValue.(float64); ok {
			// Check if it's a whole number
			if f == float64(int64(f)) {
				return dtrules.GetRBigIntFromInt64(int64(f))
			}
		}
		return dtrules.GetRBigIntFromInt64(0)

	case TypeFixed:
		// Fixed-point values must come in as strings to preserve the 10^-8
		// grid exactly; a JSON number would go through float64 and lose
		// precision on sub-satoshi amounts.
		if v, err := dtrules.GetRFixedFromString(body); err == nil {
			return v
		}
		zero, _ := dtrules.GetRFixedFromInt64(0)
		return zero

	default:
		return dtrules.NewRString(body)
	}
}

// findOrCreateEntity finds or creates an entity based on the info and data.
func (l *jsonDataLoader) findOrCreateEntity(info *EntityInfo, data map[string]interface{}) (dtrules.Entity, error) {
	entityName := info.Name
	code := ""
	if idField, ok := data[info.ID]; ok {
		code = fmt.Sprintf("%v", idField)
	}

	cardinality := l.mapping.GetEntityCardinality(entityName)

	entityRName := dtrules.GetRName(entityName)
	if entityRName == nil {
		return nil, fmt.Errorf("invalid entity name syntax: %s", entityName)
	}

	if cardinality == "1" {
		if e, ok := l.entities[entityName]; ok {
			return e, nil
		}
		entity, err := l.session.CreateEntity(entityRName)
		if err != nil {
			return nil, err
		}
		l.entities[entityName] = entity
		return entity, nil
	}

	// Multiple instances
	key := ""
	if code != "" {
		key = entityName + "$" + code
		if e, ok := l.entities[key]; ok {
			return e, nil
		}
	}

	entity, err := l.session.CreateEntity(entityRName)
	if err != nil {
		return nil, err
	}

	if code != "" {
		l.entities[key] = entity
	}
	return entity, nil
}

// goValueToDTRules converts a Go value to a DTRules object.
func (l *jsonDataLoader) goValueToDTRules(value interface{}) dtrules.Object {
	if value == nil {
		return dtrules.GetRNull()
	}

	switch v := value.(type) {
	case bool:
		return dtrules.GetRBoolean(v)
	case float64:
		if v == float64(int64(v)) {
			return dtrules.GetRIntegerValue(int64(v))
		}
		return dtrules.GetRDoubleValue(v)
	case string:
		return dtrules.GetRString(v)
	case []interface{}:
		arr, err := dtrules.NewArray(l.session, true, false)
		if err != nil {
			return dtrules.GetRNull()
		}
		for _, item := range v {
			dtItem := l.goValueToDTRules(item)
			if dtItem != nil {
				arr.Add(dtItem)
			}
		}
		return arr
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return dtrules.GetRString(fmt.Sprintf("%v", v))
		}
		return dtrules.GetRString(string(jsonBytes))
	default:
		return dtrules.GetRString(fmt.Sprintf("%v", v))
	}
}
