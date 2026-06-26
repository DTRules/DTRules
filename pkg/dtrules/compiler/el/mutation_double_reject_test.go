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

func mutSyms() map[string]string {
	return map[string]string{
		"fx": TypeFixed, "fx2": TypeFixed,
		"bi": TypeBigInt,
		"db": TypeDouble, "db2": TypeDouble,
		"it": TypeInteger,
	}
}

// TestMutationAndMulDivDoubleRejected (#882): mixing a double value into a
// fixed/bigint field via a field mutation (`add/subtract … to/from`) or a
// `multiply/divide … by` must be rejected — the same #876 policy the binary
// operators enforce, extended to these paths via the getExprType fallback that
// resolves the operand's declared type.
func TestMutationAndMulDivDoubleRejected(t *testing.T) {
	bad := []string{
		"add db to fx",
		"subtract db from fx",
		"add db to bi",
		"subtract db from bi",
		"set fx = multiply fx by db",
		"set fx = divide fx by db",
		"set bi = multiply bi by db",
	}
	for _, src := range bad {
		c := NewCompiler()
		c.SetSymbols(mutSyms())
		pf, err := c.CompileAction(src)
		if err == nil {
			t.Errorf("%q: expected compile error, got postfix %q", src, pf)
			continue
		}
		if !strings.Contains(err.Error(), "cannot combine double with") {
			t.Errorf("%q: error should explain the double/exact mix, got: %v", src, err)
		}
	}
}

// TestMutationAndMulDivAllowed guards against over-rejection: integer/fixed
// values, double-into-double, increment, and multiply by int/double still
// compile. Also covers multi-statement actions (the reject is stateless — a
// per-parent check, not carried emitter state — so a rejected pattern can't
// contaminate a later statement).
func TestMutationAndMulDivAllowed(t *testing.T) {
	ok := []string{
		"add it to fx",                 // integer into fixed (cvfp)
		"add fx2 to fx",                // fixed into fixed
		"add 5 to fx",                  // integer literal
		"add db to db2",                // double into double
		"subtract it from fx",          // integer from fixed
		"increment fx",                 // +1
		"set fx = multiply fx by 2",    // fixed by integer
		"set db = multiply db by 2.0",  // double by float
		"set db = multiply db by it",   // double by integer
		"add it to fx; add 1 to fx2",   // multi-statement, both fine
	}
	for _, src := range ok {
		c := NewCompiler()
		c.SetSymbols(mutSyms())
		if _, err := c.CompileAction(src); err != nil {
			t.Errorf("%q: expected clean compile, got error: %v", src, err)
		}
	}
}
