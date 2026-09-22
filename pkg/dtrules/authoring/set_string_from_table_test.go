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

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Issue #1255: `set <string> = new "x" table of "y"` (setStringFromTable)
// compiled to empty postfix. Hash tables were removed, and every other table
// form compiles to a "hash tables removed" elstmterror (docs/el-reference.md
// §Table lookup), so this one must too.
//
// Driven through the production chain: the authoring SDK compiles and saves,
// the strict loader loads, the session executes.

const setStrTableEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='a' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='text' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
		<field name='other' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
</entity_data_dictionary>`

const setStrTableXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>Other</table_name>
<xls_file>calc_dt.xlsx</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS></COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
<policy_statements></policy_statements>
</decision_table>
<decision_table el_compiled="true">
<table_name>NewTable</table_name>
<xls_file>calc_dt.xlsx</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS></COMMENTS>
<TABLE_NUMBER>2</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
<policy_statements></policy_statements>
</decision_table>
</decision_tables>`

func TestSetStringFromTable_IsAHashTablesRemovedError(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_edd.xml"), []byte(setStrTableEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_dt.xml"), []byte(setStrTableXML), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	author := func(name string, actions ...string) {
		t.Helper()
		tbl := p.Table(name)
		if tbl == nil {
			t.Fatalf("table %s not found", name)
		}
		nums := []int{}
		for i, el := range actions {
			if err := tbl.AddAction(authoring.Action{DSL: el}); err != nil {
				t.Fatalf("AddAction %q: %v", el, err)
			}
			nums = append(nums, i+1)
		}
		if err := tbl.AddCondition(authoring.Condition{DSL: "calc.a > 0"}); err != nil {
			t.Fatalf("AddCondition: %v", err)
		}
		if err := tbl.AddColumn(map[int]string{1: "Y"}, nums); err != nil {
			t.Fatalf("AddColumn: %v", err)
		}
	}
	// A table in the same file that does not use the form: the file must
	// still load, so an unsupported statement costs only its own table.
	author("Other", `set calc.other = "ran"`)
	author("NewTable", `set calc.text = new "x" table of "y"`, `set calc.other = "after"`)
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rs := session.NewRuleSet("setstrtable")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("LoadFromDirectory: %v", err)
	}
	run := func(table string) (dtrules.Entity, error) {
		sess, err := session.NewSession(rs)
		if err != nil {
			t.Fatalf("NewSession: %v", err)
		}
		state := sess.GetState().(*interpreter.DTState)
		state.SetOperatorTable(operators.GetOperatorTable())
		ef := sess.GetEntityFactory().(*entity.Factory)
		calc, err := ef.CreateEntity(sess, dtrules.GetRName("calc"))
		if err != nil {
			t.Fatalf("CreateEntity: %v", err)
		}
		calc.Put(dtrules.GetRName("a"), dtrules.GetRIntegerValue(1))
		state.EntityPush(calc)
		defer state.EntityPop()
		return calc, sess.Execute(table)
	}

	if calc, err := run("Other"); err != nil {
		t.Fatalf("Execute Other: %v", err)
	} else if v, _ := calc.Get(dtrules.GetRName("other")); v.StringValue() != "ran" {
		t.Errorf("Other: calc.other = %q, want \"ran\"", v.StringValue())
	}

	calc, err := run("NewTable")
	if err == nil {
		v, _ := calc.Get(dtrules.GetRName("other"))
		t.Fatalf("Execute NewTable succeeded (calc.other = %q); want a hash-tables-removed error", v.StringValue())
	}
	if !strings.Contains(err.Error(), "hash tables removed") {
		t.Errorf("error = %v; want it to say hash tables removed", err)
	}
	if v, _ := calc.Get(dtrules.GetRName("other")); v.StringValue() == "after" {
		t.Error("the action after the failed statement ran")
	}
}
