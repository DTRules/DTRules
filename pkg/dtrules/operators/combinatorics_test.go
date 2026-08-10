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

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// combHarness is a real session with the EDD types the combinatorial
// generators materialize: card (source), combo, group, and run.
type combHarness struct {
	sess  dtrules.Session
	state *interpreter.DTState
	ef    *entity.Factory
}

func newCombHarness(t *testing.T) *combHarness {
	t.Helper()
	rs := session.NewRuleSet("combtest")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	h := &combHarness{sess: sess, state: sess.GetState().(*interpreter.DTState), ef: ef}

	h.declare(t, "card", []string{"rank", "suit", "value"}, nil)
	h.declare(t, "combo", []string{"count", "sum", "distinct", "spread"}, []string{"members"})
	h.declare(t, "group", []string{"key", "count"}, []string{"members"})
	h.declare(t, "run", []string{"start", "span", "multiplicity"}, nil)
	return h
}

func (h *combHarness) declare(t *testing.T, name string, intFields, arrayFields []string) {
	t.Helper()
	ref, err := h.ef.FindCreateRefEntity(true, dtrules.GetRName(name))
	if err != nil {
		t.Fatalf("FindCreateRefEntity(%s): %v", name, err)
	}
	for _, f := range intFields {
		ref.AddAttribute(dtrules.GetRName(f), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")
	}
	for _, f := range arrayFields {
		arr, err := dtrules.NewArray(h.sess, true, false)
		if err != nil {
			t.Fatalf("NewArray: %v", err)
		}
		ref.AddAttribute(dtrules.GetRName(f), "", arr, true, true, dtrules.TypeArray, "", "", "", "")
	}
}

// cards builds card entities with the given ranks; value = min(rank, 10),
// suit fixed — the cribbage counting convention, handy for sum checks.
func (h *combHarness) cards(t *testing.T, ranks ...int) *dtrules.RArray {
	t.Helper()
	arr, err := dtrules.NewArray(h.sess, true, false)
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	for _, r := range ranks {
		c, err := h.ef.CreateEntity(h.sess, dtrules.GetRName("card"))
		if err != nil {
			t.Fatalf("CreateEntity(card): %v", err)
		}
		v := r
		if v > 10 {
			v = 10
		}
		c.Put(dtrules.GetRName("rank"), dtrules.GetRIntegerValue(int64(r)))
		c.Put(dtrules.GetRName("suit"), dtrules.GetRIntegerValue(0))
		c.Put(dtrules.GetRName("value"), dtrules.GetRIntegerValue(int64(v)))
		arr.Add(c)
	}
	return arr
}

func (h *combHarness) emptyArray(t *testing.T) *dtrules.RArray {
	t.Helper()
	arr, err := dtrules.NewArray(h.sess, true, false)
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	return arr
}

// exec pushes args and runs a registered operator.
func (h *combHarness) exec(t *testing.T, opName string, args ...dtrules.Object) error {
	t.Helper()
	for _, a := range args {
		if err := h.state.DataPush(a); err != nil {
			t.Fatalf("DataPush: %v", err)
		}
	}
	op, ok := operators.Get(dtrules.GetRName(opName))
	if !ok {
		t.Fatalf("operator %q not registered", opName)
	}
	return op.Execute(h.state)
}

func intAttr(t *testing.T, obj dtrules.Object, field string) int {
	t.Helper()
	ent, err := obj.REntityValue()
	if err != nil {
		t.Fatalf("not an entity: %v", err)
	}
	v, err := ent.Get(dtrules.GetRName(field))
	if err != nil {
		t.Fatalf("get %s: %v", field, err)
	}
	n, err := v.IntValue()
	if err != nil {
		t.Fatalf("%s not an int: %v", field, err)
	}
	return n
}

func str(s string) *dtrules.RString { return dtrules.NewRString(s) }

func TestCombinationsCountSizeAndSum(t *testing.T) {
	h := newCombHarness(t)
	src := h.cards(t, 1, 2, 3, 4, 5)
	dest := h.emptyArray(t)

	if err := h.exec(t, "combinations", src, dtrules.GetRIntegerValue(2), str("combo"), str("value"), dest); err != nil {
		t.Fatalf("combinations: %v", err)
	}
	elems, _ := dest.ArrayValue()
	if len(elems) != 10 {
		t.Fatalf("C(5,2) = 10 combos expected, got %d", len(elems))
	}
	// Sum of all pair sums = each value appearing in (n-1)=4 pairs: 4*(1+2+3+4+5) = 60.
	total := 0
	for _, e := range elems {
		if got := intAttr(t, e, "count"); got != 2 {
			t.Errorf("count = %d, want 2", got)
		}
		total += intAttr(t, e, "sum")
	}
	if total != 60 {
		t.Errorf("Σ pair sums = %d, want 60", total)
	}
	if h.state.DataStackDepth() != 0 {
		t.Errorf("stack not clean: depth %d", h.state.DataStackDepth())
	}
}

func TestSubsetsEnumeratesThePowerSet(t *testing.T) {
	h := newCombHarness(t)
	src := h.cards(t, 1, 2, 3, 4, 5)
	dest := h.emptyArray(t)

	if err := h.exec(t, "subsets", src, str("combo"), str("value"), dest); err != nil {
		t.Fatalf("subsets: %v", err)
	}
	elems, _ := dest.ArrayValue()
	if len(elems) != 31 {
		t.Fatalf("2^5-1 = 31 subsets expected, got %d", len(elems))
	}
	fifteens := 0
	for _, e := range elems {
		if intAttr(t, e, "sum") == 15 {
			fifteens++
		}
	}
	// Values {1,2,3,4,5}: only the full set sums to 15.
	if fifteens != 1 {
		t.Errorf("subsets summing to 15 = %d, want 1", fifteens)
	}
	if h.state.DataStackDepth() != 0 {
		t.Errorf("stack not clean: depth %d", h.state.DataStackDepth())
	}
}

func TestCombinationsEdgeCases(t *testing.T) {
	h := newCombHarness(t)

	// k = 0 and k > n append nothing and do not error.
	for _, k := range []int64{0, 9} {
		dest := h.emptyArray(t)
		if err := h.exec(t, "combinations", h.cards(t, 1, 2, 3), dtrules.GetRIntegerValue(k), str("combo"), str("value"), dest); err != nil {
			t.Fatalf("combinations k=%d: %v", k, err)
		}
		if dest.Size() != 0 {
			t.Errorf("k=%d: expected empty dest, got %d", k, dest.Size())
		}
	}

	// Empty sumfield stamps sum 0 without field lookups.
	dest := h.emptyArray(t)
	if err := h.exec(t, "combinations", h.cards(t, 5, 6), dtrules.GetRIntegerValue(2), str("combo"), str(""), dest); err != nil {
		t.Fatalf("combinations sumfield='': %v", err)
	}
	elems, _ := dest.ArrayValue()
	if len(elems) != 1 || intAttr(t, elems[0], "sum") != 0 {
		t.Errorf("expected one combo with sum 0")
	}

	// The cap is an error, not an OOM.
	big := make([]int, 21)
	for i := range big {
		big[i] = i + 1
	}
	err := h.exec(t, "combinations", h.cards(t, big...), dtrules.GetRIntegerValue(2), str("combo"), str(""), h.emptyArray(t))
	if err == nil || !strings.Contains(err.Error(), "cap") {
		t.Errorf("expected cap error, got %v", err)
	}
}

func TestSubsetsErrors(t *testing.T) {
	h := newCombHarness(t)

	// Unknown EDD type.
	err := h.exec(t, "subsets", h.cards(t, 1, 2), str("nosuchtype"), str("value"), h.emptyArray(t))
	if err == nil {
		t.Error("expected an error for an unknown EDD type")
	}

	// Missing sum field on the members.
	err = h.exec(t, "subsets", h.cards(t, 1, 2), str("combo"), str("nosuchfield"), h.emptyArray(t))
	if err == nil {
		t.Error("expected an error for a missing sum field")
	}

	// Cap: 13 source entities exceed the 2^12-1 design ceiling.
	big := make([]int, 13)
	for i := range big {
		big[i] = i + 1
	}
	err = h.exec(t, "subsets", h.cards(t, big...), str("combo"), str(""), h.emptyArray(t))
	if err == nil || !strings.Contains(err.Error(), "cap") {
		t.Errorf("expected cap error, got %v", err)
	}
}

func TestGroupByPartitionsInFirstSeenOrder(t *testing.T) {
	h := newCombHarness(t)
	src := h.cards(t, 5, 5, 5, 11)
	dest := h.emptyArray(t)

	if err := h.exec(t, "groupby", src, str("rank"), str("group"), dest); err != nil {
		t.Fatalf("groupby: %v", err)
	}
	elems, _ := dest.ArrayValue()
	if len(elems) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(elems))
	}
	if k, c := intAttr(t, elems[0], "key"), intAttr(t, elems[0], "count"); k != 5 || c != 3 {
		t.Errorf("first group = (key %d, count %d), want (5, 3)", k, c)
	}
	if k, c := intAttr(t, elems[1], "key"), intAttr(t, elems[1], "count"); k != 11 || c != 1 {
		t.Errorf("second group = (key %d, count %d), want (11, 1)", k, c)
	}
	if h.state.DataStackDepth() != 0 {
		t.Errorf("stack not clean: depth %d", h.state.DataStackDepth())
	}
}

func TestGroupByDegenerateShapes(t *testing.T) {
	h := newCombHarness(t)

	// All distinct → n groups of count 1.
	dest := h.emptyArray(t)
	if err := h.exec(t, "groupby", h.cards(t, 1, 2, 3), str("rank"), str("group"), dest); err != nil {
		t.Fatalf("groupby: %v", err)
	}
	if dest.Size() != 3 {
		t.Errorf("distinct ranks: got %d groups, want 3", dest.Size())
	}

	// Empty source → empty dest, no error.
	dest = h.emptyArray(t)
	if err := h.exec(t, "groupby", h.emptyArray(t), str("rank"), str("group"), dest); err != nil {
		t.Fatalf("groupby empty: %v", err)
	}
	if dest.Size() != 0 {
		t.Errorf("empty source: got %d groups, want 0", dest.Size())
	}

	// Missing key field is a descriptive error.
	if err := h.exec(t, "groupby", h.cards(t, 1), str("nosuchfield"), str("group"), h.emptyArray(t)); err == nil {
		t.Error("expected an error for a missing key field")
	}
}

func TestMaximalRunsDetection(t *testing.T) {
	h := newCombHarness(t)
	three := dtrules.GetRIntegerValue(3)

	cases := []struct {
		name  string
		ranks []int
		wants [][3]int // start, length, multiplicity
	}{
		{"simple run", []int{4, 5, 6}, [][3]int{{4, 3, 1}}},
		{"order independent", []int{6, 4, 5}, [][3]int{{4, 3, 1}}},
		{"double run", []int{7, 8, 8, 9, 10}, [][3]int{{7, 4, 2}}},
		{"two runs", []int{1, 2, 3, 8, 9, 10}, [][3]int{{1, 3, 1}, {8, 3, 1}}},
		{"gap splits below minlen", []int{1, 2, 4, 5}, nil},
		{"double double", []int{5, 5, 6, 6, 7}, [][3]int{{5, 3, 4}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest := h.emptyArray(t)
			if err := h.exec(t, "maximalruns", h.cards(t, tc.ranks...), str("rank"), three, str("run"), dest); err != nil {
				t.Fatalf("maximalruns: %v", err)
			}
			elems, _ := dest.ArrayValue()
			if len(elems) != len(tc.wants) {
				t.Fatalf("got %d runs, want %d", len(elems), len(tc.wants))
			}
			for i, want := range tc.wants {
				got := [3]int{intAttr(t, elems[i], "start"), intAttr(t, elems[i], "span"), intAttr(t, elems[i], "multiplicity")}
				if got != want {
					t.Errorf("run %d = %v, want %v", i, got, want)
				}
			}
			if h.state.DataStackDepth() != 0 {
				t.Errorf("stack not clean: depth %d", h.state.DataStackDepth())
			}
		})
	}
}

func TestMaximalRunsMinLen(t *testing.T) {
	h := newCombHarness(t)
	dest := h.emptyArray(t)
	if err := h.exec(t, "maximalruns", h.cards(t, 7, 8, 8, 9, 10), str("rank"), dtrules.GetRIntegerValue(5), str("run"), dest); err != nil {
		t.Fatalf("maximalruns: %v", err)
	}
	if dest.Size() != 0 {
		t.Errorf("minlen 5 over a length-4 run: got %d runs, want 0", dest.Size())
	}
}

func TestSuffixesEmitsTrailingWindowsLongestFirst(t *testing.T) {
	h := newCombHarness(t)
	// Lay order 4, 6, 5: the pegging classic — a run in any order.
	src := h.cards(t, 4, 6, 5)
	dest := h.emptyArray(t)

	if err := h.exec(t, "suffixes", src, dtrules.GetRIntegerValue(2), str("rank"), str("combo"), dest); err != nil {
		t.Fatalf("suffixes: %v", err)
	}
	elems, _ := dest.ArrayValue()
	if len(elems) != 2 {
		t.Fatalf("windows of len >= 2 from 3 cards: got %d, want 2", len(elems))
	}
	// Longest first: [4,6,5] then [6,5].
	first := [3]int{intAttr(t, elems[0], "count"), intAttr(t, elems[0], "distinct"), intAttr(t, elems[0], "spread")}
	if first != [3]int{3, 3, 2} {
		t.Errorf("longest window = (count,distinct,spread) %v, want {3 3 2} — a run in any order", first)
	}
	second := [3]int{intAttr(t, elems[1], "count"), intAttr(t, elems[1], "distinct"), intAttr(t, elems[1], "spread")}
	if second != [3]int{2, 2, 1} {
		t.Errorf("second window = %v, want {2 2 1}", second)
	}
	if got := intAttr(t, elems[0], "sum"); got != 15 {
		t.Errorf("sum over full window = %d, want 15", got)
	}
	if h.state.DataStackDepth() != 0 {
		t.Errorf("stack not clean: depth %d", h.state.DataStackDepth())
	}
}

func TestSuffixesPairAndBrokenPairShapes(t *testing.T) {
	h := newCombHarness(t)

	// Trailing trips: [K,8,8,8] — windows [K888]{4,2,5}, [888]{3,1,0}, [88]{2,1,0}.
	dest := h.emptyArray(t)
	if err := h.exec(t, "suffixes", h.cards(t, 13, 8, 8, 8), dtrules.GetRIntegerValue(2), str("rank"), str("combo"), dest); err != nil {
		t.Fatalf("suffixes: %v", err)
	}
	elems, _ := dest.ArrayValue()
	if len(elems) != 3 {
		t.Fatalf("got %d windows, want 3", len(elems))
	}
	// The first window with distinct == 1 is the maximal trailing block.
	if d, c := intAttr(t, elems[1], "distinct"), intAttr(t, elems[1], "count"); d != 1 || c != 3 {
		t.Errorf("trips window = (distinct %d, count %d), want (1, 3)", d, c)
	}

	// A broken pair [7,2,7] has no window with distinct == 1.
	dest = h.emptyArray(t)
	if err := h.exec(t, "suffixes", h.cards(t, 7, 2, 7), dtrules.GetRIntegerValue(2), str("rank"), str("combo"), dest); err != nil {
		t.Fatalf("suffixes: %v", err)
	}
	elems, _ = dest.ArrayValue()
	for i, e := range elems {
		if intAttr(t, e, "distinct") == 1 {
			t.Errorf("window %d claims a trailing pair in a broken-pair stack", i)
		}
	}

	// minlen respected; a single element yields nothing at minlen 2.
	dest = h.emptyArray(t)
	if err := h.exec(t, "suffixes", h.cards(t, 9), dtrules.GetRIntegerValue(2), str("rank"), str("combo"), dest); err != nil {
		t.Fatalf("suffixes single: %v", err)
	}
	if dest.Size() != 0 {
		t.Errorf("single card at minlen 2: got %d windows, want 0", dest.Size())
	}

	// Empty statfield is an error.
	if err := h.exec(t, "suffixes", h.cards(t, 1, 2), dtrules.GetRIntegerValue(2), str(""), str("combo"), h.emptyArray(t)); err == nil {
		t.Error("expected an error for an empty statfield")
	}
}

// TestGeneratorsShareMembersByReference: the members arrays must hold the
// same entities as the source — tables reach through group.members to the
// original cards, so copies would break attribute writes.
func TestGeneratorsShareMembersByReference(t *testing.T) {
	h := newCombHarness(t)
	src := h.cards(t, 5, 5)
	dest := h.emptyArray(t)
	if err := h.exec(t, "groupby", src, str("rank"), str("group"), dest); err != nil {
		t.Fatalf("groupby: %v", err)
	}
	elems, _ := dest.ArrayValue()
	grp, _ := elems[0].REntityValue()
	membersObj, _ := grp.Get(dtrules.GetRName("members"))
	members, _ := membersObj.ArrayValue()
	srcElems, _ := src.ArrayValue()
	for i := range members {
		me, _ := members[i].REntityValue()
		se, _ := srcElems[i].REntityValue()
		if me.GetID() != se.GetID() {
			t.Errorf("member %d is not the same entity as the source (ids %d vs %d)", i, me.GetID(), se.GetID())
		}
	}
}
