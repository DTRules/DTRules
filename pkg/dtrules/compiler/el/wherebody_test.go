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

// TestWhereBodyTakesTheFullPredicate pins #1121. The bexpr-hosted folds sit
// above AND/OR in the left-recursive rule, so their trailing `bexpr` was
// precedence-constrained: `where A and B` parsed as `(fold where A) and B`,
// with B evaluated after the loop — an undefined name at best, a silent read
// of the wrong binding at worst. The whereBody sub-rule re-enters bexpr at
// full precedence, so the natural phrasing must now compile identically to
// the parenthesized workaround.
func TestWhereBodyTakesTheFullPredicate(t *testing.T) {
	newC := func() *Compiler {
		c := NewCompiler()
		c.SetSymbols(map[string]string{
			"pile": "array", "kids": "array",
			"rank": "integer", "suit": "integer", "age": "integer",
			"flag": "boolean",
		})
		return c
	}

	for _, tc := range []struct{ name, natural, parenthesized string }{
		{
			"there is … where A and B",
			`there is card in pile where rank == 7 and suit == 0`,
			`there is card in pile where (rank == 7 and suit == 0)`,
		},
		{
			"there is no … where A or B",
			`there is no card in pile where rank == 7 or suit == 0`,
			`there is no card in pile where (rank == 7 or suit == 0)`,
		},
		{
			"all … have A and B",
			`all kids have age > 5 and age < 18`,
			`all kids have (age > 5 and age < 18)`,
		},
		{
			"one of … has a A and B",
			`one of kids has a age > 5 and suit == 0`,
			`one of kids has a (age > 5 and suit == 0)`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			natural, err := newC().CompileCondition(tc.natural)
			if err != nil {
				t.Fatalf("natural phrasing failed to compile: %v", err)
			}
			explicit, err := newC().CompileCondition(tc.parenthesized)
			if err != nil {
				t.Fatalf("parenthesized phrasing failed to compile: %v", err)
			}
			if natural != explicit {
				t.Errorf("natural phrasing must equal the parenthesized form\n  natural: %q\n  parens:  %q",
					natural, explicit)
			}
			// The second predicate must live inside the fold: after this fix
			// nothing may follow the forall except the fold's own not.
			trimmed := strings.TrimSuffix(strings.TrimSpace(natural), " not")
			if !strings.HasSuffix(trimmed, "forall") && !strings.HasSuffix(trimmed, "pop") {
				t.Errorf("predicate escaped the fold: %q", natural)
			}
		})
	}
}

// TestWhereBodyOuterConjunctionStillExpressible: an author who really wants
// the conjunction outside the fold can still write it — with the fold
// parenthesized — and gets a different program than the fold-internal form.
func TestWhereBodyOuterConjunctionStillExpressible(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"pile": "array", "rank": "integer", "flag": "boolean"})
	outer, err := c.CompileCondition(`(there is card in pile where rank == 7) and flag`)
	if err != nil {
		t.Fatalf("outer conjunction failed to compile: %v", err)
	}
	c2 := NewCompiler()
	c2.SetSymbols(map[string]string{"pile": "array", "rank": "integer", "flag": "boolean"})
	inner, err := c2.CompileCondition(`there is card in pile where rank == 7 and flag`)
	if err != nil {
		t.Fatalf("inner conjunction failed to compile: %v", err)
	}
	if outer == inner {
		t.Error("outer and inner conjunction must compile to different programs")
	}
	// The outer form's second operand follows the fold (AND lowers to the
	// short-circuit `{ pop B } over if` shape, so B appears after forall).
	forallPos := strings.Index(outer, "forall")
	flagPos := strings.LastIndex(outer, "flag")
	if forallPos == -1 || flagPos < forallPos {
		t.Errorf("outer form should evaluate flag after the fold: %q", outer)
	}
}
