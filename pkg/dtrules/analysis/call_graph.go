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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// TableCallGraph is the directed graph of table-to-table invocations
// extracted statically from `*_dt.xml` fragments. Edges originate at
// the caller table (key) and target every callee table name found in
// the caller's `initial_action_dsl` / `action_dsl` fragments. Self
// edges and duplicates are de-duplicated.
//
// This is the foundation for #776 phase 3 piece A (cross-table
// reference graph). The graph itself is the deliverable here; entity-
// stack propagation across edges is a follow-up piece that will
// consume this map.
type TableCallGraph struct {
	// Tables is the set of table names defined across the project's
	// `*_dt.xml` files. Names are case-preserved; comparison is
	// case-sensitive (matching the runtime's table-name lookup).
	Tables map[string]bool

	// Calls maps caller table name → set of callee table names. A
	// table with no calls is absent from the map (callers with only
	// non-perform actions don't get an empty set).
	Calls map[string]map[string]bool

	// OrphanCalls lists (caller, callee) pairs where the callee
	// isn't a declared table. These are likely typos in the DSL —
	// the runtime would fail at execution time with "table not
	// found." Surfacing them statically catches the typo at build/
	// review time.
	OrphanCalls []OrphanCall

	// DTFile maps each defined table name → the *_dt.xml file it was
	// loaded from. Useful for advisory output.
	DTFile map[string]string
}

// OrphanCall records a `perform <name>` reference whose callee isn't
// defined anywhere in the loaded `*_dt.xml` set.
type OrphanCall struct {
	Caller string
	Callee string
	DTFile string
}

// String renders an OrphanCall in the canonical advisory form.
func (o OrphanCall) String() string {
	return fmt.Sprintf("INFO %s calls undefined table %s (%s)", o.Caller, o.Callee, o.DTFile)
}

// performCallPattern matches the explicit `perform <Name>` form. The
// EL grammar accepts:
//
//   - `perform <Name>`               (performDTExplicit / performDT)
//   - `<Name>` alone                 (performDT — bare ident invokes
//     the table by name if the type resolver classifies the ident
//     as a typedDecisionTable)
//   - `perform when called`          (a context-flow keyword that is
//     NOT a table call — we exclude it explicitly)
//   - `perform table named (<expr>)` (performDynamicTable — dynamic
//     dispatch, target unknown to static analysis)
//
// The bare-ident form is ambiguous without symbol-table context (any
// CamelCase ident could be a table call, a typed entity, an operator,
// etc.). To keep the call graph conservative, we require the explicit
// `perform <Name>` keyword. False negatives from this conservatism
// are acceptable — the goal is precise edges, not exhaustive ones.
//
// The pattern requires `perform` to be followed by an identifier
// that does NOT start with `when` (which would be the `perform when
// called` keyword, not a table call) and that isn't `table` (which
// would start a `perform table named (...)` dynamic dispatch).
var performCallPattern = regexp.MustCompile(`\bperform\s+([A-Z][A-Za-z0-9_]*)`)

// reservedAfterPerform names words that follow `perform` in non-call
// keyword forms. The regex above only catches uppercase-starting
// idents so most reserved words are excluded by character class
// already, but `Table` (e.g. `perform table named` — but `table`
// starts lowercase in the grammar, so this is defensive) and any
// future uppercase keyword would slip through.
var reservedAfterPerform = map[string]bool{
	"Table": true, // defensive — grammar uses lowercase `table`
}

// AnalyzeTableCallGraph walks every `*_dt.xml` under xmlDir, extracts
// the table-call relationships, and returns the graph plus any
// orphan-call findings.
//
// Dynamic-dispatch sites (`perform table named (<expr>)`) are NOT
// recorded as edges — static analysis can't enumerate their possible
// targets without an enumeration-bounded declaration (#776 phase B).
// They're reported separately via DynamicDispatchSites in a follow-up
// piece; for now they're simply ignored.
func AnalyzeTableCallGraph(xmlDir string) (*TableCallGraph, error) {
	graph := &TableCallGraph{
		Tables:  make(map[string]bool),
		Calls:   make(map[string]map[string]bool),
		DTFile:  make(map[string]string),
	}

	// Pass 1: collect the set of defined tables. We need the full
	// set before extracting calls so orphan detection has the right
	// truth set.
	type rawTable struct {
		Name           string
		File           string
		InitialActions []string
		Actions        []string
	}
	var rawTables []rawTable

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
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

		var doc struct {
			Tables []struct {
				Name           string `xml:"table_name"`
				InitialActions []struct {
					DSL string `xml:"initial_action_dsl"`
				} `xml:"initial_actions>initial_action"`
				Actions []struct {
					DSL string `xml:"action_dsl"`
				} `xml:"actions>action_details"`
			} `xml:"decision_table"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil // skip unparseable, mirroring edd_unused
		}

		for _, t := range doc.Tables {
			if t.Name == "" {
				continue
			}
			graph.Tables[t.Name] = true
			graph.DTFile[t.Name] = name
			rt := rawTable{Name: t.Name, File: name}
			for _, ia := range t.InitialActions {
				rt.InitialActions = append(rt.InitialActions, ia.DSL)
			}
			for _, a := range t.Actions {
				rt.Actions = append(rt.Actions, a.DSL)
			}
			rawTables = append(rawTables, rt)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Pass 2: extract calls. We already have the parsed DSL
	// fragments from pass 1, so this is a pure in-memory scan.
	for _, rt := range rawTables {
		for _, dsl := range rt.InitialActions {
			recordCalls(graph, rt, dsl)
		}
		for _, dsl := range rt.Actions {
			recordCalls(graph, rt, dsl)
		}
	}

	// Sort orphan calls deterministically so output is stable.
	sort.Slice(graph.OrphanCalls, func(i, j int) bool {
		if graph.OrphanCalls[i].Caller != graph.OrphanCalls[j].Caller {
			return graph.OrphanCalls[i].Caller < graph.OrphanCalls[j].Caller
		}
		return graph.OrphanCalls[i].Callee < graph.OrphanCalls[j].Callee
	})

	return graph, nil
}

// recordCalls extracts every `perform <Name>` reference from dsl and
// records caller→callee edges in the graph. Self-edges and duplicate
// calls are deduplicated by the underlying set.
func recordCalls(graph *TableCallGraph, rt struct {
	Name           string
	File           string
	InitialActions []string
	Actions        []string
}, dsl string) {
	if dsl == "" {
		return
	}
	for _, match := range performCallPattern.FindAllStringSubmatch(dsl, -1) {
		callee := match[1]
		if reservedAfterPerform[callee] {
			continue
		}
		// Self-edges are real (a table can call itself recursively)
		// but rare. Keep them — they're informative.
		if _, ok := graph.Calls[rt.Name]; !ok {
			graph.Calls[rt.Name] = make(map[string]bool)
		}
		if graph.Calls[rt.Name][callee] {
			continue // already recorded for this caller
		}
		graph.Calls[rt.Name][callee] = true
		if !graph.Tables[callee] {
			graph.OrphanCalls = append(graph.OrphanCalls, OrphanCall{
				Caller: rt.Name,
				Callee: callee,
				DTFile: rt.File,
			})
		}
	}
}

// Callers returns the inverse of Calls — the set of tables that call
// the given table. Useful for reverse traversal (e.g. "who depends on
// this table?" for refactor impact).
//
// Empty for an entry table that no other table calls, and for any
// table not in the graph at all.
func (g *TableCallGraph) Callers(table string) []string {
	var out []string
	for caller, callees := range g.Calls {
		if callees[table] {
			out = append(out, caller)
		}
	}
	sort.Strings(out)
	return out
}

// Reachable returns the set of tables reachable from `from` via call
// edges, including `from` itself. Cycles terminate naturally because
// the visited set is checked before each descent.
func (g *TableCallGraph) Reachable(from string) map[string]bool {
	visited := make(map[string]bool)
	var walk func(string)
	walk = func(t string) {
		if visited[t] {
			return
		}
		visited[t] = true
		for callee := range g.Calls[t] {
			walk(callee)
		}
	}
	walk(from)
	return visited
}

// UnreachedTables returns the set of declared tables that aren't
// reachable from any of the given entry tables. Useful for project-
// level dead-code detection: tables declared but never invoked from
// the entry points are candidates for removal.
//
// An empty `entryTables` returns every defined table (nothing is
// reached when nothing is entered).
func (g *TableCallGraph) UnreachedTables(entryTables []string) []string {
	reached := make(map[string]bool)
	for _, entry := range entryTables {
		for t := range g.Reachable(entry) {
			reached[t] = true
		}
	}
	var out []string
	for t := range g.Tables {
		if !reached[t] {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}
