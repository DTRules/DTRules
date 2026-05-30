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

package authoring_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

const flatLayoutDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Test_Flat</table_name>
<xls_file>flat.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>always</condition_comment>
    <condition_dsl>job.flag</condition_dsl>
    <condition_postfix>job.flag</condition_postfix>
    <condition_column column_number="1" column_value="Y"/>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>noop</action_comment>
    <action_dsl></action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>
`

const flatLayoutEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="flag" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""/>
  </entity>
</entity_data_dictionary>
`

// TestOpenProject_FlatLayout is the #791 regression. Before the fix
// `authoring.OpenProject` required `<path>/xml/` to exist; library
// consumers whose rules live at e.g. `pkg/dtrules/rules/staking_dt.xml`
// (flat, no `xml/` subdir) got `no xml/ directory found in <path>` and
// every authoring surface that goes through `OpenProject` —
// `dtrules table list`, `dtrules table warnings`, `dtrules review` via
// the same loader — was unreachable for them.
//
// The fix falls back to the flat layout when no `xml/` subdir is
// present and the path itself contains `*_dt.xml`. This test pins
// that exact behavior: an OpenProject against the flat directory
// must succeed and expose the table.
func TestOpenProject_FlatLayout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flat_dt.xml"), []byte(flatLayoutDT), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flat_edd.xml"), []byte(flatLayoutEDD), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatalf("OpenProject on flat layout failed: %v", err)
	}
	if p.Table("Test_Flat") == nil {
		t.Errorf("expected Test_Flat to be discovered in flat layout, got %v", p.Tables())
	}
}

// TestOpenProject_CanonicalLayoutStillWins verifies that when both
// `<path>/xml/` and `<path>/*_dt.xml` exist, the canonical `xml/`
// subdir wins. Otherwise a project that has accidentally accumulated
// a stray `*_dt.xml` at the project root would suddenly load it
// instead of the canonical files — silent contract shift.
func TestOpenProject_CanonicalLayoutStillWins(t *testing.T) {
	dir := t.TempDir()
	// Decoy stray at the project root that must NOT be picked up.
	const decoyDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Decoy</table_name>
<xls_file>decoy.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>99</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
</decision_table>
</decision_tables>
`
	if err := os.WriteFile(filepath.Join(dir, "decoy_dt.xml"), []byte(decoyDT), 0644); err != nil {
		t.Fatal(err)
	}

	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmlDir, "flat_dt.xml"), []byte(flatLayoutDT), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	if p.Table("Test_Flat") == nil {
		t.Errorf("expected canonical xml/ table 'Test_Flat' to load, got %v", p.Tables())
	}
	if p.Table("Decoy") != nil {
		t.Error("decoy table at project root was loaded; canonical xml/ layout did not take precedence")
	}
}

// TestOpenProject_NeitherLayoutErrorsClearly confirms the error
// message names BOTH discovery paths so an author who hasn't seen
// either convention can find the right one. Before the fix the
// message only mentioned `xml/`.
func TestOpenProject_NeitherLayoutErrorsClearly(t *testing.T) {
	dir := t.TempDir()
	_, err := authoring.OpenProject(dir)
	if err == nil {
		t.Fatal("expected OpenProject to fail on an empty directory, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "xml/") {
		t.Errorf("error should mention canonical xml/ layout, got: %v", err)
	}
	if !strings.Contains(msg, "*_dt.xml") {
		t.Errorf("error should mention flat *_dt.xml fallback, got: %v", err)
	}
}

// TestOpenProject_XmlSymlinkLayout_Issue791 is the #791 regression.
// The staking project ships a layout where `<root>/xml` is a symlink
// to `<root>/rules` (the symlink lets `go:embed` find the files
// under `xml/...` paths). Pre-fix, `resolveProjectXMLDir` returned
// the symlink path and `filepath.WalkDir` treated the symlink root
// as a single non-directory entry (per Go's spec), so the walk never
// descended and the project loaded with zero tables. `dtrules table
// list --project <root>` returned `{"tables": []}` silently.
//
// The fix resolves the symlink via `filepath.EvalSymlinks` so the
// walker sees a real directory root.
func TestOpenProject_XmlSymlinkLayout_Issue791(t *testing.T) {
	root := t.TempDir()
	// Put the actual files under `rules/`, then create `xml -> rules`.
	rules := filepath.Join(root, "rules")
	if err := os.MkdirAll(rules, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rules, "flat_dt.xml"), []byte(flatLayoutDT), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rules, "flat_edd.xml"), []byte(flatLayoutEDD), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("rules", filepath.Join(root, "xml")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	p, err := authoring.OpenProject(root)
	if err != nil {
		t.Fatalf("OpenProject on symlinked xml/ layout failed: %v", err)
	}
	if p.Table("Test_Flat") == nil {
		t.Errorf("expected Test_Flat to be discovered through xml -> rules symlink, got %v", p.Tables())
	}
}
