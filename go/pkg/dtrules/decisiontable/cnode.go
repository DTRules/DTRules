// Copyright 2004-2011 DTRules.com, Inc.
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

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// CNode is a Condition Node that evaluates conditions within a Decision Table.
// It forms the internal nodes of the decision tree.
type CNode struct {
	column          int             // Column that created this node
	conditionNumber int             // Row number (0-based) of the condition
	condition       dtrules.Object  // The compiled condition code
	decisionTable   *RDecisionTable // Pointer back to the Decision Table
	star            bool            // Whether this column has a star

	IfTrue  DTNode // Branch taken if condition is true
	IfFalse DTNode // Branch taken if condition is false
}

// NewCNode creates a new condition node
func NewCNode(dt *RDecisionTable, column, conditionNumber int, condition dtrules.Object) *CNode {
	return &CNode{
		column:          column,
		conditionNumber: conditionNumber,
		condition:       condition,
		decisionTable:   dt,
	}
}

// GetRow returns the condition row number (0-based)
func (c *CNode) GetRow() int {
	return c.conditionNumber
}

// CloneDTNode creates a deep copy of this CNode and all its sub-nodes
func (c *CNode) CloneDTNode() DTNode {
	newNode := &CNode{
		column:          c.column,
		conditionNumber: c.conditionNumber,
		condition:       c.condition,
		decisionTable:   c.decisionTable,
		star:            c.star,
	}
	if c.IfTrue != nil {
		newNode.IfTrue = c.IfTrue.CloneDTNode()
	}
	if c.IfFalse != nil {
		newNode.IfFalse = c.IfFalse.CloneDTNode()
	}
	return newNode
}

// CountColumns returns the total number of columns (paths) through this node
func (c *CNode) CountColumns() int {
	t := 0
	f := 0
	if c.IfTrue != nil {
		t = c.IfTrue.CountColumns()
	}
	if c.IfFalse != nil {
		f = c.IfFalse.CountColumns()
	}
	return t + f
}

// Execute evaluates the condition and follows the appropriate branch
func (c *CNode) Execute(state dtrules.State) error {
	// Evaluate the condition
	result, err := state.EvaluateCondition(c.condition)
	if err != nil {
		return fmt.Errorf("condition %d in table %s: %w",
			c.conditionNumber+1, c.decisionTable.GetName(), err)
	}

	// Follow the appropriate branch
	if result {
		if c.IfTrue != nil {
			return c.IfTrue.Execute(state)
		}
	} else {
		if c.IfFalse != nil {
			return c.IfFalse.Execute(state)
		}
	}
	return nil
}

// Validate checks that both branches are properly connected
func (c *CNode) Validate() *Coordinate {
	if c.IfTrue == nil || c.IfFalse == nil {
		return &Coordinate{Row: c.conditionNumber, Col: c.column}
	}

	result := c.IfFalse.Validate()
	if result != nil {
		return result
	}

	return c.IfTrue.Validate()
}

// EqualsNode returns true if both nodes execute the same actions for all paths
func (c *CNode) EqualsNode(state dtrules.State, node DTNode) bool {
	if other, ok := node.(*CNode); ok {
		if other.conditionNumber != c.conditionNumber {
			return false
		}
		return other.IfFalse.EqualsNode(state, c.IfFalse) &&
			other.IfTrue.EqualsNode(state, c.IfTrue)
	}

	// Compare with an ANode by getting common paths
	me := c.GetCommonANode(state)
	if me == nil {
		return false
	}
	him := node.GetCommonANode(state)
	if him == nil {
		return false
	}
	return me.EqualsNode(state, him)
}

// GetCommonANode returns an ANode if both branches lead to the same actions
func (c *CNode) GetCommonANode(state dtrules.State) *ANode {
	// Both paths must lead to the same result
	if !c.IfTrue.EqualsNode(state, c.IfFalse) {
		return nil
	}

	trueSide := c.IfTrue.GetCommonANode(state)
	falseSide := c.IfFalse.GetCommonANode(state)
	if trueSide == nil || falseSide == nil {
		return nil
	}

	// Merge the two sides (for column tracking)
	// Ignore error here as we've already verified they're compatible
	_ = trueSide.AddNode(falseSide)
	return trueSide
}

// AddNode combines another node into this node
// Returns error if the nodes are incompatible types
func (c *CNode) AddNode(node DTNode) error {
	other, ok := node.(*CNode)
	if !ok {
		return dtrules.TypeCheckError("CNode.AddNode", "cannot combine CNode with different node type")
	}
	if err := c.IfTrue.AddNode(other.IfTrue); err != nil {
		return err
	}
	return c.IfFalse.AddNode(other.IfFalse)
}

// GetStar returns whether this column has a star
func (c *CNode) GetStar() bool {
	return c.star
}

// SetStar sets the star flag
func (c *CNode) SetStar(star bool) {
	c.star = star
}

// GetConditionNumber returns the condition row number (0-based)
func (c *CNode) GetConditionNumber() int {
	return c.conditionNumber
}

// GetColumn returns the column that created this node
func (c *CNode) GetColumn() int {
	return c.column
}

// String returns a string representation of this CNode
func (c *CNode) String() string {
	return fmt.Sprintf("Condition Number %d", c.conditionNumber+1)
}
