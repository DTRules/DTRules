// Copyright 2024 Paul Snow
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

// Issue #686: the field-mutation visitors (add-to, subtract-from,
// increment, decrement) emitted type-naive postfix regardless of the
// declared field type. Integer targets got `+`/`-`, but fixed, bigint,
// and double targets also got plain `+`/`-` — the operator called
// LongValue() on the operand and stored back a truncated integer. The
// `add N to field` statement for bare-IDENT targets was even worse: it
// emitted just `value field` with no op, so the statement was a no-op
// at runtime.
//
// This test matrix covers the 4 mutation statements × 4 numeric target
// types = 16 cases, asserting each emits the right op family and that
// the integer path still produces the historic plain-op output.

func mutationSymbols() map[string]string {
	return map[string]string{
		"count":  TypeInteger,
		"total":  TypeBigInt,
		"rate":   TypeDouble,
		"amount": TypeFixed,
	}
}

type mutationCase struct {
	dsl     string
	mustHit []string // substrings that must appear in postfix
	mustNot []string // substrings that must not appear
}

func runMutationCases(t *testing.T, cases []mutationCase) {
	t.Helper()
	c := NewCompiler()
	c.SetSymbols(mutationSymbols())
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			postfix, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			for _, want := range tc.mustHit {
				if !strings.Contains(postfix, want) {
					t.Errorf("expected %q in postfix, got: %s", want, postfix)
				}
			}
			for _, bad := range tc.mustNot {
				if strings.Contains(postfix, bad) {
					t.Errorf("unexpected %q in postfix: %s", bad, postfix)
				}
			}
		})
	}
}

// TestFieldMutation_AddTo — `add <value> to <field>` across all four types.
func TestFieldMutation_AddTo(t *testing.T) {
	runMutationCases(t, []mutationCase{
		{
			dsl:     "add 1 to count",
			mustHit: []string{"+", "/count xdef"},
			mustNot: []string{"fp+", "b+", "f+", "cvfp", "cvbi", "cvd"},
		},
		{
			dsl:     "add 1 to total",
			mustHit: []string{"cvbi", "b+", "/total xdef"},
			mustNot: []string{"fp+", " + "},
		},
		{
			dsl:     "add 1.5 to rate",
			mustHit: []string{"cvd", "f+", "/rate xdef"},
			mustNot: []string{"fp+", "b+"},
		},
		{
			dsl:     "add 1.5fp to amount",
			mustHit: []string{"cvfp", "fp+", "/amount xdef"},
			mustNot: []string{"swap addto", " + /", "b+", "f+"},
		},
	})
}

// TestFieldMutation_SubtractFrom — `subtract <value> from <field>` across
// all four types.
func TestFieldMutation_SubtractFrom(t *testing.T) {
	runMutationCases(t, []mutationCase{
		{
			dsl:     "subtract 1 from count",
			mustHit: []string{"swap", "-", "/count xdef"},
			mustNot: []string{"fp-", "b-", "f-", "cvfp", "cvbi", "cvd"},
		},
		{
			dsl:     "subtract 1 from total",
			mustHit: []string{"cvbi", "b-", "/total xdef"},
			mustNot: []string{"fp-"},
		},
		{
			dsl:     "subtract 0.5 from rate",
			mustHit: []string{"cvd", "f-", "/rate xdef"},
			mustNot: []string{"fp-", "b-"},
		},
		{
			dsl:     "subtract 0.5fp from amount",
			mustHit: []string{"cvfp", "fp-", "/amount xdef"},
			mustNot: []string{"b-", "f-"},
		},
	})
}

// TestFieldMutation_Increment — `increment <field>` across all four types.
// This matrix also guards the double path, which was previously broken:
// VisitIncrementDouble is almost never reached (typedLong wins), and
// VisitIncrementLong (before the fix) emitted `field 1 + /field xdef` —
// integer `+` on a double operand truncates to int64.
func TestFieldMutation_Increment(t *testing.T) {
	runMutationCases(t, []mutationCase{
		{
			dsl: "increment count",
			// New unified pattern is `1 count + /count xdef` (value pushed
			// first via emitOneForField, then the shared store-back helper).
			// Semantically identical to the old `count 1 + /count xdef`
			// since integer + is commutative.
			mustHit: []string{"1 count", " + ", "/count xdef"},
			mustNot: []string{"fp+", "b+", "f+", "cvfp", "cvbi", "cvd"},
		},
		{
			dsl:     "increment total",
			mustHit: []string{"cvbi", "b+", "/total xdef"},
			mustNot: []string{"fp+"},
		},
		{
			dsl:     "increment rate",
			mustHit: []string{"1.0", "cvd", "f+", "/rate xdef"},
			mustNot: []string{"fp+", "b+", "rate 1 +"},
		},
		{
			dsl:     "increment amount",
			mustHit: []string{"cvfp", "fp+", "/amount xdef"},
			mustNot: []string{"b+", "f+", "amount 1 +"},
		},
	})
}

// TestFieldMutation_Decrement — parallel to Increment.
func TestFieldMutation_Decrement(t *testing.T) {
	runMutationCases(t, []mutationCase{
		{
			dsl: "decrement count",
			// New unified pattern: `1 count swap - /count xdef` — swap
			// ensures the subtract direction is `field - value`.
			mustHit: []string{"1 count swap -", "/count xdef"},
			mustNot: []string{"fp-", "b-", "f-", "cvfp", "cvbi"},
		},
		{
			dsl:     "decrement total",
			mustHit: []string{"cvbi", "b-", "/total xdef"},
			mustNot: []string{"fp-"},
		},
		{
			dsl:     "decrement rate",
			mustHit: []string{"1.0", "cvd", "f-", "/rate xdef"},
			mustNot: []string{"fp-", "rate 1 -"},
		},
		{
			dsl:     "decrement amount",
			mustHit: []string{"cvfp", "fp-", "/amount xdef"},
			mustNot: []string{"amount 1 -"},
		},
	})
}

// TestFieldMutation_ColonRefTargets guards the possessive / colon-ref
// variant: `add 1.5fp to wp.amount` also has to route through the
// type-aware dispatch. This is the common shape in decision-table
// authoring (entity.field references).
func TestFieldMutation_ColonRefTargets(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"wp.count":  TypeInteger,
		"wp.total":  TypeBigInt,
		"wp.rate":   TypeDouble,
		"wp.amount": TypeFixed,
	})
	cases := []mutationCase{
		{
			dsl:     "add 1.5fp to wp.amount",
			mustHit: []string{"cvfp", "fp+", "/wp.amount xdef"},
			mustNot: []string{"swap addto"},
		},
		{
			dsl:     "subtract 1.5fp from wp.amount",
			mustHit: []string{"cvfp", "fp-", "/wp.amount xdef"},
		},
		{
			dsl:     "add 1 to wp.total",
			mustHit: []string{"cvbi", "b+", "/wp.total xdef"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			postfix, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			for _, want := range tc.mustHit {
				if !strings.Contains(postfix, want) {
					t.Errorf("expected %q, got: %s", want, postfix)
				}
			}
			for _, bad := range tc.mustNot {
				if strings.Contains(postfix, bad) {
					t.Errorf("unexpected %q: %s", bad, postfix)
				}
			}
		})
	}
}
