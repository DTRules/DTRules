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

// AnalyzeCompiledTable runs the post-compile advisory checks on a
// decision table whose tree has already been built (Build / Compile).
// These checks need the compiled tree because they answer
// "what does the optimizer actually reach?", not just
// "what does the matrix say?":
//
//   - #765 unreachable column: no ANode leaf in the tree references the
//     column, so no execution path can fire it.
//   - #766 dead condition row: no CNode in the tree branches on the
//     row, so the row contributes nothing to column selection.
//
// Callers that only have the XML matrix (the live authoring loop)
// should use Analyze instead; it skips the tree-based checks rather
// than crashing on a nil tree.
//
// If the table hasn't been compiled or has no tree (validation error,
// loader skipped it), AnalyzeCompiledTable returns nil — there's
// nothing to walk.
func AnalyzeCompiledTable(dt *RDecisionTable) []Warning {
	if dt == nil {
		return nil
	}
	root := dt.GetDecisionTree()
	if root == nil {
		return nil
	}
	var warnings []Warning
	warnings = append(warnings, checkUnreachableColumnsByTree(dt, root)...)
	warnings = append(warnings, checkDeadConditionsByTree(dt, root)...)
	warnings = append(warnings, checkRedundantActionSetColumns(dt, root)...)
	return warnings
}

// checkUnreachableColumnsByTree is the #765 check: walk every ANode
// leaf, union its `columns` slice into a reached set, and report any
// column in [1..maxCol] that no leaf claims. This catches:
//
//   - Exact-duplicate columns (the second copy never reached).
//   - First-policy columns whose conditions are a superset of an
//     earlier column's with no distinguishing test.
//   - Any other unreachability the tree builder has already proven by
//     collapsing equivalent subtrees.
//
// The complementary checkSubsumedColumns (matrix-driven) catches a
// narrower case based on action sets; this one is strictly
// reachability and runs as a separate signal.
func checkUnreachableColumnsByTree(dt *RDecisionTable, root DTNode) []Warning {
	maxCol := dt.GetMaxCol()
	if maxCol <= 0 {
		return nil
	}
	reached := make(map[int]struct{}, maxCol)
	walkTree(root, func(a *ANode, _ *CNode) {
		if a == nil {
			return
		}
		for _, col := range a.columns {
			reached[col] = struct{}{}
		}
	})
	var warnings []Warning
	for col := 1; col <= maxCol; col++ {
		if _, ok := reached[col]; ok {
			continue
		}
		w := newWarning(dt.GetName(), "unreachable column",
			"no decision-tree leaf reaches this column; consider removing it")
		w.Column = col
		warnings = append(warnings, w)
	}
	return warnings
}

// checkDeadConditionsByTree is the #766 check: walk every CNode,
// union its `conditionNumber` into a branchedOn set, and report any
// condition row whose number never appears. Naturally catches all-`-`
// rows (the tree builder emits no CNode for them) and rows the
// optimizer collapsed because both outcomes lead to the same subtree.
//
// CNode.conditionNumber is 0-based; the Warning's ConditionRow field
// is also 0-based (per the JSON contract from #761).
func checkDeadConditionsByTree(dt *RDecisionTable, root DTNode) []Warning {
	numConditions := len(dt.GetConditions())
	if numConditions == 0 {
		return nil
	}
	branchedOn := make(map[int]struct{}, numConditions)
	walkTree(root, func(_ *ANode, c *CNode) {
		if c == nil {
			return
		}
		branchedOn[c.conditionNumber] = struct{}{}
	})
	var warnings []Warning
	for row := 0; row < numConditions; row++ {
		if _, ok := branchedOn[row]; ok {
			continue
		}
		w := newWarning(dt.GetName(), "dead condition row",
			"no tree node branches on this condition — it contributes nothing to column selection")
		w.ConditionRow = row
		warnings = append(warnings, w)
	}
	return warnings
}

// checkRedundantActionSetColumns is the #797 check: a column is
// redundant if it doesn't introduce a new action-set combination
// into the table's reachable leaves. Walk the compiled tree, group
// columns by the ordered tuple of action numbers they reach, and
// for each group keep the smallest column number ("useful — first
// to reach this action set") while flagging every other column in
// the group ("redundant — same action set already covered by an
// earlier column").
//
// Stronger than the cell-level redundant-condition check (#762,
// capped at K-1 by #794's leave-one rule): that check operates on
// Y/N entries one cell at a time and tells the operator
// individual cells are forced; this one tells them entire columns
// are doing nothing new and points at the earlier column already
// covering the same behavior. Recommended remediation is to
// collapse the redundant columns into a single catch-all (`*` in
// every condition row) or remove them and let their inputs fall
// through to a downstream column.
//
// Action-set identity is the ordered tuple of action numbers
// reached at the leaf — same actions in a different order is NOT
// considered the same action set because the runtime executes
// them in order and a different order can have observably
// different effects (audit-trail line ordering, sequence
// dependencies between sets and reads).
//
// Pre-existing dead/unreachable columns from checkUnreachableColumnsByTree
// are skipped: a column with no ANode leaf wasn't compiled into
// the tree at all, so the "introduces this action set" question
// doesn't apply.
func checkRedundantActionSetColumns(dt *RDecisionTable, root DTNode) []Warning {
	maxCol := dt.GetMaxCol()
	if maxCol < 2 {
		return nil
	}

	// columnLeaf[col] = action-set signature reached by `col`. Built
	// by walking the tree once and recording every (col → ANode)
	// mapping. A column with no entry here is unreachable; the
	// caller already gets a separate warning for those.
	columnLeaf := make(map[int]string, maxCol)
	walkTree(root, func(a *ANode, _ *CNode) {
		if a == nil {
			return
		}
		sig := actionSetSignature(a.actionNumbers)
		for _, col := range a.columns {
			columnLeaf[col] = sig
		}
	})

	// firstColForActionSet[sig] = lowest column number reaching
	// this action set. Iterate columns in order so the lowest
	// number wins.
	firstColForActionSet := make(map[string]int, maxCol)
	var warnings []Warning
	for col := 1; col <= maxCol; col++ {
		sig, ok := columnLeaf[col]
		if !ok {
			continue // unreachable column — separate warning
		}
		earlier, seen := firstColForActionSet[sig]
		if !seen {
			firstColForActionSet[sig] = col
			continue
		}
		w := newWarning(dt.GetName(), "redundant action-set column",
			fmt.Sprintf("column %d reaches the same action set as column %d — consider collapsing into a single column (all conditions = '-') or removing if the inputs should fall through downstream",
				col, earlier))
		w.Column = col
		warnings = append(warnings, w)
	}
	return warnings
}

// actionSetSignature returns a stable string key for an ordered list
// of action numbers. Used to group columns by action-set identity in
// checkRedundantActionSetColumns. The exact format isn't observable
// — only the property that two equal sigs imply equal ordered
// tuples and vice versa.
func actionSetSignature(actionNumbers []int) string {
	if len(actionNumbers) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, n := range actionNumbers {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", n)
	}
	b.WriteByte(']')
	return b.String()
}

// walkTree does a depth-first walk over the compiled decision tree,
// invoking the visitor once per node with whichever concrete type the
// node is (the other pointer is nil). The visitor is the only seam
// the tree-based checks share — they each care about a different
// node kind, so muxing them through one walk keeps the traversal
// implementation in one place.
func walkTree(n DTNode, visit func(*ANode, *CNode)) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case *ANode:
		visit(v, nil)
	case *CNode:
		visit(nil, v)
		walkTree(v.IfTrue, visit)
		walkTree(v.IfFalse, visit)
	}
}
