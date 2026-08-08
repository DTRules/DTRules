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

// TestSyncExportWritesMapWorkbook pins #1036.
//
// syncMAPFiles was called only from build.go, so `dtrules sync export` and
// `dtrules sync import` ignored map files entirely: an export wrote the
// decision-table workbook and left the map workbook untouched, however stale.
// A later `build --from-excel` — which does import maps — then read that stale
// sheet.
//
// One project should not have two answers about what "sync" covers, and the
// half that was missing is the half that keeps the map workbook current.
func TestSyncExportWritesMapWorkbook(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	excelDir := filepath.Join(dir, "excel")
	for _, d := range []string{xmlDir, excelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	src := filepath.Join("..", "..", "sampleprojects", "TestProject", "xml")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Skipf("TestProject not available: %v", err)
	}
	var haveMap bool
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xmlDir, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(e.Name(), "_map.xml") {
			haveMap = true
		}
	}
	if !haveMap {
		t.Skip("TestProject has no map file")
	}

	c := NewCLI()
	c.xmlDir, c.excelDir = xmlDir, excelDir
	if err := c.syncMAPFiles(xmlDir, excelDir, "xml-to-excel", false); err != nil {
		t.Fatalf("syncMAPFiles: %v", err)
	}

	got, err := os.ReadDir(excelDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got {
		if strings.HasSuffix(e.Name(), "_map.xlsx") {
			return // the map workbook was written
		}
	}
	t.Errorf("no *_map.xlsx written to %s — the map was left out of the export", excelDir)
}
