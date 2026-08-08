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

package authoring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

const nestedFixedEDD = `<entity_data_dictionary version='2'>
	<entity name='budget' access='rw' comment=''>
		<field name='supply_limit' type='fixed' access='r' comment=''></field>
		<field name='acme_issued' type='fixed' access='r' comment=''></field>
		<field name='unissued_supply' type='fixed' access='w' comment=''></field>
	</entity>
</entity_data_dictionary>`

const nestedFixedDT = `<?xml version="1.0" encoding="UTF-8"?>
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

// TestNestedEDDIsLoaded (#879): loadEDD must discover EDD files in
// subdirectories (e.g. xml/states/budget_edd.xml), the same recursive scope
// loadDTFiles and the build path use. A flat Glob missed nested EDDs, so the
// fields they declare stayed untyped and the emitter degraded their fixed ops
// to integer (`-`/`cvi`) — a #874-class divergence on a different vector.
//
// Here the EDD lives under xml/states/ while the table lives at xml/; the
// fixed subtraction must still compile to fp-/cvfp through the Save path.
func TestNestedEDDIsLoaded(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	statesDir := filepath.Join(xmlDir, "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(xmlDir, "budget_dt.xml")
	if err := os.WriteFile(filepath.Join(statesDir, "budget_edd.xml"), []byte(nestedFixedEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dtPath, []byte(nestedFixedDT), 0o644); err != nil {
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
	if !strings.Contains(postfix, "fp-") || !strings.Contains(postfix, "cvfp") {
		t.Errorf("nested EDD not loaded: expected fixed ops fp-/cvfp, got %q", postfix)
	}
}
