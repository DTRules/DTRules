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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestIssue904Execution runs the in-action entity-building constructs end to
// end (#904, found implementing the staking recipient-aggregation tables).
// All four defects compiled without error and failed at runtime, so token
// tests alone cannot guard them:
//
//  1. `local entity r = new T entity` in an action underflowed at `execute`
//     (statements using the local were emitted AFTER `deallocate`).
//  2. `create T as nr` bound via xdef, which refuses undeclared names, and
//     typed sets on the alias degraded to cvi.
//  3. `add new T entity to coll` double-emitted `swap addto`.
//  4. `lowercase of s` parsed as relationship traversal and errored on
//     string operands.
func TestIssue904Execution(t *testing.T) {
	rs := session.NewRuleSet("i904")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	// token_recipient { url: string; amount: fixed }
	trName := dtrules.GetRName("token_recipient")
	trRef, err := ef.FindCreateRefEntity(true, trName)
	if err != nil {
		t.Fatal(err)
	}
	trRef.AddAttribute(dtrules.GetRName("url"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	trRef.AddAttribute(dtrules.GetRName("amount"), "", nil, true, true, dtrules.TypeFixed, "", "", "", "")

	// calculation_context { recipients: array; recip_url: string;
	//                       recip_amount: fixed; flag: boolean; s: string }
	ccName := dtrules.GetRName("calculation_context")
	ccRef, err := ef.FindCreateRefEntity(true, ccName)
	if err != nil {
		t.Fatal(err)
	}
	ccRef.AddAttribute(dtrules.GetRName("recipients"), "", nil, true, true, dtrules.TypeArray, "token_recipient", "", "", "")
	ccRef.AddAttribute(dtrules.GetRName("recip_url"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	ccRef.AddAttribute(dtrules.GetRName("recip_amount"), "", nil, true, true, dtrules.TypeFixed, "", "", "", "")
	ccRef.AddAttribute(dtrules.GetRName("flag"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")
	ccRef.AddAttribute(dtrules.GetRName("s"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")

	root, err := ef.CreateEntity(sess, ccName)
	if err != nil {
		t.Fatal(err)
	}
	arr, err := dtrules.NewArrayWithElements(sess, true, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("recipients"), arr)
	root.Put(dtrules.GetRName("recip_url"), dtrules.NewRString("acc://Staker.acme/Tokens"))
	amt, err := dtrules.GetRFixedFromInt64(42)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("recip_amount"), amt)

	state := sess.GetState().(*interpreter.DTState)
	symbols := map[string]string{
		"recip_url": "string", "recip_amount": "fixed",
		"recipients": "array", "flag": "boolean", "s": "string",
		"token_recipient.url":    "string",
		"token_recipient.amount": "fixed",
	}

	exec := func(action string) {
		t.Helper()
		// Fresh compiler per action: local slot indices are per-table
		// state (#814) and each action here models one table's cell.
		elc := el.NewCompiler()
		elc.SetSymbols(symbols)
		pf, err := elc.CompileAction(action)
		if err != nil {
			t.Fatalf("%q compile: %v", action, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", action, pf, err)
		}
		state.EntityPush(root)
		err = obj.Execute(state)
		state.EntityPop()
		if err != nil {
			t.Fatalf("%q execute %q: %v", action, pf, err)
		}
	}

	recipients := func() []*dtrules.RArray {
		v, err := root.Get(dtrules.GetRName("recipients"))
		if err != nil {
			t.Fatal(err)
		}
		a, err := v.RArrayValue()
		if err != nil {
			t.Fatal(err)
		}
		return []*dtrules.RArray{a}
	}
	checkElem := func(idx int, wantURL string, wantAmountMantissa int64) {
		t.Helper()
		a := recipients()[0]
		if a.Size() <= idx {
			t.Fatalf("recipients has %d elems, want at least %d", a.Size(), idx+1)
		}
		obj := a.GetIterator()[idx]
		ent, err := obj.REntityValue()
		if err != nil {
			t.Fatalf("elem %d not an entity: %v", idx, err)
		}
		u, _ := ent.Get(dtrules.GetRName("url"))
		if u.StringValue() != wantURL {
			t.Errorf("elem %d url = %q, want %q", idx, u.StringValue(), wantURL)
		}
		am, _ := ent.Get(dtrules.GetRName("amount"))
		fp, ok := am.(*dtrules.RFixed)
		if !ok {
			t.Fatalf("elem %d amount is %T, want *RFixed (cvi mis-dispatch?)", idx, am)
		}
		if fp.Mantissa().Int64() != wantAmountMantissa {
			t.Errorf("elem %d amount mantissa = %d, want %d", idx, fp.Mantissa().Int64(), wantAmountMantissa)
		}
	}

	// Defect 1: local entity declaration scoped to the rest of the action.
	exec("local entity r = new token_recipient entity; " +
		"set r.url = recip_url; set r.amount = recip_amount; " +
		"add r to recipients")
	if got := recipients()[0].Size(); got != 1 {
		t.Fatalf("after local-entity build: recipients len = %d, want 1", got)
	}
	checkElem(0, "acc://Staker.acme/Tokens", 42_0000_0000)

	// Defect 2: create-as with an UNDECLARED alias uses the same local
	// machinery — the xdef lowering refused undeclared names at runtime and
	// typed sets degraded to cvi.
	exec("create token_recipient as nr; " +
		"set nr.url = recip_url; set nr.amount = recip_amount; " +
		"add nr to recipients")
	if got := recipients()[0].Size(); got != 2 {
		t.Fatalf("after create-as build: recipients len = %d, want 2", got)
	}
	checkElem(1, "acc://Staker.acme/Tokens", 42_0000_0000)

	// Defect 3: add-new used to double-emit `swap addto` (stack corruption).
	exec("add new token_recipient entity to recipients")
	if got := recipients()[0].Size(); got != 3 {
		t.Fatalf("after add-new: recipients len = %d, want 3", got)
	}

	// Defect 4: the case-fold surface — both the new `lowercase of` form and
	// a case-insensitive comparison, the staking dedup shape.
	exec(`set s = lowercase of "ACC://Staker.ACME/Tokens"`)
	if v, _ := root.Get(dtrules.GetRName("s")); v.StringValue() != "acc://staker.acme/tokens" {
		t.Errorf("lowercase of: got %q", v.StringValue())
	}
	exec(`set s = uppercase of "acc://x"`)
	if v, _ := root.Get(dtrules.GetRName("s")); v.StringValue() != "ACC://X" {
		t.Errorf("uppercase of: got %q", v.StringValue())
	}
	// The staking dedup shape: case-insensitive URL comparison as a
	// condition.
	{
		elc := el.NewCompiler()
		elc.SetSymbols(symbols)
		pf, err := elc.CompileCondition(`lowercase of recip_url == lowercase of "ACC://STAKER.acme/tokens"`)
		if err != nil {
			t.Fatalf("dedup condition compile: %v", err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("dedup condition assemble %q: %v", pf, err)
		}
		state.EntityPush(root)
		err = obj.Execute(state)
		state.EntityPop()
		if err != nil {
			t.Fatalf("dedup condition execute %q: %v", pf, err)
		}
		res, err := state.DataPop()
		if err != nil {
			t.Fatalf("dedup condition left no result: %v", err)
		}
		b, err := res.BooleanValue()
		if err != nil || !b {
			t.Errorf("case-insensitive URL compare = %v (err=%v), want true (postfix %q)", res, err, pf)
		}
	}

	// A declaration as the LAST statement wraps an empty block — must still
	// assemble and execute cleanly.
	exec("local entity unused = new token_recipient entity")
	// Two locals in one action: nested scopes, sequential slots.
	exec("local entity ra = new token_recipient entity; " +
		"local entity rb = new token_recipient entity; " +
		"set ra.url = \"first\"; set rb.url = \"second\"; " +
		"add ra to recipients; add rb to recipients")
	a := recipients()[0]
	if a.Size() != 5 {
		t.Fatalf("after two-local build: recipients len = %d, want 5", a.Size())
	}
	e3, _ := a.GetIterator()[3].REntityValue()
	e4, _ := a.GetIterator()[4].REntityValue()
	u3, _ := e3.Get(dtrules.GetRName("url"))
	u4, _ := e4.Get(dtrules.GetRName("url"))
	if u3.StringValue() != "first" || u4.StringValue() != "second" {
		t.Errorf("two-local build: urls = %q, %q; want \"first\", \"second\"", u3.StringValue(), u4.StringValue())
	}
}
