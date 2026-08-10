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
	"testing"
)

// A missing excel/ is an error for every sync direction but one. Importing
// from a directory that is not there, or reporting status on it, means the
// caller is pointed at the wrong project. Exporting into it is bootstrapping,
// which the authoring contract provides for -- and refusing it is what left
// Cribbage, CorporateTax and SyntaxTests with no authoring surface at all
// (#1026, #1012).

func syncCLIWithoutExcelDir(t *testing.T) (*CLI, string) {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	excelDir := filepath.Join(root, "excel") // deliberately not created
	return &CLI{xmlDir: xmlDir, excelDir: excelDir}, excelDir
}

func TestSyncExportCreatesTheExcelDirectory(t *testing.T) {
	c, excelDir := syncCLIWithoutExcelDir(t)

	if err := c.initSyncerFor("export"); err != nil {
		t.Fatalf("export should bootstrap a missing excel dir, got: %v", err)
	}
	if _, err := os.Stat(excelDir); err != nil {
		t.Errorf("excel dir was not created: %v", err)
	}
}

func TestSyncImportAndStatusStillRefuseAMissingExcelDirectory(t *testing.T) {
	for _, subcmd := range []string{"import", "status", "check", "auto", ""} {
		t.Run("subcmd="+subcmd, func(t *testing.T) {
			c, excelDir := syncCLIWithoutExcelDir(t)

			if err := c.initSyncerFor(subcmd); err == nil {
				t.Fatalf("%q should refuse a missing excel dir rather than "+
					"create one — the caller is pointed at the wrong project", subcmd)
			}
			if _, err := os.Stat(excelDir); err == nil {
				t.Errorf("%q created the excel dir; only export may do that", subcmd)
			}
		})
	}
}

// A missing xml/ is always an error: there is nothing to export from.
func TestSyncExportStillRefusesAMissingXMLDirectory(t *testing.T) {
	root := t.TempDir()
	c := &CLI{
		xmlDir:   filepath.Join(root, "xml"), // not created
		excelDir: filepath.Join(root, "excel"),
	}
	if err := c.initSyncerFor("export"); err == nil {
		t.Fatal("export with no XML directory should be an error")
	}
}
