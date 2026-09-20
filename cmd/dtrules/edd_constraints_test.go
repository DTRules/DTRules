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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// addDiagnosisVocabulary is the patch the issue describes: give
// patient.diagnosis the closed vocabulary and length limit it always had in
// prose but never in the EDD (#1209).
const addDiagnosisVocabulary = `{"op":"update-field","entity":"patient","field":{
  "name":"diagnosis",
  "allowed_values":["Acute Sinusitis","Chronic Sinusitis"],
  "max_length":"40"
}}`

// TestEDDPatchCarriesConstraintsToXMLAndExcel is vector 10: one
// `edd patch` must land the constraints in the XML, in the workbook, and in
// what `edd get` reports — in a single operation, and with `verify` still
// clean afterwards. Excel is the system of record; XML alone is a divergence
// CI catches later, blaming nobody.
func TestEDDPatchCarriesConstraintsToXMLAndExcel(t *testing.T) {
	if testing.Short() {
		t.Skip("verify round trip rebuilds the project; skipped in the CI fast gate")
	}
	project := copyProject(t, filepath.Join("..", "..", "sampleprojects", "SinusitisTherapy"))

	if _, stderr, exit := runEDDCmd(t, project, []string{"patch"}, addDiagnosisVocabulary); exit != 0 {
		t.Fatalf("edd patch failed (%d): %s", exit, stderr)
	}

	eddXML := filepath.Join(project, "xml", "sinusitis_edd.xml")
	data, err := os.ReadFile(eddXML)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`max_length="40"`,
		`<allowed_value value="Acute Sinusitis">`,
		`<allowed_value value="Chronic Sinusitis">`,
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("the EDD XML does not carry %s", want)
		}
	}

	workbook := filepath.Join(project, "excel", "sinusitis.xlsx")
	shared := readSharedStrings(t, workbook)
	if !strings.Contains(shared, "Acute Sinusitis|Chronic Sinusitis") {
		t.Errorf("the workbook did not receive the vocabulary in the same operation")
	}
	if !strings.Contains(shared, "Allowed Values") {
		t.Errorf("the EDD sheet has no Allowed Values column")
	}

	// `edd get` must report back what was written.
	stdout, stderr, exit := runEDDCmd(t, project, []string{"get"}, "")
	if exit != 0 {
		t.Fatalf("edd get failed (%d): %s", exit, stderr)
	}
	var got EDDJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("edd get returned unparseable JSON: %v", err)
	}
	found := false
	for _, ent := range got.Entities {
		if !strings.EqualFold(ent.Name, "patient") {
			continue
		}
		for _, f := range ent.Fields {
			if !strings.EqualFold(f.Name, "diagnosis") {
				continue
			}
			found = true
			if len(f.AllowedValues) != 2 || f.AllowedValues[0] != "Acute Sinusitis" {
				t.Errorf("edd get: allowed_values = %v", f.AllowedValues)
			}
			if f.MaxLength != "40" {
				t.Errorf("edd get: max_length = %q, want \"40\"", f.MaxLength)
			}
			if f.Collect != "true" || f.QuestionText == "" {
				t.Errorf("the constraint patch disturbed the collect metadata: %+v", f)
			}
		}
	}
	if !found {
		t.Fatal("edd get did not report patient.diagnosis")
	}

	// The whole point: XML, workbook and provenance agree with no manual
	// `sync export`.
	cli := NewCLI()
	if code := cli.runVerify([]string{project}); code != 0 {
		t.Errorf("verify failed after an edd patch: XML and Excel diverged (exit %d)", code)
	}

	// Vector 11, the human path: rebuild the XML from the workbooks. The
	// constraints must come back out of the Allowed Values column, because
	// Excel — not the XML the patch happened to write — is the system of
	// record. An importer that ignored the column would regenerate this EDD
	// without them.
	if code := NewCLI().runExcelAuthoredBuild(filepath.Join(project, "xml"),
		filepath.Join(project, "excel"), &buildOptions{fromExcel: true}); code != 0 {
		t.Fatalf("build from Excel returned %d, want 0", code)
	}
	rebuilt, err := os.ReadFile(eddXML)
	if err != nil {
		t.Fatalf("build did not regenerate the EDD: %v", err)
	}
	for _, want := range []string{`max_length="40"`, `<allowed_value value="Acute Sinusitis">`} {
		if !strings.Contains(string(rebuilt), want) {
			t.Errorf("a build from Excel lost %s", want)
		}
	}
}

// TestValidateRejectsADefaultOutsideItsVocabulary is vector 9. A default the
// field's own allowed_values reject can never be corrected by an input — the
// constraint applies to supplied values — so it is an error at validate time,
// not a surprise at run time.
func TestValidateRejectsADefaultOutsideItsVocabulary(t *testing.T) {
	project := copyProject(t, filepath.Join("..", "..", "sampleprojects", "SinusitisTherapy"))
	eddXML := filepath.Join(project, "xml", "sinusitis_edd.xml")

	// A vocabulary that does not admit the declared default "Acute Sinusitis".
	// Written directly because the authoring API refuses to write it, which is
	// itself the point: validate is the net under an EDD that arrived some
	// other way.
	data, err := os.ReadFile(eddXML)
	if err != nil {
		t.Fatal(err)
	}
	const old = `<question text="Working diagnosis?" type="ascii"></question>`
	const bad = old + "\n\t\t\t<allowed_value value=\"Chronic Sinusitis\"></allowed_value>"
	if !strings.Contains(string(data), old) {
		t.Fatalf("fixture drift: patient.diagnosis no longer carries the expected question")
	}
	if err := os.WriteFile(eddXML, []byte(strings.Replace(string(data), old, bad, 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr := captureStderr(t, func() {
		if code := NewCLI().runValidate([]string{project}); code == 0 {
			t.Errorf("validate accepted a default outside its own allowed_values")
		}
	})
	for _, want := range []string{"diagnosis", "Acute Sinusitis", "Chronic Sinusitis"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the validate error does not name %q:\n%s", want, stderr)
		}
	}

	// And the authoring API refuses the same declaration up front.
	project2 := copyProject(t, filepath.Join("..", "..", "sampleprojects", "SinusitisTherapy"))
	_, apiErr, exit := runEDDCmd(t, project2, []string{"patch"},
		`{"op":"update-field","entity":"patient","field":{"name":"diagnosis","allowed_values":["Chronic Sinusitis"]}}`)
	if exit == 0 {
		t.Errorf("edd patch accepted a vocabulary that excludes the field's own default")
	}
	if !strings.Contains(apiErr, "allowed values") {
		t.Errorf("the patch error does not explain itself: %s", apiErr)
	}
}
