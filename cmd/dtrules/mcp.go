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

// `dtrules mcp` entry point: runs a Model Context Protocol server over
// stdio so agentic clients (Claude Code, IDEs, etc.) can drive the same
// authoring ops the JSON CLI exposes.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// runMCP parses flags and starts the stdio MCP server.
func (c *CLI) runMCP(args []string) int {
	project := "."
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--project", "-p":
			if i+1 < len(args) {
				project = args[i+1]
				i++
			}
		case "-h", "--help", "help":
			c.printMCPUsage()
			return 0
		default:
			// Unknown flag / positional — fail fast with a message to stderr.
			// We can't emit JSON-RPC noise before stdio takes over.
			fmt.Fprintf(os.Stderr, "dtrules mcp: unknown argument %q\n", a)
			c.printMCPUsage()
			return 1
		}
	}

	absProject, err := absProjectPath(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dtrules mcp: %v\n", err)
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := newMCPServer(os.Stdin, os.Stdout, absProject)
	if err := srv.Serve(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "dtrules mcp: %v\n", err)
		return 1
	}
	return 0
}

// absProjectPath resolves the --project value against the current working
// directory so the server can be invoked from any cwd (including none, in
// the case of editor integrations that set the cwd to `/`).
func absProjectPath(p string) (string, error) {
	if p == "" {
		p = "."
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolving --project: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("--project %s: %w", abs, err)
	}
	return abs, nil
}

func (c *CLI) printMCPUsage() {
	fmt.Println(`Usage: dtrules mcp [--project <path>]

Run a Model Context Protocol server over stdio. Expose the JSON-first
decision-table and EDD surface as MCP tools callable by agent clients such
as Claude Code.

Options:
  --project <path>    Project root (must contain xml/). Default: cwd.
  -h, --help          Show this message.

Transport:
  Newline-delimited JSON-RPC 2.0 on stdin/stdout.

Protocol:
  Model Context Protocol, version ` + mcpProtocolVersion + `.
  See: https://spec.modelcontextprotocol.io

Tools exposed:
  table_list, table_get, table_put, table_patch, table_schema,
  edd_get, edd_put, edd_patch, edd_schema,
  project_validate`)
}
