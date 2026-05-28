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

package authoring

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// Project holds all loaded decision tables for a DTRules project. It provides
// a typed Go API for reading and mutating tables without touching XML directly.
type Project struct {
	xmlDir      string
	dtFiles     []dtFileEntry // one entry per loaded _dt.xml file
	symbols     map[string]string
	edd         *EDD         // lazily loaded
	execSt      *execState   // lazily-created execution state; nil until first use
	diagnostics []Diagnostic // authoring-time diagnostics (duplicate renames, etc.)

	// OverwriteExcel, when true, suppresses the "Excel newer than last
	// export" guard in Save() / SaveEDD(). Default false: a downstream
	// Excel that's newer than what the sync manifest recorded means a
	// human edited the spreadsheet outside DTRules; silently
	// overwriting it with the XML-derived render would lose work. The
	// guard refuses with sync.ExcelModifiedError and the operator
	// either runs `dtrules build --from-excel` first to merge those
	// changes, or sets this flag to force-overwrite.
	OverwriteExcel bool
}

// dtFileEntry tracks a single decision-table XML file and its in-memory model.
type dtFileEntry struct {
	path   string
	tables *excel.DecisionTablesXML
}

// OpenProject loads a DTRules project from path.
//
// Layout discovery (in order):
//
//  1. `<path>/xml/` — canonical layout. Used by `dtrules init`,
//     sampleprojects, and the build pipeline. Takes precedence.
//  2. `<path>` itself contains `*_dt.xml` — flat layout. Used by
//     library consumers (e.g. staking) whose rules live next to the
//     Go code that embeds them.
//
// Falling back to the flat layout (#791) is what makes `dtrules review`,
// `dtrules table list`, and `dtrules table warnings` reachable from
// projects that don't follow the `xml/` convention. The compile path
// has accepted flat layouts since v1.14.1; the authoring surface now
// matches.
//
// If neither layout matches, returns an error explaining both options.
func OpenProject(path string) (*Project, error) {
	xmlDir, err := resolveProjectXMLDir(path)
	if err != nil {
		return nil, err
	}

	p := &Project{
		xmlDir:  xmlDir,
		symbols: make(map[string]string),
	}

	if err := p.loadEDD(xmlDir); err != nil {
		// non-fatal: continue without symbol table
		_ = err
	}

	if err := p.loadDTFiles(xmlDir); err != nil {
		return nil, err
	}

	p.resolveDuplicateTableNames()

	return p, nil
}

// resolveProjectXMLDir returns the directory to scan for *_dt.xml and
// *_edd.xml. It prefers `<path>/xml/` when present (the canonical
// layout); otherwise it falls back to `<path>` itself if at least one
// `*_dt.xml` lives there. Either way the returned directory is
// guaranteed to contain DT files at load time — callers can treat it
// as the project's effective XML root.
func resolveProjectXMLDir(path string) (string, error) {
	canonical := filepath.Join(path, "xml")
	if info, err := os.Stat(canonical); err == nil && info.IsDir() {
		return canonical, nil
	}
	// Flat layout: accept `path` itself when it contains *_dt.xml.
	matches, _ := filepath.Glob(filepath.Join(path, "*_dt.xml"))
	if len(matches) > 0 {
		return path, nil
	}
	return "", fmt.Errorf("no xml/ subdirectory and no *_dt.xml files found in %s", path)
}

// NewInMemoryProject creates an empty project rooted at dir. An xml/ subdirectory
// is created automatically. No files are read from disk.
func NewInMemoryProject(dir string) (*Project, error) {
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir xml: %w", err)
	}
	p := &Project{
		xmlDir:  xmlDir,
		symbols: make(map[string]string),
		dtFiles: []dtFileEntry{
			{
				path:   filepath.Join(xmlDir, "tables_dt.xml"),
				tables: &excel.DecisionTablesXML{},
			},
		},
	}
	return p, nil
}

// loadEDD parses *_edd.xml files and extracts field names+types for validation.
func (p *Project) loadEDD(xmlDir string) error {
	entries, err := filepath.Glob(filepath.Join(xmlDir, "*_edd.xml"))
	if err != nil {
		return err
	}

	type eddField struct {
		Name string `xml:"name,attr"`
		Type string `xml:"type,attr"`
	}
	type eddEntity struct {
		Name   string     `xml:"name,attr"`
		Fields []eddField `xml:"field"`
	}
	type eddFile struct {
		Entities []eddEntity `xml:"entity"`
	}

	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var f eddFile
		if err := xml.Unmarshal(data, &f); err != nil {
			continue
		}
		for _, ent := range f.Entities {
			for _, field := range ent.Fields {
				key := ent.Name + "." + field.Name
				p.symbols[key] = field.Type
			}
		}
	}
	return nil
}

// loadDTFiles recursively reads all *_dt.xml files under xmlDir. It walks
// nested subdirectories (e.g. xml/states/CO_dt.xml) so authoring operations
// see the same file set as the verify pipeline.
func (p *Project) loadDTFiles(xmlDir string) error {
	var dtPaths []string
	walkErr := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "_dt.xml") {
			dtPaths = append(dtPaths, path)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to list dt files: %w", walkErr)
	}

	for _, path := range dtPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		var tables excel.DecisionTablesXML
		if err := xml.Unmarshal(data, &tables); err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		p.dtFiles = append(p.dtFiles, dtFileEntry{path: path, tables: &tables})
	}

	// If no _dt.xml files found, try any *.xml files that carry decision_tables.
	if len(p.dtFiles) == 0 {
		_ = filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".xml") {
				return nil
			}
			if strings.HasSuffix(path, "_edd.xml") || strings.HasSuffix(path, "_map.xml") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			if !strings.Contains(string(data), "<decision_tables") {
				return nil
			}
			var tables excel.DecisionTablesXML
			if err := xml.Unmarshal(data, &tables); err != nil {
				return nil
			}
			p.dtFiles = append(p.dtFiles, dtFileEntry{path: path, tables: &tables})
			return nil
		})
	}

	return nil
}

// Save writes all pending mutations back to disk as canonical XML files
// AND refreshes any Excel workbook the project's sync manifest points at,
// keeping the "Excel is system of record" contract honored on every
// authoring write.
//
// The flow:
//
//  1. Locate the sync manifest (the `.sync-manifest.json` that
//     `dtrules build` maintains beside the project's Excel files).
//     If no manifest exists, the project is treated as XML-only and
//     Save behaves like the pre-v1.14.5 version — XML write only.
//
//  2. For each Excel file the manifest covers, run ExportGuard to
//     refuse the write if the user has touched Excel since the last
//     export (mtime-based). The error type is
//     `sync.ExcelModifiedError`; set `OverwriteExcel = true` on the
//     project to bypass this guard.
//
//  3. Refuse the write if any covered Excel file is locked by another
//     process (`~$<name>.xlsx` or `.~lock.<name>.xlsx#` siblings).
//     Writing through a lock corrupts the file from the other app's
//     point of view.
//
//  4. Write XML (as before).
//
//  5. Re-export every covered Excel from the in-memory project so
//     XML and Excel stay byte-paired on disk. Update the manifest's
//     last-export record afterward so the next Save's guard sees a
//     fresh baseline.
//
// XML is always written if the guards pass. If the Excel refresh
// fails (e.g. mid-write disk error), the XML write has already
// happened; Save returns a wrapped error naming the failed Excel
// path so the operator can recover.
func (p *Project) Save() error {
	if err := p.preWriteExcelGuard(); err != nil {
		return err
	}

	imp := excel.NewDTImporter()
	for _, entry := range p.dtFiles {
		if err := imp.WriteXML(entry.tables, entry.path); err != nil {
			return fmt.Errorf("failed to write %s: %w", entry.path, err)
		}
	}
	if err := p.saveEDDXMLOnly(); err != nil {
		return err
	}
	return p.refreshExcelFromXML()
}

// preWriteExcelGuard is the Project-method facade over GuardExcelInDir.
// It threads the project's OverwriteExcel flag through and otherwise
// delegates straight to the package-level helper, so the same code
// path runs whether the caller is Project.Save or a non-Project
// surface like dtrules compile.
func (p *Project) preWriteExcelGuard() error {
	return GuardExcelInDir(p.xmlDir, p.OverwriteExcel)
}

// refreshExcelFromXML re-exports every Excel file the sync manifest
// covers from the project's freshly-written XML state. Called after
// the XML write so XML and Excel are byte-paired on disk by the time
// Save returns.
//
// The export goes through `excel.NewExporter(rs)` which requires a
// `*session.RuleSet`. Loading the RuleSet here pays a one-time cost
// per Save — modest for a single-workbook project (staking), more
// noticeable on TaxReturn-scale corpora. Tolerant load so DSL-only
// elements (no postfix) don't error out; Save is an authoring
// surface and the operator may be in the middle of an edit.
//
// Manifest is updated via RecordExport after each successful Excel
// write so the next Save's ExportGuard sees the fresh baseline.
//
// No-manifest projects skip this whole step (same logic as the
// preWriteExcelGuard) — Save behaves as the legacy XML-only writer
// for them.
// refreshExcelFromXML is the Project-method facade over
// RefreshExcelInDir. Same pattern as preWriteExcelGuard: delegate to
// the package-level helper so the same code path runs from any caller
// (Save, SaveEDD, dtrules compile).
func (p *Project) refreshExcelFromXML() error {
	return RefreshExcelInDir(p.xmlDir)
}

// Tables lists the names of every decision table in the project.
func (p *Project) Tables() []string {
	var names []string
	for _, entry := range p.dtFiles {
		for _, t := range entry.tables.Tables {
			names = append(names, t.TableName)
		}
	}
	return names
}

// Table returns a typed view of the named decision table, or nil if not found.
func (p *Project) Table(name string) *Table {
	for fi := range p.dtFiles {
		for ti := range p.dtFiles[fi].tables.Tables {
			if p.dtFiles[fi].tables.Tables[ti].TableName == name {
				return newTableWithProject(&p.dtFiles[fi].tables.Tables[ti], p.symbols, p)
			}
		}
	}
	return nil
}

// AddTable creates a new decision table with the given name.
func (p *Project) AddTable(name string) (*Table, error) {
	for _, entry := range p.dtFiles {
		for _, t := range entry.tables.Tables {
			if t.TableName == name {
				return nil, fmt.Errorf("table %q already exists", name)
			}
		}
	}
	xmlTable := excel.DecisionTableXML{
		TableName: name,
		AttributeFields: excel.AttributeFieldsXML{
			Type: "FIRST",
		},
	}
	if len(p.dtFiles) == 0 {
		// Create a new file entry using conventional naming
		path := filepath.Join(p.xmlDir, strings.ToLower(name)+"_dt.xml")
		p.dtFiles = append(p.dtFiles, dtFileEntry{
			path:   path,
			tables: &excel.DecisionTablesXML{},
		})
	}
	// Append to the first file
	p.dtFiles[0].tables.Tables = append(p.dtFiles[0].tables.Tables, xmlTable)
	last := &p.dtFiles[0].tables.Tables[len(p.dtFiles[0].tables.Tables)-1]
	return newTableWithProject(last, p.symbols, p), nil
}

// DeleteEntity removes the named entity from the EDD, checking that no loaded
// decision table references it as a context first.
func (p *Project) DeleteEntity(name string) error {
	return p.EDD().deleteEntityChecked(name, p.dtFiles)
}

// DeleteTable removes the named table.
func (p *Project) DeleteTable(name string) error {
	for fi := range p.dtFiles {
		tables := p.dtFiles[fi].tables.Tables
		for ti, t := range tables {
			if t.TableName == name {
				p.dtFiles[fi].tables.Tables = append(tables[:ti], tables[ti+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("table %q not found", name)
}
