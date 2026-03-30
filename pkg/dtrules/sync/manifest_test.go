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

func TestNewManifest(t *testing.T) {
	m := NewManifest()

	if m.Version != ManifestVersion {
		t.Errorf("Version = %d, want %d", m.Version, ManifestVersion)
	}
	if m.Files == nil {
		t.Error("Files map should be initialized")
	}
}

func TestManifestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, ".sync-manifest.json")

	// Create and save manifest
	m := NewManifest()
	m.SetPath(manifestPath)
	m.Files["excel/test.xlsx"] = &FileEntry{
		LastExportTime:       time.Now(),
		ExcelModTimeAtExport: time.Now(),
		XMLFiles:             []string{"xml/test_dt.xml", "xml/test_edd.xml"},
	}

	if err := m.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load manifest
	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if loaded.Version != ManifestVersion {
		t.Errorf("Loaded Version = %d, want %d", loaded.Version, ManifestVersion)
	}

	entry := loaded.Files["excel/test.xlsx"]
	if entry == nil {
		t.Fatal("Expected entry for excel/test.xlsx")
	}

	if len(entry.XMLFiles) != 2 {
		t.Errorf("XMLFiles length = %d, want 2", len(entry.XMLFiles))
	}
}

func TestManifestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "nonexistent", ".sync-manifest.json")

	m, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest should not fail for non-existent file: %v", err)
	}

	if m == nil {
		t.Fatal("Should return empty manifest, not nil")
	}

	if len(m.Files) != 0 {
		t.Errorf("New manifest should have no files, got %d", len(m.Files))
	}
}

func TestCheckExcelModified_NotTracked(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	excelPath := filepath.Join(tmpDir, "test.xlsx")

	// File not in manifest - should allow export
	modified, entry, err := m.CheckExcelModified(excelPath)
	if err != nil {
		t.Fatalf("CheckExcelModified failed: %v", err)
	}

	if modified {
		t.Error("Untracked file should not be considered modified")
	}
	if entry != nil {
		t.Error("Untracked file should have nil entry")
	}
}

func TestCheckExcelModified_NotModified(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "test.xlsx")

	// Create Excel file
	if err := os.WriteFile(excelPath, []byte("excel content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create manifest and record export
	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	if err := m.RecordExport(excelPath, []string{"test_dt.xml"}); err != nil {
		t.Fatalf("RecordExport failed: %v", err)
	}

	// Check immediately - should not be modified
	modified, entry, err := m.CheckExcelModified(excelPath)
	if err != nil {
		t.Fatalf("CheckExcelModified failed: %v", err)
	}

	if modified {
		t.Error("File should not be considered modified immediately after export")
	}
	if entry == nil {
		t.Error("Should have entry after RecordExport")
	}
}

func TestCheckExcelModified_UserModified(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "test.xlsx")

	// Create Excel file
	if err := os.WriteFile(excelPath, []byte("excel content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create manifest and record export
	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	if err := m.RecordExport(excelPath, []string{"test_dt.xml"}); err != nil {
		t.Fatalf("RecordExport failed: %v", err)
	}

	// Wait a bit and modify the file (simulating user edit)
	time.Sleep(100 * time.Millisecond)

	// Modify the Excel file
	if err := os.WriteFile(excelPath, []byte("USER MODIFIED"), 0644); err != nil {
		t.Fatal(err)
	}

	// Touch the file to ensure modTime is updated
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(excelPath, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Check - should be modified
	modified, entry, err := m.CheckExcelModified(excelPath)
	if err != nil {
		t.Fatalf("CheckExcelModified failed: %v", err)
	}

	if !modified {
		t.Error("File SHOULD be considered modified after user edit")
	}
	if entry == nil {
		t.Error("Should have entry")
	}
}

func TestExportGuard_AllowsFirstExport(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	excelPath := filepath.Join(tmpDir, "new.xlsx")

	// First export should be allowed
	err := m.ExportGuard(excelPath)
	if err != nil {
		t.Errorf("First export should be allowed: %v", err)
	}
}

func TestExportGuard_AllowsUnmodifiedExport(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "test.xlsx")

	// Create Excel file
	if err := os.WriteFile(excelPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	// Record initial export
	if err := m.RecordExport(excelPath, []string{"test.xml"}); err != nil {
		t.Fatal(err)
	}

	// Second export should be allowed (file not modified)
	err := m.ExportGuard(excelPath)
	if err != nil {
		t.Errorf("Export to unmodified file should be allowed: %v", err)
	}
}

func TestExportGuard_RejectsModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "test.xlsx")

	// Create Excel file
	if err := os.WriteFile(excelPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	// Record initial export
	if err := m.RecordExport(excelPath, []string{"test.xml"}); err != nil {
		t.Fatal(err)
	}

	// Simulate user modification
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelPath, []byte("USER MODIFIED"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelPath, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Export should be REJECTED
	err := m.ExportGuard(excelPath)
	if err == nil {
		t.Fatal("Export to user-modified file should be REJECTED")
	}

	if !IsExcelModifiedError(err) {
		t.Errorf("Error should be ExcelModifiedError, got %T", err)
	}

	t.Logf("Correctly rejected with: %v", err)
}

func TestCheckAllExports(t *testing.T) {
	tmpDir := t.TempDir()
	excel1 := filepath.Join(tmpDir, "file1.xlsx")
	excel2 := filepath.Join(tmpDir, "file2.xlsx")

	// Create files
	if err := os.WriteFile(excel1, []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(excel2, []byte("2"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	// Record exports
	if err := m.RecordExport(excel1, nil); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordExport(excel2, nil); err != nil {
		t.Fatal(err)
	}

	// Modify one file
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excel2, []byte("MODIFIED"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excel2, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Check all
	modified, err := m.CheckAllExports()
	if err != nil {
		t.Fatalf("CheckAllExports failed: %v", err)
	}

	if len(modified) != 1 {
		t.Errorf("Expected 1 modified file, got %d", len(modified))
	}

	// The modified file should be file2
	if len(modified) > 0 && filepath.Base(modified[0]) != "file2.xlsx" {
		t.Errorf("Expected file2.xlsx to be modified, got %s", modified[0])
	}
}

func TestRecordExport_UpdatesEntry(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "test.xlsx")

	if err := os.WriteFile(excelPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManifest()
	m.SetPath(filepath.Join(tmpDir, ".sync-manifest.json"))

	// First export
	xmlFiles1 := []string{"a.xml"}
	if err := m.RecordExport(excelPath, xmlFiles1); err != nil {
		t.Fatal(err)
	}

	entry1 := m.GetEntry(excelPath)
	if entry1 == nil {
		t.Fatal("Expected entry after first export")
	}
	time1 := entry1.LastExportTime

	// Wait and do second export
	time.Sleep(100 * time.Millisecond)

	xmlFiles2 := []string{"a.xml", "b.xml"}
	if err := m.RecordExport(excelPath, xmlFiles2); err != nil {
		t.Fatal(err)
	}

	entry2 := m.GetEntry(excelPath)
	if entry2 == nil {
		t.Fatal("Expected entry after second export")
	}

	if !entry2.LastExportTime.After(time1) {
		t.Error("Second export should have later timestamp")
	}

	if len(entry2.XMLFiles) != 2 {
		t.Errorf("XMLFiles should be updated, got %d files", len(entry2.XMLFiles))
	}
}

func TestListTrackedFiles(t *testing.T) {
	m := NewManifest()
	m.Files["a.xlsx"] = &FileEntry{}
	m.Files["b.xlsx"] = &FileEntry{}
	m.Files["c.xlsx"] = &FileEntry{}

	files := m.ListTrackedFiles()
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

func TestRemoveEntry(t *testing.T) {
	m := NewManifest()
	m.Files["test.xlsx"] = &FileEntry{}

	if m.GetEntry("test.xlsx") == nil {
		t.Fatal("Entry should exist before removal")
	}

	m.RemoveEntry("test.xlsx")

	if m.GetEntry("test.xlsx") != nil {
		t.Error("Entry should be nil after removal")
	}
}

// TestExcelSourceOfTruth is the critical integration test that verifies
// Excel is always the source of truth.
func TestExcelSourceOfTruth(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "rules.xlsx")
	manifestPath := filepath.Join(tmpDir, ".sync-manifest.json")

	// Scenario: AI exports XML to Excel, then user modifies Excel,
	// then AI tries to export again - should be REJECTED

	// Step 1: Create initial Excel file (simulating first export)
	if err := os.WriteFile(excelPath, []byte("AI GENERATED"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	// Step 2: AI records the export
	if err := m.RecordExport(excelPath, []string{"rules_dt.xml", "rules_edd.xml"}); err != nil {
		t.Fatal(err)
	}

	// Step 3: User modifies Excel (THE SOURCE OF TRUTH)
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelPath, []byte("USER EDITED TAX RATE"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelPath, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Step 4: AI tries to export again - MUST BE REJECTED
	err = m.ExportGuard(excelPath)
	if err == nil {
		t.Fatal("CRITICAL FAILURE: AI export should be REJECTED when user modified Excel")
	}

	if !IsExcelModifiedError(err) {
		t.Fatalf("Expected ExcelModifiedError, got %T: %v", err, err)
	}

	t.Log("SUCCESS: Excel source of truth is enforced - AI cannot overwrite user changes")
}

// TestWorkflow_AIEditsXMLThenExports simulates the full AI workflow.
func TestWorkflow_AIEditsXMLThenExports(t *testing.T) {
	tmpDir := t.TempDir()
	excelPath := filepath.Join(tmpDir, "rules.xlsx")
	manifestPath := filepath.Join(tmpDir, ".sync-manifest.json")

	m, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	// Workflow Step 1: AI does first export (no Excel file exists)
	err = m.ExportGuard(excelPath)
	if err != nil {
		t.Fatalf("First export should be allowed: %v", err)
	}

	// Simulate export
	if err := os.WriteFile(excelPath, []byte("EXPORT 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordExport(excelPath, []string{"rules.xml"}); err != nil {
		t.Fatal(err)
	}

	// Workflow Step 2: AI makes more XML changes and exports again
	err = m.ExportGuard(excelPath)
	if err != nil {
		t.Fatalf("Second export should be allowed (no user changes): %v", err)
	}

	// Simulate export
	if err := os.WriteFile(excelPath, []byte("EXPORT 2"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordExport(excelPath, []string{"rules.xml"}); err != nil {
		t.Fatal(err)
	}

	// Workflow Step 3: User edits Excel
	time.Sleep(100 * time.Millisecond)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(excelPath, []byte("USER EDIT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(excelPath, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	// Workflow Step 4: AI tries to export - REJECTED
	err = m.ExportGuard(excelPath)
	if err == nil {
		t.Fatal("Export after user edit should be REJECTED")
	}

	t.Log("Workflow test passed: AI -> AI -> User -> AI(blocked)")
}
