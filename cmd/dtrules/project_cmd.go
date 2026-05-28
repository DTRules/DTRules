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

// `dtrules project …` — project-level JSON surface.
//
// Currently only one op: `diagnostics`. Exits 0 regardless of the diagnostic
// count so CI pipelines can use it as an observation tool rather than a gate.
// The compile-time gate lives in build/validate/verify.

import (
	"fmt"
	"os"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// runProject dispatches `dtrules project ...`.
func (c *CLI) runProject(args []string) int {
	ctx := &tableCmdCtx{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	if len(args) == 0 {
		c.printProjectUsage()
		return 1
	}
	sub := args[0]
	rest := args[1:]
	ctx.projectPath, ctx.forceOverwriteExcel, rest = parseProjectFlag(rest)

	switch sub {
	case "diagnostics":
		return ctx.projectDiagnostics()
	case "help", "-h", "--help":
		c.printProjectUsage()
		return 0
	default:
		_ = rest
		return emitErr(ctx.stderr, 1, "invalid_command", "", "known: diagnostics",
			fmt.Sprintf("unknown project subcommand %q", sub))
	}
}

// projectDiagnostics emits authoring-time diagnostics (duplicate renames,
// etc.) as JSON. Exits 0 even when diagnostics are present.
func (ctx *tableCmdCtx) projectDiagnostics() int {
	p, err := authoring.OpenProject(ctx.projectPath)
	if err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "project must contain an xml/ directory", err.Error())
	}
	diags := p.Diagnostics()
	if diags == nil {
		diags = []authoring.Diagnostic{}
	}
	out := map[string]interface{}{"diagnostics": diags}
	if err := writeJSON(ctx.stdout, out); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

func (c *CLI) printProjectUsage() {
	fmt.Println(`Usage: dtrules project <command> [--project <path>]

Commands:
  diagnostics              Emit authoring-time diagnostics as JSON.
                           Exits 0 regardless of diagnostic count — use
                           dtrules validate/verify/build as the hard gate.

Diagnostic kinds:
  duplicate_table          A <table_name> collided with another on load;
                           the second occurrence was renamed in memory to
                           <original>-1, -2, ... and will persist on Save().`)
}
