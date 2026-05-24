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

// Double-dispatch tests. The sibling-of-#790 gap: `promoteArithType`
// had no Double arm, so any expression promoting to Double (e.g.
// `double_field * double_field`) emitted the integer family
// (`*`, `min`, `+`, `==`). At runtime those ops call IntValue on the
// operand — fine for `*RInteger`, but `*RDouble` returns the
// "no integer value exists for this type" error, the same way #790
// played out for fixed.
//
// Fix: promoteArithType returns TypeDouble when both/either operand
// is double, arithOp/minMaxOp accept a dblOp argument and route to
// f-prefixed ops (f+, f-, fmul, fdiv, fmin, fmax, f==, f!=, f<,
// f<=, f>, f>=).
//
// Each test compiles a small DSL with double-typed operands via
// SetSymbols, then asserts the emitted postfix contains the
// f-prefixed ops AND does NOT contain the integer family it used
// to emit. Reverts to the old promoteArithType (no Double arm) make
// these fail with the bug back in place.

func compileWithDoubleSymbols(t *testing.T, dsl string, kind string) string {
	t.Helper()
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"x":        "double",
		"y":        "double",
		"z":        "double",
		"r":        "double",
		"result.x": "double",
		"result.y": "double",
		"result.z": "double",
		"result.r": "double",
	})
	var (
		out string
		err error
	)
	switch kind {
	case "action":
		out, err = c.CompileAction(dsl)
	case "condition":
		out, err = c.CompileCondition(dsl)
	default:
		t.Fatalf("unknown kind %q", kind)
	}
	if err != nil {
		t.Fatalf("compile %q: %v", dsl, err)
	}
	return out
}

// TestDoubleDispatch_Arithmetic — `set r = x op y` with double
// operands must emit f+ / f- / fmul / fdiv, not the integer family.
func TestDoubleDispatch_Arithmetic(t *testing.T) {
	tests := []struct {
		op         string
		wantOp     string
		notWantOps []string
	}{
		{"+", "f+", []string{" + ", " + cvd "}},
		{"-", "f-", []string{" - cvd ", " - cvi "}},
		{"*", "fmul", []string{" * cvd ", " * cvi "}},
		{"/", "fdiv", []string{" / cvd ", " / cvi "}},
	}
	for _, tc := range tests {
		t.Run(tc.op, func(t *testing.T) {
			dsl := "set r = x " + tc.op + " y"
			out := compileWithDoubleSymbols(t, dsl, "action")
			if !strings.Contains(out, tc.wantOp) {
				t.Errorf("postfix missing %q for double op %q; got: %s", tc.wantOp, tc.op, out)
			}
			for _, banned := range tc.notWantOps {
				if strings.Contains(out, banned) {
					t.Errorf("postfix still contains banned int-op pattern %q for double op %q; got: %s", banned, tc.op, out)
				}
			}
		})
	}
}

// TestDoubleDispatch_MinMax — `the minimum of x and y` and `the
// maximum of x and y` with double operands must emit fmin / fmax,
// not the integer min / max that calls IntValue at runtime.
func TestDoubleDispatch_MinMax(t *testing.T) {
	for _, kw := range []struct{ keyword, want string }{
		{"minimum", "fmin"},
		{"maximum", "fmax"},
	} {
		t.Run(kw.keyword, func(t *testing.T) {
			dsl := "set r = the " + kw.keyword + " of x and y"
			out := compileWithDoubleSymbols(t, dsl, "action")
			if !strings.Contains(out, kw.want) {
				t.Errorf("postfix missing %q for double %s; got: %s", kw.want, kw.keyword, out)
			}
			// The banned integer counterpart, with the cvd convert
			// that follows when the assignment target is double.
			banned := " " + kw.keyword[:3] + " cvd "
			if strings.Contains(out, banned) {
				t.Errorf("postfix still contains banned int-op pattern %q; got: %s", banned, out)
			}
		})
	}
}

// TestDoubleDispatch_OrderingComparisons — `x op y` for >, >=, <, <=
// with double operands must emit f> / f>= / f< / f<=. These go through
// the VisitBoolInt* visitors which use promoteArithType directly, so
// the dispatch table fix applies.
//
// Note on == and !=: bare-name equality (`x == y`) routes through
// VisitBoolNameEq / VisitBoolNameNeq, which use a separate
// `identNumericType` gate that intentionally excludes Double. The
// rationale (per the source comment) is that promoting a Double up
// to Fixed for a cross-type equality would silently snap the double
// onto the 10^-8 grid. So pure-double == still falls through to
// `streq` today. Fixing that cleanly needs a cross-type-safety check
// (let pure-double == use f==, keep fixed↔double cases on streq) —
// tracked as a separate follow-up.
func TestDoubleDispatch_OrderingComparisons(t *testing.T) {
	tests := []struct {
		op     string
		wantOp string
	}{
		{">", "f>"},
		{">=", "f>="},
		{"<", "f<"},
		{"<=", "f<="},
	}
	for _, tc := range tests {
		t.Run(tc.op, func(t *testing.T) {
			dsl := "x " + tc.op + " y"
			out := compileWithDoubleSymbols(t, dsl, "condition")
			if !strings.Contains(out, tc.wantOp) {
				t.Errorf("postfix missing %q for double comparison %q; got: %s", tc.wantOp, tc.op, out)
			}
		})
	}
}

// TestDoubleDispatch_NestedExpression is the staking-style smoke from
// #790's sibling: `the minimum of A and (B * C)` with all three
// operands typed double. Compound expression propagation through
// promoteArithType must keep the double type all the way up so the
// outer fmin and inner fmul both fire.
func TestDoubleDispatch_NestedExpression(t *testing.T) {
	out := compileWithDoubleSymbols(t,
		"set r = the minimum of x and (y * z)", "action")
	for _, want := range []string{"fmul", "fmin", "cvd"} {
		if !strings.Contains(out, want) {
			t.Errorf("postfix missing %q in nested double expression; got: %s", want, out)
		}
	}
	for _, banned := range []string{" * min ", " * cvi ", " min cvi "} {
		if strings.Contains(out, banned) {
			t.Errorf("postfix still contains banned int-op pattern %q; got: %s", banned, out)
		}
	}
}

// TestDoubleDispatch_IntegerStillUsesIntOps confirms the dispatch
// table didn't accidentally route Integer through the double path.
// `set r = x + y` where both are *integer* must still emit `+`, not
// `f+`. Catches a reversed-priority bug in promoteArithType.
func TestDoubleDispatch_IntegerStillUsesIntOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"x": "integer",
		"y": "integer",
		"r": "integer",
	})
	out, err := c.CompileAction("set r = x + y")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(out, "+ /") && !strings.Contains(out, "+ ") {
		t.Errorf("integer postfix missing bare `+`; got: %s", out)
	}
	if strings.Contains(out, "f+") || strings.Contains(out, "fp+") {
		t.Errorf("integer postfix unexpectedly contains fp/double op; got: %s", out)
	}
}
