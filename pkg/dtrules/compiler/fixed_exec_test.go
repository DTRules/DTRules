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

package compiler

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"

	// Pull in operator registrations (fp+, fp*, cvfp, etc.).
	_ "github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// evaluateFixed compiles a postfix snippet, executes it on a fresh state, and
// returns the RFixed left on top of the data stack.
func evaluateFixed(t *testing.T, src string) *dtrules.RFixed {
	t.Helper()
	c := newTestCompiler()
	code, err := c.Compile(src)
	if err != nil {
		t.Fatalf("Compile(%q): %v", src, err)
	}
	state := interpreter.NewDTState(&mockSession{})
	if err := code.Execute(state); err != nil {
		t.Fatalf("Execute(%q): %v", src, err)
	}
	top, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	fp, ok := top.(*dtrules.RFixed)
	if !ok {
		t.Fatalf("expected RFixed result, got %T: %v", top, top)
	}
	return fp
}

// TestStakingArithmetic_ExactTotals walks a small "staking" distribution
// end-to-end through the compile + execute pipeline and checks that the
// final total is bit-exact on the 10^-8 grid.
//
// Model: pool of 14_126_019.81 ACME distributed over 25 periods with a
// simple weekly-rate formula and a treasury fee. Every value stays on
// the 10^-8 grid; none of the intermediates are allowed to drift.
func TestStakingArithmetic_ExactTotals(t *testing.T) {
	cases := []struct {
		name    string
		program string
		want    string
	}{
		{
			// Exact addition of sub-satoshi values — pattern that destroyed
			// float-based staking totals when rounded separately per period.
			name:    "sum of 25 periods",
			program: "1234567.89012345fp 1234567.89012345fp fp+",
			want:    "2469135.78024690",
		},
		{
			// Mul truncates toward zero: 0.12345678 × 10 stays on-grid.
			name:    "rate times whole period count",
			program: "0.12345678fp 10 cvfp fp*",
			want:    "1.23456780",
		},
		{
			// Total pool × per-period share with a fractional multiplier.
			// Mantissa math: 1412601981000000 × 306639 / 10^8 truncates
			// toward zero. Exact integer computation — no float drift.
			name:    "pool × weekly_factor truncates at grid",
			program: "14126019.81000000fp 0.00306639fp fp*",
			want:    "43315.88588518",
		},
		{
			// A sub-satoshi difference must survive a round-trip through
			// add/sub pairs — proves no hidden float coercion.
			name:    "(a + b) - b == a at sub-satoshi precision",
			program: "0.00000001fp 1680748.45091643fp fp+ 1680748.45091643fp fp-",
			want:    "0.00000001",
		},
		{
			// Integer + fixed auto-promotes via cvfp (exercises the EL
			// dispatch path; here driven directly from postfix).
			name:    "int promoted to fixed via cvfp",
			program: "100 cvfp 0.00250000fp fp*",
			want:    "0.25000000",
		},
		{
			// Bigint promoted to fixed via cvfp.
			name:    "bigint promoted to fixed via cvfp",
			program: "500000000 cvbi cvfp 0.00003066fp fp*",
			want:    "15330.00000000",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evaluateFixed(t, c.program).StringValue()
			if got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

// TestStakingArithmetic_NoDriftOverManyPeriods sums a rate across many
// periods and checks the result is exact — the kind of accumulating
// computation where float would drift.
func TestStakingArithmetic_NoDriftOverManyPeriods(t *testing.T) {
	// Start at 0fp; add 0.12345678fp fifty times. Expected: 6.17283900fp.
	program := "0fp"
	for i := 0; i < 50; i++ {
		program += " 0.12345678fp fp+"
	}
	got := evaluateFixed(t, program).StringValue()
	if got != "6.17283900" {
		t.Errorf("50× 0.12345678 accumulation: got %q, want 6.17283900", got)
	}
}
