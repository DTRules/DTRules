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

package collect

import "testing"

func TestFlag(t *testing.T) {
	cases := []struct {
		value, low, high, want string
	}{
		{"1.0", "0.7", "1.3", ""},   // in range
		{"0.7", "0.7", "1.3", ""},   // on low bound (inclusive)
		{"1.3", "0.7", "1.3", ""},   // on high bound (inclusive)
		{"1.85", "0.7", "1.3", "H"}, // high
		{"0.5", "0.7", "1.3", "L"},  // low
		{"250", "", "200", "H"},     // one-sided high (e.g. cholesterol < 200)
		{"40", "60", "", "L"},       // one-sided low (e.g. eGFR > 60)
		{"5", "", "", ""},           // no bounds
		{"abc", "0.7", "1.3", ""},   // non-numeric value -> no flag
		{"1.0", "x", "y", ""},       // non-numeric bounds -> no flag
	}
	for _, c := range cases {
		if got := Flag(c.value, c.low, c.high); got != c.want {
			t.Errorf("Flag(%q,%q,%q)=%q want %q", c.value, c.low, c.high, got, c.want)
		}
	}
}

func TestRangeText(t *testing.T) {
	cases := []struct {
		low, high, units, want string
	}{
		{"0.7", "1.3", "mg/dL", "0.7–1.3 mg/dL"},
		{"0.7", "1.3", "", "0.7–1.3"},
		{"", "200", "mg/dL", "≤ 200 mg/dL"},
		{"60", "", "", "≥ 60"},
		{"", "", "", ""},
	}
	for _, c := range cases {
		if got := RangeText(c.low, c.high, c.units); got != c.want {
			t.Errorf("RangeText(%q,%q,%q)=%q want %q", c.low, c.high, c.units, got, c.want)
		}
	}
}
