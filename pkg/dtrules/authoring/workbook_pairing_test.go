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

package authoring

import (
	"os"
	"path/filepath"
	"testing"
)

// The pairing has to come from the artifacts, because `.sync-manifest.json` is
// gitignored and never leaves the machine that built it. Both the guard and
// the refresh were documented as no-ops without one — which described every
// clone anyone had ever made. Measured on a manifest-free copy of
// SinusitisTherapy before this landed: an authoring edit updated 0 of 6
// workbooks and left the project failing verify (#1091).

func writePairingFixture(t *testing.T) (xmlDir, excelDir string) {
	t.Helper()
	root := t.TempDir()
	xmlDir = filepath.Join(root, "xml")
	excelDir = filepath.Join(root, "excel")
	for _, d := range []string{xmlDir, excelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(excelDir, "Book.xlsx"), []byte("wb"), 0o644); err != nil {
		t.Fatal(err)
	}
	dt := `<decision_tables><decision_table><table_name>T</table_name>` +
		`<source><relative_path>Book.xlsx</relative_path><file_name>Book.xlsx</file_name>` +
		`<sheet_number>1</sheet_number></source></decision_table></decision_tables>`
	if err := os.WriteFile(filepath.Join(xmlDir, "Book_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatal(err)
	}
	return xmlDir, excelDir
}

func TestPairingComesFromTheArtifacts(t *testing.T) {
	xmlDir, excelDir := writePairingFixture(t)

	pairing := discoverWorkbookPairing(xmlDir, excelDir)

	want := filepath.Join(excelDir, "Book.xlsx")
	paired, ok := pairing[want]
	if !ok {
		t.Fatalf("the workbook the XML names was not discovered; got %v", pairing)
	}
	if len(paired) != 1 || filepath.Base(paired[0]) != "Book_dt.xml" {
		t.Errorf("paired with %v, want the DT file that named it", paired)
	}
}

// Templates are scaffolding to copy, not rules, and must not pull a workbook
// into the refresh — the same predicate the loader and verify use.
func TestTemplatesAreNotPaired(t *testing.T) {
	xmlDir, excelDir := writePairingFixture(t)
	tpl := `<decision_tables><decision_table><table_name>X</table_name>` +
		`<source><file_name>Template.xlsx</file_name><sheet_number>1</sheet_number></source>` +
		`</decision_table></decision_tables>`
	if err := os.WriteFile(filepath.Join(xmlDir, "TEMPLATE_dt.xml"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	for wb := range discoverWorkbookPairing(xmlDir, excelDir) {
		if filepath.Base(wb) == "Template.xlsx" {
			t.Error("a template's workbook was pulled into the pairing")
		}
	}
}

// A project whose XML records no source at all falls back to the manifest,
// so nothing written before source blocks existed breaks.
func TestNoSourceBlocksDiscoversNothing(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `<decision_tables><decision_table><table_name>T</table_name></decision_table></decision_tables>`
	if err := os.WriteFile(filepath.Join(xmlDir, "a_dt.xml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := discoverWorkbookPairing(xmlDir, filepath.Join(root, "excel")); len(got) != 0 {
		t.Errorf("discovered %v from XML that records no source", got)
	}
}

// A project that splits its rules one file per state records the workbook as a
// bare base name -- <file_name>CO.xlsx</file_name> -- while the workbook itself
// lives at excel/states/CO.xlsx. Resolving that against excelDir alone misses
// it, and the old base-name fallback could not help: filepath.Base of a name
// that is already a base name is itself.
//
// The pairing therefore came back empty, RefreshExcelIn skipped the export
// without erroring, and an authoring write landed in XML with the workbook
// untouched -- leaving the project failing verify on drift the author never
// made (#1169).
func TestNestedWorkbookIsPairedByBaseName(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml", "states")
	excelDir := filepath.Join(root, "excel")
	nested := filepath.Join(excelDir, "states")
	for _, d := range []string{xmlDir, nested} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(nested, "CO.xlsx"), []byte("wb"), 0o644); err != nil {
		t.Fatal(err)
	}
	dt := `<decision_tables><decision_table><table_name>CO_Tax</table_name>` +
		`<source><relative_path>CO.xlsx</relative_path><file_name>CO.xlsx</file_name>` +
		`<sheet_number>1</sheet_number></source></decision_table></decision_tables>`
	if err := os.WriteFile(filepath.Join(xmlDir, "CO_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatal(err)
	}

	pairing := discoverWorkbookPairing(filepath.Join(root, "xml"), excelDir)

	want := filepath.Join(nested, "CO.xlsx")
	if _, ok := pairing[want]; !ok {
		t.Fatalf("the nested workbook was not paired, so its export would be skipped silently; got %v", pairing)
	}
	if stray := filepath.Join(excelDir, "CO.xlsx"); pairing[stray] != nil {
		t.Errorf("paired against a path that does not exist: %s", stray)
	}
}

// Two workbooks sharing a base name cannot be told apart from a bare
// <file_name>, and exporting a project's rules into the wrong workbook is
// worse than not exporting them. Ambiguity is left unresolved on purpose.
func TestAmbiguousBaseNameIsNotGuessed(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	excelDir := filepath.Join(root, "excel")
	for _, d := range []string{xmlDir, filepath.Join(excelDir, "a"), filepath.Join(excelDir, "b")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, sub := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(excelDir, sub, "CO.xlsx"), []byte("wb"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dt := `<decision_tables><decision_table><table_name>CO_Tax</table_name>` +
		`<source><relative_path>CO.xlsx</relative_path><file_name>CO.xlsx</file_name>` +
		`<sheet_number>1</sheet_number></source></decision_table></decision_tables>`
	if err := os.WriteFile(filepath.Join(xmlDir, "CO_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatal(err)
	}

	pairing := discoverWorkbookPairing(xmlDir, excelDir)

	for _, sub := range []string{"a", "b"} {
		if p := filepath.Join(excelDir, sub, "CO.xlsx"); pairing[p] != nil {
			t.Errorf("guessed %s from an ambiguous base name", p)
		}
	}
}

// The recompile names its output from the workbook's base name, so it has to
// be pointed at the mirrored directory or excel/states/CO.xlsx regenerates
// xml/CO_dt.xml beside the xml/states/CO_dt.xml it was meant to replace --
// every table then declared in two files, which verify rejects (#1169).
func TestRecompileMirrorsTheWorkbookLayout(t *testing.T) {
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	excelDir := filepath.Join(root, "excel")

	got := mirroredXMLDir(xmlDir, excelDir, filepath.Join(excelDir, "states", "CO.xlsx"))
	if want := filepath.Join(xmlDir, "states"); got != want {
		t.Errorf("nested workbook recompiles into %s, want %s", got, want)
	}

	got = mirroredXMLDir(xmlDir, excelDir, filepath.Join(excelDir, "TaxReturn.xlsx"))
	if got != xmlDir {
		t.Errorf("flat workbook recompiles into %s, want %s", got, xmlDir)
	}

	outside := mirroredXMLDir(xmlDir, excelDir, filepath.Join(root, "elsewhere", "X.xlsx"))
	if outside != xmlDir {
		t.Errorf("a workbook outside excelDir should fall back to xmlDir, got %s", outside)
	}
}
