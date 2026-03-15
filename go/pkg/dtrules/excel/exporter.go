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
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

const (
	maxCol          = 16
	defaultColWidth = 12.0
	narrowColWidth  = 5.0  // Decision columns (Y/N/X values)
	wideColWidth    = 40.0
)

// Exporter exports decision tables and EDD to Excel format.
type Exporter struct {
	ruleSet *session.RuleSet
}

// NewExporter creates a new Excel exporter.
func NewExporter(rs *session.RuleSet) *Exporter {
	return &Exporter{ruleSet: rs}
}

// ExportDecisionTables exports all decision tables to an Excel file.
func (e *Exporter) ExportDecisionTables(filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Delete the default sheet
	defaultSheet := f.GetSheetName(0)

	// Get all decision table names and sort them
	tableNames := e.ruleSet.GetDecisionTableNames()
	sortedNames := make([]string, len(tableNames))
	for i, name := range tableNames {
		sortedNames[i] = name.StringValue()
	}
	sort.Strings(sortedNames)

	// Create styles
	styles, err := e.createStyles(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	// Export each decision table
	for _, name := range sortedNames {
		rname := dtrules.GetRName(name)
		dtObj := e.ruleSet.GetEntityFactory().FindDecisionTable(rname)
		if dtObj == nil {
			continue
		}
		dt, ok := dtObj.(*decisiontable.RDecisionTable)
		if !ok {
			continue
		}

		if err := e.writeDecisionTable(f, dt, styles); err != nil {
			return fmt.Errorf("failed to write table %s: %w", name, err)
		}
	}

	// Delete the default sheet after we've created others
	sheetList := f.GetSheetList()
	if len(sheetList) > 1 {
		f.DeleteSheet(defaultSheet)
	}

	return f.SaveAs(filename)
}

// ExportEDD exports the Entity Data Dictionary to an Excel file.
func (e *Exporter) ExportEDD(filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Set up sheet
	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	// Create header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "1F4E79"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9E2F3"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		return err
	}

	// Set column widths
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 10)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetColWidth(sheet, "E", "E", 20)
	f.SetColWidth(sheet, "F", "F", 10)
	f.SetColWidth(sheet, "G", "G", 10)
	f.SetColWidth(sheet, "H", "H", 50)

	// Write headers
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default Value", "Input", "Access", "comment"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Get all entities and sort them
	entities := e.ruleSet.GetEntityFactory().GetRefEntities()
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].GetName().StringValue() < entities[j].GetName().StringValue()
	})

	row := 2
	for _, ent := range entities {
		entityName := ent.GetName().StringValue()

		// Get and sort attributes
		attrNames := ent.GetAttributeNames()
		sort.Slice(attrNames, func(i, j int) bool {
			return attrNames[i].StringValue() < attrNames[j].StringValue()
		})

		for _, attrName := range attrNames {
			// Skip self-reference and mapping key
			if attrName.StringValue() == entityName || attrName.StringValue() == "mapping*key" {
				continue
			}

			entry := ent.GetEntry(attrName)
			if entry == nil {
				continue
			}

			// Build access string
			access := ""
			if entry.Readable {
				access += "r"
			}
			if entry.Writable {
				access += "w"
			}

			f.SetCellValue(sheet, cellName(1, row), entityName)
			f.SetCellValue(sheet, cellName(2, row), attrName.StringValue())
			f.SetCellValue(sheet, cellName(3, row), entry.Type.String())
			f.SetCellValue(sheet, cellName(4, row), entry.SubType)
			f.SetCellValue(sheet, cellName(5, row), entry.DefaultTxt)
			f.SetCellValue(sheet, cellName(6, row), entry.Input)
			f.SetCellValue(sheet, cellName(7, row), access)
			f.SetCellValue(sheet, cellName(8, row), entry.Comment)
			row++
		}
		// Blank row between entities
		row++
	}

	return f.SaveAs(filename)
}

// styles holds the cell styles for Excel export
type styles struct {
	title     int
	header    int
	numHeader int
	field     int
	comment   int
	formal    int
	table     int
	typeStyle int
	number    int
	policy    int
}

func (e *Exporter) createStyles(f *excelize.File) (*styles, error) {
	s := &styles{}
	var err error

	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
	}

	s.title, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 12, Color: "1F4E79"},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.header, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C0C0C0"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.numHeader, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C0C0C0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.field, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.comment, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.formal, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.table, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.typeStyle, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFF99"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.number, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}

	s.policy, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E2EFDA"}}, // Light green
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Border:    border,
		Font:      &excelize.Font{Size: 9, Italic: true},
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (e *Exporter) writeDecisionTable(f *excelize.File, dt *decisiontable.RDecisionTable, styles *styles) error {
	// Create sheet with sanitized name
	name := dt.GetName()
	sheetName := sanitizeSheetName(name)

	// Check for duplicate sheet names
	for i := 1; ; i++ {
		idx, _ := f.GetSheetIndex(sheetName)
		if idx == -1 {
			break
		}
		sheetName = fmt.Sprintf("%s_%d", sanitizeSheetName(name)[:min(28, len(name))], i)
	}

	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	_ = idx

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 4)
	f.SetColWidth(sheetName, "B", "B", wideColWidth)
	f.SetColWidth(sheetName, "C", "C", wideColWidth)

	numCols := dt.GetMaxCol()
	if numCols < maxCol {
		numCols = maxCol
	}

	for i := 0; i < numCols; i++ {
		col, _ := excelize.ColumnNumberToName(4 + i)
		f.SetColWidth(sheetName, col, col, narrowColWidth)
	}

	row := 1

	// Write Name
	row = e.writeName(f, sheetName, dt, row, numCols, styles)

	// Write Type
	row = e.writeType(f, sheetName, dt, row, numCols, styles)

	// Write Fields
	row = e.writeFields(f, sheetName, dt, row, numCols, styles)

	// Blank row
	row++

	// Write Contexts
	row = e.writeContexts(f, sheetName, dt, row, numCols, styles)

	// Write Initial Actions
	row = e.writeInitialActions(f, sheetName, dt, row, numCols, styles)

	// Write Conditions
	row = e.writeConditions(f, sheetName, dt, row, numCols, styles)

	// Write Actions
	row = e.writeActions(f, sheetName, dt, row, numCols, styles)

	// Write Policy Statements
	row = e.writePolicyStatements(f, sheetName, dt, row, numCols, styles)

	return nil
}

func (e *Exporter) writeName(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	name := dt.GetName()
	displayName := strings.ReplaceAll(name, "_", " ")

	// Merge cells for name
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "Name: "+displayName)
	f.SetCellStyle(sheet, startCell, endCell, styles.title)

	return row + 1
}

func (e *Exporter) writeType(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "Type: "+dt.GetTableType().String())
	f.SetCellStyle(sheet, startCell, endCell, styles.typeStyle)

	return row + 1
}

func (e *Exporter) writeFields(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	fields := dt.GetFields()

	// Sort field names
	names := make([]string, 0, len(fields))
	for name := range fields {
		if strings.ToUpper(name) != "TYPE" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		value := fields[name]
		startCell := cellName(1, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		f.SetCellValue(sheet, startCell, name+": "+value)
		f.SetCellStyle(sheet, startCell, endCell, styles.field)
		row++
	}

	return row
}

func (e *Exporter) writeContexts(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	// Header
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "Contexts:")
	f.SetCellStyle(sheet, startCell, endCell, styles.header)
	row++

	contexts := dt.GetContexts()
	comments := dt.GetContextsComment()

	for i, ctx := range contexts {
		// Number
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styles.number)

		// Comment
		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		f.SetCellValue(sheet, cellName(2, row), comment)
		f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), styles.comment)

		// Expression (merged)
		startCell := cellName(3, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		f.SetCellValue(sheet, startCell, ctx)
		f.SetCellStyle(sheet, startCell, endCell, styles.formal)
		row++
	}

	// Blank row
	row++
	return row
}

func (e *Exporter) writeInitialActions(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	// Header row
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	f.SetCellValue(sheet, cellName(1, row), "Initial Actions:")
	f.SetCellStyle(sheet, cellName(1, row), cellName(2, row), styles.header)

	f.MergeCell(sheet, cellName(3, row), cellName(3+numCols, row))
	f.SetCellValue(sheet, cellName(3, row), "Initial Actions")
	f.SetCellStyle(sheet, cellName(3, row), cellName(3+numCols, row), styles.header)
	row++

	actions := dt.GetInitialActions()
	comments := dt.GetInitialActionsComment()

	for i, action := range actions {
		// Number
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styles.number)

		// Comment
		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		f.SetCellValue(sheet, cellName(2, row), comment)
		f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), styles.comment)

		// Expression (merged)
		startCell := cellName(3, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		f.SetCellValue(sheet, startCell, action)
		f.SetCellStyle(sheet, startCell, endCell, styles.formal)
		row++
	}

	// Blank row
	row++
	return row
}

func (e *Exporter) writeConditions(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	// Header row
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	f.SetCellValue(sheet, cellName(1, row), "Conditions:")
	f.SetCellStyle(sheet, cellName(1, row), cellName(2, row), styles.header)

	f.SetCellValue(sheet, cellName(3, row), "Conditions")
	f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), styles.header)

	// Column numbers
	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styles.numHeader)
	}
	row++

	conditions := dt.GetConditions()
	comments := dt.GetConditionsComment()
	condTable := dt.GetConditionTable()

	for i, cond := range conditions {
		// Number
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styles.number)

		// Comment
		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		f.SetCellValue(sheet, cellName(2, row), comment)
		f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), styles.comment)

		// Expression
		f.SetCellValue(sheet, cellName(3, row), cond)
		f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), styles.formal)

		// Table values
		if i < len(condTable) {
			for j := 0; j < numCols; j++ {
				val := ""
				if j < len(condTable[i]) && condTable[i][j] != "-" {
					val = condTable[i][j]
				}
				cell := cellName(4+j, row)
				f.SetCellValue(sheet, cell, val)
				f.SetCellStyle(sheet, cell, cell, styles.table)
			}
		}
		row++
	}

	// Blank row
	row++
	return row
}

func (e *Exporter) writeActions(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	// Header row
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	f.SetCellValue(sheet, cellName(1, row), "Actions:")
	f.SetCellStyle(sheet, cellName(1, row), cellName(2, row), styles.header)

	f.SetCellValue(sheet, cellName(3, row), "Actions")
	f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), styles.header)

	// Column numbers
	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styles.numHeader)
	}
	row++

	actions := dt.GetActions()
	comments := dt.GetActionsComment()
	actionTable := dt.GetActionTable()

	for i, action := range actions {
		// Number
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styles.number)

		// Comment
		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		f.SetCellValue(sheet, cellName(2, row), comment)
		f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), styles.comment)

		// Expression
		f.SetCellValue(sheet, cellName(3, row), action)
		f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), styles.formal)

		// Table values
		if i < len(actionTable) {
			for j := 0; j < numCols; j++ {
				val := ""
				if j < len(actionTable[i]) && actionTable[i][j] != "" && actionTable[i][j] != "-" {
					val = actionTable[i][j]
				}
				cell := cellName(4+j, row)
				f.SetCellValue(sheet, cell, val)
				f.SetCellStyle(sheet, cell, cell, styles.table)
			}
		}
		row++
	}

	// Blank row
	row++
	return row
}

func (e *Exporter) writePolicyStatements(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styles *styles) int {
	policyStatements := dt.GetPolicyStatements()

	// Check if there are any policy statements
	hasStatements := false
	for i := 1; i < len(policyStatements); i++ {
		if policyStatements[i] != "" {
			hasStatements = true
			break
		}
	}
	if !hasStatements {
		return row
	}

	// Header row
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	f.SetCellValue(sheet, cellName(1, row), "Policy:")
	f.SetCellStyle(sheet, cellName(1, row), cellName(2, row), styles.header)

	f.SetCellValue(sheet, cellName(3, row), "Policy Statements")
	f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), styles.header)

	// Column numbers
	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styles.numHeader)
	}
	row++

	// Policy statement row - each statement in its column
	f.SetCellValue(sheet, cellName(1, row), "")
	f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styles.number)

	f.SetCellValue(sheet, cellName(2, row), "Column Policy")
	f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), styles.comment)

	f.SetCellValue(sheet, cellName(3, row), "")
	f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), styles.formal)

	// Put each policy statement in its respective column
	for i := 0; i < numCols; i++ {
		stmt := ""
		colIdx := i + 1 // Policy statements are 1-indexed by column
		if colIdx < len(policyStatements) {
			stmt = policyStatements[colIdx]
		}
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, stmt)
		f.SetCellStyle(sheet, cell, cell, styles.policy)
	}

	// Set row height to accommodate wrapped text
	f.SetRowHeight(sheet, row, 45)
	row++

	return row
}

// Helper functions

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func sanitizeSheetName(name string) string {
	// Excel sheet names have restrictions
	name = strings.ReplaceAll(name, "_", " ")
	// Max 31 characters
	if len(name) > 31 {
		name = name[:31]
	}
	// Remove invalid characters
	for _, c := range []string{":", "\\", "/", "?", "*", "[", "]"} {
		name = strings.ReplaceAll(name, c, "")
	}
	return name
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
