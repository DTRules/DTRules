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
)

func TestValidateProjectStructure_Valid(t *testing.T) {
	// Create temp directory with valid structure
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "excel"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "xml"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "testfiles"), 0755)

	result, err := ValidateProjectStructure(tmpDir, "", "")
	if err != nil {
		t.Fatalf("ValidateProjectStructure failed: %v", err)
	}

	if !result.IsValid() {
		t.Errorf("Expected valid structure, got errors: %v", result.Errors)
	}

	if result.Structure == nil {
		t.Fatal("Expected Structure to be set")
	}

	if result.Structure.ExcelDir == "" {
		t.Error("Expected ExcelDir to be set")
	}
	if result.Structure.XMLDir == "" {
		t.Error("Expected XMLDir to be set")
	}
}

// TestValidateProjectStructure_MapWorkbookNotOrphaned guards the fix where a
// mapping workbook (_map.xlsx) must be excluded from the Excel↔XML
// correspondence check — its _map.xml counterpart is already skipped, so
// including the workbook falsely reported it as having "no corresponding XML
// output".
func TestValidateProjectStructure_MapWorkbookNotOrphaned(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "excel"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "xml"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "excel", "rules_map.xlsx"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "xml", "rules_map.xml"), []byte("<mapping/>"), 0644)

	result, err := ValidateProjectStructure(tmpDir, "", "")
	if err != nil {
		t.Fatalf("ValidateProjectStructure failed: %v", err)
	}
	for _, w := range result.Warnings {
		if w.Path == "rules_map.xlsx" {
			t.Errorf("mapping workbook must not be flagged as missing XML, got: %s", w.Message)
		}
	}
	for _, m := range result.MissingXML {
		if m == "rules_map.xlsx" {
			t.Errorf("mapping workbook must not be in MissingXML")
		}
	}
}

func TestValidateProjectStructure_MissingExcel(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "xml"), 0755)
	// Missing excel/ directory

	result, err := ValidateProjectStructure(tmpDir, "", "")
	if err != nil {
		t.Fatalf("ValidateProjectStructure failed: %v", err)
	}

	if result.IsValid() {
		t.Error("Expected invalid structure due to missing excel/")
	}

	hasExcelError := false
	for _, e := range result.Errors {
		if e.Message == "excel/ directory not found" {
			hasExcelError = true
		}
	}
	if !hasExcelError {
		t.Error("Expected error about missing excel/ directory")
	}
}

func TestValidateProjectStructure_MissingXML(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "excel"), 0755)
	// Missing xml/ directory

	result, err := ValidateProjectStructure(tmpDir, "", "")
	if err != nil {
		t.Fatalf("ValidateProjectStructure failed: %v", err)
	}

	if result.IsValid() {
		t.Error("Expected invalid structure due to missing xml/")
	}

	hasXMLError := false
	for _, e := range result.Errors {
		if e.Message == "xml/ directory not found" {
			hasXMLError = true
		}
	}
	if !hasXMLError {
		t.Error("Expected error about missing xml/ directory")
	}
}

func TestValidateProjectStructure_OrphanedXML(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "excel"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "xml"), 0755)

	// Create XML file without Excel source
	os.WriteFile(filepath.Join(tmpDir, "xml", "orphan_dt.xml"), []byte("<decision_tables/>"), 0644)

	result, err := ValidateProjectStructure(tmpDir, "", "")
	if err != nil {
		t.Fatalf("ValidateProjectStructure failed: %v", err)
	}

	// Should still be valid (orphaned is just a warning)
	if !result.IsValid() {
		t.Errorf("Expected valid structure, got errors: %v", result.Errors)
	}

	if len(result.OrphanedXML) == 0 {
		t.Error("Expected orphaned XML file to be detected")
	}
}

func TestCreateProjectStructure(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "new-project")

	err := CreateProjectStructure(projectDir)
	if err != nil {
		t.Fatalf("CreateProjectStructure failed: %v", err)
	}

	// Check directories were created
	dirs := []string{"excel", "xml", "testfiles"}
	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Directory %s not created: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}

	// Check .gitignore was created
	gitignore := filepath.Join(projectDir, "excel", ".gitignore")
	if _, err := os.Stat(gitignore); err != nil {
		t.Errorf(".gitignore not created: %v", err)
	}
}

func TestGetProjectStructure(t *testing.T) {
	structure := GetProjectStructure("/some/project")

	if structure.ExcelDir != "/some/project/excel" {
		t.Errorf("Expected ExcelDir /some/project/excel, got %s", structure.ExcelDir)
	}
	if structure.XMLDir != "/some/project/xml" {
		t.Errorf("Expected XMLDir /some/project/xml, got %s", structure.XMLDir)
	}
	if structure.TestDir != "/some/project/testfiles" {
		t.Errorf("Expected TestDir /some/project/testfiles, got %s", structure.TestDir)
	}
}
