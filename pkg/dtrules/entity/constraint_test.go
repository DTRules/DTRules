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

package entity

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// constrainedEntity returns a `patient` reference entity with one constrained
// field (diagnosis) and one with no constraint at all (notes).
func constrainedEntity(t *testing.T) (*REntity, *dtrules.RName, *dtrules.RName) {
	t.Helper()
	factory := NewFactory(nil)
	ref, _ := factory.FindCreateRefEntity(false, dtrules.GetRName("patient"))
	diagnosis := dtrules.GetRName("diagnosis")
	notes := dtrules.GetRName("notes")
	for _, n := range []*dtrules.RName{diagnosis, notes} {
		if e := ref.AddAttribute(n, "", dtrules.NewRString(""), true, true,
			dtrules.TypeString, "", "", "", ""); e != "" {
			t.Fatalf("AddAttribute: %s", e)
		}
	}
	entry := ref.GetEntry(diagnosis)
	entry.AuthoredName = "diagnosis"
	entry.Constraints = &FieldConstraints{
		AllowedValues: []string{"Acute Sinusitis", "Chronic Sinusitis"},
		MaxLength:     40,
	}
	return ref, diagnosis, notes
}

// TestCheckExternalWrite_Vocabulary: the shared gate accepts a member of the
// set whatever its case, and refuses anything else with an error naming the
// field, the value and the set.
func TestCheckExternalWrite_Vocabulary(t *testing.T) {
	e, diagnosis, _ := constrainedEntity(t)

	for _, ok := range []string{"Acute Sinusitis", "chronic sinusitis", "ACUTE SINUSITIS"} {
		if err := CheckExternalWrite(e, diagnosis, dtrules.NewRString(ok)); err != nil {
			t.Errorf("%q must be accepted (matching is case-insensitive): %v", ok, err)
		}
	}

	err := CheckExternalWrite(e, diagnosis, dtrules.NewRString("Banana"))
	if err == nil {
		t.Fatal("a value outside the vocabulary must be refused")
	}
	for _, want := range []string{"patient.diagnosis", "Banana", "Acute Sinusitis", "Chronic Sinusitis"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

// TestCheckExternalWrite_SkippedWhenUndeclared is the cost guarantee: a field
// that declares no constraint has no FieldConstraints to consult, so the gate
// returns on a nil check without comparing anything. Same for an attribute the
// entity does not have — Put reports that, not this.
func TestCheckExternalWrite_SkippedWhenUndeclared(t *testing.T) {
	e, _, notes := constrainedEntity(t)

	if entry := e.GetEntry(notes); entry.Constraints != nil {
		t.Fatal("an undeclared field must carry no constraints at all, not empty ones")
	}
	long := strings.Repeat("Banana ", 100)
	if err := CheckExternalWrite(e, notes, dtrules.NewRString(long)); err != nil {
		t.Errorf("an unconstrained field must accept anything: %v", err)
	}
	if err := CheckExternalWrite(e, dtrules.GetRName("no_such_field"), dtrules.NewRString("x")); err != nil {
		t.Errorf("an unknown attribute is Put's business, not the gate's: %v", err)
	}
	if err := CheckExternalWrite(nil, notes, dtrules.NewRString("x")); err != nil {
		t.Errorf("a nil entity must be a no-op: %v", err)
	}
}

// TestUnmarkCollected_RestoresDefaulted: the mark a refused answer has to
// undo. Off when tracking is off, so batch execution is untouched.
func TestUnmarkCollected_RestoresDefaulted(t *testing.T) {
	e, diagnosis, _ := constrainedEntity(t)

	e.UnmarkCollected(diagnosis) // tracking off: must not panic
	e.EnableCollectTracking()
	e.MarkCollected(diagnosis)
	if !e.IsCollected(diagnosis) {
		t.Fatal("MarkCollected did not take")
	}
	e.UnmarkCollected(diagnosis)
	if e.IsCollected(diagnosis) {
		t.Error("UnmarkCollected must put the field back to defaulted")
	}
}

// BenchmarkCheckExternalWrite_Unconstrained measures what every existing
// project pays: one map lookup and one nil check per field written from
// outside, with no comparison done.
func BenchmarkCheckExternalWrite_Unconstrained(b *testing.B) {
	factory := NewFactory(nil)
	ref, _ := factory.FindCreateRefEntity(false, dtrules.GetRName("bench"))
	notes := dtrules.GetRName("notes")
	ref.AddAttribute(notes, "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	v := dtrules.NewRString("anything at all")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := CheckExternalWrite(ref, notes, v); err != nil {
			b.Fatal(err)
		}
	}
}
