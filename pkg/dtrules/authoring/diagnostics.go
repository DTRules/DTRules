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
	"sort"
)

// DuplicateTableKind is the Diagnostic.Kind for a duplicate table rename.
const DuplicateTableKind = "duplicate_table"

// Diagnostic is a structured description of an authoring-time condition the
// SDK resolved tolerantly. Callers can surface it in a UI / CLI without
// having to re-derive the reason.
type Diagnostic struct {
	Kind         string `json:"kind"`
	OriginalName string `json:"original_name"`
	AssignedName string `json:"assigned_name"`
	File         string `json:"file"`
	Message      string `json:"message"`
}

// Diagnostics returns a snapshot of every authoring-time diagnostic recorded
// since the project was opened. The slice is ordered by assigned name.
func (p *Project) Diagnostics() []Diagnostic {
	if p == nil {
		return nil
	}
	out := make([]Diagnostic, len(p.diagnostics))
	copy(out, p.diagnostics)
	return out
}

// resolveDuplicateTableNames scans every loaded decision-table file for
// duplicate <table_name> values and rewrites the in-memory name of each
// duplicate (after the first, by sorted filepath order) to "<name>-1",
// "<name>-2", ... Every rename is recorded as a Diagnostic.
//
// The first file to hold the name keeps the real name. Subsequent duplicates
// are counted per original name, not globally. For example, if three files
// define "Foo" and two define "Bar", the results are:
//
//	Foo, Foo-1, Foo-2, Bar, Bar-1
func (p *Project) resolveDuplicateTableNames() {
	type occurrence struct {
		fileIndex  int
		tableIndex int
		path       string
	}

	// Gather files in sorted order for deterministic assignment.
	order := make([]int, len(p.dtFiles))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		return p.dtFiles[order[i]].path < p.dtFiles[order[j]].path
	})

	byName := make(map[string][]occurrence)
	for _, fi := range order {
		entry := &p.dtFiles[fi]
		for ti := range entry.tables.Tables {
			name := entry.tables.Tables[ti].TableName
			byName[name] = append(byName[name], occurrence{
				fileIndex:  fi,
				tableIndex: ti,
				path:       entry.path,
			})
		}
	}

	for name, occs := range byName {
		if len(occs) < 2 {
			continue
		}
		for i := 1; i < len(occs); i++ {
			assigned := fmt.Sprintf("%s-%d", name, i)
			occ := occs[i]
			p.dtFiles[occ.fileIndex].tables.Tables[occ.tableIndex].TableName = assigned
			p.diagnostics = append(p.diagnostics, Diagnostic{
				Kind:         DuplicateTableKind,
				OriginalName: name,
				AssignedName: assigned,
				File:         occ.path,
				Message:      fmt.Sprintf("renamed from duplicate of %q; resolve by renaming or deleting one copy", name),
			})
		}
	}

	sort.Slice(p.diagnostics, func(i, j int) bool {
		return p.diagnostics[i].AssignedName < p.diagnostics[j].AssignedName
	})
}
