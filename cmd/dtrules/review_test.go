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
