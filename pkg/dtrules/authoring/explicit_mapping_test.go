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
	"strings"
	"testing"
)

// A mapping decides which entity each external tag lands in, so it is an
// argument to loading data rather than a property of the project --
// loadTestDataFromReaders builds a fresh Mapping per call and keeps none.
//
// findMapFile used to return the first match and say nothing, which is
// harmless while every project has exactly one and a silent wrong answer the
// moment one does not: per-state mappings would all have been ignored in
// favour of whichever sorted first.

func projectWithMaps(t *testing.T, names ...string) *Project {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(xmlDir, n), []byte("<mapping></mapping>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &Project{xmlDir: xmlDir}
}

func TestOneMappingIsFoundWithoutAsking(t *testing.T) {
	p := projectWithMaps(t, "P_map.xml")

	got, err := p.findMapFile()
	if err != nil {
		t.Fatalf("a single mapping should be found: %v", err)
	}
	if filepath.Base(got) != "P_map.xml" {
		t.Errorf("found %q, want P_map.xml", filepath.Base(got))
	}
}

func TestSeveralMappingsMustBeChosenBetween(t *testing.T) {
	p := projectWithMaps(t, "NV_map.xml", "AK_map.xml", "CA_map.xml")

	_, err := p.findMapFile()
	if err == nil {
		t.Fatal("with several mappings the project must not pick one; the " +
			"mapping decides where data lands and only the caller knows which")
	}
	// Naming them is what makes it actionable, and the way out has to be in
	// the message.
	for _, want := range []string{"AK_map.xml", "CA_map.xml", "NV_map.xml", "--map"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}

func TestNoMappingIsStillAnError(t *testing.T) {
	p := projectWithMaps(t)

	if _, err := p.findMapFile(); err == nil {
		t.Fatal("a project with no mapping cannot load mapped data")
	}
}
