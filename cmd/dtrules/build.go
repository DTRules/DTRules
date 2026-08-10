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
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/analysis"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	dtrsync "github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// newWorkbookImporter builds a WorkbookImporter with a live EL compiler
// wired in. Any action_dsl / condition_dsl that fails to compile is recorded
// as a named drop in the build summary instead of silently passing through.
//
// It also seeds the importer with the project-wide EDD symbol table (field →
// type) discovered under xmlDir, so a decision table compiles its double/fixed
// operands correctly even when its EDD lives in a different workbook — the
// multi-file case. A workbook that carries its own EDD still overrides these
// per-workbook during import.
func newWorkbookImporter(xmlDir string) *excel.WorkbookImporter {
	imp := excel.NewWorkbookImporter()
	c := el.NewCompiler()
	// Reject calls to operators the engine does not implement, at build time
	// rather than mid-execution (#1020).
	c.SetOperatorChecker(func(name string) bool {
		_, ok := operators.GetByString(name)
		return ok
	})
	imp.SetELCompiler(c)
	if syms := authoring.LoadEDDSymbols(xmlDir); len(syms) > 0 {
		imp.SetSymbols(syms)
	}
	return imp
}

// buildOptions holds parsed options for the build command.
type buildOptions struct {
	path      string
	fromExcel bool
	dryRun    bool
	verbose   bool
	quiet     bool
	xmlDir    string
	excelDir  string
	// Deployment gate (#768). When set, the build refuses unless a
	// passing Full Review exists in .dtrules/last-review.json whose
	// project_hash matches the current XML state.
	requireReview bool
	// reviewMaxAge bounds how stale the cached review can be. 24h by
	// default, override with --max-age (h / m / s).
	reviewMaxAge time.Duration
}

// runBuild handles the `dtrules build [path]` command.
func (c *CLI) runBuild(args []string) int {
	opts := &buildOptions{
		path:         ".",
		reviewMaxAge: 24 * time.Hour,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from-excel":
			opts.fromExcel = true
		case "--dry-run":
			opts.dryRun = true
		case "-v", "--verbose":
			opts.verbose = true
		case "-q", "--quiet":
			opts.quiet = true
		case "--require-review":
			opts.requireReview = true
		case "--max-age":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: --max-age value %q is not a valid duration (e.g. 24h, 30m): %v\n", args[i+1], err)
					return 1
				}
				opts.reviewMaxAge = d
				i++
			}
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

	// Deployment gate (#768). When --require-review is set, the build
	// refuses unless a passing Full Review exists on disk whose hash
	// matches the current XML state. This is the bright line that keeps
	// untrusted rule content from shipping.
	if opts.requireReview {
		if code := enforceReviewGate(absPath, xmlDir, opts.reviewMaxAge); code != 0 {
			return code
		}
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

	// Pre-build: catch duplicate table names already on disk.
	if code := dupGate(xmlDir); code != 0 {
		return code
	}

	if opts.dryRun {
		fmt.Printf("[dry-run] Would run %s-authored build in %s\n", path, absPath)
		return c.runBuildDryRun(xmlDir, excelDir, path, opts)
	}

	var code int
	switch path {
	case "excel":
		code = c.runExcelAuthoredBuild(xmlDir, excelDir, opts)
	case "xml":
		code = c.runXMLAuthoredBuild(xmlDir, excelDir, opts)
	default:
		// MAP files sit outside the main sync pipeline, whose "nothing to
		// sync" verdict only considers DT/EDD workbooks. Without this, a map
		// sheet stale enough to be missing createentity `list=` could never
		// be refreshed by `dtrules build` at all — and the #993 guard's
		// advice to re-export would be advice the tool could not take.
		if err := c.syncMAPFiles(xmlDir, excelDir, "xml-to-excel", opts.verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: MAP sync error: %v\n", err)
		}
		code = runNoSyncAdvisory(xmlDir)
	}
	if code != 0 {
		return code
	}

	// Post-build: the Excel→XML import copies each workbook's sheets 1:1, so a
	// workbook that carries tables belonging to another file would duplicate
	// them across *_dt.xml. Re-check after the writes and fail loudly rather
	// than leave a corrupted project on disk (a clean pre-build check can't
	// see a duplicate the build itself introduces).
	if code := dupGate(xmlDir); code != 0 {
		fmt.Fprintln(os.Stderr, "\nThe build produced duplicate table names — likely a workbook that\ncontains tables belonging to another file. Check the Excel workbooks.")
		return code
	}
	return 0
}

// dupGate scans for duplicate decision-table names and returns a non-zero
// exit code (after printing findings) if any are present.
func dupGate(xmlDir string) int {
	findings, err := checkNoDupMarkers(xmlDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for duplicate table names: %v\n", err)
		return 1
	}
	if len(findings) > 0 {
		writeDupFindings(os.Stderr, findings)
		return 1
	}
	return 0
}

// runNoSyncAdvisory is the runBuild branch taken when sync detection
// returns "none" — both xml/ and excel/ are in agreement and no
// import/export work is needed. We still run the advisory pass
// against the current XML so authors get warnings on every
// invocation (#787). The pass is XML-driven and cheap; running it
// unconditionally matches the "warnings route through
// decisiontable.Analyze on every authoring surface" contract from
// #780. Without it, `dtrules build` on an unchanged project silently
// green-lights tables that have real findings.
//
// Extracted to a top-level function so the no-sync branch can be
// unit-tested without engineering a fixture that satisfies the
// CheckSyncCombined "in sync" condition — reaching that branch from
// a copyProject is unreliable because mtimes and hashes drift on
// copy. Direct invocation pins the behavior end-to-end.
func runNoSyncAdvisory(xmlDir string) int {
	// "Nothing to sync" used to be the whole message, and it was read as
	// "Excel and XML agree". It means only that no workbook is newer than its
	// XML. Content can still differ -- an XML written by an older exporter has
	// the same mtimes and different bytes -- and that is exactly the state
	// `verify` fails on, so a user could see this line and a red gate in the
	// same minute with nothing connecting them (#1051).
	//
	// Saying which question was answered, and naming the one that was not, is
	// as far as this branch can honestly go. Actually comparing content means
	// rebuilding into a temp tree, which is what verify does and what takes
	// minutes on TaxReturn -- too much for every build. And resolving a
	// difference automatically would be worse than useless: on CHIP the XML is
	// ahead of its workbook, so taking Excel silently reverts working rules
	// (#1058).
	fmt.Println("No workbook is newer than its XML — normalizing and running the advisory pass.")
	fmt.Println("  (this compares timestamps, not content; `dtrules verify` checks the bytes)")
	// Normalization (section renumbering, entity-number backfill) is
	// idempotent — run it even when Excel and XML are in sync, so a plain
	// `dtrules build` always leaves normalized files.
	if err := normalizeDTXMLFiles(xmlDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error normalizing XML: %v\n", err)
		return 1
	}
	step := &dtrsync.StepSummary{}
	runStaticAnalysis(xmlDir, step)
	if len(step.Warnings) > 0 {
		fmt.Printf("\nAdvisory: %d warning(s) found\n", len(step.Warnings))
		for _, w := range step.Warnings {
			if w.Table != "" {
				fmt.Printf("  WARN %s: %s [%s]\n", w.Table, w.Reason, w.Item)
			} else {
				fmt.Printf("  WARN %s [%s]\n", w.Reason, w.Item)
			}
		}
	}
	return 0
}

// detectAuthoringPath returns "excel", "xml", or "none" based on modification times.
func detectAuthoringPath(xmlDir, excelDir string, opts *buildOptions) (string, error) {
	if opts.fromExcel {
		return "excel", nil
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
	importer := newWorkbookImporter(xmlDir)
	syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: importer})
	exporter := excel.NewWorkbookExporter()
	syncer.SetExporter(&workbookExporterAdapter{impl: exporter})

	direction, workbooks, err := syncer.CheckSyncCombined()
	if err != nil {
		return "", err
	}

	switch direction {
	case dtrsync.ExcelToXML:
		return "excel", nil
	case dtrsync.XMLToExcel:
		return "xml", nil
	case dtrsync.Conflict:
		// Some workbooks are newer than their XML and vice versa. Silently
		// treating this as "none" used to report "No changes detected" and
		// hide the divergence entirely. Surface it and make the user pick a
		// direction explicitly.
		fmt.Fprintln(os.Stderr, "ERROR: Excel and XML have diverged in different directions:")
		for _, wb := range workbooks {
			switch wb.Direction {
			case dtrsync.ExcelToXML:
				fmt.Fprintf(os.Stderr, "  %-40s Excel is newer\n", wb.RelPath)
			case dtrsync.XMLToExcel:
				fmt.Fprintf(os.Stderr, "  %-40s XML is newer\n", wb.RelPath)
			}
		}
		return "", fmt.Errorf("sync conflict: pass --from-excel to make Excel win, or touch the XML files to make XML win")
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

// normalizeDTXMLFiles rewrites every *_dt.xml AND *_edd.xml under xmlDir
// through the canonical WriteXML funnels, which renumber DT sections
// sequentially and backfill missing entity numbers (the 100 grid).
func normalizeDTXMLFiles(xmlDir string) error {
	return filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		lower := strings.ToLower(d.Name())
		switch {
		case strings.HasSuffix(lower, "_dt.xml"):
			before, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			doc, perr := excel.UnmarshalDecisionTablesXML(before)
			if perr != nil {
				return fmt.Errorf("%s: %w", p, perr)
			}
			imp := excel.NewDTImporter()
			if werr := imp.WriteXML(doc, p); werr != nil {
				return fmt.Errorf("%s: %w", p, werr)
			}
		case strings.HasSuffix(lower, "_edd.xml"):
			before, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			doc, perr := excel.UnmarshalEDDXML(before)
			if perr != nil {
				return fmt.Errorf("%s: %w", p, perr)
			}
			imp := excel.NewEDDImporter()
			if werr := imp.WriteXML(doc, p); werr != nil {
				return fmt.Errorf("%s: %w", p, werr)
			}
		}
		return nil
	})
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

	// Reaching this branch already decided the answer: --from-excel, or an
	// xml/ that does not exist yet, or detection saying Excel is newer.
	// Without forcing it, SyncAll re-derives direction per workbook from
	// mtimes and answers NoSync whenever the XML is not older — so the
	// import silently did not run, reporting "XML is already up to date"
	// with tables=0 and files-written=0.
	//
	// That made content drift uncorrectable. Excel is the system of record,
	// but a project whose XML was written by an older exporter could not be
	// brought back into agreement through build at any timestamp, while
	// verify's rebuild (into an empty directory, so always written) saw the
	// current format. verify then reported `content differs from build
	// output` forever (#1051).
	syncOpts.ForceDirection = dtrsync.ExcelToXML

	syncer := dtrsync.NewSyncerWithOptions(xmlDir, excelDir, syncOpts)
	syncer.SetUseCombinedWorkbooks(true)

	importer := newWorkbookImporter(xmlDir)
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

	importer := newWorkbookImporter(xmlDir)
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

	// Normalize the source XML in place BEFORE exporting: section numbers
	// (contexts/conditions/actions) become clean ordinals via the canonical
	// WriteXML funnel. The step-2 re-import may be skipped as already-in-
	// sync, so this is the only guaranteed normalization pass for
	// XML-authored projects.
	if err := normalizeDTXMLFiles(xmlDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error normalizing XML: %v\n", err)
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

	importer2 := newWorkbookImporter(xmlDir)
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

	importer := newWorkbookImporter(xmlDir)
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

// createEntityKey identifies a createentity row across a round trip. Entity
// plus tag is what the mapping loader keys on; `list` is the attribute at
// risk, so it is deliberately not part of the identity.
func createEntityKey(ce excel.MapCreateEntity) string {
	return ce.Entity + "\x00" + ce.Tag + "\x00" + ce.ID
}

// listsLost reports whether the imported rows dropped a `list=` that the
// existing XML carried. A sheet predating the list column reads back with
// every List empty (#993).
func listsLost(imported, existing []excel.MapCreateEntity) bool {
	have := map[string]string{}
	for _, ce := range existing {
		if ce.List != "" {
			have[createEntityKey(ce)] = ce.List
		}
	}
	if len(have) == 0 {
		return false
	}
	for _, ce := range imported {
		if ce.List == "" && have[createEntityKey(ce)] != "" {
			return true
		}
	}
	return false
}

// mergeCreateEntityLists restores a `list=` the sheet could not carry,
// leaving every other imported value alone — an edit made in Excel still
// wins, only the unrepresentable attribute is recovered.
func mergeCreateEntityLists(imported, existing []excel.MapCreateEntity) []excel.MapCreateEntity {
	have := map[string]string{}
	for _, ce := range existing {
		if ce.List != "" {
			have[createEntityKey(ce)] = ce.List
		}
	}
	out := make([]excel.MapCreateEntity, len(imported))
	copy(out, imported)
	for i := range out {
		if out[i].List == "" {
			if l, ok := have[createEntityKey(out[i])]; ok {
				out[i].List = l
			}
		}
	}
	return out
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
		// Clobber guard. A workbook written by an older DTRules can be
		// missing structure the XML has, and importing it would quietly
		// delete that structure from a working ruleset. Two shapes, both
		// seen in the wild:
		//
		//   1. No structural rows at all (pre-#938 sheet).
		//   2. createentity rows present but their `list` column empty,
		//      because the sheet predates that column (#993). This one is
		//      nastier: the section is non-empty, so a count-based guard
		//      passes it through, and `list=` is what appends each parsed
		//      entity onto the collection the tables iterate. Losing it
		//      leaves every `for all` running over an empty array — the
		//      Accumulate staking ruleset silently returned 0 for every
		//      staked balance while the build printed "OK — no drops".
		//
		// The rule: never let an import *remove* structure. Compare
		// against the XML being replaced and carry forward anything the
		// sheet cannot account for.
		if existing, lerr := excel.LoadMapXMLFromFile(xmlPath); lerr == nil {
			warn := func(what string) {
				fmt.Printf("  Warning: %s is missing %s that %s has; kept the existing values — re-export to refresh the sheet\n",
					filepath.Base(path), what, filepath.Base(xmlPath))
			}
			if len(mapXML.CreateEntities) == 0 && len(existing.CreateEntities) > 0 {
				mapXML.CreateEntities = existing.CreateEntities
				warn("createentity rows")
			} else if listsLost(mapXML.CreateEntities, existing.CreateEntities) {
				mapXML.CreateEntities = mergeCreateEntityLists(mapXML.CreateEntities, existing.CreateEntities)
				warn("createentity list= attributes")
			}
			if len(mapXML.EntityDecls) == 0 && len(existing.EntityDecls) > 0 {
				mapXML.EntityDecls = existing.EntityDecls
				warn("entity cardinality declarations")
			}
			if len(mapXML.InitialEntities) == 0 && len(existing.InitialEntities) > 0 {
				mapXML.InitialEntities = existing.InitialEntities
				warn("the initialization stack")
			}
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

// runStaticAnalysis walks xmlDir, runs the matrix-driven advisory pass on
// every decision table, runs the EDD usage analyzer, and appends
// warnings to step. Errors are non-fatal and printed to stderr.
//
// Every *_dt.xml file is parsed and each table is run through the same
// decisiontable.Analyze entry point used by `dtrules table warnings`
// and `dtrules review`. Covers no-op columns, subsumption, DSL-negation
// unreachability, FIRST-policy redundant conditions (#762), and
// assignment-only tables (#763) — everything that is provable from the
// Y/N matrix and DSL strings without a compiled tree.
//
// Tree-based checks (#765 unreachable column, #766 dead condition row)
// need the compiled decision tree, not just the matrix. They aren't
// wired here because building a full RuleSet on every `dtrules build`
// is too expensive on large projects (TaxReturn-scale load + optimize
// takes minutes). A future `--full` flag, or wiring that piggybacks on
// a compile the build already pays for, can re-introduce them.
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
			in := buildAnalysisInputs(table)
			for _, w := range decisiontable.Analyze(in) {
				step.AddWarning(w.Table, w.Column, w.Kind, formatWarningReason(w))
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

// formatWarningReason renders a decisiontable.Warning's reason text for the
// build summary. Row scope is folded into the reason because StepSummary
// only has (table, column, item, reason) slots — there's no row slot.
func formatWarningReason(w decisiontable.Warning) string {
	if w.ConditionRow >= 0 {
		return fmt.Sprintf("condition row %d %s", w.ConditionRow+1, w.Reason)
	}
	return w.Reason
}

// buildAnalysisInputs converts a DecisionTableXML into the Inputs the
// advisory pass consumes. Policy comes from <Type> on the table so the
// FIRST-policy redundancy check (#762) fires when applicable.
func buildAnalysisInputs(table *excel.DecisionTableXML) decisiontable.Inputs {
	// One width for the table, and each row rendered at it. The conversion
	// used to be open-coded here, which meant a fourth copy of the same
	// defaulting and bounds-checking (#1079).
	width := excel.TableWidth(table)

	conditions := make([]decisiontable.ConditionRow, len(table.Conditions))
	for i := range table.Conditions {
		conditions[i] = decisiontable.ConditionRow{
			DSL:     table.Conditions[i].DSL,
			Columns: table.Conditions[i].Row(width),
		}
	}

	actions := make([]decisiontable.ActionRow, len(table.Actions))
	for i := range table.Actions {
		actions[i] = decisiontable.ActionRow{
			DSL:     table.Actions[i].DSL,
			Columns: table.Actions[i].Row(width),
		}
	}

	return decisiontable.Inputs{
		Name:       table.TableName,
		Policy:     table.AttributeFields.Type,
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     width,
	}
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
  dtrules build --dry-run
  dtrules build --xml-dir pkg/dtrules/rules --excel-dir pkg/dtrules/excel /path/to/project`)
}
