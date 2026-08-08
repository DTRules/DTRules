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

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// customLayoutProject writes a project whose DTRules.xml declares directories
// that are not the `xml/` + `excel/` defaults.
func customLayoutProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"xml", "source"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `<DTRules>
  <xml_dir>xml</xml_dir>
  <excel_dir>source</excel_dir>
</DTRules>`
	if err := os.WriteFile(filepath.Join(dir, "DTRules.xml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestStructureHonoursDeclaredDirs pins #1031.
//
// `sync.ValidateProjectStructure` falls back to the literal "excel" and "xml"
// names when given no override. `review` and the MCP project tools both passed
// empty strings, so a project declaring <excel_dir>source</excel_dir> was told
// `excel/ directory not found` — naming a path it does not use — while
// `verify` on the same project resolved it correctly. One layout, two answers.
//
// The assertion is on the resolved directories rather than on an error string,
// because the failure mode is resolution: getting the paths right is the thing
// that makes every downstream check look in the right place.
func TestStructureHonoursDeclaredDirs(t *testing.T) {
	dir := customLayoutProject(t)

	res, err := validateStructure(dir)
	if err != nil {
		t.Fatalf("validateStructure: %v", err)
	}

	wantExcel := filepath.Join(dir, "source")
	wantXML := filepath.Join(dir, "xml")
	if res.Structure.ExcelDir != wantExcel {
		t.Errorf("ExcelDir = %s, want %s (the declared <excel_dir>)", res.Structure.ExcelDir, wantExcel)
	}
	if res.Structure.XMLDir != wantXML {
		t.Errorf("XMLDir = %s, want %s", res.Structure.XMLDir, wantXML)
	}
	for _, e := range res.Errors {
		if e != nil && filepath.Base(e.Path) == "excel" {
			t.Errorf("reported a missing `excel/` the project never declared: %s", e.Message)
		}
	}
}

// TestStructureAndVerifyAgreeOnLayout is the property the issue is really
// about: every command must answer the same question the same way. Before the
// fix `verify` resolved `source/` and `review` did not.
func TestStructureAndVerifyAgreeOnLayout(t *testing.T) {
	dir := customLayoutProject(t)

	verifyXML, verifyExcel, err := resolveDirs(dir, "", "")
	if err != nil {
		t.Fatalf("resolveDirs: %v", err)
	}
	res, err := validateStructure(dir)
	if err != nil {
		t.Fatalf("validateStructure: %v", err)
	}

	if res.Structure.XMLDir != verifyXML {
		t.Errorf("structure check and verify disagree on the XML dir:\n  structure: %s\n  verify:    %s",
			res.Structure.XMLDir, verifyXML)
	}
	if res.Structure.ExcelDir != verifyExcel {
		t.Errorf("structure check and verify disagree on the Excel dir:\n  structure: %s\n  verify:    %s",
			res.Structure.ExcelDir, verifyExcel)
	}
}

// TestStructureStillDefaultsWithoutManifest keeps the fix narrow: a project
// with no DTRules.xml must still get the conventional layout.
func TestStructureStillDefaultsWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"xml", "excel"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	res, err := validateStructure(dir)
	if err != nil {
		t.Fatalf("validateStructure: %v", err)
	}
	if res.Structure.ExcelDir != filepath.Join(dir, "excel") {
		t.Errorf("ExcelDir = %s, want the default excel/", res.Structure.ExcelDir)
	}
}
