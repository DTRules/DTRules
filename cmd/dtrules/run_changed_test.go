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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The #1233 project: one singleton with a counter and a list, and one entry
// table per kind of run. Postfix is what `dtrules compile` emits for the DSL.
const changedEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
<file_path>chg_edd</file_path>
<entity name="state" number="100" access="rw">
<field name="count" type="integer" subtype="" access="rw" input="" default_value="0" comment="a counter"></field>
<field name="items" type="array" subtype="string" access="rw" input="" default_value="" comment="a list"></field>
<field name="ratio" type="double" subtype="" access="rw" input="" default_value="0.0" comment="a double"></field>
<field name="tag" type="name" subtype="" access="rw" input="" default_value="" comment="a name"></field>
</entity>
</entity_data_dictionary>
`

const changedMap = `<?xml version="1.0" encoding="UTF-8"?>
<mapping><XMLtoEDD><map>
<setattribute tag='count' RAttribute='count' enclosure='state' type='integer'></setattribute>
<createentity entity='state' tag='state' id='id'></createentity>
</map>
<entities><entity name='state' number='1'></entity></entities>
<initialization><initialentity entity='state' epush='true'></initialentity></initialization>
</XMLtoEDD></mapping>
`

// changedData is the state every run starts from.
const changedData = `<?xml version='1.0' encoding='UTF-8'?>
<dtrules-data>
  <state>
    <count>3</count>
    <ratio>3</ratio>
    <tag>abc</tag>
    <items>
      <item>a</item>
      <item>b</item>
    </items>
  </state>
</dtrules-data>
`

var changedTables = []struct {
	name, dsl, postfix string
	changed            bool
}{
	{"Nothing", "", "", false},
	{"Same_Count", "set state.count = 3", "3 cvi /state.count xdef", false},
	{"Self_Count", "set state.count = state.count", "state.count cvi /state.count xdef", false},
	{"Same_List", `set state.items = ["a", "b"]`, `newarray dup "a" addto dup "b" addto /state.items xdef`, false},
	{"Remove_Absent", `remove "zz" from state.items array`, `state.items "zz" remove`, false},
	{"Change_Count", "set state.count = 7", "7 cvi /state.count xdef", true},
	{"Reorder_List", `set state.items = ["b", "a"]`, `newarray dup "b" addto dup "a" addto /state.items xdef`, true},
	{"Add_Item", `add "c" to state.items`, `"c" state.items swap addto`, true},
	{"Remove_Item", `remove "a" from state.items array`, `state.items "a" remove`, true},
	{"Remove_At", "remove 0 element from state.items array", "state.items 0 removeat", true},
	{"Clear_List", "clear state.items", "state.items cleararray", true},
	// No EL form leaves the entity stack unbalanced; hand postfix can, and
	// the stack decides which instance --save writes.
	{"Push_New_State", "push a new state", "/state newentity entitypush", true},
	{"Pop_State", "pop the state", "entitypop pop", true},
	{"Push_Pop_Balanced", "push and pop the state", "state entitypush entitypop pop", false},
	// Equals is looser than the saved text: doubles within 1e-9 and names
	// differing only in case are Equal, but save differently.
	{"Same_Ratio", "set state.ratio = 3.0", "3.0 /state.ratio xdef", false},
	{"Nudge_Ratio", "set state.ratio = 3.00000000001", "3.00000000001 /state.ratio xdef", true},
	{"Same_Tag", "set state.tag to abc", "/abc /state.tag xdef", false},
	// Names intern case-insensitively, first spelling wins, so /ABC is the
	// same name as abc and saves as abc: no change, and the oracle agrees.
	{"Recase_Tag", "set state.tag to ABC", "/ABC /state.tag xdef", false},
}

func writeChangedProject(t *testing.T) (project, data string) {
	t.Helper()
	project = t.TempDir()
	xml := filepath.Join(project, "xml")
	if err := os.MkdirAll(xml, 0o755); err != nil {
		t.Fatal(err)
	}
	var dt strings.Builder
	dt.WriteString("<decision_tables>\n")
	for _, tb := range changedTables {
		fmt.Fprintf(&dt, "<decision_table><table_name>%s</table_name>\n", tb.name)
		if tb.postfix != "" {
			dt.WriteString("<initial_actions><initial_action><action_comment>act</action_comment>\n")
			if tb.dsl != "" {
				fmt.Fprintf(&dt, "<initial_action_dsl>%s</initial_action_dsl>\n", xmlEscape(tb.dsl))
			}
			fmt.Fprintf(&dt, "<initial_action_postfix>%s</initial_action_postfix>\n", xmlEscape(tb.postfix))
			dt.WriteString("</initial_action></initial_actions>\n")
		}
		dt.WriteString("<conditions></conditions><actions></actions></decision_table>\n")
	}
	dt.WriteString("</decision_tables>\n")
	for name, body := range map[string]string{
		"chg_edd.xml": changedEDD,
		"chg_map.xml": changedMap,
		"chg_dt.xml":  dt.String(),
	} {
		if err := os.WriteFile(filepath.Join(xml, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data = filepath.Join(project, "in.xml")
	if err := os.WriteFile(data, []byte(changedData), 0o644); err != nil {
		t.Fatal(err)
	}
	return project, data
}

// strip removes the changed attribute, leaving the saved data.
func strip(saved []byte) string {
	return changedAttr.ReplaceAllString(string(saved), "<dtrules-data>")
}

func saveStripped(t *testing.T, project, entry, data string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.xml")
	if code, _ := runCapture(t, project, "--entry", entry, "--data", data, "--save", out); code != 0 {
		t.Fatalf("%s: exit %d", entry, code)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return strip(b)
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

var changedAttr = regexp.MustCompile(`<dtrules-data changed="(true|false)">`)

// TestRunSave_ReportsWhetherTheRunChangedTheState is #1233: `dtrules run
// --save` records on the root element whether the run changed any entity
// data. Writing a value a field already holds is not a change; changing one
// field, or adding to, removing from, clearing or reordering a list, is.
// Runs through the production path: project load, mapping, --data, --save.
//
// The byte oracle: save order is deterministic (#1231), so a run whose saved
// data (without the attribute) differs from a no-op run's has changed the
// state, and the flag must say so. A false "unchanged" is the failure that
// matters: a host would skip work it had to do.
func TestRunSave_ReportsWhetherTheRunChangedTheState(t *testing.T) {
	project, data := writeChangedProject(t)
	baseline := saveStripped(t, project, "Nothing", data)
	for _, tb := range changedTables {
		t.Run(tb.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.xml")
			if code, _ := runCapture(t, project, "--entry", tb.name, "--data", data, "--save", out); code != 0 {
				t.Fatalf("exit %d", code)
			}
			saved, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			m := changedAttr.FindSubmatch(saved)
			if m == nil {
				t.Fatalf("saved file has no changed attribute on its root:\n%s", saved)
			}
			got := string(m[1]) == "true"
			if got != tb.changed {
				t.Errorf("changed=%s, want %v\n%s", m[1], tb.changed, saved)
			}
			if differs := strip(saved) != baseline; differs != got {
				t.Errorf("changed=%v but the saved data differs from a no-op run's: %v\n--- no-op ---\n%s\n--- this run ---\n%s", got, differs, baseline, saved)
			}
		})
	}
}

// TestRunSave_ChangedFileStillLoads: the attribute is metadata about the run,
// not data — a saved file must still replay through --data.
func TestRunSave_ChangedFileStillLoads(t *testing.T) {
	project, data := writeChangedProject(t)
	dir := t.TempDir()
	first := filepath.Join(dir, "first.xml")
	if code, _ := runCapture(t, project, "--entry", "Add_Item", "--data", data, "--save", first); code != 0 {
		t.Fatalf("exit %d", code)
	}
	second := filepath.Join(dir, "second.xml")
	if code, _ := runCapture(t, project, "--entry", "Nothing", "--data", first, "--save", second); code != 0 {
		t.Fatalf("exit %d", code)
	}
	b, _ := os.ReadFile(second)
	for _, want := range []string{"<count>3</count>", "<item>a</item>", "<item>b</item>", "<item>c</item>"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("replaying the saved file lost %s:\n%s", want, b)
		}
	}
	if !strings.Contains(string(b), `changed="false"`) {
		t.Errorf("a no-op replay must report changed=\"false\":\n%s", b)
	}
}

// TestRunSave_MappedInputIsNotTheRunsChange: loading --input through a
// mapping writes attributes the same way rules do. Those writes are the
// input, not the run's change, so a no-op run over mapped input reports
// changed="false".
func TestRunSave_MappedInputIsNotTheRunsChange(t *testing.T) {
	project, _ := writeChangedProject(t)
	input := filepath.Join(t.TempDir(), "input.xml")
	if err := os.WriteFile(input, []byte("<state><count>5</count></state>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.xml")
	if code, _ := runCapture(t, project, "--entry", "Nothing", "--input", input, "--save", out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	saved, _ := os.ReadFile(out)
	if !strings.Contains(string(saved), "<count>5</count>") {
		t.Fatalf("the mapped input did not load:\n%s", saved)
	}
	if !strings.Contains(string(saved), `changed="false"`) {
		t.Errorf("loading input is not the run's change:\n%s", saved)
	}
}
