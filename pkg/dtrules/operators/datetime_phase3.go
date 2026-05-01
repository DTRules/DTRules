// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package operators

import (
	"fmt"
	"strings"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Phase 3 of #743: time-component extraction, week/quarter/year buckets, and
// calendar comparisons. Plain ops fall back to UTC; *inzone variants pop a
// zone string and operate in the resolved zone.
//
// Conventions
//   - Day-of-week numbering: ISO 8601, Mon=1..Sun=7.
//   - Week numbering: ISO 8601 (week 1 is the week containing the first
//     Thursday of the year).
//   - First-of-week / end-of-week default start day: Monday (ISO 8601).
//   - Quarter boundaries: Q1=Jan-Mar, Q2=Apr-Jun, Q3=Jul-Sep, Q4=Oct-Dec.

func init() {
	// Time-component extractors.
	Register("gethour", opGetHour)
	Register("gethourinzone", opGetHourInZone)
	Register("getminute", opGetMinute)
	Register("getminuteinzone", opGetMinuteInZone)
	Register("getsecond", opGetSecond)
	Register("getsecondinzone", opGetSecondInZone)
	Register("getdayofweek", opGetDayOfWeek)
	Register("getdayofweekinzone", opGetDayOfWeekInZone)
	Register("getweekofyear", opGetWeekOfYear)
	Register("getweekofyearinzone", opGetWeekOfYearInZone)

	// Bucket ops — week.
	Register("firstofweek", opFirstOfWeek)
	Register("firstofweekinzone", opFirstOfWeekInZone)
	Register("firstofweekstarting", opFirstOfWeekStarting)
	Register("firstofweekstartinginzone", opFirstOfWeekStartingInZone)
	Register("endofweek", opEndOfWeek)
	Register("endofweekinzone", opEndOfWeekInZone)
	Register("endofweekstarting", opEndOfWeekStarting)
	Register("endofweekstartinginzone", opEndOfWeekStartingInZone)

	// Bucket ops — quarter / year.
	Register("firstofquarter", opFirstOfQuarter)
	Register("firstofquarterinzone", opFirstOfQuarterInZone)
	Register("endofquarter", opEndOfQuarter)
	Register("endofquarterinzone", opEndOfQuarterInZone)
	Register("endofyear", opEndOfYear)
	Register("endofyearinzone", opEndOfYearInZone)

	// Calendar comparisons (zone mandatory by grammar; only *inzone variants
	// exist at the runtime level for the same reason).
	Register("samecalendardayinzone", opSameCalendarDayInZone)
	Register("samecalendarweekinzone", opSameCalendarWeekInZone)
	Register("samecalendarweekstartinginzone", opSameCalendarWeekStartingInZone)
	Register("samecalendarmonthinzone", opSameCalendarMonthInZone)
	Register("samecalendarquarterinzone", opSameCalendarQuarterInZone)
	Register("samecalendaryearinzone", opSameCalendarYearInZone)
}

// popDate pops a date from the data stack and returns its time.Time.
func popDate(state dtrules.State) (time.Time, error) {
	obj, err := state.DataPop()
	if err != nil {
		return time.Time{}, err
	}
	return obj.TimeValue()
}

// parseStartDay maps a starting-day string to time.Weekday. Accepts full
// English names case-insensitively. Defaults to Monday on the empty string.
func parseStartDay(s string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "monday", "mon":
		return time.Monday, nil
	case "sunday", "sun":
		return time.Sunday, nil
	case "tuesday", "tue":
		return time.Tuesday, nil
	case "wednesday", "wed":
		return time.Wednesday, nil
	case "thursday", "thu":
		return time.Thursday, nil
	case "friday", "fri":
		return time.Friday, nil
	case "saturday", "sat":
		return time.Saturday, nil
	}
	return time.Monday, fmt.Errorf("unrecognized week start day %q (expected Monday/Sunday/...)", s)
}

// isoDayOfWeek converts Go's time.Weekday (Sunday=0..Saturday=6) to ISO 8601
// numbering (Mon=1..Sun=7).
func isoDayOfWeek(w time.Weekday) int {
	if w == time.Sunday {
		return 7
	}
	return int(w)
}

// firstOfWeekAt computes the first-of-week date for t in loc, given the
// configured week start day. Returns midnight in loc on the start day.
func firstOfWeekAt(t time.Time, loc *time.Location, startDay time.Weekday) time.Time {
	t = t.In(loc)
	// Days back from t to the most recent startDay (0..6).
	delta := (int(t.Weekday()) - int(startDay) + 7) % 7
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -delta)
	return day
}

// quarterFromMonth returns 1..4 for the calendar quarter of m.
func quarterFromMonth(m time.Month) int {
	return (int(m)-1)/3 + 1
}

// firstMonthOfQuarter returns the first month of the quarter containing m.
func firstMonthOfQuarter(m time.Month) time.Month {
	return time.Month((quarterFromMonth(m)-1)*3 + 1)
}

// ----- Component extractors --------------------------------------------------

// opGetHour: ( date -- int ) hour 0-23 in UTC.
func opGetHour(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.UTC().Hour()))
}

// opGetHourInZone: ( date zone -- int ) hour 0-23 in zone.
func opGetHourInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.In(loc).Hour()))
}

// opGetMinute: ( date -- int ) minute 0-59 in UTC.
func opGetMinute(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.UTC().Minute()))
}

// opGetMinuteInZone: ( date zone -- int ) minute 0-59 in zone.
func opGetMinuteInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.In(loc).Minute()))
}

// opGetSecond: ( date -- int ) second 0-59 in UTC.
func opGetSecond(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.UTC().Second()))
}

// opGetSecondInZone: ( date zone -- int ) second 0-59 in zone.
func opGetSecondInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.In(loc).Second()))
}

// opGetDayOfWeek: ( date -- int ) ISO day-of-week 1..7 (Mon=1) in UTC.
func opGetDayOfWeek(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(isoDayOfWeek(t.UTC().Weekday())))
}

// opGetDayOfWeekInZone: ( date zone -- int ) ISO day-of-week 1..7 (Mon=1) in zone.
func opGetDayOfWeekInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(isoDayOfWeek(t.In(loc).Weekday())))
}

// opGetWeekOfYear: ( date -- int ) ISO 8601 week-of-year 1..53 in UTC.
func opGetWeekOfYear(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	_, week := t.UTC().ISOWeek()
	return state.DataPush(dtrules.GetRIntegerValueFromInt(week))
}

// opGetWeekOfYearInZone: ( date zone -- int ) ISO 8601 week-of-year 1..53 in zone.
func opGetWeekOfYearInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	_, week := t.In(loc).ISOWeek()
	return state.DataPush(dtrules.GetRIntegerValueFromInt(week))
}

// ----- Week buckets ----------------------------------------------------------

// opFirstOfWeek: ( date -- date ) first-of-week (Mon-anchored) in UTC.
func opFirstOfWeek(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(firstOfWeekAt(t, time.UTC, time.Monday)))
}

// opFirstOfWeekInZone: ( date zone -- date ) first-of-week (Mon-anchored) in zone.
func opFirstOfWeekInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(firstOfWeekAt(t, loc, time.Monday)))
}

// opFirstOfWeekStarting: ( date startDay -- date ) first-of-week with explicit
// start in UTC.
func opFirstOfWeekStarting(state dtrules.State) error {
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startDay, err := parseStartDay(startObj.StringValue())
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(firstOfWeekAt(t, time.UTC, startDay)))
}

// opFirstOfWeekStartingInZone: ( date startDay zone -- date ) first-of-week
// with explicit start in zone.
func opFirstOfWeekStartingInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startDay, err := parseStartDay(startObj.StringValue())
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(firstOfWeekAt(t, loc, startDay)))
}

// opEndOfWeek: ( date -- date ) end-of-week (Mon-anchored, so Sunday) in UTC.
func opEndOfWeek(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	first := firstOfWeekAt(t, time.UTC, time.Monday)
	return state.DataPush(dtrules.GetRTime(first.AddDate(0, 0, 6)))
}

// opEndOfWeekInZone: ( date zone -- date ) end-of-week (Mon-anchored) in zone.
func opEndOfWeekInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	first := firstOfWeekAt(t, loc, time.Monday)
	return state.DataPush(dtrules.GetRTime(first.AddDate(0, 0, 6)))
}

// opEndOfWeekStarting: ( date startDay -- date ) end-of-week with explicit
// start in UTC.
func opEndOfWeekStarting(state dtrules.State) error {
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startDay, err := parseStartDay(startObj.StringValue())
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	first := firstOfWeekAt(t, time.UTC, startDay)
	return state.DataPush(dtrules.GetRTime(first.AddDate(0, 0, 6)))
}

// opEndOfWeekStartingInZone: ( date startDay zone -- date ) end-of-week with
// explicit start in zone.
func opEndOfWeekStartingInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startDay, err := parseStartDay(startObj.StringValue())
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	first := firstOfWeekAt(t, loc, startDay)
	return state.DataPush(dtrules.GetRTime(first.AddDate(0, 0, 6)))
}

// ----- Quarter / year buckets ------------------------------------------------

// opFirstOfQuarter: ( date -- date ) first-of-quarter in UTC.
func opFirstOfQuarter(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	t = t.UTC()
	first := firstMonthOfQuarter(t.Month())
	return state.DataPush(dtrules.GetRTime(time.Date(t.Year(), first, 1, 0, 0, 0, 0, time.UTC)))
}

// opFirstOfQuarterInZone: ( date zone -- date ) first-of-quarter in zone.
func opFirstOfQuarterInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	t = t.In(loc)
	first := firstMonthOfQuarter(t.Month())
	return state.DataPush(dtrules.GetRTime(time.Date(t.Year(), first, 1, 0, 0, 0, 0, loc)))
}

// opEndOfQuarter: ( date -- date ) last day of quarter in UTC.
func opEndOfQuarter(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	t = t.UTC()
	first := firstMonthOfQuarter(t.Month())
	nextQ := time.Date(t.Year(), first+3, 1, 0, 0, 0, 0, time.UTC)
	return state.DataPush(dtrules.GetRTime(nextQ.AddDate(0, 0, -1)))
}

// opEndOfQuarterInZone: ( date zone -- date ) last day of quarter in zone.
func opEndOfQuarterInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	t = t.In(loc)
	first := firstMonthOfQuarter(t.Month())
	nextQ := time.Date(t.Year(), first+3, 1, 0, 0, 0, 0, loc)
	return state.DataPush(dtrules.GetRTime(nextQ.AddDate(0, 0, -1)))
}

// opEndOfYear: ( date -- date ) last day of year (Dec 31) in UTC.
func opEndOfYear(state dtrules.State) error {
	t, err := popDate(state)
	if err != nil {
		return err
	}
	t = t.UTC()
	return state.DataPush(dtrules.GetRTime(time.Date(t.Year(), 12, 31, 0, 0, 0, 0, time.UTC)))
}

// opEndOfYearInZone: ( date zone -- date ) last day of year (Dec 31) in zone.
func opEndOfYearInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	t, err := popDate(state)
	if err != nil {
		return err
	}
	t = t.In(loc)
	return state.DataPush(dtrules.GetRTime(time.Date(t.Year(), 12, 31, 0, 0, 0, 0, loc)))
}

// ----- Calendar comparisons --------------------------------------------------
//
// All same_calendar_*_inzone ops have the stack signature:
//   ( d1 d2 zone -- bool )
// (and the *starting* week variant is ( d1 d2 startDay zone -- bool )).
// Arguments are popped in reverse: zone, then startDay if applicable, then d2,
// then d1.

func popTwoDatesAndZone(state dtrules.State) (time.Time, time.Time, *time.Location, error) {
	loc, err := popZone(state)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	d2, err := popDate(state)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	d1, err := popDate(state)
	if err != nil {
		return time.Time{}, time.Time{}, nil, err
	}
	return d1, d2, loc, nil
}

func opSameCalendarDayInZone(state dtrules.State) error {
	d1, d2, loc, err := popTwoDatesAndZone(state)
	if err != nil {
		return err
	}
	a := d1.In(loc)
	b := d2.In(loc)
	same := a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
	return state.DataPush(dtrules.GetRBoolean(same))
}

func opSameCalendarWeekInZone(state dtrules.State) error {
	d1, d2, loc, err := popTwoDatesAndZone(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(sameWeek(d1, d2, loc, time.Monday)))
}

func opSameCalendarWeekStartingInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startDay, err := parseStartDay(startObj.StringValue())
	if err != nil {
		return err
	}
	d2, err := popDate(state)
	if err != nil {
		return err
	}
	d1, err := popDate(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(sameWeek(d1, d2, loc, startDay)))
}

func sameWeek(a, b time.Time, loc *time.Location, startDay time.Weekday) bool {
	fa := firstOfWeekAt(a, loc, startDay)
	fb := firstOfWeekAt(b, loc, startDay)
	return fa.Equal(fb)
}

func opSameCalendarMonthInZone(state dtrules.State) error {
	d1, d2, loc, err := popTwoDatesAndZone(state)
	if err != nil {
		return err
	}
	a := d1.In(loc)
	b := d2.In(loc)
	return state.DataPush(dtrules.GetRBoolean(a.Year() == b.Year() && a.Month() == b.Month()))
}

func opSameCalendarQuarterInZone(state dtrules.State) error {
	d1, d2, loc, err := popTwoDatesAndZone(state)
	if err != nil {
		return err
	}
	a := d1.In(loc)
	b := d2.In(loc)
	same := a.Year() == b.Year() && quarterFromMonth(a.Month()) == quarterFromMonth(b.Month())
	return state.DataPush(dtrules.GetRBoolean(same))
}

func opSameCalendarYearInZone(state dtrules.State) error {
	d1, d2, loc, err := popTwoDatesAndZone(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(d1.In(loc).Year() == d2.In(loc).Year()))
}
