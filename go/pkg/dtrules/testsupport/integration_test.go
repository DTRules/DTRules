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

package testsupport

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCoverageWithSyntheticTrace tests coverage analysis with a synthetic trace file.
func TestCoverageWithSyntheticTrace(t *testing.T) {
	// Create a temp directory for trace files
	tempDir, err := os.MkdirTemp("", "coverage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a synthetic trace file
	traceContent := `<?xml version="1.0" encoding="UTF-8"?>
<DTRulesTrace>
	<decisiontable name='Test_Table_1'>
		<column n='1 2 '>
			<action n='1'>Some action</action>
		</column>
	</decisiontable>
	<decisiontable name='Test_Table_2'>
		<column n='3 '>
			<action n='2'>Another action</action>
		</column>
	</decisiontable>
	<decisiontable name='Test_Table_1'>
		<column n='1 '>
			<action n='1'>Repeat action</action>
		</column>
	</decisiontable>
</DTRulesTrace>`

	traceFile := filepath.Join(tempDir, "test_trace.xml")
	if err := os.WriteFile(traceFile, []byte(traceContent), 0644); err != nil {
		t.Fatalf("Failed to write trace file: %v", err)
	}

	// Parse the trace XML directly to verify structure
	root, err := ParseXMLTree(strings.NewReader(traceContent))
	if err != nil {
		t.Fatalf("Failed to parse trace XML: %v", err)
	}

	// Verify structure
	if root.Name != "DTRulesTrace" {
		t.Errorf("Expected root name 'DTRulesTrace', got %s", root.Name)
	}

	// Find all decision table nodes
	dtNodes := root.FindNodes("decisiontable")
	if len(dtNodes) != 3 {
		t.Errorf("Expected 3 decisiontable nodes, got %d", len(dtNodes))
	}

	// Check names
	expectedNames := []string{"Test_Table_1", "Test_Table_2", "Test_Table_1"}
	for i, dt := range dtNodes {
		if dt.Attributes["name"] != expectedNames[i] {
			t.Errorf("Expected table name %s at index %d, got %s",
				expectedNames[i], i, dt.Attributes["name"])
		}
	}

	// Find column nodes
	columnNodes := root.FindNodes("column")
	if len(columnNodes) != 3 {
		t.Errorf("Expected 3 column nodes, got %d", len(columnNodes))
	}
}

// TestCoverageParseColumnNumbers tests parsing of column numbers from trace.
func TestCoverageParseColumnNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"1 ", []int{1}},
		{"1 2 ", []int{1, 2}},
		{"3 ", []int{3}},
		{"1 9 ", []int{1, 9}},
		{"7 ", []int{7}},
		{"3 10 ", []int{3, 10}},
		{"15 ", []int{15}},
		{"", []int{}},
	}

	for _, tt := range tests {
		// Parse columns string
		cols := strings.Fields(tt.input)
		var parsed []int
		for _, col := range cols {
			var n int
			if _, err := parseColumnNumber(col, &n); err == nil {
				parsed = append(parsed, n)
			}
		}

		if len(parsed) != len(tt.expected) {
			t.Errorf("For input %q: expected %d columns, got %d",
				tt.input, len(tt.expected), len(parsed))
			continue
		}

		for i, exp := range tt.expected {
			if parsed[i] != exp {
				t.Errorf("For input %q: expected column %d at index %d, got %d",
					tt.input, exp, i, parsed[i])
			}
		}
	}
}

// Helper function to parse column number
func parseColumnNumber(s string, n *int) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, nil
	}
	_, err := parseInteger(s)
	if err != nil {
		return false, err
	}
	*n, _ = parseInteger(s)
	return true, nil
}

func parseInteger(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// TestStatsColumnTracking tests that Stats correctly tracks column hits.
func TestStatsColumnTracking(t *testing.T) {
	stats := NewStats("TestTable", 5, 3, 2)

	// Initially all zeros
	for i := 0; i < 5; i++ {
		if stats.ColumnHits[i] != 0 {
			t.Errorf("Expected initial ColumnHits[%d] to be 0, got %d", i, stats.ColumnHits[i])
		}
	}

	// Simulate hits
	stats.ColumnHits[0]++
	stats.ColumnHits[0]++
	stats.ColumnHits[2]++
	stats.CalledCount = 2

	if stats.ColumnHits[0] != 2 {
		t.Errorf("Expected ColumnHits[0] to be 2, got %d", stats.ColumnHits[0])
	}
	if stats.ColumnHits[2] != 1 {
		t.Errorf("Expected ColumnHits[2] to be 1, got %d", stats.ColumnHits[2])
	}
	if stats.ColumnHits[1] != 0 {
		t.Errorf("Expected ColumnHits[1] to be 0, got %d", stats.ColumnHits[1])
	}
}

// TestCoverageReportOutput tests the XML report output.
func TestCoverageReportOutput(t *testing.T) {
	// Create a minimal Coverage struct manually for testing report output
	c := &Coverage{
		tables:              make(map[string]*Stats),
		traceFilesProcessed: []string{"test1_trace.xml", "test2_trace.xml"},
		minFilesNeeded:      []string{"test1_trace.xml"},
		totalColumns:        15,
	}

	// Add some stats
	stats1 := NewStats("TableA", 3, 2, 2)
	stats1.CalledCount = 5
	stats1.ColumnHits[0] = 3
	stats1.ColumnHits[1] = 2
	stats1.ColumnsSpec[0] = true
	stats1.ColumnsSpec[1] = true
	stats1.ColumnsSpec[2] = true
	c.tables["TableA"] = stats1

	stats2 := NewStats("TableB", 2, 1, 1)
	stats2.CalledCount = 3
	stats2.ColumnHits[0] = 1
	stats2.ColumnsSpec[0] = true
	stats2.ColumnsSpec[1] = true
	c.tables["TableB"] = stats2

	// Generate report
	var buf bytes.Buffer
	c.PrintReport(&buf)
	output := buf.String()

	// Verify report content
	if !strings.Contains(output, "total_columns_executed=\"15\"") {
		t.Error("Report should contain total_columns_executed")
	}
	if !strings.Contains(output, "<minimum_files_for_coverage>") {
		t.Error("Report should contain minimum_files_for_coverage section")
	}
	if !strings.Contains(output, "test1_trace.xml") {
		t.Error("Report should contain test1_trace.xml")
	}
	if !strings.Contains(output, "<tables>") {
		t.Error("Report should contain tables section")
	}
	if !strings.Contains(output, "name=\"TableA\"") {
		t.Error("Report should contain TableA")
	}
	if !strings.Contains(output, "count_of_calls=\"5\"") {
		t.Error("Report should contain count_of_calls for TableA")
	}
}

// TestCoverageGetters tests the getter methods of Coverage.
func TestCoverageGetters(t *testing.T) {
	c := &Coverage{
		tables:              make(map[string]*Stats),
		traceFilesProcessed: []string{"a_trace.xml", "b_trace.xml"},
		minFilesNeeded:      []string{"a_trace.xml"},
		totalColumns:        42,
	}

	stats := NewStats("TestTable", 3, 2, 1)
	c.tables["TestTable"] = stats

	// Test GetStats
	gotStats := c.GetStats("TestTable")
	if gotStats != stats {
		t.Error("GetStats should return the correct stats")
	}

	nilStats := c.GetStats("NonExistent")
	if nilStats != nil {
		t.Error("GetStats should return nil for non-existent table")
	}

	// Test GetAllStats
	allStats := c.GetAllStats()
	if len(allStats) != 1 {
		t.Errorf("Expected 1 table in GetAllStats, got %d", len(allStats))
	}

	// Test GetTotalColumns
	if c.GetTotalColumns() != 42 {
		t.Errorf("Expected GetTotalColumns to return 42, got %d", c.GetTotalColumns())
	}

	// Test GetMinFilesNeeded
	minFiles := c.GetMinFilesNeeded()
	if len(minFiles) != 1 || minFiles[0] != "a_trace.xml" {
		t.Error("GetMinFilesNeeded returned wrong value")
	}

	// Test GetTraceFilesProcessed
	processed := c.GetTraceFilesProcessed()
	if len(processed) != 2 {
		t.Errorf("Expected 2 processed files, got %d", len(processed))
	}
}

// TestXMLTreeFromTraceFile tests parsing a real trace XML structure.
func TestXMLTreeFromTraceFile(t *testing.T) {
	traceXML := `<DTRulesTrace>
		<loadMapping file='test_map.xml'></loadMapping>
		<createentity name='job' id='1001'></createentity>
		<entitypush entity='job' id='1001'></entitypush>
		<def id='1001' entity='job' name='status'>"active"</def>
		<decisiontable name='Evaluate_Status'>
			<condition n='1'>job.status eq "active"</condition>
			<column n='1 '>
				<action n='1'>Mark as valid</action>
			</column>
		</decisiontable>
		<entitypop></entitypop>
	</DTRulesTrace>`

	root, err := ParseXMLTree(strings.NewReader(traceXML))
	if err != nil {
		t.Fatalf("Failed to parse trace XML: %v", err)
	}

	// Verify structure
	if root.Name != "DTRulesTrace" {
		t.Errorf("Expected root 'DTRulesTrace', got %s", root.Name)
	}

	// Find load mapping
	loadMap := root.FindTag("loadMapping")
	if loadMap == nil {
		t.Error("Should find loadMapping element")
	} else if loadMap.Attributes["file"] != "test_map.xml" {
		t.Errorf("Expected file='test_map.xml', got %s", loadMap.Attributes["file"])
	}

	// Find create entity
	createEntity := root.FindTag("createentity")
	if createEntity == nil {
		t.Error("Should find createentity element")
	} else {
		if createEntity.Attributes["name"] != "job" {
			t.Errorf("Expected name='job', got %s", createEntity.Attributes["name"])
		}
		if createEntity.Attributes["id"] != "1001" {
			t.Errorf("Expected id='1001', got %s", createEntity.Attributes["id"])
		}
	}

	// Find def node
	defNodes := root.FindNodes("def")
	if len(defNodes) != 1 {
		t.Errorf("Expected 1 def node, got %d", len(defNodes))
	} else {
		def := defNodes[0]
		if def.Attributes["entity"] != "job" {
			t.Errorf("Expected entity='job', got %s", def.Attributes["entity"])
		}
		if def.Attributes["name"] != "status" {
			t.Errorf("Expected name='status', got %s", def.Attributes["name"])
		}
	}

	// Find decision table
	dtNodes := root.FindNodes("decisiontable")
	if len(dtNodes) != 1 {
		t.Errorf("Expected 1 decisiontable, got %d", len(dtNodes))
	} else {
		dt := dtNodes[0]
		if dt.Attributes["name"] != "Evaluate_Status" {
			t.Errorf("Expected name='Evaluate_Status', got %s", dt.Attributes["name"])
		}

		// Check children of decision table
		conditionNodes := dt.FindNodes("condition")
		if len(conditionNodes) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(conditionNodes))
		}

		columnNodes := dt.FindNodes("column")
		if len(columnNodes) != 1 {
			t.Errorf("Expected 1 column, got %d", len(columnNodes))
		}

		actionNodes := dt.FindNodes("action")
		if len(actionNodes) != 1 {
			t.Errorf("Expected 1 action, got %d", len(actionNodes))
		}
	}
}

// TestChangeReportCompareEDD tests EDD comparison functionality.
func TestChangeReportCompareEDD(t *testing.T) {
	// Create two EDD structures
	edd1 := &XMLNode{
		Name: "entity_dictionary",
		Children: []*XMLNode{
			{
				Name:       "entity",
				Attributes: map[string]string{"name": "person"},
				Children: []*XMLNode{
					{Name: "attribute", Attributes: map[string]string{"name": "age", "type": "integer"}},
					{Name: "attribute", Attributes: map[string]string{"name": "name", "type": "string"}},
				},
			},
			{
				Name:       "entity",
				Attributes: map[string]string{"name": "job"},
				Children: []*XMLNode{
					{Name: "attribute", Attributes: map[string]string{"name": "title", "type": "string"}},
				},
			},
		},
	}

	edd2 := &XMLNode{
		Name: "entity_dictionary",
		Children: []*XMLNode{
			{
				Name:       "entity",
				Attributes: map[string]string{"name": "person"},
				Children: []*XMLNode{
					{Name: "attribute", Attributes: map[string]string{"name": "age", "type": "integer"}},
					{Name: "attribute", Attributes: map[string]string{"name": "name", "type": "string"}},
					{Name: "attribute", Attributes: map[string]string{"name": "email", "type": "string"}}, // New attribute
				},
			},
			// job entity removed
			{
				Name:       "entity",
				Attributes: map[string]string{"name": "address"}, // New entity
				Children: []*XMLNode{
					{Name: "attribute", Attributes: map[string]string{"name": "street", "type": "string"}},
				},
			},
		},
	}

	// Find entities
	entities1 := edd1.FindNodes("entity")
	entities2 := edd2.FindNodes("entity")

	if len(entities1) != 2 {
		t.Errorf("Expected 2 entities in edd1, got %d", len(entities1))
	}
	if len(entities2) != 2 {
		t.Errorf("Expected 2 entities in edd2, got %d", len(entities2))
	}

	// Check entity names
	entityNames := make(map[string]bool)
	for _, e := range entities1 {
		entityNames[e.Attributes["name"]] = true
	}
	if !entityNames["person"] || !entityNames["job"] {
		t.Error("Expected person and job entities in edd1")
	}

	entityNames = make(map[string]bool)
	for _, e := range entities2 {
		entityNames[e.Attributes["name"]] = true
	}
	if !entityNames["person"] || !entityNames["address"] {
		t.Error("Expected person and address entities in edd2")
	}
}

// TestChangeReportCompareMappings tests mapping comparison functionality.
func TestChangeReportCompareMappings(t *testing.T) {
	mapping1 := []*XMLNode{
		{Attributes: map[string]string{"tag": "name", "enclosure": "person", "RAttribute": "fullname"}},
		{Attributes: map[string]string{"tag": "age", "enclosure": "person", "RAttribute": "years"}},
	}

	mapping2 := []*XMLNode{
		{Attributes: map[string]string{"tag": "name", "enclosure": "person", "RAttribute": "fullname"}},
		{Attributes: map[string]string{"tag": "birthdate", "enclosure": "person", "RAttribute": "dob"}}, // Changed
	}

	// Sort mappings for consistent comparison
	sortMappings(mapping1)
	sortMappings(mapping2)

	// Check for matches
	target := &XMLNode{Attributes: map[string]string{"tag": "name", "enclosure": "person", "RAttribute": "fullname"}}
	if !hasMappingMatch(mapping1, target) {
		t.Error("Should find name mapping in mapping1")
	}
	if !hasMappingMatch(mapping2, target) {
		t.Error("Should find name mapping in mapping2")
	}

	targetAge := &XMLNode{Attributes: map[string]string{"tag": "age", "enclosure": "person", "RAttribute": "years"}}
	if !hasMappingMatch(mapping1, targetAge) {
		t.Error("Should find age mapping in mapping1")
	}
	if hasMappingMatch(mapping2, targetAge) {
		t.Error("Should NOT find age mapping in mapping2")
	}

	targetBirthdate := &XMLNode{Attributes: map[string]string{"tag": "birthdate", "enclosure": "person", "RAttribute": "dob"}}
	if hasMappingMatch(mapping1, targetBirthdate) {
		t.Error("Should NOT find birthdate mapping in mapping1")
	}
	if !hasMappingMatch(mapping2, targetBirthdate) {
		t.Error("Should find birthdate mapping in mapping2")
	}
}

// TestHarnessFileOperations tests file path handling in TestHarness.
func TestHarnessFileOperations(t *testing.T) {
	harness := NewTestHarness()
	harness.SetPath("/project/rules")

	// Test directory getters
	testDir := harness.GetTestDirectory()
	if testDir != "/project/rules/testfiles/" {
		t.Errorf("Expected test directory '/project/rules/testfiles/', got %s", testDir)
	}

	outputDir := harness.GetOutputDirectory()
	if outputDir != "/project/rules/testfiles/output/" {
		t.Errorf("Expected output directory '/project/rules/testfiles/output/', got %s", outputDir)
	}

	resultDir := harness.GetResultDirectory()
	if resultDir != "/project/rules/testfiles/output/results/" {
		t.Errorf("Expected result directory '/project/rules/testfiles/output/results/', got %s", resultDir)
	}

	// Test custom directory setters
	harness.SetTestDirectory("/custom/tests")
	if harness.GetTestDirectory() != "/custom/tests/" {
		t.Error("SetTestDirectory should set the directory with trailing slash")
	}

	harness.SetOutputDirectory("/custom/output")
	if harness.GetOutputDirectory() != "/custom/output/" {
		t.Error("SetOutputDirectory should set the directory with trailing slash")
	}

	harness.SetResultDirectory("/custom/results")
	if harness.GetResultDirectory() != "/custom/results/" {
		t.Error("SetResultDirectory should set the directory with trailing slash")
	}
}

// TestTestResultStructure tests the TestResult struct.
func TestTestResultStructure(t *testing.T) {
	result := &TestResult{
		Filename: "test_001.xml",
		Success:  true,
		Error:    "",
	}

	if result.Filename != "test_001.xml" {
		t.Errorf("Expected Filename 'test_001.xml', got %s", result.Filename)
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Error != "" {
		t.Errorf("Expected empty Error, got %s", result.Error)
	}

	// Test failure case
	failResult := &TestResult{
		Filename: "test_002.xml",
		Success:  false,
		Error:    "Assertion failed",
	}

	if failResult.Success {
		t.Error("Expected Success to be false for failed result")
	}
	if failResult.Error != "Assertion failed" {
		t.Errorf("Expected Error 'Assertion failed', got %s", failResult.Error)
	}
}

// TestCaptureOutputBuffers tests the CaptureOutput struct.
func TestCaptureOutputBuffers(t *testing.T) {
	capture := &CaptureOutput{}

	// Write to each buffer
	capture.Results.WriteString("<results><item>1</item></results>")
	capture.Trace.WriteString("<trace>execution log</trace>")
	capture.Errors.WriteString("Error: something went wrong")

	// Verify contents
	results := capture.Results.String()
	if !strings.Contains(results, "<results>") {
		t.Error("Results buffer should contain <results>")
	}

	trace := capture.Trace.String()
	if !strings.Contains(trace, "<trace>") {
		t.Error("Trace buffer should contain <trace>")
	}

	errors := capture.Errors.String()
	if !strings.Contains(errors, "Error:") {
		t.Error("Errors buffer should contain 'Error:'")
	}

	// Test buffer reset
	capture.Results.Reset()
	if capture.Results.String() != "" {
		t.Error("Results buffer should be empty after Reset")
	}
}

// TestXMLNodeAbsoluteMatchIgnoreBody tests the ignoreBody parameter.
func TestXMLNodeAbsoluteMatchIgnoreBody(t *testing.T) {
	node1 := &XMLNode{
		Name:       "item",
		Attributes: map[string]string{"id": "1"},
		Body:       "Original content",
	}

	node2 := &XMLNode{
		Name:       "item",
		Attributes: map[string]string{"id": "1"},
		Body:       "Modified content",
	}

	// Should not match when not ignoring body
	if node1.AbsoluteMatch(node2, false) {
		t.Error("Nodes with different body should not match with ignoreBody=false")
	}

	// Should match when ignoring body
	if !node1.AbsoluteMatch(node2, true) {
		t.Error("Nodes should match with ignoreBody=true")
	}

	// Different attributes should never match
	node3 := &XMLNode{
		Name:       "item",
		Attributes: map[string]string{"id": "2"},
		Body:       "Original content",
	}
	if node1.AbsoluteMatch(node3, true) {
		t.Error("Nodes with different attributes should not match even with ignoreBody=true")
	}
}

// TestCompareNodesDetailedDiff tests compareNodes with detailed differences.
func TestCompareNodesDetailedDiff(t *testing.T) {
	tests := []struct {
		name     string
		node1    *XMLNode
		node2    *XMLNode
		contains string
	}{
		{
			name:     "Different type",
			node1:    &XMLNode{Name: "person"},
			node2:    &XMLNode{Name: "job"},
			contains: "Different Type",
		},
		{
			name:     "Different body",
			node1:    &XMLNode{Name: "item", Body: "content1"},
			node2:    &XMLNode{Name: "item", Body: "content2"},
			contains: "Different Body",
		},
		{
			name: "Different children count",
			node1: &XMLNode{
				Name:     "parent",
				Children: []*XMLNode{{Name: "child1"}},
			},
			node2: &XMLNode{
				Name:     "parent",
				Children: []*XMLNode{{Name: "child1"}, {Name: "child2"}},
			},
			contains: "Different number of children",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := compareNodes(tt.node1, tt.node2)
			if !strings.Contains(msg, tt.contains) {
				t.Errorf("Expected message to contain %q, got %q", tt.contains, msg)
			}
		})
	}
}

// TestEscapeXMLSpecialCases tests edge cases in XML escaping.
func TestEscapeXMLSpecialCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"normal text", "normal text"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert('xss')&lt;/script&gt;"},
		{"a=b&c=d", "a=b&amp;c=d"},
		{"\"quoted\"", "&quot;quoted&quot;"},
		{"<>&\"", "&lt;&gt;&amp;&quot;"},
		{"multiple & signs && here", "multiple &amp; signs &amp;&amp; here"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestExtractTagEdgeCases tests edge cases in extractTag.
func TestExtractTagEdgeCases(t *testing.T) {
	tests := []struct {
		xml      string
		tag      string
		expected string
	}{
		{"<root><tag>value</tag></root>", "tag", "value"},
		{"<root><tag></tag></root>", "tag", ""},
		{"<root><other>value</other></root>", "tag", ""},
		{"<root></root>", "tag", ""},
		{"", "tag", ""},
		{"<root><tag>multi\nline</tag></root>", "tag", "multi\nline"},
		{"<root><tag>  trimmed  </tag></root>", "tag", "trimmed"}, // extractTag trims whitespace
	}

	for _, tt := range tests {
		result := extractTag(tt.xml, tt.tag)
		if result != tt.expected {
			t.Errorf("extractTag(%q, %q) = %q, want %q", tt.xml, tt.tag, result, tt.expected)
		}
	}
}
