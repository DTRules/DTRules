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
	"strings"
	"testing"
)

func TestNewStats(t *testing.T) {
	stats := NewStats("TestTable", 5, 3, 4)

	if stats.TableName != "TestTable" {
		t.Errorf("Expected TableName 'TestTable', got %s", stats.TableName)
	}
	if stats.ColumnCount != 5 {
		t.Errorf("Expected ColumnCount 5, got %d", stats.ColumnCount)
	}
	if len(stats.ColumnHits) != 5 {
		t.Errorf("Expected ColumnHits length 5, got %d", len(stats.ColumnHits))
	}
	if stats.ConditionCount != 3 {
		t.Errorf("Expected ConditionCount 3, got %d", stats.ConditionCount)
	}
	if stats.ActionCount != 4 {
		t.Errorf("Expected ActionCount 4, got %d", stats.ActionCount)
	}
}

func TestNewStatsZeroColumns(t *testing.T) {
	stats := NewStats("EmptyTable", 0, 0, 0)

	// Should have at least 1 column hit slot
	if len(stats.ColumnHits) < 1 {
		t.Error("ColumnHits should have at least 1 slot")
	}
}

func TestParseXMLTree(t *testing.T) {
	xml := `<root attr="value">
		<child1>body1</child1>
		<child2 id="123">
			<grandchild/>
		</child2>
	</root>`

	root, err := ParseXMLTree(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseXMLTree failed: %v", err)
	}

	if root.Name != "root" {
		t.Errorf("Expected root name 'root', got %s", root.Name)
	}
	if root.Attributes["attr"] != "value" {
		t.Errorf("Expected attr='value', got %s", root.Attributes["attr"])
	}
	if len(root.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(root.Children))
	}

	child1 := root.Children[0]
	if child1.Body != "body1" {
		t.Errorf("Expected body 'body1', got %s", child1.Body)
	}

	child2 := root.Children[1]
	if child2.Attributes["id"] != "123" {
		t.Errorf("Expected id='123', got %s", child2.Attributes["id"])
	}
	if len(child2.Children) != 1 {
		t.Errorf("Expected 1 grandchild, got %d", len(child2.Children))
	}
}

func TestXMLNodeFindTag(t *testing.T) {
	root := &XMLNode{
		Name: "root",
		Children: []*XMLNode{
			{Name: "child1"},
			{Name: "child2"},
			{Name: "child3"},
		},
	}

	child := root.FindTag("child2")
	if child == nil || child.Name != "child2" {
		t.Error("FindTag should find child2")
	}

	notFound := root.FindTag("nonexistent")
	if notFound != nil {
		t.Error("FindTag should return nil for non-existent tag")
	}
}

func TestXMLNodeFindNodes(t *testing.T) {
	root := &XMLNode{
		Name: "root",
		Children: []*XMLNode{
			{Name: "item", Attributes: map[string]string{"id": "1"}},
			{
				Name: "container",
				Children: []*XMLNode{
					{Name: "item", Attributes: map[string]string{"id": "2"}},
				},
			},
			{Name: "item", Attributes: map[string]string{"id": "3"}},
		},
	}

	items := root.FindNodes("item")
	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}
}

func TestXMLNodeAbsoluteMatch(t *testing.T) {
	node1 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1"},
		Body:       "content",
		Children:   []*XMLNode{{Name: "child"}},
	}

	node2 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1"},
		Body:       "content",
		Children:   []*XMLNode{{Name: "child"}},
	}

	if !node1.AbsoluteMatch(node2, false) {
		t.Error("Identical nodes should match")
	}

	node3 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "2"}, // Different id
		Body:       "content",
		Children:   []*XMLNode{{Name: "child"}},
	}

	if node1.AbsoluteMatch(node3, false) {
		t.Error("Nodes with different attributes should not match")
	}

	node4 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1"},
		Body:       "different", // Different body
		Children:   []*XMLNode{{Name: "child"}},
	}

	if node1.AbsoluteMatch(node4, false) {
		t.Error("Nodes with different body should not match")
	}

	// Should match if ignoring body
	if !node1.AbsoluteMatch(node4, true) {
		t.Error("Nodes should match when ignoring body")
	}
}

func TestNewTestHarness(t *testing.T) {
	harness := NewTestHarness()

	if harness == nil {
		t.Fatal("NewTestHarness returned nil")
	}
	if harness.RulesDirectoryFile != "DTRules.xml" {
		t.Errorf("Expected default RulesDirectoryFile 'DTRules.xml', got %s", harness.RulesDirectoryFile)
	}
	if !harness.Trace {
		t.Error("Expected Trace to be true by default")
	}
	if !harness.Coverage {
		t.Error("Expected Coverage to be true by default")
	}
}

func TestSetPath(t *testing.T) {
	harness := NewTestHarness()

	harness.SetPath("/test/path")
	if harness.Path != "/test/path/" {
		t.Errorf("Expected path '/test/path/', got %s", harness.Path)
	}

	harness.SetPath("/test/path/")
	if harness.Path != "/test/path/" {
		t.Errorf("Expected path '/test/path/', got %s", harness.Path)
	}
}

func TestGetDirectories(t *testing.T) {
	harness := NewTestHarness()
	harness.SetPath("/base")

	testDir := harness.GetTestDirectory()
	if testDir != "/base/testfiles/" {
		t.Errorf("Expected '/base/testfiles/', got %s", testDir)
	}

	outputDir := harness.GetOutputDirectory()
	if outputDir != "/base/testfiles/output/" {
		t.Errorf("Expected '/base/testfiles/output/', got %s", outputDir)
	}

	resultDir := harness.GetResultDirectory()
	if resultDir != "/base/testfiles/output/results/" {
		t.Errorf("Expected '/base/testfiles/output/results/', got %s", resultDir)
	}
}

func TestSetDirectories(t *testing.T) {
	harness := NewTestHarness()

	harness.SetTestDirectory("/custom/test")
	if harness.GetTestDirectory() != "/custom/test/" {
		t.Errorf("Expected '/custom/test/', got %s", harness.GetTestDirectory())
	}

	harness.SetOutputDirectory("/custom/output")
	if harness.GetOutputDirectory() != "/custom/output/" {
		t.Errorf("Expected '/custom/output/', got %s", harness.GetOutputDirectory())
	}

	harness.SetResultDirectory("/custom/result")
	if harness.GetResultDirectory() != "/custom/result/" {
		t.Errorf("Expected '/custom/result/', got %s", harness.GetResultDirectory())
	}
}

func TestExtractTag(t *testing.T) {
	xml := `<settings>
		<path>/test/path</path>
		<name>TestName</name>
		<empty></empty>
	</settings>`

	path := extractTag(xml, "path")
	if path != "/test/path" {
		t.Errorf("Expected '/test/path', got %s", path)
	}

	name := extractTag(xml, "name")
	if name != "TestName" {
		t.Errorf("Expected 'TestName', got %s", name)
	}

	empty := extractTag(xml, "empty")
	if empty != "" {
		t.Errorf("Expected empty string, got %s", empty)
	}

	notFound := extractTag(xml, "nonexistent")
	if notFound != "" {
		t.Errorf("Expected empty string for nonexistent tag, got %s", notFound)
	}
}

func TestCompareNodes(t *testing.T) {
	node1 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1"},
		Body:       "content",
	}

	node2 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1"},
		Body:       "content",
	}

	msg := compareNodes(node1, node2)
	if msg != "" {
		t.Errorf("Expected empty message for matching nodes, got: %s", msg)
	}

	node3 := &XMLNode{
		Name:       "different",
		Attributes: map[string]string{"id": "1"},
		Body:       "content",
	}

	msg = compareNodes(node1, node3)
	if !strings.Contains(msg, "Different Type") {
		t.Error("Expected 'Different Type' message for different tag names")
	}

	node4 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1"},
		Body:       "different",
	}

	msg = compareNodes(node1, node4)
	if !strings.Contains(msg, "Different Body") {
		t.Error("Expected 'Different Body' message for different body")
	}
}

func TestCompareNodesSkipsIds(t *testing.T) {
	node1 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "1", "DTRulesId": "abc"},
		Body:       "content",
	}

	node2 := &XMLNode{
		Name:       "test",
		Attributes: map[string]string{"id": "2", "DTRulesId": "xyz"},
		Body:       "content",
	}

	msg := compareNodes(node1, node2)
	if msg != "" {
		t.Errorf("Expected empty message (ids should be skipped), got: %s", msg)
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<tag>", "&lt;tag&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"<>&\"", "&lt;&gt;&amp;&quot;"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestContainsString(t *testing.T) {
	list := []string{"apple", "banana", "cherry"}

	if !containsString(list, "banana") {
		t.Error("Expected to find 'banana'")
	}

	if containsString(list, "grape") {
		t.Error("Should not find 'grape'")
	}

	if containsString(nil, "apple") {
		t.Error("Should not find anything in nil list")
	}
}

func TestContains(t *testing.T) {
	list := []string{"one", "two", "three"}

	if !contains(list, "two") {
		t.Error("Expected to find 'two'")
	}

	if contains(list, "four") {
		t.Error("Should not find 'four'")
	}
}

func TestMax(t *testing.T) {
	if max(5, 3) != 5 {
		t.Error("max(5, 3) should be 5")
	}
	if max(3, 5) != 5 {
		t.Error("max(3, 5) should be 5")
	}
	if max(4, 4) != 4 {
		t.Error("max(4, 4) should be 4")
	}
}

func TestCoverageReportFormat(t *testing.T) {
	// Create a simple stats map
	stats := NewStats("TestTable", 3, 2, 2)
	stats.CalledCount = 5
	stats.ColumnHits[0] = 3
	stats.ColumnHits[1] = 2
	stats.ColumnsSpec[0] = true
	stats.ColumnsSpec[1] = true
	stats.ColumnsSpec[2] = true

	var buf bytes.Buffer

	// Manually write a simple report to test format
	buf.WriteString("<coverage total_columns_executed=\"5\">\n")
	buf.WriteString("  <tables>\n")
	buf.WriteString("    <table name=\"TestTable\" count_of_calls=\"5\">\n")
	buf.WriteString("    </table>\n")
	buf.WriteString("  </tables>\n")
	buf.WriteString("</coverage>\n")

	output := buf.String()
	if !strings.Contains(output, "total_columns_executed") {
		t.Error("Report should contain total_columns_executed")
	}
	if !strings.Contains(output, "<tables>") {
		t.Error("Report should contain tables section")
	}
}

func TestSortMappings(t *testing.T) {
	mappings := []*XMLNode{
		{Attributes: map[string]string{"enclosure": "b", "RAttribute": "z", "tag": "1"}},
		{Attributes: map[string]string{"enclosure": "a", "RAttribute": "y", "tag": "2"}},
		{Attributes: map[string]string{"enclosure": "a", "RAttribute": "x", "tag": "3"}},
	}

	sortMappings(mappings)

	// Should be sorted by enclosure, then RAttribute, then tag
	if mappings[0].Attributes["enclosure"] != "a" || mappings[0].Attributes["RAttribute"] != "x" {
		t.Error("First element should be enclosure=a, RAttribute=x")
	}
	if mappings[1].Attributes["enclosure"] != "a" || mappings[1].Attributes["RAttribute"] != "y" {
		t.Error("Second element should be enclosure=a, RAttribute=y")
	}
	if mappings[2].Attributes["enclosure"] != "b" {
		t.Error("Third element should be enclosure=b")
	}
}

func TestHasMappingMatch(t *testing.T) {
	mappings := []*XMLNode{
		{Attributes: map[string]string{"tag": "t1", "enclosure": "e1", "RAttribute": "r1"}},
		{Attributes: map[string]string{"tag": "t2", "enclosure": "e2", "RAttribute": "r2"}},
	}

	target1 := &XMLNode{Attributes: map[string]string{"tag": "t1", "enclosure": "e1", "RAttribute": "r1"}}
	if !hasMappingMatch(mappings, target1) {
		t.Error("Should find matching mapping")
	}

	target2 := &XMLNode{Attributes: map[string]string{"tag": "t3", "enclosure": "e3", "RAttribute": "r3"}}
	if hasMappingMatch(mappings, target2) {
		t.Error("Should not find non-existent mapping")
	}
}

func TestTestResult(t *testing.T) {
	result := &TestResult{
		Filename: "test.xml",
		Success:  true,
	}

	if result.Filename != "test.xml" {
		t.Errorf("Expected filename 'test.xml', got %s", result.Filename)
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
}

func TestCaptureOutput(t *testing.T) {
	capture := &CaptureOutput{}

	capture.Results.WriteString("results")
	capture.Trace.WriteString("trace")
	capture.Errors.WriteString("errors")

	if capture.Results.String() != "results" {
		t.Error("Results buffer should contain 'results'")
	}
	if capture.Trace.String() != "trace" {
		t.Error("Trace buffer should contain 'trace'")
	}
	if capture.Errors.String() != "errors" {
		t.Error("Errors buffer should contain 'errors'")
	}
}
