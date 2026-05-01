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

// Phase 4 of #743: explicit `format(date, layout)` rendering and a
// `with dst_rule "..."` clause for component-constructor date forms with an
// `in zone` clause.
//
// `format(date, layout)` accepts Go's time.Format reference layout
// ("2006-01-02 15:04:05"). No strftime translation.
//
// dst_rule values:
//   "earlier" — for fall-back ambiguity, return the earlier (pre-transition)
//               instant. Spring-forward impossible times are an error.
//   "later"   — for fall-back ambiguity, return the later (post-transition)
//               instant. Spring-forward impossible times are an error.
//   "error"   — any ambiguous or impossible local time is a runtime error.
//
// Without `with dst_rule`, Go's default time.Date adjustment behavior applies
// (spring-forward shifts forward; fall-back picks the post-transition zone).

func init() {
	Register("dateformat", opDateFormat)
	Register("dateformatinzone", opDateFormatInZone)
	Register("newdateinzone", opNewDateInZone)
	Register("newdateinzonewithdst", opNewDateInZoneWithDST)
}

// opDateFormat: ( date layout -- string ) renders date in UTC using the given
// Go time.Format layout.
func opDateFormat(state dtrules.State) error {
	layoutObj, err := state.DataPop()
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
	out := t.UTC().Format(layoutObj.StringValue())
	return state.DataPush(dtrules.NewRString(out))
}

// opDateFormatInZone: ( date layout zone -- string ) renders date in the
// resolved zone using the given Go time.Format layout.
func opDateFormatInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	layoutObj, err := state.DataPop()
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
	out := t.In(loc).Format(layoutObj.StringValue())
	return state.DataPush(dtrules.NewRString(out))
}

// popDateComponents pops second, minute, hour, day, month, year (in that
// stack-pop order) and returns them as ints.
func popDateComponents(state dtrules.State) (year, month, day, hour, min, sec int, err error) {
	pop := func() (int, error) {
		obj, e := state.DataPop()
		if e != nil {
			return 0, e
		}
		return obj.IntValue()
	}
	if sec, err = pop(); err != nil {
		return
	}
	if min, err = pop(); err != nil {
		return
	}
	if hour, err = pop(); err != nil {
		return
	}
	if day, err = pop(); err != nil {
		return
	}
	if month, err = pop(); err != nil {
		return
	}
	if year, err = pop(); err != nil {
		return
	}
	return
}

// opNewDateInZone: ( y m d h mi s zone -- date ) builds a date from
// components in the resolved zone, applying Go's default time.Date adjustment
// for ambiguous/impossible local times.
func opNewDateInZone(state dtrules.State) error {
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	year, month, day, hour, min, sec, err := popDateComponents(state)
	if err != nil {
		return err
	}
	t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
	return state.DataPush(dtrules.GetRTime(t))
}

// opNewDateInZoneWithDST: ( y m d h mi s zone rule -- date ) builds a date
// from components in the resolved zone with the given DST disambiguation
// rule. Returns an error if the rule cannot satisfy the request (impossible
// time, or ambiguous time under "error").
func opNewDateInZoneWithDST(state dtrules.State) error {
	ruleObj, err := state.DataPop()
	if err != nil {
		return err
	}
	rule := strings.ToLower(strings.TrimSpace(ruleObj.StringValue()))
	loc, err := popZone(state)
	if err != nil {
		return err
	}
	year, month, day, hour, min, sec, err := popDateComponents(state)
	if err != nil {
		return err
	}
	t, err := newDateWithDSTRule(year, time.Month(month), day, hour, min, sec, loc, rule)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRTime(t))
}

// componentsEqual reports whether t.Date() / t.Clock() match the requested
// fields. Used to detect DST gaps and overlaps via round-trip probing.
func componentsEqual(t time.Time, year int, month time.Month, day, hour, min, sec int) bool {
	ty, tmo, td := t.Date()
	th, tmi, tsec := t.Clock()
	return ty == year && tmo == month && td == day && th == hour && tmi == min && tsec == sec
}

// newDateWithDSTRule constructs a local-time-in-zone instant and applies the
// DST disambiguation rule. Classification:
//   - impossible (spring-forward gap): the round-trip components of
//     time.Date differ from the requested components.
//   - ambiguous (fall-back overlap): two distinct instants in this zone
//     produce the same local clock. Detected by probing candidate±1h: if
//     candidate+1h matches the request, candidate is the earlier instant
//     (e.g. NY fall-back). If candidate-1h matches, candidate is the later
//     instant (e.g. Sydney fall-back, where Go's time.Date returns the
//     post-transition instant).
func newDateWithDSTRule(year int, month time.Month, day, hour, min, sec int, loc *time.Location, rule string) (time.Time, error) {
	candidate := time.Date(year, month, day, hour, min, sec, 0, loc)
	impossible := !componentsEqual(candidate, year, month, day, hour, min, sec)

	var earlier, later time.Time
	ambiguous := false
	if !impossible {
		plus := candidate.Add(time.Hour)
		minus := candidate.Add(-time.Hour)
		switch {
		case componentsEqual(plus, year, month, day, hour, min, sec):
			earlier, later = candidate, plus
			ambiguous = true
		case componentsEqual(minus, year, month, day, hour, min, sec):
			earlier, later = minus, candidate
			ambiguous = true
		}
	}

	mkErr := func(why string) error {
		return fmt.Errorf("dst_rule=%s: %04d-%02d-%02d %02d:%02d:%02d %s in zone %q",
			rule, year, int(month), day, hour, min, sec, why, loc.String())
	}

	switch rule {
	case "", "default":
		return candidate, nil
	case "error":
		if impossible {
			return time.Time{}, mkErr("does not exist (spring-forward gap)")
		}
		if ambiguous {
			return time.Time{}, mkErr("is ambiguous (fall-back overlap)")
		}
		return candidate, nil
	case "earlier":
		if impossible {
			return time.Time{}, mkErr("does not exist (spring-forward gap)")
		}
		if ambiguous {
			return earlier, nil
		}
		return candidate, nil
	case "later":
		if impossible {
			return time.Time{}, mkErr("does not exist (spring-forward gap)")
		}
		if ambiguous {
			return later, nil
		}
		return candidate, nil
	}
	return time.Time{}, fmt.Errorf("unknown dst_rule %q (expected \"earlier\", \"later\", or \"error\")", rule)
}
