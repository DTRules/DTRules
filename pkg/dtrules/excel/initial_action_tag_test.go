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

package excel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// legacyInitialActionsXML spells the section <initial_action_details>, the way
// every other section of a DT file is spelled and the way SyntaxTests spells
// it in all 312 of its initial-action rows.
const legacyInitialActionsXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Legacy_Initial_Actions</table_name>
<xls_file>test.xls</xls_file>
<attribute_fields>
<TYPE>FIRST</TYPE>
<COMMENTS />
<TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions>
<initial_action_details>
<initial_action_comment>seed the total</initial_action_comment>
<initial_action_dsl>set totalIncome = 0</initial_action_dsl>
<initial_action_postfix>0 cvd /totalIncome xdef</initial_action_postfix></initial_action_details>
<initial_action_details>
<initial_action_comment>seed the flag</initial_action_comment>
<initial_action_dsl>set eligible = false</initial_action_dsl>
<initial_action_postfix>false cvb /eligible xdef</initial_action_postfix></initial_action_details>
</initial_actions>
<conditions></conditions>
<actions></actions>
</decision_table>
</decision_tables>
`

// TestLegacyInitialActionDetailsTagSurvivesRoundTrip guards the initial-action
// section against its second element spelling.
//
// Only <initial_action> was ever read. A table spelled
// <initial_action_details> therefore loaded with an empty initial-action list
// — the rows were not compiled, not executed, and not reachable from the
// authoring API — and the next save wrote `<initial_actions></initial_actions>`
// over them, deleting the lot. SyntaxTests carried 312 rows in that state and
// a single `table put` erased every one of them.
//
// Read must see them; write must keep them and normalise the spelling.
func TestLegacyInitialActionDetailsTagSurvivesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "legacy_dt.xml")
	if err := os.WriteFile(src, []byte(legacyInitialActionsXML), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	tables, err := UnmarshalDecisionTablesXML(raw)
	if err != nil {
		t.Fatalf("UnmarshalDecisionTablesXML: %v", err)
	}
	if len(tables.Tables) != 1 {
		t.Fatalf("loaded %d tables, want 1", len(tables.Tables))
	}
	table := tables.Tables[0]

	got := table.EffectiveInitialActions()
	if len(got) != 2 {
		t.Fatalf("EffectiveInitialActions() returned %d rows, want 2 — the legacy spelling was not read", len(got))
	}
	if got[0].DSL != "set totalIncome = 0" {
		t.Errorf("row 1 DSL = %q, want %q", got[0].DSL, "set totalIncome = 0")
	}
	if c := got[0].EffectiveComment(); c != "seed the total" {
		t.Errorf("row 1 comment = %q, want %q — <initial_action_comment> was dropped", c, "seed the total")
	}

	imp := NewDTImporter()
	out := filepath.Join(dir, "out_dt.xml")
	if err := imp.WriteXML(tables, out); err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(written)

	if strings.Contains(text, "<initial_actions></initial_actions>") {
		t.Fatal("write emitted an empty <initial_actions> — the rows were deleted on save")
	}
	if n := strings.Count(text, "<initial_action>"); n != 2 {
		t.Errorf("write emitted %d <initial_action> elements, want 2 (canonical spelling)", n)
	}
	if strings.Contains(text, "<initial_action_details>") {
		t.Error("write kept the legacy <initial_action_details> spelling; it should normalise to <initial_action>")
	}
	for _, want := range []string{"set totalIncome = 0", "set eligible = false", "seed the total"} {
		if !strings.Contains(text, want) {
			t.Errorf("write lost %q", want)
		}
	}

	// Reloading the normalised file must give back the same rows, so the
	// round trip is stable rather than merely non-destructive once.
	round, err := UnmarshalDecisionTablesXML(written)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if n := len(round.Tables[0].EffectiveInitialActions()); n != 2 {
		t.Errorf("reload returned %d rows, want 2", n)
	}
}

// TestEffectiveInitialActionsPrefersCanonical pins the precedence so a table
// that somehow carries both spellings cannot execute the stale list.
func TestEffectiveInitialActionsPrefersCanonical(t *testing.T) {
	table := DecisionTableXML{
		InitialActions:       []InitialActionXML{{DSL: "canonical"}},
		InitialActionsLegacy: []InitialActionXML{{DSL: "legacy"}, {DSL: "legacy2"}},
	}
	got := table.EffectiveInitialActions()
	if len(got) != 1 || got[0].DSL != "canonical" {
		t.Errorf("EffectiveInitialActions() = %+v, want the canonical list to win", got)
	}

	empty := DecisionTableXML{}
	if n := len(empty.EffectiveInitialActions()); n != 0 {
		t.Errorf("a table with no initial actions reports %d rows, want 0", n)
	}
}
