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

// Issue #1241: integer division compiles to a bare `/`, and the postfix
// reader took a lone `/` as a literal name with an empty body, so `6 / 2`
// left 6 and an empty name on the stack and `1 / 0` raised nothing.
//
// These tests drive the production chain: the authoring SDK compiles the EL
// and writes the postfix to the project's XML on Save, the strict loader
// reads that XML into a rule set, and the session executes the decision
// table. None of those steps is done by hand here.

const intDivEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='a' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='b' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='q' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='neg' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='fq' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='d' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='mixed' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='text' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
</entity_data_dictionary>`

// intDivTableXML declares one empty table; the actions are added through the
// authoring SDK so their postfix is whatever the EL compiler emits.
func intDivTableXML(name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table el_compiled="true">
<table_name>` + name + `</table_name>
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
}

// buildIntDivProject authors table `name` with the given EL actions (all
// firing in one column, guarded by calc.a > 0), saves it, and returns the
// project's xml directory.
func buildIntDivProject(t *testing.T, name string, actions []string) string {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_edd.xml"), []byte(intDivEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_dt.xml"), []byte(intDivTableXML(name)), 0o644); err != nil {
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
	nums := make([]int, 0, len(actions))
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
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return xmlDir
}

// runIntDivTable loads the saved project with the production loader, puts a
// calc entity (a=7, b=0) on the entity stack, and executes the table.
func runIntDivTable(t *testing.T, xmlDir, name string) (dtrules.Entity, error) {
	t.Helper()
	rs := session.NewRuleSet("intdiv")
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
		t.Fatalf("CreateEntity: %v", err)
	}
	calc.Put(dtrules.GetRName("a"), dtrules.GetRIntegerValue(7))
	calc.Put(dtrules.GetRName("b"), dtrules.GetRIntegerValue(0))

	state.EntityPush(calc)
	defer state.EntityPop()
	return calc, sess.Execute(name)
}

func TestIntegerDivision_CompiledELRunsInSession(t *testing.T) {
	xmlDir := buildIntDivProject(t, "Divide", []string{
		// The issue's reproducer: the string shows whether `/` divided.
		`set calc.text = "div:" + string value of (6 / 2) + " mul:" + string value of (6 * 2)`,
		`set calc.q = 6 / 2`,
		// Integer division that does not come out even truncates toward zero.
		`set calc.fq = calc.a / 2`,
		`set calc.neg = -7 / 2`,
		// Integer division assigned to a double is still integer division.
		`set calc.d = 7 / 2`,
		// A double operand promotes: this is float division (fdiv).
		`set calc.mixed = calc.a / 2.0`,
	})

	calc, err := runIntDivTable(t, xmlDir, "Divide")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	get := func(field string) dtrules.Object {
		t.Helper()
		v, err := calc.Get(dtrules.GetRName(field))
		if err != nil || v == nil {
			t.Fatalf("get %s: %v", field, err)
		}
		return v
	}
	if got := get("text").StringValue(); got != "div:3 mul:12" {
		t.Errorf(`text = %q, want "div:3 mul:12"`, got)
	}
	for field, want := range map[string]int64{"q": 3, "fq": 3, "neg": -3} {
		got, err := get(field).LongValue()
		if err != nil || got != want {
			t.Errorf("%s = %d (err %v), want %d", field, got, err, want)
		}
	}
	for field, want := range map[string]float64{"d": 3, "mixed": 3.5} {
		got, err := get(field).DoubleValue()
		if err != nil || got != want {
			t.Errorf("%s = %v (err %v), want %v", field, got, err, want)
		}
	}
}

func TestIntegerDivision_ByZeroIsARuntimeError(t *testing.T) {
	for _, el := range []string{
		`set calc.q = 1 / 0`,
		`set calc.q = calc.a / calc.b`,
	} {
		t.Run(el, func(t *testing.T) {
			xmlDir := buildIntDivProject(t, "DivideByZero", []string{el})
			calc, err := runIntDivTable(t, xmlDir, "DivideByZero")
			if err == nil {
				v, _ := calc.Get(dtrules.GetRName("q"))
				t.Fatalf("Execute succeeded (q = %v); want a Division by zero error", v)
			}
			if !strings.Contains(err.Error(), "Division by zero") {
				t.Errorf("error = %v; want it to say Division by zero", err)
			}
		})
	}
}
