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
	"fmt"
	"github.com/DTRules/DTRules/pkg/dtrules/project"

	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// projectConfig holds optional project settings declared in DTRules.xml.
// loadProjectConfig resolves a project's settings. Thin wrapper kept so the
// existing call sites read naturally; the resolution itself lives in
// pkg/dtrules/project, which every layer shares (#1052).
func loadProjectConfig(projectRoot string) (*project.Config, error) {
	return project.Load(projectRoot), nil
}

func resolveDirs(projectRoot, flagXMLDir, flagExcelDir string) (xmlDir, excelDir string, err error) {
	c := project.Load(projectRoot).WithDirs(flagXMLDir, flagExcelDir)
	return c.XMLDir, c.ExcelDir, nil
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

// validateStructure runs the project-structure check against the directories
// the project actually declares.
//
// `sync.ValidateProjectStructure` falls back to the literal "excel" and "xml"
// names when given no override, which is right for the function but wrong for
// every caller that has a manifest available and does not pass it. `review`
// and the MCP project tools both passed empty strings, so a project declaring
// <excel_dir>source</excel_dir> was told `excel/ directory not found` — naming
// a path it does not use — while `verify` on the same project resolved it
// correctly. One layout, two answers (#1031).
//
// Callers with their own --xml-dir/--excel-dir flags should keep calling
// resolveDirs themselves and pass the result; this is for the callers that
// have no flags to honour.
func validateStructure(projectRoot string) (*sync.StructureValidationResult, error) {
	xmlDir, excelDir, err := resolveDirs(projectRoot, "", "")
	if err != nil {
		return nil, err
	}
	return sync.ValidateProjectStructure(projectRoot, xmlDir, excelDir)
}
