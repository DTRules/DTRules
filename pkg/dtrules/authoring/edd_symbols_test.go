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

package authoring_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// TestLoadEDDSymbols is the parity unit for the single EDD→symbol builder that
// both the authoring Save path and the cmd/dtrules build path now share
// (#879). It pins: recursive discovery (top-level + nested EDDs), both the bare
// and entity-qualified key forms, and skipping of empty/typeless fields.
func TestLoadEDDSymbols(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	statesDir := filepath.Join(xmlDir, "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	topEDD := `<entity_data_dictionary version='2'>
		<entity name='budget'>
			<field name='supply_limit' type='fixed'></field>
			<field name='' type='fixed'></field>
		</entity>
	</entity_data_dictionary>`
	nestedEDD := `<entity_data_dictionary version='2'>
		<entity name='state'>
			<field name='rate' type='double'></field>
		</entity>
	</entity_data_dictionary>`

	if err := os.WriteFile(filepath.Join(xmlDir, "budget_edd.xml"), []byte(topEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(statesDir, "co_edd.xml"), []byte(nestedEDD), 0o644); err != nil {
		t.Fatal(err)
	}

	syms := authoring.LoadEDDSymbols(xmlDir)

	want := map[string]string{
		"supply_limit":        "fixed", // top-level, bare
		"budget.supply_limit": "fixed", // top-level, qualified
		"rate":                "double", // nested, bare (recursive discovery)
		"state.rate":          "double", // nested, qualified
	}
	for k, v := range want {
		if syms[k] != v {
			t.Errorf("LoadEDDSymbols[%q] = %q, want %q", k, syms[k], v)
		}
	}
	// Empty-named field must be skipped (no bare "" key, no "budget." key).
	if _, ok := syms[""]; ok {
		t.Error("empty field name was registered")
	}
	if _, ok := syms["budget."]; ok {
		t.Error("empty qualified key was registered")
	}
}
