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

// Compile-time gate for unresolved duplicate decision-table names.
//
// Two failure conditions:
//   1. A table name matches the regex `^(.+)-(\d+)$` — the reserved suffix
//      marker assigned by the authoring SDK when it detected a duplicate on a
//      previous load. Its presence on disk means a human/agent still has to
//      pick a final name.
//   2. Two or more XML files declare the same literal <table_name> — the
//      authoring SDK hasn't run yet and the rulesets are ambiguous.
//
// Both errors must fire from `dtrules build`, `dtrules validate`, and
// `dtrules verify` before any downstream step runs.

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// dupSuffixRe matches table names auto-assigned when duplicates were
// detected on a previous authoring-SDK load.
var dupSuffixRe = regexp.MustCompile(`^(.+)-(\d+)$`)

// dupCheckFinding is one duplicate/suffix violation. Only one of `Suffix`
// (layer-1 marker survived on disk) or `Files` (two or more files still hold
// the same real name) is populated for a given finding.
type dupCheckFinding struct {
	Name   string
	Suffix bool
	Files  []string
}

// Message renders the finding as the human-readable error the issue spec
// mandates. A trailing newline is included.
func (f dupCheckFinding) Message() string {
	var b strings.Builder
	if f.Suffix {
		fmt.Fprintf(&b, "ERROR: decision table %q uses the `-N` suffix reserved\n", f.Name)
		b.WriteString("for unresolved duplicates.\n")
		if len(f.Files) > 0 {
			fmt.Fprintf(&b, "  Defined in: %s\n", f.Files[0])
		}
		b.WriteString("\n")
		b.WriteString("This name was auto-assigned when duplicates were detected on a previous load.\n")
		b.WriteString("Rename this table (or delete it, or rename the conflicting copy) and remove\n")
		b.WriteString("the `-N` suffix before building.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "ERROR: decision table %q is declared in %d files:\n", f.Name, len(f.Files))
	for _, f := range f.Files {
		fmt.Fprintf(&b, "  - %s\n", f)
	}
	b.WriteString("\n")
	b.WriteString("Open the project with the authoring SDK (dtrules project diagnostics) to\n")
	b.WriteString("auto-assign a `-N` suffix to each duplicate, then rename or delete one of\n")
	b.WriteString("them so every <table_name> is unique.\n")
	return b.String()
}

// checkNoDupMarkers scans every _dt.xml under xmlDir and returns findings
// for any table that either carries the `-N` marker or shares its name with
// another table on disk. Findings are sorted by name for stable output.
//
// Setting DTRULES_ALLOW_DUPLICATE_TABLES=1 disables the check entirely. This
// is an escape hatch for test fixtures that ship with pre-existing duplicates
// and assert unrelated behavior; production callers should never set it.
func checkNoDupMarkers(xmlDir string) ([]dupCheckFinding, error) {
	if xmlDir == "" {
		return nil, nil
	}
	if os.Getenv("DTRULES_ALLOW_DUPLICATE_TABLES") == "1" {
		return nil, nil
	}
	if info, err := os.Stat(xmlDir); err != nil || !info.IsDir() {
		return nil, nil
	}

	type entry struct {
		name string
		file string
	}
	var entries []entry

	walkErr := filepath.WalkDir(xmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_dt.xml") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // non-fatal: skip unreadable file
		}
		var root struct {
			Tables []struct {
				TableName string `xml:"table_name"`
			} `xml:"decision_table"`
		}
		if err := xml.Unmarshal(data, &root); err != nil {
			return nil // non-XML or malformed — other checks surface it
		}
		for _, t := range root.Tables {
			if t.TableName == "" {
				continue
			}
			entries = append(entries, entry{name: t.TableName, file: path})
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	byName := make(map[string][]string)
	var order []string
	for _, e := range entries {
		if _, ok := byName[e.name]; !ok {
			order = append(order, e.name)
		}
		byName[e.name] = append(byName[e.name], e.file)
	}
	sort.Strings(order)

	var findings []dupCheckFinding
	for _, name := range order {
		files := byName[name]
		if dupSuffixRe.MatchString(name) {
			findings = append(findings, dupCheckFinding{
				Name:   name,
				Suffix: true,
				Files:  files,
			})
			continue
		}
		if len(files) > 1 {
			sort.Strings(files)
			findings = append(findings, dupCheckFinding{
				Name:  name,
				Files: files,
			})
		}
	}
	return findings, nil
}

// writeDupFindings prints each finding to w, separating blocks with a blank
// line.
func writeDupFindings(w *os.File, findings []dupCheckFinding) {
	for i, f := range findings {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprint(w, f.Message())
	}
}
