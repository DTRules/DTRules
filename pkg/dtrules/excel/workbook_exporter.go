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

package excel

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// WorkbookExporter exports XML files to Excel format.
// This implements the sync.ExcelExporter interface by loading XML files
// and using the Exporter to generate Excel files.
type WorkbookExporter struct {
	verbose bool
	stats   *ExportStats // accumulated stats across all export calls
}

// NewWorkbookExporter creates a new workbook exporter.
func NewWorkbookExporter() *WorkbookExporter {
	return &WorkbookExporter{}
}

// SetVerbose enables verbose output during export.
func (w *WorkbookExporter) SetVerbose(v bool) {
	w.verbose = v
}

// ResetStats resets the accumulated export statistics.
func (w *WorkbookExporter) ResetStats() {
	w.stats = &ExportStats{}
}

// TakeStats returns the accumulated export statistics and resets the collector.
func (w *WorkbookExporter) TakeStats() *ExportStats {
	s := w.stats
	w.stats = nil
	return s
}

// ExportDecisionTable exports a decision table XML file to Excel.
// This loads the XML, creates a RuleSet, and exports using the standard Exporter.
// Implements sync.ExcelExporter interface.
func (w *WorkbookExporter) ExportDecisionTable(xmlPath, excelPath string) error {
	if w.verbose {
		fmt.Printf("Exporting DT: %s -> %s\n", xmlPath, excelPath)
	}

	// Check if XML file exists
	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		return fmt.Errorf("XML file not found: %s", xmlPath)
	}

	// For decision tables, we need both EDD and DT to create a valid RuleSet.
	// Try to find corresponding EDD file.
	eddPath := w.findCorrespondingEDD(xmlPath)

	// Create a RuleSet and load the XML
	rs := session.NewRuleSet("export")
	if rs == nil {
		return fmt.Errorf("failed to create rule set")
	}

	// Load EDD first if found
	if eddPath != "" {
		if w.verbose {
			fmt.Printf("  Loading EDD: %s\n", eddPath)
		}
		if err := rs.LoadEDDFile(eddPath); err != nil {
			// EDD loading failure is not fatal for DT export
			if w.verbose {
				fmt.Printf("  Warning: failed to load EDD: %v\n", err)
			}
		}
	}

	// Load the decision table in tolerant mode — the XML-authored build's
	// export step reads partially-compiled XML to write Excel before the
	// import step runs the compile. Strict loading would refuse this
	// mid-pipeline state. The compile drop check that follows on the
	// import side is what fails the build when EL doesn't parse.
	if err := rs.LoadDecisionTablesTolerantFile(xmlPath); err != nil {
		return fmt.Errorf("failed to load decision table: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(excelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create exporter and export
	exporter := NewExporter(rs)
	if err := exporter.ExportDecisionTables(excelPath); err != nil {
		return fmt.Errorf("failed to export to Excel: %w", err)
	}

	if w.verbose {
		fmt.Printf("  Exported %d decision tables\n", len(rs.GetDecisionTableNames()))
	}

	return nil
}

// ExportEDD exports an EDD XML file to Excel.
// This loads the XML, creates a RuleSet, and exports using the standard Exporter.
// Implements sync.ExcelExporter interface.
func (w *WorkbookExporter) ExportEDD(xmlPath, excelPath string) error {
	if w.verbose {
		fmt.Printf("Exporting EDD: %s -> %s\n", xmlPath, excelPath)
	}

	// Check if XML file exists
	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		return fmt.Errorf("XML file not found: %s", xmlPath)
	}

	// Create a RuleSet and load the EDD
	rs := session.NewRuleSet("export")
	if rs == nil {
		return fmt.Errorf("failed to create rule set")
	}

	if err := rs.LoadEDDFile(xmlPath); err != nil {
		return fmt.Errorf("failed to load EDD: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(excelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create exporter and export
	exporter := NewExporter(rs)
	if err := exporter.ExportEDD(excelPath); err != nil {
		return fmt.Errorf("failed to export to Excel: %w", err)
	}

	if w.verbose {
		fmt.Printf("  Exported %d entities\n", len(rs.GetEntityNames()))
	}

	return nil
}

// ExportCombined exports both DT and EDD XML files to a single Excel workbook.
// This implements sync.CombinedExporter interface.
// It creates a single workbook containing:
//   - One sheet per decision table
//   - One EDD sheet with all entities
//
// Example: xml/states/CO_dt.xml + xml/states/CO_edd.xml -> excel/states/CO.xlsx
func (w *WorkbookExporter) ExportCombined(dtPath, eddPath, excelPath string) error {
	if w.verbose {
		fmt.Printf("Exporting combined workbook: %s\n", excelPath)
		if dtPath != "" {
			fmt.Printf("  DT: %s\n", dtPath)
		}
		if eddPath != "" {
			fmt.Printf("  EDD: %s\n", eddPath)
		}
	}

	// Create a RuleSet and load both files
	rs := session.NewRuleSet("export")
	if rs == nil {
		return fmt.Errorf("failed to create rule set")
	}

	// Load EDD first (if exists) - entities must be loaded before DT
	if eddPath != "" {
		if _, err := os.Stat(eddPath); err == nil {
			if err := rs.LoadEDDFile(eddPath); err != nil {
				return fmt.Errorf("failed to load EDD: %w", err)
			}
		}
	}

	// Load DT (if exists). Tolerant: see the comment on the matching
	// load above — export reads XML mid-pipeline, before the build's
	// compile step has filled in postfix.
	if dtPath != "" {
		if _, err := os.Stat(dtPath); err == nil {
			if err := rs.LoadDecisionTablesTolerantFile(dtPath); err != nil {
				return fmt.Errorf("failed to load decision table: %w", err)
			}
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(excelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Count postfix blocks that will be stripped (exist in XML, not in Excel).
	if dtPath != "" {
		if stripped, tables, actions, conditions := countPostfixInDTXML(dtPath); w.stats != nil {
			w.stats.PostfixStripped += stripped
			w.stats.Tables += tables
			w.stats.Actions += actions
			w.stats.Conditions += conditions
		}
	}
	if eddPath != "" {
		if entities := countEntitiesInEDDXML(eddPath); w.stats != nil {
			w.stats.Entities += entities
		}
	}

	// Create exporter
	exporter := NewExporter(rs)

	// Export both DT and EDD to a single combined workbook
	if err := exporter.ExportCombinedWorkbook(excelPath); err != nil {
		return fmt.Errorf("failed to export combined workbook: %w", err)
	}

	if w.stats != nil {
		w.stats.Files++
	}

	if w.verbose {
		fmt.Printf("  Exported %d decision tables, %d entities\n",
			len(rs.GetDecisionTableNames()), len(rs.GetEntityNames()))
	}

	return nil
}

// countPostfixInDTXML parses a DT XML file and returns the count of non-empty postfix
// blocks (which will be stripped when exported to Excel), plus table/action/condition counts.
func countPostfixInDTXML(dtPath string) (stripped, tables, actions, conditions int) {
	data, err := os.ReadFile(dtPath)
	if err != nil {
		return
	}
	var dt DecisionTablesXML
	if err := unmarshalDTXML(data, &dt); err != nil {
		return
	}
	tables = len(dt.Tables)
	for _, t := range dt.Tables {
		actions += len(t.Actions)
		conditions += len(t.Conditions)
		for _, a := range t.Actions {
			if a.Postfix != "" {
				stripped++
			}
		}
		for _, c := range t.Conditions {
			if c.Postfix != "" {
				stripped++
			}
		}
		for _, ia := range t.InitialActions {
			if ia.Postfix != "" {
				stripped++
			}
		}
	}
	return
}

// countEntitiesInEDDXML parses an EDD XML file and returns the entity count.
func countEntitiesInEDDXML(eddPath string) int {
	data, err := os.ReadFile(eddPath)
	if err != nil {
		return 0
	}
	var edd EDDXML
	if err := unmarshalEDDXML(data, &edd); err != nil {
		return 0
	}
	return len(edd.Entities)
}

// unmarshalDTXML parses a DT XML byte slice into a DecisionTablesXML struct.
func unmarshalDTXML(data []byte, out *DecisionTablesXML) error {
	return xml.Unmarshal(data, out)
}

// unmarshalEDDXML parses an EDD XML byte slice into an EDDXML struct.
func unmarshalEDDXML(data []byte, out *EDDXML) error {
	return xml.Unmarshal(data, out)
}

// findCorrespondingEDD attempts to find an EDD file that corresponds to a DT file.
// For example: CO_dt.xml -> CO_edd.xml
func (w *WorkbookExporter) findCorrespondingEDD(dtPath string) string {
	dir := filepath.Dir(dtPath)
	base := filepath.Base(dtPath)

	// Try common patterns
	patterns := []string{
		// Pattern 1: XX_dt.xml -> XX_edd.xml
		strings.Replace(base, "_dt.xml", "_edd.xml", 1),
		// Pattern 2: DecisionTables.xml -> EDD.xml
		strings.Replace(base, "DecisionTables.xml", "EDD.xml", 1),
		strings.Replace(base, "DecisionTables.xml", "edd.xml", 1),
		// Pattern 3: DT.xml -> EDD.xml
		strings.Replace(base, "_DT.xml", "_EDD.xml", 1),
		strings.Replace(base, "_dt.XML", "_edd.XML", 1),
	}

	for _, pattern := range patterns {
		if pattern == base {
			continue // Skip if pattern didn't change anything
		}
		eddPath := filepath.Join(dir, pattern)
		if _, err := os.Stat(eddPath); err == nil {
			return eddPath
		}
	}

	// Also check parent directory for EDD.xml
	parentEDD := filepath.Join(filepath.Dir(dir), "EDD.xml")
	if _, err := os.Stat(parentEDD); err == nil {
		return parentEDD
	}

	return ""
}
