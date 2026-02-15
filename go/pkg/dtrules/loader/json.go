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

package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
)

// DefaultMaxJSONSize is the default maximum size for JSON input (10 MB).
const DefaultMaxJSONSize = 10 * 1024 * 1024

// MaxJSONSize is the configurable maximum size for JSON input.
// Set to 0 to disable size limit (not recommended for untrusted input).
var MaxJSONSize int64 = DefaultMaxJSONSize

// JSONEDDLoader loads Entity Data Dictionary definitions from JSON.
//
// Expected JSON format:
//
//	{
//	  "entities": [
//	    {
//	      "name": "person",
//	      "access": "rw",
//	      "comment": "A person entity",
//	      "fields": [
//	        {
//	          "name": "age",
//	          "type": "integer",
//	          "access": "rw",
//	          "defaultValue": "0"
//	        }
//	      ]
//	    }
//	  ]
//	}
type JSONEDDLoader struct {
	session dtrules.Session
	factory *entity.Factory
	errors  []error
}

// NewJSONEDDLoader creates a new JSON EDD loader.
func NewJSONEDDLoader(session dtrules.Session, factory *entity.Factory) *JSONEDDLoader {
	return &JSONEDDLoader{
		session: session,
		factory: factory,
	}
}

// JSON structures for EDD

// JSONEDDFile represents the top-level JSON EDD structure.
type JSONEDDFile struct {
	Entities []JSONEDDEntity `json:"entities"`
}

// JSONEDDEntity represents an entity definition in JSON.
type JSONEDDEntity struct {
	Name    string         `json:"name"`
	Access  string         `json:"access"`
	Comment string         `json:"comment"`
	Fields  []JSONEDDField `json:"fields"`
}

// JSONEDDField represents a field/attribute definition in JSON.
type JSONEDDField struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	SubType      string `json:"subtype"`
	Access       string `json:"access"`
	Input        string `json:"input"`
	DefaultValue string `json:"defaultValue"`
	Comment      string `json:"comment"`
}

// Load loads an EDD from a JSON io.Reader.
// The input size is limited by MaxJSONSize (default 10 MB) to prevent memory exhaustion.
func (l *JSONEDDLoader) Load(r io.Reader) error {
	if MaxJSONSize > 0 {
		r = io.LimitReader(r, MaxJSONSize+1)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read JSON EDD: %w", err)
	}

	if MaxJSONSize > 0 && int64(len(data)) > MaxJSONSize {
		return fmt.Errorf("JSON EDD exceeds maximum size limit of %d bytes", MaxJSONSize)
	}

	var edd JSONEDDFile
	if err := json.Unmarshal(data, &edd); err != nil {
		return fmt.Errorf("failed to parse JSON EDD: %w", err)
	}

	for _, ent := range edd.Entities {
		if err := l.processEntity(&ent); err != nil {
			l.errors = append(l.errors, err)
		}
	}

	if len(l.errors) > 0 {
		return &EDDLoadError{Errors: l.errors}
	}
	return nil
}

// processEntity processes a single JSON entity definition.
func (l *JSONEDDLoader) processEntity(ent *JSONEDDEntity) error {
	trimmedName := strings.TrimSpace(ent.Name)
	if trimmedName == "" {
		return fmt.Errorf("entity name must not be empty")
	}

	entityName := dtrules.GetRName(trimmedName)
	if entityName == nil {
		return fmt.Errorf("invalid entity name syntax: %s", ent.Name)
	}

	refEntity, err := l.factory.FindCreateRefEntity(false, entityName)
	if err != nil {
		return fmt.Errorf("failed to create entity %s: %w", ent.Name, err)
	}

	for _, field := range ent.Fields {
		if err := l.processField(refEntity, &field); err != nil {
			l.errors = append(l.errors, fmt.Errorf("entity %s: %w", ent.Name, err))
		}
	}

	return nil
}

// processField processes a single JSON field definition.
func (l *JSONEDDLoader) processField(refEntity *entity.REntity, field *JSONEDDField) error {
	access := strings.ToLower(field.Access)
	writable := strings.Contains(access, "w")
	readable := strings.Contains(access, "r")

	if !writable && !readable {
		readable = true
	}

	rtype := dtrules.GetType(field.Type)
	if rtype == nil {
		return fmt.Errorf("unknown type '%s' for field %s", field.Type, field.Name)
	}

	attributeName := dtrules.GetRName(strings.TrimSpace(field.Name))
	if attributeName == nil {
		return fmt.Errorf("invalid field name syntax: %s", field.Name)
	}

	defaultValue := l.computeDefaultValue(field.DefaultValue, rtype)

	errStr := refEntity.AddAttribute(
		attributeName,
		field.DefaultValue,
		defaultValue,
		writable,
		readable,
		rtype,
		field.SubType,
		field.Comment,
		field.Input,
		"",
	)
	if errStr != "" {
		return fmt.Errorf("failed to add field %s: %s", field.Name, errStr)
	}
	return nil
}

// computeDefaultValue computes the default value for a given type.
func (l *JSONEDDLoader) computeDefaultValue(defaultStr string, rtype *dtrules.RType) dtrules.Object {
	if rtype == dtrules.TypeArray {
		if l.session != nil {
			if arr, err := dtrules.NewArray(l.session, true, false); err == nil {
				return arr
			}
		}
		return dtrules.GetRNull()
	}

	if defaultStr == "" {
		return dtrules.GetRNull()
	}

	switch rtype {
	case dtrules.TypeInteger:
		if v, err := dtrules.GetRIntegerValueFromString(defaultStr); err == nil {
			return v
		}
	case dtrules.TypeDouble:
		if v, err := dtrules.GetRDoubleValueFromString(defaultStr); err == nil {
			return v
		}
	case dtrules.TypeBoolean:
		if v, err := dtrules.ParseBooleanValue(defaultStr); err == nil {
			return dtrules.GetRBoolean(v)
		}
	case dtrules.TypeString:
		return dtrules.NewRString(defaultStr)
	case dtrules.TypeDate:
		if l.session != nil {
			if d, err := dtrules.GetRDate(l.session, defaultStr); err == nil {
				return d
			}
		}
	}

	return dtrules.GetRNull()
}

// GetErrors returns any errors encountered during loading.
func (l *JSONEDDLoader) GetErrors() []error {
	return l.errors
}

// JSONDataLoader loads JSON data into DTRules entities using the EDD schema
// for type information. Unlike the XML mapping system which requires separate
// mapping definitions, the JSON data loader uses the entity definitions
// directly to determine types.
//
// Expected JSON format:
//
//	{
//	  "person": {
//	    "name": "John",
//	    "age": 30,
//	    "active": true
//	  },
//	  "orders": [
//	    {"order_id": 1, "total": 99.99},
//	    {"order_id": 2, "total": 49.50}
//	  ]
//	}
//
// Top-level keys are entity names (or plural forms for arrays of entities).
// Values are either objects (single entity) or arrays of objects (multiple entities).
type JSONDataLoader struct {
	session  dtrules.Session
	factory  *entity.Factory
	errors   []error
	warnings []string
}

// NewJSONDataLoader creates a new JSON data loader.
func NewJSONDataLoader(session dtrules.Session, factory *entity.Factory) *JSONDataLoader {
	return &JSONDataLoader{
		session: session,
		factory: factory,
	}
}

// Load reads JSON data and creates/populates DTRules entities.
// Created entities are pushed onto the session's entity stack.
func (l *JSONDataLoader) Load(r io.Reader) error {
	if MaxJSONSize > 0 {
		r = io.LimitReader(r, MaxJSONSize+1)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read JSON data: %w", err)
	}

	if MaxJSONSize > 0 && int64(len(data)) > MaxJSONSize {
		return fmt.Errorf("JSON data exceeds maximum size limit of %d bytes", MaxJSONSize)
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	state := l.session.GetState()
	if state == nil {
		return fmt.Errorf("session has no state; cannot load data without an active session")
	}

	for key, raw := range rawData {
		if err := l.processTopLevel(key, raw, state); err != nil {
			l.errors = append(l.errors, err)
		}
	}

	if len(l.errors) > 0 {
		return &JSONDataLoadError{Errors: l.errors, Warnings: l.warnings}
	}
	return nil
}

// processTopLevel handles a top-level key/value pair from the JSON data.
func (l *JSONDataLoader) processTopLevel(key string, raw json.RawMessage, state dtrules.State) error {
	// Determine if this is an array or a single object by finding the first non-whitespace byte
	firstByte := byte(0)
	for _, b := range []byte(raw) {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			firstByte = b
			break
		}
	}
	if firstByte == 0 {
		return nil
	}

	switch firstByte {
	case '{':
		// Single entity
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return fmt.Errorf("failed to parse entity '%s': %w", key, err)
		}
		ent, err := l.loadEntity(key, obj)
		if err != nil {
			return err
		}
		return state.EntityPush(ent)

	case '[':
		// Array of entities
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return fmt.Errorf("failed to parse array '%s': %w", key, err)
		}
		return l.processEntityArray(key, arr, state)

	default:
		l.warnings = append(l.warnings,
			fmt.Sprintf("skipping top-level non-object value: %s", key))
		return nil
	}
}

// processEntityArray handles an array of entities.
func (l *JSONDataLoader) processEntityArray(key string, items []json.RawMessage, state dtrules.State) error {
	// Determine singular entity name from plural array key
	entityName := singularize(key)

	for i, item := range items {
		var obj map[string]interface{}
		if err := json.Unmarshal(item, &obj); err != nil {
			l.warnings = append(l.warnings,
				fmt.Sprintf("skipping non-object item %d in array '%s'", i, key))
			continue
		}
		ent, err := l.loadEntity(entityName, obj)
		if err != nil {
			l.warnings = append(l.warnings,
				fmt.Sprintf("failed to load %s[%d]: %v", key, i, err))
			continue
		}
		if err := state.EntityPush(ent); err != nil {
			l.warnings = append(l.warnings,
				fmt.Sprintf("failed to push %s[%d]: %v", key, i, err))
		}
	}
	return nil
}

// loadEntity creates and populates a DTRules entity from a JSON object.
func (l *JSONDataLoader) loadEntity(entityName string, data map[string]interface{}) (dtrules.Entity, error) {
	name := dtrules.GetRName(entityName)
	if name == nil {
		return nil, fmt.Errorf("invalid entity name: %s", entityName)
	}

	ent, err := l.factory.CreateEntity(l.session, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity %s: %w", entityName, err)
	}
	if ent == nil {
		return nil, fmt.Errorf("entity type not found: %s", entityName)
	}

	rentity, ok := ent.(*entity.REntity)
	if !ok {
		return nil, fmt.Errorf("entity is not an REntity: %s", entityName)
	}

	for attrKey, attrValue := range data {
		attrName := dtrules.GetRName(attrKey)
		if attrName == nil {
			l.warnings = append(l.warnings,
				fmt.Sprintf("invalid attribute name: %s", attrKey))
			continue
		}

		// Check if this is an array of objects (nested entities)
		if arr, isArray := attrValue.([]interface{}); isArray && len(arr) > 0 {
			if _, isObj := arr[0].(map[string]interface{}); isObj {
				dtArray, err := l.loadEntityArray(attrKey, arr)
				if err != nil {
					l.warnings = append(l.warnings,
						fmt.Sprintf("failed to load array %s.%s: %v", entityName, attrKey, err))
					continue
				}
				if err := rentity.Put(attrName, dtArray); err != nil {
					l.warnings = append(l.warnings,
						fmt.Sprintf("failed to set array %s.%s: %v", entityName, attrKey, err))
				}
				continue
			}
		}

		// Check entry type for date handling
		var dtValue dtrules.Object
		entry := rentity.GetEntry(attrName)
		if entry != nil && entry.Type == dtrules.TypeDate {
			if strVal, ok := attrValue.(string); ok {
				dateVal, err := dtrules.GetRDate(l.session, strVal)
				if err != nil {
					l.warnings = append(l.warnings,
						fmt.Sprintf("failed to parse date %s.%s: %v", entityName, attrKey, err))
					continue
				}
				dtValue = dateVal
			} else {
				dtValue = goValueToDTRulesObject(attrValue)
			}
		} else {
			dtValue = goValueToDTRulesObject(attrValue)
		}

		if dtValue == nil {
			continue
		}

		if err := rentity.Put(attrName, dtValue); err != nil {
			l.warnings = append(l.warnings,
				fmt.Sprintf("failed to set %s.%s: %v", entityName, attrKey, err))
		}
	}

	return rentity, nil
}

// loadEntityArray creates a DTRules array containing entity references.
func (l *JSONDataLoader) loadEntityArray(arrayName string, items []interface{}) (dtrules.Object, error) {
	entityTypeName := singularize(arrayName)

	dtArray, err := dtrules.NewArray(l.session, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create array: %w", err)
	}

	for i, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			l.warnings = append(l.warnings,
				fmt.Sprintf("array item %d is not an object", i))
			continue
		}

		childEntity, err := l.loadEntity(entityTypeName, itemMap)
		if err != nil {
			l.warnings = append(l.warnings,
				fmt.Sprintf("failed to create %s[%d]: %v", arrayName, i, err))
			continue
		}

		dtArray.Add(childEntity)
	}

	return dtArray, nil
}

// GetErrors returns any errors encountered during loading.
func (l *JSONDataLoader) GetErrors() []error {
	return l.errors
}

// GetWarnings returns any warnings encountered during loading.
func (l *JSONDataLoader) GetWarnings() []string {
	return l.warnings
}

// JSONDataLoadError contains all errors encountered during JSON data loading.
type JSONDataLoadError struct {
	Errors   []error
	Warnings []string
}

// Error implements the error interface.
func (e *JSONDataLoadError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("JSON data loading error: %v", e.Errors[0])
	}
	return fmt.Sprintf("JSON data loading completed with %d errors; first: %v", len(e.Errors), e.Errors[0])
}

// Unwrap returns the first error for errors.Is/As compatibility.
func (e *JSONDataLoadError) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// goValueToDTRulesObject converts a Go value (from JSON unmarshaling) to a DTRules object.
func goValueToDTRulesObject(value interface{}) dtrules.Object {
	if value == nil {
		return dtrules.GetRNull()
	}

	switch v := value.(type) {
	case bool:
		return dtrules.GetRBoolean(v)
	case float64:
		// JSON numbers are float64; check if it's actually an integer
		if v == float64(int64(v)) {
			return dtrules.GetRIntegerValue(int64(v))
		}
		return dtrules.GetRDoubleValue(v)
	case string:
		return dtrules.GetRString(v)
	case []interface{}:
		arr, err := dtrules.NewArray(nil, true, false)
		if err != nil {
			return dtrules.GetRNull()
		}
		for _, item := range v {
			dtItem := goValueToDTRulesObject(item)
			if dtItem != nil {
				arr.Add(dtItem)
			}
		}
		return arr
	case map[string]interface{}:
		// Nested objects that aren't entities — serialize as JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return dtrules.GetRString(fmt.Sprintf("%v", v))
		}
		return dtrules.GetRString(string(jsonBytes))
	default:
		return dtrules.GetRString(fmt.Sprintf("%v", v))
	}
}

// singularize returns the singular form of a plural noun.
// Handles common cases: "orders" -> "order", "entities" -> "entity".
func singularize(s string) string {
	if len(s) <= 1 {
		return s
	}
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") {
		return s[:len(s)-1]
	}
	return s
}
