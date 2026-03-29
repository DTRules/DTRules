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

package el

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// XML structures for parsing decision tables
type dtXMLFile struct {
	XMLName xml.Name  `xml:"decision_tables"`
	Tables  []dtTable `xml:"decision_table"`
}

type dtTable struct {
	TableName  string       `xml:"table_name"`
	Conditions dtConditions `xml:"conditions"`
	Actions    dtActions    `xml:"actions"`
}

type dtConditions struct {
	Conditions []dtCondition `xml:"condition_details"`
}

type dtCondition struct {
	Number      int    `xml:"condition_number"`
	Description string `xml:"condition_description"`
	Postfix     string `xml:"condition_postfix"`
}

type dtActions struct {
	Actions []dtAction `xml:"action_details"`
}

type dtAction struct {
	Number      int    `xml:"action_number"`
	Description string `xml:"action_description"`
	Postfix     string `xml:"action_postfix"`
}

func TestValidateSampleProjectEL(t *testing.T) {
	// Find all _dt.xml files in sampleprojects
	sampleDir := filepath.Join("..", "..", "..", "..", "sampleprojects")

	var dtFiles []string
	err := filepath.Walk(sampleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, "_dt.xml") {
			dtFiles = append(dtFiles, path)
		}
		return nil
	})

	if err != nil {
		t.Skipf("Could not walk sampleprojects: %v", err)
	}

	if len(dtFiles) == 0 {
		t.Skip("No _dt.xml files found in sampleprojects")
	}

	compiler := NewCompiler()

	var totalConditions, parsedConditions int
	var totalActions, parsedActions int
	var failedEL []string

	for _, dtFile := range dtFiles {
		data, err := os.ReadFile(dtFile)
		if err != nil {
			t.Logf("Warning: could not read %s: %v", dtFile, err)
			continue
		}

		var dt dtXMLFile
		if err := xml.Unmarshal(data, &dt); err != nil {
			t.Logf("Warning: could not parse %s: %v", dtFile, err)
			continue
		}

		for _, table := range dt.Tables {
			// Try to compile each condition's EL
			for _, cond := range table.Conditions.Conditions {
				desc := strings.TrimSpace(cond.Description)
				if desc == "" {
					continue
				}
				totalConditions++

				_, err := compiler.CompileCondition(desc)
				if err == nil {
					parsedConditions++
				} else {
					failedEL = append(failedEL,
						filepath.Base(dtFile)+":"+table.TableName+":cond:"+desc)
				}
			}

			// Try to compile each action's EL
			for _, action := range table.Actions.Actions {
				desc := strings.TrimSpace(action.Description)
				if desc == "" {
					continue
				}
				totalActions++

				_, err := compiler.CompileAction(desc)
				if err == nil {
					parsedActions++
				} else {
					failedEL = append(failedEL,
						filepath.Base(dtFile)+":"+table.TableName+":action:"+desc)
				}
			}
		}
	}

	t.Logf("Conditions: %d/%d parsed (%.1f%%)",
		parsedConditions, totalConditions,
		float64(parsedConditions)/float64(totalConditions)*100)
	t.Logf("Actions: %d/%d parsed (%.1f%%)",
		parsedActions, totalActions,
		float64(parsedActions)/float64(totalActions)*100)

	if len(failedEL) > 0 {
		t.Logf("Failed to parse %d expressions:", len(failedEL))
		for i, el := range failedEL {
			if i >= 20 {
				t.Logf("  ... and %d more", len(failedEL)-20)
				break
			}
			t.Logf("  %s", el)
		}
	}
}
