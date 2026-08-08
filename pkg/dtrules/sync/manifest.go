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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ManifestVersion is the current manifest format version.
const ManifestVersion = 1

// DefaultManifestName is the default filename for the sync manifest.
const DefaultManifestName = ".sync-manifest.json"

// Manifest tracks synchronization state between Excel and XML files.
// Excel is the source of truth. The manifest records when XML was last
// exported to Excel, allowing detection of user modifications to Excel
// that would be overwritten by a subsequent export.
type Manifest struct {
	// Version is the manifest format version.
	Version int `json:"version"`

	// LastUpdated is when the manifest was last modified.
	LastUpdated time.Time `json:"lastUpdated"`

	// Files maps Excel file paths (relative to manifest location) to their sync state.
	Files map[string]*FileEntry `json:"files"`

	// path is the filesystem path to this manifest (not serialized).
	path string
}

// FileEntry tracks the sync state for a single Excel file.
type FileEntry struct {
	// LastExportTime is when XML was last exported to this Excel file.
	// This is the timestamp we compare against Excel's current modTime.
	LastExportTime time.Time `json:"lastExportTime"`

	// ExcelModTimeAtExport is the Excel file's modTime at the time of last export.
	// This provides a secondary check - if Excel's modTime differs from this,
	// the file was modified externally.
	ExcelModTimeAtExport time.Time `json:"excelModTimeAtExport"`

	// XMLFiles lists the XML files that were exported to create this Excel file.
	// For combined workbooks: ["states/CO_dt.xml", "states/CO_edd.xml"]
	// For legacy mode: ["TaxReturn_dt.xml"]
	XMLFiles []string `json:"xmlFiles,omitempty"`
}

// NewManifest creates a new empty manifest.
func NewManifest() *Manifest {
	return &Manifest{
		Version: ManifestVersion,
		Files:   make(map[string]*FileEntry),
	}
}

// LoadManifest loads a manifest from the specified path.
// If the file doesn't exist, returns a new empty manifest.
func LoadManifest(path string) (*Manifest, error) {
	m := NewManifest()
	m.path = path

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Ensure Files map is initialized even if JSON was empty
	if m.Files == nil {
		m.Files = make(map[string]*FileEntry)
	}

	m.path = path
	return m, nil
}

// LoadManifestFromDir loads the manifest from a directory using the default name.
func LoadManifestFromDir(dir string) (*Manifest, error) {
	return LoadManifest(filepath.Join(dir, DefaultManifestName))
}

// Save writes the manifest to disk.
func (m *Manifest) Save() error {
	if m.path == "" {
		return fmt.Errorf("manifest path not set")
	}

	m.LastUpdated = time.Now()

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	if err := os.WriteFile(m.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// SaveTo writes the manifest to a specific path.
func (m *Manifest) SaveTo(path string) error {
	m.path = path
	return m.Save()
}

// GetPath returns the manifest file path.
func (m *Manifest) GetPath() string {
	return m.path
}

// SetPath sets the manifest file path.
func (m *Manifest) SetPath(path string) {
	m.path = path
}

// CheckExcelModified checks if an Excel file has been modified since the last export.
// Returns:
//   - modified: true if Excel was modified by user (export should be rejected)
//   - entry: the manifest entry for this file (nil if not tracked)
//   - err: error if file cannot be checked
//
// If the file is not in the manifest, returns (false, nil, nil) - first export is allowed.
func (m *Manifest) CheckExcelModified(excelPath string) (modified bool, entry *FileEntry, err error) {
	// Normalize path for lookup
	relPath := m.normalizePath(excelPath)

	entry = m.Files[relPath]
	if entry == nil {
		// Not in manifest - this is the first export, allow it
		return false, nil, nil
	}

	// Get current Excel file modTime
	info, err := os.Stat(excelPath)
	if os.IsNotExist(err) {
		// Excel file doesn't exist - allow export (it will create it)
		return false, entry, nil
	}
	if err != nil {
		return false, entry, fmt.Errorf("failed to stat Excel file: %w", err)
	}

	currentModTime := info.ModTime()

	// Check if Excel was modified after our last export
	// Use a small grace period (1 second) for filesystem timestamp precision
	gracePeriod := time.Second

	// Primary check: was Excel modified after we last exported to it?
	if currentModTime.After(entry.LastExportTime.Add(gracePeriod)) {
		return true, entry, nil
	}

	// Secondary check: does Excel's modTime differ from what we recorded?
	// This catches cases where the file was modified and then modified back
	// (same content but different timestamp)
	if !entry.ExcelModTimeAtExport.IsZero() {
		diff := currentModTime.Sub(entry.ExcelModTimeAtExport)
		if diff < -gracePeriod || diff > gracePeriod {
			return true, entry, nil
		}
	}

	return false, entry, nil
}

// RecordExport records that XML was exported to an Excel file.
// Call this AFTER a successful export to update the manifest.
func (m *Manifest) RecordExport(excelPath string, xmlFiles []string) error {
	relPath := m.normalizePath(excelPath)

	// Get Excel file's current modTime (after export)
	var excelModTime time.Time
	if info, err := os.Stat(excelPath); err == nil {
		excelModTime = info.ModTime()
	}

	now := time.Now()

	// Store XML paths relative to the manifest dir, like every other path
	// in the manifest. Absolute paths are machine-specific: a committed
	// manifest (Excel as system of record means the manifest ships with it)
	// would break on any other checkout.
	relXML := make([]string, 0, len(xmlFiles))
	for _, f := range xmlFiles {
		relXML = append(relXML, m.normalizePath(f))
	}

	m.Files[relPath] = &FileEntry{
		LastExportTime:       now,
		ExcelModTimeAtExport: excelModTime,
		XMLFiles:             relXML,
	}

	return m.Save()
}

// RemoveEntry removes an entry from the manifest.
func (m *Manifest) RemoveEntry(excelPath string) {
	relPath := m.normalizePath(excelPath)
	delete(m.Files, relPath)
}

// GetEntry returns the manifest entry for an Excel file, or nil if not tracked.
func (m *Manifest) GetEntry(excelPath string) *FileEntry {
	relPath := m.normalizePath(excelPath)
	return m.Files[relPath]
}

// normalizePath converts an absolute or relative path to a normalized relative path.
func (m *Manifest) normalizePath(path string) string {
	// If manifest has a path, try to make the input path relative to manifest's directory
	if m.path != "" {
		manifestDir := filepath.Dir(m.path)
		if rel, err := filepath.Rel(manifestDir, path); err == nil {
			// Only use relative path if it doesn't escape the directory
			if len(rel) < len(path) && rel != ".." && !filepath.IsAbs(rel) {
				return filepath.ToSlash(rel)
			}
		}
	}

	// Fall back to cleaning and normalizing the path
	return filepath.ToSlash(filepath.Clean(path))
}

// ExportGuard checks if an export is safe and returns an error if not.
// This is the main enforcement function - call before exporting XML to Excel.
func (m *Manifest) ExportGuard(excelPath string) error {
	modified, entry, err := m.CheckExcelModified(excelPath)
	if err != nil {
		return fmt.Errorf("failed to check Excel modification: %w", err)
	}

	if modified {
		return &ExcelModifiedError{
			ExcelPath:      excelPath,
			LastExportTime: entry.LastExportTime,
			Message: fmt.Sprintf(
				"Excel file %q was modified after last export at %s. "+
					"User changes would be lost. Import Excel to XML first, then re-apply your changes.",
				excelPath, entry.LastExportTime.Format(time.RFC3339)),
		}
	}

	return nil
}

// ExcelModifiedError is returned when attempting to export to an Excel file
// that has been modified since the last export.
type ExcelModifiedError struct {
	ExcelPath      string
	LastExportTime time.Time
	Message        string
}

func (e *ExcelModifiedError) Error() string {
	return e.Message
}

// IsExcelModifiedError returns true if the error is an ExcelModifiedError.
func IsExcelModifiedError(err error) bool {
	_, ok := err.(*ExcelModifiedError)
	return ok
}

// ListTrackedFiles returns all Excel files tracked by the manifest.
func (m *Manifest) ListTrackedFiles() []string {
	files := make([]string, 0, len(m.Files))
	for path := range m.Files {
		files = append(files, path)
	}
	return files
}

// CheckAllExports checks all tracked files for modifications.
// Returns a list of files that have been modified since last export.
func (m *Manifest) CheckAllExports() ([]string, error) {
	var modified []string

	for relPath := range m.Files {
		// Construct absolute path if we have manifest location
		excelPath := relPath
		if m.path != "" {
			excelPath = filepath.Join(filepath.Dir(m.path), relPath)
		}

		isModified, _, err := m.CheckExcelModified(excelPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check %s: %w", relPath, err)
		}

		if isModified {
			modified = append(modified, relPath)
		}
	}

	return modified, nil
}
