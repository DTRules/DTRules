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

	"github.com/DTRules/DTRules/pkg/dtrules/analysis"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	dtrsync "github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// newWorkbookImporter builds a WorkbookImporter with a live EL compiler
// wired in. Any action_dsl / condition_dsl that fails to compile is recorded
// as a named drop in the build summary instead of silently passing through.
func newWorkbookImporter() *excel.WorkbookImporter {
	imp := excel.NewWorkbookImporter()
	imp.SetELCompiler(el.NewCompiler())
	return imp
}

// buildOptions holds parsed options for the build command.
type buildOptions struct {
	path      string
	fromXML   bool
	fromExcel bool
	dryRun    bool
	verbose   bool
	quiet     bool
	xmlDir    string
	excelDir  string
}

// runBuild handles the `dtrules build [path]` command.
func (c *CLI) runBuild(args []string) int {
	opts := &buildOptions{
		path: ".",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from-xml":
			opts.fromXML = true
		case "--from-excel":
			opts.fromExcel = true
		case "--dry-run":
			opts.dryRun = true
		case "-v", "--verbose":
			opts.verbose = true
		case "-q", "--quiet":
			opts.quiet = true
		case "--xml-dir":
			if i+1 < len(args) {
				opts.xmlDir = args[i+1]
				i++
			}
		case "--excel-dir":
			if i+1 < len(args) {
				opts.excelDir = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				opts.path = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				c.printBuildUsage()
				return 1
			}
		}
	}

	if opts.fromXML && opts.fromExcel {
		fmt.Fprintln(os.Stderr, "Error: --from-xml and --from-excel are mutually exclusive")
		return 1
	}

	absPath, err := filepath.Abs(opts.path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		return 1
	}

	xmlDir, excelDir, err := resolveDirs(absPath, opts.xmlDir, opts.excelDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	// Validate directories exist
	if !dirExists(xmlDir) && !dirExists(excelDir) {
		if opts.xmlDir != "" || opts.excelDir != "" {
			if !dirExists(xmlDir) {
				fmt.Fprintf(os.Stderr, "ERROR: could not find xml directory\n  Tried: %s\n  Use --xml-dir <path> or declare <xml_dir> in DTRules.xml.\n", xmlDir)
			}
			if !dirExists(excelDir) {
				fmt.Fprintf(os.Stderr, "ERROR: could not find excel directory\n  Tried: %s\n  Use --excel-dir <path> or declare <excel_dir> in DTRules.xml.\n", excelDir)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: no xml/ or excel/ directory found in %s\n", absPath)
			fmt.Fprintln(os.Stderr, "Run 'dtrules init' to create a new project.")
		}
		return 1
	}

	// Detect authoring path when neither flag is given
	path, err := detectAuthoringPath(xmlDir, excelDir, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting authoring path: %v\n", err)
		return 1
	}

	if opts.dryRun {
		fmt.Printf("[dry-run] Would run %s-authored build in %s\n", path, absPath)
		return c.runBuildDryRun(xmlDir, excelDir, path, opts)
	}

	switch path {
	case "excel":
		return c.runExcelAuthoredBuild(xmlDir, excelDir, opts)
	case "xml":
		return c.runXMLAuthoredBuild(xmlDir, excelDir, opts)
	default:
		fmt.Println("Nothing to do: all files are in sync.")
		return 0
	}
}

// detectAuthoringPath returns "excel", "xml", or "none" based on modification times.
func detectAuthoringPath(xmlDir, excelDir string, opts *buildOptions) (string, error) {
	if opts.fromExcel {
		return "excel", nil
	}
	if opts.fromXML {
		return "xml", nil
	}

	if !dirExists(xmlDir) {
		// No XML yet — must be Excel-authored
		return "excel", nil
	}
	if !dirExists(excelDir) {
		// No Excel yet — must be XML-authored
		return "xml", nil
	}

	syncer := dtrsync.NewSyncerWithOptions(xmlDir, excelDir, dtrsync.DefaultOptions())
	syncer.SetUseCombinedWorkbooks(true)
	importer := newWorkbookImporter()
	syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: importer})
	exporter := excel.NewWorkbookExporter()
	syncer.SetExporter(&workbookExporterAdapter{impl: exporter})

	direction, _, err := syncer.CheckSyncCombined()
	if err != nil {
		return "", err
	}

	switch direction {
	case dtrsync.ExcelToXML:
		return "excel", nil
	case dtrsync.XMLToExcel:
		return "xml", nil
	default:
		// No sync needed — re-compile from existing Excel to produce execution XML
		return "none", nil
	}
}

// importStatsToStep converts an excel.ImportStats to a dtrsync.StepSummary.
func importStatsToStep(s *excel.ImportStats) *dtrsync.StepSummary {
	if s == nil {
		return &dtrsync.StepSummary{}
	}
	step := &dtrsync.StepSummary{
		Tables:       s.Tables,
		Actions:      s.Actions,
		Conditions:   s.Conditions,
		Entities:     s.Entities,
		Mappings:     s.Mappings,
		Compiled:     s.Compiled,
		FilesWritten: s.Files,
	}
	for _, d := range s.Drops {
		step.Drops = append(step.Drops, dtrsync.Drop{
			Table:  d.Table,
			Column: d.Column,
			Item:   d.Item,
			Reason: d.Reason,
		})
	}
	for _, w := range s.Warnings {
		step.Warnings = append(step.Warnings, dtrsync.Drop{
			Table:  w.Table,
			Column: w.Column,
			Item:   w.Item,
			Reason: w.Reason,
		})
	}
	return step
}

// exportStatsToStep converts an excel.ExportStats to a dtrsync.StepSummary.
func exportStatsToStep(s *excel.ExportStats) *dtrsync.StepSummary {
	if s == nil {
		return &dtrsync.StepSummary{}
	}
	step := &dtrsync.StepSummary{
		Tables:          s.Tables,
		Actions:         s.Actions,
		Conditions:      s.Conditions,
		Entities:        s.Entities,
		Mappings:        s.Mappings,
		PostfixStripped: s.PostfixStripped,
		FilesWritten:    s.Files,
	}
	for _, d := range s.Drops {
		step.Drops = append(step.Drops, dtrsync.Drop{
			Table:  d.Table,
			Column: d.Column,
			Item:   d.Item,
			Reason: d.Reason,
		})
	}
	return step
}

// printBuildSummary prints the build summary unless quiet mode suppresses it.
func printBuildSummary(summary *dtrsync.BuildSummary, quiet bool) {
	if quiet && !summary.HasErrors() {
		return
	}
	fmt.Print(summary.Format())
}

// runExcelAuthoredBuild: Excel → XML (import + EL compile).
// The canonical Excel already exists; we update XML to match.
func (c *CLI) runExcelAuthoredBuild(xmlDir, excelDir string, opts *buildOptions) int {
	fmt.Println("Build: Excel-authored path")
	fmt.Println("  Importing Excel → XML (compiling EL expressions)...")

	syncOpts := dtrsync.DefaultOptions()
	syncOpts.Verbose = opts.verbose

	syncer := dtrsync.NewSyncerWithOptions(xmlDir, excelDir, syncOpts)
	syncer.SetUseCombinedWorkbooks(true)

	importer := newWorkbookImporter()
	importer.SetVerbose(opts.verbose)
	importer.ResetStats()
	syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: importer})

	wbExporter := excel.NewWorkbookExporter()
	wbExporter.SetVerbose(opts.verbose)
	syncer.SetExporter(&workbookExporterAdapter{impl: wbExporter})

	// Ensure XML dir exists
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating xml dir: %v\n", err)
		return 1
	}

	result, err := syncer.SyncAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during import: %v\n", err)
		return 1
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", e)
		}
		return 1
	}

	if result.ExcelToXMLCount > 0 {
		fmt.Printf("  Imported %d workbook(s).\n", result.ExcelToXMLCount)
	} else {
		fmt.Println("  XML is already up to date.")
	}

	// Sync MAP files (excel→xml direction, outside main sync pipeline)
	if err := c.syncMAPFiles(xmlDir, excelDir, "excel-to-xml", opts.verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: MAP sync error: %v\n", err)
	}

	summary := &dtrsync.BuildSummary{
		ImportStep: importStatsToStep(importer.TakeStats()),
	}
	runStaticAnalysis(xmlDir, summary.ImportStep)
	printBuildSummary(summary, opts.quiet)

	fmt.Println("Build complete.")
	if summary.HasErrors() {
		return 1
	}
	return 0
}

// runXMLAuthoredBuild: XML → Excel (export) then Excel → XML (re-import to normalize).
func (c *CLI) runXMLAuthoredBuild(xmlDir, excelDir string, opts *buildOptions) int {
	fmt.Println("Build: XML-authored path")

	// Step 1: Export XML → Excel
	fmt.Println("  Step 1/2: Exporting XML → Excel...")

	syncOpts := dtrsync.DefaultOptions()
	syncOpts.Verbose = opts.verbose
	syncOpts.ConflictResolution = "prefer-xml"

	syncer := dtrsync.NewSyncerWithOptions(xmlDir, excelDir, syncOpts)
	syncer.SetUseCombinedWorkbooks(true)

	importer := newWorkbookImporter()
	importer.SetVerbose(opts.verbose)
	syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: importer})

	wbExporter := excel.NewWorkbookExporter()
	wbExporter.SetVerbose(opts.verbose)
	wbExporter.ResetStats()
	syncer.SetExporter(&workbookExporterAdapter{impl: wbExporter})

	if err := os.MkdirAll(excelDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating excel dir: %v\n", err)
		return 1
	}

	result, err := syncer.SyncAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during export: %v\n", err)
		return 1
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", e)
		}
		return 1
	}

	if result.XMLToExcelCount > 0 {
		fmt.Printf("  Exported %d workbook(s) to Excel.\n", result.XMLToExcelCount)
	}

	exportStats := wbExporter.TakeStats()

	// Step 2: Re-import Excel → XML to normalize formatting and compile EL
	fmt.Println("  Step 2/2: Re-importing Excel → XML (normalizing + compiling EL)...")

	syncer2 := dtrsync.NewSyncerWithOptions(xmlDir, excelDir, syncOpts)
	syncer2.SetUseCombinedWorkbooks(true)

	importer2 := newWorkbookImporter()
	importer2.SetVerbose(opts.verbose)
	importer2.ResetStats()
	syncer2.SetWorkbookImporter(&workbookImporterAdapter{impl: importer2})

	wbExporter2 := excel.NewWorkbookExporter()
	wbExporter2.SetVerbose(opts.verbose)
	syncer2.SetExporter(&workbookExporterAdapter{impl: wbExporter2})

	result2, err := syncer2.SyncAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during re-import: %v\n", err)
		return 1
	}

	if len(result2.Errors) > 0 {
		for _, e := range result2.Errors {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", e)
		}
		return 1
	}

	if result2.ExcelToXMLCount > 0 {
		fmt.Printf("  Normalized %d workbook(s).\n", result2.ExcelToXMLCount)
	}

	// Sync MAP files (xml→excel direction, outside main sync pipeline)
	if err := c.syncMAPFiles(xmlDir, excelDir, "xml-to-excel", opts.verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: MAP sync error: %v\n", err)
	}

	summary := &dtrsync.BuildSummary{
		ExportStep: exportStatsToStep(exportStats),
		ImportStep: importStatsToStep(importer2.TakeStats()),
	}
	runStaticAnalysis(xmlDir, summary.ImportStep)
	printBuildSummary(summary, opts.quiet)

	fmt.Println("Build complete.")
	if summary.HasErrors() {
		return 1
	}
	return 0
}

// runBuildDryRun reports what would change without writing files.
func (c *CLI) runBuildDryRun(xmlDir, excelDir, authoringPath string, opts *buildOptions) int {
	if authoringPath == "none" {
		fmt.Println("[dry-run] No changes detected; nothing would be written.")
		return 0
	}

	syncOpts := dtrsync.DefaultOptions()
	syncOpts.Verbose = opts.verbose
	syncOpts.DryRun = true

	syncer := dtrsync.NewSyncerWithOptions(xmlDir, excelDir, syncOpts)
	syncer.SetUseCombinedWorkbooks(true)

	importer := newWorkbookImporter()
	syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: importer})

	wbExporter := excel.NewWorkbookExporter()
	syncer.SetExporter(&workbookExporterAdapter{impl: wbExporter})

	direction, workbooks, err := syncer.CheckSyncCombined()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking sync: %v\n", err)
		return 1
	}

	fmt.Printf("[dry-run] Detected direction: %s\n", direction)
	for _, wb := range workbooks {
		if wb.Direction != dtrsync.NoSync {
			fmt.Printf("[dry-run]   %s → %s\n", filepath.Base(wb.ExcelPath), wb.Direction)
		}
	}

	return 0
}

// syncMAPFiles handles _map.xml ↔ _map.xlsx synchronization which the main
// sync pipeline skips. direction is "xml-to-excel" or "excel-to-xml".
func (c *CLI) syncMAPFiles(xmlDir, excelDir, direction string, verbose bool) error {

	// Walk XML dir for _map.xml files (xml-to-excel direction)
	if direction == "xml-to-excel" {
		return filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(d.Name(), "_map.xml") {
				return nil
			}
			rel, _ := filepath.Rel(xmlDir, path)
			base := strings.TrimSuffix(rel, "_map.xml")
			xlsxPath := filepath.Join(excelDir, base+"_map.xlsx")

			// Check timestamps: export if XML is newer
			xmlInfo, _ := os.Stat(path)
			xlsxInfo, _ := os.Stat(xlsxPath)
			if xlsxInfo != nil && !xmlInfo.ModTime().After(xlsxInfo.ModTime()) {
				return nil // xlsx already up to date
			}

			mapXML, err := excel.LoadMapXMLFromFile(path)
			if err != nil {
				return fmt.Errorf("load MAP xml %s: %w", rel, err)
			}
			if err := excel.NewMapExporter().ExportToFile(mapXML, xlsxPath); err != nil {
				return fmt.Errorf("export MAP %s: %w", rel, err)
			}
			if verbose {
				fmt.Printf("  Exported %s → %s\n", rel, filepath.Base(xlsxPath))
			}
			return nil
		})
	}

	// Walk Excel dir for _map.xlsx files (excel-to-xml direction)
	return filepath.WalkDir(excelDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), "_map.xlsx") {
			return nil
		}
		rel, _ := filepath.Rel(excelDir, path)
		base := strings.TrimSuffix(rel, "_map.xlsx")
		xmlPath := filepath.Join(xmlDir, base+"_map.xml")

		// Check timestamps: import if xlsx is newer
		xlsxInfo, _ := os.Stat(path)
		xmlInfo, _ := os.Stat(xmlPath)
		if xmlInfo != nil && !xlsxInfo.ModTime().After(xmlInfo.ModTime()) {
			return nil // xml already up to date
		}

		mapXML, err := excel.NewMapImporter().ImportFile(path)
		if err != nil {
			return fmt.Errorf("import MAP xlsx %s: %w", rel, err)
		}
		if mapXML == nil || len(mapXML.Entries) == 0 {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(xmlPath), 0755); err != nil {
			return err
		}
		if err := excel.WriteMapXML(mapXML, xmlPath); err != nil {
			return fmt.Errorf("write MAP XML %s: %w", xmlPath, err)
		}
		if verbose {
			fmt.Printf("  Imported %s → %s\n", filepath.Base(path), filepath.Base(xmlPath))
		}
		return nil
	})
}

// runStaticAnalysis walks xmlDir, runs per-table and EDD usage checks, and
// appends warnings to step. Errors are non-fatal and printed to stderr.
func runStaticAnalysis(xmlDir string, step *dtrsync.StepSummary) {
	if err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), "_dt.xml") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		dt, parseErr := excel.UnmarshalDecisionTablesXML(data)
		if parseErr != nil {
			return nil
		}
		for i := range dt.Tables {
			table := &dt.Tables[i]
			conds, acts, maxCol := buildAnalysisInputs(table)
			warns := analyzeTableStructure(table.TableName, conds, acts, maxCol)
			for _, w := range warns {
				step.AddWarning(w.table, w.column, w.kind, w.reason)
			}
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: static analysis walk error: %v\n", err)
	}

	eddWarns, err := analysis.AnalyzeEDDUsage(xmlDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: EDD analysis error: %v\n", err)
		return
	}
	for _, w := range eddWarns {
		step.AddWarning("", 0, w.Field, w.Reason)
	}
}

type analysisWarn struct {
	table  string
	column int
	kind   string
	reason string
}

type analysisCondRow struct {
	dsl  string
	cols []string
}

type analysisActRow struct {
	dsl  string
	cols []string
}

var analysisNegationPairs = [][2]string{
	{" is equal to ", " is not equal to "},
	{" is not equal to ", " is equal to "},
	{" > ", " <= "},
	{" <= ", " > "},
	{" < ", " >= "},
	{" >= ", " < "},
}

func analysisDSLNegation(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	for _, pair := range analysisNegationPairs {
		if strings.Contains(a, pair[0]) && strings.Contains(b, pair[1]) {
			aBase := strings.Replace(a, pair[0], "\x00", 1)
			bBase := strings.Replace(b, pair[1], "\x00", 1)
			if aBase == bBase {
				return true
			}
		}
	}
	if strings.HasPrefix(b, "not ") && strings.TrimPrefix(b, "not ") == a {
		return true
	}
	if strings.HasPrefix(a, "not ") && strings.TrimPrefix(a, "not ") == b {
		return true
	}
	return false
}

// buildAnalysisInputs converts a DecisionTableXML into column arrays for analysis.
func buildAnalysisInputs(table *excel.DecisionTableXML) ([]analysisCondRow, []analysisActRow, int) {
	maxCol := 0
	for _, c := range table.Conditions {
		for _, cv := range c.Columns {
			if cv.Number > maxCol {
				maxCol = cv.Number
			}
		}
	}
	for _, a := range table.Actions {
		for _, cv := range a.Columns {
			if cv.Number > maxCol {
				maxCol = cv.Number
			}
		}
	}

	conditions := make([]analysisCondRow, len(table.Conditions))
	for i, c := range table.Conditions {
		row := analysisCondRow{dsl: c.DSL, cols: make([]string, maxCol)}
		for j := range row.cols {
			row.cols[j] = "-"
		}
		for _, cv := range c.Columns {
			if cv.Number >= 1 && cv.Number <= maxCol {
				row.cols[cv.Number-1] = cv.Value
			}
		}
		conditions[i] = row
	}

	actions := make([]analysisActRow, len(table.Actions))
	for i, a := range table.Actions {
		row := analysisActRow{dsl: a.DSL, cols: make([]string, maxCol)}
		for _, cv := range a.Columns {
			if cv.Number >= 1 && cv.Number <= maxCol {
				row.cols[cv.Number-1] = cv.Value
			}
		}
		actions[i] = row
	}

	return conditions, actions, maxCol
}

// analyzeTableStructure runs Parts 1+2 of static analysis on a single table.
func analyzeTableStructure(tableName string, conditions []analysisCondRow, actions []analysisActRow, maxCol int) []analysisWarn {
	if maxCol == 0 {
		return nil
	}

	var warnings []analysisWarn

	// Part 1a: no-action columns
	for col := 0; col < maxCol; col++ {
		hasAction := false
		for _, a := range actions {
			if col < len(a.cols) && strings.ToUpper(strings.TrimSpace(a.cols[col])) == "X" {
				hasAction = true
				break
			}
		}
		if !hasAction {
			warnings = append(warnings, analysisWarn{
				table:  tableName,
				column: col + 1,
				kind:   "no-op column",
				reason: fmt.Sprintf("column %d is redundant (no actions)", col+1),
			})
		}
	}

	// Build per-column constraint and action profiles for subsumption check.
	type profile struct {
		constraints map[int]string
		actionSet   map[int]bool
	}
	profiles := make([]profile, maxCol)
	for col := 0; col < maxCol; col++ {
		p := profile{
			constraints: make(map[int]string),
			actionSet:   make(map[int]bool),
		}
		for row, c := range conditions {
			v := ""
			if col < len(c.cols) {
				v = strings.ToUpper(strings.TrimSpace(c.cols[col]))
			}
			if v == "Y" || v == "N" {
				p.constraints[row] = v
			}
		}
		for row, a := range actions {
			if col < len(a.cols) && strings.ToUpper(strings.TrimSpace(a.cols[col])) == "X" {
				p.actionSet[row] = true
			}
		}
		profiles[col] = p
	}

	// Part 1b: subsumed columns
	for a := 0; a < maxCol; a++ {
		if len(profiles[a].actionSet) == 0 {
			continue
		}
		for b := 0; b < maxCol; b++ {
			if a == b {
				continue
			}
			bSubsumesA := true
			for row, bVal := range profiles[b].constraints {
				aVal, ok := profiles[a].constraints[row]
				if !ok || aVal != bVal {
					bSubsumesA = false
					break
				}
			}
			if bSubsumesA {
				for row := range profiles[a].actionSet {
					if !profiles[b].actionSet[row] {
						bSubsumesA = false
						break
					}
				}
			}
			if bSubsumesA && len(profiles[b].constraints) < len(profiles[a].constraints) {
				warnings = append(warnings, analysisWarn{
					table:  tableName,
					column: a + 1,
					kind:   "no-op column",
					reason: fmt.Sprintf("column %d is redundant (subsumed by column %d)", a+1, b+1),
				})
				break
			}
		}
	}

	// Part 2: unreachable columns — mutually exclusive Y conditions
	for col := 0; col < maxCol; col++ {
		var yRows []int
		for row, c := range conditions {
			v := ""
			if col < len(c.cols) {
				v = strings.ToUpper(strings.TrimSpace(c.cols[col]))
			}
			if v == "Y" {
				yRows = append(yRows, row)
			}
		}
		found := false
		for i := 0; i < len(yRows) && !found; i++ {
			for j := i + 1; j < len(yRows) && !found; j++ {
				if analysisDSLNegation(conditions[yRows[i]].dsl, conditions[yRows[j]].dsl) {
					warnings = append(warnings, analysisWarn{
						table:  tableName,
						column: col + 1,
						kind:   "unreachable column",
						reason: fmt.Sprintf("column %d can never match (conditions %d and %d are mutually exclusive)", col+1, yRows[i]+1, yRows[j]+1),
					})
					found = true
				}
			}
		}
	}

	return warnings
}

func (c *CLI) printBuildUsage() {
	fmt.Println(`Usage: dtrules build [path] [options]

Runs the full normalize-and-compile pipeline. Detects whether Excel or XML
files are newer and runs the appropriate path. Both paths end with canonical
Excel files and compiled execution XML on disk.

After every build a structured summary is printed showing table/action/condition
counts and any drops (EL compile errors, structural problems). The build exits
non-zero if any drops are detected.

Arguments:
  path                Project directory to build (default: .)

Options:
  --xml-dir <path>    Override XML directory (relative to project root or absolute)
  --excel-dir <path>  Override Excel directory (relative to project root or absolute)
  --from-excel        Force Excel-authored path (Excel → XML)
  --from-xml          Force XML-authored path (XML → Excel → XML)
  --dry-run           Report what would change without writing files
  -v, --verbose       Verbose output
  -q, --quiet         Suppress summary unless there are drops

Directory resolution (highest to lowest priority):
  1. --xml-dir / --excel-dir flags
  2. <xml_dir> / <excel_dir> elements in DTRules.xml
  3. Default: xml/ and excel/ relative to the project root

Examples:
  dtrules build
  dtrules build ./sampleprojects/TaxReturn
  dtrules build --from-xml
  dtrules build --dry-run
  dtrules build --xml-dir pkg/dtrules/rules --excel-dir pkg/dtrules/excel /path/to/project`)
}
