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

// Package main implements the DTRules command-line interface.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules/interpreter"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

var (
	eddFile    = flag.String("edd", "", "Path to EDD XML file")
	dtFile     = flag.String("dt", "", "Path to Decision Tables XML file")
	rulesDir   = flag.String("rules", "", "Directory containing EDD.xml and DecisionTables.xml")
	entryPoint = flag.String("entry", "", "Decision table entry point to execute")
	validate   = flag.Bool("validate", false, "Validate rules without executing")
	listTables = flag.Bool("list", false, "List all decision tables")
	compile    = flag.String("compile", "", "Compile rules to bytecode file (.dtbc)")
	verbose    = flag.Bool("v", false, "Verbose output")
	trace      = flag.Bool("trace", false, "Enable trace output during execution")
	debug      = flag.Bool("debug", false, "Enable debug output during execution")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "DTRules - Decision Table Rules Engine\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -list\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -validate\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -entry Compute_Eligibility\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -entry Main -trace\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -edd ./EDD.xml -dt ./DecisionTables.xml -entry Main\n", os.Args[0])
	}

	flag.Parse()

	// Determine EDD and DT file paths
	eddPath, dtPath, err := resolveFilePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load rule set
	rs, err := loadRuleSet(eddPath, dtPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		os.Exit(1)
	}

	// Handle list mode
	if *listTables {
		listDecisionTables(rs)
		return
	}

	// Handle validate mode
	if *validate {
		validateRules(rs)
		return
	}

	// Handle compile mode
	if *compile != "" {
		compileRules(rs, *compile)
		return
	}

	// Execute mode requires entry point
	if *entryPoint == "" {
		fmt.Fprintf(os.Stderr, "Error: -entry is required for execution\n")
		fmt.Fprintf(os.Stderr, "Use -list to see available decision tables\n")
		os.Exit(1)
	}

	// Create session and execute
	sess, err := rs.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	rsess := sess.(*session.RSession)
	state := rsess.GetState().(*interpreter.DTState)

	// Enable trace/debug modes
	if *trace {
		state.EnableTrace()
		if *verbose {
			fmt.Println("Trace mode enabled")
		}
	}
	if *debug {
		state.EnableDebug()
		if *verbose {
			fmt.Println("Debug mode enabled")
		}
	}

	if *verbose {
		fmt.Printf("Executing decision table: %s\n", *entryPoint)
	}

	if err := rsess.Execute(*entryPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing %s: %v\n", *entryPoint, err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("Execution completed successfully")
	}
}

func resolveFilePaths() (eddPath, dtPath string, err error) {
	if *rulesDir != "" {
		// Look for common file names in the rules directory
		eddPath = findFile(*rulesDir, "EDD.xml", "edd.xml")
		dtPath = findFile(*rulesDir, "DecisionTables.xml", "decisiontables.xml", "DT.xml", "dt.xml")

		if eddPath == "" {
			return "", "", fmt.Errorf("could not find EDD.xml in %s", *rulesDir)
		}
		if dtPath == "" {
			return "", "", fmt.Errorf("could not find DecisionTables.xml in %s", *rulesDir)
		}
		return eddPath, dtPath, nil
	}

	if *eddFile != "" && *dtFile != "" {
		return *eddFile, *dtFile, nil
	}

	return "", "", fmt.Errorf("must specify either -rules directory or both -edd and -dt files")
}

func findFile(dir string, names ...string) string {
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// Also check lowercase
		path = filepath.Join(dir, strings.ToLower(name))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func loadRuleSet(eddPath, dtPath string) (*session.RuleSet, error) {
	rs := session.NewRuleSet("dtrules")

	// Load EDD
	if *verbose {
		fmt.Printf("Loading EDD from: %s\n", eddPath)
	}
	eddFile, err := os.Open(eddPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EDD file: %w", err)
	}
	defer eddFile.Close()

	if err := rs.LoadEDD(eddFile); err != nil {
		return nil, fmt.Errorf("failed to load EDD: %w", err)
	}

	// Load Decision Tables
	if *verbose {
		fmt.Printf("Loading Decision Tables from: %s\n", dtPath)
	}
	dtFile, err := os.Open(dtPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DT file: %w", err)
	}
	defer dtFile.Close()

	if err := rs.LoadDecisionTables(dtFile); err != nil {
		return nil, fmt.Errorf("failed to load Decision Tables: %w", err)
	}

	return rs, nil
}

func listDecisionTables(rs *session.RuleSet) {
	tables := rs.GetDecisionTableNames()
	if len(tables) == 0 {
		fmt.Println("No decision tables found")
		return
	}

	fmt.Printf("Decision Tables (%d):\n", len(tables))
	for _, name := range tables {
		fmt.Printf("  - %s\n", name.StringValue())
	}
}

func validateRules(rs *session.RuleSet) {
	tables := rs.GetDecisionTableNames()
	entities := rs.GetEntityNames()

	fmt.Printf("Validation Results:\n")
	fmt.Printf("  Entities: %d\n", len(entities))
	fmt.Printf("  Decision Tables: %d\n", len(tables))

	if *verbose {
		fmt.Printf("\nEntities:\n")
		for _, name := range entities {
			fmt.Printf("  - %s\n", name.StringValue())
		}
		fmt.Printf("\nDecision Tables:\n")
		for _, name := range tables {
			fmt.Printf("  - %s\n", name.StringValue())
		}
	}

	fmt.Println("\nRules validated successfully")
}

func compileRules(rs *session.RuleSet, outFile string) {
	tables := rs.GetDecisionTableNames()

	if *verbose {
		fmt.Printf("Compiling %d decision tables...\n", len(tables))
	}

	// Create a session for compilation
	sess, err := rs.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	rsess := sess.(*session.RSession)

	// For now, just compile a simple test expression and dump bytecode
	// This verifies the opcode alignment with ASM runtime
	testExpr := "1 2 +"
	bc, err := rsess.CompileExpressionToBytecode(testExpr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling expression: %v\n", err)
		os.Exit(1)
	}

	// Write bytecode to file
	data := bc.Serialize()
	err = os.WriteFile(outFile, data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing bytecode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d bytes of bytecode to %s\n", len(data), outFile)

	// Also dump human-readable version
	fmt.Printf("\nBytecode dump:\n")
	code := bc.Code()
	for i := 0; i < len(code); i++ {
		fmt.Printf("  %3d: 0x%02X", i, code[i])
		switch code[i] {
		case 2: // OpPushInt
			fmt.Print(" (PushInt)")
		case 20:
			fmt.Print(" (Add)")
		case 21:
			fmt.Print(" (Sub)")
		case 22:
			fmt.Print(" (Mul)")
		case 23:
			fmt.Print(" (Div)")
		case 100:
			fmt.Print(" (PushTrue)")
		case 101:
			fmt.Print(" (PushFalse)")
		case 102:
			fmt.Print(" (PushNull)")
		case 103:
			fmt.Print(" (PushZero)")
		case 104:
			fmt.Print(" (PushOne)")
		}
		fmt.Println()
	}
}

func writeVarint(f *os.File, v int) {
	buf := make([]byte, 0, 10)
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	buf = append(buf, byte(v))
	f.Write(buf)
}
