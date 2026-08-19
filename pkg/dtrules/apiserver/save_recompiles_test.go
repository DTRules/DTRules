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

package apiserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

const saveTestEDD = `<entity_data_dictionary>
  <entity name="client" number="100" access="rw">
    <field name="age" type="integer" access="rw"></field>
    <field name="eligible" type="boolean" access="rw"></field>
  </entity>
</entity_data_dictionary>`

// The stored postfix is deliberately WRONG for the DSL below (18 vs 21): it
// stands in for the postfix left over from a previous edit, which is exactly
// what a save used to preserve.
const saveTestDT = `<decision_tables>
<decision_table>
  <table_name>Decide</table_name>
  <attribute_fields><Type>FIRST</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
  <conditions><condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>client.age &gt;= 21</condition_dsl>
    <condition_postfix>client.age 18 i&gt;=</condition_postfix>
    <condition_column column_number="1" column_value="Y" />
  </condition_details></conditions>
  <actions><action_details>
    <action_number>1</action_number>
    <action_dsl>set client.eligible = true</action_dsl>
    <action_postfix>false client.eligible bset</action_postfix>
    <action_column column_number="1" column_value="X" />
  </action_details></actions>
</decision_table>
</decision_tables>`

// saveTestServer writes a tiny project to a temp dir and returns a Server
// wired to it, plus the DT file's path.
func saveTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(xmlDir, "p_dt.xml")
	if err := os.WriteFile(filepath.Join(xmlDir, "p_edd.xml"), []byte(saveTestEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dtPath, []byte(saveTestDT), 0o644); err != nil {
		t.Fatal(err)
	}
	return &Server{projectPath: dir, modified: map[string]bool{}}, dtPath
}

// tableFromFile reads back the one table a save wrote.
func tableFromFile(t *testing.T, path string) excel.DecisionTableXML {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := excel.UnmarshalDecisionTablesXML(data)
	if err != nil {
		t.Fatalf("saved file does not parse: %v", err)
	}
	if len(doc.Tables) != 1 {
		t.Fatalf("want 1 table in the saved file, got %d", len(doc.Tables))
	}
	return doc.Tables[0]
}

// TestSaveRecompilesDSL pins #928.
//
// The old saveDTFile merged edited DSL into the XML and wrote it without recompiling,
// so a saved table carried the new DSL against the postfix from before the
// edit. That is worse than writing none: the file stays well-formed and loads,
// so the rule goes on executing the old logic while displaying the new. The
// editor's "Run speculation" disagreed with its own Save, because speculation
// has always compiled.
func TestSaveRecompilesDSL(t *testing.T) {
	s, dtPath := saveTestServer(t)
	s.tables = []*DecisionTableData{{
		TableName:  "Decide",
		Source:     "xml/p_dt.xml",
		Type:       "FIRST",
		Conditions: []ConditionData{{Number: 1, Description: "client.age >= 21", Columns: map[string]string{"1": "Y"}}},
		Actions:    []ActionData{{Number: 1, Description: "set client.eligible = true", Columns: map[string]string{"1": "X"}}},
	}}

	s.dtFiles = []string{"xml/p_dt.xml"}
	s.modified = map[string]bool{"xml/p_dt.xml": true}
	if _, err := s.saveViaProject(); err != nil {
		t.Fatalf("saveViaProject: %v", err)
	}

	got := tableFromFile(t, dtPath)
	if len(got.Conditions) == 0 {
		t.Fatal("saved table has no conditions")
	}
	pf := got.Conditions[0].Postfix
	if strings.Contains(pf, "18") {
		t.Errorf("condition postfix still holds the pre-edit constant 18: %q", pf)
	}
	if !strings.Contains(pf, "21") {
		t.Errorf("condition postfix was not recompiled from the edited DSL, got %q", pf)
	}
	if apf := got.Actions[0].Postfix; strings.Contains(apf, "false") {
		t.Errorf("action postfix still holds the pre-edit value: %q", apf)
	}
	if !got.ELCompiled {
		t.Error("saved table should be marked el_compiled")
	}
}

// TestSaveRejectsUncompilableDSL: a save that cannot compile must fail loudly
// rather than write the stale postfix it was going to replace. Refusing to
// save is a problem the author can see and fix; a silent stale write is not.
func TestSaveRejectsUncompilableDSL(t *testing.T) {
	s, dtPath := saveTestServer(t)
	before, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	s.tables = []*DecisionTableData{{
		TableName:  "Decide",
		Source:     "xml/p_dt.xml",
		Type:       "FIRST",
		Conditions: []ConditionData{{Number: 1, Description: "client.age >= >= and and", Columns: map[string]string{"1": "Y"}}},
	}}

	s.dtFiles = []string{"xml/p_dt.xml"}
	s.modified = map[string]bool{"xml/p_dt.xml": true}
	_, err = s.saveViaProject()
	if err == nil {
		t.Fatal("save accepted DSL that does not compile")
	}
	// The editor needs to tell "fix your rule" from "the server broke".
	var ce *compileError
	if !errors.As(err, &ce) {
		t.Errorf("error should be a *compileError so the handler can answer 400, got %T: %v", err, err)
	}
	if ce != nil && ce.Table != "Decide" {
		t.Errorf("compileError names table %q, want Decide", ce.Table)
	}

	after, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("a failed save must not modify the file on disk")
	}
}

// postSave drives the real handler and returns the response.
func postSave(t *testing.T, s *Server) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleProjectSave(rec, httptest.NewRequest("POST", "/api/project/save", nil))
	return rec
}

// decodeBody reads jsonError/jsonResponse's envelope.
func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v (body: %s)", err, rec.Body.String())
	}
	return body
}

// TestSaveHandlerReportsCompileErrorAsBadRequest drives POST /api/project/save
// end to end.
//
// The status code is the part the editor acts on. saveDTFile returning a typed
// error is only half the fix — if the handler flattened it back to 500 the
// editor would still be told the server broke when the author's rule is what
// needs fixing (#928).
func TestSaveHandlerReportsCompileErrorAsBadRequest(t *testing.T) {
	s, dtPath := saveTestServer(t)
	before, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	s.dtFiles = []string{"xml/p_dt.xml"}
	s.modified = map[string]bool{"xml/p_dt.xml": true}
	s.tables = []*DecisionTableData{{
		TableName:  "Decide",
		Source:     "xml/p_dt.xml",
		Type:       "FIRST",
		Conditions: []ConditionData{{Number: 1, Description: "client.age >= >= and and", Columns: map[string]string{"1": "Y"}}},
	}}

	rec := postSave(t, s)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d — a rule that does not compile is the author's input, not a server fault",
			rec.Code, http.StatusBadRequest)
	}
	body := decodeBody(t, rec)
	if body["success"] != false {
		t.Errorf("success = %v, want false", body["success"])
	}
	// The message has to name the table, or the editor cannot point at anything.
	if msg, _ := body["error"].(string); !strings.Contains(msg, "Decide") {
		t.Errorf("error message does not name the table: %q", msg)
	}

	// A rejected save leaves the file alone and the file still marked dirty —
	// otherwise the editor would show the edit as saved and the next save
	// would skip it entirely.
	after, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("a rejected save must not modify the file on disk")
	}
	if !s.modified["xml/p_dt.xml"] {
		t.Error("a rejected save must leave the file marked modified")
	}
}

// TestSaveHandlerSucceedsOnValidDSL is the other half: the same route with DSL
// that compiles returns 200, reports the file, and clears the dirty flag.
// Without it the test above would pass just as well against a handler that
// rejected everything.
func TestSaveHandlerSucceedsOnValidDSL(t *testing.T) {
	s, dtPath := saveTestServer(t)
	s.dtFiles = []string{"xml/p_dt.xml"}
	s.modified = map[string]bool{"xml/p_dt.xml": true}
	s.tables = []*DecisionTableData{{
		TableName:  "Decide",
		Source:     "xml/p_dt.xml",
		Type:       "FIRST",
		Conditions: []ConditionData{{Number: 1, Description: "client.age >= 21", Columns: map[string]string{"1": "Y"}}},
		Actions:    []ActionData{{Number: 1, Description: "set client.eligible = true", Columns: map[string]string{"1": "X"}}},
	}}

	rec := postSave(t, s)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["success"] != true {
		t.Errorf("success = %v, want true", body["success"])
	}
	if s.modified["xml/p_dt.xml"] {
		t.Error("a successful save must clear the modified flag")
	}
	if pf := tableFromFile(t, dtPath).Conditions[0].Postfix; !strings.Contains(pf, "21") {
		t.Errorf("the saved postfix was not recompiled through the handler: %q", pf)
	}
}
