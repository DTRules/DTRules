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

// The acceptance suite for #1148: every way EL arithmetic must group once
// precedence belongs to operators instead of type-pair alternatives.
//
// Grouping today is UNSTABLE. It depends on which alternative ANTLR's
// prediction picks per operator, which depends on surrounding tokens and
// literal kinds: `a + 2 - b` groups left and `a + 2.0 - b` groups right;
// `the minimum of (a / b * c)` is correct and `the maximum of (a + b - c)`
// is not; `-a + b - c` negates the whole chain. iexpr-only chains are always
// correct; fexpr chains are correct only when prediction happens to route
// every operator through the same alternative.
//
// Every `want` below is the C/C++/Go reading. Every `brokenToday` is the
// output verified on main before the nexpr work, kept so the suite is green
// now, catches drift in either direction, and defines done for #1148:
//
//	THE FIX IS COMPLETE WHEN EVERY brokenToday FIELD IS DELETED
//	AND THE SUITE STILL PASSES.
type nexprVector struct {
	src  string
	want string // the C-family grouping
	// brokenToday: what main emits instead. Empty means correct today, and
	// the vector is a regression guard.
	brokenToday string
	why         string
}

// Fields: r.a..r.d,r.x doubles; r.i,r.j,r.k,r.n integers; r.g bigint; r.s string.
func nexprSyms() map[string]string {
	return map[string]string{
		"r.a": "double", "r.b": "double", "r.c": "double", "r.d": "double", "r.x": "double",
		"r.i": "integer", "r.j": "integer", "r.k": "integer", "r.n": "integer",
		"r.g": "bigint", "r.s": "string",
	}
}

var nexprActionVectors = []nexprVector{
	// --- same-type chains: correct since #1147, must stay ---
	{src: "set r.x = r.a - r.b + r.c", want: "r.a r.b f- r.c f+ cvd /r.x xdef",
		why: "(a-b)+c — equal precedence associates left"},
	{src: "set r.x = r.a / r.b * r.c", want: "r.a r.b fdiv r.c fmul cvd /r.x xdef",
		why: "(a/b)*c"},
	{src: "set r.x = r.i - r.j + r.k", want: "r.i r.j - r.k + cvd /r.x xdef",
		why: "integer chains were always correct: iexpr has no mixed alternatives"},
	{src: "set r.x = r.a - r.b + r.c - r.d", want: "r.a r.b f- r.c f+ r.d f- cvd /r.x xdef",
		why: "((a-b)+c)-d — four operands"},
	{src: "set r.x = r.a / r.b * r.c / r.d", want: "r.a r.b fdiv r.c fmul r.d fdiv cvd /r.x xdef",
		why: "((a/b)*c)/d"},
	{src: "set r.x = r.i - r.j + r.k - r.n", want: "r.i r.j - r.k + r.n - cvd /r.x xdef",
		why: "((i-j)+k)-n"},

	// --- mixed-type chains: correct today by prediction luck, must become guaranteed ---
	{src: "set r.x = r.a + r.b - r.i", want: "r.a r.b f+ r.i f- cvd /r.x xdef",
		why: "(a+b)-i — mixed operand types must not affect grouping"},
	{src: "set r.x = r.i + r.a - r.b", want: "r.i r.a f+ r.b f- cvd /r.x xdef",
		why: "(i+a)-b"},
	{src: "set r.x = r.a / r.b * r.i", want: "r.a r.b fdiv r.i fmul cvd /r.x xdef",
		why: "(a/b)*i"},
	{src: "set r.x = r.a * r.i / r.b", want: "r.a r.i fmul r.b fdiv cvd /r.x xdef",
		why: "(a*i)/b"},

	// --- literals in chains: the kind of the literal must not flip the grouping ---
	{src: "set r.x = r.a + 2 - r.b", want: "r.a 2 f+ r.b f- cvd /r.x xdef",
		why: "(a+2)-b — int literal mid-chain"},
	{src: "set r.x = r.a + 2.0 - r.b", want: "r.a 2.0 f+ r.b f- cvd /r.x xdef",
		brokenToday: "r.a 2.0 r.b f- f+ cvd /r.x xdef",
		why: "a float literal mid-chain flips a BARE chain to a+(2.0-b); an int literal does not"},
	{src: "set r.x = r.i + 2 - r.j", want: "r.i 2 + r.j - cvd /r.x xdef",
		why: "(i+2)-j"},

	// --- construct operand positions: where #1147's fix does not reach ---
	{src: "set r.x = the maximum of (r.a + r.b - r.c) and (0.0)",
		want:        "r.a r.b f+ r.c f- 0.0 fmax cvd /r.x xdef",
		brokenToday: "r.a r.b r.c f- f+ 0.0 fmax cvd /r.x xdef",
		why: "a+(b-c) inside `the maximum of`, though the same chain is correct bare"},
	{src: "set r.x = the minimum of (r.a / r.b * r.c) and (r.d)",
		want: "r.a r.b fdiv r.c fmul r.d fmin cvd /r.x xdef",
		why: "correct today — the instability cuts both ways; pin the lucky case too"},
	{src: "set r.x = the maximum of (r.i - r.j + r.k) and (0)",
		want: "r.i r.j - r.k + 0 max cvd /r.x xdef",
		why: "integer chains stay correct even inside constructs"},
	{src: "set r.x = the maximum of (r.a - r.b + r.c - r.d) and (0.0)",
		want:        "r.a r.b f- r.c f+ r.d f- 0.0 fmax cvd /r.x xdef",
		brokenToday: "r.a r.b r.c r.d f- f+ f- 0.0 fmax cvd /r.x xdef",
		why: "four operands fully right-nested: a-(b+(c-d))"},
	{src: "set r.x = the maximum of (the minimum of (r.a + r.b - r.c) and (r.d)) and (0.0)",
		want:        "r.a r.b f+ r.c f- r.d fmin 0.0 fmax cvd /r.x xdef",
		brokenToday: "r.a r.b r.c f- f+ r.d fmin 0.0 fmax cvd /r.x xdef",
		why: "the mis-grouping survives nesting"},

	// --- unary minus: binds tighter than binary operators ---
	{src: "set r.x = -r.a + r.b - r.c",
		want:        "r.a fnegate r.b f+ r.c f- cvd /r.x xdef",
		brokenToday: "r.a r.b f+ r.c f- fnegate cvd /r.x xdef",
		why: "((-a)+b)-c; today the negation swallows the whole chain: -(a+b-c)"},

	// --- between-class precedence and parentheses: correct, must stay ---
	{src: "set r.x = r.a + r.b * r.c", want: "r.a r.b r.c fmul f+ cvd /r.x xdef",
		why: "multiplicative binds tighter than additive"},
	{src: "set r.x = (r.a - r.b) * r.c", want: "r.a r.b f- r.c fmul cvd /r.x xdef",
		why: "parentheses always win"},

	// --- adjacent rules that share the + token and must not be absorbed ---
	{src: `set r.s = r.s + "x"`, want: `r.s "x" strconcat cvs /r.s xdef`,
		why: "string concatenation is a different operation sharing the token"},
}

var nexprConditionVectors = []nexprVector{
	{src: "r.a + r.b - r.i > r.c", want: "r.a r.b f+ r.i f- r.c f>",
		why: "arithmetic binds tighter than comparison, chain grouped left"},
	{src: "r.i - r.j + r.k > r.n", want: "r.i r.j - r.k + r.n >",
		why: "integer form"},
	{src: "the maximum of (r.a + r.b - r.c) and (0.0) > r.d",
		want:        "r.a r.b f+ r.c f- 0.0 fmax r.d f>",
		brokenToday: "r.a r.b r.c f- f+ 0.0 fmax r.d f>",
		why: "construct operand mis-grouping reaches conditions too"},
	{src: "r.a + r.b - r.c > r.d and r.i - r.j + r.k > r.n",
		want: "r.a r.b f+ r.c f- r.d f> { pop r.i r.j - r.k + r.n > } over if",
		why: "both chains left, comparison tighter than `and`"},
}

func compileVector(t *testing.T, src string, condition bool) (string, error) {
	t.Helper()
	c := NewCompiler()
	c.SetSymbols(nexprSyms())
	if condition {
		return c.CompileCondition(src)
	}
	return c.CompileAction(src)
}

func runNexprVectors(t *testing.T, vecs []nexprVector, condition bool) {
	t.Helper()
	broken := 0
	for _, v := range vecs {
		got, err := compileVector(t, v.src, condition)
		if err != nil {
			t.Errorf("%s: %v", v.src, err)
			continue
		}
		got = strings.TrimSpace(got)
		expect := v.want
		if v.brokenToday != "" {
			broken++
			expect = v.brokenToday
			if got == v.want {
				t.Errorf("%s\n  now emits the CORRECT grouping:\n    %s\n"+
					"  the #1148 fix has reached this vector — delete its brokenToday field",
					v.src, got)
				continue
			}
		}
		if got != expect {
			t.Errorf("%s\n  got  %s\n  want %s\n  (%s)", v.src, got, expect, v.why)
		}
	}
	if broken > 0 {
		t.Logf("%d vector(s) still pinned to the broken grouping — #1148's work is deleting them", broken)
	}
}

func TestNexprActionVectors(t *testing.T)    { runNexprVectors(t, nexprActionVectors, false) }
func TestNexprConditionVectors(t *testing.T) { runNexprVectors(t, nexprConditionVectors, true) }

// The emitter's type policy that nexpr extends: lossy coercion is refused with
// a message naming the cast to write. This must survive the grammar change
// byte-for-byte in spirit — an error, naming both types and both casts.
func TestLossyCoercionStillRefused(t *testing.T) {
	_, err := compileVector(t, "set r.x = r.g + r.a", false)
	if err == nil {
		t.Fatal("bigint + double compiled; the fraction has nowhere to go and this must refuse")
	}
	for _, want := range []string{"bigint", "double", "cast"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal should mention %q: %v", want, err)
		}
	}
}
