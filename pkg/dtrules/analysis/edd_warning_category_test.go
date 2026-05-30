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

// Issue #776 piece C: EDDWarning gets a Category field so consumers
// can route by finding class instead of parsing Reason text. This
// test pins that the existing two emitters (unused, write-only) set
// the right category, and that the Reason field stays populated for
// back-compat.

const writeOnlyEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="client" access="rw">
    <field name="never_read" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
    <field name="declared_but_never_seen" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
  </entity>
</entity_data_dictionary>
`

const writeOnlyDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>WriteOnlyDemo</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>set client.never_read = true;</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

// TestEDDWarningCategory_UnusedAndWriteOnly: each category is emitted
// with the right tag, the Reason field stays human-readable, and the
// EddFile / Field fields keep their existing meaning.
func TestEDDWarningCategory_UnusedAndWriteOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x_edd.xml"), []byte(writeOnlyEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x_dt.xml"), []byte(writeOnlyDT), 0644); err != nil {
		t.Fatal(err)
	}

	warnings, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage: %v", err)
	}

	byField := map[string]EDDWarning{}
	for _, w := range warnings {
		byField[w.Field] = w
	}

	wo, ok := byField["client.never_read"]
	if !ok {
		t.Fatalf("expected write-only warning for client.never_read, got %v", warnings)
	}
	if wo.Category != EDDUsageWriteOnly {
		t.Errorf("client.never_read category = %q, want %q", wo.Category, EDDUsageWriteOnly)
	}
	if wo.Reason == "" {
		t.Errorf("Reason must remain populated for back-compat, got empty")
	}

	un, ok := byField["client.declared_but_never_seen"]
	if !ok {
		t.Fatalf("expected unused warning for client.declared_but_never_seen, got %v", warnings)
	}
	if un.Category != EDDUsageUnused {
		t.Errorf("client.declared_but_never_seen category = %q, want %q", un.Category, EDDUsageUnused)
	}
	if un.Reason == "" {
		t.Errorf("Reason must remain populated for back-compat, got empty")
	}
}

// TestEDDWarningCategory_ConstantsAreStable: the public category
// constants are part of the API surface — callers route on them.
// Pin the literal string values so a rename has to be deliberate.
func TestEDDWarningCategory_ConstantsAreStable(t *testing.T) {
	cases := []struct {
		got, want EDDUsageCategory
	}{
		{EDDUsageUnused, "unused"},
		{EDDUsageWriteOnly, "write_only"},
		{EDDUsagePossibly, "possibly_used"},
	}
	for _, c := range cases {
		if string(c.got) != string(c.want) {
			t.Errorf("category constant drift: got %q, want %q", c.got, c.want)
		}
	}
}
