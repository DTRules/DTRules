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
)

// `days from d1 to d2` counts calendar days (#1265): the calendar date of d2
// minus the calendar date of d1, both read in d1's zone. The time of day does
// not enter into it, and neither does a daylight-saving change. Each case is
// compiled from EL and executed; the answers are fixed by hand.
func TestDaysFromCountsCalendarDays(t *testing.T) {
	chicago := mustZone(t, "America/Chicago")
	london := mustZone(t, "Europe/London")
	utc := time.UTC

	cases := []struct {
		name   string
		d1, d2 time.Time
		want   int64
	}{
		// Across daylight-saving changes: 23-, 47- and 25-hour spans.
		{"chicago spring forward, 1 day",
			time.Date(2026, 3, 8, 0, 0, 0, 0, chicago), time.Date(2026, 3, 9, 0, 0, 0, 0, chicago), 1},
		{"chicago spring forward, 2 days",
			time.Date(2026, 3, 8, 0, 0, 0, 0, chicago), time.Date(2026, 3, 10, 0, 0, 0, 0, chicago), 2},
		{"chicago fall back, 1 day",
			time.Date(2026, 11, 1, 0, 0, 0, 0, chicago), time.Date(2026, 11, 2, 0, 0, 0, 0, chicago), 1},
		{"london spring forward, 1 day",
			time.Date(2026, 3, 29, 0, 0, 0, 0, london), time.Date(2026, 3, 30, 0, 0, 0, 0, london), 1},
		// The time of day does not count: two hours across midnight is a day,
		// twenty-two hours inside one date is none.
		{"23:00 to 01:00 next day",
			time.Date(2026, 3, 8, 23, 0, 0, 0, utc), time.Date(2026, 3, 9, 1, 0, 0, 0, utc), 1},
		{"01:00 to 23:00 same day",
			time.Date(2026, 3, 8, 1, 0, 0, 0, utc), time.Date(2026, 3, 8, 23, 0, 0, 0, utc), 0},
		{"backwards across midnight",
			time.Date(2026, 3, 9, 1, 0, 0, 0, utc), time.Date(2026, 3, 8, 23, 0, 0, 0, utc), -1},
		{"noon to 06:00 two dates on",
			time.Date(2026, 3, 8, 12, 0, 0, 0, utc), time.Date(2026, 3, 10, 6, 0, 0, 0, utc), 2},
		// Different zones: d2 is read in d1's zone. d2 here is 19:00 on
		// Mar 8 in Chicago, earlier than d1, on the same Chicago date. (Each
		// date in its own zone would say 1 day, with d2 before d1.)
		{"mixed zones, d2 earlier on d1's date",
			time.Date(2026, 3, 8, 23, 0, 0, 0, chicago), time.Date(2026, 3, 9, 1, 0, 0, 0, utc), 0},
		// d1 is 20:00 Mar 8 in Chicago (01:00 Mar 9 UTC); d2 is 01:00 Mar 9
		// in Chicago. One Chicago midnight lies between them, none in UTC.
		{"mixed zones, one midnight in d1's zone",
			time.Date(2026, 3, 8, 20, 0, 0, 0, chicago), time.Date(2026, 3, 9, 6, 0, 0, 0, utc), 1},
		// Exact across the whole date range, beyond time.Duration's ~292 years.
		{"year 1 to year 9999",
			time.Date(1, 1, 1, 0, 0, 0, 0, utc), time.Date(9999, 12, 31, 0, 0, 0, 0, utc), 3652058},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newTimeRig(t)
			r.put("d1", c.d1)
			r.put("d2", c.d2)
			r.exec("set n = days from d1 to d2")
			if got := r.int("n"); got != c.want {
				t.Errorf("days from %s to %s = %d, want %d", c.d1, c.d2, got, c.want)
			}
		})
	}
}

// `days from d to d + N days` is N: the count is the inverse of calendar day
// arithmetic, including across a daylight-saving change.
func TestDaysFromInvertsAddDays(t *testing.T) {
	chicago := mustZone(t, "America/Chicago")
	for _, start := range []time.Time{
		time.Date(2026, 3, 6, 12, 30, 0, 0, chicago),
		time.Date(2026, 10, 30, 23, 0, 0, 0, chicago),
	} {
		for _, n := range []string{"-3", "-1", "0", "1", "2", "3", "5"} {
			r := newTimeRig(t)
			r.put("d1", start)
			r.exec("set out = d1 + " + n + " days")
			r.exec("set n = days from d1 to out")
			want := map[string]int64{"-3": -3, "-1": -1, "0": 0, "1": 1, "2": 2, "3": 3, "5": 5}[n]
			if got := r.int("n"); got != want {
				t.Errorf("start %s: days from d1 to d1 + %s days = %d (out %s)", start, n, got, r.date("out"))
			}
		}
	}
}

func (r *timeRig) put(field string, tm time.Time) {
	r.root.Put(dtrules.GetRName(field), dtrules.GetRTime(tm))
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("zone %s: %v", name, err)
	}
	return loc
}
