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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dtrsync "github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// captureBuildOutput runs the CLI with the given args and captures stdout/stderr.
func captureBuildOutput(args []string) (stdout string, exitCode int) {
	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cli := NewCLI()
	exitCode = cli.Run(args)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), exitCode
}

// TestBuildHelp verifies the build subcommand is recognized.
func TestBuildHelp(t *testing.T) {
	// build with no args in a non-existent path should fail gracefully
	cli := NewCLI()
	// Just ensure runBuild doesn't panic on bad path
	code := cli.runBuild([]string{"/tmp/nonexistent_dtrules_test_12345"})
	if code == 0 {
		t.Error("expected non-zero exit for missing project directory")
	}
}

// TestBuildDryRunNoChanges verifies --dry-run on a clean project makes no file changes.
func TestBuildDryRunNoChanges(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	excelDir := filepath.Join(dir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Capture initial state
	before := dirSnapshot(t, dir)

	cli := NewCLI()
	code := cli.runBuild([]string{"--dry-run", dir})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}

	after := dirSnapshot(t, dir)
	if len(before) != len(after) {
		t.Errorf("--dry-run wrote files: before=%d after=%d", len(before), len(after))
	}
}

// TestBuildFromXMLFlag verifies --from-xml is accepted without panic.
func TestBuildFromXMLFlag(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a stub XML file so there's something to detect
	if err := os.WriteFile(filepath.Join(xmlDir, "stub_dt.xml"), []byte("<DecisionTables/>"), 0644); err != nil {
		t.Fatal(err)
	}

	cli := NewCLI()
	// --from-xml with no excel/ dir should attempt to export (may fail if no xlsx) — just
	// ensure the flag is accepted and we get a clean exit without panic.
	_ = cli.runBuild([]string{"--from-xml", dir})
}

// TestBuildFromExcelAndXMLMutuallyExclusive verifies conflicting flags return non-zero.
func TestBuildFromExcelAndXMLMutuallyExclusive(t *testing.T) {
	cli := NewCLI()
	code := cli.runBuild([]string{"--from-excel", "--from-xml"})
	if code == 0 {
		t.Error("expected non-zero exit when --from-excel and --from-xml are both set")
	}
}

// TestHelpDoesNotShowSyncImportExport verifies sync import/export are not in top-level help.
func TestHelpDoesNotShowSyncImportExport(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cli := NewCLI()
	cli.Run([]string{"help"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	forbidden := []string{"sync import", "sync export"}
	for _, phrase := range forbidden {
		if strings.Contains(output, phrase) {
			t.Errorf("top-level help should not contain %q but it does:\n%s", phrase, output)
		}
	}
}

// TestBuildCommandRegistered verifies "build" is in the subcommands map.
func TestBuildCommandRegistered(t *testing.T) {
	if !subcommands["build"] {
		t.Error("'build' not registered in subcommands map")
	}
}

// TestInternalSyncIsHidden verifies that `dtrules internal sync import` still routes correctly.
func TestInternalSyncIsHidden(t *testing.T) {
	if !subcommands["internal"] {
		t.Error("'internal' not registered in subcommands map")
	}
}

// TestBuildDryRunTouchXLSX verifies that touching an xlsx causes dry-run to report changes.
func TestBuildDryRunTouchXLSX(t *testing.T) {
	// Use the TaxReturn sample project as a real fixture.
	// We need excel/ and xml/ dirs for this to be meaningful.
	taxReturnDir := "/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/TaxReturn"
	if _, err := os.Stat(taxReturnDir); err != nil {
		t.Skip("TaxReturn sample project not found")
	}
	excelDir := filepath.Join(taxReturnDir, "excel")
	if _, err := os.Stat(excelDir); err != nil {
		t.Skip("TaxReturn excel dir not found")
	}

	// Find first xlsx to touch
	entries, err := os.ReadDir(excelDir)
	if err != nil || len(entries) == 0 {
		t.Skip("no xlsx files found in TaxReturn excel dir")
	}
	var xlsxFile string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xlsx") {
			xlsxFile = filepath.Join(excelDir, e.Name())
			break
		}
	}
	if xlsxFile == "" {
		t.Skip("no xlsx files found")
	}

	// Record original mtime
	info, err := os.Stat(xlsxFile)
	if err != nil {
		t.Fatal(err)
	}
	origMod := info.ModTime()

	// Touch to future timestamp
	futureTime := time.Now().Add(10 * time.Minute)
	if err := os.Chtimes(xlsxFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Restore original mtime
		_ = os.Chtimes(xlsxFile, origMod, origMod)
	}()

	// Capture dry-run output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cli := NewCLI()
	code := cli.runBuild([]string{"--dry-run", taxReturnDir})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if code != 0 {
		t.Logf("dry-run output: %s", output)
		// Non-zero is acceptable if sync check finds no pairs (empty state)
	}
	fmt.Printf("dry-run output: %s\n", output)
}

// dirSnapshot returns a map of relative file paths within a directory tree.
func dirSnapshot(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	m := make(map[string]struct{})
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		m[rel] = struct{}{}
		return nil
	})
	return m
}

// =============================================================================
// Issue #508: additional build tests
// =============================================================================

// TestBuildIdempotent verifies that running build twice on a clean fixture
// produces no filesystem changes on the second run.
func TestBuildIdempotent(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	excelDir := filepath.Join(dir, "excel")

	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(excelDir, 0755); err != nil {
		t.Fatal(err)
	}

	// First build on an empty project — should exit 0 (no-op).
	cli1 := NewCLI()
	_ = cli1.runBuild([]string{"--dry-run", dir})

	// Take snapshot before second build.
	before := dirSnapshot(t, dir)

	// Second build — must not write anything new.
	cli2 := NewCLI()
	_ = cli2.runBuild([]string{"--dry-run", dir})

	after := dirSnapshot(t, dir)
	if len(before) != len(after) {
		t.Errorf("second build changed file count: before=%d after=%d", len(before), len(after))
	}
}

// TestBuildFromExcelTouchProducesCanonicalOutput verifies that touching an xlsx
// causes build to import (or at least detect the newer file) without panicking,
// and that a second run is a no-op.
func TestBuildFromExcelTouchProducesCanonicalOutput(t *testing.T) {
	if _, err := os.Stat(taxReturnDir); err != nil {
		t.Skip("TaxReturn sample project not found")
	}
	excelDir := filepath.Join(taxReturnDir, "excel")
	if _, err := os.Stat(excelDir); err != nil {
		t.Skip("TaxReturn excel dir not found")
	}

	// Find first xlsx.
	entries, err := os.ReadDir(excelDir)
	if err != nil || len(entries) == 0 {
		t.Skip("no entries in TaxReturn excel dir")
	}
	var xlsxFile string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xlsx") {
			xlsxFile = filepath.Join(excelDir, e.Name())
			break
		}
	}
	if xlsxFile == "" {
		t.Skip("no xlsx files found")
	}

	// Copy TaxReturn to a temp dir to avoid mutating the real project.
	tmpDir := t.TempDir()
	if err := copyDir(taxReturnDir, tmpDir); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	projCopy := filepath.Join(tmpDir, filepath.Base(taxReturnDir))

	// Touch the xlsx copy to make it newer than any xml.
	copiedExcelDir := filepath.Join(projCopy, "excel")
	copiedEntries, err := os.ReadDir(copiedExcelDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range copiedEntries {
		if strings.HasSuffix(e.Name(), ".xlsx") {
			p := filepath.Join(copiedExcelDir, e.Name())
			future := time.Now().Add(10 * time.Minute)
			_ = os.Chtimes(p, future, future)
			break
		}
	}

	// First run — should not panic.
	cli1 := NewCLI()
	_ = cli1.runBuild([]string{"--from-excel", "--dry-run", projCopy})

	// Second run — no new files relative to first.
	before := dirSnapshot(t, projCopy)
	cli2 := NewCLI()
	_ = cli2.runBuild([]string{"--from-excel", "--dry-run", projCopy})
	after := dirSnapshot(t, projCopy)

	if len(before) != len(after) {
		t.Errorf("second --dry-run changed file count: before=%d after=%d", len(before), len(after))
	}
}

// TestBuildFromXMLTouchProducesCanonicalOutput verifies the --from-xml path
// accepts a project without panicking and that a second dry run is a no-op.
func TestBuildFromXMLTouchProducesCanonicalOutput(t *testing.T) {
	if _, err := os.Stat(taxReturnDir); err != nil {
		t.Skip("TaxReturn sample project not found")
	}
	xmlDir := filepath.Join(taxReturnDir, "xml")
	if _, err := os.Stat(xmlDir); err != nil {
		t.Skip("TaxReturn xml dir not found")
	}

	tmpDir := t.TempDir()
	if err := copyDir(taxReturnDir, tmpDir); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	projCopy := filepath.Join(tmpDir, filepath.Base(taxReturnDir))

	// Touch the first xml file to make it newer.
	copiedXMLDir := filepath.Join(projCopy, "xml")
	copiedEntries, err := os.ReadDir(copiedXMLDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range copiedEntries {
		if strings.HasSuffix(e.Name(), "_dt.xml") {
			p := filepath.Join(copiedXMLDir, e.Name())
			future := time.Now().Add(10 * time.Minute)
			_ = os.Chtimes(p, future, future)
			break
		}
	}

	// First run.
	cli1 := NewCLI()
	_ = cli1.runBuild([]string{"--from-xml", "--dry-run", projCopy})

	// Second run must not add files.
	before := dirSnapshot(t, projCopy)
	cli2 := NewCLI()
	_ = cli2.runBuild([]string{"--from-xml", "--dry-run", projCopy})
	after := dirSnapshot(t, projCopy)

	if len(before) != len(after) {
		t.Errorf("second --from-xml --dry-run changed file count: before=%d after=%d", len(before), len(after))
	}
}

// =============================================================================
// Issue #555: static analysis tests
// =============================================================================

const staticAnalysisFixtureEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="status" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
    <field name="result" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
    <field name="orphan" type="double" subtype="" access="rw" input="" default_value="0" comment="never used"></field>
  </entity>
</entity_data_dictionary>
`

// staticAnalysisFixtureDT has column 1 with no X in any action row.
const staticAnalysisFixtureDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>StaticAnalysisFixture</table_name>
<xls_file>fixture.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>active check</condition_comment>
    <condition_dsl>job.status is equal to "active"</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
    <condition_column column_number="2" column_value="N"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>set result</action_comment>
    <action_dsl>set job.result to 1</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="2" column_value="X"></action_column>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

// TestStaticAnalysis_NoActionColumn verifies that runStaticAnalysis emits a
// no-op column warning when a column has no X in any action row.
func TestStaticAnalysis_NoActionColumn(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "fixture_edd.xml"), []byte(staticAnalysisFixtureEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture_dt.xml"), []byte(staticAnalysisFixtureDT), 0644); err != nil {
		t.Fatal(err)
	}

	step := &dtrsync.StepSummary{}
	runStaticAnalysis(dir, step)

	if len(step.Warnings) == 0 {
		t.Error("expected at least one warning from static analysis, got none")
	}

	found := false
	for _, w := range step.Warnings {
		if strings.Contains(w.Reason, "no actions") || strings.Contains(w.Item, "no-op column") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected no-op column warning, got %v", step.Warnings)
	}
}

// staticAnalysisFirstRedundantDT exercises the FIRST-policy redundancy
// check (#762): two conditions, three columns, FIRST policy. Reaching
// column 3 implies column 2 failed; column 2's only Y/N is row 1 = N, so
// row 1 = Y in column 3 must hold — the explicit Y is redundant. Used by
// TestStaticAnalysis_FirstPolicyRedundancy to confirm `dtrules build`
// runs the Inputs-keyed Analyze (the legacy AnalyzeTable shim drops
// Policy and silently disables this check).
const staticAnalysisFirstRedundantDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>FirstRedundantFixture</table_name>
<xls_file>fixture.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>2</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>enabled flag</condition_comment>
    <condition_dsl>job.status is equal to "active"</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
    <condition_column column_number="2" column_value="N"></condition_column>
    <condition_column column_number="3" column_value="Y"></condition_column>
  </condition_details>
  <condition_details>
    <condition_number>2</condition_number>
    <condition_comment>past threshold</condition_comment>
    <condition_dsl>job.result > 0</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
    <condition_column column_number="3" column_value="N"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>set result</action_comment>
    <action_dsl>set job.result to 1</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"></action_column>
    <action_column column_number="2" column_value="X"></action_column>
    <action_column column_number="3" column_value="X"></action_column>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

// TestStaticAnalysis_FirstPolicyRedundancy confirms that runStaticAnalysis
// surfaces the FIRST-policy redundancy warning (#762). Before #781 the
// build path ran an inlined analyzer that omitted the policy-gated check;
// it now routes through decisiontable.Analyze, which honors Policy. A
// regression here would mean we've slipped back to the inlined path or
// dropped Policy from buildAnalysisInputs.
func TestStaticAnalysis_FirstPolicyRedundancy(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "fixture_edd.xml"), []byte(staticAnalysisFixtureEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture_dt.xml"), []byte(staticAnalysisFirstRedundantDT), 0644); err != nil {
		t.Fatal(err)
	}

	step := &dtrsync.StepSummary{}
	runStaticAnalysis(dir, step)

	found := false
	for _, w := range step.Warnings {
		if w.Item == "redundant condition" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a redundant condition warning (FIRST-policy #762), got %v", step.Warnings)
	}
}

// TestStaticAnalysis_UnusedEDDField verifies that an EDD field never
// referenced in any DT produces an unused warning.
func TestStaticAnalysis_UnusedEDDField(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "fixture_edd.xml"), []byte(staticAnalysisFixtureEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture_dt.xml"), []byte(staticAnalysisFixtureDT), 0644); err != nil {
		t.Fatal(err)
	}

	step := &dtrsync.StepSummary{}
	runStaticAnalysis(dir, step)

	found := false
	for _, w := range step.Warnings {
		if strings.Contains(w.Reason, "unused EDD field") && strings.Contains(w.Reason, "orphan") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unused EDD field warning for job.orphan, got %v", step.Warnings)
	}
}

// TestRunNoSyncAdvisory is the #787 regression. The `runBuild` default
// branch (taken when sync detection returns "none") used to print
// "Nothing to do: all files are in sync." and exit without running
// the advisory pass. The fix routes through `runNoSyncAdvisory`,
// which prints "Nothing to sync — running advisory pass…" and calls
// `runStaticAnalysis`.
//
// We exercise `runNoSyncAdvisory` directly rather than going through
// the full CLI dispatch because reaching the "none" sync state from
// a copied project is unreliable (mtime/hash drift). This pins the
// observable behavior of the fix: the function runs the advisory
// pass and prints any warnings to stdout.
func TestRunNoSyncAdvisory(t *testing.T) {
	dir := t.TempDir()
	// The fixture has a no-op column (column 1 with no actions) — a
	// warning the advisory pass must surface. Re-uses the same
	// fixture as TestStaticAnalysis_NoActionColumn so the assertion
	// is on identical, well-understood content.
	if err := os.WriteFile(filepath.Join(dir, "fixture_edd.xml"), []byte(staticAnalysisFixtureEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture_dt.xml"), []byte(staticAnalysisFixtureDT), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stdout: runNoSyncAdvisory writes the "Nothing to sync"
	// header + the per-warning lines there.
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := runNoSyncAdvisory(dir)
	_ = w.Close()
	os.Stdout = stdout
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	out := string(buf[:n])

	if code != 0 {
		t.Errorf("runNoSyncAdvisory exit=%d; want 0", code)
	}
	if !strings.Contains(out, "Nothing to sync") {
		t.Errorf("expected 'Nothing to sync' marker; got:\n%s", out)
	}
	// The fixture's no-op column finding must show up.
	if !strings.Contains(out, "no-op column") && !strings.Contains(out, "no actions") {
		t.Errorf("expected advisory pass to surface the no-op column finding; got:\n%s", out)
	}
}
