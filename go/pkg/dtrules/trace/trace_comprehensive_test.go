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

// =============================================================================
// Change Key and String tests
// =============================================================================

func TestChangeKey(t *testing.T) {
	c := NewChange(42, "name", nil)
	key := c.Key()
	if key.entityID != 42 {
		t.Errorf("expected entityID 42, got %d", key.entityID)
	}
	if key.attributeKey != "name" {
		t.Errorf("expected attributeKey 'name', got %q", key.attributeKey)
	}
}

func TestChangeKeySameChange(t *testing.T) {
	c1 := NewChange(1, "attr", nil)
	c2 := NewChange(1, "attr", nil)
	if c1.Key() != c2.Key() {
		t.Error("same entity/attribute should produce equal keys")
	}
}

func TestChangeKeyDifferentEntity(t *testing.T) {
	c1 := NewChange(1, "attr", nil)
	c2 := NewChange(2, "attr", nil)
	if c1.Key() == c2.Key() {
		t.Error("different entities should produce different keys")
	}
}

func TestChangeKeyDifferentAttribute(t *testing.T) {
	c1 := NewChange(1, "attr1", nil)
	c2 := NewChange(1, "attr2", nil)
	if c1.Key() == c2.Key() {
		t.Error("different attributes should produce different keys")
	}
}

func TestChangeString(t *testing.T) {
	node := NewTraceNode(10, "execute_table", nil)
	c := NewChange(42, "status", node)
	str := c.String()

	if !strings.Contains(str, "42") {
		t.Error("String should contain entity ID")
	}
	if !strings.Contains(str, "status") {
		t.Error("String should contain attribute name")
	}
	if !strings.Contains(str, "10") {
		t.Error("String should contain execute table node number")
	}
}

func TestChangeStringNilExecuteTable(t *testing.T) {
	c := NewChange(1, "attr", nil)
	str := c.String()
	if !strings.Contains(str, "-1") {
		t.Error("String should contain -1 for nil execute table")
	}
}

// =============================================================================
// ChangeTracker tests
// =============================================================================

func TestChangeTrackerGet(t *testing.T) {
	tracker := NewChangeTracker()
	node := NewTraceNode(1, "execute_table", nil)
	c := NewChange(10, "age", node)
	tracker.Record(c)

	got := tracker.Get(10, "age")
	if got != c {
		t.Error("Get should return the recorded change")
	}

	got = tracker.Get(10, "name")
	if got != nil {
		t.Error("Get should return nil for unrecorded attribute")
	}

	got = tracker.Get(99, "age")
	if got != nil {
		t.Error("Get should return nil for unrecorded entity")
	}
}

func TestChangeTrackerOverwrite(t *testing.T) {
	tracker := NewChangeTracker()
	node1 := NewTraceNode(1, "execute_table", nil)
	node2 := NewTraceNode(2, "execute_table", nil)

	tracker.Record(NewChange(1, "attr", node1))
	tracker.Record(NewChange(1, "attr", node2))

	if tracker.Count() != 1 {
		t.Errorf("expected count 1 after overwrite, got %d", tracker.Count())
	}

	got := tracker.Get(1, "attr")
	if got.ExecuteTable != node2 {
		t.Error("should have the latest change")
	}
}

func TestChangeTrackerAll(t *testing.T) {
	tracker := NewChangeTracker()
	tracker.Record(NewChange(1, "a", nil))
	tracker.Record(NewChange(2, "b", nil))
	tracker.Record(NewChange(3, "c", nil))

	all := tracker.All()
	if len(all) != 3 {
		t.Errorf("expected 3 changes, got %d", len(all))
	}
}

func TestChangeTrackerGetExecuteTableNil(t *testing.T) {
	tracker := NewChangeTracker()
	node := tracker.GetExecuteTable(99, "nonexistent")
	if node != nil {
		t.Error("should return nil for nonexistent change")
	}
}

// =============================================================================
// TraceNode GetEntity tests
// =============================================================================

func TestGetEntity(t *testing.T) {
	node := NewTraceNode(1, "entitypush", map[string]string{
		"id":     "100",
		"entity": "person",
	})

	id, name := node.GetEntity()
	if id != "100" {
		t.Errorf("expected id '100', got %q", id)
	}
	if name != "person" {
		t.Errorf("expected name 'person', got %q", name)
	}
}

func TestGetEntityMissing(t *testing.T) {
	node := NewTraceNode(1, "other", nil)
	id, name := node.GetEntity()
	if id != "" || name != "" {
		t.Error("expected empty strings for missing attributes")
	}
}

// =============================================================================
// TraceNode SearchTree tests
// =============================================================================

func TestSearchTree(t *testing.T) {
	tr := NewTrace()

	root := NewTraceNode(1, "DTRulesTrace", nil)
	create1 := NewTraceNode(2, "createentity", map[string]string{"name": "person", "id": "100"})
	create2 := NewTraceNode(3, "createentity", map[string]string{"name": "job", "id": "200"})
	create3 := NewTraceNode(4, "createentity", map[string]string{"name": "person", "id": "101"})

	root.AddChild(create1)
	root.AddChild(create2)
	root.AddChild(create3)

	tr.root = root
	tr.position = create3 // Stop at node 4

	var ids []int
	root.SearchTree(tr, "person", &ids)

	if len(ids) != 2 {
		t.Errorf("expected 2 person entities, got %d", len(ids))
	}
}

func TestSearchTreeNoDuplicates(t *testing.T) {
	tr := NewTrace()

	root := NewTraceNode(1, "root", nil)
	create1 := NewTraceNode(2, "createentity", map[string]string{"name": "person", "id": "100"})
	create2 := NewTraceNode(3, "createentity", map[string]string{"name": "person", "id": "100"}) // same ID

	root.AddChild(create1)
	root.AddChild(create2)

	tr.root = root
	tr.position = create2

	var ids []int
	root.SearchTree(tr, "person", &ids)

	if len(ids) != 1 {
		t.Errorf("expected 1 unique ID, got %d", len(ids))
	}
}

// =============================================================================
// Trace getters tests
// =============================================================================

func TestTraceGetters(t *testing.T) {
	tr := NewTrace()

	// GetPosition should return nil before SetState
	if tr.GetPosition() != nil {
		t.Error("GetPosition should return nil initially")
	}

	// GetSession should return nil before SetState
	if tr.GetSession() != nil {
		t.Error("GetSession should return nil initially")
	}

	// GetEntityTable should return empty (not nil)
	et := tr.GetEntityTable()
	if et == nil {
		t.Error("GetEntityTable should not return nil")
	}
	if len(et) != 0 {
		t.Error("GetEntityTable should return empty map initially")
	}

	// GetArrayTable should return empty
	at := tr.GetArrayTable()
	if at == nil {
		t.Error("GetArrayTable should not return nil")
	}
	if len(at) != 0 {
		t.Error("GetArrayTable should return empty map initially")
	}

	// GetExecuteTable should return nil
	if tr.GetExecuteTable() != nil {
		t.Error("GetExecuteTable should return nil initially")
	}

	// GetChanges should return empty
	changes := tr.GetChanges()
	if len(changes) != 0 {
		t.Error("GetChanges should return empty initially")
	}
}

func TestTraceGetActionsNilPosition(t *testing.T) {
	tr := NewTrace()
	actions := tr.GetActions()
	if actions != nil {
		t.Error("GetActions should return nil when position is nil")
	}
}

func TestTraceGetActionsAtNilNode(t *testing.T) {
	tr := NewTrace()
	actions := tr.GetActionsAt(nil)
	if actions != nil {
		t.Error("GetActionsAt should return nil for nil node")
	}
}

func TestTraceGetActionsAt(t *testing.T) {
	tr := NewTrace()
	column := NewTraceNode(1, "column", map[string]string{"n": "1"})
	action1 := NewTraceNode(2, "action", map[string]string{"n": "5"})
	action2 := NewTraceNode(3, "action", map[string]string{"n": "6"})
	column.AddChild(action1)
	column.AddChild(action2)

	actions := tr.GetActionsAt(column)
	if len(actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(actions))
	}
}

func TestTraceIsChanged(t *testing.T) {
	tr := NewTrace()
	// Without any state set, changes should be empty
	node := tr.IsChanged(1, "attr")
	if node != nil {
		t.Error("IsChanged should return nil when no changes recorded")
	}
}

func TestTraceIsDefaultValue(t *testing.T) {
	tr := NewTrace()
	if !tr.IsDefaultValue(1, "attr") {
		t.Error("IsDefaultValue should return true when no changes recorded")
	}
}

func TestTraceInstancesOfNoSession(t *testing.T) {
	tr := NewTrace()
	result := tr.InstancesOf("person")
	if result != nil {
		t.Error("InstancesOf should return nil when session is nil")
	}
}

func TestTraceInstancesOfNoRoot(t *testing.T) {
	tr := NewTrace()
	result := tr.InstancesOf("person")
	if result != nil {
		t.Error("InstancesOf should return nil when root is nil")
	}
}

func TestTraceFindNilRoot(t *testing.T) {
	tr := NewTrace()
	result := tr.Find(1)
	if result != nil {
		t.Error("Find should return nil when root is nil")
	}
}

func TestTracePrintNilRoot(t *testing.T) {
	tr := NewTrace()
	var buf bytes.Buffer
	tr.Print(&buf)
	if !strings.Contains(buf.String(), "No tree has been loaded") {
		t.Error("Print should indicate no tree loaded")
	}
}

// =============================================================================
// TraceLoader tests
// =============================================================================

func TestNewTraceLoader(t *testing.T) {
	loader := NewTraceLoader()
	if loader == nil {
		t.Fatal("NewTraceLoader returned nil")
	}
	if loader.GetNodeCount() != 0 {
		t.Errorf("expected 0 node count, got %d", loader.GetNodeCount())
	}
}

func TestTraceLoaderGetNodeCount(t *testing.T) {
	xml := `<root><child1/><child2/></root>`
	root, count, err := LoadAndCount(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("LoadAndCount failed: %v", err)
	}
	if root == nil {
		t.Fatal("root is nil")
	}
	if count != 3 {
		t.Errorf("expected 3 nodes, got %d", count)
	}
}

func TestLoadEmptyXML(t *testing.T) {
	_, err := Load(strings.NewReader(""))
	if err == nil {
		t.Error("expected error for empty XML")
	}
}

func TestLoadFileNonExistent(t *testing.T) {
	_, err := LoadFile("/nonexistent/path.xml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// =============================================================================
// TraceNode Print edge cases
// =============================================================================

func TestPrintSelfClosingNode(t *testing.T) {
	node := NewTraceNode(1, "empty", map[string]string{"id": "1"})
	var buf bytes.Buffer
	node.Print(&buf)
	output := buf.String()
	if !strings.Contains(output, "/>") {
		t.Error("node with no children and no body should self-close")
	}
}

func TestPrintNodeWithBody(t *testing.T) {
	node := NewTraceNode(1, "def", map[string]string{"name": "age"})
	node.Body = "25"
	var buf bytes.Buffer
	node.Print(&buf)
	output := buf.String()
	if !strings.Contains(output, ">25</def>") {
		t.Errorf("node with body should contain body, got %q", output)
	}
}

func TestPrintNodeWithChildren(t *testing.T) {
	parent := NewTraceNode(1, "parent", nil)
	child := NewTraceNode(2, "child", nil)
	parent.AddChild(child)

	var buf bytes.Buffer
	parent.Print(&buf)
	output := buf.String()
	if !strings.Contains(output, "</parent>") {
		t.Error("node with children should have closing tag")
	}
	if !strings.Contains(output, "<child") {
		t.Error("should contain child element")
	}
}

// =============================================================================
// escapeXML tests
// =============================================================================

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<tag>", "&lt;tag&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"", ""},
	}
	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// =============================================================================
// GetActions edge cases
// =============================================================================

func TestGetActionsNotColumn(t *testing.T) {
	// A node that is not a column and has no parent
	node := NewTraceNode(1, "other", nil)
	actions := node.GetActions()
	if actions != nil {
		t.Error("GetActions on non-column node without parent should return nil")
	}
}

func TestGetActionsInvalidNumber(t *testing.T) {
	column := NewTraceNode(1, "column", nil)
	action := NewTraceNode(2, "action", map[string]string{"n": "invalid"})
	column.AddChild(action)

	actions := column.GetActions()
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for invalid number, got %d", len(actions))
	}
}

// =============================================================================
// GetArrayID edge cases
// =============================================================================

func TestGetArrayIDInvalid(t *testing.T) {
	node := NewTraceNode(1, "addto", map[string]string{"arrayID": "invalid"})
	if node.GetArrayID() != 0 {
		t.Error("expected 0 for non-numeric arrayID")
	}
}

// =============================================================================
// GetCondition edge cases
// =============================================================================

func TestGetConditionInvalidNumber(t *testing.T) {
	node := NewTraceNode(1, "Condition", map[string]string{"n": "abc", "v": "Y"})
	_, _, ok := node.GetCondition()
	if ok {
		t.Error("expected ok=false for invalid condition number")
	}
}

// =============================================================================
// Count single node
// =============================================================================

func TestCountSingleNode(t *testing.T) {
	node := NewTraceNode(1, "leaf", nil)
	if node.Count() != 1 {
		t.Errorf("expected count 1 for leaf, got %d", node.Count())
	}
}

// =============================================================================
// WalkDepthFirst with single node
// =============================================================================

func TestWalkDepthFirstSingleNode(t *testing.T) {
	node := NewTraceNode(1, "leaf", nil)
	count := 0
	node.WalkDepthFirst(func(n *TraceNode) bool {
		count++
		return true
	})
	if count != 1 {
		t.Errorf("expected 1 visit, got %d", count)
	}
}

// =============================================================================
// XML with body text and attributes
// =============================================================================

func TestLoadXMLWithBody(t *testing.T) {
	xml := `<root>
		<def name="age" entity="person">25</def>
		<def name="status" entity="job">active</def>
	</root>`

	root, err := Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}

	child1 := root.Children[0]
	if child1.Body != "25" {
		t.Errorf("expected body '25', got %q", child1.Body)
	}
	if child1.Attributes["name"] != "age" {
		t.Errorf("expected name 'age', got %q", child1.Attributes["name"])
	}

	child2 := root.Children[1]
	if child2.Body != "active" {
		t.Errorf("expected body 'active', got %q", child2.Body)
	}
}

// =============================================================================
// Trace Load from reader
// =============================================================================

func TestTraceLoadReader(t *testing.T) {
	xml := `<DTRulesTrace>
		<createentity name="test" id="1"/>
		<entitypush entity="test" id="1"/>
	</DTRulesTrace>`

	tr := NewTrace()
	root, err := tr.LoadReader(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("LoadReader failed: %v", err)
	}

	if root != tr.Root() {
		t.Error("LoadReader should set and return root")
	}
	if root.Name != "DTRulesTrace" {
		t.Errorf("expected root name 'DTRulesTrace', got %q", root.Name)
	}
}
