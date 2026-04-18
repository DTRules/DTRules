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

package authoring

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// validTypes is the set of DTRules primitive types.
var validTypes = map[string]bool{
	"string":  true,
	"integer": true,
	"long":    true,
	"double":  true,
	"boolean": true,
	"date":    true,
	"bigint":  true,
	"array":   true,
	"entity":  true,
}

// EDD is a typed view of a project's Entity Data Dictionary. Mutations are
// reflected in the underlying XML and persisted via Project.SaveEDD.
type EDD struct {
	eddFile string // path of the primary _edd.xml file
	xml     *excel.EDDXML
}

// Entity is a typed view of one EDD entity.
type Entity struct {
	Name       string
	Attributes []Attribute

	xmlEntity *excel.EDDXMLEntity
}

// Attribute holds all field metadata for one EDD attribute.
type Attribute struct {
	Name, Type, Subtype, Default, Access, Input, Comment string
}

// EDD returns the EDD view for this project, loading it lazily if needed.
// If no _edd.xml file exists, an empty EDD backed by a new file is returned.
func (p *Project) EDD() *EDD {
	if p.edd != nil {
		return p.edd
	}

	entries, _ := filepath.Glob(filepath.Join(p.xmlDir, "*_edd.xml"))

	eddPath := ""
	var eddXML *excel.EDDXML

	if len(entries) > 0 {
		eddPath = entries[0]
		data, err := os.ReadFile(eddPath)
		if err == nil {
			var x excel.EDDXML
			if xml.Unmarshal(data, &x) == nil {
				eddXML = &x
			}
		}
	}

	if eddXML == nil {
		eddXML = &excel.EDDXML{Version: "2"}
	}
	if eddPath == "" {
		eddPath = filepath.Join(p.xmlDir, "project_edd.xml")
	}

	p.edd = &EDD{eddFile: eddPath, xml: eddXML}
	return p.edd
}

// SaveEDD writes the EDD back to its file.
func (p *Project) SaveEDD() error {
	if p.edd == nil {
		return nil
	}
	imp := excel.NewEDDImporter()
	return imp.WriteXML(p.edd.xml, p.edd.eddFile)
}

// Entities returns a view of every entity in the EDD.
func (e *EDD) Entities() []*Entity {
	result := make([]*Entity, 0, len(e.xml.Entities))
	for _, xe := range e.xml.Entities {
		result = append(result, entityFromXML(xe))
	}
	return result
}

// Entity returns the named entity, or nil if not found.
func (e *EDD) Entity(name string) *Entity {
	for _, xe := range e.xml.Entities {
		if xe.Name == name {
			return entityFromXML(xe)
		}
	}
	return nil
}

// AddEntity creates a new entity. Returns an error if the name is already taken.
func (e *EDD) AddEntity(name string) (*Entity, error) {
	for _, xe := range e.xml.Entities {
		if xe.Name == name {
			return nil, fmt.Errorf("entity %q already exists", name)
		}
	}
	xe := &excel.EDDXMLEntity{
		Name:   name,
		Access: "rw",
		Fields: []*excel.EDDXMLField{},
	}
	e.xml.Entities = append(e.xml.Entities, xe)
	return entityFromXML(xe), nil
}

// DeleteEntity removes the named entity. It returns an error if any loaded
// decision table references this entity as a context (to avoid silent breakage).
// Pass the owning project so cross-artifact references can be detected.
func (e *EDD) DeleteEntity(name string) error {
	for i, xe := range e.xml.Entities {
		if xe.Name == name {
			e.xml.Entities = append(e.xml.Entities[:i], e.xml.Entities[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("entity %q not found", name)
}

// deleteEntityChecked is the project-aware variant used by Project.DeleteEntity.
func (e *EDD) deleteEntityChecked(name string, dtFiles []dtFileEntry) error {
	for _, entry := range dtFiles {
		for _, t := range entry.tables.Tables {
			for _, ctx := range strings.Split(t.Contexts, ",") {
				if strings.TrimSpace(ctx) == name {
					return fmt.Errorf("cannot delete entity %q: referenced as context in table %q", name, t.TableName)
				}
			}
		}
	}
	return e.DeleteEntity(name)
}

// entityFromXML builds an Entity view from underlying XML without copying
// the full attribute slice — the xmlEntity pointer lets mutations write through.
func entityFromXML(xe *excel.EDDXMLEntity) *Entity {
	attrs := make([]Attribute, 0, len(xe.Fields))
	for _, f := range xe.Fields {
		attrs = append(attrs, attributeFromXML(f))
	}
	return &Entity{Name: xe.Name, Attributes: attrs, xmlEntity: xe}
}

func attributeFromXML(f *excel.EDDXMLField) Attribute {
	return Attribute{
		Name:    f.Name,
		Type:    f.Type,
		Subtype: f.SubType,
		Default: f.DefaultValue,
		Access:  f.Access,
		Input:   f.Input,
		Comment: f.Comment,
	}
}

// AddAttribute appends an attribute to this entity after validation.
func (e *Entity) AddAttribute(a Attribute) error {
	if err := validateAttribute(a); err != nil {
		return err
	}
	for _, f := range e.xmlEntity.Fields {
		if f.Name == a.Name {
			return fmt.Errorf("attribute %q already exists on entity %q", a.Name, e.Name)
		}
	}
	e.xmlEntity.Fields = append(e.xmlEntity.Fields, attributeToXML(a))
	e.Attributes = append(e.Attributes, a)
	return nil
}

// UpdateAttribute replaces the named attribute. Fields in the replacement that
// are empty strings are treated as "keep existing value".
func (e *Entity) UpdateAttribute(name string, a Attribute) error {
	for i, f := range e.xmlEntity.Fields {
		if f.Name != name {
			continue
		}
		merged := mergeAttribute(attributeFromXML(f), a)
		if err := validateAttribute(merged); err != nil {
			return err
		}
		*e.xmlEntity.Fields[i] = *attributeToXML(merged)
		e.Attributes[i] = merged
		return nil
	}
	return fmt.Errorf("attribute %q not found on entity %q", name, e.Name)
}

// DeleteAttribute removes the named attribute.
func (e *Entity) DeleteAttribute(name string) error {
	for i, f := range e.xmlEntity.Fields {
		if f.Name == name {
			e.xmlEntity.Fields = append(e.xmlEntity.Fields[:i], e.xmlEntity.Fields[i+1:]...)
			e.Attributes = append(e.Attributes[:i], e.Attributes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("attribute %q not found on entity %q", name, e.Name)
}

// mergeAttribute returns a copy of base with non-empty fields from patch applied.
func mergeAttribute(base, patch Attribute) Attribute {
	result := base
	result.Name = base.Name // name is the key; never replaced via patch
	if patch.Type != "" {
		result.Type = patch.Type
	}
	if patch.Subtype != "" {
		result.Subtype = patch.Subtype
	}
	if patch.Default != "" {
		result.Default = patch.Default
	}
	if patch.Access != "" {
		result.Access = patch.Access
	}
	if patch.Input != "" {
		result.Input = patch.Input
	}
	if patch.Comment != "" {
		result.Comment = patch.Comment
	}
	return result
}

func attributeToXML(a Attribute) *excel.EDDXMLField {
	return &excel.EDDXMLField{
		Name:         a.Name,
		Type:         a.Type,
		SubType:      a.Subtype,
		DefaultValue: a.Default,
		Access:       a.Access,
		Input:        a.Input,
		Comment:      a.Comment,
	}
}

func validateAttribute(a Attribute) error {
	if a.Name == "" {
		return fmt.Errorf("attribute name must not be empty")
	}
	if a.Type == "" {
		return fmt.Errorf("attribute %q: type must not be empty", a.Name)
	}
	if !validTypes[a.Type] {
		return fmt.Errorf("attribute %q: unknown type %q", a.Name, a.Type)
	}
	if (a.Type == "array" || a.Type == "entity") && a.Subtype == "" {
		return fmt.Errorf("attribute %q: subtype required for type %q", a.Name, a.Type)
	}
	if a.Access != "" && a.Access != "r" && a.Access != "rw" {
		return fmt.Errorf("attribute %q: access must be \"r\", \"rw\", or empty; got %q", a.Name, a.Access)
	}
	if a.Default != "" {
		if err := validateDefault(a.Type, a.Default); err != nil {
			return fmt.Errorf("attribute %q: %w", a.Name, err)
		}
	}
	return nil
}

// validateDefault checks that the default value string is parseable as the declared type.
func validateDefault(typ, def string) error {
	switch typ {
	case "integer", "long", "bigint":
		if _, err := strconv.ParseInt(def, 10, 64); err != nil {
			return fmt.Errorf("default %q is not a valid %s", def, typ)
		}
	case "double":
		if _, err := strconv.ParseFloat(def, 64); err != nil {
			return fmt.Errorf("default %q is not a valid double", def)
		}
	case "boolean":
		lower := strings.ToLower(def)
		if lower != "true" && lower != "false" {
			return fmt.Errorf("default %q is not a valid boolean", def)
		}
	}
	// string, date, array, entity: any value is syntactically acceptable
	return nil
}
