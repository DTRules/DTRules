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

	"github.com/DTRules/DTRules/go/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
	"github.com/DTRules/DTRules/go/pkg/dtrules/testsupport"
	"github.com/DTRules/DTRules/go/pkg/dtrules/trace"
)

var (
	// Basic options
	eddFile    = flag.String("edd", "", "Path to EDD XML file")
	dtFile     = flag.String("dt", "", "Path to Decision Tables XML file")
	rulesDir   = flag.String("rules", "", "Directory containing EDD.xml and DecisionTables.xml")
	entryPoint = flag.String("entry", "", "Decision table entry point to execute")
	validate   = flag.Bool("validate", false, "Validate rules without executing")
	listTables = flag.Bool("list", false, "List all decision tables")
	verbose    = flag.Bool("v", false, "Verbose output")
	traceFlag  = flag.Bool("trace", false, "Enable trace output during execution")
	debug      = flag.Bool("debug", false, "Enable debug output during execution")

	// Trace analysis options
	traceFile = flag.String("trace-file", "", "Analyze a trace file")
	traceNode = flag.Int("trace-node", 0, "Set state to specific node number in trace")

	// Coverage options
	coverageDir = flag.String("coverage", "", "Generate coverage report from trace files in directory")

	// Test harness options
	testDir       = flag.String("test", "", "Run tests from directory")
	testOutputDir = flag.String("test-output", "", "Test output directory")

	// Change report options
	comparePath = flag.String("compare", "", "Compare with reference rules at path")

	// Compile and export options (from 5.0-SNAPSHOT)
	compile      = flag.String("compile", "", "Compile rules to bytecode file (.dtbc)")
	exportXLS    = flag.String("export", "", "Export decision tables and EDD to Excel files (prefix, deprecated)")
	exportDT     = flag.String("export-dt", "", "Export decision tables to Excel file (.xlsx)")
	exportEDD    = flag.String("export-edd", "", "Export EDD to Excel file (.xlsx)")
	exportDTDir  = flag.String("export-dt-dir", "", "Export decision tables to directory (grouped by xls_file)")
	exportEDDDir = flag.String("export-edd-dir", "", "Export EDD to directory (grouped by xls_file)")
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
		fmt.Fprintf(os.Stderr, "  %s -edd ./EDD.xml -dt ./DecisionTables.xml -export ./output\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nTrace Analysis:\n")
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -trace-file trace.xml -trace-node 428\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nCoverage:\n")
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -coverage ./output/\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nTesting:\n")
		fmt.Fprintf(os.Stderr, "  %s -rules ./rules -test ./testfiles/ -test-output ./output/ -entry Main\n", os.Args[0])
	}

	flag.Parse()

	// Handle trace analysis mode
	if *traceFile != "" {
		handleTraceAnalysis()
		return
	}

	// Handle coverage mode
	if *coverageDir != "" {
		handleCoverage()
		return
	}

	// Handle test mode
	if *testDir != "" {
		handleTest()
		return
	}

	// Standard execution mode requires rules
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

	// Handle compare mode
	if *comparePath != "" {
		handleCompare(rs)
		return
	}

	// Handle compile mode
	if *compile != "" {
		compileRules(rs, *compile)
		return
	}

	// Handle export mode (legacy prefix-based)
	if *exportXLS != "" {
		exportToExcel(rs, *exportXLS)
		return
	}

	// Handle separate export modes
	if *exportDT != "" || *exportEDD != "" {
		exportSeparate(rs, *exportDT, *exportEDD)
		return
	}

	// Handle directory export modes (grouped by xls_file)
	if *exportDTDir != "" || *exportEDDDir != "" {
		exportToDirectories(rs, *exportDTDir, *exportEDDDir)
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
	if *traceFlag {
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

func handleTraceAnalysis() {
	// Load trace file
	t := trace.NewTrace()
	root, err := t.Load(*traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading trace file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded trace with %d nodes\n", root.Count())

	// If no node specified, just print summary
	if *traceNode == 0 {
		t.Print(os.Stdout)
		return
	}

	// Find the specified node
	node := t.Find(*traceNode)
	if node == nil {
		fmt.Fprintf(os.Stderr, "Node %d not found in trace\n", *traceNode)
		os.Exit(1)
	}

	fmt.Printf("Found node %d: %s\n", *traceNode, node.Name)

	// If rules are specified, reconstruct state
	if *rulesDir != "" || (*eddFile != "" && *dtFile != "") {
		eddPath, dtPath, err := resolveFilePaths()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		rs, err := loadRuleSet(eddPath, dtPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
			os.Exit(1)
		}

		sess, err := t.SetState(rs, node)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting state: %v\n", err)
			os.Exit(1)
		}

		if sess != nil {
			fmt.Println("\nState reconstructed successfully")
			fmt.Printf("Entities created: %d\n", len(t.GetEntityTable()))
			fmt.Printf("Arrays created: %d\n", len(t.GetArrayTable()))
			fmt.Printf("Changes tracked: %d\n", len(t.GetChanges()))

			// Print entity info
			if *verbose {
				fmt.Println("\nEntities:")
				for id, entity := range t.GetEntityTable() {
					fmt.Printf("  %s: %s (id=%s)\n",
						entity.GetName().StringValue(),
						entity.GetName().StringValue(),
						id)
				}

				fmt.Println("\nChanges:")
				for _, change := range t.GetChanges() {
					tableNum := -1
					if change.ExecuteTable != nil {
						tableNum = change.ExecuteTable.Number
					}
					fmt.Printf("  Entity %d, attr %s changed at node %d\n",
						change.EntityID, change.AttributeKey, tableNum)
				}
			}
		}
	}

	// Print actions at this position
	actions := node.GetActions()
	if actions != nil {
		fmt.Printf("\nActions at position: %v\n", actions)
	}
}

func handleCoverage() {
	// Need rules for coverage
	eddPath, dtPath, err := resolveFilePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rs, err := loadRuleSet(eddPath, dtPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		os.Exit(1)
	}

	// Create coverage analyzer
	cov, err := testsupport.NewCoverage(rs, *coverageDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating coverage analyzer: %v\n", err)
		os.Exit(1)
	}

	// Compute coverage
	if err := cov.Compute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error computing coverage: %v\n", err)
		os.Exit(1)
	}

	// Print report
	cov.PrintReport(os.Stdout)

	fmt.Printf("\nProcessed %d trace files\n", len(cov.GetTraceFilesProcessed()))
	fmt.Printf("Total column executions: %d\n", cov.GetTotalColumns())
	fmt.Printf("Minimum files for coverage: %d\n", len(cov.GetMinFilesNeeded()))
}

func handleTest() {
	// Need rules for testing
	eddPath, dtPath, err := resolveFilePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rs, err := loadRuleSet(eddPath, dtPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		os.Exit(1)
	}

	// Create test harness
	harness := testsupport.NewTestHarness()
	harness.SetTestDirectory(*testDir)
	if *testOutputDir != "" {
		harness.SetOutputDirectory(*testOutputDir)
	}
	harness.Trace = *traceFlag
	harness.Verbose = *verbose

	if *entryPoint != "" {
		harness.DecisionTableName = *entryPoint
	}

	// Set rule set
	err = harness.LoadRuleSet(eddPath, dtPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules into harness: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	if err := harness.RunTests(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
		os.Exit(1)
	}

	// Suppress unused warning
	_ = rs
}

func handleCompare(_ *session.RuleSet) {
	// For now, just report that compare would happen
	fmt.Printf("Would compare rules with reference at: %s\n", *comparePath)
	fmt.Println("(Full comparison functionality requires DTRules.xml configuration)")
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

func exportToExcel(rs *session.RuleSet, prefix string) {
	exporter := excel.NewExporter(rs)

	// Export decision tables
	dtFile := prefix + "_dt.xlsx"
	if *verbose {
		fmt.Printf("Exporting decision tables to: %s\n", dtFile)
	}
	if err := exporter.ExportDecisionTables(dtFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting decision tables: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s with %d decision tables\n", dtFile, len(rs.GetDecisionTableNames()))

	// Export EDD
	eddFile := prefix + "_edd.xlsx"
	if *verbose {
		fmt.Printf("Exporting EDD to: %s\n", eddFile)
	}
	if err := exporter.ExportEDD(eddFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting EDD: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s with %d entities\n", eddFile, len(rs.GetEntityNames()))
}

func exportSeparate(rs *session.RuleSet, dtPath, eddPath string) {
	exporter := excel.NewExporter(rs)

	// Export decision tables if path provided
	if dtPath != "" {
		if *verbose {
			fmt.Printf("Exporting decision tables to: %s\n", dtPath)
		}
		if err := exporter.ExportDecisionTables(dtPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting decision tables: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created %s with %d decision tables\n", dtPath, len(rs.GetDecisionTableNames()))
	}

	// Export EDD if path provided
	if eddPath != "" {
		if *verbose {
			fmt.Printf("Exporting EDD to: %s\n", eddPath)
		}
		if err := exporter.ExportEDD(eddPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting EDD: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created %s with %d entities\n", eddPath, len(rs.GetEntityNames()))
	}
}

func exportToDirectories(rs *session.RuleSet, dtDir, eddDir string) {
	exporter := excel.NewExporter(rs)

	// Export decision tables if directory provided
	if dtDir != "" {
		if *verbose {
			fmt.Printf("Exporting decision tables to directory: %s\n", dtDir)
		}
		if err := exporter.ExportDecisionTablesToDir(dtDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting decision tables: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported decision tables to %s (grouped by xls_file)\n", dtDir)
	}

	// Export EDD if directory provided
	if eddDir != "" {
		if *verbose {
			fmt.Printf("Exporting EDD to directory: %s\n", eddDir)
		}
		if err := exporter.ExportEDDToDir(eddDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting EDD: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported EDD to %s (grouped by xls_file)\n", eddDir)
	}
}
