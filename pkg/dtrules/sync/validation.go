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

package sync

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ELComplianceStatus indicates the EL compliance status of a file.
type ELComplianceStatus int

const (
	// StatusUnknown indicates the file format could not be determined.
	StatusUnknown ELComplianceStatus = iota
	// StatusCompliant indicates the file has EL descriptions with compiled postfix.
	StatusCompliant
	// StatusLegacy indicates the file has hand-coded postfix without EL descriptions.
	StatusLegacy
	// StatusNeedsCompilation indicates the file has EL descriptions but no compiled postfix.
	StatusNeedsCompilation
	// StatusEmpty indicates the file has no conditions or actions to validate.
	StatusEmpty
)

func (s ELComplianceStatus) String() string {
	switch s {
	case StatusCompliant:
		return "compliant"
	case StatusLegacy:
		return "legacy"
	case StatusNeedsCompilation:
		return "needs_compilation"
	case StatusEmpty:
		return "empty"
	default:
		return "unknown"
	}
}

// ValidationError represents an EL compliance validation error.
type ValidationError struct {
	FilePath    string
	TableName   string
	ElementType string // "condition", "action", "context", "initial_action"
	RowNumber   int
	Message     string
}

func (e *ValidationError) Error() string {
	if e.TableName != "" {
		return fmt.Sprintf("%s (table %s, %s row %d): %s",
			e.FilePath, e.TableName, e.ElementType, e.RowNumber, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.FilePath, e.Message)
}

// ValidationWarning represents a non-fatal validation warning.
type ValidationWarning struct {
	FilePath  string
	TableName string
	Message   string
}

func (w *ValidationWarning) String() string {
	if w.TableName != "" {
		return fmt.Sprintf("%s (table %s): %s", w.FilePath, w.TableName, w.Message)
	}
	return fmt.Sprintf("%s: %s", w.FilePath, w.Message)
}

// FileValidationResult holds the validation result for a single file.
type FileValidationResult struct {
	FilePath string
	Status   ELComplianceStatus
	Tables   []TableValidationResult
}

// TableValidationResult holds the validation result for a single table.
type TableValidationResult struct {
	TableName       string
	HasELConditions bool
	HasELActions    bool
	HasELContexts   bool
	HasPostfix      bool
	IsLegacy        bool
	ELCompiled      bool // true if el_compiled attribute is set
}

// ValidationResult holds the result of EL compliance validation.
type ValidationResult struct {
	// Errors contains validation errors (files are non-compliant).
	Errors []*ValidationError
	// Warnings contains non-fatal validation warnings.
	Warnings []*ValidationWarning
	// LegacyFiles lists files with hand-coded postfix.
	LegacyFiles []string
	// CompliantFiles lists files with EL-compiled postfix.
	CompliantFiles []string
	// NeedsCompilationFiles lists files that need EL compilation.
	NeedsCompilationFiles []string
	// FileResults contains detailed results for each file.
	FileResults []*FileValidationResult
}

// IsCompliant returns true if all files are EL compliant (no legacy postfix).
func (r *ValidationResult) IsCompliant() bool {
	return len(r.LegacyFiles) == 0 && len(r.Errors) == 0
}

// HasLegacyPostfix returns true if any files have hand-coded postfix.
func (r *ValidationResult) HasLegacyPostfix() bool {
	return len(r.LegacyFiles) > 0
}

// ValidateELCompliance validates EL compliance for all XML files in a directory.
// It checks each decision table file to determine if it uses:
// - EL expressions with compiled postfix (compliant)
// - Hand-coded postfix without EL descriptions (legacy - error)
// - EL expressions without postfix (needs compilation)
func ValidateELCompliance(xmlDir string) (*ValidationResult, error) {
	result := &ValidationResult{}

	// Find all DT XML files
	dtFiles, err := findDTFiles(xmlDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan XML directory: %w", err)
	}

	for _, filePath := range dtFiles {
		fileResult, err := validateDTFile(filePath)
		if err != nil {
			result.Warnings = append(result.Warnings, &ValidationWarning{
				FilePath: filePath,
				Message:  fmt.Sprintf("failed to validate: %v", err),
			})
			continue
		}

		result.FileResults = append(result.FileResults, fileResult)

		// Categorize the file
		switch fileResult.Status {
		case StatusLegacy:
			result.LegacyFiles = append(result.LegacyFiles, filePath)
		case StatusCompliant:
			result.CompliantFiles = append(result.CompliantFiles, filePath)
		case StatusNeedsCompilation:
			result.NeedsCompilationFiles = append(result.NeedsCompilationFiles, filePath)
		}

		// Add errors for legacy files
		for _, tableResult := range fileResult.Tables {
			if tableResult.IsLegacy {
				result.Errors = append(result.Errors, &ValidationError{
					FilePath:  filePath,
					TableName: tableResult.TableName,
					Message:   "table has hand-coded postfix without EL descriptions",
				})
			}
		}
	}

	return result, nil
}

// IsLegacyPostfixFile checks if a single XML file contains legacy hand-coded postfix.
// Returns true if the file has postfix but no EL descriptions.
func IsLegacyPostfixFile(xmlPath string) (bool, error) {
	result, err := validateDTFile(xmlPath)
	if err != nil {
		return false, err
	}
	return result.Status == StatusLegacy, nil
}

// findDTFiles finds all decision table XML files in a directory.
func findDTFiles(xmlDir string) ([]string, error) {
	var dtFiles []string

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip testfiles and schemas directories
			name := d.Name()
			if name == "testfiles" || name == "schemas" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for DT files
		if isDTFile(path) {
			dtFiles = append(dtFiles, path)
		}

		return nil
	})

	return dtFiles, err
}

// validateDTFile validates a single decision table XML file.
func validateDTFile(filePath string) (*FileValidationResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	result := &FileValidationResult{
		FilePath: filePath,
		Status:   StatusEmpty,
	}

	// Parse the XML
	tables, err := parseDTXML(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	hasAnyContent := false
	hasAnyLegacy := false
	allCompliant := true

	for _, table := range tables {
		tableResult := validateTable(&table)
		result.Tables = append(result.Tables, tableResult)

		if tableResult.HasPostfix || tableResult.HasELConditions || tableResult.HasELActions {
			hasAnyContent = true
		}

		if tableResult.IsLegacy {
			hasAnyLegacy = true
			allCompliant = false
		}

		if !tableResult.ELCompiled && (tableResult.HasELConditions || tableResult.HasELActions) {
			allCompliant = false
		}
	}

	// Determine overall file status
	if !hasAnyContent {
		result.Status = StatusEmpty
	} else if hasAnyLegacy {
		result.Status = StatusLegacy
	} else if allCompliant {
		result.Status = StatusCompliant
	} else {
		result.Status = StatusNeedsCompilation
	}

	return result, nil
}

// validateTable validates a single decision table for EL compliance.
func validateTable(table *dtTableXML) TableValidationResult {
	result := TableValidationResult{
		TableName: table.getTableName(),
	}

	// Check contexts
	for i := range table.Contexts.Contexts {
		ctx := &table.Contexts.Contexts[i]
		if hasContent(ctx.GetDSL()) {
			result.HasELContexts = true
		}
		if hasContent(ctx.Postfix) {
			result.HasPostfix = true
		}
	}

	// Check conditions
	for i := range table.Conditions.Conditions {
		cond := &table.Conditions.Conditions[i]
		if hasContent(cond.GetDSL()) {
			result.HasELConditions = true
		}
		if hasContent(cond.Postfix) {
			result.HasPostfix = true
		}
	}

	// Check actions
	for i := range table.Actions.Actions {
		action := &table.Actions.Actions[i]
		if hasContent(action.GetDSL()) {
			result.HasELActions = true
		}
		if hasContent(action.Postfix) {
			result.HasPostfix = true
		}
	}

	// Check initial actions
	for i := range table.InitialActions.Actions {
		action := &table.InitialActions.Actions[i]
		if hasContent(action.GetDSL()) {
			result.HasELActions = true
		}
		if hasContent(action.Postfix) {
			result.HasPostfix = true
		}
	}

	// Determine legacy status
	// A table is legacy if it has postfix but no EL descriptions
	if result.HasPostfix && !result.HasELConditions && !result.HasELActions && !result.HasELContexts {
		result.IsLegacy = true
	}

	// Check for el_compiled attribute (indicates compliant file)
	result.ELCompiled = table.ELCompiled

	return result
}

// hasContent checks if a string has meaningful content (not just whitespace).
func hasContent(s string) bool {
	return strings.TrimSpace(s) != ""
}

// XML structures for parsing decision table files
// These are simplified versions just for compliance checking

type dtTablesXML struct {
	XMLName xml.Name     `xml:"decision_tables"`
	Tables  []dtTableXML `xml:"decision_table"`
}

type dtTableXML struct {
	NameAttr        string             `xml:"name,attr"`
	TableName       string             `xml:"table_name"`
	ELCompiled      bool               `xml:"el_compiled,attr"`
	Contexts        dtContextsXML      `xml:"contexts"`
	InitialActions  dtInitialActionsXML `xml:"initial_actions"`
	Conditions      dtConditionsXML    `xml:"conditions"`
	Actions         dtActionsXML       `xml:"actions"`
}

func (t *dtTableXML) getTableName() string {
	if t.NameAttr != "" {
		return t.NameAttr
	}
	return t.TableName
}

type dtContextsXML struct {
	Contexts []dtContextDetailXML `xml:"context_details"`
}

type dtContextDetailXML struct {
	Description string `xml:"context_description"`
	DSL         string `xml:"context_dsl"`
	Postfix     string `xml:"context_postfix"`
}

func (c *dtContextDetailXML) GetDSL() string {
	if c.DSL != "" {
		return c.DSL
	}
	return c.Description
}

type dtInitialActionsXML struct {
	Actions []dtInitialActionXML `xml:"initial_action_details"`
}

type dtInitialActionXML struct {
	Description string `xml:"initial_action_description"`
	DSL         string `xml:"initial_action_dsl"`
	Postfix     string `xml:"initial_action_postfix"`
}

func (a *dtInitialActionXML) GetDSL() string {
	if a.DSL != "" {
		return a.DSL
	}
	return a.Description
}

type dtConditionsXML struct {
	Conditions []dtConditionDetailXML `xml:"condition_details"`
}

type dtConditionDetailXML struct {
	Description string `xml:"condition_description"`
	DSL         string `xml:"condition_dsl"`
	Postfix     string `xml:"condition_postfix"`
}

func (c *dtConditionDetailXML) GetDSL() string {
	if c.DSL != "" {
		return c.DSL
	}
	return c.Description
}

type dtActionsXML struct {
	Actions []dtActionDetailXML `xml:"action_details"`
}

type dtActionDetailXML struct {
	Description string `xml:"action_description"`
	DSL         string `xml:"action_dsl"`
	Postfix     string `xml:"action_postfix"`
}

func (a *dtActionDetailXML) GetDSL() string {
	if a.DSL != "" {
		return a.DSL
	}
	return a.Description
}

// parseDTXML parses a decision table XML file.
func parseDTXML(r io.Reader) ([]dtTableXML, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var dtFile dtTablesXML
	if err := xml.Unmarshal(data, &dtFile); err != nil {
		return nil, err
	}

	return dtFile.Tables, nil
}
