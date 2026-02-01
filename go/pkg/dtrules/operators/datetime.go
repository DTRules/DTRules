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

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
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
	Register("datecmp", opDateCmp)
	Register("firstofmonth", opFirstOfMonth)
	Register("firstofyear", opFirstOfYear)
	Register("endofmonth", opEndOfMonth)
	Register("yearof", opYearOf)
	Register("monthof", opMonthOf)
	Register("dayof", opDayOf)
	Register("getdaysinyear", opGetDaysInYear)
	Register("getdaysinmonth", opGetDaysInMonth)
	Register("getdayofmonth", opGetDayOfMonth)
	Register("d<", opDateLT)
	Register("d>", opDateGT)
	Register("d==", opDateEQ)
	Register("d+", opDatePlus)
	Register("d-", opDateMinus)
	Register("getdate", opGetDate)
	Register("gettimestamp", opGetTimestamp)
}

// opNow: ( -- date ) pushes the current date/time
func opNow(state dtrules.State) error {
	return state.DataPush(dtrules.GetRTime(time.Now()))
}

// opToday: ( -- date ) pushes today's date (midnight)
func opToday(state dtrules.State) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
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

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	return state.DataPush(dtrules.GetRTime(date))
}

// opGetYear: ( date -- year ) gets the year from a date
func opGetYear(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.Year()))
}

// opGetMonth: ( date -- month ) gets the month from a date (1-12)
func opGetMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(int(t.Month())))
}

// opGetDay: ( date -- day ) gets the day of month from a date
func opGetDay(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
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

// opDateCmp: ( date1 date2 -- n ) compares two dates, returns -1, 0, or 1
func opDateCmp(state dtrules.State) error {
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

	var result int
	if t1.Before(t2) {
		result = -1
	} else if t1.After(t2) {
		result = 1
	} else {
		result = 0
	}

	return state.DataPush(dtrules.GetRIntegerValueFromInt(result))
}

// opFirstOfMonth: ( date -- date2 ) returns first day of the month
func opFirstOfMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	result := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return state.DataPush(dtrules.GetRTime(result))
}

// opFirstOfYear: ( date -- date2 ) returns first day of the year
func opFirstOfYear(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	result := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	return state.DataPush(dtrules.GetRTime(result))
}

// opEndOfMonth: ( date -- date2 ) returns last day of the month
func opEndOfMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	// Go to next month, then subtract one day
	nextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	result := nextMonth.AddDate(0, 0, -1)
	return state.DataPush(dtrules.GetRTime(result))
}

// opYearOf: ( date -- int ) returns the year of the date
func opYearOf(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.Year()))
}

// opMonthOf: ( date -- int ) returns the month of the date (0-11 for Java compat)
func opMonthOf(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	// Java Calendar months are 0-11
	return state.DataPush(dtrules.GetRIntegerValueFromInt(int(t.Month()) - 1))
}

// opDayOf: ( date -- int ) returns the day of month
func opDayOf(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(t.Day()))
}

// opGetDaysInYear: ( date -- int ) returns number of days in year (365 or 366)
func opGetDaysInYear(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	year := t.Year()
	// Check if leap year
	if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
		return state.DataPush(dtrules.GetRIntegerValueFromInt(366))
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(365))
}

// opGetDaysInMonth: ( date -- int ) returns number of days in month
func opGetDaysInMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
	// Go to first of next month and subtract a day to get last day of current month
	nextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	lastDay := nextMonth.AddDate(0, 0, -1)
	return state.DataPush(dtrules.GetRIntegerValueFromInt(lastDay.Day()))
}

// opGetDayOfMonth: ( date -- int ) returns day of month
func opGetDayOfMonth(state dtrules.State) error {
	dateObj, err := state.DataPop()
	if err != nil {
		return err
	}
	t, err := dateObj.TimeValue()
	if err != nil {
		return err
	}
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

// opDatePlus: ( date1 date2 -- date ) adds two dates
func opDatePlus(state dtrules.State) error {
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
	result := time.Unix(0, t1.UnixNano()+t2.UnixNano())
	return state.DataPush(dtrules.GetRTime(result))
}

// opDateMinus: ( date1 date2 -- date ) subtracts date2 from date1
func opDateMinus(state dtrules.State) error {
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
