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

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRunSave_ByteIdenticalAcrossRuns is #1231: the same rules on the same
// input must save the same bytes, so a host can tell "this run changed the
// state" with a plain byte comparison instead of an order-insensitive one.
func TestRunSave_ByteIdenticalAcrossRuns(t *testing.T) {
	data := writeData(t, "in.xml")
	dir := t.TempDir()

	var first []byte
	for i := 0; i < 8; i++ {
		out := filepath.Join(dir, "out.xml")
		if code, _ := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy", "--data", data, "--save", out); code != 0 {
			t.Fatalf("run %d: exit %d", i, code)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if i == 0 {
			first = b
			continue
		}
		if !bytes.Equal(first, b) {
			t.Fatalf("run %d saved different bytes than run 0\n--- run 0 ---\n%s\n--- run %d ---\n%s", i, first, i, b)
		}
	}
}

// TestRunSave_EDDDeclarationOrder pins which deterministic order --save
// uses: entities and their fields appear in the order the EDD declares them.
func TestRunSave_EDDDeclarationOrder(t *testing.T) {
	edd, err := os.ReadFile(filepath.Join(sinusitisProject, "xml", "sinusitis_edd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	entityRe := regexp.MustCompile(`<entity name="([^"]+)"`)
	fieldRe := regexp.MustCompile(`<field name="([^"]+)"`)
	var eddEntities []string
	eddFields := map[string][]string{}
	for _, block := range strings.Split(string(edd), "<entity ")[1:] {
		m := entityRe.FindStringSubmatch("<entity " + block)
		eddEntities = append(eddEntities, m[1])
		for _, f := range fieldRe.FindAllStringSubmatch(block, -1) {
			eddFields[m[1]] = append(eddFields[m[1]], f[1])
		}
	}

	out := filepath.Join(t.TempDir(), "out.xml")
	if code, _ := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy", "--data", writeData(t, "in.xml"), "--save", out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	// Top-level elements (two-space indent) and the fields under each.
	var gotEntities []string
	gotFields := map[string][]string{}
	var cur string
	tagRe := regexp.MustCompile(`^( *)<([A-Za-z_][A-Za-z0-9_]*)>`)
	for _, line := range strings.Split(string(saved), "\n") {
		m := tagRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch len(m[1]) {
		case 2:
			cur = m[2]
			gotEntities = append(gotEntities, cur)
		case 4:
			gotFields[cur] = append(gotFields[cur], m[2])
		}
	}

	assertSubsequence(t, "entities", gotEntities, eddEntities)
	for _, e := range gotEntities {
		assertSubsequence(t, "fields of "+e, gotFields[e], eddFields[e])
	}
}

// assertSubsequence fails unless got lists a subset of want, in want's order.
func assertSubsequence(t *testing.T, what string, got, want []string) {
	t.Helper()
	j := 0
	for _, g := range got {
		for j < len(want) && want[j] != g {
			j++
		}
		if j == len(want) {
			t.Errorf("%s: saved order %v is not in EDD order %v", what, got, want)
			return
		}
		j++
	}
}
