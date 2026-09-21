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

package dtrules_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Locals are addressed relative to a frame, and the compiler numbers them per
// table from 0. A performed table must therefore get its own frame: without
// one, its slot 0 is the caller's slot 0, and after `perform` the caller's
// context alias (or `local`) holds whatever the callee last stored there
// (#1226).
//
// Every table below is written through the authoring API, so its postfix comes
// from the production EL compiler with the production per-table numbering, and
// every run starts at RSession.Execute, the entry-point path. Nothing here sets
// a frame by hand.

const frameEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="state" number="100" xls_file="f.xlsx" access="rw">
		<field name="nodes" type="array" subtype="node" access="rw" input="" default_value="" comment=""></field>
	</entity>
	<entity name="node" number="200" xls_file="f.xlsx" access="rw">
		<field name="id" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
	</entity>
	<entity name="result" number="300" xls_file="f.xlsx" access="rw">
		<field name="holder" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="missing" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="finding" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="a1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="a2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="b1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="b2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="c1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="c2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="pairs" type="integer" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="d1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="d1after" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="d2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="d2after" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="d3" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="handled" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="e1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="e2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="e3" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="f1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="g1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="g2" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="h1" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
		<field name="num" type="integer" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
	<entity name="failure" number="400" xls_file="f.xlsx" access="rw">
		<field name="message" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
	</entity>
</entity_data_dictionary>
`

// frameTable is one table: its contexts, initial actions, conditions and
// actions, in order. A table with conditions or actions gets one column, in
// which every condition must be true and every action executes.
type frameTable struct {
	name       string
	contexts   []string
	initial    []string
	conditions []string
	actions    []string
}

// frameTables is the rule set every test below runs against.
var frameTables = []frameTable{
	{name: "Setup", initial: []string{
		`create node as x; set x.id = "n1"; add x to state.nodes`,
		`create node as x; set x.id = "n2"; add x to state.nodes`,
		`create node as x; set x.id = "n3"; add x to state.nodes`,
	}},

	// The issue's repro: the caller holds an alias, the callee iterates the
	// same collection under another.
	{name: "Caller_Alias",
		contexts: []string{`for all state.nodes as n where n.id == "n1"`},
		initial: []string{
			`set result.holder = n.id`,
			`perform Callee_One`,
			`set result.missing = n.id`,
		}},
	{name: "Callee_One",
		contexts: []string{`for all state.nodes as m`},
		initial:  []string{`set result.finding = m.id`}},

	// The same with a `local` in place of the alias.
	{name: "Caller_Local",
		contexts: []string{`local string keep = "kept"`},
		initial: []string{
			`set result.holder = keep`,
			`perform Callee_One`,
			`set result.missing = keep`,
		}},

	// Two locals in the caller, three in the callee.
	{name: "Caller_Two",
		contexts: []string{
			`for all state.nodes as p where p.id == "n1"`,
			`for all state.nodes as q where q.id == "n2"`,
		},
		initial: []string{
			`set result.a1 = p.id`,
			`set result.a2 = q.id`,
			`perform Callee_Three`,
			`set result.b1 = p.id`,
			`set result.b2 = q.id`,
		}},
	{name: "Callee_Three",
		contexts: []string{
			`for all state.nodes as r where r.id == "n3"`,
			`for all state.nodes as s`,
			`local string tag = "callee"`,
		},
		initial: []string{
			// The callee's own locals must be its own, too.
			`set result.c1 = r.id`,
			`set result.c2 = tag`,
			`set result.pairs = result.pairs + 1`,
		}},

	// Three deep: every level holds its alias across the perform below it.
	{name: "Depth_One",
		contexts: []string{`for all state.nodes as outer where outer.id == "n1"`},
		initial: []string{
			`set result.d1 = outer.id`,
			`perform Depth_Two`,
			`set result.d1after = outer.id`,
		}},
	{name: "Depth_Two",
		contexts: []string{`for all state.nodes as middle where middle.id == "n2"`},
		initial: []string{
			`set result.d2 = middle.id`,
			`perform Depth_Three`,
			`set result.d2after = middle.id`,
		}},
	{name: "Depth_Three",
		contexts: []string{`for all state.nodes as inner`},
		initial:  []string{`set result.d3 = inner.id`}},

	// A callee that fails part way through its context, caught by
	// performcatcherror: its frame must be gone before the handler runs, and
	// the caller's slot must be untouched.
	{name: "Caller_Catch",
		contexts: []string{`for all state.nodes as n where n.id == "n1"`},
		initial: []string{
			`set result.holder = n.id`,
			`perform Fails_Midway and on error add failure to context and perform Handle_Failure`,
			`set result.missing = n.id`,
		}},
	{name: "Fails_Midway",
		contexts: []string{`for all state.nodes as m`},
		initial:  []string{`perform No_Such_Table`}},
	{name: "Handle_Failure",
		contexts: []string{`for all state.nodes as h`},
		initial:  []string{`set result.handled = h.id`}},

	// A perform from an action column, in an action that declares its own
	// local: both the action's local and the context's alias must survive.
	{name: "Caller_Action",
		contexts:   []string{`for all state.nodes as n where n.id == "n1"`},
		conditions: []string{`n.id == "n1"`},
		actions: []string{
			`create node as mine; set mine.id = "own"; set result.e1 = mine.id; ` +
				`perform Callee_One; set result.e2 = mine.id; set result.e3 = n.id`,
		}},

	// A callee with no context whose action column declares a local: that
	// local is the callee's slot 0, not the caller's alias.
	{name: "Caller_To_Action_Local",
		contexts: []string{`for all state.nodes as n where n.id == "n2"`},
		initial: []string{
			`set result.g1 = n.id`,
			`perform Callee_Action_Local`,
			`set result.g2 = n.id`,
		}},
	{name: "Callee_Action_Local",
		conditions: []string{`result.g1 == "n2"`},
		actions: []string{
			`create node as tmp; set tmp.id = "callee-tmp"; set result.f1 = tmp.id`,
		}},

	// A condition that reads the context's alias after two performs.
	{name: "Caller_Condition",
		contexts: []string{`for all state.nodes as n where n.id == "n1"`},
		initial: []string{
			`perform Callee_One`,
			`perform Callee_One`,
		},
		conditions: []string{`n.id == "n1"`},
		actions:    []string{`set result.h1 = n.id`}},

	// A table that fails at the entry point, not under a catch.
	{name: "Fails_At_Entry",
		contexts: []string{
			`for all state.nodes as m`,
			`local string tag = "entry"`,
		},
		initial: []string{`perform No_Such_Table`}},
}

// frameSession authors frameTables through the SDK, loads what it wrote, and
// returns a session holding state (with three nodes) and result.
func frameSession(t *testing.T) (*session.RSession, dtrules.Entity) {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "f_edd.xml"), []byte(frameEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	skeleton := `<?xml version="1.0" encoding="UTF-8"?>` + "\n<decision_tables>\n"
	for i, ft := range frameTables {
		skeleton += `<decision_table el_compiled="true"><table_name>` + ft.name +
			`</table_name><xls_file>f.xlsx</xls_file><attribute_fields><Type>ALL</Type><COMMENTS></COMMENTS><TABLE_NUMBER>` +
			strconv.Itoa(100+i) + `</TABLE_NUMBER></attribute_fields>` +
			`<contexts></contexts><initial_actions></initial_actions><conditions></conditions><actions></actions></decision_table>` + "\n"
	}
	skeleton += "</decision_tables>\n"
	if err := os.WriteFile(filepath.Join(xmlDir, "f_dt.xml"), []byte(skeleton), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatalf("open project: %v", err)
	}
	for _, ft := range frameTables {
		tbl := p.Table(ft.name)
		if tbl == nil {
			t.Fatalf("table %s not in the project", ft.name)
		}
		for _, c := range ft.contexts {
			if err := tbl.AddContext(authoring.Context{DSL: c}); err != nil {
				t.Fatalf("%s: context %q: %v", ft.name, c, err)
			}
		}
		for _, a := range ft.initial {
			if err := tbl.AddInitialAction(authoring.InitialAction{DSL: a}); err != nil {
				t.Fatalf("%s: initial action %q: %v", ft.name, a, err)
			}
		}
		if len(ft.conditions)+len(ft.actions) == 0 {
			continue
		}
		cells := map[int]string{}
		for i, c := range ft.conditions {
			if err := tbl.AddCondition(authoring.Condition{DSL: c}); err != nil {
				t.Fatalf("%s: condition %q: %v", ft.name, c, err)
			}
			cells[i+1] = "Y"
		}
		var fire []int
		for i, a := range ft.actions {
			if err := tbl.AddAction(authoring.Action{DSL: a}); err != nil {
				t.Fatalf("%s: action %q: %v", ft.name, a, err)
			}
			fire = append(fire, i+1)
		}
		if err := tbl.AddColumn(cells, fire); err != nil {
			t.Fatalf("%s: column: %v", ft.name, err)
		}
	}
	if err := p.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	rs := session.NewRuleSet("frames")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("load: %v", err)
	}
	sess, err := session.NewSession(rs)
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	for _, name := range []string{"state", "result"} {
		e, err := sess.CreateEntity(dtrules.GetRName(name))
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		state.EntityPush(e)
	}
	if err := sess.Execute("Setup"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	result, err := state.Find(dtrules.GetRName("result"))
	if err != nil {
		t.Fatalf("find result: %v", err)
	}
	re, ok := result.(dtrules.Entity)
	if !ok {
		t.Fatalf("result is %T, not an entity", result)
	}
	return sess, re
}

func field(t *testing.T, e dtrules.Entity, name string) string {
	t.Helper()
	v, err := e.Get(dtrules.GetRName(name))
	if err != nil || v == nil {
		t.Fatalf("result.%s: %v", name, err)
	}
	return v.StringValue()
}

// wantFields checks each result field against its expected value.
func wantFields(t *testing.T, e dtrules.Entity, want map[string]string) {
	t.Helper()
	for k, w := range want {
		if got := field(t, e, k); got != w {
			t.Errorf("result.%s = %q, want %q", k, got, w)
		}
	}
}

// The stack is where the entry point found it once the table returns,
// whichever way it returns.
func wantStacksClean(t *testing.T, sess *session.RSession) {
	t.Helper()
	st, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Fatalf("state is %T", sess.GetState())
	}
	if d := st.CtrlStackDepth(); d != 0 {
		t.Errorf("control stack holds %d value(s) after the entry table returned, want 0", d)
	}
	if f := st.GetCurrentFrame(); f != 0 {
		t.Errorf("current frame is %d after the entry table returned, want 0", f)
	}
}

func TestPerformKeepsTheCallersAlias(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_Alias"); err != nil {
		t.Fatalf("Caller_Alias: %v", err)
	}
	wantFields(t, result, map[string]string{
		"holder":  "n1",
		"missing": "n1", // n after the perform: the callee must not have moved it
		"finding": "n3", // the callee's own loop did run, to the last node
	})
	wantStacksClean(t, sess)
}

func TestPerformKeepsTheCallersLocal(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_Local"); err != nil {
		t.Fatalf("Caller_Local: %v", err)
	}
	wantFields(t, result, map[string]string{"holder": "kept", "missing": "kept", "finding": "n3"})
	wantStacksClean(t, sess)
}

func TestPerformKeepsEveryCallerSlotAndGivesTheCalleeItsOwn(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_Two"); err != nil {
		t.Fatalf("Caller_Two: %v", err)
	}
	wantFields(t, result, map[string]string{
		"a1": "n1", "a2": "n2",
		"b1": "n1", "b2": "n2",
		"c1": "n3", "c2": "callee",
	})
	// One pass per node of the callee's second context.
	if got := field(t, result, "pairs"); got != "3" {
		t.Errorf("result.pairs = %s, want 3", got)
	}
	wantStacksClean(t, sess)
}

func TestNestedPerformThreeDeep(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Depth_One"); err != nil {
		t.Fatalf("Depth_One: %v", err)
	}
	wantFields(t, result, map[string]string{
		"d1": "n1", "d1after": "n1",
		"d2": "n2", "d2after": "n2",
		"d3": "n3",
	})
	wantStacksClean(t, sess)
}

func TestFailedPerformUnderCatchLeavesTheCallersSlot(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_Catch"); err != nil {
		t.Fatalf("Caller_Catch: %v", err)
	}
	wantFields(t, result, map[string]string{
		"holder":  "n1",
		"missing": "n1",
		"handled": "n3",
	})
	wantStacksClean(t, sess)
}

func TestPerformFromAnActionColumnKeepsTheActionsLocal(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_Action"); err != nil {
		t.Fatalf("Caller_Action: %v", err)
	}
	wantFields(t, result, map[string]string{
		"e1": "own", "e2": "own", // the action's own local, across the perform
		"e3":      "n1", // the context's alias, across the perform
		"finding": "n3",
	})
	wantStacksClean(t, sess)
}

func TestContextlessCalleeLocalIsItsOwnSlot(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_To_Action_Local"); err != nil {
		t.Fatalf("Caller_To_Action_Local: %v", err)
	}
	wantFields(t, result, map[string]string{
		"g1": "n2", "g2": "n2",
		"f1": "callee-tmp",
	})
	wantStacksClean(t, sess)
}

func TestConditionReadsTheAliasAfterTwoPerforms(t *testing.T) {
	sess, result := frameSession(t)
	if err := sess.Execute("Caller_Condition"); err != nil {
		t.Fatalf("Caller_Condition: %v", err)
	}
	// Empty means the condition saw a node other than n1.
	wantFields(t, result, map[string]string{"h1": "n1", "finding": "n3"})
	wantStacksClean(t, sess)
}

func TestFailedEntryTableReleasesItsFrame(t *testing.T) {
	sess, _ := frameSession(t)
	if err := sess.Execute("Fails_At_Entry"); err == nil {
		t.Fatal("Fails_At_Entry performs a table that does not exist and must fail")
	}
	wantStacksClean(t, sess)
}
