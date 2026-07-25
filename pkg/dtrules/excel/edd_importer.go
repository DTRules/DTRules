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

package excel

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// EDDXML represents the root EDD structure for XML output
type EDDXML struct {
	XMLName  xml.Name       `xml:"entity_data_dictionary"`
	Version  string         `xml:"version,attr"`
	Entities []*EDDXMLEntity `xml:"entity"`
}

// EDDXMLEntity represents an entity in the EDD XML
type EDDXMLEntity struct {
	Name    string          `xml:"name,attr"`
	// Number orders entities like TABLE_NUMBER orders decision tables.
	// Optional in authored XML; WriteXML backfills missing numbers in
	// increments of 100 so every emitted EDD is fully numbered.
	Number  string          `xml:"number,attr,omitempty"`
	XlsFile string          `xml:"xls_file,attr,omitempty"`
	Access  string          `xml:"access,attr"`
	Comment string          `xml:"comment,attr,omitempty"`
	Source  *SourceXML      `xml:"source,omitempty"`
	Fields  []*EDDXMLField  `xml:"field"`
}

// EDDXMLField represents a field in the EDD XML
type EDDXMLField struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	SubType      string `xml:"subtype,attr"`
	Access       string `xml:"access,attr"`
	Input        string `xml:"input,attr"`
	DefaultValue string `xml:"default_value,attr"`
	Comment      string `xml:"comment,attr"`
	// Collect marks a field whose value must be collected from the user
	// (asked) rather than taken from its default. "true"/"false"/"" (empty
	// == false). Distinct from Access: a collected field is always writable.
	// See #850.
	Collect string `xml:"collect,attr,omitempty"`
	// Question is the metadata used to ask for a Collect field.
	Question *EDDXMLQuestion `xml:"question,omitempty"`
}

// EDDXMLQuestion describes how to ask the user for a Collect field.
type EDDXMLQuestion struct {
	Text    string          `xml:"text,attr"`
	Type    string          `xml:"type,attr"` // multiple_choice | ascii | number | date
	Options []*EDDXMLOption `xml:"option,omitempty"`
	// Reference range for a number question (lab-report style, #850).
	RefLow  string `xml:"ref_low,attr,omitempty"`
	RefHigh string `xml:"ref_high,attr,omitempty"`
	Units   string `xml:"units,attr,omitempty"`
}

// EDDXMLOption is one choice for a multiple_choice question.
type EDDXMLOption struct {
	Value string `xml:"value,attr"`
	Label string `xml:"label,attr"`
}

// eddColumnCount is the number of columns in the EDD sheet. Columns A–H are
// the legacy field metadata; I–M carry the collect/question metadata (#850):
// I=Collect, J=Question text, K=Question type, L=Options, M=Reference range.
const eddColumnCount = 13

// encodeEDDOptions packs multiple_choice options into one Excel cell as
// `value=label|value=label`. Quick-and-dirty (#850): values/labels are
// assumed not to contain `=` or `|`.
func encodeEDDOptions(opts []EDDXMLOption) string {
	parts := make([]string, 0, len(opts))
	for _, o := range opts {
		if o.Label == "" {
			parts = append(parts, o.Value)
		} else {
			parts = append(parts, o.Value+"="+o.Label)
		}
	}
	return strings.Join(parts, "|")
}

// decodeEDDOptions parses the cell encoding produced by encodeEDDOptions.
func decodeEDDOptions(s string) []*EDDXMLOption {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []*EDDXMLOption
	for _, part := range strings.Split(s, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, label, found := strings.Cut(part, "=")
		o := &EDDXMLOption{Value: strings.TrimSpace(value)}
		if found {
			o.Label = strings.TrimSpace(label)
		}
		out = append(out, o)
	}
	return out
}

// encodeEDDRef packs a number question's reference range into one Excel cell
// as `low:high:units` (#850). Trailing empties are trimmed; "" when no bounds
// or units are set.
func encodeEDDRef(low, high, units string) string {
	if low == "" && high == "" && units == "" {
		return ""
	}
	parts := []string{low, high, units}
	for len(parts) > 1 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, ":")
}

// decodeEDDRef parses the cell encoding produced by encodeEDDRef.
func decodeEDDRef(s string) (low, high, units string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", ""
	}
	// SplitN with 3 so a units string containing ':' isn't truncated (M5).
	parts := strings.SplitN(s, ":", 3)
	get := func(i int) string {
		if i < len(parts) {
			return strings.TrimSpace(parts[i])
		}
		return ""
	}
	return get(0), get(1), get(2)
}

// questionFromCells builds an EDDXMLQuestion from the collect/question cells
// (columns I–M). Returns ("", nil) when the field is not collected.
func questionFromCells(collect, qText, qType, qOptions, qRef string) (string, *EDDXMLQuestion) {
	if !strings.EqualFold(strings.TrimSpace(collect), "true") {
		return "", nil
	}
	q := &EDDXMLQuestion{Text: strings.TrimSpace(qText), Type: strings.TrimSpace(qType)}
	q.Options = decodeEDDOptions(qOptions)
	q.RefLow, q.RefHigh, q.Units = decodeEDDRef(qRef)
	if q.Text == "" && q.Type == "" && len(q.Options) == 0 && q.RefLow == "" && q.RefHigh == "" && q.Units == "" {
		return "true", nil
	}
	return "true", q
}

// EDDImporter imports Entity Data Dictionary from Excel files
type EDDImporter struct {
	// SourceFile tracks the source file for xls_file metadata
	SourceFile string
	// sourceRelPath is the project-relative path for <source> element
	sourceRelPath string
	// sheetIndexMap maps sheet name to 1-based sheet number within the workbook
	sheetIndexMap map[string]int
}

// NewEDDImporter creates a new EDD importer
func NewEDDImporter() *EDDImporter {
	return &EDDImporter{}
}

// ImportEDD reads an Excel file and returns EDD XML structure.
// Supports two formats:
// 1. Single "EDD" sheet format (from exporter.go ExportEDD):
//   - Row 1: Headers (Entity, Attribute, Type, SubType, Default, Input, Access, Description)
//   - Entity header rows: entity name in col A, "(N attributes)" in col B
//   - Attribute rows: blank col A, attribute details in cols B-H
//
// 2. Multi-sheet format (from edd2excel, one sheet per entity):
//   - Each sheet name is the entity name
//   - Row 1: "Entity" label, entity name
//   - Row 2: "Access" label, access value
//   - Row 3: "Comment" label, comment value
//   - Row 5: Field headers (Field Name, Type, Subtype, Access, Input, Default, Comment)
//   - Row 6+: Field data
func (i *EDDImporter) ImportEDD(filename string) (*EDDXML, error) {
	return i.importEDDWithSource(filename, filepath.Base(filename), filepath.Base(filename))
}

// importEDDWithSource reads an Excel file and returns EDD XML, setting xlsFile and relPath
// for xls_file attribute and <source> element respectively.
func (i *EDDImporter) importEDDWithSource(filename, xlsFile, relPath string) (*EDDXML, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	i.SourceFile = xlsFile
	i.sourceRelPath = relPath

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	// Build sheet index map (1-based) for source tracking
	i.sheetIndexMap = make(map[string]int, len(sheets))
	for idx, name := range sheets {
		i.sheetIndexMap[name] = idx + 1
	}

	// Check if this is the single "EDD" sheet format
	for _, s := range sheets {
		if s == "EDD" {
			return i.parseEDDSheet(f, "EDD")
		}
	}

	// Check if this is the multi-sheet per-entity format
	// by looking at the first sheet's structure
	firstSheet := sheets[0]
	rows, err := f.GetRows(firstSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %s: %w", firstSheet, err)
	}

	// Multi-sheet format has "Entity" in A1 and entity name in B1
	if len(rows) > 0 && len(rows[0]) >= 2 {
		if strings.ToLower(strings.TrimSpace(rows[0][0])) == "entity" {
			return i.parseMultiSheetFormat(f)
		}
	}

	// Default: try parsing as single sheet EDD format
	return i.parseEDDSheet(f, firstSheet)
}

// ImportEDDFromDir reads all Excel files in a directory recursively and merges them into one EDD.
// Files are processed in sorted order within each directory level.
// The xls_file metadata is set to the relative path from the base directory.
func (i *EDDImporter) ImportEDDFromDir(dir string) (*EDDXML, error) {
	edd := &EDDXML{
		Version:  "2",
		Entities: make([]*EDDXMLEntity, 0),
	}

	// Collect all xlsx files with their paths
	type fileEntry struct {
		path    string
		relPath string
	}
	var xlsxFiles []fileEntry

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".xlsx") {
			return nil
		}
		// Skip temp files (Excel creates these while files are open)
		if strings.HasPrefix(name, "~$") {
			return nil
		}

		// Calculate relative path for xls_file metadata
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = name
		}

		xlsxFiles = append(xlsxFiles, fileEntry{path: path, relPath: relPath})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort by relative path for consistent ordering
	sort.Slice(xlsxFiles, func(a, b int) bool {
		return xlsxFiles[a].relPath < xlsxFiles[b].relPath
	})

	// Process each file
	for _, entry := range xlsxFiles {
		fileEDD, err := i.importEDDWithSource(entry.path, entry.relPath, entry.relPath)
		if err != nil {
			return nil, fmt.Errorf("failed to import %s: %w", entry.relPath, err)
		}

		edd.Entities = append(edd.Entities, fileEDD.Entities...)
	}

	return edd, nil
}

// normalizeEntityNumbers backfills missing entity numbers. Entities that
// already carry a numeric number keep it; the rest are assigned numbers in
// document order, in increments of 100, continuing above the highest
// existing number (starting at 100 when none are numbered).
func normalizeEntityNumbers(edd *EDDXML) {
	max := 0
	for _, e := range edd.Entities {
		if n, err := strconv.Atoi(strings.TrimSpace(e.Number)); err == nil && n > max {
			max = n
		}
	}
	next := (max/100)*100 + 100
	for _, e := range edd.Entities {
		if _, err := strconv.Atoi(strings.TrimSpace(e.Number)); err != nil {
			e.Number = strconv.Itoa(next)
			next += 100
		}
	}
}

// WriteXML writes the EDD to an XML file
func (i *EDDImporter) WriteXML(edd *EDDXML, filename string) error {
	if edd.Version == "" {
		edd.Version = "2"
	}
	normalizeEntityNumbers(edd)

	output, err := xml.MarshalIndent(edd, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML header
	xmlContent := xml.Header + string(output) + "\n"

	if err := os.WriteFile(filename, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// parseMultiSheetFormat parses an Excel file where each sheet represents an entity.
// This is the format produced by edd2excel tool.
func (i *EDDImporter) parseMultiSheetFormat(f *excelize.File) (*EDDXML, error) {
	edd := &EDDXML{
		Version:  "2",
		Entities: make([]*EDDXMLEntity, 0),
	}

	sheets := f.GetSheetList()
	for _, sheetName := range sheets {
		entity, err := i.parseEntitySheet(f, sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sheet %s: %w", sheetName, err)
		}
		if entity != nil {
			edd.Entities = append(edd.Entities, entity)
		}
	}

	return edd, nil
}

// parseEntitySheet parses a single entity sheet from the multi-sheet format.
// Format:
// - Row 1: "Entity", entity_name
// - Row 2: "Access", access_value
// - Row 3: "Comment", comment_value
// - Row 4: (blank)
// - Row 5: Field Name, Type, Subtype, Access, Input, Default, Comment
// - Row 6+: field data
func (i *EDDImporter) parseEntitySheet(f *excelize.File, sheetName string) (*EDDXMLEntity, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) < 5 {
		return nil, nil // Not enough rows for entity data
	}

	// Extract entity metadata
	entityName := strings.TrimSpace(getCellValue(rows[0], 1))
	if entityName == "" {
		entityName = sheetName // Use sheet name as fallback
	}

	sheetNum := i.sheetIndexMap[sheetName]
	entity := &EDDXMLEntity{
		Name:    entityName,
		XlsFile: i.SourceFile,
		Access:  strings.TrimSpace(getCellValue(rows[1], 1)),
		Comment: strings.TrimSpace(getCellValue(rows[2], 1)),
		Fields:  make([]*EDDXMLField, 0),
		Source: &SourceXML{
			RelativePath: i.sourceRelPath,
			FileName:     filepath.Base(i.SourceFile),
			SheetNumber:  sheetNum,
		},
	}

	if entity.Access == "" {
		entity.Access = "rw"
	}

	// Parse fields starting from row 6 (index 5)
	// Headers are in row 5 (index 4): Field Name, Type, Subtype, Access, Input, Default, Comment
	for rowIdx := 5; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if len(row) == 0 {
			continue
		}

		fieldName := strings.TrimSpace(getCellValue(row, 0))
		if fieldName == "" {
			continue
		}

		field := &EDDXMLField{
			Name:         fieldName,
			Type:         strings.TrimSpace(getCellValue(row, 1)),
			SubType:      strings.TrimSpace(getCellValue(row, 2)),
			Access:       strings.TrimSpace(getCellValue(row, 3)),
			Input:        strings.TrimSpace(getCellValue(row, 4)),
			DefaultValue: strings.TrimSpace(getCellValue(row, 5)),
			Comment:      strings.TrimSpace(getCellValue(row, 6)),
		}

		// Apply defaults
		if field.Type == "" {
			field.Type = "string"
		}
		if field.Access == "" {
			field.Access = "rw"
		}

		entity.Fields = append(entity.Fields, field)
	}

	return entity, nil
}

// parseEDDSheet parses an EDD sheet from an Excel file
func (i *EDDImporter) parseEDDSheet(f *excelize.File, sheetName string) (*EDDXML, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %s: %w", sheetName, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet %s has no data rows", sheetName)
	}

	edd := &EDDXML{
		Version:  "2",
		Entities: make([]*EDDXMLEntity, 0),
	}

	// Skip header row (row 0)
	var currentEntity *EDDXMLEntity

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if len(row) == 0 {
			continue
		}

		// Check if this is an entity header row
		// Entity header rows have a value in column A (entity name) and
		// typically have "(N attributes)" in column B
		colA := strings.TrimSpace(getCellValue(row, 0))
		colB := strings.TrimSpace(getCellValue(row, 1))

		if colA != "" {
			// This is an entity header row
			// Check if column B contains "(N attributes)" pattern
			if isEntityHeader(colB) || (colA != "" && colB == "") {
				// Start a new entity
				sheetNum := i.sheetIndexMap[sheetName]
				currentEntity = &EDDXMLEntity{
					Name:    colA,
					XlsFile: i.SourceFile,
					Access:  "rw", // default access
					Fields:  make([]*EDDXMLField, 0),
					Source: &SourceXML{
						RelativePath: i.sourceRelPath,
						FileName:     filepath.Base(i.SourceFile),
						SheetNumber:  sheetNum,
					},
				}
				edd.Entities = append(edd.Entities, currentEntity)
				continue
			}
		}

		// This is an attribute row (column A is blank, or we're parsing entity with inline attributes)
		if currentEntity == nil {
			// No current entity, skip this row
			continue
		}

		// Extract attribute values
		// Columns: Entity(A), Attribute(B), Type(C), SubType(D), Default(E), Input(F), Access(G), Description(H),
		//          Collect(I), Question(J), Q Type(K), Options(L)  — I–L are #850 metadata
		attrName := strings.TrimSpace(getCellValue(row, 1))
		if attrName == "" {
			continue
		}

		field := &EDDXMLField{
			Name:         attrName,
			Type:         strings.TrimSpace(getCellValue(row, 2)),
			SubType:      strings.TrimSpace(getCellValue(row, 3)),
			DefaultValue: strings.TrimSpace(getCellValue(row, 4)),
			Input:        strings.TrimSpace(getCellValue(row, 5)),
			Access:       strings.TrimSpace(getCellValue(row, 6)),
			Comment:      strings.TrimSpace(getCellValue(row, 7)),
		}
		field.Collect, field.Question = questionFromCells(
			getCellValue(row, 8), getCellValue(row, 9), getCellValue(row, 10), getCellValue(row, 11), getCellValue(row, 12))

		// Apply defaults
		if field.Type == "" {
			field.Type = "string"
		}
		if field.Access == "" {
			field.Access = "rw"
		}

		currentEntity.Fields = append(currentEntity.Fields, field)
	}

	return edd, nil
}

// getCellValue safely gets a cell value from a row
func getCellValue(row []string, col int) string {
	if col < len(row) {
		return row[col]
	}
	return ""
}

// isEntityHeader checks if a string matches the "(N attributes)" pattern
func isEntityHeader(s string) bool {
	// Match patterns like "(5 attributes)", "(12 attributes)", etc.
	pattern := regexp.MustCompile(`^\(\d+\s+attributes?\)$`)
	return pattern.MatchString(strings.ToLower(s))
}

// MergeEDD merges multiple EDD structures into one
func MergeEDD(edds ...*EDDXML) *EDDXML {
	result := &EDDXML{
		Version:  "2",
		Entities: make([]*EDDXMLEntity, 0),
	}

	// Use a map to merge entities with the same name
	entityMap := make(map[string]*EDDXMLEntity)

	for _, edd := range edds {
		for _, ent := range edd.Entities {
			existing, ok := entityMap[ent.Name]
			if !ok {
				// Clone the entity
				cloned := &EDDXMLEntity{
					Name:    ent.Name,
					XlsFile: ent.XlsFile,
					Access:  ent.Access,
					Comment: ent.Comment,
					Fields:  make([]*EDDXMLField, 0, len(ent.Fields)),
				}
				for _, field := range ent.Fields {
					cloned.Fields = append(cloned.Fields, &EDDXMLField{
						Name:         field.Name,
						Type:         field.Type,
						SubType:      field.SubType,
						DefaultValue: field.DefaultValue,
						Input:        field.Input,
						Access:       field.Access,
						Comment:      field.Comment,
						Collect:      field.Collect,
						Question:     field.Question,
					})
				}
				entityMap[ent.Name] = cloned
				result.Entities = append(result.Entities, cloned)
			} else {
				// Merge fields into existing entity
				fieldMap := make(map[string]bool)
				for _, f := range existing.Fields {
					fieldMap[f.Name] = true
				}
				for _, field := range ent.Fields {
					if !fieldMap[field.Name] {
						existing.Fields = append(existing.Fields, &EDDXMLField{
							Name:         field.Name,
							Type:         field.Type,
							SubType:      field.SubType,
							DefaultValue: field.DefaultValue,
							Input:        field.Input,
							Access:       field.Access,
							Comment:      field.Comment,
							Collect:      field.Collect,
							Question:     field.Question,
						})
						fieldMap[field.Name] = true
					}
				}
			}
		}
	}

	return result
}
