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
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ResolveZone parses an IANA timezone name or an ISO 8601 fixed offset and
// returns the corresponding *time.Location.
//
// Resolution chain (Phase 2 of #743):
//
//  1. "Z" / "UTC" / "" → time.UTC.
//  2. ISO 8601 fixed offset: +HH:MM, -HH:MM, +HHMM, -HHMM, +HH, -HH.
//     Resolved via time.FixedZone — supports arbitrary 15-minute offsets
//     such as +05:45 (Asia/Kathmandu).
//  3. IANA name: time.LoadLocation, which uses the bundled tzdata.
//  4. Anything else → structured error pointing the caller at the two
//     supported formats.
func ResolveZone(s string) (*time.Location, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return time.UTC, nil
	}

	// Treat Z and UTC as fast-path UTC literals; LoadLocation also handles
	// "UTC" but going through it for a literal is wasteful and "Z" is not
	// a valid IANA name.
	upper := strings.ToUpper(trimmed)
	if upper == "Z" || upper == "UTC" {
		return time.UTC, nil
	}

	// ISO 8601 fixed offset must start with + or -. Try this before
	// LoadLocation so a typo like "-05:00x" surfaces a useful error rather
	// than a generic "unknown time zone".
	if first := trimmed[0]; first == '+' || first == '-' {
		loc, err := parseISOOffset(trimmed)
		if err != nil {
			return nil, fmt.Errorf("unknown timezone %q: %w", s, err)
		}
		return loc, nil
	}

	// Fall through to IANA. Reject obviously-bad inputs that LoadLocation
	// would otherwise resolve to "Local" silently.
	if trimmed == "Local" {
		return nil, fmt.Errorf(
			"unknown timezone %q: expected IANA name (e.g. \"America/New_York\") or ISO 8601 offset (e.g. \"+05:45\")",
			s,
		)
	}

	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, fmt.Errorf(
			"unknown timezone %q: expected IANA name (e.g. \"America/New_York\") or ISO 8601 offset (e.g. \"+05:45\"): %w",
			s, err,
		)
	}
	return loc, nil
}

// parseISOOffset parses ISO 8601 fixed-offset forms. Accepts +HH, +HHMM,
// +HH:MM, and the same with a leading minus.
func parseISOOffset(s string) (*time.Location, error) {
	if len(s) < 2 {
		return nil, fmt.Errorf("offset too short")
	}
	sign := 1
	switch s[0] {
	case '+':
		sign = 1
	case '-':
		sign = -1
	default:
		return nil, fmt.Errorf("expected leading + or -")
	}
	body := s[1:]

	var hh, mm string
	switch {
	case strings.Contains(body, ":"):
		parts := strings.Split(body, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected HH:MM")
		}
		hh, mm = parts[0], parts[1]
	case len(body) == 4:
		hh, mm = body[:2], body[2:]
	case len(body) == 2:
		hh, mm = body, "00"
	default:
		return nil, fmt.Errorf("expected HH, HHMM, or HH:MM")
	}

	if len(hh) != 2 || len(mm) != 2 {
		return nil, fmt.Errorf("hour and minute must be two digits each")
	}

	hours, err := strconv.Atoi(hh)
	if err != nil {
		return nil, fmt.Errorf("invalid hour %q", hh)
	}
	minutes, err := strconv.Atoi(mm)
	if err != nil {
		return nil, fmt.Errorf("invalid minute %q", mm)
	}
	if hours < 0 || hours > 23 {
		return nil, fmt.Errorf("hour %d out of range 0-23", hours)
	}
	if minutes < 0 || minutes > 59 {
		return nil, fmt.Errorf("minute %d out of range 0-59", minutes)
	}

	totalSeconds := sign * (hours*3600 + minutes*60)
	// Use the canonical name format Go itself emits for FixedZone offsets
	// so round-tripping through time.Time.String is predictable.
	name := s
	return time.FixedZone(name, totalSeconds), nil
}

// RebaseDateInZone rewrites a date that was parsed as if tz-naïve (typically
// UTC by default) so that its calendar components Y/M/D/H/M/S land in the
// declared timezone tzName. The absolute instant changes because the same
// wall clock now refers to a different point on the global timeline.
//
// Phase 2 of #743: used by the EDD loader for fields with `timezone="..."`,
// where author-provided default strings should be interpreted in the field's
// declared zone rather than UTC.
//
// Returns the rebased *RDate and true on success. If the zone is invalid
// the original date is left untouched and the boolean is false.
func RebaseDateInZone(d *RDate, tzName string) (*RDate, bool) {
	if d == nil {
		return d, false
	}
	loc, err := ResolveZone(tzName)
	if err != nil {
		return d, false
	}
	src := d.time
	rebased := time.Date(src.Year(), src.Month(), src.Day(),
		src.Hour(), src.Minute(), src.Second(), src.Nanosecond(), loc)
	return &RDate{time: rebased}, true
}
