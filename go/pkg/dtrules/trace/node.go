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

// Package trace provides trace analysis capabilities for DTRules.
//
// It can parse XML trace files generated during rule execution and
// reconstruct the rules engine state at any point in the trace tree.
// This is useful for debugging rule logic and understanding how the
// engine arrived at a particular result.
//
// The main entry point is [Trace], which provides a high-level API
// for loading traces, navigating the tree, reconstructing state, and
// tracking attribute changes. The trace tree is made up of [TraceNode]
// values, each corresponding to an XML element in the trace file.
package trace

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// TraceNode represents a node in the trace tree structure.
// Each node corresponds to an XML element in the trace file.
type TraceNode struct {
	Number     int               // Sequential node number assigned during loading
	Name       string            // XML tag name
	Attributes map[string]string // XML attributes
	Children   []*TraceNode      // Child nodes
	Body       string            // Text content of the element
	Parent     *TraceNode        // Parent node (nil for root)
}

// NewTraceNode creates a new TraceNode with the given parameters.
func NewTraceNode(number int, name string, attributes map[string]string) *TraceNode {
	return &TraceNode{
		Number:     number,
		Name:       name,
		Attributes: attributes,
		Children:   make([]*TraceNode, 0),
	}
}

// Find searches the tree for a node with the given number.
// Returns nil if not found.
func (n *TraceNode) Find(num int) *TraceNode {
	if n.Number == num {
		return n
	}
	for _, child := range n.Children {
		if found := child.Find(num); found != nil {
			return found
		}
	}
	return nil
}

// AddChild adds a child node and sets its parent reference.
func (n *TraceNode) AddChild(child *TraceNode) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// GetActions returns the action numbers from the current node or nearest
// enclosing column tag. Returns nil if no column is found.
func (n *TraceNode) GetActions() []int {
	// If this is a column node, extract action numbers from children
	if n.Name == "column" {
		actions := make([]int, 0)
		for _, child := range n.Children {
			if child.Name == "action" {
				if numStr, ok := child.Attributes["n"]; ok {
					if num, err := strconv.Atoi(numStr); err == nil {
						actions = append(actions, num)
					}
				}
			}
		}
		return actions
	}

	// Otherwise, look to parent
	if n.Parent != nil {
		return n.Parent.GetActions()
	}

	return nil
}

// GetEntity returns the entity ID and name from this node's attributes.
// Returns empty strings if not present.
func (n *TraceNode) GetEntity() (id string, name string) {
	return n.Attributes["id"], n.Attributes["entity"]
}

// GetArrayID returns the array ID from the arrayID or arrayId attribute.
func (n *TraceNode) GetArrayID() int {
	if idStr, ok := n.Attributes["arrayID"]; ok {
		if id, err := strconv.Atoi(idStr); err == nil {
			return id
		}
	}
	if idStr, ok := n.Attributes["arrayId"]; ok {
		if id, err := strconv.Atoi(idStr); err == nil {
			return id
		}
	}
	return 0
}

// Print outputs the trace node as XML to the writer.
func (n *TraceNode) Print(w io.Writer) {
	n.printIndent(w, 0)
}

func (n *TraceNode) printIndent(w io.Writer, indent int) {
	prefix := strings.Repeat("  ", indent)

	// Build attribute string using a builder to avoid repeated concatenation
	var attrs strings.Builder
	fmt.Fprintf(&attrs, " t_num=\"%d\"", n.Number)
	for k, v := range n.Attributes {
		fmt.Fprintf(&attrs, " %s=\"%s\"", k, escapeXML(v))
	}
	attrStr := attrs.String()

	if len(n.Children) > 0 {
		fmt.Fprintf(w, "%s<%s%s>\n", prefix, n.Name, attrStr)
		for _, child := range n.Children {
			child.printIndent(w, indent+1)
		}
		fmt.Fprintf(w, "%s</%s>\n", prefix, n.Name)
	} else if n.Body != "" {
		fmt.Fprintf(w, "%s<%s%s>%s</%s>\n", prefix, n.Name, attrStr, escapeXML(n.Body), n.Name)
	} else {
		fmt.Fprintf(w, "%s<%s%s/>\n", prefix, n.Name, attrStr)
	}
}

// String returns a string representation of the node.
func (n *TraceNode) String() string {
	return fmt.Sprintf("%s (%s) %v", n.Name, n.Body, n.Attributes)
}

// SearchTree recursively searches for entity creation ("createentity") nodes
// matching entityName, collecting their IDs into entityList. The search stops
// when it reaches the Trace's current position. Returns true if the position
// was reached during the search.
func (n *TraceNode) SearchTree(t *Trace, entityName string, entityList *[]int) bool {
	if n.Name == "createentity" {
		if strings.EqualFold(n.Attributes["name"], entityName) {
			id := n.Attributes["id"]
			if idInt, err := strconv.Atoi(id); err == nil {
				// Check if already in list
				found := false
				for _, existing := range *entityList {
					if existing == idInt {
						found = true
						break
					}
				}
				if !found {
					*entityList = append(*entityList, idInt)
				}
			}
		}
	}

	if n == t.position {
		return true
	}

	for _, child := range n.Children {
		if child.SearchTree(t, entityName, entityList) {
			return true
		}
	}
	return false
}

// escapeXML escapes special characters for XML output.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// GetDecisionTableName returns the decision table name if this is a
// decisiontable node, otherwise returns empty string.
func (n *TraceNode) GetDecisionTableName() string {
	if n.Name == "decisiontable" {
		return n.Attributes["name"]
	}
	return ""
}

// GetColumn returns the column specification if this is a column node.
func (n *TraceNode) GetColumn() string {
	if n.Name == "column" {
		return strings.TrimSpace(n.Attributes["n"])
	}
	return ""
}

// GetCondition returns the condition number and value if this is a Condition node.
func (n *TraceNode) GetCondition() (num int, value string, ok bool) {
	if n.Name == "Condition" {
		numStr := n.Attributes["n"]
		value = n.Attributes["v"]
		if num, err := strconv.Atoi(numStr); err == nil {
			return num, value, true
		}
	}
	return 0, "", false
}

// FindEnclosingDecisionTable walks up the tree to find the enclosing
// decisiontable node.
func (n *TraceNode) FindEnclosingDecisionTable() *TraceNode {
	current := n
	for current != nil {
		if current.Name == "decisiontable" {
			return current
		}
		current = current.Parent
	}
	return nil
}

// FindEnclosingExecuteTable walks up the tree to find the enclosing
// execute_table node.
func (n *TraceNode) FindEnclosingExecuteTable() *TraceNode {
	current := n
	for current != nil {
		if current.Name == "execute_table" {
			return current
		}
		current = current.Parent
	}
	return nil
}

// WalkDepthFirst traverses the tree depth-first, calling the visitor
// function for each node. If the visitor returns false, traversal stops.
func (n *TraceNode) WalkDepthFirst(visitor func(*TraceNode) bool) bool {
	if !visitor(n) {
		return false
	}
	for _, child := range n.Children {
		if !child.WalkDepthFirst(visitor) {
			return false
		}
	}
	return true
}

// Count returns the total number of nodes in the subtree rooted at this node.
func (n *TraceNode) Count() int {
	count := 1
	for _, child := range n.Children {
		count += child.Count()
	}
	return count
}
