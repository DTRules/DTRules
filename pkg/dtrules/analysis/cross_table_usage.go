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

package analysis

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// #776 piece A continued: cross-table entity-stack propagation.
//
// The per-file analyzer (collectDTReferences in edd_unused.go) computes
// each table's references using that table's own contexts + inline
// pushes. It doesn't know that when `Caller_Table` runs
// `for all client.cases` and then performs `Compute_Case_Status`, the
// callee's body executes with `case` on the entity stack. So a bare
// `status_code` reference in `Compute_Case_Status` looks unbound to
// the per-file pass and `case.status_code` gets flagged as unused.
//
// This pass closes that gap. Steps:
//
//   1. Walk every *_dt.xml again, recording per-table:
//        - the context-derived entity stack
//        - every DSL fragment with its kind (condition / action /
//          initial_action) — needed to drive write vs. read split
//   2. Pull in the call graph (analysis.AnalyzeTableCallGraph).
//   3. Compute the effective entity stack per table via fixed-point:
//        effective[T] = context_stack[T] ∪ ⋃ effective[caller] for every
//        caller that perform-references T directly. Iterate until no
//        change.
//   4. Re-extract bare references for each table's DSL fragments using
//      effective[T]. Add the discovered references to the supplied
//      read/write sets. The pass is purely additive — it only adds
//      references, never removes — so it cannot introduce false
//      negatives.
//
// The fixed point is guaranteed because each iteration can only grow
// the per-table stack, and the total stack size is bounded by the
// number of declared entity types in the schema.

// crossTableTableInfo retains the bits of a parsed table needed to
// re-extract references with an augmented entity stack.
type crossTableTableInfo struct {
	name         string
	contextStack []string
	conditions   []string
	actions      []string
	initials     []string
}

// applyCrossTablePropagation walks xmlDir, builds the call graph, and
// extends readRefs/writeRefs with bare-identifier references resolved
// against effective entity stacks. The augmented stack for a table is
// the union of its own context stack and the context stacks of every
// caller that perform-references it (transitively, via the call
// graph).
//
// schema may be nil; in that case the function is a no-op — bare-
// identifier resolution requires the EDD's fields-by-entity map.
//
// Errors during file walking are returned; in that case readRefs and
// writeRefs are left in whatever partial state they were before the
// failure (callers should treat a non-nil error as "propagation did
// not run to completion").
func applyCrossTablePropagation(xmlDir string, schema *eddSchema, readRefs, writeRefs map[string]bool) error {
	if schema == nil {
		return nil
	}

	infos, err := loadCrossTableInfo(xmlDir, schema)
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		return nil
	}

	graph, err := AnalyzeTableCallGraph(xmlDir)
	if err != nil {
		return err
	}
	if graph == nil || len(graph.Calls) == 0 {
		// No edges → nothing to propagate.
		return nil
	}

	effective := computeEffectiveStacks(infos, graph)

	// Re-extract bare references with the augmented stack. Only the
	// new references (those not already in the read/write sets) come
	// from this pass; the per-file pass already captured everything
	// the table-level context provided.
	for _, info := range infos {
		stack := effective[info.name]
		if len(stack) == 0 {
			continue
		}
		for _, dsl := range info.conditions {
			extractBareReads(dsl, stack, schema, readRefs)
		}
		for _, dsl := range info.actions {
			extractBareWritesAndReads(dsl, stack, schema, writeRefs, readRefs)
		}
		for _, dsl := range info.initials {
			extractBareWritesAndReads(dsl, stack, schema, writeRefs, readRefs)
		}
	}
	return nil
}

// loadCrossTableInfo walks every *_dt.xml under xmlDir and returns
// the per-table info needed for the propagation pass. Format errors
// in individual files are tolerated (the file is skipped), mirroring
// the leniency of collectDTReferences.
func loadCrossTableInfo(xmlDir string, schema *eddSchema) (map[string]*crossTableTableInfo, error) {
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
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil
		}
		for _, t := range doc.Tables {
			if t.Name == "" {
				continue
			}
			info := &crossTableTableInfo{
				name:         t.Name,
				contextStack: stackFromContexts(schema, t.Contexts),
			}
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

// computeEffectiveStacks runs the fixed-point propagation. The
// effective stack for a table T is the union of:
//   - T's own context stack
//   - every caller's effective stack (transitively, via the call graph)
//
// The iteration terminates because each pass can only grow each
// table's stack set, and the universe of entities is finite.
func computeEffectiveStacks(infos map[string]*crossTableTableInfo, graph *TableCallGraph) map[string][]string {
	// Seed each table with its own context stack as a set.
	stackSets := make(map[string]map[string]bool, len(infos))
	for name, info := range infos {
		set := make(map[string]bool, len(info.contextStack))
		for _, ent := range info.contextStack {
			set[ent] = true
		}
		stackSets[name] = set
	}

	// Iterate until no set changes. A pass walks every call edge
	// and unions the caller's set into the callee's set.
	for changed := true; changed; {
		changed = false
		for caller, callees := range graph.Calls {
			callerSet, ok := stackSets[caller]
			if !ok || len(callerSet) == 0 {
				continue
			}
			for callee := range callees {
				calleeSet, ok := stackSets[callee]
				if !ok {
					// Callee not in our info map (e.g. orphan call
					// or callee defined in a file we couldn't parse).
					// Skip — there's nothing to propagate into.
					continue
				}
				for ent := range callerSet {
					if !calleeSet[ent] {
						calleeSet[ent] = true
						changed = true
					}
				}
			}
		}
	}

	// Convert sets back to deterministic ordered slices for the
	// downstream extractors. Order matters only for "topmost entity"
	// resolution in bare-identifier lookup; the lookup walks the
	// slice and picks the first entity whose schema has the field,
	// so we sort to make results reproducible.
	out := make(map[string][]string, len(stackSets))
	for name, set := range stackSets {
		if len(set) == 0 {
			continue
		}
		stack := make([]string, 0, len(set))
		for ent := range set {
			stack = append(stack, ent)
		}
		sort.Strings(stack)
		out[name] = stack
	}
	return out
}
