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

package session

import (
	"testing"
	"time"
)

// The ISO 8601 local form -- a T and no offset -- is what most JSON
// producers and spreadsheets emit, and the parser refused it (#1275). It
// reads exactly as the space-separated form does: UTC, fractional seconds
// allowed.
func TestDateParserNaiveISOTimestamp(t *testing.T) {
	p := NewDateParser()
	cases := []struct {
		in   string
		want time.Time
	}{
		{"2020-01-01T10:00:00", time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)},
		{"2026-12-31T23:59:59", time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)},
		{"2026-04-17T21:05:30.25", time.Date(2026, 4, 17, 21, 5, 30, 250000000, time.UTC)},
		{"2026-04-17T21:05:30.123456789", time.Date(2026, 4, 17, 21, 5, 30, 123456789, time.UTC)},
	}
	for _, c := range cases {
		got, err := p.GetDate(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		if !got.Equal(c.want) || got.Location() != time.UTC {
			t.Errorf("%s = %s, want %s", c.in, got, c.want)
		}
		// The same moment written with a space parses to the same value.
		space, err := p.GetDate(c.in[:10] + " " + c.in[11:])
		if err != nil || !space.Equal(got) {
			t.Errorf("%s: space form gives %s (%v)", c.in, space, err)
		}
	}
	// Offsets still win where they are given, and nonsense is still refused.
	if got, _ := p.GetDate("2020-01-01T10:00:00-05:00"); !got.Equal(time.Date(2020, 1, 1, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("offset form = %s", got)
	}
	for _, bad := range []string{"2020-01-01T25:00:00", "2020-13-01T10:00:00", "2020-01-01T10"} {
		if got, err := p.GetDate(bad); err == nil {
			t.Errorf("%s parsed as %s", bad, got)
		}
	}
}
