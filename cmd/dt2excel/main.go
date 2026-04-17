package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// DecisionTables represents the root XML element
type DecisionTables struct {
	XMLName xml.Name         `xml:"decision_tables"`
	Tables  []DecisionTable `xml:"decision_table"`
}

// DecisionTable represents a single decision table
type DecisionTable struct {
	// NameAttr captures EL format: <decision_table name="...">
	NameAttr         string            `xml:"name,attr"`
	// NumberAttr captures EL format: <decision_table number="...">
	NumberAttr       string            `xml:"number,attr"`
	TableName        string            `xml:"table_name"`
	XLSFile          string            `xml:"xls_file"`
	AttributeFields  AttributeFields   `xml:"attribute_fields"`
	Contexts         string            `xml:"contexts"`
	InitialActions   []InitialAction   `xml:"initial_actions>initial_action"`
	Conditions       []ConditionDetail `xml:"conditions>condition_details"`
	Actions          []ActionDetail    `xml:"actions>action_details"`
	PolicyStatements []PolicyStatement `xml:"policy_statements>policy_statement"`
}

// GetTableName returns the table name from either attribute or element form.
func (t *DecisionTable) GetTableName() string {
	if t.NameAttr != "" {
		return t.NameAttr
	}
	return t.TableName
}

// GetTableNumber returns the table number from either attribute or element form.
func (t *DecisionTable) GetTableNumber() string {
	if t.NumberAttr != "" {
		return t.NumberAttr
	}
	return t.AttributeFields.TableNumber
}

type AttributeFields struct {
	Type        string `xml:"Type"`
	Comments    string `xml:"COMMENTS"`
	TableNumber string `xml:"TABLE_NUMBER"`
}

type InitialAction struct {
	Description string `xml:"action_description"`
	Postfix     string `xml:"action_postfix"`
}

type ConditionDetail struct {
	Number      string          `xml:"condition_number"`
	Comment     string          `xml:"condition_comment"`
	Description string          `xml:"condition_description"`
	Postfix     string          `xml:"condition_postfix"`
	Columns     []ColumnValue   `xml:"condition_column"`
}

type ActionDetail struct {
	Number      string          `xml:"action_number"`
	Comment     string          `xml:"action_comment"`
	Description string          `xml:"action_description"`
	Postfix     string          `xml:"action_postfix"`
	Columns     []ColumnValue   `xml:"action_column"`
}

type ColumnValue struct {
	Number int    `xml:"column_number,attr"`
	Value  string `xml:"column_value,attr"`
}

type PolicyStatement struct {
	Column      string `xml:"column,attr"`
	Description string `xml:"policy_description"`
	Postfix     string `xml:"policy_statement_postfix"`
}

func main() {
	inputFile := flag.String("input", "", "Input XML file path")
	outputDir := flag.String("output", "", "Output directory for Excel files")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: dt2excel -input <xml_file> -output <output_dir>")
		fmt.Println("Example: dt2excel -input TaxReturn_dt.xml -output excel/")
		os.Exit(1)
	}

	if *outputDir == "" {
		*outputDir = "excel"
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Read and parse XML
	xmlFile, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("Error opening XML file: %v\n", err)
		os.Exit(1)
	}
	defer xmlFile.Close()

	xmlData, err := io.ReadAll(xmlFile)
	if err != nil {
		fmt.Printf("Error reading XML file: %v\n", err)
		os.Exit(1)
	}

	var tables DecisionTables
	if err := xml.Unmarshal(xmlData, &tables); err != nil {
		fmt.Printf("Error parsing XML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d decision tables\n", len(tables.Tables))

	// Convert each table to Excel
	for i, table := range tables.Tables {
		tableName := table.GetTableName()
		if err := convertTableToExcel(table, *outputDir, i+1); err != nil {
			fmt.Printf("Error converting table %s: %v\n", tableName, err)
		} else {
			fmt.Printf("✓ Exported: %s\n", tableName)
		}
	}

	fmt.Printf("\nExport complete! Files saved to: %s\n", *outputDir)
}

func convertTableToExcel(table DecisionTable, outputDir string, index int) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"

	// Get table name (supports both attribute and element forms)
	tableName := table.GetTableName()
	tableNumber := table.GetTableNumber()

	// Set sheet name to table name (sanitized)
	safeName := sanitizeSheetName(tableName)
	if safeName != sheetName {
		f.SetSheetName(sheetName, safeName)
		sheetName = safeName
	}

	row := 1

	// Header section
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Decision Table")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), tableName)
	f.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("E%d", row))
	setHeaderStyle(f, sheetName, fmt.Sprintf("A%d", row))
	row++

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Table Number")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), tableNumber)
	row++

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Type")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), table.AttributeFields.Type)
	row++

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Comments")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), table.AttributeFields.Comments)
	f.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("F%d", row))
	row += 2

	// Initial Actions
	if len(table.InitialActions) > 0 {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Initial Actions")
		setHeaderStyle(f, sheetName, fmt.Sprintf("A%d", row))
		row++
		for _, action := range table.InitialActions {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), action.Description)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), action.Postfix)
			f.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("F%d", row))
			row++
		}
		row++
	}

	// Determine number of columns
	maxCol := getMaxColumns(table)

	// Conditions
	if len(table.Conditions) > 0 {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Conditions")
		setHeaderStyle(f, sheetName, fmt.Sprintf("A%d", row))
		row++

		// Header row
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "#")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "Description")
		for col := 1; col <= maxCol; col++ {
			f.SetCellValue(sheetName, getColLetter(col+2)+fmt.Sprintf("%d", row), fmt.Sprintf("Col %d", col))
		}
		row++

		for _, cond := range table.Conditions {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), cond.Number)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), cond.Comment)

			for _, colVal := range cond.Columns {
				if colVal.Number > 0 && colVal.Number <= maxCol {
					f.SetCellValue(sheetName, getColLetter(colVal.Number+2)+fmt.Sprintf("%d", row), colVal.Value)
				}
			}
			row++
		}
		row++
	}

	// Actions
	if len(table.Actions) > 0 {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Actions")
		setHeaderStyle(f, sheetName, fmt.Sprintf("A%d", row))
		row++

		// Header row
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "#")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "Description")
		for col := 1; col <= maxCol; col++ {
			f.SetCellValue(sheetName, getColLetter(col+2)+fmt.Sprintf("%d", row), fmt.Sprintf("Col %d", col))
		}
		row++

		for _, action := range table.Actions {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), action.Number)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), action.Comment)

			for _, colVal := range action.Columns {
				if colVal.Number > 0 && colVal.Number <= maxCol {
					f.SetCellValue(sheetName, getColLetter(colVal.Number+2)+fmt.Sprintf("%d", row), colVal.Value)
				}
			}
			row++
		}
		row++
	}

	// Policy Statements
	if len(table.PolicyStatements) > 0 {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Policy Statements")
		setHeaderStyle(f, sheetName, fmt.Sprintf("A%d", row))
		row++

		for _, policy := range table.PolicyStatements {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Column "+policy.Column)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), policy.Description)
			f.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("F%d", row))
			row++
		}
	}

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 15)
	f.SetColWidth(sheetName, "B", "B", 50)
	for col := 3; col <= maxCol+2; col++ {
		f.SetColWidth(sheetName, getColLetter(col), getColLetter(col), 12)
	}

	// Save file
	filename := filepath.Join(outputDir, fmt.Sprintf("%03d_%s.xlsx", index, sanitizeFilename(tableName)))
	if err := f.SaveAs(filename); err != nil {
		return fmt.Errorf("error saving Excel file: %v", err)
	}

	return nil
}

func getMaxColumns(table DecisionTable) int {
	maxCol := 0
	for _, cond := range table.Conditions {
		for _, col := range cond.Columns {
			if col.Number > maxCol {
				maxCol = col.Number
			}
		}
	}
	for _, action := range table.Actions {
		for _, col := range action.Columns {
			if col.Number > maxCol {
				maxCol = col.Number
			}
		}
	}
	return maxCol
}

func getColLetter(num int) string {
	letter := ""
	for num > 0 {
		num--
		letter = string(rune('A'+num%26)) + letter
		num /= 26
	}
	if letter == "" {
		return "A"
	}
	return letter
}

func setHeaderStyle(f *excelize.File, sheet, cell string) {
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   10,
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"E8E8E8"},
			Pattern: 1,
		},
		Border: thinBorder(),
	})
	f.SetCellStyle(sheet, cell, cell, style)
}

func thinBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "CCCCCC", Style: 1},
		{Type: "top", Color: "CCCCCC", Style: 1},
		{Type: "bottom", Color: "CCCCCC", Style: 1},
		{Type: "right", Color: "CCCCCC", Style: 1},
	}
}

func sanitizeSheetName(name string) string {
	// Excel sheet names can't exceed 31 chars and can't contain: \ / ? * [ ]
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")

	if len(name) > 31 {
		name = name[:31]
	}
	return name
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}
