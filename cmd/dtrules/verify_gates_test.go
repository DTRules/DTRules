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

	"github.com/DTRules/DTRules/pkg/dtrules/analysis"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// gateProject writes an xml/ (and optionally excel/) tree under a temp dir
// and returns the resolved xml and excel dirs.
func gateProject(t *testing.T, dtXML, eddXML string, withExcel bool) (xmlDir, excelDir string) {
	t.Helper()
	root := t.TempDir()
	xmlDir = filepath.Join(root, "xml")
	excelDir = filepath.Join(root, "excel")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatal(err)
	}
	if eddXML != "" {
		if err := os.WriteFile(filepath.Join(xmlDir, "p_edd.xml"), []byte(eddXML), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if dtXML != "" {
		if err := os.WriteFile(filepath.Join(xmlDir, "p_dt.xml"), []byte(dtXML), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if withExcel {
		if err := os.MkdirAll(excelDir, 0755); err != nil {
			t.Fatal(err)
		}
		// hasWorkbook only checks for a .xlsx file by name.
		if err := os.WriteFile(filepath.Join(excelDir, "p.xlsx"), []byte("stub"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return xmlDir, excelDir
}

const gateEDD = `<entity_data_dictionary version='2'>
  <entity name='client' access='rw'>
    <field name='age' type='integer' access='rw' />
  </entity>
</entity_data_dictionary>`

const gateCleanDT = `<decision_tables>
<decision_table><table_name>Entry</table_name>
<conditions><condition_details><condition_dsl>client.age &gt; 18</condition_dsl>
<condition_postfix>client.age 18 f&gt;</condition_postfix></condition_details></conditions>
</decision_table>
</decision_tables>`

// TestCheckExcelPresence covers the gate that catches rules authored straight
// into XML with no Excel system-of-record.
func TestCheckExcelPresence(t *testing.T) {
	// Rule XML but no Excel ⇒ failure.
	xmlDir, excelDir := gateProject(t, gateCleanDT, gateEDD, false)
	if fails := checkExcelPresence(xmlDir, excelDir); len(fails) == 0 {
		t.Error("expected an excel-presence failure when rule XML has no workbook")
	}

	// Rule XML with a workbook ⇒ no failure.
	xmlDir2, excelDir2 := gateProject(t, gateCleanDT, gateEDD, true)
	if fails := checkExcelPresence(xmlDir2, excelDir2); len(fails) != 0 {
		t.Errorf("expected no excel-presence failure when a workbook exists, got %v", fails)
	}
}

// TestCheckExternalRefs covers the gate that rejects tables depending on
// undefined tables, fields, or operators.
func TestCheckExternalRefs(t *testing.T) {
	badDT := `<decision_tables>
<decision_table><table_name>Bad</table_name>
<conditions><condition_details><condition_dsl>client.bogus &gt; 5</condition_dsl>
<condition_postfix>client.age 5 madeupop</condition_postfix></condition_details></conditions>
<actions><action_details><action_dsl>perform Nonexistent_Table</action_dsl>
<action_postfix>/Nonexistent_Table performtable</action_postfix></action_details></actions>
</decision_table>
</decision_tables>`
	xmlDir, _ := gateProject(t, badDT, gateEDD, true)
	fails := checkExternalRefs(xmlDir)
	if len(fails) == 0 {
		t.Fatal("expected external-reference failures for the bad project")
	}

	// A clean project produces none.
	xmlDirClean, _ := gateProject(t, gateCleanDT, gateEDD, true)
	if fails := checkExternalRefs(xmlDirClean); len(fails) != 0 {
		t.Errorf("expected no external-reference failures for a clean project, got %v", fails)
	}
}

// TestConformantSampleHasNoExternalRefs is the regression guard: a project
// authored under the contract (compiled postfix, complete EDD) must produce
// zero external-reference findings against the real operator registry. This
// is what pins the resolution rules so a future change can't reintroduce the
// false positives (declared entities/fields/keywords/locals mistaken for
// undefined operators).
func TestConformantSampleHasNoExternalRefs(t *testing.T) {
	xmlDir := filepath.FromSlash("../../sampleprojects/SinusitisTherapy/xml")
	if _, err := os.Stat(xmlDir); err != nil {
		t.Skipf("SinusitisTherapy sample not found: %v", err)
	}
	isOperator := func(tok string) bool {
		_, ok := operators.GetByString(tok)
		return ok
	}
	findings, err := analysis.AnalyzeExternalRefs(xmlDir, isOperator)
	if err != nil {
		t.Fatalf("AnalyzeExternalRefs: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("conformant sample must have no external-reference findings, got %d:\n%v",
			len(findings), findings)
	}
}
