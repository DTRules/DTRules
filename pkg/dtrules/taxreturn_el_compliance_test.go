//go:build archive

// ARCHIVED: exercises the legacy TaxReturn sample (out of scope; #872 — finishes
// the #520 archive). Run with `go test -tags archive ./...`.

// Copyright 2026 Paul Snow
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

package dtrules_test

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
)

// TestTaxReturn_NoHandCodedPostfix is the project-level enforcement of the
// runtime's hand-coded-postfix gate: every element in every decision
// table (contexts, initial actions, conditions, actions) that has
// non-empty postfix must also carry EL DSL. The authoring API is the
// only supported edit surface — any element bypassing it is rejected.
//
// This matches the rule that `loader.processTable` enforces by setting
// `RDecisionTable.handCodedPostfix`, which `Execute`/`ExecuteTable`
// refuse. The project-level test surfaces the violations all at once
// (with offending kind + number) so they can be authored in bulk
// rather than one-table-at-a-time during execution.
func TestTaxReturn_NoHandCodedPostfix(t *testing.T) {
	cwd, _ := os.Getwd()
	xmlDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn", "xml")

	type dtFile struct {
		Tables []struct {
			Name     string `xml:"table_name"`
			Contexts struct {
				Details []struct {
					Number  int    `xml:"context_number"`
					DSL     string `xml:"context_dsl"`
					Postfix string `xml:"context_postfix"`
				} `xml:"context_details"`
			} `xml:"contexts"`
			InitialActions struct {
				Items []struct {
					IDSL string `xml:"initial_action_dsl"`
					IPF  string `xml:"initial_action_postfix"`
					ADSL string `xml:"action_dsl"`
					APF  string `xml:"action_postfix"`
				} `xml:"initial_action"`
			} `xml:"initial_actions"`
			Conditions struct {
				Items []struct {
					Number  int    `xml:"condition_number"`
					DSL     string `xml:"condition_dsl"`
					Postfix string `xml:"condition_postfix"`
				} `xml:"condition_details"`
			} `xml:"conditions"`
			Actions struct {
				Items []struct {
					Number  int    `xml:"action_number"`
					DSL     string `xml:"action_dsl"`
					Postfix string `xml:"action_postfix"`
				} `xml:"action_details"`
			} `xml:"actions"`
		} `xml:"decision_table"`
	}

	type violation struct {
		file  string
		table string
		desc  string
	}
	var hits []violation

	err := filepath.WalkDir(xmlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_dt.xml") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		var f dtFile
		if xmlErr := xml.Unmarshal(data, &f); xmlErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(xmlDir, path)
		for _, tbl := range f.Tables {
			var entries []decisiontable.PostfixEntry
			for _, c := range tbl.Contexts.Details {
				entries = append(entries, decisiontable.PostfixEntry{
					Kind: "context", Number: c.Number, DSL: c.DSL, Postfix: c.Postfix,
				})
			}
			for i, ia := range tbl.InitialActions.Items {
				dsl := ia.IDSL
				if strings.TrimSpace(dsl) == "" {
					dsl = ia.ADSL
				}
				pf := ia.IPF
				if strings.TrimSpace(pf) == "" {
					pf = ia.APF
				}
				entries = append(entries, decisiontable.PostfixEntry{
					Kind: "initial_action", Number: i + 1, DSL: dsl, Postfix: pf,
				})
			}
			for _, c := range tbl.Conditions.Items {
				entries = append(entries, decisiontable.PostfixEntry{
					Kind: "condition", Number: c.Number, DSL: c.DSL, Postfix: c.Postfix,
				})
			}
			for _, a := range tbl.Actions.Items {
				entries = append(entries, decisiontable.PostfixEntry{
					Kind: "action", Number: a.Number, DSL: a.DSL, Postfix: a.Postfix,
				})
			}
			if !decisiontable.HasAnyHandCodedPostfix(entries) {
				continue
			}
			for _, w := range decisiontable.CheckHandCodedPostfix(tbl.Name, entries) {
				hits = append(hits, violation{file: rel, table: tbl.Name, desc: w.Reason})
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", xmlDir, err)
	}

	if len(hits) == 0 {
		return
	}

	// Group by file → table → list of offending elements, for a readable
	// failure summary.
	type byTable struct {
		name string
		hits []string
	}
	byFile := map[string]map[string][]string{}
	for _, h := range hits {
		if _, ok := byFile[h.file]; !ok {
			byFile[h.file] = map[string][]string{}
		}
		byFile[h.file][h.table] = append(byFile[h.file][h.table], h.desc)
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	t.Errorf("TaxReturn project has %d hand-coded-postfix element(s) across %d file(s); every postfix-bearing element must also carry EL DSL. The runtime refuses to execute these tables.", len(hits), len(files))
	for _, f := range files {
		tables := byFile[f]
		tnames := make([]string, 0, len(tables))
		for n := range tables {
			tnames = append(tnames, n)
		}
		sort.Strings(tnames)
		for _, name := range tnames {
			t.Errorf("  %s :: %s", f, name)
			for _, d := range tables[name] {
				t.Errorf("      %s", d)
			}
		}
	}
	t.Errorf("\nTotal: %d violations. Fix via `dtrules table patch update-<kind>-dsl …` per row.", len(hits))
}

// Compile-time check that fmt is referenced (Go vet otherwise complains
// because no log statement in this file uses it; left in to ease future
// debug additions).
var _ = fmt.Sprintf