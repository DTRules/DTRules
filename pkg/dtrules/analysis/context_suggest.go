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

package analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Context-push advisory.
//
// Regulation- and policy-heavy rule sets reference a "constants" entity
// over and over: constants.reduced_dose, constants.standard_dose,
// constants.adult_age, ... Qualifying every read makes the tables noisy.
//
// DTRules resolves a bare identifier against the entity stack (see
// interpreter FindEntity), and the stack propagates DOWN the perform call
// chain. So pushing the entity ONCE onto the context of an entry table —
// `add constants to context of this table` — lets every table it reaches
// write `reduced_dose` instead of `constants.reduced_dose`.
//
// This advisory finds entities that are referenced with a qualifier many
// times where the entity is NOT already on the table's effective stack,
// and suggests pushing them at the call-graph root that covers the most
// tables. It is advisory only — never an error.

// ContextSuggestion recommends pushing an entity onto the context stack at
// an entry table so heavily-qualified field references can be written
// unqualified.
type ContextSuggestion struct {
	Entity     string   `json:"entity"`     // entity referenced with a qualifier (e.g. "constants")
	References int      `json:"references"` // count of entity.field reads that could drop the prefix
	Fields     []string `json:"fields"`     // distinct fields referenced (sorted)
	Tables     []string `json:"tables"`     // tables containing those references (sorted)
	PushAt     []string `json:"push_at"`    // entry/root tables to push the entity onto (sorted)
	Reason     string   `json:"reason"`     // human-readable suggestion
}

// ctxMinQualifiedRefs is how many qualified references an entity needs
// (across tables where it isn't already on the effective stack) before the
// advisory fires. Tuned to flag "many" without nagging on one-offs.
const ctxMinQualifiedRefs = 3

// addToContextPattern matches the EL push `add <entity> to context ...`.
var addToContextPattern = regexp.MustCompile(`(?i)\badd\s+([a-z_][a-z0-9_]*)\s+to\s+context`)

// SuggestContextPushes scans a project's decision tables and recommends
// pushing an entity onto the context stack when its fields are referenced
// with a qualifier (entity.field) many times and the entity is not already
// on the table's effective entity stack (own contexts plus everything
// inherited from callers via the perform call graph).
func SuggestContextPushes(xmlDir string) ([]ContextSuggestion, error) {
	schema, err := loadEDDSchema(xmlDir)
	if err != nil {
		return nil, err
	}
	if schema == nil || len(schema.FieldsByEntity) == 0 {
		return nil, nil
	}

	infos, err := loadContextSuggestInfo(xmlDir, schema)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, nil
	}

	graph, err := AnalyzeTableCallGraph(xmlDir)
	if err != nil {
		return nil, err
	}

	// Effective entity stack per table: own contexts ∪ inherited from
	// callers. With no call graph, fall back to each table's own stack.
	var effective map[string][]string
	if graph != nil && len(graph.Calls) > 0 {
		effective = computeEffectiveStacks(infos, graph)
	} else {
		effective = make(map[string][]string, len(infos))
		for name, info := range infos {
			effective[name] = info.contextStack
		}
	}
	onStack := func(table, entity string) bool {
		return slices.Contains(effective[table], entity)
	}

	// Aggregate qualified references per entity across tables where the
	// entity is not already reachable unqualified.
	type agg struct {
		refs   int
		fields map[string]bool
		tables map[string]bool
	}
	byEntity := make(map[string]*agg)

	for _, info := range infos {
		frags := make([]string, 0, len(info.conditions)+len(info.actions)+len(info.initials))
		frags = append(frags, info.conditions...)
		frags = append(frags, info.actions...)
		frags = append(frags, info.initials...)
		for _, dsl := range frags {
			for _, m := range identifierPattern.FindAllStringSubmatch(strings.ToLower(dsl), -1) {
				entity, field := m[1], m[2]
				// "context" is always on the entity stack — never suggest it.
				if entity == "context" {
					continue
				}
				// Only real EDD field references count.
				fields, ok := schema.FieldsByEntity[entity]
				if !ok || !fields[field] {
					continue
				}
				// Already reachable unqualified — nothing to gain.
				if onStack(info.name, entity) {
					continue
				}
				a := byEntity[entity]
				if a == nil {
					a = &agg{fields: map[string]bool{}, tables: map[string]bool{}}
					byEntity[entity] = a
				}
				a.refs++
				a.fields[field] = true
				a.tables[info.name] = true
			}
		}
	}

	// Roots (tables with no callers) and what each reaches, for targeting
	// the push so one statement covers the most tables.
	var roots []string
	reachFrom := make(map[string]map[string]bool)
	if graph != nil {
		for t := range graph.Tables {
			if len(graph.Callers(t)) == 0 {
				roots = append(roots, t)
				reachFrom[t] = graph.Reachable(t)
			}
		}
		sort.Strings(roots)
	}

	var out []ContextSuggestion
	for entity, a := range byEntity {
		if a.refs < ctxMinQualifiedRefs {
			continue
		}
		fields := ctxSortedKeys(a.fields)
		tables := ctxSortedKeys(a.tables)

		// Push points: roots that reach a referencing table. A referencing
		// table reached by no root is its own push point.
		pushSet := make(map[string]bool)
		for _, t := range tables {
			covered := false
			for _, r := range roots {
				if reachFrom[r][t] {
					pushSet[r] = true
					covered = true
				}
			}
			if !covered {
				pushSet[t] = true
			}
		}
		pushAt := ctxSortedKeys(pushSet)
		if len(pushAt) == 0 {
			pushAt = tables
		}

		out = append(out, ContextSuggestion{
			Entity:     entity,
			References: a.refs,
			Fields:     fields,
			Tables:     tables,
			PushAt:     pushAt,
			Reason: fmt.Sprintf(
				"%q fields are referenced with a qualifier %d time(s) across %d table(s) where %q is not on the entity stack. "+
					"Push it once — `add %s to context of this table` — on entry table(s) %s so fields like %s can be written unqualified.",
				entity, a.refs, len(tables), entity, entity,
				strings.Join(pushAt, ", "), strings.Join(ctxFirstN(fields, 3), ", ")),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Entity < out[j].Entity })
	return out, nil
}

// loadContextSuggestInfo parses every *_dt.xml and returns per-table info
// for the advisory. Unlike loadCrossTableInfo it also folds `add X to
// context` pushes (in the context row or initial actions) into the table's
// own stack, so the advisory recognises a table that has already adopted
// the pattern and stops suggesting it.
func loadContextSuggestInfo(xmlDir string, schema *eddSchema) (map[string]*crossTableTableInfo, error) {
	infos := make(map[string]*crossTableTableInfo)

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		if !strings.HasSuffix(d.Name(), "_dt.xml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc struct {
			Tables []struct {
				Name           string         `xml:"table_name"`
				Contexts       []contextEntry `xml:"contexts>context_details"`
				InitialActions []struct {
					DSL string `xml:"initial_action_dsl"`
				} `xml:"initial_actions>initial_action"`
				Conditions []struct {
					DSL string `xml:"condition_dsl"`
				} `xml:"conditions>condition_details"`
				Actions []struct {
					DSL string `xml:"action_dsl"`
				} `xml:"actions>action_details"`
			} `xml:"decision_table"`
		}
		if xml.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, t := range doc.Tables {
			if t.Name == "" {
				continue
			}
			stack := stackFromContexts(schema, t.Contexts)
			seen := make(map[string]bool, len(stack))
			for _, e := range stack {
				seen[e] = true
			}
			addCtx := func(s string) {
				for _, m := range addToContextPattern.FindAllStringSubmatch(strings.ToLower(s), -1) {
					e := m[1]
					if _, ok := schema.FieldsByEntity[e]; ok && !seen[e] {
						stack = append(stack, e)
						seen[e] = true
					}
				}
			}
			for _, c := range t.Contexts {
				addCtx(c.DSL)
			}
			for _, a := range t.InitialActions {
				addCtx(a.DSL)
			}

			info := &crossTableTableInfo{name: t.Name, contextStack: stack}
			for _, c := range t.Conditions {
				info.conditions = append(info.conditions, c.DSL)
			}
			for _, a := range t.Actions {
				info.actions = append(info.actions, a.DSL)
			}
			for _, a := range t.InitialActions {
				info.initials = append(info.initials, a.DSL)
			}
			infos[t.Name] = info
		}
		return nil
	})
	return infos, err
}

func ctxSortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ctxFirstN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
