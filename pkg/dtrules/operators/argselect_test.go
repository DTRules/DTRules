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

package operators_test

import (
	"strings"
	"testing"
)

// `max of <field> in <array>` gives the winning value; these give the winning
// element, which is what a rule needs before it can read that element's other
// fields. Without them the table can score the options and not say which one
// won, so the choice moves to host code even when the criterion is pure
// policy (#1024).

func TestArgMaxSelectsTheWinningElement(t *testing.T) {
	h := newCombHarness(t)
	src := h.cards(t, 5, 11, 3)
	dest := h.emptyArray(t)

	if err := h.exec(t, "argmax", src, str("rank"), dest); err != nil {
		t.Fatalf("argmax: %v", err)
	}
	if dest.Size() != 1 {
		t.Fatalf("expected exactly one winner, got %d", dest.Size())
	}
	elems, _ := dest.ArrayValue()
	if got := intAttr(t, elems[0], "rank"); got != 11 {
		t.Errorf("argmax picked rank %d, want 11", got)
	}
	if h.state.DataStackDepth() != 0 {
		t.Errorf("stack not clean: depth %d", h.state.DataStackDepth())
	}
}

func TestArgMinSelectsTheSmallest(t *testing.T) {
	h := newCombHarness(t)
	dest := h.emptyArray(t)

	if err := h.exec(t, "argmin", h.cards(t, 5, 11, 3), str("rank"), dest); err != nil {
		t.Fatalf("argmin: %v", err)
	}
	elems, _ := dest.ArrayValue()
	if got := intAttr(t, elems[0], "rank"); got != 3 {
		t.Errorf("argmin picked rank %d, want 3", got)
	}
}

// Ties go to the first in array order, so re-running the same rules on the
// same data gives the same answer. That matters more for an advice rule than
// any rule about which of two equal options is better.
func TestTiesGoToTheFirstElement(t *testing.T) {
	h := newCombHarness(t)
	dest := h.emptyArray(t)

	src := h.cards(t, 7, 7, 2)
	if err := h.exec(t, "argmax", src, str("rank"), dest); err != nil {
		t.Fatalf("argmax: %v", err)
	}
	srcElems, _ := src.ArrayValue()
	destElems, _ := dest.ArrayValue()
	if destElems[0] != srcElems[0] {
		t.Error("a tie did not go to the first element, so the result is not stable across runs")
	}
}

// "No options to choose between" is a state a rule can test with `number of`;
// the operator should not throw at a table that has already established there
// are options.
func TestEmptySourceLeavesDestEmpty(t *testing.T) {
	h := newCombHarness(t)
	dest := h.emptyArray(t)

	if err := h.exec(t, "argmax", h.emptyArray(t), str("rank"), dest); err != nil {
		t.Fatalf("argmax on an empty source should not fail: %v", err)
	}
	if dest.Size() != 0 {
		t.Errorf("empty source produced %d winners", dest.Size())
	}
}

// A table that runs twice must select, not accumulate.
func TestDestIsClearedNotAppendedTo(t *testing.T) {
	h := newCombHarness(t)
	dest := h.emptyArray(t)

	for i := 0; i < 2; i++ {
		if err := h.exec(t, "argmax", h.cards(t, 4, 9), str("rank"), dest); err != nil {
			t.Fatalf("argmax run %d: %v", i, err)
		}
	}
	if dest.Size() != 1 {
		t.Errorf("after two runs dest holds %d elements, want 1", dest.Size())
	}
}

func TestMissingFieldIsADescriptiveError(t *testing.T) {
	h := newCombHarness(t)
	err := h.exec(t, "argmax", h.cards(t, 1), str("nosuchfield"), h.emptyArray(t))
	if err == nil {
		t.Fatal("selecting on a field that does not exist should be an error")
	}
	if !strings.Contains(err.Error(), "nosuchfield") {
		t.Errorf("error should name the missing field: %v", err)
	}
}
