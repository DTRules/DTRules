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
	"time"
)

// --from-excel selects the Excel-authored code path, but SyncAll used to
// re-derive direction per workbook from mtimes and answer NoSync whenever the
// XML was not older -- so the import did not run and the command reported
// "XML is already up to date" with tables=0 and files-written=0. Content drift
// was then uncorrectable through build at any timestamp, while verify's
// rebuild always saw the current format, so verify stayed red forever (#1051).

// touchNewer sets every file matching pattern to a time well after every file
// in other, putting the two directories in the state that used to defeat the
// import: XML no older than Excel.
func touchNewer(t *testing.T, dir, pattern string) {
	t.Helper()
	future := time.Now().Add(time.Hour)
	paths, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(paths) == 0 {
		t.Fatalf("glob %s/%s: %v (%d matches)", dir, pattern, err, len(paths))
	}
	for _, p := range paths {
		if err := os.Chtimes(p, future, future); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBuildFromExcelImportsEvenWhenXMLIsNotOlder(t *testing.T) {
	project := copyProject(t, filepath.Join("..", "..", "sampleprojects", "SinusitisTherapy"))
	xmlDir := filepath.Join(project, "xml")
	excelDir := filepath.Join(project, "excel")

	// The condition that used to silence the import.
	touchNewer(t, xmlDir, "*.xml")

	cli := NewCLI()
	opts := &buildOptions{fromExcel: true}
	if code := cli.runExcelAuthoredBuild(xmlDir, excelDir, opts); code != 0 {
		t.Fatalf("runExcelAuthoredBuild returned %d, want 0", code)
	}

	// The observable claim: the workbooks were actually read. Before the fix
	// this path completed successfully having imported nothing, so every
	// *_dt.xml kept the future mtime forced above. A real import rewrites
	// them, stamping them with now.
	paths, _ := filepath.Glob(filepath.Join(xmlDir, "*_dt.xml"))
	if len(paths) == 0 {
		t.Fatal("no *_dt.xml in the fixture")
	}
	cutoff := time.Now().Add(30 * time.Minute)
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.ModTime().After(cutoff) {
			t.Errorf("%s still carries its forced future mtime -- the import was skipped",
				filepath.Base(p))
		}
	}
}
