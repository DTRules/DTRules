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
	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

// Coordinate represents a position in the decision table (row, column)
type Coordinate struct {
	Row int
	Col int
}

// DTNode is the interface for nodes in a decision tree.
// The decision tree is built from CNodes (condition nodes) and ANodes (action nodes).
type DTNode interface {
	// GetRow returns the row number for condition nodes, -1 for action nodes
	GetRow() int

	// CloneDTNode returns a deep copy of this node and all sub-nodes
	CloneDTNode() DTNode

	// CountColumns returns the number of columns (paths) through this node
	CountColumns() int

	// Execute evaluates this node and its children
	Execute(state dtrules.State) error

	// Validate checks that the decision tree is complete
	// Returns nil if valid, or a Coordinate pointing to the problem location
	Validate() *Coordinate

	// EqualsNode returns true if both nodes execute the same actions
	// for all possible paths
	EqualsNode(state dtrules.State, node DTNode) bool

	// GetCommonANode returns an ANode that represents the common execution
	// path through this node. Returns nil if different paths execute
	// different actions.
	GetCommonANode(state dtrules.State) *ANode

	// AddNode combines another node into this node (used during optimization)
	// Returns error if nodes cannot be combined (type mismatch)
	AddNode(node DTNode) error

	// GetStar returns whether this column has a star (*) entry
	GetStar() bool

	// SetStar sets the star flag for this column
	SetStar(star bool)
}
