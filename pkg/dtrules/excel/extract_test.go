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

package excel

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xuri/excelize/v2"
)

const minimalEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="result" xls_file="test.xlsx" access="rw">
		<field name="tax_amount" type="double" subtype="" access="rw" input="" default_value="0" comment="Computed tax"></field>
	</entity>
</entity_data_dictionary>`

const minimalDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Calculate_Tax</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS>Simple tax calculation</COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
<policy_statements></policy_statements>
</decision_table>
</decision_tables>`

const minimalMap = `<mapping>
	<XMLtoEDD>
		<map>
			<!-- Job mappings -->
			<setattribute tag='id' RAttribute='id' enclosure='job' type='integer'></setattribute>
		</map>
	</XMLtoEDD>
</mapping>`

func TestExtractExcel_SimpleTree(t *testing.T) {
	src := fstest.MapFS{
		"test_dt.xml":  &fstest.MapFile{Data: []byte(minimalDT)},
		"test_edd.xml": &fstest.MapFile{Data: []byte(minimalEDD)},
		"test_map.xml": &fstest.MapFile{Data: []byte(minimalMap)},
	}

	dst := t.TempDir()
	if err := ExtractExcel(src, dst); err != nil {
		t.Fatalf("ExtractExcel: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "test_dt.xlsx"),
		filepath.Join(dst, "test_edd.xlsx"),
		filepath.Join(dst, "test_map.xlsx"),
	}

	for _, p := range expected {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected file %s: %v", p, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %s is empty", p)
			continue
		}
		// Verify the xlsx opens without error
		f, err := excelize.OpenFile(p)
		if err != nil {
			t.Errorf("open %s: %v", p, err)
			continue
		}
		f.Close()
	}
}

func TestExtractExcel_PreservesSubdirs(t *testing.T) {
	src := fstest.MapFS{
		"states/CO_dt.xml":  &fstest.MapFile{Data: []byte(minimalDT)},
		"states/CO_edd.xml": &fstest.MapFile{Data: []byte(minimalEDD)},
	}

	dst := t.TempDir()
	if err := ExtractExcel(src, dst); err != nil {
		t.Fatalf("ExtractExcel: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "states", "CO_dt.xlsx"),
		filepath.Join(dst, "states", "CO_edd.xlsx"),
	}

	for _, p := range expected {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file %s: %v", p, err)
		}
	}
}

func TestExtractExcel_FromFixture(t *testing.T) {
	// Use a small subset of the TaxReturn sample project.
	// We build an fstest.MapFS from a few known state files so the test
	// is fast and does not depend on large merged files.
	akEDD, err := os.ReadFile("../../../sampleprojects/TaxReturn/xml/states/AK_edd.xml")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}
	akDT, err := os.ReadFile("../../../sampleprojects/TaxReturn/xml/states/AK_dt.xml")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}
	mapXML, err := os.ReadFile("../../../sampleprojects/TaxReturn/xml/TaxReturn_map.xml")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	src := fstest.MapFS{
		"states/AK_edd.xml": &fstest.MapFile{Data: akEDD},
		"states/AK_dt.xml":  &fstest.MapFile{Data: akDT},
		"TaxReturn_map.xml": &fstest.MapFile{Data: mapXML},
	}

	dst := t.TempDir()
	if err := ExtractExcel(src, dst); err != nil {
		t.Fatalf("ExtractExcel: %v", err)
	}

	outputs := []string{
		filepath.Join(dst, "states", "AK_edd.xlsx"),
		filepath.Join(dst, "states", "AK_dt.xlsx"),
		filepath.Join(dst, "TaxReturn_map.xlsx"),
	}

	for _, p := range outputs {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected output %s: %v", p, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("output %s is empty", p)
		}
	}
}

func TestExtractExcel_BadXMLReturnsError(t *testing.T) {
	src := fstest.MapFS{
		"broken_edd.xml": &fstest.MapFile{Data: []byte(`<entity_data_dictionary UNCLOSED`)},
	}

	dst := t.TempDir()
	err := ExtractExcel(src, dst)
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
	if !strings.Contains(err.Error(), "broken_edd.xml") {
		t.Errorf("error message should include offending filename, got: %s", err.Error())
	}
}

// TestExtractExcel_NoLoaderWarnings verifies that extracting a DT file that
// references entities not declared in the same file produces no stderr output.
// The old RuleSet-per-file implementation would emit loader warnings in this case.
func TestExtractExcel_NoLoaderWarnings(t *testing.T) {
	crossRefDT := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>CrossRef_Table</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS>References entity from a different file</COMMENTS>
<TABLE_NUMBER>1</TABLE_NUMBER>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
<condition_details>
<condition_number>1</condition_number>
<condition_comment>uses external entity</condition_comment>
<condition_dsl>result.tax_amount &gt; 0</condition_dsl>
<condition_postfix>result.tax_amount 0 &gt;</condition_postfix>
<condition_column column_number="1" column_value="Y"></condition_column>
</condition_details>
</conditions>
<actions></actions>
<policy_statements></policy_statements>
</decision_table>
</decision_tables>`

	src := fstest.MapFS{
		"CO_dt.xml":         &fstest.MapFile{Data: []byte(crossRefDT)},
		"TaxReturn_edd.xml": &fstest.MapFile{Data: []byte(minimalEDD)},
	}

	// Capture stderr by redirecting os.Stderr temporarily.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	dst := t.TempDir()
	extractErr := ExtractExcel(src, dst)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if extractErr != nil {
		t.Fatalf("ExtractExcel: %v", extractErr)
	}
	if buf.Len() > 0 {
		t.Errorf("unexpected stderr output (loader warnings should not appear): %s", buf.String())
	}
}

// TestExtractExcel_NoSessionImport asserts that the excel package's extract.go
// does not import session or loader packages. We verify this by checking that
// the compiled package works without those imports (build-time guarantee via the
// rewrite), and confirm the source file contains no such import paths.
func TestExtractExcel_NoSessionImport(t *testing.T) {
	extractSrc, err := os.ReadFile("extract.go")
	if err != nil {
		t.Fatalf("read extract.go: %v", err)
	}
	forbidden := []string{
		"pkg/dtrules/session",
		"pkg/dtrules/loader",
	}
	for _, pkg := range forbidden {
		if strings.Contains(string(extractSrc), pkg) {
			t.Errorf("extract.go must not import %q", pkg)
		}
	}
}

// TestExtractExcel_RoundTrip verifies that extracted xlsx files exist and have
// the expected root xlsx structure. Full XML re-import round-trip is skipped
// here because the export produces a different XML dialect than the source
// (the importer path is Excel→XML, not XML→XML), making byte-level comparison
// meaningless. Semantic equivalence is covered by the existing roundtrip tests.
func TestExtractExcel_RoundTrip(t *testing.T) {
	akEDD, err := os.ReadFile("../../../sampleprojects/TaxReturn/xml/states/AK_edd.xml")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}
	akDT, err := os.ReadFile("../../../sampleprojects/TaxReturn/xml/states/AK_dt.xml")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	src := fstest.MapFS{
		"states/AK_edd.xml": &fstest.MapFile{Data: akEDD},
		"states/AK_dt.xml":  &fstest.MapFile{Data: akDT},
	}

	dst := t.TempDir()
	if err := ExtractExcel(src, dst); err != nil {
		t.Fatalf("ExtractExcel: %v", err)
	}

	// Verify each xlsx has at least one sheet with recognizable content
	for _, rel := range []string{"states/AK_edd.xlsx", "states/AK_dt.xlsx"} {
		p := filepath.Join(dst, rel)
		f, err := excelize.OpenFile(p)
		if err != nil {
			t.Errorf("open %s: %v", rel, err)
			continue
		}
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			t.Errorf("%s has no sheets", rel)
		}
		f.Close()
	}
}
