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

// The EDD sheet has no cell for an entity comment, so a dictionary that has
// one is still kept as XML rather than lose it. That project cannot yet pass
// verify; losing the comment silently would be worse.
func TestBootstrapKeepsAnEntityComment(t *testing.T) {
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
}
