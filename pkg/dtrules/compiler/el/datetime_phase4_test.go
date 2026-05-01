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

// Phase 4 of #743: pin grammar surface → postfix mapping for the new
// format(date, layout) [in zone Z] strexpr and the
// `new date Y, M, D[, h, m, s] in zone Z [with dst_rule R]` constructor.

func TestPhase4_FormatDate_Compiles(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	cases := []struct {
		name string
		dsl  string
		want string
	}{
		{
			name: "format(date, layout) — UTC",
			dsl:  `format(job.filing_date, "2006-01-02") is equal to "2026-04-15"`,
			want: `job.filing_date "2006-01-02" dateformat "2026-04-15" streq`,
		},
		{
			name: "format(date, layout) in zone NY",
			dsl:  `format(job.filing_date, "2006-01-02 15:04 MST") in zone "America/New_York" is equal to "x"`,
			want: `job.filing_date "2006-01-02 15:04 MST" "America/New_York" dateformatinzone "x" streq`,
		},
		{
			name: "format(date, layout) in zone fixed offset",
			dsl:  `format(job.filing_date, "15:04") in zone "+05:45" is equal to "00:00"`,
			want: `job.filing_date "15:04" "+05:45" dateformatinzone "00:00" streq`,
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

func TestPhase4_NewDateInZone_Compiles(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	cases := []struct {
		name string
		dsl  string
		want string
	}{
		{
			name: "new date Y, M, D in zone (no dst_rule)",
			dsl:  `new date 2026, 4, 15 in zone "Europe/London" is equal to job.filing_date`,
			want: `2026 4 15 0 0 0 "Europe/London" newdateinzone job.filing_date d==`,
		},
		{
			name: "new date Y, M, D, h, m, s in zone",
			dsl:  `new date 2026, 4, 15, 12, 30, 0 in zone "Europe/London" is equal to job.filing_date`,
			want: `2026 4 15 12 30 0 "Europe/London" newdateinzone job.filing_date d==`,
		},
		{
			name: "new date Y, M, D in zone with dst_rule",
			dsl:  `new date 2026, 11, 1 in zone "America/New_York" with dst_rule "earlier" is equal to job.filing_date`,
			want: `2026 11 1 0 0 0 "America/New_York" "earlier" newdateinzonewithdst job.filing_date d==`,
		},
		{
			name: "new date Y, M, D, h, m, s in zone with dst_rule",
			dsl:  `new date 2026, 3, 8, 2, 30, 0 in zone "America/New_York" with dst_rule "error" is equal to job.filing_date`,
			want: `2026 3 8 2 30 0 "America/New_York" "error" newdateinzonewithdst job.filing_date d==`,
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

// TestPhase4_FormatDate_RequiresParenAndComma: malformed format() forms
// surface as compile errors, not silent fall-through.
func TestPhase4_FormatDate_RequiresParenAndComma(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	bad := []string{
		// Missing comma between date and layout.
		`format(job.filing_date "2006-01-02") is equal to "x"`,
		// Missing closing paren.
		`format(job.filing_date, "2006-01-02" is equal to "x"`,
	}
	for _, dsl := range bad {
		t.Run(dsl, func(t *testing.T) {
			if _, err := c.CompileCondition(dsl); err == nil {
				t.Fatalf("expected compile error for %q, got none", dsl)
			}
		})
	}
}

// TestPhase4_NewDate_RequiresInZone: the `new date Y, M, D` constructor
// requires `in zone <s>` — there is no plain alt at the grammar level.
func TestPhase4_NewDate_RequiresInZone(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	bad := []string{
		`new date 2026, 4, 15 is equal to job.filing_date`,
		`new date 2026, 4, 15, 12, 30, 0 is equal to job.filing_date`,
	}
	for _, dsl := range bad {
		t.Run(dsl, func(t *testing.T) {
			if _, err := c.CompileCondition(dsl); err == nil {
				t.Fatalf("expected compile error (missing in zone) for %q", dsl)
			}
		})
	}
}
