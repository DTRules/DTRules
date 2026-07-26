// Copyright 2026 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// runReport handles `dtrules report <trace.xml> --spec <file> [options]`:
// generate an EDD-driven report from a trace, optionally diffed against a
// baseline trace run with the same spec.
func (c *CLI) runReport(args []string) int {
	path, specPath, baselinePath, tracePath := ".", "", "", ""
	asJSON := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--spec":
			if i+1 < len(args) {
				specPath = args[i+1]
				i++
			}
		case "--baseline":
			if i+1 < len(args) {
				baselinePath = args[i+1]
				i++
			}
		case "--project":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
		case "--json":
			asJSON = true
		case "-h", "--help":
			fmt.Println(`Usage: dtrules report <trace.xml> --spec <report.json> [options]

Generates an EDD-driven report from a trace: the spec picks entities (all
instances the run created, or elements of an array like
staking_transaction.to), their fields, filters, and sort order.

Options:
  --spec <file>      Report spec (JSON). Sections:
                     {"entity"|"source", "fields", "where", "sort", "key"}
  --baseline <file>  A second trace: the report runs on both and the
                     row-level diff (added/removed/changed) is appended
  --project <dir>    Project root (default .)
  --json             Emit JSON instead of markdown

Example spec:
  {"name": "Payouts", "sections": [
    {"title": "Recipients", "source": "staking_transaction.to",
     "fields": ["url", "amount"], "sort": "amount", "key": "url"}]}`)
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				return 1
			}
			if tracePath == "" {
				tracePath = args[i]
			}
		}
	}
	if tracePath == "" || specPath == "" {
		fmt.Fprintln(os.Stderr, "Error: dtrules report <trace.xml> --spec <report.json>")
		return 1
	}

	specData, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading spec: %v\n", err)
		return 1
	}
	var spec trace.ReportSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing spec: %v\n", err)
		return 1
	}
	if spec.Name == "" {
		spec.Name = strings.TrimSuffix(filepath.Base(specPath), ".report.json")
	}

	xmlDir, _, err := resolveDirs(mustAbs(path), "", "")
	if err != nil || !dirExists(xmlDir) {
		fmt.Fprintf(os.Stderr, "Error: could not find rules under %s\n", path)
		return 1
	}
	rs := session.NewRuleSet("report")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		return 1
	}

	report, err := reportFromTrace(rs, tracePath, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	var diff *trace.ReportDiff
	if baselinePath != "" {
		base, err := reportFromTrace(rs, baselinePath, spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on baseline: %v\n", err)
			return 1
		}
		diff = trace.DiffReports(base, report)
	}

	if asJSON {
		out := map[string]interface{}{"report": report}
		if diff != nil {
			out["diff"] = diff
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	fmt.Print(report.Markdown())
	if diff != nil {
		fmt.Print(diffMarkdown(diff))
	}
	return 0
}

// reportFromTrace loads a trace, replays to its final state, and runs spec.
func reportFromTrace(rs *session.RuleSet, tracePath string, spec trace.ReportSpec) (*trace.Report, error) {
	tr := trace.NewTrace()
	if _, err := tr.Load(tracePath); err != nil {
		return nil, fmt.Errorf("load trace %s: %w", tracePath, err)
	}
	fs := tr.FinalState()
	if fs == nil {
		return nil, fmt.Errorf("%s records no finalState", tracePath)
	}
	sess, err := tr.SetState(rs, fs)
	if err != nil {
		return nil, fmt.Errorf("replay %s: %w", tracePath, err)
	}
	return tr.GenerateReport(sess, spec), nil
}

// diffMarkdown renders a report diff compactly.
func diffMarkdown(d *trace.ReportDiff) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n# Changes vs baseline\n")
	for _, s := range d.Sections {
		if len(s.Added) == 0 && len(s.Removed) == 0 && len(s.Changed) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "\n## %s — %d added, %d removed, %d changed\n\n",
			s.Title, len(s.Added), len(s.Removed), len(s.Changed))
		for _, r := range s.Added {
			fmt.Fprintf(&sb, "+ %s\n", rowLine(r, s.Fields))
		}
		for _, r := range s.Removed {
			fmt.Fprintf(&sb, "- %s\n", rowLine(r, s.Fields))
		}
		for _, ch := range s.Changed {
			fmt.Fprintf(&sb, "~ %s:", ch.Key)
			for _, f := range ch.Fields {
				fmt.Fprintf(&sb, "  %s %s → %s", f, ch.Before[f], ch.After[f])
			}
			fmt.Fprintln(&sb)
		}
	}
	if len(d.Sections) == 0 {
		fmt.Fprintln(&sb, "\n_no sections_")
	}
	return sb.String()
}

// rowLine formats one row on a single line, fields in order.
func rowLine(row map[string]string, fields []string) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s=%s", f, row[f]))
	}
	return strings.Join(parts, "  ")
}
