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

package excel

import (
	"strings"
	"testing"
)

// tableWithDSL is a minimal table carrying one DSL row in each section.
func tableWithDSL() *DecisionTableXML {
	t := &DecisionTableXML{TableName: "Decide"}
	t.Contexts.Details = []ContextDetailXML{{Number: 1, DSL: "for all job.clients"}}
	t.InitialActions = []InitialActionXML{{DSL: "set result.total = 0"}}
	t.Conditions = []ConditionXML{{DSL: "client.age >= 18"}}
	t.Actions = []ActionXML{{DSL: "set result.total = 1"}}
	return t
}

// TestNoCompilerWithDSLIsRecorded pins the guard added for #929.
//
// An importer with no EL compiler writes every postfix element empty. That is
// correct when there is no DSL and catastrophic when there is: `dtrules sync
// import` built its importer without a compiler and produced a 490-line
// degradation of KidAid's decision table — every postfix emptied, the rule set
// reduced to one that loads and decides nothing — while reporting success.
//
// Skipping compilation must therefore be distinguishable from having nothing
// to compile. It is recorded as a drop, which build and sync already surface.
func TestNoCompilerWithDSLIsRecorded(t *testing.T) {
	imp := NewDTImporter()
	stats := &ImportStats{}
	imp.SetStats(stats)

	if err := imp.compileTableEL(tableWithDSL()); err != nil {
		t.Fatalf("compileTableEL: %v", err)
	}

	if len(stats.Drops) != 1 {
		t.Fatalf("want 1 drop for a table with DSL and no compiler, got %d: %+v",
			len(stats.Drops), stats.Drops)
	}
	drop := stats.Drops[0]
	if drop.Table != "Decide" {
		t.Errorf("drop names table %q, want %q", drop.Table, "Decide")
	}
	// The count matters: it is what tells a reader this was not a no-op.
	if !strings.Contains(drop.Reason, "4 DSL row(s)") {
		t.Errorf("drop should say how many rows were affected, got: %s", drop.Reason)
	}
	if !strings.Contains(drop.Reason, "no EL compiler wired") {
		t.Errorf("drop should name the cause, got: %s", drop.Reason)
	}
}

// TestNoCompilerWithoutDSLIsSilent keeps the guard from crying wolf. Plenty of
// callers construct an importer purely to inspect a workbook's shape; a table
// with no DSL has nothing to compile and must not be reported as a drop.
func TestNoCompilerWithoutDSLIsSilent(t *testing.T) {
	imp := NewDTImporter()
	stats := &ImportStats{}
	imp.SetStats(stats)

	bare := &DecisionTableXML{TableName: "Empty"}
	bare.Conditions = []ConditionXML{{DSL: "   "}} // whitespace is not DSL
	if err := imp.compileTableEL(bare); err != nil {
		t.Fatalf("compileTableEL: %v", err)
	}
	if len(stats.Drops) != 0 {
		t.Errorf("a table with no DSL must not be reported: %+v", stats.Drops)
	}
}

// TestCountDSLRowsSeesBothInitialActionSpellings guards the interaction with
// the `<initial_action_details>` fix: a table whose initial actions use the
// legacy spelling must still be counted, or a project spelling them that way
// would silently fall back into the case this guard exists to catch.
func TestCountDSLRowsSeesBothInitialActionSpellings(t *testing.T) {
	legacy := &DecisionTableXML{TableName: "Legacy"}
	legacy.InitialActionsLegacy = []InitialActionXML{{DSL: "set result.total = 0"}}
	if n := countDSLRows(legacy); n != 1 {
		t.Errorf("countDSLRows sees %d rows in the legacy spelling, want 1", n)
	}
}
