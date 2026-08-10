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

package apiserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// Excel is the system of record, so every authoring surface obeys the same
// contract: refuse to write over a workbook someone has edited, and refresh
// Excel from the XML in the same operation. The HTTP surface did neither, and
// `dtrules edit` embeds this server -- so the browser editor broke the
// contract on every save (#804).

// contractProject lays out a project with a workbook and a sync manifest,
// opened the way the UI opens one: by its rules directory, not its root.
func contractProject(t *testing.T) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	excelDir := filepath.Join(root, "excel")
	for _, d := range []string{xmlDir, excelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	excelPath := filepath.Join(excelDir, "P.xlsx")
	if err := os.WriteFile(excelPath, []byte("workbook"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := sync.NewManifest()
	m.SetPath(filepath.Join(excelDir, ".sync-manifest.json"))
	if err := m.RecordExport(excelPath, []string{filepath.Join(xmlDir, "P_dt.xml")}); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ProjectRoot: root})
	s.projectPath = xmlDir // as the UI opens it
	return s, excelPath
}

// The directories the guard and refresh act on have to be the ones the project
// actually uses. Resolving a declared excel_dir against the rules directory
// gives <root>/xml/excel, which does not exist -- and a non-existent excelDir
// is read as "declared, never exported", silently skipping the refresh. That
// is the shape the bug hid in.
func TestAuthoringDirsFindTheWorkbooksWhenOpenedByRulesDir(t *testing.T) {
	s, excelPath := contractProject(t)

	xmlDir, excelDir := s.authoringDirs()

	if xmlDir != s.projectPath {
		t.Errorf("xmlDir = %q, want %q", xmlDir, s.projectPath)
	}
	// Either the real directory, or "" so the authoring helpers search for it.
	// What must not happen is a path that does not exist.
	if excelDir != "" && excelDir != filepath.Dir(excelPath) {
		if _, err := os.Stat(excelDir); err != nil {
			t.Errorf("authoringDirs returned a directory that does not exist: %q", excelDir)
		}
	}
}

// A workbook touched since the last export means unexported user edits. The
// save must refuse rather than overwrite them.
func TestSaveRefusesWhenAWorkbookWasEdited(t *testing.T) {
	s, excelPath := contractProject(t)

	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(excelPath, future, future); err != nil {
		t.Fatal(err)
	}

	xmlDir, excelDir := s.authoringDirs()
	if err := authoring.GuardExcelIn(xmlDir, excelDir, false); err == nil {
		t.Fatal("a workbook edited since the last export must block the save; " +
			"overwriting it loses the author's work")
	}
}

func TestSaveProceedsWhenNoWorkbookWasEdited(t *testing.T) {
	s, _ := contractProject(t)

	xmlDir, excelDir := s.authoringDirs()
	if err := authoring.GuardExcelIn(xmlDir, excelDir, false); err != nil {
		t.Fatalf("nothing was edited, so the save should proceed: %v", err)
	}
}
