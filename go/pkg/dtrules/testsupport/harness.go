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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/interpreter"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// TestHarness provides test execution and comparison capabilities.
type TestHarness struct {
	Path               string
	RuleSetName        string
	DecisionTableName  string
	RulesDirectoryFile string
	TestDirectory      string
	OutputDirectory    string
	ResultDirectory    string
	Trace              bool
	Coverage           bool
	Verbose            bool
	Console            bool
	Numbered           bool
	fileCount          int
	currentFile        string
	ruleSet            *session.RuleSet
}

// NewTestHarness creates a new test harness with default settings.
func NewTestHarness() *TestHarness {
	return &TestHarness{
		RulesDirectoryFile: "DTRules.xml",
		Trace:              true,
		Coverage:           true,
	}
}

// SetPath sets the base path for the test harness.
func (th *TestHarness) SetPath(path string) {
	th.Path = path
	if !strings.HasSuffix(th.Path, "/") {
		th.Path += "/"
	}
}

// SetTestDirectory sets the test input directory.
func (th *TestHarness) SetTestDirectory(dir string) {
	th.TestDirectory = dir
	if !strings.HasSuffix(th.TestDirectory, "/") {
		th.TestDirectory += "/"
	}
}

// SetOutputDirectory sets the test output directory.
func (th *TestHarness) SetOutputDirectory(dir string) {
	th.OutputDirectory = dir
	if !strings.HasSuffix(th.OutputDirectory, "/") {
		th.OutputDirectory += "/"
	}
}

// SetResultDirectory sets the reference results directory.
func (th *TestHarness) SetResultDirectory(dir string) {
	th.ResultDirectory = dir
	if !strings.HasSuffix(th.ResultDirectory, "/") {
		th.ResultDirectory += "/"
	}
}

// GetTestDirectory returns the test directory, using defaults if not set.
func (th *TestHarness) GetTestDirectory() string {
	if th.TestDirectory != "" {
		return th.TestDirectory
	}
	return th.Path + "testfiles/"
}

// GetOutputDirectory returns the output directory, using defaults if not set.
func (th *TestHarness) GetOutputDirectory() string {
	if th.OutputDirectory != "" {
		return th.OutputDirectory
	}
	return th.GetTestDirectory() + "output/"
}

// GetResultDirectory returns the reference results directory.
func (th *TestHarness) GetResultDirectory() string {
	if th.ResultDirectory != "" {
		return th.ResultDirectory
	}
	return th.GetOutputDirectory() + "results/"
}

// LoadRuleSet loads the rule set from the configured paths.
func (th *TestHarness) LoadRuleSet(eddPath, dtPath string) error {
	th.ruleSet = session.NewRuleSet(th.RuleSetName)
	if th.ruleSet == nil {
		return fmt.Errorf("failed to create rule set: %s", th.RuleSetName)
	}

	// Load EDD
	eddFile, err := os.Open(eddPath)
	if err != nil {
		return fmt.Errorf("failed to open EDD: %w", err)
	}
	defer eddFile.Close()

	if err := th.ruleSet.LoadEDD(eddFile); err != nil {
		return fmt.Errorf("failed to load EDD: %w", err)
	}

	// Load Decision Tables
	dtFile, err := os.Open(dtPath)
	if err != nil {
		return fmt.Errorf("failed to open DT: %w", err)
	}
	defer dtFile.Close()

	if err := th.ruleSet.LoadDecisionTables(dtFile); err != nil {
		return fmt.Errorf("failed to load DT: %w", err)
	}

	return nil
}

// GetRuleSet returns the loaded rule set.
func (th *TestHarness) GetRuleSet() *session.RuleSet {
	return th.ruleSet
}

// GetFiles returns the XML test files in the test directory, sorted.
func (th *TestHarness) GetFiles() ([]os.FileInfo, error) {
	entries, err := os.ReadDir(th.GetTestDirectory())
	if err != nil {
		return nil, fmt.Errorf("cannot read test directory: %w", err)
	}

	var files []os.FileInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
			info, err := entry.Info()
			if err == nil {
				files = append(files, info)
			}
		}
	}

	// Sort by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	return files, nil
}

// RunTests executes all tests and generates results.
func (th *TestHarness) RunTests() error {
	if th.ruleSet == nil {
		return fmt.Errorf("rule set not loaded")
	}

	// Ensure output directory exists
	outputDir := th.GetOutputDirectory()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Clean output directory
	th.cleanOutputDirectory(outputDir)

	// Get test files
	files, err := th.GetFiles()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no test files found in %s", th.GetTestDirectory())
	}

	// Run each test
	start := time.Now()
	th.fileCount = 0

	for i, file := range files {
		th.currentFile = file.Name()
		err := th.RunFile(i+1, th.GetTestDirectory(), file.Name())
		if err != nil {
			fmt.Printf("Error running %s: %v\n", file.Name(), err)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("\nTotal execution time: %v\n", elapsed)
	if len(files) > 0 {
		fmt.Printf("Average per test: %v\n", elapsed/time.Duration(len(files)))
	}

	// Generate coverage if enabled
	if th.Trace && th.Coverage {
		if err := th.generateCoverage(); err != nil {
			fmt.Printf("Warning: failed to generate coverage: %v\n", err)
		}
	}

	// Compare results
	if err := th.CompareTestResults(); err != nil {
		fmt.Printf("Warning: failed to compare results: %v\n", err)
	}

	return nil
}

func (th *TestHarness) cleanOutputDirectory(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// RunFile executes a single test file.
func (th *TestHarness) RunFile(num int, path, filename string) error {
	th.currentFile = filename
	th.fileCount++

	// Get root name (without extension)
	root := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Build numbered prefix if needed
	prefix := ""
	if th.Numbered {
		prefix = fmt.Sprintf("%04d_", th.fileCount)
	}

	// Create session
	sess, err := th.ruleSet.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	rsess := sess.(*session.RSession)
	state := rsess.GetState().(*interpreter.DTState)

	// Set up output files
	outputDir := th.GetOutputDirectory()
	resultsPath := filepath.Join(outputDir, prefix+root+"_results.xml")

	resultsFile, err := os.Create(resultsPath)
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()

	var traceFile *os.File
	if th.Trace {
		tracePath := filepath.Join(outputDir, prefix+root+"_trace.xml")
		traceFile, err = os.Create(tracePath)
		if err != nil {
			return fmt.Errorf("failed to create trace file: %w", err)
		}
		defer traceFile.Close()
		state.SetOutput(traceFile, resultsFile)
		state.EnableTrace()
		if th.Verbose {
			state.EnableVerbose()
		}
		// Write trace header
		fmt.Fprintln(traceFile, "<DTRulesTrace>")
	}

	// Load test data
	testFilePath := filepath.Join(path, filename)
	testData, err := os.ReadFile(testFilePath)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}

	// Execute decision table
	if th.DecisionTableName != "" {
		if err := rsess.Execute(th.DecisionTableName); err != nil {
			th.writeError(resultsFile, err)
			if th.Trace && traceFile != nil {
				fmt.Fprintln(traceFile, "</DTRulesTrace>")
			}
			return fmt.Errorf("execution error: %w", err)
		}
	}

	// Write results
	th.printReport(rsess, resultsFile, testData)

	if th.Trace && traceFile != nil {
		fmt.Fprintln(traceFile, "</DTRulesTrace>")
	}

	if th.Console {
		fmt.Printf("Completed: %s\n", filename)
	}

	return nil
}

func (th *TestHarness) writeError(w io.Writer, err error) {
	fmt.Fprintf(w, "<error>%s</error>\n", escapeXML(err.Error()))
}

func (th *TestHarness) printReport(sess *session.RSession, w io.Writer, testData []byte) {
	// Default: print entity stack as XML
	state := sess.GetState()
	fmt.Fprintln(w, "<results>")
	fmt.Fprintf(w, "  <test_data>%s</test_data>\n", escapeXML(string(testData)))

	// Print entity stack info
	fmt.Fprintln(w, "  <entity_stack>")
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		entity, err := state.EntityFetch(depth - 1 - i) // Fetch from top
		if err == nil && entity != nil {
			fmt.Fprintf(w, "    <entity name=\"%s\" id=\"%d\"/>\n",
				escapeXML(entity.GetName().StringValue()), entity.GetID())
		}
	}
	fmt.Fprintln(w, "  </entity_stack>")

	fmt.Fprintln(w, "</results>")
}

func (th *TestHarness) generateCoverage() error {
	cov, err := NewCoverage(th.ruleSet, th.GetOutputDirectory())
	if err != nil {
		return err
	}

	if err := cov.Compute(); err != nil {
		return err
	}

	coveragePath := filepath.Join(th.GetOutputDirectory(), "coverage.xml")
	coverageFile, err := os.Create(coveragePath)
	if err != nil {
		return err
	}
	defer coverageFile.Close()

	cov.PrintReport(coverageFile)
	return nil
}

// CompareTestResults compares output results with reference results.
func (th *TestHarness) CompareTestResults() error {
	outputDir := th.GetOutputDirectory()
	resultDir := th.GetResultDirectory()

	// Create TestResults.xml
	reportPath := filepath.Join(outputDir, "TestResults.xml")
	reportFile, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create test results report: %w", err)
	}
	defer reportFile.Close()

	fmt.Fprintln(reportFile, "<results>")

	// Find all results files
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		fmt.Fprintln(reportFile, "  <error>cannot read output directory</error>")
		fmt.Fprintln(reportFile, "</results>")
		return err
	}

	changes := false
	missing := false

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_results.xml") {
			result1Path := filepath.Join(outputDir, entry.Name())
			result2Path := filepath.Join(resultDir, entry.Name())

			result1, err1 := loadResultFile(result1Path)
			result2, err2 := loadResultFile(result2Path)

			if err1 != nil || err2 != nil {
				missing = true
				fmt.Fprintf(reportFile, "  <unknown file=\"%s\">Missing Files to do the compare</unknown>\n",
					escapeXML(entry.Name()))
				continue
			}

			msg := compareNodes(result1, result2)
			if msg == "" {
				fmt.Fprintf(reportFile, "  <match file=\"%s\"/>\n", escapeXML(entry.Name()))
			} else {
				changes = true
				fmt.Fprintf(reportFile, "  <resultChanged file=\"%s\">%s</resultChanged>\n",
					escapeXML(entry.Name()), escapeXML(msg))
			}
		}
	}

	fmt.Fprintln(reportFile, "</results>")

	if changes {
		fmt.Println("\nSome results have changed. Check TestResults.xml for details.")
	} else if !missing {
		fmt.Println("\nALL PASS: No changes found when compared to results files.")
	}
	if missing {
		fmt.Println("\nSome tests have no results with which to compare. Check TestResults.xml for details.")
	}

	return nil
}

func loadResultFile(path string) (*XMLNode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ParseXMLTree(file)
}

func compareNodes(n1, n2 *XMLNode) string {
	if n1 == nil || n2 == nil {
		return "one or both nodes are nil"
	}

	if n1.Name != n2.Name {
		return fmt.Sprintf("Different Type found on a tag: new:{%s} old:{%s}", n1.Name, n2.Name)
	}

	// Compare attributes
	if len(n1.Attributes) != len(n2.Attributes) {
		return fmt.Sprintf("Different Attributes on tag: %s", n1.Name)
	}
	for k, v := range n1.Attributes {
		// Skip DTRulesId and id attributes for comparison
		if k == "DTRulesId" || k == "id" {
			continue
		}
		if n2.Attributes[k] != v {
			return fmt.Sprintf("Different Attributes on tag: %s", n1.Name)
		}
	}

	// Compare body (skip mapping_key body)
	if n1.Name != "mapping_key" && n1.Body != n2.Body {
		return fmt.Sprintf("Different Body on tag %s: new:{%s} old:{%s}", n1.Name, n1.Body, n2.Body)
	}

	// Compare children
	if len(n1.Children) != len(n2.Children) {
		return fmt.Sprintf("Different number of children on tag: %s", n1.Name)
	}

	for i := range n1.Children {
		if msg := compareNodes(n1.Children[i], n2.Children[i]); msg != "" {
			return msg
		}
	}

	return ""
}

// TestResult represents the result of running a test.
type TestResult struct {
	Filename string
	Success  bool
	Error    string
	Duration time.Duration
}

// RunSingleTest runs a single test file and returns the result.
func (th *TestHarness) RunSingleTest(testFile string) (*TestResult, error) {
	start := time.Now()

	result := &TestResult{
		Filename: testFile,
		Success:  true,
	}

	err := th.RunFile(1, th.GetTestDirectory(), testFile)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	}

	result.Duration = time.Since(start)
	return result, nil
}

// EntityReport generates a report of all entities in the session.
func EntityReport(sess dtrules.Session, w io.Writer) error {
	state := sess.GetState()

	fmt.Fprintln(w, "<entityReport>")
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		entity, err := state.EntityFetch(depth - 1 - i) // Fetch from top
		if err != nil || entity == nil {
			continue
		}

		fmt.Fprintf(w, "  <entity name=\"%s\" id=\"%d\">\n",
			escapeXML(entity.GetName().StringValue()), entity.GetID())

		// Print attributes
		attrNames := entity.GetAttributeNames()
		for _, attrName := range attrNames {
			value, _ := entity.Get(attrName)
			var valueStr string
			if value != nil {
				valueStr = value.StringValue()
			}
			fmt.Fprintf(w, "    <attribute name=\"%s\">%s</attribute>\n",
				escapeXML(attrName.StringValue()), escapeXML(valueStr))
		}

		fmt.Fprintln(w, "  </entity>")
	}
	fmt.Fprintln(w, "</entityReport>")

	return nil
}

// LoadSettings loads test harness settings from an XML configuration file.
func (th *TestHarness) LoadSettings(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open settings file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	// Simple XML parsing for settings
	s := string(content)

	// Extract values from XML tags
	if v := extractTag(s, "path"); v != "" {
		th.SetPath(v)
	}
	if v := extractTag(s, "ruleSetName"); v != "" {
		th.RuleSetName = v
	}
	if v := extractTag(s, "decisionTableName"); v != "" {
		th.DecisionTableName = v
	}
	if v := extractTag(s, "testDirectory"); v != "" {
		th.SetTestDirectory(v)
	}
	if v := extractTag(s, "outputDirectory"); v != "" {
		th.SetOutputDirectory(v)
	}
	if v := extractTag(s, "trace"); v != "" {
		th.Trace = strings.EqualFold(v, "true")
	}
	if v := extractTag(s, "verbose"); v != "" {
		th.Verbose = strings.EqualFold(v, "true")
	}
	if v := extractTag(s, "coverage"); v != "" {
		th.Coverage = strings.EqualFold(v, "true")
	}
	if v := extractTag(s, "numbered"); v != "" {
		th.Numbered = strings.EqualFold(v, "true")
	}

	return nil
}

func extractTag(s, tagName string) string {
	start := strings.Index(s, "<"+tagName+">")
	if start < 0 {
		return ""
	}
	start += len("<" + tagName + ">")
	end := strings.Index(s[start:], "</"+tagName+">")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(s[start : start+end])
}

// GenerateTestDataReport generates a report of test data coverage.
func (th *TestHarness) GenerateTestDataReport(w io.Writer) error {
	files, err := th.GetFiles()
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "<testDataReport>")
	fmt.Fprintf(w, "  <testCount>%d</testCount>\n", len(files))

	fmt.Fprintln(w, "  <files>")
	for _, file := range files {
		fmt.Fprintf(w, "    <file name=\"%s\" size=\"%d\"/>\n",
			escapeXML(file.Name()), file.Size())
	}
	fmt.Fprintln(w, "  </files>")

	fmt.Fprintln(w, "</testDataReport>")
	return nil
}

// CaptureOutput captures the output of a test run.
type CaptureOutput struct {
	Results bytes.Buffer
	Trace   bytes.Buffer
	Errors  bytes.Buffer
}

// RunFileWithCapture runs a test file and captures all output.
func (th *TestHarness) RunFileWithCapture(path, filename string) (*CaptureOutput, error) {
	capture := &CaptureOutput{}

	if th.ruleSet == nil {
		return nil, fmt.Errorf("rule set not loaded")
	}

	// Create session
	sess, err := th.ruleSet.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	rsess := sess.(*session.RSession)
	state := rsess.GetState().(*interpreter.DTState)

	if th.Trace {
		state.SetOutput(&capture.Trace, &capture.Results)
		state.EnableTrace()
		if th.Verbose {
			state.EnableVerbose()
		}
		// Write trace header
		fmt.Fprintln(&capture.Trace, "<DTRulesTrace>")
	}

	// Execute
	if th.DecisionTableName != "" {
		if err := rsess.Execute(th.DecisionTableName); err != nil {
			fmt.Fprintf(&capture.Errors, "Execution error: %v\n", err)
		}
	}

	// Read test file for report
	testFilePath := filepath.Join(path, filename)
	testData, _ := os.ReadFile(testFilePath)

	th.printReport(rsess, &capture.Results, testData)

	if th.Trace {
		fmt.Fprintln(&capture.Trace, "</DTRulesTrace>")
	}

	return capture, nil
}
