// Copyright 2024 Paul Snow
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

// TestFirstPassPredicate covers the EL `first pass` boolean (#764).
// It lowers to the single postfix op `firstpass`, which queries the
// runtime's innermost-loop iteration counter.
func TestFirstPassPredicate(t *testing.T) {
	cases := []struct {
		name string
		el   string
		want string
	}{
		{
			name: "bare",
			el:   "first pass",
			want: "firstpass",
		},
		{
			name: "negated",
			el:   "not first pass",
			want: "firstpass not",
		},
	}
	c := NewCompiler()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.CompileCondition(tt.el)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.el, err)
			}
			got = strings.TrimSpace(got)
			want := strings.TrimSpace(tt.want)
			if got != want {
				t.Errorf("CompileCondition(%q)\n  got:  %q\n  want: %q", tt.el, got, want)
			}
		})
	}
}

// TestFirstPassPredicate_InCompoundBoolean confirms `first pass` is
// usable inside a compound boolean — the actual postfix lowering goes
// through EL's short-circuit and/or rewriter (`over if` form), so we
// only assert the `firstpass` token appears, not the exact shape.
func TestFirstPassPredicate_InCompoundBoolean(t *testing.T) {
	c := NewCompiler()
	dsl := `first pass and taxpayer.filing_status is equal to "MFJ"`
	got, err := c.CompileCondition(dsl)
	if err != nil {
		t.Fatalf("CompileCondition(%q) error: %v", dsl, err)
	}
	if !strings.Contains(got, "firstpass") {
		t.Errorf("expected `firstpass` token in postfix, got %q", got)
	}
}
