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
	"regexp"
	"sort"
)

// performRe matches "perform <identifier>" in DSL text.
// Limitation: does not distinguish perform inside a quoted string, which is
// uncommon in practice. Callers that need precise analysis should use the EL compiler.
var performRe = regexp.MustCompile(`\bperform\s+([A-Za-z_][A-Za-z0-9_]*)`)

// dslTexts returns all action and initial-action DSL strings for this table.
func (t *Table) dslTexts() []string {
	texts := make([]string, 0, len(t.InitialActions)+len(t.Actions))
	for _, ia := range t.InitialActions {
		texts = append(texts, ia.DSL)
	}
	for _, a := range t.Actions {
		texts = append(texts, a.DSL)
	}
	return texts
}

// Dependencies returns the sorted, deduplicated list of table names that this
// table calls via "perform <name>" in its action or initial-action DSL.
func (t *Table) Dependencies() []string {
	seen := make(map[string]bool)
	for _, dsl := range t.dslTexts() {
		for _, m := range performRe.FindAllStringSubmatch(dsl, -1) {
			seen[m[1]] = true
		}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// Callers returns the sorted, deduplicated list of table names in the project
// that call this table via "perform <name>". Returns nil if the table has no
// project context (constructed outside of a Project).
func (t *Table) Callers() []string {
	if t.project == nil {
		return nil
	}
	seen := make(map[string]bool)
	for _, entry := range t.project.dtFiles {
		for i := range entry.tables.Tables {
			x := &entry.tables.Tables[i]
			if x.TableName == t.Name {
				continue
			}
			other := newTable(x, t.project.symbols)
			for _, dsl := range other.dslTexts() {
				for _, m := range performRe.FindAllStringSubmatch(dsl, -1) {
					if m[1] == t.Name {
						seen[x.TableName] = true
					}
				}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
