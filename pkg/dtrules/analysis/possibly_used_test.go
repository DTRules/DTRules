package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// possiblyUsedFixture writes a project whose orchestrator reaches
// Determine_CA_Thing through a dynamic dispatch. dispatch is the exact DSL of
// that site, so a test can hand it a derived bound, an `among` list, or a
// static perform beside it.
func possiblyUsedFixture(t *testing.T, dispatch string) string {
	t.Helper()
	dir := t.TempDir()
	const edd = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="job" access="rw">
    <field name="kind" type="string" subtype="" access="r" input="main" default_value="" comment=""></field>
  </entity>
  <entity name="result" access="rw">
    <field name="definite" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
    <field name="possible" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
    <field name="never" type="double" subtype="" access="rw" input="" default_value="0" comment=""></field>
  </entity>
</entity_data_dictionary>
`
	table := func(name, condition, action string) string {
		return fmt.Sprintf(`<decision_table>
<table_name>%s</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_dsl>%s</condition_dsl>
    <condition_postfix></condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>%s</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"></action_column>
  </action_details>
</actions>
<policy_statements></policy_statements>
</decision_table>
`, name, condition, action)
	}
	dt := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
` + table("Orchestrator", "perform when called", "perform Static_Thing; "+dispatch) +
		table("Static_Thing", "result.definite > 0", "set result.definite = 1") +
		table("Determine_CA_Thing", "result.possible > 0", "set result.possible = 1") +
		`</decision_tables>
`
	if err := os.WriteFile(filepath.Join(dir, "test_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test_dt.xml"), []byte(dt), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func categoriesOf(t *testing.T, dir string) map[string]EDDUsageCategory {
	t.Helper()
	warnings, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("AnalyzeEDDUsage: %v", err)
	}
	got := map[string]EDDUsageCategory{}
	for _, w := range warnings {
		got[w.Field] = w.Category
	}
	return got
}

// A field referenced only in a table that a dispatch *could* name is neither
// used nor unused: the derived bound over-approximates, and nothing else
// reaches the table (#776).
func TestPossiblyUsed_DerivedBoundOnly(t *testing.T) {
	dir := possiblyUsedFixture(t, `perform table named ("Determine_" + job.kind + "_Thing")`)
	got := categoriesOf(t, dir)
	if got["result.possible"] != EDDUsagePossibly {
		t.Errorf("result.possible: category %q, want possibly_used — its only reference is in Determine_CA_Thing, reached through the derived bound alone", got["result.possible"])
	}
	if got["result.never"] != EDDUsageUnused {
		t.Errorf("result.never: category %q, want unused", got["result.never"])
	}
	if c, ok := got["result.definite"]; ok {
		t.Errorf("result.definite: reported %q, want no finding — Static_Thing is performed outright", c)
	}
}

// An `among` list is the author's declared bound: exact edges, so the target
// is definitely reached and its references count in full.
func TestPossiblyUsed_AmongIsDefinite(t *testing.T) {
	dir := possiblyUsedFixture(t, `perform table named ("Determine_" + job.kind + "_Thing") among Determine_CA_Thing`)
	got := categoriesOf(t, dir)
	if c, ok := got["result.possible"]; ok {
		t.Errorf("result.possible: reported %q, want no finding — Determine_CA_Thing is listed in among", c)
	}
}

// A static perform beside the derived bound sanctions the same edge, so the
// table is definitely reached whatever the dispatch also matches.
func TestPossiblyUsed_StaticEdgeWins(t *testing.T) {
	dir := possiblyUsedFixture(t, `perform table named ("Determine_" + job.kind + "_Thing"); perform Determine_CA_Thing`)
	got := categoriesOf(t, dir)
	if c, ok := got["result.possible"]; ok {
		t.Errorf("result.possible: reported %q, want no finding — Orchestrator also performs Determine_CA_Thing directly", c)
	}
	graph, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatal(err)
	}
	if graph.DerivedCalls["Orchestrator"]["Determine_CA_Thing"] {
		t.Error("the edge is recorded as derived although a static perform sanctions it")
	}
	if !graph.Calls["Orchestrator"]["Determine_CA_Thing"] {
		t.Error("the edge is missing from Calls")
	}
}
