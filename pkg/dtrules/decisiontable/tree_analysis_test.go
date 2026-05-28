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

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// newTreeTestTable returns a stripped-down RDecisionTable suitable for
// the tree-walk tests. We can't use NewRDecisionTable because it
// requires a Session (which would pull in compilation, the entity
// factory, etc.) — but the tree-analysis checks only touch
// name / maxCol / decisionTree / conditions, so building the struct
// directly is fine.
func newTreeTestTable(name string, maxCol int, conditions []string, tree DTNode) *RDecisionTable {
	dt := &RDecisionTable{
		name:         dtrules.GetRName(name),
		maxCol:       maxCol,
		conditions:   conditions,
		decisionTree: tree,
	}
	return dt
}

// TestAnalyzeCompiledTable_NilTable covers the defensive nil paths so
// callers from the loader / authoring code can invoke this without
// guards even when compilation hasn't run.
func TestAnalyzeCompiledTable_NilTable(t *testing.T) {
	if got := AnalyzeCompiledTable(nil); got != nil {
		t.Errorf("nil table: want nil warnings, got %v", got)
	}
}

// TestAnalyzeCompiledTable_NilTree covers a table that exists but
// never compiled (e.g. loader skipped it because of a parse error).
// We don't crash — just return nil.
func TestAnalyzeCompiledTable_NilTree(t *testing.T) {
	dt := newTreeTestTable("T", 0, nil, nil)
	if got := AnalyzeCompiledTable(dt); got != nil {
		t.Errorf("nil tree: want nil warnings, got %v", got)
	}
}

// TestCheckUnreachableColumnsByTree_AllReached: every column appears
// on at least one ANode, so no #765 warning fires.
func TestCheckUnreachableColumnsByTree_AllReached(t *testing.T) {
	tree := &CNode{
		conditionNumber: 0,
		IfTrue:          &ANode{columns: []int{1}},
		IfFalse: &CNode{
			conditionNumber: 1,
			IfTrue:          &ANode{columns: []int{2}},
			IfFalse:         &ANode{columns: []int{3}},
		},
	}
	dt := newTreeTestTable("AllReached", 3, []string{`a`, `b`}, tree)
	ws := AnalyzeCompiledTable(dt)
	for _, w := range ws {
		if w.Kind == "unreachable column" {
			t.Errorf("unexpected unreachable warning %v", w)
		}
	}
}

// TestCheckUnreachableColumnsByTree_OneUnreached: maxCol=3 but only
// columns 1 and 3 appear on ANode leaves. Column 2 must be flagged.
func TestCheckUnreachableColumnsByTree_OneUnreached(t *testing.T) {
	tree := &CNode{
		conditionNumber: 0,
		IfTrue:          &ANode{columns: []int{1}},
		IfFalse:         &ANode{columns: []int{3}},
	}
	dt := newTreeTestTable("OneUnreached", 3, []string{`a`}, tree)
	ws := AnalyzeCompiledTable(dt)
	found := false
	for _, w := range ws {
		if w.Kind == "unreachable column" && w.Column == 2 {
			found = true
		}
		if w.Kind == "unreachable column" && (w.Column == 1 || w.Column == 3) {
			t.Errorf("did not expect unreachable warning for column %d", w.Column)
		}
	}
	if !found {
		t.Errorf("expected unreachable warning for column 2, got %v", ws)
	}
}

// TestCheckDeadConditionsByTree_AllBranched: every condition row
// number appears on some CNode → no dead-row warning.
func TestCheckDeadConditionsByTree_AllBranched(t *testing.T) {
	tree := &CNode{
		conditionNumber: 0,
		IfTrue: &CNode{
			conditionNumber: 1,
			IfTrue:          &ANode{columns: []int{1}},
			IfFalse:         &ANode{columns: []int{2}},
		},
		IfFalse: &ANode{columns: []int{2}},
	}
	dt := newTreeTestTable("AllBranched", 2, []string{`a`, `b`}, tree)
	ws := AnalyzeCompiledTable(dt)
	for _, w := range ws {
		if w.Kind == "dead condition row" {
			t.Errorf("unexpected dead-row warning %v", w)
		}
	}
}

// TestCheckDeadConditionsByTree_RowSkipped: two condition rows in
// the table but only row 0 branches in the tree. Row 1 is dead.
func TestCheckDeadConditionsByTree_RowSkipped(t *testing.T) {
	tree := &CNode{
		conditionNumber: 0,
		IfTrue:          &ANode{columns: []int{1}},
		IfFalse:         &ANode{columns: []int{2}},
	}
	dt := newTreeTestTable("RowSkipped", 2, []string{`a`, `b`}, tree)
	ws := AnalyzeCompiledTable(dt)
	found := false
	for _, w := range ws {
		if w.Kind == "dead condition row" && w.ConditionRow == 1 {
			found = true
		}
		if w.Kind == "dead condition row" && w.ConditionRow == 0 {
			t.Errorf("did not expect dead-row warning for row 0")
		}
	}
	if !found {
		t.Errorf("expected dead-row warning for row 1, got %v", ws)
	}
}

// TestCheckRedundantActionSetColumns_TwoActionSets is the #797 happy
// path: two action sets in the table, the second set is reached by
// three columns, and only one of those three is the "first" (useful)
// one. Cols 3 and 4 must be flagged as redundant action-set columns
// pointing at col 2 as the earlier column that already covers their
// action set.
//
// This reproduces the staking Calculate_Withholding shape: col 1
// fires action set A (full withholding actions), cols 2/3/4 all fire
// action set B (zero-withholding fallback) under different
// condition combinations.
func TestCheckRedundantActionSetColumns_TwoActionSets(t *testing.T) {
	// Tree shape: condition row 0 splits col 1 (Y) from "the rest".
	// Then row 1 splits col 2 vs col 3 vs col 4 — but we collapse to
	// the same action-set ANode (separate node instances; same
	// actionNumbers tuple) so the check sees three distinct ANodes
	// sharing an action-set signature.
	leafA := &ANode{actionNumbers: []int{0, 1, 2}, columns: []int{1}}
	leafB2 := &ANode{actionNumbers: []int{3, 4, 5}, columns: []int{2}}
	leafB3 := &ANode{actionNumbers: []int{3, 4, 5}, columns: []int{3}}
	leafB4 := &ANode{actionNumbers: []int{3, 4, 5}, columns: []int{4}}

	tree := &CNode{
		conditionNumber: 0,
		IfTrue:          leafA,
		IfFalse: &CNode{
			conditionNumber: 1,
			IfTrue:          leafB2,
			IfFalse: &CNode{
				conditionNumber: 2,
				IfTrue:          leafB3,
				IfFalse:         leafB4,
			},
		},
	}
	dt := newTreeTestTable("Calculate_Withholding_Shape", 4,
		[]string{`enabled`, `past_start`, `cap_not_reached`}, tree)

	ws := AnalyzeCompiledTable(dt)
	gotCols := map[int]string{}
	for _, w := range ws {
		if w.Kind == "redundant action-set column" {
			gotCols[w.Column] = w.Reason
		}
	}
	if len(gotCols) != 2 {
		t.Fatalf("expected exactly 2 redundant action-set warnings (cols 3, 4), got %d: %v", len(gotCols), gotCols)
	}
	for _, col := range []int{3, 4} {
		reason, ok := gotCols[col]
		if !ok {
			t.Errorf("expected warning for column %d, got %v", col, gotCols)
			continue
		}
		// The earlier column reported should be col 2, not col 3.
		if !strings.Contains(reason, "column 2") {
			t.Errorf("col %d warning should point at col 2 as earlier; got %q", col, reason)
		}
	}
	if _, gotCol1 := gotCols[1]; gotCol1 {
		t.Error("col 1 introduces its action set; must not be flagged")
	}
	if _, gotCol2 := gotCols[2]; gotCol2 {
		t.Error("col 2 is the first to reach action set B; must not be flagged")
	}
}

// TestCheckRedundantActionSetColumns_DistinctActionSets confirms the
// check is silent when every column has a distinct action set — the
// table genuinely needs each column to express different behavior.
func TestCheckRedundantActionSetColumns_DistinctActionSets(t *testing.T) {
	tree := &CNode{
		conditionNumber: 0,
		IfTrue:          &ANode{actionNumbers: []int{0}, columns: []int{1}},
		IfFalse: &CNode{
			conditionNumber: 1,
			IfTrue:          &ANode{actionNumbers: []int{1}, columns: []int{2}},
			IfFalse:         &ANode{actionNumbers: []int{2}, columns: []int{3}},
		},
	}
	dt := newTreeTestTable("DistinctActions", 3, []string{`a`, `b`}, tree)
	for _, w := range AnalyzeCompiledTable(dt) {
		if w.Kind == "redundant action-set column" {
			t.Errorf("did not expect redundant action-set warning, got %v", w)
		}
	}
}

// TestCheckRedundantActionSetColumns_OrderMatters confirms the check
// treats different action orders as different action sets — the
// runtime executes actions in order and order affects observable
// behavior (audit-trail line ordering, sequence between sets and
// reads). [0,1] and [1,0] are distinct.
func TestCheckRedundantActionSetColumns_OrderMatters(t *testing.T) {
	tree := &CNode{
		conditionNumber: 0,
		IfTrue:          &ANode{actionNumbers: []int{0, 1}, columns: []int{1}},
		IfFalse:         &ANode{actionNumbers: []int{1, 0}, columns: []int{2}},
	}
	dt := newTreeTestTable("OrderMatters", 2, []string{`a`}, tree)
	for _, w := range AnalyzeCompiledTable(dt) {
		if w.Kind == "redundant action-set column" {
			t.Errorf("different action orders must not be flagged as same set, got %v", w)
		}
	}
}
