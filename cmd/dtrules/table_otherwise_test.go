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
	"encoding/json"
	"strings"
	"testing"
)

// The otherwise column through the authoring CLI (#1215).
//
// '*' does not mean "don't care" -- that is '-'. It marks the otherwise
// column: the last column, holding no Y or N, which executes iff no other
// column executed. `table put` used to accept it in any cell of any column
// with no check at all, and `table patch` refused it outright, so the one
// column DTRules has for "anything not handled above" could be written
// wrongly by one entry point and not at all by the other. Both now apply the
// engine's rule, and say it when they refuse.

// otherwiseTable is a two-condition table whose third column is the
// otherwise column.
func otherwiseTable(t *testing.T, cells1, cells2 map[string]string) string {
	t.Helper()
	tbl := TableJSON{
		Name:   "Otherwise_Probe",
		File:   "probe_dt.xml",
		Number: 9500,
		Policy: "FIRST",
		Conditions: []ConditionJSON{
			{Number: 1, DSL: "client.age > 5", Columns: cells1},
			{Number: 2, DSL: "client.age > 18", Columns: cells2},
		},
		Actions: []ActionJSON{
			{Number: 1, DSL: "set client.eligible = true", Columns: map[string]bool{"1": true}},
			{Number: 2, DSL: "set client.eligible = true", Columns: map[string]bool{"2": true}},
			{Number: 3, DSL: "set client.eligible = false", Columns: map[string]bool{"3": true}},
		},
	}
	payload, err := json.Marshal(tbl)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(payload)
}

func putOtherwise(t *testing.T, dir, payload string) (string, int) {
	t.Helper()
	_, se, code := runTableCmd(t, dir, []string{"put", "Otherwise_Probe",
		"--file", "probe_dt.xml", "--range", "9500-9599", "--reason", "otherwise vector"}, payload)
	return se, code
}

// TestTablePutAcceptsOtherwiseColumn: a well-formed otherwise column goes in,
// and comes back out as '*'.
func TestTablePutAcceptsOtherwiseColumn(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	payload := otherwiseTable(t,
		map[string]string{"1": "Y", "2": "-", "3": "*"},
		map[string]string{"1": "-", "2": "N", "3": "-"})
	if se, code := putOtherwise(t, dir, payload); code != 0 {
		t.Fatalf("put exit %d stderr=%s", code, se)
	}
	out, _, code := runTableCmd(t, dir, []string{"get", "Otherwise_Probe"}, "")
	if code != 0 {
		t.Fatalf("get exit %d", code)
	}
	var got TableJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("get not valid JSON: %v", err)
	}
	if got.Conditions[0].Columns["3"] != "*" {
		t.Errorf("column 3 came back %q, want \"*\"", got.Conditions[0].Columns["3"])
	}
}

// TestTablePutRejectsIllFormedStar: the two shapes the definition rules out,
// refused before anything is written, with the rule in the message.
func TestTablePutRejectsIllFormedStar(t *testing.T) {
	cases := []struct {
		name           string
		cells1, cells2 map[string]string
	}{
		{"star outside the last column",
			map[string]string{"1": "Y", "2": "*", "3": "-"},
			map[string]string{"1": "-", "2": "-", "3": "N"}},
		{"last column also holds a Y",
			map[string]string{"1": "Y", "2": "-", "3": "*"},
			map[string]string{"1": "-", "2": "N", "3": "Y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyProject(t, "../../sampleprojects/CHIP")
			se, code := putOtherwise(t, dir, otherwiseTable(t, tc.cells1, tc.cells2))
			if code == 0 {
				t.Fatalf("put accepted %s", tc.name)
			}
			if !strings.Contains(se, "otherwise column") || !strings.Contains(se, "last column") {
				t.Errorf("error does not state the rule: %s", se)
			}
			// Nothing written: the table never reached the file.
			if out, _, code := runTableCmd(t, dir, []string{"get", "Otherwise_Probe"}, ""); code == 0 {
				t.Errorf("refused table was written anyway: %s", out)
			}
		})
	}
}

// TestTablePatchSetConditionCellStar: patch is the entry point that adds one
// cell at a time, and it used to answer "value must be one of Y, N, -".
func TestTablePatchSetConditionCellStar(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	payload := otherwiseTable(t,
		map[string]string{"1": "Y", "2": "-", "3": "-"},
		map[string]string{"1": "-", "2": "N", "3": "-"})
	if se, code := putOtherwise(t, dir, payload); code != 0 {
		t.Fatalf("put exit %d stderr=%s", code, se)
	}

	// Column 3 is last and holds no Y or N: '*' belongs there.
	_, se, code := runTableCmd(t, dir, []string{"patch", "Otherwise_Probe"},
		`{"op":"set-condition-cell","condition_number":1,"column":3,"value":"*"}`)
	if code != 0 {
		t.Fatalf("patch rejected a legal otherwise cell: %s", se)
	}

	// Column 2 is not the last column.
	_, se, code = runTableCmd(t, dir, []string{"patch", "Otherwise_Probe"},
		`{"op":"set-condition-cell","condition_number":2,"column":2,"value":"*"}`)
	if code == 0 {
		t.Fatalf("patch accepted '*' outside the last column")
	}
	if !strings.Contains(se, "last column") {
		t.Errorf("error does not state the rule: %s", se)
	}

	// And the rule the other way round: no Y or N in the otherwise column.
	_, se, code = runTableCmd(t, dir, []string{"patch", "Otherwise_Probe"},
		`{"op":"set-condition-cell","condition_number":2,"column":3,"value":"N"}`)
	if code == 0 {
		t.Fatalf("patch put an N in the otherwise column")
	}
	if !strings.Contains(se, "otherwise column") {
		t.Errorf("error does not state the rule: %s", se)
	}

	// The legal '*' is still there, unharmed by the two refusals.
	out, _, code := runTableCmd(t, dir, []string{"get", "Otherwise_Probe"}, "")
	if code != 0 {
		t.Fatalf("get exit %d", code)
	}
	var got TableJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("get not valid JSON: %v", err)
	}
	if got.Conditions[0].Columns["3"] != "*" {
		t.Errorf("otherwise cell lost: %q", got.Conditions[0].Columns["3"])
	}
}

// TestTableSchemaListsStar: the schema is what an AI session reads before it
// writes, and it said the otherwise column was impossible.
func TestTableSchemaListsStar(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	for _, args := range [][]string{{"schema"}, {"schema", "--patch"}} {
		out, _, code := runTableCmd(t, dir, args, "")
		if code != 0 {
			t.Fatalf("%v exit %d", args, code)
		}
		if !strings.Contains(out, `"*"`) {
			t.Errorf("%v does not list \"*\"", args)
		}
	}
}

// #1221: add-column appended after the otherwise column. Empty, the new column
// was accepted as trailing padding -- `patched`, the '*' no longer last, and
// in a BALANCED table the otherwise column switched off. It now takes the
// otherwise column's place, which moves one right, and patch says where.
func TestAddColumnGoesBeforeTheOtherwiseColumn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		patch string
		// want is condition 1's and 2's cells after the patch.
		want1, want2 map[string]string
	}{
		{"empty column",
			`{"op":"add-column"}`,
			map[string]string{"1": "Y", "2": "-", "3": "-", "4": "*"},
			map[string]string{"1": "-", "2": "N", "3": "-", "4": "-"}},
		{"column with content",
			`{"op":"add-column","conditions":{"1":"N","2":"Y"},"actions":[2]}`,
			map[string]string{"1": "Y", "2": "-", "3": "N", "4": "*"},
			map[string]string{"1": "-", "2": "N", "3": "Y", "4": "-"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyProject(t, "../../sampleprojects/CHIP")
			if se, code := putOtherwise(t, dir, otherwiseTable(t,
				map[string]string{"1": "Y", "2": "-", "3": "*"},
				map[string]string{"1": "-", "2": "N", "3": "-"})); code != 0 {
				t.Fatalf("put exit %d stderr=%s", code, se)
			}
			out, se, code := runTableCmd(t, dir, []string{"patch", "Otherwise_Probe"}, tc.patch)
			if code != 0 {
				t.Fatalf("patch exit %d stderr=%s", code, se)
			}
			if !strings.Contains(out, `"column": "3"`) {
				t.Errorf("patch should report the new column as 3:\n%s", out)
			}

			got, _, _ := runTableCmd(t, dir, []string{"get", "Otherwise_Probe"}, "")
			var tbl TableJSON
			if err := json.Unmarshal([]byte(got), &tbl); err != nil {
				t.Fatalf("get: %v", err)
			}
			for i, want := range []map[string]string{tc.want1, tc.want2} {
				for col, v := range want {
					if g := tbl.Conditions[i].Columns[col]; g != v {
						t.Errorf("condition %d column %s = %q, want %q (all: %v)", i+1, col, g, v, tbl.Conditions[i].Columns)
					}
				}
			}
			// The otherwise column's action moved with it.
			if !tbl.Actions[2].Columns["4"] || tbl.Actions[2].Columns["3"] {
				t.Errorf("action 3 (the otherwise action) = %v, want it on column 4 only", tbl.Actions[2].Columns)
			}
		})
	}
}

// Without an otherwise column, add-column still appends.
func TestAddColumnAppendsWithoutAnOtherwiseColumn(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	if se, code := putOtherwise(t, dir, otherwiseTable(t,
		map[string]string{"1": "Y", "2": "-", "3": "N"},
		map[string]string{"1": "-", "2": "N", "3": "-"})); code != 0 {
		t.Fatalf("put exit %d stderr=%s", code, se)
	}
	out, se, code := runTableCmd(t, dir, []string{"patch", "Otherwise_Probe"},
		`{"op":"add-column","conditions":{"1":"N","2":"Y"}}`)
	if code != 0 {
		t.Fatalf("patch exit %d stderr=%s", code, se)
	}
	if !strings.Contains(out, `"column": "4"`) {
		t.Errorf("patch should report the new column as 4:\n%s", out)
	}
}
