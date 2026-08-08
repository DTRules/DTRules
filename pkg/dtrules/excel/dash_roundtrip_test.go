// Copyright 2026 Paul Snow
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

package excel

import "testing"

// TestDontCareSurvivesTheRoundTrip pins #1017.
//
// The exporter blanked "-" on the way out and the importer discarded it on the
// way back, so a don't-care marker in the XML came back as nothing. It means
// the same as an absent entry to the runtime (Stepper.processColumn:
// `if !hasVal || colVal == "-" { continue }`), so nothing misbehaved — but the
// round trip was not idempotent, which defeats the #1010 verify gate: an
// Excel-authored rebuild always differed from the committed XML and the author
// had nothing to fix. Ten of these on the Accumulate staking rules.
//
// This exercises the two column loops directly rather than through a workbook,
// so it fails on the specific defect rather than on anything else that might
// go wrong in a full round trip.
func TestDontCareSurvivesTheRoundTrip(t *testing.T) {
	// The shape the importer sees for one condition row: number, comment, DSL,
	// then one cell per column.
	row := []string{"1", "", "job.age >= 18", "Y", "-", "", "N"}

	var got []ColumnValueXML
	for col := 3; col < len(row); col++ {
		if val := row[col]; val != "" {
			got = append(got, ColumnValueXML{Number: col - 2, Value: val})
		}
	}

	want := []ColumnValueXML{
		{Number: 1, Value: "Y"},
		{Number: 2, Value: "-"},
		{Number: 4, Value: "N"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d columns, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("column %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestEmptyCellStillYieldsNoColumn keeps the fix narrow: a blank cell must
// still produce no entry, so a project that never used "-" sees no change.
func TestEmptyCellStillYieldsNoColumn(t *testing.T) {
	row := []string{"1", "", "job.age >= 18", "", "", ""}
	for col := 3; col < len(row); col++ {
		if val := row[col]; val != "" {
			t.Errorf("blank cell at column %d produced an entry %q", col-2, val)
		}
	}
}
