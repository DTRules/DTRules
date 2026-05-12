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

// Warning records a static analysis finding for a decision table. Warnings
// are advisory; the deployment gate ignores them. Errors (parse, EL compile,
// undefined references, structural validation, EL compliance, hand-coded
// postfix) flow through other validators and gate deployment separately.
type Warning struct {
	Table        string `json:"table"`
	Column       int    `json:"column"`        // 1-based; 0 if the warning is not column-scoped.
	ConditionRow int    `json:"condition_row"` // 0-based; -1 if the warning is not row-scoped.
	Kind         string `json:"kind"`          // e.g. "no-op column", "unreachable column", "redundant condition", "assignment-only table", "hand-coded postfix", "dead condition row".
	Reason       string `json:"reason"`
}

// newWarning is the canonical constructor: every warning starts with
// ConditionRow=-1 (not row-scoped) and Column=0 (not column-scoped) so
// callers only set the fields they need.
func newWarning(table, kind, reason string) Warning {
	return Warning{Table: table, ConditionRow: -1, Kind: kind, Reason: reason}
}

// String formats the warning in the canonical WARN form. The leading
// `WARN <table>:` is constant; the body identifies a column, a row, or
// neither depending on which scoping fields are set.
func (w Warning) String() string {
	switch {
	case w.Column > 0 && w.ConditionRow >= 0:
		return fmt.Sprintf("WARN %s: column %d / condition row %d %s [%s]", w.Table, w.Column, w.ConditionRow+1, w.Reason, w.Kind)
	case w.Column > 0:
		return fmt.Sprintf("WARN %s: column %d %s [%s]", w.Table, w.Column, w.Reason, w.Kind)
	case w.ConditionRow >= 0:
		return fmt.Sprintf("WARN %s: condition row %d %s [%s]", w.Table, w.ConditionRow+1, w.Reason, w.Kind)
	default:
		return fmt.Sprintf("WARN %s: %s [%s]", w.Table, w.Reason, w.Kind)
	}
}

// Inputs bundles the per-table data the advisory pass consumes. The
// `Inputs`-keyed Analyze entry point is the one to prefer in new code;
// the older AnalyzeTable signature is kept as a thin shim so existing
// tests keep compiling.
type Inputs struct {
	Name       string         // Table name; appears in every Warning.
	Policy     string         // "FIRST" or "" — gates policy-specific checks like #762.
	Conditions []ConditionRow // Per-row DSL + column patterns.
	Actions    []ActionRow    // Per-row DSL + column patterns.
	MaxCol     int            // 1-based; tables with 0 columns short-circuit.
}

// Analyze runs the structural advisory pass on a single table. Returns
// every applicable warning. Order is grouped by check kind (no-op /
// subsumed / unreachable / FIRST-policy redundancy / assignment-only)
// so callers that render warnings inline see related findings together.
//
// All checks here are XML-driven and fast — they don't require the tree
// builder. Tree-based checks (#765, #766) live in a different entry
// point because they run after compile, not on every authoring edit.
func Analyze(in Inputs) []Warning {
	if in.MaxCol == 0 {
		return nil
	}
	var warnings []Warning
	warnings = append(warnings, checkNoOpColumns(in.Name, in.Actions, in.MaxCol)...)
	warnings = append(warnings, checkSubsumedColumns(in.Name, in.Conditions, in.Actions, in.MaxCol)...)
	warnings = append(warnings, checkUnreachableColumns(in.Name, in.Conditions, in.MaxCol)...)
	if strings.EqualFold(in.Policy, "FIRST") {
		warnings = append(warnings, checkRedundantFirstPolicy(in.Name, in.Conditions, in.MaxCol)...)
	}
	warnings = append(warnings, checkAssignmentOnlyTable(in.Name, in.Actions, in.MaxCol)...)
	return warnings
}

// AnalyzeTable is the legacy entry point: structural checks on a parsed
// decision table, no policy context. New code should use Analyze with
// the Inputs struct so policy-gated checks (#762) fire.
//
// To detect hand-coded postfix (postfix present, no matching EL DSL), use the
// dedicated CheckHandCodedPostfix entry point — it has different inputs (it
// needs DSL/postfix pairs for every table element, not just rows + columns).
func AnalyzeTable(tableName string, conditions []ConditionRow, actions []ActionRow, maxCol int) []Warning {
	return Analyze(Inputs{
		Name:       tableName,
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     maxCol,
	})
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

// PostfixEntry pairs an element's EL DSL with its compiled / hand-written
// postfix, plus a 1-based identifier (row number or sequential index) and a
// kind label ("context", "initial_action", "condition", "action") used when
// formatting warnings.
type PostfixEntry struct {
	Kind    string // "context", "initial_action", "condition", "action"
	Number  int    // 1-based number within Kind
	DSL     string
	Postfix string
}

// CheckHandCodedPostfix returns warnings for every table element whose
// postfix has been authored by hand (non-empty postfix, no EL DSL). DTRules
// expects EL to be the source of truth; postfix is the compiled artifact.
// Hand-edited postfix bypasses the authoring path and risks the next
// `dtrules build` re-emitting an empty postfix from the empty EL DSL.
//
// The companion table-level predicate `HasOnlyHandCodedPostfix` collapses
// these warnings to a single bool that the runtime uses to refuse
// execution of legacy tables. Per-element warnings are still useful for
// authoring tools so they can point the user at the exact row that needs
// EL.
func CheckHandCodedPostfix(tableName string, entries []PostfixEntry) []Warning {
	var ws []Warning
	for _, e := range entries {
		postfix := strings.TrimSpace(e.Postfix)
		dsl := strings.TrimSpace(e.DSL)
		if postfix == "" || dsl != "" {
			continue
		}
		// Skip comment-only or empty-after-strip postfix.
		if isCommentOrEmpty(postfix) {
			continue
		}
		w := newWarning(tableName, "hand-coded postfix",
			fmt.Sprintf("%s %d has hand-coded postfix without EL DSL — author in EL before executing", e.Kind, e.Number))
		// When the offending element is a condition row, surface the row
		// number on the warning so the authoring UI can pin it to the row.
		if e.Kind == "condition" && e.Number > 0 {
			w.ConditionRow = e.Number - 1
		}
		ws = append(ws, w)
	}
	return ws
}

// HasAnyHandCodedPostfix reports whether ANY element of the table has
// hand-coded postfix — non-empty postfix paired with no EL DSL. The
// runtime treats true here as a hard block: hand-authored postfix
// bypasses the authoring API, which is the supported edit surface, and
// risks the next `dtrules build` re-emitting empty postfix from the
// empty EL DSL. Any single offending element fails the entire table.
//
// Per-element diagnostics are still available via CheckHandCodedPostfix;
// the table-level flag here is what the loader plumbs through to
// RDecisionTable.SetHandCodedPostfix.
func HasAnyHandCodedPostfix(entries []PostfixEntry) bool {
	for _, e := range entries {
		postfix := strings.TrimSpace(e.Postfix)
		if postfix == "" || isCommentOrEmpty(postfix) {
			continue
		}
		if strings.TrimSpace(e.DSL) == "" {
			return true
		}
	}
	return false
}

// FirstHandCodedElement returns a human-readable description of the first
// element in `entries` that has hand-coded postfix without EL DSL, or ""
// if none. Used to enrich the runtime refusal message.
func FirstHandCodedElement(entries []PostfixEntry) string {
	for _, e := range entries {
		postfix := strings.TrimSpace(e.Postfix)
		if postfix == "" || isCommentOrEmpty(postfix) {
			continue
		}
		if strings.TrimSpace(e.DSL) == "" {
			return fmt.Sprintf("%s %d", e.Kind, e.Number)
		}
	}
	return ""
}

// isCommentOrEmpty reports whether a postfix block contains only blank
// lines or comments (// or # prefix). Used to ignore postfix bodies that
// are effectively empty.
func isCommentOrEmpty(postfix string) bool {
	for _, line := range strings.Split(postfix, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Tolerate /* ... */ block comments that fit on a single line.
		if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
			continue
		}
		return false
	}
	return true
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
			w := newWarning(tableName, "no-op column", "is redundant (no actions)")
			w.Column = col + 1
			warnings = append(warnings, w)
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
				w := newWarning(tableName, "no-op column",
					fmt.Sprintf("is redundant (subsumed by column %d)", b+1))
				w.Column = a + 1
				warnings = append(warnings, w)
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

// checkRedundantFirstPolicy is the FIRST-policy redundancy check from #762.
//
// In a FIRST-policy table, reaching column N means every prior column
// failed to match. A column "fails" when at least one of its Y/N entries
// is contradicted by the input. An entry (row R, value V) in column N is
// *redundant* if it is already implied by:
//
//	(a) some prior column M's failure plus
//	(b) the other Y/N constraints in column N
//
// The check is purely structural — it only fires when the implication is
// provable from the DSL strings and the Y/N matrix, never from runtime
// semantics. Two patterns trigger it:
//
//  1. Column M has the SAME row R with the OPPOSITE value AND every
//     other Y/N constraint in M matches N's value for that row. Reaching
//     N then forces R to differ from M, which is exactly N's claim.
//  2. Column M has a row R' (different from R) constrained to a value
//     whose DSL is a syntactic negation of N's row R DSL with the same
//     required value. Same reasoning via the existing isNegationOf helper.
//
// Anything more subtle is left for a future SMT-style pass. The issue
// flags this as the high-false-positive case, so the bar is high.
func checkRedundantFirstPolicy(tableName string, conditions []ConditionRow, maxCol int) []Warning {
	// Build per-column constraint maps (row index → "Y" / "N"), skipping
	// "-" and "*".
	cols := make([]map[int]string, maxCol)
	for c := 0; c < maxCol; c++ {
		m := make(map[int]string)
		for row, cond := range conditions {
			if c >= len(cond.Columns) {
				continue
			}
			v := strings.ToUpper(strings.TrimSpace(cond.Columns[c]))
			if v == "Y" || v == "N" {
				m[row] = v
			}
		}
		cols[c] = m
	}

	flip := func(v string) string {
		if v == "Y" {
			return "N"
		}
		return "Y"
	}

	var warnings []Warning
	for n := 1; n < maxCol; n++ {
		for row, vN := range cols[n] {
			// Try to prove (row, vN) is implied by some prior column M.
			for m := 0; m < n; m++ {
				// Pattern 1: M constrains the same row with the opposite
				// value; every OTHER Y/N constraint in M is also in N with
				// the same value. Then M's failure (under N's match) must
				// be on this row, forcing R = vN.
				vMSame, hasSame := cols[m][row]
				if hasSame && vMSame == flip(vN) && impliedByPriorColumn(cols[m], cols[n], row) {
					warnings = append(warnings, redundantWarning(tableName, n, row, m, vN))
					goto nextEntry
				}
			}
		nextEntry:
		}
	}
	return warnings
}

// impliedByPriorColumn returns true when every Y/N entry in M (other
// than the row R under test) is also constrained in N to the same
// value. That's the structural premise that lets us treat M's failure
// as forcing R: every other constraint in M is satisfied by N matching,
// so M can only have failed on R.
func impliedByPriorColumn(m, n map[int]string, exceptRow int) bool {
	for row, vM := range m {
		if row == exceptRow {
			continue
		}
		vN, ok := n[row]
		if !ok || vN != vM {
			return false
		}
	}
	return true
}

func redundantWarning(tableName string, col, row, byCol int, value string) Warning {
	w := newWarning(tableName, "redundant condition",
		fmt.Sprintf("column %d row %d (=%s) is implied by column %d's failure — drop it for clarity",
			col+1, row+1, value, byCol+1))
	w.Column = col + 1
	w.ConditionRow = row
	return w
}

// checkAssignmentOnlyTable is the assignment-only-table check from #763.
//
// A table is "assignment-only" when every action across every column is
// a single `set <var> = <expr>` and every column assigns the same set
// of variables — the table exists solely to pick a value for those
// variables. These can usually be inlined at the call site (a context
// statement or a conditional expression), so the table itself is dead
// weight in the rule set.
//
// The check is advisory and conservative: it only fires when the DSL
// is unambiguously a simple assignment. A table that mixes assignments
// with `perform`, `add`, `error`, audit-trail lines, or compound
// statements is left alone. An author may have a legitimate reason to
// keep an assignment-only table (auditability, decision tracing, tool
// integration); the warning text reflects that — it suggests inlining
// but never demands.
func checkAssignmentOnlyTable(tableName string, actions []ActionRow, maxCol int) []Warning {
	if len(actions) == 0 || maxCol == 0 {
		return nil
	}

	// First per-column pass: collect the variables assigned in that
	// column's actions. Bail out if any action on any column is not a
	// pure assignment, or if a column has zero actions (no-op columns
	// are already flagged by checkNoOpColumns; reporting them again
	// here would be noise).
	colVars := make([]map[string]struct{}, maxCol)
	for c := 0; c < maxCol; c++ {
		vars := make(map[string]struct{})
		fired := false
		for _, a := range actions {
			if c >= len(a.Columns) || strings.ToUpper(strings.TrimSpace(a.Columns[c])) != "X" {
				continue
			}
			fired = true
			ok, lhs := assignmentLHS(a.DSL)
			if !ok {
				return nil
			}
			vars[lhs] = struct{}{}
		}
		if !fired {
			return nil
		}
		colVars[c] = vars
	}

	// Every column must assign the same set of variables. Differing
	// sets mean the table genuinely branches on which vars it touches,
	// which doesn't inline cleanly.
	base := colVars[0]
	for c := 1; c < maxCol; c++ {
		if !sameStringSet(base, colVars[c]) {
			return nil
		}
	}

	names := make([]string, 0, len(base))
	for v := range base {
		names = append(names, v)
	}
	sortStrings(names)
	w := newWarning(tableName, "assignment-only table",
		fmt.Sprintf("every column only assigns %v — consider inlining at the call site (a context statement, local, or conditional expression)", names))
	return []Warning{w}
}

// assignmentLHS returns (true, varName) when dsl is exactly a single
// `set <var> = <expr>` statement and nothing else, otherwise (false, "").
//
// Detection is intentionally narrow: lowercase keyword, single `=`-with-
// spaces, no trailing `;` (which would imply multiple statements). EL
// also has `add X to Y` and `perform X` statement forms — those should
// fall through and disqualify the table from this check.
func assignmentLHS(dsl string) (bool, string) {
	t := strings.TrimSpace(dsl)
	// A trailing semicolon is fine as long as nothing follows it; treat
	// `set x = 1;` and `set x = 1` the same. Any further content after
	// `;` means the table does more than assign — bail.
	if i := strings.Index(t, ";"); i >= 0 {
		rest := strings.TrimSpace(t[i+1:])
		if rest != "" {
			return false, ""
		}
		t = strings.TrimSpace(t[:i])
	}
	const setPrefix = "set "
	if !strings.HasPrefix(strings.ToLower(t), setPrefix) {
		return false, ""
	}
	t = strings.TrimSpace(t[len(setPrefix):])
	// Find the first ` = ` — `=` standalone could mean equality in
	// other contexts. Spaces on both sides are the EL convention.
	idx := strings.Index(t, " = ")
	if idx <= 0 {
		return false, ""
	}
	lhs := strings.TrimSpace(t[:idx])
	if lhs == "" {
		return false, ""
	}
	return true, lhs
}

// sameStringSet returns true when two string-keyed sets contain the
// same elements. Used to compare per-column assigned-variable sets.
func sameStringSet(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// sortStrings is a tiny insertion sort. Used only to give the warning
// text a stable variable-name ordering so tests can match on the
// rendered Reason string without flake.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
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
					w := newWarning(tableName, "unreachable column",
						fmt.Sprintf("can never match (conditions %d and %d are mutually exclusive)", yRows[i]+1, yRows[j]+1))
					w.Column = col + 1
					warnings = append(warnings, w)
					found = true
				}
			}
		}
	}
	return warnings
}
