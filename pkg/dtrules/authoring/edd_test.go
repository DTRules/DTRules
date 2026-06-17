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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// openMinimalProject creates a temp project dir with a bare _edd.xml and
// optionally a _dt.xml, returning an open Project and a cleanup func.
func openMinimalProject(t *testing.T, dtXML string) (*authoring.Project, func()) {
	t.Helper()
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
</entity_data_dictionary>
`
	if err := os.WriteFile(filepath.Join(xmlDir, "test_edd.xml"), []byte(eddContent), 0644); err != nil {
		t.Fatal(err)
	}

	if dtXML != "" {
		if err := os.WriteFile(filepath.Join(xmlDir, "test_dt.xml"), []byte(dtXML), 0644); err != nil {
			t.Fatal(err)
		}
	}

	p, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	return p, func() {}
}

func TestAddEntity_UniqueNames(t *testing.T) {
	p, cleanup := openMinimalProject(t, "")
	defer cleanup()

	edd := p.EDD()

	if _, err := edd.AddEntity("person"); err != nil {
		t.Fatalf("first AddEntity: %v", err)
	}
	if _, err := edd.AddEntity("person"); err == nil {
		t.Fatal("expected error for duplicate entity name, got nil")
	}
}

func TestAddAttribute_ValidType(t *testing.T) {
	p, cleanup := openMinimalProject(t, "")
	defer cleanup()

	edd := p.EDD()
	ent, err := edd.AddEntity("person")
	if err != nil {
		t.Fatal(err)
	}

	if err := ent.AddAttribute(authoring.Attribute{
		Name: "age", Type: "integer", Access: "rw",
	}); err != nil {
		t.Fatalf("valid type rejected: %v", err)
	}

	if err := ent.AddAttribute(authoring.Attribute{
		Name: "score", Type: "invented_type", Access: "rw",
	}); err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}

	// `fixed` (10⁻⁸ fp) is a first-class type — projects in the wild
	// declare e.g. `weekly_budget type="fixed"`, and any new attribute
	// authored via the SDK must be allowed to use it too. Regression
	// guard: prior to this commit, validTypes omitted "fixed" and an
	// SDK AddAttribute with Type:"fixed" was rejected as "unknown type"
	// even though the loader accepted the same value on the read path.
	if err := ent.AddAttribute(authoring.Attribute{
		Name: "weekly_budget", Type: "fixed", Access: "rw",
	}); err != nil {
		t.Fatalf("`fixed` type rejected by SDK: %v", err)
	}
}

func TestAddAttribute_CollectValidation(t *testing.T) {
	p, cleanup := openMinimalProject(t, "")
	defer cleanup()
	ent, err := p.EDD().AddEntity("patient")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		a    authoring.Attribute
		ok   bool
	}{
		{"valid number", authoring.Attribute{Name: "age", Type: "integer", Access: "rw",
			Collect: "true", QuestionText: "Age?", QuestionType: "number"}, true},
		{"valid multiple_choice", authoring.Attribute{Name: "allergic", Type: "boolean", Access: "rw",
			Collect: "true", QuestionText: "Allergic?", QuestionType: "multiple_choice",
			Options: []authoring.Option{{Value: "true", Label: "Yes"}, {Value: "false", Label: "No"}}}, true},
		{"collect on read-only", authoring.Attribute{Name: "ro", Type: "integer", Access: "r",
			Collect: "true", QuestionText: "X?", QuestionType: "number"}, false},
		{"question without collect", authoring.Attribute{Name: "q1", Type: "integer", Access: "rw",
			QuestionText: "X?", QuestionType: "number"}, false},
		{"multiple_choice without options", authoring.Attribute{Name: "mc", Type: "string", Access: "rw",
			Collect: "true", QuestionText: "Pick", QuestionType: "multiple_choice"}, false},
		{"options without multiple_choice", authoring.Attribute{Name: "opt", Type: "integer", Access: "rw",
			Collect: "true", QuestionText: "X?", QuestionType: "number",
			Options: []authoring.Option{{Value: "1"}}}, false},
		{"bad question type", authoring.Attribute{Name: "bad", Type: "integer", Access: "rw",
			Collect: "true", QuestionText: "X?", QuestionType: "slider"}, false},
		{"collect true no question text", authoring.Attribute{Name: "noq", Type: "integer", Access: "rw",
			Collect: "true", QuestionType: "number"}, false},
	}
	for _, c := range cases {
		err := ent.AddAttribute(c.a)
		if c.ok && err != nil {
			t.Errorf("%s: expected accept, got %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: expected reject, got nil", c.name)
		}
	}
}

func TestEDDRoundTrip_Collect(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "rt_edd.xml"),
		[]byte("<?xml version=\"1.0\"?>\n<entity_data_dictionary version=\"2\">\n</entity_data_dictionary>\n"), 0644); err != nil {
		t.Fatal(err)
	}

	p1, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	ent, err := p1.EDD().AddEntity("patient")
	if err != nil {
		t.Fatal(err)
	}
	if err := ent.AddAttribute(authoring.Attribute{
		Name: "penicillin_allergic", Type: "boolean", Access: "rw", Default: "false",
		Collect: "true", QuestionText: "Penicillin-allergic?", QuestionType: "multiple_choice",
		Options: []authoring.Option{{Value: "true", Label: "Yes"}, {Value: "false", Label: "No"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := p1.SaveEDD(); err != nil {
		t.Fatalf("SaveEDD: %v", err)
	}

	p2, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	ent2 := p2.EDD().Entity("patient")
	if ent2 == nil || len(ent2.Attributes) != 1 {
		t.Fatalf("entity/attr not round-tripped: %+v", ent2)
	}
	a := ent2.Attributes[0]
	if a.Collect != "true" || a.QuestionType != "multiple_choice" || a.QuestionText != "Penicillin-allergic?" {
		t.Errorf("collect/question not round-tripped: %+v", a)
	}
	if len(a.Options) != 2 || a.Options[0].Value != "true" || a.Options[0].Label != "Yes" {
		t.Errorf("options not round-tripped: %+v", a.Options)
	}
}

func TestAddAttribute_DefaultMatchesType(t *testing.T) {
	p, cleanup := openMinimalProject(t, "")
	defer cleanup()

	edd := p.EDD()
	ent, err := edd.AddEntity("account")
	if err != nil {
		t.Fatal(err)
	}

	// Non-numeric default on integer field must be rejected.
	err = ent.AddAttribute(authoring.Attribute{
		Name: "balance", Type: "integer", Default: "foo",
	})
	if err == nil {
		t.Fatal("expected error for non-integer default, got nil")
	}

	// Numeric default on integer field must be accepted.
	if err := ent.AddAttribute(authoring.Attribute{
		Name: "balance", Type: "integer", Default: "42",
	}); err != nil {
		t.Fatalf("valid integer default rejected: %v", err)
	}
}

func TestUpdateAttribute_PreservesOtherFields(t *testing.T) {
	p, cleanup := openMinimalProject(t, "")
	defer cleanup()

	edd := p.EDD()
	ent, err := edd.AddEntity("order")
	if err != nil {
		t.Fatal(err)
	}

	if err := ent.AddAttribute(authoring.Attribute{
		Name:    "amount",
		Type:    "double",
		Access:  "rw",
		Comment: "order total",
		Default: "0",
	}); err != nil {
		t.Fatal(err)
	}

	// Update only the comment; all other fields must be preserved.
	if err := ent.UpdateAttribute("amount", authoring.Attribute{Comment: "revised total"}); err != nil {
		t.Fatalf("UpdateAttribute: %v", err)
	}

	updated := edd.Entity("order")
	if updated == nil {
		t.Fatal("entity not found after update")
	}
	if len(updated.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(updated.Attributes))
	}
	a := updated.Attributes[0]
	if a.Type != "double" {
		t.Errorf("Type changed: got %q want %q", a.Type, "double")
	}
	if a.Access != "rw" {
		t.Errorf("Access changed: got %q want %q", a.Access, "rw")
	}
	if a.Default != "0" {
		t.Errorf("Default changed: got %q want %q", a.Default, "0")
	}
	if a.Comment != "revised total" {
		t.Errorf("Comment not updated: got %q", a.Comment)
	}
}

func TestDeleteEntity_FailsIfReferenced(t *testing.T) {
	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table tablename="SomeTable">
    <contexts>customer</contexts>
    <attribute_fields type="FIRST"/>
  </decision_table>
</decision_tables>
`
	p, cleanup := openMinimalProject(t, dtXML)
	defer cleanup()

	edd := p.EDD()
	if _, err := edd.AddEntity("customer"); err != nil {
		t.Fatal(err)
	}

	// Via project-level delete (cross-artifact check)
	if err := p.DeleteEntity("customer"); err == nil {
		t.Fatal("expected error deleting entity referenced by DT context, got nil")
	} else {
		t.Logf("got expected error: %v", err)
	}
}

func TestEDDRoundTrip(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	eddContent := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
</entity_data_dictionary>
`
	if err := os.WriteFile(filepath.Join(xmlDir, "rt_edd.xml"), []byte(eddContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Open, mutate, save.
	p1, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	edd1 := p1.EDD()
	ent, err := edd1.AddEntity("invoice")
	if err != nil {
		t.Fatal(err)
	}
	if err := ent.AddAttribute(authoring.Attribute{
		Name:    "total",
		Type:    "double",
		Default: "0",
		Access:  "rw",
		Comment: "invoice total",
	}); err != nil {
		t.Fatal(err)
	}
	if err := p1.SaveEDD(); err != nil {
		t.Fatalf("SaveEDD: %v", err)
	}

	// Reopen, assert mutations present.
	p2, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	edd2 := p2.EDD()
	ent2 := edd2.Entity("invoice")
	if ent2 == nil {
		t.Fatal("entity 'invoice' not found after round-trip")
	}
	if len(ent2.Attributes) != 1 {
		t.Fatalf("expected 1 attribute after round-trip, got %d", len(ent2.Attributes))
	}
	a := ent2.Attributes[0]
	if a.Name != "total" || a.Type != "double" || a.Default != "0" || a.Comment != "invoice total" {
		t.Errorf("attribute mismatch after round-trip: %+v", a)
	}
}
