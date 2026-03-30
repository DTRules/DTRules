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

// EntityDataDictionary represents the root XML element
type EntityDataDictionary struct {
	XMLName  xml.Name `xml:"entity_data_dictionary"`
	Version  string   `xml:"version,attr"`
	Entities []Entity `xml:"entity"`
}

// Entity represents a single entity definition
type Entity struct {
	Name    string  `xml:"name,attr"`
	Access  string  `xml:"access,attr"`
	Comment string  `xml:"comment,attr"`
	Fields  []Field `xml:"field"`
}

// Field represents a field within an entity
type Field struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	Subtype      string `xml:"subtype,attr"`
	Access       string `xml:"access,attr"`
	Input        string `xml:"input,attr"`
	DefaultValue string `xml:"default_value,attr"`
	Comment      string `xml:"comment,attr"`
}

func main() {
	inputFile := flag.String("input", "", "Input EDD XML file path")
	outputFile := flag.String("output", "", "Output Excel file path")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: edd2excel -input <edd.xml> -output <output.xlsx>")
		fmt.Println("Example: edd2excel -input TaxReturn_edd.xml -output TaxReturn_edd.xlsx")
		os.Exit(1)
	}

	if *outputFile == "" {
		*outputFile = strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile)) + ".xlsx"
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

	var edd EntityDataDictionary
	if err := xml.Unmarshal(xmlData, &edd); err != nil {
		fmt.Printf("Error parsing XML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d entities\n", len(edd.Entities))

	// Create Excel file
	if err := convertEDDToExcel(edd, *outputFile); err != nil {
		fmt.Printf("Error converting to Excel: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ EDD exported to: %s\n", *outputFile)
}

func convertEDDToExcel(edd EntityDataDictionary, outputFile string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Create a sheet for each entity
	for i, entity := range edd.Entities {
		sheetName := sanitizeSheetName(entity.Name)

		// Create sheet (first one already exists as Sheet1)
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			_, err := f.NewSheet(sheetName)
			if err != nil {
				return fmt.Errorf("error creating sheet %s: %v", sheetName, err)
			}
		}

		// Write entity header
		row := 1
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Entity")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entity.Name)
		setHeaderStyle(f, sheetName, fmt.Sprintf("A%d:F%d", row, row))
		row++

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Access")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entity.Access)
		row++

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Comment")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entity.Comment)
		f.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("F%d", row))
		row += 2

		// Field headers
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Field Name")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "Type")
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), "Subtype")
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), "Access")
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), "Input")
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), "Default")
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), "Comment")
		setHeaderStyle(f, sheetName, fmt.Sprintf("A%d:G%d", row, row))
		row++

		// Fields
		for _, field := range entity.Fields {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), field.Name)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), field.Type)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), field.Subtype)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), field.Access)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), field.Input)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), field.DefaultValue)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), field.Comment)
			row++
		}

		// Set column widths
		f.SetColWidth(sheetName, "A", "A", 30)
		f.SetColWidth(sheetName, "B", "B", 15)
		f.SetColWidth(sheetName, "C", "C", 15)
		f.SetColWidth(sheetName, "D", "D", 10)
		f.SetColWidth(sheetName, "E", "E", 10)
		f.SetColWidth(sheetName, "F", "F", 15)
		f.SetColWidth(sheetName, "G", "G", 50)

		// Freeze header row
		f.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			XSplit:      0,
			YSplit:      5,
			TopLeftCell: "A6",
			ActivePane:  "bottomLeft",
		})
	}

	// Save file
	if err := f.SaveAs(outputFile); err != nil {
		return fmt.Errorf("error saving Excel file: %v", err)
	}

	return nil
}

func setHeaderStyle(f *excelize.File, sheet, cellRange string) {
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  11,
			Color: "FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
	})
	f.SetCellStyle(sheet, cellRange, cellRange, style)
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
