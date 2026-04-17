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

package session

import (
	"testing"
	"time"
)

func TestDateParserRFC3339(t *testing.T) {
	parser := NewDateParser()

	tests := []struct {
		input   string
		wantH   int
		wantM   int
		wantS   int
		wantNs  int
	}{
		{"2026-04-17T21:05:30Z", 21, 5, 30, 0},
		{"2026-04-17 21:05:30", 21, 5, 30, 0},
		{"2026-04-17T21:05:30.123456789Z", 21, 5, 30, 123456789},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parser.GetDate(tt.input)
			if err != nil {
				t.Fatalf("GetDate failed for %q: %v", tt.input, err)
			}
			utc := result.UTC()
			if utc.Hour() != tt.wantH || utc.Minute() != tt.wantM || utc.Second() != tt.wantS {
				t.Errorf("time: got %02d:%02d:%02d, want %02d:%02d:%02d", utc.Hour(), utc.Minute(), utc.Second(), tt.wantH, tt.wantM, tt.wantS)
			}
			if tt.wantNs != 0 && utc.Nanosecond() != tt.wantNs {
				t.Errorf("nanoseconds: got %d, want %d", utc.Nanosecond(), tt.wantNs)
			}
		})
	}
}

func TestDateParserPureDateRegressionTimestamp(t *testing.T) {
	parser := NewDateParser()
	result, err := parser.GetDate("2026-04-17")
	if err != nil {
		t.Fatalf("pure date parse failed: %v", err)
	}
	utc := result.UTC()
	if utc.Hour() != 0 || utc.Minute() != 0 || utc.Second() != 0 {
		t.Errorf("pure date should have zero time: %v", utc)
	}
}

// TestSyncTimeFixture simulates the staking sync_time scenario:
// a date-typed field receives an ISO-8601 timestamp value via the date parser.
func TestSyncTimeFixture(t *testing.T) {
	parser := NewDateParser()

	// The value that would arrive from an input XML like:
	// <sync_time>2026-04-17T21:05:30.123Z</sync_time>
	input := "2026-04-17T21:05:30.123Z"

	parsed, err := parser.GetDate(input)
	if err != nil {
		t.Fatalf("sync_time parse failed for %q: %v", input, err)
	}

	utc := parsed.UTC()
	if utc.Year() != 2026 || utc.Month() != 4 || utc.Day() != 17 {
		t.Errorf("wrong date: %v", utc)
	}
	if utc.Hour() != 21 || utc.Minute() != 5 || utc.Second() != 30 {
		t.Errorf("wrong time: %v", utc)
	}
	// .123Z = 123 milliseconds = 123_000_000 nanoseconds
	if utc.Nanosecond() != 123_000_000 {
		t.Errorf("nanoseconds: got %d, want 123000000", utc.Nanosecond())
	}
}

// TestDateParserRoundTripTimestamp verifies parse -> serialize -> parse preserves the instant.
func TestDateParserRoundTripTimestamp(t *testing.T) {
	parser := NewDateParser()

	input := "2026-04-17T21:05:30Z"
	parsed, err := parser.GetDate(input)
	if err != nil {
		t.Fatalf("initial parse failed: %v", err)
	}

	// Simulate what RDate.StringValue() produces for a timestamp
	serialized := parsed.UTC().Format(time.RFC3339Nano)

	roundTripped, err := parser.GetDate(serialized)
	if err != nil {
		t.Fatalf("round-trip parse failed for %q: %v", serialized, err)
	}

	if !parsed.Equal(roundTripped) {
		t.Errorf("round-trip mismatch: orig=%v, got=%v", parsed, roundTripped)
	}
}
