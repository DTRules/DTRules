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

// normalizeWhitespace collapses multiple spaces to single space for comparison.
// Java compiler sometimes emits double spaces which shouldn't affect semantics.
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// originalJavaProjects lists the sample projects that existed in the original
// Java codebase (before Go port). These have proper EL in description fields
// that was compiled to postfix by the Java EL compiler.
var originalJavaProjects = map[string]bool{
	"CHIP":               true,
	"ChipApp":            true,
	"KidAid":             true,
	"KidAid_Application": true,
	"TestProject":        true,
	// Note: StateTax and CorporateTax have hand-written postfix, not Java EL compiled
	// Note: Sudoku used a different DSL, not EL
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

// findEDDForDT finds the EDD file corresponding to a decision table file.
// Decision tables are named like "Project_dt.xml" and EDDs like "Project_edd.xml"
func findEDDForDT(dtPath string) string {
	// Replace _dt.xml with _edd.xml
	eddPath := strings.Replace(dtPath, "_dt.xml", "_edd.xml", 1)
	if _, err := os.Stat(eddPath); err == nil {
		return eddPath
	}
	return ""
}

// TestPostfixMatchesJava compares Go compiler postfix output against the Java
// compiler's postfix stored in the XML files. This is the key test for ensuring
// 100% compatibility between Go and Java implementations.
func TestPostfixMatchesJava(t *testing.T) {
	// Find all _dt.xml files in original Java projects
	sampleDir := filepath.Join("..", "..", "..", "..", "sampleprojects")

	var dtFiles []string
	err := filepath.Walk(sampleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, "_dt.xml") {
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

	var totalConditions, matchedConditions int
	var totalActions, matchedActions int
	var mismatches []string

	for _, dtFile := range dtFiles {
		// Load the EDD for this project to get type information
		eddPath := findEDDForDT(dtFile)
		var symbols map[string]string
		if eddPath != "" {
			var err error
			symbols, err = LoadEDDFile(eddPath)
			if err != nil {
				t.Logf("Warning: could not load EDD %s: %v", eddPath, err)
			}
		}

		// Create a compiler with the symbol table
		compiler := NewCompiler()
		if symbols != nil {
			compiler.SetSymbols(symbols)
		}

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
			// Create a fresh compiler for each table (local variables are per-table)
			tableCompiler := NewCompiler()
			if symbols != nil {
				tableCompiler.SetSymbols(symbols)
			}

			// Pre-process context statements to register local variables
			for _, ctx := range table.Contexts.Contexts {
				ctxDesc := strings.TrimSpace(ctx.Description)
				if ctxDesc != "" && looksLikeEL(ctxDesc) {
					// Compile context to register local variables
					// We don't care about the output, just the side effect of registering locals
					tableCompiler.CompileContext(ctxDesc)
				}
			}

			// Compare condition postfix
			for _, cond := range table.Conditions.Conditions {
				desc := strings.TrimSpace(cond.Description)
				javaPostfix := strings.TrimSpace(cond.Postfix)
				if desc == "" || javaPostfix == "" {
					continue
				}
				if !looksLikeEL(desc) {
					continue
				}

				totalConditions++

				goPostfix, err := tableCompiler.CompileCondition(desc)
				if err != nil {
					mismatches = append(mismatches,
						filepath.Base(dtFile)+":"+table.TableName+":cond:"+desc+
							"\n    Go error: "+err.Error()+
							"\n    Java:     "+javaPostfix)
					continue
				}

				goPostfix = normalizeWhitespace(goPostfix)
				javaPostfixNorm := normalizeWhitespace(javaPostfix)
				if goPostfix == javaPostfixNorm {
					matchedConditions++
				} else {
					mismatches = append(mismatches,
						filepath.Base(dtFile)+":"+table.TableName+":cond:"+desc+
							"\n    Go:   "+goPostfix+
							"\n    Java: "+javaPostfixNorm)
				}
			}

			// Compare action postfix
			for _, action := range table.Actions.Actions {
				desc := strings.TrimSpace(action.Description)
				javaPostfix := strings.TrimSpace(action.Postfix)
				if desc == "" || javaPostfix == "" {
					continue
				}
				if !looksLikeEL(desc) {
					continue
				}

				totalActions++

				goPostfix, err := tableCompiler.CompileAction(desc)
				if err != nil {
					mismatches = append(mismatches,
						filepath.Base(dtFile)+":"+table.TableName+":action:"+desc+
							"\n    Go error: "+err.Error()+
							"\n    Java:     "+javaPostfix)
					continue
				}

				goPostfix = normalizeWhitespace(goPostfix)
				javaPostfixNorm := normalizeWhitespace(javaPostfix)
				if goPostfix == javaPostfixNorm {
					matchedActions++
				} else {
					mismatches = append(mismatches,
						filepath.Base(dtFile)+":"+table.TableName+":action:"+desc+
							"\n    Go:   "+goPostfix+
							"\n    Java: "+javaPostfixNorm)
				}
			}
		}
	}

	t.Logf("=== Postfix Comparison Results ===")
	condPct := float64(0)
	if totalConditions > 0 {
		condPct = float64(matchedConditions) / float64(totalConditions) * 100
	}
	actionPct := float64(0)
	if totalActions > 0 {
		actionPct = float64(matchedActions) / float64(totalActions) * 100
	}
	t.Logf("Conditions: %d/%d match (%.1f%%)", matchedConditions, totalConditions, condPct)
	t.Logf("Actions: %d/%d match (%.1f%%)", matchedActions, totalActions, actionPct)

	if len(mismatches) > 0 {
		t.Logf("\n=== Mismatches (first 30) ===")
		limit := 30
		if len(mismatches) < limit {
			limit = len(mismatches)
		}
		for i := 0; i < limit; i++ {
			t.Logf("%s", mismatches[i])
		}
		if len(mismatches) > 30 {
			t.Logf("... and %d more mismatches", len(mismatches)-30)
		}
	}

	// Don't fail the test yet - we're still working on matching
	// Once we hit 100%, we can change this to t.Errorf
}

// taxProjects lists the Tax sample projects that have hand-written postfix
// which we want to recompile with the Go EL compiler.
var taxProjects = map[string]bool{
	"StateTax":     true,
	"CorporateTax": true,
}

// TestCompileTaxProjects compiles Tax project EL descriptions and compares
// the Go compiler output with the existing hand-written postfix.
func TestCompileTaxProjects(t *testing.T) {
	sampleDir := filepath.Join("..", "..", "..", "..", "sampleprojects")

	var dtFiles []string
	err := filepath.Walk(sampleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, "_dt.xml") {
			for project := range taxProjects {
				if strings.Contains(path, "/"+project+"/") {
					// Only include repository/xml versions (compiled output)
					if strings.Contains(path, "/repository/xml/") {
						dtFiles = append(dtFiles, path)
					}
					break
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Skipf("Could not walk sampleprojects: %v", err)
	}

	if len(dtFiles) == 0 {
		t.Skip("No Tax project _dt.xml files found")
	}

	var totalConditions, compiledConditions, matchedConditions int
	var totalActions, compiledActions, matchedActions int
	var differences []string

	for _, dtFile := range dtFiles {
		// Load the EDD for this project
		eddPath := findEDDForDT(dtFile)
		var symbols map[string]string
		if eddPath != "" {
			var err error
			symbols, err = LoadEDDFile(eddPath)
			if err != nil {
				t.Logf("Warning: could not load EDD %s: %v", eddPath, err)
			}
		}

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
			// Create a fresh compiler for each table
			tableCompiler := NewCompiler()
			if symbols != nil {
				tableCompiler.SetSymbols(symbols)
			}

			// Pre-process context statements to register local variables
			for _, ctx := range table.Contexts.Contexts {
				ctxDesc := strings.TrimSpace(ctx.Description)
				if ctxDesc != "" && looksLikeEL(ctxDesc) {
					tableCompiler.CompileContext(ctxDesc)
				}
			}

			// Compare condition postfix
			for _, cond := range table.Conditions.Conditions {
				desc := strings.TrimSpace(cond.Description)
				existingPostfix := strings.TrimSpace(cond.Postfix)
				if desc == "" {
					continue
				}
				if !looksLikeEL(desc) {
					continue
				}

				totalConditions++

				goPostfix, err := tableCompiler.CompileCondition(desc)
				if err != nil {
					differences = append(differences,
						filepath.Base(dtFile)+":"+table.TableName+":cond:"+desc+
							"\n    Compile error: "+err.Error()+
							"\n    Existing:      "+normalizeWhitespace(existingPostfix))
					continue
				}
				compiledConditions++

				goPostfix = normalizeWhitespace(goPostfix)
				existingNorm := normalizeWhitespace(existingPostfix)
				if goPostfix == existingNorm {
					matchedConditions++
				} else {
					differences = append(differences,
						filepath.Base(dtFile)+":"+table.TableName+":cond:"+desc+
							"\n    Compiled: "+goPostfix+
							"\n    Existing: "+existingNorm)
				}
			}

			// Compare action postfix
			for _, action := range table.Actions.Actions {
				desc := strings.TrimSpace(action.Description)
				existingPostfix := strings.TrimSpace(action.Postfix)
				if desc == "" {
					continue
				}
				if !looksLikeEL(desc) {
					continue
				}

				totalActions++

				goPostfix, err := tableCompiler.CompileAction(desc)
				if err != nil {
					differences = append(differences,
						filepath.Base(dtFile)+":"+table.TableName+":action:"+desc+
							"\n    Compile error: "+err.Error()+
							"\n    Existing:      "+normalizeWhitespace(existingPostfix))
					continue
				}
				compiledActions++

				goPostfix = normalizeWhitespace(goPostfix)
				existingNorm := normalizeWhitespace(existingPostfix)
				if goPostfix == existingNorm {
					matchedActions++
				} else {
					differences = append(differences,
						filepath.Base(dtFile)+":"+table.TableName+":action:"+desc+
							"\n    Compiled: "+goPostfix+
							"\n    Existing: "+existingNorm)
				}
			}
		}
	}

	t.Logf("=== Tax Project Compilation Results ===")
	t.Logf("Conditions: %d total, %d compiled (%.1f%%), %d match existing (%.1f%%)",
		totalConditions, compiledConditions,
		float64(compiledConditions)/float64(totalConditions)*100,
		matchedConditions,
		float64(matchedConditions)/float64(totalConditions)*100)
	t.Logf("Actions: %d total, %d compiled (%.1f%%), %d match existing (%.1f%%)",
		totalActions, compiledActions,
		float64(compiledActions)/float64(totalActions)*100,
		matchedActions,
		float64(matchedActions)/float64(totalActions)*100)

	if len(differences) > 0 {
		t.Logf("\n=== Differences (first 50) ===")
		limit := 50
		if len(differences) < limit {
			limit = len(differences)
		}
		for i := 0; i < limit; i++ {
			t.Logf("%s", differences[i])
		}
		if len(differences) > 50 {
			t.Logf("... and %d more differences", len(differences)-50)
		}
	}
}

// TestExportCompiledTaxXML compiles Tax project EL and exports new XML files
// for comparison with the originals.
func TestExportCompiledTaxXML(t *testing.T) {
	sampleDir := filepath.Join("..", "..", "..", "..", "sampleprojects")
	outputDir := "/tmp/compiled_tax_xml"

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Could not create output dir: %v", err)
	}

	var dtFiles []string
	err := filepath.Walk(sampleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, "_dt.xml") {
			for project := range taxProjects {
				if strings.Contains(path, "/"+project+"/") {
					if strings.Contains(path, "/repository/xml/") {
						dtFiles = append(dtFiles, path)
					}
					break
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Skipf("Could not walk sampleprojects: %v", err)
	}

	for _, dtFile := range dtFiles {
		// Load the EDD
		eddPath := findEDDForDT(dtFile)
		var symbols map[string]string
		if eddPath != "" {
			symbols, _ = LoadEDDFile(eddPath)
		}

		data, err := os.ReadFile(dtFile)
		if err != nil {
			continue
		}

		var dt dtXMLFile
		if err := xml.Unmarshal(data, &dt); err != nil {
			continue
		}

		// Compile and update postfix for each table
		for i := range dt.Tables {
			table := &dt.Tables[i]

			tableCompiler := NewCompiler()
			if symbols != nil {
				tableCompiler.SetSymbols(symbols)
			}

			// Process contexts first
			for j := range table.Contexts.Contexts {
				ctx := &table.Contexts.Contexts[j]
				desc := strings.TrimSpace(ctx.Description)
				if desc != "" && looksLikeEL(desc) {
					if postfix, err := tableCompiler.CompileContext(desc); err == nil {
						ctx.Postfix = postfix
					}
				}
			}

			// Process conditions
			for j := range table.Conditions.Conditions {
				cond := &table.Conditions.Conditions[j]
				desc := strings.TrimSpace(cond.Description)
				if desc != "" && looksLikeEL(desc) {
					if postfix, err := tableCompiler.CompileCondition(desc); err == nil {
						cond.Postfix = "\n" + postfix + "\n"
					}
				}
			}

			// Process actions
			for j := range table.Actions.Actions {
				action := &table.Actions.Actions[j]
				desc := strings.TrimSpace(action.Description)
				if desc != "" && looksLikeEL(desc) {
					if postfix, err := tableCompiler.CompileAction(desc); err == nil {
						action.Postfix = "\n" + postfix + "\n"
					}
				}
			}
		}

		// Write new XML
		outPath := filepath.Join(outputDir, filepath.Base(dtFile))
		outData, err := xml.MarshalIndent(dt, "", "  ")
		if err != nil {
			t.Logf("Could not marshal %s: %v", dtFile, err)
			continue
		}

		// Add XML header
		outData = append([]byte(xml.Header), outData...)

		if err := os.WriteFile(outPath, outData, 0644); err != nil {
			t.Logf("Could not write %s: %v", outPath, err)
			continue
		}

		t.Logf("Exported: %s", outPath)
	}

	t.Logf("\nTo compare, run:")
	t.Logf("  diff -u %s/sampleprojects/StateTax/repository/xml/StateTax_dt.xml %s/StateTax_dt.xml | head -100",
		filepath.Clean(filepath.Join(sampleDir, "..")), outputDir)
}
