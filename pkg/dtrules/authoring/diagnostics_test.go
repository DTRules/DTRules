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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// dupTableXML returns a minimal decision-table XML wrapping the given names.
func dupTableXML(names ...string) string {
	body := "<decision_tables>\n"
	for _, n := range names {
		body += fmt.Sprintf("  <decision_table>\n"+
			"    <table_name>%s</table_name>\n"+
			"    <attribute_fields><type>FIRST</type></attribute_fields>\n"+
			"  </decision_table>\n", n)
	}
	body += "</decision_tables>\n"
	return body
}

// writeDupProject writes each (filename → table names) mapping into a fresh
// xml/ under dir and returns the project root.
func writeDupProject(t *testing.T, layout map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("mkdir xml: %v", err)
	}
	for name, tables := range layout {
		path := filepath.Join(xmlDir, name)
		if err := os.WriteFile(path, []byte(dupTableXML(tables...)), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

func TestDiagnostics_DuplicateTwoFiles(t *testing.T) {
	root := writeDupProject(t, map[string][]string{
		"001_first_dt.xml":  {"Foo"},
		"002_second_dt.xml": {"Foo"},
	})

	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	if p.Table("Foo") == nil {
		t.Fatalf("expected Foo to exist (first file keeps the real name)")
	}
	if p.Table("Foo-1") == nil {
		t.Fatalf("expected Foo-1 to exist (second-file duplicate renamed)")
	}

	diags := p.Diagnostics()
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Kind != authoring.DuplicateTableKind {
		t.Errorf("Kind = %q, want %q", d.Kind, authoring.DuplicateTableKind)
	}
	if d.OriginalName != "Foo" {
		t.Errorf("OriginalName = %q, want %q", d.OriginalName, "Foo")
	}
	if d.AssignedName != "Foo-1" {
		t.Errorf("AssignedName = %q, want %q", d.AssignedName, "Foo-1")
	}
	if filepath.Base(d.File) != "002_second_dt.xml" {
		t.Errorf("File = %q, want 002_second_dt.xml", d.File)
	}
}

func TestDiagnostics_DuplicateThreeFiles(t *testing.T) {
	root := writeDupProject(t, map[string][]string{
		"a_dt.xml": {"Foo"},
		"b_dt.xml": {"Foo"},
		"c_dt.xml": {"Foo"},
	})

	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	for _, name := range []string{"Foo", "Foo-1", "Foo-2"} {
		if p.Table(name) == nil {
			t.Errorf("expected table %q to exist", name)
		}
	}

	diags := p.Diagnostics()
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diags))
	}
	if diags[0].AssignedName != "Foo-1" || diags[1].AssignedName != "Foo-2" {
		t.Errorf("diagnostics not in assigned-name order: %+v", diags)
	}
}

func TestDiagnostics_SaveAndReopenPersists(t *testing.T) {
	root := writeDupProject(t, map[string][]string{
		"001_a_dt.xml": {"Foo"},
		"002_b_dt.xml": {"Foo"},
	})

	p1, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject (first): %v", err)
	}
	if got := len(p1.Diagnostics()); got != 1 {
		t.Fatalf("first open: want 1 diagnostic, got %d", got)
	}
	if err := p1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	p2, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject (second): %v", err)
	}
	// After save, the XML now holds "Foo" and "Foo-1" as real names; no diagnostics
	// should fire on reopen because there are no true duplicates left on disk.
	if got := len(p2.Diagnostics()); got != 0 {
		t.Errorf("second open: want 0 diagnostics, got %d: %+v", got, p2.Diagnostics())
	}
	if p2.Table("Foo") == nil || p2.Table("Foo-1") == nil {
		t.Errorf("expected both Foo and Foo-1 after reopen; tables=%v", p2.Tables())
	}
}

func TestDiagnostics_NoDuplicatesMeansNoDiagnostics(t *testing.T) {
	root := writeDupProject(t, map[string][]string{
		"a_dt.xml": {"Foo"},
		"b_dt.xml": {"Bar"},
	})

	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	if got := len(p.Diagnostics()); got != 0 {
		t.Errorf("want 0 diagnostics, got %d: %+v", got, p.Diagnostics())
	}
}
