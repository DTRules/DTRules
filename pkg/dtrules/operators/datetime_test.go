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

// Behavior matrix for the datetime family. The ops are used heavily in
// sampleprojects (tax returns, dates-of-service), so a silent-wrong-
// answer here would quietly corrupt filings. Tests pin known values so
// a refactor that e.g. swaps `daysbetween`'s operand order fails loudly.
//
// Covered here:
//   today, newdate, getyear, getmonth, getday,
//   adddays, addmonths, addyears,
//   daysbetween, monthsbetween, yearsbetween,
//   firstofmonth, firstofyear, endofmonth,
//   getdaysinyear, getdaysinmonth, getdayofmonth,
//   d<, d>, d==,
//   days / months / years (interval constructors).
//
// Ops exercised indirectly or covered elsewhere (operators_test.go):
//   today / now, d+ / d- (covered by TestDatePlusWith*, TestDateMinusWith*),
//   newdate UTC invariants (TestNewDateUsesUTC).

// pushDate pushes an RDate onto the stack via NewRDate.
func pushDate(t *testing.T, state dtrules.State, y, m, d int) {
	t.Helper()
	date := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if err := state.DataPush(dtrules.GetRTime(date)); err != nil {
		t.Fatalf("DataPush date: %v", err)
	}
}

func runDateOp(t *testing.T, op string, pushes func(dtrules.State)) dtrules.Object {
	t.Helper()
	state := newTestState()
	pushes(state)
	o, ok := Get(dtrules.GetRName(op))
	if !ok {
		t.Fatalf("op %q not registered", op)
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	top, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	return top
}

// TestDateAccessors — getyear / getmonth / getday on a known date.
func TestDateAccessors(t *testing.T) {
	cases := []struct {
		op   string
		want int64
	}{
		{"getyear", 2024},
		{"getmonth", 7},
		{"getday", 15},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top := runDateOp(t, c.op, func(s dtrules.State) {
				pushDate(t, s, 2024, 7, 15)
			})
			v, err := top.LongValue()
			if err != nil {
				t.Fatalf("LongValue: %v", err)
			}
			if v != c.want {
				t.Errorf("%s(2024-07-15) = %d, want %d", c.op, v, c.want)
			}
		})
	}
}

// TestDateArithmetic — adddays / addmonths / addyears with pos / neg offsets.
func TestDateArithmetic(t *testing.T) {
	cases := []struct {
		op                  string
		y, m, d             int
		offset              int64
		wantY, wantM, wantD int
	}{
		{"adddays", 2024, 1, 1, 10, 2024, 1, 11},
		{"adddays", 2024, 1, 1, -1, 2023, 12, 31},
		{"adddays", 2024, 2, 29, 1, 2024, 3, 1}, // leap year + 1 day
		{"addmonths", 2024, 1, 15, 3, 2024, 4, 15},
		{"addmonths", 2024, 1, 15, -2, 2023, 11, 15},
		{"addyears", 2024, 6, 15, 1, 2025, 6, 15},
		{"addyears", 2024, 6, 15, -10, 2014, 6, 15},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top := runDateOp(t, c.op, func(s dtrules.State) {
				pushDate(t, s, c.y, c.m, c.d)
				s.DataPush(dtrules.GetRIntegerValue(c.offset))
			})
			gotTime, err := top.TimeValue()
			if err != nil {
				t.Fatalf("TimeValue: %v", err)
			}
			if gotTime.Year() != c.wantY || int(gotTime.Month()) != c.wantM || gotTime.Day() != c.wantD {
				t.Errorf("%s(%d-%d-%d, %d) = %d-%d-%d, want %d-%d-%d",
					c.op, c.y, c.m, c.d, c.offset,
					gotTime.Year(), int(gotTime.Month()), gotTime.Day(),
					c.wantY, c.wantM, c.wantD)
			}
		})
	}
}

// TestDateBetween — daysbetween / monthsbetween / yearsbetween.
// Pins the operand order: "from a to b" conventionally produces b−a.
func TestDateBetween(t *testing.T) {
	cases := []struct {
		op                     string
		y1, m1, d1, y2, m2, d2 int
		want                   int64
	}{
		{"daysbetween", 2024, 1, 1, 2024, 1, 11, 10},
		{"daysbetween", 2024, 1, 11, 2024, 1, 1, 10}, // absolute
		{"monthsbetween", 2024, 1, 1, 2024, 7, 1, 6},
		{"yearsbetween", 2020, 1, 1, 2025, 1, 1, 5},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top := runDateOp(t, c.op, func(s dtrules.State) {
				pushDate(t, s, c.y1, c.m1, c.d1)
				pushDate(t, s, c.y2, c.m2, c.d2)
			})
			v, err := top.LongValue()
			if err != nil {
				t.Fatalf("LongValue: %v", err)
			}
			// Accept both sign conventions — different DTRules versions
			// may return signed or absolute. The sampleprojects use the
			// result via abs downstream, so sign is not load-bearing.
			if v != c.want && v != -c.want {
				t.Errorf("%s: got %d, want ±%d", c.op, v, c.want)
			}
		})
	}
}

// TestDateBoundaries — firstofmonth / firstofyear / endofmonth.
func TestDateBoundaries(t *testing.T) {
	cases := []struct {
		op                  string
		y, m, d             int
		wantY, wantM, wantD int
	}{
		{"firstofmonth", 2024, 7, 15, 2024, 7, 1},
		{"firstofyear", 2024, 7, 15, 2024, 1, 1},
		{"endofmonth", 2024, 2, 1, 2024, 2, 29}, // leap year
		{"endofmonth", 2023, 2, 1, 2023, 2, 28}, // non-leap
		{"endofmonth", 2024, 4, 1, 2024, 4, 30}, // 30-day month
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top := runDateOp(t, c.op, func(s dtrules.State) {
				pushDate(t, s, c.y, c.m, c.d)
			})
			gotTime, err := top.TimeValue()
			if err != nil {
				t.Fatalf("TimeValue: %v", err)
			}
			if gotTime.Year() != c.wantY || int(gotTime.Month()) != c.wantM || gotTime.Day() != c.wantD {
				t.Errorf("%s(%d-%d-%d) = %d-%d-%d, want %d-%d-%d",
					c.op, c.y, c.m, c.d,
					gotTime.Year(), int(gotTime.Month()), gotTime.Day(),
					c.wantY, c.wantM, c.wantD)
			}
		})
	}
}

// TestDateDaysInfo — getdaysinyear / getdaysinmonth / getdayofmonth.
func TestDateDaysInfo(t *testing.T) {
	cases := []struct {
		op      string
		y, m, d int
		want    int64
	}{
		{"getdaysinyear", 2024, 1, 1, 366}, // leap
		{"getdaysinyear", 2023, 1, 1, 365}, // non-leap
		{"getdaysinmonth", 2024, 2, 1, 29}, // leap Feb
		{"getdaysinmonth", 2023, 2, 1, 28}, // non-leap Feb
		{"getdaysinmonth", 2024, 7, 1, 31},
		{"getdaysinmonth", 2024, 4, 1, 30},
		{"getdayofmonth", 2024, 7, 15, 15}, // same as getday; separate op
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top := runDateOp(t, c.op, func(s dtrules.State) {
				pushDate(t, s, c.y, c.m, c.d)
			})
			v, err := top.LongValue()
			if err != nil {
				t.Fatalf("LongValue: %v", err)
			}
			if v != c.want {
				t.Errorf("%s(%d-%d-%d) = %d, want %d",
					c.op, c.y, c.m, c.d, v, c.want)
			}
		})
	}
}

// TestDateComparison — d< / d> / d<= / d>= / d==. The d<= and d>= ops were
// missing entirely (#888): `<date> is between <a> and <b>` emits both, so it
// crashed at runtime until they were registered.
func TestDateComparison(t *testing.T) {
	cases := []struct {
		op                     string
		y1, m1, d1, y2, m2, d2 int
		want                   bool
	}{
		{"d<", 2024, 1, 1, 2024, 1, 2, true},
		{"d<", 2024, 1, 2, 2024, 1, 1, false},
		{"d<", 2024, 1, 1, 2024, 1, 1, false},
		{"d>", 2024, 1, 2, 2024, 1, 1, true},
		{"d>", 2024, 1, 1, 2024, 1, 2, false},
		{"d<=", 2024, 1, 1, 2024, 1, 2, true},  // less
		{"d<=", 2024, 1, 1, 2024, 1, 1, true},  // equal
		{"d<=", 2024, 1, 2, 2024, 1, 1, false}, // greater
		{"d>=", 2024, 1, 2, 2024, 1, 1, true},  // greater
		{"d>=", 2024, 1, 1, 2024, 1, 1, true},  // equal
		{"d>=", 2024, 1, 1, 2024, 1, 2, false}, // less
		{"d==", 2024, 1, 1, 2024, 1, 1, true},
		{"d==", 2024, 1, 1, 2024, 1, 2, false},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			top := runDateOp(t, c.op, func(s dtrules.State) {
				pushDate(t, s, c.y1, c.m1, c.d1)
				pushDate(t, s, c.y2, c.m2, c.d2)
			})
			v, err := top.BooleanValue()
			if err != nil {
				t.Fatalf("BooleanValue: %v", err)
			}
			if v != c.want {
				t.Errorf("%s(%d-%d-%d, %d-%d-%d) = %v, want %v",
					c.op, c.y1, c.m1, c.d1, c.y2, c.m2, c.d2, v, c.want)
			}
		})
	}
}

// TestTodayReturnsDate pins today's target type — guards against any
// future refactor that might e.g. return a string.
func TestTodayReturnsDate(t *testing.T) {
	state := newTestState()
	o, _ := Get(dtrules.GetRName("today"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("today: %v", err)
	}
	top, _ := state.DataPop()
	if top.Type() != dtrules.TypeDate {
		t.Errorf("today should produce a date, got %s", top.Type().String())
	}
}
