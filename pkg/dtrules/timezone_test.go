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

package dtrules

import (
	"strings"
	"testing"
	"time"
)

// Phase 2 of #743: ResolveZone parses both IANA names and ISO 8601 fixed
// offsets into a *time.Location. Tests below pin the resolution chain.

func TestResolveZone_IANAStandardNames(t *testing.T) {
	cases := []string{
		"UTC",
		"America/New_York",
		"Europe/London",
		"Asia/Kolkata",      // +05:30
		"Asia/Kathmandu",    // +05:45
		"Pacific/Chatham",   // +12:45
		"America/St_Johns",  // -03:30
		"Australia/Sydney",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			loc, err := ResolveZone(name)
			if err != nil {
				t.Fatalf("ResolveZone(%q) err = %v", name, err)
			}
			if loc == nil {
				t.Fatalf("ResolveZone(%q) returned nil location", name)
			}
			// Sanity: we can convert a time into the zone without panicking.
			_ = time.Now().In(loc)
		})
	}
}

func TestResolveZone_LiteralUTCAndZ(t *testing.T) {
	for _, s := range []string{"Z", "z", "UTC", "utc", " UTC ", ""} {
		loc, err := ResolveZone(s)
		if err != nil {
			t.Fatalf("ResolveZone(%q) err = %v", s, err)
		}
		if loc != time.UTC {
			t.Fatalf("ResolveZone(%q) = %v, want time.UTC", s, loc)
		}
	}
}

func TestResolveZone_ISOOffsets(t *testing.T) {
	// Each case is (input, expected offset in seconds).
	cases := []struct {
		in     string
		offset int
	}{
		{"+05:30", 5*3600 + 30*60},
		{"-05:30", -(5*3600 + 30*60)},
		{"+05:45", 5*3600 + 45*60}, // Kathmandu / Chatham minute granularity
		{"-03:30", -(3*3600 + 30*60)},
		{"+0530", 5*3600 + 30*60},
		{"-0330", -(3*3600 + 30*60)},
		{"+05", 5 * 3600},
		{"-05", -5 * 3600},
		{"+00:00", 0},
		{"-00:00", 0},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			loc, err := ResolveZone(tc.in)
			if err != nil {
				t.Fatalf("ResolveZone(%q) err = %v", tc.in, err)
			}
			// Pick an arbitrary instant; FixedZone offsets are constant.
			_, off := time.Date(2026, 6, 15, 12, 0, 0, 0, loc).Zone()
			if off != tc.offset {
				t.Fatalf("ResolveZone(%q) offset = %d, want %d", tc.in, off, tc.offset)
			}
		})
	}
}

func TestResolveZone_Errors(t *testing.T) {
	cases := []string{
		"Not/A/Zone",
		"Eastern", // Regional shorthand is intentionally rejected.
		"Local",   // Reserved Go keyword we explicitly reject.
		"+25:00",  // Hour out of range.
		"-05:99",  // Minute out of range.
		"+5:30",   // Hour must be 2 digits.
		"+5",      // Hour must be 2 digits.
		"+05:3",   // Minute must be 2 digits.
		"+05:30:00", // Seconds not supported.
		"05:30",   // Missing leading sign.
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			_, err := ResolveZone(s)
			if err == nil {
				t.Fatalf("ResolveZone(%q) = nil err, want error", s)
			}
			if !strings.Contains(err.Error(), "unknown timezone") {
				t.Fatalf("ResolveZone(%q) err = %q, want substring 'unknown timezone'", s, err)
			}
		})
	}
}

// TestRebaseDateInZone_Wallclock: a tz-naïve "2026-04-15" parsed as midnight
// UTC, rebased into NY, must read 2026-04-15 midnight NY (-4h DST → an
// instant 4 hours later than UTC midnight on that date).
func TestRebaseDateInZone_Wallclock(t *testing.T) {
	src := GetRTime(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))
	rebased, ok := RebaseDateInZone(src, "America/New_York")
	if !ok {
		t.Fatal("RebaseDateInZone failed")
	}
	tv, err := rebased.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue: %v", err)
	}
	if tv.Location().String() != "America/New_York" {
		t.Fatalf("location = %v, want America/New_York", tv.Location())
	}
	if tv.Year() != 2026 || tv.Month() != time.April || tv.Day() != 15 {
		t.Fatalf("date = %s, want 2026-04-15 NY", tv.Format(time.RFC3339))
	}
	if tv.Hour() != 0 || tv.Minute() != 0 {
		t.Fatalf("time-of-day = %02d:%02d, want 00:00 NY", tv.Hour(), tv.Minute())
	}
}

func TestRebaseDateInZone_InvalidZoneReturnsFalse(t *testing.T) {
	src := GetRTime(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))
	out, ok := RebaseDateInZone(src, "Not/A/Zone")
	if ok {
		t.Fatal("expected ok=false for invalid zone")
	}
	if out != src {
		t.Fatal("expected original date returned on failure")
	}
}
