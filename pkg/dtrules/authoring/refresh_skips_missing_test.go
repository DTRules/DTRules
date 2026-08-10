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

	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// A sync manifest outlives the files it names. TaxReturn's committed manifest
// still listed 114 per-table workbooks that the samples consolidation had
// replaced with a single TaxReturn.xlsx, and RefreshExcelIn re-created every
// one of them on each authoring write -- turning a one-action edit into a
// 244-file diff with 172 binary workbooks (#1062).

// staleManifestProject writes a project whose manifest names two workbooks:
// one that exists and one that does not.
func staleManifestProject(t *testing.T) (xmlDir, excelDir, live, ghost string) {
	t.Helper()
	root := t.TempDir()
	xmlDir = filepath.Join(root, "xml")
	excelDir = filepath.Join(root, "excel")
	for _, d := range []string{xmlDir, excelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, s string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(xmlDir, "P_edd.xml"), declaredEDD)
	write(filepath.Join(xmlDir, "P_dt.xml"), declaredDT)

	live = filepath.Join(excelDir, "P.xlsx")
	ghost = filepath.Join(excelDir, "001_Only_dt.xlsx")
	write(live, "") // presence is what matters; the exporter rewrites it

	m := sync.NewManifest()
	m.SetPath(filepath.Join(excelDir, ".sync-manifest.json"))
	for _, p := range []string{live, ghost} {
		if err := m.RecordExport(p, []string{filepath.Join(xmlDir, "P_dt.xml")}); err != nil {
			t.Fatalf("RecordExport %s: %v", p, err)
		}
	}
	// RecordExport stats the file, so the ghost entry exists with a zero
	// mod time -- exactly the shape a deleted workbook leaves behind.
	if err := os.Remove(ghost); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return xmlDir, excelDir, live, ghost
}

func TestRefreshExcelDoesNotResurrectDeletedWorkbooks(t *testing.T) {
	xmlDir, excelDir, live, ghost := staleManifestProject(t)

	if err := RefreshExcelIn(xmlDir, excelDir); err != nil {
		t.Fatalf("RefreshExcelIn: %v", err)
	}

	if _, err := os.Stat(ghost); err == nil {
		t.Errorf("%s was re-created; a workbook that is not there is not a "+
			"source of truth and an export must not invent it",
			filepath.Base(ghost))
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("the workbook that does exist should still be refreshed: %v", err)
	}
}
