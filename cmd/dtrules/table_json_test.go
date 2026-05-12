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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyProject duplicates a sampleproject into a tmp dir so tests can mutate
// XML without touching the committed fixtures.
func copyProject(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy project: %v", err)
	}
	return dst
}

// runTableCmd executes runTable against the given stdin and captures stdout/stderr.
func runTableCmd(t *testing.T, projectPath string, args []string, stdin string) (stdout, stderr string, exit int) {
	t.Helper()
	var so, se bytes.Buffer
	cmd := &tableCmdCtx{
		stdin:       strings.NewReader(stdin),
		stdout:      &so,
		stderr:      &se,
		projectPath: projectPath,
	}
	// Dispatch by hand so we bypass flag parsing — the caller supplies the
	// project path directly.
	exit = dispatchTable(cmd, args)
	return so.String(), se.String(), exit
}

func runEDDCmd(t *testing.T, projectPath string, args []string, stdin string) (stdout, stderr string, exit int) {
	t.Helper()
	var so, se bytes.Buffer
	cmd := &tableCmdCtx{
		stdin:       strings.NewReader(stdin),
		stdout:      &so,
		stderr:      &se,
		projectPath: projectPath,
	}
	exit = dispatchEDD(cmd, args)
	return so.String(), se.String(), exit
}

// dispatchTable is the test-only dispatcher mirroring CLI.runTable without
// the os.Std* wiring.
func dispatchTable(ctx *tableCmdCtx, args []string) int {
	if len(args) == 0 {
		return emitErr(ctx.stderr, 1, "invalid_command", "", "", "missing subcommand")
	}
	switch args[0] {
	case "list":
		return ctx.tableList()
	case "get":
		return ctx.tableGet(args[1:])
	case "put":
		return ctx.tablePut(args[1:])
	case "patch":
		return ctx.tablePatch(args[1:])
	case "warnings":
		return ctx.tableWarnings(args[1:])
	case "schema":
		return ctx.tableSchema(args[1:])
	default:
		return emitErr(ctx.stderr, 1, "invalid_command", "", "", "unknown subcommand")
	}
}

func dispatchEDD(ctx *tableCmdCtx, args []string) int {
	if len(args) == 0 {
		return emitErr(ctx.stderr, 1, "invalid_command", "", "", "missing subcommand")
	}
	switch args[0] {
	case "get":
		return ctx.eddGet()
	case "put":
		return ctx.eddPut()
	case "patch":
		return ctx.eddPatch()
	case "schema":
		return ctx.eddSchema(args[1:])
	default:
		return emitErr(ctx.stderr, 1, "invalid_command", "", "", "unknown subcommand")
	}
}

// --- tests ---

func TestTableList(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, code := runTableCmd(t, dir, []string{"list"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct{ Tables []string }
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse list: %v", err)
	}
	if len(got.Tables) == 0 {
		t.Fatalf("expected tables, got none")
	}
}

func TestTableGetJSON(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, code := runTableCmd(t, dir, []string{"get", "Compute_Eligibility"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var tj TableJSON
	if err := json.Unmarshal([]byte(out), &tj); err != nil {
		t.Fatalf("parse table: %v", err)
	}
	if tj.Name != "Compute_Eligibility" {
		t.Errorf("name %q", tj.Name)
	}
	if len(tj.Conditions) == 0 {
		t.Errorf("expected conditions")
	}
}

func TestTableRoundTripJSON(t *testing.T) {
	// Round-trip via JSON (not XML) because the SDK's XML writer is not
	// bytewise-stable across map iteration — see note in table_json.go.
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out1, _, code := runTableCmd(t, dir, []string{"get", "Compute_Eligibility"}, "")
	if code != 0 {
		t.Fatalf("get1 exit %d", code)
	}
	if _, _, code := runTableCmd(t, dir, []string{"put", "Compute_Eligibility"}, out1); code != 0 {
		t.Fatalf("put exit %d", code)
	}
	out2, _, code := runTableCmd(t, dir, []string{"get", "Compute_Eligibility"}, "")
	if code != 0 {
		t.Fatalf("get2 exit %d", code)
	}
	var a, b TableJSON
	if err := json.Unmarshal([]byte(out1), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out2), &b); err != nil {
		t.Fatal(err)
	}
	// Re-marshal both indented, compare strings. This hides any iteration
	// order differences that cannot survive the native SDK structs.
	raw1, _ := json.MarshalIndent(a, "", "  ")
	raw2, _ := json.MarshalIndent(b, "", "  ")
	if string(raw1) != string(raw2) {
		t.Errorf("round-trip drift:\n--- first ---\n%s\n--- second ---\n%s", raw1, raw2)
	}
}

func TestTablePutInvalidDSL(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	badTable := TableJSON{
		Name:   "Compute_Eligibility",
		Policy: "FIRST",
		Conditions: []ConditionJSON{
			{Number: 1, DSL: "!!!this is not valid EL!!!"},
		},
		Actions: []ActionJSON{
			{Number: 1, DSL: "perform Calculate_Individual_Income"},
		},
	}
	payload, _ := json.Marshal(badTable)
	_, se, code := runTableCmd(t, dir, []string{"put", "Compute_Eligibility"}, string(payload))
	if code == 0 {
		t.Fatalf("expected non-zero exit on invalid DSL")
	}
	var je jsonError
	if err := json.Unmarshal([]byte(se), &je); err != nil {
		t.Fatalf("stderr not JSON: %v\n%s", err, se)
	}
	if je.Error != "compile_error" {
		t.Errorf("expected compile_error, got %q", je.Error)
	}
}

// TestTableGetIncludesWarnings checks that table_get embeds the
// authoring-channel warnings array (#761). The shape is locked so that
// MCP clients can iterate `warnings` unconditionally.
func TestTableGetIncludesWarnings(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, code := runTableCmd(t, dir, []string{"get", "Compute_Eligibility"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := raw["warnings"].([]interface{}); !ok {
		t.Errorf("expected warnings as JSON array, got %T %v",
			raw["warnings"], raw["warnings"])
	}
	// The bare TableJSON fields must still be present at the top level
	// — TestTableGetJSON parses into a TableJSON directly and depends on
	// that flat shape.
	if raw["name"] != "Compute_Eligibility" {
		t.Errorf("name field missing or wrong: %v", raw["name"])
	}
}

// TestTableWarningsCommand exercises `dtrules table warnings <name>`:
// a read-only fetch of the advisory-pass warnings, used by the agent
// loop when it wants warnings without re-fetching the whole table.
func TestTableWarningsCommand(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, se, code := runTableCmd(t, dir, []string{"warnings", "Compute_Eligibility"}, "")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, se)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if raw["table"] != "Compute_Eligibility" {
		t.Errorf("table field wrong: %v", raw["table"])
	}
	if _, ok := raw["warnings"].([]interface{}); !ok {
		t.Errorf("expected warnings as JSON array, got %T %v",
			raw["warnings"], raw["warnings"])
	}
}

// TestTableWarningsNotFound surfaces a not_found error envelope when
// the named table doesn't exist — same shape as other table commands.
func TestTableWarningsNotFound(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	_, se, code := runTableCmd(t, dir, []string{"warnings", "NoSuchTable"}, "")
	if code == 0 {
		t.Fatalf("expected non-zero exit")
	}
	var je jsonError
	if err := json.Unmarshal([]byte(se), &je); err != nil {
		t.Fatalf("stderr not JSON: %v", err)
	}
	if je.Error != "not_found" {
		t.Errorf("expected not_found, got %q", je.Error)
	}
}

func TestTableGetNotFound(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	_, se, code := runTableCmd(t, dir, []string{"get", "NoSuchTable"}, "")
	if code == 0 {
		t.Fatalf("expected non-zero exit")
	}
	var je jsonError
	if err := json.Unmarshal([]byte(se), &je); err != nil {
		t.Fatalf("stderr not JSON: %v", err)
	}
	if je.Error != "not_found" {
		t.Errorf("expected not_found, got %q", je.Error)
	}
}

func TestTablePatchSetConditionCell(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	patch := `{"op":"set-condition-cell","condition_number":1,"column":1,"value":"N"}`
	_, se, code := runTableCmd(t, dir, []string{"patch", "Compute_Eligibility"}, patch)
	if code != 0 {
		t.Fatalf("patch exit %d stderr=%s", code, se)
	}
	// Re-get and check.
	out, _, _ := runTableCmd(t, dir, []string{"get", "Compute_Eligibility"}, "")
	var tj TableJSON
	_ = json.Unmarshal([]byte(out), &tj)
	found := false
	for _, c := range tj.Conditions {
		if c.Number == 1 {
			if c.Columns["1"] != "N" {
				t.Errorf("condition 1 column 1: want N got %q", c.Columns["1"])
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("condition 1 missing")
	}
}

func TestTablePatchDeleteCondition(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	patch := `{"op":"delete-condition","condition_number":2}`
	_, se, code := runTableCmd(t, dir, []string{"patch", "Compute_Eligibility"}, patch)
	if code != 0 {
		t.Fatalf("patch exit %d stderr=%s", code, se)
	}
	out, _, _ := runTableCmd(t, dir, []string{"get", "Compute_Eligibility"}, "")
	var tj TableJSON
	_ = json.Unmarshal([]byte(out), &tj)
	for _, c := range tj.Conditions {
		if c.Number == 2 {
			t.Fatalf("condition 2 should be deleted")
		}
	}
	// Other conditions still present.
	if len(tj.Conditions) < 2 {
		t.Errorf("expected other conditions to survive")
	}
}

func TestTablePatchAddColumn(t *testing.T) {
	// Target table: use one that has no '*' columns so AddColumn's legal-
	// value check accepts it.
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, _ := runTableCmd(t, dir, []string{"get", "Evaluate_CHIP_Eligibility"}, "")
	var before TableJSON
	_ = json.Unmarshal([]byte(out), &before)
	if len(before.Conditions) == 0 {
		t.Fatalf("fixture has no conditions")
	}
	// Build an add-column patch that supplies "N" for every condition.
	// We can't use "-" because the SDK elides dash cells on serialize.
	condsMap := map[string]string{}
	for _, c := range before.Conditions {
		condsMap[itoa(c.Number)] = "N"
	}
	patch := struct {
		Op         string            `json:"op"`
		Conditions map[string]string `json:"conditions"`
		Actions    []int             `json:"actions"`
	}{"add-column", condsMap, []int{}}
	payload, _ := json.Marshal(patch)
	_, se, code := runTableCmd(t, dir, []string{"patch", "Evaluate_CHIP_Eligibility"}, string(payload))
	if code != 0 {
		t.Fatalf("add-column exit %d stderr=%s", code, se)
	}
	// The new column should live at max+1.
	maxCol := 0
	for _, c := range before.Conditions {
		for k := range c.Columns {
			n := atoi(k)
			if n > maxCol {
				maxCol = n
			}
		}
	}
	out2, _, _ := runTableCmd(t, dir, []string{"get", "Evaluate_CHIP_Eligibility"}, "")
	var after TableJSON
	_ = json.Unmarshal([]byte(out2), &after)
	newCol := maxCol + 1
	for _, c := range after.Conditions {
		if c.Columns[itoa(newCol)] == "" {
			t.Errorf("condition %d has no value at column %d", c.Number, newCol)
		}
	}
}

func TestTableSchema(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, code := runTableCmd(t, dir, []string{"schema"}, "")
	if code != 0 {
		t.Fatalf("schema exit %d", code)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}
	if _, ok := m["$schema"]; !ok {
		t.Errorf("schema missing $schema")
	}
	// Patch schema
	outP, _, code := runTableCmd(t, dir, []string{"schema", "--patch"}, "")
	if code != 0 {
		t.Fatalf("schema --patch exit %d", code)
	}
	var mp map[string]interface{}
	if err := json.Unmarshal([]byte(outP), &mp); err != nil {
		t.Fatalf("patch schema not JSON: %v", err)
	}
	// Ensure enum list is present.
	raw, _ := json.Marshal(mp)
	for _, op := range []string{"set-condition-cell", "add-column", "delete-condition"} {
		if !strings.Contains(string(raw), op) {
			t.Errorf("patch schema missing op %q", op)
		}
	}
}

func TestTablePutInvalidJSON(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	_, se, code := runTableCmd(t, dir, []string{"put", "Compute_Eligibility"}, "{not json}")
	if code == 0 {
		t.Fatalf("expected non-zero exit")
	}
	var je jsonError
	if err := json.Unmarshal([]byte(se), &je); err != nil {
		t.Fatalf("stderr not JSON: %v", err)
	}
	if je.Error != "parse_error" {
		t.Errorf("expected parse_error, got %q", je.Error)
	}
}

func TestEDDGet(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, code := runEDDCmd(t, dir, []string{"get"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var ej EDDJSON
	if err := json.Unmarshal([]byte(out), &ej); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ej.Entities) == 0 {
		t.Errorf("expected entities")
	}
}

func TestEDDPatchAddEntity(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	patch := `{"op":"add-entity","entity":"TestAuthoringEntity"}`
	_, se, code := runEDDCmd(t, dir, []string{"patch"}, patch)
	if code != 0 {
		t.Fatalf("patch exit %d stderr=%s", code, se)
	}
	out, _, _ := runEDDCmd(t, dir, []string{"get"}, "")
	if !strings.Contains(out, "TestAuthoringEntity") {
		t.Errorf("new entity missing from edd get")
	}
}

func TestEDDSchema(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	out, _, code := runEDDCmd(t, dir, []string{"schema"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("schema not JSON: %v", err)
	}
	if _, ok := m["$schema"]; !ok {
		t.Errorf("schema missing $schema")
	}
}

// itoa / atoi — small helpers kept local to the test file.
func itoa(n int) string {
	return string(formatInt(int64(n)))
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// formatInt avoids pulling strconv into the test just for Itoa.
func formatInt(n int64) []byte {
	if n == 0 {
		return []byte("0")
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return buf[i:]
}

