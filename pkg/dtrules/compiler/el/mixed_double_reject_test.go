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

func mdSymbols() map[string]string {
	return map[string]string{
		"fx": TypeFixed, "fx2": TypeFixed,
		"db": TypeDouble, "db2": TypeDouble,
		"bi": TypeBigInt, "bi2": TypeBigInt,
		"it": TypeInteger,
	}
}

// TestMixedDoubleExactRejected (#876): mixing a double with an exact type
// (fixed or bigint) in arithmetic or a comparison must be rejected at compile
// time, not silently compiled to an op that crashes at runtime. The runtime
// deliberately refuses to promote double→fixed implicitly; authors opt in with
// an explicit cast.
func TestMixedDoubleExactRejected(t *testing.T) {
	// action statements that must fail to compile
	badActions := []string{
		"set fx = fx2 + db",
		"set fx = fx2 - db",
		"set fx = fx2 * db",
		"set fx = fx2 / db",
		"set bi = bi2 + db",
		"set bi = db + bi2", // order independent
		"set fx = the minimum of fx2 and db",
		"set fx = the maximum of fx2 and db",
	}
	for _, src := range badActions {
		c := NewCompiler()
		c.SetSymbols(mdSymbols())
		pf, err := c.CompileAction(src)
		if err == nil {
			t.Errorf("%q: expected compile error, got postfix %q", src, pf)
			continue
		}
		if !strings.Contains(err.Error(), "cannot combine double with") {
			t.Errorf("%q: error should explain the double/exact mix, got: %v", src, err)
		}
	}

	// conditions that must fail to compile. Cover both comparison paths:
	// bare `name OP name` (name path: ==/!= → VisitBoolNameEq/Neq, ordering →
	// VisitBoolIntGt/…) and an iexpr context (`fx2 + fx == db` → VisitBoolIntEq).
	badConds := []string{
		"fx > db", "fx < db", "fx >= db", "bi <= db", "db >= bi", // ordering
		"fx == db", "fx != db", // equality, name path
		"fx2 + fx == db", "fx2 + fx != db", // equality, iexpr path
	}
	for _, src := range badConds {
		c := NewCompiler()
		c.SetSymbols(mdSymbols())
		if pf, err := c.CompileCondition(src); err == nil {
			t.Errorf("%q: expected compile error, got postfix %q", src, pf)
		}
	}
}

// TestMixedDoubleExactAllowedForms confirms the reject does not over-fire: an
// explicit cast and the legitimate widening cases still compile.
func TestMixedDoubleExactAllowedForms(t *testing.T) {
	ok := []string{
		"set fx = fx2 + (fixed) db", // explicit opt-in
		"set db = db2 + (double) fx", // explicit opt-in the other way
		"set fx = fx2 + it",          // fixed + integer (auto cvfp)
		"set fx = fx2 + bi",          // fixed + bigint (both exact)
		"set db = db2 + it",          // double + integer (widen)
		"set db = db2 + db",          // double + double
		"set bi = bi2 + it",          // bigint + integer
	}
	for _, src := range ok {
		c := NewCompiler()
		c.SetSymbols(mdSymbols())
		if _, err := c.CompileAction(src); err != nil {
			t.Errorf("%q: expected clean compile, got error: %v", src, err)
		}
	}
}
