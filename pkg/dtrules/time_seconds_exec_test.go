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

package dtrules_test

import (
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Time arithmetic below a day (#1232): `seconds from d1 to d2`, `minutes
// from d1 to d2`, and `d + N seconds` / `d - N minutes` with their
// `add … to` / `subtract … from` spellings. Each case is compiled from EL and
// executed, and the answers are fixed by hand, not by what the code returns.
// Differences are elapsed time between two instants: the zone a date carries
// does not change them.

type timeRig struct {
	t     *testing.T
	sess  dtrules.Session
	state *interpreter.DTState
	root  dtrules.Entity
}

func newTimeRig(t *testing.T) *timeRig {
	t.Helper()
	rs := session.NewRuleSet("t1232")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	name := dtrules.GetRName("clock")
	ref, err := ef.FindCreateRefEntity(true, name)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"d1", "d2", "out"} {
		ref.AddAttribute(dtrules.GetRName(f), "", nil, true, true, dtrules.TypeDate, "", "", "", "")
	}
	ref.AddAttribute(dtrules.GetRName("n"), "", dtrules.GetRIntegerValueFromInt(0), true, true, dtrules.TypeInteger, "", "", "", "")
	ref.AddAttribute(dtrules.GetRName("idle"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")
	root, err := ef.CreateEntity(sess, name)
	if err != nil {
		t.Fatal(err)
	}
	return &timeRig{t: t, sess: sess, state: sess.GetState().(*interpreter.DTState), root: root}
}

func (r *timeRig) exec(action string) {
	r.t.Helper()
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"d1": "date", "d2": "date", "out": "date", "n": "integer", "idle": "boolean"})
	pf, err := elc.CompileAction(action)
	if err != nil {
		r.t.Fatalf("%q compile: %v", action, err)
	}
	obj, err := r.sess.Compile(pf)
	if err != nil {
		r.t.Fatalf("%q assemble %q: %v", action, pf, err)
	}
	r.state.EntityPush(r.root)
	err = obj.Execute(r.state)
	r.state.EntityPop()
	if err != nil {
		r.t.Fatalf("%q execute %q: %v", action, pf, err)
	}
}

func (r *timeRig) set(field, rfc3339 string) {
	r.t.Helper()
	tm, err := time.Parse(time.RFC3339Nano, rfc3339)
	if err != nil {
		r.t.Fatal(err)
	}
	r.root.Put(dtrules.GetRName(field), dtrules.GetRTime(tm))
}

func (r *timeRig) int(field string) int64 {
	r.t.Helper()
	v, err := r.root.Get(dtrules.GetRName(field))
	if err != nil {
		r.t.Fatal(err)
	}
	n, err := v.LongValue()
	if err != nil {
		r.t.Fatal(err)
	}
	return n
}

func (r *timeRig) date(field string) time.Time {
	r.t.Helper()
	v, err := r.root.Get(dtrules.GetRName(field))
	if err != nil {
		r.t.Fatal(err)
	}
	tm, err := v.TimeValue()
	if err != nil {
		r.t.Fatal(err)
	}
	return tm
}

func TestSecondsAndMinutesFromExecution(t *testing.T) {
	r := newTimeRig(t)
	cases := []struct {
		name, d1, d2  string
		secs, minutes int64
	}{
		{"same day", "2026-09-21T10:00:00Z", "2026-09-21T10:02:00Z", 120, 2},
		{"across midnight", "2026-01-01T23:59:30Z", "2026-01-02T00:00:15Z", 45, 0},
		{"across a month end", "2026-01-31T23:00:00Z", "2026-02-01T01:00:00Z", 7200, 120},
		{"across a year end", "2025-12-31T23:59:59Z", "2026-01-01T00:00:01Z", 2, 0},
		{"across a leap day", "2024-02-28T12:00:00Z", "2024-03-01T12:00:00Z", 172800, 2880},
		{"pure dates, one day", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", 86400, 1440},
		{"backwards is negative", "2026-09-21T10:02:00Z", "2026-09-21T10:00:00Z", -120, -2},
		{"partial minute truncates", "2026-09-21T10:00:00Z", "2026-09-21T10:01:59Z", 119, 1},
		{"partial minute backwards truncates toward zero", "2026-09-21T10:01:59Z", "2026-09-21T10:00:00Z", -119, -1},
		{"sub-second truncates", "2026-09-21T10:00:00.900Z", "2026-09-21T10:00:02.100Z", 1, 0},
		// The same instants written in two zones: 16:06:30 at -05:00 is
		// 21:06:30Z, one minute after 21:05:30Z.
		{"offsets do not matter", "2026-04-17T21:05:30Z", "2026-04-17T16:06:30-05:00", 60, 1},
		{"centuries apart", "1900-01-01T00:00:00Z", "2300-01-01T00:00:00Z", 12622780800, 210379680},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r.t = t
			r.set("d1", c.d1)
			r.set("d2", c.d2)
			r.exec("set n = seconds from d1 to d2")
			if got := r.int("n"); got != c.secs {
				t.Errorf("seconds from = %d, want %d", got, c.secs)
			}
			r.exec("set n = minutes from d1 to d2")
			if got := r.int("n"); got != c.minutes {
				t.Errorf("minutes from = %d, want %d", got, c.minutes)
			}
		})
	}
}

// The scheduler shape from the issue: "idle for more than 120 s".
func TestSecondsFromInACondition(t *testing.T) {
	r := newTimeRig(t)
	r.set("d1", "2026-09-21T23:58:30Z")
	r.set("d2", "2026-09-22T00:00:31Z")
	r.exec("set idle = seconds from d1 to d2 > 120")
	v, _ := r.root.Get(dtrules.GetRName("idle"))
	if b, err := v.BooleanValue(); err != nil || !b {
		t.Errorf("121 s idle across midnight read as not > 120")
	}
}

func TestPlusMinusSecondsAndMinutesExecution(t *testing.T) {
	r := newTimeRig(t)
	cases := []struct{ name, d1, action, want string }{
		{"plus seconds across a year end", "2026-12-31T23:59:30Z", "set out = d1 + 45 seconds", "2027-01-01T00:00:15Z"},
		{"one second", "2026-09-21T10:00:00Z", "set out = d1 + 1 second", "2026-09-21T10:00:01Z"},
		{"minus minutes across a month end", "2026-03-01T00:30:00Z", "set out = d1 - 90 minutes", "2026-02-28T23:00:00Z"},
		{"minus minutes across a leap day", "2024-03-01T00:30:00Z", "set out = d1 - 90 minutes", "2024-02-29T23:00:00Z"},
		{"plus minutes across midnight", "2026-09-21T23:50:00Z", "set out = d1 + 15 minutes", "2026-09-22T00:05:00Z"},
		{"minus seconds", "2026-01-01T00:00:10Z", "set out = d1 - 11 seconds", "2025-12-31T23:59:59Z"},
		{"negative count", "2026-09-21T10:00:00Z", "set out = d1 + -60 seconds", "2026-09-21T09:59:00Z"},
		{"add … to expression", "2026-09-21T10:00:00Z", "set out = add 30 seconds to d1", "2026-09-21T10:00:30Z"},
		{"subtract … from expression", "2026-09-21T10:00:00Z", "set out = subtract 2 minutes from d1", "2026-09-21T09:58:00Z"},
		{"a count from a field", "2026-09-21T10:00:00Z", "set n = 300; set out = d1 + n seconds", "2026-09-21T10:05:00Z"},
		{"the zone a date carries is kept", "2026-09-21T10:00:00-05:00", "set out = d1 + 60 minutes", "2026-09-21T11:00:00-05:00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r.t = t
			r.set("d1", c.d1)
			r.exec(c.action)
			// Compared as text so the zone is checked, not only the instant.
			if got := r.date("out").Format(time.RFC3339); got != c.want {
				t.Errorf("%s = %s, want %s", c.action, got, c.want)
			}
		})
	}
}

// `add N seconds to d` / `subtract N minutes from d` as statements change the
// field in place, like their day forms.
func TestAddSubtractSecondsStatements(t *testing.T) {
	r := newTimeRig(t)
	r.set("d1", "2026-12-31T23:59:59Z")
	r.exec("add 2 seconds to d1")
	if got, want := r.date("d1"), time.Date(2027, 1, 1, 0, 0, 1, 0, time.UTC); !got.Equal(want) {
		t.Errorf("add 2 seconds to d1: %s, want %s", got, want)
	}
	r.exec("subtract 1 minute from d1")
	if got, want := r.date("d1"), time.Date(2026, 12, 31, 23, 59, 1, 0, time.UTC); !got.Equal(want) {
		t.Errorf("subtract 1 minute from d1: %s, want %s", got, want)
	}
}

// A date round-trips: seconds from d to (d + N seconds) is N.
func TestSecondsRoundTrip(t *testing.T) {
	r := newTimeRig(t)
	r.set("d1", "2026-02-28T23:59:59Z")
	r.exec("set out = d1 + 86401 seconds; set n = seconds from d1 to out")
	if got := r.int("n"); got != 86401 {
		t.Errorf("round trip = %d, want 86401", got)
	}
	if got, want := r.date("out"), time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("out = %s, want %s", got, want)
	}
}

// "Now" is `current date in zone "UTC"`; `current date` is today at
// midnight. The docs point schedulers at the first, so it must measure
// elapsed seconds against the clock.
func TestSecondsFromNow(t *testing.T) {
	r := newTimeRig(t)
	r.root.Put(dtrules.GetRName("d1"), dtrules.GetRTime(time.Now().Add(-300*time.Second)))
	r.exec(`set n = seconds from d1 to current date in zone "UTC"`)
	if got := r.int("n"); got < 299 || got > 330 {
		t.Errorf("seconds since a stamp 300 s ago = %d, want about 300", got)
	}
}
