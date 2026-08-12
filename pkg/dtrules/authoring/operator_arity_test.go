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

package authoring

import (
	"strings"
	"testing"
)

// TestOperatorArityIsChecked pins #1105, the half #1020 deferred.
//
// The name check landed in #1046; arity did not, because the registry recorded
// no argument count. A short call is the worse of the two failures: the
// runtime pops by position, so `subsets(hand.cards)` does not fail for want of
// three arguments — it takes whatever three values sit beneath it on the stack
// and treats them as the type name, sum field and destination array. That is a
// wrong answer, or a write into an unrelated array, not an error.
//
// Lives in authoring for the same reason as TestUnknownOperatorIsRejected: el
// cannot import the operator registry, so the lookup is injected, and this is
// the layer that injects it.
func TestOperatorArityIsChecked(t *testing.T) {
	symbols := map[string]string{
		"hand.cards": "array", "hand.combos": "array", "num": "integer",
	}

	for _, tc := range []struct {
		dsl  string
		want string // fragment the error must carry
	}{
		{`subsets(hand.cards)`, "4 arguments"},
		{`subsets(hand.cards, "combo", "value", hand.combos, num)`, "4 arguments"},
		{`groupby(hand.cards, "rank", "grp")`, "4 arguments"},
		{`combinations(hand.cards, num, "combo", "value")`, "5 arguments"},
		{`maximalruns(hand.cards, "rank", num, "seq")`, "5 arguments"},
		{`suffixes(hand.cards, num, "rank", "combo")`, "5 arguments"},
	} {
		_, err := CheckAction(tc.dsl, symbols)
		if err == nil {
			t.Errorf("compiled a call with the wrong argument count: %s", tc.dsl)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("error should say what was expected for %s\n  want fragment: %s\n  got: %v",
				tc.dsl, tc.want, err)
		}
		if !strings.Contains(err.Error(), "arguments") {
			t.Errorf("error should read as an arity problem for %s, got: %v", tc.dsl, err)
		}
	}
}

// TestCorrectArityStillCompiles is the guard that matters more than the check:
// rejecting a correct call would break every project using these operators.
// The Cribbage sample's eleven tables are the live case.
func TestCorrectArityStillCompiles(t *testing.T) {
	symbols := map[string]string{"src": "array", "dest": "array", "num": "integer"}
	for _, dsl := range []string{
		`subsets(src, "c", "v", dest)`,
		`combinations(src, num, "c", "v", dest)`,
		`groupby(src, "k", "g", dest)`,
		`maximalruns(src, "r", num, "g", dest)`,
		`suffixes(src, num, "r", "c", dest)`,
	} {
		if _, err := CheckAction(dsl, symbols); err != nil {
			t.Errorf("correct call rejected: %s\n  %v", dsl, err)
		}
	}
}

// TestUndeclaredArityIsNotChecked keeps the table sparse by design: an
// operator that never appears in the `name(a, b, …)` call form needs no
// entry, and must not be rejected for lacking one.
func TestUndeclaredArityIsNotChecked(t *testing.T) {
	symbols := map[string]string{"src": "array", "dest": "array"}
	// `copy` is registered and has no declared arity.
	if _, err := CheckAction(`copy(src, dest)`, symbols); err != nil &&
		strings.Contains(err.Error(), "arguments") {
		t.Errorf("an operator with no declared arity was arity-checked: %v", err)
	}
}
