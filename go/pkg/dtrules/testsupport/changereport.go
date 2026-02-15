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
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// RulesConfig holds the file paths and cached XML trees for a single
// version of a rule set. It reads a DTRules.xml configuration file to
// discover the EDD, decision table, and mapping file paths, then
// lazily loads and caches each XML tree on first access.
// Used by [ChangeReport] for comparison.
type RulesConfig struct {
	RuleSetName    string
	Path           string
	ConfigFile     string
	Description    string
	XMLPath        string
	DTSPath        string
	EDDPath        string
	MappingPath    string
	dtsRoot        *XMLNode
	eddRoot        *XMLNode
	mappingRoot    *XMLNode
}

// NewRulesConfig creates a new rules configuration by loading the
// DTRules.xml config file at path/configFile to discover the EDD,
// decision table, and mapping file paths.
func NewRulesConfig(ruleSetName, description, path, configFile string) (*RulesConfig, error) {
	rc := &RulesConfig{
		RuleSetName: ruleSetName,
		Description: description,
		Path:        path,
		ConfigFile:  configFile,
	}

	// Ensure path ends with separator
	if !strings.HasSuffix(rc.Path, "/") && !strings.HasSuffix(rc.Path, "\\") {
		rc.Path += "/"
	}

	// Load config file to get paths
	if err := rc.loadConfig(); err != nil {
		return nil, err
	}

	return rc, nil
}

// loadConfig loads the DTRules.xml configuration file.
func (rc *RulesConfig) loadConfig() error {
	configPath := rc.Path + rc.ConfigFile
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity

	inRuleSet := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("XML parse error: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "RuleSet" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "name" && strings.EqualFold(attr.Value, rc.RuleSetName) {
						inRuleSet = true
					}
				}
			}

		case xml.EndElement:
			if t.Name.Local == "RuleSet" {
				inRuleSet = false
			}

		case xml.CharData:
			// Body content handling would go here if needed
		}

		// Process tags within our rule set
		if inRuleSet {
			if start, ok := token.(xml.StartElement); ok {
				switch start.Name.Local {
				case "RuleSetFilePath":
					// Next char data is the path
					if body, err := rc.readBody(decoder); err == nil {
						rc.XMLPath = rc.normalizePath(body)
					}
				case "Decisiontables":
					for _, attr := range start.Attr {
						if attr.Name.Local == "name" {
							rc.DTSPath = attr.Value
						}
					}
				case "Entities", "EDD":
					for _, attr := range start.Attr {
						if attr.Name.Local == "name" {
							rc.EDDPath = attr.Value
						}
					}
				case "Map":
					for _, attr := range start.Attr {
						if attr.Name.Local == "name" {
							rc.MappingPath = attr.Value
						}
					}
				}
			}
		}
	}

	return nil
}

func (rc *RulesConfig) readBody(decoder *xml.Decoder) (string, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		if cd, ok := token.(xml.CharData); ok {
			return strings.TrimSpace(string(cd)), nil
		}
		if _, ok := token.(xml.EndElement); ok {
			return "", nil
		}
	}
}

func (rc *RulesConfig) normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if !strings.HasSuffix(p, "/") && !strings.HasSuffix(p, "\\") {
		p += "/"
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		p = p[1:]
	}
	return p
}

// GetDTSRoot returns the decision tables XML tree.
func (rc *RulesConfig) GetDTSRoot() (*XMLNode, error) {
	if rc.dtsRoot != nil {
		return rc.dtsRoot, nil
	}
	var err error
	rc.dtsRoot, err = rc.loadXML(rc.DTSPath)
	return rc.dtsRoot, err
}

// GetEDDRoot returns the EDD XML tree.
func (rc *RulesConfig) GetEDDRoot() (*XMLNode, error) {
	if rc.eddRoot != nil {
		return rc.eddRoot, nil
	}
	var err error
	rc.eddRoot, err = rc.loadXML(rc.EDDPath)
	return rc.eddRoot, err
}

// GetMappingRoot returns the mapping XML tree.
func (rc *RulesConfig) GetMappingRoot() (*XMLNode, error) {
	if rc.mappingRoot != nil {
		return rc.mappingRoot, nil
	}
	var err error
	rc.mappingRoot, err = rc.loadXML(rc.MappingPath)
	return rc.mappingRoot, err
}

func (rc *RulesConfig) loadXML(filename string) (*XMLNode, error) {
	if filename == "" {
		return nil, fmt.Errorf("no file specified")
	}

	path := rc.Path + rc.XMLPath + filename
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	return ParseXMLTree(file)
}

// XMLNode represents a node in a parsed XML tree. It is used by
// [ChangeReport] to compare two versions of EDD, decision table,
// or mapping XML files structurally.
type XMLNode struct {
	Name       string
	Attributes map[string]string
	Body       string
	Children   []*XMLNode
}

// ParseXMLTree parses an XML file into an XMLNode tree.
func ParseXMLTree(r io.Reader) (*XMLNode, error) {
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity

	var stack []*XMLNode
	var root *XMLNode

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("XML parse error: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			node := &XMLNode{
				Name:       t.Name.Local,
				Attributes: make(map[string]string),
				Children:   make([]*XMLNode, 0),
			}
			for _, attr := range t.Attr {
				node.Attributes[attr.Name.Local] = attr.Value
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}
			stack = append(stack, node)

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Body += strings.TrimSpace(string(t))
			}
		}
	}

	return root, nil
}

// FindTag finds a child tag by name.
func (n *XMLNode) FindTag(name string) *XMLNode {
	for _, child := range n.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// FindNodes finds all nodes with a given tag name in the subtree.
func (n *XMLNode) FindNodes(tagName string) []*XMLNode {
	var result []*XMLNode
	n.findNodesRecursive(tagName, &result)
	return result
}

func (n *XMLNode) findNodesRecursive(tagName string, result *[]*XMLNode) {
	if n.Name == tagName {
		*result = append(*result, n)
		return
	}
	for _, child := range n.Children {
		child.findNodesRecursive(tagName, result)
	}
}

// AbsoluteMatch compares two nodes for exact structural equality,
// recursively checking name, attributes, body, and children.
// If ignoreBody is true, the Body text of each node is not compared.
func (n *XMLNode) AbsoluteMatch(other *XMLNode, ignoreBody bool) bool {
	if n.Name != other.Name {
		return false
	}
	if len(n.Attributes) != len(other.Attributes) {
		return false
	}
	for k, v := range n.Attributes {
		if other.Attributes[k] != v {
			return false
		}
	}
	if !ignoreBody && n.Body != other.Body {
		return false
	}
	if len(n.Children) != len(other.Children) {
		return false
	}
	for i, child := range n.Children {
		if !child.AbsoluteMatch(other.Children[i], ignoreBody) {
			return false
		}
	}
	return true
}

// ChangeReport compares two versions of a rule set by examining their
// decision tables, EDD (Entity Data Dictionary), and mappings. It identifies
// new, deleted, and modified elements, and distinguishes structural changes
// (like added/removed conditions) from execution-affecting changes (like
// modified condition expressions or action columns).
type ChangeReport struct {
	Rules1 *RulesConfig // New/development version
	Rules2 *RulesConfig // Reference/deployed version

	// Tags to check for structural changes
	checkStructure []string
	// Tags to check for value changes
	checkValue []string
}

// NewChangeReport creates a new change report comparing two rule sets.
func NewChangeReport(ruleSetName, path1, config1, desc1, path2, config2, desc2 string) (*ChangeReport, error) {
	rules1, err := NewRulesConfig(ruleSetName, desc1, path1, config1)
	if err != nil {
		return nil, fmt.Errorf("failed to load first rule set: %w", err)
	}

	rules2, err := NewRulesConfig(ruleSetName, desc2, path2, config2)
	if err != nil {
		return nil, fmt.Errorf("failed to load second rule set: %w", err)
	}

	return &ChangeReport{
		Rules1: rules1,
		Rules2: rules2,
		checkStructure: []string{
			"initial_actions",
			"contexts",
			"conditions",
			"actions",
			"policy_statements",
		},
		checkValue: []string{
			"type",
			"context_postfix",
			"initial_action_postfix",
			"condition_postfix",
			"condition_column",
			"action_column",
			"policy_statement",
		},
	}, nil
}

// NewChangeReportFromRuleSets creates a change report from existing RuleSets.
func NewChangeReportFromRuleSets(rs1, rs2 *session.RuleSet, desc1, desc2 string) *ChangeReport {
	return &ChangeReport{
		Rules1: &RulesConfig{Description: desc1},
		Rules2: &RulesConfig{Description: desc2},
		checkStructure: []string{
			"initial_actions", "contexts", "conditions", "actions", "policy_statements",
		},
		checkValue: []string{
			"type", "context_postfix", "initial_action_postfix", "condition_postfix",
			"condition_column", "action_column", "policy_statement",
		},
	}
}

// Compare generates a full XML comparison report covering decision tables,
// EDD, and mappings. The report identifies new, deleted, and modified
// elements, and flags which decision table changes affect execution.
func (cr *ChangeReport) Compare(w io.Writer) error {
	fmt.Fprintln(w, "<changeReport>")
	fmt.Fprintln(w, "  <header>")
	fmt.Fprintf(w, "    <File1>%s</File1>\n", escapeXML(cr.Rules1.Description))
	fmt.Fprintf(w, "    <File2>%s</File2>\n", escapeXML(cr.Rules2.Description))
	fmt.Fprintln(w, "  </header>")

	if err := cr.CompareDecisionTables(w); err != nil {
		fmt.Fprintf(w, "  <error>%s</error>\n", escapeXML(err.Error()))
	}

	if err := cr.CompareEDD(w); err != nil {
		fmt.Fprintf(w, "  <error>%s</error>\n", escapeXML(err.Error()))
	}

	if err := cr.CompareMapping(w); err != nil {
		fmt.Fprintf(w, "  <error>%s</error>\n", escapeXML(err.Error()))
	}

	fmt.Fprintln(w, "</changeReport>")
	return nil
}

// CompareDecisionTables compares decision tables between rule sets,
// reporting new tables, deleted tables, modified tables, and tables
// whose execution behavior has changed (condition expressions, action
// columns, etc.).
func (cr *ChangeReport) CompareDecisionTables(w io.Writer) error {
	fmt.Fprintln(w, "  <decisionTables>")

	root1, err1 := cr.Rules1.GetDTSRoot()
	root2, err2 := cr.Rules2.GetDTSRoot()

	if err1 != nil {
		fmt.Fprintf(w, "    <error>decision tables for %s were not found</error>\n",
			escapeXML(cr.Rules1.Description))
	}
	if err2 != nil {
		fmt.Fprintf(w, "    <error>decision tables for %s were not found</error>\n",
			escapeXML(cr.Rules2.Description))
	}
	if err1 != nil || err2 != nil {
		fmt.Fprintln(w, "  </decisionTables>")
		return nil
	}

	dts1 := root1.FindNodes("decision_table")
	dts2 := root2.FindNodes("decision_table")

	// Find new tables
	for _, dt := range dts1 {
		name := getTableName(dt)
		if !hasTableByName(dts2, name) {
			fmt.Fprintf(w, "    <NewTable>%s</NewTable>\n", escapeXML(name))
		}
	}

	// Find deleted tables
	for _, dt := range dts2 {
		name := getTableName(dt)
		if !hasTableByName(dts1, name) {
			fmt.Fprintf(w, "    <DeletedTable>%s</DeletedTable>\n", escapeXML(name))
		}
	}

	// Find modified tables
	fmt.Fprintln(w, "    <modified_tables>")
	var modifiedTables []string
	for _, dt1 := range dts1 {
		name := getTableName(dt1)
		dt2 := findTableByName(dts2, name)
		if dt2 != nil && !dt1.AbsoluteMatch(dt2, false) {
			fmt.Fprintf(w, "      <table_name>%s</table_name>\n", escapeXML(name))
			modifiedTables = append(modifiedTables, name)
		}
	}
	fmt.Fprintln(w, "    </modified_tables>")

	// Find tables with execution changes
	executionChanges := 0
	fmt.Fprintln(w, "    <tables_with_changes_in_execution>")
	for _, name := range modifiedTables {
		dt1 := findTableByName(dts1, name)
		dt2 := findTableByName(dts2, name)
		if dt1 != nil && dt2 != nil && cr.executionChanged(dt1, dt2) {
			fmt.Fprintf(w, "      <table_name>%s</table_name>\n", escapeXML(name))
			executionChanges++
		}
	}
	fmt.Fprintln(w, "    </tables_with_changes_in_execution>")

	if executionChanges > 0 {
		fmt.Fprintf(w, "    <execution>%d Decision Table(s) have had their execution modified</execution>\n", executionChanges)
	} else {
		fmt.Fprintln(w, "    <execution>Changes have not been made that effect execution</execution>")
	}

	fmt.Fprintln(w, "  </decisionTables>")
	return nil
}

func getTableName(dt *XMLNode) string {
	if nameNode := dt.FindTag("table_name"); nameNode != nil {
		return nameNode.Body
	}
	return ""
}

func hasTableByName(tables []*XMLNode, name string) bool {
	return findTableByName(tables, name) != nil
}

func findTableByName(tables []*XMLNode, name string) *XMLNode {
	for _, dt := range tables {
		if getTableName(dt) == name {
			return dt
		}
	}
	return nil
}

func (cr *ChangeReport) executionChanged(n1, n2 *XMLNode) bool {
	if containsString(cr.checkStructure, n1.Name) {
		if len(n1.Children) != len(n2.Children) {
			return true
		}
	} else if containsString(cr.checkValue, n1.Name) {
		if !n1.AbsoluteMatch(n2, false) {
			return true
		}
	}

	maxLen := len(n1.Children)
	if len(n2.Children) > maxLen {
		maxLen = len(n2.Children)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(n1.Children) {
			if containsString(cr.checkStructure, n2.Children[i].Name) ||
				containsString(cr.checkValue, n2.Children[i].Name) {
				return true
			}
		} else if i >= len(n2.Children) {
			if containsString(cr.checkStructure, n1.Children[i].Name) ||
				containsString(cr.checkValue, n1.Children[i].Name) {
				return true
			}
		} else {
			if cr.executionChanged(n1.Children[i], n2.Children[i]) {
				return true
			}
		}
	}
	return false
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// CompareEDD compares Entity Data Dictionaries between rule sets,
// reporting new entities, deleted entities, new/deleted attributes,
// and attributes whose definitions have changed.
func (cr *ChangeReport) CompareEDD(w io.Writer) error {
	fmt.Fprintln(w, "  <edd>")

	root1, err1 := cr.Rules1.GetEDDRoot()
	root2, err2 := cr.Rules2.GetEDDRoot()

	if err1 != nil {
		fmt.Fprintf(w, "    <error>the EDD for %s was not found</error>\n",
			escapeXML(cr.Rules1.Description))
	}
	if err2 != nil {
		fmt.Fprintf(w, "    <error>the EDD for %s was not found</error>\n",
			escapeXML(cr.Rules2.Description))
	}
	if err1 != nil || err2 != nil {
		fmt.Fprintln(w, "  </edd>")
		return nil
	}

	entities1 := root1.FindNodes("entity")
	entities2 := root2.FindNodes("entity")

	// Find new entities
	for _, e := range entities1 {
		name := e.Attributes["name"]
		if !hasEntityByName(entities2, name) {
			fmt.Fprintf(w, "    <NewEntity>%s</NewEntity>\n", escapeXML(name))
		}
	}

	// Find deleted entities
	for _, e := range entities2 {
		name := e.Attributes["name"]
		if !hasEntityByName(entities1, name) {
			fmt.Fprintf(w, "    <DeletedEntity>%s</DeletedEntity>\n", escapeXML(name))
		}
	}

	// Compare matching entities
	for _, e1 := range entities1 {
		name := e1.Attributes["name"]
		e2 := findEntityByName(entities2, name)
		if e2 == nil {
			continue
		}

		fields1 := e1.FindNodes("field")
		fields2 := e2.FindNodes("field")

		// New attributes
		for _, f := range fields1 {
			attrName := f.Attributes["name"]
			if !hasFieldByName(fields2, attrName) {
				fmt.Fprintf(w, "    <NewAttribute entity=\"%s\">%s</NewAttribute>\n",
					escapeXML(name), escapeXML(attrName))
			}
		}

		// Deleted attributes
		for _, f := range fields2 {
			attrName := f.Attributes["name"]
			if !hasFieldByName(fields1, attrName) {
				fmt.Fprintf(w, "    <DeletedAttribute entity=\"%s\">%s</DeletedAttribute>\n",
					escapeXML(name), escapeXML(attrName))
			}
		}

		// Changed attributes
		for _, f1 := range fields1 {
			attrName := f1.Attributes["name"]
			f2 := findFieldByName(fields2, attrName)
			if f2 != nil && !f1.AbsoluteMatch(f2, false) {
				fmt.Fprintf(w, "    <AttributeChanged entity=\"%s\" attribute=\"%s\"/>\n",
					escapeXML(name), escapeXML(attrName))
			}
		}
	}

	fmt.Fprintln(w, "  </edd>")
	return nil
}

func hasEntityByName(entities []*XMLNode, name string) bool {
	return findEntityByName(entities, name) != nil
}

func findEntityByName(entities []*XMLNode, name string) *XMLNode {
	for _, e := range entities {
		if e.Attributes["name"] == name {
			return e
		}
	}
	return nil
}

func hasFieldByName(fields []*XMLNode, name string) bool {
	return findFieldByName(fields, name) != nil
}

func findFieldByName(fields []*XMLNode, name string) *XMLNode {
	for _, f := range fields {
		if f.Attributes["name"] == name {
			return f
		}
	}
	return nil
}

// CompareMapping compares data mappings between rule sets, reporting
// new and deleted setattribute mappings.
func (cr *ChangeReport) CompareMapping(w io.Writer) error {
	fmt.Fprintln(w, "  <mapping>")

	root1, err1 := cr.Rules1.GetMappingRoot()
	root2, err2 := cr.Rules2.GetMappingRoot()

	if err1 != nil {
		fmt.Fprintf(w, "    <error>the Mapping File for %s was not found</error>\n",
			escapeXML(cr.Rules1.Description))
	}
	if err2 != nil {
		fmt.Fprintf(w, "    <error>the Mapping File for %s was not found</error>\n",
			escapeXML(cr.Rules2.Description))
	}
	if err1 != nil || err2 != nil {
		fmt.Fprintln(w, "  </mapping>")
		return nil
	}

	// Compare setattribute mappings
	attrs1 := root1.FindNodes("setattribute")
	attrs2 := root2.FindNodes("setattribute")

	// Sort by tag/attribute for consistent comparison
	sortMappings(attrs1)
	sortMappings(attrs2)

	// Find new mappings
	for _, a := range attrs1 {
		if !hasMappingMatch(attrs2, a) {
			fmt.Fprintf(w, "    <NewMappings>%s now maps to %s.%s</NewMappings>\n",
				escapeXML(a.Attributes["tag"]),
				escapeXML(a.Attributes["enclosure"]),
				escapeXML(a.Attributes["RAttribute"]))
		}
	}

	// Find deleted mappings
	for _, a := range attrs2 {
		if !hasMappingMatch(attrs1, a) {
			fmt.Fprintf(w, "    <DeletedMappings>%s no longer maps to %s.%s</DeletedMappings>\n",
				escapeXML(a.Attributes["tag"]),
				escapeXML(a.Attributes["enclosure"]),
				escapeXML(a.Attributes["RAttribute"]))
		}
	}

	fmt.Fprintln(w, "  </mapping>")
	return nil
}

func sortMappings(mappings []*XMLNode) {
	sort.Slice(mappings, func(i, j int) bool {
		// Sort by enclosure, then RAttribute, then tag
		if mappings[i].Attributes["enclosure"] != mappings[j].Attributes["enclosure"] {
			return mappings[i].Attributes["enclosure"] < mappings[j].Attributes["enclosure"]
		}
		if mappings[i].Attributes["RAttribute"] != mappings[j].Attributes["RAttribute"] {
			return mappings[i].Attributes["RAttribute"] < mappings[j].Attributes["RAttribute"]
		}
		return mappings[i].Attributes["tag"] < mappings[j].Attributes["tag"]
	})
}

func hasMappingMatch(mappings []*XMLNode, target *XMLNode) bool {
	for _, m := range mappings {
		if m.Attributes["tag"] == target.Attributes["tag"] &&
			m.Attributes["enclosure"] == target.Attributes["enclosure"] &&
			m.Attributes["RAttribute"] == target.Attributes["RAttribute"] {
			return true
		}
	}
	return false
}
