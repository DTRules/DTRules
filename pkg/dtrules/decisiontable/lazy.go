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

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// lazyLeafThreshold is the number of leaves above which ALL tables switch to lazy encoding.
const lazyLeafThreshold = 64

// lazyCondThreshold is the condition count above which FIRST and ALL tables
// switch to lazy encoding. The unbalanced tree grows with conditions as well
// as columns: TaxReturn's Dispatch_State_Tax -- FIRST policy, 43 conditions,
// ~51 columns, under the column threshold -- built a tree that peaked at
// 24.5GB and took 30 seconds, on every load until #1149 and on every first
// execution after it (#1148).
const lazyCondThreshold = 24

// condReq encodes the requirement a column has for a condition.
// 1 = requires Y, 0 = requires N, -1 = don't care.
type condReq int8

const (
	reqDontCare condReq = -1
	reqN        condReq = 0
	reqY        condReq = 1
)

// LazyTable is an alternate execution representation for ALL-policy tables
// whose compiled decision tree would exceed lazyLeafThreshold leaves.
//
// Instead of a binary tree, it keeps a flat list of columns. On each
// execution it maintains a set of "alive" columns, evaluates only the
// conditions needed to eliminate columns, and executes all surviving
// columns in authored order. Each condition is evaluated at most once.
type LazyTable struct {
	dt           *RDecisionTable
	conditions   []dtrules.Object // shared from dt.rconditions
	requirements [][]condReq      // requirements[col][condIdx]: reqY/reqN/reqDontCare
	columnNodes  []*ANode         // per-column action node, in authored order
	numCols      int
	numConds     int
	star         bool
	// first: execute only the first surviving column (FIRST policy). The
	// elimination loop is unchanged; only the final sweep differs.
	first bool
}

// buildLazyTable constructs a LazyTable from the decision table's raw data.
// It is called by buildUnbalanced when the tree exceeds lazyLeafThreshold leaves.
func buildLazyTable(dt *RDecisionTable) *LazyTable {
	numCols := dt.maxCol
	numConds := len(dt.conditions)

	reqs := make([][]condReq, numCols)
	for col := 0; col < numCols; col++ {
		colReqs := make([]condReq, numConds)
		for row := 0; row < numConds; row++ {
			colReqs[row] = reqDontCare
			if row < len(dt.conditionTable) && col < len(dt.conditionTable[row]) {
				v := dt.conditionTable[row][col]
				switch {
				case equalsIgnoreCase(v, "y"):
					colReqs[row] = reqY
				case equalsIgnoreCase(v, "n"):
					colReqs[row] = reqN
				}
			}
		}
		reqs[col] = colReqs
	}

	nodes := make([]*ANode, numCols)
	for col := 0; col < numCols; col++ {
		nodes[col] = NewANodeForColumn(dt, col)
	}

	return &LazyTable{
		dt:           dt,
		conditions:   dt.rconditions,
		requirements: reqs,
		columnNodes:  nodes,
		numCols:      numCols,
		numConds:     numConds,
	}
}

// Execute runs the lazy evaluation algorithm.
func (lt *LazyTable) Execute(state dtrules.State) error {
	alive := make([]bool, lt.numCols)
	for i := range alive {
		alive[i] = true
	}

	evaluated := make([]bool, lt.numConds)
	condValue := make([]bool, lt.numConds)

	for {
		// Find any condition referenced by an alive column that hasn't been evaluated yet.
		needed := -1
		for cond := 0; cond < lt.numConds; cond++ {
			if evaluated[cond] {
				continue
			}
			for col := 0; col < lt.numCols; col++ {
				if alive[col] && lt.requirements[col][cond] != reqDontCare {
					needed = cond
					break
				}
			}
			if needed >= 0 {
				break
			}
		}
		if needed < 0 {
			break
		}

		// Evaluate the condition exactly once.
		var condObj dtrules.Object
		if needed < len(lt.conditions) {
			condObj = lt.conditions[needed]
		}
		if condObj == nil {
			return fmt.Errorf("lazy table %s: nil condition at index %d",
				lt.dt.GetName(), needed)
		}

		v, err := state.EvaluateCondition(condObj)
		if err != nil {
			return fmt.Errorf("condition %d in table %s: %w",
				needed+1, lt.dt.GetName(), err)
		}
		evaluated[needed] = true
		condValue[needed] = v

		// Eliminate columns whose requirement for this condition is not met.
		want := reqY
		if !v {
			want = reqN
		}
		for col := 0; col < lt.numCols; col++ {
			if !alive[col] {
				continue
			}
			req := lt.requirements[col][needed]
			if req != reqDontCare && req != want {
				alive[col] = false
			}
		}
	}

	// Execute actions for the surviving columns in authored order -- all of
	// them, or only the first for a FIRST-policy table.
	for col := 0; col < lt.numCols; col++ {
		if alive[col] {
			if err := lt.columnNodes[col].Execute(state); err != nil {
				return err
			}
			if lt.first {
				break
			}
		}
	}
	return nil
}

// --- DTNode interface implementation ---

// GetRow returns -1; LazyTable is a leaf-like node.
func (lt *LazyTable) GetRow() int { return -1 }

// CloneDTNode returns the same instance; LazyTable is immutable after build.
func (lt *LazyTable) CloneDTNode() DTNode { return lt }

// CountColumns returns the number of authored columns.
func (lt *LazyTable) CountColumns() int { return lt.numCols }

// Validate always returns nil; lazy tables are structurally valid by construction.
func (lt *LazyTable) Validate() *Coordinate { return nil }

// EqualsNode is not used during optimization for lazy tables.
func (lt *LazyTable) EqualsNode(_ dtrules.State, node DTNode) bool {
	other, ok := node.(*LazyTable)
	if !ok {
		return false
	}
	return lt == other
}

// GetCommonANode returns nil; lazy tables don't participate in tree optimization.
func (lt *LazyTable) GetCommonANode(_ dtrules.State) *ANode { return nil }

// AddNode is not supported for lazy tables.
func (lt *LazyTable) AddNode(_ DTNode) error {
	return dtrules.TypeCheckError("LazyTable.AddNode", "lazy tables cannot be combined")
}

// GetStar returns the star flag.
func (lt *LazyTable) GetStar() bool { return lt.star }

// SetStar sets the star flag.
func (lt *LazyTable) SetStar(star bool) { lt.star = star }
