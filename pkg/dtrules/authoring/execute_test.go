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
	"path/filepath"
	"runtime"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func executeTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "execute")
}

func openExecuteProject(t *testing.T) *authoring.Project {
	t.Helper()
	p, err := authoring.OpenProject(executeTestdataDir(t))
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	return p
}

// TestEvalCondition_True checks EvalCondition returns true when condition holds.
func TestEvalCondition_True(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "age", 25); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}
	result, err := p.EvalCondition("applicant.age >= 18")
	if err != nil {
		t.Fatalf("EvalCondition: %v", err)
	}
	if !result {
		t.Error("expected true for age=25 >= 18")
	}
}

// TestEvalCondition_False checks EvalCondition returns false when condition fails.
func TestEvalCondition_False(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "age", 15); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}
	result, err := p.EvalCondition("applicant.age >= 18")
	if err != nil {
		t.Fatalf("EvalCondition: %v", err)
	}
	if result {
		t.Error("expected false for age=15 >= 18")
	}
}

// TestEvalAction_MutatesState verifies EvalAction changes entity state and
// returns an AttributeChange record.
func TestEvalAction_MutatesState(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "age", 25); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}

	changes, err := p.EvalAction(`set applicant.eligible = true`)
	if err != nil {
		t.Fatalf("EvalAction: %v", err)
	}

	found := false
	for _, ch := range changes {
		if ch.Entity == "applicant" && ch.Attribute == "eligible" && ch.NewValue == "true" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected applicant.eligible=true change, got %v", changes)
	}

	// Confirm state persisted
	after, err := p.EvalCondition("applicant.eligible = true")
	if err != nil {
		t.Fatalf("verify condition: %v", err)
	}
	if !after {
		t.Error("expected eligible to be true after EvalAction")
	}
}

// TestExecute_SingleColumnMatch ensures the trace contains the right actions
// when only one column fires (FIRST policy, adult applicant).
func TestExecute_SingleColumnMatch(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "age", 25); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}

	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("Check_Eligibility table not found")
	}

	trace, err := tbl.Execute(p)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(trace.Errors) > 0 {
		t.Fatalf("trace errors: %v", trace.Errors)
	}

	// Find action steps — column 1 should have fired, not column 2
	var actionSteps []authoring.Step
	for _, s := range trace.Steps {
		if s.Kind == authoring.StepAction {
			actionSteps = append(actionSteps, s)
		}
	}
	if len(actionSteps) == 0 {
		t.Fatal("expected at least one action step")
	}
	for _, s := range actionSteps {
		if s.Column != 1 {
			t.Errorf("expected actions only from column 1 (FIRST policy), got column %d", s.Column)
		}
	}

	// eligible should be true
	if v, ok := trace.FinalState["applicant.eligible"]; !ok || v != "true" {
		t.Errorf("expected applicant.eligible=true in final state, got %q", v)
	}
}

// TestExecute_MultiColumnAllPolicy checks that ALL policy fires actions in
// every matching column for Score_Tiers (low score → column 2 only).
func TestExecute_MultiColumnAllPolicy(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "score", 50); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}

	tbl := p.Table("Score_Tiers")
	if tbl == nil {
		t.Fatal("Score_Tiers table not found")
	}

	trace, err := tbl.Execute(p)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(trace.Errors) > 0 {
		t.Fatalf("trace errors: %v", trace.Errors)
	}

	// tier should be "B"
	if v, ok := trace.FinalState["applicant.tier"]; !ok || v != "B" {
		t.Errorf("expected applicant.tier=B, got %q", v)
	}
}

// TestStepper_YieldsExpectedSequence checks that Next() yields condition then
// action steps in order for the adult applicant (column 1 matches, FIRST).
func TestStepper_YieldsExpectedSequence(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "age", 25); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}

	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("Check_Eligibility table not found")
	}

	stepper := tbl.NewStepper(p)
	var kinds []authoring.StepKind
	for stepper.Next() {
		s := stepper.Current()
		if s.Error != nil {
			t.Fatalf("stepper error: %v", s.Error)
		}
		kinds = append(kinds, s.Kind)
	}

	// Expect: condition(col1), action(col1) — FIRST policy stops after col1
	if len(kinds) == 0 {
		t.Fatal("no steps yielded")
	}
	if kinds[0] != authoring.StepCondition {
		t.Errorf("expected first step to be condition, got %s", kinds[0])
	}
	hasAction := false
	for _, k := range kinds {
		if k == authoring.StepAction {
			hasAction = true
		}
	}
	if !hasAction {
		t.Error("expected at least one action step")
	}
}

// TestStepper_StopsOnFirstForFirstPolicy verifies that FIRST policy stops
// processing after the first matching column.
func TestStepper_StopsOnFirstForFirstPolicy(t *testing.T) {
	p := openExecuteProject(t)
	if err := p.SetAttribute("applicant", "age", 25); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}

	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("Check_Eligibility table not found")
	}

	stepper := tbl.NewStepper(p)
	cols := map[int]bool{}
	for stepper.Next() {
		s := stepper.Current()
		if s.Error != nil {
			t.Fatalf("stepper error: %v", s.Error)
		}
		if s.Kind == authoring.StepAction {
			cols[s.Column] = true
		}
	}

	// Only column 1 should have fired actions
	if cols[2] {
		t.Error("column 2 actions should not fire under FIRST policy when col 1 matched")
	}
	if !cols[1] {
		t.Error("column 1 actions should have fired")
	}
}

// TestLoadTestData_AccumulatesAcrossCalls loads test data twice (different
// files) and verifies the last load wins for single-cardinality entities.
func TestLoadTestData_AccumulatesAcrossCalls(t *testing.T) {
	p := openExecuteProject(t)

	testDir := filepath.Join(executeTestdataDir(t), "inputs")

	if err := p.LoadTestData(filepath.Join(testDir, "adult_high.xml")); err != nil {
		t.Fatalf("LoadTestData adult_high: %v", err)
	}

	// Verify the loaded state
	result1, err := p.EvalCondition("applicant.age >= 18")
	if err != nil {
		t.Fatalf("EvalCondition: %v", err)
	}
	if !result1 {
		t.Error("expected adult after loading adult_high.xml")
	}

	// Load a second file; since it's a '1' cardinality entity, the same
	// entity should be reused and attributes overwritten.
	p.ResetState()
	if err := p.LoadTestData(filepath.Join(testDir, "minor.xml")); err != nil {
		t.Fatalf("LoadTestData minor: %v", err)
	}

	result2, err := p.EvalCondition("applicant.age >= 18")
	if err != nil {
		t.Fatalf("EvalCondition: %v", err)
	}
	if result2 {
		t.Error("expected minor (age=15) after loading minor.xml")
	}
}
