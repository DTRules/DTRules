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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
	"github.com/DTRules/DTRules/pkg/dtrules/version"
)

// CLI implements the dtrules command-line interface with subcommands.
type CLI struct {
	// Project paths
	xmlDir   string
	excelDir string
	verbose  bool
	force    bool

	// Sync components
	syncer   *sync.Syncer
	importer *excel.WorkbookImporter
	exporter *excel.Exporter
}

// workbookImporterAdapter adapts excel.WorkbookImporter to sync.WorkbookImporter interface.
type workbookImporterAdapter struct {
	impl *excel.WorkbookImporter
}

func (a *workbookImporterAdapter) ImportWorkbook(excelPath, xmlDir string) (dtPath, eddPath string, err error) {
	return a.impl.ImportWorkbookToDir(excelPath, xmlDir)
}

func (a *workbookImporterAdapter) HasDTSheet(excelPath string) (bool, error) {
	return a.impl.HasDTSheet(excelPath)
}

func (a *workbookImporterAdapter) HasEDDSheet(excelPath string) (bool, error) {
	return a.impl.HasEDDSheet(excelPath)
}

// workbookExporterAdapter adapts excel.WorkbookExporter to sync.ExcelExporter
// and sync.CombinedExporter interfaces.
type workbookExporterAdapter struct {
	impl *excel.WorkbookExporter
}

func (a *workbookExporterAdapter) ExportDecisionTable(xmlPath, excelPath string) error {
	return a.impl.ExportDecisionTable(xmlPath, excelPath)
}

func (a *workbookExporterAdapter) ExportEDD(xmlPath, excelPath string) error {
	return a.impl.ExportEDD(xmlPath, excelPath)
}

func (a *workbookExporterAdapter) ExportCombined(dtXMLPath, eddXMLPath, excelPath string) error {
	return a.impl.ExportCombined(dtXMLPath, eddXMLPath, excelPath)
}

// NewCLI creates a new CLI instance.
func NewCLI() *CLI {
	return &CLI{}
}

// Run executes the CLI with the given arguments.
func (c *CLI) Run(args []string) int {
	if len(args) < 1 {
		c.printUsage()
		return 1
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "sync":
		return c.runSync(cmdArgs)
	case "init":
		return c.runInit(cmdArgs)
	case "validate":
		return c.runValidate(cmdArgs)
	case "version":
		return c.runVersion()
	case "docs":
		return c.runDocs(cmdArgs)
	case "help", "-h", "--help":
		c.printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		c.printUsage()
		return 1
	}
}

func (c *CLI) printUsage() {
	fmt.Println(`DTRules - Decision Table Rules Engine

Usage: dtrules <command> [options]

Commands:
  sync      Synchronize Excel and XML files
  init      Initialize a DTRules project structure
  validate  Validate decision tables and EDD
  docs      Show embedded documentation (for AI and developers)
  version   Show version information
  help      Show this help message

Sync Commands:
  dtrules sync status              Show sync status of all files
  dtrules sync check               Check for pending user edits (exits non-zero if any)
  dtrules sync import              Import Excel files to XML
  dtrules sync export              Export XML files to Excel
  dtrules sync auto                Auto-sync based on timestamps

Documentation:
  dtrules docs                     List available documentation topics
  dtrules docs bigint              Arbitrary-precision integer support
  dtrules docs el                  Expression Language syntax
  dtrules docs decision-tables     How to write decision tables
  dtrules docs operators           All operators with examples

Options:
  --xml-dir      Directory containing XML files (default: ./xml)
  --excel-dir    Directory containing Excel files (default: ./excel)
  -v, --verbose  Verbose output

Examples:
  # Check if any Excel files have pending user edits
  dtrules sync check

  # Import all newer Excel files to XML
  dtrules sync import

  # Export all XML files to Excel (fails if user edits pending)
  dtrules sync export

  # Initialize a new DTRules project
  dtrules init my-rules

  # Get help with writing decision tables
  dtrules docs decision-tables

For more information, visit: https://github.com/DTRules/DTRules`)
}

// runDocs shows embedded documentation.
func (c *CLI) runDocs(args []string) int {
	if err := runDocs(args); err != nil {
		return 1
	}
	return 0
}

// runSync handles the sync subcommand.
func (c *CLI) runSync(args []string) int {
	if len(args) < 1 {
		c.printSyncUsage()
		return 1
	}

	// Parse all arguments, collecting flags and finding subcommand
	var subcmd string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--xml-dir":
			if i+1 < len(args) {
				c.xmlDir = args[i+1]
				i++
			}
		case "--excel-dir":
			if i+1 < len(args) {
				c.excelDir = args[i+1]
				i++
			}
		case "-v", "--verbose":
			c.verbose = true
		case "-f", "--force":
			c.force = true
		default:
			// Not a flag - must be the subcommand
			if !strings.HasPrefix(args[i], "-") && subcmd == "" {
				subcmd = args[i]
			}
		}
	}

	// Set defaults
	if c.xmlDir == "" {
		c.xmlDir = "./xml"
	}
	if c.excelDir == "" {
		c.excelDir = "./excel"
	}

	// Check subcommand was found
	if subcmd == "" {
		c.printSyncUsage()
		return 1
	}

	// Initialize syncer
	if err := c.initSyncer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing syncer: %v\n", err)
		return 1
	}

	switch subcmd {
	case "status":
		return c.syncStatus()
	case "check":
		return c.syncCheck()
	case "import":
		return c.syncImport()
	case "export":
		return c.syncExport()
	case "auto":
		return c.syncAuto()
	default:
		fmt.Fprintf(os.Stderr, "Unknown sync command: %s\n\n", subcmd)
		c.printSyncUsage()
		return 1
	}
}

func (c *CLI) printSyncUsage() {
	fmt.Println(`Usage: dtrules sync <command> [options]

Commands:
  status    Show sync status of all files
  check     Check for pending user edits (exits non-zero if any)
  import    Import Excel files to XML
  export    Export XML files to Excel (fails if user edits pending)
  auto      Auto-sync based on timestamps

Options:
  --xml-dir      Directory containing XML files (default: ./xml)
  --excel-dir    Directory containing Excel files (default: ./excel)
  -f, --force    Force export even if Excel files have pending edits
  -v, --verbose  Verbose output

Examples:
  dtrules sync status --xml-dir ./rules/xml --excel-dir ./rules/excel
  dtrules sync check
  dtrules sync import
  dtrules sync export`)
}

func (c *CLI) initSyncer() error {
	// Check directories exist
	if _, err := os.Stat(c.xmlDir); os.IsNotExist(err) {
		return fmt.Errorf("XML directory not found: %s", c.xmlDir)
	}
	if _, err := os.Stat(c.excelDir); os.IsNotExist(err) {
		return fmt.Errorf("Excel directory not found: %s", c.excelDir)
	}

	// Create syncer
	opts := sync.DefaultOptions()
	opts.Verbose = c.verbose
	if c.force {
		opts.ConflictResolution = "prefer-xml"
	}
	c.syncer = sync.NewSyncerWithOptions(c.xmlDir, c.excelDir, opts)

	// Set up importer (combined workbook mode)
	c.syncer.SetUseCombinedWorkbooks(true)
	c.importer = excel.NewWorkbookImporter()
	c.importer.SetVerbose(c.verbose)
	c.syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: c.importer})

	// Set up exporter
	exporter := excel.NewWorkbookExporter()
	exporter.SetVerbose(c.verbose)
	c.syncer.SetExporter(&workbookExporterAdapter{impl: exporter})

	return nil
}

// syncStatus shows the current sync status.
func (c *CLI) syncStatus() int {
	fmt.Printf("Sync Status\n")
	fmt.Printf("  XML Directory:   %s\n", c.xmlDir)
	fmt.Printf("  Excel Directory: %s\n", c.excelDir)
	fmt.Println()

	// Check manifest for user modifications
	manifest, err := c.syncer.GetManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading manifest: %v\n", err)
		return 1
	}

	modified, err := manifest.CheckAllExports()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking modifications: %v\n", err)
		return 1
	}

	if len(modified) > 0 {
		fmt.Println("⚠️  Excel files with pending user edits:")
		for _, f := range modified {
			fmt.Printf("    %s\n", f)
		}
		fmt.Println()
		fmt.Println("Run 'dtrules sync import' to import user changes to XML.")
	} else {
		fmt.Println("✓ No pending user edits detected.")
	}

	// Show tracked files
	tracked := manifest.ListTrackedFiles()
	if len(tracked) > 0 && c.verbose {
		fmt.Printf("\nTracked files (%d):\n", len(tracked))
		for _, f := range tracked {
			entry := manifest.GetEntry(f)
			if entry != nil {
				fmt.Printf("    %s (last export: %s)\n", f, entry.LastExportTime.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// Check sync direction
	direction, pairs, err := c.syncer.CheckSync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking sync: %v\n", err)
		return 1
	}

	if c.verbose && len(pairs) > 0 {
		fmt.Printf("\nFile pairs (%d):\n", len(pairs))
		for _, p := range pairs {
			status := "in sync"
			switch p.Direction {
			case sync.ExcelToXML:
				status = "Excel → XML needed"
			case sync.XMLToExcel:
				status = "XML → Excel needed"
			case sync.Conflict:
				status = "CONFLICT"
			}
			fmt.Printf("    %s: %s\n", filepath.Base(p.XMLPath), status)
		}
	}

	fmt.Printf("\nOverall direction: %s\n", direction)

	return 0
}

// syncCheck checks for pending user edits and exits non-zero if any found.
func (c *CLI) syncCheck() int {
	modified, err := c.syncer.CheckExcelModified()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking modifications: %v\n", err)
		return 1
	}

	if len(modified) > 0 {
		fmt.Fprintf(os.Stderr, "ERROR: Excel files have pending user edits:\n")
		for _, f := range modified {
			fmt.Fprintf(os.Stderr, "  %s\n", f)
		}
		fmt.Fprintf(os.Stderr, "\nRun 'dtrules sync import' to import user changes to XML.\n")
		fmt.Fprintf(os.Stderr, "Do NOT edit XML files until user changes are imported.\n")
		return 1
	}

	fmt.Println("✓ No pending user edits. Safe to edit XML.")
	return 0
}

// syncImport imports Excel files to XML.
func (c *CLI) syncImport() int {
	fmt.Println("Importing Excel to XML...")

	result, err := c.syncer.SyncAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during import: %v\n", err)
		return 1
	}

	if result.ExcelToXMLCount > 0 {
		fmt.Printf("✓ Imported %d file(s) from Excel to XML.\n", result.ExcelToXMLCount)
	} else {
		fmt.Println("No files needed import (XML is up to date).")
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors occurred:\n")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		return 1
	}

	return 0
}

// syncExport exports XML files to Excel.
func (c *CLI) syncExport() int {
	// Check for pending user edits (skip with --force)
	modified, err := c.syncer.CheckExcelModified()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking modifications: %v\n", err)
		return 1
	}

	if len(modified) > 0 && !c.force {
		fmt.Fprintf(os.Stderr, "ERROR: Cannot export - Excel files have pending user edits:\n")
		for _, f := range modified {
			fmt.Fprintf(os.Stderr, "  %s\n", f)
		}
		fmt.Fprintf(os.Stderr, "\nUser changes would be lost. Run 'dtrules sync import' first,\nor use --force to overwrite Excel files.\n")
		return 1
	}

	if len(modified) > 0 {
		fmt.Printf("Overwriting %d Excel file(s) with pending edits (--force).\n", len(modified))
	}

	fmt.Println("Exporting XML to Excel...")

	// For export, we need to load rules to create the exporter
	// This is a simplified version - full implementation would load rules
	result, err := c.syncer.SyncAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during export: %v\n", err)
		return 1
	}

	if result.XMLToExcelCount > 0 {
		fmt.Printf("✓ Exported %d file(s) from XML to Excel.\n", result.XMLToExcelCount)
	} else {
		fmt.Println("No files needed export (Excel is up to date).")
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors occurred:\n")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		return 1
	}

	return 0
}

// syncAuto performs automatic bidirectional sync.
func (c *CLI) syncAuto() int {
	fmt.Println("Auto-syncing based on timestamps...")

	result, err := c.syncer.SyncAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during sync: %v\n", err)
		return 1
	}

	if result.ExcelToXMLCount > 0 {
		fmt.Printf("  Imported %d file(s) from Excel to XML.\n", result.ExcelToXMLCount)
	}
	if result.XMLToExcelCount > 0 {
		fmt.Printf("  Exported %d file(s) from XML to Excel.\n", result.XMLToExcelCount)
	}
	if result.ConflictCount > 0 {
		fmt.Printf("  %d conflict(s) detected.\n", result.ConflictCount)
	}

	if result.ExcelToXMLCount == 0 && result.XMLToExcelCount == 0 {
		fmt.Println("✓ All files are in sync.")
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors occurred:\n")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		return 1
	}

	return 0
}

// runInit initializes a new DTRules project.
func (c *CLI) runInit(args []string) int {
	var projectDir string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectDir = args[0]
	} else {
		projectDir = "."
	}

	fmt.Printf("Initializing DTRules project in %s...\n", projectDir)

	// Create directory structure
	dirs := []string{
		filepath.Join(projectDir, "xml"),
		filepath.Join(projectDir, "excel"),
		filepath.Join(projectDir, "testfiles"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
			return 1
		}
		fmt.Printf("  Created %s/\n", dir)
	}

	// Create .gitignore for sync manifest
	gitignore := filepath.Join(projectDir, "excel", ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		content := "# DTRules sync manifest (local state, don't commit)\n.sync-manifest.json\n"
		if err := os.WriteFile(gitignore, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create .gitignore: %v\n", err)
		} else {
			fmt.Printf("  Created excel/.gitignore\n")
		}
	}

	fmt.Println("\n✓ DTRules project initialized.")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Create your decision tables in Excel")
	fmt.Println("  2. Run 'dtrules sync import' to generate XML")
	fmt.Println("  3. Run 'dtrules validate' to check your rules")

	return 0
}

// runValidate validates the decision tables and project structure.
func (c *CLI) runValidate(args []string) int {
	var strict bool
	var allowLegacy bool
	var projectDir string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--xml-dir":
			if i+1 < len(args) {
				c.xmlDir = args[i+1]
				i++
			}
		case "--excel-dir":
			if i+1 < len(args) {
				c.excelDir = args[i+1]
				i++
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectDir = args[i+1]
				i++
			}
		case "-v", "--verbose":
			c.verbose = true
		case "--strict":
			strict = true
		case "--allow-legacy":
			allowLegacy = true
		}
	}

	// Set defaults
	if projectDir == "" {
		projectDir = "."
	}
	if c.xmlDir == "" {
		c.xmlDir = filepath.Join(projectDir, "xml")
	}
	if c.excelDir == "" {
		c.excelDir = filepath.Join(projectDir, "excel")
	}

	fmt.Printf("Validating DTRules project in %s\n", projectDir)
	fmt.Println()

	exitCode := 0

	// 1. Validate project structure
	fmt.Println("Checking project structure...")
	structResult, err := sync.ValidateProjectStructure(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error validating structure: %v\n", err)
		return 1
	}

	if !structResult.IsValid() {
		for _, e := range structResult.Errors {
			fmt.Fprintf(os.Stderr, "  ERROR: %s\n", e.Error())
		}
		exitCode = 1
	} else {
		fmt.Println("  ✓ Project structure valid")
	}

	// Show structure warnings
	if len(structResult.Warnings) > 0 {
		for _, w := range structResult.Warnings {
			fmt.Printf("  WARNING: %s\n", w.String())
		}
	}

	// Report orphaned XML and missing XML
	if len(structResult.OrphanedXML) > 0 && c.verbose {
		fmt.Printf("\n  Orphaned XML files (no Excel source): %d\n", len(structResult.OrphanedXML))
		for _, f := range structResult.OrphanedXML {
			fmt.Printf("    %s\n", f)
		}
	}
	if len(structResult.MissingXML) > 0 && c.verbose {
		fmt.Printf("\n  Missing XML files (Excel not imported): %d\n", len(structResult.MissingXML))
		for _, f := range structResult.MissingXML {
			fmt.Printf("    %s\n", f)
		}
	}
	fmt.Println()

	// 2. Validate EL compliance (if XML directory exists)
	if structResult.IsValid() || dirExists(c.xmlDir) {
		fmt.Println("Checking EL compliance...")
		elResult, err := sync.ValidateELCompliance(c.xmlDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error validating EL compliance: %v\n", err)
			return 1
		}

		// Report compliant files
		if c.verbose && len(elResult.CompliantFiles) > 0 {
			fmt.Printf("  Compliant files: %d\n", len(elResult.CompliantFiles))
		}

		// Report files needing compilation
		if len(elResult.NeedsCompilationFiles) > 0 {
			fmt.Printf("  Files needing EL compilation: %d\n", len(elResult.NeedsCompilationFiles))
			for _, f := range elResult.NeedsCompilationFiles {
				fmt.Printf("    %s\n", filepath.Base(f))
			}
			fmt.Println("  Run 'dtrules sync import' to compile EL expressions.")
		}

		// Report legacy files
		if len(elResult.LegacyFiles) > 0 {
			fmt.Printf("\n  ⚠️  Legacy postfix files: %d\n", len(elResult.LegacyFiles))
			for _, f := range elResult.LegacyFiles {
				fmt.Printf("    %s\n", filepath.Base(f))
			}
			fmt.Println()
			fmt.Println("  These files contain hand-coded postfix without EL descriptions.")
			fmt.Println("  All postfix MUST come from EL compilation. To fix:")
			fmt.Println("    1. Add EL expressions to the Excel file")
			fmt.Println("    2. Run 'dtrules sync import' to compile")

			if strict {
				exitCode = 1
			}
			if !allowLegacy {
				fmt.Println()
				fmt.Println("  Use --allow-legacy to suppress this as an error.")
			}
		} else {
			fmt.Println("  ✓ All files are EL compliant")
		}
		fmt.Println()
	}

	// 3. Check sync status (if both directories exist)
	if dirExists(c.xmlDir) && dirExists(c.excelDir) {
		fmt.Println("Checking sync status...")
		if err := c.initSyncer(); err == nil {
			modified, err := c.syncer.CheckExcelModified()
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: could not check sync status: %v\n", err)
			} else if len(modified) > 0 {
				fmt.Printf("  ⚠️  Excel files with pending user edits: %d\n", len(modified))
				for _, f := range modified {
					fmt.Printf("    %s\n", f)
				}
				fmt.Println("  Run 'dtrules sync import' to import user changes.")
			} else {
				fmt.Println("  ✓ No pending user edits")
			}
		}
		fmt.Println()
	}

	// Summary
	if exitCode == 0 {
		fmt.Println("✓ Validation passed")
	} else {
		fmt.Println("✗ Validation failed")
	}

	return exitCode
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// runVersion shows version information.
func (c *CLI) runVersion() int {
	fmt.Println(version.Full())
	fmt.Println("Decision Table Rules Engine")
	fmt.Println("https://github.com/DTRules/DTRules")
	return 0
}
