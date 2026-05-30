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

package analysis

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// writeDTFile is a tiny test helper for writing a single-table
// `*_dt.xml` fixture into dir.
func writeDTFile(t *testing.T, dir, fname, tableName, actionDSL string) {
	t.Helper()
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>` + tableName + `</table_name>
<xls_file>test.xlsx</xls_file>
<attribute_fields><Type>FIRST</Type><COMMENTS></COMMENTS><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_dsl>` + actionDSL + `</action_dsl>
    <action_postfix></action_postfix>
    <action_column column_number="1" column_value="X"/>
  </action_details>
</actions>
</decision_table>
</decision_tables>`
	if err := os.WriteFile(filepath.Join(dir, fname), []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestCallGraph_SingleEdge: a single `perform <Name>` records the
// caller → callee edge and recognizes the callee as a known table.
func TestCallGraph_SingleEdge(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA", "perform TableB;")
	writeDTFile(t, dir, "b_dt.xml", "TableB", "set client.flag = true;")

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	if !g.Tables["TableA"] || !g.Tables["TableB"] {
		t.Errorf("expected both tables in Tables set, got %v", g.Tables)
	}
	if !g.Calls["TableA"]["TableB"] {
		t.Errorf("expected TableA → TableB edge, got %v", g.Calls)
	}
	if len(g.OrphanCalls) != 0 {
		t.Errorf("expected no orphan calls, got %v", g.OrphanCalls)
	}
}

// TestCallGraph_OrphanCall: a `perform <Name>` whose target isn't
// defined anywhere lands in OrphanCalls.
func TestCallGraph_OrphanCall(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA", "perform Missing_Table;")

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	if len(g.OrphanCalls) != 1 {
		t.Fatalf("expected 1 orphan call, got %v", g.OrphanCalls)
	}
	o := g.OrphanCalls[0]
	if o.Caller != "TableA" || o.Callee != "Missing_Table" {
		t.Errorf("unexpected orphan: caller=%s callee=%s", o.Caller, o.Callee)
	}
	if o.DTFile != "a_dt.xml" {
		t.Errorf("expected DTFile=a_dt.xml, got %q", o.DTFile)
	}
}

// TestCallGraph_DedupesDuplicateCalls: multiple `perform TableB`
// references in the same caller collapse to a single edge.
func TestCallGraph_DedupesDuplicateCalls(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA",
		"perform TableB; perform TableB; perform TableB;")
	writeDTFile(t, dir, "b_dt.xml", "TableB", "")

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	if got := len(g.Calls["TableA"]); got != 1 {
		t.Errorf("expected single edge after dedup, got %d", got)
	}
}

// TestCallGraph_IgnoresPerformWhenCalled: the `perform when called`
// context-flow keyword must not be mistaken for a `perform <Table>`
// call. Pre-filter would have produced a `perform when` edge to
// nowhere if this guard regressed.
func TestCallGraph_IgnoresPerformWhenCalled(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA",
		"perform when called set client.flag = true;")

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	if len(g.Calls) != 0 {
		t.Errorf("expected no edges from `perform when called`, got %v", g.Calls)
	}
	if len(g.OrphanCalls) != 0 {
		t.Errorf("expected no orphan calls, got %v", g.OrphanCalls)
	}
}

// TestCallGraph_IgnoresPerformTableNamed: dynamic-dispatch sites
// (`perform table named (<expr>)`) are not recorded as edges. They'll
// be surfaced through a separate channel once enumeration-bounded
// dispatch lands (#776 piece B).
func TestCallGraph_IgnoresPerformTableNamed(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA",
		`perform table named ("Calculate_" + job.state + "_Tax");`)

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	// The regex won't match `perform table` because `table` is
	// lowercase (the regex requires `[A-Z]` start). So no edges.
	if len(g.Calls) != 0 {
		t.Errorf("expected no edges from dynamic dispatch, got %v", g.Calls)
	}
}

// TestCallGraph_Callers: the inverse of Calls is recovered correctly
// — sorted, deduplicated.
func TestCallGraph_Callers(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA", "perform TableC;")
	writeDTFile(t, dir, "b_dt.xml", "TableB", "perform TableC;")
	writeDTFile(t, dir, "c_dt.xml", "TableC", "")

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	got := g.Callers("TableC")
	want := []string{"TableA", "TableB"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Callers(TableC) = %v, want %v", got, want)
	}
}

// TestCallGraph_Reachable: transitive reach is computed correctly and
// cycles don't loop forever.
func TestCallGraph_Reachable(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA", "perform TableB;")
	writeDTFile(t, dir, "b_dt.xml", "TableB", "perform TableC; perform TableA;") // cycle back to A
	writeDTFile(t, dir, "c_dt.xml", "TableC", "perform TableD;")
	writeDTFile(t, dir, "d_dt.xml", "TableD", "")
	writeDTFile(t, dir, "e_dt.xml", "TableE", "") // not in the reachable set from A

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	reach := g.Reachable("TableA")
	wantReached := []string{"TableA", "TableB", "TableC", "TableD"}
	for _, t1 := range wantReached {
		if !reach[t1] {
			t.Errorf("expected %s reachable from TableA, got %v", t1, reach)
		}
	}
	if reach["TableE"] {
		t.Errorf("TableE should NOT be reachable from TableA, got %v", reach)
	}
}

// TestCallGraph_UnreachedTables: tables not reachable from the entry
// set are project-level dead code.
func TestCallGraph_UnreachedTables(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "a_dt.xml", "TableA", "perform TableB;")
	writeDTFile(t, dir, "b_dt.xml", "TableB", "")
	writeDTFile(t, dir, "c_dt.xml", "TableC", "") // orphan

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	got := g.UnreachedTables([]string{"TableA"})
	want := []string{"TableC"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnreachedTables([TableA]) = %v, want %v", got, want)
	}
}

// TestCallGraph_OrphanCalls_SortedDeterministically: the OrphanCalls
// slice should sort by (caller, callee) so callers can rely on stable
// output for diffing.
func TestCallGraph_OrphanCalls_SortedDeterministically(t *testing.T) {
	dir := t.TempDir()
	writeDTFile(t, dir, "z_dt.xml", "TableZ", "perform Missing_Z;")
	writeDTFile(t, dir, "a_dt.xml", "TableA", "perform Missing_B; perform Missing_A;")

	g, err := AnalyzeTableCallGraph(dir)
	if err != nil {
		t.Fatalf("AnalyzeTableCallGraph: %v", err)
	}
	if len(g.OrphanCalls) != 3 {
		t.Fatalf("expected 3 orphans, got %d: %v", len(g.OrphanCalls), g.OrphanCalls)
	}
	got := make([]string, 0, len(g.OrphanCalls))
	for _, o := range g.OrphanCalls {
		got = append(got, o.Caller+"→"+o.Callee)
	}
	want := []string{"TableA→Missing_A", "TableA→Missing_B", "TableZ→Missing_Z"}
	if !sort.StringsAreSorted(got) {
		t.Errorf("OrphanCalls not sorted: %v", got)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OrphanCalls order = %v, want %v", got, want)
	}
}
