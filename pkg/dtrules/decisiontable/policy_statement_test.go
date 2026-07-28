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

import "testing"

// policyTable builds a table whose policy statements are indexed by 1-based
// column number, matching what the loader hands the builder.
func policyTable(statements ...string) *RDecisionTable {
	dt := &RDecisionTable{}
	dt.policyStatements = append([]string{""}, statements...)
	return dt
}

// TestEqualsNodeSeparatesDistinctPolicyStatements is the guard for #949: the
// optimizer collapses branches that run the same actions, which erases which
// column got there. When each column carries its own policy statement, that
// erasure is a behavior change — the fired column's statement is part of what
// the column does — so such nodes must not compare equal.
func TestEqualsNodeSeparatesDistinctPolicyStatements(t *testing.T) {
	dt := policyTable("value is 1", "value is 2")
	col1 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{1}}
	col2 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{2}}

	if col1.EqualsNode(nil, col2) {
		t.Error("columns with different policy statements must not be interchangeable")
	}
}

// TestEqualsNodeMergesSharedPolicyStatements keeps the optimization working
// where it is safe: same actions and the same statement (typically none at
// all) means the columns really are interchangeable.
func TestEqualsNodeMergesSharedPolicyStatements(t *testing.T) {
	dt := policyTable("", "")
	col1 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{1}}
	col2 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{2}}

	if !col1.EqualsNode(nil, col2) {
		t.Error("columns with no policy statements and the same actions must still merge")
	}

	same := policyTable("same text", "same text")
	a := &ANode{decisionTable: same, actionNumbers: []int{0}, columns: []int{1}}
	b := &ANode{decisionTable: same, actionNumbers: []int{0}, columns: []int{2}}
	if !a.EqualsNode(nil, b) {
		t.Error("columns whose policy statements are identical must still merge")
	}
}

// TestEqualsNodeStillComparesActions confirms the policy check is additional,
// not a replacement: differing action sets remain unequal.
func TestEqualsNodeStillComparesActions(t *testing.T) {
	dt := policyTable("", "")
	col1 := &ANode{decisionTable: dt, actionNumbers: []int{0}, columns: []int{1}}
	col2 := &ANode{decisionTable: dt, actionNumbers: []int{1}, columns: []int{2}}

	if col1.EqualsNode(nil, col2) {
		t.Error("nodes running different actions must not be interchangeable")
	}
}

// TestPolicyStatementOfMergedNode covers a node that already answers for
// several columns: it is only mergeable further if those columns agree.
func TestPolicyStatementOfMergedNode(t *testing.T) {
	dt := policyTable("a", "a", "b")

	agreeing := &ANode{decisionTable: dt, columns: []int{1, 2}}
	if got, ok := agreeing.policyStatement(); !ok || got != "a" {
		t.Errorf("policyStatement() = %q, %v; want \"a\", true", got, ok)
	}

	disagreeing := &ANode{decisionTable: dt, columns: []int{1, 3}}
	if _, ok := disagreeing.policyStatement(); ok {
		t.Error("a node spanning columns with different statements must report disagreement")
	}
}

// TestColumnsAccessor is what the policystatements operator reads through.
func TestColumnsAccessor(t *testing.T) {
	a := &ANode{columns: []int{2, 5}}
	got := a.Columns()
	if len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Errorf("Columns() = %v, want [2 5]", got)
	}
}
