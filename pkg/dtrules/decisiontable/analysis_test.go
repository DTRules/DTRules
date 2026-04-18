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
