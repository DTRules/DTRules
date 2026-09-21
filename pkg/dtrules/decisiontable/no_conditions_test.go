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

// A table with no conditions has no decision tree: ExecuteTable runs its
// initial_actions and nothing else. Actions marked in a column of such a
// table can never run, and the table compiled without a word (#1230).

func TestAnalyze_ColumnActionsWithoutConditions(t *testing.T) {
	warns := Analyze(Inputs{
		Name:    "NoCond",
		Policy:  "ALL",
		Actions: makeActions([]string{`set result.table = "nocond-ALL"`}, [][]string{{"X"}}),
		MaxCol:  1,
	})
	if len(warns) != 1 {
		t.Fatalf("want exactly one warning, got %d: %v", len(warns), warns)
	}
	w := warns[0]
	if w.Kind != KindColumnActionsWithoutConditions {
		t.Errorf("kind = %q, want %q", w.Kind, KindColumnActionsWithoutConditions)
	}
	if !strings.Contains(w.Reason, "initial_actions") || !strings.Contains(w.Reason, "never run") {
		t.Errorf("reason does not say the actions never run and where they belong: %q", w.Reason)
	}
	if !strings.Contains(w.Reason, "action 1") {
		t.Errorf("reason does not name the stranded action: %q", w.Reason)
	}
}

// The column checks all presume columns are selected by conditions. With no
// conditions none of them means anything, and "assignment-only table" in
// particular would advise inlining a table that does nothing.
func TestAnalyze_ColumnActionsWithoutConditions_Only(t *testing.T) {
	warns := Analyze(Inputs{
		Name: "NoCond",
		Actions: makeActions(
			[]string{`set result.a = 1`, `set result.b = 2`},
			[][]string{{"X", ""}, {"", ""}}),
		MaxCol: 2,
	})
	if len(warns) != 1 || warns[0].Kind != KindColumnActionsWithoutConditions {
		t.Fatalf("want only the no-conditions warning, got %v", warns)
	}
	if !strings.Contains(warns[0].Reason, "action 1") || strings.Contains(warns[0].Reason, "action 2") {
		t.Errorf("reason should name action 1 (marked) and not action 2 (unmarked): %q", warns[0].Reason)
	}
}

// One condition — even `true` — makes the columns reachable, and the
// warning must not fire.
func TestAnalyze_ColumnActionsWithOneCondition_NoWarning(t *testing.T) {
	warns := Analyze(Inputs{
		Name:       "OneCond",
		Conditions: makeConditions([]string{"true"}, [][]string{{"Y"}}),
		Actions:    makeActions([]string{`perform Other`}, [][]string{{"X"}}),
		MaxCol:     1,
	})
	for _, w := range warns {
		if w.Kind == KindColumnActionsWithoutConditions {
			t.Errorf("warning fired on a table with a condition: %v", w)
		}
	}
}
