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
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Phase 2 of #743: explicit `in zone <strexpr>` operator family.
// Plain ops (today, firstofmonth, ...) stay UTC-anchored; the *inzone
// variants pop a zone string from the data stack first and operate in the
// resolved zone.

func pushZone(t *testing.T, state dtrules.State, zone string) {
	t.Helper()
	if err := state.DataPush(dtrules.NewRString(zone)); err != nil {
		t.Fatalf("DataPush zone: %v", err)
	}
}

// TestTodayInZone_NewYork: today in NY returns NY-midnight, location pinned
// to America/New_York.
func TestTodayInZone_NewYork(t *testing.T) {
	state := newTestState()
	pushZone(t, state, "America/New_York")
	o, ok := Get(dtrules.GetRName("todayinzone"))
	if !ok {
		t.Fatal("todayinzone not registered")
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("todayinzone: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	if tv.Location().String() != "America/New_York" {
		t.Fatalf("location = %v, want America/New_York", tv.Location())
	}
	if tv.Hour() != 0 || tv.Minute() != 0 || tv.Second() != 0 || tv.Nanosecond() != 0 {
		t.Fatalf("time-of-day = %v, want midnight in NY", tv)
	}
}

// TestTodayInZone_Kathmandu pins the +05:45 offset case (Asia/Kathmandu).
// The plain `today` op is UTC midnight; in Kathmandu it must be Kathmandu
// midnight, which sits at 18:15 the previous UTC day.
func TestTodayInZone_Kathmandu(t *testing.T) {
	state := newTestState()
	pushZone(t, state, "+05:45")
	o, _ := Get(dtrules.GetRName("todayinzone"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("todayinzone +05:45: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	_, offset := tv.Zone()
	if offset != 5*3600+45*60 {
		t.Fatalf("offset = %d, want %d (+05:45)", offset, 5*3600+45*60)
	}
	if tv.Hour() != 0 || tv.Minute() != 0 {
		t.Fatalf("time-of-day = %v, want midnight in +05:45", tv)
	}
}

// TestTodayInZone_Z: zone "Z" resolves to UTC.
func TestTodayInZone_Z(t *testing.T) {
	state := newTestState()
	pushZone(t, state, "Z")
	o, _ := Get(dtrules.GetRName("todayinzone"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("todayinzone Z: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	if tv.Location() != time.UTC {
		t.Fatalf("Z location = %v, want UTC", tv.Location())
	}
}

// TestFirstOfMonthInZone: a date carrying a fixed PT zone but interpreted in
// NY yields the right NY first-of-month. 2026-04-01 02:30 PT == 2026-04-01
// 05:30 NY (no DST drift on this date), so the first-of-month is 2026-04-01.
// The post-Phase-1 plain firstofmonth would return 2026-04-01 (UTC) for this
// instant — the in-zone variant proves the explicit clause flows through to
// the runtime.
func TestFirstOfMonthInZone_NewYork(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 4, 1, 2, 30, 0, 0, pacific) // 2026-04-01 05:30 NY
	state := newTestState()
	pushTime(t, state, in)
	pushZone(t, state, "America/New_York")
	o, _ := Get(dtrules.GetRName("firstofmonthinzone"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("firstofmonthinzone: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	if tv.Location().String() != "America/New_York" {
		t.Fatalf("location = %v, want America/New_York", tv.Location())
	}
	if tv.Year() != 2026 || tv.Month() != time.April || tv.Day() != 1 {
		t.Fatalf("firstofmonth in NY = %s, want 2026-04-01", tv.Format(time.RFC3339))
	}
}

// TestGetYearInZone_BoundaryDifference is the canonical round-trip from the
// issue body: 2026-12-31 23:30Z is 2026 in UTC but 2026 in NY (-5 hours →
// 2026-12-31 18:30 NY). Use 2026-01-01 04:30 UTC, which is 2025-12-31 23:30
// in NY (-5h), so getyear-in-NY = 2025 while plain getyear = 2026.
func TestGetYearInZone_BoundaryDifference(t *testing.T) {
	in := time.Date(2026, 1, 1, 4, 30, 0, 0, time.UTC)

	// plain getyear → 2026 (UTC).
	state := newTestState()
	pushTime(t, state, in)
	op, _ := Get(dtrules.GetRName("getyear"))
	if err := op.Execute(state); err != nil {
		t.Fatalf("getyear: %v", err)
	}
	gotPlain, _ := state.DataPop()
	yp, _ := gotPlain.IntValue()
	if yp != 2026 {
		t.Fatalf("plain getyear = %d, want 2026 (UTC)", yp)
	}

	// in-zone getyear NY → 2025.
	state = newTestState()
	pushTime(t, state, in)
	pushZone(t, state, "America/New_York")
	op, _ = Get(dtrules.GetRName("getyearinzone"))
	if err := op.Execute(state); err != nil {
		t.Fatalf("getyearinzone: %v", err)
	}
	gotZ, _ := state.DataPop()
	yz, _ := gotZ.IntValue()
	if yz != 2025 {
		t.Fatalf("getyearinzone NY = %d, want 2025", yz)
	}
}

// TestDateInZone_Rewrap: same instant, rewrapped in a different zone.
func TestDateInZone_Rewrap(t *testing.T) {
	in := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	state := newTestState()
	pushTime(t, state, in)
	pushZone(t, state, "Asia/Kolkata")
	o, _ := Get(dtrules.GetRName("dateinzone"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("dateinzone: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	if !tv.Equal(in) {
		t.Fatalf("instant changed; got %s, want %s", tv, in)
	}
	if tv.Location().String() != "Asia/Kolkata" {
		t.Fatalf("location = %v, want Asia/Kolkata", tv.Location())
	}
	// Same instant, different wall clock: 17:30 in Kolkata.
	if tv.Hour() != 17 || tv.Minute() != 30 {
		t.Fatalf("local time = %02d:%02d, want 17:30", tv.Hour(), tv.Minute())
	}
}

// TestInZone_InvalidZoneError: ResolveZone failure surfaces as a runtime
// error from the operator.
func TestTodayInZone_InvalidZoneError(t *testing.T) {
	state := newTestState()
	pushZone(t, state, "Not/A/Zone")
	o, _ := Get(dtrules.GetRName("todayinzone"))
	if err := o.Execute(state); err == nil {
		t.Fatal("expected error for invalid zone, got nil")
	}
}
