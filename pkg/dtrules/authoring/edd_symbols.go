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
	"strings"
)

// LoadEDDSymbols walks the *_edd.xml files under root (recursively, so nested
// EDDs like xml/states/CO_edd.xml are found) and returns a map of EDD symbol
// name → type. Both the bare field name and the entity-qualified
// "entity.field" form are recorded, because DSL references fields by bare name
// while qualified references need the dotted key.
//
// If root is a file rather than a directory, its containing directory is
// walked. Read/parse errors on individual files are skipped so a single bad
// EDD doesn't blank the whole symbol table.
//
// This is the single source of truth for EDD→symbol construction shared by the
// authoring Save path (Project.loadEDD) and the cmd/dtrules build path, so the
// two cannot drift in discovery scope or key form (#874, #879).
//
// EL is case-insensitive, so keys and types are lower-cased; consumers must
// lower-case their queries (the emitter's lookupType already does). Typed-case
// keys silently missed camel-case fields, so `add X to client.totalIncome`
// lost its numeric type and compiled to the array addto path.
func LoadEDDSymbols(root string) map[string]string {
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

	walkDir := root
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		walkDir = filepath.Dir(root)
	}

	symbols := make(map[string]string)
	_ = filepath.WalkDir(walkDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_edd.xml") {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		var f eddFile
		if xml.Unmarshal(data, &f) != nil {
			return nil
		}
		for _, ent := range f.Entities {
			entName := strings.ToLower(ent.Name)
			for _, fld := range ent.Fields {
				if fld.Name == "" || fld.Type == "" {
					continue
				}
				name := strings.ToLower(fld.Name)
				typ := strings.ToLower(fld.Type)
				symbols[name] = typ
				if entName != "" {
					symbols[entName+"."+name] = typ
				}
			}
		}
		return nil
	})
	return symbols
}
