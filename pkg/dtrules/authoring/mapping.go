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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// Mapping is a view over a project's _map.xml file, supporting read and mutate operations.
type Mapping struct {
	project  *Project
	path     string
	mapXML   *excel.MapXML
}

// SetAttribute is a single non-section entry in the mapping.
type SetAttribute struct {
	Tag        string
	RAttribute string
	Enclosure  string
	Type       string
}

// Mapping returns the Mapping view for this project. It lazily loads the first
// *_map.xml found in the xml/ directory. Returns nil if no map file exists.
func (p *Project) Mapping() (*Mapping, error) {
	paths, err := filepath.Glob(filepath.Join(p.xmlDir, "*_map.xml"))
	if err != nil || len(paths) == 0 {
		return nil, fmt.Errorf("no _map.xml file found in %s", p.xmlDir)
	}
	mx, err := excel.LoadMapXMLFromFile(paths[0])
	if err != nil {
		return nil, fmt.Errorf("load map xml: %w", err)
	}
	return &Mapping{project: p, path: paths[0], mapXML: mx}, nil
}

// Entries returns all non-section SetAttribute entries in the mapping.
func (m *Mapping) Entries() []SetAttribute {
	var out []SetAttribute
	for _, e := range m.mapXML.Entries {
		if !e.IsSection {
			out = append(out, SetAttribute{
				Tag:        e.Tag,
				RAttribute: e.RAttribute,
				Enclosure:  e.Enclosure,
				Type:       e.Type,
			})
		}
	}
	return out
}

// AddEntry appends a new SetAttribute entry and validates it against the EDD.
func (m *Mapping) AddEntry(e SetAttribute) error {
	if err := m.validateEntry(e); err != nil {
		return err
	}
	m.mapXML.Entries = append(m.mapXML.Entries, excel.MapEntry{
		Tag:        e.Tag,
		RAttribute: e.RAttribute,
		Enclosure:  e.Enclosure,
		Type:       e.Type,
	})
	return m.save()
}

// UpdateEntry finds the entry by Tag and replaces it, validating against the EDD.
func (m *Mapping) UpdateEntry(tag string, e SetAttribute) error {
	idx := m.findByTag(tag)
	if idx < 0 {
		return fmt.Errorf("mapping entry with tag %q not found", tag)
	}
	if err := m.validateEntry(e); err != nil {
		return err
	}
	m.mapXML.Entries[idx] = excel.MapEntry{
		Tag:        e.Tag,
		RAttribute: e.RAttribute,
		Enclosure:  e.Enclosure,
		Type:       e.Type,
	}
	return m.save()
}

// DeleteEntry removes the entry with the given Tag.
func (m *Mapping) DeleteEntry(tag string) error {
	idx := m.findByTag(tag)
	if idx < 0 {
		return fmt.Errorf("mapping entry with tag %q not found", tag)
	}
	m.mapXML.Entries = append(m.mapXML.Entries[:idx], m.mapXML.Entries[idx+1:]...)
	return m.save()
}

// findByTag returns the index of the first non-section entry with the given Tag,
// or -1 if not found.
func (m *Mapping) findByTag(tag string) int {
	for i, e := range m.mapXML.Entries {
		if !e.IsSection && e.Tag == tag {
			return i
		}
	}
	return -1
}

// validateEntry checks that Enclosure and RAttribute exist in the EDD and that
// the declared Type matches.
func (m *Mapping) validateEntry(e SetAttribute) error {
	enclosure := strings.ToLower(e.Enclosure)
	rattr := strings.ToLower(e.RAttribute)

	// Collect all known entities from the project symbols.
	entityExists := false
	for key := range m.project.symbols {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == enclosure {
			entityExists = true
			break
		}
	}
	if !entityExists {
		return fmt.Errorf("enclosure %q does not exist in the EDD", e.Enclosure)
	}

	// Check the attribute exists on the entity. Symbol keys are lower-cased
	// (LoadEDDSymbols), as are enclosure and rattr above.
	attrKey := enclosure + "." + rattr
	declaredType, ok := m.project.symbols[attrKey]
	if !ok {
		return fmt.Errorf("attribute %q not found on entity %q in the EDD", e.RAttribute, e.Enclosure)
	}

	// Compare types (case-insensitive).
	if !strings.EqualFold(declaredType, e.Type) {
		return fmt.Errorf("type mismatch for %s.%s: EDD declares %q, entry has %q",
			e.Enclosure, e.RAttribute, declaredType, e.Type)
	}

	return nil
}

// save writes the in-memory MapXML back to disk.
func (m *Mapping) save() error {
	return excel.WriteMapXML(m.mapXML, m.path)
}
