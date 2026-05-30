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

// Issue #776 piece A continued: cross-table entity-stack propagation.
// These tests verify the false-positive class this fixes — a field
// referenced only inside a perform-called helper table looks
// "unused" to the per-file pass but is actually live via the
// caller's entity stack.

const crossTableEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="client" access="rw">
    <field name="cases" type="array" subtype="case" owns="true" access="rw" input="" default_value="" comment=""/>
    <field name="flag" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
  </entity>
  <entity name="case" access="rw">
    <field name="status_code" type="string" subtype="" access="rw" input="" default_value="" comment=""/>
    <field name="reviewed" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
  </entity>
</entity_data_dictionary>
`

// Caller pushes `case` via `for all client.cases`, then perform-calls
// Compute_Case_Status. Compute_Case_Status' body references bare
// `status_code` and `reviewed` — which should resolve to
// case.status_code and case.reviewed via the propagated stack.
const crossTableCallerDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Caller_Loop</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts>
  <context_details>
    <context_dsl>for all client.cases</context_dsl>
    <context_postfix>client.cases forall</context_postfix>
  </context_details>
</contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>perform Compute_Case_Status;</action_dsl>
    <action_postfix>/Compute_Case_Status</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

const crossTableCalleeDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Compute_Case_Status</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>reviewed</condition_dsl>
    <condition_postfix>reviewed</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>set status_code = "OK";</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

// TestCrossTablePropagation_CalleeFieldsRecognized: the bare
// `reviewed` and `status_code` references inside Compute_Case_Status
// must be recognized as case.reviewed / case.status_code via the
// propagated stack — so neither shows up as "unused" in the warnings.
func TestCrossTablePropagation_CalleeFieldsRecognized(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x_edd.xml"), []byte(crossTableEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "caller_dt.xml"), []byte(crossTableCallerDT), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "callee_dt.xml"), []byte(crossTableCalleeDT), 0644); err != nil {
		t.Fatal(err)
	}

	warnings, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage: %v", err)
	}

	// `case.reviewed` is read but never written → must have NO
	// warning at all. Pre-propagation it'd be flagged unused.
	// `case.status_code` is written but not read → should appear as
	// write-only, NOT as unused. The propagation distinguishes the
	// two correctly only if the write side resolves as well.
	for _, w := range warnings {
		switch w.Field {
		case "case.reviewed":
			t.Errorf("cross-table propagation missed read of %s: %v", w.Field, w)
		case "case.status_code":
			if w.Category != EDDUsageWriteOnly {
				t.Errorf("expected write-only for %s, got category=%q reason=%q",
					w.Field, w.Category, w.Reason)
			}
		}
	}
}

// TestCrossTablePropagation_DeepChainConverges: a 3-level chain
// (A → B → C) must propagate A's stack all the way to C. Without
// fixed-point iteration C would only see B's empty stack and miss
// A's `case` push.
func TestCrossTablePropagation_DeepChainConverges(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x_edd.xml"), []byte(crossTableEDD), 0644); err != nil {
		t.Fatal(err)
	}

	// A: pushes case, calls B.
	if err := os.WriteFile(filepath.Join(dir, "a_dt.xml"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>TableA</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts>
  <context_details>
    <context_dsl>for all client.cases</context_dsl>
    <context_postfix>client.cases forall</context_postfix>
  </context_details>
</contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>perform TableB;</action_dsl>
    <action_postfix>/TableB</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`), 0644); err != nil {
		t.Fatal(err)
	}
	// B: no own context, calls C.
	if err := os.WriteFile(filepath.Join(dir, "b_dt.xml"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>TableB</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>perform TableC;</action_dsl>
    <action_postfix>/TableC</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`), 0644); err != nil {
		t.Fatal(err)
	}
	// C: references bare `reviewed`. Resolves to case.reviewed via
	// the transitive A→B→C propagation.
	if err := os.WriteFile(filepath.Join(dir, "c_dt.xml"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>TableC</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>reviewed</condition_dsl>
    <condition_postfix>reviewed</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions></actions>
</decision_table>
</decision_tables>
`), 0644); err != nil {
		t.Fatal(err)
	}

	warnings, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage: %v", err)
	}

	for _, w := range warnings {
		if w.Field == "case.reviewed" {
			t.Errorf("transitive propagation missed case.reviewed at depth 3: %v", w)
		}
	}
}

// TestCrossTablePropagation_CycleTerminates: A → B → A is a cycle.
// Fixed-point iteration must terminate without infinite looping.
// Both tables should see each other's stacks merged.
func TestCrossTablePropagation_CycleTerminates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x_edd.xml"), []byte(crossTableEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a_dt.xml"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>CycleA</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts>
  <context_details>
    <context_dsl>for all client.cases</context_dsl>
    <context_postfix>client.cases forall</context_postfix>
  </context_details>
</contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>perform CycleB;</action_dsl>
    <action_postfix>/CycleB</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b_dt.xml"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>CycleB</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>reviewed</condition_dsl>
    <condition_postfix>reviewed</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>perform CycleA;</action_dsl>
    <action_postfix>/CycleA</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`), 0644); err != nil {
		t.Fatal(err)
	}

	// If this doesn't terminate, the test will time out (Go test
	// runner uses a default per-test timeout).
	warnings, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage: %v", err)
	}
	for _, w := range warnings {
		if w.Field == "case.reviewed" {
			t.Errorf("cycle terminated but didn't propagate: %v", w)
		}
	}
}
