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

// Issue #1251: `<s> + <x>` with a literal number, a date, or an entity or
// array expression on the right parses as strConcatInt / Float / Date /
// Entity / Array, which emitted nothing. The runtime behaviour is pinned in
// pkg/dtrules/authoring/strconcat_nonstring_test.go; this pins the postfix.
func TestStrConcatNonString_Postfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"accounts": "array", "account": "entity"})
	for _, tc := range []struct{ el, want string }{
		{`"x" + 1 == "x1"`, `"x" 1 strconcat "x1" streq`},
		{`"x" + 1.5 == "x"`, `"x" 1.5 strconcat "x" streq`},
		{`"x" + current date == "x"`, `"x" today strconcat "x" streq`},
		{`"x" + first of accounts == "x"`, `"x" accounts first cve strconcat "x" streq`},
		{`"x" + copy of accounts == "x"`, `"x" accounts copy strconcat "x" streq`},
	} {
		got, err := c.CompileCondition(tc.el)
		if err != nil {
			t.Errorf("%s: %v", tc.el, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s\n got  %q\n want %q", tc.el, got, tc.want)
		}
	}
}
