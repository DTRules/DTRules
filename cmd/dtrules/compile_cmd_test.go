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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompile_FixedOperandsEmitFixedOps is the #790 regression. Before
// v1.14.2 the standalone `dtrules compile` did not pass a symbol table
// to the EL compiler, so operations on `fixed`-typed operands compiled
// to integer operators (`-`, `min`, `cvi`) — postfix that the strict
// runtime then rejected at execution time with
// "[Conversion Error] IntValue: No Integer value exists for this type"
// because the actual operands were `*RFixed`.
//
// The fix: discover and load *_edd.xml beside the target, build a
// symbol table, call `cmp.SetSymbols(symbols)` before the compile
// loop. With that in place, the EL compiler picks `fp-` / `fpmin` /
// `cvfp` for fixed operands.
//
// This test pins that behavior end-to-end through the compile-file
// pipeline: write a small project (EDD + DT) to a temp dir, run
// `dtrules compile`, and assert the emitted postfix contains the
// fixed-point operators. A regression here would mean SetSymbols was
// dropped from the call site again or stopped being passed the EDD
// type info.
func TestCompile_FixedOperandsEmitFixedOps(t *testing.T) {
	dir := t.TempDir()

	// Minimal EDD: a single `job` entity with three `fixed`-typed
	// fields that mirror staking's withholding-table operands. These
	// are the operands the EL compiler must see as `fixed` for the
	// dispatcher to pick fp- and fpmin instead of integer ops.
	const eddXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="effective_withholding" type="fixed" subtype="" access="rw" input="" default_value="0fp" comment=""/>
    <field name="withholding_cap"       type="fixed" subtype="" access="r"  input="" default_value="0fp" comment=""/>
    <field name="withholding_paid"      type="fixed" subtype="" access="r"  input="" default_value="0fp" comment=""/>
  </entity>
</entity_data_dictionary>
`

	// DT with the smoking-gun DSL: `set fixed = the minimum of fixed and (fixed - fixed)`.
	// Postfix is left empty so compile must produce it from the DSL.
	const dtXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Calculate_Withholding</table_name>
<xls_file>calc.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>always</condition_comment>
    <condition_dsl>withholding_cap &gt; 0fp</condition_dsl>
    <condition_postfix>withholding_cap 0fp fp&gt;</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>cap withholding</action_comment>
    <action_dsl>set effective_withholding = the minimum of effective_withholding and (withholding_cap - withholding_paid)</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

	if err := os.WriteFile(filepath.Join(dir, "fixture_edd.xml"), []byte(eddXML), 0644); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(dir, "fixture_dt.xml")
	if err := os.WriteFile(dtPath, []byte(dtXML), 0644); err != nil {
		t.Fatal(err)
	}

	// Run the same code path `dtrules compile <dir>` runs. We exercise
	// the CLI dispatch (not just compileFile directly) so the EDD
	// discovery + SetSymbols wiring is also covered.
	cli := NewCLI()
	if code := cli.runCompile([]string{dir}); code != 0 {
		t.Fatalf("runCompile exit=%d; want 0", code)
	}

	out, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	postfix := string(out)

	// Positive: the compiled postfix must use the fixed-point operators.
	// `fp-` is the diagnostic the issue specifically calls out; pinning
	// it is the regression catch. `fpmin` confirms `the minimum of` also
	// dispatches correctly. `cvfp` is the assignment-conversion op for
	// fixed (vs `cvi` for integer).
	for _, want := range []string{"fp-", "fpmin", "cvfp"} {
		if !strings.Contains(postfix, want) {
			t.Errorf("compiled postfix missing %q; got:\n%s", want, postfix)
		}
	}

	// Negative: the integer-op pattern from the bug must NOT appear.
	// Spaces around the operators rule out accidental substring hits
	// (e.g. matching `fpmin` instead of bare `min`).
	for _, banned := range []string{" - cvi ", " - min ", " min cvi "} {
		if strings.Contains(postfix, banned) {
			t.Errorf("compiled postfix still contains banned int-op pattern %q; got:\n%s", banned, postfix)
		}
	}
}

// TestCompile_LocalsResetPerTable_Issue814 is the #814 regression.
// Before the fix, `compileFile` called `cmp.ResetLocals()` once per
// FILE rather than per TABLE. With two tables in one file that each
// declared a local with the same name, the slot counter and name map
// leaked across them — Table A's actions ended up referencing Table
// B's slot index (last write wins on the map). At runtime, Table A's
// frame had only its own slot 0 → `OutOfBounds` on GetFrameValue.
//
// The fix processes each `<decision_table>` block in isolation and
// calls ResetLocals at every boundary. This test pins that behavior
// by compiling two tables with a same-named local and asserting both
// tables' postfix references slot 0 — the canonical-isolated index.
//
// Reproducer (pre-fix output, with the bug):
//
//	TableA condition postfix: "1 local@ /value get 0 >"   ← wrong: slot 1 doesn't exist in A's frame
//	TableB condition postfix: "1 local@ /value get 0 >"
//
// Post-fix output:
//
//	TableA condition postfix: "0 local@ /value get 0 >"
//	TableB condition postfix: "0 local@ /value get 0 >"
func TestCompile_LocalsResetPerTable_Issue814(t *testing.T) {
	dir := t.TempDir()

	// EDD: two entity types (one per table) plus a job entity for the
	// action's RHS to assign into.
	const eddXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="result_a" access="rw">
    <field name="value" type="integer" subtype="" access="rw" input="" default_value="0" comment=""/>
  </entity>
  <entity name="result_b" access="rw">
    <field name="value" type="integer" subtype="" access="rw" input="" default_value="0" comment=""/>
  </entity>
  <entity name="job" access="rw">
    <field name="summary" type="integer" subtype="" access="rw" input="" default_value="0" comment=""/>
  </entity>
</entity_data_dictionary>
`

	// Two tables, each declaring a local entity `r` in their context.
	// Both reference `r.value` in conditions and actions. Pre-fix,
	// each table would compile against the WRONG slot because the
	// slot map leaked across tables.
	const dtXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>TableA</table_name>
<xls_file>x.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts>
  <context_details>
    <context_number>1</context_number>
    <context_comment></context_comment>
    <context_dsl>local entity r = new result_a entity</context_dsl>
    <context_postfix></context_postfix>
  </context_details>
</contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>always</condition_comment>
    <condition_dsl>r.value &gt; 0</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment></action_comment>
    <action_dsl>set summary = r.value</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
<decision_table>
<table_name>TableB</table_name>
<xls_file>x.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>2</TABLE_NUMBER></attribute_fields>
<contexts>
  <context_details>
    <context_number>1</context_number>
    <context_comment></context_comment>
    <context_dsl>local entity r = new result_b entity</context_dsl>
    <context_postfix></context_postfix>
  </context_details>
</contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>always</condition_comment>
    <condition_dsl>r.value &gt; 0</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment></action_comment>
    <action_dsl>set summary = r.value</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

	if err := os.WriteFile(filepath.Join(dir, "fixture_edd.xml"), []byte(eddXML), 0644); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(dir, "fixture_dt.xml")
	if err := os.WriteFile(dtPath, []byte(dtXML), 0644); err != nil {
		t.Fatal(err)
	}

	cli := NewCLI()
	if code := cli.runCompile([]string{dir}); code != 0 {
		t.Fatalf("runCompile exit=%d; want 0", code)
	}

	out, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	// Split the output by table so we can assert each table's slot
	// references independently.
	splitAt := strings.Index(got, "<table_name>TableB</table_name>")
	if splitAt < 0 {
		t.Fatalf("TableB section not found in compiled output")
	}
	tableA := got[:splitAt]
	tableB := got[splitAt:]

	// Each table's locals reset cleanly: the only `r` in scope is the
	// one declared in *its own* context, which always lands at slot 0.
	// Pre-fix, TableA's postfix contained "1 local@" because TableB's
	// declaration had bumped the counter.
	for _, sec := range []struct{ name, body string }{
		{"TableA", tableA},
		{"TableB", tableB},
	} {
		// Positive: the canonical slot-0 access must appear in both
		// the condition (read) and the action (read).
		want := "0 local@ /value get"
		if !strings.Contains(sec.body, want) {
			t.Errorf("%s missing canonical slot-0 access %q; got:\n%s",
				sec.name, want, sec.body)
		}
		// Negative: the leaked slot-1 access from the pre-fix bug must
		// NOT appear. (Other slot numbers ≥ 1 would be equally wrong
		// here — only one local is declared per table.)
		if strings.Contains(sec.body, "1 local@") {
			t.Errorf("%s still contains `1 local@` — locals leaked across tables (#814); body:\n%s",
				sec.name, sec.body)
		}
	}
}
