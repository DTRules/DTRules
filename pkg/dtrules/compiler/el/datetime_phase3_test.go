// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package el

import (
	"strings"
	"testing"
)

// Phase 3 of #743: pin grammar surface → postfix mapping for the new calendar
// comparison and week/quarter/year ops. The negative tests guard the rule
// that all `same calendar X as` forms require an `in zone <strexpr>` clause.

func TestPhase3_BucketAndComponentOps(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	cases := []struct {
		name string
		dsl  string
		want string
	}{
		// Component extractors — UTC and in-zone.
		{
			name: "get hourof plain",
			dsl:  `get hourof job.filing_date is equal to 12`,
			want: `job.filing_date gethour 12 ==`,
		},
		{
			name: "get hourof in zone",
			dsl:  `get hourof job.filing_date in zone "America/New_York" is equal to 12`,
			want: `job.filing_date "America/New_York" gethourinzone 12 ==`,
		},
		{
			name: "get minuteof",
			dsl:  `get minuteof job.filing_date is equal to 0`,
			want: `job.filing_date getminute 0 ==`,
		},
		{
			name: "get secondof",
			dsl:  `get secondof job.filing_date is equal to 0`,
			want: `job.filing_date getsecond 0 ==`,
		},
		{
			name: "get day of week",
			dsl:  `get day of week of job.filing_date is equal to 1`,
			want: `job.filing_date getdayofweek 1 ==`,
		},
		{
			name: "get day of week in zone",
			dsl:  `get day of week of job.filing_date in zone "UTC" is equal to 1`,
			want: `job.filing_date "UTC" getdayofweekinzone 1 ==`,
		},
		{
			name: "get week of year",
			dsl:  `get week of year of job.filing_date is equal to 1`,
			want: `job.filing_date getweekofyear 1 ==`,
		},
		{
			name: "get week of year in zone",
			dsl:  `get week of year of job.filing_date in zone "UTC" is equal to 1`,
			want: `job.filing_date "UTC" getweekofyearinzone 1 ==`,
		},

		// Bucket ops — week, quarter, year.
		{
			name: "first of weeks of date",
			dsl:  `first of weeks of job.filing_date is equal to job.filing_date`,
			want: `job.filing_date firstofweek job.filing_date d==`,
		},
		{
			name: "first of weeks in zone",
			dsl:  `first of weeks of job.filing_date in zone "UTC" is equal to job.filing_date`,
			want: `job.filing_date "UTC" firstofweekinzone job.filing_date d==`,
		},
		{
			name: "first of weeks starting Sunday",
			dsl:  `first of weeks of job.filing_date starting "Sunday" is equal to job.filing_date`,
			want: `job.filing_date "Sunday" firstofweekstarting job.filing_date d==`,
		},
		{
			name: "first of weeks starting + zone",
			dsl:  `first of weeks of job.filing_date starting "Sunday" in zone "UTC" is equal to job.filing_date`,
			want: `job.filing_date "Sunday" "UTC" firstofweekstartinginzone job.filing_date d==`,
		},
		{
			name: "end of weeks of date",
			dsl:  `end of weeks of job.filing_date is equal to job.filing_date`,
			want: `job.filing_date endofweek job.filing_date d==`,
		},
		{
			name: "first of quarters",
			dsl:  `first of quarters of job.filing_date is equal to job.filing_date`,
			want: `job.filing_date firstofquarter job.filing_date d==`,
		},
		{
			name: "end of quarters in zone",
			dsl:  `end of quarters of job.filing_date in zone "America/New_York" is equal to job.filing_date`,
			want: `job.filing_date "America/New_York" endofquarterinzone job.filing_date d==`,
		},
		{
			name: "end of years",
			dsl:  `end of years of job.filing_date is equal to job.filing_date`,
			want: `job.filing_date endofyear job.filing_date d==`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			postfix, err := c.CompileCondition(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			got := strings.TrimSpace(postfix)
			if got != tc.want {
				t.Fatalf("compile %q\n  got:  %q\n  want: %q", tc.dsl, got, tc.want)
			}
		})
	}
}

// TestSameCalendarOps_Compiles pins the bool calendar-comparison surface →
// postfix mapping. INZONE is mandatory in grammar, so every form here carries
// an explicit zone.
func TestSameCalendarOps_Compiles(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.start_date": TypeDate,
		"job.end_date":   TypeDate,
	})
	cases := []struct {
		name string
		dsl  string
		want string
	}{
		{
			name: "same calendar day",
			dsl:  `job.start_date is the same calendar day as job.end_date in zone "America/New_York"`,
			want: `job.start_date job.end_date "America/New_York" samecalendardayinzone`,
		},
		{
			name: "same calendar week (default Monday)",
			dsl:  `job.start_date is the same calendar week as job.end_date in zone "UTC"`,
			want: `job.start_date job.end_date "UTC" samecalendarweekinzone`,
		},
		{
			name: "same calendar week starting Sunday",
			dsl:  `job.start_date is the same calendar week as job.end_date starting "Sunday" in zone "UTC"`,
			want: `job.start_date job.end_date "Sunday" "UTC" samecalendarweekstartinginzone`,
		},
		{
			name: "same calendar month",
			dsl:  `job.start_date is the same calendar month as job.end_date in zone "Asia/Kolkata"`,
			want: `job.start_date job.end_date "Asia/Kolkata" samecalendarmonthinzone`,
		},
		{
			name: "same calendar quarter",
			dsl:  `job.start_date is the same calendar quarter as job.end_date in zone "UTC"`,
			want: `job.start_date job.end_date "UTC" samecalendarquarterinzone`,
		},
		{
			name: "same calendar year",
			dsl:  `job.start_date is the same calendar year as job.end_date in zone "UTC"`,
			want: `job.start_date job.end_date "UTC" samecalendaryearinzone`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			postfix, err := c.CompileCondition(tc.dsl)
			if err != nil {
				t.Fatalf("compile %q: %v", tc.dsl, err)
			}
			got := strings.TrimSpace(postfix)
			if got != tc.want {
				t.Fatalf("compile %q\n  got:  %q\n  want: %q", tc.dsl, got, tc.want)
			}
		})
	}
}

// TestSameCalendarOps_RequireZone verifies the negative case: omitting `in
// zone X` from a `same calendar ... as` expression must fail to compile.
func TestSameCalendarOps_RequireZone(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.start_date": TypeDate,
		"job.end_date":   TypeDate,
	})
	bad := []string{
		`job.start_date is the same calendar day as job.end_date`,
		`job.start_date is the same calendar week as job.end_date`,
		`job.start_date is the same calendar month as job.end_date`,
		`job.start_date is the same calendar quarter as job.end_date`,
		`job.start_date is the same calendar year as job.end_date`,
	}
	for _, dsl := range bad {
		t.Run(dsl, func(t *testing.T) {
			if _, err := c.CompileCondition(dsl); err == nil {
				t.Fatalf("expected compile error for %q (missing `in zone`)", dsl)
			}
		})
	}
}
