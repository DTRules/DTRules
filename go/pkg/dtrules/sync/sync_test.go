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
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSyncDirection tests the SyncDirection type.
func TestSyncDirection(t *testing.T) {
	tests := []struct {
		dir      SyncDirection
		expected string
	}{
		{NoSync, "NoSync"},
		{ExcelToXML, "ExcelToXML"},
		{XMLToExcel, "XMLToExcel"},
		{Conflict, "Conflict"},
		{SyncDirection(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.dir.String(); got != tt.expected {
			t.Errorf("SyncDirection(%d).String() = %q, want %q", tt.dir, got, tt.expected)
		}
	}
}

// TestNewSyncer tests syncer creation.
func TestNewSyncer(t *testing.T) {
	syncer := NewSyncer("/xml", "/excel")

	if syncer.GetXMLDir() != "/xml" {
		t.Errorf("GetXMLDir() = %q, want %q", syncer.GetXMLDir(), "/xml")
	}
	if syncer.GetExcelDir() != "/excel" {
		t.Errorf("GetExcelDir() = %q, want %q", syncer.GetExcelDir(), "/excel")
	}
	if syncer.GetOptions() == nil {
		t.Error("GetOptions() returned nil")
	}
}

// TestNewSyncerWithOptions tests syncer creation with custom options.
func TestNewSyncerWithOptions(t *testing.T) {
	opts := &SyncOptions{
		ConflictResolution: "prefer-excel",
		DryRun:             true,
		GracePeriod:        5 * time.Second,
	}

	syncer := NewSyncerWithOptions("/xml", "/excel", opts)

	if syncer.GetOptions().ConflictResolution != "prefer-excel" {
		t.Errorf("ConflictResolution = %q, want %q",
			syncer.GetOptions().ConflictResolution, "prefer-excel")
	}
	if !syncer.GetOptions().DryRun {
		t.Error("DryRun should be true")
	}
}

// TestNewSyncerWithNilOptions tests syncer creation with nil options.
func TestNewSyncerWithNilOptions(t *testing.T) {
	syncer := NewSyncerWithOptions("/xml", "/excel", nil)

	if syncer.GetOptions() == nil {
		t.Error("Options should not be nil")
	}
	if syncer.GetOptions().ConflictResolution != "error" {
		t.Errorf("Default ConflictResolution = %q, want %q",
			syncer.GetOptions().ConflictResolution, "error")
	}
}

// TestDefaultOptions tests default options.
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.ConflictResolution != "error" {
		t.Errorf("ConflictResolution = %q, want %q", opts.ConflictResolution, "error")
	}
	if opts.DryRun {
		t.Error("DryRun should be false by default")
	}
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}
	if opts.GracePeriod != 2*time.Second {
		t.Errorf("GracePeriod = %v, want %v", opts.GracePeriod, 2*time.Second)
	}
}

// TestIsEDDFile tests EDD file detection.
func TestIsEDDFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"TaxReturn_edd.xml", true},
		{"TaxReturn_edd.xlsx", true},
		{"edd_TaxReturn.xml", true},
		{"TaxReturn_dt.xml", false},
		{"TaxReturn_dt.xlsx", false},
		{"other.xml", false},
	}

	for _, tt := range tests {
		if got := isEDDFile(tt.path); got != tt.expected {
			t.Errorf("isEDDFile(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

// TestXMLToExcelPath tests path conversion.
func TestXMLToExcelPath(t *testing.T) {
	syncer := NewSyncer("/project/xml", "/project/excel")

	tests := []struct {
		xmlPath   string
		wantExcel string
	}{
		{"/project/xml/TaxReturn_dt.xml", "/project/excel/TaxReturn_dt.xlsx"},
		{"/project/xml/states/CO_dt.xml", "/project/excel/states/CO_dt.xlsx"},
		{"/project/xml/TaxReturn_edd.xml", "/project/excel/TaxReturn_edd.xlsx"},
	}

	for _, tt := range tests {
		got := syncer.xmlToExcelPath(tt.xmlPath)
		if got != tt.wantExcel {
			t.Errorf("xmlToExcelPath(%q) = %q, want %q", tt.xmlPath, got, tt.wantExcel)
		}
	}
}

// TestExcelToXMLPath tests path conversion.
func TestExcelToXMLPath(t *testing.T) {
	syncer := NewSyncer("/project/xml", "/project/excel")

	tests := []struct {
		excelPath string
		wantXML   string
	}{
		{"/project/excel/TaxReturn_dt.xlsx", "/project/xml/TaxReturn_dt.xml"},
		{"/project/excel/states/CO_dt.xlsx", "/project/xml/states/CO_dt.xml"},
		{"/project/excel/TaxReturn_edd.xlsx", "/project/xml/TaxReturn_edd.xml"},
	}

	for _, tt := range tests {
		got := syncer.excelToXMLPath(tt.excelPath)
		if got != tt.wantXML {
			t.Errorf("excelToXMLPath(%q) = %q, want %q", tt.excelPath, got, tt.wantXML)
		}
	}
}

// TestDetermineDirection tests direction determination.
func TestDetermineDirection(t *testing.T) {
	syncer := NewSyncer("/xml", "/excel")
	now := time.Now()

	tests := []struct {
		name      string
		pair      FilePair
		wantDir   SyncDirection
	}{
		{
			name: "XML only",
			pair: FilePair{
				XMLExists:   true,
				ExcelExists: false,
			},
			wantDir: XMLToExcel,
		},
		{
			name: "Excel only",
			pair: FilePair{
				XMLExists:   false,
				ExcelExists: true,
			},
			wantDir: ExcelToXML,
		},
		{
			name: "Neither exists",
			pair: FilePair{
				XMLExists:   false,
				ExcelExists: false,
			},
			wantDir: NoSync,
		},
		{
			name: "Excel newer",
			pair: FilePair{
				XMLExists:   true,
				ExcelExists: true,
				XMLMod:      now.Add(-1 * time.Hour),
				ExcelMod:    now,
			},
			wantDir: ExcelToXML,
		},
		{
			name: "XML newer",
			pair: FilePair{
				XMLExists:   true,
				ExcelExists: true,
				XMLMod:      now,
				ExcelMod:    now.Add(-1 * time.Hour),
			},
			wantDir: XMLToExcel,
		},
		{
			name: "Within grace period",
			pair: FilePair{
				XMLExists:   true,
				ExcelExists: true,
				XMLMod:      now,
				ExcelMod:    now.Add(1 * time.Second),
			},
			wantDir: NoSync,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncer.determineDirection(&tt.pair)
			if got != tt.wantDir {
				t.Errorf("determineDirection() = %v, want %v", got, tt.wantDir)
			}
		})
	}
}

// TestDetermineOverallDirection tests overall direction determination.
func TestDetermineOverallDirection(t *testing.T) {
	syncer := NewSyncer("/xml", "/excel")

	tests := []struct {
		name    string
		pairs   []FilePair
		wantDir SyncDirection
	}{
		{
			name:    "Empty",
			pairs:   []FilePair{},
			wantDir: NoSync,
		},
		{
			name: "All NoSync",
			pairs: []FilePair{
				{Direction: NoSync},
				{Direction: NoSync},
			},
			wantDir: NoSync,
		},
		{
			name: "All ExcelToXML",
			pairs: []FilePair{
				{Direction: ExcelToXML},
				{Direction: ExcelToXML},
			},
			wantDir: ExcelToXML,
		},
		{
			name: "All XMLToExcel",
			pairs: []FilePair{
				{Direction: XMLToExcel},
				{Direction: XMLToExcel},
			},
			wantDir: XMLToExcel,
		},
		{
			name: "Mixed directions",
			pairs: []FilePair{
				{Direction: ExcelToXML},
				{Direction: XMLToExcel},
			},
			wantDir: Conflict,
		},
		{
			name: "Has conflict",
			pairs: []FilePair{
				{Direction: ExcelToXML},
				{Direction: Conflict},
			},
			wantDir: Conflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncer.determineOverallDirection(tt.pairs)
			if got != tt.wantDir {
				t.Errorf("determineOverallDirection() = %v, want %v", got, tt.wantDir)
			}
		})
	}
}

// TestFilePairCreation tests creating file pairs with real files.
func TestFilePairCreation(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	xmlFile := filepath.Join(xmlDir, "test.xml")
	if err := os.WriteFile(xmlFile, []byte("<test/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	excelPath := syncer.xmlToExcelPath(xmlFile)

	pair := syncer.createFilePair(xmlFile, excelPath)

	if !pair.XMLExists {
		t.Error("XMLExists should be true")
	}
	if pair.ExcelExists {
		t.Error("ExcelExists should be false")
	}
	if pair.XMLMod.IsZero() {
		t.Error("XMLMod should not be zero")
	}
}

// TestCheckSyncWithTempFiles tests CheckSync with actual files.
func TestCheckSyncWithTempFiles(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test 1: Empty directories
	syncer := NewSyncer(xmlDir, excelDir)
	direction, pairs, err := syncer.CheckSync()
	if err != nil {
		t.Fatalf("CheckSync failed: %v", err)
	}
	if direction != NoSync {
		t.Errorf("direction = %v, want NoSync", direction)
	}
	if len(pairs) != 0 {
		t.Errorf("len(pairs) = %d, want 0", len(pairs))
	}

	// Test 2: XML file only -> should want XMLToExcel
	xmlFile := filepath.Join(xmlDir, "test_dt.xml")
	if err := os.WriteFile(xmlFile, []byte("<test/>"), 0644); err != nil {
		t.Fatal(err)
	}

	direction, pairs, err = syncer.CheckSync()
	if err != nil {
		t.Fatalf("CheckSync failed: %v", err)
	}
	if direction != XMLToExcel {
		t.Errorf("direction = %v, want XMLToExcel", direction)
	}
	if len(pairs) != 1 {
		t.Errorf("len(pairs) = %d, want 1", len(pairs))
	}

	// Test 3: Both files, Excel newer
	excelFile := filepath.Join(excelDir, "test_dt.xlsx")
	if err := os.WriteFile(excelFile, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}
	// Make Excel file newer
	futureTime := time.Now().Add(1 * time.Hour)
	if err := os.Chtimes(excelFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	direction, pairs, err = syncer.CheckSync()
	if err != nil {
		t.Fatalf("CheckSync failed: %v", err)
	}
	if direction != ExcelToXML {
		t.Errorf("direction = %v, want ExcelToXML", direction)
	}
}

// TestSkipDirectories tests that testfiles and schemas are skipped.
func TestSkipDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")

	// Create directories to skip
	testfilesDir := filepath.Join(xmlDir, "testfiles")
	schemasDir := filepath.Join(xmlDir, "schemas")

	if err := os.MkdirAll(testfilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files in skip directories
	if err := os.WriteFile(filepath.Join(testfilesDir, "test.xml"), []byte("<test/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemasDir, "schema.xml"), []byte("<schema/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a regular file
	if err := os.WriteFile(filepath.Join(xmlDir, "real.xml"), []byte("<real/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, filepath.Join(tmpDir, "excel"))
	files, err := syncer.collectFiles(xmlDir, ".xml")
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should only find the real.xml file
	if len(files) != 1 {
		t.Errorf("len(files) = %d, want 1", len(files))
	}
	if len(files) > 0 && filepath.Base(files[0]) != "real.xml" {
		t.Errorf("found file %q, want real.xml", files[0])
	}
}

// TestSkipTemplateFiles tests that template files are skipped.
func TestSkipTemplateFiles(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create template files (should be skipped)
	templates := []string{
		"TEMPLATE_dt.xml",
		"template_edd.xml",
		"CO_TEMPLATE.xml",
	}
	for _, name := range templates {
		if err := os.WriteFile(filepath.Join(xmlDir, name), []byte("<t/>"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a regular file
	if err := os.WriteFile(filepath.Join(xmlDir, "real.xml"), []byte("<real/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, filepath.Join(tmpDir, "excel"))
	files, err := syncer.collectFiles(xmlDir, ".xml")
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should only find the real.xml file
	if len(files) != 1 {
		t.Errorf("len(files) = %d, want 1", len(files))
	}
}

// TestSkipMapFiles tests that mapping files are skipped.
func TestSkipMapFiles(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mapping file (should be skipped)
	if err := os.WriteFile(filepath.Join(xmlDir, "TaxReturn_map.xml"), []byte("<map/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a regular file
	if err := os.WriteFile(filepath.Join(xmlDir, "TaxReturn_dt.xml"), []byte("<dt/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, filepath.Join(tmpDir, "excel"))
	files, err := syncer.collectFiles(xmlDir, ".xml")
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should only find the dt file
	if len(files) != 1 {
		t.Errorf("len(files) = %d, want 1", len(files))
	}
	if len(files) > 0 && filepath.Base(files[0]) != "TaxReturn_dt.xml" {
		t.Errorf("found file %q, want TaxReturn_dt.xml", files[0])
	}
}

// mockImporter is a mock implementation of ExcelImporter for testing.
type mockImporter struct {
	importedDTs  []string
	importedEDDs []string
	failOnPath   string
}

func (m *mockImporter) ImportDecisionTable(excelPath, xmlPath string) error {
	if m.failOnPath != "" && excelPath == m.failOnPath {
		return os.ErrNotExist
	}
	m.importedDTs = append(m.importedDTs, excelPath)
	return nil
}

func (m *mockImporter) ImportEDD(excelPath, xmlPath string) error {
	if m.failOnPath != "" && excelPath == m.failOnPath {
		return os.ErrNotExist
	}
	m.importedEDDs = append(m.importedEDDs, excelPath)
	return nil
}

// mockExporter is a mock implementation of ExcelExporter for testing.
type mockExporter struct {
	exportedDTs  []string
	exportedEDDs []string
	failOnPath   string
}

func (m *mockExporter) ExportDecisionTable(xmlPath, excelPath string) error {
	if m.failOnPath != "" && xmlPath == m.failOnPath {
		return os.ErrNotExist
	}
	m.exportedDTs = append(m.exportedDTs, xmlPath)
	return nil
}

func (m *mockExporter) ExportEDD(xmlPath, excelPath string) error {
	if m.failOnPath != "" && xmlPath == m.failOnPath {
		return os.ErrNotExist
	}
	m.exportedEDDs = append(m.exportedEDDs, xmlPath)
	return nil
}

// TestSyncWithMockImporter tests sync operations with a mock importer.
func TestSyncWithMockImporter(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Excel file only (should trigger ExcelToXML)
	excelFile := filepath.Join(excelDir, "test_dt.xlsx")
	if err := os.WriteFile(excelFile, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	importer := &mockImporter{}
	syncer.SetImporter(importer)

	// Sync should import Excel to XML
	if err := syncer.SyncExcelToXML(); err != nil {
		t.Fatalf("SyncExcelToXML failed: %v", err)
	}

	if len(importer.importedDTs) != 1 {
		t.Errorf("importedDTs = %d, want 1", len(importer.importedDTs))
	}
}

// TestSyncWithMockExporter tests sync operations with a mock exporter.
func TestSyncWithMockExporter(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create XML file only (should trigger XMLToExcel)
	xmlFile := filepath.Join(xmlDir, "test_dt.xml")
	if err := os.WriteFile(xmlFile, []byte("<test/>"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	exporter := &mockExporter{}
	syncer.SetExporter(exporter)

	// Sync should export XML to Excel
	if err := syncer.SyncXMLToExcel(); err != nil {
		t.Fatalf("SyncXMLToExcel failed: %v", err)
	}

	if len(exporter.exportedDTs) != 1 {
		t.Errorf("exportedDTs = %d, want 1", len(exporter.exportedDTs))
	}
}

// TestSyncAllDryRun tests SyncAll in dry run mode.
func TestSyncAllDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files
	xmlFile := filepath.Join(xmlDir, "test_dt.xml")
	if err := os.WriteFile(xmlFile, []byte("<test/>"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &SyncOptions{
		ConflictResolution: "error",
		DryRun:             true,
		GracePeriod:        2 * time.Second,
	}
	syncer := NewSyncerWithOptions(xmlDir, excelDir, opts)

	result, err := syncer.SyncAll()
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// In dry run, should report count but not actually sync
	if result.XMLToExcelCount != 1 {
		t.Errorf("XMLToExcelCount = %d, want 1", result.XMLToExcelCount)
	}
}

// TestConflictResolution tests different conflict resolution strategies.
func TestConflictResolution(t *testing.T) {
	// Create temp directories for tests that need real paths
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		resolution string
		wantErr    bool
	}{
		{"error", true},
		{"prefer-excel", false},
		{"prefer-xml", false},
		{"skip", false},
	}

	for _, tt := range tests {
		t.Run(tt.resolution, func(t *testing.T) {
			opts := &SyncOptions{
				ConflictResolution: tt.resolution,
				GracePeriod:        2 * time.Second,
			}
			syncer := NewSyncerWithOptions(xmlDir, excelDir, opts)

			importer := &mockImporter{}
			exporter := &mockExporter{}
			syncer.SetImporter(importer)
			syncer.SetExporter(exporter)

			pair := &FilePair{
				XMLPath:     filepath.Join(xmlDir, "test_dt.xml"),
				ExcelPath:   filepath.Join(excelDir, "test_dt.xlsx"),
				XMLExists:   true,
				ExcelExists: true,
				Direction:   Conflict,
			}

			err := syncer.handleConflict(pair)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestRecursiveDirectorySync tests sync with subdirectories.
func TestRecursiveDirectorySync(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	// Create directory structure
	statesXMLDir := filepath.Join(xmlDir, "states")
	statesExcelDir := filepath.Join(excelDir, "states")

	if err := os.MkdirAll(statesXMLDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(statesExcelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files at root level
	if err := os.WriteFile(filepath.Join(xmlDir, "TaxReturn_dt.xml"), []byte("<dt/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "TaxReturn_edd.xml"), []byte("<edd/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create files in subdirectory
	if err := os.WriteFile(filepath.Join(statesXMLDir, "CO_dt.xml"), []byte("<co_dt/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(statesXMLDir, "CO_edd.xml"), []byte("<co_edd/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(statesXMLDir, "CA_dt.xml"), []byte("<ca_dt/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test file collection
	syncer := NewSyncer(xmlDir, excelDir)
	files, err := syncer.collectFiles(xmlDir, ".xml")
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should find 5 files (2 root + 3 in states)
	if len(files) != 5 {
		t.Errorf("len(files) = %d, want 5", len(files))
	}

	// Verify subdirectory files are included
	foundCODT := false
	foundCOEDD := false
	foundCADT := false
	for _, f := range files {
		if filepath.Base(f) == "CO_dt.xml" {
			foundCODT = true
		}
		if filepath.Base(f) == "CO_edd.xml" {
			foundCOEDD = true
		}
		if filepath.Base(f) == "CA_dt.xml" {
			foundCADT = true
		}
	}

	if !foundCODT || !foundCOEDD || !foundCADT {
		t.Errorf("missing subdirectory files: CO_dt=%v, CO_edd=%v, CA_dt=%v",
			foundCODT, foundCOEDD, foundCADT)
	}
}

// TestCombinedWorkbookSync tests sync with combined workbooks.
func TestCombinedWorkbookSync(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	statesXMLDir := filepath.Join(xmlDir, "states")
	statesExcelDir := filepath.Join(excelDir, "states")

	if err := os.MkdirAll(statesXMLDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(statesExcelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Excel file only (should trigger ExcelToXML)
	excelFile := filepath.Join(statesExcelDir, "CO.xlsx")
	if err := os.WriteFile(excelFile, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	syncer.SetUseCombinedWorkbooks(true)

	// Check sync should detect the workbook
	direction, workbooks, err := syncer.CheckSyncCombined()
	if err != nil {
		t.Fatalf("CheckSyncCombined failed: %v", err)
	}

	if direction != ExcelToXML {
		t.Errorf("direction = %v, want ExcelToXML", direction)
	}

	if len(workbooks) != 1 {
		t.Fatalf("len(workbooks) = %d, want 1", len(workbooks))
	}

	wb := workbooks[0]
	if !wb.ExcelExists {
		t.Error("ExcelExists should be true")
	}
	if wb.DTExists {
		t.Error("DTExists should be false")
	}
	if wb.EDDExists {
		t.Error("EDDExists should be false")
	}
	if wb.Direction != ExcelToXML {
		t.Errorf("Direction = %v, want ExcelToXML", wb.Direction)
	}
}

// TestSubdirectoryPathConversion tests path conversion with subdirectories.
func TestSubdirectoryPathConversion(t *testing.T) {
	syncer := NewSyncer("/project/xml", "/project/excel")
	syncer.SetUseCombinedWorkbooks(true)

	tests := []struct {
		name      string
		xmlPath   string
		wantExcel string
	}{
		{
			name:      "Root DT file",
			xmlPath:   "/project/xml/TaxReturn_dt.xml",
			wantExcel: "/project/excel/TaxReturn.xlsx",
		},
		{
			name:      "Root EDD file",
			xmlPath:   "/project/xml/TaxReturn_edd.xml",
			wantExcel: "/project/excel/TaxReturn.xlsx",
		},
		{
			name:      "Subdirectory DT file",
			xmlPath:   "/project/xml/states/CO_dt.xml",
			wantExcel: "/project/excel/states/CO.xlsx",
		},
		{
			name:      "Subdirectory EDD file",
			xmlPath:   "/project/xml/states/CO_edd.xml",
			wantExcel: "/project/excel/states/CO.xlsx",
		},
		{
			name:      "Deep subdirectory",
			xmlPath:   "/project/xml/regions/west/CA_dt.xml",
			wantExcel: "/project/excel/regions/west/CA.xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncer.xmlToExcelPath(tt.xmlPath)
			if got != tt.wantExcel {
				t.Errorf("xmlToExcelPath(%q) = %q, want %q", tt.xmlPath, got, tt.wantExcel)
			}
		})
	}
}

// TestExcelToXMLPaths tests the ExcelToXMLPaths function.
func TestExcelToXMLPaths(t *testing.T) {
	syncer := NewSyncer("/project/xml", "/project/excel")
	syncer.SetUseCombinedWorkbooks(true)

	tests := []struct {
		name       string
		excelPath  string
		wantPaths  []string
	}{
		{
			name:      "Root workbook",
			excelPath: "/project/excel/TaxReturn.xlsx",
			wantPaths: []string{
				"/project/xml/TaxReturn_dt.xml",
				"/project/xml/TaxReturn_edd.xml",
			},
		},
		{
			name:      "Subdirectory workbook",
			excelPath: "/project/excel/states/CO.xlsx",
			wantPaths: []string{
				"/project/xml/states/CO_dt.xml",
				"/project/xml/states/CO_edd.xml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncer.excelToXMLPaths(tt.excelPath)
			if len(got) != len(tt.wantPaths) {
				t.Fatalf("len(paths) = %d, want %d", len(got), len(tt.wantPaths))
			}
			for i, p := range got {
				if p != tt.wantPaths[i] {
					t.Errorf("path[%d] = %q, want %q", i, p, tt.wantPaths[i])
				}
			}
		})
	}
}

// TestExportedPathFunctions tests the exported standalone path functions.
func TestExportedPathFunctions(t *testing.T) {
	t.Run("XMLToExcelPath", func(t *testing.T) {
		tests := []struct {
			xmlPath   string
			xmlDir    string
			excelDir  string
			wantExcel string
		}{
			{
				xmlPath:   "/project/xml/states/CO_dt.xml",
				xmlDir:    "/project/xml",
				excelDir:  "/project/excel",
				wantExcel: "/project/excel/states/CO.xlsx",
			},
			{
				xmlPath:   "/project/xml/states/CO_edd.xml",
				xmlDir:    "/project/xml",
				excelDir:  "/project/excel",
				wantExcel: "/project/excel/states/CO.xlsx",
			},
		}

		for _, tt := range tests {
			got := XMLToExcelPath(tt.xmlPath, tt.xmlDir, tt.excelDir)
			if got != tt.wantExcel {
				t.Errorf("XMLToExcelPath(%q, %q, %q) = %q, want %q",
					tt.xmlPath, tt.xmlDir, tt.excelDir, got, tt.wantExcel)
			}
		}
	})

	t.Run("ExcelToXMLPaths", func(t *testing.T) {
		paths := ExcelToXMLPaths(
			"/project/excel/states/CO.xlsx",
			"/project/excel",
			"/project/xml",
		)

		if len(paths) != 2 {
			t.Fatalf("len(paths) = %d, want 2", len(paths))
		}

		expectedDT := "/project/xml/states/CO_dt.xml"
		expectedEDD := "/project/xml/states/CO_edd.xml"

		if paths[0] != expectedDT {
			t.Errorf("paths[0] = %q, want %q", paths[0], expectedDT)
		}
		if paths[1] != expectedEDD {
			t.Errorf("paths[1] = %q, want %q", paths[1], expectedEDD)
		}
	})

	t.Run("GetRelativeBase", func(t *testing.T) {
		tests := []struct {
			filePath string
			baseDir  string
			want     string
		}{
			{"/project/xml/states/CO_dt.xml", "/project/xml", "states/CO"},
			{"/project/xml/states/CO_edd.xml", "/project/xml", "states/CO"},
			{"/project/excel/states/CO.xlsx", "/project/excel", "states/CO"},
			{"/project/xml/TaxReturn_dt.xml", "/project/xml", "TaxReturn"},
		}

		for _, tt := range tests {
			got := GetRelativeBase(tt.filePath, tt.baseDir)
			if got != tt.want {
				t.Errorf("GetRelativeBase(%q, %q) = %q, want %q",
					tt.filePath, tt.baseDir, got, tt.want)
			}
		}
	})
}

// TestCombinedWorkbookDirection tests direction determination for combined workbooks.
func TestCombinedWorkbookDirection(t *testing.T) {
	syncer := NewSyncer("/xml", "/excel")
	now := time.Now()

	tests := []struct {
		name    string
		wb      CombinedWorkbook
		wantDir SyncDirection
	}{
		{
			name: "Excel only",
			wb: CombinedWorkbook{
				ExcelExists: true,
				DTExists:    false,
				EDDExists:   false,
			},
			wantDir: ExcelToXML,
		},
		{
			name: "XML only (DT)",
			wb: CombinedWorkbook{
				ExcelExists: false,
				DTExists:    true,
				EDDExists:   false,
			},
			wantDir: XMLToExcel,
		},
		{
			name: "XML only (EDD)",
			wb: CombinedWorkbook{
				ExcelExists: false,
				DTExists:    false,
				EDDExists:   true,
			},
			wantDir: XMLToExcel,
		},
		{
			name: "XML only (both)",
			wb: CombinedWorkbook{
				ExcelExists: false,
				DTExists:    true,
				EDDExists:   true,
			},
			wantDir: XMLToExcel,
		},
		{
			name: "Nothing exists",
			wb: CombinedWorkbook{
				ExcelExists: false,
				DTExists:    false,
				EDDExists:   false,
			},
			wantDir: NoSync,
		},
		{
			name: "Excel newer than both XML",
			wb: CombinedWorkbook{
				ExcelExists: true,
				DTExists:    true,
				EDDExists:   true,
				ExcelMod:    now,
				DTMod:       now.Add(-1 * time.Hour),
				EDDMod:      now.Add(-1 * time.Hour),
			},
			wantDir: ExcelToXML,
		},
		{
			name: "DT newer than Excel",
			wb: CombinedWorkbook{
				ExcelExists: true,
				DTExists:    true,
				EDDExists:   true,
				ExcelMod:    now.Add(-1 * time.Hour),
				DTMod:       now,
				EDDMod:      now.Add(-2 * time.Hour),
			},
			wantDir: XMLToExcel,
		},
		{
			name: "EDD newer than Excel",
			wb: CombinedWorkbook{
				ExcelExists: true,
				DTExists:    true,
				EDDExists:   true,
				ExcelMod:    now.Add(-1 * time.Hour),
				DTMod:       now.Add(-2 * time.Hour),
				EDDMod:      now,
			},
			wantDir: XMLToExcel,
		},
		{
			name: "Within grace period",
			wb: CombinedWorkbook{
				ExcelExists: true,
				DTExists:    true,
				EDDExists:   true,
				ExcelMod:    now,
				DTMod:       now.Add(1 * time.Second),
				EDDMod:      now.Add(-1 * time.Second),
			},
			wantDir: NoSync,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncer.determineWorkbookDirection(&tt.wb)
			if got != tt.wantDir {
				t.Errorf("determineWorkbookDirection() = %v, want %v", got, tt.wantDir)
			}
		})
	}
}

// TestCollectCombinedWorkbooks tests workbook collection.
func TestCollectCombinedWorkbooks(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	statesXMLDir := filepath.Join(xmlDir, "states")
	statesExcelDir := filepath.Join(excelDir, "states")

	if err := os.MkdirAll(statesXMLDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(statesExcelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create root-level files
	if err := os.WriteFile(filepath.Join(xmlDir, "TaxReturn_dt.xml"), []byte("<dt/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "TaxReturn_edd.xml"), []byte("<edd/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(excelDir, "TaxReturn.xlsx"), []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create state files
	if err := os.WriteFile(filepath.Join(statesXMLDir, "CO_dt.xml"), []byte("<co_dt/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(statesXMLDir, "CO_edd.xml"), []byte("<co_edd/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(statesExcelDir, "CO.xlsx"), []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Excel-only state (no XML yet)
	if err := os.WriteFile(filepath.Join(statesExcelDir, "CA.xlsx"), []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	syncer.SetUseCombinedWorkbooks(true)

	workbooks, err := syncer.collectCombinedWorkbooks()
	if err != nil {
		t.Fatalf("collectCombinedWorkbooks failed: %v", err)
	}

	// Should find 3 workbooks: TaxReturn, CO, CA
	if len(workbooks) != 3 {
		t.Errorf("len(workbooks) = %d, want 3", len(workbooks))
	}

	// Verify workbook properties
	for _, wb := range workbooks {
		switch wb.RelPath {
		case "TaxReturn":
			if !wb.ExcelExists || !wb.DTExists || !wb.EDDExists {
				t.Errorf("TaxReturn: ExcelExists=%v, DTExists=%v, EDDExists=%v",
					wb.ExcelExists, wb.DTExists, wb.EDDExists)
			}
		case "states/CO":
			if !wb.ExcelExists || !wb.DTExists || !wb.EDDExists {
				t.Errorf("states/CO: ExcelExists=%v, DTExists=%v, EDDExists=%v",
					wb.ExcelExists, wb.DTExists, wb.EDDExists)
			}
		case "states/CA":
			if !wb.ExcelExists || wb.DTExists || wb.EDDExists {
				t.Errorf("states/CA: ExcelExists=%v, DTExists=%v, EDDExists=%v",
					wb.ExcelExists, wb.DTExists, wb.EDDExists)
			}
		}
	}
}

// mockWorkbookImporter is a mock implementation of WorkbookImporter for testing.
type mockWorkbookImporter struct {
	importedWorkbooks []string
	hasDT             bool
	hasEDD            bool
}

func (m *mockWorkbookImporter) ImportWorkbook(excelPath, xmlDir string) (dtPath, eddPath string, err error) {
	m.importedWorkbooks = append(m.importedWorkbooks, excelPath)
	base := filepath.Base(excelPath)
	base = base[:len(base)-5] // Remove .xlsx
	if m.hasDT {
		dtPath = filepath.Join(xmlDir, base+"_dt.xml")
	}
	if m.hasEDD {
		eddPath = filepath.Join(xmlDir, base+"_edd.xml")
	}
	return
}

func (m *mockWorkbookImporter) HasDTSheet(excelPath string) (bool, error) {
	return m.hasDT, nil
}

func (m *mockWorkbookImporter) HasEDDSheet(excelPath string) (bool, error) {
	return m.hasEDD, nil
}

// TestSyncWithWorkbookImporter tests sync with the workbook importer.
func TestSyncWithWorkbookImporter(t *testing.T) {
	tmpDir := t.TempDir()
	xmlDir := filepath.Join(tmpDir, "xml")
	excelDir := filepath.Join(tmpDir, "excel")

	statesExcelDir := filepath.Join(excelDir, "states")
	statesXMLDir := filepath.Join(xmlDir, "states")

	if err := os.MkdirAll(statesExcelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(statesXMLDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Excel file
	excelFile := filepath.Join(statesExcelDir, "CO.xlsx")
	if err := os.WriteFile(excelFile, []byte("excel"), 0644); err != nil {
		t.Fatal(err)
	}

	syncer := NewSyncer(xmlDir, excelDir)
	syncer.SetUseCombinedWorkbooks(true)

	importer := &mockWorkbookImporter{hasDT: true, hasEDD: true}
	syncer.SetWorkbookImporter(importer)

	// Sync should import the workbook
	if err := syncer.SyncCombinedExcelToXML(); err != nil {
		t.Fatalf("SyncCombinedExcelToXML failed: %v", err)
	}

	if len(importer.importedWorkbooks) != 1 {
		t.Errorf("importedWorkbooks = %d, want 1", len(importer.importedWorkbooks))
	}
}

// TestIsDTFile tests the isDTFile function.
func TestIsDTFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"TaxReturn_dt.xml", true},
		{"TaxReturn_dt.xlsx", true},
		{"dt_TaxReturn.xml", true},
		{"TaxReturn_edd.xml", false},
		{"TaxReturn_edd.xlsx", false},
		{"other.xml", false},
	}

	for _, tt := range tests {
		if got := isDTFile(tt.path); got != tt.expected {
			t.Errorf("isDTFile(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}
