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

package authoring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// fixedEDD declares three fixed-point fields. The DSL below references them by
// bare name, so the postfix emitter can only pick fixed-point ops if loadEDD
// registered the bare keys (#874).
const fixedEDD = `<entity_data_dictionary version='2'>
	<entity name='budget' access='rw' comment=''>
		<field name='supply_limit' type='fixed' access='r' comment=''></field>
		<field name='acme_issued' type='fixed' access='r' comment=''></field>
		<field name='unissued_supply' type='fixed' access='w' comment=''></field>
	</entity>
</entity_data_dictionary>`

const fixedDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>Calculate_Budget</table_name>
<xls_file>budget_dt.xlsx</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS></COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions>
</initial_actions>
<conditions>
</conditions>
<actions>
<action_details>
<action_number>1</action_number>
<action_comment>Compute unissued supply</action_comment>
<action_dsl>set unissued_supply = supply_limit - acme_issued</action_dsl>
<action_postfix>
supply_limit acme_issued - cvi /unissued_supply xdef
</action_postfix>
<action_column column_number="1" column_value="X"></action_column>
</action_details>
</actions>
<policy_statements>
</policy_statements>
</decision_table>
</decision_tables>
`

// TestSaveFixedSubtractionEmitsFixedOps is the #874 regression: re-saving a
// table whose action subtracts two fixed-typed fields must keep fixed-point
// ops (fp-, cvfp). Before the fix, loadEDD registered only entity-qualified
// symbol keys, so the bare-name operands resolved as unknown and the emitter
// fell back to integer ops (-, cvi), silently corrupting the postfix.
func TestSaveFixedSubtractionEmitsFixedOps(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(xmlDir, "budget_dt.xml")
	if err := os.WriteFile(filepath.Join(xmlDir, "budget_edd.xml"), []byte(fixedEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dtPath, []byte(fixedDT), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Calculate_Budget")
	if tbl == nil {
		t.Fatal("table Calculate_Budget not found")
	}

	// Re-save the action with its own unchanged DSL — this recompiles postfix
	// from DSL through the Save path (the canonical generator since #817).
	a := tbl.Actions[0]
	if err := tbl.UpdateAction(1, authoring.Action{DSL: a.DSL, Comment: a.Comment}); err != nil {
		t.Fatalf("UpdateAction: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := os.ReadFile(dtPath)
	if err != nil {
		t.Fatal(err)
	}
	postfix := extractActionPostfix(t, string(out))

	if !strings.Contains(postfix, "fp-") {
		t.Errorf("expected fixed-point subtraction op fp-, got postfix: %q", postfix)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected convert-to-fixed op cvfp, got postfix: %q", postfix)
	}
	for _, intOp := range []string{" cvi", " - "} {
		if strings.Contains(" "+postfix+" ", intOp) {
			t.Errorf("postfix still emits integer op %q (fixed operands degraded): %q", strings.TrimSpace(intOp), postfix)
		}
	}
}

// extractActionPostfix pulls the text inside the first <action_postfix> tag.
func extractActionPostfix(t *testing.T, xml string) string {
	t.Helper()
	const open, close = "<action_postfix>", "</action_postfix>"
	i := strings.Index(xml, open)
	j := strings.Index(xml, close)
	if i < 0 || j < 0 || j < i {
		t.Fatalf("no <action_postfix> in saved XML:\n%s", xml)
	}
	return strings.TrimSpace(xml[i+len(open) : j])
}
