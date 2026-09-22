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

// Issue #1252: `using <entity> ( <string expression> )` compiled to nothing,
// so an action assigning it stored an empty value and a condition comparing
// it compared whatever was left on the stack.
//
// The test drives the production chain: the authoring SDK compiles the EL and
// writes the postfix on Save, the strict loader reads the XML into a rule
// set, and the session executes the decision table.

const strUsingEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='a' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='acct' type='entity' subtype='acct' access='rw' input='' default_value='' comment=''></field>
		<field name='text' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
		<field name='matched' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
	<entity name='acct' access='rw' comment=''>
		<field name='acct' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='owner' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
</entity_data_dictionary>`

const strUsingTableXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>StrUsing</table_name>
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
</decision_tables>`

func TestStrUsing_EvaluatesInTheEntityScope(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_edd.xml"), []byte(strUsingEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_dt.xml"), []byte(strUsingTableXML), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("StrUsing")
	if tbl == nil {
		t.Fatal("table StrUsing not found")
	}
	// `owner` is a field of acct, not of calc: it resolves only while
	// calc.acct is on the entity stack.
	for _, el := range []string{
		`set calc.text = using calc.acct ( "owner:" + owner )`,
		`set calc.matched = "yes"`,
	} {
		if err := tbl.AddAction(authoring.Action{DSL: el}); err != nil {
			t.Fatalf("AddAction %q: %v", el, err)
		}
	}
	for _, el := range []string{
		`calc.a > 0`,
		`using calc.acct ( "owner:" + owner ) == "owner:alice"`,
	} {
		if err := tbl.AddCondition(authoring.Condition{DSL: el}); err != nil {
			t.Fatalf("AddCondition %q: %v", el, err)
		}
	}
	// Column 1: both conditions true → both actions. Column 2: the using
	// condition false → only the text action, so a using condition that
	// evaluated to nothing cannot pass by landing in the other column.
	if err := tbl.AddColumn(map[int]string{1: "Y", 2: "Y"}, []int{1, 2}); err != nil {
		t.Fatalf("AddColumn: %v", err)
	}
	if err := tbl.AddColumn(map[int]string{1: "Y", 2: "N"}, []int{1}); err != nil {
		t.Fatalf("AddColumn: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	run := func(owner string) (dtrules.Entity, error) {
		t.Helper()
		rs := session.NewRuleSet("strusing")
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
		calc, err := ef.CreateEntity(sess, dtrules.GetRName("calc"))
		if err != nil {
			t.Fatalf("CreateEntity calc: %v", err)
		}
		acct, err := ef.CreateEntity(sess, dtrules.GetRName("acct"))
		if err != nil {
			t.Fatalf("CreateEntity acct: %v", err)
		}
		acct.Put(dtrules.GetRName("owner"), dtrules.NewRString(owner))
		calc.Put(dtrules.GetRName("a"), dtrules.GetRIntegerValue(1))
		calc.Put(dtrules.GetRName("acct"), acct.(dtrules.Object))
		state.EntityPush(calc)
		defer state.EntityPop()
		return calc, sess.Execute("StrUsing")
	}

	for _, tc := range []struct {
		owner, text, matched string
	}{
		{"alice", "owner:alice", "yes"},
		{"bob", "owner:bob", ""},
	} {
		calc, err := run(tc.owner)
		if err != nil {
			t.Fatalf("owner %s: Execute: %v", tc.owner, err)
		}
		get := func(field string) string {
			v, err := calc.Get(dtrules.GetRName(field))
			if err != nil || v == nil {
				t.Fatalf("get %s: %v", field, err)
			}
			return v.StringValue()
		}
		if got := get("text"); got != tc.text {
			t.Errorf("owner %s: text = %q, want %q", tc.owner, got, tc.text)
		}
		if got := get("matched"); got != tc.matched {
			t.Errorf("owner %s: matched = %q, want %q", tc.owner, got, tc.matched)
		}
	}
}
