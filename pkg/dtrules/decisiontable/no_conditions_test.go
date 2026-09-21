// Copyright 2026 Paul Snow
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

package decisiontable

import (
	"strings"
	"testing"
)

// A table with no conditions has no condition to select a column (#1230).
// Under FIRST and ALL the engine builds no tree and runs no column; under
// BALANCED (the default when no policy is given) it runs column 1 only.
// Execution against the real loader is pinned in
// pkg/dtrules/no_conditions_policy_test.go.

func noCondWarnings(ws []Warning) []Warning {
	var out []Warning
	for _, w := range ws {
		if w.Kind == KindColumnActionsWithoutConditions {
			out = append(out, w)
		}
	}
	return out
}

func TestAnalyze_ColumnActionsWithoutConditions_FirstAll(t *testing.T) {
	for _, policy := range []string{"FIRST", "ALL", "all"} {
		ws := noCondWarnings(Analyze(Inputs{
			Name:   "NoCond",
			Policy: policy,
			Actions: makeActions(
				[]string{`set result.table = "nocond"`, `set result.b = 2`, `set result.c = 3`},
				[][]string{{"X", ""}, {"", "X"}, {"", ""}}),
			MaxCol: 2,
		}))
		if len(ws) != 1 {
			t.Fatalf("%s: want one warning, got %v", policy, ws)
		}
		r := ws[0].Reason
		if !strings.Contains(r, "actions 1, 2 marked") || !strings.Contains(r, "never run") || !strings.Contains(r, "initial_actions") {
			t.Errorf("%s: reason should name actions 1 and 2 (not the unmarked 3) and point at initial_actions: %q", policy, r)
		}
	}
}

// Under BALANCED, or with no policy (which loads as BALANCED), column 1 runs.
// Only an action marked in later columns and not in column 1 is dead.
func TestAnalyze_ColumnActionsWithoutConditions_Balanced(t *testing.T) {
	for _, policy := range []string{"", "BALANCED"} {
		acts := makeActions(
			[]string{`set result.a = 1`, `set result.b = 2`, `set result.c = 3`},
			[][]string{{"X", ""}, {"", "X"}, {"X", "X"}})
		ws := noCondWarnings(Analyze(Inputs{Name: "NoCond", Policy: policy, Actions: acts, MaxCol: 2}))
		if len(ws) != 1 {
			t.Fatalf("policy %q: want one warning, got %v", policy, ws)
		}
		r := ws[0].Reason
		if !strings.Contains(r, "action 2 marked") || !strings.Contains(r, "column 1") {
			t.Errorf("policy %q: reason should name only action 2 and say column 1 runs: %q", policy, r)
		}

		// Actions in column 1 only: that is a working table, no warning.
		ws = noCondWarnings(Analyze(Inputs{Name: "NoCond", Policy: policy,
			Actions: makeActions([]string{`set result.a = 1`}, [][]string{{"X"}}), MaxCol: 1}))
		if len(ws) != 0 {
			t.Errorf("policy %q: column-1-only table warned: %v", policy, ws)
		}
	}
}

// The other checks still run on a table with no conditions: a column with
// no actions is still reported as a no-op column.
func TestAnalyze_NoConditions_KeepsOtherChecks(t *testing.T) {
	ws := Analyze(Inputs{
		Name:    "Skeleton",
		Policy:  "FIRST",
		Actions: makeActions([]string{`// placeholder`}, [][]string{{""}}),
		MaxCol:  1,
	})
	found := false
	for _, w := range ws {
		if w.Kind == "no-op column" && w.Column == 1 {
			found = true
		}
		if w.Kind == KindColumnActionsWithoutConditions {
			t.Errorf("no action is marked, yet: %v", w)
		}
	}
	if !found {
		t.Errorf("no-op column warning suppressed for a table with no conditions: %v", ws)
	}
}

// One condition — even `true` — makes the columns reachable, and the
// warning must not fire.
func TestAnalyze_ColumnActionsWithOneCondition_NoWarning(t *testing.T) {
	warns := Analyze(Inputs{
		Name:       "OneCond",
		Policy:     "FIRST",
		Conditions: makeConditions([]string{"true"}, [][]string{{"Y"}}),
		Actions:    makeActions([]string{`perform Other`}, [][]string{{"X"}}),
		MaxCol:     1,
	})
	if ws := noCondWarnings(warns); len(ws) != 0 {
		t.Errorf("warning fired on a table with a condition: %v", ws)
	}
}
