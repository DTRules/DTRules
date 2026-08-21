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

package operators

import (
	"math"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// TestRoundToScale is the reproducer for the opRoundTo scale bug that was
// latent until the PostfixEmitter started routing `<X> rounded to N
// decimal places` rules through this op: the old scale formula was
// `10*places` (multiplication) where it should be `10^places`
// (exponentiation), so `3.141 rounded to 2` returned 3.15 instead of
// 3.14 — silently wrong. The places=0 branch also divided by zero.
//
// These cases pin the arithmetic on well-known values where the buggy
// output differs from the correct output, guarding against regression.
func TestRoundToScale(t *testing.T) {
	cases := []struct {
		name     string
		number   float64
		places   int64
		boundary float64
		want     float64
	}{
		// 3.141 rounded to 2 decimal places → 3.14 (buggy: 3.15)
		{"pi to 2 places", 3.141, 2, 0.5, 3.14},
		// 3.145 rounded to 2 with half-up boundary → 3.15
		{"pi-5 to 2 places", 3.145, 2, 0.5, 3.15},
		// 0.01234 rounded to 3 places → 0.012 (buggy: 0.012 coincidentally OK
		// for this particular value, pick one that distinguishes)
		{"0.01549 to 3 places", 0.01549, 3, 0.5, 0.015},
		{"0.01551 to 3 places", 0.01551, 3, 0.5, 0.016},
		// places=0 is "round to integer". Buggy version divided by zero.
		{"1.6 to 0 places half-up", 1.6, 0, 0.5, 2},
		{"1.4 to 0 places half-up", 1.4, 0, 0.5, 1},
		// Negative places: "round to nearest 10^|places|".
		// 1234 rounded to -2 → 1200 (buggy: scale = 20, gives 1240).
		{"1234 to -2 places", 1234, -2, 0.5, 1200},
		{"1274 to -2 places", 1274, -2, 0.5, 1300},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := newTestState()
			if err := state.DataPush(dtrules.GetRDoubleValue(c.number)); err != nil {
				t.Fatal(err)
			}
			if err := state.DataPush(dtrules.GetRIntegerValue(c.places)); err != nil {
				t.Fatal(err)
			}
			if err := state.DataPush(dtrules.GetRDoubleValue(c.boundary)); err != nil {
				t.Fatal(err)
			}
			op, ok := Get(dtrules.GetRName("roundto"))
			if !ok {
				t.Fatal("roundto not registered")
			}
			if err := op.Execute(state); err != nil {
				t.Fatalf("roundto: %v", err)
			}
			top, err := state.DataPop()
			if err != nil {
				t.Fatal(err)
			}
			got, err := top.DoubleValue()
			if err != nil {
				t.Fatal(err)
			}
			// Allow a small epsilon for float arithmetic — the scale
			// multiply / divide can introduce ULP noise even after rounding.
			if math.Abs(got-c.want) > 1e-9 {
				t.Errorf("roundto(%v, %d, %v) = %v, want %v",
					c.number, c.places, c.boundary, got, c.want)
			}
		})
	}
}

// TestRoundToRejectsGarbage guards the hardening checks added in the
// review follow-up: NaN, Inf, and extreme `places` values used to
// produce garbage output (int(NaN) is implementation-defined; 10^16
// loses integer precision in float64). These inputs now error loudly
// rather than silently returning wrong numbers.
func TestRoundToRejectsGarbage(t *testing.T) {
	cases := []struct {
		name              string
		number, boundary  float64
		places            int64
		expectErrContains string
	}{
		{"NaN number", math.NaN(), 0.5, 2, "NaN or Inf"},
		{"+Inf number", math.Inf(1), 0.5, 2, "NaN or Inf"},
		{"-Inf number", math.Inf(-1), 0.5, 2, "NaN or Inf"},
		{"NaN boundary", 1.0, math.NaN(), 2, "boundary"},
		{"Inf boundary", 1.0, math.Inf(1), 2, "boundary"},
		{"places too large", 1.0, 0.5, 16, "out of range"},
		{"places too negative", 1.0, 0.5, -16, "out of range"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.GetRDoubleValue(c.number))
			state.DataPush(dtrules.GetRIntegerValue(c.places))
			state.DataPush(dtrules.GetRDoubleValue(c.boundary))
			op, _ := Get(dtrules.GetRName("roundto"))
			err := op.Execute(state)
			if err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
			if !strings.Contains(err.Error(), c.expectErrContains) {
				t.Errorf("error should mention %q: %v", c.expectErrContains, err)
			}
		})
	}
}

// TestRoundToBoundaryMaxPlaces — 15 places is accepted (just below the
// hardening threshold) so legitimate high-precision rounding still works.
func TestRoundToBoundaryMaxPlaces(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRDoubleValue(1.123456789012345))
	state.DataPush(dtrules.GetRIntegerValue(15))
	state.DataPush(dtrules.GetRDoubleValue(0.5))
	op, _ := Get(dtrules.GetRName("roundto"))
	if err := op.Execute(state); err != nil {
		t.Fatalf("places=15 should be accepted: %v", err)
	}
}
