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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// TestScenarioRun_Pass verifies a scenario with correct expectations passes.
func TestScenarioRun_Pass(t *testing.T) {
	p := openExecuteProject(t)

	s := &authoring.Scenario{
		Name:       "adult eligible",
		Inputs:     map[string]any{"applicant.age": 25},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "true"},
	}

	result := s.Run(p)
	if result.Err != nil {
		t.Fatalf("Run error: %v", result.Err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true, failures: %v", result.Failures)
	}
}

// TestScenarioRun_Fail verifies a scenario with wrong expected values fails and
// reports the attribute name.
func TestScenarioRun_Fail(t *testing.T) {
	p := openExecuteProject(t)

	s := &authoring.Scenario{
		Name:       "wrong expectation",
		Inputs:     map[string]any{"applicant.age": 25},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "false"}, // wrong: should be true
	}

	result := s.Run(p)
	if result.Err != nil {
		t.Fatalf("Run error: %v", result.Err)
	}
	if result.Pass {
		t.Error("expected Pass=false for wrong expected value")
	}
	found := false
	for _, f := range result.Failures {
		if f.Attribute == "applicant.eligible" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure for applicant.eligible, got: %v", result.Failures)
	}
}

// TestScenarioRun_NumericComparison verifies int inputs compared to string
// snapshots do not cause spurious failures.
func TestScenarioRun_NumericComparison(t *testing.T) {
	p := openExecuteProject(t)

	s := &authoring.Scenario{
		Name:       "tier check",
		Inputs:     map[string]any{"applicant.score": 90},
		EntryTable: "Score_Tiers",
		Expected:   map[string]any{"applicant.tier": "A"},
	}

	result := s.Run(p)
	if result.Err != nil {
		t.Fatalf("Run error: %v", result.Err)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true, failures: %v", result.Failures)
	}
}

// TestScenarioRun_ResetsBetweenRuns ensures state from one Run does not bleed
// into the next.
func TestScenarioRun_ResetsBetweenRuns(t *testing.T) {
	p := openExecuteProject(t)

	adult := &authoring.Scenario{
		Name:       "adult",
		Inputs:     map[string]any{"applicant.age": 25},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "true"},
	}
	minor := &authoring.Scenario{
		Name:       "minor",
		Inputs:     map[string]any{"applicant.age": 15},
		EntryTable: "Check_Eligibility",
		Expected:   map[string]any{"applicant.eligible": "false"},
	}

	r1 := adult.Run(p)
	if r1.Err != nil {
		t.Fatalf("adult Run error: %v", r1.Err)
	}
	if !r1.Pass {
		t.Errorf("adult: expected Pass=true, failures: %v", r1.Failures)
	}

	// Run minor immediately after — state must be reset
	r2 := minor.Run(p)
	if r2.Err != nil {
		t.Fatalf("minor Run error: %v", r2.Err)
	}
	if !r2.Pass {
		t.Errorf("minor: expected Pass=true, failures: %v", r2.Failures)
	}
}

// TestAssertState_DirectCall exercises AssertState standalone with typed Go values.
func TestAssertState_DirectCall(t *testing.T) {
	actual := map[string]any{
		"applicant.age":      "25",
		"applicant.eligible": "true",
		"applicant.tier":     "A",
	}

	// All match (mixed types — int vs string "25")
	expected := map[string]any{
		"applicant.age":      25,
		"applicant.eligible": "true",
		"applicant.tier":     "A",
	}
	failures := authoring.AssertState(actual, expected)
	if len(failures) != 0 {
		t.Errorf("expected no failures, got: %v", failures)
	}

	// One mismatch
	expectedBad := map[string]any{
		"applicant.tier": "B",
	}
	failures = authoring.AssertState(actual, expectedBad)
	if len(failures) != 1 {
		t.Errorf("expected 1 failure, got %d: %v", len(failures), failures)
	}
	if failures[0].Attribute != "applicant.tier" {
		t.Errorf("expected failure on applicant.tier, got %q", failures[0].Attribute)
	}

	// Missing key
	expectedMissing := map[string]any{
		"applicant.unknown": "x",
	}
	failures = authoring.AssertState(actual, expectedMissing)
	if len(failures) != 1 {
		t.Errorf("expected 1 failure for missing key, got %d: %v", len(failures), failures)
	}
	if failures[0].Actual != nil {
		t.Errorf("expected Actual=nil for missing key, got %v", failures[0].Actual)
	}
}
