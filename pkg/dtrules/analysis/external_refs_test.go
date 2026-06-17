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

package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a tiny helper for building fixture projects in a temp dir.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// stubOperators is a fixed operator set standing in for the real registry so
// this leaf package's tests don't depend on the operators package.
func stubOperators(names ...string) OperatorChecker {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return func(tok string) bool { return set[tok] }
}

const cleanEDD = `<entity_data_dictionary version='2'>
  <entity name='client' access='rw'>
    <field name='age' type='integer' access='rw' />
    <field name='score' type='integer' access='rw' />
  </entity>
  <entity name='result' access='rw'>
    <field name='eligible' type='boolean' access='rw' />
  </entity>
</entity_data_dictionary>`

// TestAnalyzeExternalRefs_Clean verifies a self-contained project — every
// perform target defined, every field declared, every operator registered —
// produces no findings.
func TestAnalyzeExternalRefs_Clean(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main_edd.xml", cleanEDD)
	writeFile(t, dir, "main_dt.xml", `<decision_tables>
<decision_table>
<table_name>Entry</table_name>
<conditions><condition_details><condition_dsl>client.age &gt; 18</condition_dsl>
<condition_postfix>client.age 18 f&gt;</condition_postfix></condition_details></conditions>
<actions><action_details><action_dsl>perform Score_It</action_dsl>
<action_postfix>/Score_It performtable</action_postfix></action_details></actions>
</decision_table>
<decision_table>
<table_name>Score_It</table_name>
<actions><action_details><action_dsl>set result.eligible = true</action_dsl>
<action_postfix>true /result.eligible xdef</action_postfix></action_details></actions>
</decision_table>
</decision_tables>`)

	findings, err := AnalyzeExternalRefs(dir, stubOperators("f>", "performtable", "xdef", "true"))
	if err != nil {
		t.Fatalf("AnalyzeExternalRefs: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings on a clean project, got %d:\n%v", len(findings), findings)
	}
}

// TestAnalyzeExternalRefs_AllKinds verifies each undefined-symbol class is
// flagged: an undefined perform target, a field absent from a declared
// entity, and a postfix operator absent from the registry.
func TestAnalyzeExternalRefs_AllKinds(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main_edd.xml", cleanEDD)
	writeFile(t, dir, "main_dt.xml", `<decision_tables>
<decision_table>
<table_name>Bad</table_name>
<conditions><condition_details><condition_dsl>client.bogus &gt; 5</condition_dsl>
<condition_postfix>client.age 5 madeupop</condition_postfix></condition_details></conditions>
<actions><action_details><action_dsl>perform Nonexistent_Table</action_dsl>
<action_postfix>/Nonexistent_Table performtable</action_postfix></action_details></actions>
</decision_table>
</decision_tables>`)

	findings, err := AnalyzeExternalRefs(dir, stubOperators("f>", "performtable"))
	if err != nil {
		t.Fatalf("AnalyzeExternalRefs: %v", err)
	}

	got := map[ExternalRefKind]string{}
	for _, f := range findings {
		got[f.Kind] = f.Symbol
	}
	if got[ExternalRefTable] != "Nonexistent_Table" {
		t.Errorf("undefined_table: want Nonexistent_Table, got %q (all: %v)", got[ExternalRefTable], findings)
	}
	if got[ExternalRefField] != "client.bogus" {
		t.Errorf("undefined_field: want client.bogus, got %q (all: %v)", got[ExternalRefField], findings)
	}
	if got[ExternalRefOperator] != "madeupop" {
		t.Errorf("undefined_operator: want madeupop, got %q (all: %v)", got[ExternalRefOperator], findings)
	}
}

// TestAnalyzeExternalRefs_NoFalsePositives pins the resolution rules that the
// initial implementation got wrong: a declared entity name appearing as a
// bare postfix token is not an undefined operator; an EL keyword is not an
// operator; a `%` comment line is not operator tokens; and a postfix-
// allocated local (`/name allocate` … bare `name`) is defined.
func TestAnalyzeExternalRefs_NoFalsePositives(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main_edd.xml", cleanEDD)
	writeFile(t, dir, "main_dt.xml", `<decision_tables>
<decision_table>
<table_name>Tricky</table_name>
<actions><action_details>
<action_dsl>set result.eligible = true</action_dsl>
<action_postfix>% client age threshold check for 2025: see policy
client.score /threshold allocate threshold 10 f+ otherwise true /result.eligible xdef</action_postfix>
</action_details></actions>
</decision_table>
</decision_tables>`)

	findings, err := AnalyzeExternalRefs(dir, stubOperators("f+", "allocate", "xdef", "true"))
	if err != nil {
		t.Fatalf("AnalyzeExternalRefs: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings; comment text, entity names, keywords, and "+
			"allocated locals must all resolve. Got %d:\n%v", len(findings), findings)
	}
}

// TestAnalyzeExternalRefs_NilOperatorChecker verifies the field and perform
// checks still run when no operator checker is supplied; only the operator
// check is skipped.
func TestAnalyzeExternalRefs_NilOperatorChecker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main_edd.xml", cleanEDD)
	writeFile(t, dir, "main_dt.xml", `<decision_tables>
<decision_table>
<table_name>Bad</table_name>
<conditions><condition_details><condition_dsl>client.bogus &gt; 5</condition_dsl>
<condition_postfix>client.age 5 madeupop</condition_postfix></condition_details></conditions>
</decision_table>
</decision_tables>`)

	findings, err := AnalyzeExternalRefs(dir, nil)
	if err != nil {
		t.Fatalf("AnalyzeExternalRefs: %v", err)
	}
	for _, f := range findings {
		if f.Kind == ExternalRefOperator {
			t.Errorf("operator check should be skipped with a nil checker, got %v", f)
		}
	}
	var sawField bool
	for _, f := range findings {
		if f.Kind == ExternalRefField && f.Symbol == "client.bogus" {
			sawField = true
		}
	}
	if !sawField {
		t.Errorf("field check should still run with a nil checker; findings: %v", findings)
	}
}
