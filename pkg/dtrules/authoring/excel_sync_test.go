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

package authoring_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// Minimal DT XML used across the Excel-sync tests below. One table,
// one condition, one action, postfix populated — the loader's strict
// path won't reject it during the auto-export refresh.
const excelSyncDTFixture = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>ExcelSyncFixture</table_name>
<xls_file>fixture.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>flag</condition_comment>
    <condition_dsl>job.flag</condition_dsl>
    <condition_postfix>job.flag</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>noop</action_comment>
    <action_dsl></action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

const excelSyncEDDFixture = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="flag" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
  </entity>
</entity_data_dictionary>
`

// setupExcelSyncProject lays out the staking-style flat project that
// the Excel-sync helpers are meant to support: a `.sync-manifest.json`
// next to a fake xlsx, in a sibling directory to the xml. Returns the
// xml dir, the manifest dir, and the fake xlsx path. The xlsx is just
// an empty placeholder file — the helpers care about its mtime, not
// its bytes.
func setupExcelSyncProject(t *testing.T) (xmlDir, manifestDir, xlsxPath string) {
	t.Helper()
	root := t.TempDir()
	xmlDir = filepath.Join(root, "rules")
	manifestDir = filepath.Join(root, "source")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a placeholder Excel file. The exporter will overwrite it
	// during refresh; for guard tests we only need it to exist.
	xlsxPath = filepath.Join(manifestDir, "fixture.xlsx")
	if err := os.WriteFile(xlsxPath, []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a manifest that records the xlsx as last-exported a long
	// time ago (well before any subsequent file touches in the test).
	pastTime := time.Now().Add(-24 * time.Hour)
	manifest := map[string]interface{}{
		"version":     1,
		"lastUpdated": pastTime.Format(time.RFC3339Nano),
		"files": map[string]interface{}{
			"fixture.xlsx": map[string]interface{}{
				"lastExportTime":       pastTime.Format(time.RFC3339Nano),
				"excelModTimeAtExport": pastTime.Format(time.RFC3339Nano),
				"xmlFiles":             []string{"../rules/fixture_dt.xml", "../rules/fixture_edd.xml"},
			},
		},
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(manifestDir, ".sync-manifest.json"), manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write the DT + EDD XML.
	if err := os.WriteFile(filepath.Join(xmlDir, "fixture_dt.xml"), []byte(excelSyncDTFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "fixture_edd.xml"), []byte(excelSyncEDDFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make the xlsx older than the manifest's last-export time so the
	// guard sees "Excel not modified since export" by default.
	veryPast := pastTime.Add(-time.Hour)
	if err := os.Chtimes(xlsxPath, veryPast, veryPast); err != nil {
		t.Fatal(err)
	}
	return xmlDir, manifestDir, xlsxPath
}

// TestGuardExcelInDir_RefusesWhenExcelNewer is the core Phase 1
// contract: AI/CLI writes that would clobber human Excel edits get
// refused. The guard returns sync.ExcelModifiedError naming the file
// so the operator can investigate.
func TestGuardExcelInDir_RefusesWhenExcelNewer(t *testing.T) {
	xmlDir, _, xlsxPath := setupExcelSyncProject(t)

	// Touch the xlsx to make it "newer than last export" — the
	// exact condition that flags it as "user edited Excel since the
	// last build."
	now := time.Now()
	if err := os.Chtimes(xlsxPath, now, now); err != nil {
		t.Fatal(err)
	}

	err := authoring.GuardExcelInDir(xmlDir, false)
	if err == nil {
		t.Fatal("expected guard to refuse (Excel newer than last export); got nil")
	}
	if !sync.IsExcelModifiedError(err) {
		t.Errorf("expected sync.ExcelModifiedError, got %T: %v", err, err)
	}
}

// TestGuardExcelInDir_PassesWithOverwriteFlag verifies the
// --force-overwrite-excel CLI flag's underlying behavior: when the
// caller asserts the XML version should win, the mtime guard yields.
// The lock-file check still runs (an open Excel still blocks the
// write — the override is for human edits, not for live conflicts).
func TestGuardExcelInDir_PassesWithOverwriteFlag(t *testing.T) {
	xmlDir, _, xlsxPath := setupExcelSyncProject(t)
	now := time.Now()
	if err := os.Chtimes(xlsxPath, now, now); err != nil {
		t.Fatal(err)
	}

	if err := authoring.GuardExcelInDir(xmlDir, true); err != nil {
		t.Errorf("guard should yield with overwrite=true; got %v", err)
	}
}

// TestGuardExcelInDir_NoManifestIsNoOp pins the backward-compatibility
// promise: projects that don't have a `.sync-manifest.json` (legacy
// XML-only or fresh fixtures with no Excel side) skip the guard
// entirely. Without this every test in the codebase would break.
func TestGuardExcelInDir_NoManifestIsNoOp(t *testing.T) {
	dir := t.TempDir()
	if err := authoring.GuardExcelInDir(dir, false); err != nil {
		t.Errorf("guard on no-manifest dir should be no-op; got %v", err)
	}
}

// TestGuardExcelInDir_RefusesWhenExcelLocked is the lock-file path.
// Microsoft Excel's `~$<name>.xlsx` sidecar and LibreOffice's
// `.~lock.<name>.xlsx#` sidecar both indicate an open spreadsheet;
// writing through either corrupts the in-app state. The guard
// refuses regardless of the OverwriteExcel flag — overrides apply
// to human-edits-vs-AI, not to live-conflicts.
func TestGuardExcelInDir_RefusesWhenExcelLocked(t *testing.T) {
	xmlDir, manifestDir, xlsxPath := setupExcelSyncProject(t)

	// LibreOffice convention.
	lockPath := filepath.Join(manifestDir, ".~lock.fixture.xlsx#")
	if err := os.WriteFile(lockPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// Even with overwrite=true the lock blocks.
	err := authoring.GuardExcelInDir(xmlDir, true)
	if err == nil {
		t.Fatal("expected guard to refuse on lock file; got nil")
	}
	if !strings.Contains(err.Error(), "appears to be open") {
		t.Errorf("expected lock-file message; got: %v", err)
	}
	os.Remove(lockPath)

	// Microsoft Excel convention.
	excelLockPath := filepath.Join(manifestDir, "~$fixture.xlsx")
	if err := os.WriteFile(excelLockPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	err = authoring.GuardExcelInDir(xmlDir, true)
	if err == nil {
		t.Fatal("expected guard to refuse on MS Excel lock file; got nil")
	}
	if !strings.Contains(err.Error(), "appears to be open") {
		t.Errorf("expected lock-file message; got: %v", err)
	}

	// Keep xlsxPath referenced so the linter doesn't flag the helper
	// return value as unused on this branch.
	_ = xlsxPath
}

// TestRefreshExcelInDir_NoOpOnNoManifest is the symmetric backward-
// compatibility check for Phase 2. Fresh fixtures without a manifest
// must not trigger any RuleSet load or Excel write — Save / compile
// against them stay XML-only as in v1.14.4.
func TestRefreshExcelInDir_NoOpOnNoManifest(t *testing.T) {
	dir := t.TempDir()
	if err := authoring.RefreshExcelInDir(dir); err != nil {
		t.Errorf("refresh on no-manifest dir should be no-op; got %v", err)
	}
}

// TestRefreshExcelInDir_UpdatesExcelMtime is the Phase 2 happy path:
// after the refresh runs, the xlsx file's mtime has moved forward
// (proof that the exporter wrote bytes) AND the manifest's
// LastExportTime has advanced (proof that RecordExport ran).
//
// We don't assert on the xlsx contents — the existing
// pkg/dtrules/excel tests cover that — only that the refresh
// happens at all.
func TestRefreshExcelInDir_UpdatesExcelMtime(t *testing.T) {
	xmlDir, manifestDir, xlsxPath := setupExcelSyncProject(t)

	beforeMtime := mtimeOf(t, xlsxPath)
	manifestPathBefore := filepath.Join(manifestDir, ".sync-manifest.json")
	manifestBefore, err := os.ReadFile(manifestPathBefore)
	if err != nil {
		t.Fatal(err)
	}

	// Sleep briefly so an mtime change is observable; some filesystems
	// have second-resolution mtime.
	time.Sleep(1100 * time.Millisecond)

	if err := authoring.RefreshExcelInDir(xmlDir); err != nil {
		t.Fatalf("RefreshExcelInDir: %v", err)
	}

	afterMtime := mtimeOf(t, xlsxPath)
	if !afterMtime.After(beforeMtime) {
		t.Errorf("xlsx mtime did not advance after refresh: before=%v after=%v",
			beforeMtime, afterMtime)
	}

	manifestAfter, err := os.ReadFile(manifestPathBefore)
	if err != nil {
		t.Fatal(err)
	}
	if string(manifestBefore) == string(manifestAfter) {
		t.Error("manifest unchanged after refresh — RecordExport should have advanced LastExportTime")
	}
}

func mtimeOf(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}
