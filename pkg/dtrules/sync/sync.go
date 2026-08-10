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

// Package sync provides bidirectional Excel <-> XML synchronization for DTRules.
// It uses timestamp-based detection to determine which files have been modified
// and syncs them in the appropriate direction before and after compilation.
package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SyncDirection indicates which way to sync files.
type SyncDirection int

const (
	// NoSync indicates no synchronization is needed.
	NoSync SyncDirection = iota
	// ExcelToXML indicates Excel is newer and should be imported to XML.
	ExcelToXML
	// XMLToExcel indicates XML is newer and should be exported to Excel.
	XMLToExcel
	// Conflict indicates both files have been modified since last sync.
	Conflict
)

// String returns the string representation of the sync direction.
func (d SyncDirection) String() string {
	switch d {
	case NoSync:
		return "NoSync"
	case ExcelToXML:
		return "ExcelToXML"
	case XMLToExcel:
		return "XMLToExcel"
	case Conflict:
		return "Conflict"
	default:
		return "Unknown"
	}
}

// FilePair represents matched XML and Excel files.
type FilePair struct {
	// XMLPath is the path to the XML file.
	XMLPath string
	// ExcelPath is the path to the corresponding Excel file.
	ExcelPath string
	// XMLMod is the modification time of the XML file.
	XMLMod time.Time
	// ExcelMod is the modification time of the Excel file.
	ExcelMod time.Time
	// Direction indicates which way to sync this file pair.
	Direction SyncDirection
	// XMLExists indicates whether the XML file exists.
	XMLExists bool
	// ExcelExists indicates whether the Excel file exists.
	ExcelExists bool
}

// CombinedWorkbook represents an Excel workbook that may contain both DT and EDD sheets.
// This supports the new directory structure where one Excel file maps to multiple XML files.
type CombinedWorkbook struct {
	// ExcelPath is the path to the Excel workbook.
	ExcelPath string
	// DTXMLPath is the path to the decision table XML file (may not exist).
	DTXMLPath string
	// EDDXMLPath is the path to the EDD XML file (may not exist).
	EDDXMLPath string
	// RelPath is the relative path from the base directory (without extension).
	RelPath string
	// ExcelMod is the modification time of the Excel file.
	ExcelMod time.Time
	// DTMod is the modification time of the DT XML file.
	DTMod time.Time
	// EDDMod is the modification time of the EDD XML file.
	EDDMod time.Time
	// ExcelExists indicates whether the Excel file exists.
	ExcelExists bool
	// DTExists indicates whether the DT XML file exists.
	DTExists bool
	// EDDExists indicates whether the EDD XML file exists.
	EDDExists bool
	// Direction indicates which way to sync this workbook.
	Direction SyncDirection
}

// outputDir is the directory this workbook's XML belongs in.
//
// A workbook produces a DT file, an EDD file, or both, and the unused path is
// left empty — so taking filepath.Dir of DTXMLPath alone answers "." for an
// EDD-only workbook, and the import lands in the process's working directory
// instead of the project. That is how `dtrules verify` came to leave a
// TaxReturn_edd.xml at the root of any project holding a *_edd.xlsx (#1056);
// verify is a read-only gate and should not write into the tree at all.
//
// fallback is the syncer's own xml directory, used when neither path is set.
func (wb *CombinedWorkbook) outputDir(fallback string) string {
	for _, p := range []string{wb.DTXMLPath, wb.EDDXMLPath} {
		if p != "" {
			return filepath.Dir(p)
		}
	}
	return fallback
}

// SyncResult holds the result of a sync operation.
type SyncResult struct {
	// Pairs contains all file pairs discovered.
	Pairs []FilePair
	// ExcelToXMLCount is the number of files synced from Excel to XML.
	ExcelToXMLCount int
	// XMLToExcelCount is the number of files synced from XML to Excel.
	XMLToExcelCount int
	// ConflictCount is the number of files with conflicts.
	ConflictCount int
	// Errors contains any errors that occurred during sync.
	Errors []error
}

// SyncOptions configures sync behavior.
type SyncOptions struct {
	// ConflictResolution specifies how to handle conflicts.
	// Values: "error" (default), "prefer-excel", "prefer-xml", "skip"
	ConflictResolution string
	// DryRun if true, reports what would be done without making changes.
	DryRun bool
	// Verbose if true, provides detailed output.
	Verbose bool
	// GracePeriod is the duration within which timestamps are considered equal.
	// This helps handle filesystem timestamp precision issues.
	GracePeriod time.Duration

	// ForceDirection overrides timestamp-based direction detection and syncs
	// every pair the named way. NoSync (the zero value) leaves detection in
	// charge.
	//
	// This exists for `verify`, which must rebuild XML from Excel and compare.
	// Detection cannot serve that: hand-editing the XML is precisely what
	// makes it newer, so detection answers XMLToExcel and the edit is
	// exported INTO Excel rather than being overwritten by it. The rebuild
	// then matches the edit and the gate passes — the check could never fail
	// for the one thing it exists to catch.
	ForceDirection SyncDirection
}

// DefaultOptions returns default sync options.
func DefaultOptions() *SyncOptions {
	return &SyncOptions{
		ConflictResolution: "error",
		DryRun:             false,
		Verbose:            false,
		GracePeriod:        2 * time.Second, // Handle filesystem timestamp precision
	}
}

// Syncer handles Excel <-> XML synchronization.
type Syncer struct {
	xmlDir   string
	excelDir string
	options  *SyncOptions

	// Importer and exporter interfaces (stubbed until actual implementations exist)
	importer         ExcelImporter
	exporter         ExcelExporter
	workbookImporter WorkbookImporter

	// UseCombinedWorkbooks enables the new combined workbook mode where
	// one Excel file (e.g., CO.xlsx) maps to multiple XML files (CO_dt.xml, CO_edd.xml).
	UseCombinedWorkbooks bool

	// manifest tracks export timestamps to enforce Excel as source of truth.
	// If Excel was modified after the last export, subsequent exports are rejected.
	manifest *Manifest
}

// ExcelImporter defines the interface for importing Excel to XML.
// This will be implemented by the excel importer package.
type ExcelImporter interface {
	// ImportDecisionTable imports a decision table from Excel to XML.
	ImportDecisionTable(excelPath, xmlPath string) error
	// ImportEDD imports an Entity Data Dictionary from Excel to XML.
	ImportEDD(excelPath, xmlPath string) error
}

// WorkbookImporter defines the interface for importing combined workbooks.
// A combined workbook may contain both DT and EDD sheets in a single Excel file.
type WorkbookImporter interface {
	// ImportWorkbook imports a combined workbook to separate XML files.
	// It returns the paths to the generated XML files (dtPath, eddPath).
	// Either path may be empty if that sheet type doesn't exist in the workbook.
	ImportWorkbook(excelPath, xmlDir string) (dtPath, eddPath string, err error)
	// HasDTSheet checks if the workbook contains a decision table sheet.
	HasDTSheet(excelPath string) (bool, error)
	// HasEDDSheet checks if the workbook contains an EDD sheet.
	HasEDDSheet(excelPath string) (bool, error)
}

// ExcelExporter defines the interface for exporting XML to Excel.
// This is already implemented in the excel package.
type ExcelExporter interface {
	// ExportDecisionTable exports a decision table from XML to Excel.
	ExportDecisionTable(xmlPath, excelPath string) error
	// ExportEDD exports an Entity Data Dictionary from XML to Excel.
	ExportEDD(xmlPath, excelPath string) error
}

// CombinedExporter is an optional extension of ExcelExporter that supports
// exporting both DT and EDD to a single workbook.
type CombinedExporter interface {
	ExcelExporter
	// ExportCombined exports both DT and EDD XML files to a single Excel workbook.
	// Either path may be empty if that file doesn't exist.
	ExportCombined(dtXMLPath, eddXMLPath, excelPath string) error
}

// NewSyncer creates a new syncer for the given directories.
func NewSyncer(xmlDir, excelDir string) *Syncer {
	return &Syncer{
		xmlDir:   xmlDir,
		excelDir: excelDir,
		options:  DefaultOptions(),
	}
}

// NewSyncerWithOptions creates a new syncer with custom options.
func NewSyncerWithOptions(xmlDir, excelDir string, options *SyncOptions) *Syncer {
	if options == nil {
		options = DefaultOptions()
	}
	return &Syncer{
		xmlDir:   xmlDir,
		excelDir: excelDir,
		options:  options,
	}
}

// SetImporter sets the Excel importer.
func (s *Syncer) SetImporter(importer ExcelImporter) {
	s.importer = importer
}

// SetExporter sets the Excel exporter.
func (s *Syncer) SetExporter(exporter ExcelExporter) {
	s.exporter = exporter
}

// SetWorkbookImporter sets the combined workbook importer.
func (s *Syncer) SetWorkbookImporter(importer WorkbookImporter) {
	s.workbookImporter = importer
}

// SetUseCombinedWorkbooks enables or disables combined workbook mode.
func (s *Syncer) SetUseCombinedWorkbooks(enabled bool) {
	s.UseCombinedWorkbooks = enabled
}

// LoadManifest loads the sync manifest from the Excel directory.
// The manifest tracks when exports occurred to enforce Excel as source of truth.
func (s *Syncer) LoadManifest() error {
	m, err := LoadManifestFromDir(s.excelDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}
	s.manifest = m
	return nil
}

// GetManifest returns the current manifest, loading it if necessary.
func (s *Syncer) GetManifest() (*Manifest, error) {
	if s.manifest == nil {
		if err := s.LoadManifest(); err != nil {
			return nil, err
		}
	}
	return s.manifest, nil
}

// SetManifest sets the manifest (primarily for testing).
func (s *Syncer) SetManifest(m *Manifest) {
	s.manifest = m
}

// CheckExcelModified checks if any Excel files have been modified since last export.
// Returns the list of modified files. If Excel was modified, AI should NOT export.
func (s *Syncer) CheckExcelModified() ([]string, error) {
	m, err := s.GetManifest()
	if err != nil {
		return nil, err
	}
	return m.CheckAllExports()
}

// RequireNoUserEdits checks that no Excel files have been modified since the last export.
// Call this before making XML changes to ensure you won't be blocked from exporting.
// Returns an error if any Excel files have pending user changes.
func (s *Syncer) RequireNoUserEdits() error {
	modified, err := s.CheckExcelModified()
	if err != nil {
		return fmt.Errorf("failed to check Excel modifications: %w", err)
	}

	if len(modified) > 0 {
		return fmt.Errorf("Excel files have been modified by user since last export: %v. "+
			"Import Excel to XML first, then make your changes", modified)
	}

	return nil
}

// CheckSync determines sync direction based on timestamps.
// Returns the overall direction and list of files needing sync.
func (s *Syncer) CheckSync() (SyncDirection, []FilePair, error) {
	pairs, err := s.collectFilePairs()
	if err != nil {
		return NoSync, nil, fmt.Errorf("failed to collect file pairs: %w", err)
	}

	if len(pairs) == 0 {
		return NoSync, nil, nil
	}

	// Determine direction for each pair
	for i := range pairs {
		pairs[i].Direction = s.determineDirection(&pairs[i])
	}

	// Determine overall direction
	overallDirection := s.determineOverallDirection(pairs)

	return overallDirection, pairs, nil
}

// collectFilePairs discovers all XML/Excel file pairs.
func (s *Syncer) collectFilePairs() ([]FilePair, error) {
	var pairs []FilePair
	seenPairs := make(map[string]bool)

	// Scan XML directory for XML files
	xmlFiles, err := s.collectFiles(s.xmlDir, ".xml")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan XML directory: %w", err)
	}

	for _, xmlPath := range xmlFiles {
		excelPath := s.xmlToExcelPath(xmlPath)
		pairKey := s.normalizePath(xmlPath)

		if seenPairs[pairKey] {
			continue
		}
		seenPairs[pairKey] = true

		pair := s.createFilePair(xmlPath, excelPath)
		pairs = append(pairs, pair)
	}

	// Scan Excel directory for Excel files (catch files without XML counterpart)
	excelFiles, err := s.collectFiles(s.excelDir, ".xlsx")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan Excel directory: %w", err)
	}

	for _, excelPath := range excelFiles {
		xmlPath := s.excelToXMLPath(excelPath)
		pairKey := s.normalizePath(xmlPath)

		if seenPairs[pairKey] {
			continue
		}
		seenPairs[pairKey] = true

		pair := s.createFilePair(xmlPath, excelPath)
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// collectFiles collects all files with the given extension from a directory.
func (s *Syncer) collectFiles(dir, ext string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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

		// Check file extension
		if strings.ToLower(filepath.Ext(path)) == ext {
			// Skip template files
			if strings.Contains(strings.ToUpper(filepath.Base(path)), "TEMPLATE") {
				return nil
			}
			// Skip mapping files
			if strings.HasSuffix(filepath.Base(path), "_map.xml") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// xmlToExcelPath converts an XML path to its corresponding Excel path.
// In combined workbook mode:
//
//	xml/states/CO_dt.xml -> excel/states/CO.xlsx        (mixed workbook, no suffix file)
//	xml/001_Foo_dt.xml   -> excel/001_Foo_dt.xlsx       (single-artifact, suffix file exists)
//
// In legacy mode:
//
//	xml/TaxReturn_dt.xml -> excel/TaxReturn_dt.xlsx
func (s *Syncer) xmlToExcelPath(xmlPath string) string {
	// Get relative path from XML directory
	relPath, err := filepath.Rel(s.xmlDir, xmlPath)
	if err != nil {
		// Fallback: just use the filename
		relPath = filepath.Base(xmlPath)
	}

	if s.UseCombinedWorkbooks {
		// Prefer a same-suffix xlsx if it exists on disk (single-artifact convention).
		// e.g. 001_Foo_dt.xml → 001_Foo_dt.xlsx (when that file exists)
		sameNameXlsx := filepath.Join(s.excelDir, strings.TrimSuffix(relPath, ".xml")+".xlsx")
		if _, err := os.Stat(sameNameXlsx); err == nil {
			return sameNameXlsx
		}

		// Fall back: strip _dt or _edd suffix to find the combined workbook.
		// e.g. CO_dt.xml → CO.xlsx (shared by CO_dt.xml and CO_edd.xml)
		relPath = strings.TrimSuffix(relPath, "_dt.xml")
		relPath = strings.TrimSuffix(relPath, "_edd.xml")
		relPath = strings.TrimSuffix(relPath, ".xml")
		return filepath.Join(s.excelDir, relPath+".xlsx")
	}

	// Legacy mode: direct 1:1 mapping
	relPath = strings.TrimSuffix(relPath, ".xml") + ".xlsx"
	return filepath.Join(s.excelDir, relPath)
}

// excelToXMLPath converts an Excel path to its corresponding XML path.
// In legacy mode: excel/TaxReturn_dt.xlsx -> xml/TaxReturn_dt.xml
// In combined mode, use excelToXMLPaths instead.
func (s *Syncer) excelToXMLPath(excelPath string) string {
	// Get relative path from Excel directory
	relPath, err := filepath.Rel(s.excelDir, excelPath)
	if err != nil {
		// Fallback: just use the filename
		relPath = filepath.Base(excelPath)
	}

	// Change extension from .xlsx to .xml
	relPath = strings.TrimSuffix(relPath, ".xlsx") + ".xml"

	return filepath.Join(s.xmlDir, relPath)
}

// excelToXMLPaths converts an Excel path to its corresponding XML paths.
// In combined workbook mode, one Excel file maps to multiple XML files:
//
//	excel/states/CO.xlsx -> [xml/states/CO_dt.xml, xml/states/CO_edd.xml]
//
// Returns both paths; the caller should check if each file exists.
func (s *Syncer) excelToXMLPaths(excelPath string) []string {
	// Get relative path from Excel directory
	relPath, err := filepath.Rel(s.excelDir, excelPath)
	if err != nil {
		// Fallback: just use the filename
		relPath = filepath.Base(excelPath)
	}

	// Remove .xlsx extension
	base := strings.TrimSuffix(relPath, ".xlsx")

	if s.UseCombinedWorkbooks {
		// Single-artifact workbooks (suffix in filename) map to only their artifact type.
		// e.g. 001_Foo_dt.xlsx → [001_Foo_dt.xml] only
		if strings.HasSuffix(base, "_dt") {
			return []string{filepath.Join(s.xmlDir, base+".xml")}
		}
		if strings.HasSuffix(base, "_edd") {
			return []string{filepath.Join(s.xmlDir, base+".xml")}
		}

		// Mixed-artifact workbooks: return both DT and EDD paths.
		// e.g. CO.xlsx → [CO_dt.xml, CO_edd.xml]
		return []string{
			filepath.Join(s.xmlDir, base+"_dt.xml"),
			filepath.Join(s.xmlDir, base+"_edd.xml"),
		}
	}

	// Legacy mode: single XML file
	return []string{filepath.Join(s.xmlDir, base+".xml")}
}

// normalizePath returns a normalized path key for deduplication.
func (s *Syncer) normalizePath(xmlPath string) string {
	relPath, err := filepath.Rel(s.xmlDir, xmlPath)
	if err != nil {
		return filepath.Base(xmlPath)
	}
	return strings.ToLower(relPath)
}

// createFilePair creates a FilePair with file metadata.
func (s *Syncer) createFilePair(xmlPath, excelPath string) FilePair {
	pair := FilePair{
		XMLPath:   xmlPath,
		ExcelPath: excelPath,
	}

	// Check XML file
	if info, err := os.Stat(xmlPath); err == nil {
		pair.XMLExists = true
		pair.XMLMod = info.ModTime()
	}

	// Check Excel file
	if info, err := os.Stat(excelPath); err == nil {
		pair.ExcelExists = true
		pair.ExcelMod = info.ModTime()
	}

	return pair
}

// determineDirection determines the sync direction for a file pair.
func (s *Syncer) determineDirection(pair *FilePair) SyncDirection {
	// If only one file exists, sync from that file
	if pair.XMLExists && !pair.ExcelExists {
		return XMLToExcel
	}
	if !pair.XMLExists && pair.ExcelExists {
		return ExcelToXML
	}
	if !pair.XMLExists && !pair.ExcelExists {
		return NoSync
	}

	// Both files exist - compare timestamps
	diff := pair.ExcelMod.Sub(pair.XMLMod)

	// Within grace period = considered equal (no sync needed)
	if diff > -s.options.GracePeriod && diff < s.options.GracePeriod {
		return NoSync
	}

	// Excel is newer
	if diff > 0 {
		return ExcelToXML
	}

	// XML is newer
	return XMLToExcel
}

// determineOverallDirection determines the overall sync direction.
// If all files agree, return that direction. If mixed, return Conflict.
func (s *Syncer) determineOverallDirection(pairs []FilePair) SyncDirection {
	hasExcelToXML := false
	hasXMLToExcel := false
	hasConflict := false

	for _, pair := range pairs {
		switch pair.Direction {
		case ExcelToXML:
			hasExcelToXML = true
		case XMLToExcel:
			hasXMLToExcel = true
		case Conflict:
			hasConflict = true
		}
	}

	// If any conflicts, overall is conflict
	if hasConflict {
		return Conflict
	}

	// If mixed directions, treat as conflict
	if hasExcelToXML && hasXMLToExcel {
		return Conflict
	}

	// Single direction
	if hasExcelToXML {
		return ExcelToXML
	}
	if hasXMLToExcel {
		return XMLToExcel
	}

	return NoSync
}

// SyncExcelToXML imports all newer Excel files to XML.
func (s *Syncer) SyncExcelToXML() error {
	_, pairs, err := s.CheckSync()
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		if pair.Direction != ExcelToXML {
			continue
		}

		if err := s.importExcelToXML(&pair); err != nil {
			return fmt.Errorf("failed to import %s: %w", pair.ExcelPath, err)
		}
	}

	return nil
}

// SyncXMLToExcel exports all XML files to Excel.
func (s *Syncer) SyncXMLToExcel() error {
	_, pairs, err := s.CheckSync()
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		if pair.Direction != XMLToExcel && pair.Direction != NoSync {
			continue
		}

		// Skip if XML doesn't exist
		if !pair.XMLExists {
			continue
		}

		if err := s.exportXMLToExcel(&pair); err != nil {
			return fmt.Errorf("failed to export %s: %w", pair.XMLPath, err)
		}
	}

	return nil
}

// SyncAll performs full bidirectional sync based on timestamps.
func (s *Syncer) SyncAll() (*SyncResult, error) {
	result := &SyncResult{}

	// Use combined workbook mode if enabled
	if s.UseCombinedWorkbooks {
		return s.syncAllCombined()
	}

	// Legacy pair-based mode
	_, pairs, err := s.CheckSync()
	if err != nil {
		return result, err
	}

	result.Pairs = pairs

	for i := range pairs {
		pair := &pairs[i]

		if s.options.ForceDirection != NoSync {
			pair.Direction = s.options.ForceDirection
		}

		switch pair.Direction {
		case ExcelToXML:
			if s.options.DryRun {
				result.ExcelToXMLCount++
				continue
			}
			if err := s.importExcelToXML(pair); err != nil {
				result.Errors = append(result.Errors, err)
			} else {
				result.ExcelToXMLCount++
			}

		case XMLToExcel:
			if s.options.DryRun {
				result.XMLToExcelCount++
				continue
			}
			if err := s.exportXMLToExcel(pair); err != nil {
				result.Errors = append(result.Errors, err)
			} else {
				result.XMLToExcelCount++
			}

		case Conflict:
			result.ConflictCount++
			if err := s.handleConflict(pair); err != nil {
				result.Errors = append(result.Errors, err)
			}

		case NoSync:
			// Nothing to do
		}
	}

	return result, nil
}

// syncAllCombined performs sync using combined workbook mode.
// In this mode, one Excel file maps to multiple XML files (DT + EDD).
func (s *Syncer) syncAllCombined() (*SyncResult, error) {
	result := &SyncResult{}

	_, workbooks, err := s.CheckSyncCombined()
	if err != nil {
		return result, err
	}

	for i := range workbooks {
		wb := &workbooks[i]

		if s.options.ForceDirection != NoSync {
			wb.Direction = s.options.ForceDirection
		}

		switch wb.Direction {
		case ExcelToXML:
			if s.options.DryRun {
				result.ExcelToXMLCount++
				continue
			}
			if err := s.importCombinedWorkbook(wb); err != nil {
				result.Errors = append(result.Errors, err)
			} else {
				result.ExcelToXMLCount++
			}

		case XMLToExcel:
			if s.options.DryRun {
				result.XMLToExcelCount++
				continue
			}
			if err := s.exportCombinedWorkbook(wb); err != nil {
				result.Errors = append(result.Errors, err)
			} else {
				result.XMLToExcelCount++
			}

		case Conflict:
			result.ConflictCount++
			// Handle conflict based on resolution strategy
			switch s.options.ConflictResolution {
			case "prefer-excel":
				if err := s.importCombinedWorkbook(wb); err != nil {
					result.Errors = append(result.Errors, err)
				}
			case "prefer-xml":
				if err := s.exportCombinedWorkbook(wb); err != nil {
					result.Errors = append(result.Errors, err)
				}
			case "skip":
				// Skip this workbook
			default: // "error"
				result.Errors = append(result.Errors,
					fmt.Errorf("conflict detected for %s", wb.ExcelPath))
			}

		case NoSync:
			// Nothing to do
		}
	}

	return result, nil
}

// importExcelToXML imports a single Excel file to XML.
func (s *Syncer) importExcelToXML(pair *FilePair) error {
	// In combined workbook mode, use the workbook importer
	if s.UseCombinedWorkbooks && s.workbookImporter != nil {
		// Determine the XML directory from the pair's XMLPath
		xmlDir := filepath.Dir(pair.XMLPath)

		// Import the workbook
		_, _, err := s.workbookImporter.ImportWorkbook(pair.ExcelPath, xmlDir)
		return err
	}

	// Fall back to legacy importer
	if s.importer == nil {
		return fmt.Errorf("Excel importer not configured (stub: import %s -> %s)",
			pair.ExcelPath, pair.XMLPath)
	}

	// Determine file type based on filename convention
	if isEDDFile(pair.ExcelPath) {
		return s.importer.ImportEDD(pair.ExcelPath, pair.XMLPath)
	}
	return s.importer.ImportDecisionTable(pair.ExcelPath, pair.XMLPath)
}

// exportXMLToExcel exports a single XML file to Excel.
// This function enforces Excel as source of truth by checking the manifest
// before exporting. If Excel was modified since the last export, the export
// is rejected to prevent overwriting user changes.
func (s *Syncer) exportXMLToExcel(pair *FilePair) error {
	if s.exporter == nil {
		return fmt.Errorf("Excel exporter not configured (stub: export %s -> %s)",
			pair.XMLPath, pair.ExcelPath)
	}

	// ENFORCEMENT: Check if Excel was modified since last export (skip with prefer-xml)
	m, err := s.GetManifest()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if s.options.ConflictResolution != "prefer-xml" {
		if err := m.ExportGuard(pair.ExcelPath); err != nil {
			return err // Returns ExcelModifiedError if user modified Excel
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(pair.ExcelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine file type based on filename convention
	var exportErr error
	if isEDDFile(pair.XMLPath) {
		exportErr = s.exporter.ExportEDD(pair.XMLPath, pair.ExcelPath)
	} else {
		exportErr = s.exporter.ExportDecisionTable(pair.XMLPath, pair.ExcelPath)
	}

	if exportErr != nil {
		return exportErr
	}

	// Record successful export in manifest
	if err := m.RecordExport(pair.ExcelPath, []string{pair.XMLPath}); err != nil {
		// Log warning but don't fail the export
		if s.options.Verbose {
			fmt.Printf("Warning: failed to update manifest: %v\n", err)
		}
	}

	return nil
}

// handleConflict handles a file pair with conflicting modifications.
func (s *Syncer) handleConflict(pair *FilePair) error {
	switch s.options.ConflictResolution {
	case "prefer-excel":
		return s.importExcelToXML(pair)
	case "prefer-xml":
		return s.exportXMLToExcel(pair)
	case "skip":
		return nil
	case "error":
		fallthrough
	default:
		return fmt.Errorf("conflict detected: both %s and %s have been modified",
			pair.XMLPath, pair.ExcelPath)
	}
}

// isEDDFile determines if a file is an EDD based on naming convention.
func isEDDFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "_edd") || strings.Contains(base, "edd_")
}

// isDTFile determines if a file is a DT based on naming convention.
func isDTFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "_dt") || strings.Contains(base, "dt_")
}

// collectCombinedWorkbooks discovers all Excel workbooks and their corresponding XML files.
// This is used in combined workbook mode where one Excel file maps to multiple XML files.
func (s *Syncer) collectCombinedWorkbooks() ([]CombinedWorkbook, error) {
	workbookMap := make(map[string]*CombinedWorkbook)

	// Scan Excel directory for Excel files
	excelFiles, err := s.collectFiles(s.excelDir, ".xlsx")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan Excel directory: %w", err)
	}

	for _, excelPath := range excelFiles {
		relPath, _ := filepath.Rel(s.excelDir, excelPath)
		base := strings.TrimSuffix(relPath, ".xlsx")

		// Mapping workbooks have no sync support yet (#929). Pairing one as
		// a mixed-artifact workbook would look for <base>_dt.xml, conclude
		// "Excel exists but no XML", and IMPORT the map sheets into
		// fabricated <base>_dt.xml/_edd.xml files — corrupting the project
		// and flipping the whole build's direction detection to Conflict.
		// Skip them until map sync exists.
		if strings.HasSuffix(base, "_map") {
			continue
		}

		var wb *CombinedWorkbook

		// Single-artifact workbooks have a _dt or _edd suffix in the filename.
		// e.g. 001_Foo_dt.xlsx → DTXMLPath = 001_Foo_dt.xml (same base, not 001_Foo_dt_dt.xml)
		if strings.HasSuffix(base, "_dt") {
			wb = &CombinedWorkbook{
				ExcelPath:  excelPath,
				DTXMLPath:  filepath.Join(s.xmlDir, base+".xml"),
				EDDXMLPath: "", // not expected
				RelPath:    base,
			}
		} else if strings.HasSuffix(base, "_edd") {
			wb = &CombinedWorkbook{
				ExcelPath:  excelPath,
				DTXMLPath:  "", // not expected
				EDDXMLPath: filepath.Join(s.xmlDir, base+".xml"),
				RelPath:    base,
			}
		} else {
			// Mixed-artifact workbook: maps to both _dt and _edd XML files.
			wb = &CombinedWorkbook{
				ExcelPath:  excelPath,
				DTXMLPath:  filepath.Join(s.xmlDir, base+"_dt.xml"),
				EDDXMLPath: filepath.Join(s.xmlDir, base+"_edd.xml"),
				RelPath:    base,
			}
		}

		// Check Excel file
		if info, err := os.Stat(excelPath); err == nil {
			wb.ExcelExists = true
			wb.ExcelMod = info.ModTime()
		}

		// Check DT XML file
		if wb.DTXMLPath != "" {
			if info, err := os.Stat(wb.DTXMLPath); err == nil {
				wb.DTExists = true
				wb.DTMod = info.ModTime()
			}
		}

		// Check EDD XML file
		if wb.EDDXMLPath != "" {
			if info, err := os.Stat(wb.EDDXMLPath); err == nil {
				wb.EDDExists = true
				wb.EDDMod = info.ModTime()
			}
		}

		workbookMap[base] = wb
	}

	// Scan XML directory for XML files that might not have Excel counterparts
	xmlFiles, err := s.collectFiles(s.xmlDir, ".xml")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan XML directory: %w", err)
	}

	for _, xmlPath := range xmlFiles {
		relPath, _ := filepath.Rel(s.xmlDir, xmlPath)

		// Determine base name (strip _dt.xml or _edd.xml)
		isDT := strings.HasSuffix(relPath, "_dt.xml")
		isEDD := strings.HasSuffix(relPath, "_edd.xml")
		var base string
		if isDT {
			base = strings.TrimSuffix(relPath, "_dt.xml")
		} else if isEDD {
			base = strings.TrimSuffix(relPath, "_edd.xml")
		} else {
			// Not a standard naming convention, skip
			continue
		}

		// For single-artifact convention, the workbook map key is base+"_dt" or base+"_edd"
		// because the Excel file is named with the same suffix.
		singleArtifactKey := base
		if isDT {
			singleArtifactKey = base + "_dt"
		} else {
			singleArtifactKey = base + "_edd"
		}

		// Check if we already have this workbook (single-artifact key first, then mixed key)
		var lookupKey string
		if _, exists := workbookMap[singleArtifactKey]; exists {
			lookupKey = singleArtifactKey
		} else if _, exists := workbookMap[base]; exists {
			lookupKey = base
		}

		if lookupKey != "" {
			// Update XML existence info
			wb := workbookMap[lookupKey]
			if isDT {
				if info, err := os.Stat(xmlPath); err == nil {
					wb.DTExists = true
					wb.DTMod = info.ModTime()
				}
			} else {
				if info, err := os.Stat(xmlPath); err == nil {
					wb.EDDExists = true
					wb.EDDMod = info.ModTime()
				}
			}
			continue
		}

		// Create new workbook entry — XML exists but no Excel counterpart yet.
		// For single-artifact, check if _dt.xlsx / _edd.xlsx exists on disk.
		var wb *CombinedWorkbook
		singleXlsx := filepath.Join(s.excelDir, base+(map[bool]string{true: "_dt", false: "_edd"}[isDT])+".xlsx")
		if _, err := os.Stat(singleXlsx); err == nil {
			// Single-artifact workbook with suffix-named xlsx
			if isDT {
				wb = &CombinedWorkbook{
					ExcelPath:  singleXlsx,
					DTXMLPath:  xmlPath,
					EDDXMLPath: "",
					RelPath:    base + "_dt",
				}
				if info, err := os.Stat(singleXlsx); err == nil {
					wb.ExcelExists = true
					wb.ExcelMod = info.ModTime()
				}
				if info, err := os.Stat(xmlPath); err == nil {
					wb.DTExists = true
					wb.DTMod = info.ModTime()
				}
				workbookMap[base+"_dt"] = wb
			} else {
				wb = &CombinedWorkbook{
					ExcelPath:  singleXlsx,
					DTXMLPath:  "",
					EDDXMLPath: xmlPath,
					RelPath:    base + "_edd",
				}
				if info, err := os.Stat(singleXlsx); err == nil {
					wb.ExcelExists = true
					wb.ExcelMod = info.ModTime()
				}
				if info, err := os.Stat(xmlPath); err == nil {
					wb.EDDExists = true
					wb.EDDMod = info.ModTime()
				}
				workbookMap[base+"_edd"] = wb
			}
			continue
		}

		// Fall back: mixed-artifact workbook where only XML exists
		wb = &CombinedWorkbook{
			ExcelPath:  filepath.Join(s.excelDir, base+".xlsx"),
			DTXMLPath:  filepath.Join(s.xmlDir, base+"_dt.xml"),
			EDDXMLPath: filepath.Join(s.xmlDir, base+"_edd.xml"),
			RelPath:    base,
		}

		// Check files
		if info, err := os.Stat(wb.ExcelPath); err == nil {
			wb.ExcelExists = true
			wb.ExcelMod = info.ModTime()
		}
		if info, err := os.Stat(wb.DTXMLPath); err == nil {
			wb.DTExists = true
			wb.DTMod = info.ModTime()
		}
		if info, err := os.Stat(wb.EDDXMLPath); err == nil {
			wb.EDDExists = true
			wb.EDDMod = info.ModTime()
		}

		workbookMap[base] = wb
	}

	// Convert map to slice, in a fixed order.
	//
	// Go randomizes map iteration, so this handed back the workbooks in a
	// different order every run. That is invisible while each workbook owns
	// its own output, and decides the result when two of them do not: the
	// last import wins, so the file's content depended on the run and
	// `dtrules verify` gave 1, 1, 2, 0 and 2 findings on five passes over an
	// unchanged TaxReturn (#1089).
	//
	// A list of things to process in turn is a list.
	bases := make([]string, 0, len(workbookMap))
	for base := range workbookMap {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	workbooks := make([]CombinedWorkbook, 0, len(bases))
	for _, base := range bases {
		workbooks = append(workbooks, *workbookMap[base])
	}

	if err := assertNoOutputCollision(workbooks); err != nil {
		return nil, err
	}
	return workbooks, nil
}

// assertNoOutputCollision refuses a workbook set in which two workbooks write
// the same XML file.
//
// Sorting above makes the outcome repeatable; it does not make it right. When
// TaxReturn.xlsx carries an EDD sheet and TaxReturn_edd.xlsx exists beside it,
// both produce xml/TaxReturn_edd.xml and one of them is silently discarded --
// so the committed EDD is whichever the ordering happened to favour, which
// nobody chose.
//
// Naming both and stopping is the useful answer. Picking a winner by rule
// would make the behaviour predictable and leave the project ambiguous.
func assertNoOutputCollision(workbooks []CombinedWorkbook) error {
	claimed := map[string]string{} // xml output -> the workbook claiming it
	for _, wb := range workbooks {
		for _, out := range []string{wb.DTXMLPath, wb.EDDXMLPath} {
			if out == "" {
				continue
			}
			if first, taken := claimed[out]; taken {
				return fmt.Errorf(
					"two workbooks produce %s: %s and %s\n"+
						"  One of them would be discarded, and which one depends on "+
						"ordering. Remove or rename one -- a mixed workbook already "+
						"carries its own EDD sheet, so a separate *_edd.xlsx beside it "+
						"is redundant",
					filepath.Base(out), filepath.Base(first), filepath.Base(wb.ExcelPath))
			}
			claimed[out] = wb.ExcelPath
		}
	}
	return nil
}

// determineWorkbookDirection determines the sync direction for a combined workbook.
func (s *Syncer) determineWorkbookDirection(wb *CombinedWorkbook) SyncDirection {
	// If Excel doesn't exist but XML does, export to Excel
	if !wb.ExcelExists && (wb.DTExists || wb.EDDExists) {
		return XMLToExcel
	}

	// If Excel exists but no XML, import from Excel
	if wb.ExcelExists && !wb.DTExists && !wb.EDDExists {
		return ExcelToXML
	}

	// If nothing exists, no sync needed
	if !wb.ExcelExists && !wb.DTExists && !wb.EDDExists {
		return NoSync
	}

	// Both exist - compare timestamps
	// Use the newest XML file for comparison
	var newestXMLMod time.Time
	if wb.DTExists && wb.DTMod.After(newestXMLMod) {
		newestXMLMod = wb.DTMod
	}
	if wb.EDDExists && wb.EDDMod.After(newestXMLMod) {
		newestXMLMod = wb.EDDMod
	}

	diff := wb.ExcelMod.Sub(newestXMLMod)

	// Within grace period = considered equal (no sync needed)
	if diff > -s.options.GracePeriod && diff < s.options.GracePeriod {
		return NoSync
	}

	// Excel is newer
	if diff > 0 {
		return ExcelToXML
	}

	// XML is newer
	return XMLToExcel
}

// CheckSyncCombined determines sync direction for combined workbooks.
// This should be used when UseCombinedWorkbooks is true.
func (s *Syncer) CheckSyncCombined() (SyncDirection, []CombinedWorkbook, error) {
	workbooks, err := s.collectCombinedWorkbooks()
	if err != nil {
		return NoSync, nil, fmt.Errorf("failed to collect workbooks: %w", err)
	}

	if len(workbooks) == 0 {
		return NoSync, nil, nil
	}

	// Determine direction for each workbook
	for i := range workbooks {
		workbooks[i].Direction = s.determineWorkbookDirection(&workbooks[i])
	}

	// Determine overall direction
	hasExcelToXML := false
	hasXMLToExcel := false

	for _, wb := range workbooks {
		switch wb.Direction {
		case ExcelToXML:
			hasExcelToXML = true
		case XMLToExcel:
			hasXMLToExcel = true
		}
	}

	// If mixed directions, treat as conflict
	if hasExcelToXML && hasXMLToExcel {
		return Conflict, workbooks, nil
	}

	if hasExcelToXML {
		return ExcelToXML, workbooks, nil
	}
	if hasXMLToExcel {
		return XMLToExcel, workbooks, nil
	}

	return NoSync, workbooks, nil
}

// SyncCombinedExcelToXML imports all newer Excel workbooks to XML files.
func (s *Syncer) SyncCombinedExcelToXML() error {
	_, workbooks, err := s.CheckSyncCombined()
	if err != nil {
		return err
	}

	for _, wb := range workbooks {
		if wb.Direction != ExcelToXML {
			continue
		}

		if err := s.importCombinedWorkbook(&wb); err != nil {
			return fmt.Errorf("failed to import %s: %w", wb.ExcelPath, err)
		}
	}

	return nil
}

// SyncCombinedXMLToExcel exports all XML files to combined Excel workbooks.
func (s *Syncer) SyncCombinedXMLToExcel() error {
	_, workbooks, err := s.CheckSyncCombined()
	if err != nil {
		return err
	}

	for _, wb := range workbooks {
		if wb.Direction != XMLToExcel && wb.Direction != NoSync {
			continue
		}

		// Skip if no XML files exist
		if !wb.DTExists && !wb.EDDExists {
			continue
		}

		if err := s.exportCombinedWorkbook(&wb); err != nil {
			return fmt.Errorf("failed to export to %s: %w", wb.ExcelPath, err)
		}
	}

	return nil
}

// importCombinedWorkbook imports a combined Excel workbook to XML files.
func (s *Syncer) importCombinedWorkbook(wb *CombinedWorkbook) error {
	if s.workbookImporter != nil {
		// Use the combined workbook importer
		// Ensure parent directory exists
		xmlDir := wb.outputDir(s.xmlDir)
		if err := os.MkdirAll(xmlDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if _, _, err := s.workbookImporter.ImportWorkbook(wb.ExcelPath, xmlDir); err != nil {
			return err
		}
		s.recordSynced(wb)
		return nil
	}

	// Fallback to separate importers if workbook importer not available
	if s.importer == nil {
		return fmt.Errorf("no importer configured (stub: import %s)", wb.ExcelPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(wb.outputDir(s.xmlDir), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Import DT if the sheet exists (we can't check without workbook importer)
	if err := s.importer.ImportDecisionTable(wb.ExcelPath, wb.DTXMLPath); err != nil {
		// Log but don't fail - the workbook might not have a DT sheet
		if s.options.Verbose {
			fmt.Printf("Note: could not import DT from %s: %v\n", wb.ExcelPath, err)
		}
	}

	// Import EDD if the sheet exists
	if err := s.importer.ImportEDD(wb.ExcelPath, wb.EDDXMLPath); err != nil {
		// Log but don't fail - the workbook might not have an EDD sheet
		if s.options.Verbose {
			fmt.Printf("Note: could not import EDD from %s: %v\n", wb.ExcelPath, err)
		}
	}

	s.recordSynced(wb)
	return nil
}

// recordSynced marks a workbook and its XML as agreeing as of now.
//
// The manifest used to be written only when exporting, so an import left the
// old export timestamp in place. The authoring guard compares that timestamp
// against the workbook's mtime and refuses to write when the workbook is
// newer -- advising "Import Excel to XML first, then re-apply your changes".
// Doing exactly that changed nothing, because the import did not touch the
// manifest, so the guard blocked forever on any project whose workbooks were
// merely touched (a fresh clone stamps every file with the checkout time).
// The tool's own instructions have to work (#1000).
//
// A failure here is not worth failing the import over -- the files are
// already written -- so it is reported only under Verbose, matching the
// export path.
func (s *Syncer) recordSynced(wb *CombinedWorkbook) {
	m, err := s.GetManifest()
	if err != nil {
		if s.options.Verbose {
			fmt.Printf("Warning: could not load manifest to record import: %v\n", err)
		}
		return
	}
	var xmlFiles []string
	for _, p := range []string{wb.DTXMLPath, wb.EDDXMLPath} {
		if p != "" {
			xmlFiles = append(xmlFiles, p)
		}
	}
	if err := m.RecordExport(wb.ExcelPath, xmlFiles); err != nil && s.options.Verbose {
		fmt.Printf("Warning: failed to update manifest after import: %v\n", err)
	}
}

// exportCombinedWorkbook exports XML files to a combined Excel workbook.
// This function enforces Excel as source of truth by checking the manifest
// before exporting. If Excel was modified since the last export, the export
// is rejected to prevent overwriting user changes.
func (s *Syncer) exportCombinedWorkbook(wb *CombinedWorkbook) error {
	if s.exporter == nil {
		return fmt.Errorf("Excel exporter not configured (stub: export to %s)", wb.ExcelPath)
	}

	// ENFORCEMENT: Check if Excel was modified since last export (skip with prefer-xml)
	m, err := s.GetManifest()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if s.options.ConflictResolution != "prefer-xml" {
		if err := m.ExportGuard(wb.ExcelPath); err != nil {
			return err // Returns ExcelModifiedError if user modified Excel
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(wb.ExcelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Try to use combined exporter if available (preferred for combined workbooks)
	if combined, ok := s.exporter.(CombinedExporter); ok {
		dtPath := ""
		eddPath := ""
		if wb.DTExists {
			dtPath = wb.DTXMLPath
		}
		if wb.EDDExists {
			eddPath = wb.EDDXMLPath
		}
		if err := combined.ExportCombined(dtPath, eddPath, wb.ExcelPath); err != nil {
			return fmt.Errorf("failed to export combined workbook: %w", err)
		}
	} else {
		// Fall back to separate exports (note: EDD will overwrite DT)
		// Export DT if it exists
		if wb.DTExists {
			if err := s.exporter.ExportDecisionTable(wb.DTXMLPath, wb.ExcelPath); err != nil {
				return fmt.Errorf("failed to export DT: %w", err)
			}
		}

		// Export EDD if it exists
		if wb.EDDExists {
			if err := s.exporter.ExportEDD(wb.EDDXMLPath, wb.ExcelPath); err != nil {
				return fmt.Errorf("failed to export EDD: %w", err)
			}
		}
	}

	// Record successful export in manifest
	var xmlFiles []string
	if wb.DTExists {
		xmlFiles = append(xmlFiles, wb.DTXMLPath)
	}
	if wb.EDDExists {
		xmlFiles = append(xmlFiles, wb.EDDXMLPath)
	}

	if err := m.RecordExport(wb.ExcelPath, xmlFiles); err != nil {
		// Log warning but don't fail the export
		if s.options.Verbose {
			fmt.Printf("Warning: failed to update manifest: %v\n", err)
		}
	}

	return nil
}

// Utility functions for testing and inspection

// GetXMLDir returns the XML directory path.
func (s *Syncer) GetXMLDir() string {
	return s.xmlDir
}

// GetExcelDir returns the Excel directory path.
func (s *Syncer) GetExcelDir() string {
	return s.excelDir
}

// GetOptions returns the sync options.
func (s *Syncer) GetOptions() *SyncOptions {
	return s.options
}

// XMLToExcelPath converts an XML path to its corresponding Excel path.
// This is an exported standalone function for use outside the Syncer.
//
// In combined workbook mode (when xmlPath ends with _dt.xml or _edd.xml):
//
//	xml/states/CO_dt.xml -> excel/states/CO.xlsx
//	xml/states/CO_edd.xml -> excel/states/CO.xlsx
//
// In legacy mode:
//
//	xml/TaxReturn_dt.xml -> excel/TaxReturn_dt.xlsx
func XMLToExcelPath(xmlPath, xmlDir, excelDir string) string {
	relPath, err := filepath.Rel(xmlDir, xmlPath)
	if err != nil {
		relPath = filepath.Base(xmlPath)
	}

	// Check if this is a combined workbook file (ends with _dt.xml or _edd.xml)
	if strings.HasSuffix(relPath, "_dt.xml") || strings.HasSuffix(relPath, "_edd.xml") {
		// Combined workbook mode: strip _dt or _edd suffix
		base := strings.TrimSuffix(relPath, "_dt.xml")
		base = strings.TrimSuffix(base, "_edd.xml")
		return filepath.Join(excelDir, base+".xlsx")
	}

	// Legacy mode: direct 1:1 mapping
	relPath = strings.TrimSuffix(relPath, ".xml") + ".xlsx"
	return filepath.Join(excelDir, relPath)
}

// ExcelToXMLPaths converts an Excel path to its corresponding XML paths.
// One Excel file may map to multiple XML files in combined workbook mode:
//
//	excel/states/CO.xlsx -> [xml/states/CO_dt.xml, xml/states/CO_edd.xml]
//
// The caller should check if each returned path actually exists.
func ExcelToXMLPaths(excelPath, excelDir, xmlDir string) []string {
	relPath, err := filepath.Rel(excelDir, excelPath)
	if err != nil {
		relPath = filepath.Base(excelPath)
	}

	base := strings.TrimSuffix(relPath, ".xlsx")

	// Always return both possible paths in combined workbook mode
	return []string{
		filepath.Join(xmlDir, base+"_dt.xml"),
		filepath.Join(xmlDir, base+"_edd.xml"),
	}
}

// GetRelativeBase extracts the relative base path from an XML or Excel file.
// This strips the _dt, _edd suffixes and file extension.
//
//	xml/states/CO_dt.xml -> states/CO
//	excel/states/CO.xlsx -> states/CO
func GetRelativeBase(filePath, baseDir string) string {
	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}

	// Remove extension
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)

	// Remove _dt or _edd suffix if present
	base = strings.TrimSuffix(base, "_dt")
	base = strings.TrimSuffix(base, "_edd")

	return base
}

// SetForceDirection pins the sync direction, overriding the per-workbook
// timestamp detection.
//
// `dtrules sync import` and `dtrules sync export` name a direction in the
// command itself, but both called SyncAll and let detection decide, so each
// silently did nothing whenever the timestamps disagreed with the request --
// "No files needed export (Excel is up to date)", exit 0. `--force` did not
// help: it bypasses the pending-user-edits check, not direction detection
// (#1069). This is the same defect #1051 fixed for `build --from-excel`.
func (s *Syncer) SetForceDirection(d SyncDirection) {
	s.options.ForceDirection = d
}
