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

// JSON-first CLI surface for decision tables and the EDD.
//
// Every command emits JSON on stdout; every non-zero exit writes a JSON
// error record to stderr with the shape defined by jsonError.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
)

// parseTableFlags extracts --file/--range/--reason from the argument list,
// returning them plus the remaining (positional) args.
func parseTableFlags(rest []string) (file, rng, reason string, out []string) {
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--file":
			if i+1 < len(rest) {
				file = rest[i+1]
				i++
			}
		case "--range":
			if i+1 < len(rest) {
				rng = rest[i+1]
				i++
			}
		case "--reason":
			if i+1 < len(rest) {
				reason = rest[i+1]
				i++
			}
		default:
			out = append(out, rest[i])
		}
	}
	return file, rng, reason, out
}

// parseRange parses a "LO-HI" range string.
func parseRange(s string) (lo, hi int, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("range must be LO-HI (e.g. 3000-3500)")
	}
	if lo, err = strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
		return 0, 0, fmt.Errorf("range lo: %w", err)
	}
	if hi, err = strconv.Atoi(strings.TrimSpace(parts[1])); err != nil {
		return 0, 0, fmt.Errorf("range hi: %w", err)
	}
	return lo, hi, nil
}

// ensureFile makes file ready to receive a table: validates a matching range if
// the file exists, or creates it (requiring range + reason) if it is new.
func ensureFile(p *authoring.Project, file, rng, reason string) error {
	if p.HasFile(file) {
		if rng != "" {
			lo, hi, err := parseRange(rng)
			if err != nil {
				return err
			}
			if clo, chi, _ := p.RangeOf(file); clo != lo || chi != hi {
				return fmt.Errorf("range [%d-%d] differs from file's [%d-%d]; use set-range to change it", lo, hi, clo, chi)
			}
		}
		return nil
	}
	if rng == "" {
		return fmt.Errorf("file %q is new; provide --range LO-HI to create it", file)
	}
	lo, hi, err := parseRange(rng)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("creating file %q requires --reason", file)
	}
	return p.CreateFile(file, lo, hi, reason)
}

// jsonError is the single shape every non-zero exit writes to stderr.
type jsonError struct {
	Error  string `json:"error"`
	Path   string `json:"path,omitempty"`
	Hint   string `json:"hint,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// tableCmdCtx holds the pieces each handler needs.
type tableCmdCtx struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	projectPath string
	// forceOverwriteExcel, when true, bypasses Project.Save's
	// "Excel newer than last export" guard for this command. Set by
	// the `--force-overwrite-excel` CLI flag; the default is false,
	// which preserves human Excel edits by refusing the XML write.
	forceOverwriteExcel bool
	// eddFile names which of the project's EDD files to work on, from
	// --edd-file. Empty is fine when the project has one; with several the
	// command refuses rather than pick.
	eddFile string
}

// emitErr writes a JSON error record to stderr and returns the exit code.
func emitErr(w io.Writer, code int, errKind, path, hint, detail string) int {
	e := jsonError{Error: errKind, Path: path, Hint: hint, Detail: detail}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(e)
	return code
}

// writeJSON emits a value as indented JSON to stdout.
func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// parseProjectFlag pulls --project / -p and --force-overwrite-excel
// out of a flag slice, returning the parsed values and a copy of args
// with those flags removed. Defaults: projectPath=".",
// forceOverwriteExcel=false.
func parseProjectFlag(args []string) (projectPath string, forceOverwriteExcel bool, rest []string) {
	projectPath, _, forceOverwriteExcel, rest = parseProjectFlags(args)
	return projectPath, forceOverwriteExcel, rest
}

// parseProjectFlags is parseProjectFlag plus --edd-file, which names which of
// a project's EDD files to work on.
func parseProjectFlags(args []string) (projectPath, eddFile string, forceOverwriteExcel bool, rest []string) {
	projectPath = "."
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--edd-file" {
			if i+1 < len(args) {
				eddFile = args[i+1]
				i++
				continue
			}
		}
		if args[i] == "--project" || args[i] == "-p" {
			if i+1 < len(args) {
				projectPath = args[i+1]
				i++
				continue
			}
		}
		if args[i] == "--force-overwrite-excel" {
			forceOverwriteExcel = true
			continue
		}
		out = append(out, args[i])
	}
	return projectPath, eddFile, forceOverwriteExcel, out
}

// selectEDD points the project at the EDD file the caller named, and refuses
// to guess when there is more than one and nobody said which.
//
// A project may hold many EDD files. The Project model works on one at a time
// and used to take whichever sorted first, silently, so 51 of CorporateTax's
// 52 were unreachable through the authoring API. Same rule as mappings: when
// there is a choice to make, the caller makes it.
func selectEDD(p *authoring.Project, eddFile string) error {
	if eddFile != "" {
		return p.UseEDDFile(eddFile)
	}
	files := p.EDDFiles()
	if len(files) <= 1 {
		return nil
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	if len(names) > 6 {
		names = append(names[:6], fmt.Sprintf("... and %d more", len(files)-6))
	}
	return fmt.Errorf("this project has %d EDD files (%s); say which with --edd-file <name>",
		len(files), strings.Join(names, ", "))
}

// runTable dispatches `dtrules table ...`.
func (c *CLI) runTable(args []string) int {
	ctx := &tableCmdCtx{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	if len(args) == 0 {
		c.printTableUsage()
		return 1
	}
	sub := args[0]
	rest := args[1:]
	ctx.projectPath, ctx.forceOverwriteExcel, rest = parseProjectFlag(rest)

	switch sub {
	case "list":
		return ctx.tableList()
	case "get":
		return ctx.tableGet(rest)
	case "put":
		return ctx.tablePut(rest)
	case "patch":
		return ctx.tablePatch(rest)
	case "delete":
		return ctx.tableDelete(rest)
	case "files":
		return ctx.tableFiles()
	case "note":
		return ctx.tableNote(rest)
	case "warnings":
		return ctx.tableWarnings(rest)
	case "schema":
		return ctx.tableSchema(rest)
	case "help", "-h", "--help":
		c.printTableUsage()
		return 0
	default:
		return emitErr(ctx.stderr, 1, "invalid_command", "", "known: list|get|put|patch|delete|files|note|warnings|schema",
			fmt.Sprintf("unknown table subcommand %q", sub))
	}
}

// runEDD dispatches `dtrules edd ...`.
func (c *CLI) runEDD(args []string) int {
	ctx := &tableCmdCtx{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	if len(args) == 0 {
		c.printEDDUsage()
		return 1
	}
	sub := args[0]
	rest := args[1:]
	ctx.projectPath, ctx.eddFile, ctx.forceOverwriteExcel, rest = parseProjectFlags(rest)

	switch sub {
	case "get":
		return ctx.eddGet()
	case "put":
		return ctx.eddPut()
	case "patch":
		return ctx.eddPatch()
	case "schema":
		return ctx.eddSchema(rest)
	case "help", "-h", "--help":
		c.printEDDUsage()
		return 0
	default:
		return emitErr(ctx.stderr, 1, "invalid_command", "", "known: get|put|patch|schema",
			fmt.Sprintf("unknown edd subcommand %q", sub))
	}
}

// --- table handlers ---

func (ctx *tableCmdCtx) openProject() (*authoring.Project, int) {
	p, err := authoring.OpenProject(ctx.projectPath)
	if err != nil {
		return nil, emitErr(ctx.stderr, 1, "io_error", "", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	// Thread the --force-overwrite-excel flag through so Save / SaveEDD
	// know whether to bypass the Excel-mtime guard for this command.
	p.OverwriteExcel = ctx.forceOverwriteExcel

	// Honour --edd-file when given. Requiring a choice is the EDD commands'
	// business, not every command's: `table get` works on decision tables and
	// does not care which EDD is loaded, and demanding --edd-file from it
	// broke `dtrules table` outright on the one project with several.
	if ctx.eddFile != "" {
		if err := p.UseEDDFile(ctx.eddFile); err != nil {
			return nil, emitErr(ctx.stderr, 1, "io_error", "", "check the path", err.Error())
		}
	}
	return p, 0
}

// openProjectForEDD is openProject for the commands that read or write the
// EDD, where which file is being operated on is the whole question. A project
// with several and no --edd-file is refused rather than silently served the
// first (#1099).
func (ctx *tableCmdCtx) openProjectForEDD() (*authoring.Project, int) {
	p, code := ctx.openProject()
	if code != 0 {
		return nil, code
	}
	if ctx.eddFile == "" {
		if err := selectEDD(p, ""); err != nil {
			return nil, emitErr(ctx.stderr, 1, "ambiguous_edd", "", "pass --edd-file <name>", err.Error())
		}
	}
	return p, 0
}

func (ctx *tableCmdCtx) tableList() int {
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	names := p.Tables()
	if names == nil {
		names = []string{}
	}
	if err := writeJSON(ctx.stdout, map[string]interface{}{"tables": names}); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

func (ctx *tableCmdCtx) tableGet(rest []string) int {
	name, code := ctx.requireName(rest, "table get <name>")
	if code != 0 {
		return code
	}
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	t := p.Table(name)
	if t == nil {
		return emitErr(ctx.stderr, 1, "not_found", "", "check `dtrules table list`",
			fmt.Sprintf("table %q not found", name))
	}
	// tableToJSON returns a struct, but the response also embeds the
	// authoring-channel warnings so the agent loop (#761) sees structural
	// findings on every read. Empty `warnings` renders as `[]`, never null.
	tj := tableToJSON(t)
	tj.File = p.FileOf(name)
	payload := tableGetResponse{
		TableJSON: tj,
		Warnings:  warningsForJSON(analyzeAuthoringTable(t)),
	}
	if err := writeJSON(ctx.stdout, payload); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

// tableGetResponse adds the authoring-channel warnings array to the
// existing TableJSON shape. Inlined via Go's struct-embedding so JSON
// field names match the bare TableJSON form when warnings is empty.
type tableGetResponse struct {
	TableJSON
	Warnings []decisiontable.Warning `json:"warnings"`
}

func (ctx *tableCmdCtx) tablePut(rest []string) int {
	fileFlag, rngFlag, reasonFlag, rest := parseTableFlags(rest)
	name, code := ctx.requireName(rest, "table put <name> --file <path> [--range LO-HI] [--reason R]")
	if code != 0 {
		return code
	}
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}

	var tj TableJSON
	if code := decodeStdin(ctx, &tj); code != 0 {
		return code
	}
	if tj.Name == "" {
		tj.Name = name
	}
	// File: --file flag wins over a body "file"; range/reason are flags.
	file := fileFlag
	if file == "" {
		file = tj.File
	}

	t := p.Table(name)
	if t == nil {
		// New table — a file is required (no default).
		if strings.TrimSpace(file) == "" {
			return emitErr(ctx.stderr, 1, "invalid_input", "",
				"creating a table requires a file", "provide --file <path> (or \"file\" in the body)")
		}
		if err := ensureFile(p, file, rngFlag, reasonFlag); err != nil {
			return emitErr(ctx.stderr, 1, "invalid_input", "", "", err.Error())
		}
		newT, err := p.AddTable(name, file, "")
		if err != nil {
			return emitErr(ctx.stderr, 1, "invalid_input", "", "", err.Error())
		}
		t = newT
	} else if strings.TrimSpace(file) != "" && p.FileRel(file) != p.FileOf(name) {
		// Existing table whose file changed → move (renumbers into target range).
		if err := ensureFile(p, file, rngFlag, reasonFlag); err != nil {
			return emitErr(ctx.stderr, 1, "invalid_input", "", "", err.Error())
		}
		if strings.TrimSpace(reasonFlag) == "" {
			return emitErr(ctx.stderr, 1, "invalid_input", "", "moving a table requires --reason", "")
		}
		if err := p.MoveTable(name, file, reasonFlag); err != nil {
			return emitErr(ctx.stderr, 1, "invalid_input", "", "", err.Error())
		}
		t = p.Table(name) // re-fetch: MoveTable relocated the underlying record
	} else if rngFlag != "" && strings.TrimSpace(file) != "" {
		// File unchanged but a range was supplied — validate it matches.
		if err := ensureFile(p, file, rngFlag, reasonFlag); err != nil {
			return emitErr(ctx.stderr, 1, "invalid_input", "", "", err.Error())
		}
	}

	if err := tj.ApplyTo(t); err != nil {
		return emitErr(ctx.stderr, 1, "compile_error", "", "an EL expression failed to compile", err.Error())
	}
	if err := p.Save(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOKWithWarnings(ctx, "updated",
		map[string]string{"table": tj.Name, "file": p.FileOf(tj.Name)},
		analyzeAuthoringTable(t))
}

// tableDelete removes a table (reason required for the change log).
func (ctx *tableCmdCtx) tableDelete(rest []string) int {
	_, _, reason, rest := parseTableFlags(rest)
	name, code := ctx.requireName(rest, "table delete <name> --reason R")
	if code != 0 {
		return code
	}
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	if strings.TrimSpace(reason) == "" {
		return emitErr(ctx.stderr, 1, "invalid_input", "", "deleting a table requires --reason", "")
	}
	if err := p.DeleteTable(name, reason); err != nil {
		return emitErr(ctx.stderr, 1, "not_found", "", "check `dtrules table list`", err.Error())
	}
	if err := p.Save(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOK(ctx, "deleted", map[string]string{"table": name})
}

// tableFiles reports the project's DT files with ranges, purposes, and members.
func (ctx *tableCmdCtx) tableFiles() int {
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	if err := writeJSON(ctx.stdout, map[string]interface{}{"files": p.Files()}); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

// tableNote appends a free-form dated entry to authoring-notes.md.
func (ctx *tableCmdCtx) tableNote(rest []string) int {
	_, _, _, rest = parseTableFlags(rest)
	text := strings.TrimSpace(strings.Join(rest, " "))
	if text == "" {
		return emitErr(ctx.stderr, 1, "invalid_input", "", "table note \"<text>\"", "missing note text")
	}
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	p.AppendNote(text)
	if err := p.Save(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOK(ctx, "noted", map[string]string{"note": text})
}

func (ctx *tableCmdCtx) tablePatch(rest []string) int {
	name, code := ctx.requireName(rest, "table patch <name>")
	if code != 0 {
		return code
	}
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	t := p.Table(name)
	if t == nil {
		return emitErr(ctx.stderr, 1, "not_found", "", "check `dtrules table list`",
			fmt.Sprintf("table %q not found", name))
	}
	data, err := io.ReadAll(ctx.stdin)
	if err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	var patch tablePatch
	// Unknown fields are rejected, not ignored. A payload with its fields
	// nested one level deep decoded to all-zero values and half-applied an
	// empty row while reporting `patched` (#1144). The error names the field,
	// which is what points at the mis-shape.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&patch); err != nil {
		return emitErr(ctx.stderr, 1, "parse_error", "",
			"patch input must be a flat JSON object matching `table schema --patch`", err.Error())
	}
	if err := patch.apply(p, t); err != nil {
		return emitErr(ctx.stderr, 1, "invalid_patch", "", patch.hint(), err.Error())
	}
	if err := p.Save(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	// set-file relocates the underlying record; re-fetch for accurate warnings.
	if t2 := p.Table(t.Name); t2 != nil {
		t = t2
	}
	return writeOKWithWarnings(ctx, "patched",
		map[string]string{"table": t.Name, "op": patch.Op, "file": p.FileOf(t.Name)},
		analyzeAuthoringTable(t))
}

func (ctx *tableCmdCtx) tableSchema(rest []string) int {
	wantPatch := false
	for _, a := range rest {
		if a == "--patch" {
			wantPatch = true
		}
	}
	if wantPatch {
		if _, err := ctx.stdout.Write([]byte(tablePatchSchema)); err != nil {
			return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
		}
		return 0
	}
	if _, err := ctx.stdout.Write([]byte(tableSchemaJSON)); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

// --- EDD handlers ---

func (ctx *tableCmdCtx) eddGet() int {
	p, code := ctx.openProjectForEDD()
	if code != 0 {
		return code
	}
	if err := writeJSON(ctx.stdout, eddToJSON(p.EDD())); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

func (ctx *tableCmdCtx) eddPut() int {
	p, code := ctx.openProjectForEDD()
	if code != 0 {
		return code
	}
	var ej EDDJSON
	if code := decodeStdin(ctx, &ej); code != 0 {
		return code
	}
	if err := ej.ApplyTo(p.EDD()); err != nil {
		return emitErr(ctx.stderr, 1, "invalid_patch", "", "EDD validation failed", err.Error())
	}
	if err := p.SaveEDD(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOK(ctx, "updated", map[string]string{"edd": "ok"})
}

func (ctx *tableCmdCtx) eddPatch() int {
	p, code := ctx.openProjectForEDD()
	if code != 0 {
		return code
	}
	data, err := io.ReadAll(ctx.stdin)
	if err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	var patch eddPatchOp
	if err := json.Unmarshal(data, &patch); err != nil {
		return emitErr(ctx.stderr, 1, "parse_error", "", "patch input must be a JSON object", err.Error())
	}
	if err := patch.apply(p.EDD()); err != nil {
		return emitErr(ctx.stderr, 1, "invalid_patch", "", patch.hint(), err.Error())
	}
	if err := p.SaveEDD(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOK(ctx, "patched", map[string]string{"op": patch.Op})
}

func (ctx *tableCmdCtx) eddSchema(rest []string) int {
	wantPatch := false
	for _, a := range rest {
		if a == "--patch" {
			wantPatch = true
		}
	}
	if wantPatch {
		if _, err := ctx.stdout.Write([]byte(eddPatchSchema)); err != nil {
			return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
		}
		return 0
	}
	if _, err := ctx.stdout.Write([]byte(eddSchemaJSON)); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

// --- helpers ---

// requireName pulls the first positional arg from rest, or returns an error
// with the given usage hint.
func (ctx *tableCmdCtx) requireName(rest []string, usage string) (string, int) {
	for _, a := range rest {
		if !strings.HasPrefix(a, "-") {
			return a, 0
		}
	}
	return "", emitErr(ctx.stderr, 1, "invalid_input", "", usage, "missing table name")
}

// decodeStdin reads stdin as JSON into v.
func decodeStdin(ctx *tableCmdCtx, v interface{}) int {
	data, err := io.ReadAll(ctx.stdin)
	if err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return emitErr(ctx.stderr, 1, "parse_error", "", "stdin was empty — pipe JSON in", "empty input")
	}
	if err := json.Unmarshal(data, v); err != nil {
		return emitErr(ctx.stderr, 1, "parse_error", "", "stdin must be valid JSON", err.Error())
	}
	return 0
}

// writeOK emits {"status":"...","...":"..."} for simple success responses.
func writeOK(ctx *tableCmdCtx, status string, extras map[string]string) int {
	out := map[string]interface{}{"status": status}
	for k, v := range extras {
		out[k] = v
	}
	if err := writeJSON(ctx.stdout, out); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

// writeOKWithWarnings is writeOK plus the authoring-channel warnings
// array. Used by table put / patch responses (#761) so agents see
// optimizer findings on every write without re-fetching.
func writeOKWithWarnings(ctx *tableCmdCtx, status string, extras map[string]string, warns []decisiontable.Warning) int {
	out := map[string]interface{}{"status": status}
	for k, v := range extras {
		out[k] = v
	}
	out["warnings"] = warningsForJSON(warns)
	if err := writeJSON(ctx.stdout, out); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

// --- usage ---

func (c *CLI) printTableUsage() {
	fmt.Println(`Usage: dtrules table <command> [--project <path>]

Commands:
  list                     List decision-table names (JSON).
  get <name>               Print one table as JSON (includes file + advisory warnings).
  put <name> --file F      Replace/create a table from JSON on stdin. A file is
      [--range LO-HI]      required when creating; --range and --reason are
      [--reason R]         required when the file is new. Changing --file moves it.
  patch <name>             Apply a JSON patch op on stdin (response includes warnings).
  delete <name> --reason R Delete a table (reason recorded in authoring-notes.md).
  files                    List DT files with ranges, purposes, and member tables.
  note "<text>"            Append a dated note to authoring-notes.md.
  warnings <name>          Print the advisory-pass warnings for a table as JSON.
  schema                   Emit JSON Schema for a Table document.
  schema --patch           Emit JSON Schema for a table patch op.

Examples:
  dtrules table list --project .
  dtrules table files --project .
  dtrules table put CO_Tax --file states/CO_dt.xml --range 8000-10000 \
    --reason "Colorado tax; own file to avoid merge conflicts" --project . < table.json
  echo '{"op":"set-file","file":"states/CO_dt.xml","reason":"group state logic"}' \
    | dtrules table patch CO_Tax --project .

Patch operations:
  set-name, set-number, set-file, set-range, set-policy,
  set-condition-cell, set-action-cell,
  add-column, update-column, delete-column,
  add-condition, update-condition, update-condition-dsl, delete-condition,
  add-action, update-action, update-action-dsl, delete-action,
  add-initial-action, update-initial-action, delete-initial-action,
  add-context, update-context, delete-context,
  set-policy-statement, delete-policy-statement`)
}

func (c *CLI) printEDDUsage() {
	fmt.Println(`Usage: dtrules edd <command> [--project <path>]

Commands:
  get                      Print the EDD as JSON.
  put                      Replace the EDD from JSON on stdin.
  patch                    Apply a JSON patch op on stdin.
  schema                   Emit JSON Schema for an EDD document.
  schema --patch           Emit JSON Schema for an EDD patch op.

Patch operations:
  add-entity, delete-entity,
  add-field, update-field, delete-field,
  set-comment`)
}
