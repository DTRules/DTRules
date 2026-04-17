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

package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

// projectConfig holds optional directory overrides declared in DTRules.xml.
type projectConfig struct {
	XMLDir   string `xml:"xml_dir"`
	ExcelDir string `xml:"excel_dir"`
}

// loadProjectConfig reads DTRules.xml from projectRoot and returns any
// declared directory overrides. Returns an empty config (not an error) if
// the file does not exist or contains no override elements.
func loadProjectConfig(projectRoot string) (*projectConfig, error) {
	configPath := filepath.Join(projectRoot, "DTRules.xml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &projectConfig{}, nil
		}
		return nil, fmt.Errorf("reading DTRules.xml: %w", err)
	}

	var cfg projectConfig
	if err := xml.Unmarshal(data, &cfg); err != nil {
		// Malformed DTRules.xml — ignore overrides, don't fail.
		return &projectConfig{}, nil
	}
	return &cfg, nil
}

// resolveDirs determines the xml and excel directories for a project using
// the three-level precedence:
//  1. CLI flags (flagXMLDir / flagExcelDir) — non-empty string wins
//  2. DTRules.xml declarations in projectRoot
//  3. Default: "xml" and "excel" relative to projectRoot
//
// Returned paths are absolute.
func resolveDirs(projectRoot, flagXMLDir, flagExcelDir string) (xmlDir, excelDir string, err error) {
	cfg, err := loadProjectConfig(projectRoot)
	if err != nil {
		return "", "", err
	}

	resolveOne := func(flag, fromCfg, defaultName string) (string, error) {
		rel := flag
		if rel == "" {
			rel = fromCfg
		}
		if rel == "" {
			rel = defaultName
		}
		if filepath.IsAbs(rel) {
			return rel, nil
		}
		return filepath.Abs(filepath.Join(projectRoot, rel))
	}

	xmlDir, err = resolveOne(flagXMLDir, cfg.XMLDir, "xml")
	if err != nil {
		return "", "", err
	}
	excelDir, err = resolveOne(flagExcelDir, cfg.ExcelDir, "excel")
	if err != nil {
		return "", "", err
	}
	return xmlDir, excelDir, nil
}

// checkDirWithHint returns a non-nil error with a helpful message if the
// directory does not exist, telling the user which flag to use.
func checkDirWithHint(dir, flagName string) error {
	if dirExists(dir) {
		return nil
	}
	return fmt.Errorf(
		"could not find directory\n  Tried: %s\n  Use %s <path> or declare the element in DTRules.xml.",
		dir, flagName,
	)
}
