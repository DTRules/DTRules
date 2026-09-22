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

// Issue #1253: `add to <x> <n>` / `subtract from <x> <n>` in an expression
// emitted nothing. The runtime behaviour is pinned in
// pkg/dtrules/authoring/add_to_expression_test.go; this pins the postfix,
// including the fixed and bigint forms.
func TestAddToExpression_Postfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"n": TypeInteger,
		"d": TypeDouble,
		"f": TypeFixed,
		"b": TypeBigInt,
	})
	for _, tc := range []struct{ el, want string }{
		{`add to n 5 > 0`, `n 5 + 0 >`},
		{`subtract from n 5 > 0`, `n 5 - 0 >`},
		{`add to d 5 > 0`, `d 5 f+ 0 >`},
		{`subtract from d 5 > 0`, `d 5 f- 0 >`},
		{`add to f 5 > 0`, `f 5 cvfp fp+ 0 >`},
		{`subtract from b 5 > 0`, `b 5 cvbi b- 0 >`},
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
