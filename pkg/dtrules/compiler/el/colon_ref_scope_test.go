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

package el

import "testing"

// Issue #1257: colon and possessive references pushed the entity onto the
// data stack and never made it current. The runtime behaviour is pinned in
// pkg/dtrules/authoring/colon_ref_scope_test.go; this pins the postfix,
// including the chain forms, whose entity stack must come back balanced.
func TestColonRefScope_Postfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"calc":      TypeEntity,
		"account":   TypeEntity,
		"accounts":  TypeArray,
		"flag":      TypeBoolean,
		"n":         TypeInteger,
		"account.n": TypeInteger,
	})
	for _, tc := range []struct {
		kind, el, want string
	}{
		{"cond", `:account: flag`, `account entitypush flag entitypop pop`},
		{"cond", `account's flag`, `account entitypush flag entitypop pop`},
		{"cond", `:account: flag is true`, `account entitypush flag entitypop pop true beq`},
		{"cond", `:account: n == 7`, `account entitypush n entitypop pop 7 ==`},
		{"cond", `:account: account == first of accounts`,
			`account entitypush account entitypop pop accounts first cve req`},
		{"cond", `calc's account's n == 7`,
			`calc entitypush account entitypop pop entitypush n entitypop pop 7 ==`},
		{"cond", `:calc: account's n == 7`,
			`calc entitypush account entitypop pop entitypush n entitypop pop 7 ==`},
		{"action", `set :account: n = 1`, `1 cvi account entitypush /n xdef entitypop pop`},
		{"action", `set account's flag = true`, `true cvb account entitypush /flag xdef entitypop pop`},
	} {
		compile := c.CompileCondition
		if tc.kind == "action" {
			compile = c.CompileAction
		}
		got, err := compile(tc.el)
		if err != nil {
			t.Errorf("%s: %v", tc.el, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s\n got  %q\n want %q", tc.el, got, tc.want)
		}
	}
}
