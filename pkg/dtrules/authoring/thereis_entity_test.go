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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Issue #1283: `there is [no] <x> in <entity> where <p>` emitted
// `entitypop swap pop`, which kept the entity and dropped the boolean. Driven
// through the production chain with runUsingForm (using_forms_test.go):
// calc.acct is an entity with owner=alice.
func TestThereIsInEntity_ReturnsTheBoolean(t *testing.T) {
	text := func(t *testing.T, calc dtrules.Entity) string {
		t.Helper()
		v, err := calc.Get(dtrules.GetRName("text"))
		if err != nil || v == nil {
			t.Fatalf("get calc.text: %v", err)
		}
		return v.StringValue()
	}
	for _, tc := range []struct{ cond, want string }{
		// boolThereIsInEntityWhere
		{`there is k in calc.acct where owner == "alice"`, "yes"},
		{`there is k in calc.acct where owner == "bob"`, "no"},
		// boolThereIsNoInEntityWhere
		{`there is no k in calc.acct where owner == "alice"`, "no"},
		{`there is no k in calc.acct where owner == "bob"`, "yes"},
	} {
		t.Run(tc.cond, func(t *testing.T) {
			if got := text(t, runUsingForm(t, "", tc.cond)); got != tc.want {
				t.Errorf("calc.text = %q, want %q", got, tc.want)
			}
		})
	}

	// The same forms as an action's right-hand side.
	for _, tc := range []struct {
		action string
		want   bool
	}{
		{`set calc.b = there is k in calc.acct where owner == "alice"`, true},
		{`set calc.b = there is no k in calc.acct where owner == "bob"`, true},
	} {
		t.Run(tc.action, func(t *testing.T) {
			calc := runUsingForm(t, tc.action, "")
			v, err := calc.Get(dtrules.GetRName("b"))
			if err != nil || v == nil {
				t.Fatalf("get calc.b: %v", err)
			}
			if b, err := v.BooleanValue(); err != nil || b != tc.want {
				t.Errorf("calc.b = %v (err %v), want %v", b, err, tc.want)
			}
		})
	}
}
