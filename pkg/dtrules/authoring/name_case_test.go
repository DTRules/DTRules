package authoring_test

import "testing"

// EL is case-insensitive: `NJ_Tax` and `nj_tax` are one name. The authoring
// lookups compared the authored spelling with ==, so `dtrules table get
// NJ_Tax` reported not_found for a table authored as `nj_tax`, and a second
// `AddTable("nj_tax")` would have been allowed beside `NJ_Tax`. Matching
// normalises; the authored spelling is still what is written back (#1040).
func TestTableLookupIgnoresCase(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddTable("NJ_Tax", "tables_dt.xml", "test"); err != nil {
		t.Fatal(err)
	}
	for _, spelling := range []string{"NJ_Tax", "nj_tax", "Nj_TAX"} {
		tbl := p.Table(spelling)
		if tbl == nil {
			t.Fatalf("Table(%q) = nil; the table is authored as NJ_Tax and that is the same name", spelling)
		}
		if tbl.Name != "NJ_Tax" {
			t.Errorf("Table(%q).Name = %q, want the authored spelling NJ_Tax", spelling, tbl.Name)
		}
		if got := p.FileOf(spelling); got != "tables_dt.xml" {
			t.Errorf("FileOf(%q) = %q, want tables_dt.xml", spelling, got)
		}
	}
	if _, err := p.AddTable("nj_tax", "tables_dt.xml", "test"); err == nil {
		t.Error("AddTable(\"nj_tax\") beside NJ_Tax succeeded; that is the same name twice")
	}
}

func TestEntityLookupIgnoresCase(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.EDD().AddEntity("Result"); err != nil {
		t.Fatal(err)
	}
	for _, spelling := range []string{"Result", "result", "RESULT"} {
		e := p.EDD().Entity(spelling)
		if e == nil {
			t.Fatalf("Entity(%q) = nil; the entity is declared as Result and that is the same name", spelling)
		}
		if e.Name != "Result" {
			t.Errorf("Entity(%q).Name = %q, want the declared spelling Result", spelling, e.Name)
		}
	}
	if _, err := p.EDD().AddEntity("RESULT"); err == nil {
		t.Error("AddEntity(\"RESULT\") beside Result succeeded; that is the same name twice")
	}
}
