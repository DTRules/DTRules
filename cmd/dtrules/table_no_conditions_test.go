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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
)

// TestTablePutWarnsColumnActionsWithoutConditions: `table put` of a table
// with no conditions and an action marked in column 1 used to answer with an
// empty warnings list, although that action can never run (#1230). The put
// still succeeds -- warnings never gate -- but the response now says so.
func TestTablePutWarnsColumnActionsWithoutConditions(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	tbl := TableJSON{
		Name:   "NoCond_Probe",
		File:   "probe_dt.xml",
		Number: 9500,
		Policy: "ALL",
		Actions: []ActionJSON{
			{Number: 1, DSL: "set client.eligible = true", Columns: map[string]bool{"1": true}},
		},
	}
	payload, err := json.Marshal(tbl)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, se, code := runTableCmd(t, dir, []string{"put", "NoCond_Probe",
		"--file", "probe_dt.xml", "--range", "9500-9599", "--reason", "#1230 vector"}, string(payload))
	if code != 0 {
		t.Fatalf("put exit %d stderr=%s", code, se)
	}
	var resp struct {
		Warnings []decisiontable.Warning `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("put response not JSON: %v\n%s", err, out)
	}
	found := false
	for _, w := range resp.Warnings {
		if w.Kind == decisiontable.KindColumnActionsWithoutConditions {
			found = true
		}
	}
	if !found {
		t.Errorf("put response carries no %q warning: %s", decisiontable.KindColumnActionsWithoutConditions, out)
	}

	// The same table with its action in initial_actions is the fix the
	// warning points at, and draws no such warning.
	tbl.Actions = nil
	tbl.InitialActions = []InitialActionJSON{{DSL: "set client.eligible = true"}}
	payload, _ = json.Marshal(tbl)
	out, se, code = runTableCmd(t, dir, []string{"put", "NoCond_Probe"}, string(payload))
	if code != 0 {
		t.Fatalf("second put exit %d stderr=%s", code, se)
	}
	resp.Warnings = nil
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("put response not JSON: %v\n%s", err, out)
	}
	for _, w := range resp.Warnings {
		if w.Kind == decisiontable.KindColumnActionsWithoutConditions {
			t.Errorf("initial_actions-only table still warned: %v", w)
		}
	}
}
