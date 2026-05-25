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
	"encoding/json"
	"strings"
	"testing"
)

// makeConditions builds ConditionRow slices with the given DSL texts and
// column values. colVals[i] is the Y/N/- pattern for condition row i across
// all columns.
func makeConditions(dsls []string, colVals [][]string) []ConditionRow {
	rows := make([]ConditionRow, len(dsls))
	for i, dsl := range dsls {
		cols := []string{}
		if i < len(colVals) {
			cols = colVals[i]
		}
		rows[i] = ConditionRow{DSL: dsl, Columns: cols}
	}
	return rows
}

// makeActions builds ActionRow slices. colVals[i] is the X/"" pattern for action row i.
func makeActions(dsls []string, colVals [][]string) []ActionRow {
	rows := make([]ActionRow, len(dsls))
	for i, dsl := range dsls {
		cols := []string{}
		if i < len(colVals) {
			cols = colVals[i]
		}
		rows[i] = ActionRow{DSL: dsl, Columns: cols}
	}
	return rows
}

func TestAnalyzeTable_NoActionColumn(t *testing.T) {
	// Column 1 has no X in any action row → should be flagged as no-op.
	conditions := makeConditions(
		[]string{`job.status is equal to "active"`},
		[][]string{{"Y", "N"}},
	)
	actions := makeActions(
		[]string{`set job.result to 1`},
		[][]string{{"", "X"}}, // col 1 = no X, col 2 = X
	)

	warns := AnalyzeTable("TestTable", conditions, actions, 2)

	found := false
	for _, w := range warns {
		if w.Column == 1 && w.Kind == "no-op column" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected no-op warning for column 1, got %v", warns)
	}
}

func TestAnalyzeTable_CleanTable_NoWarnings(t *testing.T) {
	// All columns have actions, no contradictions.
	conditions := makeConditions(
		[]string{`job.amount > 100`},
		[][]string{{"Y", "N"}},
	)
	actions := makeActions(
		[]string{`set job.tier to "high"`, `set job.tier to "low"`},
		[][]string{{"X", ""}, {"", "X"}},
	)

	warns := AnalyzeTable("CleanTable", conditions, actions, 2)
	if len(warns) != 0 {
		t.Errorf("expected no warnings for clean table, got %v", warns)
	}
}

func TestAnalyzeTable_SubsumedColumn(t *testing.T) {
	// Column 1 requires Y on conditions 0 and 1, has action 0.
	// Column 2 requires Y on condition 0 only (superset of conditions), has action 0.
	// → Column 1 is subsumed by column 2 (col 2 is more permissive with same actions).
	conditions := makeConditions(
		[]string{`job.a is equal to "x"`, `job.b is equal to "y"`},
		[][]string{
			{"Y", "Y"}, // cond 0: col1=Y, col2=Y
			{"Y", "-"}, // cond 1: col1=Y, col2=- (unconstrained)
		},
	)
	actions := makeActions(
		[]string{`set job.result to 1`},
		[][]string{{"X", "X"}}, // both columns have action 0
	)

	warns := AnalyzeTable("SubsumedTable", conditions, actions, 2)

	found := false
	for _, w := range warns {
		if w.Column == 1 && strings.Contains(w.Reason, "subsumed") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected subsumed warning for column 1, got %v", warns)
	}
}

func TestAnalyzeTable_UnreachableColumn(t *testing.T) {
	// Column 1 requires Y on both:
	//   cond 0: `job.status is equal to "active"`
	//   cond 1: `job.status is not equal to "active"`
	// These are syntactic negations → column 1 is unreachable.
	conditions := makeConditions(
		[]string{
			`job.status is equal to "active"`,
			`job.status is not equal to "active"`,
		},
		[][]string{
			{"Y", "N"}, // cond 0
			{"Y", "Y"}, // cond 1
		},
	)
	actions := makeActions(
		[]string{`set job.result to 1`, `set job.result to 2`},
		[][]string{{"X", ""}, {"", "X"}},
	)

	warns := AnalyzeTable("UnreachableTable", conditions, actions, 2)

	found := false
	for _, w := range warns {
		if w.Column == 1 && w.Kind == "unreachable column" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unreachable warning for column 1, got %v", warns)
	}
}

// TestWarningJSONShape locks the JSON wire format the authoring channel
// (#761) returns from table_get/put/patch and the new table_warnings
// tool. Consumers (MCP clients, the API server) parse this shape, so the
// field names, the lower-cased keys, and the sentinel values for
// "not column-scoped" (column=0) and "not row-scoped" (condition_row=-1)
// must not silently change.
func TestWarningJSONShape(t *testing.T) {
	w := newWarning("MyTable", "no-op column", "is redundant (no actions)")
	w.Column = 4
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	want := `{"table":"MyTable","column":4,"condition_row":-1,"kind":"no-op column","reason":"is redundant (no actions)"}`
	if got != want {
		t.Errorf("JSON shape drift:\n  got:  %s\n  want: %s", got, want)
	}
}

// TestNewWarning_DefaultsConditionRowToMinusOne verifies the
// newWarning constructor sets the "not row-scoped" sentinel so callers
// only have to set ConditionRow when they actually mean a specific row.
func TestNewWarning_DefaultsConditionRowToMinusOne(t *testing.T) {
	w := newWarning("T", "kind", "reason")
	if w.ConditionRow != -1 {
		t.Errorf("ConditionRow default = %d, want -1", w.ConditionRow)
	}
	if w.Column != 0 {
		t.Errorf("Column default = %d, want 0", w.Column)
	}
}

// TestWarningString_AllScopes checks the four ways String() formats a
// warning depending on which scoping fields are set.
func TestWarningString_AllScopes(t *testing.T) {
	cases := []struct {
		name string
		w    Warning
		want string
	}{
		{
			name: "table-scoped only",
			w:    Warning{Table: "T", ConditionRow: -1, Kind: "k", Reason: "r"},
			want: "WARN T: r [k]",
		},
		{
			name: "column-scoped",
			w:    Warning{Table: "T", Column: 2, ConditionRow: -1, Kind: "k", Reason: "r"},
			want: "WARN T: column 2 r [k]",
		},
		{
			name: "row-scoped",
			w:    Warning{Table: "T", ConditionRow: 1, Kind: "k", Reason: "r"},
			want: "WARN T: condition row 2 r [k]",
		},
		{
			name: "cell-scoped (column + row)",
			w:    Warning{Table: "T", Column: 2, ConditionRow: 0, Kind: "k", Reason: "r"},
			want: "WARN T: column 2 / condition row 1 r [k]",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.w.String(); got != c.want {
				t.Errorf("String() = %q, want %q", got, c.want)
			}
		})
	}
}

// TestAnalyze_RedundantFirstPolicy exercises the FIRST-policy
// redundant-Y/N check from #762. The canonical example from the issue:
//
//	                  c1   c2
//	income > 100k:    Y    N    ← c2's N is implied by c1 failing
//	filing = MFJ:     Y    Y
//
// When c2 matches with filing=MFJ=Y, c1 (which also required filing=Y)
// must have failed on the other constraint — so income>100k is forced
// to N. The explicit N is redundant; the check should flag it.
func TestAnalyze_RedundantFirstPolicy(t *testing.T) {
	conditions := makeConditions(
		[]string{`income > 100k`, `filing = MFJ`},
		[][]string{
			{"Y", "N"}, // row 0: c1=Y, c2=N
			{"Y", "Y"}, // row 1: c1=Y, c2=Y
		},
	)
	actions := makeActions(
		[]string{`set high_earner_mfj`, `set other_mfj`},
		[][]string{{"X", ""}, {"", "X"}},
	)
	warns := Analyze(Inputs{
		Name:       "FirstPolicyRedundancy",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     2,
	})
	found := false
	for _, w := range warns {
		if w.Kind == "redundant condition" && w.Column == 2 && w.ConditionRow == 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected redundant-condition warning for col 2 row 1, got %v", warns)
	}
}

// TestAnalyze_RedundantFirstPolicy_NonFirstPolicy verifies that the
// check is gated by Policy=="FIRST". An ALL or unspecified policy must
// not emit the warning because the failure-implies-X reasoning only
// holds when prior columns are guaranteed to have failed.
func TestAnalyze_RedundantFirstPolicy_NonFirstPolicy(t *testing.T) {
	conditions := makeConditions(
		[]string{`income > 100k`, `filing = MFJ`},
		[][]string{{"Y", "N"}, {"Y", "Y"}},
	)
	actions := makeActions(
		[]string{`set a`, `set b`},
		[][]string{{"X", ""}, {"", "X"}},
	)
	for _, policy := range []string{"", "ALL"} {
		warns := Analyze(Inputs{
			Name:       "T",
			Policy:     policy,
			Conditions: conditions,
			Actions:    actions,
			MaxCol:     2,
		})
		for _, w := range warns {
			if w.Kind == "redundant condition" {
				t.Errorf("policy=%q: did not expect redundant-condition warning, got %v", policy, w)
			}
		}
	}
}

// TestAnalyze_RedundantFirstPolicy_DistinguishingRow verifies the
// implication only fires when every OTHER Y/N constraint in the prior
// column matches the current column. If columns differ on more than
// the candidate row, removing the candidate doesn't preserve the
// implication and the entry isn't redundant.
func TestAnalyze_RedundantFirstPolicy_DistinguishingRow(t *testing.T) {
	conditions := makeConditions(
		[]string{`a`, `b`, `c`},
		[][]string{
			{"Y", "N"}, // row 0
			{"Y", "N"}, // row 1 — both columns differ here too, so col2's row0=N is NOT forced
			{"Y", "Y"}, // row 2
		},
	)
	actions := makeActions(
		[]string{`set x`, `set y`},
		[][]string{{"X", ""}, {"", "X"}},
	)
	warns := Analyze(Inputs{
		Name:       "T",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     2,
	})
	for _, w := range warns {
		if w.Kind == "redundant condition" {
			t.Errorf("did not expect redundant-condition warning when columns differ on multiple rows, got %v", w)
		}
	}
}

// TestAnalyze_RedundantFirstPolicy_LeaveOneRule is the #794 regression.
// In pure FIRST semantics, every non-dash cell in a column can be
// individually proven redundant by a prior column's failure. But the
// runtime treats an all-dash column as having no discriminator — if the
// operator applies every recommendation, the column collapses and a
// downstream catch-all stops firing.
//
// Reproduces the staking project's `Calculate_Weights` table: 4 columns
// with the catch-all (col 4) carrying three non-dash cells, all of
// which the analyzer used to flag. Applying all three made the runtime
// crash on null when later tables tried to consume variables the
// catch-all was supposed to have set.
//
// After the fix, the analyzer emits at most `K-1` warnings per column
// with K non-dash cells. The "leave one" rule keeps the column
// explicit; authors who really want it gone delete the whole column.
//
// Asserted properties:
//   - Exactly 2 redundant-condition warnings for col 4 (not 3).
//   - The remaining warnings point at rows 0 and 1 (lowest row indices).
//   - The highest-row candidate (row 2) is the one suppressed,
//     deterministically.
func TestAnalyze_RedundantFirstPolicy_LeaveOneRule(t *testing.T) {
	// Calculate_Weights shape from #794:
	//   cond 1: effective_type == "coreValidator"      col1=Y col4=N
	//   cond 2: effective_type == "coreFollower"       col2=Y col4=N
	//   cond 3: effective_type == "stakingValidator"   col3=Y col4=N
	// All three col-4 cells are independently provable redundant via
	// col 1 / col 2 / col 3 failures respectively.
	conditions := makeConditions(
		[]string{
			`effective_type == "coreValidator"`,
			`effective_type == "coreFollower"`,
			`effective_type == "stakingValidator"`,
		},
		[][]string{
			{"Y", "-", "-", "N"}, // row 0 — flagged by col 1's failure
			{"-", "Y", "-", "N"}, // row 1 — flagged by col 2's failure
			{"-", "-", "Y", "N"}, // row 2 — flagged by col 3's failure
		},
	)
	actions := makeActions(
		[]string{`set weight = 1.3`, `set weight = 1.0`},
		[][]string{
			{"X", "X", "X", ""},
			{"", "", "", "X"},
		},
	)
	warns := Analyze(Inputs{
		Name:       "Calculate_Weights",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     4,
	})

	col4 := 0
	gotRows := map[int]bool{}
	for _, w := range warns {
		if w.Kind == "redundant condition" && w.Column == 4 {
			col4++
			gotRows[w.ConditionRow] = true
		}
	}

	// Pre-fix: col4 == 3 (every cell flagged, applying all = crash).
	// Post-fix: col4 == 2 (K-1 = 2; one cell stays explicit).
	if col4 != 2 {
		t.Errorf("expected exactly 2 redundant-condition warnings for col 4 "+
			"(K-1 with K=3); got %d (applying all 3 would all-dash the column "+
			"and break the runtime per #794)", col4)
	}
	// Sorted iteration suppresses the highest-row candidate. Rows 0 and
	// 1 should be flagged; row 2 should not.
	if !gotRows[0] {
		t.Error("expected redundant warning for col 4 row 0")
	}
	if !gotRows[1] {
		t.Error("expected redundant warning for col 4 row 1")
	}
	if gotRows[2] {
		t.Error("did not expect redundant warning for col 4 row 2 — that's the cell the leave-one rule keeps")
	}
}

// TestAnalyze_RedundantFirstPolicy_SingleCellInColumn is the sub-case
// the staking issue called out separately: a column whose ONLY non-dash
// cell is redundant. The old behavior flagged it (K=1 cells, 1
// candidate). Applying it dashes the column → runtime crash. The fix
// caps at K-1 = 0; no warning.
//
// This is the pattern shared by the seven staking tables filed in #794
// (Calculate_Budget, Resolve_Effective_Types, etc.) where col 2 has
// just one cell `cond 1 = N` and the analyzer recommended dashing it.
func TestAnalyze_RedundantFirstPolicy_SingleCellInColumn(t *testing.T) {
	conditions := makeConditions(
		[]string{`x > 0`, `y > 0`},
		[][]string{
			{"Y", "N"}, // row 0 — the lone col-2 cell; pre-fix flagged, post-fix not
			{"Y", "-"}, // row 1
		},
	)
	actions := makeActions(
		[]string{`set a`, `set b`},
		[][]string{{"X", ""}, {"", "X"}},
	)
	warns := Analyze(Inputs{
		Name:       "T",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     2,
	})
	for _, w := range warns {
		if w.Kind == "redundant condition" {
			t.Errorf("did not expect redundant-condition warning for the sole non-dash cell in a column (#794): %v", w)
		}
	}
}

// TestAnalyze_AssignmentOnlyTable covers #763: tables that only assign
// the same variable in every column should be flagged as candidates
// for inlining.
func TestAnalyze_AssignmentOnlyTable(t *testing.T) {
	conditions := makeConditions(
		[]string{`state == "CA"`, `state == "TX"`},
		[][]string{
			{"Y", "N", "N"},
			{"N", "Y", "N"},
		},
	)
	actions := makeActions(
		[]string{
			`set rate = 0.075`,
			`set rate = 0.0625`,
			`set rate = 0.05`,
		},
		[][]string{
			{"X", "", ""},
			{"", "X", ""},
			{"", "", "X"},
		},
	)
	warns := Analyze(Inputs{
		Name:       "PickRate",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     3,
	})
	found := false
	for _, w := range warns {
		if w.Kind == "assignment-only table" && strings.Contains(w.Reason, "rate") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected assignment-only warning naming `rate`, got %v", warns)
	}
}

// TestAnalyze_AssignmentOnlyTable_MixedActions: a table with any
// non-assignment action (add, perform, audit-trail) is NOT
// assignment-only. The check has to bail.
func TestAnalyze_AssignmentOnlyTable_MixedActions(t *testing.T) {
	conditions := makeConditions(
		[]string{`state == "CA"`},
		[][]string{{"Y", "N"}},
	)
	actions := makeActions(
		[]string{
			`set rate = 0.075`,                            // assignment
			`add "CA processed" to job.audit_trail`,       // not assignment
		},
		[][]string{
			{"X", ""},
			{"X", "X"}, // audit on both columns
		},
	)
	warns := Analyze(Inputs{
		Name:       "MixedTable",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     2,
	})
	for _, w := range warns {
		if w.Kind == "assignment-only table" {
			t.Errorf("did not expect assignment-only warning when actions include audit-trail, got %v", w)
		}
	}
}

// TestAnalyze_AssignmentOnlyTable_DifferentVars: a table where the
// assigned variable differs per column is NOT a candidate (inlining
// would lose the branching choice of WHICH variable to set).
func TestAnalyze_AssignmentOnlyTable_DifferentVars(t *testing.T) {
	conditions := makeConditions(
		[]string{`mode == "fast"`},
		[][]string{{"Y", "N"}},
	)
	actions := makeActions(
		[]string{
			`set speed = 100`,
			`set quality = "high"`,
		},
		[][]string{
			{"X", ""},
			{"", "X"},
		},
	)
	warns := Analyze(Inputs{
		Name:       "DifferingVars",
		Policy:     "FIRST",
		Conditions: conditions,
		Actions:    actions,
		MaxCol:     2,
	})
	for _, w := range warns {
		if w.Kind == "assignment-only table" {
			t.Errorf("did not expect assignment-only warning when columns assign different vars, got %v", w)
		}
	}
}

// TestAssignmentLHS unit-tests the assignment detector. It must accept
// the EL `set X = Y` form and only that form — `add X to Y`,
// `perform T`, multi-statement DSL, and `set X to Y` all return false.
func TestAssignmentLHS(t *testing.T) {
	cases := []struct {
		dsl     string
		wantOK  bool
		wantLHS string
	}{
		{"set rate = 0.075", true, "rate"},
		{"set rate = 0.075;", true, "rate"},
		{"  set  rate  =  0.075", true, "rate"},
		{"SET rate = 0.075", true, "rate"},
		{"set rate = a + b", true, "rate"},
		{"set result.tax = result.agi * 0.1", true, "result.tax"},
		// Compound statement — second statement disqualifies.
		{"set rate = 0.075; add 1 to count", false, ""},
		// Other EL forms.
		{"perform Calculate_Tax", false, ""},
		{"add 1 to count", false, ""},
		{"error \"x\"", false, ""},
		// Malformed.
		{"", false, ""},
		{"set", false, ""},
		{"set rate", false, ""},
		{"set = 5", false, ""},
		// EL convention requires spaces around `=`; `set x=5` is not
		// what the production DSL emits and we don't try to be clever.
		{"set rate=5", false, ""},
	}
	for _, c := range cases {
		ok, lhs := assignmentLHS(c.dsl)
		if ok != c.wantOK || lhs != c.wantLHS {
			t.Errorf("assignmentLHS(%q) = (%v, %q), want (%v, %q)", c.dsl, ok, lhs, c.wantOK, c.wantLHS)
		}
	}
}

func TestIsDSLNegation(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{`job.x is equal to "A"`, `job.x is not equal to "A"`, true},
		{`job.x is not equal to "A"`, `job.x is equal to "A"`, true},
		{`job.amount > 100`, `job.amount <= 100`, true},
		{`job.amount <= 100`, `job.amount > 100`, true},
		{`job.amount < 50`, `job.amount >= 50`, true},
		{`job.amount >= 50`, `job.amount < 50`, true},
		{`not job.active`, `job.active`, true},
		{`job.active`, `not job.active`, true},
		{`job.a > 5`, `job.b <= 5`, false},  // different identifiers
		{`job.a > 5`, `job.a > 5`, false},    // same, not negation
		{`job.a > 5`, `job.a < 5`, false},    // not a pair in our table
	}
	for _, tt := range tests {
		got := isNegationOf(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("isNegationOf(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
