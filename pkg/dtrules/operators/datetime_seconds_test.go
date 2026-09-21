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

package operators

import (
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Operator-level pins for #1232 that the EL surface cannot reach easily:
// daylight-saving changes in a named zone, and spans beyond the ~292 years a
// time.Duration can hold.

func pushInstant(t *testing.T, s dtrules.State, tm time.Time) {
	t.Helper()
	if err := s.DataPush(dtrules.GetRTime(tm)); err != nil {
		t.Fatal(err)
	}
}

func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := dtrules.ResolveZone("America/Chicago")
	if err != nil {
		t.Fatalf("resolve zone: %v", err)
	}
	return loc
}

// On 2026-03-08 Chicago springs forward at 02:00 CST to 03:00 CDT. One hour
// of elapsed time from 01:30 CST is 03:30 CDT on the wall clock, and the
// result stays in Chicago's zone.
func TestAddSecondsAcrossSpringForward(t *testing.T) {
	loc := chicago(t)
	start := time.Date(2026, 3, 8, 1, 30, 0, 0, loc)
	top := runDateOp(t, "addseconds", func(s dtrules.State) {
		pushInstant(t, s, start)
		s.DataPush(dtrules.GetRIntegerValue(3600))
	})
	got, err := top.TimeValue()
	if err != nil {
		t.Fatal(err)
	}
	if got.Location().String() != "America/Chicago" {
		t.Errorf("zone = %s, want America/Chicago", got.Location())
	}
	if h, m := got.Hour(), got.Minute(); h != 3 || m != 30 {
		t.Errorf("wall clock = %02d:%02d, want 03:30", h, m)
	}
	if !got.Equal(start.Add(time.Hour)) {
		t.Errorf("instant = %s, want one elapsed hour after %s", got, start)
	}
}

// From 00:00 to 03:00 local on the spring-forward day is two elapsed hours,
// not three: the difference counts time that passes, not wall-clock digits.
// On the fall-back day (2026-11-01) the same wall-clock span is four hours.
func TestSecondsBetweenAcrossDST(t *testing.T) {
	loc := chicago(t)
	cases := []struct {
		name   string
		d1, d2 time.Time
		want   int64
	}{
		{"spring forward", time.Date(2026, 3, 8, 0, 0, 0, 0, loc), time.Date(2026, 3, 8, 3, 0, 0, 0, loc), 7200},
		{"fall back", time.Date(2026, 11, 1, 0, 0, 0, 0, loc), time.Date(2026, 11, 1, 3, 0, 0, 0, loc), 14400},
		{"a zone against UTC", time.Date(2026, 9, 21, 10, 0, 0, 0, loc), time.Date(2026, 9, 21, 15, 1, 0, 0, time.UTC), 60},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			top := runDateOp(t, "secondsbetween", func(s dtrules.State) {
				pushInstant(t, s, c.d1)
				pushInstant(t, s, c.d2)
			})
			if v, _ := top.LongValue(); v != c.want {
				t.Errorf("secondsbetween = %d, want %d", v, c.want)
			}
		})
	}
}

// time.Duration saturates at about 292 years; the operators do not.
func TestSecondsBeyondDurationRange(t *testing.T) {
	d1 := time.Date(1600, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2400, 1, 1, 0, 0, 0, 0, time.UTC)
	// 800 Gregorian years are exactly 2 cycles of 400 years, 146097 days each.
	const want = int64(2*146097) * 86400
	top := runDateOp(t, "secondsbetween", func(s dtrules.State) {
		pushInstant(t, s, d1)
		pushInstant(t, s, d2)
	})
	if v, _ := top.LongValue(); v != want {
		t.Errorf("secondsbetween = %d, want %d", v, want)
	}
	top = runDateOp(t, "addseconds", func(s dtrules.State) {
		pushInstant(t, s, d1)
		s.DataPush(dtrules.GetRIntegerValue(want))
	})
	if got, _ := top.TimeValue(); !got.Equal(d2) {
		t.Errorf("addseconds = %s, want %s", got, d2)
	}
	top = runDateOp(t, "minutesbetween", func(s dtrules.State) {
		pushInstant(t, s, d2)
		pushInstant(t, s, d1)
	})
	if v, _ := top.LongValue(); v != -want/60 {
		t.Errorf("minutesbetween = %d, want %d", v, -want/60)
	}
}

// A minute count too large to turn into seconds is an error, not a wrapped
// date.
func TestAddMinutesOverflow(t *testing.T) {
	state := newTestState()
	pushInstant(t, state, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	state.DataPush(dtrules.GetRIntegerValue(1 << 62))
	o, _ := Get(dtrules.GetRName("addminutes"))
	if err := o.Execute(state); err == nil {
		t.Errorf("addminutes accepted 2^62 minutes")
	}
}
