package main

// `validate` must resolve xml/excel directories the same way every other
// command does: flags first, then the project manifest, then the defaults.
// It used to pass the raw flags straight through, so an unset flag became
// an empty string and the structure check fell back to a literal "excel"
// — telling a project that declares <excel_dir>source</excel_dir> that
// `excel/ directory not found`, while verify and review resolved it (#1031).

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, dir, xmlDir, excelDir string) {
	t.Helper()
	m := `<DTRules>
  <xml_dir>` + xmlDir + `</xml_dir>
  <excel_dir>` + excelDir + `</excel_dir>
</DTRules>`
	if err := os.WriteFile(filepath.Join(dir, "DTRules.xml"), []byte(m), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateHonoursDeclaredExcelDir(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"xml", "source"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest(t, dir, "xml", "source")

	xmlDir, excelDir, err := resolveDirs(dir, "", "")
	if err != nil {
		t.Fatalf("resolveDirs: %v", err)
	}
	if filepath.Base(excelDir) != "source" {
		t.Errorf("excelDir = %q, want the declared source/", excelDir)
	}
	if filepath.Base(xmlDir) != "xml" {
		t.Errorf("xmlDir = %q, want xml/", xmlDir)
	}

	// The structure check must then find it, which is what validate does.
	res, err := validateStructure(dir)
	if err != nil {
		t.Fatalf("validateStructure: %v", err)
	}
	for _, e := range res.Errors {
		if e != nil {
			t.Errorf("declared layout reported an error: %s", e.Error())
		}
	}
}

// An explicit flag still outranks the manifest.
func TestValidateFlagOutranksManifest(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"xml", "source", "other"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest(t, dir, "xml", "source")

	_, excelDir, err := resolveDirs(dir, "", "other")
	if err != nil {
		t.Fatalf("resolveDirs: %v", err)
	}
	if filepath.Base(excelDir) != "other" {
		t.Errorf("excelDir = %q, want the flag value other/", excelDir)
	}
}
