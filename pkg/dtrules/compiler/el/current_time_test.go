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

// `current time` is the current instant (#1266), a phrase like `current
// date` and `dexpr` alternative of its own (`dateCurrentTime`), so the
// compiler knows it is a date. These rows pin the postfix for every form an
// author writes.
func TestCurrentTimeCompiles(t *testing.T) {
	symbols := map[string]string{
		"clock.out": "date", "clock.d1": "date", "clock.n": "integer",
		"clock.b": "boolean", "clock.s": "string",
	}
	cases := []struct{ name, action, want string }{
		{"assignment", "set clock.out = current time", "now cvdate /clock.out xdef"},
		{"calendar arithmetic", "set clock.out = current time + 3 days",
			"now 3 adddays cvdate /clock.out xdef"},
		// A date comparison (`d>`), not the generic `>`: the grammar types
		// `current time`, so the compiler knows both sides are dates.
		{"comparison", "set clock.b = current time > clock.d1", "now clock.d1 d> cvb /clock.b xdef"},
		{"date part", "set clock.n = get yearof current time", "now yearof cvi /clock.n xdef"},
		// The scheduler case #1266 was raised for.
		{"elapsed seconds", "set clock.n = seconds from clock.d1 to current time",
			"clock.d1 now secondsbetween cvi /clock.n xdef"},
		// `in zone` on an instant is the generic rewrap: same instant, new
		// zone. On `today` it means midnight in that zone instead (#1273).
		{"in zone rewraps", `set clock.out = current time in zone "America/Chicago"`,
			`now "America/Chicago" dateinzone cvdate /clock.out xdef`},
		// The replacement for `get current timestamp`: cvs stringifies a
		// date carrying a time as RFC3339Nano.
		{"as a string", "set clock.s = current time", "now cvs /clock.s xdef"},
		{"current date is still midnight", "set clock.out = current date",
			"today cvdate /clock.out xdef"},
		{"case is not significant", "set clock.out = CURRENT TIME", "now cvdate /clock.out xdef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			comp := NewCompiler()
			comp.SetSymbols(symbols)
			got, err := comp.CompileAction(c.action)
			if err != nil {
				t.Fatalf("%q: %v", c.action, err)
			}
			if got != c.want {
				t.Errorf("%q\n got %q\nwant %q", c.action, got, c.want)
			}
		})
	}
}

// A phrase reserves no identifier: `current` on its own is still an ordinary
// name, and a field called `time` or `current.time` keeps working. This is
// why `current time` was chosen over a `now` keyword, which would have taken
// the name away from any rule set already using it.
func TestCurrentTimeReservesNoName(t *testing.T) {
	cases := []struct{ name, action, want string }{
		{"a field named current.time", "set clock.out = current.time",
			"current.time cvdate /clock.out xdef"},
		{"an entity named current", "set clock.out = current.updated",
			"current.updated cvdate /clock.out xdef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			comp := NewCompiler()
			comp.SetSymbols(map[string]string{
				"clock.out": "date", "current.time": "date", "current.updated": "date",
			})
			got, err := comp.CompileAction(c.action)
			if err != nil {
				t.Fatalf("%q: %v", c.action, err)
			}
			if got != c.want {
				t.Errorf("%q\n got %q\nwant %q", c.action, got, c.want)
			}
		})
	}
}

// The grammar types `current time` as a date, so a bare number cannot be
// added to it. Before #1266 the only spelling for the instant was
// `current date in zone "UTC"`, where `in zone` changed the meaning.
func TestCurrentTimeRejectsNumericArithmetic(t *testing.T) {
	comp := NewCompiler()
	comp.SetSymbols(map[string]string{"clock.n": "integer", "clock.out": "date"})
	if got, err := comp.CompileAction("set clock.n = current time + 1"); err == nil {
		t.Errorf("`current time + 1` compiled to %q; it should be a compile error", got)
	}
	got, err := comp.CompileAction("set clock.out = current time + 1 days")
	if err != nil {
		t.Fatalf("`current time + 1 days`: %v", err)
	}
	if got != "now 1 adddays cvdate /clock.out xdef" {
		t.Errorf("`current time + 1 days` compiled to %q", got)
	}
}

// `get current timestamp` is gone (#1266): it emitted a bare `gettimestamp`,
// an operator that *pops* a date and formats it, so it underflowed the stack
// or stringified whatever was under it.
func TestGetCurrentTimestampIsRemoved(t *testing.T) {
	comp := NewCompiler()
	comp.SetSymbols(map[string]string{"clock.s": "string"})
	got, err := comp.CompileAction("set clock.s = get current timestamp")
	if err == nil {
		t.Fatalf("expected a compile error, got %q", got)
	}
	if !strings.Contains(err.Error(), "current time") {
		t.Errorf("the error should name the replacement, got: %v", err)
	}
}
