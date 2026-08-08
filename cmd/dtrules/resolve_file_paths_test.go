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
	"strings"
	"testing"
)

// The legacy -rules/-edd/-dt interface used to hard-error the moment no flag
// was given, even standing inside a project that declares where its rules
// live. It now falls back to the same resolved config every other command
// uses (#1052).

// legacyProject lays out a project whose rules directory is declared rather
// than conventional, chdirs into it, and clears the legacy flags.
func legacyProject(t *testing.T, dtNames ...string) string {
	t.Helper()
	root := t.TempDir()
	rules := filepath.Join(root, "rules")
	if err := os.MkdirAll(rules, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "<DTRules><xml_dir>rules</xml_dir></DTRules>"
	if err := os.WriteFile(filepath.Join(root, "DTRules.xml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(rules, name), []byte("<x/>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("thing_edd.xml")
	for _, n := range dtNames {
		write(n)
	}

	prevRules, prevEDD, prevDT := *rulesDir, *eddFile, *dtFile
	*rulesDir, *eddFile, *dtFile = "", "", ""
	t.Cleanup(func() { *rulesDir, *eddFile, *dtFile = prevRules, prevEDD, prevDT })

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	return rules
}

func TestResolveFilePathsFallsBackToDeclaredXMLDir(t *testing.T) {
	rules := legacyProject(t, "thing_dt.xml")

	eddPath, dtPath, err := resolveFilePaths()
	if err != nil {
		t.Fatalf("resolveFilePaths: %v", err)
	}
	if want := filepath.Join(rules, "thing_edd.xml"); eddPath != want {
		t.Errorf("edd = %q, want %q", eddPath, want)
	}
	if want := filepath.Join(rules, "thing_dt.xml"); dtPath != want {
		t.Errorf("dt = %q, want %q", dtPath, want)
	}
}

// One EDD and one DT is all this interface can load. A project split across
// several files must be refused: loading one of four and reporting the count
// as the whole rule set is a silent wrong answer.
func TestResolveFilePathsRefusesMultiFileProject(t *testing.T) {
	legacyProject(t, "a_dt.xml", "b_dt.xml", "c_dt.xml")

	_, _, err := resolveFilePaths()
	if err == nil {
		t.Fatal("a multi-file project must be refused, not half-loaded")
	}
	if !strings.Contains(err.Error(), "loads one of each") {
		t.Errorf("error should explain the one-file limit, got: %v", err)
	}
	if !strings.Contains(err.Error(), "dtrules table list") {
		t.Errorf("error should point at the subcommands, got: %v", err)
	}
}

// Explicit flags still outrank the project, and still report against the
// directory the caller named.
func TestResolveFilePathsExplicitRulesDirWins(t *testing.T) {
	legacyProject(t, "thing_dt.xml")

	elsewhere := t.TempDir()
	*rulesDir = elsewhere

	if _, _, err := resolveFilePaths(); err == nil {
		t.Fatal("an explicit empty -rules dir should error, not fall back to the project")
	} else if !strings.Contains(err.Error(), elsewhere) {
		t.Errorf("error should name the directory the caller gave, got: %v", err)
	}
}

func TestResolveFilePathsExplicitFilesWin(t *testing.T) {
	legacyProject(t, "thing_dt.xml")
	*eddFile, *dtFile = "/somewhere/e.xml", "/somewhere/d.xml"

	eddPath, dtPath, err := resolveFilePaths()
	if err != nil {
		t.Fatalf("resolveFilePaths: %v", err)
	}
	if eddPath != "/somewhere/e.xml" || dtPath != "/somewhere/d.xml" {
		t.Errorf("explicit -edd/-dt must be used verbatim, got %q and %q", eddPath, dtPath)
	}
}

// Outside a project the fallback finds nothing, and the message has to name
// all three ways out rather than only the flags.
func TestResolveFilePathsOutsideProject(t *testing.T) {
	legacyProject(t) // no *_dt.xml written

	_, _, err := resolveFilePaths()
	if err == nil {
		t.Fatal("expected an error with no decision tables present")
	}
	for _, want := range []string{"-rules", "-edd", "xml_dir"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}
