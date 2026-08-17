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
	"archive/zip"
	"path/filepath"
	"strings"
	"testing"
)

// readSharedStrings returns the concatenated shared-strings XML of a
// workbook — where every DSL cell's text lives.
func readSharedStrings(t *testing.T, xlsxPath string) string {
	t.Helper()
	z, err := zip.OpenReader(xlsxPath)
	if err != nil {
		t.Fatalf("open %s: %v", xlsxPath, err)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == "xl/sharedStrings.xml" {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			var sb strings.Builder
			buf := make([]byte, 64*1024)
			for {
				n, err := r.Read(buf)
				sb.Write(buf[:n])
				if err != nil {
					break
				}
			}
			return sb.String()
		}
	}
	t.Fatalf("%s has no xl/sharedStrings.xml", xlsxPath)
	return ""
}

// TestPatchUpdatesTheWorkbookCell pins #1127. `table patch` (and put) must
// carry a DSL change into the EXISTING workbook cell in the same operation
// — Excel is the system of record, and when the update path skipped it the
// divergence surfaced only later, in CI's verify gate, blaming nobody. The
// mechanism that fixed it is #1130's artifact-based workbook pairing; this
// test is the regression guard the issue asked for: patch one cell, then
// the full verify round trip must be clean with no manual `sync export`.
func TestPatchUpdatesTheWorkbookCell(t *testing.T) {
	if testing.Short() {
		t.Skip("verify round trip rebuilds the project; skipped in the CI fast gate")
	}
	project := copyProject(t, filepath.Join("..", "..", "sampleprojects", "Scopa"))

	const oldDSL = "there is card in player.pile where card.rank == 7 and card.suit == 0"
	const newDSL = "there is card in player.pile where card.rank == 7 and card.suit == 3"
	workbook := filepath.Join(project, "excel", "Scopa.xlsx")

	if !strings.Contains(readSharedStrings(t, workbook), oldDSL) {
		t.Fatalf("fixture drift: the workbook no longer carries the expected settebello DSL")
	}

	_, stderr, exit := runTableCmd(t, project,
		[]string{"patch", "Score_Settebello"},
		`{"op":"update-condition-dsl","condition_number":1,"dsl":"`+newDSL+`"}`)
	if exit != 0 {
		t.Fatalf("table patch failed (%d): %s", exit, stderr)
	}

	shared := readSharedStrings(t, workbook)
	if !strings.Contains(shared, newDSL) {
		t.Errorf("the workbook cell was not updated by the patch — #1127 has regressed")
	}
	if strings.Contains(shared, oldDSL) {
		t.Errorf("the workbook still carries the old DSL alongside the new")
	}

	// The whole point: XML, workbook, and provenance must agree without any
	// manual `sync export` — the exact check CI's verify gate runs.
	cli := NewCLI()
	if code := cli.runVerify([]string{project}); code != 0 {
		t.Errorf("verify failed after a patch: XML and Excel diverged (exit %d)", code)
	}
}
