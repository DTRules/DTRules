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
	"fmt"

	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

// CompilerOptions configures the sync-enabled compiler.
type CompilerOptions struct {
	// XMLDir is the directory containing XML rule files.
	XMLDir string
	// ExcelDir is the directory containing Excel rule files.
	ExcelDir string
	// SyncOptions configures synchronization behavior.
	SyncOptions *SyncOptions
	// SkipPreSync disables pre-compile Excel-to-XML sync.
	SkipPreSync bool
	// SkipPostSync disables post-compile XML-to-Excel sync.
	SkipPostSync bool
	// UseCombinedWorkbooks enables combined workbook mode where one Excel file
	// maps to multiple XML files (e.g., CO.xlsx -> CO_dt.xml, CO_edd.xml).
	UseCombinedWorkbooks bool
}

// CompileResult holds the result of a sync-enabled compilation.
type CompileResult struct {
	// RuleSet is the compiled rule set.
	RuleSet *session.RuleSet
	// PreSyncResult contains results from pre-compile sync (Excel -> XML).
	PreSyncResult *SyncResult
	// PostSyncResult contains results from post-compile sync (XML -> Excel).
	PostSyncResult *SyncResult
}

// SyncCompiler wraps rule compilation with Excel synchronization.
type SyncCompiler struct {
	options          *CompilerOptions
	syncer           *Syncer
	importer         ExcelImporter
	exporter         ExcelExporter
	workbookImporter WorkbookImporter
}

// NewSyncCompiler creates a new sync-enabled compiler.
func NewSyncCompiler(opts *CompilerOptions) *SyncCompiler {
	if opts.SyncOptions == nil {
		opts.SyncOptions = DefaultOptions()
	}

	syncer := NewSyncerWithOptions(opts.XMLDir, opts.ExcelDir, opts.SyncOptions)
	syncer.SetUseCombinedWorkbooks(opts.UseCombinedWorkbooks)

	return &SyncCompiler{
		options: opts,
		syncer:  syncer,
	}
}

// SetImporter sets the Excel importer for the compiler.
func (c *SyncCompiler) SetImporter(importer ExcelImporter) {
	c.importer = importer
	c.syncer.SetImporter(importer)
}

// SetExporter sets the Excel exporter for the compiler.
func (c *SyncCompiler) SetExporter(exporter ExcelExporter) {
	c.exporter = exporter
	c.syncer.SetExporter(exporter)
}

// SetWorkbookImporter sets the combined workbook importer for the compiler.
func (c *SyncCompiler) SetWorkbookImporter(importer WorkbookImporter) {
	c.workbookImporter = importer
	c.syncer.SetWorkbookImporter(importer)
}

// Compile performs compilation with optional pre/post sync.
// The workflow is:
//  1. Pre-compile: If Excel is newer than XML, import Excel to XML
//  2. Compile: Load XML files into a RuleSet
//  3. Post-compile: Export XML to Excel to maintain formatting/sync
func (c *SyncCompiler) Compile(name string) (*CompileResult, error) {
	result := &CompileResult{}

	// Step 1: Pre-compile sync (Excel -> XML)
	if !c.options.SkipPreSync {
		preSyncResult, err := c.preSyncExcelToXML()
		if err != nil {
			return result, fmt.Errorf("pre-compile sync failed: %w", err)
		}
		result.PreSyncResult = preSyncResult

		// Check for unresolved conflicts
		if preSyncResult.ConflictCount > 0 && c.options.SyncOptions.ConflictResolution == "error" {
			return result, fmt.Errorf("pre-compile sync found %d conflicts", preSyncResult.ConflictCount)
		}
	}

	// Step 2: Compile XML to RuleSet
	ruleSet, err := session.LoadRulesFromDirectory(name, c.options.XMLDir)
	if err != nil {
		return result, fmt.Errorf("compilation failed: %w", err)
	}
	result.RuleSet = ruleSet

	// Step 3: Post-compile sync (XML -> Excel)
	if !c.options.SkipPostSync {
		postSyncResult, err := c.postSyncXMLToExcel()
		if err != nil {
			// Log warning but don't fail compilation
			fmt.Printf("Warning: post-compile sync failed: %v\n", err)
		}
		result.PostSyncResult = postSyncResult
	}

	return result, nil
}

// preSyncExcelToXML syncs Excel files that are newer than their XML counterparts.
func (c *SyncCompiler) preSyncExcelToXML() (*SyncResult, error) {
	// Use combined workbook mode if enabled
	if c.options.UseCombinedWorkbooks {
		return c.preSyncCombinedWorkbooks()
	}

	direction, pairs, err := c.syncer.CheckSync()
	if err != nil {
		return nil, err
	}

	result := &SyncResult{Pairs: pairs}

	// Only sync Excel -> XML files
	for i := range pairs {
		pair := &pairs[i]
		if pair.Direction != ExcelToXML {
			continue
		}

		if c.options.SyncOptions.DryRun {
			result.ExcelToXMLCount++
			continue
		}

		if err := c.syncer.importExcelToXML(pair); err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.ExcelToXMLCount++
		}
	}

	// Count conflicts
	if direction == Conflict {
		for _, pair := range pairs {
			if pair.Direction == Conflict {
				result.ConflictCount++
			}
		}
	}

	return result, nil
}

// preSyncCombinedWorkbooks syncs combined Excel workbooks to XML files.
func (c *SyncCompiler) preSyncCombinedWorkbooks() (*SyncResult, error) {
	direction, workbooks, err := c.syncer.CheckSyncCombined()
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}

	// Only sync Excel -> XML workbooks
	for i := range workbooks {
		wb := &workbooks[i]
		if wb.Direction != ExcelToXML {
			continue
		}

		if c.options.SyncOptions.DryRun {
			result.ExcelToXMLCount++
			continue
		}

		if err := c.syncer.importCombinedWorkbook(wb); err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.ExcelToXMLCount++
		}
	}

	// Count conflicts
	if direction == Conflict {
		result.ConflictCount++
	}

	return result, nil
}

// postSyncXMLToExcel exports all XML files to Excel.
func (c *SyncCompiler) postSyncXMLToExcel() (*SyncResult, error) {
	// Use combined workbook mode if enabled
	if c.options.UseCombinedWorkbooks {
		return c.postSyncCombinedWorkbooks()
	}

	_, pairs, err := c.syncer.CheckSync()
	if err != nil {
		return nil, err
	}

	result := &SyncResult{Pairs: pairs}

	// Export all XML files that exist
	for i := range pairs {
		pair := &pairs[i]
		if !pair.XMLExists {
			continue
		}

		if c.options.SyncOptions.DryRun {
			result.XMLToExcelCount++
			continue
		}

		if err := c.syncer.exportXMLToExcel(pair); err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.XMLToExcelCount++
		}
	}

	return result, nil
}

// postSyncCombinedWorkbooks exports XML files to combined Excel workbooks.
func (c *SyncCompiler) postSyncCombinedWorkbooks() (*SyncResult, error) {
	_, workbooks, err := c.syncer.CheckSyncCombined()
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}

	// Export all workbooks that have XML files
	for i := range workbooks {
		wb := &workbooks[i]
		if !wb.DTExists && !wb.EDDExists {
			continue
		}

		if c.options.SyncOptions.DryRun {
			result.XMLToExcelCount++
			continue
		}

		if err := c.syncer.exportCombinedWorkbook(wb); err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.XMLToExcelCount++
		}
	}

	return result, nil
}

// GetSyncer returns the underlying syncer.
func (c *SyncCompiler) GetSyncer() *Syncer {
	return c.syncer
}

// CompileWithSync is a convenience function that wraps compilation with Excel sync.
// This is the main entry point for most use cases.
func CompileWithSync(name, xmlDir, excelDir string) (*CompileResult, error) {
	opts := &CompilerOptions{
		XMLDir:   xmlDir,
		ExcelDir: excelDir,
	}

	compiler := NewSyncCompiler(opts)
	return compiler.Compile(name)
}

// CompileWithSyncOptions is like CompileWithSync but with custom options.
func CompileWithSyncOptions(name, xmlDir, excelDir string, syncOpts *SyncOptions) (*CompileResult, error) {
	opts := &CompilerOptions{
		XMLDir:      xmlDir,
		ExcelDir:    excelDir,
		SyncOptions: syncOpts,
	}

	compiler := NewSyncCompiler(opts)
	return compiler.Compile(name)
}

// PreCompileSync performs only the pre-compile sync (Excel -> XML).
// Use this when you want to sync before using an external compiler.
func PreCompileSync(xmlDir, excelDir string) (*SyncResult, error) {
	syncer := NewSyncer(xmlDir, excelDir)
	return syncer.SyncAll()
}

// PostCompileSync performs only the post-compile sync (XML -> Excel).
// Use this when you want to regenerate Excel after compilation.
func PostCompileSync(xmlDir, excelDir string) error {
	syncer := NewSyncer(xmlDir, excelDir)
	return syncer.SyncXMLToExcel()
}
