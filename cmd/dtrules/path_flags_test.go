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

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDir returns the path to the custom_layout test fixture.
func fixtureDir() string {
	return filepath.Join("testdata", "custom_layout")
}

// TestVerifyCustomLayoutViaFlags verifies that --xml-dir and --excel-dir flags
// allow verify to find non-standard directories.
func TestVerifyCustomLayoutViaFlags(t *testing.T) {
	fixture := fixtureDir()
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	absFixture, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}

	var code int
	stderr := captureStderr(t, func() {
		cli := NewCLI()
		code = cli.runVerify([]string{
			"--xml-dir", "rules",
			"--excel-dir", "workbooks",
			absFixture,
		})
	})

	// Path resolution is the subject, but the fixture is now contract-clean
	// (its XML is what building from its workbook produces), so this asserts
	// the stronger thing: verify passes end to end.
	//
	// It briefly asserted only resolution, on my mistaken reading that the
	// fixture could not round-trip. It can — regenerating its XML from the
	// workbook was all it needed (#1013).
	assertDirsResolved(t, stderr)
	if code != 0 {
		t.Errorf("expected exit 0 on a clean custom-layout project, got %d\n%s", code, stderr)
	}
}

// TestVerifyCustomLayoutViaDTRulesXML verifies that DTRules.xml declaring
// <xml_dir> and <excel_dir> is respected without any CLI flags.
func TestVerifyCustomLayoutViaDTRulesXML(t *testing.T) {
	fixture := fixtureDir()
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	absFixture, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}

	// No --xml-dir / --excel-dir flags; resolution should come from
	// DTRules.xml.
	var code int
	stderr := captureStderr(t, func() {
		cli := NewCLI()
		code = cli.runVerify([]string{absFixture})
	})
	assertDirsResolved(t, stderr)
	if code != 0 {
		t.Errorf("expected exit 0 when DTRules.xml declares the dirs, got %d\n%s", code, stderr)
	}
}

// assertDirsResolved fails if verify could not locate the project's xml or
// excel directory. That error, and not the exit code, is what these tests are
// actually about.
func assertDirsResolved(t *testing.T, stderr string) {
	t.Helper()
	for _, bad := range []string{
		"could not find xml directory",
		"could not find excel directory",
		"no xml/ or excel/ directory found",
	} {
		if strings.Contains(stderr, bad) {
			t.Errorf("directory resolution failed: %s\n%s", bad, stderr)
		}
	}
}

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	os.Stderr = orig
	_ = w.Close()
	out := <-done
	_ = r.Close()
	return out
}

// TestVerifyLegacyLayoutStillWorks ensures that a project with the standard
// xml/ and excel/ directories continues to work with no flags (regression).
func TestVerifyLegacyLayoutStillWorks(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "xml"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "excel"), 0755); err != nil {
		t.Fatal(err)
	}

	cli := NewCLI()
	code := cli.runVerify([]string{dir})
	// An empty project with xml/ and excel/ should pass (nothing to check).
	if code != 0 {
		t.Errorf("expected exit 0 for empty project with standard dirs, got %d", code)
	}
}

// TestVerifyBadPathErrorMessage verifies that a missing --xml-dir produces a
// non-zero exit and mentions the flag name in the error output.
func TestVerifyBadPathErrorMessage(t *testing.T) {
	dir := t.TempDir()

	cli := NewCLI()
	code := cli.runVerify([]string{
		"--xml-dir", "nonexistent",
		dir,
	})
	if code == 0 {
		t.Error("expected non-zero exit for nonexistent --xml-dir")
	}
}

// TestBuildCustomLayoutViaFlags verifies that --xml-dir / --excel-dir are
// accepted by the build command and resolve correctly.
func TestBuildCustomLayoutViaFlags(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}

	cli := NewCLI()
	// --dry-run so we don't actually write files.
	code := cli.runBuild([]string{
		"--xml-dir", "rules",
		"--excel-dir", "workbooks",
		"--dry-run",
		dir,
	})
	// rules/ exists, workbooks/ does not — both absent check fails only if both missing.
	// Since rules/ exists, build should proceed (no-op dry-run).
	if code != 0 {
		t.Errorf("expected exit 0 for build with --xml-dir flag, got %d", code)
	}
}

// TestResolveDirsFromDTRulesXML unit-tests the resolveDirs helper reading a
// DTRules.xml file that declares custom directories.
func TestResolveDirsFromDTRulesXML(t *testing.T) {
	dir := t.TempDir()

	// Write a DTRules.xml with custom dir declarations.
	xmlContent := `<dtrules><xml_dir>myrules</xml_dir><excel_dir>mybooks</excel_dir></dtrules>`
	if err := os.WriteFile(filepath.Join(dir, "DTRules.xml"), []byte(xmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	xmlDir, excelDir, err := resolveDirs(dir, "", "")
	if err != nil {
		t.Fatalf("resolveDirs: %v", err)
	}

	wantXML := filepath.Join(dir, "myrules")
	wantExcel := filepath.Join(dir, "mybooks")

	if xmlDir != wantXML {
		t.Errorf("xmlDir: got %s, want %s", xmlDir, wantXML)
	}
	if excelDir != wantExcel {
		t.Errorf("excelDir: got %s, want %s", excelDir, wantExcel)
	}
}

// TestResolveDirsFlagOverridesDTRulesXML verifies that a CLI flag takes
// precedence over what DTRules.xml declares.
func TestResolveDirsFlagOverridesDTRulesXML(t *testing.T) {
	dir := t.TempDir()

	xmlContent := `<dtrules><xml_dir>fromconfig</xml_dir><excel_dir>fromconfig</excel_dir></dtrules>`
	if err := os.WriteFile(filepath.Join(dir, "DTRules.xml"), []byte(xmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	xmlDir, _, err := resolveDirs(dir, "fromflag", "")
	if err != nil {
		t.Fatalf("resolveDirs: %v", err)
	}

	wantXML := filepath.Join(dir, "fromflag")
	if xmlDir != wantXML {
		t.Errorf("flag should override config: got %s, want %s", xmlDir, wantXML)
	}
}

// TestResolveDirsDefaultsWhenNoDTRulesXML verifies that defaults (xml/ and
// excel/) apply when there is no DTRules.xml and no flags.
func TestResolveDirsDefaultsWhenNoDTRulesXML(t *testing.T) {
	dir := t.TempDir()

	xmlDir, excelDir, err := resolveDirs(dir, "", "")
	if err != nil {
		t.Fatalf("resolveDirs: %v", err)
	}

	if xmlDir != filepath.Join(dir, "xml") {
		t.Errorf("xmlDir: got %s, want %s", xmlDir, filepath.Join(dir, "xml"))
	}
	if excelDir != filepath.Join(dir, "excel") {
		t.Errorf("excelDir: got %s, want %s", excelDir, filepath.Join(dir, "excel"))
	}
}
