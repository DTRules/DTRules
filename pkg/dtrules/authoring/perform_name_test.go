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

// Issue #1254 reproduction: `perform $x` compiles to empty postfix. Driven
// through the production chain (authoring SDK → Save → LoadFromDirectory →
// Session.Execute). `perform Target` is the control.

const performNameEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='a' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='ran' type='string' subtype='' access='rw' input='' default_value='' comment=''></field>
	</entity>
</entity_data_dictionary>`

func performNameTableXML(name string) string {
	return `<decision_table el_compiled="true">
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
</decision_table>`
}

func runPerformName(t *testing.T, performEL string) (string, error) {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_edd.xml"), []byte(performNameEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	tables := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
` + performNameTableXML("Caller") + performNameTableXML("Target") + `
</decision_tables>`
	if err := os.WriteFile(filepath.Join(xmlDir, "calc_dt.xml"), []byte(tables), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	author := func(name string, actions ...string) {
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
	author("Target", `set calc.ran = "target"`)
	author("Caller", performEL)
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rs := session.NewRuleSet("performname")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		return "", err
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
	calc.Put(dtrules.GetRName("a"), dtrules.GetRIntegerValue(1))
	state.EntityPush(calc)
	defer state.EntityPop()
	if err := sess.Execute("Caller"); err != nil {
		return "", err
	}
	v, _ := calc.Get(dtrules.GetRName("ran"))
	return v.StringValue(), nil
}

func TestPerformName_RunsTheNamedTable(t *testing.T) {
	for _, el := range []string{`perform Target`, `perform $Target`} {
		t.Run(el, func(t *testing.T) {
			ran, err := runPerformName(t, el)
			if err != nil {
				t.Fatalf("%s: %v", el, err)
			}
			if ran != "target" {
				t.Errorf("%s: calc.ran = %q, want \"target\"", el, ran)
			}
		})
	}
}
