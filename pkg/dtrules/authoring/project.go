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
	xmlDir    string
	dtFiles   []dtFileEntry // one entry per loaded _dt.xml file
	symbols   map[string]string
	edd       *EDD // lazily loaded
}

// dtFileEntry tracks a single decision-table XML file and its in-memory model.
type dtFileEntry struct {
	path   string
	tables *excel.DecisionTablesXML
}

// OpenProject loads a DTRules project from path. It looks for an xml/ subdirectory
// by convention and reads all *_dt.xml files from it.
func OpenProject(path string) (*Project, error) {
	xmlDir := filepath.Join(path, "xml")
	if _, err := os.Stat(xmlDir); err != nil {
		return nil, fmt.Errorf("no xml/ directory found in %s", path)
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

// loadDTFiles reads all *_dt.xml files from xmlDir.
func (p *Project) loadDTFiles(xmlDir string) error {
	entries, err := filepath.Glob(filepath.Join(xmlDir, "*_dt.xml"))
	if err != nil {
		return fmt.Errorf("failed to list dt files: %w", err)
	}

	for _, path := range entries {
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

	// If no _dt.xml files found, try any _dt.xml suffix-less XML files
	if len(p.dtFiles) == 0 {
		all, _ := filepath.Glob(filepath.Join(xmlDir, "*.xml"))
		for _, path := range all {
			if strings.HasSuffix(path, "_edd.xml") || strings.HasSuffix(path, "_map.xml") {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if !strings.Contains(string(data), "<decision_tables") {
				continue
			}
			var tables excel.DecisionTablesXML
			if err := xml.Unmarshal(data, &tables); err != nil {
				continue
			}
			p.dtFiles = append(p.dtFiles, dtFileEntry{path: path, tables: &tables})
		}
	}

	return nil
}

// Save writes all pending mutations back to disk as canonical XML files.
// The output uses the same format as the DTImporter.WriteXML pipeline.
func (p *Project) Save() error {
	imp := excel.NewDTImporter()
	for _, entry := range p.dtFiles {
		if err := imp.WriteXML(entry.tables, entry.path); err != nil {
			return fmt.Errorf("failed to write %s: %w", entry.path, err)
		}
	}
	return p.SaveEDD()
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
				return newTable(&p.dtFiles[fi].tables.Tables[ti], p.symbols)
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
	return newTable(last, p.symbols), nil
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
