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
	"os"
	"path/filepath"
	"testing"
)

// TestBuild_MultiFileNoDuplicateTables guards the multi-file authoring
// round-trip: building a project whose decision tables are split across
// several *_dt.xml files must not flatten every table into every file
// (which produced duplicate table names). Regression for the SinusitisTherapy
// rebuild corruption.
func TestBuild_MultiFileNoDuplicateTables(t *testing.T) {
	src := filepath.Join("..", "..", "sampleprojects", "SinusitisTherapy")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("sample project not present: %v", err)
	}
	dir := t.TempDir()
	if err := copyTree(src, dir); err != nil {
		t.Fatalf("copy sample: %v", err)
	}

	cli := NewCLI()
	if code := cli.runBuild([]string{dir}); code != 0 {
		t.Fatalf("build exit %d — a multi-file build must not corrupt the project", code)
	}

	findings, err := checkNoDupMarkers(filepath.Join(dir, "xml"))
	if err != nil {
		t.Fatalf("dup scan: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("build produced %d duplicate-table finding(s): %+v", len(findings), findings)
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
