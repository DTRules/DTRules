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
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
)

// mcpRPC is a tiny harness that runs an mcpServer in a goroutine over a pair
// of os.Pipes and lets the test send requests / read responses synchronously.
// Simpler than spawning a subprocess and more reliable — we're testing the
// protocol handler, not the CLI binary.
type mcpRPC struct {
	t         *testing.T
	stdinW    *io.PipeWriter
	stdoutR   *bufio.Reader
	stdoutW   *io.PipeWriter
	serverErr chan error
	cancel    context.CancelFunc
	mu        sync.Mutex
	nextID    int
}

func newMCPRPC(t *testing.T, projectPath string) *mcpRPC {
	t.Helper()
	pr1, pw1 := io.Pipe() // client -> server stdin
	pr2, pw2 := io.Pipe() // server -> client stdout

	ctx, cancel := context.WithCancel(context.Background())
	srv := newMCPServer(pr1, pw2, projectPath)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
		_ = pw2.Close()
	}()

	return &mcpRPC{
		t:         t,
		stdinW:    pw1,
		stdoutR:   bufio.NewReader(pr2),
		stdoutW:   pw2,
		serverErr: errCh,
		cancel:    cancel,
	}
}

func (r *mcpRPC) close() {
	_ = r.stdinW.Close()
	// Give the server a moment to flush, then cancel.
	r.cancel()
	<-r.serverErr
}

// call sends a request and returns the parsed response.
func (r *mcpRPC) call(method string, params interface{}) map[string]interface{} {
	r.t.Helper()
	r.mu.Lock()
	r.nextID++
	id := r.nextID
	r.mu.Unlock()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	raw, err := json.Marshal(req)
	if err != nil {
		r.t.Fatalf("marshal req: %v", err)
	}
	raw = append(raw, '\n')
	if _, err := r.stdinW.Write(raw); err != nil {
		r.t.Fatalf("write req: %v", err)
	}

	line, err := r.stdoutR.ReadBytes('\n')
	if err != nil {
		r.t.Fatalf("read resp: %v", err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(line, &resp); err != nil {
		r.t.Fatalf("parse resp: %v\n%s", err, line)
	}
	return resp
}

// callRaw writes a literal byte sequence to the server and reads one line
// back. Used to exercise malformed-envelope handling.
func (r *mcpRPC) callRaw(raw string) map[string]interface{} {
	r.t.Helper()
	if _, err := r.stdinW.Write([]byte(raw + "\n")); err != nil {
		r.t.Fatalf("write raw: %v", err)
	}
	line, err := r.stdoutR.ReadBytes('\n')
	if err != nil {
		r.t.Fatalf("read raw resp: %v", err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(line, &resp); err != nil {
		r.t.Fatalf("parse raw resp: %v\n%s", err, line)
	}
	return resp
}

// --- tests ---

func TestMCPInitialize(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test-client", "version": "0.0.0"},
	})
	if errVal, ok := resp["error"]; ok && errVal != nil {
		t.Fatalf("initialize failed: %v", errVal)
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp["result"])
	}
	if got := result["protocolVersion"]; got != mcpProtocolVersion {
		t.Errorf("protocolVersion: want %q got %v", mcpProtocolVersion, got)
	}
	caps, ok := result["capabilities"].(map[string]interface{})
	if !ok || caps["tools"] == nil {
		t.Errorf("expected capabilities.tools, got %v", result["capabilities"])
	}
}

func TestMCPToolsList(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/list", nil)
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %v", resp)
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools array, got %T", result["tools"])
	}
	want := map[string]bool{
		"table_list": true, "table_get": true, "table_put": true,
		"table_patch": true, "table_schema": true,
		"edd_get": true, "edd_put": true, "edd_patch": true, "edd_schema": true,
		"project_validate": true, "project_diagnostics": true,
	}
	for _, tool := range tools {
		tm := tool.(map[string]interface{})
		delete(want, tm["name"].(string))
		// Each tool must declare a non-empty input schema.
		if _, ok := tm["inputSchema"]; !ok {
			t.Errorf("tool %v missing inputSchema", tm["name"])
		}
	}
	if len(want) != 0 {
		t.Errorf("missing tools: %v", want)
	}
}

func TestMCPTableList(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "table_list",
		"arguments": map[string]interface{}{},
	})
	result := mustResult(t, resp)
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structuredContent, got %v", result)
	}
	tables, _ := structured["tables"].([]interface{})
	if len(tables) == 0 {
		t.Errorf("expected at least one table")
	}
}

func TestMCPTableGet(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "table_get",
		"arguments": map[string]interface{}{"name": "Compute_Eligibility"},
	})
	result := mustResult(t, resp)
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structuredContent, got %v", result)
	}
	if got := structured["name"]; got != "Compute_Eligibility" {
		t.Errorf("name: want Compute_Eligibility got %v", got)
	}
}

func TestMCPTablePutInvalidDSL(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name": "table_put",
		"arguments": map[string]interface{}{
			"name": "Compute_Eligibility",
			"table": map[string]interface{}{
				"name":   "Compute_Eligibility",
				"policy": "FIRST",
				"conditions": []interface{}{
					map[string]interface{}{"number": 1, "dsl": "!!!this is not valid EL!!!"},
				},
				"actions": []interface{}{
					map[string]interface{}{"number": 1, "dsl": "perform Calculate_Individual_Income"},
				},
			},
		},
	})
	result := mustResult(t, resp)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError=true, got result=%v", result)
	}
	// structuredContent should include the kind so clients can branch on it.
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structuredContent in error result")
	}
	if kind, _ := structured["error"].(string); kind != "compile_error" {
		t.Errorf("expected compile_error, got %q", kind)
	}
}

func TestMCPUnknownTool(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "definitely_not_a_tool",
		"arguments": map[string]interface{}{},
	})
	result := mustResult(t, resp)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError=true for unknown tool, got %v", result)
	}
}

func TestMCPMalformedEnvelope(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	// Send a line that is not valid JSON. Server should respond with a
	// parse-error envelope and the loop should keep running.
	resp := rpc.callRaw(`{"jsonrpc": "2.0", "id": 1, "method": `)
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object, got %v", resp)
	}
	code, _ := errObj["code"].(float64)
	if int(code) != jrpcParseError {
		t.Errorf("want parse-error code %d, got %d", jrpcParseError, int(code))
	}

	// Loop should still be alive — a well-formed request after the broken
	// one should succeed.
	resp = rpc.call("tools/list", nil)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("server did not recover: %v", resp)
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("no/such/method", nil)
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error, got %v", resp)
	}
	code, _ := errObj["code"].(float64)
	if int(code) != jrpcMethodNotFound {
		t.Errorf("want method-not-found %d, got %d", jrpcMethodNotFound, int(code))
	}
}

func TestMCPProjectValidate(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "project_validate",
		"arguments": map[string]interface{}{},
	})
	result := mustResult(t, resp)
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structuredContent, got %v", result)
	}
	if _, ok := structured["structure"]; !ok {
		t.Errorf("expected structure key in validation report")
	}
}

func TestMCPTableSchema(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "table_schema",
		"arguments": map[string]interface{}{},
	})
	result := mustResult(t, resp)
	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("expected content array, got %v", result)
	}
	text, _ := content[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "DTRulesTable") {
		t.Errorf("expected TableJSON schema in response, got %q", text)
	}
}

func TestMCPProjectDiagnostics(t *testing.T) {
	dir := writeDupXMLProject(t, map[string][]string{
		"001_a_dt.xml": {"Foo"},
		"002_b_dt.xml": {"Foo"},
	})
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "project_diagnostics",
		"arguments": map[string]interface{}{},
	})
	result := mustResult(t, resp)
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structuredContent, got %v", result)
	}
	diags, _ := structured["diagnostics"].([]interface{})
	if len(diags) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	d := diags[0].(map[string]interface{})
	if d["kind"] != "duplicate_table" {
		t.Errorf("kind = %v, want duplicate_table", d["kind"])
	}
	if d["original_name"] != "Foo" || d["assigned_name"] != "Foo-1" {
		t.Errorf("unexpected diagnostic: %+v", d)
	}
}

func TestMCPProjectDiagnosticsClean(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rpc := newMCPRPC(t, dir)
	defer rpc.close()

	resp := rpc.call("tools/call", map[string]interface{}{
		"name":      "project_diagnostics",
		"arguments": map[string]interface{}{},
	})
	result := mustResult(t, resp)
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structuredContent, got %v", result)
	}
	diags, _ := structured["diagnostics"].([]interface{})
	if len(diags) != 0 {
		t.Errorf("CHIP project should have no duplicates, got %+v", diags)
	}
}

// mustResult extracts result map or fails the test with the error detail.
func mustResult(t *testing.T, resp map[string]interface{}) map[string]interface{} {
	t.Helper()
	if errVal, ok := resp["error"]; ok && errVal != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", errVal)
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T: %v", resp["result"], resp)
	}
	return result
}
