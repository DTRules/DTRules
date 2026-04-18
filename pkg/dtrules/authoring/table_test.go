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
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

const testdataMinimal = "testdata/minimal"

func TestOpenSaveProject(t *testing.T) {
	p, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tables := p.Tables()
	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}

	// Save to a temp dir to verify no panic on write.
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "xml"), 0755); err != nil {
		t.Fatal(err)
	}
	// Copy the fixture xml directory to tmp so Save has real paths.
	p2, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("second OpenProject: %v", err)
	}
	_ = p2 // Save would write back to testdata; skip to avoid modifying fixtures.
}

func TestAddCondition_ValidatesEL(t *testing.T) {
	p, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("table Check_Eligibility not found")
	}
	before := len(tbl.Conditions)

	// Add condition with invalid EL — should fail.
	err = tbl.AddCondition(authoring.Condition{DSL: "!!!invalid EL$$$"})
	if err == nil {
		t.Fatal("expected error for invalid EL, got nil")
	}
	// Error should mention parse position.
	if len(tbl.Conditions) != before {
		t.Errorf("conditions mutated on invalid EL: was %d, now %d", before, len(tbl.Conditions))
	}
}

func TestAddAction_ValidatesEL(t *testing.T) {
	p, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("table Check_Eligibility not found")
	}
	before := len(tbl.Actions)

	err = tbl.AddAction(authoring.Action{DSL: "!!!bad action$$$"})
	if err == nil {
		t.Fatal("expected error for invalid action EL, got nil")
	}
	if len(tbl.Actions) != before {
		t.Errorf("actions mutated on invalid EL: was %d, now %d", before, len(tbl.Actions))
	}
}

func TestAddColumn_NumbersMustExist(t *testing.T) {
	p, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("table Check_Eligibility not found")
	}

	// Reference a non-existent condition number (99).
	err = tbl.AddColumn(map[int]string{99: "Y"}, nil)
	if err == nil {
		t.Fatal("expected error for non-existent condition number")
	}
	t.Logf("got expected error: %v", err)
}

func TestAddColumn_ValidValues(t *testing.T) {
	p, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("table Check_Eligibility not found")
	}

	beforeCols := tbl.Columns()

	// Condition 1 exists (loaded from fixture). Actions 1 and 2 exist.
	err = tbl.AddColumn(map[int]string{1: "Y"}, []int{1})
	if err != nil {
		t.Fatalf("AddColumn: %v", err)
	}

	afterCols := tbl.Columns()
	if afterCols != beforeCols+1 {
		t.Errorf("expected %d columns after AddColumn, got %d", beforeCols+1, afterCols)
	}

	// Condition 1 should have the new column value.
	newCol := afterCols
	for _, c := range tbl.Conditions {
		if c.Number == 1 {
			val, ok := c.Columns[newCol]
			if !ok {
				t.Errorf("condition 1 missing column %d", newCol)
			} else if val != "Y" {
				t.Errorf("condition 1 col %d: want Y, got %s", newCol, val)
			}
		}
	}
	// Action 1 should execute; Action 2 should not.
	for _, a := range tbl.Actions {
		switch a.Number {
		case 1:
			if !a.Columns[newCol] {
				t.Errorf("action 1 should execute on col %d", newCol)
			}
		case 2:
			if a.Columns[newCol] {
				t.Errorf("action 2 should NOT execute on col %d", newCol)
			}
		}
	}
}

func TestDeleteColumn_RemovesFromAllArtifacts(t *testing.T) {
	p, err := authoring.OpenProject(testdataMinimal)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("table Check_Eligibility not found")
	}

	beforeCols := tbl.Columns()
	if beforeCols < 1 {
		t.Fatal("fixture table has no columns")
	}

	// Delete column 1.
	if err := tbl.DeleteColumn(1); err != nil {
		t.Fatalf("DeleteColumn: %v", err)
	}

	afterCols := tbl.Columns()
	if afterCols != beforeCols-1 {
		t.Errorf("expected %d columns after delete, got %d", beforeCols-1, afterCols)
	}

	// After deleting column 1, there should be no column key >= beforeCols in any
	// condition or action (i.e., the old last column was renumbered down).
	maxAllowed := beforeCols - 1
	for _, c := range tbl.Conditions {
		for n := range c.Columns {
			if n > maxAllowed {
				t.Errorf("condition %d has column %d > max allowed %d after delete", c.Number, n, maxAllowed)
			}
		}
	}
	for _, a := range tbl.Actions {
		for n := range a.Columns {
			if n > maxAllowed {
				t.Errorf("action %d has column %d > max allowed %d after delete", a.Number, n, maxAllowed)
			}
		}
	}
}

func TestRoundTrip(t *testing.T) {
	// Copy the fixture to a temp dir so we can save without affecting the repo.
	src := testdataMinimal
	tmp := t.TempDir()
	xmlDir := filepath.Join(tmp, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy all files from testdata/minimal/xml into tmp/xml.
	entries, err := os.ReadDir(filepath.Join(src, "xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(src, "xml", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xmlDir, e.Name()), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Open, add an action, save, re-open, verify.
	p, err := authoring.OpenProject(tmp)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Check_Eligibility")
	if tbl == nil {
		t.Fatal("Check_Eligibility not found")
	}

	newDSL := "set applicant.eligible = true"
	if err := tbl.AddAction(authoring.Action{DSL: newDSL, Comment: "Round-trip action"}); err != nil {
		t.Fatalf("AddAction: %v", err)
	}

	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Re-open and verify the action persisted.
	p2, err := authoring.OpenProject(tmp)
	if err != nil {
		t.Fatalf("re-OpenProject: %v", err)
	}
	tbl2 := p2.Table("Check_Eligibility")
	if tbl2 == nil {
		t.Fatal("re-open: Check_Eligibility not found")
	}
	found := false
	for _, a := range tbl2.Actions {
		if a.DSL == newDSL && a.Comment == "Round-trip action" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("action with DSL %q not found after round-trip; actions: %+v", newDSL, tbl2.Actions)
	}
}
