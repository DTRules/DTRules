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
