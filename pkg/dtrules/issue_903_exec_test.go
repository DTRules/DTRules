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

package dtrules_test

import (
	"math/big"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestIssue903Execution (#903) runs the staking budget shape end-to-end at a
// mantissa scale beyond double's exact-integer range (2^53). Before the fix
// the dividend multiply compiled to fmul, so the product went through a
// float64 and the low-order mantissa digits were lost; a token test alone
// can't prove the fp path is actually exact at runtime.
func TestIssue903Execution(t *testing.T) {
	rs := session.NewRuleSet("i903")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)

	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"weekly_budget", "rate_fixed", "withholding_denom", "staker_budget"} {
		rootRef.AddAttribute(dtrules.GetRName(n), "", nil, true, true, dtrules.TypeFixed, "", "", "", "")
	}
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}

	mustFp := func(mantissa string) *dtrules.RFixed {
		t.Helper()
		m, ok := new(big.Int).SetString(mantissa, 10)
		if !ok {
			t.Fatalf("bad mantissa %q", mantissa)
		}
		fp, err := dtrules.GetRFixedFromMantissa(m)
		if err != nil {
			t.Fatalf("mantissa %s: %v", mantissa, err)
		}
		return fp
	}

	// weekly_budget mantissa = 2^53 + 1: the first integer a float64 cannot
	// represent. ~9.007e15 nanoACME-scale units, i.e. staking magnitude.
	// rate = 1.00000001 keeps the result mantissa odd and above 2^53, so the
	// expected value is NOT float64-representable (checked below).
	weeklyBudget := mustFp("9007199254740993")
	rate := mustFp("100000001")
	denom := mustFp("100000000") // 1.0
	root.Put(dtrules.GetRName("weekly_budget"), weeklyBudget)
	root.Put(dtrules.GetRName("rate_fixed"), rate)
	root.Put(dtrules.GetRName("withholding_denom"), denom)

	// Expected value via the exact RFixed API: trunc-mul then half-up divide.
	product, err := weeklyBudget.Mul(rate)
	if err != nil {
		t.Fatal(err)
	}
	want, err := product.DivRoundHalfUp(denom)
	if err != nil {
		t.Fatal(err)
	}

	// Teeth check: the expected mantissa must NOT be float64-representable,
	// otherwise a regression to double math could still pass this test.
	if _, acc := new(big.Float).SetInt(want.Mantissa()).Float64(); acc == big.Exact {
		t.Fatalf("expected mantissa %s is float64-exact; pick operands whose result exceeds 2^53 and is odd", want.Mantissa())
	}

	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{
		"weekly_budget":     "fixed",
		"rate_fixed":        "fixed",
		"withholding_denom": "fixed",
		"staker_budget":     "fixed",
	})
	pf, err := elc.CompileAction(
		"set staker_budget = divide weekly_budget * rate_fixed by withholding_denom rounding by 0.5fp")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	obj, err := sess.Compile(pf)
	if err != nil {
		t.Fatalf("assemble %q: %v", pf, err)
	}

	state := sess.GetState().(*interpreter.DTState)
	state.EntityPush(root)
	if err := obj.Execute(state); err != nil {
		state.EntityPop()
		t.Fatalf("execute %q: %v", pf, err)
	}
	state.EntityPop()

	got, err := root.Get(dtrules.GetRName("staker_budget"))
	if err != nil {
		t.Fatal(err)
	}
	gotFp, ok := got.(*dtrules.RFixed)
	if !ok {
		t.Fatalf("staker_budget is %T, want *RFixed (postfix: %s)", got, pf)
	}
	if gotFp.Mantissa().Cmp(want.Mantissa()) != 0 {
		t.Errorf("staker_budget mantissa = %s, want %s (postfix: %s)",
			gotFp.Mantissa(), want.Mantissa(), pf)
	}
}
