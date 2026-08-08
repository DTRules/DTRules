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

// TestUnknownOperatorIsRejected pins #1020.
//
// Statement-form operator calls were emitted without checking the name against
// the registry, so a typo compiled clean, wrote postfix, passed build, and
// failed only when that row executed — reported as "The Name 'subests' was not
// defined by any Entity on the Entity Stack", which does not read like a typo.
// For a rule set computing money that is a defect found mid-period.
//
// This lives in authoring rather than in el because el cannot import the
// operator registry: operators imports pkg/dtrules, whose tests import el. The
// check is injected, and this is the layer that injects it.
func TestUnknownOperatorIsRejected(t *testing.T) {
	symbols := map[string]string{
		"hand.cards": "array", "hand.combos": "array",
	}

	if _, err := CheckAction(`subsets(hand.cards, "combo", "value", hand.combos)`, symbols); err != nil {
		t.Fatalf("a registered operator was rejected: %v", err)
	}

	for _, dsl := range []string{
		`subests(hand.cards, "combo", "value", hand.combos)`, // typo
		`frobnicate(hand.cards, "x", 3, hand.combos)`,        // invented
		`windows(hand.cards, 3, "combo", hand.combos)`,       // designed, not implemented
	} {
		_, err := CheckAction(dsl, symbols)
		if err == nil {
			t.Errorf("compiled a call to an operator the engine does not implement: %s", dsl)
			continue
		}
		if !strings.Contains(err.Error(), "unknown operator") {
			t.Errorf("error should name the cause for %s, got: %v", dsl, err)
		}
	}
}

// TestKnownOperatorsStillCompile guards against the check rejecting real
// operators — the failure mode that would be far worse than the bug.
func TestKnownOperatorsStillCompile(t *testing.T) {
	// Not "a" — the parser skips a/an/the as articles, so a single-letter
	// name fails to parse for reasons unrelated to the operator check.
	symbols := map[string]string{"src": "array", "dest": "array", "num": "integer"}
	for _, dsl := range []string{
		`subsets(src, "c", "v", dest)`,
		`combinations(src, num, "c", "v", dest)`,
		`groupby(src, "k", "g", dest)`,
		`maximalruns(src, "r", num, "g", dest)`,
	} {
		if _, err := CheckAction(dsl, symbols); err != nil {
			t.Errorf("registered operator rejected: %s\n  %v", dsl, err)
		}
	}
}
