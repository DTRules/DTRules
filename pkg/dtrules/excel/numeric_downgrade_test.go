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

package excel

import (
	"strings"
	"testing"
)

// A committed artifact can carry postfix an older compiler produced from DSL
// the current one reads differently, and a rebuild then changes the arithmetic
// with nothing reported. The staking rules are the case: DSL says `/`, which
// truncates, while the committed postfix says `fphalfup/`, which their spec
// requires and their own test asserts (#1019).

func warningsFor(t *testing.T, before, after string) []string {
	t.Helper()
	imp := &DTImporter{stats: &ImportStats{}}
	imp.warnNumericDowngrade("T", "action 1", before, after)
	msgs := make([]string, 0, len(imp.stats.Warnings))
	for _, w := range imp.stats.Warnings {
		msgs = append(msgs, w.Reason)
	}
	return msgs
}

func TestWarnsWhenRoundHalfUpBecomesTruncating(t *testing.T) {
	got := warningsFor(t,
		"weighted_balance staker_budget fp* total_weighted fphalfup/",
		"weighted_balance staker_budget fp* total_weighted fp/")

	if len(got) != 1 {
		t.Fatalf("want exactly one warning, got %d: %v", len(got), got)
	}
	for _, want := range []string{"fphalfup/", "fp/", "round-half-up", "0.5fp"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("warning should mention %q, got: %s", want, got[0])
		}
	}
}

func TestWarnsWhenFixedPointBecomesInteger(t *testing.T) {
	got := warningsFor(t, "a b fp- amount cvfp", "a b - amount cvi")
	if len(got) != 2 {
		t.Fatalf("want a warning for each of fp- and cvfp, got %d: %v", len(got), got)
	}
}

// fp/ is a substring of fphalfup/. Matching on substrings would report every
// correct round-half-up row as having lost a fixed-point division.
func TestRoundHalfUpIsNotReadAsLosingFixedDivide(t *testing.T) {
	if got := warningsFor(t,
		"a b fp/", "a b fphalfup/"); len(got) != 0 {
		t.Errorf("gaining rounding is an upgrade, not a downgrade: %v", got)
	}
}

func TestNoWarningWhenPostfixIsUnchanged(t *testing.T) {
	p := "a b fphalfup/"
	if got := warningsFor(t, p, p); len(got) != 0 {
		t.Errorf("identical postfix must be silent: %v", got)
	}
}

// A first compile has nothing to compare against, and must not be reported as
// a loss just because the new postfix contains the weaker operator.
func TestNoWarningWhenThereWasNoPriorPostfix(t *testing.T) {
	if got := warningsFor(t, "", "a b fp/"); len(got) != 0 {
		t.Errorf("a first compile has nothing to downgrade from: %v", got)
	}
}

// #1015 rewrote multiply chains across the samples. Reordering operands must
// stay silent -- the operator set is untouched.
func TestReorderingOperandsIsNotADowngrade(t *testing.T) {
	if got := warningsFor(t,
		"a b fp* c fp* d fphalfup/",
		"b c fp* a fp* d fphalfup/"); len(got) != 0 {
		t.Errorf("same operators in a different order is not a downgrade: %v", got)
	}
}

// Losing one fp* of several still matters, but only when a plain * appears in
// its place -- otherwise the row was simply edited.
func TestOperatorRemovedWithoutAWeakerReplacementIsSilent(t *testing.T) {
	if got := warningsFor(t, "a b fp* c fp*", "a b fp*"); len(got) != 0 {
		t.Errorf("deleting an operation is an edit, not a downgrade: %v", got)
	}
}
