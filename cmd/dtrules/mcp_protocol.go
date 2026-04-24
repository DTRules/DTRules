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

// JSON-RPC 2.0 framing + MCP-level handlers for `dtrules mcp`.
//
// Wire format: newline-delimited JSON objects on stdin/stdout, one request
// or response per line. The MCP spec permits either this or the
// Content-Length header framing used by LSP; we pick ND-JSON because it's
// what the reference TypeScript SDK emits by default for the stdio
// transport and what Claude Code speaks.
//
// The spec: https://spec.modelcontextprotocol.io

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// mcpProtocolVersion is the protocol version we negotiate on `initialize`.
// MCP clients that support a newer version will typically still accept this
// and downgrade; older ones will reject.
const mcpProtocolVersion = "2024-11-05"

// serverName and serverVersion surface in the `initialize` result.
const (
	mcpServerName    = "dtrules"
	mcpServerVersion = "1.9.1"
)

// JSON-RPC 2.0 error codes (subset we actually emit).
const (
	jrpcParseError     = -32700
	jrpcInvalidRequest = -32600
	jrpcMethodNotFound = -32601
	jrpcInvalidParams  = -32602
	jrpcInternalError  = -32603
)

// jrpcRequest is a parsed JSON-RPC 2.0 request envelope. The `id` may be a
// string, number, or null (for notifications) — we keep it as json.RawMessage
// so responses can echo it verbatim.
type jrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jrpcResponse is a JSON-RPC 2.0 response envelope. Exactly one of Result or
// Error must be set; we rely on the caller to populate correctly.
type jrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jrpcError      `json:"error,omitempty"`
}

type jrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// mcpServer is the stdio MCP server. It holds the input decoder, the output
// writer, and the default project path used for any tool call that doesn't
// override it.
type mcpServer struct {
	in             *bufio.Scanner
	out            io.Writer
	defaultProject string
	tools          []mcpToolDef
}

// mcpToolDef describes a single MCP tool exposed via `tools/list`.
type mcpToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// newMCPServer constructs a server bound to the given streams and project.
func newMCPServer(stdin io.Reader, stdout io.Writer, defaultProject string) *mcpServer {
	s := bufio.NewScanner(stdin)
	// The default 64 KiB buffer is too tight for a large TableJSON payload.
	s.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	return &mcpServer{
		in:             s,
		out:            stdout,
		defaultProject: defaultProject,
		tools:          buildToolDefs(),
	}
}

// Serve runs the dispatch loop until stdin closes or the context is cancelled.
// A single malformed line produces a parse-error response but does not kill
// the loop — clients sometimes recover from one bad message.
func (s *mcpServer) Serve(ctx context.Context) error {
	for s.in.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := s.in.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handleLine(line)
	}
	if err := s.in.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// handleLine parses one envelope and dispatches. Parse failures are written
// back as JSON-RPC parse errors with a null id; the loop keeps running.
func (s *mcpServer) handleLine(line []byte) {
	var req jrpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.writeError(nil, jrpcParseError, "parse error", err.Error())
		return
	}
	if req.JSONRPC != "2.0" {
		s.writeError(req.ID, jrpcInvalidRequest, "jsonrpc must be \"2.0\"", nil)
		return
	}
	// Notifications (no id) are dispatched but produce no response.
	isNotification := len(req.ID) == 0 || string(req.ID) == "null"
	resp, werr := s.dispatch(&req)
	if isNotification {
		return
	}
	if werr != nil {
		s.writeError(req.ID, werr.Code, werr.Message, werr.Data)
		return
	}
	s.writeResult(req.ID, resp)
}

// dispatch routes method calls to the appropriate handler. Returns either a
// result value to be wrapped in `{"result": ...}` or a jrpcError. Tool-call
// failures that are user-visible (unknown tool, invalid DSL, etc.) surface
// as MCP tool results with `isError: true` rather than JSON-RPC errors —
// that mirrors the reference SDK's behavior.
func (s *mcpServer) dispatch(req *jrpcRequest) (interface{}, *jrpcError) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params)
	case "initialized", "notifications/initialized":
		return nil, nil
	case "tools/list":
		return s.handleToolsList()
	case "tools/call":
		return s.handleToolsCall(req.Params)
	case "ping":
		return map[string]interface{}{}, nil
	default:
		return nil, &jrpcError{
			Code:    jrpcMethodNotFound,
			Message: fmt.Sprintf("method %q not found", req.Method),
		}
	}
}

// handleInitialize echoes back the server capabilities. We only advertise
// `tools` — no resources, prompts, sampling, or logging.
func (s *mcpServer) handleInitialize(_ json.RawMessage) (interface{}, *jrpcError) {
	return map[string]interface{}{
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				// listChanged:false — our tool set is static.
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    mcpServerName,
			"version": mcpServerVersion,
		},
	}, nil
}

// handleToolsList returns the static tool catalogue.
func (s *mcpServer) handleToolsList() (interface{}, *jrpcError) {
	return map[string]interface{}{
		"tools": s.tools,
	}, nil
}

// handleToolsCall dispatches to the named tool. The spec says user-facing
// failures should be returned as `{content, isError: true}` in the result,
// not as a JSON-RPC error — reserve JSON-RPC errors for protocol-level
// failures (bad params shape, unknown tool name at the protocol level, etc.).
func (s *mcpServer) handleToolsCall(params json.RawMessage) (interface{}, *jrpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &jrpcError{
				Code:    jrpcInvalidParams,
				Message: "tools/call params must be an object",
				Data:    err.Error(),
			}
		}
	}
	if p.Name == "" {
		return nil, &jrpcError{
			Code:    jrpcInvalidParams,
			Message: "tools/call requires a name",
		}
	}
	result, callErr := s.callTool(p.Name, p.Arguments)
	if callErr != nil {
		// Tool-level error → MCP tool-result with isError.
		return mcpToolError(callErr), nil
	}
	return result, nil
}

// --- wire helpers ---

// writeResult emits a successful response.
func (s *mcpServer) writeResult(id json.RawMessage, result interface{}) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	resp := jrpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	s.writeJSON(resp)
}

// writeError emits a JSON-RPC-level error.
func (s *mcpServer) writeError(id json.RawMessage, code int, msg string, data interface{}) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	resp := jrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jrpcError{Code: code, Message: msg, Data: data},
	}
	s.writeJSON(resp)
}

// writeJSON marshals v as a single line of JSON followed by a newline.
func (s *mcpServer) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		// Very unlikely — best-effort fallback so we don't silently hang.
		data = []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"marshal failure"}}`)
	}
	data = append(data, '\n')
	_, _ = s.out.Write(data)
}

// mcpToolError packages a Go error as an MCP tool-call result with isError=true.
// The JSON error shape from the #716 CLI layer (jsonError) is preserved when
// the wrapper hands it to us explicitly; generic errors become plain text.
func mcpToolError(err error) map[string]interface{} {
	var text string
	var structured *jsonError
	if se, ok := err.(*toolError); ok {
		text = fmt.Sprintf("%s: %s", se.payload.Error, se.payload.Detail)
		structured = &se.payload
	} else {
		text = err.Error()
	}
	result := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": text},
		},
		"isError": true,
	}
	if structured != nil {
		result["structuredContent"] = structured
	}
	return result
}
