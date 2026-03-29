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
	"os"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
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

// ExportDecisionTables exports all decision tables to a single Excel file.
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

// ExportDecisionTablesToDir exports decision tables grouped by xls_file to a directory.
// Creates multiple xlsx files and an index.md for navigation.
func (e *Exporter) ExportDecisionTablesToDir(dir string) error {
	// Group tables by xls_file
	groups := make(map[string][]*decisiontable.RDecisionTable)
	tableNames := e.ruleSet.GetDecisionTableNames()

	for _, rname := range tableNames {
		dtObj := e.ruleSet.GetEntityFactory().FindDecisionTable(rname)
		if dtObj == nil {
			continue
		}
		dt, ok := dtObj.(*decisiontable.RDecisionTable)
		if !ok {
			continue
		}

		filePath := dt.GetFilePath()
		if filePath == "" {
			filePath = "Other"
		}
		xlsFile := filePath
		// Normalize to .xlsx extension
		xlsFile = strings.TrimSuffix(xlsFile, ".xls")
		xlsFile = strings.TrimSuffix(xlsFile, ".xlsx")
		xlsFile = xlsFile + ".xlsx"

		groups[xlsFile] = append(groups[xlsFile], dt)
	}

	// Sort tables within each group by name
	for _, tables := range groups {
		sort.Slice(tables, func(i, j int) bool {
			return tables[i].GetName() < tables[j].GetName()
		})
	}

	// Write each group to a separate file
	for xlsFile, tables := range groups {
		if err := e.writeDecisionTableGroup(dir, xlsFile, tables); err != nil {
			return err
		}
	}

	// Write index.md
	return e.writeDecisionTableIndex(dir, groups)
}

func (e *Exporter) writeDecisionTableGroup(dir, xlsFile string, tables []*decisiontable.RDecisionTable) error {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)

	styles, err := e.createStyles(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	for _, dt := range tables {
		if err := e.writeDecisionTable(f, dt, styles); err != nil {
			return fmt.Errorf("failed to write table %s: %w", dt.GetName(), err)
		}
	}

	sheetList := f.GetSheetList()
	if len(sheetList) > 1 {
		f.DeleteSheet(defaultSheet)
	}

	filepath := dir + "/" + xlsFile
	return f.SaveAs(filepath)
}

func (e *Exporter) writeDecisionTableIndex(dir string, groups map[string][]*decisiontable.RDecisionTable) error {
	var sb strings.Builder

	sb.WriteString("# Decision Tables Index\n\n")
	sb.WriteString("This folder contains decision tables organized by functional area.\n\n")

	// Sort group names
	groupNames := make([]string, 0, len(groups))
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| File | Tables | Description |\n")
	sb.WriteString("|------|--------|-------------|\n")

	totalTables := 0
	for _, name := range groupNames {
		tables := groups[name]
		totalTables += len(tables)
		desc := getGroupDescription(name)
		sb.WriteString(fmt.Sprintf("| [%s](%s) | %d | %s |\n", name, name, len(tables), desc))
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** | |\n\n", totalTables))

	// Detailed listing
	sb.WriteString("## Tables by File\n\n")

	for _, name := range groupNames {
		tables := groups[name]
		sb.WriteString(fmt.Sprintf("### %s\n\n", name))
		sb.WriteString("| Table | Description |\n")
		sb.WriteString("|-------|-------------|\n")

		for _, dt := range tables {
			fields := dt.GetFields()
			comment := fields["COMMENTS"]
			if len(comment) > 80 {
				comment = comment[:80] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", dt.GetName(), comment))
		}
		sb.WriteString("\n")
	}

	filepath := dir + "/index.md"
	return writeFile(filepath, sb.String())
}

func getGroupDescription(filename string) string {
	name := strings.TrimSuffix(filename, ".xlsx")
	descriptions := map[string]string{
		"MainFlow":       "Entry point and main tax calculation flow",
		"Income":         "Income processing tables",
		"Deductions":     "Deduction calculations",
		"TaxCalculation": "Tax brackets and liability calculation",
		"Credits":        "Tax credit calculations",
		"SpecialForms":   "Special IRS forms (K-1, foreign, etc.)",
		"Validation":     "Test validation tables",
		"Helpers":        "Helper and utility tables",
		"Other":          "Uncategorized tables",
	}
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return ""
}

func writeFile(filepath, content string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}

// ExportEDD exports the Entity Data Dictionary to an Excel file.
func (e *Exporter) ExportEDD(filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Set up sheet
	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	// Create styles
	eddStyles, err := e.createEDDStyles(f)
	if err != nil {
		return err
	}

	// Set column widths
	f.SetColWidth(sheet, "A", "A", 18)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetColWidth(sheet, "C", "C", 10)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 18)
	f.SetColWidth(sheet, "F", "F", 8)
	f.SetColWidth(sheet, "G", "G", 8)
	f.SetColWidth(sheet, "H", "H", 55)

	// Write headers
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
		f.SetCellStyle(sheet, cell, cell, eddStyles.header)
	}

	// Freeze the header row
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

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

		// Count valid attributes
		attrCount := 0
		for _, attrName := range attrNames {
			if attrName.StringValue() != entityName && attrName.StringValue() != "mapping*key" {
				if ent.GetEntry(attrName) != nil {
					attrCount++
				}
			}
		}

		if attrCount == 0 {
			continue
		}

		// Write entity header row
		f.SetCellValue(sheet, cellName(1, row), entityName)
		f.SetCellValue(sheet, cellName(2, row), fmt.Sprintf("(%d attributes)", attrCount))
		for col := 1; col <= 8; col++ {
			f.SetCellStyle(sheet, cellName(col, row), cellName(col, row), eddStyles.entityHeader)
		}
		row++

		// Write attributes
		attrRow := 0
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

			// Determine row style (alternating)
			rowStyle := eddStyles.rowEven
			if attrRow%2 == 1 {
				rowStyle = eddStyles.rowOdd
			}

			// Entity column is blank (merged visual)
			f.SetCellValue(sheet, cellName(1, row), "")
			f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), rowStyle)

			f.SetCellValue(sheet, cellName(2, row), attrName.StringValue())
			f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), eddStyles.attribute)

			// Type with color coding
			typeStyle := e.getTypeStyle(eddStyles, entry.Type.String())
			f.SetCellValue(sheet, cellName(3, row), entry.Type.String())
			f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), typeStyle)

			f.SetCellValue(sheet, cellName(4, row), entry.SubType)
			f.SetCellStyle(sheet, cellName(4, row), cellName(4, row), rowStyle)

			f.SetCellValue(sheet, cellName(5, row), entry.DefaultTxt)
			f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), rowStyle)

			// Input field styling
			inputStyle := rowStyle
			if entry.Input != "" {
				inputStyle = eddStyles.input
			}
			f.SetCellValue(sheet, cellName(6, row), entry.Input)
			f.SetCellStyle(sheet, cellName(6, row), cellName(6, row), inputStyle)

			f.SetCellValue(sheet, cellName(7, row), access)
			f.SetCellStyle(sheet, cellName(7, row), cellName(7, row), rowStyle)

			f.SetCellValue(sheet, cellName(8, row), entry.Comment)
			f.SetCellStyle(sheet, cellName(8, row), cellName(8, row), eddStyles.comment)

			row++
			attrRow++
		}
	}

	return f.SaveAs(filename)
}

// ExportEDDToDir exports entities grouped by xls_file to a directory.
// Creates multiple xlsx files and an index.md for navigation.
func (e *Exporter) ExportEDDToDir(dir string) error {
	// Group entities by xls_file
	groups := make(map[string][]*entity.REntity)
	entities := e.ruleSet.GetEntityFactory().GetRefEntities()

	for _, ent := range entities {
		filePath := ent.GetFilePath()
		if filePath == "" {
			filePath = "Other"
		}
		xlsFile := filePath
		// Normalize to .xlsx extension
		xlsFile = strings.TrimSuffix(xlsFile, ".xls")
		xlsFile = strings.TrimSuffix(xlsFile, ".xlsx")
		xlsFile = xlsFile + ".xlsx"

		groups[xlsFile] = append(groups[xlsFile], ent)
	}

	// Sort entities within each group by name
	for _, ents := range groups {
		sort.Slice(ents, func(i, j int) bool {
			return ents[i].GetName().StringValue() < ents[j].GetName().StringValue()
		})
	}

	// Write each group to a separate file
	for xlsFile, ents := range groups {
		if err := e.writeEDDGroup(dir, xlsFile, ents); err != nil {
			return err
		}
	}

	// Write index.md
	return e.writeEDDIndex(dir, groups)
}

func (e *Exporter) writeEDDGroup(dir, xlsFile string, entities []*entity.REntity) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	eddStyles, err := e.createEDDStyles(f)
	if err != nil {
		return err
	}

	// Set column widths
	f.SetColWidth(sheet, "A", "A", 18)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetColWidth(sheet, "C", "C", 10)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 18)
	f.SetColWidth(sheet, "F", "F", 8)
	f.SetColWidth(sheet, "G", "G", 8)
	f.SetColWidth(sheet, "H", "H", 55)

	// Write headers
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
		f.SetCellStyle(sheet, cell, cell, eddStyles.header)
	}

	// Freeze header row
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	row := 2
	for _, ent := range entities {
		entityName := ent.GetName().StringValue()
		attrNames := ent.GetAttributeNames()

		sort.Slice(attrNames, func(i, j int) bool {
			return attrNames[i].StringValue() < attrNames[j].StringValue()
		})

		// Count valid attributes
		attrCount := 0
		for _, attrName := range attrNames {
			if attrName.StringValue() != entityName && attrName.StringValue() != "mapping*key" {
				if ent.GetEntry(attrName) != nil {
					attrCount++
				}
			}
		}

		if attrCount == 0 {
			continue
		}

		// Write entity header row
		f.SetCellValue(sheet, cellName(1, row), entityName)
		f.SetCellValue(sheet, cellName(2, row), fmt.Sprintf("(%d attributes)", attrCount))
		for col := 1; col <= 8; col++ {
			f.SetCellStyle(sheet, cellName(col, row), cellName(col, row), eddStyles.entityHeader)
		}
		row++

		// Write attributes
		attrRow := 0
		for _, attrName := range attrNames {
			if attrName.StringValue() == entityName || attrName.StringValue() == "mapping*key" {
				continue
			}

			entry := ent.GetEntry(attrName)
			if entry == nil {
				continue
			}

			access := ""
			if entry.Readable {
				access += "r"
			}
			if entry.Writable {
				access += "w"
			}

			rowStyle := eddStyles.rowEven
			if attrRow%2 == 1 {
				rowStyle = eddStyles.rowOdd
			}

			f.SetCellValue(sheet, cellName(1, row), "")
			f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), rowStyle)

			f.SetCellValue(sheet, cellName(2, row), attrName.StringValue())
			f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), eddStyles.attribute)

			typeStyle := e.getTypeStyle(eddStyles, entry.Type.String())
			f.SetCellValue(sheet, cellName(3, row), entry.Type.String())
			f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), typeStyle)

			f.SetCellValue(sheet, cellName(4, row), entry.SubType)
			f.SetCellStyle(sheet, cellName(4, row), cellName(4, row), rowStyle)

			f.SetCellValue(sheet, cellName(5, row), entry.DefaultTxt)
			f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), rowStyle)

			inputStyle := rowStyle
			if entry.Input != "" {
				inputStyle = eddStyles.input
			}
			f.SetCellValue(sheet, cellName(6, row), entry.Input)
			f.SetCellStyle(sheet, cellName(6, row), cellName(6, row), inputStyle)

			f.SetCellValue(sheet, cellName(7, row), access)
			f.SetCellStyle(sheet, cellName(7, row), cellName(7, row), rowStyle)

			f.SetCellValue(sheet, cellName(8, row), entry.Comment)
			f.SetCellStyle(sheet, cellName(8, row), cellName(8, row), eddStyles.comment)

			row++
			attrRow++
		}
	}

	filepath := dir + "/" + xlsFile
	return f.SaveAs(filepath)
}

func (e *Exporter) writeEDDIndex(dir string, groups map[string][]*entity.REntity) error {
	var sb strings.Builder

	sb.WriteString("# Entity Data Dictionary Index\n\n")
	sb.WriteString("This folder contains entity definitions organized by functional area.\n\n")

	// Sort group names
	groupNames := make([]string, 0, len(groups))
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| File | Entities | Attributes |\n")
	sb.WriteString("|------|----------|------------|\n")

	totalEntities := 0
	totalAttrs := 0
	for _, name := range groupNames {
		entities := groups[name]
		entCount := len(entities)
		attrCount := 0
		for _, ent := range entities {
			attrCount += len(ent.GetAttributeNames()) - 2 // exclude self-ref and mapping key
		}
		totalEntities += entCount
		totalAttrs += attrCount
		sb.WriteString(fmt.Sprintf("| [%s](%s) | %d | %d |\n", name, name, entCount, attrCount))
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** | **%d** |\n\n", totalEntities, totalAttrs))

	// Detailed listing
	sb.WriteString("## Entities by File\n\n")

	for _, name := range groupNames {
		entities := groups[name]
		sb.WriteString(fmt.Sprintf("### %s\n\n", name))
		sb.WriteString("| Entity | Attributes | Description |\n")
		sb.WriteString("|--------|------------|-------------|\n")

		for _, ent := range entities {
			attrCount := len(ent.GetAttributeNames()) - 2
			comment := ent.GetComment()
			if len(comment) > 60 {
				comment = comment[:60] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %s |\n", ent.GetName().StringValue(), attrCount, comment))
		}
		sb.WriteString("\n")
	}

	filepath := dir + "/index.md"
	return writeFile(filepath, sb.String())
}

// eddStyles holds styles for EDD export
type eddStyles struct {
	header       int
	entityHeader int
	rowEven      int
	rowOdd       int
	attribute    int
	typeString   int
	typeDouble   int
	typeInteger  int
	typeBoolean  int
	typeDate     int
	typeArray    int
	typeDefault  int
	input        int
	comment      int
}

func (e *Exporter) createEDDStyles(f *excelize.File) (*eddStyles, error) {
	s := &eddStyles{}
	var err error

	thinBorder := []excelize.Border{
		{Type: "left", Color: "CCCCCC", Style: 1},
		{Type: "top", Color: "CCCCCC", Style: 1},
		{Type: "bottom", Color: "CCCCCC", Style: 1},
		{Type: "right", Color: "CCCCCC", Style: 1},
	}

	// Header style - dark blue
	s.header, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F4E79"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Entity header style - medium blue
	s.entityHeader, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "1F4E79"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"BDD7EE"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Even row style - white
	s.rowEven, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Odd row style - light gray
	s.rowOdd, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F2F2F2"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Attribute name style - bold
	s.attribute, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Type styles with color coding
	s.typeString, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "0070C0"}, // Blue
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	s.typeDouble, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "00B050"}, // Green
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	s.typeInteger, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "00B050"}, // Green
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	s.typeBoolean, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "ED7D31"}, // Orange
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	s.typeDate, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "7030A0"}, // Purple
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	s.typeArray, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "C00000", Italic: true}, // Red italic
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	s.typeDefault, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Input field style - light yellow
	s.input, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFFC0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	// Comment style - wrapped text
	s.comment, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9, Color: "666666"},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    thinBorder,
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (e *Exporter) getTypeStyle(s *eddStyles, typeName string) int {
	switch strings.ToLower(typeName) {
	case "string":
		return s.typeString
	case "double":
		return s.typeDouble
	case "integer":
		return s.typeInteger
	case "boolean":
		return s.typeBoolean
	case "date":
		return s.typeDate
	case "array":
		return s.typeArray
	default:
		return s.typeDefault
	}
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
