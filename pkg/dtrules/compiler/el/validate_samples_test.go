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
	Contexts   dtContexts   `xml:"contexts"`
	Conditions dtConditions `xml:"conditions"`
	Actions    dtActions    `xml:"actions"`
}

type dtContexts struct {
	Contexts []dtContext `xml:"context_details"`
}

type dtContext struct {
	Number      int    `xml:"context_number"`
	Description string `xml:"context_description"`
	Postfix     string `xml:"context_postfix"`
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

// looksLikeEL returns true if the description looks like EL syntax rather than
// a human-readable comment. Comments typically have spaces but no EL operators.
func looksLikeEL(desc string) bool {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return false
	}

	descLower := strings.ToLower(desc)

	// Starts with // is a comment
	if strings.HasPrefix(desc, "//") {
		return false
	}

	// ============================================================
	// Check comment/shorthand patterns FIRST (before EL patterns)
	// This prevents false positives like "Person is 18 or older"
	// from matching " or " and being treated as EL
	// ============================================================

	// Common comment/shorthand patterns (not valid EL)
	commentPatterns := []string{
		// Human-readable descriptions
		"person is", "this person", "does not have", "is pregnant",
		"at least one", "can be head", "must be", "has income",
		"subject to", "all income from", "get number",
		"is financial", "can be", "who can", "what is",
		"are there", "is there", "no territory", "no multi-state",
		"is taxpayer", "is eligible", "determine who",
		"set total", "set income", "set is_", "set the",
		"mark person", "note that", "select the", "calculate ",
		"filing required", "apportionment will", "threshold:",
		"% ceiling", "% rate", // shorthand like "0% ceiling"
		"entity exists", "exists in context", // shorthand
		"sum of business", "credits greater than",
		// Action shorthand
		"based on age", "based on status",
	}
	for _, pattern := range commentPatterns {
		if strings.Contains(descLower, pattern) {
			return false
		}
	}

	// Dollar amounts are not valid EL (should use plain numbers)
	if strings.Contains(desc, "$") {
		return false
	}

	// Percent in text (like "0% ceiling") - not valid EL
	if strings.Contains(desc, "%") && strings.Contains(desc, " ") {
		return false
	}

	// Chained comparisons like "> 0 and <= 0.50" are not valid EL
	if strings.Contains(descLower, " and <=") || strings.Contains(descLower, " and >=") ||
		strings.Contains(descLower, " or <=") || strings.Contains(descLower, " or >=") {
		return false
	}

	// "X" or "Y" shorthand (should be status == "X" or status == "Y")
	if strings.Contains(descLower, `" or "`) || strings.Contains(descLower, `' or '`) {
		return false
	}

	// Parenthetical explanations like "(full exemption)" - not EL
	if strings.Contains(desc, "(") && strings.Contains(descLower, "exemption") {
		return false
	}

	// "Joint/HOH" style shorthand
	if strings.Contains(desc, "/") && !strings.Contains(desc, "//") {
		// Slash used as "or" separator, not division
		parts := strings.Split(desc, "/")
		if len(parts) == 2 && !strings.Contains(parts[0], " ") && !strings.Contains(parts[1], " ") {
			// Could be like "MFJ/QW" - likely shorthand
			return false
		}
	}

	// ============================================================
	// Now check for actual EL patterns
	// ============================================================

	// Single word (likely a table name or boolean) is EL
	if !strings.Contains(desc, " ") {
		return true
	}

	// Check for EL operators and keywords
	elPatterns := []string{
		"==", "!=", ">=", "<=", ">", "<", "=",
		" is equal to ", " is not equal to ",
		" is greater than ", " is less than ",
		" and ", " or ", " not ",
		" to ", " from ", " in ", " of ",
		"forall ", "foreach ", "perform ",
		" add ", "set ", "new ", "clear ",
		" includes ", " does not include ",
		" is null", " is not null",
		"otherwise", "default", "true", "false",
	}

	for _, pattern := range elPatterns {
		if strings.Contains(descLower, pattern) {
			return true
		}
	}

	// Check for quotes (string literals)
	if strings.Contains(desc, "\"") || strings.Contains(desc, "'") {
		return true
	}

	// Check for dots (entity.attribute)
	if strings.Contains(desc, ".") {
		return true
	}

	// Contains multiple words but no EL patterns - probably a comment
	return false
}

// originalJavaProjects lists the sample projects that existed in the original
// Java codebase (before Go port), whose EL descriptions this test parses.
//
// Whether stored postfix matches its DSL is not checked here: every sample's
// postfix is recompiled and compared by TestSamplePostfixIsCompiledFromDSL in
// pkg/dtrules/authoring (#1300), which replaced this file's Java-comparison
// tests.
var originalJavaProjects = map[string]bool{
	"CHIP":        true,
	"ChipApp":     true,
	"KidAid":      true,
	"TestProject": true,
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
			// Only include original Java-era projects that have proper EL
			isOriginal := false
			for project := range originalJavaProjects {
				if strings.Contains(path, "/"+project+"/") {
					isOriginal = true
					break
				}
			}
			if isOriginal {
				dtFiles = append(dtFiles, path)
			}
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
	var elConditions, elConditionsParsed int // Only entries that look like EL
	var elActions, elActionsParsed int
	var failedEL []string
	var skippedComments int

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

				isEL := looksLikeEL(desc)
				if isEL {
					elConditions++
				} else {
					skippedComments++
				}

				_, err := compiler.CompileCondition(desc)
				if err == nil {
					parsedConditions++
					if isEL {
						elConditionsParsed++
					}
				} else if isEL {
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

				isEL := looksLikeEL(desc)
				if isEL {
					elActions++
				} else {
					skippedComments++
				}

				_, err := compiler.CompileAction(desc)
				if err == nil {
					parsedActions++
					if isEL {
						elActionsParsed++
					}
				} else if isEL {
					failedEL = append(failedEL,
						filepath.Base(dtFile)+":"+table.TableName+":action:"+desc)
				}
			}
		}
	}

	t.Logf("=== All Descriptions (including comments) ===")
	t.Logf("Conditions: %d/%d parsed (%.1f%%)",
		parsedConditions, totalConditions,
		float64(parsedConditions)/float64(totalConditions)*100)
	t.Logf("Actions: %d/%d parsed (%.1f%%)",
		parsedActions, totalActions,
		float64(parsedActions)/float64(totalActions)*100)

	t.Logf("")
	t.Logf("=== EL Expressions Only (excluding %d comments) ===", skippedComments)
	if elConditions > 0 {
		t.Logf("Conditions: %d/%d parsed (%.1f%%)",
			elConditionsParsed, elConditions,
			float64(elConditionsParsed)/float64(elConditions)*100)
	}
	if elActions > 0 {
		t.Logf("Actions: %d/%d parsed (%.1f%%)",
			elActionsParsed, elActions,
			float64(elActionsParsed)/float64(elActions)*100)
	}

	// Count failures by type
	var condFailures, actionFailures int
	for _, el := range failedEL {
		if strings.Contains(el, ":cond:") {
			condFailures++
		} else {
			actionFailures++
		}
	}

	if len(failedEL) > 0 {
		t.Logf("Failed to parse %d expressions (%d conditions, %d actions):",
			len(failedEL), condFailures, actionFailures)

		// Show all condition failures
		if condFailures > 0 {
			t.Logf("")
			t.Logf("Condition failures:")
			for _, el := range failedEL {
				if strings.Contains(el, ":cond:") {
					t.Logf("  %s", el)
				}
			}
		}

		// Show action failures (first 20)
		if actionFailures > 0 {
			t.Logf("")
			t.Logf("Action failures (first 20):")
			actionCount := 0
			for _, el := range failedEL {
				if strings.Contains(el, ":action:") {
					if actionCount < 20 {
						t.Logf("  %s", el)
					}
					actionCount++
				}
			}
			if actionCount > 20 {
				t.Logf("  ... and %d more", actionCount-20)
			}
		}
	}
}
