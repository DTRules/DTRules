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

package trace

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewTraceNode(t *testing.T) {
	attrs := map[string]string{"id": "123", "name": "test"}
	node := NewTraceNode(1, "element", attrs)

	if node.Number != 1 {
		t.Errorf("Expected Number 1, got %d", node.Number)
	}
	if node.Name != "element" {
		t.Errorf("Expected Name 'element', got %s", node.Name)
	}
	if node.Attributes["id"] != "123" {
		t.Errorf("Expected Attributes['id'] '123', got %s", node.Attributes["id"])
	}
	if len(node.Children) != 0 {
		t.Errorf("Expected empty Children, got %d", len(node.Children))
	}
}

func TestAddChild(t *testing.T) {
	parent := NewTraceNode(1, "parent", nil)
	child := NewTraceNode(2, "child", nil)

	parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}
	if child.Parent != parent {
		t.Error("Child's parent should be set")
	}
}

func TestFind(t *testing.T) {
	root := NewTraceNode(1, "root", nil)
	child1 := NewTraceNode(2, "child1", nil)
	child2 := NewTraceNode(3, "child2", nil)
	grandchild := NewTraceNode(4, "grandchild", nil)

	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	tests := []struct {
		num      int
		expected *TraceNode
	}{
		{1, root},
		{2, child1},
		{3, child2},
		{4, grandchild},
		{5, nil},
	}

	for _, tt := range tests {
		result := root.Find(tt.num)
		if result != tt.expected {
			t.Errorf("Find(%d) = %v, expected %v", tt.num, result, tt.expected)
		}
	}
}

func TestGetActions(t *testing.T) {
	column := NewTraceNode(1, "column", map[string]string{"n": "1"})
	action1 := NewTraceNode(2, "action", map[string]string{"n": "1"})
	action2 := NewTraceNode(3, "action", map[string]string{"n": "2"})
	other := NewTraceNode(4, "other", nil)

	column.AddChild(action1)
	column.AddChild(action2)
	column.AddChild(other)

	actions := column.GetActions()
	if len(actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(actions))
	}
	if actions[0] != 1 || actions[1] != 2 {
		t.Errorf("Expected actions [1, 2], got %v", actions)
	}

	// Test getting actions from child
	childActions := action1.GetActions()
	if len(childActions) != 2 {
		t.Errorf("Expected 2 actions from child, got %d", len(childActions))
	}
}

func TestCount(t *testing.T) {
	root := NewTraceNode(1, "root", nil)
	child1 := NewTraceNode(2, "child1", nil)
	child2 := NewTraceNode(3, "child2", nil)
	grandchild := NewTraceNode(4, "grandchild", nil)

	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	if root.Count() != 4 {
		t.Errorf("Expected count 4, got %d", root.Count())
	}
}

func TestTraceNodeString(t *testing.T) {
	node := NewTraceNode(1, "test", map[string]string{"key": "value"})
	node.Body = "body content"

	str := node.String()
	if !strings.Contains(str, "test") {
		t.Error("String() should contain node name")
	}
	if !strings.Contains(str, "body content") {
		t.Error("String() should contain body")
	}
}

func TestPrint(t *testing.T) {
	root := NewTraceNode(1, "root", map[string]string{"id": "1"})
	child := NewTraceNode(2, "child", nil)
	child.Body = "test body"
	root.AddChild(child)

	var buf bytes.Buffer
	root.Print(&buf)

	output := buf.String()
	if !strings.Contains(output, "<root") {
		t.Error("Output should contain root tag")
	}
	if !strings.Contains(output, "</root>") {
		t.Error("Output should contain closing root tag")
	}
	if !strings.Contains(output, "test body") {
		t.Error("Output should contain child body")
	}
}

func TestWalkDepthFirst(t *testing.T) {
	root := NewTraceNode(1, "root", nil)
	child1 := NewTraceNode(2, "child1", nil)
	child2 := NewTraceNode(3, "child2", nil)
	grandchild := NewTraceNode(4, "grandchild", nil)

	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	var visited []int
	root.WalkDepthFirst(func(n *TraceNode) bool {
		visited = append(visited, n.Number)
		return true
	})

	expected := []int{1, 2, 4, 3}
	if len(visited) != len(expected) {
		t.Errorf("Expected %d nodes visited, got %d", len(expected), len(visited))
	}
	for i, v := range expected {
		if visited[i] != v {
			t.Errorf("Expected visit order %v, got %v", expected, visited)
			break
		}
	}
}

func TestWalkDepthFirstStop(t *testing.T) {
	root := NewTraceNode(1, "root", nil)
	child1 := NewTraceNode(2, "child1", nil)
	child2 := NewTraceNode(3, "child2", nil)

	root.AddChild(child1)
	root.AddChild(child2)

	var visited []int
	root.WalkDepthFirst(func(n *TraceNode) bool {
		visited = append(visited, n.Number)
		return n.Number != 2 // Stop at node 2
	})

	if len(visited) != 2 {
		t.Errorf("Expected 2 nodes visited (stopped at 2), got %d", len(visited))
	}
}

// Test loader
func TestLoad(t *testing.T) {
	xml := `<DTRulesTrace>
		<createentity name="test" id="100"></createentity>
		<entitypush entity="test" id="100"></entitypush>
		<def id="100" entity="test" name="attr">value</def>
		<entitypop></entitypop>
	</DTRulesTrace>`

	root, err := Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if root.Name != "DTRulesTrace" {
		t.Errorf("Expected root name 'DTRulesTrace', got %s", root.Name)
	}

	if len(root.Children) != 4 {
		t.Errorf("Expected 4 children, got %d", len(root.Children))
	}
}

func TestLoadAndCount(t *testing.T) {
	xml := `<root>
		<child1/>
		<child2>
			<grandchild/>
		</child2>
	</root>`

	root, count, err := LoadAndCount(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("LoadAndCount failed: %v", err)
	}

	if root.Name != "root" {
		t.Errorf("Expected root name 'root', got %s", root.Name)
	}

	if count != 4 {
		t.Errorf("Expected count 4, got %d", count)
	}
}

// Test change tracking
func TestChangeTracker(t *testing.T) {
	tracker := NewChangeTracker()

	executeTable := NewTraceNode(100, "execute_table", nil)
	change := NewChange(1, "attribute1", executeTable)
	tracker.Record(change)

	if !tracker.IsChanged(1, "attribute1") {
		t.Error("Expected attribute1 to be marked as changed")
	}

	if tracker.IsChanged(1, "attribute2") {
		t.Error("Expected attribute2 to not be marked as changed")
	}

	if tracker.IsChanged(2, "attribute1") {
		t.Error("Expected entity 2 to not be marked as changed")
	}

	execTable := tracker.GetExecuteTable(1, "attribute1")
	if execTable != executeTable {
		t.Error("Expected to get back the execute table")
	}

	if tracker.Count() != 1 {
		t.Errorf("Expected count 1, got %d", tracker.Count())
	}

	tracker.Clear()
	if tracker.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", tracker.Count())
	}
}

func TestNewTrace(t *testing.T) {
	tr := NewTrace()
	if tr == nil {
		t.Fatal("NewTrace returned nil")
	}
	if tr.Root() != nil {
		t.Error("New trace should have nil root")
	}
}

func TestTraceLoad(t *testing.T) {
	xml := `<DTRulesTrace>
		<createentity name="client" id="10001"></createentity>
		<entitypush entity="client" id="10001"></entitypush>
		<def id="10001" entity="client" name="age">25</def>
		<entitypop></entitypop>
	</DTRulesTrace>`

	tr := NewTrace()
	root, err := tr.LoadReader(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if root != tr.Root() {
		t.Error("Load should set and return root")
	}

	found := tr.Find(1)
	if found == nil || found.Name != "DTRulesTrace" {
		t.Error("Find should return the root node")
	}
}

func TestTracePrint(t *testing.T) {
	xml := `<root><child/></root>`

	tr := NewTrace()
	_, err := tr.LoadReader(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	var buf bytes.Buffer
	tr.Print(&buf)

	output := buf.String()
	if !strings.Contains(output, "<root") {
		t.Error("Print should output root tag")
	}
}

func TestFindEnclosingDecisionTable(t *testing.T) {
	dt := NewTraceNode(1, "decisiontable", map[string]string{"name": "TestTable"})
	execTable := NewTraceNode(2, "execute_table", nil)
	column := NewTraceNode(3, "column", map[string]string{"n": "1"})
	action := NewTraceNode(4, "action", map[string]string{"n": "1"})

	dt.AddChild(execTable)
	execTable.AddChild(column)
	column.AddChild(action)

	found := action.FindEnclosingDecisionTable()
	if found != dt {
		t.Error("Should find enclosing decision table")
	}

	if dt.FindEnclosingDecisionTable() != dt {
		t.Error("Decision table should find itself")
	}

	orphan := NewTraceNode(5, "orphan", nil)
	if orphan.FindEnclosingDecisionTable() != nil {
		t.Error("Orphan node should not find decision table")
	}
}

func TestFindEnclosingExecuteTable(t *testing.T) {
	execTable := NewTraceNode(1, "execute_table", nil)
	column := NewTraceNode(2, "column", nil)
	action := NewTraceNode(3, "action", nil)

	execTable.AddChild(column)
	column.AddChild(action)

	found := action.FindEnclosingExecuteTable()
	if found != execTable {
		t.Error("Should find enclosing execute_table")
	}
}

func TestGetDecisionTableName(t *testing.T) {
	dt := NewTraceNode(1, "decisiontable", map[string]string{"name": "TestTable"})
	if dt.GetDecisionTableName() != "TestTable" {
		t.Errorf("Expected 'TestTable', got '%s'", dt.GetDecisionTableName())
	}

	other := NewTraceNode(2, "other", nil)
	if other.GetDecisionTableName() != "" {
		t.Error("Non-decisiontable node should return empty string")
	}
}

func TestGetColumn(t *testing.T) {
	column := NewTraceNode(1, "column", map[string]string{"n": "1 2 3"})
	if column.GetColumn() != "1 2 3" {
		t.Errorf("Expected '1 2 3', got '%s'", column.GetColumn())
	}

	other := NewTraceNode(2, "other", nil)
	if other.GetColumn() != "" {
		t.Error("Non-column node should return empty string")
	}
}

func TestGetCondition(t *testing.T) {
	cond := NewTraceNode(1, "Condition", map[string]string{"n": "1", "v": "Y"})
	num, val, ok := cond.GetCondition()
	if !ok || num != 1 || val != "Y" {
		t.Errorf("Expected (1, Y, true), got (%d, %s, %v)", num, val, ok)
	}

	other := NewTraceNode(2, "other", nil)
	_, _, ok = other.GetCondition()
	if ok {
		t.Error("Non-Condition node should return ok=false")
	}
}

func TestGetArrayID(t *testing.T) {
	node1 := NewTraceNode(1, "addto", map[string]string{"arrayID": "123"})
	if node1.GetArrayID() != 123 {
		t.Errorf("Expected 123, got %d", node1.GetArrayID())
	}

	node2 := NewTraceNode(2, "addto", map[string]string{"arrayId": "456"})
	if node2.GetArrayID() != 456 {
		t.Errorf("Expected 456, got %d", node2.GetArrayID())
	}

	node3 := NewTraceNode(3, "other", nil)
	if node3.GetArrayID() != 0 {
		t.Errorf("Expected 0 for missing arrayID, got %d", node3.GetArrayID())
	}
}
