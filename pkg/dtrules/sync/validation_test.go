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

func TestValidateELCompliance_Compliant(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a compliant DT file (has descriptions and postfix)
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table el_compiled="true">
        <table_name>Test_Table</table_name>
        <conditions>
            <condition_details>
                <condition_description>input.age >= 18</condition_description>
                <condition_postfix>input.age 18 >=</condition_postfix>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_description>set result.eligible = true</action_description>
                <action_postfix>true result.eligible =</action_postfix>
            </action_details>
        </actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "test_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	if !result.IsCompliant() {
		t.Errorf("Expected compliant result, got errors: %v", result.Errors)
	}

	if len(result.CompliantFiles) != 1 {
		t.Errorf("Expected 1 compliant file, got %d", len(result.CompliantFiles))
	}

	if len(result.LegacyFiles) != 0 {
		t.Errorf("Expected 0 legacy files, got %d", len(result.LegacyFiles))
	}
}

func TestValidateELCompliance_Legacy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a legacy DT file (has postfix but no descriptions)
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Legacy_Table</table_name>
        <conditions>
            <condition_details>
                <condition_description></condition_description>
                <condition_postfix>input.age 18 >=</condition_postfix>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_description></action_description>
                <action_postfix>true result.eligible =</action_postfix>
            </action_details>
        </actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "legacy_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	if result.IsCompliant() {
		t.Error("Expected non-compliant result for legacy file")
	}

	if len(result.LegacyFiles) != 1 {
		t.Errorf("Expected 1 legacy file, got %d", len(result.LegacyFiles))
	}

	if !result.HasLegacyPostfix() {
		t.Error("Expected HasLegacyPostfix() to return true")
	}
}

func TestValidateELCompliance_NeedsCompilation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a DT file with descriptions but no postfix (needs compilation)
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Uncompiled_Table</table_name>
        <conditions>
            <condition_details>
                <condition_description>input.age >= 18</condition_description>
                <condition_postfix></condition_postfix>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_description>set result.eligible = true</action_description>
                <action_postfix></action_postfix>
            </action_details>
        </actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "uncompiled_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	// Should be compliant (no legacy postfix), but needs compilation
	if !result.IsCompliant() {
		t.Errorf("Expected compliant (no legacy), but got errors: %v", result.Errors)
	}

	if len(result.NeedsCompilationFiles) != 1 {
		t.Errorf("Expected 1 file needing compilation, got %d", len(result.NeedsCompilationFiles))
	}
}

func TestIsLegacyPostfixFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a legacy file
	legacyContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Legacy</table_name>
        <conditions>
            <condition_details>
                <condition_postfix>a b =</condition_postfix>
            </condition_details>
        </conditions>
    </decision_table>
</decision_tables>`

	legacyPath := filepath.Join(tmpDir, "legacy_dt.xml")
	os.WriteFile(legacyPath, []byte(legacyContent), 0644)

	isLegacy, err := IsLegacyPostfixFile(legacyPath)
	if err != nil {
		t.Fatalf("IsLegacyPostfixFile failed: %v", err)
	}

	if !isLegacy {
		t.Error("Expected file to be detected as legacy")
	}

	// Create a compliant file
	compliantContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Compliant</table_name>
        <conditions>
            <condition_details>
                <condition_description>a == b</condition_description>
                <condition_postfix>a b =</condition_postfix>
            </condition_details>
        </conditions>
    </decision_table>
</decision_tables>`

	compliantPath := filepath.Join(tmpDir, "compliant_dt.xml")
	os.WriteFile(compliantPath, []byte(compliantContent), 0644)

	isLegacy, err = IsLegacyPostfixFile(compliantPath)
	if err != nil {
		t.Fatalf("IsLegacyPostfixFile failed: %v", err)
	}

	if isLegacy {
		t.Error("Expected file to NOT be detected as legacy")
	}
}

func TestELComplianceStatus_String(t *testing.T) {
	tests := []struct {
		status   ELComplianceStatus
		expected string
	}{
		{StatusCompliant, "compliant"},
		{StatusLegacy, "legacy"},
		{StatusNeedsCompilation, "needs_compilation"},
		{StatusEmpty, "empty"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("%v.String() = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

func TestValidateELCompliance_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	// No files, so should be compliant (nothing to violate)
	if !result.IsCompliant() {
		t.Errorf("Expected compliant result for empty directory, got errors: %v", result.Errors)
	}

	if len(result.FileResults) != 0 {
		t.Errorf("Expected 0 file results, got %d", len(result.FileResults))
	}
}

func TestValidateELCompliance_EmptyTable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a DT file with no conditions or actions
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Empty_Table</table_name>
        <conditions></conditions>
        <actions></actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "empty_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	// Empty tables should be compliant (nothing to check)
	if !result.IsCompliant() {
		t.Errorf("Expected compliant result for empty table, got errors: %v", result.Errors)
	}

	if len(result.FileResults) != 1 {
		t.Errorf("Expected 1 file result, got %d", len(result.FileResults))
	}

	if result.FileResults[0].Status != StatusEmpty {
		t.Errorf("Expected StatusEmpty, got %v", result.FileResults[0].Status)
	}
}

func TestValidateELCompliance_ContextsAndInitialActions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a DT file with contexts and initial actions
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table el_compiled="true">
        <table_name>Full_Table</table_name>
        <contexts>
            <context_details>
                <context_description>input.data</context_description>
                <context_postfix>input.data</context_postfix>
            </context_details>
        </contexts>
        <initial_actions>
            <initial_action_details>
                <initial_action_description>set temp.value = 0</initial_action_description>
                <initial_action_postfix>0 temp.value =</initial_action_postfix>
            </initial_action_details>
        </initial_actions>
        <conditions>
            <condition_details>
                <condition_description>input.age >= 18</condition_description>
                <condition_postfix>input.age 18 >=</condition_postfix>
            </condition_details>
        </conditions>
        <actions>
            <action_details>
                <action_description>set result.eligible = true</action_description>
                <action_postfix>true result.eligible =</action_postfix>
            </action_details>
        </actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "full_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	if !result.IsCompliant() {
		t.Errorf("Expected compliant result, got errors: %v", result.Errors)
	}

	if len(result.FileResults) != 1 {
		t.Fatalf("Expected 1 file result, got %d", len(result.FileResults))
	}

	if len(result.FileResults[0].Tables) != 1 {
		t.Fatalf("Expected 1 table result, got %d", len(result.FileResults[0].Tables))
	}

	table := result.FileResults[0].Tables[0]
	if !table.HasELContexts {
		t.Error("Expected HasELContexts to be true")
	}
	if !table.HasELConditions {
		t.Error("Expected HasELConditions to be true")
	}
	if !table.HasELActions {
		t.Error("Expected HasELActions to be true")
	}
}

func TestValidationError_Error(t *testing.T) {
	// With table name
	err := &ValidationError{
		FilePath:    "test.xml",
		TableName:   "TestTable",
		ElementType: "condition",
		RowNumber:   5,
		Message:     "test error",
	}
	expected := "test.xml (table TestTable, condition row 5): test error"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}

	// Without table name
	err2 := &ValidationError{
		FilePath: "test.xml",
		Message:  "file error",
	}
	expected2 := "test.xml: file error"
	if got := err2.Error(); got != expected2 {
		t.Errorf("Error() = %q, want %q", got, expected2)
	}
}

func TestValidationWarning_String(t *testing.T) {
	// With table name
	warn := &ValidationWarning{
		FilePath:  "test.xml",
		TableName: "TestTable",
		Message:   "test warning",
	}
	expected := "test.xml (table TestTable): test warning"
	if got := warn.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}

	// Without table name
	warn2 := &ValidationWarning{
		FilePath: "test.xml",
		Message:  "file warning",
	}
	expected2 := "test.xml: file warning"
	if got := warn2.String(); got != expected2 {
		t.Errorf("String() = %q, want %q", got, expected2)
	}
}

func TestValidateELCompliance_MultipleTables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a DT file with multiple tables (mixed compliance)
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table el_compiled="true">
        <table_name>Compliant_Table</table_name>
        <conditions>
            <condition_details>
                <condition_description>input.age >= 18</condition_description>
                <condition_postfix>input.age 18 >=</condition_postfix>
            </condition_details>
        </conditions>
        <actions></actions>
    </decision_table>
    <decision_table>
        <table_name>Legacy_Table</table_name>
        <conditions>
            <condition_details>
                <condition_description></condition_description>
                <condition_postfix>a b =</condition_postfix>
            </condition_details>
        </conditions>
        <actions></actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "mixed_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	// Should not be compliant due to legacy table
	if result.IsCompliant() {
		t.Error("Expected non-compliant result for file with legacy table")
	}

	if len(result.LegacyFiles) != 1 {
		t.Errorf("Expected 1 legacy file, got %d", len(result.LegacyFiles))
	}

	// Should have one error for the legacy table
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error for legacy table")
	}
}

func TestValidateELCompliance_NameAttribute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a DT file using name attribute instead of table_name element
	dtContent := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table name="AttrNameTable" el_compiled="true">
        <conditions>
            <condition_details>
                <condition_description>input.age >= 18</condition_description>
                <condition_postfix>input.age 18 >=</condition_postfix>
            </condition_details>
        </conditions>
        <actions></actions>
    </decision_table>
</decision_tables>`

	os.WriteFile(filepath.Join(tmpDir, "attr_dt.xml"), []byte(dtContent), 0644)

	result, err := ValidateELCompliance(tmpDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	if len(result.FileResults) != 1 || len(result.FileResults[0].Tables) != 1 {
		t.Fatal("Expected 1 file with 1 table")
	}

	tableName := result.FileResults[0].Tables[0].TableName
	if tableName != "AttrNameTable" {
		t.Errorf("Expected table name 'AttrNameTable', got %q", tableName)
	}
}
