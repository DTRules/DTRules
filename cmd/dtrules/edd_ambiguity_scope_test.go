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

// Requiring --edd-file is the EDD commands' business. `table get` works on
// decision tables and does not care which EDD is loaded; demanding a choice
// from it broke `dtrules table` outright on the one project that has several
// EDD files, and no test caught it because every other sample has one.

func multiEDDProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	states := filepath.Join(xmlDir, "states")
	if err := os.MkdirAll(states, 0o755); err != nil {
		t.Fatal(err)
	}
	edd := `<entity_data_dictionary version="2"><entity name="job" number="100">` +
		`<field name="age" type="integer" subtype="" access="rw" input="" default_value="" comment=""></field>` +
		`</entity></entity_data_dictionary>`
	dt := `<decision_tables><decision_table><table_name>Only</table_name>` +
		`<attribute_fields><Type>FIRST</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>` +
		`<contexts></contexts><initial_actions></initial_actions>` +
		`<conditions></conditions><actions></actions></decision_table></decision_tables>`
	for p, body := range map[string]string{
		filepath.Join(xmlDir, "Core_edd.xml"): edd,
		filepath.Join(states, "NV_edd.xml"):   edd,
		filepath.Join(xmlDir, "Core_dt.xml"):  dt,
	} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTableCommandsDoNotRequireAnEDDChoice(t *testing.T) {
	root := multiEDDProject(t)
	ctx := &tableCmdCtx{stdin: strings.NewReader(""), stdout: &strings.Builder{},
		stderr: &strings.Builder{}, projectPath: root}

	if _, code := ctx.openProject(); code != 0 {
		t.Fatalf("a table command must open a multi-EDD project without --edd-file; got exit %d", code)
	}
}

func TestEDDCommandsDoRequireAChoice(t *testing.T) {
	root := multiEDDProject(t)
	var errOut strings.Builder
	ctx := &tableCmdCtx{stdin: strings.NewReader(""), stdout: &strings.Builder{},
		stderr: &errOut, projectPath: root}

	if _, code := ctx.openProjectForEDD(); code == 0 {
		t.Fatal("an EDD command on a project with several EDD files must ask " +
			"which one, not serve the first")
	}
	if !strings.Contains(errOut.String(), "--edd-file") {
		t.Errorf("the error should name the way out, got: %s", errOut.String())
	}
}

func TestEDDCommandsAcceptTheChoice(t *testing.T) {
	root := multiEDDProject(t)
	ctx := &tableCmdCtx{stdin: strings.NewReader(""), stdout: &strings.Builder{},
		stderr: &strings.Builder{}, projectPath: root, eddFile: "states/NV_edd.xml"}

	if _, code := ctx.openProjectForEDD(); code != 0 {
		t.Fatalf("naming the EDD file should be enough; got exit %d", code)
	}
}
