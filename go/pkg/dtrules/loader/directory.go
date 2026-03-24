// Copyright 2004-2011 DTRules.com, Inc.
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

package loader

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
)

// xmlFileInfo holds metadata about an XML file.
type xmlFileInfo struct {
	Path            string
	Number          int
	IsDecisionTable bool
	FilePath        string // The FILE_PATH or file_path value
}

// shouldSkipFile determines if a file should be skipped during collection.
func shouldSkipFile(path string) bool {
	filename := filepath.Base(path)

	// Skip template files
	if strings.Contains(strings.ToUpper(filename), "TEMPLATE") {
		return true
	}

	// Skip mapping files (different XML format)
	if strings.HasSuffix(filename, "_map.xml") {
		return true
	}

	// Skip files in testfiles directory
	if strings.Contains(path, "/testfiles/") || strings.Contains(path, "\\testfiles\\") {
		return true
	}

	// Skip schema files
	if strings.Contains(path, "/schemas/") || strings.Contains(path, "\\schemas\\") {
		return true
	}

	return false
}

// CollectXMLFiles scans a directory recursively for *.xml files.
func CollectXMLFiles(dirPath string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".xml") {
			if !shouldSkipFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

// ParseFileMetadata extracts metadata from an XML file.
func ParseFileMetadata(filePath string) (*xmlFileInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	// Quick validation - check if file looks like it contains decision_tables or entity_data_dictionary
	dataStr := string(data)
	hasDecisionTables := strings.Contains(dataStr, "<decision_tables")
	hasEntityDict := strings.Contains(dataStr, "<entity_data_dictionary")

	if !hasDecisionTables && !hasEntityDict {
		return nil, fmt.Errorf("file %s does not appear to be a DTRules XML file", filePath)
	}

	info := &xmlFileInfo{
		Path: filePath,
	}

	// Try parsing as decision table first
	if hasDecisionTables {
		var dtFile DTFile
		if err := xml.Unmarshal(data, &dtFile); err == nil && len(dtFile.Tables) > 0 {
			// It's a decision table file
			info.IsDecisionTable = true

			// Extract TABLE_NUMBER from first table
			if dtFile.Tables[0].AttributeFields.TableNumber != "" {
				tableNumStr := strings.TrimSpace(dtFile.Tables[0].AttributeFields.TableNumber)

				// Check for template placeholders
				if strings.Contains(tableNumStr, "X") || strings.Contains(tableNumStr, "?") {
					return nil, fmt.Errorf("template file with placeholder TABLE_NUMBER: %s", tableNumStr)
				}

				num, err := strconv.Atoi(tableNumStr)
				if err != nil {
					return nil, fmt.Errorf("invalid TABLE_NUMBER in %s: %w", filePath, err)
				}
				info.Number = num
			}

			// Extract FILE_PATH
			filePathStr := strings.TrimSpace(dtFile.Tables[0].AttributeFields.FilePath)
			info.FilePath = filePathStr

			// Skip merged/monolithic files that don't have FILE_PATH
			// These are typically generated files like TaxReturn_dt.xml
			// Individual table files should always have a FILE_PATH
			if filePathStr == "" {
				return nil, fmt.Errorf("decision table file missing FILE_PATH (likely a merged/generated file)")
			}

			return info, nil
		}
	}

	// Try parsing as EDD
	if hasEntityDict {
		var eddFile EDDFile
		if err := xml.Unmarshal(data, &eddFile); err == nil {
			// It's an EDD file
			info.IsDecisionTable = false

			// Extract file_path from metadata
			filePathStr := strings.TrimSpace(eddFile.FileMetadata.FilePath)
			info.FilePath = filePathStr

			// Skip merged/monolithic files that don't have file_path
			// These are typically generated files like TaxReturn_edd.xml
			// Individual EDD files should always have a file_path
			if filePathStr == "" {
				return nil, fmt.Errorf("EDD file missing file_path (likely a merged/generated file)")
			}

			// Parse number from file_path (e.g., "states/AL/40100_AL_constants" -> 40100)
			parts := strings.Split(filePathStr, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				// Extract leading digits
				numStr := ""
				for _, ch := range lastPart {
					if ch >= '0' && ch <= '9' {
						numStr += string(ch)
					} else {
						break
					}
				}
				if numStr != "" {
					num, err := strconv.Atoi(numStr)
					if err != nil {
						return nil, fmt.Errorf("invalid file_path number in %s: %w", filePathStr, err)
					}
					info.Number = num
				}
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("file %s is neither a valid decision table nor EDD file", filePath)
}

// LoadRulesFromDirectory loads all XML files from a directory.
// EDDs are loaded first (in order), then decision tables (in order).
func LoadRulesFromDirectory(rs dtrules.RuleSet, dirPath string) error {
	// 1. Collect all XML files
	files, err := CollectXMLFiles(dirPath)
	if err != nil {
		return fmt.Errorf("failed to collect XML files from %s: %w", dirPath, err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no XML files found in directory: %s", dirPath)
	}

	// 2. Parse metadata for each file
	var fileInfos []*xmlFileInfo
	for _, file := range files {
		info, err := ParseFileMetadata(file)
		if err != nil {
			// Log warning but continue
			fmt.Printf("Warning: skipping file %s: %v\n", file, err)
			continue
		}
		fileInfos = append(fileInfos, info)
	}

	if len(fileInfos) == 0 {
		return fmt.Errorf("no valid XML files found in directory: %s", dirPath)
	}

	// 3. Sort by number
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].Number < fileInfos[j].Number
	})

	// 4. Separate EDDs and DTs
	var edds []*xmlFileInfo
	var dts []*xmlFileInfo
	for _, info := range fileInfos {
		if info.IsDecisionTable {
			dts = append(dts, info)
		} else {
			edds = append(edds, info)
		}
	}

	// Get session and factory
	session, err := rs.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	factory := session.GetEntityFactory().(*entity.Factory)

	var errs []error

	// 5. Load EDDs in order
	for _, info := range edds {
		if err := loadEDDFile(session, factory, info.Path); err != nil {
			errs = append(errs, fmt.Errorf("failed to load EDD %s: %w", info.Path, err))
		}
	}

	// 6. Load DTs in order
	for _, info := range dts {
		if err := loadDTFile(session, factory, info.Path); err != nil {
			errs = append(errs, fmt.Errorf("failed to load DT %s: %w", info.Path, err))
		}
	}

	// 7. Return aggregated errors
	if len(errs) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString(fmt.Sprintf("failed to load %d files:\n", len(errs)))
		for i, err := range errs {
			errMsg.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
		}
		return errors.New(errMsg.String())
	}

	return nil
}

// loadEDDFile loads a single EDD file.
func loadEDDFile(session dtrules.Session, factory *entity.Factory, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	loader := NewEDDLoader(session, factory)
	return loader.Load(f)
}

// loadDTFile loads a single decision table file.
func loadDTFile(session dtrules.Session, factory *entity.Factory, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	loader := NewDTLoader(session, factory)
	return loader.Load(f)
}
