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

package dtrules

import (
	"testing"
	"time"
)

// stubDateParser is a minimal DateParser that uses Go's time.Parse with RFC3339 formats.
// It mirrors the production NewDateParser() order so tests remain independent of session/.
type stubDateParser struct{}

func (s *stubDateParser) GetDate(str string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, str); err == nil {
			return t, nil
		}
	}
	return time.Time{}, NewRulesError("DateParser", "GetDate", "Could not parse date: "+str)
}

func (s *stubDateParser) Parse(str string) (time.Time, error) {
	return s.GetDate(str)
}

// stubSession wraps the stub date parser for GetRDate usage.
type stubSession struct{}

func (s *stubSession) GetDateParser() DateParser { return &stubDateParser{} }

func getTestDate(str string) (*RDate, error) {
	parser := &stubDateParser{}
	t, err := parser.GetDate(str)
	if err != nil {
		return nil, err
	}
	return GetRTime(t), nil
}

func TestRDateParseRFC3339(t *testing.T) {
	d, err := getTestDate("2026-04-17T21:05:30Z")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tt := d.Time().UTC()
	if tt.Year() != 2026 || tt.Month() != 4 || tt.Day() != 17 {
		t.Errorf("wrong date: %v", tt)
	}
	if tt.Hour() != 21 || tt.Minute() != 5 || tt.Second() != 30 {
		t.Errorf("wrong time: %v", tt)
	}
}

func TestRDateParseSpaceSeparated(t *testing.T) {
	d, err := getTestDate("2026-04-17 21:05:30")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tt := d.Time().UTC()
	if tt.Hour() != 21 || tt.Minute() != 5 || tt.Second() != 30 {
		t.Errorf("wrong time: %v", tt)
	}
}

func TestRDateParseNanoseconds(t *testing.T) {
	d, err := getTestDate("2026-04-17T21:05:30.123456789Z")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tt := d.Time().UTC()
	if tt.Nanosecond() != 123456789 {
		t.Errorf("wrong nanoseconds: got %d, want 123456789", tt.Nanosecond())
	}
}

func TestRDateParsePureDate(t *testing.T) {
	d, err := getTestDate("2026-04-17")
	if err != nil {
		t.Fatalf("parse failed (regression): %v", err)
	}
	tt := d.Time().UTC()
	if tt.Hour() != 0 || tt.Minute() != 0 || tt.Second() != 0 || tt.Nanosecond() != 0 {
		t.Errorf("pure date should have zero time component: %v", tt)
	}
}

func TestRDateParseInvalid(t *testing.T) {
	_, err := getTestDate("not-a-date")
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestRDateStringValuePureDate(t *testing.T) {
	d := GetRTime(time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC))
	got := d.StringValue()
	if got != "2026-04-17" {
		t.Errorf("pure date StringValue = %q, want 2026-04-17", got)
	}
}

func TestRDateStringValueTimestamp(t *testing.T) {
	d := GetRTime(time.Date(2026, 4, 17, 21, 5, 30, 0, time.UTC))
	got := d.StringValue()
	if got != "2026-04-17T21:05:30Z" {
		t.Errorf("timestamp StringValue = %q, want 2026-04-17T21:05:30Z", got)
	}
}

func TestRDateComparisonTimestamps(t *testing.T) {
	earlier, _ := getTestDate("2026-04-17T12:00:00Z")
	later, _ := getTestDate("2026-04-17T18:00:00Z")

	cmp, err := earlier.Compare(later)
	if err != nil {
		t.Fatal(err)
	}
	if cmp >= 0 {
		t.Errorf("expected earlier < later, got Compare=%d", cmp)
	}
}

func TestRDatePureDateEqualsTimestampMidnight(t *testing.T) {
	pure, _ := getTestDate("2026-04-17")
	midnight, _ := getTestDate("2026-04-17T00:00:00Z")

	eq, err := pure.Equals(midnight)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Error("pure date 2026-04-17 should equal 2026-04-17T00:00:00Z")
	}
}

func TestRDatePureDateBeforeOneSecondLater(t *testing.T) {
	pure, _ := getTestDate("2026-04-17")
	oneSecLater, _ := getTestDate("2026-04-17T00:00:01Z")

	cmp, err := pure.Compare(oneSecLater)
	if err != nil {
		t.Fatal(err)
	}
	if cmp >= 0 {
		t.Errorf("2026-04-17 should be before 2026-04-17T00:00:01Z, Compare=%d", cmp)
	}
}

func TestRDateRoundTrip(t *testing.T) {
	orig, _ := getTestDate("2026-04-17T21:05:30Z")
	serialized := orig.StringValue()

	roundTripped, err := getTestDate(serialized)
	if err != nil {
		t.Fatalf("round-trip parse failed for %q: %v", serialized, err)
	}

	origTime := orig.Time().UTC().Truncate(time.Second)
	rtTime := roundTripped.Time().UTC().Truncate(time.Second)
	if !origTime.Equal(rtTime) {
		t.Errorf("round-trip mismatch: orig=%v, got=%v", origTime, rtTime)
	}
}

func TestRDateRoundTripNanoseconds(t *testing.T) {
	orig, _ := getTestDate("2026-04-17T21:05:30.123456789Z")
	serialized := orig.StringValue()

	roundTripped, err := getTestDate(serialized)
	if err != nil {
		t.Fatalf("round-trip parse failed for %q: %v", serialized, err)
	}

	if !orig.Time().Equal(roundTripped.Time()) {
		t.Errorf("nanosecond round-trip mismatch: orig=%v, got=%v", orig.Time(), roundTripped.Time())
	}
}
