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

package analysis

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeProject lays down an EDD and the given _dt.xml files in a temp dir.
func writeProject(t *testing.T, edd string, dtFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, body := range dtFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const constantsEDD = `<entity_data_dictionary>
  <entity name="constants">
    <field name="reduced_dose" type="integer" access="r"/>
    <field name="standard_dose" type="integer" access="r"/>
    <field name="adult_age" type="integer" access="r"/>
  </entity>
  <entity name="result">
    <field name="dose_mg" type="integer" access="w"/>
  </entity>
</entity_data_dictionary>`

func dt(tableName, contexts, conditionDSL string, actionDSLs ...string) string {
	acts := ""
	for _, a := range actionDSLs {
		acts += `<action_details><action_dsl>` + a + `</action_dsl></action_details>`
	}
	return `<decision_tables><decision_table>` +
		`<table_name>` + tableName + `</table_name>` +
		`<contexts>` + contexts + `</contexts>` +
		`<conditions><condition_details><condition_dsl>` + conditionDSL + `</condition_dsl></condition_details></conditions>` +
		`<actions>` + acts + `</actions>` +
		`</decision_table></decision_tables>`
}

func findHint(hints []ContextSuggestion, entity string) *ContextSuggestion {
	for i := range hints {
		if hints[i].Entity == entity {
			return &hints[i]
		}
	}
	return nil
}

// A leaf table reached from a root, both referencing constants.X many
// times with no context: the advisory should fire and target the root.
func TestSuggestContextPushes_SuggestsAtRoot(t *testing.T) {
	dir := writeProject(t, constantsEDD, map[string]string{
		"root_dt.xml": dt("Root", "", "true", "perform Leaf"),
		"leaf_dt.xml": dt("Leaf", "",
			"constants.adult_age > 0",
			"set result.dose_mg = constants.reduced_dose",
			"set result.dose_mg = constants.standard_dose"),
	})

	hints, err := SuggestContextPushes(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := findHint(hints, "constants")
	if h == nil {
		t.Fatalf("expected a hint for 'constants', got %+v", hints)
	}
	if !slices.Contains(h.PushAt, "Root") {
		t.Errorf("expected push-at to target the root table 'Root', got %v", h.PushAt)
	}
	if h.References < 3 {
		t.Errorf("expected >=3 references, got %d", h.References)
	}
}

// Same references, but the entity is already pushed onto the entry table's
// context: the advisory must NOT fire (it propagates down the call chain).
func TestSuggestContextPushes_SilentWhenAlreadyPushed(t *testing.T) {
	dir := writeProject(t, constantsEDD, map[string]string{
		"root_dt.xml": dt("Root",
			`<context_details><context_dsl>add constants to context of this table</context_dsl></context_details>`,
			"true", "perform Leaf"),
		"leaf_dt.xml": dt("Leaf", "",
			"constants.adult_age > 0",
			"set result.dose_mg = constants.reduced_dose",
			"set result.dose_mg = constants.standard_dose"),
	})

	hints, err := SuggestContextPushes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h := findHint(hints, "constants"); h != nil {
		t.Errorf("expected no 'constants' hint when already pushed at root, got %+v", h)
	}
}

// Entities the map file epush'es at session start are globally on the stack,
// so the advisory must not suggest pushing them (regression: it used to fire
// on the SinusitisTherapy `constants`/`result`/`patient`).
func TestSuggestContextPushes_SkipsEpushedEntities(t *testing.T) {
	dir := writeProject(t, constantsEDD, map[string]string{
		"root_dt.xml": dt("Root", "",
			"constants.adult_age > 0",
			"set result.dose_mg = constants.reduced_dose",
			"set result.dose_mg = constants.standard_dose"),
	})
	// Map file pushes `constants` onto the stack at init.
	mapXML := `<mapping><XMLtoEDD><initialization>` +
		`<initialentity entity='constants' epush='true'></initialentity>` +
		`</initialization></XMLtoEDD></mapping>`
	if err := os.WriteFile(filepath.Join(dir, "p_map.xml"), []byte(mapXML), 0o644); err != nil {
		t.Fatal(err)
	}

	hints, err := SuggestContextPushes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h := findHint(hints, "constants"); h != nil {
		t.Errorf("epush'd 'constants' must not be suggested, got %+v", h)
	}
}

// An entity pushed inline by a `for all` in an action/condition (not the
// context row) is on the stack for that fragment, so its qualified field
// reads must not be counted.
func TestSuggestContextPushes_SkipsInlineForAllPushes(t *testing.T) {
	edd := `<entity_data_dictionary>
  <entity name="root"><field name="meds" type="array" subtype="med"/></entity>
  <entity name="med">
    <field name="dose" type="integer" access="r"/>
    <field name="freq" type="integer" access="r"/>
    <field name="qty" type="integer" access="r"/>
  </entity>
</entity_data_dictionary>`
	// The action iterates meds inline, then reads med.* — med is on the stack.
	dir := writeProject(t, edd, map[string]string{
		"root_dt.xml": dt("Root", "", "true",
			"for all meds { set dose = med.dose + med.freq + med.qty }"),
	})
	hints, err := SuggestContextPushes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h := findHint(hints, "med"); h != nil {
		t.Errorf("inline-pushed 'med' must not be suggested, got %+v", h)
	}
}

// Below the threshold: a single qualified reference should not trip it.
func TestSuggestContextPushes_BelowThreshold(t *testing.T) {
	dir := writeProject(t, constantsEDD, map[string]string{
		"root_dt.xml": dt("Root", "", "constants.adult_age > 0", "set result.dose_mg = 1"),
	})
	hints, err := SuggestContextPushes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h := findHint(hints, "constants"); h != nil {
		t.Errorf("expected no hint for a single reference, got %+v", h)
	}
}
