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

package authoring

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/loader"
)

// discoverWorkbookPairing answers "which workbook does each XML file belong
// to" by reading the artifacts instead of a side file.
//
// Every table and every entity already records its workbook in a `<source>`
// block. That is the same pairing `.sync-manifest.json` held, except it
// travels: the manifest is gitignored, so the pairing never left the machine
// that built it. A fresh clone had no manifest, so the guard and the refresh
// silently did nothing at all -- both are documented as no-ops on projects
// without one, which described every clone anyone had ever made (#1091).
//
// Keyed by absolute workbook path, valued by the XML files that named it.
// Workbooks are resolved against excelDir, falling back to the recorded
// relative path when that does not exist, because a project's workbooks are
// not always flat.
func discoverWorkbookPairing(xmlDir, excelDir string) map[string][]string {
	pairing := make(map[string][]string)

	add := func(workbook, xmlPath string) {
		name := strings.TrimSpace(workbook)
		if name == "" {
			return
		}
		abs := filepath.Join(excelDir, name)
		if _, err := os.Stat(abs); err != nil {
			// Recorded as a path rather than a base name, or nested.
			if alt := filepath.Join(excelDir, filepath.Base(name)); alt != abs {
				if _, err2 := os.Stat(alt); err2 == nil {
					abs = alt
				}
			}
		}
		for _, seen := range pairing[abs] {
			if seen == xmlPath {
				return
			}
		}
		pairing[abs] = append(pairing[abs], xmlPath)
	}

	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if filepath.Ext(name) != ".xml" || loader.SkipRuleFile(p) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		switch {
		case strings.HasSuffix(name, "_dt.xml"):
			var doc excel.DecisionTablesXML
			if xml.Unmarshal(data, &doc) != nil {
				return nil
			}
			for i := range doc.Tables {
				if src := doc.Tables[i].Source; src != nil {
					add(sourceWorkbook(src), p)
				}
			}
		case strings.HasSuffix(name, "_edd.xml"):
			var doc excel.EDDXML
			if xml.Unmarshal(data, &doc) != nil {
				return nil
			}
			for _, ent := range doc.Entities {
				if ent == nil {
					continue
				}
				if ent.Source != nil {
					add(sourceWorkbook(ent.Source), p)
				} else if ent.XlsFile != "" {
					add(ent.XlsFile, p)
				}
			}
		}
		return nil
	})

	for k := range pairing {
		sort.Strings(pairing[k])
	}
	return pairing
}

// sourceWorkbook prefers the file name a source block records, falling back to
// its relative path. Both spellings appear in the corpus.
func sourceWorkbook(src *excel.SourceXML) string {
	if n := strings.TrimSpace(src.FileName); n != "" {
		return n
	}
	return strings.TrimSpace(src.RelativePath)
}
