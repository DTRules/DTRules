package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// #1300: `map get` dropped section comments, so `map put` of get's own output
// erased every note an author had left in the mapping. Cribbage's and Scopa's
// hand-written mappings could not be brought under the authoring API without
// losing them.
func TestMapGetPutKeepsSectionComments(t *testing.T) {
	src := `<?xml version="1.0" encoding="UTF-8"?>
<mapping>
	<XMLtoEDD>
		<map>
			<!-- The hand is the root singleton. -->
			<setattribute tag='is_crib' RAttribute='is_crib' enclosure='hand' type='boolean'></setattribute>
			<!-- One card stream. -->
			<setattribute tag='rank' RAttribute='rank' enclosure='card' type='integer'></setattribute>
		</map>
		<entities>
			<entity name='hand' number='1'></entity>
			<entity name='card' number='*'></entity>
		</entities>
		<initialization>
			<initialentity entity='hand' epush='true'></initialentity>
		</initialization>
	</XMLtoEDD>
</mapping>
`
	dir := t.TempDir()
	in := filepath.Join(dir, "in_map.xml")
	if err := os.WriteFile(in, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := excel.LoadMapXMLFromFile(in)
	if err != nil {
		t.Fatal(err)
	}

	// Through the wire format, as `map get | map put` does.
	raw, err := json.Marshal(mapToJSON(m))
	if err != nil {
		t.Fatal(err)
	}
	var doc mapJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	back, err := mapFromJSON(doc)
	if err != nil {
		t.Fatalf("put refused get's own output: %v", err)
	}

	var comments []string
	for _, e := range back.Entries {
		if e.IsSection {
			comments = append(comments, e.Comment)
		}
	}
	want := []string{"The hand is the root singleton.", "One card stream."}
	if len(comments) != len(want) || comments[0] != want[0] || comments[1] != want[1] {
		t.Fatalf("comments after get|put = %q, want %q", comments, want)
	}
	if len(back.Entries) != 4 || back.Entries[1].Tag != "is_crib" || back.Entries[3].Tag != "rank" {
		t.Fatalf("comments moved relative to the attributes they describe: %+v", back.Entries)
	}

	out := filepath.Join(dir, "out_map.xml")
	if err := excel.WriteMapXML(back, out); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.ReadFile(in)
	got, _ := os.ReadFile(out)
	if string(orig) != string(got) {
		t.Errorf("get|put did not reproduce the file:\n--- want\n%s\n--- got\n%s", orig, got)
	}
}

func TestMapPutRefusesACommentThatIsAlsoAnAttribute(t *testing.T) {
	_, err := mapFromJSON(mapJSON{Attributes: []mapAttributeJSON{
		{Comment: "note", Tag: "rank", Enclosure: "card"},
	}})
	if err == nil {
		t.Fatal("an entry carrying both a comment and an attribute was accepted")
	}
}
