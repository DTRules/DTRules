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
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Issue #1270: every value-returning `using <entity> ( <expr> )` form left the
// wrong thing on the data stack. entitypop pushes the popped entity, so
// `entitypop swap pop` (boolUsing, nameUsing) kept the entity and dropped the
// value, and a bare `entitypop` (floatUsing, intUsingArray, bigUsing,
// dateUsing) left the entity on top of it. Each form here is driven through
// the production chain: authoring SDK → Save → LoadFromDirectory →
// Session.Execute. Each case is named for the grammar label its EL reaches
// (checked against the parse tree). intUsing has no case: every
// `using e ( <int expr> )` tried parses as intUsingArray or floatUsing.

const usingFormsEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='a' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='acct' type='entity' subtype='acct' access='rw' input='' default_value='' comment=''></field>
		<field name='text' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
		<field name='b' type='boolean' subtype='' access='rw' input='' default_value='false' comment=''></field>
		<field name='nm' type='name' subtype='' access='rw' input='' default_value='' comment=''></field>
		<field name='d' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='i' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='big' type='bigint' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='when' type='date' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
	<entity name='acct' access='rw' comment=''>
		<field name='acct' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='owner' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
		<field name='bal' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='n' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='opened' type='date' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
</entity_data_dictionary>`

const usingFormsTableXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>Using</table_name>
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

// runUsingForm authors table Using, saves it, loads it with the production
// loader and executes it against calc (a=1) whose acct is owner=alice,
// bal=2.5, n=7, opened=2026-01-15.
//
// With cond == "", the table is one column running action. With a cond, the
// table has two columns on `calc.a > 0` and cond: Y,Y sets calc.text = "yes"
// and Y,N sets calc.text = "no".
func runUsingForm(t *testing.T, action, cond string) dtrules.Entity {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_edd.xml"), []byte(usingFormsEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_dt.xml"), []byte(usingFormsTableXML), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table("Using")
	if tbl == nil {
		t.Fatal("table Using not found")
	}
	add := func(el string) {
		t.Helper()
		if err := tbl.AddAction(authoring.Action{DSL: el}); err != nil {
			t.Fatalf("AddAction %q: %v", el, err)
		}
	}
	if err := tbl.AddCondition(authoring.Condition{DSL: "calc.a > 0"}); err != nil {
		t.Fatalf("AddCondition: %v", err)
	}
	if cond == "" {
		add(action)
		if err := tbl.AddColumn(map[int]string{1: "Y"}, []int{1}); err != nil {
			t.Fatalf("AddColumn: %v", err)
		}
	} else {
		add(`set calc.text = "yes"`)
		add(`set calc.text = "no"`)
		if err := tbl.AddCondition(authoring.Condition{DSL: cond}); err != nil {
			t.Fatalf("AddCondition %q: %v", cond, err)
		}
		if err := tbl.AddColumn(map[int]string{1: "Y", 2: "Y"}, []int{1}); err != nil {
			t.Fatalf("AddColumn: %v", err)
		}
		if err := tbl.AddColumn(map[int]string{1: "Y", 2: "N"}, []int{2}); err != nil {
			t.Fatalf("AddColumn: %v", err)
		}
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rs := session.NewRuleSet("usingforms")
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
	opened, err := dtrules.GetRDate(sess, "2026-01-15")
	if err != nil {
		t.Fatalf("GetRDate: %v", err)
	}
	acct.Put(dtrules.GetRName("owner"), dtrules.NewRString("alice"))
	acct.Put(dtrules.GetRName("bal"), dtrules.GetRDoubleValue(2.5))
	acct.Put(dtrules.GetRName("n"), dtrules.GetRIntegerValue(7))
	acct.Put(dtrules.GetRName("opened"), opened)
	calc.Put(dtrules.GetRName("a"), dtrules.GetRIntegerValue(1))
	calc.Put(dtrules.GetRName("acct"), acct.(dtrules.Object))
	state.EntityPush(calc)
	defer state.EntityPop()
	if err := sess.Execute("Using"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return calc
}

func TestUsingForms_ReturnTheValueNotTheEntity(t *testing.T) {
	field := func(t *testing.T, calc dtrules.Entity, name string) dtrules.Object {
		t.Helper()
		v, err := calc.Get(dtrules.GetRName(name))
		if err != nil || v == nil {
			t.Fatalf("get calc.%s: %v", name, err)
		}
		return v
	}
	cases := []struct {
		name, action, cond string
		check              func(t *testing.T, calc dtrules.Entity)
	}{
		{
			name:   "boolUsing action",
			action: `set calc.b = using calc.acct ( owner == "alice" )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if b, err := field(t, calc, "b").BooleanValue(); err != nil || !b {
					t.Errorf("calc.b = %v (err %v), want true", b, err)
				}
			},
		},
		{
			name: "boolUsing condition",
			cond: `using calc.acct ( owner == "alice" )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "text").StringValue(); got != "yes" {
					t.Errorf("calc.text = %q, want \"yes\"", got)
				}
			},
		},
		{
			name: "boolUsing condition false",
			cond: `using calc.acct ( owner == "bob" )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "text").StringValue(); got != "no" {
					t.Errorf("calc.text = %q, want \"no\"", got)
				}
			},
		},
		{
			name:   "nameUsing action",
			action: `set calc.nm = using calc.acct ( name )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "nm").StringValue(); got != "name" {
					t.Errorf("calc.nm = %q, want \"name\"", got)
				}
			},
		},
		{
			name:   "floatUsing action",
			action: `set calc.d = using calc.acct ( bal )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if d, err := field(t, calc, "d").DoubleValue(); err != nil || d != 2.5 {
					t.Errorf("calc.d = %v (err %v), want 2.5", d, err)
				}
			},
		},
		{
			name: "floatUsing condition",
			cond: `using calc.acct ( bal ) > 1.0`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "text").StringValue(); got != "yes" {
					t.Errorf("calc.text = %q, want \"yes\"", got)
				}
			},
		},
		{
			name:   "intUsingArray action",
			action: `set calc.i = using calc.acct ( n * 2 )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if i, err := field(t, calc, "i").LongValue(); err != nil || i != 14 {
					t.Errorf("calc.i = %v (err %v), want 14", i, err)
				}
			},
		},
		{
			name: "intUsingArray condition",
			cond: `using calc.acct ( n + 1 ) > 7`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "text").StringValue(); got != "yes" {
					t.Errorf("calc.text = %q, want \"yes\"", got)
				}
			},
		},
		{
			name:   "bigUsing action",
			action: `set calc.big = using calc.acct ( (bigint) n )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "big").StringValue(); got != "7" {
					t.Errorf("calc.big = %q, want 7", got)
				}
			},
		},
		{
			name:   "dateUsing action",
			action: `set calc.when = using calc.acct ( opened + 1 days )`,
			check: func(t *testing.T, calc dtrules.Entity) {
				d, err := field(t, calc, "when").TimeValue()
				if err != nil || d.Format(time.DateOnly) != "2026-01-16" {
					t.Errorf("calc.when = %v (err %v), want 2026-01-16", d, err)
				}
			},
		},
		{
			name: "dateUsing condition",
			// opened is 2026-01-15, so this holds for any clock after that.
			cond: `using calc.acct ( opened ) < current date`,
			check: func(t *testing.T, calc dtrules.Entity) {
				if got := field(t, calc, "text").StringValue(); got != "yes" {
					t.Errorf("calc.text = %q, want \"yes\"", got)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, runUsingForm(t, tc.action, tc.cond))
		})
	}
}
