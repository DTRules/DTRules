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

// TestAnalyzeEDDUsage_BareNamesUnderForAll exercises the #776 phase 1
// entity-stack-aware path: when a table's context is `for all <field>`,
// bare identifiers in conditions/actions should resolve to fields of
// the iterated entity rather than counting as unknown tokens.
//
// Without this resolution, the analyzer used to mark every field
// accessed via the bare-name path as "unused EDD field." The downstream
// staking project reported 97 such false positives. The fixture below
// reproduces the minimum signal: an entity `taxpayer` with an `agi`
// field referenced bare from inside a `for all taxpayers` context.
// The analyzer must not flag `taxpayer.agi` as unused.
func TestAnalyzeEDDUsage_BareNamesUnderForAll(t *testing.T) {
	dir := t.TempDir()

	const edd = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="taxpayers" type="array" subtype="taxpayer" access="r" input="" default_value="" comment="">
    </field>
    <field name="incomes" type="array" subtype="income" access="r" input="" default_value="" comment="">
    </field>
  </entity>
  <entity name="taxpayer" access="rw">
    <field name="agi" type="double" subtype="" access="rw" input="" default_value="0" comment="">
    </field>
    <field name="name" type="string" subtype="" access="r" input="main" default_value="" comment="">
    </field>
    <field name="forgotten_field" type="double" subtype="" access="r" input="" default_value="0" comment="">
    </field>
  </entity>
  <entity name="income" access="rw">
    <field name="amount" type="double" subtype="" access="r" input="main" default_value="0" comment="">
    </field>
  </entity>
</entity_data_dictionary>
`

	// A DT that iterates taxpayers and references their fields by
	// bare name. The regex-only pass would have flagged taxpayer.agi
	// and taxpayer.name as "unused" — the bare-name pass should
	// pick them up and only forgotten_field should remain unused.
	const dt = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Per_Taxpayer</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts>
  <context_details>
    <context_number>1</context_number>
    <context_dsl>for all taxpayers</context_dsl>
    <context_postfix></context_postfix>
  </context_details>
</contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>agi > 0</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>add "processed " + name to job.audit_trail</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"></action_column>
  </action_details>
</actions>
<policy_statements></policy_statements>
</decision_table>
</decision_tables>
`

	if err := os.WriteFile(filepath.Join(dir, "test_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatalf("write edd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatalf("write dt: %v", err)
	}

	warnings, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage: %v", err)
	}

	// The fixture has three taxpayer fields: agi (read+set context),
	// name (read), forgotten_field (never referenced). Only
	// forgotten_field should appear as unused.
	wantUnused := map[string]bool{
		"taxpayer.forgotten_field": true,
		// job.incomes IS declared but never used — the analyzer
		// should still flag it. Confirms the bare-name pass doesn't
		// shadow the existing dotted detection.
		"job.incomes": true,
	}
	mustNotBeUnused := map[string]bool{
		"taxpayer.agi":  true,
		"taxpayer.name": true,
		// job.taxpayers is read by the `for all taxpayers` context.
		// extractIteratedFieldReads counts the iterating field
		// itself as a read so it's not flagged as unused.
		"job.taxpayers": true,
	}

	got := map[string]bool{}
	for _, w := range warnings {
		got[w.Field] = true
	}
	for field := range wantUnused {
		if !got[field] {
			t.Errorf("expected %q to be flagged unused, got warnings %v", field, warnings)
		}
	}
	for field := range mustNotBeUnused {
		if got[field] {
			t.Errorf("did NOT expect %q to be flagged unused — it's referenced via the bare-name path inside `for all taxpayers`; got %v", field, warnings)
		}
	}
}

// TestStackFromContexts_DottedAndBare covers both `for all taxpayers`
// (bare field on some root entity) and `for all job.state_periods`
// (explicit dotted reference). Both must resolve to the field's
// declared array subtype.
func TestStackFromContexts_DottedAndBare(t *testing.T) {
	schema := &eddSchema{
		Fields:         map[string]eddField{},
		ArraySubtype:   map[string]string{"job.taxpayers": "taxpayer", "job.state_periods": "state_period"},
		FieldsByEntity: map[string]map[string]bool{},
	}
	cases := []struct {
		name string
		dsl  string
		want []string
	}{
		{"bare", "for all taxpayers", []string{"taxpayer"}},
		{"dotted", "for all job.state_periods", []string{"state_period"}},
		{"with where clause", "for all taxpayers whose is_self_employed is true", []string{"taxpayer"}},
		{"unknown field", "for all gremlins", nil},
		{"no for-all", "perform when called", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stackFromContexts(schema, []contextEntry{{DSL: c.dsl}})
			if !equalStringSlices(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestAttribute_PrefersInnermost: with two entities on the stack and a
// field name that exists on both, the innermost wins. Matches the
// runtime entity-stack lookup order.
func TestAttribute_PrefersInnermost(t *testing.T) {
	schema := &eddSchema{
		FieldsByEntity: map[string]map[string]bool{
			"outer": {"shared": true},
			"inner": {"shared": true},
		},
	}
	got := attribute("shared", []string{"outer", "inner"}, schema)
	if got != "inner.shared" {
		t.Errorf("got %q, want inner.shared (innermost-first)", got)
	}
}

// TestAttribute_OnlyOuterDeclares: when only the outer entity declares
// a matching field, the bare reference still resolves — matches Go's
// scope-chain semantics and the EL runtime's entity-stack walk.
func TestAttribute_OnlyOuterDeclares(t *testing.T) {
	schema := &eddSchema{
		FieldsByEntity: map[string]map[string]bool{
			"outer": {"only_on_outer": true},
			"inner": {"only_on_inner": true},
		},
	}
	got := attribute("only_on_outer", []string{"outer", "inner"}, schema)
	if got != "outer.only_on_outer" {
		t.Errorf("got %q, want outer.only_on_outer", got)
	}
}

// TestExtractBareReads_SkipsStringsAndKeywords confirms the bare-name
// pass does not pick up identifiers inside string literals or EL
// keywords. Both classes would generate noise if unfiltered.
func TestExtractBareReads_SkipsStringsAndKeywords(t *testing.T) {
	schema := &eddSchema{
		FieldsByEntity: map[string]map[string]bool{
			"taxpayer": {"name": true, "agi": true, "filing": true},
		},
	}
	stack := []string{"taxpayer"}
	reads := map[string]bool{}

	// `name` is a real field reference. `"name"` (in quotes) is a
	// string literal — must not count. `is` / `equal` / `to` / `and`
	// / `or` are EL keywords — must not count, even though some of
	// them would otherwise look like identifiers.
	extractBareReads(`name is equal to "name" and agi > 0`, stack, schema, reads)

	if !reads["taxpayer.name"] {
		t.Errorf("expected taxpayer.name as read, got %v", reads)
	}
	if !reads["taxpayer.agi"] {
		t.Errorf("expected taxpayer.agi as read, got %v", reads)
	}
	for k := range reads {
		if !strings.HasPrefix(k, "taxpayer.") {
			t.Errorf("unexpected non-taxpayer read %q", k)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
