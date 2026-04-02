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

package el

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Type constants matching Java IRObject types
const (
	TypeInteger  = "integer"
	TypeLong     = "long"
	TypeDouble   = "double"
	TypeString   = "string"
	TypeBoolean  = "boolean"
	TypeEntity   = "entity"
	TypeArray    = "array"
	TypeDate     = "date"
	TypeName     = "name"
	TypeTable    = "table"
	TypeXmlValue = "xmlvalue"
	TypeBigInt   = "bigint"
)

// EDDFile represents the root entity_data_dictionary element
type EDDFile struct {
	XMLName  xml.Name    `xml:"entity_data_dictionary"`
	Version  string      `xml:"version,attr"`
	Entities []EDDEntity `xml:"entity"`
}

// EDDFileAlt represents the alternate root entity_dictionary element
// used by some projects like CorporateTax
type EDDFileAlt struct {
	XMLName  xml.Name    `xml:"entity_dictionary"`
	Name     string      `xml:"name,attr"`
	Entities []EDDEntity `xml:"entity"`
}

// EDDEntity represents an entity definition
type EDDEntity struct {
	Name   string     `xml:"name,attr"`
	Fields []EDDField `xml:"field"`
}

// EDDField represents a field/attribute in an entity
type EDDField struct {
	Name    string `xml:"name,attr"`
	Type    string `xml:"type,attr"`
	SubType string `xml:"subtype,attr"`
}

// LoadEDDFile loads an EDD from a file path and returns a symbol table.
// The symbol table maps identifier names to their types.
func LoadEDDFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open EDD file: %w", err)
	}
	defer f.Close()
	return LoadEDD(f)
}

// LoadEDD loads an EDD from an io.Reader and returns a symbol table.
// The symbol table maps identifier names to their types.
// Keys include both "fieldname" and "entity.fieldname" forms.
// Supports both <entity_data_dictionary> and <entity_dictionary> root elements.
func LoadEDD(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read EDD: %w", err)
	}

	// Try standard format first (entity_data_dictionary)
	var edd EDDFile
	var entities []EDDEntity
	if err := xml.Unmarshal(data, &edd); err == nil && len(edd.Entities) > 0 {
		entities = edd.Entities
	} else {
		// Try alternate format (entity_dictionary)
		var eddAlt EDDFileAlt
		if err := xml.Unmarshal(data, &eddAlt); err == nil && len(eddAlt.Entities) > 0 {
			entities = eddAlt.Entities
		} else {
			// Neither format worked
			return nil, fmt.Errorf("failed to parse EDD: unsupported root element")
		}
	}

	symbols := make(map[string]string)

	for _, entity := range entities {
		entityName := strings.ToLower(entity.Name)

		for _, field := range entity.Fields {
			fieldName := strings.ToLower(field.Name)
			fieldType := normalizeType(field.Type)

			// Add as "fieldname" (may be overwritten if multiple entities have same field)
			symbols[fieldName] = fieldType

			// Add as "entity.fieldname" (unique)
			symbols[entityName+"."+fieldName] = fieldType
		}
	}

	return symbols, nil
}

// normalizeType converts EDD type strings to our canonical type constants
func normalizeType(t string) string {
	switch strings.ToLower(t) {
	case "integer", "int", "long":
		return TypeInteger
	case "double", "float":
		return TypeDouble
	case "string":
		return TypeString
	case "boolean", "bool":
		return TypeBoolean
	case "entity":
		return TypeEntity
	case "array":
		return TypeArray
	case "date":
		return TypeDate
	case "name":
		return TypeName
	case "table":
		return TypeTable
	case "xmlvalue":
		return TypeXmlValue
	case "bigint", "biginteger":
		return TypeBigInt
	default:
		return t
	}
}
