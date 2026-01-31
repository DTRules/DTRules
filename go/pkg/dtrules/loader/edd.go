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

// Package loader implements XML loaders for EDD and Decision Table files.
package loader

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
)

// EDDLoader loads Entity Data Dictionary XML files.
type EDDLoader struct {
	session dtrules.Session
	factory *entity.Factory
	errors  []error
}

// NewEDDLoader creates a new EDD loader.
func NewEDDLoader(session dtrules.Session, factory *entity.Factory) *EDDLoader {
	return &EDDLoader{
		session: session,
		factory: factory,
		errors:  make([]error, 0),
	}
}

// XML structures matching the actual DTRules EDD format

// EDDFile represents the root entity_data_dictionary element
type EDDFile struct {
	XMLName  xml.Name    `xml:"entity_data_dictionary"`
	Version  string      `xml:"version,attr"`
	Entities []EDDEntity `xml:"entity"`
}

// EDDEntity represents an entity definition
type EDDEntity struct {
	Name    string     `xml:"name,attr"`
	Access  string     `xml:"access,attr"`
	Comment string     `xml:"comment,attr"`
	Fields  []EDDField `xml:"field"`
}

// EDDField represents a field/attribute in an entity
type EDDField struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	SubType      string `xml:"subtype,attr"`
	Access       string `xml:"access,attr"`
	Input        string `xml:"input,attr"`
	DefaultValue string `xml:"default_value,attr"`
	Comment      string `xml:"comment,attr"`
}

// Load loads an EDD from an io.Reader.
func (l *EDDLoader) Load(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read EDD: %w", err)
	}

	var edd EDDFile
	if err := xml.Unmarshal(data, &edd); err != nil {
		return fmt.Errorf("failed to parse EDD XML: %w", err)
	}

	// Process each entity
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

// EDDLoadError contains all errors encountered during EDD loading.
type EDDLoadError struct {
	Errors []error
}

// Error implements the error interface.
func (e *EDDLoadError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("EDD loading error: %v", e.Errors[0])
	}
	return fmt.Sprintf("EDD loading completed with %d errors; first: %v", len(e.Errors), e.Errors[0])
}

// Unwrap returns the first error for errors.Is/As compatibility.
func (e *EDDLoadError) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// processEntity processes a single entity definition.
func (l *EDDLoader) processEntity(ent *EDDEntity) error {
	entityName := dtrules.GetRName(strings.TrimSpace(ent.Name))
	if entityName == nil {
		return fmt.Errorf("invalid entity name syntax: %s", ent.Name)
	}

	// Create or find the reference entity
	refEntity, err := l.factory.FindCreateRefEntity(false, entityName)
	if err != nil {
		return fmt.Errorf("failed to create entity %s: %w", ent.Name, err)
	}

	// Process each field
	for _, field := range ent.Fields {
		if err := l.processField(refEntity, &field); err != nil {
			l.errors = append(l.errors, fmt.Errorf("entity %s: %w", ent.Name, err))
		}
	}

	return nil
}

// processField processes a single field definition.
func (l *EDDLoader) processField(refEntity *entity.REntity, field *EDDField) error {
	// Parse access flags
	access := strings.ToLower(field.Access)
	writable := strings.Contains(access, "w")
	readable := strings.Contains(access, "r")

	if !writable && !readable {
		// Default to readable if nothing specified
		readable = true
	}

	// Get the type
	rtype := dtrules.GetType(field.Type)
	if rtype == nil {
		return fmt.Errorf("unknown type '%s' for field %s", field.Type, field.Name)
	}

	attributeName := dtrules.GetRName(strings.TrimSpace(field.Name))
	if attributeName == nil {
		return fmt.Errorf("invalid field name syntax: %s", field.Name)
	}

	// Compute default value
	defaultValue := l.computeDefaultValue(field.DefaultValue, rtype)

	// Add the attribute
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
		"", // output - not in this format
	)
	if errStr != "" {
		return fmt.Errorf("failed to add field %s: %s", field.Name, errStr)
	}
	return nil
}

// computeDefaultValue computes the default value for a given type.
func (l *EDDLoader) computeDefaultValue(defaultStr string, rtype *dtrules.RType) dtrules.Object {
	// For array types, always create an empty array even if no default value
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
	case dtrules.TypeArray:
		// Arrays can have default values like '{ "AA" "BB" }'
		// For now, return null and let the compiler handle it later
		if l.session != nil {
			if arr, err := dtrules.NewArray(l.session, true, false); err == nil {
				return arr
			}
		}
	}

	return dtrules.GetRNull()
}

// GetErrors returns any errors encountered during loading.
func (l *EDDLoader) GetErrors() []error {
	return l.errors
}
