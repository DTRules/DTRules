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

package decisiontable

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// TestAuthoredNameSurvivesInterning pins #1040.
//
// RNames are interned case-insensitively and the cache keeps the first
// spelling it sees, process-wide and across namespaces. So an EDD field named
// `matches_onchain`, loaded before the decision tables, made GetName() report
// a table authored as `Matches_Onchain` under the field's casing — and the
// exporter, which named the sheet from GetName(), silently renamed the table.
// The next import read the new name back, so the name could never round-trip.
//
// Resolution stays case-insensitive. Only what gets written back out changes.
func TestAuthoredNameSurvivesInterning(t *testing.T) {
	// Intern the lower-case spelling first, the way an EDD field would.
	if rn := dtrules.GetRName("matches_onchain"); rn == nil {
		t.Fatal("could not intern the lower-case name")
	}

	b := NewBuilder("Matches_Onchain", nil)
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}
	dt, err := b.Build(nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := dt.AuthoredName(); got != "Matches_Onchain" {
		t.Errorf("AuthoredName() = %q, want the spelling the author wrote", got)
	}
	// GetName is allowed to report the interned spelling — that is what makes
	// lookup case-insensitive, and it is not the bug.
	t.Logf("GetName() reports %q (interned); AuthoredName() reports %q", dt.GetName(), dt.AuthoredName())
}

// TestAuthoredNameFallsBack keeps the accessor safe for tables built by paths
// that never record a source spelling.
func TestAuthoredNameFallsBack(t *testing.T) {
	rn := dtrules.GetRName("Some_Table")
	if rn == nil {
		t.Fatal("intern failed")
	}
	dt := NewRDecisionTable(rn, nil)
	if got := dt.AuthoredName(); got != dt.GetName() {
		t.Errorf("AuthoredName() = %q, want it to fall back to GetName() = %q", got, dt.GetName())
	}
}
