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

package authoring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// newProj returns an in-memory project plus its root dir (xml lives at root/xml,
// authoring-notes.md at root/authoring-notes.md).
func newProj(t *testing.T) (*authoring.Project, string) {
	t.Helper()
	root := t.TempDir()
	p, err := authoring.NewInMemoryProject(root)
	if err != nil {
		t.Fatalf("NewInMemoryProject: %v", err)
	}
	return p, root
}

func TestAddTable_RequiresFile(t *testing.T) {
	p, _ := newProj(t)
	if _, err := p.AddTable("X", "", "r"); err == nil {
		t.Error("AddTable without a file should error")
	}
	if _, err := p.AddTable("X", "nope_dt.xml", "r"); err == nil {
		t.Error("AddTable into a non-existent file should error")
	}
	// tables_dt.xml is seeded by NewInMemoryProject and exists.
	if _, err := p.AddTable("X", "tables_dt.xml", ""); err != nil {
		t.Errorf("AddTable into existing file: %v", err)
	}
}

func TestCreateFile_RangeOverlapAndDuplicate(t *testing.T) {
	p, _ := newProj(t)
	if err := p.CreateFile("a_dt.xml", 1000, 2000, "a"); err != nil {
		t.Fatalf("CreateFile a: %v", err)
	}
	if err := p.CreateFile("b_dt.xml", 1500, 2500, "b"); err == nil {
		t.Error("overlapping range should be rejected")
	}
	if err := p.CreateFile("b_dt.xml", 3000, 3500, "b"); err != nil {
		t.Errorf("non-overlapping range: %v", err)
	}
	if err := p.CreateFile("a_dt.xml", 5000, 6000, "dup"); err == nil {
		t.Error("duplicate file should be rejected")
	}
	if err := p.CreateFile("c_dt.xml", 10, 5, "bad"); err == nil {
		t.Error("invalid range should be rejected")
	}
	if err := p.CreateFile("d_dt.xml", 100, 200, ""); err == nil {
		t.Error("missing reason should be rejected")
	}
}

func TestInRangeNumberingAndExhaustion(t *testing.T) {
	p, _ := newProj(t)
	if err := p.CreateFile("r_dt.xml", 1000, 1015, "r"); err != nil {
		t.Fatal(err)
	}
	t1, err := p.AddTable("t1", "r_dt.xml", "")
	if err != nil || t1.Number != 1000 {
		t.Fatalf("t1 number=%d err=%v, want 1000", numOr(t1), err)
	}
	t2, err := p.AddTable("t2", "r_dt.xml", "")
	if err != nil || t2.Number != 1010 {
		t.Fatalf("t2 number=%d err=%v, want 1010", numOr(t2), err)
	}
	if _, err := p.AddTable("t3", "r_dt.xml", ""); err == nil {
		t.Error("t3 should fail: range exhausted (1020 > 1015)")
	}
}

func TestMoveTable_RenumberAndAutoDelete(t *testing.T) {
	p, _ := newProj(t)
	if err := p.CreateFile("src_dt.xml", 1000, 2000, "src"); err != nil {
		t.Fatal(err)
	}
	if err := p.CreateFile("dst_dt.xml", 3000, 4000, "dst"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddTable("only", "src_dt.xml", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.MoveTable("only", "dst_dt.xml", "consolidate"); err != nil {
		t.Fatalf("MoveTable: %v", err)
	}
	if got := p.FileOf("only"); got != "dst_dt.xml" {
		t.Errorf("FileOf after move = %q, want dst_dt.xml", got)
	}
	if mv := p.Table("only"); mv == nil || mv.Number != 3000 {
		t.Errorf("moved table number = %d, want 3000 (dst range start)", numOr(mv))
	}
	for _, f := range p.Files() {
		if f.Path == "src_dt.xml" {
			t.Error("emptied src_dt.xml should be dropped from Files()")
		}
	}
	if err := p.MoveTable("only", "src_dt.xml", ""); err == nil {
		t.Error("move without reason should error")
	}
}

func TestDeleteTable_RemovesOrphanFileOnDisk(t *testing.T) {
	p, root := newProj(t)
	must(t, p.CreateFile("gone_dt.xml", 5000, 5500, "temporary"))
	if _, err := p.AddTable("solo", "gone_dt.xml", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	onDisk := filepath.Join(root, "xml", "gone_dt.xml")
	if _, err := os.Stat(onDisk); err != nil {
		t.Fatalf("gone_dt.xml should exist after first save: %v", err)
	}
	if err := p.DeleteTable("solo", ""); err == nil {
		t.Error("delete without reason should error")
	}
	if err := p.DeleteTable("solo", "no longer needed"); err != nil {
		t.Fatalf("DeleteTable: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(onDisk); !os.IsNotExist(err) {
		t.Errorf("emptied gone_dt.xml should be deleted from disk, stat err=%v", err)
	}
}

func TestSetFileRange_Validation(t *testing.T) {
	p, _ := newProj(t)
	must(t, p.CreateFile("a_dt.xml", 1000, 2000, "a"))
	must(t, p.CreateFile("b_dt.xml", 3000, 4000, "b"))
	if _, err := p.AddTable("t", "a_dt.xml", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.SetFileRange("a_dt.xml", 3500, 4500, "x"); err == nil {
		t.Error("set-range overlapping b should be rejected")
	}
	if err := p.SetFileRange("a_dt.xml", 1100, 2000, "x"); err == nil {
		t.Error("set-range excluding an in-use number (1000) should be rejected")
	}
	if err := p.SetFileRange("a_dt.xml", 500, 2000, "widen"); err != nil {
		t.Errorf("valid widen: %v", err)
	}
	if err := p.SetFileRange("a_dt.xml", 500, 2000, ""); err == nil {
		t.Error("set-range without reason should be rejected")
	}
}

func TestFilesAndNote(t *testing.T) {
	p, root := newProj(t)
	must(t, p.CreateFile("budgets_dt.xml", 3000, 3500, "budget computation"))
	if _, err := p.AddTable("Budget", "budgets_dt.xml", ""); err != nil {
		t.Fatal(err)
	}
	p.AppendNote("design discussion goes here")

	var found *authoring.FileInfo
	files := p.Files()
	for i := range files {
		if files[i].Path == "budgets_dt.xml" {
			found = &files[i]
		}
	}
	if found == nil || found.Lo != 3000 || found.Hi != 3500 || found.Purpose != "budget computation" {
		t.Fatalf("Files() entry wrong: %+v", found)
	}
	if len(found.Tables) != 1 || found.Tables[0] != "Budget" {
		t.Errorf("Files() tables = %v, want [Budget]", found.Tables)
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	notes := readFile(t, filepath.Join(root, "authoring-notes.md"))
	if !strings.Contains(notes, "budgets_dt.xml") || !strings.Contains(notes, "[3000-3500]") {
		t.Errorf("notes missing Files entry:\n%s", notes)
	}
	if !strings.Contains(notes, "design discussion goes here") {
		t.Errorf("notes missing the appended note:\n%s", notes)
	}
}

func TestAuthoringNotes_RoundTripPreservesConventions(t *testing.T) {
	tmp := t.TempDir()
	xmlDir := filepath.Join(tmp, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "main_dt.xml"), []byte(minimalDT), 0644); err != nil {
		t.Fatal(err)
	}
	const conventions = "Compute constants in the constants file; never inline literals"
	notes := "# Authoring Notes\n\n## Files\n- `main_dt.xml` [1000-2000] — main tables\n\n## Conventions\n" +
		conventions + "\n\n## Change log\n- 2026-01-01 — seeded\n"
	if err := os.WriteFile(filepath.Join(tmp, "authoring-notes.md"), []byte(notes), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(tmp)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	// Range loaded from notes (opt-in) so a new file must not overlap.
	if err := p.CreateFile("extra_dt.xml", 3000, 3500, "extra logic"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := readFile(t, filepath.Join(tmp, "authoring-notes.md"))
	if !strings.Contains(got, conventions) {
		t.Errorf("Conventions section not preserved:\n%s", got)
	}
	if !strings.Contains(got, "- 2026-01-01 — seeded") {
		t.Errorf("existing change-log entry not preserved:\n%s", got)
	}
	if !strings.Contains(got, "main_dt.xml` [1000-2000]") {
		t.Errorf("Files section lost main entry:\n%s", got)
	}
	if !strings.Contains(got, "extra_dt.xml` [3000-3500]") {
		t.Errorf("Files section missing new file:\n%s", got)
	}
	if !strings.Contains(got, "add file `extra_dt.xml`") {
		t.Errorf("change log missing new-file entry:\n%s", got)
	}
}

func TestEDD_AlphabeticalSortOnSave(t *testing.T) {
	p, root := newProj(t)
	for _, n := range []string{"zebra", "apple", "Mango"} {
		if _, err := p.EDD().AddEntity(n); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	s := readFile(t, filepath.Join(root, "xml", "project_edd.xml"))
	ia := strings.Index(s, `name="apple"`)
	im := strings.Index(s, `name="Mango"`)
	iz := strings.Index(s, `name="zebra"`)
	if ia < 0 || im < 0 || iz < 0 {
		// Attribute order may render with single quotes; try those too.
		ia = strings.Index(s, "apple")
		im = strings.Index(s, "Mango")
		iz = strings.Index(s, "zebra")
	}
	if ia < 0 || im < 0 || iz < 0 {
		t.Fatalf("entities missing in EDD:\n%s", s)
	}
	if !(ia < im && im < iz) {
		t.Errorf("entities not sorted ascending by lowercase name: apple@%d Mango@%d zebra@%d", ia, im, iz)
	}
}

// helpers

func numOr(t *authoring.Table) int {
	if t == nil {
		return -1
	}
	return t.Number
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

const minimalDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Main</table_name>
<xls_file>main.xlsx</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<TABLE_NUMBER>1000</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
</decision_table>
</decision_tables>
`
