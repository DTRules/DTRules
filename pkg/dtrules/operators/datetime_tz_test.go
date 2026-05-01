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

// Phase 1 of #743 anchors implicit-zone date operators to UTC. Before this
// fix the same instant could yield different year/month/day depending on
// the server's TZ or the input date's stored zone. The tests below pin the
// UTC contract by feeding inputs in non-UTC zones and asserting UTC results.

// pushTime pushes an arbitrary time.Time as an RDate. Unlike pushDate (in
// datetime_test.go) it preserves the supplied location, so tests can feed
// dates carrying a non-UTC zone.
func pushTime(t *testing.T, state dtrules.State, tt time.Time) {
	t.Helper()
	if err := state.DataPush(dtrules.GetRTime(tt)); err != nil {
		t.Fatalf("DataPush time: %v", err)
	}
}

// TestToday_UTCAnchored_AcrossServerZones exercises opToday across several
// candidate server zones. Before the fix opToday used now.Location(); the
// test simulates the post-fix invariant by asserting today's UTC midnight
// regardless of any other interpretation.
func TestToday_UTCAnchored_AcrossServerZones(t *testing.T) {
	o, ok := Get(dtrules.GetRName("today"))
	if !ok {
		t.Fatal("op today not registered")
	}
	state := newTestState()
	if err := o.Execute(state); err != nil {
		t.Fatalf("today: %v", err)
	}
	got, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	tv, err := got.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue: %v", err)
	}
	// today's result is UTC-anchored: location must be UTC and the time
	// component must be 00:00:00.
	if tv.Location() != time.UTC {
		t.Fatalf("today location = %v, want UTC", tv.Location())
	}
	if tv.Hour() != 0 || tv.Minute() != 0 || tv.Second() != 0 || tv.Nanosecond() != 0 {
		t.Fatalf("today time-of-day = %v, want 00:00:00.000000000", tv)
	}
	// Sanity: it should match today's UTC calendar date.
	now := time.Now().UTC()
	wantY, wantM, wantD := now.Date()
	gotY, gotM, gotD := tv.Date()
	if gotY != wantY || gotM != wantM || gotD != wantD {
		t.Fatalf("today date = %d-%02d-%02d, want %d-%02d-%02d",
			gotY, gotM, gotD, wantY, wantM, wantD)
	}
}

// TestFirstOfMonth_NormalizesToUTC: a date constructed at 23:30 on the last
// day of a month in a UTC-8 zone is, in UTC, the first of the next month.
// firstofmonth should bucket using UTC, not the input's stored zone.
func TestFirstOfMonth_NormalizesToUTC(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	// 2026-03-31 23:30 PT == 2026-04-01 07:30 UTC
	in := time.Date(2026, 3, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("firstofmonth"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("firstofmonth: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	tv = tv.UTC()
	if tv.Year() != 2026 || tv.Month() != time.April || tv.Day() != 1 {
		t.Fatalf("firstofmonth = %s, want 2026-04-01 UTC", tv.Format(time.RFC3339))
	}
	if tv.Hour() != 0 || tv.Minute() != 0 {
		t.Fatalf("firstofmonth time-of-day = %v, want midnight", tv)
	}
}

// TestFirstOfYear_NormalizesToUTC: 2026-12-31 23:30 PT == 2027-01-01 07:30 UTC,
// so firstofyear should land on 2027-01-01 UTC.
func TestFirstOfYear_NormalizesToUTC(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 12, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("firstofyear"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("firstofyear: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	tv = tv.UTC()
	if tv.Year() != 2027 || tv.Month() != time.January || tv.Day() != 1 {
		t.Fatalf("firstofyear = %s, want 2027-01-01 UTC", tv.Format(time.RFC3339))
	}
}

// TestEndOfMonth_NormalizesToUTC: 2026-03-31 23:30 PT lives in UTC-April,
// so endofmonth in UTC is 2026-04-30 (not 2026-03-31 from PT's perspective).
func TestEndOfMonth_NormalizesToUTC(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 3, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("endofmonth"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("endofmonth: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	tv = tv.UTC()
	if tv.Year() != 2026 || tv.Month() != time.April || tv.Day() != 30 {
		t.Fatalf("endofmonth = %s, want 2026-04-30 UTC", tv.Format(time.RFC3339))
	}
}

// TestGetYear_UTCExtraction_NearYearBoundary: 2026-12-31 23:30 PT is
// 2027-01-01 07:30 UTC. getyear should return 2027.
func TestGetYear_UTCExtraction_NearYearBoundary(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 12, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("getyear"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getyear: %v", err)
	}
	got, _ := state.DataPop()
	v, err := got.IntValue()
	if err != nil {
		t.Fatalf("IntValue: %v", err)
	}
	if v != 2027 {
		t.Fatalf("getyear = %d, want 2027", v)
	}
}

// TestGetMonth_UTCExtraction_NearMonthBoundary: same instant as above —
// UTC month is January (1).
func TestGetMonth_UTCExtraction_NearMonthBoundary(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 12, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("getmonth"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getmonth: %v", err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	if v != 1 {
		t.Fatalf("getmonth = %d, want 1 (UTC January)", v)
	}
}

// TestGetDayOfMonth_UTC: same instant — UTC day-of-month is 1.
func TestGetDayOfMonth_UTC(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 12, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("getdayofmonth"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getdayofmonth: %v", err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	if v != 1 {
		t.Fatalf("getdayofmonth = %d, want 1 (UTC)", v)
	}
}

// TestGetDay_UTC mirrors TestGetDayOfMonth_UTC for the alias `getday`.
func TestGetDay_UTC(t *testing.T) {
	pacific := time.FixedZone("PT", -8*3600)
	in := time.Date(2026, 12, 31, 23, 30, 0, 0, pacific)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("getday"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getday: %v", err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	if v != 1 {
		t.Fatalf("getday = %d, want 1 (UTC)", v)
	}
}

// TestGetDaysInMonth_UTC: 2026-01-31 23:30 in UTC+12:45 (Pacific/Chatham
// style fixed offset) is already in February UTC. days-in-month should
// reflect February (28 in 2026), not January.
func TestGetDaysInMonth_UTC(t *testing.T) {
	chatham := time.FixedZone("CHADT", 12*3600+45*60)
	// 2026-02-01 12:30 CHADT == 2026-01-31 23:45 UTC, still January UTC.
	in := time.Date(2026, 2, 1, 12, 30, 0, 0, chatham)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("getdaysinmonth"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getdaysinmonth: %v", err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	if v != 31 {
		t.Fatalf("getdaysinmonth = %d, want 31 (UTC January)", v)
	}
}

// TestGetDaysInYear_UTC: 2024-01-01 03:30 in Asia/Kolkata (+05:30) is
// 2023-12-31 22:00 UTC. UTC year is 2023 (365 days), not 2024 (366).
func TestGetDaysInYear_UTC(t *testing.T) {
	kolkata := time.FixedZone("IST", 5*3600+30*60)
	in := time.Date(2024, 1, 1, 3, 30, 0, 0, kolkata)
	state := newTestState()
	pushTime(t, state, in)
	o, _ := Get(dtrules.GetRName("getdaysinyear"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getdaysinyear: %v", err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	if v != 365 {
		t.Fatalf("getdaysinyear = %d, want 365 (UTC year 2023)", v)
	}
}
