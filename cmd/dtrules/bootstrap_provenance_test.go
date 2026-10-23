package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1303: the first authoring write to a project with no Excel bootstraps its
// workbooks (#1215), but only the tables' ones. The EDD's entities named no
// workbook, so none exported them; the mapping was not exported at all. Both
// stayed rule XML that nothing produces, and verify's provenance check (#1301)
// failed every bootstrapped project.

const bootstrapFixture = "../../test/vectors/otherwise/ok"

const bootstrapTable = `{"name":"A_put","policy":"FIRST",
  "conditions":[{"number":1,"dsl":"policy.flag == 7","columns":{"1":"Y","2":"*"}}],
  "actions":[{"number":1,"dsl":"set result.fired = \"x\"","columns":{"1":true,"2":false}}]}`

func bootstrap(t *testing.T, edit func(xmlDir string)) string {
	t.Helper()
	proj := copyProject(t, bootstrapFixture)
	if _, err := os.Stat(filepath.Join(proj, "excel")); err == nil {
		t.Fatal("fixture already has excel/: nothing to bootstrap")
	}
	if edit != nil {
		edit(filepath.Join(proj, "xml"))
	}
	mustRun(t, proj, []string{"put", "A_put", "--file", "a_dt.xml", "--range", "9500-9599", "--reason", "bootstrap"},
		bootstrapTable)
	return proj
}

func TestBootstrapBacksEveryRuleFileWithAWorkbook(t *testing.T) {
	proj := bootstrap(t, nil)
	for _, wb := range []string{"a.xlsx", "v.xlsx", "v_map.xlsx"} {
		assertExists(t, filepath.Join(proj, "excel", wb))
	}
	assertVerifies(t, proj)
}

// An entity comment travels in the Description cell of the entity's header
// row on the EDD sheet (#1306), so a dictionary that has one bootstraps like
// any other: its XML becomes what compiling the new workbook produces, and
// still says what it said.
func TestBootstrapRoundTripsAnEntityComment(t *testing.T) {
	proj := bootstrap(t, func(xmlDir string) {
		p := filepath.Join(xmlDir, "v_edd.xml")
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		edited := strings.Replace(string(data), `<entity name="policy"`, `<entity name="policy" comment="the case under test"`, 1)
		if err := os.WriteFile(p, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	data, err := os.ReadFile(filepath.Join(proj, "xml", "v_edd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `comment="the case under test"`) {
		t.Errorf("bootstrap lost the entity comment:\n%s", data)
	}
	assertVerifies(t, proj)
}

// Two EDD files that comment the same entity each keep their own through a
// bootstrap. One comment per entity, last file wins, deleted the first
// file's comment with no warning -- `main` at least failed verify on it.
func TestBootstrapKeepsEveryFilesEntityComment(t *testing.T) {
	const second = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="policy" number="100" access="rw" comment="second">
		<field name="other" type="string" subtype="" access="rw" input="" default_value="" comment=""></field>
	</entity>
</entity_data_dictionary>
`
	proj := bootstrap(t, func(xmlDir string) {
		p := filepath.Join(xmlDir, "v_edd.xml")
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		edited := strings.Replace(string(data), `<entity name="policy"`, `<entity name="policy" comment="first"`, 1)
		if err := os.WriteFile(p, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xmlDir, "w_edd.xml"), []byte(second), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	for file, want := range map[string]string{"v_edd.xml": `comment="first"`, "w_edd.xml": `comment="second"`} {
		data, err := os.ReadFile(filepath.Join(proj, "xml", file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s lost its entity comment:\n%s", file, data)
		}
	}
	assertVerifies(t, proj)
}
