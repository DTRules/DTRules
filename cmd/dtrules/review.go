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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/analysis"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// reviewReport is the project-wide Full Review output (#768). Reads of
// .dtrules/last-review.json (the persisted form) and the
// project_full_review MCP tool both return this exact shape, so field
// names and JSON tags are part of the public contract.
//
// Errors gate deployment; warnings never do. The split is the whole
// reason this report exists separately from project_validate.
type reviewReport struct {
	ProjectHash  string                  `json:"project_hash"`
	Timestamp    string                  `json:"timestamp"` // RFC3339
	Errors       []reviewError           `json:"errors"`
	Warnings     []decisiontable.Warning `json:"warnings"`
	EDDWarnings  []analysis.EDDWarning   `json:"edd_warnings"`
	ContextHints []analysis.ContextSuggestion `json:"context_hints"`
	Diagnostics  []authoring.Diagnostic  `json:"diagnostics"`
	Structure    interface{}             `json:"structure"`
	ELCompliance interface{}             `json:"el_compliance"`
	Passed       bool                    `json:"passed"`
}

// reviewError tags every hard error with its source so consumers can
// surface them grouped (parse / EL compile / structure / EL compliance /
// load-time). Category strings are stable; UIs that route on category
// can rely on the constants here.
type reviewError struct {
	Category string `json:"category"`
	File     string `json:"file,omitempty"`
	Table    string `json:"table,omitempty"`
	Message  string `json:"message"`
}

const (
	reviewCategoryStructure    = "structure"
	reviewCategoryELCompliance = "el_compliance"
	reviewCategoryDiagnostic   = "diagnostic"
	reviewCategoryLoad         = "load"
)

// runFullReview is the entry point used by both `dtrules review` and the
// project_full_review MCP tool. It collects every check, persists the
// report, and returns it.
//
// Order:
//  1. Structure validation (sync.ValidateProjectStructure) → errors.
//  2. EL compliance (sync.ValidateELCompliance) → errors per-file.
//  3. Project load + load-time diagnostics (authoring.Project.Diagnostics)
//     → errors when a diagnostic represents an unresolved duplicate-name.
//  4. Per-table optimizer pass (#762/#763 via Analyze, #765/#766 via
//     AnalyzeCompiledTable) → warnings.
//  5. EDD-unused (analysis.AnalyzeEDDUsage) → warnings.
//  6. Table call graph (analysis.AnalyzeTableCallGraph) → warnings
//     when `perform <Name>` references a table that isn't defined.
//     The runtime would fail with "table not found" if the offending
//     rule fires, but the reference may sit on an unreachable branch;
//     warning-not-error keeps the finding visible without gating
//     deployment.
//
// `passed = (len(errors) == 0)`. Warnings never affect passed. The
// caller decides whether to surface the report as a CLI output or an
// MCP response; both go through this function.
func runFullReview(projectPath string) (*reviewReport, error) {
	rep := &reviewReport{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Errors:      []reviewError{},
		Warnings:    []decisiontable.Warning{},
		EDDWarnings:  []analysis.EDDWarning{},
		ContextHints: []analysis.ContextSuggestion{},
		Diagnostics:  []authoring.Diagnostic{},
	}

	// 1. Structure validation.
	structRes, err := validateStructure(projectPath)
	if err != nil {
		rep.Errors = append(rep.Errors, reviewError{
			Category: reviewCategoryStructure,
			Message:  err.Error(),
		})
	}
	if structRes != nil {
		rep.Structure = map[string]interface{}{
			"valid":    structRes.IsValid(),
			"errors":   structErrors(structRes),
			"warnings": structWarnings(structRes),
		}
		if !structRes.IsValid() {
			for _, e := range structErrors(structRes) {
				rep.Errors = append(rep.Errors, reviewError{
					Category: reviewCategoryStructure,
					Message:  e,
				})
			}
		}
	}

	// Resolve XML directory for the rest of the checks.
	xmlDir := ""
	if structRes != nil && structRes.Structure != nil {
		xmlDir = structRes.Structure.XMLDir
	}
	if xmlDir == "" {
		xmlDir = filepath.Join(projectPath, "xml")
	}

	// 2. EL compliance (file-level: which files still have legacy
	// postfix-only DSL). Each non-compliant file becomes an error
	// because deployment cannot proceed with legacy DSL.
	elRes, elErr := sync.ValidateELCompliance(xmlDir)
	if elErr != nil {
		rep.Errors = append(rep.Errors, reviewError{
			Category: reviewCategoryELCompliance,
			Message:  elErr.Error(),
		})
	}
	if elRes != nil {
		rep.ELCompliance = map[string]interface{}{
			"compliant":         elRes.IsCompliant(),
			"legacy_files":      elRes.LegacyFiles,
			"compliant_files":   elRes.CompliantFiles,
			"needs_compilation": elRes.NeedsCompilationFiles,
		}
		for _, f := range elRes.LegacyFiles {
			rep.Errors = append(rep.Errors, reviewError{
				Category: reviewCategoryELCompliance,
				File:     f,
				Message:  "file still authored as legacy postfix — author EL DSL before shipping",
			})
		}
	}

	// 3. Project-load diagnostics. Open the project once and reuse for
	// the per-table optimizer pass.
	p, openErr := authoring.OpenProject(projectPath)
	if openErr != nil {
		rep.Errors = append(rep.Errors, reviewError{
			Category: reviewCategoryLoad,
			Message:  openErr.Error(),
		})
		// Without a project we can't run the optimizer or EDD checks.
		rep.ProjectHash = hashProject(xmlDir)
		rep.Passed = len(rep.Errors) == 0
		return rep, persistReview(projectPath, rep)
	}

	for _, d := range p.Diagnostics() {
		rep.Diagnostics = append(rep.Diagnostics, d)
		// A diagnostic with kind="duplicate_table" left unresolved is
		// an error per #722's contract — the `-N` suffix means the
		// loader had to rename a collision and the author hasn't
		// confirmed which name to keep.
		if strings.Contains(strings.ToLower(d.Kind), "duplicate") {
			rep.Errors = append(rep.Errors, reviewError{
				Category: reviewCategoryDiagnostic,
				Table:    d.OriginalName,
				File:     d.File,
				Message:  d.Message,
			})
		}
	}

	// 4. Per-table optimizer pass — matrix-driven (#761/#762/#763) on
	// every authoring table. The tree-driven checks (#765/#766) need a
	// compiled RuleSet and currently pay too high a cost on large
	// projects; reintroducing them awaits an opt-in flag or shared
	// compiled state that makes paying that cost practical.
	for _, name := range p.Tables() {
		t := p.Table(name)
		if t == nil {
			continue
		}
		warns := analyzeAuthoringTable(t)
		rep.Warnings = append(rep.Warnings, warns...)
	}

	// 5. EDD-unused warnings.
	eddWarns, eddErr := analysis.AnalyzeEDDUsage(xmlDir)
	if eddErr == nil {
		rep.EDDWarnings = append(rep.EDDWarnings, eddWarns...)
	}

	// 5b. Context-push hints (advisory): entities referenced with a
	// qualifier many times that could be pushed onto an entry table's
	// context so their fields read unqualified. Never an error.
	if hints, hintErr := analysis.SuggestContextPushes(xmlDir); hintErr == nil {
		rep.ContextHints = append(rep.ContextHints, hints...)
	}

	// 6. Table call graph — orphan `perform <Name>` calls surface as
	// warnings (not errors). They're real bugs in the sense that the
	// runtime would fail with "table not found" if the offending rule
	// fires, but a project can have orphan references in branches
	// that aren't reachable from any actual entry table or that are
	// gated by conditions never satisfied in production. Demoting
	// from error to warning keeps the finding visible without
	// gating deployment; consumers can promote to error via their
	// own policy if they want stricter behavior.
	graph, graphErr := analysis.AnalyzeTableCallGraph(xmlDir)
	if graphErr == nil && graph != nil {
		for _, o := range graph.OrphanCalls {
			w := decisiontable.Warning{
				Table:        o.Caller,
				Column:       0,
				ConditionRow: -1,
				Kind:         "orphan perform target",
				Reason: fmt.Sprintf("calls undefined table %q (in %s) — runtime would fail with table-not-found if this rule fires",
					o.Callee, o.DTFile),
			}
			rep.Warnings = append(rep.Warnings, w)
		}
	}

	rep.ProjectHash = hashProject(xmlDir)
	rep.Passed = len(rep.Errors) == 0
	return rep, persistReview(projectPath, rep)
}

// hashProject returns a stable sha256 over every *_dt.xml and *_edd.xml
// file under xmlDir. The Full Review report carries this hash so the
// deployment gate (#768) can detect whether the project has changed
// since the last review.
//
// Hash inputs: each tracked file's path (relative to xmlDir) plus its
// content, sorted by path. Anything outside xmlDir (build artefacts,
// .dtrules/, .git/) is intentionally excluded so the hash only tracks
// authoritative rule content.
func hashProject(xmlDir string) string {
	type entry struct {
		path    string
		content []byte
	}
	var entries []entry
	_ = filepath.WalkDir(xmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, "_dt.xml") && !strings.HasSuffix(name, "_edd.xml") {
			return nil
		}
		rel, _ := filepath.Rel(xmlDir, path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		entries = append(entries, entry{path: rel, content: data})
		return nil
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.path))
		h.Write([]byte{0})
		h.Write(e.content)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// persistReview writes the report to `<project>/.dtrules/last-review.json`.
// The deployment gate reads this file and recomputes the project hash to
// decide whether deployment can proceed.
func persistReview(projectPath string, rep *reviewReport) error {
	dir := filepath.Join(projectPath, ".dtrules")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last-review.json"), data, 0o644)
}

// runReview is the `dtrules review` CLI handler. Default output is the
// report JSON on stdout, with non-zero exit when `passed=false`. The
// canonical output lives at `<project>/.dtrules/last-review.json`
// regardless.
//
// Errors gate deployment via `dtrules build --require-review`; warnings
// surface but never fail the command. A passing review with warnings
// still exits 0.
func (c *CLI) runReview(args []string) int {
	projectPath := "."
	projectFromFlag := false
	parsedArgs := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--project":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--project requires a value")
				return 1
			}
			projectPath = args[i+1]
			projectFromFlag = true
			i++
		case strings.HasPrefix(a, "--project="):
			projectPath = strings.TrimPrefix(a, "--project=")
			projectFromFlag = true
		default:
			parsedArgs = append(parsedArgs, a)
		}
	}
	// Positional argument is the project path (#788). The flag still
	// wins if both were passed, so `--project X Y` keeps the documented
	// flag-takes-precedence behavior; bare `dtrules review X` no longer
	// silently runs against CWD.
	if !projectFromFlag && len(parsedArgs) > 0 {
		projectPath = parsedArgs[0]
	}

	rep, err := runFullReview(projectPath)
	if err != nil {
		// Persist may have failed; the report itself is still returned
		// and worth showing.
		fmt.Fprintf(os.Stderr, "warning: could not persist review: %v\n", err)
	}
	out, marshalErr := json.MarshalIndent(rep, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", marshalErr)
		return 1
	}
	fmt.Println(string(out))
	if rep == nil || !rep.Passed {
		return 1
	}
	return 0
}

// enforceReviewGate implements the deployment gate from #768. Called
// from `dtrules build --require-review`. The gate refuses unless:
//
//   - .dtrules/last-review.json exists and is parseable;
//   - its project_hash matches the current XML state;
//   - its timestamp is within maxAge of now;
//   - its passed field is true (errors empty).
//
// Returns 0 on success or a non-zero exit code. Writes a clear
// human-readable explanation to stderr on refusal — the gate is the
// last thing standing between a rule author and production, so the
// failure message has to point at the next step.
func enforceReviewGate(projectPath, xmlDir string, maxAge time.Duration) int {
	rep, err := readLastReview(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Deployment gate: %v\n", err)
		return 1
	}
	currentHash := hashProject(xmlDir)
	if rep.ProjectHash != currentHash {
		fmt.Fprintln(os.Stderr, "Deployment gate: project has changed since last review.")
		fmt.Fprintf(os.Stderr, "  Cached hash:  %s\n", rep.ProjectHash)
		fmt.Fprintf(os.Stderr, "  Current hash: %s\n", currentHash)
		fmt.Fprintln(os.Stderr, "  Run `dtrules review` to refresh.")
		return 1
	}
	ts, parseErr := time.Parse(time.RFC3339, rep.Timestamp)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Deployment gate: last-review.json has unparseable timestamp %q; re-run `dtrules review`.\n", rep.Timestamp)
		return 1
	}
	if age := time.Since(ts); age > maxAge {
		fmt.Fprintf(os.Stderr, "Deployment gate: cached review is %s old (limit %s); re-run `dtrules review`.\n",
			age.Truncate(time.Second), maxAge)
		return 1
	}
	if !rep.Passed {
		fmt.Fprintf(os.Stderr, "Deployment gate: review reported %d error(s); fix them before deploying:\n",
			len(rep.Errors))
		for _, e := range rep.Errors {
			loc := ""
			switch {
			case e.Table != "" && e.File != "":
				loc = fmt.Sprintf(" [%s in %s]", e.Table, e.File)
			case e.Table != "":
				loc = fmt.Sprintf(" [%s]", e.Table)
			case e.File != "":
				loc = fmt.Sprintf(" [%s]", e.File)
			}
			fmt.Fprintf(os.Stderr, "  %s: %s%s\n", e.Category, e.Message, loc)
		}
		return 1
	}
	return 0
}

// readLastReview returns the persisted Full Review report, or an error
// describing why it isn't usable. The deployment gate calls this.
func readLastReview(projectPath string) (*reviewReport, error) {
	path := filepath.Join(projectPath, ".dtrules", "last-review.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no full review on file (%s): run `dtrules review` first", path)
	}
	var rep reviewReport
	if err := json.Unmarshal(data, &rep); err != nil {
		return nil, fmt.Errorf("last-review.json is not valid: %w", err)
	}
	return &rep, nil
}
