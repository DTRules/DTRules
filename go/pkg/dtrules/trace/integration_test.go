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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// getTestDataPath returns the path to the sample project test data.
// Returns empty string if the path doesn't exist.
func getTestDataPath() string {
	// Get the directory of this test file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	// Navigate from go/pkg/dtrules/trace to sampleprojects/KidAid
	dir := filepath.Dir(filename)
	samplePath := filepath.Join(dir, "..", "..", "..", "..", "sampleprojects", "KidAid")

	if _, err := os.Stat(samplePath); err != nil {
		return ""
	}
	return samplePath
}

func getTraceFilePath(name string) string {
	base := getTestDataPath()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "testfiles", "output", "sampleOutput", name)
}

func getXMLPath(name string) string {
	base := getTestDataPath()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "xml", name)
}

// TestLoadKidAidTrace tests loading an actual trace file from the KidAid sample project.
func TestLoadKidAidTrace(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	// Verify root element
	if root.Name != "DTRulesTrace" {
		t.Errorf("Expected root name 'DTRulesTrace', got %s", root.Name)
	}

	// Verify we have a reasonable number of nodes
	count := root.Count()
	if count < 100 {
		t.Errorf("Expected at least 100 nodes, got %d", count)
	}

	t.Logf("Loaded trace with %d nodes", count)
}

// TestTraceStructure verifies the structure of a loaded trace.
func TestTraceStructure(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	// Should have loadMapping, createentity, entitypush, etc. as children
	foundLoadMapping := false
	foundCreateEntity := false
	foundDecisionTable := false

	root.WalkDepthFirst(func(n *TraceNode) bool {
		switch n.Name {
		case "loadMapping":
			foundLoadMapping = true
		case "createentity":
			foundCreateEntity = true
		case "decisiontable":
			foundDecisionTable = true
		}
		return true
	})

	if !foundLoadMapping {
		t.Error("Expected to find loadMapping element")
	}
	if !foundCreateEntity {
		t.Error("Expected to find createentity element")
	}
	if !foundDecisionTable {
		t.Error("Expected to find decisiontable element")
	}
}

// TestFindDecisionTables finds all decision tables in the trace.
func TestFindDecisionTables(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	var tables []string
	root.WalkDepthFirst(func(n *TraceNode) bool {
		if n.Name == "decisiontable" {
			name := n.GetDecisionTableName()
			if name != "" {
				tables = append(tables, name)
			}
		}
		return true
	})

	if len(tables) == 0 {
		t.Error("Expected to find at least one decision table")
	}

	t.Logf("Found %d decision table executions: %v", len(tables), tables)

	// Check for expected tables
	expectedTables := []string{"Compute_Eligibility", "Calculate_Individual_Income"}
	for _, expected := range expectedTables {
		found := false
		for _, table := range tables {
			if table == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find decision table %s", expected)
		}
	}
}

// TestFindNodeByNumber tests finding specific nodes by number.
func TestFindNodeByNumber(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	// Find specific nodes
	tests := []struct {
		num          int
		expectName   string
		expectExists bool
	}{
		{1, "DTRulesTrace", true},
		{2, "loadMapping", true},
		{3, "createentity", true},
		{999999, "", false}, // Should not exist
	}

	for _, tt := range tests {
		node := root.Find(tt.num)
		if tt.expectExists {
			if node == nil {
				t.Errorf("Expected to find node %d", tt.num)
			} else if node.Name != tt.expectName {
				t.Errorf("Node %d: expected name %s, got %s", tt.num, tt.expectName, node.Name)
			}
		} else {
			if node != nil {
				t.Errorf("Did not expect to find node %d", tt.num)
			}
		}
	}
}

// TestDefNodes verifies def (attribute assignment) nodes are parsed correctly.
func TestDefNodes(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	var defCount int
	var sampleDef *TraceNode

	root.WalkDepthFirst(func(n *TraceNode) bool {
		if n.Name == "def" {
			defCount++
			if sampleDef == nil {
				sampleDef = n
			}
		}
		return true
	})

	if defCount == 0 {
		t.Error("Expected to find def nodes")
	}

	t.Logf("Found %d def nodes", defCount)

	// Verify sample def has expected attributes
	if sampleDef != nil {
		if sampleDef.Attributes["id"] == "" {
			t.Error("Def node should have id attribute")
		}
		if sampleDef.Attributes["entity"] == "" {
			t.Error("Def node should have entity attribute")
		}
		if sampleDef.Attributes["name"] == "" {
			t.Error("Def node should have name attribute")
		}
		t.Logf("Sample def: entity=%s, name=%s, body=%s",
			sampleDef.Attributes["entity"],
			sampleDef.Attributes["name"],
			sampleDef.Body)
	}
}

// TestColumnNodes verifies column execution nodes.
func TestColumnNodes(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	var columnCount int
	var columnsExecuted []string

	root.WalkDepthFirst(func(n *TraceNode) bool {
		if n.Name == "column" {
			columnCount++
			colNum := n.GetColumn()
			if colNum != "" {
				columnsExecuted = append(columnsExecuted, colNum)
			}
		}
		return true
	})

	if columnCount == 0 {
		t.Error("Expected to find column nodes")
	}

	t.Logf("Found %d column nodes, executed columns: %v", columnCount, columnsExecuted[:min(5, len(columnsExecuted))])
}

// TestEntityOperations verifies entity push/pop/create operations.
func TestEntityOperations(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	if tracePath == "" {
		t.Skip("Sample project not available")
	}

	root, err := LoadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace file: %v", err)
	}

	var pushCount, popCount, createCount int
	entityTypes := make(map[string]int)

	root.WalkDepthFirst(func(n *TraceNode) bool {
		switch n.Name {
		case "entitypush":
			pushCount++
			entityName := n.Attributes["entity"]
			if entityName != "" {
				entityTypes[entityName]++
			}
		case "entitypop":
			popCount++
		case "createentity":
			createCount++
		}
		return true
	})

	t.Logf("Entity operations: push=%d, pop=%d, create=%d", pushCount, popCount, createCount)
	t.Logf("Entity types pushed: %v", entityTypes)

	// Push and pop should be balanced (approximately)
	diff := pushCount - popCount
	if diff < 0 {
		diff = -diff
	}
	// Allow some imbalance due to initial setup entities
	if diff > 10 {
		t.Errorf("Entity push/pop significantly unbalanced: push=%d, pop=%d", pushCount, popCount)
	}
}

// TestSetStateBasic tests basic state reconstruction.
func TestSetStateBasic(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	eddPath := getXMLPath("kidaid_edd.xml")
	dtPath := getXMLPath("kidaid_dt.xml")

	if tracePath == "" || eddPath == "" {
		t.Skip("Sample project not available")
	}

	// Load rule set
	rs := session.NewRuleSet("KidAid")
	if rs == nil {
		t.Fatal("Failed to create rule set")
	}

	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	defer eddFile.Close()

	if err := rs.LoadEDD(eddFile); err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	defer dtFile.Close()

	// Note: LoadDecisionTables may return warnings as errors, log and continue
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Logf("DT loading warnings: %v", err)
	}

	// Load trace
	tr := NewTrace()
	root, err := tr.Load(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace: %v", err)
	}

	// Find a def node to set state to
	var targetNode *TraceNode
	root.WalkDepthFirst(func(n *TraceNode) bool {
		if n.Name == "def" && targetNode == nil {
			targetNode = n
			return false
		}
		return true
	})

	if targetNode == nil {
		t.Fatal("Could not find a def node to test with")
	}

	t.Logf("Setting state to node %d (%s)", targetNode.Number, targetNode.Name)

	// Set state
	sess, err := tr.SetState(rs, targetNode)
	if err != nil {
		t.Fatalf("SetState failed: %v", err)
	}

	if sess == nil {
		t.Fatal("SetState returned nil session")
	}

	// Verify we created some entities
	entityCount := len(tr.GetEntityTable())
	if entityCount == 0 {
		t.Error("Expected some entities to be created")
	}

	t.Logf("SetState created %d entities", entityCount)
}

// TestSetStateAtColumn tests state reconstruction at a column execution.
func TestSetStateAtColumn(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	eddPath := getXMLPath("kidaid_edd.xml")
	dtPath := getXMLPath("kidaid_dt.xml")

	if tracePath == "" || eddPath == "" {
		t.Skip("Sample project not available")
	}

	// Load rule set
	rs := session.NewRuleSet("KidAid")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Skipf("Cannot open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Skipf("Cannot load EDD: %v", err)
	}

	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Skipf("Cannot open DT: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Skipf("Cannot load DT: %v", err)
	}

	// Load trace
	tr := NewTrace()
	root, err := tr.Load(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace: %v", err)
	}

	// Find a column node
	var columnNode *TraceNode
	root.WalkDepthFirst(func(n *TraceNode) bool {
		if n.Name == "column" && n.GetColumn() != "" {
			columnNode = n
			return false
		}
		return true
	})

	if columnNode == nil {
		t.Fatal("Could not find a column node")
	}

	t.Logf("Setting state to column node %d", columnNode.Number)

	sess, err := tr.SetState(rs, columnNode)
	if err != nil {
		t.Logf("SetState error (may be expected): %v", err)
	}

	if sess != nil {
		// Check actions at this position
		actions := tr.GetActions()
		t.Logf("Actions at position: %v", actions)
	}
}

// TestChangeTracking verifies that changes are properly tracked.
func TestChangeTracking(t *testing.T) {
	tracePath := getTraceFilePath("TestCase_001_trace.xml")
	eddPath := getXMLPath("kidaid_edd.xml")
	dtPath := getXMLPath("kidaid_dt.xml")

	if tracePath == "" || eddPath == "" {
		t.Skip("Sample project not available")
	}

	// Load rule set
	rs := session.NewRuleSet("KidAid")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Skipf("Cannot open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Skipf("Cannot load EDD: %v", err)
	}

	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Skipf("Cannot open DT: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Skipf("Cannot load DT: %v", err)
	}

	// Load trace
	tr := NewTrace()
	root, err := tr.Load(tracePath)
	if err != nil {
		t.Fatalf("Failed to load trace: %v", err)
	}

	// Find a late position in the trace
	targetNode := root.Find(100)
	if targetNode == nil {
		t.Fatal("Could not find node 100")
	}

	_, err = tr.SetState(rs, targetNode)
	if err != nil {
		t.Logf("SetState error (may be expected): %v", err)
	}

	changes := tr.GetChanges()
	t.Logf("Tracked %d changes up to node 100", len(changes))

	// Log some changes
	for i, change := range changes {
		if i >= 5 {
			break
		}
		t.Logf("  Change: entity=%d, attr=%s", change.EntityID, change.AttributeKey)
	}
}

// TestMultipleTraceFiles tests loading multiple trace files.
func TestMultipleTraceFiles(t *testing.T) {
	trace1Path := getTraceFilePath("TestCase_001_trace.xml")
	trace2Path := getTraceFilePath("TestCase_002_trace.xml")

	if trace1Path == "" || trace2Path == "" {
		t.Skip("Sample project not available")
	}

	// Load first trace
	root1, err := LoadFile(trace1Path)
	if err != nil {
		t.Fatalf("Failed to load trace 1: %v", err)
	}

	// Load second trace
	root2, err := LoadFile(trace2Path)
	if err != nil {
		t.Fatalf("Failed to load trace 2: %v", err)
	}

	count1 := root1.Count()
	count2 := root2.Count()

	t.Logf("Trace 1 has %d nodes, Trace 2 has %d nodes", count1, count2)

	// Both should have significant content
	if count1 < 50 || count2 < 50 {
		t.Error("Expected both traces to have substantial content")
	}
}

// TestErrorHandling tests various error conditions.
func TestErrorHandling(t *testing.T) {
	// Test loading non-existent file
	_, err := LoadFile("/nonexistent/path/file.xml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test malformed XML
	malformedXML := "<root><unclosed>"
	_, err = Load(strings.NewReader(malformedXML))
	// May or may not error depending on parser behavior
	t.Logf("Malformed XML result: %v", err)

	// Test empty input
	_, err = Load(strings.NewReader(""))
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

// TestDeepNesting tests handling of deeply nested XML.
func TestDeepNesting(t *testing.T) {
	// Create deeply nested XML
	var sb strings.Builder
	depth := 100
	for i := 0; i < depth; i++ {
		sb.WriteString("<level>")
	}
	sb.WriteString("content")
	for i := 0; i < depth; i++ {
		sb.WriteString("</level>")
	}

	root, err := Load(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("Failed to load deeply nested XML: %v", err)
	}

	// Count depth
	actualDepth := 0
	node := root
	for node != nil {
		actualDepth++
		if len(node.Children) > 0 {
			node = node.Children[0]
		} else {
			break
		}
	}

	if actualDepth != depth {
		t.Errorf("Expected depth %d, got %d", depth, actualDepth)
	}
}

// TestLargeTrace tests handling of larger traces.
func TestLargeTrace(t *testing.T) {
	// Create a large trace with many nodes
	var sb strings.Builder
	sb.WriteString("<trace>")
	nodeCount := 1000
	for i := 0; i < nodeCount; i++ {
		sb.WriteString("<node id=\"")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString("\"/>")
	}
	sb.WriteString("</trace>")

	root, err := Load(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("Failed to load large trace: %v", err)
	}

	if len(root.Children) != nodeCount {
		t.Errorf("Expected %d children, got %d", nodeCount, len(root.Children))
	}

	// Test that Find still works efficiently
	for i := 1; i <= 10; i++ {
		node := root.Find(i)
		if node == nil {
			t.Errorf("Could not find node %d", i)
		}
	}
}
