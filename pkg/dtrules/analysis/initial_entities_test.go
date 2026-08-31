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

package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// The entities a mapping pushes before any table runs are what a bare name in
// a table with no context of its own resolves against. Reading them is the
// whole basis of that resolution, so when it silently returns nothing the
// resolution is not wrong -- it simply never happens, and every such field is
// reported unused.
//
// That is what it did. The decode was anchored at the document element
// (`initialization>initialentity`) while every mapping in the corpus nests the
// block under <XMLtoEDD>, so it matched nothing, in every project, since it
// shipped. The example its own comment cited as fixed -- CHIP's
// job.currentdate, read on every run -- was still being reported (#776).
func TestInitialEntitiesAreFoundWhereverTheyAreNested(t *testing.T) {
	dir := t.TempDir()

	// The shape every mapping in the corpus actually has: the block is a
	// grandchild of the document element, not a child.
	nested := `<?xml version="1.0" encoding="UTF-8"?>
<mapping>
	<XMLtoEDD>
		<map></map>
		<initialization>
			<initialentity entity='result' epush='true'></initialentity>
			<initialentity entity='job' epush='true'></initialentity>
			<initialentity entity='ignored' epush='false'></initialentity>
		</initialization>
	</XMLtoEDD>
</mapping>`
	if err := os.WriteFile(filepath.Join(dir, "X_map.xml"), []byte(nested), 0o644); err != nil {
		t.Fatal(err)
	}

	got := initialEntities(dir)

	want := map[string]bool{"job": true, "result": true}
	if len(got) != len(want) {
		t.Fatalf("initialEntities = %v, want job and result — a stack read as empty means "+
			"every bare name in a context-free table is reported unused", got)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected entity %q", g)
		}
	}
}

// epush='false' declares the entity without putting it on the stack, so a bare
// name cannot resolve to it and the analyzer must not pretend otherwise.
func TestInitialEntitiesRespectEPush(t *testing.T) {
	dir := t.TempDir()
	doc := `<mapping><XMLtoEDD><initialization>
		<initialentity entity='pushed' epush='true'></initialentity>
		<initialentity entity='declared_only' epush='false'></initialentity>
		<initialentity entity='no_attr'></initialentity>
	</initialization></XMLtoEDD></mapping>`
	if err := os.WriteFile(filepath.Join(dir, "Y_map.xml"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	got := initialEntities(dir)
	if len(got) != 1 || got[0] != "pushed" {
		t.Errorf("initialEntities = %v, want only the epush='true' entity", got)
	}
}

// A flat mapping, were one ever written, must work too — depth carries no
// meaning here, and pinning the path is what broke it.
func TestInitialEntitiesToleratesAFlatMapping(t *testing.T) {
	dir := t.TempDir()
	doc := `<mapping><initialization>
		<initialentity entity='job' epush='true'></initialentity>
	</initialization></mapping>`
	if err := os.WriteFile(filepath.Join(dir, "Z_map.xml"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	got := initialEntities(dir)
	if len(got) != 1 || got[0] != "job" {
		t.Errorf("initialEntities = %v, want [job]", got)
	}
}

// The end-to-end shape of the bug: a field read only as a bare name, in a
// table with no context of its own, against an entity the mapping pushed.
func TestBareNameResolvesAgainstThePushedStack(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("P_map.xml", `<mapping><XMLtoEDD><initialization>
		<initialentity entity='job' epush='true'></initialentity>
	</initialization></XMLtoEDD></mapping>`)
	write("P_edd.xml", `<entity_data_dictionary>
		<entity name="job">
			<field name="threshold" type="double"></field>
			<field name="never_read" type="double"></field>
		</entity>
	</entity_data_dictionary>`)
	write("P_dt.xml", `<decision_tables><decision_table>
		<table_name>T</table_name>
		<conditions><condition_details>
			<condition_dsl>threshold &gt; 0</condition_dsl>
		</condition_details></conditions>
	</decision_table></decision_tables>`)

	warns, err := AnalyzeEDDUsage(dir)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	reported := map[string]bool{}
	for _, w := range warns {
		reported[w.Field] = true
	}
	if reported["job.threshold"] {
		t.Error("job.threshold is read as a bare name against the pushed job entity, " +
			"but was reported unused")
	}
	if !reported["job.never_read"] {
		t.Error("job.never_read is read nowhere and should still be reported — " +
			"the fix must not silence real findings")
	}
}
