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

// MCP tool definitions and dispatch. Each tool is a thin stateless wrapper
// around the same authoring-SDK calls that back the #716 JSON CLI: load the
// project, run the op, save, discard. No persistent project state.

import (
	"encoding/json"
	"fmt"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// toolError wraps a jsonError so the protocol layer can surface the same
// structured shape the CLI emits on stderr.
type toolError struct {
	payload jsonError
}

func (e *toolError) Error() string {
	return fmt.Sprintf("%s: %s", e.payload.Error, e.payload.Detail)
}

func newToolError(kind, hint, detail string) *toolError {
	return &toolError{payload: jsonError{Error: kind, Hint: hint, Detail: detail}}
}

// buildToolDefs returns the static MCP tool catalogue. Input schemas reuse
// the hand-written JSON Schemas from schemas.go (wrapped so they expose the
// `project_path` + `name` envelope each tool actually accepts).
func buildToolDefs() []mcpToolDef {
	return []mcpToolDef{
		{
			Name:        "table_list",
			Description: "List every decision-table name in the project.",
			InputSchema: schemaProjectOnly(),
		},
		{
			Name:        "table_get",
			Description: "Return a decision table as JSON (TableJSON shape).",
			InputSchema: schemaProjectAndName("name", "Decision table name."),
		},
		{
			Name:        "table_put",
			Description: "Replace a decision table from a TableJSON payload. Creates the table if it does not exist.",
			InputSchema: schemaTablePut(),
		},
		{
			Name:        "table_patch",
			Description: "Apply a single JSON patch op to a decision table (set-condition-cell, add-column, etc.).",
			InputSchema: schemaTablePatch(),
		},
		{
			Name:        "table_warnings",
			Description: "Return the per-table advisory warnings (no-op columns, subsumption, unreachable columns, hand-coded postfix, etc.) for a single table as JSON. Warnings are advisory; the deployment gate (project_full_review) ignores them.",
			InputSchema: schemaProjectAndName("name", "Decision table name."),
		},
		{
			Name:        "table_schema",
			Description: "Return the JSON Schema for a TableJSON document.",
			InputSchema: schemaEmpty(),
		},
		{
			Name:        "edd_get",
			Description: "Return the entity data dictionary (EDD) as JSON.",
			InputSchema: schemaProjectOnly(),
		},
		{
			Name:        "edd_put",
			Description: "Replace the EDD from an EDDJSON payload.",
			InputSchema: schemaEDDPut(),
		},
		{
			Name:        "edd_patch",
			Description: "Apply a single JSON patch op to the EDD (add-entity, add-field, set-comment, etc.).",
			InputSchema: schemaEDDPatchOp(),
		},
		{
			Name:        "edd_schema",
			Description: "Return the JSON Schema for an EDDJSON document.",
			InputSchema: schemaEmpty(),
		},
		{
			Name:        "project_validate",
			Description: "Run structural and EL-compliance validation on the project. Returns a structured report.",
			InputSchema: schemaProjectOnly(),
		},
		{
			Name:        "project_diagnostics",
			Description: "Return authoring-time diagnostics (e.g. duplicate_table renames) for the project as JSON.",
			InputSchema: schemaProjectOnly(),
		},
		{
			Name:        "project_full_review",
			Description: "Run the project-wide Full Review (structure + EL compliance + load diagnostics + per-table optimizer + EDD-unused + table call graph orphan-call detection — surfaced as warnings). Persists the report to .dtrules/last-review.json and returns it. The deployment gate (dtrules build --require-review) reads this file. Errors gate deployment; warnings do not.",
			InputSchema: schemaProjectOnly(),
		},
	}
}

// callTool is the entry point invoked by handleToolsCall. Success returns an
// MCP result envelope; failure returns a *toolError the caller surfaces as
// an MCP tool error (not a JSON-RPC error).
func (s *mcpServer) callTool(name string, args json.RawMessage) (map[string]interface{}, error) {
	// Resolve project path: tool-arg wins, otherwise the server default.
	project := s.defaultProject
	var base struct {
		ProjectPath string `json:"project_path"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &base); err != nil {
			return nil, newToolError("parse_error", "arguments must be a JSON object", err.Error())
		}
	}
	if base.ProjectPath != "" {
		project = base.ProjectPath
	}

	switch name {
	case "table_list":
		return s.toolTableList(project)
	case "table_get":
		return s.toolTableGet(project, args)
	case "table_put":
		return s.toolTablePut(project, args)
	case "table_patch":
		return s.toolTablePatch(project, args)
	case "table_warnings":
		return s.toolTableWarnings(project, args)
	case "table_schema":
		return mcpTextResult(tableSchemaJSON), nil
	case "edd_get":
		return s.toolEDDGet(project)
	case "edd_put":
		return s.toolEDDPut(project, args)
	case "edd_patch":
		return s.toolEDDPatch(project, args)
	case "edd_schema":
		return mcpTextResult(eddSchemaJSON), nil
	case "project_validate":
		return s.toolProjectValidate(project)
	case "project_diagnostics":
		return s.toolProjectDiagnostics(project)
	case "project_full_review":
		return s.toolProjectFullReview(project)
	default:
		return nil, newToolError("invalid_command", "check tools/list", fmt.Sprintf("unknown tool %q", name))
	}
}

// --- read tools ---

func (s *mcpServer) toolTableList(project string) (map[string]interface{}, error) {
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	names := p.Tables()
	if names == nil {
		names = []string{}
	}
	return mcpJSONResult(map[string]interface{}{"tables": names})
}

func (s *mcpServer) toolTableGet(project string, args json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, newToolError("parse_error", "arguments must be a JSON object", err.Error())
	}
	if req.Name == "" {
		return nil, newToolError("invalid_command", "arguments.name is required", "missing table name")
	}
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	t := p.Table(req.Name)
	if t == nil {
		return nil, newToolError("not_found", "call table_list to enumerate",
			fmt.Sprintf("table %q not found", req.Name))
	}
	return mcpJSONResult(tableGetResponse{
		TableJSON: tableToJSON(t),
		Warnings:  warningsForJSON(analyzeAuthoringTable(t)),
	})
}

// toolTableWarnings is the read-only authoring-channel pass (#761):
// runs the structural checks against a single table without re-fetching
// or saving. Mirrors the `dtrules table warnings <name>` CLI.
func (s *mcpServer) toolTableWarnings(project string, args json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, newToolError("parse_error", "arguments must be a JSON object", err.Error())
	}
	if req.Name == "" {
		return nil, newToolError("invalid_command", "arguments.name is required", "missing table name")
	}
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	t := p.Table(req.Name)
	if t == nil {
		return nil, newToolError("not_found", "call table_list to enumerate",
			fmt.Sprintf("table %q not found", req.Name))
	}
	return mcpJSONResult(map[string]interface{}{
		"table":    t.Name,
		"warnings": warningsForJSON(analyzeAuthoringTable(t)),
	})
}

func (s *mcpServer) toolEDDGet(project string) (map[string]interface{}, error) {
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	return mcpJSONResult(eddToJSON(p.EDD()))
}

func (s *mcpServer) toolProjectValidate(project string) (map[string]interface{}, error) {
	structRes, err := sync.ValidateProjectStructure(project, "", "")
	if err != nil {
		return nil, newToolError("io_error", "structure validation failed", err.Error())
	}
	report := map[string]interface{}{
		"structure": map[string]interface{}{
			"valid":    structRes.IsValid(),
			"errors":   structErrors(structRes),
			"warnings": structWarnings(structRes),
		},
	}
	// EL compliance requires the XML dir — locate via the structure result.
	xmlDir := ""
	if structRes.Structure != nil {
		xmlDir = structRes.Structure.XMLDir
	}
	if xmlDir == "" {
		xmlDir = project + "/xml"
	}
	elRes, err := sync.ValidateELCompliance(xmlDir)
	if err == nil && elRes != nil {
		report["el_compliance"] = map[string]interface{}{
			"compliant":           elRes.IsCompliant(),
			"legacy_files":        elRes.LegacyFiles,
			"compliant_files":     elRes.CompliantFiles,
			"needs_compilation":   elRes.NeedsCompilationFiles,
		}
	}
	return mcpJSONResult(report)
}

// toolProjectFullReview is the MCP-facing Full Review (#768). It mirrors
// `dtrules review`: runs every check, persists the report to
// .dtrules/last-review.json, and returns the same JSON envelope. The
// deployment gate reads the persisted file separately.
func (s *mcpServer) toolProjectFullReview(project string) (map[string]interface{}, error) {
	rep, err := runFullReview(project)
	if err != nil {
		// A persist failure is non-fatal — the report is still usable
		// as a one-off MCP response even if the cache write blew up.
		// Pass it through unchanged so the caller sees both signals.
		_ = err
	}
	if rep == nil {
		return nil, newToolError("io_error", "review returned no report", "")
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		return nil, newToolError("io_error", "could not marshal report", err.Error())
	}
	var asMap map[string]interface{}
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return nil, newToolError("io_error", "could not remarshal report", err.Error())
	}
	return mcpJSONResult(asMap)
}

func (s *mcpServer) toolProjectDiagnostics(project string) (map[string]interface{}, error) {
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	diags := p.Diagnostics()
	if diags == nil {
		diags = []authoring.Diagnostic{}
	}
	return mcpJSONResult(map[string]interface{}{"diagnostics": diags})
}

// --- write tools ---

func (s *mcpServer) toolTablePut(project string, args json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		Name  string    `json:"name"`
		Table TableJSON `json:"table"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, newToolError("parse_error", "arguments must be {name, table}", err.Error())
	}
	if req.Name == "" {
		return nil, newToolError("invalid_command", "arguments.name is required", "missing table name")
	}
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	t := p.Table(req.Name)
	if t == nil {
		newT, addErr := p.AddTable(req.Name)
		if addErr != nil {
			return nil, newToolError("io_error", "could not create table", addErr.Error())
		}
		t = newT
	}
	if req.Table.Name == "" {
		req.Table.Name = req.Name
	}
	if err := req.Table.ApplyTo(t); err != nil {
		return nil, newToolError("compile_error", "an EL expression failed to compile", err.Error())
	}
	if err := p.Save(); err != nil {
		return nil, newToolError("io_error", "save failed", err.Error())
	}
	return mcpJSONResult(map[string]interface{}{
		"status":   "updated",
		"table":    req.Table.Name,
		"warnings": warningsForJSON(analyzeAuthoringTable(t)),
	})
}

func (s *mcpServer) toolTablePatch(project string, args json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		Name  string          `json:"name"`
		Patch json.RawMessage `json:"patch"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, newToolError("parse_error", "arguments must be {name, patch}", err.Error())
	}
	if req.Name == "" {
		return nil, newToolError("invalid_command", "arguments.name is required", "missing table name")
	}
	if len(req.Patch) == 0 {
		return nil, newToolError("invalid_command", "arguments.patch is required", "missing patch object")
	}
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	t := p.Table(req.Name)
	if t == nil {
		return nil, newToolError("not_found", "call table_list to enumerate",
			fmt.Sprintf("table %q not found", req.Name))
	}
	var op tablePatch
	if err := json.Unmarshal(req.Patch, &op); err != nil {
		return nil, newToolError("parse_error", "patch must be a JSON object", err.Error())
	}
	if err := op.apply(t); err != nil {
		return nil, newToolError("invalid_patch", op.hint(), err.Error())
	}
	if err := p.Save(); err != nil {
		return nil, newToolError("io_error", "save failed", err.Error())
	}
	return mcpJSONResult(map[string]interface{}{
		"status":   "patched",
		"table":    t.Name,
		"op":       op.Op,
		"warnings": warningsForJSON(analyzeAuthoringTable(t)),
	})
}

func (s *mcpServer) toolEDDPut(project string, args json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		EDD EDDJSON `json:"edd"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, newToolError("parse_error", "arguments must be {edd}", err.Error())
	}
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	if err := req.EDD.ApplyTo(p.EDD()); err != nil {
		return nil, newToolError("invalid_patch", "EDD validation failed", err.Error())
	}
	if err := p.SaveEDD(); err != nil {
		return nil, newToolError("io_error", "save failed", err.Error())
	}
	return mcpJSONResult(map[string]interface{}{"status": "updated", "edd": "ok"})
}

func (s *mcpServer) toolEDDPatch(project string, args json.RawMessage) (map[string]interface{}, error) {
	var req struct {
		Patch json.RawMessage `json:"patch"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, newToolError("parse_error", "arguments must be {patch}", err.Error())
	}
	if len(req.Patch) == 0 {
		return nil, newToolError("invalid_command", "arguments.patch is required", "missing patch object")
	}
	p, err := authoring.OpenProject(project)
	if err != nil {
		return nil, newToolError("io_error", "project must contain an xml/ subdirectory or *_dt.xml files directly", err.Error())
	}
	var op eddPatchOp
	if err := json.Unmarshal(req.Patch, &op); err != nil {
		return nil, newToolError("parse_error", "patch must be a JSON object", err.Error())
	}
	if err := op.apply(p.EDD()); err != nil {
		return nil, newToolError("invalid_patch", op.hint(), err.Error())
	}
	if err := p.SaveEDD(); err != nil {
		return nil, newToolError("io_error", "save failed", err.Error())
	}
	return mcpJSONResult(map[string]interface{}{"status": "patched", "op": op.Op})
}

// --- result helpers ---

// mcpJSONResult wraps a value as the canonical MCP tool result: a single
// text content block containing the JSON, plus a structuredContent field
// that holds the same value in structured form (so clients that understand
// structured results don't have to re-parse the text).
func mcpJSONResult(v interface{}) (map[string]interface{}, error) {
	text, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, newToolError("io_error", "serialization failed", err.Error())
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": string(text)},
		},
		"structuredContent": v,
	}, nil
}

// mcpTextResult wraps a raw string (already JSON or plain text) as a text
// content block with no structuredContent.
func mcpTextResult(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": text},
		},
	}
}

// --- structure-result helpers ---

func structErrors(r *sync.StructureValidationResult) []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.Errors))
	for _, e := range r.Errors {
		out = append(out, e.Error())
	}
	return out
}

func structWarnings(r *sync.StructureValidationResult) []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.Warnings))
	for _, w := range r.Warnings {
		out = append(out, w.String())
	}
	return out
}

// --- input-schema builders ---
//
// The MCP spec requires each tool to declare a JSON Schema for its inputs.
// These schemas embed the table/EDD schemas from schemas.go by $ref where
// appropriate rather than re-emitting the payload shape.

func schemaEmpty() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"project_path":{"type":"string","description":"Absolute path to the DTRules project root (defaults to the server's --project value)."}}}`)
}

func schemaProjectOnly() json.RawMessage {
	return schemaEmpty()
}

func schemaProjectAndName(field, desc string) json.RawMessage {
	s := fmt.Sprintf(`{"type":"object","required":[%q],"properties":{"project_path":{"type":"string"},%q:{"type":"string","description":%q}}}`,
		field, field, desc)
	return json.RawMessage(s)
}

func schemaTablePut() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["name", "table"],
  "properties": {
    "project_path": {"type": "string"},
    "name":  {"type": "string", "description": "Decision-table name. If it does not exist it will be created."},
    "table": {"type": "object", "description": "TableJSON document — see table_schema for the full shape."}
  }
}`)
}

func schemaTablePatch() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["name", "patch"],
  "properties": {
    "project_path": {"type": "string"},
    "name":  {"type": "string", "description": "Decision-table name."},
    "patch": {"type": "object", "description": "Single patch op. See dtrules table schema --patch for the full enum."}
  }
}`)
}

func schemaEDDPut() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["edd"],
  "properties": {
    "project_path": {"type": "string"},
    "edd": {"type": "object", "description": "EDDJSON document — see edd_schema for the full shape."}
  }
}`)
}

func schemaEDDPatchOp() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["patch"],
  "properties": {
    "project_path": {"type": "string"},
    "patch": {"type": "object", "description": "Single patch op. See dtrules edd schema --patch for the enum."}
  }
}`)
}
