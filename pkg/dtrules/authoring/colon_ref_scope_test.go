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

package authoring_test

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Issue #1257: `:e: field` and `e's field` pushed e onto the data stack and
// never made it the current entity, so the field was read from (or written
// to) whatever entity was current and e was left on the stack. calc and
// account both declare n and flag with different values, so a reference
// that resolves against the wrong one shows.
func elrt1257Seed(calc, account dtrules.Entity) {
	calc.Put(dtrules.GetRName("n"), dtrules.GetRIntegerValue(100))
	calc.Put(dtrules.GetRName("flag"), dtrules.GetRBoolean(false))
	account.Put(dtrules.GetRName("n"), dtrules.GetRIntegerValue(7))
	account.Put(dtrules.GetRName("flag"), dtrules.GetRBoolean(true))
}

func TestColonRefScope_Reads(t *testing.T) {
	for _, cond := range []string{
		`:account: flag`,
		`account's flag`,
		`:account: flag is true`,
		`:account: n == 7`,
		`account's n == 7`,
		`:account: n + 1 == 8`,
		// A chain reads each link with the one before it current.
		`calc's account's n == 7`,
		`:calc: account's n == 7`,
	} {
		t.Run(cond, func(t *testing.T) {
			calc, _, err := elrtRun(t, []string{cond}, []string{`set calc.hit = true`}, elrt1257Seed)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got, _ := elrtGet(t, calc, "hit").BooleanValue(); !got {
				t.Errorf("condition %s was false", cond)
			}
		})
	}
}

func TestColonRefScope_Writes(t *testing.T) {
	for _, tc := range []struct {
		el   string
		want int64
	}{
		{`set :account: n = 1`, 1},
		{`set account's n = 2`, 2},
	} {
		t.Run(tc.el, func(t *testing.T) {
			calc, account, err := elrtRun(t, nil, []string{tc.el}, elrt1257Seed)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got, err := elrtGet(t, account, "n").LongValue(); err != nil || got != tc.want {
				t.Errorf("account.n = %d (err %v), want %d", got, err, tc.want)
			}
			if got := elrtGet(t, calc, "n").StringValue(); got != "100" {
				t.Errorf("calc.n = %s, want 100 (untouched)", got)
			}
		})
	}
}
