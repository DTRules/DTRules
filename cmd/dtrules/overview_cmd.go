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
	"sort"
	"strings"
)

// projectOverview prints a project-aware summary when `dtrules` is run bare
// inside a project: what the config declares, what rules/inputs/traces are
// on disk, and the commands that make sense here. Returns false when the
// current directory doesn't look like a DTRules project (caller falls back
// to generic help).
func (c *CLI) projectOverview() bool {
	root, err := os.Getwd()
	if err != nil {
		return false
	}
	cfg, _ := loadProjectConfig(root)
	xmlDir, _, err := resolveDirs(root, "", "")
	if err != nil || !dirExists(xmlDir) {
		return false
	}

	dtFiles, _ := filepath.Glob(filepath.Join(xmlDir, "*_dt.xml"))
	eddFiles, _ := filepath.Glob(filepath.Join(xmlDir, "*_edd.xml"))
	if len(dtFiles) == 0 && len(eddFiles) == 0 {
		return false
	}

	tables, entities := 0, 0
	for _, f := range dtFiles {
		if data, err := os.ReadFile(f); err == nil {
			tables += strings.Count(string(data), "<table_name>")
		}
	}
	for _, f := range eddFiles {
		if data, err := os.ReadFile(f); err == nil {
			entities += strings.Count(string(data), "<entity ")
		}
	}

	relXML, _ := filepath.Rel(root, xmlDir)
	fmt.Printf("DTRules project: %s\n", root)
	fmt.Printf("  Rules: %s  (%d tables, %d entities)\n", relXML, tables, entities)
	if cfg != nil && cfg.Entry != "" {
		fmt.Printf("  Entry table: %s  (from DTRules.xml)\n", cfg.Entry)
	}

	inputs := findDataFiles(root, xmlDir)
	if len(inputs) > 0 {
		fmt.Println("\nInputs found:")
		for _, in := range limitList(inputs, 8) {
			fmt.Printf("  %s\n", in)
		}
	}
	traces, _ := filepath.Glob(filepath.Join(root, "traces", "*.trace.xml"))
	if len(traces) > 0 {
		fmt.Println("\nTraces:")
		for _, t := range limitList(traces, 8) {
			rel, _ := filepath.Rel(root, t)
			fmt.Printf("  %s\n", rel)
		}
	}

	fmt.Println("\nCommon commands:")
	exampleInput := "<input.xml>"
	if len(inputs) > 0 {
		exampleInput = inputs[0]
	}
	if cfg != nil && cfg.Entry != "" {
		fmt.Printf("  dtrules debug %s\n      run %s on that input and open the debugger\n", exampleInput, cfg.Entry)
	} else {
		fmt.Printf("  dtrules debug %s --entry <table>\n      run a table on that input and open the debugger\n", exampleInput)
	}
	fmt.Println("  dtrules edit .\n      open the editor on this project")
	fmt.Println("  dtrules run . --input <file> --trace\n      run and write a trace to traces/")
	fmt.Println("  dtrules help\n      all commands")
	return true
}

// findDataFiles looks in the conventional data directories (testdata,
// testfiles, input, inputs — up to two levels deep) for XML input files,
// excluding rules (edd/dt/map) and traces. Paths are root-relative.
func findDataFiles(root, xmlDir string) []string {
	var out []string
	seen := map[string]bool{}
	dataDirNames := map[string]bool{"testdata": true, "testfiles": true, "input": true, "inputs": true}

	var scan func(dir string, depth int)
	scan = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if e.IsDir() {
				if depth < 4 && !strings.HasPrefix(e.Name(), ".") && e.Name() != "node_modules" {
					if dataDirNames[strings.ToLower(e.Name())] {
						collectXMLInputs(p, xmlDir, seen, &out)
					} else {
						scan(p, depth+1)
					}
				}
				continue
			}
		}
	}
	scan(root, 0)

	for i, p := range out {
		if rel, err := filepath.Rel(root, p); err == nil {
			out[i] = rel
		}
	}
	sort.Strings(out)
	return out
}

// collectXMLInputs gathers plausible input XML files directly under dir and
// one level down, skipping rules files and traces.
func collectXMLInputs(dir, xmlDir string, seen map[string]bool, out *[]string) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		name := strings.ToLower(filepath.Base(path))
		if !strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".trace.xml") {
			return nil
		}
		if strings.Contains(name, "_edd") || strings.Contains(name, "_dt") || strings.Contains(name, "_map") || strings.Contains(name, "output") {
			return nil
		}
		if strings.HasPrefix(path, xmlDir+string(filepath.Separator)) {
			return nil
		}
		if !seen[path] {
			seen[path] = true
			*out = append(*out, path)
		}
		return nil
	})
}

// limitList truncates a list for display, noting how many were omitted.
func limitList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := make([]string, max+1)
	copy(out, items[:max])
	out[max] = fmt.Sprintf("… and %d more", len(items)-max)
	return out
}
