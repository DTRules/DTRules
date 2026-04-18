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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// TestDiff_IdenticalProjects_NoDivergence opens the same project twice and
// asserts no divergences.
func TestDiff_IdenticalProjects_NoDivergence(t *testing.T) {
	p1 := openExecuteProject(t)
	p2 := openExecuteProject(t)

	scenarios := []*authoring.Scenario{
		{
			Name:       "adult eligible",
			Inputs:     map[string]any{"applicant.age": 25},
			EntryTable: "Check_Eligibility",
			Expected:   map[string]any{"applicant.eligible": "true"},
		},
		{
			Name:       "minor denied",
			Inputs:     map[string]any{"applicant.age": 15},
			EntryTable: "Check_Eligibility",
			Expected:   map[string]any{"applicant.eligible": "false"},
		},
	}

	report := authoring.Diff(p1, p2, scenarios)

	if report.Total != 2 {
		t.Errorf("expected Total=2, got %d", report.Total)
	}
	if report.Matching != 2 {
		t.Errorf("expected Matching=2, got %d", report.Matching)
	}
	if len(report.Diverged) != 0 {
		t.Errorf("expected no divergences, got %d: %v", len(report.Diverged), report.Diverged)
	}
}

// TestDiff_ChangedRuleTrigger_NamedDivergence opens two copies of the project,
// mutates p2 so the adult scenario sets eligible=false instead of true, and
// asserts the report names that divergence.
func TestDiff_ChangedRuleTrigger_NamedDivergence(t *testing.T) {
	p1 := openExecuteProject(t)
	p2 := openExecuteProject(t)

	// Flip the action in p2: column 1 (adult → eligible=true) becomes eligible=false
	tbl2 := p2.Table("Check_Eligibility")
	if tbl2 == nil {
		t.Fatal("Check_Eligibility not found in p2")
	}

	err := tbl2.UpdateAction(1, authoring.Action{
		Number:  1,
		Comment: "Deny eligibility (mutated)",
		DSL:     `set applicant.eligible = false`,
		Columns: map[int]bool{1: true},
	})
	if err != nil {
		t.Fatalf("UpdateAction: %v", err)
	}

	// Invalidate p2's exec state so it picks up the mutation
	p2.ResetState()

	adultScenario := &authoring.Scenario{
		Name:       "adult eligible",
		Inputs:     map[string]any{"applicant.age": 25},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "true"},
	}
	minorScenario := &authoring.Scenario{
		Name:       "minor denied",
		Inputs:     map[string]any{"applicant.age": 15},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "false"},
	}

	report := authoring.Diff(p1, p2, []*authoring.Scenario{adultScenario, minorScenario})

	if report.Total != 2 {
		t.Errorf("expected Total=2, got %d", report.Total)
	}
	if len(report.Diverged) != 1 {
		t.Fatalf("expected 1 diverged scenario, got %d", len(report.Diverged))
	}

	div := report.Diverged[0]
	if div.Scenario.Name != "adult eligible" {
		t.Errorf("expected diverged scenario %q, got %q", "adult eligible", div.Scenario.Name)
	}

	found := false
	for _, ad := range div.Diffs {
		if ad.Attribute == "applicant.eligible" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected AttributeDiff for applicant.eligible, got: %v", div.Diffs)
	}
}

// TestDiff_ScenarioFailsInBothProjects verifies that a scenario that errors in
// both projects is excluded from Total, Matching, and Diverged.
func TestDiff_ScenarioFailsInBothProjects(t *testing.T) {
	p1 := openExecuteProject(t)
	p2 := openExecuteProject(t)

	// Reference a non-existent entry table — both projects will error
	badScenario := &authoring.Scenario{
		Name:       "bad entry",
		Inputs:     map[string]any{"applicant.age": 25},
		EntryTable: "NonExistentTable",
		Expected:   map[string]any{},
	}
	goodScenario := &authoring.Scenario{
		Name:       "good",
		Inputs:     map[string]any{"applicant.age": 25},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "true"},
	}

	report := authoring.Diff(p1, p2, []*authoring.Scenario{badScenario, goodScenario})

	// badScenario errored in both → excluded; goodScenario is fine → Total=1
	if report.Total != 1 {
		t.Errorf("expected Total=1 (shared-failure excluded), got %d", report.Total)
	}
	if report.Matching != 1 {
		t.Errorf("expected Matching=1, got %d", report.Matching)
	}
	if len(report.Diverged) != 0 {
		t.Errorf("expected no divergences, got %d", len(report.Diverged))
	}
}

// TestDiffSummary_Format checks the Summary string format.
func TestDiffSummary_Format(t *testing.T) {
	p1 := openExecuteProject(t)
	p2 := openExecuteProject(t)

	scenarios := []*authoring.Scenario{
		{
			Name:       "adult",
			Inputs:     map[string]any{"applicant.age": 25},
			EntryTable: "Check_Eligibility",
			Expected:   map[string]any{"applicant.eligible": "true"},
		},
	}

	report := authoring.Diff(p1, p2, scenarios)
	summary := report.Summary()

	if !strings.Contains(summary, "1 total") {
		t.Errorf("summary missing '1 total': %q", summary)
	}
	if !strings.Contains(summary, "1 matching") {
		t.Errorf("summary missing '1 matching': %q", summary)
	}
	if !strings.Contains(summary, "0 diverged") {
		t.Errorf("summary missing '0 diverged': %q", summary)
	}
}

// TestDiffSummary_DivergenceBlock checks that diverged scenarios appear in the
// summary with their attribute diffs.
func TestDiffSummary_DivergenceBlock(t *testing.T) {
	p1 := openExecuteProject(t)
	p2 := openExecuteProject(t)

	// Mutate p2 so adult scenario diverges
	tbl2 := p2.Table("Check_Eligibility")
	if tbl2 == nil {
		t.Fatal("Check_Eligibility not found in p2")
	}
	if err := tbl2.UpdateAction(1, authoring.Action{
		Number:  1,
		Comment: "mutated",
		DSL:     `set applicant.eligible = false`,
		Columns: map[int]bool{1: true},
	}); err != nil {
		t.Fatalf("UpdateAction: %v", err)
	}
	p2.ResetState()

	scenarios := []*authoring.Scenario{
		{
			Name:       "adult",
			Inputs:     map[string]any{"applicant.age": 25},
			EntryTable: "Check_Eligibility",
			Expected:   map[string]any{"applicant.eligible": "true"},
		},
	}

	report := authoring.Diff(p1, p2, scenarios)
	summary := report.Summary()

	if !strings.Contains(summary, "1 diverged") {
		t.Errorf("summary missing '1 diverged': %q", summary)
	}
	if !strings.Contains(summary, "adult") {
		t.Errorf("summary missing scenario name 'adult': %q", summary)
	}
	if !strings.Contains(summary, "applicant.eligible") {
		t.Errorf("summary missing attribute 'applicant.eligible': %q", summary)
	}
}
