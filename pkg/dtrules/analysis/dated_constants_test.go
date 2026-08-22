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

package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// A stale constant produces a plausible answer, and the scenarios agree with
// the constant they were derived from — six were found by hand and none by a
// failing test (#1140). The rule: a comment citing any year must also cite
// the project's tax year.

func writeDatedFixture(t *testing.T, taxYear string) string {
	t.Helper()
	dir := t.TempDir()
	edd := `<entity_data_dictionary version="2">
<entity name="job" number="100" access="rw">
<field name="tax_year" type="integer" subtype="" access="r" input="" default_value="` + taxYear + `" comment="Tax year"></field>
</entity>
<entity name="constants" number="200" access="r">
<field name="stale_cap" type="double" subtype="" access="r" input="" default_value="16810" comment="Maximum credit (2024)"></field>
<field name="fresh_cap" type="double" subtype="" access="r" input="" default_value="17280" comment="Maximum credit, 2025. Was 16,810, the 2024 figure."></field>
<field name="nol_2019_remaining" type="double" subtype="" access="r" input="" default_value="0" comment="NOL vintage from 2019"></field>
<field name="plain" type="double" subtype="" access="r" input="" default_value="1" comment="No year here"></field>
</entity>
</entity_data_dictionary>`
	if err := os.WriteFile(filepath.Join(dir, "t_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStaleYearIsFlaggedAndHistoryIsNot(t *testing.T) {
	w, err := AnalyzeDatedConstants(writeDatedFixture(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 1 || w[0].Field != "constants.stale_cap" {
		t.Fatalf("want exactly constants.stale_cap flagged, got %v", w)
	}
	// fresh_cap cites 2024 as history alongside 2025 — welcome, not flagged.
	// nol_2019_remaining's year is in its own name — the year IS the fact.
	// plain has no year — nothing to check.
}

// A project that declares no tax year has nothing to check against.
func TestNoDeclaredYearMeansNoFindings(t *testing.T) {
	dir := t.TempDir()
	edd := `<entity_data_dictionary version="2"><entity name="c" number="1" access="r">
<field name="x" type="double" subtype="" access="r" input="" default_value="1" comment="a 2024 figure"></field>
</entity></entity_data_dictionary>`
	if err := os.WriteFile(filepath.Join(dir, "t_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := AnalyzeDatedConstants(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 0 {
		t.Errorf("no tax_year declared, nothing to check: %v", w)
	}
}
