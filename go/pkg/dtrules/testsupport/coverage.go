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

// Package testsupport provides test infrastructure for DTRules rule sets.
//
// It contains three main components:
//
//   - [TestHarness]: Executes XML test files against a loaded rule set,
//     generates trace and result files, and compares output against
//     reference results to detect regressions.
//
//   - [Coverage]: Analyzes trace files produced during test execution to
//     compute decision table column coverage. It reports which columns
//     were exercised and identifies the minimum set of test files needed
//     to achieve coverage.
//
//   - [ChangeReport]: Compares two versions of a rule set (EDD, decision
//     tables, and mappings) to identify structural and execution-affecting
//     changes between them.
package testsupport

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// Stats holds coverage statistics for a single decision table.
// Each field tracks a different dimension of coverage: how many times
// each column was hit, how often the table was called, and whether
// a "no match" (null) column exists.
type Stats struct {
	TableName      string // Name of the decision table
	ColumnCount    int    // Total columns defined in the table
	ColumnHits     []int  // Hit count per column (0-indexed)
	ConditionCount int    // Number of condition rows
	Conditions     []int  // Hit count per condition row
	ActionCount    int    // Number of action rows
	Actions        []int  // Hit count per action row
	NoColumns      int    // Times the table was called but no column matched
	CalledCount    int    // Total invocations of the table across all traces
	HasNullColumn  bool   // True if the table has an all-dash (wildcard) column
	ColumnsSpec    []bool // True for each column index that is specified in the table
}

// NewStats creates a new Stats instance for a decision table.
func NewStats(tableName string, columnCount, conditionCount, actionCount int) *Stats {
	return &Stats{
		TableName:      tableName,
		ColumnCount:    columnCount,
		ColumnHits:     make([]int, max(columnCount, 1)),
		ConditionCount: conditionCount,
		Conditions:     make([]int, conditionCount),
		ActionCount:    actionCount,
		Actions:        make([]int, actionCount),
		ColumnsSpec:    make([]bool, max(columnCount, 1)),
	}
}

// Coverage computes decision table column coverage from trace files.
//
// Create a Coverage with [NewCoverage], call [Coverage.Compute] to process
// the trace files, then use [Coverage.PrintReport] to output an XML report
// or [Coverage.GetStats] to inspect per-table statistics.
type Coverage struct {
	rs                  *session.RuleSet
	traceFilesPath      string
	tables              map[string]*Stats
	traceFilesProcessed []string
	minFilesNeeded      []string
	totalColumns        int
	currentFile         string
}

// NewCoverage creates a new Coverage analyzer for the given rule set.
func NewCoverage(rs *session.RuleSet, traceFilesPath string) (*Coverage, error) {
	c := &Coverage{
		rs:                  rs,
		traceFilesPath:      traceFilesPath,
		tables:              make(map[string]*Stats),
		traceFilesProcessed: make([]string, 0),
		minFilesNeeded:      make([]string, 0),
	}

	if err := c.initTables(); err != nil {
		return nil, err
	}
	return c, nil
}

// initTables initializes Stats for all decision tables in the rule set.
func (c *Coverage) initTables() error {
	tableNames := c.rs.GetDecisionTableNames()
	ef := c.rs.GetEntityFactory()

	for _, name := range tableNames {
		dt, err := ef.GetDecisionTable(name)
		if err != nil {
			continue
		}
		if dt == nil {
			continue
		}

		// Try to get dimensions through type assertion to the concrete type
		var columns, conditions, actions int
		var hasNull bool
		var columnsSpec []bool

		// Type assert to get the RDecisionTable implementation
		if rdt, ok := dt.(interface {
			GetMaxCol() int
			GetConditions() []string
			GetActions() []string
			GetConditionTable() [][]string
		}); ok {
			columns = rdt.GetMaxCol()
			conditions = len(rdt.GetConditions())
			actions = len(rdt.GetActions())

			// Determine hasNullColumn from condition table - if there's a column
			// with all dashes in conditions, it's a null column
			condTable := rdt.GetConditionTable()
			if len(condTable) > 0 {
				for col := 0; col < columns; col++ {
					allDash := true
					for row := 0; row < len(condTable); row++ {
						if col < len(condTable[row]) && condTable[row][col] != "-" {
							allDash = false
							break
						}
					}
					if allDash {
						hasNull = true
						break
					}
				}
			}

			// Mark all columns as specified for now
			columnsSpec = make([]bool, columns)
			for i := range columnsSpec {
				columnsSpec[i] = true
			}
		}

		stats := NewStats(name.StringValue(), columns, conditions, actions)
		stats.HasNullColumn = hasNull

		// Copy columns specified
		if columnsSpec != nil {
			for i := 0; i < len(columnsSpec) && i < len(stats.ColumnsSpec); i++ {
				stats.ColumnsSpec[i] = columnsSpec[i]
			}
		}

		c.tables[name.StringValue()] = stats
	}
	return nil
}

// Compute analyzes trace files and computes coverage statistics.
// If traceFilesPath points to a file, that single file is analyzed.
// If it points to a directory, all *_trace.xml files in it are processed.
func (c *Coverage) Compute() error {
	fi, err := os.Stat(c.traceFilesPath)
	if err != nil {
		return fmt.Errorf("cannot access trace files path: %w", err)
	}

	if !fi.IsDir() {
		// Single file
		return c.processTraceFile(c.traceFilesPath)
	}

	// Directory of files
	entries, err := os.ReadDir(c.traceFilesPath)
	if err != nil {
		return fmt.Errorf("cannot read trace directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_trace.xml") {
			tracePath := filepath.Join(c.traceFilesPath, entry.Name())
			c.traceFilesProcessed = append(c.traceFilesProcessed, entry.Name())
			if err := c.processTraceFile(tracePath); err != nil {
				// Log error but continue processing
				fmt.Printf("Warning: error processing %s: %v\n", entry.Name(), err)
			}
		}
	}
	return nil
}

// processTraceFile processes a single trace file for coverage.
func (c *Coverage) processTraceFile(filepath string) error {
	c.currentFile = filepath

	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return c.parseTraceXML(file)
}

// traceParser handles XML parsing for coverage analysis.
type traceParser struct {
	coverage *Coverage
	dtStack  []string // Stack of decision table names being processed
}

// parseTraceXML parses a trace XML file for coverage data.
func (c *Coverage) parseTraceXML(r io.Reader) error {
	parser := &traceParser{
		coverage: c,
		dtStack:  make([]string, 0),
	}

	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

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
			parser.beginTag(t.Name.Local, getAttrs(t.Attr))

		case xml.EndElement:
			parser.endTag(t.Name.Local)
		}
	}
	return nil
}

func getAttrs(attrs []xml.Attr) map[string]string {
	result := make(map[string]string)
	for _, attr := range attrs {
		result[attr.Name.Local] = attr.Value
	}
	return result
}

func (p *traceParser) pushDT(dt string) {
	p.dtStack = append(p.dtStack, dt)
}

func (p *traceParser) popDT() string {
	if len(p.dtStack) == 0 {
		return ""
	}
	dt := p.dtStack[len(p.dtStack)-1]
	p.dtStack = p.dtStack[:len(p.dtStack)-1]
	return dt
}

func (p *traceParser) currentDT() string {
	if len(p.dtStack) == 0 {
		return ""
	}
	return p.dtStack[len(p.dtStack)-1]
}

func (p *traceParser) beginTag(tag string, attrs map[string]string) {
	if tag == "decisiontable" {
		name := attrs["name"]
		p.pushDT(name)
		if stats, ok := p.coverage.tables[name]; ok {
			stats.CalledCount++
		}
	} else if tag == "column" {
		p.addColumns(attrs["n"])
	}
}

func (p *traceParser) endTag(tag string) {
	if tag == "decisiontable" {
		p.popDT()
	}
}

func (p *traceParser) addColumns(columns string) {
	columns = strings.TrimSpace(columns)

	currentDT := p.currentDT()
	stats, ok := p.coverage.tables[currentDT]
	if !ok {
		return
	}

	if columns == "" {
		// No column executed
		if stats.NoColumns == 0 {
			if !contains(p.coverage.minFilesNeeded, p.coverage.currentFile) {
				p.coverage.minFilesNeeded = append(p.coverage.minFilesNeeded, p.coverage.currentFile)
			}
		}
		stats.NoColumns++
		return
	}

	cols := strings.Fields(columns)
	for _, col := range cols {
		index, err := strconv.Atoi(col)
		if err != nil {
			continue
		}
		index-- // Convert to 0-based index
		if index >= 0 && index < len(stats.ColumnHits) {
			if stats.ColumnHits[index] == 0 {
				if !contains(p.coverage.minFilesNeeded, p.coverage.currentFile) {
					p.coverage.minFilesNeeded = append(p.coverage.minFilesNeeded, p.coverage.currentFile)
				}
			}
			stats.ColumnHits[index]++
			p.coverage.totalColumns++
		}
	}
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

// PrintReport outputs the coverage report as XML.
func (c *Coverage) PrintReport(w io.Writer) {
	fmt.Fprintf(w, "<coverage total_columns_executed=\"%d\">\n", c.totalColumns)

	// Minimum files for coverage
	fmt.Fprintln(w, "  <minimum_files_for_coverage>")
	for _, file := range c.minFilesNeeded {
		fmt.Fprintf(w, "    <trace_file>%s</trace_file>\n", escapeXML(file))
	}
	fmt.Fprintln(w, "  </minimum_files_for_coverage>")

	// All processed trace files
	fmt.Fprintln(w, "  <trace_files>")
	for _, file := range c.traceFilesProcessed {
		fmt.Fprintf(w, "    <trace_file>%s</trace_file>\n", escapeXML(file))
	}
	fmt.Fprintln(w, "  </trace_files>")

	// Sort table names
	tableNames := make([]string, 0, len(c.tables))
	for name := range c.tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	// Tables
	fmt.Fprintln(w, "  <tables>")
	for _, tableName := range tableNames {
		stats := c.tables[tableName]

		totalColumns := 0
		columnsCovered := 0

		if stats.HasNullColumn {
			totalColumns++
		}
		if stats.NoColumns > 0 {
			columnsCovered++
		}

		for i := 0; i < stats.ColumnCount; i++ {
			if i < len(stats.ColumnsSpec) && stats.ColumnsSpec[i] {
				totalColumns++
				if i < len(stats.ColumnHits) && stats.ColumnHits[i] > 0 {
					columnsCovered++
				}
			}
		}

		var percent float64
		if totalColumns > 0 {
			percent = float64(columnsCovered) / float64(totalColumns) * 100.0
		}

		fmt.Fprintf(w, "    <table name=\"%s\" count_of_calls=\"%d\" percent_covered=\"%.2f\">\n",
			escapeXML(tableName), stats.CalledCount, percent)

		if stats.HasNullColumn {
			fmt.Fprintf(w, "      <column n=\"none\" hits=\"%d\"/>\n", stats.NoColumns)
		}

		for i := 0; i < stats.ColumnCount; i++ {
			if i < len(stats.ColumnsSpec) && stats.ColumnsSpec[i] {
				hits := 0
				if i < len(stats.ColumnHits) {
					hits = stats.ColumnHits[i]
				}
				fmt.Fprintf(w, "      <column n=\"%d\" hits=\"%d\"/>\n", i+1, hits)
			}
		}

		fmt.Fprintln(w, "    </table>")
	}
	fmt.Fprintln(w, "  </tables>")

	fmt.Fprintln(w, "</coverage>")
}

// GetStats returns the coverage stats for a specific table.
func (c *Coverage) GetStats(tableName string) *Stats {
	return c.tables[tableName]
}

// GetAllStats returns all table statistics.
func (c *Coverage) GetAllStats() map[string]*Stats {
	return c.tables
}

// GetTotalColumns returns the total number of column executions.
func (c *Coverage) GetTotalColumns() int {
	return c.totalColumns
}

// GetMinFilesNeeded returns the minimum files needed for coverage.
func (c *Coverage) GetMinFilesNeeded() []string {
	return c.minFilesNeeded
}

// GetTraceFilesProcessed returns the list of processed trace files.
func (c *Coverage) GetTraceFilesProcessed() []string {
	return c.traceFilesProcessed
}

// escapeXML escapes special characters for XML output.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

