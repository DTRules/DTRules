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

// Package analysis provides project-wide static analysis for DTRules projects.
package analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EDDWarning records an informational finding about EDD field usage.
type EDDWarning struct {
	Field   string // "entity.attribute"
	EddFile string // source EDD file name
	Reason  string
}

// String formats the warning in the canonical INFO form.
func (w EDDWarning) String() string {
	return fmt.Sprintf("INFO %s (%s)", w.Reason, w.EddFile)
}

// AnalyzeEDDUsage walks all *_dt.xml files under xmlDir, collects every
// identifier referenced in DSL fields, then diffs against the EDD declared
// in eddFile. It returns informational warnings for unused and write-only fields.
func AnalyzeEDDUsage(xmlDir string) ([]EDDWarning, error) {
	// Load all EDD declarations.
	eddDecls, err := loadEDDDeclarations(xmlDir)
	if err != nil {
		return nil, err
	}

	// Collect identifiers referenced in DT XML files.
	readRefs, writeRefs, err := collectDTReferences(xmlDir)
	if err != nil {
		return nil, err
	}

	return diffEDDUsage(eddDecls, readRefs, writeRefs), nil
}

// eddField is a declared EDD field.
type eddField struct {
	EntityAttr string // "entity.attr"
	Access     string // "r", "rw", etc.
	EddFile    string
}

// loadEDDDeclarations reads all *_edd.xml files under xmlDir and returns
// declared fields keyed by "entity.attr".
func loadEDDDeclarations(xmlDir string) (map[string]eddField, error) {
	decls := make(map[string]eddField)

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if !strings.HasSuffix(name, "_edd.xml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		var edd struct {
			Entities []struct {
				Name   string `xml:"name,attr"`
				Fields []struct {
					Name   string `xml:"name,attr"`
					Access string `xml:"access,attr"`
				} `xml:"field"`
			} `xml:"entity"`
		}
		if err := xml.Unmarshal(data, &edd); err != nil {
			// Not a valid EDD file — skip silently.
			return nil
		}

		for _, entity := range edd.Entities {
			entityName := strings.ToLower(entity.Name)
			for _, field := range entity.Fields {
				key := entityName + "." + strings.ToLower(field.Name)
				if isRuntimeReserved(key) {
					continue
				}
				decls[key] = eddField{
					EntityAttr: key,
					Access:     field.Access,
					EddFile:    name,
				}
			}
		}
		return nil
	})
	return decls, err
}

// identifierPattern matches dotted identifiers like "job.income" or "taxpayer.agi".
var identifierPattern = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\b`)

// setPrefix matches LHS assignment: "set job.foo" or "/job.foo xdef" or "/job.foo set"
var assignPattern = regexp.MustCompile(`(?i)(?:set\s+([a-z_][a-z0-9_]*)\.([a-z0-9_]+)|/([a-z_][a-z0-9_]*)\.([a-z0-9_]+)\s+(?:xdef|set)|/([a-z_][a-z0-9_]*)\.([a-z0-9_]+)\s+swap\s+addto)`)

// createEntityPattern detects "new X entity" or "createentity" near entity push
var createEntityPattern = regexp.MustCompile(`(?i)new\s+([a-z_][a-z0-9_]*)\s+entity`)

// collectDTReferences walks all *_dt.xml files and collects identifiers.
// readRefs: identifiers appearing in read positions (conditions, RHS of assignments).
// writeRefs: identifiers appearing as LHS of assignment statements.
func collectDTReferences(xmlDir string) (readRefs map[string]bool, writeRefs map[string]bool, err error) {
	readRefs = make(map[string]bool)
	writeRefs = make(map[string]bool)

	err = filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if !strings.HasSuffix(name, "_dt.xml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		var tables struct {
			Tables []struct {
				InitialActions []struct {
					DSL     string `xml:"initial_action_dsl"`
					Postfix string `xml:"action_postfix"`
				} `xml:"initial_actions>initial_action"`
				Conditions []struct {
					DSL     string `xml:"condition_dsl"`
					Postfix string `xml:"condition_postfix"`
				} `xml:"conditions>condition_details"`
				Actions []struct {
					DSL     string `xml:"action_dsl"`
					Postfix string `xml:"action_postfix"`
				} `xml:"actions>action_details"`
			} `xml:"decision_table"`
		}
		if err := xml.Unmarshal(data, &tables); err != nil {
			return nil // skip unparseable
		}

		for _, table := range tables.Tables {
			// Conditions are always reads.
			for _, c := range table.Conditions {
				extractReads(c.DSL, readRefs)
				extractReads(c.Postfix, readRefs)
			}
			// Actions: extract writes explicitly, then reads for non-write positions.
			for _, a := range table.InitialActions {
				extractWritesAndReads(a.DSL, writeRefs, readRefs)
				extractWritesAndReads(a.Postfix, writeRefs, readRefs)
			}
			for _, a := range table.Actions {
				extractWritesAndReads(a.DSL, writeRefs, readRefs)
				extractWritesAndReads(a.Postfix, writeRefs, readRefs)
			}
		}
		return nil
	})
	return readRefs, writeRefs, err
}

// extractReads collects all dotted identifiers from text as read references.
func extractReads(text string, into map[string]bool) {
	for _, m := range identifierPattern.FindAllStringSubmatch(strings.ToLower(text), -1) {
		key := m[1] + "." + m[2]
		if !isRuntimeReserved(key) {
			into[key] = true
		}
	}
}

// extractWritesAndReads separates write targets (LHS of set/xdef) from read references.
// Write targets go into writes; all other identifiers go into reads.
func extractWritesAndReads(text string, writes, reads map[string]bool) {
	lower := strings.ToLower(text)

	// Collect write targets first.
	localWrites := make(map[string]bool)
	for _, m := range assignPattern.FindAllStringSubmatch(lower, -1) {
		pairs := [][2]string{
			{m[1], m[2]},
			{m[3], m[4]},
			{m[5], m[6]},
		}
		for _, p := range pairs {
			if p[0] != "" && p[1] != "" {
				key := p[0] + "." + p[1]
				if !isRuntimeReserved(key) {
					localWrites[key] = true
					writes[key] = true
				}
			}
		}
	}

	// All other identifiers are reads.
	for _, m := range identifierPattern.FindAllStringSubmatch(lower, -1) {
		key := m[1] + "." + m[2]
		if !isRuntimeReserved(key) && !localWrites[key] {
			reads[key] = true
		}
	}
}

// isRuntimeReserved returns true for identifiers that the runtime manages
// and should not be flagged as unused.
func isRuntimeReserved(key string) bool {
	// mapping*key and other known runtime fields
	if strings.Contains(key, "*") {
		return true
	}
	reserved := []string{
		"mapping.key",
		"constants.",
	}
	for _, r := range reserved {
		if strings.HasPrefix(key, r) {
			return true
		}
	}
	return false
}

// diffEDDUsage compares declarations against references and produces warnings.
func diffEDDUsage(decls map[string]eddField, readRefs, writeRefs map[string]bool) []EDDWarning {
	var warnings []EDDWarning

	for key, f := range decls {
		isRead := readRefs[key]
		isWritten := writeRefs[key]

		// Fields with access="r" are input-only (no decision table sets them).
		if f.Access == "r" {
			if !isRead {
				warnings = append(warnings, EDDWarning{
					Field:   key,
					EddFile: f.EddFile,
					Reason:  fmt.Sprintf("unused EDD field: %s (declared in %s, never referenced)", key, f.EddFile),
				})
			}
			continue
		}

		if !isRead && !isWritten {
			warnings = append(warnings, EDDWarning{
				Field:   key,
				EddFile: f.EddFile,
				Reason:  fmt.Sprintf("unused EDD field: %s (declared in %s, never referenced)", key, f.EddFile),
			})
		} else if isWritten && !isRead {
			warnings = append(warnings, EDDWarning{
				Field:   key,
				EddFile: f.EddFile,
				Reason:  fmt.Sprintf("write-only EDD field: %s (set but never read)", key),
			})
		}
	}

	return warnings
}
