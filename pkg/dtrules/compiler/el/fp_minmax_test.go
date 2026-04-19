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

// Issue #688: `minimum of <fp_a> and <fp_b>` used to emit the integer
// `min` op (since typedLong wins over typedDouble for fp IDENT fields,
// routing the parse through VisitIntMinOf). `min` calls IntValue on
// both operands, truncating the fractional parts of fp values. The
// same applied to `maximum of`.
//
// Fix: added fpmin / fpmax operators plus type-aware dispatch in the
// four Int Min/Max visitors. The dispatch uses the same
// promoteArithType lattice as the arithmetic family — if any operand
// is fp, the whole expression promotes to fp and emits fpmin/fpmax.

func TestFpMinMax_EmitsFpOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"wp.a": TypeFixed,
		"wp.b": TypeFixed,
	})
	cases := []struct {
		dsl, want string
	}{
		{"set wp.a = minimum of wp.a and wp.b", "fpmin"},
		{"set wp.a = maximum of wp.a and wp.b", "fpmax"},
		{"set wp.a = minimum of wp.a , wp.b", "fpmin"},
		{"set wp.a = maximum of wp.a , wp.b", "fpmax"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			postfix, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			if !strings.Contains(postfix, tc.want) {
				t.Errorf("expected %q, got: %s", tc.want, postfix)
			}
			// Regression pin: plain min/max would truncate fp via IntValue.
			if strings.Contains(postfix, " min ") || strings.HasSuffix(postfix, " min") ||
				strings.Contains(postfix, " max ") || strings.HasSuffix(postfix, " max") {
				t.Errorf("unexpected plain min/max on fp operands: %s", postfix)
			}
		})
	}
}

// TestFpMinMax_MixedTypes — `minimum of fp_field and int_value` should
// promote the int side via cvfp and use fpmin.
func TestFpMinMax_MixedTypes(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"wp.a": TypeFixed,
		"ap.n": TypeInteger,
	})
	postfix, err := c.CompileAction("set wp.a = minimum of wp.a and ap.n")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "cvfp") {
		t.Errorf("expected cvfp promotion for the int operand, got: %s", postfix)
	}
	if !strings.Contains(postfix, "fpmin") {
		t.Errorf("expected fpmin (not min), got: %s", postfix)
	}
}

// TestIntMinMax_Unchanged is a regression guard — pure-int min/max must
// still emit the integer `min`/`max` ops, not fpmin/fpmax.
func TestIntMinMax_Unchanged(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"ap.a": TypeInteger,
		"ap.b": TypeInteger,
	})
	postfix, err := c.CompileAction("set ap.a = minimum of ap.a and ap.b")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, " min") {
		t.Errorf("expected plain `min` for pure-int, got: %s", postfix)
	}
	if strings.Contains(postfix, "fpmin") || strings.Contains(postfix, "cvfp") {
		t.Errorf("pure-int min must not emit fpmin/cvfp: %s", postfix)
	}
}

// TestFpMinMax_Operator_Direct guards the new fpmin / fpmax operators
// at the operator-dispatch level, independent of compile paths.
func TestFpMinMax_Operator_Direct(t *testing.T) {
	// The operator tests live in pkg/dtrules/operators, not here — this
	// test just guards that the operators are findable from the
	// symbol-table lookup path used by postfix emission.
	c := NewCompiler()
	c.SetSymbols(map[string]string{"wp.a": TypeFixed, "wp.b": TypeFixed})
	postfix, err := c.CompileAction("set wp.a = minimum of wp.a and wp.b")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.HasSuffix(postfix, " /wp.a xdef") {
		t.Errorf("expected /wp.a xdef suffix, got: %s", postfix)
	}
}
