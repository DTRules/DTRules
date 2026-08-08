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
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func TestQuestionSummary(t *testing.T) {
	num := func(low, high, units string) authoring.Attribute {
		return authoring.Attribute{
			Collect: "true", QuestionType: "number",
			QuestionRefLow: low, QuestionRefHigh: high, QuestionUnits: units,
		}
	}
	cases := []struct {
		name string
		attr authoring.Attribute
		want string
	}{
		{"not collected", authoring.Attribute{Collect: "false", QuestionType: "number"}, ""},
		{"both bounds", num("0.7", "1.3", "mg/dL"), "number normal 0.7–1.3 mg/dL"},
		{"low only", num("0.7", "", "mg/dL"), "number normal ≥ 0.7 mg/dL"},
		{"high only", num("", "1.3", "mg/dL"), "number normal ≤ 1.3 mg/dL"},
		{"units only", num("", "", "years"), "number in years"},
		{"no metadata", num("", "", ""), "number"},
		{"multiple choice", authoring.Attribute{
			Collect: "true", QuestionType: "multiple_choice",
			Options: []authoring.Option{{Value: "y", Label: "Yes"}, {Value: "n", Label: "No"}},
		}, "multiple_choice (Yes, No)"},
		{"option value fallback", authoring.Attribute{
			Collect: "true", QuestionType: "multiple_choice",
			Options: []authoring.Option{{Value: "A"}, {Value: "B"}},
		}, "multiple_choice (A, B)"},
	}
	for _, c := range cases {
		if got := questionSummary(c.attr); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPolicyLabel(t *testing.T) {
	for in, want := range map[string]string{
		"": "BALANCED", "balanced": "BALANCED", "first": "FIRST",
		"FIRST": "FIRST", "all": "ALL", " first ": "FIRST",
	} {
		if got := policyLabel(in); got != want {
			t.Errorf("policyLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIOLabel(t *testing.T) {
	for in, want := range map[string]string{
		"r": "input", "w": "output", "rw": "in·out", "": "—", "R": "input", "junk": "—",
	} {
		if got := ioLabel(in); got != want {
			t.Errorf("ioLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTypeLabel(t *testing.T) {
	if got := typeLabel("array", "med"); got != "array of med" {
		t.Errorf("got %q", got)
	}
	if got := typeLabel("string", ""); got != "string" {
		t.Errorf("got %q", got)
	}
}

func TestProjectName(t *testing.T) {
	for in, want := range map[string]string{
		"/a/b/Sinusitis": "Sinusitis", "Sinusitis/": "Sinusitis",
		".": "DTRules Project", "": "DTRules Project",
	} {
		if got := projectName(in); got != want {
			t.Errorf("projectName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMakeTableViewCells(t *testing.T) {
	tbl := &authoring.Table{
		Name:   "T",
		Policy: "FIRST",
		Conditions: []authoring.Condition{
			{Number: 1, DSL: "a > 0", Columns: map[int]string{1: "Y", 2: "N"}}, // col 3 unset → "-"
		},
		Actions: []authoring.Action{
			{Number: 1, DSL: "set x = 1", Columns: map[int]bool{1: true, 3: true}},
		},
	}
	v := makeTableView(tbl)
	if len(v.Cols) != 3 {
		t.Fatalf("expected 3 columns (max index from action), got %d", len(v.Cols))
	}
	if got := v.Conditions[0].Cells; got[0] != "Y" || got[1] != "N" || got[2] != "-" {
		t.Errorf("condition cells = %v, want [Y N -]", got)
	}
	if got := v.Actions[0].Cells; got[0] != "X" || got[1] != "" || got[2] != "X" {
		t.Errorf("action cells = %v, want [X \"\" X]", got)
	}
}

// Golden end-to-end: a tiny project renders an HTML doc that contains the
// EDD, the table grid, the collected-field question, and escapes user text.
func TestRunGolden(t *testing.T) {
	dir := t.TempDir()
	edd := `<entity_data_dictionary>
  <entity name="patient">
    <field name="age" type="integer" access="rw" collect="true">
      <question text="Age?" type="number" units="years"></question>
    </field>
  </entity>
</entity_data_dictionary>`
	dt := `<decision_tables><decision_table>
  <table_name>Decide</table_name>
  <attribute_fields><Type>FIRST</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
  <conditions><condition_details><condition_number>1</condition_number>
    <condition_dsl>patient.age &gt;= 18 and "a &amp; b"</condition_dsl>
    <condition_column column_number="1" column_value="Y" /></condition_details></conditions>
  <actions><action_details><action_number>1</action_number>
    <action_dsl>set patient.age = 0</action_dsl>
    <action_column column_number="1" column_value="X" /></action_details></actions>
</decision_table></decision_tables>`
	if err := os.WriteFile(filepath.Join(dir, "p_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.html")
	if err := run(dir, out, "Tiny"); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	for _, want := range []string{"Tiny", "patient", "age", "Decide", "100", "number in years", "●"} {
		if !strings.Contains(html, want) {
			t.Errorf("output missing %q", want)
		}
	}
	// HTML-escaping: the DSL's & and < must be escaped, not raw.
	if strings.Contains(html, `"a & b"`) || strings.Contains(html, "age >= 18") {
		t.Errorf("user DSL not HTML-escaped")
	}
	if !strings.Contains(html, "&amp;") {
		t.Errorf("expected escaped &amp; in output")
	}
}

// publishedDoc is the one generated document this repo actually ships:
// website/public/sinusitis-decision-tables.pdf, rendered from the web demo's
// rules. Regenerate both with
//
//	go run ./cmd/tabledoc -o sinusitis.html -title SinusitisTherapy cmd/sinusitis-web/rules
//	google-chrome --headless --disable-gpu --no-pdf-header-footer \
//	  --print-to-pdf=website/public/sinusitis-decision-tables.pdf sinusitis.html
//	sha256sum sinusitis.html   # paste below
//
// The --no-pdf-header-footer matters: without it Chrome stamps every page with
// the date and the source file:// URL, which is how you can tell a regenerated
// PDF apart from the shipped one.
const (
	publishedRulesDir  = "../sinusitis-web/rules"
	publishedDocTitle  = "SinusitisTherapy"
	publishedDocDigest = "11c25be19a08b4d8ca250a18ff0d38bf5d40786048af99006176445e6401b09f"
)

// TestPublishedPDFIsCurrent fails when the rules change without the shipped PDF
// being regenerated.
//
// The PDF is a committed binary; nothing about editing a rule forces it to be
// rebuilt, so without this it goes quietly stale and the website serves a
// reference document that no longer describes the rules — the same
// documentation-drifts-from-behaviour failure #961 found in el-reference.md.
//
// It guards the rendered HTML rather than the PDF because that is what can be
// checked without Chrome installed: the renderer's output is deterministic, so
// a digest over it covers every rule change that reaches the page. A change
// visible only in the PDF (styling, print flags) is not caught, and does not
// need to be.
func TestPublishedPDFIsCurrent(t *testing.T) {
	if _, err := os.Stat(publishedRulesDir); err != nil {
		t.Skipf("web demo rules not present: %v", err)
	}
	out := filepath.Join(t.TempDir(), "published.html")
	if err := run(publishedRulesDir, out, publishedDocTitle); err != nil {
		t.Fatalf("render %s: %v", publishedRulesDir, err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(data))
	if got != publishedDocDigest {
		t.Errorf("rules changed but website/public/sinusitis-decision-tables.pdf was not regenerated\n"+
			"  rendered digest: %s\n"+
			"  recorded digest: %s\n"+
			"Regenerate the PDF and update publishedDocDigest — see the comment above this test.",
			got, publishedDocDigest)
	}
}
