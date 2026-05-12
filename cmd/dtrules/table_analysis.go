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
	"fmt"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
)

// analyzeAuthoringTable runs the per-table advisory pass on a single typed
// table. The authoring channel surfaces these warnings from
// `dtrules table get / put / patch` and `dtrules table warnings` so an
// agent or human editing one table at a time sees structural findings
// without paying the cost of a project-wide review.
//
// This is the authoring half of the advisory split (#761): structural
// checks only, fast, XML-driven. Tree-based checks (#765, #766) and
// cross-table analysis run from the Full Review (#768).
func analyzeAuthoringTable(t *authoring.Table) []decisiontable.Warning {
	if t == nil {
		return nil
	}
	maxCol := t.Columns()
	if maxCol == 0 {
		return nil
	}
	conds := make([]decisiontable.ConditionRow, len(t.Conditions))
	for i, c := range t.Conditions {
		cols := make([]string, maxCol)
		for n, v := range c.Columns {
			if n >= 1 && n <= maxCol {
				cols[n-1] = v
			}
		}
		conds[i] = decisiontable.ConditionRow{DSL: c.DSL, Columns: cols}
	}
	acts := make([]decisiontable.ActionRow, len(t.Actions))
	for i, a := range t.Actions {
		cols := make([]string, maxCol)
		for n, executes := range a.Columns {
			if executes && n >= 1 && n <= maxCol {
				cols[n-1] = "X"
			}
		}
		acts[i] = decisiontable.ActionRow{DSL: a.DSL, Columns: cols}
	}
	return decisiontable.AnalyzeTable(t.Name, conds, acts, maxCol)
}

// warningsForJSON marshals a warning slice into a stable shape for the
// CLI / MCP wire formats. Empty input renders as `[]` (never `null`) so
// consumers can iterate unconditionally.
func warningsForJSON(ws []decisiontable.Warning) []decisiontable.Warning {
	if ws == nil {
		return []decisiontable.Warning{}
	}
	return ws
}

// tableWarnings handles `dtrules table warnings <name>`: a read-only fetch
// that runs the authoring-channel checks and returns the JSON shape locked
// by TestWarningJSONShape.
func (ctx *tableCmdCtx) tableWarnings(rest []string) int {
	name, code := ctx.requireName(rest, "table warnings <name>")
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
	warns := warningsForJSON(analyzeAuthoringTable(t))
	if err := writeJSON(ctx.stdout, map[string]interface{}{
		"table":    t.Name,
		"warnings": warns,
	}); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}
