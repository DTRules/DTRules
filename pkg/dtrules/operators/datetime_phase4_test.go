// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package operators

import (
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Phase 4 of #743: format(date, layout) and `with dst_rule` for component
// constructors with `in zone`.

func runFormat(t *testing.T, in time.Time, layout string) string {
	t.Helper()
	state := newTestState()
	pushTime(t, state, in)
	if err := state.DataPush(dtrules.NewRString(layout)); err != nil {
		t.Fatalf("DataPush layout: %v", err)
	}
	op, ok := Get(dtrules.GetRName("dateformat"))
	if !ok {
		t.Fatal("dateformat not registered")
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("dateformat: %v", err)
	}
	got, _ := state.DataPop()
	return got.StringValue()
}

func runFormatInZone(t *testing.T, in time.Time, layout, zone string) string {
	t.Helper()
	state := newTestState()
	pushTime(t, state, in)
	if err := state.DataPush(dtrules.NewRString(layout)); err != nil {
		t.Fatalf("DataPush layout: %v", err)
	}
	pushZone(t, state, zone)
	op, ok := Get(dtrules.GetRName("dateformatinzone"))
	if !ok {
		t.Fatal("dateformatinzone not registered")
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("dateformatinzone: %v", err)
	}
	got, _ := state.DataPop()
	return got.StringValue()
}

// TestFormatDate_UTC pins the no-zone form: a date carrying any zone is
// rendered against its UTC clock.
func TestFormatDate_UTC(t *testing.T) {
	in := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	if got := runFormat(t, in, "2006-01-02"); got != "2026-04-15" {
		t.Fatalf("dateformat = %q, want 2026-04-15", got)
	}
}

// TestFormatDate_UTCAcrossZones: input date carrying a non-UTC zone still
// renders against UTC under plain dateformat.
func TestFormatDate_UTCAcrossZones(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	// 2026-04-15 23:30 NY == 2026-04-16 03:30 UTC.
	in := time.Date(2026, 4, 15, 23, 30, 0, 0, ny)
	if got := runFormat(t, in, "2006-01-02"); got != "2026-04-16" {
		t.Fatalf("dateformat utc-anchored = %q, want 2026-04-16", got)
	}
}

// TestFormatDate_InZoneNY: format the same instant in NY produces the NY
// calendar date.
func TestFormatDate_InZoneNY(t *testing.T) {
	// 2026-04-16 03:30 UTC == 2026-04-15 23:30 NY.
	in := time.Date(2026, 4, 16, 3, 30, 0, 0, time.UTC)
	if got := runFormatInZone(t, in, "2006-01-02", "America/New_York"); got != "2026-04-15" {
		t.Fatalf("dateformatinzone NY date = %q, want 2026-04-15", got)
	}
	if got := runFormatInZone(t, in, "15:04", "America/New_York"); got != "23:30" {
		t.Fatalf("dateformatinzone NY time = %q, want 23:30", got)
	}
}

// TestFormatDate_RFC3339: round-trip a UTC date through RFC3339 layout in UTC.
func TestFormatDate_RFC3339(t *testing.T) {
	in := time.Date(2026, 4, 15, 12, 30, 45, 0, time.UTC)
	if got := runFormat(t, in, time.RFC3339); got != "2026-04-15T12:30:45Z" {
		t.Fatalf("dateformat RFC3339 = %q", got)
	}
}

// TestFormatDate_TimeOnly: layout selecting only time fields.
func TestFormatDate_TimeOnly(t *testing.T) {
	in := time.Date(2026, 4, 15, 9, 7, 5, 0, time.UTC)
	if got := runFormat(t, in, "15:04:05"); got != "09:07:05" {
		t.Fatalf("dateformat time-only = %q", got)
	}
}

// TestFormatDate_FixedOffset: zone "+05:45" renders the local-clock components
// with the fixed offset.
func TestFormatDate_FixedOffset(t *testing.T) {
	in := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	got := runFormatInZone(t, in, "2006-01-02 15:04 -0700", "+05:45")
	want := "2026-04-15 05:45 +0545"
	if got != want {
		t.Fatalf("dateformatinzone +05:45 = %q, want %q", got, want)
	}
}

// runNewDateInZone calls newdateinzone with the given components.
func runNewDateInZone(t *testing.T, y, mo, d, h, mi, s int, zone string) (time.Time, error) {
	t.Helper()
	state := newTestState()
	for _, v := range []int{y, mo, d, h, mi, s} {
		if err := state.DataPush(dtrules.GetRIntegerValue(int64(v))); err != nil {
			t.Fatalf("push int: %v", err)
		}
	}
	pushZone(t, state, zone)
	op, _ := Get(dtrules.GetRName("newdateinzone"))
	if err := op.Execute(state); err != nil {
		return time.Time{}, err
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	return tv, nil
}

// runNewDateInZoneWithDST calls newdateinzonewithdst with the given components
// and dst_rule.
func runNewDateInZoneWithDST(t *testing.T, y, mo, d, h, mi, s int, zone, rule string) (time.Time, error) {
	t.Helper()
	state := newTestState()
	for _, v := range []int{y, mo, d, h, mi, s} {
		if err := state.DataPush(dtrules.GetRIntegerValue(int64(v))); err != nil {
			t.Fatalf("push int: %v", err)
		}
	}
	pushZone(t, state, zone)
	if err := state.DataPush(dtrules.NewRString(rule)); err != nil {
		t.Fatalf("push rule: %v", err)
	}
	op, ok := Get(dtrules.GetRName("newdateinzonewithdst"))
	if !ok {
		t.Fatal("newdateinzonewithdst not registered")
	}
	if err := op.Execute(state); err != nil {
		return time.Time{}, err
	}
	got, _ := state.DataPop()
	tv, _ := got.TimeValue()
	return tv, nil
}

// TestNewDateInZone_DefaultBehavior: the no-rule constructor preserves Go's
// default time.Date adjustment for non-DST inputs.
func TestNewDateInZone_DefaultBehavior(t *testing.T) {
	got, err := runNewDateInZone(t, 2026, 4, 15, 12, 0, 0, "America/New_York")
	if err != nil {
		t.Fatalf("newdateinzone: %v", err)
	}
	if got.Location().String() != "America/New_York" {
		t.Fatalf("location = %v", got.Location())
	}
	if got.Year() != 2026 || got.Month() != time.April || got.Day() != 15 || got.Hour() != 12 {
		t.Fatalf("date = %v, want 2026-04-15 12:00 NY", got)
	}
}

// TestDSTRule_SpringForwardError: NY 2026-03-08 02:30 does not exist; with
// dst_rule "error" the op must error.
func TestDSTRule_SpringForwardError(t *testing.T) {
	_, err := runNewDateInZoneWithDST(t, 2026, 3, 8, 2, 30, 0, "America/New_York", "error")
	if err == nil {
		t.Fatal("expected spring-forward error, got nil")
	}
	if !strings.Contains(err.Error(), "spring-forward") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDSTRule_SpringForwardEarlierError: "earlier" cannot satisfy a
// spring-forward gap either — author must clarify.
func TestDSTRule_SpringForwardEarlierError(t *testing.T) {
	_, err := runNewDateInZoneWithDST(t, 2026, 3, 8, 2, 30, 0, "America/New_York", "earlier")
	if err == nil {
		t.Fatal("expected spring-forward error under \"earlier\", got nil")
	}
}

// TestDSTRule_FallbackEarlier: NY 2026-11-01 01:30 is ambiguous; "earlier"
// returns the EDT instant (05:30 UTC).
func TestDSTRule_FallbackEarlier(t *testing.T) {
	got, err := runNewDateInZoneWithDST(t, 2026, 11, 1, 1, 30, 0, "America/New_York", "earlier")
	if err != nil {
		t.Fatalf("newdateinzonewithdst earlier: %v", err)
	}
	utc := got.UTC()
	want := time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC)
	if !utc.Equal(want) {
		t.Fatalf("earlier = %v UTC, want %v UTC", utc, want)
	}
	// Verify the offset is EDT (-4h).
	_, off := got.Zone()
	if off != -4*3600 {
		t.Fatalf("offset = %d, want -14400 (EDT)", off)
	}
}

// TestDSTRule_FallbackLater: "later" returns the EST instant (06:30 UTC).
func TestDSTRule_FallbackLater(t *testing.T) {
	got, err := runNewDateInZoneWithDST(t, 2026, 11, 1, 1, 30, 0, "America/New_York", "later")
	if err != nil {
		t.Fatalf("newdateinzonewithdst later: %v", err)
	}
	utc := got.UTC()
	want := time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC)
	if !utc.Equal(want) {
		t.Fatalf("later = %v UTC, want %v UTC", utc, want)
	}
	_, off := got.Zone()
	if off != -5*3600 {
		t.Fatalf("offset = %d, want -18000 (EST)", off)
	}
}

// TestDSTRule_FallbackError: "error" rejects the ambiguous local time.
func TestDSTRule_FallbackError(t *testing.T) {
	_, err := runNewDateInZoneWithDST(t, 2026, 11, 1, 1, 30, 0, "America/New_York", "error")
	if err == nil {
		t.Fatal("expected fall-back error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") && !strings.Contains(err.Error(), "fall-back") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDSTRule_NonDSTUnaffected: a clear-of-DST date returns the same instant
// under all rules.
func TestDSTRule_NonDSTUnaffected(t *testing.T) {
	want := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for _, rule := range []string{"earlier", "later", "error"} {
		got, err := runNewDateInZoneWithDST(t, 2026, 6, 15, 8, 0, 0, "America/New_York", rule)
		if err != nil {
			t.Fatalf("rule %q: %v", rule, err)
		}
		if !got.UTC().Equal(want) {
			t.Fatalf("rule %q = %v UTC, want %v UTC", rule, got.UTC(), want)
		}
	}
}

// TestDSTRule_UnknownRule: unknown dst_rule values surface a runtime error.
func TestDSTRule_UnknownRule(t *testing.T) {
	_, err := runNewDateInZoneWithDST(t, 2026, 6, 15, 8, 0, 0, "America/New_York", "skip")
	if err == nil {
		t.Fatal("expected error for unknown rule, got nil")
	}
	if !strings.Contains(err.Error(), "unknown dst_rule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestPhase4_NewDateInZone_AustraliaSydney: southern-hemisphere DST runs in
// October/April. 2026-04-05 02:30 Sydney is the fall-back (DST ends),
// duplicated in local time. Verify "earlier" / "later" disambiguate.
func TestPhase4_NewDateInZone_AustraliaSydney(t *testing.T) {
	gotE, err := runNewDateInZoneWithDST(t, 2026, 4, 5, 2, 30, 0, "Australia/Sydney", "earlier")
	if err != nil {
		t.Fatalf("Sydney earlier: %v", err)
	}
	gotL, err := runNewDateInZoneWithDST(t, 2026, 4, 5, 2, 30, 0, "Australia/Sydney", "later")
	if err != nil {
		t.Fatalf("Sydney later: %v", err)
	}
	if !gotE.UTC().Before(gotL.UTC()) {
		t.Fatalf("earlier %v should be before later %v (UTC)", gotE.UTC(), gotL.UTC())
	}
}
