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

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestDefaultsWithoutManifest: a project that declares nothing still resolves.
func TestDefaultsWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	c := Load(dir)
	if c.XMLDir != filepath.Join(dir, "xml") {
		t.Errorf("XMLDir = %s, want the default xml/", c.XMLDir)
	}
	if c.ExcelDir != filepath.Join(dir, "excel") {
		t.Errorf("ExcelDir = %s, want the default excel/", c.ExcelDir)
	}
	if c.DefaultTimezone != "UTC" {
		t.Errorf("DefaultTimezone = %q, want UTC", c.DefaultTimezone)
	}
}

// TestManifestOverridesDefaults covers every field the manifest can set.
func TestManifestOverridesDefaults(t *testing.T) {
	dir := writeManifest(t, `<DTRules>
  <xml_dir>pkg/rules</xml_dir>
  <excel_dir>books</excel_dir>
  <entry>Compute</entry>
  <default_timezone>America/New_York</default_timezone>
</DTRules>`)
	c := Load(dir)
	for _, tc := range []struct{ name, got, want string }{
		{"XMLDir", c.XMLDir, filepath.Join(dir, "pkg", "rules")},
		{"ExcelDir", c.ExcelDir, filepath.Join(dir, "books")},
		{"Entry", c.Entry, "Compute"},
		{"DefaultTimezone", c.DefaultTimezone, "America/New_York"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestLegacyRuleSetFilePath: the pre-xml_dir spelling, root-relative with a
// leading slash. Eight sample projects declare it and nothing read it.
func TestLegacyRuleSetFilePath(t *testing.T) {
	dir := writeManifest(t, `<DTRules>
  <RuleSet name="L" source="file">
    <RuleSetFilePath>/legacyrules</RuleSetFilePath>
  </RuleSet>
</DTRules>`)
	c := Load(dir)
	if want := filepath.Join(dir, "legacyrules"); c.XMLDir != want {
		t.Errorf("XMLDir = %s, want %s — the leading slash is root-relative, not absolute", c.XMLDir, want)
	}
	if !c.Declared["RuleSetFilePath"] {
		t.Error("Declared should record that the legacy spelling supplied the value")
	}
}

// TestXMLDirWinsOverLegacy pins the precedence when both are present.
func TestXMLDirWinsOverLegacy(t *testing.T) {
	dir := writeManifest(t, `<DTRules>
  <xml_dir>modern</xml_dir>
  <RuleSet><RuleSetFilePath>/legacy</RuleSetFilePath></RuleSet>
</DTRules>`)
	if c := Load(dir); c.XMLDir != filepath.Join(dir, "modern") {
		t.Errorf("XMLDir = %s, want the modern xml_dir to win", c.XMLDir)
	}
}

// TestFlagsWinOverManifest pins the documented precedence: flags, manifest,
// defaults.
func TestFlagsWinOverManifest(t *testing.T) {
	dir := writeManifest(t, `<DTRules><xml_dir>declared</xml_dir></DTRules>`)
	c := Load(dir).WithDirs("fromflag", "")
	if want := filepath.Join(dir, "fromflag"); c.XMLDir != want {
		t.Errorf("XMLDir = %s, want the flag to win (%s)", c.XMLDir, want)
	}
	if c.ExcelDir != filepath.Join(dir, "excel") {
		t.Errorf("ExcelDir = %s — an empty flag must not clear the resolved value", c.ExcelDir)
	}
}

// TestMalformedManifestFallsBack: a broken manifest must not break every
// command. `dtrules verify` reports it properly; everything else carries on
// with the conventional layout.
func TestMalformedManifestFallsBack(t *testing.T) {
	dir := writeManifest(t, `<DTRules><xml_dir>oops`)
	c := Load(dir)
	if c.XMLDir != filepath.Join(dir, "xml") {
		t.Errorf("XMLDir = %s, want the default after a parse failure", c.XMLDir)
	}
}

// TestLegacyExcelFoldersAreNotExcelDir. DTExcelFolder and EDDExcelFolder name
// the historical .xls authoring directories, not the .xlsx workbooks. Treating
// them as excel_dir would point the toolchain at the wrong files.
func TestLegacyExcelFoldersAreNotExcelDir(t *testing.T) {
	dir := writeManifest(t, `<DTRules>
  <DTExcelFolder>/DecisionTables/</DTExcelFolder>
  <EDDExcelFolder>/edd/</EDDExcelFolder>
</DTRules>`)
	c := Load(dir)
	if c.ExcelDir != filepath.Join(dir, "excel") {
		t.Errorf("ExcelDir = %s, want the default — the legacy .xls folders are not excel_dir", c.ExcelDir)
	}
}

// The three cases below came from pkg/dtrules/loader, which had its own
// DTRules.xml reader that parsed default_timezone and nothing else, and which
// nothing outside its own tests ever called (#1052). The behaviour it
// specified is now Config's.

func TestParseDefaultTimezoneExplicit(t *testing.T) {
	c, err := Parse(strings.NewReader(
		`<DTRules><default_timezone>America/New_York</default_timezone></DTRules>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.DefaultTimezone != "America/New_York" {
		t.Fatalf("DefaultTimezone = %q, want America/New_York", c.DefaultTimezone)
	}
	if !c.Declared["default_timezone"] {
		t.Fatal("an explicit zone should be marked declared")
	}
}

func TestParseDefaultTimezoneAbsentIsUTC(t *testing.T) {
	c, err := Parse(strings.NewReader(`<DTRules><entry>Foo</entry></DTRules>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.DefaultTimezone != "UTC" {
		t.Fatalf("DefaultTimezone = %q, want UTC", c.DefaultTimezone)
	}
	if c.Declared["default_timezone"] {
		t.Fatal("an absent zone must not be marked declared")
	}
}

func TestParseDefaultTimezoneBlankIsUTC(t *testing.T) {
	c, err := Parse(strings.NewReader(
		`<DTRules><default_timezone>   </default_timezone></DTRules>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.DefaultTimezone != "UTC" {
		t.Fatalf("DefaultTimezone = %q, want UTC", c.DefaultTimezone)
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	if _, err := Parse(strings.NewReader(`<DTRules><entry>oops`)); err == nil {
		t.Fatal("Parse should report a malformed manifest; only Load falls back")
	}
}
