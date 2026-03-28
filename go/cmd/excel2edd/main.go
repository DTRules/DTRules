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

// excel2edd converts Excel EDD files to DTRules EDD XML format.
//
// Usage:
//
//	excel2edd -input <excel_file.xlsx> -output <output_edd.xml>
//	excel2edd -dir <excel_directory> -output <output_edd.xml>
//
// The Excel file should have a sheet named "EDD" with the format:
// - Row 1: Headers (Entity, Attribute, Type, SubType, Default, Input, Access, Description)
// - Entity header rows: entity name in col A, "(N attributes)" in col B
// - Attribute rows: blank col A, attribute details in cols B-H
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/go/pkg/dtrules/excel"
)

func main() {
	inputFile := flag.String("input", "", "Input Excel file path (.xlsx)")
	inputDir := flag.String("dir", "", "Input directory containing Excel files")
	outputFile := flag.String("output", "", "Output XML file path")
	flag.Parse()

	// Validate inputs
	if *inputFile == "" && *inputDir == "" {
		fmt.Println("Usage: excel2edd -input <excel_file.xlsx> -output <output_edd.xml>")
		fmt.Println("   or: excel2edd -dir <excel_directory> -output <output_edd.xml>")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -input string")
		fmt.Println("        Input Excel file path (.xlsx)")
		fmt.Println("  -dir string")
		fmt.Println("        Input directory containing Excel files")
		fmt.Println("  -output string")
		fmt.Println("        Output XML file path (default: derived from input)")
		fmt.Println()
		fmt.Println("Excel Format:")
		fmt.Println("  Sheet named 'EDD' with columns:")
		fmt.Println("    A: Entity (entity name on header rows)")
		fmt.Println("    B: Attribute (attribute name)")
		fmt.Println("    C: Type (string, integer, double, boolean, date, array, entity)")
		fmt.Println("    D: SubType (for array/entity types)")
		fmt.Println("    E: Default (default value)")
		fmt.Println("    F: Input (input mapping name)")
		fmt.Println("    G: Access (r, w, rw)")
		fmt.Println("    H: Description (comment)")
		os.Exit(1)
	}

	if *inputFile != "" && *inputDir != "" {
		fmt.Println("Error: cannot specify both -input and -dir")
		os.Exit(1)
	}

	// Determine output file
	if *outputFile == "" {
		if *inputFile != "" {
			*outputFile = strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile)) + ".xml"
		} else {
			*outputFile = filepath.Join(*inputDir, "edd.xml")
		}
	}

	importer := excel.NewEDDImporter()
	var edd *excel.EDDXML
	var err error

	if *inputFile != "" {
		// Import single file
		fmt.Printf("Reading: %s\n", *inputFile)
		edd, err = importer.ImportEDD(*inputFile)
		if err != nil {
			fmt.Printf("Error importing Excel file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Import directory
		fmt.Printf("Reading directory: %s\n", *inputDir)
		edd, err = importer.ImportEDDFromDir(*inputDir)
		if err != nil {
			fmt.Printf("Error importing Excel directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Report what was found
	totalFields := 0
	for _, ent := range edd.Entities {
		totalFields += len(ent.Fields)
	}
	fmt.Printf("Found %d entities with %d total fields\n", len(edd.Entities), totalFields)

	// Write output
	if err := importer.WriteXML(edd, *outputFile); err != nil {
		fmt.Printf("Error writing XML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("EDD XML written to: %s\n", *outputFile)
}
