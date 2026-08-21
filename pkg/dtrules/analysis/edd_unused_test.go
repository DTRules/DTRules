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
	"strings"
	"testing"
)

const fixtureEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="used_field" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
    <field name="unused_field" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
    <field name="write_only_field" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
  </entity>
</entity_data_dictionary>
`

// fixtureDT references job.used_field (read) and job.write_only_field (write only).
const fixtureDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>TestTable</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>test</condition_comment>
    <condition_dsl>job.used_field > 0</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
    <condition_column column_number="2" column_value="N"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>set write-only</action_comment>
    <action_dsl>set job.write_only_field to 42</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"></action_column>
  </action_details>
  <action_details>
    <action_number>2</action_number>
    <action_comment>fallthrough</action_comment>
    <action_dsl>set job.used_field to 0</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="2" column_value="X"></action_column>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

func TestAnalyzeEDDUsage_UnusedAndWriteOnly(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "test_edd.xml"), []byte(fixtureEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test_dt.xml"), []byte(fixtureDT), 0644); err != nil {
		t.Fatal(err)
	}

	warns, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage error: %v", err)
	}

	var unusedFound, writeOnlyFound bool
	for _, w := range warns {
		if strings.Contains(w.Reason, "unused EDD field") && strings.Contains(w.Reason, "unused_field") {
			unusedFound = true
		}
		if strings.Contains(w.Reason, "write-only EDD field") && strings.Contains(w.Reason, "write_only_field") {
			writeOnlyFound = true
		}
	}

	if !unusedFound {
		t.Errorf("expected unused EDD field warning for job.unused_field, got %v", warns)
	}
	if !writeOnlyFound {
		t.Errorf("expected write-only EDD field warning for job.write_only_field, got %v", warns)
	}

	// job.used_field should NOT be flagged (it's both read and written).
	for _, w := range warns {
		if strings.Contains(w.Reason, "job.used_field") && !strings.Contains(w.Reason, "unused") {
			t.Errorf("unexpected warning for job.used_field: %v", w)
		}
		// The unused_field warning is expected; ensure used_field without "unused" prefix is not there.
		if strings.Contains(w.Field, "job.used_field") {
			t.Errorf("unexpected warning with field job.used_field: %v", w)
		}
	}
}

// TestAnalyzeEDDUsage_OutputField covers access="w" (output) semantics: a
// field written by the rules and consumed externally is NOT a write-only
// finding, but declaring an output that no rule produces is.
func TestAnalyzeEDDUsage_OutputField(t *testing.T) {
	const outEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="result" access="rw">
    <field name="produced_output" type="double" subtype="" access="w" input="" default_value="0" comment=""></field>
    <field name="missing_output"  type="double" subtype="" access="w" input="" default_value="0" comment=""></field>
  </entity>
</entity_data_dictionary>`
	const outDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Out</table_name>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>set result.produced_output = 42</action_dsl>
    <action_column column_number="1" column_value="X"></action_column>
  </action_details>
</actions>
</decision_table>
</decision_tables>`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "out_edd.xml"), []byte(outEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "out_dt.xml"), []byte(outDT), 0644); err != nil {
		t.Fatal(err)
	}

	warns, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage error: %v", err)
	}

	for _, w := range warns {
		if strings.Contains(w.Field, "produced_output") {
			t.Errorf("an output that IS written must not warn, got %v", w)
		}
	}
	var missingWarned bool
	for _, w := range warns {
		if strings.Contains(w.Field, "missing_output") && strings.Contains(w.Reason, "never written") {
			missingWarned = true
		}
	}
	if !missingWarned {
		t.Errorf("expected a 'declared output never written' warning for result.missing_output, got %v", warns)
	}
}

func TestAnalyzeEDDUsage_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	warns, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("expected no warnings for empty dir, got %v", warns)
	}
}
