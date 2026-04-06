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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// WorkbookResult contains both DT and EDD from a single workbook.
type WorkbookResult struct {
	DTables  *DecisionTablesXML
	EDD      *EDDXML
	FilePath string // Relative path from base directory
}

// WorkbookImporter imports complete workbooks with both DT and EDD sheets.
type WorkbookImporter struct {
	dtImporter  *DTImporter
	eddImporter *EDDImporter
	verbose     bool
}

// NewWorkbookImporter creates a new workbook importer.
func NewWorkbookImporter() *WorkbookImporter {
	return &WorkbookImporter{
		dtImporter:  NewDTImporter(),
		eddImporter: NewEDDImporter(),
	}
}

// SetVerbose enables verbose output during import.
func (w *WorkbookImporter) SetVerbose(v bool) {
	w.verbose = v
	w.dtImporter.SetVerbose(v)
}

// ImportWorkbook reads a single Excel file and extracts both DT and EDD sheets.
// It detects sheet types automatically:
// - EDD sheet: Sheet named "EDD" OR first row has headers "Entity, Attribute, Type, SubType..."
// - DT sheet: Cell A1 contains "Decision Table" or "Name:" pattern
func (w *WorkbookImporter) ImportWorkbook(excelPath string) (*WorkbookResult, error) {
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	result := &WorkbookResult{
		FilePath: filepath.Base(excelPath),
		DTables:  &DecisionTablesXML{},
		EDD:      &EDDXML{Version: "2", Entities: make([]*EDDXMLEntity, 0)},
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	// Categorize sheets
	var eddSheets []string
	var dtSheets []string

	for _, sheet := range sheets {
		if w.isEDDSheet(f, sheet) {
			eddSheets = append(eddSheets, sheet)
		} else if w.isDTSheet(f, sheet) {
			dtSheets = append(dtSheets, sheet)
		}
	}

	if w.verbose {
		fmt.Printf("Workbook %s: found %d EDD sheets, %d DT sheets\n",
			filepath.Base(excelPath), len(eddSheets), len(dtSheets))
	}

	// Process EDD sheets
	if len(eddSheets) > 0 {
		w.eddImporter.SourceFile = filepath.Base(excelPath)
		for _, sheet := range eddSheets {
			eddData, err := w.importEDDSheetFromFile(f, sheet)
			if err != nil {
				if w.verbose {
					fmt.Printf("  Warning: failed to parse EDD sheet %s: %v\n", sheet, err)
				}
				continue
			}
			result.EDD.Entities = append(result.EDD.Entities, eddData.Entities...)
		}
	}

	// Process DT sheets
	for _, sheet := range dtSheets {
		if w.verbose {
			fmt.Printf("  Processing DT sheet: %s\n", sheet)
		}
		table, err := w.parseDTSheet(f, sheet, filepath.Base(excelPath))
		if err != nil {
			if w.verbose {
				fmt.Printf("  Warning: failed to parse DT sheet %s: %v\n", sheet, err)
			}
			continue
		}
		if table != nil {
			result.DTables.Tables = append(result.DTables.Tables, *table)
		}
	}

	return result, nil
}

// ImportDirectory recursively processes all Excel files, preserving structure.
// Returns a slice of WorkbookResult with relative paths from the base directory.
func (w *WorkbookImporter) ImportDirectory(excelDir string) ([]*WorkbookResult, error) {
	var results []*WorkbookResult

	// Collect all xlsx files with their paths
	type fileEntry struct {
		path    string
		relPath string
	}
	var xlsxFiles []fileEntry

	err := filepath.WalkDir(excelDir, func(path string, d os.DirEntry, err error) error {
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

		// Calculate relative path for metadata
		relPath, err := filepath.Rel(excelDir, path)
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
	sort.Slice(xlsxFiles, func(i, j int) bool {
		return xlsxFiles[i].relPath < xlsxFiles[j].relPath
	})

	// Process each file
	for _, entry := range xlsxFiles {
		if w.verbose {
			fmt.Printf("Processing workbook: %s\n", entry.relPath)
		}

		result, err := w.ImportWorkbook(entry.path)
		if err != nil {
			if w.verbose {
				fmt.Printf("  Warning: %v\n", err)
			}
			continue
		}

		// Update FilePath to relative path from base directory
		result.FilePath = entry.relPath

		// Update xls_file metadata in all tables and entities
		for idx := range result.DTables.Tables {
			result.DTables.Tables[idx].XLSFile = entry.relPath
		}
		for _, ent := range result.EDD.Entities {
			ent.XlsFile = entry.relPath
		}

		results = append(results, result)
	}

	return results, nil
}

// stripDTOrEDDSuffix removes _dt or _edd suffix from a base name to prevent doubling.
// For example: "TaxReturn_edd" -> "TaxReturn", "CO_dt" -> "CO", "MyFile" -> "MyFile"
func stripDTOrEDDSuffix(baseName string) string {
	if strings.HasSuffix(baseName, "_edd") {
		return strings.TrimSuffix(baseName, "_edd")
	}
	if strings.HasSuffix(baseName, "_dt") {
		return strings.TrimSuffix(baseName, "_dt")
	}
	return baseName
}

// WriteAll writes all results to XML, mirroring the Excel directory structure.
// For example: excel/states/CO.xlsx -> xml/states/CO_dt.xml + xml/states/CO_edd.xml
func (w *WorkbookImporter) WriteAll(results []*WorkbookResult, xmlDir string) error {
	for _, result := range results {
		// Calculate output paths based on relative path
		// Remove .xlsx extension and add appropriate suffix
		// Also strip any existing _dt or _edd suffix to prevent doubling
		basePath := stripDTOrEDDSuffix(strings.TrimSuffix(result.FilePath, filepath.Ext(result.FilePath)))

		// Write DT file if there are tables
		if len(result.DTables.Tables) > 0 {
			dtPath := filepath.Join(xmlDir, basePath+"_dt.xml")

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(dtPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", dtPath, err)
			}

			if err := w.dtImporter.WriteXML(result.DTables, dtPath); err != nil {
				return fmt.Errorf("failed to write DT XML %s: %w", dtPath, err)
			}

			if w.verbose {
				fmt.Printf("Wrote %d tables to %s\n", len(result.DTables.Tables), dtPath)
			}
		}

		// Write EDD file if there are entities
		if len(result.EDD.Entities) > 0 {
			eddPath := filepath.Join(xmlDir, basePath+"_edd.xml")

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(eddPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", eddPath, err)
			}

			if err := w.eddImporter.WriteXML(result.EDD, eddPath); err != nil {
				return fmt.Errorf("failed to write EDD XML %s: %w", eddPath, err)
			}

			if w.verbose {
				fmt.Printf("Wrote %d entities to %s\n", len(result.EDD.Entities), eddPath)
			}
		}
	}

	return nil
}

// MergeResults merges all WorkbookResults into single DT and EDD structures.
func (w *WorkbookImporter) MergeResults(results []*WorkbookResult) (*DecisionTablesXML, *EDDXML) {
	mergedDT := &DecisionTablesXML{}
	var edds []*EDDXML

	for _, result := range results {
		if result.DTables != nil {
			mergedDT.Tables = append(mergedDT.Tables, result.DTables.Tables...)
		}
		if result.EDD != nil && len(result.EDD.Entities) > 0 {
			edds = append(edds, result.EDD)
		}
	}

	mergedEDD := MergeEDD(edds...)

	return mergedDT, mergedEDD
}

// isEDDSheet detects if a sheet contains EDD data.
// Returns true if:
// - Sheet is named "EDD"
// - First row has EDD header pattern (Entity, Attribute, Type, SubType...)
// - Sheet uses multi-sheet entity format (A1="Entity", B1=entity_name)
func (w *WorkbookImporter) isEDDSheet(f *excelize.File, sheet string) bool {
	// Check sheet name
	if sheet == "EDD" {
		return true
	}

	// Get first row
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return false
	}

	row := rows[0]
	if len(row) == 0 {
		return false
	}

	firstCell := strings.ToLower(strings.TrimSpace(row[0]))

	// Check for single-sheet EDD format header
	if firstCell == "entity" {
		// Could be header row "Entity, Attribute, Type..." or
		// multi-sheet format "Entity, <entity_name>"
		if len(row) > 1 {
			secondCell := strings.ToLower(strings.TrimSpace(row[1]))
			// Single-sheet format has "Attribute" as second column header
			if secondCell == "attribute" {
				return true
			}
			// Multi-sheet format has entity name in second column
			// Distinguish from DT by checking for typical EDD structure
			if len(rows) > 1 {
				// Multi-sheet EDD format has "Access" in A2
				if len(rows[1]) > 0 {
					a2 := strings.ToLower(strings.TrimSpace(rows[1][0]))
					if a2 == "access" {
						return true
					}
				}
			}
		}
	}

	return false
}

// isDTSheet detects if a sheet contains Decision Table data.
// Returns true if:
// - Cell A1 contains "Decision Table"
// - Cell A1 contains "Name:" pattern
// - Sheet contains "Conditions:" marker
func (w *WorkbookImporter) isDTSheet(f *excelize.File, sheet string) bool {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return false
	}

	// Check first cell
	if len(rows[0]) > 0 {
		firstCell := strings.ToLower(strings.TrimSpace(rows[0][0]))

		// dt2excel format starts with "Decision Table"
		if strings.HasPrefix(firstCell, "decision table") {
			return true
		}

		// exporter.go format starts with "Name:"
		if strings.HasPrefix(firstCell, "name:") {
			return true
		}
	}

	// Look for DT markers in first 20 rows
	maxRows := 20
	if len(rows) < maxRows {
		maxRows = len(rows)
	}

	for i := 0; i < maxRows; i++ {
		if len(rows[i]) > 0 {
			cell := strings.ToLower(strings.TrimSpace(rows[i][0]))
			if cell == "conditions:" || cell == "conditions" {
				return true
			}
			if cell == "actions:" || cell == "actions" {
				return true
			}
		}
	}

	return false
}

// importEDDSheetFromFile imports EDD data from a specific sheet in an open file.
func (w *WorkbookImporter) importEDDSheetFromFile(f *excelize.File, sheet string) (*EDDXML, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %s: %w", sheet, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet %s has no data rows", sheet)
	}

	// Detect format based on first row
	if len(rows[0]) >= 2 {
		firstCell := strings.ToLower(strings.TrimSpace(rows[0][0]))
		secondCell := strings.ToLower(strings.TrimSpace(rows[0][1]))

		// Multi-sheet format: "Entity", <entity_name>
		if firstCell == "entity" && secondCell != "attribute" {
			return w.parseMultiSheetEntityFromRows(rows, sheet)
		}
	}

	// Single-sheet EDD format
	return w.parseEDDSheetFromRows(rows, sheet)
}

// parseMultiSheetEntityFromRows parses EDD data from rows in multi-sheet format.
func (w *WorkbookImporter) parseMultiSheetEntityFromRows(rows [][]string, sheetName string) (*EDDXML, error) {
	edd := &EDDXML{
		Version:  "2",
		Entities: make([]*EDDXMLEntity, 0),
	}

	if len(rows) < 5 {
		return nil, nil // Not enough rows for entity data
	}

	// Extract entity metadata
	entityName := strings.TrimSpace(getCellValue(rows[0], 1))
	if entityName == "" {
		entityName = sheetName // Use sheet name as fallback
	}

	entity := &EDDXMLEntity{
		Name:    entityName,
		XlsFile: w.eddImporter.SourceFile,
		Access:  strings.TrimSpace(getCellValue(rows[1], 1)),
		Comment: strings.TrimSpace(getCellValue(rows[2], 1)),
		Fields:  make([]*EDDXMLField, 0),
	}

	if entity.Access == "" {
		entity.Access = "rw"
	}

	// Parse fields starting from row 6 (index 5)
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

	edd.Entities = append(edd.Entities, entity)
	return edd, nil
}

// parseEDDSheetFromRows parses EDD data from rows in single-sheet format.
func (w *WorkbookImporter) parseEDDSheetFromRows(rows [][]string, _ string) (*EDDXML, error) {
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
		colA := strings.TrimSpace(getCellValue(row, 0))
		colB := strings.TrimSpace(getCellValue(row, 1))

		if colA != "" {
			// This is an entity header row
			if isEntityHeader(colB) || (colA != "" && colB == "") {
				// Start a new entity
				currentEntity = &EDDXMLEntity{
					Name:    colA,
					XlsFile: w.eddImporter.SourceFile,
					Access:  "rw",
					Fields:  make([]*EDDXMLField, 0),
				}
				edd.Entities = append(edd.Entities, currentEntity)
				continue
			}
		}

		// This is an attribute row
		if currentEntity == nil {
			continue
		}

		// Extract attribute values
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

// parseDTSheet parses a decision table from a sheet in an open file.
func (w *WorkbookImporter) parseDTSheet(f *excelize.File, sheetName, xlsFile string) (*DecisionTableXML, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty sheet")
	}

	table := &DecisionTableXML{
		XLSFile: xlsFile,
	}

	// Detect format
	format := w.detectDTFormat(rows)

	switch format {
	case "dt2excel":
		return w.dtImporter.parseDT2ExcelFormat(rows, table)
	case "exporter":
		return w.dtImporter.parseExporterFormat(rows, sheetName, table)
	default:
		return nil, fmt.Errorf("unrecognized Excel format")
	}
}

// detectDTFormat detects which Excel format the sheet uses.
func (w *WorkbookImporter) detectDTFormat(rows [][]string) string {
	if len(rows) == 0 {
		return "unknown"
	}

	// Check first row for format indicators
	if len(rows[0]) > 0 {
		firstCell := strings.ToLower(strings.TrimSpace(rows[0][0]))

		// dt2excel format starts with "Decision Table"
		if firstCell == "decision table" {
			return "dt2excel"
		}

		// exporter.go format starts with "Name: <table_name>"
		if strings.HasPrefix(firstCell, "name:") {
			return "exporter"
		}
	}

	// Try to detect by looking for other markers
	for _, row := range rows {
		if len(row) > 0 {
			cell := strings.ToLower(strings.TrimSpace(row[0]))
			if cell == "conditions:" {
				return "exporter"
			}
			if cell == "conditions" && len(row) > 1 {
				return "dt2excel"
			}
		}
	}

	return "unknown"
}

// =============================================================================
// sync.WorkbookImporter interface implementation
// =============================================================================

// HasDTSheet checks if the workbook contains a decision table sheet.
// Implements sync.WorkbookImporter interface.
func (w *WorkbookImporter) HasDTSheet(excelPath string) (bool, error) {
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return false, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		if w.isDTSheet(f, sheet) {
			return true, nil
		}
	}

	return false, nil
}

// HasEDDSheet checks if the workbook contains an EDD sheet.
// Implements sync.WorkbookImporter interface.
func (w *WorkbookImporter) HasEDDSheet(excelPath string) (bool, error) {
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return false, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		if w.isEDDSheet(f, sheet) {
			return true, nil
		}
	}

	return false, nil
}

// ImportWorkbookToDir imports a workbook to separate XML files in the given directory.
// Returns the paths to the generated XML files (dtPath, eddPath).
// Either path may be empty if that sheet type doesn't exist in the workbook.
// Implements sync.WorkbookImporter interface.
func (w *WorkbookImporter) ImportWorkbookToDir(excelPath, xmlDir string) (dtPath, eddPath string, err error) {
	result, err := w.ImportWorkbook(excelPath)
	if err != nil {
		return "", "", err
	}

	// Generate base name from Excel file
	// Strip any existing _dt or _edd suffix to prevent doubling (e.g., TaxReturn_edd.xlsx -> TaxReturn_edd.xml, not TaxReturn_edd_edd.xml)
	baseName := stripDTOrEDDSuffix(strings.TrimSuffix(filepath.Base(excelPath), filepath.Ext(excelPath)))

	// Ensure directory exists
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write DT XML if it has tables
	if result.DTables != nil && len(result.DTables.Tables) > 0 {
		dtPath = filepath.Join(xmlDir, baseName+"_dt.xml")
		if err := w.dtImporter.WriteXML(result.DTables, dtPath); err != nil {
			return "", "", fmt.Errorf("failed to write DT XML: %w", err)
		}
	}

	// Write EDD XML if it has entities
	if result.EDD != nil && len(result.EDD.Entities) > 0 {
		eddPath = filepath.Join(xmlDir, baseName+"_edd.xml")
		if err := w.eddImporter.WriteXML(result.EDD, eddPath); err != nil {
			return "", "", fmt.Errorf("failed to write EDD XML: %w", err)
		}
	}

	return dtPath, eddPath, nil
}
