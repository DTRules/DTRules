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

// Issue #1287: the colon/possessive add and subtract statements drop the
// entity entitypop leaves on the data stack. The runtime behaviour is pinned
// in pkg/dtrules/authoring/colon_dest_stack_test.go.
func TestColonDestStack_Postfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"account":   TypeEntity,
		"account.n": TypeInteger,
		"account.d": TypeDouble,
		"n":         TypeInteger,
		"d":         TypeDouble,
	})
	for _, tc := range []struct{ el, want string }{
		{`add 1 to :account: n`, `1 account entitypush n + /n xdef entitypop pop`},
		{`add 1 to account's n`, `1 account entitypush n + /n xdef entitypop pop`},
		{`add 1.5 to account's d`, `1.5 account entitypush d f+ /d xdef entitypop pop`},
		{`subtract 1 from :account: n`, `1 account entitypush n swap - /n xdef entitypop pop`},
	} {
		got, err := c.CompileAction(tc.el)
		if err != nil {
			t.Errorf("%s: %v", tc.el, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s\n got  %q\n want %q", tc.el, got, tc.want)
		}
	}
}
