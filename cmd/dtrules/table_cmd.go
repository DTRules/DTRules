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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

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

// parseProjectFlag pulls --project out of a flag slice, returning the value
// and a copy of args with that flag removed. Defaults to ".".
func parseProjectFlag(args []string) (projectPath string, rest []string) {
	projectPath = "."
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" || args[i] == "-p" {
			if i+1 < len(args) {
				projectPath = args[i+1]
				i++
				continue
			}
		}
		out = append(out, args[i])
	}
	return projectPath, out
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
	ctx.projectPath, rest = parseProjectFlag(rest)

	switch sub {
	case "list":
		return ctx.tableList()
	case "get":
		return ctx.tableGet(rest)
	case "put":
		return ctx.tablePut(rest)
	case "patch":
		return ctx.tablePatch(rest)
	case "schema":
		return ctx.tableSchema(rest)
	case "help", "-h", "--help":
		c.printTableUsage()
		return 0
	default:
		return emitErr(ctx.stderr, 1, "invalid_command", "", "known: list|get|put|patch|schema",
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
	ctx.projectPath, rest = parseProjectFlag(rest)

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
		return nil, emitErr(ctx.stderr, 1, "io_error", "", "project must contain an xml/ directory", err.Error())
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
	if err := writeJSON(ctx.stdout, tableToJSON(t)); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

func (ctx *tableCmdCtx) tablePut(rest []string) int {
	name, code := ctx.requireName(rest, "table put <name>")
	if code != 0 {
		return code
	}
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	t := p.Table(name)
	if t == nil {
		// Allow creating a new table via put.
		newT, err := p.AddTable(name)
		if err != nil {
			return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
		}
		t = newT
	}

	var tj TableJSON
	if code := decodeStdin(ctx, &tj); code != 0 {
		return code
	}
	// Allow omitting the "name" field in input — the URL-ish argument wins.
	if tj.Name == "" {
		tj.Name = name
	}
	if err := tj.ApplyTo(t); err != nil {
		return emitErr(ctx.stderr, 1, "compile_error", "", "an EL expression failed to compile", err.Error())
	}
	if err := p.Save(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOK(ctx, "updated", map[string]string{"table": tj.Name})
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
	if err := json.Unmarshal(data, &patch); err != nil {
		return emitErr(ctx.stderr, 1, "parse_error", "", "patch input must be a JSON object", err.Error())
	}
	if err := patch.apply(t); err != nil {
		return emitErr(ctx.stderr, 1, "invalid_patch", "", patch.hint(), err.Error())
	}
	if err := p.Save(); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	return writeOK(ctx, "patched", map[string]string{"table": t.Name, "op": patch.Op})
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
	p, code := ctx.openProject()
	if code != 0 {
		return code
	}
	if err := writeJSON(ctx.stdout, eddToJSON(p.EDD())); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

func (ctx *tableCmdCtx) eddPut() int {
	p, code := ctx.openProject()
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
	p, code := ctx.openProject()
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
	return "", emitErr(ctx.stderr, 1, "invalid_command", "", usage, "missing table name")
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

// --- usage ---

func (c *CLI) printTableUsage() {
	fmt.Println(`Usage: dtrules table <command> [--project <path>]

Commands:
  list                     List decision-table names (JSON).
  get <name>               Print one table as JSON.
  put <name>               Replace a table from JSON on stdin.
  patch <name>             Apply a JSON patch op on stdin.
  schema                   Emit JSON Schema for a Table document.
  schema --patch           Emit JSON Schema for a table patch op.

Examples:
  dtrules table list --project .
  dtrules table get Compute_Eligibility --project .
  dtrules table put NewTable --project . < table.json
  echo '{"op":"set-condition-cell","condition_number":1,"column":2,"value":"Y"}' \
    | dtrules table patch Compute_Eligibility --project .

Patch operations:
  set-name, set-policy,
  set-condition-cell, set-action-cell,
  add-column, update-column, delete-column,
  add-condition, update-condition, update-condition-dsl, delete-condition,
  add-action, update-action, update-action-dsl, delete-action,
  add-initial-action, update-initial-action, delete-initial-action,
  add-context, update-context, delete-context`)
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
