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

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
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
	case dtrules.TypeBigInt:
		if v, err := dtrules.GetRBigIntFromString(defaultStr); err == nil {
			return v
		}
	case dtrules.TypeFixed:
		if v, err := dtrules.GetRFixedFromString(defaultStr); err == nil {
			return v
		}
	}

	return dtrules.GetRNull()
}

// GetErrors returns any errors encountered during loading.
func (l *JSONEDDLoader) GetErrors() []error {
	return l.errors
}
