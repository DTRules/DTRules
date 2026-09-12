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

package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// `create <Type> as <alias>` binds a local, and every reference through it is
// dotted — `st_result.state_withholding`. The plain dotted scan sees that and
// attributes it to an entity called st_result, which the EDD has never heard
// of; the field it really names goes unreferenced and is reported unused while
// being written on every multi-state return (#776).

func writeAliasFixture(t *testing.T, dt string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("A_edd.xml", `<entity_data_dictionary>
		<entity name="job">
			<field name="results" type="array" subtype="tax_result"></field>
		</entity>
		<entity name="tax_result">
			<field name="amount" type="double"></field>
			<field name="rate" type="double"></field>
			<field name="never_touched" type="double"></field>
		</entity>
	</entity_data_dictionary>`)
	write("A_dt.xml", dt)
	return dir
}

func reportedBy(t *testing.T, dir string) map[string]EDDUsageCategory {
	t.Helper()
	warns, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	out := map[string]EDDUsageCategory{}
	for _, w := range warns {
		out[w.Field] = w.Category
	}
	return out
}

func TestAliasedWriteIsAttributedToTheEntity(t *testing.T) {
	dir := writeAliasFixture(t, `<decision_tables><decision_table>
		<table_name>T</table_name>
		<actions><action_details>
			<action_dsl>create tax_result as r; set r.amount = 1.0; add r to job.results</action_dsl>
		</action_details></actions>
	</decision_table></decision_tables>`)

	got := reportedBy(t, dir)
	if got["tax_result.amount"] == EDDUsageUnused {
		t.Error("tax_result.amount is written through the alias r, but was reported unused")
	}
	if _, ok := got["tax_result.never_touched"]; !ok {
		t.Error("a field nothing touches should still be reported — the fix must not silence findings")
	}
}

func TestAliasedReadIsAttributedToTheEntity(t *testing.T) {
	dir := writeAliasFixture(t, `<decision_tables><decision_table>
		<table_name>T</table_name>
		<actions><action_details>
			<action_dsl>create tax_result as r; set r.amount = r.rate * 2.0</action_dsl>
		</action_details></actions>
	</decision_table></decision_tables>`)

	got := reportedBy(t, dir)
	if got["tax_result.rate"] == EDDUsageUnused {
		t.Error("tax_result.rate is read through the alias r, but was reported unused")
	}
}

// The other spelling of the same binding.
func TestLocalEntityFormIsAlsoAttributed(t *testing.T) {
	dir := writeAliasFixture(t, `<decision_tables><decision_table>
		<table_name>T</table_name>
		<contexts><context_details>
			<context_dsl>local entity r = new tax_result entity</context_dsl>
		</context_details></contexts>
		<actions><action_details>
			<action_dsl>set r.amount = 1.0</action_dsl>
		</action_details></actions>
	</decision_table></decision_tables>`)

	if reportedBy(t, dir)["tax_result.amount"] == EDDUsageUnused {
		t.Error("a local entity bound with `local entity r = new tax_result entity` was not resolved")
	}
}

// The create and the uses routinely sit in different actions of one table, so
// the binding is collected table-wide rather than per row.
func TestAliasBindingSpansTheWholeTable(t *testing.T) {
	dir := writeAliasFixture(t, `<decision_tables><decision_table>
		<table_name>T</table_name>
		<actions>
			<action_details><action_dsl>create tax_result as r</action_dsl></action_details>
			<action_details><action_dsl>set r.amount = 1.0</action_dsl></action_details>
		</actions>
	</decision_table></decision_tables>`)

	if reportedBy(t, dir)["tax_result.amount"] == EDDUsageUnused {
		t.Error("the alias was bound in one action and used in another; the binding is table-wide")
	}
}

// An alias naming a type the EDD does not declare cannot rewrite a reference
// into anything real, and guessing would invent references to fields that do
// not exist.
func TestAliasForAnUndeclaredTypeIsIgnored(t *testing.T) {
	dir := writeAliasFixture(t, `<decision_tables><decision_table>
		<table_name>T</table_name>
		<actions><action_details>
			<action_dsl>create not_an_entity as r; set r.amount = 1.0</action_dsl>
		</action_details></actions>
	</decision_table></decision_tables>`)

	got := reportedBy(t, dir)
	if got["tax_result.amount"] != EDDUsageUnused {
		t.Errorf("tax_result.amount = %q — an alias for an undeclared type invented a reference",
			got["tax_result.amount"])
	}
}
