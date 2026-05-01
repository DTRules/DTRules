// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package operators

import (
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Phase 3 of #743: time-component extractors, week/quarter/year buckets, and
// calendar comparisons. Plain ops anchor in UTC; *inzone variants pop a zone
// string and operate in the resolved zone.

// runDateOpInt pushes a date, executes the op, and returns the int result.
func runDateOpInt(t *testing.T, name string, in time.Time) int {
	t.Helper()
	state := newTestState()
	pushTime(t, state, in)
	op, ok := Get(dtrules.GetRName(name))
	if !ok {
		t.Fatalf("op %q not registered", name)
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	return v
}

// runDateOpIntInZone pushes (date, zone), executes, and returns the int.
func runDateOpIntInZone(t *testing.T, name string, in time.Time, zone string) int {
	t.Helper()
	state := newTestState()
	pushTime(t, state, in)
	pushZone(t, state, zone)
	op, ok := Get(dtrules.GetRName(name))
	if !ok {
		t.Fatalf("op %q not registered", name)
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	got, _ := state.DataPop()
	v, _ := got.IntValue()
	return v
}

// runDateOpDate pushes a date, executes, returns date.
func runDateOpDate(t *testing.T, name string, in time.Time) time.Time {
	t.Helper()
	state := newTestState()
	pushTime(t, state, in)
	op, _ := Get(dtrules.GetRName(name))
	if err := op.Execute(state); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	return tv
}

func runDateOpDateInZone(t *testing.T, name string, in time.Time, zone string) time.Time {
	t.Helper()
	state := newTestState()
	pushTime(t, state, in)
	pushZone(t, state, zone)
	op, _ := Get(dtrules.GetRName(name))
	if err := op.Execute(state); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	return tv
}

// TestPhase3_GetHour: plain (UTC) and in-zone variants for hour extraction.
// 2026-01-01 04:30 UTC == 2025-12-31 23:30 NY; hour-in-NY=23, plain=4.
func TestPhase3_GetHour(t *testing.T) {
	in := time.Date(2026, 1, 1, 4, 30, 0, 0, time.UTC)
	if got := runDateOpInt(t, "gethour", in); got != 4 {
		t.Fatalf("gethour = %d, want 4 (UTC)", got)
	}
	if got := runDateOpIntInZone(t, "gethourinzone", in, "America/New_York"); got != 23 {
		t.Fatalf("gethourinzone NY = %d, want 23", got)
	}
}

// TestPhase3_GetMinute: minute extraction in UTC and in zone.
func TestPhase3_GetMinute(t *testing.T) {
	in := time.Date(2026, 6, 1, 12, 45, 30, 0, time.UTC)
	if got := runDateOpInt(t, "getminute", in); got != 45 {
		t.Fatalf("getminute = %d, want 45", got)
	}
	if got := runDateOpIntInZone(t, "getminuteinzone", in, "Asia/Kolkata"); got != 15 {
		// Kolkata is +05:30; 12:45 UTC + 5:30 = 18:15 → minute 15.
		t.Fatalf("getminuteinzone Kolkata = %d, want 15", got)
	}
}

// TestPhase3_GetSecond: second extraction.
func TestPhase3_GetSecond(t *testing.T) {
	in := time.Date(2026, 6, 1, 12, 45, 42, 0, time.UTC)
	if got := runDateOpInt(t, "getsecond", in); got != 42 {
		t.Fatalf("getsecond = %d, want 42", got)
	}
	if got := runDateOpIntInZone(t, "getsecondinzone", in, "America/New_York"); got != 42 {
		t.Fatalf("getsecondinzone = %d, want 42", got)
	}
}

// TestPhase3_DayOfWeek: ISO numbering Mon=1..Sun=7. 2026-01-05 is a Monday in
// UTC; in NY at 03:00 UTC the same instant is 2026-01-04 22:00 (Sunday=7).
func TestPhase3_DayOfWeek(t *testing.T) {
	mondayUTC := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	if got := runDateOpInt(t, "getdayofweek", mondayUTC); got != 1 {
		t.Fatalf("getdayofweek 2026-01-05 = %d, want 1 (Mon)", got)
	}
	earlyMon := time.Date(2026, 1, 5, 3, 0, 0, 0, time.UTC) // 22:00 NY 2026-01-04
	if got := runDateOpIntInZone(t, "getdayofweekinzone", earlyMon, "America/New_York"); got != 7 {
		t.Fatalf("getdayofweekinzone NY = %d, want 7 (Sun in NY)", got)
	}
}

// TestPhase3_WeekOfYear: ISO 8601. 2026-01-01 (Thursday) → ISO week 1 of 2026.
// 2025-12-29 (Monday) → ISO week 1 of 2026 too.
func TestPhase3_WeekOfYear(t *testing.T) {
	jan1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if got := runDateOpInt(t, "getweekofyear", jan1); got != 1 {
		t.Fatalf("getweekofyear 2026-01-01 = %d, want 1", got)
	}
	dec29 := time.Date(2025, 12, 29, 12, 0, 0, 0, time.UTC)
	if got := runDateOpInt(t, "getweekofyear", dec29); got != 1 {
		t.Fatalf("getweekofyear 2025-12-29 = %d, want 1 (ISO week 1 of 2026)", got)
	}
	if got := runDateOpIntInZone(t, "getweekofyearinzone", jan1, "UTC"); got != 1 {
		t.Fatalf("getweekofyearinzone UTC = %d, want 1", got)
	}
}

// TestFirstOfWeek_DefaultMonday: default Monday-anchored. 2026-04-15 is a
// Wednesday → first-of-week is Monday 2026-04-13.
func TestFirstOfWeek_DefaultMonday(t *testing.T) {
	wed := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	got := runDateOpDate(t, "firstofweek", wed)
	if got.Year() != 2026 || got.Month() != 4 || got.Day() != 13 {
		t.Fatalf("firstofweek = %s, want 2026-04-13", got.Format(time.RFC3339))
	}
}

// TestFirstOfWeek_StartingSunday: explicit Sunday start.
func TestFirstOfWeek_StartingSunday(t *testing.T) {
	wed := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	state := newTestState()
	pushTime(t, state, wed)
	if err := state.DataPush(dtrules.NewRString("Sunday")); err != nil {
		t.Fatalf("push start: %v", err)
	}
	op, _ := Get(dtrules.GetRName("firstofweekstarting"))
	if err := op.Execute(state); err != nil {
		t.Fatalf("firstofweekstarting: %v", err)
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	if tv.Year() != 2026 || tv.Month() != 4 || tv.Day() != 12 {
		t.Fatalf("firstofweekstarting Sun = %s, want 2026-04-12", tv.Format(time.RFC3339))
	}
}

// TestEndOfWeek_Monday: end-of-week for Monday-start is Sunday.
func TestEndOfWeek_Monday(t *testing.T) {
	wed := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	got := runDateOpDate(t, "endofweek", wed)
	if got.Year() != 2026 || got.Month() != 4 || got.Day() != 19 {
		t.Fatalf("endofweek = %s, want 2026-04-19", got.Format(time.RFC3339))
	}
}

// TestFirstOfQuarter: covers all four quarters and the boundary case where
// Mar 31 is in Q1 and Apr 1 is in Q2.
func TestFirstOfQuarter(t *testing.T) {
	cases := []struct {
		in   time.Time
		want time.Time
	}{
		{time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 3, 31, 23, 59, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got := runDateOpDate(t, "firstofquarter", c.in)
		if !got.Equal(c.want) {
			t.Errorf("firstofquarter %s = %s, want %s",
				c.in.Format("2006-01-02"), got.Format("2006-01-02"), c.want.Format("2006-01-02"))
		}
	}
}

// TestEndOfQuarter: Q1=Mar31, Q2=Jun30, Q3=Sep30, Q4=Dec31.
func TestEndOfQuarter(t *testing.T) {
	cases := []struct {
		in   time.Time
		want time.Time
	}{
		{time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got := runDateOpDate(t, "endofquarter", c.in)
		if !got.Equal(c.want) {
			t.Errorf("endofquarter %s = %s, want %s",
				c.in.Format("2006-01-02"), got.Format("2006-01-02"), c.want.Format("2006-01-02"))
		}
	}
}

// TestEndOfYear: Dec 31 in UTC and in NY (different calendar years possible).
func TestEndOfYear(t *testing.T) {
	in := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	got := runDateOpDate(t, "endofyear", in)
	if got.Year() != 2026 || got.Month() != 12 || got.Day() != 31 {
		t.Fatalf("endofyear = %s, want 2026-12-31", got.Format(time.RFC3339))
	}
	// 2027-01-01 02:00 UTC == 2026-12-31 21:00 NY → NY year 2026 → Dec 31 2026.
	jan1 := time.Date(2027, 1, 1, 2, 0, 0, 0, time.UTC)
	gotNY := runDateOpDateInZone(t, "endofyearinzone", jan1, "America/New_York")
	if gotNY.Year() != 2026 || gotNY.Month() != 12 || gotNY.Day() != 31 {
		t.Fatalf("endofyearinzone NY = %s, want 2026-12-31", gotNY.Format(time.RFC3339))
	}
}

// runSameCalendarOp pushes (d1, d2, zone), runs op, returns bool.
func runSameCalendarOp(t *testing.T, name string, d1, d2 time.Time, zone string) bool {
	t.Helper()
	state := newTestState()
	pushTime(t, state, d1)
	pushTime(t, state, d2)
	pushZone(t, state, zone)
	op, ok := Get(dtrules.GetRName(name))
	if !ok {
		t.Fatalf("op %q not registered", name)
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	got, _ := state.DataPop()
	b, _ := got.BooleanValue()
	return b
}

// TestSameCalendarDay_StraddlesUTCBoundary: two instants on the same UTC day
// can be different NY days, and vice versa. 2026-04-16 03:00 UTC and
// 2026-04-15 23:00 UTC are different UTC days but the second one in NY (-4 in
// April with DST) is 19:00 NY 2026-04-15 — and the first in NY is 23:00 NY
// 2026-04-15. So same NY day, different UTC days.
func TestSameCalendarDay_StraddlesUTCBoundary(t *testing.T) {
	a := time.Date(2026, 4, 16, 3, 0, 0, 0, time.UTC)
	b := time.Date(2026, 4, 15, 23, 0, 0, 0, time.UTC)
	if !runSameCalendarOp(t, "samecalendardayinzone", a, b, "America/New_York") {
		t.Fatalf("expected same calendar day in NY for %s and %s", a, b)
	}
	if runSameCalendarOp(t, "samecalendardayinzone", a, b, "UTC") {
		t.Fatalf("expected different calendar day in UTC for %s and %s", a, b)
	}
}

// TestSameCalendarWeek_DefaultMonday: two dates in the same Mon-Sun week.
func TestSameCalendarWeek_DefaultMonday(t *testing.T) {
	mon := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	sun := time.Date(2026, 4, 19, 23, 0, 0, 0, time.UTC)
	if !runSameCalendarOp(t, "samecalendarweekinzone", mon, sun, "UTC") {
		t.Fatalf("expected same week (Mon-anchored)")
	}
	nextMon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	if runSameCalendarOp(t, "samecalendarweekinzone", mon, nextMon, "UTC") {
		t.Fatalf("expected different week")
	}
}

// TestSameCalendarWeek_StartingSunday: switching the start day to Sunday
// shifts the boundary. 2026-04-19 (Sunday) and 2026-04-20 (Monday) are in
// different weeks Mon-anchored, but the same week Sun-anchored.
func TestSameCalendarWeek_StartingSunday(t *testing.T) {
	state := newTestState()
	sun := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	pushTime(t, state, sun)
	pushTime(t, state, mon)
	if err := state.DataPush(dtrules.NewRString("Sunday")); err != nil {
		t.Fatalf("push start: %v", err)
	}
	pushZone(t, state, "UTC")
	op, _ := Get(dtrules.GetRName("samecalendarweekstartinginzone"))
	if err := op.Execute(state); err != nil {
		t.Fatalf("samecalendarweekstartinginzone: %v", err)
	}
	got, _ := state.DataPop()
	if b, _ := got.BooleanValue(); !b {
		t.Fatalf("expected same week with Sunday start")
	}
}

// TestSameCalendarMonth: month boundaries.
func TestSameCalendarMonth(t *testing.T) {
	a := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	b := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if runSameCalendarOp(t, "samecalendarmonthinzone", a, b, "UTC") {
		t.Fatalf("expected different months")
	}
	c := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	if !runSameCalendarOp(t, "samecalendarmonthinzone", b, c, "UTC") {
		t.Fatalf("expected same month")
	}
}

// TestSameCalendarQuarter: Q1/Q2 boundary at Mar31/Apr1.
func TestSameCalendarQuarter(t *testing.T) {
	mar31 := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	apr1 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if runSameCalendarOp(t, "samecalendarquarterinzone", mar31, apr1, "UTC") {
		t.Fatalf("Mar31 vs Apr1: expected different quarters")
	}
	jun30 := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	if !runSameCalendarOp(t, "samecalendarquarterinzone", apr1, jun30, "UTC") {
		t.Fatalf("Apr1 vs Jun30: expected same quarter")
	}
}

// TestSameCalendarYear: across year boundary.
func TestSameCalendarYear(t *testing.T) {
	dec31 := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)
	jan1 := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	if runSameCalendarOp(t, "samecalendaryearinzone", dec31, jan1, "UTC") {
		t.Fatalf("expected different years")
	}
	if !runSameCalendarOp(t, "samecalendaryearinzone", dec31, dec31, "UTC") {
		t.Fatalf("expected same year")
	}
}

// TestPhase3_BadStartDay: parseStartDay rejects garbage with an error.
func TestPhase3_BadStartDay(t *testing.T) {
	state := newTestState()
	pushTime(t, state, time.Now())
	if err := state.DataPush(dtrules.NewRString("Funday")); err != nil {
		t.Fatalf("push: %v", err)
	}
	op, _ := Get(dtrules.GetRName("firstofweekstarting"))
	if err := op.Execute(state); err == nil {
		t.Fatalf("expected error on bad start day")
	}
}
