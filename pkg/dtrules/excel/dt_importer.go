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

// Package excel provides Excel import/export functionality for DTRules.
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

// DTImporter imports decision tables from Excel files.
type DTImporter struct {
	verbose     bool
	compileEL   bool                         // If true, compile EL expressions to postfix
	elCompiler  ELCompiler                   // Optional EL compiler
	symbols     map[string]string            // Symbol table for EL compilation
}

// ELCompiler defines the interface for compiling EL expressions to postfix.
// This is implemented by the el.Compiler in pkg/dtrules/compiler/el.
type ELCompiler interface {
	// SetSymbols sets the symbol table for type resolution.
	SetSymbols(symbols map[string]string)
	// CompileCondition compiles a condition expression to postfix.
	CompileCondition(el string) (string, error)
	// CompileAction compiles an action statement to postfix.
	CompileAction(el string) (string, error)
	// CompileContext compiles a context statement to postfix.
	CompileContext(el string) (string, error)
}

// NewDTImporter creates a new decision table importer.
func NewDTImporter() *DTImporter {
	return &DTImporter{}
}

// SetVerbose enables verbose output during import.
func (i *DTImporter) SetVerbose(v bool) {
	i.verbose = v
}

// SetELCompiler sets the EL compiler for compiling expressions to postfix.
// When set, imported EL descriptions are compiled to postfix automatically.
func (i *DTImporter) SetELCompiler(compiler ELCompiler) {
	i.elCompiler = compiler
	i.compileEL = compiler != nil
}

// SetSymbols sets the symbol table used for EL compilation.
// The symbol table maps identifier names to their types (entity, long, double, etc.)
func (i *DTImporter) SetSymbols(symbols map[string]string) {
	i.symbols = symbols
	if i.elCompiler != nil {
		i.elCompiler.SetSymbols(symbols)
	}
}

// SourceXML records the Excel workbook and sheet from which an artifact was imported.
// It enables the exporter to route the artifact back to the exact workbook and sheet.
type SourceXML struct {
	RelativePath string `xml:"relative_path"`
	FileName     string `xml:"file_name"`
	SheetNumber  int    `xml:"sheet_number"`
}

// DecisionTablesXML represents the root XML element for decision tables.
type DecisionTablesXML struct {
	XMLName xml.Name           `xml:"decision_tables"`
	Tables  []DecisionTableXML `xml:"decision_table"`
}

// DecisionTableXML represents a single decision table in XML format.
type DecisionTableXML struct {
	Source           *SourceXML           `xml:"source,omitempty"`
	TableName        string               `xml:"table_name"`
	XLSFile          string               `xml:"xls_file"`
	AttributeFields  AttributeFieldsXML   `xml:"attribute_fields"`
	Contexts         string               `xml:"contexts"`
	InitialActions   []InitialActionXML   `xml:"initial_actions>initial_action"`
	Conditions       []ConditionXML       `xml:"conditions>condition_details"`
	Actions          []ActionXML          `xml:"actions>action_details"`
	PolicyStatements []PolicyStatementXML `xml:"policy_statements>policy_statement"`
	ELCompiled       bool                 `xml:"el_compiled,attr"` // True if postfix was generated from EL
}

// AttributeFieldsXML holds metadata fields for the table.
type AttributeFieldsXML struct {
	Type        string `xml:"Type"`
	Comments    string `xml:"COMMENTS"`
	TableNumber string `xml:"TABLE_NUMBER"`
	FilePath    string `xml:"FILE_PATH,omitempty"`
}

// InitialActionXML represents an initial action.
type InitialActionXML struct {
	DSL     string `xml:"initial_action_dsl"`
	Postfix string `xml:"action_postfix"`
}

// ConditionXML represents a condition row with its column values.
type ConditionXML struct {
	Number  string           `xml:"condition_number"`
	Comment string           `xml:"condition_comment"`
	DSL     string           `xml:"condition_dsl"`
	Postfix string           `xml:"condition_postfix"`
	Columns []ColumnValueXML `xml:"condition_column"`
}

// ActionXML represents an action row with its column values.
type ActionXML struct {
	Number  string           `xml:"action_number"`
	Comment string           `xml:"action_comment"`
	DSL     string           `xml:"action_dsl"`
	Postfix string           `xml:"action_postfix"`
	Columns []ColumnValueXML `xml:"action_column"`
}

// ColumnValueXML represents a column value in conditions or actions.
type ColumnValueXML struct {
	Number int    `xml:"column_number,attr"`
	Value  string `xml:"column_value,attr"`
}

// PolicyStatementXML represents a policy statement for a column.
type PolicyStatementXML struct {
	Column      string `xml:"column,attr"`
	Description string `xml:"policy_description"`
	Postfix     string `xml:"policy_statement_postfix"`
}

// ImportDecisionTables reads an Excel file and returns decision table XML.
func (i *DTImporter) ImportDecisionTables(filename string) (*DecisionTablesXML, error) {
	return i.importDecisionTablesWithSource(filename, filepath.Base(filename), filepath.Base(filename))
}

// importDecisionTablesWithSource reads an Excel file and returns decision table XML,
// setting xlsFile in XLSFile and relPath in the <source> element.
func (i *DTImporter) importDecisionTablesWithSource(filename, xlsFile, relPath string) (*DecisionTablesXML, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	tables := &DecisionTablesXML{}

	// Process each sheet as a decision table, tracking 1-based sheet index
	sheets := f.GetSheetList()
	for sheetIdx, sheetName := range sheets {
		if i.verbose {
			fmt.Printf("Processing sheet: %s\n", sheetName)
		}

		table, err := i.parseSheet(f, sheetName, xlsFile)
		if err != nil {
			if i.verbose {
				fmt.Printf("  Warning: %v\n", err)
			}
			continue
		}
		if table != nil {
			// Compile EL expressions if compiler is set
			if err := i.compileTableEL(table); err != nil {
				if i.verbose {
					fmt.Printf("  EL compilation warning: %v\n", err)
				}
			}
			table.Source = &SourceXML{
				RelativePath: relPath,
				FileName:     filepath.Base(filename),
				SheetNumber:  sheetIdx + 1, // 1-based
			}
			tables.Tables = append(tables.Tables, *table)
		}
	}

	return tables, nil
}

// ImportDecisionTablesFromDir reads all Excel files in a directory recursively.
// Files are processed in sorted order within each directory level.
// The xls_file metadata is set to the relative path from the base directory.
func (i *DTImporter) ImportDecisionTablesFromDir(dir string) (*DecisionTablesXML, error) {
	tables := &DecisionTablesXML{}

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
	sort.Slice(xlsxFiles, func(i, j int) bool {
		return xlsxFiles[i].relPath < xlsxFiles[j].relPath
	})

	// Process each file
	for _, entry := range xlsxFiles {
		if i.verbose {
			fmt.Printf("Processing file: %s\n", entry.relPath)
		}

		fileTables, err := i.importDecisionTablesWithSource(entry.path, entry.relPath, entry.relPath)
		if err != nil {
			if i.verbose {
				fmt.Printf("  Warning: %v\n", err)
			}
			continue
		}

		tables.Tables = append(tables.Tables, fileTables.Tables...)
	}

	return tables, nil
}

// WriteXML writes the decision tables to an XML file.
func (i *DTImporter) WriteXML(tables *DecisionTablesXML, filename string) error {
	// Open file for writing
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create XML file: %w", err)
	}
	defer f.Close()

	// Write XML header
	f.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")

	// Write opening tag
	f.WriteString("<decision_tables>\n")

	// Write each table
	for _, table := range tables.Tables {
		if err := i.writeTable(f, &table); err != nil {
			return err
		}
	}

	// Write closing tag
	f.WriteString("</decision_tables>\n")

	return nil
}

// writeTable writes a single decision table to the file with proper formatting.
func (i *DTImporter) writeTable(f *os.File, table *DecisionTableXML) error {
	// Table header comment
	tableNum := table.AttributeFields.TableNumber
	f.WriteString(fmt.Sprintf("\n<!-- TABLE %s: %s -->\n", tableNum, table.TableName))

	// Write decision_table opening tag with optional el_compiled attribute
	if table.ELCompiled {
		f.WriteString("<decision_table el_compiled=\"true\">\n")
	} else {
		f.WriteString("<decision_table>\n")
	}

	// Write <source> as first child if present
	if table.Source != nil {
		f.WriteString("<source>\n")
		f.WriteString(fmt.Sprintf("<relative_path>%s</relative_path>\n", xmlEscape(table.Source.RelativePath)))
		f.WriteString(fmt.Sprintf("<file_name>%s</file_name>\n", xmlEscape(table.Source.FileName)))
		f.WriteString(fmt.Sprintf("<sheet_number>%d</sheet_number>\n", table.Source.SheetNumber))
		f.WriteString("</source>\n")
	}

	f.WriteString(fmt.Sprintf("<table_name>%s</table_name>\n", xmlEscape(table.TableName)))
	f.WriteString(fmt.Sprintf("<xls_file>%s</xls_file>\n", xmlEscape(table.XLSFile)))

	// Attribute fields
	f.WriteString("<attribute_fields>\n")
	f.WriteString(fmt.Sprintf("<Type>%s</Type>\n", xmlEscape(table.AttributeFields.Type)))
	f.WriteString(fmt.Sprintf("<COMMENTS>%s</COMMENTS>\n", xmlEscape(table.AttributeFields.Comments)))
	f.WriteString(fmt.Sprintf("<TABLE_NUMBER>%s</TABLE_NUMBER>\n", xmlEscape(table.AttributeFields.TableNumber)))
	if table.AttributeFields.FilePath != "" {
		f.WriteString(fmt.Sprintf("<FILE_PATH>%s</FILE_PATH>\n", xmlEscape(table.AttributeFields.FilePath)))
	}
	f.WriteString("</attribute_fields>\n")

	// Contexts
	f.WriteString(fmt.Sprintf("<contexts>%s</contexts>\n", xmlEscape(table.Contexts)))

	// Initial actions
	f.WriteString("<initial_actions>\n")
	for _, action := range table.InitialActions {
		f.WriteString("<initial_action>\n")
		f.WriteString(fmt.Sprintf("<initial_action_dsl>%s</initial_action_dsl>\n", xmlEscape(action.DSL)))
		f.WriteString(fmt.Sprintf("<action_postfix>\n%s\n</action_postfix>\n", xmlEscape(action.Postfix)))
		f.WriteString("</initial_action>\n")
	}
	f.WriteString("</initial_actions>\n")

	// Conditions
	f.WriteString("<conditions>\n")
	for _, cond := range table.Conditions {
		f.WriteString("<condition_details>\n")
		f.WriteString(fmt.Sprintf("<condition_number>%s</condition_number>\n", xmlEscape(cond.Number)))
		f.WriteString(fmt.Sprintf("<condition_comment>%s</condition_comment>\n", xmlEscape(cond.Comment)))
		f.WriteString(fmt.Sprintf("<condition_dsl>%s</condition_dsl>\n", xmlEscape(cond.DSL)))
		f.WriteString(fmt.Sprintf("<condition_postfix>\n%s\n</condition_postfix>\n", xmlEscape(cond.Postfix)))
		for _, col := range cond.Columns {
			f.WriteString(fmt.Sprintf("<condition_column column_number=\"%d\" column_value=\"%s\"></condition_column>\n",
				col.Number, xmlEscape(col.Value)))
		}
		f.WriteString("</condition_details>\n")
	}
	f.WriteString("</conditions>\n")

	// Actions
	f.WriteString("<actions>\n")
	for _, action := range table.Actions {
		f.WriteString("<action_details>\n")
		f.WriteString(fmt.Sprintf("<action_number>%s</action_number>\n", xmlEscape(action.Number)))
		f.WriteString(fmt.Sprintf("<action_comment>%s</action_comment>\n", xmlEscape(action.Comment)))
		f.WriteString(fmt.Sprintf("<action_dsl>%s</action_dsl>\n", xmlEscape(action.DSL)))
		f.WriteString(fmt.Sprintf("<action_postfix>\n%s\n</action_postfix>\n", xmlEscape(action.Postfix)))
		for _, col := range action.Columns {
			f.WriteString(fmt.Sprintf("<action_column column_number=\"%d\" column_value=\"%s\"></action_column>\n",
				col.Number, xmlEscape(col.Value)))
		}
		f.WriteString("</action_details>\n")
	}
	f.WriteString("</actions>\n")

	// Policy statements
	f.WriteString("<policy_statements>\n")
	for _, policy := range table.PolicyStatements {
		f.WriteString(fmt.Sprintf("<policy_statement column=\"%s\">\n", xmlEscape(policy.Column)))
		f.WriteString(fmt.Sprintf("<policy_description>%s</policy_description>\n", xmlEscape(policy.Description)))
		f.WriteString(fmt.Sprintf("<policy_statement_postfix>%s</policy_statement_postfix>\n", xmlEscape(policy.Postfix)))
		f.WriteString("</policy_statement>\n")
	}
	f.WriteString("</policy_statements>\n")

	f.WriteString("</decision_table>\n")
	return nil
}

// xmlEscape escapes special XML characters.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// parseSheet parses a single Excel sheet into a decision table.
func (i *DTImporter) parseSheet(f *excelize.File, sheetName, xlsFile string) (*DecisionTableXML, error) {
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

	// Try to detect format (dt2excel format vs exporter.go format)
	format := i.detectFormat(rows)

	switch format {
	case "dt2excel":
		return i.parseDT2ExcelFormat(rows, table)
	case "exporter":
		return i.parseExporterFormat(rows, sheetName, table)
	default:
		return nil, fmt.Errorf("unrecognized Excel format")
	}
}

// detectFormat detects which Excel format the sheet uses.
func (i *DTImporter) detectFormat(rows [][]string) string {
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
			if strings.HasPrefix(cell, "conditions:") {
				return "exporter"
			}
			if cell == "conditions" && len(row) > 1 {
				return "dt2excel"
			}
		}
	}

	return "unknown"
}

// parseDT2ExcelFormat parses the format generated by cmd/dt2excel.
func (i *DTImporter) parseDT2ExcelFormat(rows [][]string, table *DecisionTableXML) (*DecisionTableXML, error) {
	var currentSection string
	_ = 0 // maxCol not needed since we process all non-empty cells

	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}

		firstCell := strings.TrimSpace(row[0])
		firstCellLower := strings.ToLower(firstCell)

		switch {
		case firstCellLower == "decision table":
			if len(row) > 1 {
				table.TableName = strings.TrimSpace(row[1])
			}

		case firstCellLower == "table number":
			if len(row) > 1 {
				table.AttributeFields.TableNumber = strings.TrimSpace(row[1])
			}

		case firstCellLower == "type":
			if len(row) > 1 {
				table.AttributeFields.Type = strings.TrimSpace(row[1])
			}

		case firstCellLower == "comments":
			if len(row) > 1 {
				table.AttributeFields.Comments = strings.TrimSpace(row[1])
			}

		case firstCellLower == "file_path":
			if len(row) > 1 {
				table.AttributeFields.FilePath = strings.TrimSpace(row[1])
			}

		case firstCellLower == "initial actions":
			currentSection = "initial_actions"

		case firstCellLower == "conditions":
			currentSection = "conditions"

		case firstCellLower == "actions":
			currentSection = "actions"
			// Keep maxCol from conditions section

		case firstCellLower == "policy statements":
			currentSection = "policy"

		case firstCellLower == "#":
			// Header row, skip
			continue

		default:
			switch currentSection {
			case "initial_actions":
				if len(row) > 1 && firstCell != "" {
					action := InitialActionXML{
						DSL:     firstCell,
						Postfix: strings.TrimSpace(row[1]),
					}
					table.InitialActions = append(table.InitialActions, action)
				}

			case "conditions":
				// Parse condition row: #, DSL, Col1, Col2, ...
				// Only process rows that start with a number
				if len(row) > 1 && firstCell != "" {
					if _, err := strconv.Atoi(firstCell); err == nil {
						cond := ConditionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 1)), // Use comment as DSL
						}
						// Parse column values - columns start at index 2
						for col := 2; col < len(row); col++ {
							val := strings.TrimSpace(row[col])
							if val != "" {
								cond.Columns = append(cond.Columns, ColumnValueXML{
									Number: col - 1, // column 1 is at index 2, so col-1
									Value:  val,
								})
							}
						}
						table.Conditions = append(table.Conditions, cond)
					}
				}

			case "actions":
				// Parse action row: #, DSL, Col1, Col2, ...
				// Only process rows that start with a number
				if len(row) > 1 && firstCell != "" {
					if _, err := strconv.Atoi(firstCell); err == nil {
						action := ActionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 1)), // Use comment as DSL
						}
						// Parse column values - columns start at index 2
						for col := 2; col < len(row); col++ {
							val := strings.TrimSpace(row[col])
							if val != "" {
								action.Columns = append(action.Columns, ColumnValueXML{
									Number: col - 1, // column 1 is at index 2, so col-1
									Value:  val,
								})
							}
						}
						table.Actions = append(table.Actions, action)
					}
				}

			case "policy":
				// Parse policy row: "Column N", Description
				if strings.HasPrefix(firstCellLower, "column ") {
					colStr := strings.TrimPrefix(firstCellLower, "column ")
					policy := PolicyStatementXML{
						Column:      colStr,
						Description: strings.TrimSpace(safeGet(row, 1)),
					}
					table.PolicyStatements = append(table.PolicyStatements, policy)
				}
			}
		}

		_ = rowIdx // Used in conditions section
	}

	if table.TableName == "" {
		return nil, fmt.Errorf("no table name found")
	}

	return table, nil
}

// parseExporterFormat parses the format generated by exporter.go.
func (i *DTImporter) parseExporterFormat(rows [][]string, sheetName string, table *DecisionTableXML) (*DecisionTableXML, error) {
	var currentSection string
	var numCols int

	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}

		firstCell := strings.TrimSpace(row[0])
		firstCellLower := strings.ToLower(firstCell)

		switch {
		case strings.HasPrefix(firstCellLower, "name:"):
			// Name row: "Name: Table_Name"
			name := strings.TrimPrefix(firstCell, "Name: ")
			name = strings.TrimPrefix(name, "name: ")
			// Convert display name back to underscore format
			table.TableName = strings.ReplaceAll(strings.TrimSpace(name), " ", "_")

		case strings.HasPrefix(firstCellLower, "type:"):
			// Type row: "Type: FIRST"
			tableType := strings.TrimPrefix(firstCell, "Type: ")
			tableType = strings.TrimPrefix(tableType, "type: ")
			table.AttributeFields.Type = strings.TrimSpace(tableType)

		case strings.HasPrefix(firstCellLower, "comments:"):
			// Comments field
			comment := strings.TrimPrefix(firstCell, "COMMENTS: ")
			comment = strings.TrimPrefix(comment, "Comments: ")
			comment = strings.TrimPrefix(comment, "comments: ")
			table.AttributeFields.Comments = strings.TrimSpace(comment)

		case strings.HasPrefix(firstCellLower, "table_number:"):
			// Table number field
			num := strings.TrimPrefix(firstCell, "TABLE_NUMBER: ")
			num = strings.TrimPrefix(num, "Table_number: ")
			num = strings.TrimPrefix(num, "table_number: ")
			table.AttributeFields.TableNumber = strings.TrimSpace(num)

		case strings.HasPrefix(firstCellLower, "file_path:"):
			// File path field
			path := strings.TrimPrefix(firstCell, "FILE_PATH: ")
			path = strings.TrimPrefix(path, "File_path: ")
			path = strings.TrimPrefix(path, "file_path: ")
			table.AttributeFields.FilePath = strings.TrimSpace(path)

		case firstCellLower == "contexts:":
			currentSection = "contexts"

		case strings.HasPrefix(firstCellLower, "initial actions"):
			currentSection = "initial_actions_header"

		case strings.HasPrefix(firstCellLower, "conditions"):
			currentSection = "conditions_header"

		case strings.HasPrefix(firstCellLower, "actions"):
			currentSection = "actions_header"

		case strings.HasPrefix(firstCellLower, "policy"):
			currentSection = "policy_header"

		default:
			switch currentSection {
			case "contexts":
				// Context row: number, comment, expression (merged)
				if len(row) > 2 && firstCell != "" {
					// Contexts are typically comma-separated entity names
					if table.Contexts == "" {
						table.Contexts = strings.TrimSpace(row[2])
					} else {
						table.Contexts += "," + strings.TrimSpace(row[2])
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "initial_actions_header":
				// Skip the "Initial Actions" label row, next row is header with column numbers
				currentSection = "initial_actions"

			case "initial_actions":
				// Initial action row: number, comment, expression (merged)
				if len(row) > 2 && firstCell != "" {
					num, err := strconv.Atoi(firstCell)
					if err == nil && num > 0 {
						action := InitialActionXML{
							DSL:     strings.TrimSpace(row[1]),
							Postfix: strings.TrimSpace(row[2]),
						}
						table.InitialActions = append(table.InitialActions, action)
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "conditions_header":
				// Header row with column numbers
				numCols = i.countHeaderColumns(row)
				currentSection = "conditions"

			case "conditions":
				// Condition row: number, comment, expression, col1, col2, ...
				if len(row) > 3 && firstCell != "" {
					num, err := strconv.Atoi(firstCell)
					if err == nil && num > 0 {
						cond := ConditionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 1)),
							Postfix: strings.TrimSpace(safeGet(row, 2)),
						}
						// Parse column values (columns start at index 3)
						for col := 3; col < len(row) && col < numCols+3; col++ {
							val := strings.TrimSpace(row[col])
							if val != "" && val != "-" {
								cond.Columns = append(cond.Columns, ColumnValueXML{
									Number: col - 2, // 1-indexed
									Value:  val,
								})
							}
						}
						table.Conditions = append(table.Conditions, cond)
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "actions_header":
				// Header row with column numbers
				numCols = i.countHeaderColumns(row)
				currentSection = "actions"

			case "actions":
				// Action row: number, comment, expression, col1, col2, ...
				if len(row) > 3 && firstCell != "" {
					num, err := strconv.Atoi(firstCell)
					if err == nil && num > 0 {
						action := ActionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 1)),
							Postfix: strings.TrimSpace(safeGet(row, 2)),
						}
						// Parse column values (columns start at index 3)
						for col := 3; col < len(row) && col < numCols+3; col++ {
							val := strings.TrimSpace(row[col])
							if val != "" && val != "-" {
								action.Columns = append(action.Columns, ColumnValueXML{
									Number: col - 2, // 1-indexed
									Value:  val,
								})
							}
						}
						table.Actions = append(table.Actions, action)
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "policy_header":
				// Header row with column numbers
				currentSection = "policy"

			case "policy":
				// Policy row: empty, "Column Policy", empty, col1_stmt, col2_stmt, ...
				if len(row) > 3 {
					for col := 3; col < len(row); col++ {
						val := strings.TrimSpace(row[col])
						if val != "" {
							policy := PolicyStatementXML{
								Column:      strconv.Itoa(col - 2), // 1-indexed
								Description: val,
								Postfix:     fmt.Sprintf(`"%s"`, val),
							}
							table.PolicyStatements = append(table.PolicyStatements, policy)
						}
					}
				}
				currentSection = ""
			}
		}

		_ = rowIdx // Suppress unused warning
	}

	// Fallback: use sheet name if no table name found
	if table.TableName == "" {
		table.TableName = strings.ReplaceAll(sheetName, " ", "_")
	}

	if table.TableName == "" {
		return nil, fmt.Errorf("no table name found")
	}

	return table, nil
}

// countColumns counts the number of decision columns from a header row.
func (i *DTImporter) countColumns(row []string) int {
	count := 0
	colPattern := regexp.MustCompile(`(?i)^col\s*\d+$`)
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if colPattern.MatchString(cell) {
			count++
		}
	}
	return count
}

// countHeaderColumns counts columns from the exporter format header (1, 2, 3, ...).
func (i *DTImporter) countHeaderColumns(row []string) int {
	count := 0
	for j := 3; j < len(row); j++ {
		cell := strings.TrimSpace(row[j])
		if _, err := strconv.Atoi(cell); err == nil {
			count++
		}
	}
	if count == 0 {
		// Fallback: count non-empty cells after column 3
		for j := 3; j < len(row); j++ {
			cell := strings.TrimSpace(row[j])
			if cell != "" {
				count++
			}
		}
	}
	return count
}

// safeGet safely gets a string from a slice, returning empty string if out of bounds.
func safeGet(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

// compileTableEL compiles EL expressions in a table to postfix.
// This is called after parsing to generate postfix from EL descriptions.
// If no EL compiler is set, this method does nothing.
func (i *DTImporter) compileTableEL(table *DecisionTableXML) error {
	if !i.compileEL || i.elCompiler == nil {
		return nil
	}

	// Set symbols if available
	if i.symbols != nil {
		i.elCompiler.SetSymbols(i.symbols)
	}

	var errors []string

	// Compile initial actions
	for idx := range table.InitialActions {
		action := &table.InitialActions[idx]
		if action.DSL != "" {
			postfix, err := i.elCompiler.CompileAction(action.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("initial action %d: %v", idx+1, err))
				continue
			}
			action.Postfix = postfix
		}
	}

	// Compile conditions
	for idx := range table.Conditions {
		cond := &table.Conditions[idx]
		if cond.DSL != "" {
			postfix, err := i.elCompiler.CompileCondition(cond.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("condition %s: %v", cond.Number, err))
				continue
			}
			cond.Postfix = postfix
		}
	}

	// Compile actions
	for idx := range table.Actions {
		action := &table.Actions[idx]
		if action.DSL != "" {
			postfix, err := i.elCompiler.CompileAction(action.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("action %s: %v", action.Number, err))
				continue
			}
			action.Postfix = postfix
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("EL compilation errors in table %s: %s",
			table.TableName, strings.Join(errors, "; "))
	}

	// Mark the table as EL-compiled
	table.ELCompiled = true

	return nil
}

// MergeTables merges multiple DecisionTablesXML into one.
func MergeTables(tables ...*DecisionTablesXML) *DecisionTablesXML {
	result := &DecisionTablesXML{}
	for _, t := range tables {
		if t != nil {
			result.Tables = append(result.Tables, t.Tables...)
		}
	}
	return result
}

// SortTablesByNumber sorts tables by their TABLE_NUMBER field.
func SortTablesByNumber(tables *DecisionTablesXML) {
	sort.Slice(tables.Tables, func(i, j int) bool {
		numI, errI := strconv.Atoi(tables.Tables[i].AttributeFields.TableNumber)
		numJ, errJ := strconv.Atoi(tables.Tables[j].AttributeFields.TableNumber)
		if errI != nil || errJ != nil {
			return tables.Tables[i].TableName < tables.Tables[j].TableName
		}
		return numI < numJ
	})
}
