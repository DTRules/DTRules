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

func dlSyms() map[string]string {
	return map[string]string{"fx": TypeFixed, "bi": TypeBigInt, "db": TypeDouble, "iv": TypeInteger}
}

// TestDoubleLiteralMixRejected (#894): a fixed/bigint field mixed with a double
// LITERAL bypassed the #876 reject because the field matched the TypedDouble
// grammar branch, so VisitFloatAddFloat hardcoded both operands to double. It
// now resolves each operand's declared type, so the exact/double mix is caught.
func TestDoubleLiteralMixRejected(t *testing.T) {
	bad := []string{
		"set fx = fx + 1.5",
		"set fx = fx - 1.5",
		"set fx = 1.5 + fx", // order independent
		"set bi = bi + 1.5",
		"set bi = bi - 1.5",
	}
	for _, src := range bad {
		c := NewCompiler()
		c.SetSymbols(dlSyms())
		pf, err := c.CompileAction(src)
		if err == nil {
			t.Errorf("%q: expected reject, got postfix %q", src, pf)
			continue
		}
		if !strings.Contains(err.Error(), "cannot combine double with") {
			t.Errorf("%q: error should explain the mix, got: %v", src, err)
		}
	}

	// Must NOT over-reject: double+literal, fixed+integer, double+integer.
	for _, src := range []string{
		"set db = db + 1.5",
		"set db = 1.5 + 2.5",
		"set fx = fx + iv",
		"set db = db + iv",
	} {
		c := NewCompiler()
		c.SetSymbols(dlSyms())
		if _, err := c.CompileAction(src); err != nil {
			t.Errorf("%q: expected clean compile, got: %v", src, err)
		}
	}
}

// TestDoubleNegateOp (#894): `- <double field>` routed through VisitIntNegate's
// default and emitted integer `negate` (truncates); it now emits `fnegate`.
func TestDoubleNegateOp(t *testing.T) {
	cases := map[string]string{
		"set db = - db": "fnegate",  // double
		"set fx = - fx": "fpnegate", // fixed
		"set bi = - bi": "bnegate",  // bigint
		"set iv = - iv": "negate",   // integer
	}
	for src, want := range cases {
		c := NewCompiler()
		c.SetSymbols(dlSyms())
		pf, err := c.CompileAction(src)
		if err != nil {
			t.Fatalf("%q compile: %v", src, err)
		}
		if !strings.Contains(pf, " "+want+" ") && !strings.Contains(pf, want+" ") {
			t.Errorf("%q: expected %q in postfix, got %q", src, want, pf)
		}
	}
}
