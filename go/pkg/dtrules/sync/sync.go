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
//	xml/states/CO_dt.xml -> excel/states/CO.xlsx
//	xml/states/CO_edd.xml -> excel/states/CO.xlsx (same file!)
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
		// Combined workbook mode: strip _dt or _edd suffix
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
		// Combined mode: return both DT and EDD paths
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

	_, pairs, err := s.CheckSync()
	if err != nil {
		return result, err
	}

	result.Pairs = pairs

	for i := range pairs {
		pair := &pairs[i]

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

// importExcelToXML imports a single Excel file to XML.
func (s *Syncer) importExcelToXML(pair *FilePair) error {
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
func (s *Syncer) exportXMLToExcel(pair *FilePair) error {
	if s.exporter == nil {
		return fmt.Errorf("Excel exporter not configured (stub: export %s -> %s)",
			pair.XMLPath, pair.ExcelPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(pair.ExcelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine file type based on filename convention
	if isEDDFile(pair.XMLPath) {
		return s.exporter.ExportEDD(pair.XMLPath, pair.ExcelPath)
	}
	return s.exporter.ExportDecisionTable(pair.XMLPath, pair.ExcelPath)
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

		wb := &CombinedWorkbook{
			ExcelPath:  excelPath,
			DTXMLPath:  filepath.Join(s.xmlDir, base+"_dt.xml"),
			EDDXMLPath: filepath.Join(s.xmlDir, base+"_edd.xml"),
			RelPath:    base,
		}

		// Check Excel file
		if info, err := os.Stat(excelPath); err == nil {
			wb.ExcelExists = true
			wb.ExcelMod = info.ModTime()
		}

		// Check DT XML file
		if info, err := os.Stat(wb.DTXMLPath); err == nil {
			wb.DTExists = true
			wb.DTMod = info.ModTime()
		}

		// Check EDD XML file
		if info, err := os.Stat(wb.EDDXMLPath); err == nil {
			wb.EDDExists = true
			wb.EDDMod = info.ModTime()
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
		var base string
		if strings.HasSuffix(relPath, "_dt.xml") {
			base = strings.TrimSuffix(relPath, "_dt.xml")
		} else if strings.HasSuffix(relPath, "_edd.xml") {
			base = strings.TrimSuffix(relPath, "_edd.xml")
		} else {
			// Not a standard naming convention, skip
			continue
		}

		// Check if we already have this workbook
		if _, exists := workbookMap[base]; exists {
			// Update XML existence info
			wb := workbookMap[base]
			if strings.HasSuffix(relPath, "_dt.xml") {
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

		// Create new workbook entry
		wb := &CombinedWorkbook{
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

	// Convert map to slice
	var workbooks []CombinedWorkbook
	for _, wb := range workbookMap {
		workbooks = append(workbooks, *wb)
	}

	return workbooks, nil
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
		xmlDir := filepath.Dir(wb.DTXMLPath)
		if err := os.MkdirAll(xmlDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		_, _, err := s.workbookImporter.ImportWorkbook(wb.ExcelPath, xmlDir)
		return err
	}

	// Fallback to separate importers if workbook importer not available
	if s.importer == nil {
		return fmt.Errorf("no importer configured (stub: import %s)", wb.ExcelPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(wb.DTXMLPath), 0755); err != nil {
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

	return nil
}

// exportCombinedWorkbook exports XML files to a combined Excel workbook.
func (s *Syncer) exportCombinedWorkbook(wb *CombinedWorkbook) error {
	if s.exporter == nil {
		return fmt.Errorf("Excel exporter not configured (stub: export to %s)", wb.ExcelPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(wb.ExcelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

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
