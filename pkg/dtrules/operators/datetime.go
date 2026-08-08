// Copyright 2004-2011 DTRules.com, Inc.
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
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	Register("now", opNow)
	Register("today", opToday)
	Register("newdate", opNewDate)
	Register("getyear", opGetYear)
	Register("getmonth", opGetMonth)
	Register("getday", opGetDay)
	Register("adddays", opAddDays)
	Register("addmonths", opAddMonths)
	Register("addyears", opAddYears)
	Register("daysbetween", opDaysBetween)
	Alias("daysbetween", "numberofdays")
	Register("monthsbetween", opMonthsBetween)
	Alias("monthsbetween", "numberofmonths")
	Register("yearsbetween", opYearsBetween)
	Alias("yearsbetween", "numberofyears")
	Register("firstofmonth", opFirstOfMonth)
	Register("firstofyear", opFirstOfYear)
	Register("endofmonth", opEndOfMonth)
	// yearof/monthof/dayof are aliases for getyear/getmonth/getday —
	// identical implementation, kept for DSL/author convenience.
	Alias("getyear", "yearof")
	Alias("getmonth", "monthof")
	Alias("getday", "dayof")
	Register("getdaysinyear", opGetDaysInYear)
	Register("getdaysinmonth", opGetDaysInMonth)
	Register("getdayofmonth", opGetDayOfMonth)
	Register("d<", opDateLT)
	Register("d>", opDateGT)
	Register("d<=", opDateLE)
	Register("d>=", opDateGE)
	Register("d==", opDateEQ)
	Register("d+", opDatePlus)
	Register("d-", opDateMinus)
	Register("getdate", opGetDate)
	Register("gettimestamp", opGetTimestamp)
	// Date interval operators
	Register("days", opDays)
	Register("months", opMonths)
	Register("years", opYears)

	// Phase 2 of #743: explicit timezone variants. Each pops a zone string
	// from the data stack first, then any prior date arg, and operates in
	// the resolved zone. Plain ops above stay UTC-anchored.
	Register("todayinzone", opTodayInZone)
	Register("currentdateinzone", opCurrentDateInZone)
	Register("dateinzone", opDateInZone)
	Register("getyearinzone", opGetYearInZone)
	Alias("getyearinzone", "yearofinzone")
	Register("firstofmonthinzone", opFirstOfMonthInZone)
	Register("firstofyearinzone", opFirstOfYearInZone)
	Register("endofmonthinzone", opEndOfMonthInZone)
	Register("getdaysinyearinzone", opGetDaysInYearInZone)
	Register("getdaysinmonthinzone", opGetDaysInMonthInZone)
	Register("getdayofmonthinzone", opGetDayOfMonthInZone)
}

// popZone pops a zone string from the data stack and resolves it. Used by
// the *inzone family of operators introduced in Phase 2 of #743.
func popZone(state dtrules.State) (*time.Location, error) {
	zoneObj, err := state.DataPop()
	if err != nil {
		return nil, err
	}
	return dtrules.ResolveZone(zoneObj.StringValue())
}

// opTodayInZone: ( zone -- date ) pushes today's date in the given zone.
func opTodayInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return state.DataPush(dtrules.GetRTime(today))
}

// opCurrentDateInZone: ( zone -- date ) pushes the current instant interpreted
// in the given zone. Same instant as `current date`, but stamped with the
// requested location so downstream extraction reads the local calendar.
func opCurrentDateInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(time.Now().In(loc)))
}

// opDateInZone: ( date zone -- date ) returns the same instant rewrapped in
// the given zone. Used by the `<dexpr> in zone <strexpr>` rewrap form so
// component extractions read the local calendar.
func opDateInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(t.In(loc)))
}

// opGetYearInZone: ( date zone -- year ) extracts the year in the given zone.
func opGetYearInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.In(loc).Year()))
}

// opFirstOfMonthInZone: ( date zone -- date ) first-of-month bucket in zone.
func opFirstOfMonthInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.In(loc)
	result := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	return state.DataPush(dtrules.GetRTime(result))
}

// opFirstOfYearInZone: ( date zone -- date ) first-of-year bucket in zone.
func opFirstOfYearInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.In(loc)
	result := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, loc)
	return state.DataPush(dtrules.GetRTime(result))
}

// opEndOfMonthInZone: ( date zone -- date ) last-day-of-month bucket in zone.
func opEndOfMonthInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.In(loc)
	nextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
	result := nextMonth.AddDate(0, 0, -1)
	return state.DataPush(dtrules.GetRTime(result))
}

// opGetDaysInYearInZone: ( date zone -- int ) days-in-year in the given zone.
func opGetDaysInYearInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	year := t.In(loc).Year()
	if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
		return state.DataPush(dtrules.GetRIntegerValueFromInt(366))
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(365))
}

// opGetDaysInMonthInZone: ( date zone -- int ) days-in-month in the given zone.
func opGetDaysInMonthInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.In(loc)
	nextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
	lastDay := nextMonth.AddDate(0, 0, -1)
	return state.DataPush(dtrules.GetRIntegerValueFromInt(lastDay.Day()))
}

// opGetDayOfMonthInZone: ( date zone -- int ) day-of-month read in zone.
func opGetDayOfMonthInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.In(loc).Day()))
}

// opNow: ( -- date ) pushes the current date/time
func opNow(state dtrules.State) error {
	return state.DataPush(dtrules.GetRTime(time.Now()))
}

// opToday: ( -- date ) pushes today's date (midnight)
//
// Phase 1 of #743: anchored to UTC so the result does not depend on the
// server's local timezone. Phase 2+ will introduce explicit `in zone` syntax.
func opToday(state dtrules.State) error {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return state.DataPush(dtrules.GetRTime(today))
}

// opNewDate: ( year month day -- date ) creates a date from components
func opNewDate(state dtrules.State) error {
	dayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	monthObj, err := state.DataPop()
	if err != nil {
		return err
	}
	yearObj, err := state.DataPop()
	if err != nil {
		return err
	}

	year, err := yearObj.IntValue()
	if err != nil {
		return err
	}
	month, err := monthObj.IntValue()
	if err != nil {
		return err
	}
	day, err := dayObj.IntValue()
	if err != nil {
		return err
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return state.DataPush(dtrules.GetRTime(date))
}

// opGetYear: ( date -- year ) gets the year from a date
//
// Phase 1 of #743: extraction is performed in UTC so the result is stable
// regardless of the input date's stored zone or the server's local zone.
func opGetYear(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.Year()))
}

// opGetMonth: ( date -- month ) gets the month from a date (1-12)
//
// Phase 1 of #743: extraction is performed in UTC.
func opGetMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	return state.DataPush(dtrules.GetRIntegerValueFromInt(int(t.Month())))
}

// opGetDay: ( date -- day ) gets the day of month from a date
//
// Phase 1 of #743: extraction is performed in UTC.
func opGetDay(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.Day()))
}

// opAddDays: ( date days -- date ) adds days to a date
func opAddDays(state dtrules.State) error {
	daysObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}

	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	days, err := daysObj.IntValue()
	if err != nil {
		return err
	}

	result := t.AddDate(0, 0, days)
	return state.DataPush(dtrules.GetRTime(result))
}

// opAddMonths: ( date months -- date ) adds months to a date
func opAddMonths(state dtrules.State) error {
	monthsObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}

	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	months, err := monthsObj.IntValue()
	if err != nil {
		return err
	}

	result := t.AddDate(0, months, 0)
	return state.DataPush(dtrules.GetRTime(result))
}

// opAddYears: ( date years -- date ) adds years to a date
func opAddYears(state dtrules.State) error {
	yearsObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}

	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	years, err := yearsObj.IntValue()
	if err != nil {
		return err
	}

	result := t.AddDate(years, 0, 0)
	return state.DataPush(dtrules.GetRTime(result))
}

// opDaysBetween: ( date1 date2 -- days ) returns days between two dates
func opDaysBetween(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}

	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}

	duration := t2.Sub(t1)
	days := int(duration.Hours() / 24)
	return state.DataPush(dtrules.GetRIntegerValueFromInt(days))
}

// opMonthsBetween: ( date1 date2 -- months ) returns whole months between dates
func opMonthsBetween(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}

	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}

	months := (t2.Year()-t1.Year())*12 + int(t2.Month()) - int(t1.Month())
	return state.DataPush(dtrules.GetRIntegerValueFromInt(months))
}

// opYearsBetween: ( date1 date2 -- years ) returns whole years between dates
func opYearsBetween(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}

	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}

	years := t2.Year() - t1.Year()
	// Adjust if we haven't reached the anniversary yet
	if t2.Month() < t1.Month() || (t2.Month() == t1.Month() && t2.Day() < t1.Day()) {
		years--
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(years))
}

// opFirstOfMonth: ( date -- date2 ) returns first day of the month
//
// Phase 1 of #743: input is normalized to UTC and the bucket is constructed
// in UTC, so first-of-month is consistent regardless of the input's zone.
func opFirstOfMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	result := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return state.DataPush(dtrules.GetRTime(result))
}

// opFirstOfYear: ( date -- date2 ) returns first day of the year
//
// Phase 1 of #743: input is normalized to UTC.
func opFirstOfYear(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	result := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	return state.DataPush(dtrules.GetRTime(result))
}

// opEndOfMonth: ( date -- date2 ) returns last day of the month
//
// Phase 1 of #743: input is normalized to UTC.
func opEndOfMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	// Go to next month, then subtract one day
	nextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	result := nextMonth.AddDate(0, 0, -1)
	return state.DataPush(dtrules.GetRTime(result))
}

// opGetDaysInYear: ( date -- int ) returns number of days in year (365 or 366)
//
// Phase 1 of #743: year is read in UTC so the leap-year decision matches
// the UTC calendar regardless of the input's stored zone.
func opGetDaysInYear(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	year := t.UTC().Year()
	// Check if leap year
	if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
		return state.DataPush(dtrules.GetRIntegerValueFromInt(366))
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(365))
}

// opGetDaysInMonth: ( date -- int ) returns number of days in month
//
// Phase 1 of #743: month/year are read in UTC so the count reflects the
// UTC calendar month rather than the input's stored zone.
func opGetDaysInMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	// Go to first of next month and subtract a day to get last day of current month
	nextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	lastDay := nextMonth.AddDate(0, 0, -1)
	return state.DataPush(dtrules.GetRIntegerValueFromInt(lastDay.Day()))
}

// opGetDayOfMonth: ( date -- int ) returns day of month
//
// Phase 1 of #743: extraction is performed in UTC.
func opGetDayOfMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t = t.UTC()
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.Day()))
}

// opDateLT: ( date1 date2 -- boolean ) returns true if date1 < date2
func opDateLT(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(t1.Before(t2)))
}

// opDateGT: ( date1 date2 -- boolean ) returns true if date1 > date2
func opDateGT(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(t1.After(t2)))
}

// opDateLE: ( date1 date2 -- boolean ) returns true if date1 <= date2
func opDateLE(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(!t1.After(t2)))
}

// opDateGE: ( date1 date2 -- boolean ) returns true if date1 >= date2
func opDateGE(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(!t1.Before(t2)))
}

// opDateEQ: ( date1 date2 -- boolean ) returns true if dates are equal
func opDateEQ(state dtrules.State) error {
	date2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	date1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	t1, err := date1Obj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := date2Obj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(t1.Equal(t2)))
}

// opDatePlus: ( date interval -- date ) or ( date1 date2 -- date )
// If the second operand is an interval, adds the interval to the date.
// Otherwise, adds two dates (legacy behavior).
func opDatePlus(state dtrules.State) error {
	operandObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}

	// Check if operand is an interval
	if interval, ok := dtrules.AsInterval(operandObj); ok {
		t, err := dateObj.TimeValue()
		if err != nil {
			return err
		}
		var result time.Time
		switch interval.GetUnit() {
		case dtrules.IntervalDays:
			result = t.AddDate(0, 0, interval.GetAmount())
		case dtrules.IntervalMonths:
			result = t.AddDate(0, interval.GetAmount(), 0)
		case dtrules.IntervalYears:
			result = t.AddDate(interval.GetAmount(), 0, 0)
		}
		return state.DataPush(dtrules.GetRTime(result))
	}

	// Legacy behavior: add two dates
	t1, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := operandObj.TimeValue()
	if err != nil {
		return err
	}
	result := time.Unix(0, t1.UnixNano()+t2.UnixNano())
	return state.DataPush(dtrules.GetRTime(result))
}

// opDateMinus: ( date interval -- date ) or ( date1 date2 -- date )
// If the second operand is an interval, subtracts the interval from the date.
// Otherwise, subtracts date2 from date1 (legacy behavior).
func opDateMinus(state dtrules.State) error {
	operandObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}

	// Check if operand is an interval
	if interval, ok := dtrules.AsInterval(operandObj); ok {
		t, err := dateObj.TimeValue()
		if err != nil {
			return err
		}
		var result time.Time
		switch interval.GetUnit() {
		case dtrules.IntervalDays:
			result = t.AddDate(0, 0, -interval.GetAmount())
		case dtrules.IntervalMonths:
			result = t.AddDate(0, -interval.GetAmount(), 0)
		case dtrules.IntervalYears:
			result = t.AddDate(-interval.GetAmount(), 0, 0)
		}
		return state.DataPush(dtrules.GetRTime(result))
	}

	// Legacy behavior: subtract two dates
	t1, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	t2, err := operandObj.TimeValue()
	if err != nil {
		return err
	}
	result := time.Unix(0, t1.UnixNano()-t2.UnixNano())
	return state.DataPush(dtrules.GetRTime(result))
}

// opGetDate: ( -- date ) returns current system date
func opGetDate(state dtrules.State) error {
	return state.DataPush(dtrules.GetRTime(time.Now()))
}

// opGetTimestamp: ( date -- string ) returns timestamp string
func opGetTimestamp(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.NewRString(t.Format("2006-01-02 15:04:05.000000000")))
}

// opDays: ( n -- interval ) creates a days interval
func opDays(state dtrules.State) error {
	nObj, err := state.DataPop()
	if err != nil {
		return err
	}
	n, err := nObj.IntValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.NewRInterval(n, dtrules.IntervalDays))
}

// opMonths: ( n -- interval ) creates a months interval
func opMonths(state dtrules.State) error {
	nObj, err := state.DataPop()
	if err != nil {
		return err
	}
	n, err := nObj.IntValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.NewRInterval(n, dtrules.IntervalMonths))
}

// opYears: ( n -- interval ) creates a years interval
func opYears(state dtrules.State) error {
	nObj, err := state.DataPop()
	if err != nil {
		return err
	}
	n, err := nObj.IntValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.NewRInterval(n, dtrules.IntervalYears))
}
