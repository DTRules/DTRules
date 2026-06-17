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
	"sort"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// validTypes is the set of DTRules primitive types accepted by the
// authoring SDK. Must stay in sync with what the EDD loader and runtime
// accept on the read path — projects in the wild use `fixed`
// (10⁻⁸-scaled bigint mantissa, see pkg/dtrules/fixed.go::RFixed) on
// existing EDD entities; omitting it here means a project loaded from
// an XML that has `type="fixed"` reads fine but any SDK
// AddAttribute/UpdateAttribute with `Type: "fixed"` is rejected as
// "unknown type". The authoring surface is meant to be a strict
// superset of what the loader accepts, never a stricter subset.
var validTypes = map[string]bool{
	"string":  true,
	"integer": true,
	"long":    true,
	"double":  true,
	"boolean": true,
	"date":    true,
	"bigint":  true,
	"fixed":   true, // 10^-8 fp; see RFixed in pkg/dtrules/fixed.go
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

	// Collect marks a field that must be asked of the user rather than
	// taken from its default. "true"/"false"/"" (empty == not collected,
	// and == "keep existing" on a patch). Distinct from Access — a
	// collected field is always writable. See #850.
	Collect string
	// QuestionText / QuestionType / Options describe how to ask for a
	// Collect field. QuestionType is one of: multiple_choice, ascii,
	// number, date. Options apply only to multiple_choice.
	QuestionText string
	QuestionType string
	Options      []Option
	// Reference range for a number question (lab-report style, #850):
	// QuestionRefLow/QuestionRefHigh bound the expected ("normal") range and
	// QuestionUnits labels the value (e.g. "mg/dL"). Either bound may be
	// empty (one-sided). Guidance only — out-of-range values are flagged, not
	// rejected.
	QuestionRefLow  string
	QuestionRefHigh string
	QuestionUnits   string
}

// Option is one choice for a multiple_choice question.
type Option struct {
	Value, Label string
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

// SaveEDD writes the project's EDD XML AND refreshes any covered Excel
// workbook. Same guard + auto-refresh contract as Project.Save: the
// pre-write check refuses if a covered Excel has been touched since
// the last export (set OverwriteExcel = true to bypass), and a
// successful write triggers a fresh Excel export so the two formats
// stay paired on disk.
//
// Save() bypasses this wrapper internally — it runs its own guard
// once and then calls saveEDDXMLOnly directly so we don't double-
// guard or double-refresh.
func (p *Project) SaveEDD() error {
	if err := p.preWriteExcelGuard(); err != nil {
		return err
	}
	if err := p.saveEDDXMLOnly(); err != nil {
		return err
	}
	return p.refreshExcelFromXML()
}

// saveEDDXMLOnly is the legacy EDD-XML-only writer. Callers that want
// the Excel-sync contract use the public SaveEDD wrapper above; only
// internal Save() (which guards once for the whole project) calls
// this directly.
func (p *Project) saveEDDXMLOnly() error {
	if p.edd == nil {
		return nil
	}
	// Canonical order: entities ascending by lowercase name. The EDD carries no
	// per-entity number; sorting on write gives a deterministic, mergeable file.
	sort.SliceStable(p.edd.xml.Entities, func(i, j int) bool {
		return strings.ToLower(p.edd.xml.Entities[i].Name) < strings.ToLower(p.edd.xml.Entities[j].Name)
	})
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
			// Match either a legacy <context_entity> directive or a DSL
			// statement that names the entity.
			for _, ent := range t.Contexts.Entities {
				if strings.TrimSpace(ent) == name {
					return fmt.Errorf("cannot delete entity %q: referenced as context in table %q", name, t.TableName)
				}
			}
			for _, ctx := range t.Contexts.DSLLines() {
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
	a := Attribute{
		Name:    f.Name,
		Type:    f.Type,
		Subtype: f.SubType,
		Default: f.DefaultValue,
		Access:  f.Access,
		Input:   f.Input,
		Comment: f.Comment,
		Collect: f.Collect,
	}
	if f.Question != nil {
		a.QuestionText = f.Question.Text
		a.QuestionType = f.Question.Type
		a.QuestionRefLow = f.Question.RefLow
		a.QuestionRefHigh = f.Question.RefHigh
		a.QuestionUnits = f.Question.Units
		for _, o := range f.Question.Options {
			a.Options = append(a.Options, Option{Value: o.Value, Label: o.Label})
		}
	}
	return a
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
	if patch.Collect != "" {
		result.Collect = patch.Collect
	}
	if patch.QuestionText != "" {
		result.QuestionText = patch.QuestionText
	}
	if patch.QuestionType != "" {
		result.QuestionType = patch.QuestionType
	}
	if patch.Options != nil {
		result.Options = patch.Options
	}
	if patch.QuestionRefLow != "" {
		result.QuestionRefLow = patch.QuestionRefLow
	}
	if patch.QuestionRefHigh != "" {
		result.QuestionRefHigh = patch.QuestionRefHigh
	}
	if patch.QuestionUnits != "" {
		result.QuestionUnits = patch.QuestionUnits
	}
	// A field that isn't collected carries no question metadata.
	if !strings.EqualFold(result.Collect, "true") {
		result.QuestionText, result.QuestionType, result.Options = "", "", nil
		result.QuestionRefLow, result.QuestionRefHigh, result.QuestionUnits = "", "", ""
	}
	return result
}

func attributeToXML(a Attribute) *excel.EDDXMLField {
	f := &excel.EDDXMLField{
		Name:         a.Name,
		Type:         a.Type,
		SubType:      a.Subtype,
		DefaultValue: a.Default,
		Access:       a.Access,
		Input:        a.Input,
		Comment:      a.Comment,
	}
	if strings.EqualFold(a.Collect, "true") {
		f.Collect = "true"
		if a.QuestionText != "" || a.QuestionType != "" || len(a.Options) > 0 ||
			a.QuestionRefLow != "" || a.QuestionRefHigh != "" || a.QuestionUnits != "" {
			q := &excel.EDDXMLQuestion{
				Text:    a.QuestionText,
				Type:    a.QuestionType,
				RefLow:  a.QuestionRefLow,
				RefHigh: a.QuestionRefHigh,
				Units:   a.QuestionUnits,
			}
			for _, o := range a.Options {
				q.Options = append(q.Options, &excel.EDDXMLOption{Value: o.Value, Label: o.Label})
			}
			f.Question = q
		}
	}
	return f
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
	if a.Access != "" && a.Access != "r" && a.Access != "w" && a.Access != "rw" {
		return fmt.Errorf("attribute %q: access must be \"r\" (input), \"w\" (output), \"rw\", or empty; got %q", a.Name, a.Access)
	}
	if a.Default != "" {
		if err := validateDefault(a.Type, a.Default); err != nil {
			return fmt.Errorf("attribute %q: %w", a.Name, err)
		}
	}
	if err := validateCollect(a); err != nil {
		return err
	}
	return nil
}

// validQuestionTypes is the set of question types the collection UI can
// render (see #850). Deliberately minimal for now.
var validQuestionTypes = map[string]bool{
	"multiple_choice": true,
	"ascii":           true,
	"number":          true,
	"date":            true,
}

// validateCollect checks the collect flag and question metadata. collect is
// metadata distinct from access; a collected field must be writable, and
// question metadata is only meaningful when collect is true.
func validateCollect(a Attribute) error {
	collect := strings.ToLower(a.Collect)
	if collect != "" && collect != "true" && collect != "false" {
		return fmt.Errorf("attribute %q: collect must be \"true\", \"false\", or empty; got %q", a.Name, a.Collect)
	}
	hasRange := a.QuestionRefLow != "" || a.QuestionRefHigh != "" || a.QuestionUnits != ""
	hasQuestion := a.QuestionText != "" || a.QuestionType != "" || len(a.Options) > 0 || hasRange
	if collect != "true" {
		if hasQuestion {
			return fmt.Errorf("attribute %q: question metadata requires collect=\"true\"", a.Name)
		}
		return nil
	}
	// collect == "true"
	if a.Access == "r" {
		return fmt.Errorf("attribute %q: a collected field must be writable (access \"w\" or \"rw\"), not \"r\"", a.Name)
	}
	if a.QuestionText == "" {
		return fmt.Errorf("attribute %q: a collected field requires question text", a.Name)
	}
	if !validQuestionTypes[a.QuestionType] {
		return fmt.Errorf("attribute %q: question type must be multiple_choice|ascii|number|date; got %q", a.Name, a.QuestionType)
	}
	if a.QuestionType == "multiple_choice" {
		if len(a.Options) == 0 {
			return fmt.Errorf("attribute %q: multiple_choice question requires options", a.Name)
		}
		for i, o := range a.Options {
			if o.Value == "" {
				return fmt.Errorf("attribute %q: option %d has an empty value", a.Name, i+1)
			}
		}
	} else if len(a.Options) > 0 {
		return fmt.Errorf("attribute %q: options are only valid for a multiple_choice question", a.Name)
	}
	if hasRange && a.QuestionType != "number" {
		return fmt.Errorf("attribute %q: a reference range or units is only valid for a number question", a.Name)
	}
	if err := validateRefBound(a.Name, "ref_low", a.QuestionRefLow); err != nil {
		return err
	}
	if err := validateRefBound(a.Name, "ref_high", a.QuestionRefHigh); err != nil {
		return err
	}
	if a.QuestionRefLow != "" && a.QuestionRefHigh != "" {
		lo, _ := strconv.ParseFloat(a.QuestionRefLow, 64)
		hi, _ := strconv.ParseFloat(a.QuestionRefHigh, 64)
		if lo > hi {
			return fmt.Errorf("attribute %q: ref_low (%s) exceeds ref_high (%s)", a.Name, a.QuestionRefLow, a.QuestionRefHigh)
		}
	}
	return nil
}

// validateRefBound checks that a reference-range bound, if present, is numeric.
func validateRefBound(name, which, v string) error {
	if v == "" {
		return nil
	}
	if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
		return fmt.Errorf("attribute %q: %s %q is not a number", name, which, v)
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
	case "fixed":
		// Accept the same surface forms as GetRFixedFromString: bare
		// integers, decimals, optional sign, optional "fp" suffix.
		// "0" and "0.00000000" are both valid; empty is allowed too
		// (treated as 0 by the loader).
		if def == "" {
			return nil
		}
		if _, err := dtrules.GetRFixedFromString(def); err != nil {
			return fmt.Errorf("default %q is not a valid fixed: %v", def, err)
		}
	}
	// string, date, array, entity: any value is syntactically acceptable
	return nil
}
