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

// Issue #1287: `add N to :e: field` / `subtract N from :e: field` ended with a
// bare entitypop, which here leaves the entity on the data stack. Inside
// `{ body } for all <array> where <b>` the loop keeps its body on top of the
// stack and runs it with `dup execute`, so after the first element the
// leftover entity was duplicated and executed instead of the body, and the
// remaining elements were skipped.
//
// The harness in el_runtime_harness_test.go has no array field, so this test
// builds its own project, through the same chain: authoring SDK, Save, the
// strict loader, a session.
const colonDestEDD = `<entity_data_dictionary version='2'>
	<entity name='calc' access='rw' comment=''>
		<field name='calc' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='accounts' type='array' subtype='account' access='rw' input='' default_value='' comment=''></field>
		<field name='go' type='boolean' subtype='' access='rw' input='' default_value='true' comment=''></field>
	</entity>
	<entity name='account' access='rw' comment=''>
		<field name='account' type='entity' subtype='' access='r' input='' default_value='' comment=''></field>
		<field name='n' type='integer' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='d' type='double' subtype='' access='rw' input='' default_value='0' comment=''></field>
		<field name='flag' type='boolean' subtype='' access='rw' input='' default_value='true' comment=''></field>
	</entity>
</entity_data_dictionary>`

func runColonDest(t *testing.T, action string, accounts int) []dtrules.Entity {
	t.Helper()
	const name = "ColonDest"
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "cd_edd.xml"), []byte(colonDestEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "cd_dt.xml"), []byte(elrtTableXML(name)), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	tbl := p.Table(name)
	if err := tbl.AddCondition(authoring.Condition{DSL: `calc.go`}); err != nil {
		t.Fatalf("AddCondition: %v", err)
	}
	if err := tbl.AddAction(authoring.Action{DSL: action}); err != nil {
		t.Fatalf("AddAction %q: %v", action, err)
	}
	if err := tbl.AddColumn(map[int]string{1: "Y"}, []int{1}); err != nil {
		t.Fatalf("AddColumn: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rs := session.NewRuleSet("colondest")
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
	var accts []dtrules.Entity
	arr, err := dtrules.NewArray(sess, false, false)
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	for i := 0; i < accounts; i++ {
		a, err := ef.CreateEntity(sess, dtrules.GetRName("account"))
		if err != nil {
			t.Fatalf("CreateEntity account: %v", err)
		}
		arr.Add(a)
		accts = append(accts, a)
	}
	calc.Put(dtrules.GetRName("accounts"), arr)
	state.EntityPush(calc)
	defer state.EntityPop()
	if err := sess.Execute(name); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return accts
}

func TestColonDest_LoopRunsForEveryElement(t *testing.T) {
	for _, tc := range []struct {
		el    string
		field string
		want  string
	}{
		{`{ add 1 to :account: n; } for all calc.accounts where flag`, "n", "1"},
		{`{ add 1.5 to account's d; } for all calc.accounts where flag`, "d", "1.5"},
		{`{ subtract 1 from :account: n; } for all calc.accounts where flag`, "n", "-1"},
		{`{ subtract 2 from account's n; } for all calc.accounts where flag`, "n", "-2"},
	} {
		t.Run(tc.el, func(t *testing.T) {
			for i, a := range runColonDest(t, tc.el, 3) {
				v, err := a.Get(dtrules.GetRName(tc.field))
				if err != nil || v == nil {
					t.Fatalf("account %d: get %s: %v", i, tc.field, err)
				}
				if got := v.StringValue(); got != tc.want {
					t.Errorf("account %d: %s = %s, want %s", i, tc.field, got, tc.want)
				}
			}
		})
	}
}
