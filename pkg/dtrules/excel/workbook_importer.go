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
	stats       *ImportStats // accumulated stats across all ImportWorkbookToDir calls
	// projectSymbols is every EDD in the project, field→type. Held so each
	// workbook's own EDD can be layered over a clean copy of it rather than
	// replacing it for the rest of the run.
	projectSymbols map[string]string
}

// NewWorkbookImporter creates a new workbook importer.
func NewWorkbookImporter() *WorkbookImporter {
	return &WorkbookImporter{
		dtImporter:  NewDTImporter(),
		eddImporter: NewEDDImporter(),
	}
}

// ResetStats resets the accumulated import statistics.
func (w *WorkbookImporter) ResetStats() {
	w.stats = &ImportStats{}
	w.dtImporter.SetStats(w.stats)
}

// TakeStats returns the accumulated import statistics and resets the collector.
func (w *WorkbookImporter) TakeStats() *ImportStats {
	s := w.stats
	w.stats = nil
	w.dtImporter.SetStats(nil)
	return s
}

// SetVerbose enables verbose output during import.
func (w *WorkbookImporter) SetVerbose(v bool) {
	w.verbose = v
	w.dtImporter.SetVerbose(v)
}

// SetELCompiler wires an EL compiler into the inner DT importer so the
// build summary can report compile failures as named drops instead of
// silently passing uncompilable DSL through.
func (w *WorkbookImporter) SetELCompiler(c ELCompiler) {
	w.dtImporter.SetELCompiler(c)
}

// SetSymbols seeds the inner DT importer with a project-wide EDD field→type
// map so the EL compiler types operands correctly even when a decision table
// lives in a different workbook than its EDD (multi-file projects). A workbook
// that carries its own EDD layers its entities over these during import;
// DT-only workbooks use this project-wide map as-is.
//
// The map is kept here as well as handed down, because the per-workbook layer
// is rebuilt from it for each workbook rather than written over it.
func (w *WorkbookImporter) SetSymbols(symbols map[string]string) {
	w.projectSymbols = symbols
	w.dtImporter.SetSymbols(symbols)
}

// symbolsFor layers one workbook's own EDD over the project-wide map.
//
// This used to replace the project map outright, which broke multi-file
// projects in both directions. A workbook's tables could only see the entities
// declared in that same workbook, so a state table reading the shared
// `apportionment` entity -- declared in the core workbook -- found no type for
// it and compiled money as integers: `f-` became `-` and `cvd` became `cvi`,
// silently. And because one importer is reused for every workbook in the
// build loop with no save/restore, the replacement leaked forward: a DT-only
// workbook inherited whichever EDD happened to be imported before it.
//
// The workbook's own declarations still win, which is what "overrides" was
// reaching for -- an entity is defined by the EDD it ships with.
func (w *WorkbookImporter) symbolsFor(edd *EDDXML) map[string]string {
	own := EDDSymbols(edd)
	if len(own) == 0 {
		return w.projectSymbols
	}
	merged := make(map[string]string, len(w.projectSymbols)+len(own))
	for k, v := range w.projectSymbols {
		merged[k] = v
	}
	for k, v := range own {
		merged[k] = v
	}
	return merged
}

// ImportWorkbook reads a single Excel file and extracts both DT and EDD sheets.
// It detects sheet types automatically:
// - EDD sheet: Sheet named "EDD" OR first row has headers "Entity, Attribute, Type, SubType..."
// - DT sheet: Cell A1 contains "Decision Table" or "Name:" pattern
func (w *WorkbookImporter) ImportWorkbook(excelPath string) (*WorkbookResult, error) {
	return w.importWorkbookWithSource(excelPath, filepath.Base(excelPath), "")
}

// EDDSymbols flattens a parsed EDD into the field→type map the EL compiler
// expects, keyed by both the bare field name and the entity-qualified name
// (e.g. "lean_body_weight" and "patient.lean_body_weight"). This is the
// content-based (workbook-import) builder; it must agree with the file-based
// authoring.LoadEDDSymbols so every generator types operands identically — a
// cross-generator parity test pins that (#874/#898). Exported for that test.
func EDDSymbols(edd *EDDXML) map[string]string {
	if edd == nil {
		return nil
	}
	symbols := make(map[string]string)
	for _, ent := range edd.Entities {
		if ent == nil {
			continue
		}
		// EL is case-insensitive: lower-cased keys/types, matching the
		// emitter's lower-cased lookups (see authoring.LoadEDDSymbols).
		entName := strings.ToLower(ent.Name)
		for _, fld := range ent.Fields {
			if fld == nil || fld.Name == "" || fld.Type == "" {
				continue
			}
			name := strings.ToLower(fld.Name)
			typ := strings.ToLower(fld.Type)
			symbols[name] = typ
			if entName != "" {
				symbols[entName+"."+name] = typ
			}
		}
	}
	return symbols
}

// importWorkbookWithSource reads a workbook and sets xlsFile and relPath for metadata.
func (w *WorkbookImporter) importWorkbookWithSource(excelPath, xlsFile, relPath string) (*WorkbookResult, error) {
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	srcRelPath := relPath
	if srcRelPath == "" {
		srcRelPath = filepath.Base(excelPath)
	}

	result := &WorkbookResult{
		FilePath: xlsFile,
		DTables:  &DecisionTablesXML{},
		EDD:      &EDDXML{Version: "2", Entities: make([]*EDDXMLEntity, 0)},
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	// Build sheet index map (1-based) for source tracking
	sheetIndexMap := make(map[string]int, len(sheets))
	for idx, name := range sheets {
		sheetIndexMap[name] = idx + 1
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
		w.eddImporter.SourceFile = xlsFile
		w.eddImporter.sourceRelPath = srcRelPath
		w.eddImporter.sheetIndexMap = sheetIndexMap
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

	// Wire the EDD field types into the compiler before compiling DT sheets.
	// Without this, double/fixed fields compile as integers — e.g. a Cockcroft
	// -Gault "(140 - age) * weight / (pcr * 72)" emits integer ops and cvi,
	// silently truncating the result to 0. Combined workbooks carry their EDD
	// in the same file (parsed above), so the symbols are available here; the
	// rest of the project's entities come from the project-wide map.
	if symbols := w.symbolsFor(result.EDD); len(symbols) > 0 {
		w.dtImporter.SetSymbols(symbols)
	}

	// Process DT sheets
	for _, sheet := range dtSheets {
		if w.verbose {
			fmt.Printf("  Processing DT sheet: %s\n", sheet)
		}
		table, err := w.parseDTSheet(f, sheet, xlsFile)
		if err != nil {
			if w.verbose {
				fmt.Printf("  Warning: failed to parse DT sheet %s: %v\n", sheet, err)
			}
			continue
		}
		if table != nil {
			// Compile EL expressions and record stats/drops before attaching metadata.
			if err := w.dtImporter.compileTableEL(table); err != nil {
				if w.verbose {
					fmt.Printf("  EL compilation warning in %s: %v\n", sheet, err)
				}
			}
			// Accumulate table/action/condition counts into stats.
			if w.stats != nil {
				w.stats.Tables++
				w.stats.Actions += len(table.Actions)
				w.stats.Conditions += len(table.Conditions)
			}
			// Attach source metadata
			table.Source = &SourceXML{
				RelativePath: srcRelPath,
				FileName:     filepath.Base(excelPath),
				SheetNumber:  sheetIndexMap[sheet],
			}
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

		result, err := w.importWorkbookWithSource(entry.path, entry.relPath, entry.relPath)
		if err != nil {
			if w.verbose {
				fmt.Printf("  Warning: %v\n", err)
			}
			continue
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
// - A1 contains "EDD:" type marker (new combined-workbook format)
// - Sheet is named "EDD"
// - First row has EDD header pattern (Entity, Attribute, Type, SubType...)
// - Sheet uses multi-sheet entity format (A1="Entity", B1=entity_name)
func (w *WorkbookImporter) isEDDSheet(f *excelize.File, sheet string) bool {
	// Get first row
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return false
	}

	row := rows[0]
	if len(row) == 0 {
		// Check sheet name as fallback
		return sheet == "EDD"
	}

	firstCell := strings.ToLower(strings.TrimSpace(row[0]))

	// New combined-workbook format: "EDD: <name>"
	if strings.HasPrefix(firstCell, "edd:") {
		return true
	}

	// Check sheet name
	if sheet == "EDD" {
		return true
	}

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

// isMAPSheet detects if a sheet contains Mapping data.
// MAP sheet support is stubbed — the A1 marker is recognized but
// no import logic is implemented yet.
func (w *WorkbookImporter) isMAPSheet(f *excelize.File, sheet string) bool {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 || len(rows[0]) == 0 {
		return false
	}
	firstCell := strings.ToLower(strings.TrimSpace(rows[0][0]))
	return strings.HasPrefix(firstCell, "map:")
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

		// New combined-workbook format: "DT: <table_name>"
		if strings.HasPrefix(firstCell, "dt:") {
			return true
		}

		// dt2excel format starts with "Decision Table"
		if strings.HasPrefix(firstCell, "decision table") {
			return true
		}

		// legacy exporter format starts with "Name:"
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

	// Strip the "EDD: ..." type marker row if present, so downstream parsers
	// see the same layout as the legacy (unmarked) format.
	if len(rows[0]) > 0 {
		firstCell := strings.ToLower(strings.TrimSpace(rows[0][0]))
		if strings.HasPrefix(firstCell, "edd:") {
			rows = rows[1:]
		}
	}

	if len(rows) < 1 {
		return nil, fmt.Errorf("sheet %s has no data rows after marker", sheet)
	}

	// Detect format based on first (now non-marker) row
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

	sheetNum := w.eddImporter.sheetIndexMap[sheetName]
	entity := &EDDXMLEntity{
		Name:    entityName,
		XlsFile: w.eddImporter.SourceFile,
		Access:  strings.TrimSpace(getCellValue(rows[1], 1)),
		Comment: strings.TrimSpace(getCellValue(rows[2], 1)),
		Fields:  make([]*EDDXMLField, 0),
		Source: &SourceXML{
			RelativePath: w.eddImporter.sourceRelPath,
			FileName:     filepath.Base(w.eddImporter.SourceFile),
			SheetNumber:  sheetNum,
		},
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
func (w *WorkbookImporter) parseEDDSheetFromRows(rows [][]string, sheetName string) (*EDDXML, error) {
	edd := &EDDXML{
		Version:  "2",
		Entities: make([]*EDDXMLEntity, 0),
	}

	sheetNum := w.eddImporter.sheetIndexMap[sheetName]

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
					Source: &SourceXML{
						RelativePath: w.eddImporter.sourceRelPath,
						FileName:     filepath.Base(w.eddImporter.SourceFile),
						SheetNumber:  sheetNum,
					},
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
		// Collect + question metadata (#850), columns I–M.
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

		// New combined-workbook format: "DT: <table_name>"
		if strings.HasPrefix(firstCell, "dt:") {
			return "exporter"
		}

		// dt2excel format starts with "Decision Table"
		if firstCell == "decision table" {
			return "dt2excel"
		}

		// legacy exporter format starts with "Name: <table_name>"
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

// HasMAPSheet checks if the workbook contains a mapping sheet.
func (w *WorkbookImporter) HasMAPSheet(excelPath string) (bool, error) {
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return false, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		if w.isMAPSheet(f, sheet) {
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
		if w.stats != nil {
			w.stats.Files++
		}
	}

	// Write EDD XML if it has entities
	if result.EDD != nil && len(result.EDD.Entities) > 0 {
		eddPath = filepath.Join(xmlDir, baseName+"_edd.xml")
		if err := w.eddImporter.WriteXML(result.EDD, eddPath); err != nil {
			return "", "", fmt.Errorf("failed to write EDD XML: %w", err)
		}
		if w.stats != nil {
			w.stats.Entities += len(result.EDD.Entities)
			w.stats.Files++
		}
	}

	return dtPath, eddPath, nil
}
