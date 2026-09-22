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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// This harness runs compiled EL through the production chain: the authoring
// SDK compiles each row and writes its postfix to the project XML on Save,
// the strict loader reads the XML into a rule set, and a session executes
// the table. Nothing in between is done by hand.
//
// The rule set has two entities. `calc` is the current entity when the table
// runs; its `account` field holds an `account` entity. Both declare n, d and
// flag, so a test can tell which of the two a reference resolved against.

const elrtEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='account' type='entity' subtype='account' access='rw' input='' default_value='' comment=''></field>
		<field name='n' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='d' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='flag' type='boolean' subtype='' access='rw' input='' default_value='false' comment=''></field>
		<field name='text' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
		<field name='hit' type='boolean' subtype='' access='rw' input='' default_value='false' comment=''></field>
		<field name='r' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='rd' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
	</entity>
	<entity name='account' access='rw' comment=''>
		<field name='account' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='n' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='d' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='flag' type='boolean' subtype='' access='rw' input='' default_value='false' comment=''></field>
	</entity>
</entity_data_dictionary>`

func elrtTableXML(name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>` + name + `</table_name>
<xls_file>elrt_dt.xlsx</xls_file>
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
</decision_tables>`
}

// elrtRun authors a table whose one column requires every condition to be
// true (a column needs at least one; none means `1 == 1`) and then fires
// every action, saves it, loads it with the production loader and executes
// it with calc current. set, if not nil, seeds the two entities first. It
// returns them for the caller to inspect.
func elrtRun(t *testing.T, conditions, actions []string, set func(calc, account dtrules.Entity)) (calc, account dtrules.Entity, err error) {
	t.Helper()
	const name = "ELRuntime"
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "elrt_edd.xml"), []byte(elrtEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	dtPath := filepath.Join(xmlDir, "elrt_dt.xml")
	if err := os.WriteFile(dtPath, []byte(elrtTableXML(name)), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table(name)
	if tbl == nil {
		t.Fatalf("table %s not found", name)
	}
	if len(conditions) == 0 {
		// A column with no condition rows never fires; give it one that
		// always holds.
		conditions = []string{`1 == 1`}
	}
	conds := map[int]string{}
	for i, el := range conditions {
		if err := tbl.AddCondition(authoring.Condition{DSL: el}); err != nil {
			t.Fatalf("AddCondition %q: %v", el, err)
		}
		conds[i+1] = "Y"
	}
	nums := make([]int, 0, len(actions))
	for i, el := range actions {
		if err := tbl.AddAction(authoring.Action{DSL: el}); err != nil {
			t.Fatalf("AddAction %q: %v", el, err)
		}
		nums = append(nums, i+1)
	}
	if err := tbl.AddColumn(conds, nums); err != nil {
		t.Fatalf("AddColumn: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			saved, _ := os.ReadFile(dtPath)
			t.Logf("saved table:\n%s", saved)
		}
	})

	rs := session.NewRuleSet("elrt")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("LoadFromDirectory: %v", err)
	}
	sess, err := session.NewSession(rs)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	state := sess.GetState().(*interpreter.DTState)
	state.SetOperatorTable(operators.GetOperatorTable())

	ef := sess.GetEntityFactory().(*entity.Factory)
	calc, err = ef.CreateEntity(sess, dtrules.GetRName("calc"))
	if err != nil {
		t.Fatalf("CreateEntity calc: %v", err)
	}
	account, err = ef.CreateEntity(sess, dtrules.GetRName("account"))
	if err != nil {
		t.Fatalf("CreateEntity account: %v", err)
	}
	calc.Put(dtrules.GetRName("account"), account)
	if set != nil {
		set(calc, account)
	}

	state.EntityPush(calc)
	defer state.EntityPop()
	return calc, account, sess.Execute(name)
}

func elrtGet(t *testing.T, e dtrules.Entity, field string) dtrules.Object {
	t.Helper()
	v, err := e.Get(dtrules.GetRName(field))
	if err != nil || v == nil {
		t.Fatalf("get %s: %v", field, err)
	}
	return v
}
