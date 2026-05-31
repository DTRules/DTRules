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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRunFullReview_CleanProject runs the Full Review against a known-good
// sample project (CHIP). Expectations:
//
//   - returns a report with a non-empty project hash
//   - persists .dtrules/last-review.json with that report
//   - passed flag follows len(errors) == 0
//   - warnings / edd_warnings / diagnostics arrays exist (possibly empty)
func TestRunFullReview_CleanProject(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rep, err := runFullReview(dir)
	if err != nil {
		t.Fatalf("runFullReview: %v", err)
	}
	if rep == nil {
		t.Fatal("expected report, got nil")
	}
	if rep.ProjectHash == "" {
		t.Errorf("expected non-empty project_hash")
	}
	if (len(rep.Errors) == 0) != rep.Passed {
		t.Errorf("passed=%v but errors=%d (must agree)", rep.Passed, len(rep.Errors))
	}
	// Persisted to .dtrules/last-review.json.
	cached, readErr := readLastReview(dir)
	if readErr != nil {
		t.Fatalf("readLastReview after run: %v", readErr)
	}
	if cached.ProjectHash != rep.ProjectHash {
		t.Errorf("cached hash %q != returned hash %q", cached.ProjectHash, rep.ProjectHash)
	}
}

// TestRunReview_PositionalPath is the #788 regression. Before the fix
// `dtrules review <path>` collected positional args into `parsedArgs`
// and then threw them away (`_ = parsedArgs`); the structural check
// ran against the CWD instead. This left `--project <path>` as the
// only working form and silently mis-targeted every documented
// positional invocation.
//
// The fix uses parsedArgs[0] as projectPath when no `--project` flag
// is given. This test confirms the positional path is actually used:
// when CWD is some other directory and the positional arg names a
// real project, the report's project hash matches that project's
// xml contents (and is non-empty).
func TestRunReview_PositionalPath(t *testing.T) {
	projectDir := copyProject(t, "../../sampleprojects/CHIP")

	// Run runReview from a CWD that has no project of its own. If the
	// positional arg is ignored, runFullReview would run against the
	// empty CWD and return an empty / SHA256-of-empty project_hash.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	otherDir := t.TempDir()
	if err := os.Chdir(otherDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	cli := NewCLI()
	// Suppress stdout/stderr noise during the test by redirecting them
	// to discard; the report is persisted to .dtrules/last-review.json
	// inside projectDir and we read it back from there.
	if code := cli.runReview([]string{projectDir}); code != 0 {
		// non-zero is acceptable (sample-project warnings → passed=true
		// still has exit 0, but if errors are surfaced it exits 1). The
		// thing we're verifying is *where* it ran, not the result code.
	}

	// The review's persistence target is <projectDir>/.dtrules/last-review.json.
	// If the positional arg was honored, this file exists with a
	// project hash. If it was ignored, no file was written here (the
	// CWD-based run wrote it under otherDir, or hit an error before
	// writing).
	cached, err := readLastReview(projectDir)
	if err != nil {
		t.Fatalf("review didn't persist under the positional project path (%s); err=%v", projectDir, err)
	}
	if cached.ProjectHash == "" {
		t.Errorf("review ran but project_hash is empty — was the positional path actually used?")
	}
	// SHA256-of-empty hash is the signature of a project with zero
	// XML files. CHIP has plenty; a hash matching e3b0c4... would mean
	// runReview ran against the empty otherDir.
	const sha256OfEmpty = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if cached.ProjectHash == sha256OfEmpty {
		t.Errorf("project_hash is the SHA256-of-empty value — review ran against an empty directory, positional arg was ignored")
	}
}

// TestHashProject_Stable verifies repeated hashes over the same files
// match. Stability matters because the deployment gate compares the
// cached hash to a freshly-computed one before allowing a build to
// ship.
func TestHashProject_Stable(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	xmlDir := filepath.Join(dir, "xml")
	h1 := hashProject(xmlDir)
	h2 := hashProject(xmlDir)
	if h1 == "" {
		t.Fatalf("expected non-empty hash")
	}
	if h1 != h2 {
		t.Errorf("hash drifted: %q != %q", h1, h2)
	}
}

// TestHashProject_ChangesWithContent: a hash change on file content is
// the signal the gate uses to detect "the project moved since last
// review." Edit any tracked file and the hash must differ.
func TestHashProject_ChangesWithContent(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	xmlDir := filepath.Join(dir, "xml")
	h1 := hashProject(xmlDir)

	// Append a comment to any _dt.xml under xmlDir.
	entries, err := os.ReadDir(xmlDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var target string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".xml" {
			target = filepath.Join(xmlDir, e.Name())
			break
		}
	}
	if target == "" {
		t.Skip("no XML files in sample project")
	}
	f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, _ = f.WriteString("\n<!-- modified -->\n")
	f.Close()

	h2 := hashProject(xmlDir)
	if h1 == h2 {
		t.Errorf("hash did not change after editing %s", filepath.Base(target))
	}
}

// TestEnforceReviewGate_NoCache verifies the deployment gate refuses
// when no last-review.json exists.
func TestEnforceReviewGate_NoCache(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	code := enforceReviewGate(dir, filepath.Join(dir, "xml"), 24*time.Hour)
	if code == 0 {
		t.Errorf("expected non-zero exit when no review on file, got 0")
	}
}

// TestEnforceReviewGate_HashMismatch verifies the gate refuses when
// the project has changed since the cached review. Run a review,
// modify the XML, then try to enforce.
func TestEnforceReviewGate_HashMismatch(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	if _, err := runFullReview(dir); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	// Mutate any _dt.xml so the hash drifts.
	xmlDir := filepath.Join(dir, "xml")
	entries, _ := os.ReadDir(xmlDir)
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".xml" {
			f, _ := os.OpenFile(filepath.Join(xmlDir, e.Name()), os.O_APPEND|os.O_WRONLY, 0o644)
			f.WriteString("\n<!-- drift -->\n")
			f.Close()
			break
		}
	}

	code := enforceReviewGate(dir, xmlDir, 24*time.Hour)
	if code == 0 {
		t.Errorf("expected non-zero exit on hash drift, got 0")
	}
}

// TestEnforceReviewGate_StaleByAge verifies the gate refuses when the
// cached review is older than maxAge even if the hash matches.
func TestEnforceReviewGate_StaleByAge(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	if _, err := runFullReview(dir); err != nil {
		t.Fatalf("seed review: %v", err)
	}
	// Rewrite the cached report with a timestamp far in the past so
	// the maxAge clamp fires.
	path := filepath.Join(dir, ".dtrules", "last-review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var cached map[string]interface{}
	if err := json.Unmarshal(raw, &cached); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cached["timestamp"] = time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	out, _ := json.Marshal(cached)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	code := enforceReviewGate(dir, filepath.Join(dir, "xml"), 24*time.Hour)
	if code == 0 {
		t.Errorf("expected non-zero exit on stale review, got 0")
	}
}

// TestEnforceReviewGate_Passes verifies the gate accepts a fresh
// passing review. CHIP must compile cleanly (no errors) for this to
// hold — if it ever stops passing, the failure points at the upstream
// regression, not at the gate.
func TestEnforceReviewGate_Passes(t *testing.T) {
	dir := copyProject(t, "../../sampleprojects/CHIP")
	rep, err := runFullReview(dir)
	if err != nil {
		t.Fatalf("seed review: %v", err)
	}
	if !rep.Passed {
		t.Skipf("CHIP review reports errors; gate cannot pass: %v", rep.Errors)
	}
	code := enforceReviewGate(dir, filepath.Join(dir, "xml"), 24*time.Hour)
	if code != 0 {
		t.Errorf("expected exit 0 on fresh passing review, got %d", code)
	}
}

// TestRunFullReview_OrphanCallIsWarning pins the #776 piece-A
// integration: a `perform <X>` referencing a table that isn't defined
// must surface as a warning in the review report. The runtime would
// fail with "table not found" if the offending rule fires, but the
// reference may sit on an unreachable branch, so we surface but don't
// gate.
func TestRunFullReview_OrphanCallIsWarning(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	const eddXML = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="client" access="rw">
    <field name="flag" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
  </entity>
</entity_data_dictionary>
`
	const dtXML = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Caller_Table</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>always</condition_comment>
    <condition_dsl>client.flag</condition_dsl>
    <condition_postfix>client.flag</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>call missing</action_comment>
    <action_dsl>perform Definitely_Missing_Table;</action_dsl>
    <action_postfix>/Definitely_Missing_Table</action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`
	if err := os.WriteFile(filepath.Join(xmlDir, "tiny_dt.xml"), []byte(dtXML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "tiny_edd.xml"), []byte(eddXML), 0644); err != nil {
		t.Fatal(err)
	}

	rep, err := runFullReview(dir)
	if err != nil {
		t.Fatalf("runFullReview: %v", err)
	}
	if rep == nil {
		t.Fatal("expected report, got nil")
	}

	// Find the orphan-call warning.
	var found bool
	for _, w := range rep.Warnings {
		if w.Kind == "orphan perform target" &&
			w.Table == "Caller_Table" &&
			strings.Contains(w.Reason, "Definitely_Missing_Table") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected orphan-perform-target warning pointing at Definitely_Missing_Table, got warnings: %+v", rep.Warnings)
	}
	// Orphan-call findings must NOT land in rep.Errors — that's the
	// behavioral change from PR #842's error policy to this warning
	// policy. Other unrelated errors (e.g. missing excel/ dir on this
	// minimal fixture) are fine; we just need no error tagged with
	// the orphan-call shape.
	for _, e := range rep.Errors {
		if strings.Contains(e.Message, "Definitely_Missing_Table") {
			t.Errorf("orphan-call surfaced as error instead of warning: %+v", e)
		}
	}
}
