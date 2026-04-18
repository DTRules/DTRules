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

package authoring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// openMappingFixture copies the minimal fixture to a temp dir and opens it,
// so tests can mutate without affecting the checked-in testdata.
func openMappingFixture(t *testing.T) (*authoring.Project, string) {
	t.Helper()
	tmp := t.TempDir()
	xmlTmp := filepath.Join(tmp, "xml")
	if err := os.MkdirAll(xmlTmp, 0755); err != nil {
		t.Fatal(err)
	}

	src := "testdata/minimal/xml"
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(src, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xmlTmp, entry.Name()), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	p, err := authoring.OpenProject(tmp)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	return p, tmp
}

func TestAddEntry_ValidatesEnclosureExists(t *testing.T) {
	p, _ := openMappingFixture(t)
	m, err := p.Mapping()
	if err != nil {
		t.Fatalf("Mapping(): %v", err)
	}

	err = m.AddEntry(authoring.SetAttribute{
		Tag:        "foo",
		RAttribute: "foo",
		Enclosure:  "nonexistent_entity",
		Type:       "string",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent enclosure, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent_entity") {
		t.Errorf("error should name the missing enclosure, got: %v", err)
	}
}

func TestAddEntry_ValidatesAttributeExists(t *testing.T) {
	p, _ := openMappingFixture(t)
	m, err := p.Mapping()
	if err != nil {
		t.Fatalf("Mapping(): %v", err)
	}

	err = m.AddEntry(authoring.SetAttribute{
		Tag:        "nosuchfield",
		RAttribute: "nosuchfield",
		Enclosure:  "applicant",
		Type:       "string",
	})
	if err == nil {
		t.Fatal("expected error for missing attribute, got nil")
	}
	if !strings.Contains(err.Error(), "nosuchfield") {
		t.Errorf("error should name the missing attribute, got: %v", err)
	}
}

func TestAddEntry_ValidatesTypeMatch(t *testing.T) {
	p, _ := openMappingFixture(t)
	m, err := p.Mapping()
	if err != nil {
		t.Fatalf("Mapping(): %v", err)
	}

	// 'age' is declared as 'integer' in the EDD; use 'string' to trigger mismatch.
	err = m.AddEntry(authoring.SetAttribute{
		Tag:        "age",
		RAttribute: "age",
		Enclosure:  "applicant",
		Type:       "string",
	})
	if err == nil {
		t.Fatal("expected type mismatch error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "type") {
		t.Errorf("error should mention type mismatch, got: %v", err)
	}
}

func TestUpdateEntry_ByTag(t *testing.T) {
	p, _ := openMappingFixture(t)
	m, err := p.Mapping()
	if err != nil {
		t.Fatalf("Mapping(): %v", err)
	}

	// 'eligible' is boolean; update RAttribute name (keeping same enclosure+type).
	err = m.UpdateEntry("eligible", authoring.SetAttribute{
		Tag:        "eligible",
		RAttribute: "eligible",
		Enclosure:  "applicant",
		Type:       "boolean",
	})
	if err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}

	// Confirm entry still present.
	found := false
	for _, e := range m.Entries() {
		if e.Tag == "eligible" && e.Enclosure == "applicant" {
			found = true
		}
	}
	if !found {
		t.Error("updated entry not found in Entries()")
	}
}

func TestDeleteEntry(t *testing.T) {
	p, _ := openMappingFixture(t)
	m, err := p.Mapping()
	if err != nil {
		t.Fatalf("Mapping(): %v", err)
	}

	before := len(m.Entries())
	if err := m.DeleteEntry("age"); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	after := len(m.Entries())
	if after != before-1 {
		t.Errorf("expected %d entries after delete, got %d", before-1, after)
	}
	for _, e := range m.Entries() {
		if e.Tag == "age" {
			t.Error("deleted entry 'age' still present")
		}
	}
}

func TestMappingRoundTrip(t *testing.T) {
	p, tmp := openMappingFixture(t)
	m, err := p.Mapping()
	if err != nil {
		t.Fatalf("Mapping(): %v", err)
	}

	// Delete one entry and add a new one.
	if err := m.DeleteEntry("eligible"); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}

	// Reopen project and verify the mutation persisted.
	p2, err := authoring.OpenProject(tmp)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	m2, err := p2.Mapping()
	if err != nil {
		t.Fatalf("reopen Mapping(): %v", err)
	}
	for _, e := range m2.Entries() {
		if e.Tag == "eligible" {
			t.Error("deleted entry 'eligible' still present after round-trip")
		}
	}
}
