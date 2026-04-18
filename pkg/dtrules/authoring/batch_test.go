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

package authoring_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// writeScenario writes a scenario JSON file into dir and returns its path.
func writeScenario(t *testing.T, dir, filename string, sf authoring.ScenarioFile) {
	t.Helper()
	data, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("marshal scenario: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		t.Fatalf("write scenario file: %v", err)
	}
}

// TestRunAllScenarios_MultipleFiles checks that 3 passing scenarios all pass.
func TestRunAllScenarios_MultipleFiles(t *testing.T) {
	p := openExecuteProject(t)
	dir := t.TempDir()

	writeScenario(t, dir, "adult.json", authoring.ScenarioFile{
		Name:       "adult eligible",
		EntryTable: "Check_Eligibility",
		Inputs:     map[string]any{"applicant.age": 25},
		Expected:   map[string]any{"applicant.eligible": "true"},
	})
	writeScenario(t, dir, "minor.json", authoring.ScenarioFile{
		Name:       "minor not eligible",
		EntryTable: "Check_Eligibility",
		Inputs:     map[string]any{"applicant.age": 15},
		Expected:   map[string]any{"applicant.eligible": "false"},
	})
	writeScenario(t, dir, "tier_a.json", authoring.ScenarioFile{
		Name:       "high score tier A",
		EntryTable: "Score_Tiers",
		Inputs:     map[string]any{"applicant.score": 90},
		Expected:   map[string]any{"applicant.tier": "A"},
	})

	result, err := p.RunAllScenarios(dir)
	if err != nil {
		t.Fatalf("RunAllScenarios: %v", err)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
	if result.Passed != 3 {
		t.Errorf("expected 3 passed, got %d", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if !result.AllPassed() {
		t.Error("AllPassed should be true")
	}
}

// TestRunAllScenarios_MixedResults checks counts when 2 pass and 1 fails.
func TestRunAllScenarios_MixedResults(t *testing.T) {
	p := openExecuteProject(t)
	dir := t.TempDir()

	writeScenario(t, dir, "pass1.json", authoring.ScenarioFile{
		Name:       "adult eligible",
		EntryTable: "Check_Eligibility",
		Inputs:     map[string]any{"applicant.age": 25},
		Expected:   map[string]any{"applicant.eligible": "true"},
	})
	writeScenario(t, dir, "pass2.json", authoring.ScenarioFile{
		Name:       "minor not eligible",
		EntryTable: "Check_Eligibility",
		Inputs:     map[string]any{"applicant.age": 15},
		Expected:   map[string]any{"applicant.eligible": "false"},
	})
	writeScenario(t, dir, "fail1.json", authoring.ScenarioFile{
		Name:       "wrong expectation",
		EntryTable: "Check_Eligibility",
		Inputs:     map[string]any{"applicant.age": 25},
		Expected:   map[string]any{"applicant.eligible": "false"}, // wrong
	})

	result, err := p.RunAllScenarios(dir)
	if err != nil {
		t.Fatalf("RunAllScenarios: %v", err)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
	if result.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if result.AllPassed() {
		t.Error("AllPassed should be false")
	}
}

// TestRunAllScenarios_InvalidJSON verifies malformed JSON surfaces as Skipped
// and does not prevent other scenarios from running.
func TestRunAllScenarios_InvalidJSON(t *testing.T) {
	p := openExecuteProject(t)
	dir := t.TempDir()

	// Valid scenario
	writeScenario(t, dir, "good.json", authoring.ScenarioFile{
		Name:       "adult eligible",
		EntryTable: "Check_Eligibility",
		Inputs:     map[string]any{"applicant.age": 25},
		Expected:   map[string]any{"applicant.eligible": "true"},
	})

	// Malformed JSON
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("write bad.json: %v", err)
	}

	result, err := p.RunAllScenarios(dir)
	if err != nil {
		t.Fatalf("RunAllScenarios: %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}

	// Find the skipped result and verify it has an error
	skippedFound := false
	for _, r := range result.Results {
		if r.Err != nil {
			skippedFound = true
			break
		}
	}
	if !skippedFound {
		t.Error("expected a result with Err set for invalid JSON")
	}

	if result.AllPassed() {
		t.Error("AllPassed should be false when there are skipped scenarios")
	}
}

// TestBatchResult_Summary checks the human-readable summary string.
func TestBatchResult_Summary(t *testing.T) {
	t.Run("all pass", func(t *testing.T) {
		b := &authoring.BatchResult{Passed: 3}
		got := b.Summary()
		if got != "3 passed, 0 failed, 0 skipped" {
			t.Errorf("unexpected summary: %q", got)
		}
	})

	t.Run("with failures", func(t *testing.T) {
		s := &authoring.Scenario{Name: "bad scenario"}
		b := &authoring.BatchResult{
			Passed: 1,
			Failed: 1,
			Results: []authoring.ScenarioResult{
				{Scenario: s, Pass: false, Failures: []authoring.AssertionFailure{
					{Attribute: "applicant.eligible", Expected: "true", Actual: "false"},
				}},
			},
		}
		got := b.Summary()
		if !strings.HasPrefix(got, "1 passed, 1 failed, 0 skipped") {
			t.Errorf("unexpected summary prefix: %q", got)
		}
		if !strings.Contains(got, "bad scenario") {
			t.Errorf("expected scenario name in summary: %q", got)
		}
		if !strings.Contains(got, "applicant.eligible") {
			t.Errorf("expected attribute in summary: %q", got)
		}
	})

	t.Run("with skipped", func(t *testing.T) {
		b := &authoring.BatchResult{Passed: 2, Skipped: 1}
		got := b.Summary()
		if got != "2 passed, 0 failed, 1 skipped" {
			t.Errorf("unexpected summary: %q", got)
		}
	})
}

// TestBatchResult_AllPassed checks AllPassed logic.
func TestBatchResult_AllPassed(t *testing.T) {
	tests := []struct {
		name    string
		batch   authoring.BatchResult
		want    bool
	}{
		{"all pass", authoring.BatchResult{Passed: 3}, true},
		{"has failure", authoring.BatchResult{Passed: 2, Failed: 1}, false},
		{"has skip", authoring.BatchResult{Passed: 2, Skipped: 1}, false},
		{"has both", authoring.BatchResult{Passed: 1, Failed: 1, Skipped: 1}, false},
		{"empty", authoring.BatchResult{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.batch.AllPassed(); got != tt.want {
				t.Errorf("AllPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}
