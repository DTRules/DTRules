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

// Issue #1258: `subtract N from :e: field` / `subtract N from e's field`
// (subDestColon) stored N - field instead of field - N, and used the
// integer `-` whatever the field's type.
func TestSubtractFromColon_Runs(t *testing.T) {
	seed := func(calc, account dtrules.Entity) {
		calc.Put(dtrules.GetRName("n"), dtrules.GetRIntegerValue(100))
		calc.Put(dtrules.GetRName("d"), dtrules.GetRDoubleValue(100))
		account.Put(dtrules.GetRName("n"), dtrules.GetRIntegerValue(10))
		account.Put(dtrules.GetRName("d"), dtrules.GetRDoubleValue(10))
	}
	for _, tc := range []struct {
		el    string
		field string
		want  float64
	}{
		{`subtract 1 from :account: n`, "n", 9},
		{`subtract 1 from account's n`, "n", 9},
		{`subtract 1.5 from :account: d`, "d", 8.5},
		{`subtract 1.5 from account's d`, "d", 8.5},
		{`subtract 2 from account's d`, "d", 8},
	} {
		t.Run(tc.el, func(t *testing.T) {
			calc, account, err := elrtRun(t, nil, []string{tc.el}, seed)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			got, err := elrtGet(t, account, tc.field).DoubleValue()
			if err != nil || got != tc.want {
				t.Errorf("account.%s = %v (err %v), want %v", tc.field, got, err, tc.want)
			}
			// The subtraction happens on account; calc's same-named field
			// is left alone.
			if got, _ := elrtGet(t, calc, tc.field).DoubleValue(); got != 100 {
				t.Errorf("calc.%s = %v, want 100 (untouched)", tc.field, got)
			}
		})
	}
}
