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

package excel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateEntityOmitsEmptyAttributes pins #1036.
//
// The writer emitted `id=”` unconditionally. An empty attribute is not what an
// author writes and not what the reader needs, and it meant a hand-authored map
// could never round-trip to itself: the first build rewrote every singleton
// createentity, so `verify` reported a difference with nothing behind it. On
// the Accumulate staking map that was 6 of 11 elements rewritten on a build
// that changed nothing.
//
// `list` was already conditional; `id` was not.
func TestCreateEntityOmitsEmptyAttributes(t *testing.T) {
	m := &MapXML{
		CreateEntities: []MapCreateEntity{
			{Entity: "budget_params", Tag: "budget_params"},                                      // neither
			{Entity: "acme_issuance", Tag: "acme_issuance", ID: "major_block", List: "issuance"}, // both
			{Entity: "period", Tag: "period", ID: "seq"},                                         // id only
			{Entity: "accounts", Tag: "accounts", List: "accounts"},                              // list only
		},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "T_map.xml")
	if err := WriteMapXML(m, path); err != nil {
		t.Fatalf("WriteMapXML: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)

	for _, unwanted := range []string{"id=''", `id=""`, "list=''", `list=""`} {
		if strings.Contains(out, unwanted) {
			t.Errorf("writer emitted an empty attribute %s:\n%s", unwanted, out)
		}
	}
	for _, want := range []string{
		"<createentity entity='budget_params' tag='budget_params'>",
		"id='major_block'",
		"list='issuance'",
		"<createentity entity='period' tag='period' id='seq'>",
		"<createentity entity='accounts' tag='accounts' list='accounts'>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("writer output missing %q:\n%s", want, out)
		}
	}
}

// TestMapWriterIsAFixedPoint is the property that matters: whatever the writer
// emits, re-reading and re-writing it must produce the same bytes. Without
// that, `dtrules verify` can never go green on a map, because every build
// reports a difference the author cannot act on.
func TestMapWriterIsAFixedPoint(t *testing.T) {
	m := &MapXML{
		CreateEntities: []MapCreateEntity{
			{Entity: "budget_params", Tag: "budget_params"},
			{Entity: "staking_account", Tag: "staking_account", ID: "account_url", List: "accounts"},
		},
		EntityDecls:     []MapEntityDecl{{Name: "budget_params", Number: "1"}},
		InitialEntities: []MapInitialEntity{{Entity: "budget_params", EPush: true}},
	}

	dir := t.TempDir()
	p1 := filepath.Join(dir, "A_map.xml")
	if err := WriteMapXML(m, p1); err != nil {
		t.Fatalf("WriteMapXML: %v", err)
	}
	firstB, err := os.ReadFile(p1)
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := LoadMapXMLFromFile(p1)
	if err != nil {
		t.Fatalf("the writer's own output does not parse: %v\n%s", err, firstB)
	}
	p2 := filepath.Join(dir, "B_map.xml")
	if err := WriteMapXML(reparsed, p2); err != nil {
		t.Fatalf("WriteMapXML (second): %v", err)
	}
	secondB, err := os.ReadFile(p2)
	if err != nil {
		t.Fatal(err)
	}
	first, second := string(firstB), string(secondB)

	if first != second {
		t.Errorf("write → read → write is not stable:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}
