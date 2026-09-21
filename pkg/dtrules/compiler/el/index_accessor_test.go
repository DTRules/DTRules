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

// TestIndexAccessorFollowsDeclaredType (#1229): `<base>[<i>]` compiles to
// bytesidx only when base is bytes — declared bytes, or an expression that
// can only be bytes (a hex literal, a hash) — and to getat otherwise. An
// undeclared name gets getat whichever grammar rule parsed the index: the
// cast form (indxExpr) and the bare integer form (intBytesIndex) agree.
func TestIndexAccessorFollowsDeclaredType(t *testing.T) {
	cases := []struct {
		name    string
		symbols map[string]string
		action  string
		want    string
	}{
		{"no symbols, cast", nil, `set n = (long) x[0]`, "x 0 getat"},
		{"no symbols, bare", nil, `set n = x[0]`, "x 0 getat"},
		{"array, cast", map[string]string{"x": "array", "n": "integer"}, `set n = (long) x[0]`, "x 0 getat"},
		{"array, bare", map[string]string{"x": "array", "n": "integer"}, `set n = x[0]`, "x 0 getat"},
		{"bytes, cast", map[string]string{"x": "bytes", "n": "integer"}, `set n = (long) x[0]`, "x 0 bytesidx"},
		{"bytes, bare", map[string]string{"x": "bytes", "n": "integer"}, `set n = x[0]`, "x 0 bytesidx"},
		{"hex literal, no symbols", nil, `set n = 0xdeadbeef[1]`, "bytesidx"},
		{"hash, no symbols", nil, `set n = (sha256 of 0xdeadbeef)[1]`, "bytesidx"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCompiler()
			if tc.symbols != nil {
				c.SetSymbols(tc.symbols)
			}
			got, err := c.CompileAction(tc.action)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.action, err)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("%q compiled to %q, want it to contain %q", tc.action, got, tc.want)
			}
		})
	}
}
