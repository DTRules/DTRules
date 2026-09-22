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

// Issue #1258: subDestColon stored number - field, with the integer `-` for
// every field type. The runtime behaviour is pinned in
// pkg/dtrules/authoring/subtract_from_colon_test.go; this pins the postfix,
// including the fixed-point form the runtime test's EDD does not declare.
func TestSubtractFromColon_Postfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"account":   TypeEntity,
		"account.n": TypeInteger,
		"account.d": TypeDouble,
		"account.f": TypeFixed,
	})
	for _, tc := range []struct{ el, want string }{
		{`subtract 1 from :account: n`, `1 account entitypush n swap - /n xdef entitypop`},
		{`subtract 1 from account's n`, `1 account entitypush n swap - /n xdef entitypop`},
		{`subtract 1.5 from account's d`, `1.5 account entitypush cvd d swap f- /d xdef entitypop`},
		{`subtract 1 from :account: f`, `1 account entitypush cvfp f swap fp- /f xdef entitypop`},
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
