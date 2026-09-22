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

// TestTablePutRejectsSortByFieldRead: `table put` compiles every cell against
// the project's EDD, and `sort … by client.gender` (a string field read, not
// a name) used to go in clean and fail only when the row ran (#1227). It is
// now a compile error carrying the form that works; that form goes in.
func TestTablePutRejectsSortByFieldRead(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	put := func(action string) (string, int) {
		tbl := TableJSON{
			Name:   "Sort_Probe",
			File:   "probe_dt.xml",
			Number: 9500,
			Policy: "FIRST",
			Conditions: []ConditionJSON{
				{Number: 1, DSL: "true", Columns: map[string]string{"1": "Y"}},
			},
			Actions: []ActionJSON{
				{Number: 1, DSL: action, Columns: map[string]bool{"1": true}},
			},
		}
		payload, err := json.Marshal(tbl)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		_, se, code := runTableCmd(t, dir, []string{"put", "Sort_Probe",
			"--file", "probe_dt.xml", "--range", "9500-9599", "--reason", "#1227 vector"}, string(payload))
		return se, code
	}

	se, code := put("sort case.clients in ascending order by client.gender")
	if code == 0 {
		t.Fatalf("put accepted a field read after `by`")
	}
	if !strings.Contains(se, `the name \"gender\"`) && !strings.Contains(se, `the name "gender"`) {
		t.Errorf("error does not point at `the name \"gender\"`: %s", se)
	}

	if se, code := put(`sort case.clients in ascending order by the name "gender"`); code != 0 {
		t.Fatalf("put refused the name form: %s", se)
	}
}
