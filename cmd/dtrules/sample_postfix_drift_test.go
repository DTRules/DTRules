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

// TestSampleProjectsMatchTheirBuild (#1260): every sample project's stored
// XML — the compiled postfix that actually executes — is what building it
// from its Excel source produces today. Many stored rows are executed by
// nothing (SyntaxTests is a catalogue whose tables mostly cannot run, and
// whose one runnable table has no evaluated conditions), so when a compiler
// change alters what a row compiles to, only this comparison sees it.
//
// CI's verify workflow runs the same check, but only for pull requests that
// touch the paths it filters on; running it here makes it part of every
// `make check`, whichever package the change is in.
func TestSampleProjectsMatchTheirBuild(t *testing.T) {
	root := filepath.Join("..", "..", "sampleprojects")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("sampleprojects not found: %v", err)
	}
	checked := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if _, err := os.Stat(filepath.Join(dir, "excel")); err != nil {
			continue // the CI workflow skips projects with no excel/ too
		}
		checked++
		t.Run(e.Name(), func(t *testing.T) {
			tmp := t.TempDir()
			if err := copyDir(dir, tmp); err != nil { // copies to tmp/<name>
				t.Fatalf("copy %s: %v", dir, err)
			}
			if code := NewCLI().runVerify([]string{filepath.Join(tmp, e.Name())}); code != 0 {
				t.Errorf("verify %s exited %d: its stored XML is not what `dtrules build` produces from its Excel source; rebuild it with `dtrules build sampleprojects/%s --from-excel` and review the diff", e.Name(), code, e.Name())
			}
		})
	}
	if checked == 0 {
		t.Fatal("no sample project with an excel/ directory was found")
	}
}

// Test fixtures with an excel/ directory are held to the same standard as
// the samples. test/vectors/entity-comment is a project whose `policy`
// entity is declared by two EDD files with two different comments: no
// sample's workbooks carry an entity comment, so without it the feature is
// exercised by nothing verify checks (#1306).
func TestVectorProjectsMatchTheirBuild(t *testing.T) {
	root := filepath.Join("..", "..", "test", "vectors")
	checked := 0
	_ = filepath.WalkDir(root, func(dir string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if _, err := os.Stat(filepath.Join(dir, "excel")); err != nil {
			return nil
		}
		if _, err := os.Stat(filepath.Join(dir, "xml")); err != nil {
			return nil
		}
		checked++
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			if err := copyDir(dir, tmp); err != nil {
				t.Fatalf("copy %s: %v", dir, err)
			}
			if code := NewCLI().runVerify([]string{filepath.Join(tmp, name)}); code != 0 {
				t.Errorf("verify %s exited %d: its stored XML is not what `dtrules build` produces from its Excel source", dir, code)
			}
		})
		return filepath.SkipDir
	})
	if checked == 0 {
		t.Fatal("no test/vectors project with an excel/ directory was found")
	}
}
