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

func debugTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "debug")
}

func openDebugProject(t *testing.T) *authoring.Project {
	t.Helper()
	p, err := authoring.OpenProject(debugTestdataDir(t))
	if err != nil {
		t.Fatalf("OpenProject debug: %v", err)
	}
	return p
}

// loadDebugTestData loads the given test input file into the debug project.
func loadDebugTestData(t *testing.T, p *authoring.Project, filename string) {
	t.Helper()
	path := filepath.Join(debugTestdataDir(t), "inputs", filename)
	if err := p.LoadTestData(path); err != nil {
		t.Fatalf("LoadTestData %s: %v", filename, err)
	}
}

// TestExecuteEntry_CapturesSubTables verifies that ExecuteEntry records all
// table invocations in pre-order: entry table at depth 0, sub-tables at depth 1.
func TestExecuteEntry_CapturesSubTables(t *testing.T) {
	p := openDebugProject(t)
	loadDebugTestData(t, p, "adult_high.xml") // age=25, score=90 → eligible=true, Score_Tiers

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}

	// Expect: Check_Eligibility (depth 0), Score_Tiers (depth 1).
	// adult_high: age>=18 → column1 fires → sets eligible=true, then perform Score_Tiers.
	if len(trace.Invocations) < 2 {
		t.Fatalf("expected at least 2 invocations, got %d: %v", len(trace.Invocations), trace.Invocations)
	}

	if trace.Invocations[0].TableName != "Check_Eligibility" {
		t.Errorf("invocation 0: want Check_Eligibility, got %s", trace.Invocations[0].TableName)
	}
	if trace.Invocations[0].Depth != 0 {
		t.Errorf("invocation 0 depth: want 0, got %d", trace.Invocations[0].Depth)
	}

	// Second invocation should be Score_Tiers at depth 1.
	found := false
	for _, inv := range trace.Invocations[1:] {
		if inv.TableName == "Score_Tiers" && inv.Depth == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Score_Tiers at depth 1 in invocations: %v", trace.Invocations)
	}

	// Verify indices are sequential.
	for i, inv := range trace.Invocations {
		if inv.Index != i {
			t.Errorf("invocation %d has Index=%d, want %d", i, inv.Index, i)
		}
	}

	// Verify StartState is populated.
	for _, inv := range trace.Invocations {
		if inv.StartState == nil {
			t.Errorf("invocation %s has nil StartState", inv.TableName)
		}
		if inv.EndState == nil {
			t.Errorf("invocation %s has nil EndState", inv.TableName)
		}
	}
}

// TestResumeAt_PauseBeforeIndex verifies that ResumeAt pauses at the correct
// invocation and EntityStack reflects the state at that pause point.
func TestResumeAt_PauseBeforeIndex(t *testing.T) {
	p := openDebugProject(t)
	loadDebugTestData(t, p, "adult_high.xml")

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}
	if len(trace.Invocations) < 2 {
		t.Fatalf("need at least 2 invocations, got %d", len(trace.Invocations))
	}

	// Pause before invocation 1 (first sub-table call).
	dbg, err := p.ResumeAt(trace, 1)
	if err != nil {
		t.Fatalf("ResumeAt(trace,1): %v", err)
	}
	defer dbg.Close()

	next := dbg.NextInvocation()
	if next == nil {
		t.Fatal("NextInvocation returned nil")
	}
	if next.Index != 1 {
		t.Errorf("NextInvocation().Index: want 1, got %d", next.Index)
	}

	// EntityStack should be non-empty (applicant entity pushed from test data).
	stack := dbg.EntityStack()
	foundApplicant := false
	for _, view := range stack {
		if view.Name == "applicant" {
			foundApplicant = true
		}
	}
	if !foundApplicant {
		t.Errorf("expected applicant entity on stack, got: %v", stack)
	}
}

// TestStep_RunsOneInvocation verifies that Step() executes the paused invocation
// and returns it with EndState populated, then pauses before the next one.
func TestStep_RunsOneInvocation(t *testing.T) {
	p := openDebugProject(t)
	loadDebugTestData(t, p, "adult_high.xml")

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}
	if len(trace.Invocations) < 2 {
		t.Fatalf("need at least 2 invocations, got %d", len(trace.Invocations))
	}

	dbg, err := p.ResumeAt(trace, 0)
	if err != nil {
		t.Fatalf("ResumeAt(trace,0): %v", err)
	}
	defer dbg.Close()

	// Step executes invocation 0 (Check_Eligibility).
	stepped, err := dbg.Step()
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if stepped == nil {
		t.Fatal("Step returned nil invocation")
	}
	if stepped.TableName != "Check_Eligibility" {
		t.Errorf("stepped.TableName: want Check_Eligibility, got %s", stepped.TableName)
	}
	if stepped.EndState == nil {
		t.Error("stepped.EndState should be populated")
	}

	// After stepping the entry table (which includes its sub-table calls),
	// pauseIndex should be past all depth>0 sub-invocations.
	next := dbg.NextInvocation()
	if next != nil {
		// If there's a next, it should be at depth 0 (a new top-level invocation).
		if next.Depth != 0 {
			t.Errorf("after step, NextInvocation().Depth: want 0, got %d", next.Depth)
		}
	}
}

// TestContinue_ReproducesOriginalTrace verifies that Continue without mutations
// produces a trace that executes the same tables in the same order.
func TestContinue_ReproducesOriginalTrace(t *testing.T) {
	p := openDebugProject(t)
	loadDebugTestData(t, p, "adult_high.xml")

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}
	if len(trace.Invocations) < 2 {
		t.Fatalf("need at least 2 invocations, got %d", len(trace.Invocations))
	}

	// ResumeAt invocation 1 (the first sub-table), then Continue without mutation.
	dbg, err := p.ResumeAt(trace, 1)
	if err != nil {
		t.Fatalf("ResumeAt(trace,1): %v", err)
	}
	defer dbg.Close()

	continuedTrace, err := dbg.Continue()
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}

	// The continued trace should include the sub-table(s) starting at index 1.
	if len(continuedTrace.Invocations) == 0 {
		t.Error("expected invocations in continued trace")
	}

	// Final state should match the original trace's final state.
	for key, origVal := range trace.FinalState {
		if contVal, ok := continuedTrace.FinalState[key]; ok {
			if origVal != contVal {
				t.Errorf("FinalState[%s]: original=%s, continued=%s", key, origVal, contVal)
			}
		}
	}
}

// TestContinue_DivergesAfterStateMutation verifies that mutating state in a
// DebugSession before Continue produces a different trace.
func TestContinue_DivergesAfterStateMutation(t *testing.T) {
	p := openDebugProject(t)
	// minor_high: age=15, score=95 → Check_Eligibility column2 (minor) fires,
	// then performs Check_Bonus.
	loadDebugTestData(t, p, "minor_high.xml")

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}
	if len(trace.Invocations) < 2 {
		t.Fatalf("need at least 2 invocations, got %d", len(trace.Invocations))
	}

	// Pause at invocation 1 (Check_Bonus sub-table), mutate score to below 90
	// so Check_Bonus takes a different branch.
	dbg, err := p.ResumeAt(trace, 1)
	if err != nil {
		t.Fatalf("ResumeAt(trace,1): %v", err)
	}
	defer dbg.Close()

	// Mutate: score=50 means Check_Bonus column2 fires (no bonus) instead of column1.
	if err := dbg.SetAttribute("applicant", "score", 50); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}

	continuedTrace, err := dbg.Continue()
	if err != nil {
		t.Fatalf("Continue after mutation: %v", err)
	}

	// With score=50, Check_Bonus should set bonus=false.
	// With score=95 (original), it set bonus=true.
	if bonusVal, ok := continuedTrace.FinalState["applicant.bonus"]; ok {
		if bonusVal != "false" {
			t.Errorf("expected bonus=false after mutation (score=50), got %s", bonusVal)
		}
	} else {
		// bonus may not appear in FinalState if it wasn't changed; check via Resolve.
		val, _, err := dbg.Resolve("applicant.bonus")
		if err != nil {
			t.Errorf("Resolve applicant.bonus: %v", err)
		} else if val != "false" {
			t.Errorf("expected bonus=false after mutation, got %v", val)
		}
	}
}

// TestResolve_PostAndPreState verifies that Resolve returns the current live value.
func TestResolve_PostAndPreState(t *testing.T) {
	p := openDebugProject(t)
	loadDebugTestData(t, p, "adult_high.xml") // age=25

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}

	// Pause at the start (before invocation 0).
	dbg, err := p.ResumeAt(trace, 0)
	if err != nil {
		t.Fatalf("ResumeAt(trace,0): %v", err)
	}
	defer dbg.Close()

	// Before stepping, age should be 25.
	val, entName, err := dbg.Resolve("applicant.age")
	if err != nil {
		t.Fatalf("Resolve applicant.age: %v", err)
	}
	if val != "25" {
		t.Errorf("pre-step age: want 25, got %v", val)
	}
	if entName != "applicant" {
		t.Errorf("pre-step entity: want applicant, got %s", entName)
	}

	// Step to run Check_Eligibility.
	if _, err := dbg.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}

	// After step, eligible should be set.
	val2, _, err := dbg.Resolve("applicant.eligible")
	if err != nil {
		t.Fatalf("Resolve applicant.eligible post-step: %v", err)
	}
	if val2 != "true" {
		t.Errorf("post-step eligible: want true, got %v", val2)
	}
}

// TestResumeAt_InvalidIndex verifies that out-of-range indices return errors.
func TestResumeAt_InvalidIndex(t *testing.T) {
	p := openDebugProject(t)
	loadDebugTestData(t, p, "adult_high.xml")

	trace, err := p.ExecuteEntry("Check_Eligibility")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}

	cases := []int{-1, len(trace.Invocations), len(trace.Invocations) + 5}
	for _, idx := range cases {
		_, err := p.ResumeAt(trace, idx)
		if err == nil {
			t.Errorf("ResumeAt(trace, %d): expected error, got nil", idx)
		}
	}
}
