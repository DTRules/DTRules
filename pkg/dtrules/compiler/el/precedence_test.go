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

import (
	"strings"
	"testing"
)

// EL arithmetic groups the way C, C++ and Go do, which is what anyone reading
// a rule will assume: multiplicative binds tighter than additive, and
// operators of equal precedence associate left.
//
// Each ANTLR alternative is its own precedence level, so an operator on its
// own line gets its own level. That gave four levels where arithmetic has two:
// `*` outranked `/` and `+` outranked `-`, so `a - b + c` meant `a - (b + c)`
// and `a / b * c` meant `a / (b * c)`. The second surfaced as a division by
// zero in TaxReturn's CTC phase-out; the first is worse, because it just
// returns a wrong number — off by twice the third term (#1146).
func TestArithmeticGroupsLikeC(t *testing.T) {
	cases := []struct{ src, want, why string }{
		{"set r.x = r.a - r.b + r.c", "r.a r.b - r.c +", "(a-b)+c — equal precedence, left"},
		{"set r.x = r.a + r.b - r.c", "r.a r.b + r.c -", "(a+b)-c"},
		{"set r.x = r.a / r.b * r.c", "r.a r.b / r.c *", "(a/b)*c"},
		{"set r.x = r.a * r.b / r.c", "r.a r.b * r.c /", "(a*b)/c"},
		{"set r.x = r.a - r.b - r.c", "r.a r.b - r.c -", "(a-b)-c — repeated operator"},
		{"set r.x = r.a / r.b / r.c", "r.a r.b / r.c /", "(a/b)/c"},
		{"set r.x = r.a + r.b * r.c", "r.a r.b r.c * +", "a+(b*c) — multiplicative binds tighter"},
		{"set r.x = r.a * r.b + r.c", "r.a r.b * r.c +", "(a*b)+c"},
		{"set r.x = r.a - r.b * r.c", "r.a r.b r.c * -", "a-(b*c)"},
		{"set r.x = (r.a - r.b) * r.c", "r.a r.b - r.c *", "parentheses still win"},
	}
	syms := map[string]string{"r.a": "integer", "r.b": "integer", "r.c": "integer", "r.x": "integer"}
	for _, tc := range cases {
		c := NewCompiler()
		c.SetSymbols(syms)
		got, err := c.CompileAction(tc.src)
		if err != nil {
			t.Errorf("%s: %v", tc.src, err)
			continue
		}
		if !strings.HasPrefix(got, tc.want) {
			t.Errorf("%s\n  got  %s\n  want %s...  (%s)", tc.src, got, tc.want, tc.why)
		}
	}
}

// The same grouping has to hold for doubles, which take a different set of
// grammar alternatives and a different opcode family.
func TestArithmeticGroupsLikeCForDoubles(t *testing.T) {
	cases := []struct{ src, want string }{
		{"set r.x = r.a - r.b + r.c", "r.a r.b f- r.c f+"},
		{"set r.x = r.a / r.b * r.c", "r.a r.b fdiv r.c fmul"},
		{"set r.x = r.a + r.b * r.c", "r.a r.b r.c fmul f+"},
	}
	syms := map[string]string{"r.a": "double", "r.b": "double", "r.c": "double", "r.x": "double"}
	for _, tc := range cases {
		c := NewCompiler()
		c.SetSymbols(syms)
		got, err := c.CompileAction(tc.src)
		if err != nil {
			t.Errorf("%s: %v", tc.src, err)
			continue
		}
		if !strings.HasPrefix(got, tc.want) {
			t.Errorf("%s\n  got  %s\n  want %s...", tc.src, got, tc.want)
		}
	}
}

// Ordering between classes was already right and must stay right.
func TestClassOrderingUnchanged(t *testing.T) {
	syms := map[string]string{"r.a": "integer", "r.b": "integer", "r.c": "integer", "r.d": "integer"}
	c := NewCompiler()
	c.SetSymbols(syms)
	got, err := c.CompileCondition("r.a + r.b > r.c")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "r.a r.b + r.c >") {
		t.Errorf("arithmetic must bind tighter than comparison: %s", got)
	}
}
