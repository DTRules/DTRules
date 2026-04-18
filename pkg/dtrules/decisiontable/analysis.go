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

package decisiontable

import (
	"fmt"
	"strings"
)

// Warning records a static analysis finding for a decision table.
type Warning struct {
	Table  string
	Column int // 1-based, 0 if not column-specific
	Reason string
	Kind   string // "no-op column", "unreachable column"
}

// String formats the warning in the canonical WARN form.
func (w Warning) String() string {
	if w.Column > 0 {
		return fmt.Sprintf("WARN %s: column %d %s [%s]", w.Table, w.Column, w.Reason, w.Kind)
	}
	return fmt.Sprintf("WARN %s: %s [%s]", w.Table, w.Reason, w.Kind)
}

// AnalyzeTable runs structural checks on a parsed decision table and
// returns any warnings found. It does not require a compiled table.
//
// Part 1: redundant / no-op columns (empty actions, or subsumed by another column).
// Part 2: unreachable columns (contradictory condition requirements in same column).
func AnalyzeTable(tableName string, conditions []ConditionRow, actions []ActionRow, maxCol int) []Warning {
	var warnings []Warning

	warnings = append(warnings, checkNoOpColumns(tableName, actions, maxCol)...)
	warnings = append(warnings, checkSubsumedColumns(tableName, conditions, actions, maxCol)...)
	warnings = append(warnings, checkUnreachableColumns(tableName, conditions, maxCol)...)

	return warnings
}

// ConditionRow holds the DSL text and Y/N pattern for a condition row.
type ConditionRow struct {
	DSL     string
	Columns []string // indexed by column (0-based), values: "Y", "N", "-", "*"
}

// ActionRow holds the DSL text and X pattern for an action row.
type ActionRow struct {
	DSL     string
	Columns []string // indexed by column (0-based), values: "X" or ""
}

// checkNoOpColumns flags columns with no actions marked X.
func checkNoOpColumns(tableName string, actions []ActionRow, maxCol int) []Warning {
	var warnings []Warning
	for col := 0; col < maxCol; col++ {
		hasAction := false
		for _, a := range actions {
			if col < len(a.Columns) && strings.ToUpper(strings.TrimSpace(a.Columns[col])) == "X" {
				hasAction = true
				break
			}
		}
		if !hasAction {
			warnings = append(warnings, Warning{
				Table:  tableName,
				Column: col + 1,
				Reason: "is redundant (no actions)",
				Kind:   "no-op column",
			})
		}
	}
	return warnings
}

// checkSubsumedColumns flags column A if another column B is more permissive
// (fewer or equal constraints) and has all of A's actions.
// Column B subsumes column A when:
//   - B's condition constraints are a proper subset of A's (B has fewer required Y/N)
//   - B's actions include all of A's actions
func checkSubsumedColumns(tableName string, conditions []ConditionRow, actions []ActionRow, maxCol int) []Warning {
	// Build per-column constraint sets and action sets.
	type colProfile struct {
		constraints map[int]string // row → required value (Y or N only; dash = unconstrained)
		actionSet   map[int]bool   // action row index → true
	}

	profiles := make([]colProfile, maxCol)
	for col := 0; col < maxCol; col++ {
		p := colProfile{
			constraints: make(map[int]string),
			actionSet:   make(map[int]bool),
		}
		for row, c := range conditions {
			v := ""
			if col < len(c.Columns) {
				v = strings.ToUpper(strings.TrimSpace(c.Columns[col]))
			}
			if v == "Y" || v == "N" {
				p.constraints[row] = v
			}
		}
		for row, a := range actions {
			if col < len(a.Columns) && strings.ToUpper(strings.TrimSpace(a.Columns[col])) == "X" {
				p.actionSet[row] = true
			}
		}
		profiles[col] = p
	}

	var warnings []Warning
	for a := 0; a < maxCol; a++ {
		if len(profiles[a].actionSet) == 0 {
			// Already flagged as no-op; skip subsumption check.
			continue
		}
		for b := 0; b < maxCol; b++ {
			if a == b {
				continue
			}
			// B subsumes A if B's constraints are a subset of A's constraints
			// (B is more permissive) and B has all of A's actions.
			bSubsumesA := func() bool {
				// Every constraint in B must match what A requires.
				for row, bVal := range profiles[b].constraints {
					aVal, ok := profiles[a].constraints[row]
					if !ok || aVal != bVal {
						return false
					}
				}
				// B must have all actions that A has.
				for row := range profiles[a].actionSet {
					if !profiles[b].actionSet[row] {
						return false
					}
				}
				// B must be strictly more permissive (fewer constraints) than A.
				return len(profiles[b].constraints) < len(profiles[a].constraints)
			}()
			if bSubsumesA {
				warnings = append(warnings, Warning{
					Table:  tableName,
					Column: a + 1,
					Reason: fmt.Sprintf("is redundant (subsumed by column %d)", b+1),
					Kind:   "no-op column",
				})
				break // one subsumer is enough
			}
		}
	}
	return warnings
}

// negationPrefixes are DSL patterns where the negated form can be detected
// by simple string inspection. We look for pairs where one condition's DSL
// is the negation of another within the same column.
var negationPairs = [][2]string{
	{" is equal to ", " is not equal to "},
	{" is not equal to ", " is equal to "},
	{" > ", " <= "},
	{" <= ", " > "},
	{" < ", " >= "},
	{" >= ", " < "},
}

// isNegationOf returns true if b is a simple syntactic negation of a.
func isNegationOf(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	for _, pair := range negationPairs {
		if strings.Contains(a, pair[0]) && strings.Contains(b, pair[1]) {
			// Replace the operator and compare the rest
			aBase := strings.Replace(a, pair[0], "§", 1)
			bBase := strings.Replace(b, pair[1], "§", 1)
			if aBase == bBase {
				return true
			}
		}
	}
	// Handle "not X" / "X" prefix form: condition b = "not " + a or vice versa.
	if strings.HasPrefix(b, "not ") && strings.TrimPrefix(b, "not ") == a {
		return true
	}
	if strings.HasPrefix(a, "not ") && strings.TrimPrefix(a, "not ") == b {
		return true
	}
	return false
}

// checkUnreachableColumns flags columns where two conditions require Y but
// their DSL expressions are syntactic negations of each other.
func checkUnreachableColumns(tableName string, conditions []ConditionRow, maxCol int) []Warning {
	var warnings []Warning
	for col := 0; col < maxCol; col++ {
		// Collect all rows that require Y for this column.
		var yRows []int
		for row, c := range conditions {
			v := ""
			if col < len(c.Columns) {
				v = strings.ToUpper(strings.TrimSpace(c.Columns[col]))
			}
			if v == "Y" {
				yRows = append(yRows, row)
			}
		}
		// Look for pairs of Y-required conditions that are mutual negations.
		found := false
		for i := 0; i < len(yRows) && !found; i++ {
			for j := i + 1; j < len(yRows) && !found; j++ {
				dslI := conditions[yRows[i]].DSL
				dslJ := conditions[yRows[j]].DSL
				if isNegationOf(dslI, dslJ) {
					warnings = append(warnings, Warning{
						Table:  tableName,
						Column: col + 1,
						Reason: fmt.Sprintf("can never match (conditions %d and %d are mutually exclusive)", yRows[i]+1, yRows[j]+1),
						Kind:   "unreachable column",
					})
					found = true
				}
			}
		}
	}
	return warnings
}
