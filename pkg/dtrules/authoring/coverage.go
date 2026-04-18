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
	"strings"
)

// CoverageReport summarises which decision tables and columns were exercised
// across a set of ScenarioResult runs.
type CoverageReport struct {
	ExercisedTables  map[string]bool
	ExercisedColumns map[string]map[int]int // table -> col -> hit count
	UntouchedTables  []string
	UntouchedColumns map[string][]int // table -> sorted list of unexercised columns
}

// Cover builds a CoverageReport by walking the RunTrace of every ScenarioResult.
// Results whose RunTrace is nil (e.g. errored scenarios) are skipped.
func Cover(p *Project, results []ScenarioResult) *CoverageReport {
	r := &CoverageReport{
		ExercisedTables:  make(map[string]bool),
		ExercisedColumns: make(map[string]map[int]int),
		UntouchedColumns: make(map[string][]int),
	}

	for i := range results {
		rt := results[i].RunTrace
		if rt == nil {
			continue
		}
		for _, inv := range rt.Invocations {
			r.ExercisedTables[inv.TableName] = true
			if r.ExercisedColumns[inv.TableName] == nil {
				r.ExercisedColumns[inv.TableName] = make(map[int]int)
			}
			if inv.Column > 0 {
				r.ExercisedColumns[inv.TableName][inv.Column]++
			}
		}
	}

	// Determine untouched tables.
	allTables := p.Tables()
	allTableSet := make(map[string]bool, len(allTables))
	for _, name := range allTables {
		allTableSet[name] = true
	}
	for _, name := range allTables {
		if !r.ExercisedTables[name] {
			r.UntouchedTables = append(r.UntouchedTables, name)
		}
	}
	sort.Strings(r.UntouchedTables)

	// Determine untouched columns for each exercised table.
	for tableName := range r.ExercisedTables {
		tbl := p.Table(tableName)
		if tbl == nil {
			continue
		}
		numCols := tbl.Columns()
		exercised := r.ExercisedColumns[tableName]
		var untouched []int
		for col := 1; col <= numCols; col++ {
			if exercised[col] == 0 {
				untouched = append(untouched, col)
			}
		}
		if len(untouched) > 0 {
			r.UntouchedColumns[tableName] = untouched
		}
	}

	return r
}

// Summary returns a human-readable multi-line coverage report.
func (c *CoverageReport) Summary() string {
	var sb strings.Builder

	totalTables := len(c.ExercisedTables) + len(c.UntouchedTables)
	fmt.Fprintf(&sb, "Tables:  %d/%d exercised\n", len(c.ExercisedTables), totalTables)

	totalUntouchedCols := 0
	for _, cols := range c.UntouchedColumns {
		totalUntouchedCols += len(cols)
	}
	if totalUntouchedCols > 0 {
		fmt.Fprintf(&sb, "Columns: %d untouched\n", totalUntouchedCols)
	}

	if len(c.UntouchedTables) > 0 {
		sb.WriteString("\nUntouched tables:\n")
		for _, name := range c.UntouchedTables {
			fmt.Fprintf(&sb, "  - %s\n", name)
		}
	}

	if len(c.UntouchedColumns) > 0 {
		sb.WriteString("\nUntouched columns:\n")
		tableNames := make([]string, 0, len(c.UntouchedColumns))
		for name := range c.UntouchedColumns {
			tableNames = append(tableNames, name)
		}
		sort.Strings(tableNames)
		for _, name := range tableNames {
			cols := c.UntouchedColumns[name]
			colStrs := make([]string, len(cols))
			for i, col := range cols {
				colStrs[i] = fmt.Sprintf("%d", col)
			}
			fmt.Fprintf(&sb, "  - %s col %s\n", name, strings.Join(colStrs, ", "))
		}
	}

	return sb.String()
}
