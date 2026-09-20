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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The API server is an outside write path: whatever a caller PUTs in the
// request body has to satisfy the field's declared constraints (#1209). The
// CLI vectors cannot reach it, so it is covered here.

const constraintEDDXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="Acute Sinusitis" max_length="40">
      <allowed_value value="Acute Sinusitis"></allowed_value>
      <allowed_value value="Chronic Sinusitis"></allowed_value>
    </field>
    <field name="notes" type="string" access="rw" default_value=""></field>
  </entity>
  <entity name="result" access="rw">
    <field name="seen" type="string" access="rw" default_value=""></field>
  </entity>
</entity_data_dictionary>`

const constraintDTXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>Echo_Diagnosis</table_name>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS></COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
<action_details>
<action_number>1</action_number>
<action_comment>Copy the diagnosis through so the response shows what loaded</action_comment>
<action_dsl>set result.seen = patient.diagnosis</action_dsl>
<action_postfix>
patient.diagnosis cvs /result.seen xdef
</action_postfix>
<columns>*</columns>
</action_details>
</actions>
</decision_table>
</decision_tables>`

// constraintServer writes a two-entity project with one constrained field and
// returns a router over it.
func constraintServer(t *testing.T) http.Handler {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(xmlDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("constraint_edd.xml", constraintEDDXML)
	write("constraint_dt.xml", constraintDTXML)

	s := New(Config{ProjectRoot: root, ReadOnly: true})
	if err := s.LoadProject(xmlDir); err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	return s.Routes()
}

func executeWith(t *testing.T, h http.Handler, diagnosis string) (int, map[string]any) {
	t.Helper()
	body := fmt.Sprintf(`{"tableName":"Echo_Diagnosis","data":{"patient":{"diagnosis":%q}}}`, diagnosis)
	return do(t, h, "POST", "/api/execute", body)
}

// TestAPIExecute_RefusesValueOutsideVocabulary: a value the EDD does not admit
// is a bad request, named as such — not a result computed from a default.
func TestAPIExecute_RefusesValueOutsideVocabulary(t *testing.T) {
	h := constraintServer(t)
	code, body := executeWith(t, h, "Banana")
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 for a value outside the vocabulary, got %d (%v)", code, body)
	}
	msg, _ := body["error"].(string)
	for _, want := range []string{"patient.diagnosis", "Banana", "Chronic Sinusitis"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not name %q", msg, want)
		}
	}
	if _, hasResult := body["result"]; hasResult {
		t.Errorf("a refused load must not return a result: %v", body)
	}
}

// TestAPIExecute_RefusesOverlongValue: the size limits are enforced on the
// same path, and the error names the limit and the actual length.
func TestAPIExecute_RefusesOverlongValue(t *testing.T) {
	h := constraintServer(t)
	code, body := executeWith(t, h, strings.Repeat("x", 41))
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 for a 41-character value, got %d (%v)", code, body)
	}
	msg, _ := body["error"].(string)
	for _, want := range []string{"patient.diagnosis", "41 characters", "max_length is 40"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not name %q", msg, want)
		}
	}
}

// TestAPIExecute_AcceptsValueInVocabulary: a legal value runs, and the match
// is case-insensitive.
func TestAPIExecute_AcceptsValueInVocabulary(t *testing.T) {
	h := constraintServer(t)
	code, body := executeWith(t, h, "chronic sinusitis")
	if code != http.StatusOK {
		t.Fatalf("want 200 for a value in the vocabulary, got %d (%v)", code, body)
	}
	if fmt.Sprint(body["success"]) != "true" {
		t.Errorf("execution did not succeed: %v", body)
	}
}
