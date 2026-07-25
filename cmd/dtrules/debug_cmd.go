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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// traceDefaultSentinel marks a bare --trace flag: the path is derived from
// the project layout after argument parsing (traces/<input>.trace.xml).
const traceDefaultSentinel = "\x00default"

// defaultTracePath places a trace under <project>/traces/, named after the
// input file (or the entry table when there is no input). The directory is
// created so the very first `dtrules run --trace` in a project just works.
func defaultTracePath(projectRoot, entry, input string) string {
	stem := entry
	if input != "" {
		stem = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	}
	dir := filepath.Join(projectRoot, "traces")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, stem+".trace.xml")
}

// runDebug handles `dtrules debug [input.xml] [options]` — the one-command
// debugging workflow: run the project's entry table over the input while
// tracing, then open the editor with the trace already loaded in the Debug
// tab. The entry table comes from --entry or the project's DTRules.xml.
func (c *CLI) runDebug(args []string) int {
	path, entry, input := ".", "", ""
	var passthrough []string // flags forwarded to `edit` (--port, --no-browser)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--entry":
			if i+1 < len(args) {
				entry = args[i+1]
				i++
			}
		case "--input":
			if i+1 < len(args) {
				input = args[i+1]
				i++
			}
		case "--port", "-p", "--host":
			if i+1 < len(args) {
				passthrough = append(passthrough, args[i], args[i+1])
				i++
			}
		case "--no-browser":
			passthrough = append(passthrough, args[i])
		case "-h", "--help":
			fmt.Println(`Usage: dtrules debug [input.xml] [options]

Runs the project's entry decision table over the input file while recording
a trace, then opens the editor with the trace loaded in the Debug tab.

The entry table is taken from --entry, or from <entry> in the project's
DTRules.xml. The trace is written to traces/<input>.trace.xml.

Arguments:
  input.xml          Input data file (loaded via the project mapping).
                     A directory argument sets the project root (default .).

Options:
  --entry <table>    Entry decision table (overrides DTRules.xml)
  --input <file>     Alternative way to name the input file
  --port, -p <n>     Editor port (default 8080)
  --no-browser       Don't open the browser automatically

Example (from a project root with <entry> declared in DTRules.xml):
  dtrules debug pkg/dtrules/testdata/period174.xml`)
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				return 1
			}
			// A directory sets the project root; a file is the input.
			if info, err := os.Stat(args[i]); err == nil && info.IsDir() {
				path = args[i]
			} else {
				input = args[i]
			}
		}
	}

	root := mustAbs(path)
	if entry == "" {
		if cfg, err := loadProjectConfig(root); err == nil {
			entry = cfg.Entry
		}
	}
	if entry == "" {
		fmt.Fprintln(os.Stderr, "Error: no entry table — pass --entry <table> or declare <entry> in DTRules.xml")
		return 1
	}

	tracePath := defaultTracePath(root, entry, input)

	runArgs := []string{path, "--entry", entry, "--trace", tracePath}
	if input != "" {
		runArgs = append(runArgs, "--input", input)
	}
	fmt.Printf("Running %s%s → %s\n", entry,
		map[bool]string{true: " on " + input, false: ""}[input != ""], tracePath)
	if code := c.runRun(runArgs); code != 0 {
		// The partial trace is still on disk; the editor can often load it
		// for post-mortem, but don't auto-open on a failed run.
		fmt.Fprintf(os.Stderr, "Run failed — not opening the editor. Partial trace (if any): %s\n", tracePath)
		return code
	}

	editArgs := append([]string{path, "--trace", tracePath}, passthrough...)
	return c.runEdit(editArgs)
}
