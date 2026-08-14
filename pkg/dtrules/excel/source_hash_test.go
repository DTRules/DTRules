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

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Provenance replaces asking the filesystem which file was touched last.
// Timestamps answer "which was touched", a proxy for "which changed", and the
// proxy is wrong constantly — checkout, `cp`, containers, CI (#1091).

func TestWorkbookHashChangesWithTheBytes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "wb.xlsx")
	if err := os.WriteFile(p, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := WorkbookHash(p)
	if !strings.HasPrefix(first, "sha256:") {
		t.Errorf("hash should name its algorithm, got %q", first)
	}
	if WorkbookHash(p) != first {
		t.Error("hashing the same bytes twice gave different answers")
	}
	if err := os.WriteFile(p, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if WorkbookHash(p) == first {
		t.Error("changed bytes produced the same hash")
	}
}

// A missing hash means "unknown provenance", which callers must treat as
// "fall back to timestamps" rather than as "unchanged".
func TestMissingFileHashesToEmpty(t *testing.T) {
	if got := WorkbookHash(filepath.Join(t.TempDir(), "absent.xlsx")); got != "" {
		t.Errorf("a file that does not exist hashed to %q, want empty", got)
	}
}

func TestRecordedHashNeedsAgreement(t *testing.T) {
	stamped := func(h string) DecisionTableXML {
		return DecisionTableXML{Source: &SourceXML{SourceHash: h}}
	}

	if got := RecordedWorkbookHash([]DecisionTableXML{stamped("a"), stamped("a")}); got != "a" {
		t.Errorf("tables agreeing on a stamp gave %q, want a", got)
	}
	// A file assembled from several workbooks has no single provenance, and
	// the timestamp fallback is the right answer for it.
	if got := RecordedWorkbookHash([]DecisionTableXML{stamped("a"), stamped("b")}); got != "" {
		t.Errorf("disagreeing stamps gave %q, want empty so the caller falls back", got)
	}
	if got := RecordedWorkbookHash([]DecisionTableXML{{}}); got != "" {
		t.Errorf("an unstamped table gave %q, want empty", got)
	}
	// One stamped table is enough; unstamped siblings do not veto it.
	if got := RecordedWorkbookHash([]DecisionTableXML{{}, stamped("a")}); got != "a" {
		t.Errorf("mixed stamped/unstamped gave %q, want a", got)
	}
}
