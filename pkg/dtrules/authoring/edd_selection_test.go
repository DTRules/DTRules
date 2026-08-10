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
)

// The Project works on one EDD at a time and used to take whichever sorted
// first, silently. CorporateTax has 52 EDD files -- a core plus 51 states --
// so 51 of them could not be read or written through the authoring API at
// all, and nothing said so.

func projectWithEDDs(t *testing.T, rel ...string) *Project {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	for _, r := range rel {
		p := filepath.Join(xmlDir, r)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		body := `<entity_data_dictionary version="2"><entity name="` +
			filepath.Base(filepath.Dir(p+"x")) + `"></entity></entity_data_dictionary>`
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &Project{xmlDir: xmlDir}
}

func TestEDDFilesFindsNestedOnesToo(t *testing.T) {
	p := projectWithEDDs(t, "Core_edd.xml", "states/NV_edd.xml", "states/AK_edd.xml")

	files := p.EDDFiles()
	if len(files) != 3 {
		t.Fatalf("found %d EDD files, want 3: %v", len(files), files)
	}
	// Top level first, so a project with one obvious EDD and nested extras
	// still resolves to the obvious one when nobody chooses.
	if filepath.Base(files[0]) != "Core_edd.xml" {
		t.Errorf("top-level EDD should come first, got %q", filepath.Base(files[0]))
	}
}

func TestUseEDDFileSelectsANestedOne(t *testing.T) {
	p := projectWithEDDs(t, "Core_edd.xml", "states/NV_edd.xml")

	if err := p.UseEDDFile("states/NV_edd.xml"); err != nil {
		t.Fatalf("UseEDDFile: %v", err)
	}
	if filepath.Base(p.eddFile) != "NV_edd.xml" {
		t.Errorf("selected %q, want NV_edd.xml", p.eddFile)
	}
}

func TestUseEDDFileRejectsOneThatIsNotThere(t *testing.T) {
	p := projectWithEDDs(t, "Core_edd.xml")

	if err := p.UseEDDFile("states/XX_edd.xml"); err == nil {
		t.Fatal("selecting a file that does not exist must fail, not silently " +
			"fall back to another EDD")
	}
}

// Ordering must not depend on the filesystem, or which EDD a project resolves
// to would vary by machine.
func TestEDDCandidateOrderIsStable(t *testing.T) {
	p := projectWithEDDs(t, "Core_edd.xml", "states/NV_edd.xml", "states/AK_edd.xml", "states/CA_edd.xml")

	first := p.EDDFiles()
	for i := 0; i < 5; i++ {
		got := p.EDDFiles()
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("order changed between calls: %v vs %v", first, got)
			}
		}
	}
}
