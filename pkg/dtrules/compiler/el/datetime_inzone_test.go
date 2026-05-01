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

package el

import (
	"strings"
	"testing"
)

// Phase 2 of #743: every date-construction / bucket / extraction alt that
// accepts an `in zone <strexpr>` clause must compile through the labeled
// alternative and emit the *inzone op. Plain alternatives stay UTC-anchored.

// TestInZone_Compiles pins the surface DSL → postfix mapping for the new
// labeled alternatives.
func TestInZone_Compiles(t *testing.T) {
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
			name: "current date in zone literal",
			dsl:  `current date in zone "America/New_York" is equal to job.filing_date`,
			want: `"America/New_York" currentdateinzone job.filing_date d==`,
		},
		{
			name: "first of months of date in zone",
			dsl:  `first of months of job.filing_date in zone "America/New_York" is equal to job.filing_date`,
			want: `job.filing_date "America/New_York" firstofmonthinzone job.filing_date d==`,
		},
		{
			name: "first of years of date in zone",
			dsl:  `first of years of job.filing_date in zone "UTC" is equal to job.filing_date`,
			want: `job.filing_date "UTC" firstofyearinzone job.filing_date d==`,
		},
		{
			name: "end of months of date in zone",
			dsl:  `end of months of job.filing_date in zone "Asia/Kolkata" is equal to job.filing_date`,
			want: `job.filing_date "Asia/Kolkata" endofmonthinzone job.filing_date d==`,
		},
		{
			name: "rewrap dexpr in zone",
			dsl:  `job.filing_date in zone "America/New_York" is equal to job.filing_date`,
			want: `job.filing_date "America/New_York" dateinzone job.filing_date d==`,
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

// TestInZone_GetYear pins the iexpr in-zone form. `get yearof <date> in zone X`
// must emit the date, then the zone strexpr, then the yearofinzone op.
func TestInZone_GetYear(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	got, err := c.CompileCondition(
		`get yearof job.filing_date in zone "America/New_York" is equal to 2026`,
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := `job.filing_date "America/New_York" yearofinzone 2026 ==`
	if strings.TrimSpace(got) != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// TestInZone_PreservesPlainPostfix: without `in zone`, postfix is unchanged
// from Phase 1. Belt-and-suspenders that the in-zone alts didn't accidentally
// shadow the plain ones.
func TestInZone_PreservesPlainPostfix(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.filing_date": TypeDate,
	})
	got, err := c.CompileCondition(`current date is equal to job.filing_date`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := `currentdate job.filing_date d==`
	if strings.TrimSpace(got) != want {
		t.Fatalf("plain currentdate regressed:\n  got  %q\n  want %q", got, want)
	}
}

// TestInZone_ParseError: an `in zone` clause without a string expression is
// a parse error.
func TestInZone_ParseError(t *testing.T) {
	c := NewCompiler()
	_, err := c.CompileCondition(`current date in zone is equal to current date`)
	if err == nil {
		t.Fatal("expected parse error for `in zone` with no strexpr, got nil")
	}
}
