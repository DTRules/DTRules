package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1300: a rule file no workbook produces passed verify, because the
// idempotency rebuild runs on a copy that already holds it and leaves it
// untouched. checkWorkbookProvenance rebuilds without it and reports what does
// not come back.
func TestVerifyProvenance(t *testing.T) {
	skipIfNoSampleFixture(t)

	fresh := func(t *testing.T) (proj, xmlDir, excelDir string) {
		t.Helper()
		tmp := t.TempDir()
		if err := copyDir(sampleFixture, tmp); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}
		proj = filepath.Join(tmp, sampleFixtureName)
		return proj, filepath.Join(proj, "xml"), filepath.Join(proj, "excel")
	}
	named := func(fails []verifyFailure, file string) bool {
		for _, f := range fails {
			if f.kind == "provenance" && strings.Contains(f.message, file) {
				return true
			}
		}
		return false
	}

	t.Run("every file of the fixture comes from a workbook", func(t *testing.T) {
		proj, xmlDir, excelDir := fresh(t)
		if fails := checkWorkbookProvenance(proj, xmlDir, excelDir); len(fails) != 0 {
			t.Fatalf("clean fixture reported: %+v", fails)
		}
	})

	t.Run("a hand-written table file", func(t *testing.T) {
		proj, xmlDir, excelDir := fresh(t)
		src, err := os.ReadFile(filepath.Join(xmlDir, "Test_dt.xml"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xmlDir, "Handmade_dt.xml"), src, 0o644); err != nil {
			t.Fatal(err)
		}
		fails := checkWorkbookProvenance(proj, xmlDir, excelDir)
		if !named(fails, "Handmade_dt.xml") {
			t.Fatalf("hand-written Handmade_dt.xml not reported: %+v", fails)
		}
		if named(fails, "Test_dt.xml") {
			t.Errorf("the workbook-backed Test_dt.xml was reported too: %+v", fails)
		}
	})

	t.Run("a mapping whose workbook is gone", func(t *testing.T) {
		proj, xmlDir, excelDir := fresh(t)
		if err := os.Remove(filepath.Join(excelDir, "Test_map.xlsx")); err != nil {
			t.Fatal(err)
		}
		if fails := checkWorkbookProvenance(proj, xmlDir, excelDir); !named(fails, "Test_map.xml") {
			t.Fatalf("Test_map.xml with no workbook not reported: %+v", fails)
		}
	})

	t.Run("templates are not rule files", func(t *testing.T) {
		proj, xmlDir, excelDir := fresh(t)
		src, err := os.ReadFile(filepath.Join(xmlDir, "Test_dt.xml"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xmlDir, "TEMPLATE_dt.xml"), src, 0o644); err != nil {
			t.Fatal(err)
		}
		if fails := checkWorkbookProvenance(proj, xmlDir, excelDir); len(fails) != 0 {
			t.Fatalf("a template, which the loader never reads, was reported: %+v", fails)
		}
	})
}

// The idempotency rebuild now includes mappings; before, no committed
// _map.xml was ever compared with its workbook.
func TestVerifyComparesMappingWithItsWorkbook(t *testing.T) {
	skipIfNoSampleFixture(t)
	tmp := t.TempDir()
	if err := copyDir(sampleFixture, tmp); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}
	proj := filepath.Join(tmp, sampleFixtureName)
	xmlDir, excelDir := filepath.Join(proj, "xml"), filepath.Join(proj, "excel")

	mapPath := filepath.Join(xmlDir, "Test_map.xml")
	src, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(src), "<map>", "<map>\n\t\t\t<!-- edited by hand -->", 1)
	if edited == string(src) {
		t.Fatal("fixture mapping has no <map> element to edit")
	}
	if err := os.WriteFile(mapPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	fails := checkBuildIdempotency(proj, xmlDir, excelDir, &verifyOptions{})
	for _, f := range fails {
		if strings.Contains(f.message, "Test_map.xml") {
			return
		}
	}
	t.Fatalf("a hand edit to Test_map.xml passed the idempotency check: %+v", fails)
}
