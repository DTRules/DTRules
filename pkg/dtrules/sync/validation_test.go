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
