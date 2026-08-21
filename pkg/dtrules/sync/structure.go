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
	"os"
	"path/filepath"
	"strings"
)

// ProjectStructure describes the validated structure of a DTRules project.
type ProjectStructure struct {
	// ProjectRoot is the root directory of the project.
	ProjectRoot string
	// ExcelDir is the directory containing Excel files (source of truth).
	ExcelDir string
	// XMLDir is the directory containing XML files (compiled from Excel).
	XMLDir string
	// TestDir is the directory containing test files.
	TestDir string
	// ManifestPath is the path to the sync manifest file.
	ManifestPath string
	// ExcelFiles lists all Excel files found.
	ExcelFiles []string
	// XMLFiles lists all XML files found.
	XMLFiles []string
}

// StructureError represents a validation error in project structure.
type StructureError struct {
	Path    string
	Message string
}

func (e *StructureError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// StructureWarning represents a non-fatal issue in project structure.
type StructureWarning struct {
	Path    string
	Message string
}

func (w *StructureWarning) String() string {
	if w.Path != "" {
		return fmt.Sprintf("%s: %s", w.Path, w.Message)
	}
	return w.Message
}

// StructureValidationResult holds the result of project structure validation.
type StructureValidationResult struct {
	// Structure contains the validated project structure if validation succeeded.
	Structure *ProjectStructure
	// Errors contains any validation errors (structure is invalid).
	Errors []*StructureError
	// Warnings contains non-fatal issues (structure is valid but has issues).
	Warnings []*StructureWarning
	// OrphanedXML lists XML files without corresponding Excel source.
	OrphanedXML []string
	// MissingXML lists Excel files without corresponding XML output.
	MissingXML []string
}

// IsValid returns true if no validation errors were found.
func (r *StructureValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// ValidateProjectStructure validates the DTRules project structure.
// It checks that:
// - excel/ directory exists
// - xml/ directory exists
// - For each Excel file, corresponding XML files exist (or warns if not)
// - No orphaned XML files without Excel source (warns)
//
// xmlDirOverride and excelDirOverride, when non-empty, replace the default
// xml/ and excel/ subdirectory paths respectively.
func ValidateProjectStructure(projectRoot string, xmlDirOverride, excelDirOverride string) (*StructureValidationResult, error) {
	result := &StructureValidationResult{
		Structure: &ProjectStructure{
			ProjectRoot: projectRoot,
		},
	}

	// Resolve absolute path
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}
	result.Structure.ProjectRoot = absRoot

	// Set paths, respecting overrides.
	resolveDir := func(override, defaultName string) string {
		if override != "" {
			if filepath.IsAbs(override) {
				return override
			}
			return filepath.Join(absRoot, override)
		}
		return filepath.Join(absRoot, defaultName)
	}
	result.Structure.ExcelDir = resolveDir(excelDirOverride, "excel")
	result.Structure.XMLDir = resolveDir(xmlDirOverride, "xml")
	result.Structure.TestDir = filepath.Join(absRoot, "testfiles")
	result.Structure.ManifestPath = filepath.Join(result.Structure.ExcelDir, DefaultManifestName)

	// Check required directories exist
	if !dirExists(result.Structure.ExcelDir) {
		result.Errors = append(result.Errors, &StructureError{
			Path:    result.Structure.ExcelDir,
			Message: "excel/ directory not found",
		})
	}

	if !dirExists(result.Structure.XMLDir) {
		result.Errors = append(result.Errors, &StructureError{
			Path:    result.Structure.XMLDir,
			Message: "xml/ directory not found",
		})
	}

	// If directories don't exist, return early
	if len(result.Errors) > 0 {
		return result, nil
	}

	// Collect Excel files
	excelFiles, err := collectFilesByExt(result.Structure.ExcelDir, ".xlsx")
	if err != nil {
		return nil, fmt.Errorf("failed to scan excel directory: %w", err)
	}
	result.Structure.ExcelFiles = excelFiles

	// Collect XML files
	xmlFiles, err := collectFilesByExt(result.Structure.XMLDir, ".xml")
	if err != nil {
		return nil, fmt.Errorf("failed to scan xml directory: %w", err)
	}
	result.Structure.XMLFiles = xmlFiles

	// Build maps for correspondence checking
	// Excel base name -> exists (without extension, removing _dt/_edd suffix from XML)
	excelBases := make(map[string]bool)
	for _, f := range excelFiles {
		relPath, _ := filepath.Rel(result.Structure.ExcelDir, f)
		base := strings.TrimSuffix(relPath, ".xlsx")
		excelBases[strings.ToLower(base)] = true
	}

	// XML base name -> path (strip _dt.xml or _edd.xml)
	xmlBases := make(map[string][]string)
	for _, f := range xmlFiles {
		relPath, _ := filepath.Rel(result.Structure.XMLDir, f)
		base := getXMLBaseName(relPath)
		xmlBases[strings.ToLower(base)] = append(xmlBases[strings.ToLower(base)], relPath)
	}

	// Check for Excel files without XML counterparts
	for _, f := range excelFiles {
		relPath, _ := filepath.Rel(result.Structure.ExcelDir, f)
		base := strings.TrimSuffix(relPath, ".xlsx")
		if _, hasXML := xmlBases[strings.ToLower(base)]; !hasXML {
			result.MissingXML = append(result.MissingXML, relPath)
			result.Warnings = append(result.Warnings, &StructureWarning{
				Path:    relPath,
				Message: "Excel file has no corresponding XML output - run 'dtrules sync import'",
			})
		}
	}

	// Check for XML files without Excel source
	for baseLower, xmlPaths := range xmlBases {
		if !excelBases[baseLower] {
			for _, xmlPath := range xmlPaths {
				result.OrphanedXML = append(result.OrphanedXML, xmlPath)
				result.Warnings = append(result.Warnings, &StructureWarning{
					Path:    xmlPath,
					Message: "XML file has no corresponding Excel source",
				})
			}
		}
	}

	// Warn if testfiles directory doesn't exist
	if !dirExists(result.Structure.TestDir) {
		result.Warnings = append(result.Warnings, &StructureWarning{
			Path:    result.Structure.TestDir,
			Message: "testfiles/ directory not found - create it for test scenarios",
		})
	}

	return result, nil
}

// CreateProjectStructure creates the standard DTRules project directory structure.
func CreateProjectStructure(projectRoot string) error {
	dirs := []string{
		filepath.Join(projectRoot, "excel"),
		filepath.Join(projectRoot, "xml"),
		filepath.Join(projectRoot, "testfiles"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// No .gitignore is written any more. Its only content was the sync
	// manifest exclusion, and new projects never grow a manifest (#1091);
	// gitignoring it is also the exact reason a fresh clone used to arrive
	// with no sync state at all.

	return nil
}

// GetProjectStructure returns the project structure paths without validation.
// Use ValidateProjectStructure for full validation.
func GetProjectStructure(projectRoot string) *ProjectStructure {
	absRoot, _ := filepath.Abs(projectRoot)
	return &ProjectStructure{
		ProjectRoot:  absRoot,
		ExcelDir:     filepath.Join(absRoot, "excel"),
		XMLDir:       filepath.Join(absRoot, "xml"),
		TestDir:      filepath.Join(absRoot, "testfiles"),
		ManifestPath: filepath.Join(absRoot, "excel", DefaultManifestName),
	}
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// collectFilesByExt recursively collects all files with the given extension.
func collectFilesByExt(dir, ext string) ([]string, error) {
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
			// Skip temp files
			if strings.HasPrefix(d.Name(), "~$") {
				return nil
			}
			// Skip template files
			if strings.Contains(strings.ToUpper(d.Name()), "TEMPLATE") {
				return nil
			}
			// Skip mapping files — a mapping (_map.xml / _map.xlsx) has no
			// DT/EDD counterpart, so it must be excluded from BOTH the XML and
			// the Excel collection or the correspondence check falsely reports
			// the workbook as having "no corresponding XML output".
			if strings.HasSuffix(d.Name(), "_map.xml") || strings.HasSuffix(d.Name(), "_map.xlsx") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// getXMLBaseName extracts the base name from an XML file path.
// It strips the _dt.xml or _edd.xml suffix to get the common base name
// that corresponds to the Excel file.
func getXMLBaseName(xmlPath string) string {
	if strings.HasSuffix(xmlPath, "_dt.xml") {
		return strings.TrimSuffix(xmlPath, "_dt.xml")
	}
	if strings.HasSuffix(xmlPath, "_edd.xml") {
		return strings.TrimSuffix(xmlPath, "_edd.xml")
	}
	// Legacy format: just strip .xml
	return strings.TrimSuffix(xmlPath, ".xml")
}
