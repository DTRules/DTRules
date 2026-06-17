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

// Package loader implements XML and JSON loaders for EDD (Entity Data Dictionary)
// and Decision Table files. It supports loading entity definitions and decision
// tables from both the traditional DTRules XML format and from JSON.
package loader

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// DefaultMaxXMLSize is the default maximum size for XML input (10 MB).
const DefaultMaxXMLSize = 10 * 1024 * 1024

// MaxXMLSize is the configurable maximum size for XML input.
// Set to 0 to disable size limit (not recommended for untrusted input).
var MaxXMLSize int64 = DefaultMaxXMLSize

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

// FileMetadata represents the file_metadata element in EDD files
type FileMetadata struct {
	FilePath string `xml:"file_path"`
}

// EDDFile represents the root entity_data_dictionary element
type EDDFile struct {
	XMLName      xml.Name     `xml:"entity_data_dictionary" json:"-"`
	Version      string       `xml:"version,attr" json:"version"`
	FileMetadata FileMetadata `xml:"file_metadata" json:"file_metadata,omitempty"`
	Entities     []EDDEntity  `xml:"entity" json:"entities"`
}

// EDDFileLegacy represents the legacy entity_dictionary format (used by CorporateTax)
type EDDFileLegacy struct {
	XMLName  xml.Name         `xml:"entity_dictionary"`
	Name     string           `xml:"name,attr"`
	Entities []EDDEntityShort `xml:"entity"`
}

// EDDEntityShort represents an entity in the legacy format with simpler field structure
type EDDEntityShort struct {
	Name   string          `xml:"name,attr"`
	Fields []EDDFieldShort `xml:"field"`
}

// EDDFieldShort represents a field in the legacy format
type EDDFieldShort struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	DefaultValue string `xml:"default_value,attr"`
	Comment      string `xml:"comment,attr"`
}

// EDDSource records the Excel workbook and sheet from which an entity was imported.
type EDDSource struct {
	RelativePath string `xml:"relative_path"`
	FileName     string `xml:"file_name"`
	SheetNumber  int    `xml:"sheet_number"`
}

// EDDEntity represents an entity definition
type EDDEntity struct {
	Name    string     `xml:"name,attr" json:"name"`
	Access  string     `xml:"access,attr" json:"access"`
	Comment string     `xml:"comment,attr" json:"comment,omitempty"`
	XlsFile string     `xml:"xls_file,attr" json:"xls_file,omitempty"`
	Source  *EDDSource `xml:"source,omitempty" json:"-"`
	Fields  []EDDField `xml:"field" json:"fields"`
}

// EDDField represents a field/attribute in an entity
type EDDField struct {
	Name         string `xml:"name,attr" json:"name"`
	Type         string `xml:"type,attr" json:"type"`
	SubType      string `xml:"subtype,attr" json:"subtype,omitempty"`
	Access       string `xml:"access,attr" json:"access,omitempty"`
	Input        string `xml:"input,attr" json:"input,omitempty"`
	DefaultValue string `xml:"default_value,attr" json:"default_value,omitempty"`
	Comment      string `xml:"comment,attr" json:"comment,omitempty"`
	// Timezone is the IANA name or ISO offset declared for date fields
	// (Phase 2 of #743). When set, tz-naïve default-value strings are
	// parsed in this zone instead of UTC.
	Timezone string `xml:"timezone,attr,omitempty" json:"timezone,omitempty"`
	// Collect marks a field whose value is asked of the user rather than
	// taken from its default (#850). "true"/"false"/"" (empty == false).
	// The runtime collection wiring (read-point resolver) lands in #852;
	// the loader parses it here so it survives load/round-trip.
	Collect string `xml:"collect,attr,omitempty" json:"collect,omitempty"`
	// Question is the metadata used to ask for a Collect field.
	Question *EDDQuestion `xml:"question,omitempty" json:"question,omitempty"`
}

// EDDQuestion describes how to ask the user for a Collect field (#850).
type EDDQuestion struct {
	Text    string      `xml:"text,attr" json:"text,omitempty"`
	Type    string      `xml:"type,attr" json:"type,omitempty"` // multiple_choice|ascii|number|date
	Options []EDDOption `xml:"option,omitempty" json:"options,omitempty"`
}

// EDDOption is one choice for a multiple_choice question.
type EDDOption struct {
	Value string `xml:"value,attr" json:"value"`
	Label string `xml:"label,attr" json:"label,omitempty"`
}

// Load loads an EDD from an io.Reader.
// The input size is limited by MaxXMLSize (default 10 MB) to prevent memory exhaustion.
// The format (XML or JSON) is auto-detected based on the first non-whitespace character.
func (l *EDDLoader) Load(r io.Reader) error {
	// Detect format first
	format, newReader, err := DetectFormat(r)
	if err != nil {
		return fmt.Errorf("failed to detect EDD format: %w", err)
	}
	r = newReader

	// Apply size limit if configured
	if MaxXMLSize > 0 {
		r = io.LimitReader(r, MaxXMLSize+1) // +1 to detect overflow
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read EDD: %w", err)
	}

	// Check if we hit the size limit
	if MaxXMLSize > 0 && int64(len(data)) > MaxXMLSize {
		return fmt.Errorf("EDD exceeds maximum size limit of %d bytes", MaxXMLSize)
	}

	var edd EDDFile
	var usedLegacy bool

	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &edd); err != nil {
			return fmt.Errorf("failed to parse EDD JSON: %w", err)
		}
	case FormatXML:
		// Try standard entity_data_dictionary format first
		if err := xml.Unmarshal(data, &edd); err != nil || len(edd.Entities) == 0 {
			// Try legacy entity_dictionary format
			var legacyEdd EDDFileLegacy
			if legacyErr := xml.Unmarshal(data, &legacyEdd); legacyErr == nil && len(legacyEdd.Entities) > 0 {
				// Convert legacy format to standard format
				edd = EDDFile{}
				for _, legacyEnt := range legacyEdd.Entities {
					ent := EDDEntity{
						Name:   legacyEnt.Name,
						Access: "rw",
						Fields: make([]EDDField, len(legacyEnt.Fields)),
					}
					for i, f := range legacyEnt.Fields {
						ent.Fields[i] = EDDField{
							Name:         f.Name,
							Type:         f.Type,
							DefaultValue: f.DefaultValue,
							Comment:      f.Comment,
						}
					}
					edd.Entities = append(edd.Entities, ent)
				}
				usedLegacy = true
			} else if err != nil {
				return fmt.Errorf("failed to parse EDD XML: %w", err)
			}
		}
	default:
		return fmt.Errorf("unknown EDD format: %s", format)
	}

	// Capture file-level path from metadata
	fileLevelPath := edd.FileMetadata.FilePath
	_ = usedLegacy // We may use this for logging later

	// Process each entity
	for _, ent := range edd.Entities {
		if err := l.processEntity(&ent, fileLevelPath); err != nil {
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
func (l *EDDLoader) processEntity(ent *EDDEntity, fileLevelPath string) error {
	entityName := dtrules.GetRName(strings.TrimSpace(ent.Name))
	if entityName == nil {
		return fmt.Errorf("invalid entity name syntax: %s", ent.Name)
	}

	// Create or find the reference entity
	refEntity, err := l.factory.FindCreateRefEntity(false, entityName)
	if err != nil {
		return fmt.Errorf("failed to create entity %s: %w", ent.Name, err)
	}

	// Set entity comment and xls_file for export grouping
	if ent.Comment != "" {
		refEntity.SetComment(ent.Comment)
	}
	if ent.XlsFile != "" {
		refEntity.SetXlsFile(ent.XlsFile)
	}

	// Set file path from file-level metadata
	if fileLevelPath != "" {
		refEntity.SetFilePath(fileLevelPath)
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

	// Compute default value, applying field-level timezone (Phase 2 of #743)
	// when a date field declares one.
	defaultValue := l.computeDefaultValue(field.DefaultValue, rtype, field.Timezone)

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

	// Carry collect/question metadata onto the entity entry (#850) so the
	// runtime read-point and the Excel exporter can see it.
	if strings.EqualFold(field.Collect, "true") {
		if entry := refEntity.GetEntry(attributeName); entry != nil {
			entry.Collect = true
			if field.Question != nil {
				q := &entity.QuestionMeta{Text: field.Question.Text, Type: field.Question.Type}
				for _, o := range field.Question.Options {
					q.Options = append(q.Options, entity.QuestionOption{Value: o.Value, Label: o.Label})
				}
				entry.Question = q
			}
		}
	}
	return nil
}

// computeDefaultValue computes the default value for a given type. The
// tzName argument (Phase 2 of #743) is the field's declared timezone — when
// non-empty and the type is date, tz-naïve default strings are interpreted
// in that zone instead of UTC.
func (l *EDDLoader) computeDefaultValue(defaultStr string, rtype *dtrules.RType, tzName string) dtrules.Object {
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
				if tzName != "" {
					if rd, ok := dtrules.RebaseDateInZone(d, tzName); ok {
						return rd
					}
				}
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
func (l *EDDLoader) GetErrors() []error {
	return l.errors
}

// CollectionRef describes one owner-entity / field-name pair where an array
// of a given subtype is declared in the EDD.
type CollectionRef struct {
	OwnerEntity string
	FieldName   string
}

// BuildCollectionIndex walks every reference entity in the factory and
// records, for each array-typed field with a non-empty subtype, the
// (ownerEntity, fieldName) pair. The returned map is keyed by the lower-cased
// subtype (the entity-type name of the array elements).
//
// Used by the EL compiler to resolve `for all <type> entities` to the
// canonical `<owner>.<field>` path.
func BuildCollectionIndex(factory *entity.Factory) map[string][]CollectionRef {
	idx := map[string][]CollectionRef{}
	for _, ent := range factory.GetRefEntities() {
		ownerName := strings.ToLower(ent.GetName().StringValue())
		for _, entry := range ent.GetEntries() {
			if entry.Type == nil || entry.Type != dtrules.TypeArray {
				continue
			}
			subtype := strings.TrimSpace(entry.SubType)
			if subtype == "" {
				continue
			}
			key := strings.ToLower(subtype)
			idx[key] = append(idx[key], CollectionRef{
				OwnerEntity: ownerName,
				FieldName:   strings.ToLower(entry.Attribute.StringValue()),
			})
		}
	}
	return idx
}

// MakeCollectionResolver returns a resolver closure over the given index
// suitable for handing to the EL compiler. When multiple owners declare a
// collection of the same subtype, the resolver returns an error listing all
// candidates; when none match, it returns a "no collection registered for
// type" error.
func MakeCollectionResolver(idx map[string][]CollectionRef) func(entityType string) (string, string, error) {
	return func(entityType string) (string, string, error) {
		key := strings.ToLower(strings.TrimSpace(entityType))
		refs, ok := idx[key]
		if !ok || len(refs) == 0 {
			return "", "", fmt.Errorf("no collection registered for type %q", entityType)
		}
		if len(refs) > 1 {
			candidates := make([]string, len(refs))
			for i, r := range refs {
				candidates[i] = r.OwnerEntity + "." + r.FieldName
			}
			return "", "", fmt.Errorf("ambiguous collection for type %q: candidates [%s]", entityType, strings.Join(candidates, ", "))
		}
		return refs[0].OwnerEntity, refs[0].FieldName, nil
	}
}
