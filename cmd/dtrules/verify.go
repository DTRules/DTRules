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
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/analysis"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
	"github.com/xuri/excelize/v2"
)

// verifyOptions holds parsed options for the verify command.
type verifyOptions struct {
	path     string
	diff     bool
	strict   bool
	xmlDir   string
	excelDir string
}

// verifyFailure records one failing check.
type verifyFailure struct {
	kind    string // "build", "source", "order"
	message string
	diff    string // populated when --diff is set and kind=="build"
}

// runVerify handles the `dtrules verify [path]` command.
func (c *CLI) runVerify(args []string) int {
	opts := &verifyOptions{path: "."}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--diff":
			opts.diff = true
		case "--strict":
			opts.strict = true
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
				c.printVerifyUsage()
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

	if !dirExists(xmlDir) && !dirExists(excelDir) {
		if opts.xmlDir != "" || opts.excelDir != "" {
			if !dirExists(xmlDir) {
				fmt.Fprintf(os.Stderr, "ERROR: could not find xml directory\n  Tried: %s\n  Use --xml-dir <path> or declare <xml_dir> in DTRules.xml.\n", xmlDir)
			}
			if !dirExists(excelDir) {
				fmt.Fprintf(os.Stderr, "ERROR: could not find excel directory\n  Tried: %s\n  Use --excel-dir <path> or declare <excel_dir> in DTRules.xml.\n", excelDir)
			}
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: no xml/ or excel/ directory found in %s\n", absPath)
		}
		return 1
	}

	var failures []verifyFailure

	// Check 0: unique table names (compile-time gate for #722 `-N` markers).
	if dirExists(xmlDir) {
		dupFindings, err := checkNoDupMarkers(xmlDir)
		if err != nil {
			failures = append(failures, verifyFailure{kind: "dup", message: fmt.Sprintf("dup scan error: %v", err)})
		}
		for _, f := range dupFindings {
			failures = append(failures, verifyFailure{kind: "dup", message: strings.TrimRight(f.Message(), "\n")})
		}
	}

	// Check 1: build idempotency
	buildFails := checkBuildIdempotency(absPath, xmlDir, excelDir, opts)
	failures = append(failures, buildFails...)

	// Check 2: <source> header validity
	if dirExists(xmlDir) {
		sourceFails := checkSourceHeaders(xmlDir, excelDir, opts.strict)
		failures = append(failures, sourceFails...)
	}

	// Check 3: NNN_ prefix ordering vs workbook sheet order
	if dirExists(xmlDir) && dirExists(excelDir) {
		orderFails := checkPrefixOrdering(xmlDir, excelDir)
		failures = append(failures, orderFails...)
	}

	// Check 4: filename suffix matches sheet content
	if dirExists(excelDir) {
		suffixFails := checkSuffixContentConsistency(excelDir)
		failures = append(failures, suffixFails...)
	}

	// Check 5: every decision-table project has an Excel representation.
	// Catches the contract violation where rules were authored straight
	// into XML without ever building the Excel system-of-record.
	if dirExists(xmlDir) {
		failures = append(failures, checkExcelPresence(xmlDir, excelDir)...)
	}

	// Check 6: no decision table depends on external/undefined business
	// logic — undefined `perform` targets, EDD fields the schema doesn't
	// declare, or operators the engine doesn't implement.
	if dirExists(xmlDir) {
		failures = append(failures, checkExternalRefs(xmlDir)...)
	}

	if len(failures) == 0 {
		fmt.Printf("verified: %s is consistent with its Excel source\n", absPath)
		return 0
	}

	fmt.Fprintf(os.Stderr, "verify FAILED: %d issue(s) found in %s\n\n", len(failures), absPath)
	for _, f := range failures {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", f.kind, f.message)
		if f.diff != "" {
			fmt.Fprintln(os.Stderr, f.diff)
		}
	}
	return 1
}

// checkBuildIdempotency copies the project to a temp dir, runs the build
// pipeline, then compares excel/ and xml/ byte-for-byte.
func checkBuildIdempotency(projectDir, xmlDir, excelDir string, opts *verifyOptions) []verifyFailure {
	tmpDir, err := os.MkdirTemp("", "dtrules-verify-*")
	if err != nil {
		return []verifyFailure{{kind: "build", message: fmt.Sprintf("failed to create temp dir: %v", err)}}
	}
	defer os.RemoveAll(tmpDir)

	// Copy project into tmp
	if err := copyDir(projectDir, tmpDir); err != nil {
		return []verifyFailure{{kind: "build", message: fmt.Sprintf("failed to copy project: %v", err)}}
	}

	// copyDir places the project under tmpDir/<basename>, and xml/ and excel/
	// are not always spelled that way — a project's DTRules.xml can point
	// elsewhere, which is why this function is handed xmlDir and excelDir
	// rather than guessing. Mirror the real layout instead of assuming.
	//
	// Both were previously hardcoded as tmpDir/xml and tmpDir/excel, one level
	// above where the copy lands. Nothing existed at those paths, so the
	// rebuild below was skipped and the comparison loop skipped both trees:
	// this gate has never compared anything, and reported "consistent with its
	// Excel source" for every project it was ever run on (#1010).
	copyRoot := filepath.Join(tmpDir, filepath.Base(projectDir))
	tmpXML := relocate(projectDir, copyRoot, xmlDir)
	tmpExcel := relocate(projectDir, copyRoot, excelDir)

	// A gate that cannot see its inputs must say so rather than pass. Silently
	// skipping is what made this check inert for its entire existence.
	if !dirExists(tmpXML) {
		return []verifyFailure{{kind: "build", message: fmt.Sprintf(
			"internal: copied XML tree not found at %s — the idempotency check cannot run", tmpXML)}}
	}

	// Run the build pipeline on the copy (always Excel-authored: Excel→XML)
	if dirExists(tmpExcel) {
		syncOpts := sync.DefaultOptions()
		// FORCE Excel→XML. Detection cannot serve this check: hand-editing
		// the XML is exactly what makes it newer, so detection answers
		// XMLToExcel and exports the edit INTO Excel instead of letting
		// Excel overwrite it. The rebuild then matches the edit and verify
		// passes — the gate could never fail for the one thing it exists to
		// catch (#1010).
		syncOpts.ForceDirection = sync.ExcelToXML
		syncer := sync.NewSyncerWithOptions(tmpXML, tmpExcel, syncOpts)
		syncer.SetUseCombinedWorkbooks(true)

		// Same constructor as `build`: verify runs the build pipeline on a
		// copy, so an importer without the EL compiler would have it
		// verifying a pipeline nobody runs, and passing (#929).
		importer := newWorkbookImporter(tmpXML)
		syncer.SetWorkbookImporter(&workbookImporterAdapter{impl: importer})

		exporter := excel.NewWorkbookExporter()
		syncer.SetExporter(&workbookExporterAdapter{impl: exporter})

		_ = os.MkdirAll(tmpXML, 0755)
		result, err := syncer.SyncAll()
		if err != nil {
			return []verifyFailure{{kind: "build", message: fmt.Sprintf("build pipeline failed: %v", err)}}
		}
		for _, e := range result.Errors {
			// Non-fatal: record but continue
			_ = e
		}
	}

	// Compare the two trees
	var failures []verifyFailure

	for _, sub := range []struct{ orig, built string }{
		{xmlDir, tmpXML},
		{excelDir, tmpExcel},
	} {
		if !dirExists(sub.orig) || !dirExists(sub.built) {
			continue
		}
		diffs, err := compareDirectories(sub.orig, sub.built)
		if err != nil {
			failures = append(failures, verifyFailure{kind: "build", message: fmt.Sprintf("compare error: %v", err)})
			continue
		}
		for _, d := range diffs {
			rel, _ := filepath.Rel(projectDir, filepath.Join(sub.orig, d.path))
			msg := fmt.Sprintf("%s: %s", rel, d.reason)
			df := ""
			if opts.diff && d.diffText != "" {
				df = d.diffText
			}
			failures = append(failures, verifyFailure{kind: "build", message: msg, diff: df})
		}
	}

	return failures
}

type fileDiff struct {
	path     string
	reason   string
	diffText string
}

// compareDirectories compares two directory trees byte-for-byte.
// Only considers files present in orig; extra files in built are ignored.
func compareDirectories(orig, built string) ([]fileDiff, error) {
	var diffs []fileDiff

	err := filepath.WalkDir(orig, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(orig, path)
		// Skip manifest and temp Excel files
		if filepath.Base(rel) == ".sync-manifest.json" {
			return nil
		}
		if strings.HasPrefix(filepath.Base(rel), "~$") {
			return nil
		}

		builtPath := filepath.Join(built, rel)
		if _, err := os.Stat(builtPath); os.IsNotExist(err) {
			diffs = append(diffs, fileDiff{
				path:   rel,
				reason: "missing after build",
			})
			return nil
		}

		origData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		builtData, err := os.ReadFile(builtPath)
		if err != nil {
			return err
		}

		if !bytes.Equal(origData, builtData) {
			diff := simpleDiff(rel, origData, builtData)
			diffs = append(diffs, fileDiff{
				path:     rel,
				reason:   "content differs from build output",
				diffText: diff,
			})
		}

		return nil
	})

	return diffs, err
}

// simpleDiff produces a rudimentary unified-style diff for two byte slices.
func simpleDiff(name string, a, b []byte) string {
	aLines := strings.Split(string(a), "\n")
	bLines := strings.Split(string(b), "\n")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- committed/%s\n", name))
	sb.WriteString(fmt.Sprintf("+++ built/%s\n", name))

	maxCtx := 3
	type change struct {
		lineA int
		lineB int
		kind  string // "add", "del", "chg"
		old   string
		new   string
	}
	var changes []change

	// Simple LCS-free diff: find first divergence and last divergence
	minLen := len(aLines)
	if len(bLines) < minLen {
		minLen = len(bLines)
	}

	first := -1
	for i := 0; i < minLen; i++ {
		if aLines[i] != bLines[i] {
			first = i
			break
		}
	}
	if first == -1 {
		if len(aLines) != len(bLines) {
			first = minLen
		}
	}

	if first == -1 {
		return ""
	}

	// Show context around first change only (keep output bounded)
	start := first - maxCtx
	if start < 0 {
		start = 0
	}
	end := first + maxCtx + 1
	if end > minLen {
		end = minLen
	}

	sb.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", start+1, start+1))
	for i := start; i < first; i++ {
		sb.WriteString(" " + aLines[i] + "\n")
	}
	if first < len(aLines) {
		sb.WriteString("-" + aLines[first] + "\n")
	}
	if first < len(bLines) {
		sb.WriteString("+" + bLines[first] + "\n")
	}
	for i := first + 1; i < end && i < len(aLines) && i < len(bLines); i++ {
		if aLines[i] == bLines[i] {
			sb.WriteString(" " + aLines[i] + "\n")
		}
	}

	_ = changes
	return sb.String()
}

// xmlSourceCheck holds what we parse from an XML decision table file.
type xmlSourceCheck struct {
	// From <source>
	RelativePath string
	FileName     string
	SheetNumber  int
	// From <xls_file> (legacy)
	XLSFile string
}

// checkSourceHeaders validates that XML files have valid source references.
func checkSourceHeaders(xmlDir, excelDir string, strict bool) []verifyFailure {
	var failures []verifyFailure

	err := filepath.WalkDir(xmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_dt.xml") {
			return nil
		}

		rel, _ := filepath.Rel(xmlDir, path)
		checks, parseErr := parseXMLSourceHeaders(path)
		if parseErr != nil {
			// Can't parse — not a DTRules file
			return nil
		}

		for _, src := range checks {
			if src.RelativePath == "" && src.FileName == "" && src.XLSFile == "" {
				// No source info at all
				if strict {
					failures = append(failures, verifyFailure{
						kind:    "source",
						message: fmt.Sprintf("%s: table has no <source> or <xls_file> header (--strict)", rel),
					})
				}
				continue
			}

			// Determine the referenced Excel file
			excelRef := src.FileName
			if excelRef == "" {
				excelRef = src.XLSFile
			}
			if excelRef == "" && src.RelativePath != "" {
				excelRef = filepath.Base(src.RelativePath)
			}

			if excelRef == "" {
				if strict {
					failures = append(failures, verifyFailure{
						kind:    "source",
						message: fmt.Sprintf("%s: <source> has no identifiable workbook name (--strict)", rel),
					})
				}
				continue
			}

			// Check in both direct and states subdirectory
			found := false
			candidates := []string{
				filepath.Join(excelDir, excelRef),
				filepath.Join(excelDir, src.RelativePath),
			}
			for _, cand := range candidates {
				if _, statErr := os.Stat(cand); statErr == nil {
					found = true
					break
				}
			}

			if !found {
				// Also do a recursive search for the filename
				_ = filepath.WalkDir(excelDir, func(p string, de fs.DirEntry, e error) error {
					if e != nil {
						return nil
					}
					if !de.IsDir() && filepath.Base(p) == excelRef {
						found = true
						return io.EOF
					}
					return nil
				})
			}

			if !found {
				failures = append(failures, verifyFailure{
					kind:    "source",
					message: fmt.Sprintf("%s: references workbook %q which does not exist in excel/", rel, excelRef),
				})
			}
		}

		return nil
	})

	if err != nil {
		failures = append(failures, verifyFailure{kind: "source", message: fmt.Sprintf("walk error: %v", err)})
	}

	return failures
}

// parseXMLSourceHeaders parses decision table XML and extracts source metadata.
func parseXMLSourceHeaders(path string) ([]xmlSourceCheck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type sourceElem struct {
		RelativePath string `xml:"relative_path"`
		FileName     string `xml:"file_name"`
		SheetNumber  int    `xml:"sheet_number"`
	}
	type tableElem struct {
		Source  *sourceElem `xml:"source"`
		XLSFile string      `xml:"xls_file"`
	}
	type root struct {
		Tables []tableElem `xml:"decision_table"`
	}

	var r root
	if err := xml.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	var checks []xmlSourceCheck
	for _, t := range r.Tables {
		c := xmlSourceCheck{XLSFile: t.XLSFile}
		if t.Source != nil {
			c.RelativePath = t.Source.RelativePath
			c.FileName = t.Source.FileName
			c.SheetNumber = t.Source.SheetNumber
		}
		checks = append(checks, c)
	}
	return checks, nil
}

// nnnPrefixRe matches filenames like 001_Name_dt.xml or 042_Foo_Bar_dt.xml
var nnnPrefixRe = regexp.MustCompile(`^(\d+)_`)

// checkPrefixOrdering verifies that NNN_ prefixed XML DT files agree with the
// workbook's actual sheet order.
func checkPrefixOrdering(xmlDir, excelDir string) []verifyFailure {
	var failures []verifyFailure

	// Group DT XML files by their workbook (derived from <xls_file> or <source>).
	type dtEntry struct {
		xmlPath     string
		prefix      int    // numeric NNN prefix
		xlsFile     string // referenced workbook filename (base)
		sheetNumber int    // from <source> if present
	}

	var entries []dtEntry

	err := filepath.WalkDir(xmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, "_dt.xml") {
			return nil
		}

		m := nnnPrefixRe.FindStringSubmatch(base)
		if m == nil {
			return nil // no NNN prefix
		}

		prefix, _ := strconv.Atoi(m[1])
		checks, parseErr := parseXMLSourceHeaders(path)
		if parseErr != nil || len(checks) == 0 {
			return nil
		}

		// All tables in a file should share the same workbook
		src := checks[0]
		xls := src.FileName
		if xls == "" {
			xls = src.XLSFile
		}
		if xls == "" && src.RelativePath != "" {
			xls = filepath.Base(src.RelativePath)
		}
		if xls == "" {
			return nil // can't determine workbook
		}

		entries = append(entries, dtEntry{
			xmlPath:     path,
			prefix:      prefix,
			xlsFile:     xls,
			sheetNumber: src.SheetNumber,
		})
		return nil
	})

	if err != nil {
		failures = append(failures, verifyFailure{kind: "order", message: fmt.Sprintf("walk error: %v", err)})
		return failures
	}

	// Group entries by workbook
	byWorkbook := make(map[string][]dtEntry)
	for _, e := range entries {
		byWorkbook[e.xlsFile] = append(byWorkbook[e.xlsFile], e)
	}

	// For each workbook with multiple entries, check that NNN prefix order
	// matches sheet order (either from <source> sheet_number or workbook sheet list).
	for wbFile, wbEntries := range byWorkbook {
		if len(wbEntries) < 2 {
			continue
		}

		// Sort entries by NNN prefix
		prefixSorted := make([]dtEntry, len(wbEntries))
		copy(prefixSorted, wbEntries)
		sort.Slice(prefixSorted, func(i, j int) bool {
			return prefixSorted[i].prefix < prefixSorted[j].prefix
		})

		// Check if sheet_number is available
		hasSheetNumbers := prefixSorted[0].sheetNumber > 0
		if hasSheetNumbers {
			// Sort by sheet_number and compare ordering
			sheetSorted := make([]dtEntry, len(wbEntries))
			copy(sheetSorted, wbEntries)
			sort.Slice(sheetSorted, func(i, j int) bool {
				return sheetSorted[i].sheetNumber < sheetSorted[j].sheetNumber
			})

			for i := range prefixSorted {
				if filepath.Base(prefixSorted[i].xmlPath) != filepath.Base(sheetSorted[i].xmlPath) {
					failures = append(failures, verifyFailure{
						kind: "order",
						message: fmt.Sprintf(
							"workbook %s: NNN_ prefix order disagrees with sheet order at position %d"+
								" (prefix order: %s, sheet order: %s)",
							wbFile, i+1,
							filepath.Base(prefixSorted[i].xmlPath),
							filepath.Base(sheetSorted[i].xmlPath),
						),
					})
				}
			}
			continue
		}

		// No sheet_number: try to read workbook sheet order directly
		excelPath := findExcelFile(excelDir, wbFile)
		if excelPath == "" {
			continue
		}

		f, err := excelize.OpenFile(excelPath)
		if err != nil {
			continue
		}
		sheets := f.GetSheetList()
		f.Close()

		// Build sheet-name → position map
		sheetPos := make(map[string]int, len(sheets))
		for i, s := range sheets {
			sheetPos[s] = i
		}

		// Map each entry to its sheet position using its table name (strip NNN_ prefix and _dt suffix)
		type entryWithPos struct {
			dtEntry
			pos int
		}
		var withPos []entryWithPos
		for _, e := range wbEntries {
			base := strings.TrimSuffix(filepath.Base(e.xmlPath), "_dt.xml")
			// Strip NNN_ prefix
			if m := nnnPrefixRe.FindStringSubmatch(base); m != nil {
				base = base[len(m[0]):]
			}
			pos := -1
			for sheetName, p := range sheetPos {
				// Compare case-insensitively and ignoring spaces/underscores
				if sheetNameMatches(sheetName, base) {
					pos = p
					break
				}
			}
			withPos = append(withPos, entryWithPos{e, pos})
		}

		// Sort by prefix
		sort.Slice(withPos, func(i, j int) bool {
			return withPos[i].prefix < withPos[j].prefix
		})

		// Check that sheet positions are non-decreasing
		for i := 1; i < len(withPos); i++ {
			if withPos[i-1].pos >= 0 && withPos[i].pos >= 0 && withPos[i].pos < withPos[i-1].pos {
				failures = append(failures, verifyFailure{
					kind: "order",
					message: fmt.Sprintf(
						"workbook %s: NNN_ prefix order disagrees with sheet order: "+
							"%s (prefix %d) should come after %s (prefix %d) but sheet order disagrees",
						wbFile,
						filepath.Base(withPos[i].xmlPath), withPos[i].prefix,
						filepath.Base(withPos[i-1].xmlPath), withPos[i-1].prefix,
					),
				})
			}
		}
	}

	return failures
}

// sheetNameMatches checks if a sheet name corresponds to a base name
// (both stripped of underscores/spaces and compared case-insensitively).
func sheetNameMatches(sheetName, base string) bool {
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "_", "")
		return s
	}
	return normalize(sheetName) == normalize(base)
}

// findExcelFile searches excelDir recursively for a file named wbFile.
func findExcelFile(excelDir, wbFile string) string {
	var found string
	_ = filepath.WalkDir(excelDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Base(path) == wbFile {
			found = path
			return io.EOF
		}
		return nil
	})
	return found
}

// checkSuffixContentConsistency verifies that workbooks with a _dt, _edd, or _map
// suffix contain only the expected sheet type. A mismatch is a verify failure.
func checkSuffixContentConsistency(excelDir string) []verifyFailure {
	var failures []verifyFailure

	importer := excel.NewWorkbookImporter()

	_ = filepath.WalkDir(excelDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasPrefix(d.Name(), "~$") {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".xlsx") {
			return nil
		}

		suffix, hasSuffix := excel.ArtifactTypeFromFilename(path)
		if !hasSuffix {
			return nil // mixed-artifact workbook — no constraint
		}

		hasDT, _ := importer.HasDTSheet(path)
		hasEDD, _ := importer.HasEDDSheet(path)
		hasMAP, _ := importer.HasMAPSheet(path)

		rel, _ := filepath.Rel(excelDir, path)
		msg := excel.ArtifactTypeMismatch(suffix, hasDT, hasEDD, hasMAP)
		if msg != "" {
			failures = append(failures, verifyFailure{
				kind:    "suffix",
				message: fmt.Sprintf("%s: %s", rel, msg),
			})
		}
		return nil
	})

	return failures
}

// checkExcelPresence fails when a project carries decision-table or EDD
// XML but has no Excel workbook to back it. Excel is the system of record;
// XML is a generated artifact. A project with rule XML and no `.xlsx`
// anywhere was authored by writing XML directly — exactly the bypass the
// authoring contract forbids — so it fails the gate rather than committing.
//
// The check is project-level and coarse on purpose: per-table workbook
// references are already validated by checkSourceHeaders, and build drift
// by checkBuildIdempotency. This catches only the gross "no Excel at all"
// case, which neither of those flags when the XML has no <source> headers.
func checkExcelPresence(xmlDir, excelDir string) []verifyFailure {
	if !hasRuleXML(xmlDir) {
		return nil // not a rule project — nothing to back with Excel
	}
	if dirExists(excelDir) && hasWorkbook(excelDir) {
		return nil
	}
	return []verifyFailure{{
		kind: "excel",
		message: "project has decision-table/EDD XML but no Excel workbook in excel/ — " +
			"Excel is the system of record; author through Excel (dtrules build) or the " +
			"authoring API (dtrules table/edd), not by writing XML directly",
	}}
}

// hasRuleXML reports whether xmlDir contains any *_dt.xml or *_edd.xml file.
func hasRuleXML(xmlDir string) bool {
	found := false
	_ = filepath.WalkDir(xmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, "_dt.xml") || strings.HasSuffix(name, "_edd.xml") {
			found = true
			return io.EOF
		}
		return nil
	})
	return found
}

// hasWorkbook reports whether excelDir contains any non-temp .xlsx file.
func hasWorkbook(excelDir string) bool {
	found := false
	_ = filepath.WalkDir(excelDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "~$") {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(name), ".xlsx") {
			found = true
			return io.EOF
		}
		return nil
	})
	return found
}

// checkExternalRefs fails when any decision table depends on a symbol the
// project doesn't define: an undefined `perform` target, an EDD field the
// schema never declares, or an operator absent from the registry. A table
// that leans on logic defined outside the project isn't self-contained and
// must not commit.
func checkExternalRefs(xmlDir string) []verifyFailure {
	isOperator := func(tok string) bool {
		_, ok := operators.GetByString(tok)
		return ok
	}
	findings, err := analysis.AnalyzeExternalRefs(xmlDir, isOperator)
	if err != nil {
		return []verifyFailure{{kind: "external", message: fmt.Sprintf("external-reference scan error: %v", err)}}
	}
	var failures []verifyFailure
	for _, f := range findings {
		failures = append(failures, verifyFailure{kind: "external", message: f.String()})
	}
	return failures
}

// copyDir copies src directory tree into dst, preserving structure.
// dst is the parent directory; src's contents are copied under dst/<basename(src)>.
func copyDir(src, dst string) error {
	srcBase := filepath.Base(src)
	dstBase := filepath.Join(dst, srcBase)

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dstBase, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		// Skip manifest files and temp Excel files to avoid stale timestamps
		base := filepath.Base(path)
		if base == ".sync-manifest.json" || strings.HasPrefix(base, "~$") {
			return nil
		}

		return copyFile(path, target)
	})
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func (c *CLI) printVerifyUsage() {
	fmt.Println(`Usage: dtrules verify [path] [options]

Gates CI/pre-commit by checking that committed XML matches what dtrules build
from the committed Excel would produce.

Arguments:
  path              Project directory to verify (default: .)

Options:
  --xml-dir <path>    Override XML directory (relative to project root or absolute)
  --excel-dir <path>  Override Excel directory (relative to project root or absolute)
  --diff              On failure, show diff between committed and build output
  --strict            Also fail on warnings (e.g. missing <source> header)

Directory resolution (highest to lowest priority):
  1. --xml-dir / --excel-dir flags
  2. <xml_dir> / <excel_dir> elements in DTRules.xml
  3. Default: xml/ and excel/ relative to the project root

Exit codes:
  0  All checks passed
  1  One or more checks failed

Checks performed:
  build     Running dtrules build would not change any file in excel/ or xml/
  source    Every XML artifact has a valid <source> or <xls_file> reference
  order     NNN_ prefix ordering agrees with workbook sheet order
  excel     A rule project has an Excel system-of-record workbook
  external  No table depends on an undefined table, EDD field, or operator

Examples:
  dtrules verify
  dtrules verify ./sampleprojects/TaxReturn
  dtrules verify --diff --strict
  dtrules verify --xml-dir pkg/dtrules/rules --excel-dir pkg/dtrules/excel /path/to/project`)
}

// relocate maps a path under root onto the same position under newRoot.
// Used to find the copied xml/ and excel/ trees whatever they are named,
// rather than assuming the conventional spelling (#1010).
func relocate(root, newRoot, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Outside the project: not something the copy contains.
		return filepath.Join(newRoot, filepath.Base(path))
	}
	return filepath.Join(newRoot, rel)
}
