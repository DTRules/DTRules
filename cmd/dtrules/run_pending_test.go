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
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/collect"
)

// The test vectors of #1210 run against sampleprojects/SinusitisTherapy,
// entry Determine_Therapy, whose patient entity has five collect fields
// (diagnosis, age, lean_body_weight, pcr, penicillin_allergic).
const sinusitisProject = "../../sampleprojects/SinusitisTherapy"

// sinusitisData is a canonical (mapping-free) data file answering every
// collect field. Tests drop one field at a time from it.
const sinusitisData = `<?xml version='1.0' encoding='UTF-8'?>
<dtrules-data>
  <patient>
    <age>40</age>
    <lean_body_weight>80</lean_body_weight>
    <pcr>0.9</pcr>
    <penicillin_allergic>false</penicillin_allergic>
    <diagnosis>Acute Sinusitis</diagnosis>
  </patient>
</dtrules-data>
`

// writeData writes sinusitisData with the named fields removed.
func writeData(t *testing.T, name string, drop ...string) string {
	t.Helper()
	var kept []string
	for _, line := range strings.Split(sinusitisData, "\n") {
		skip := false
		for _, d := range drop {
			if strings.Contains(line, "<"+d+">") {
				skip = true
			}
		}
		if !skip {
			kept = append(kept, line)
		}
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// runCapture runs `dtrules run` with args, returning the exit code and stdout.
func runCapture(t *testing.T, args ...string) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	code := NewCLI().runRun(args)

	os.Stdout = saved
	w.Close()
	out := <-done
	r.Close()
	return code, out
}

// readPending parses the JSON array --pending wrote.
func readPending(t *testing.T, path string) []collect.Pending {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out []collect.Pending
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("parse %s: %v\n%s", path, err, b)
	}
	if strings.TrimSpace(string(b)) == "null" {
		t.Errorf("%s holds null; an empty recording must be []", path)
	}
	return out
}

// resultLines returns the run's printed result block as a sorted set of
// lines, so two runs can be compared without depending on field order.
func resultLines(out string) []string {
	i := strings.Index(out, "=== result ===")
	if i < 0 {
		return nil
	}
	var lines []string
	for _, l := range strings.Split(out[i:], "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	sort.Strings(lines)
	return lines
}

// Vector 1: every collect field supplied. Nothing is pending, so the run is
// not provisional: exit 0, an empty array, and the same result a run without
// --pending prints.
func TestRunPending_NothingPending(t *testing.T) {
	data := writeData(t, "full.xml")
	out := filepath.Join(t.TempDir(), "pending.json")

	code, stdout := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
		"--data", data, "--pending", out)
	if code != 0 {
		t.Fatalf("exit %d, want 0\n%s", code, stdout)
	}
	if got := readPending(t, out); len(got) != 0 {
		t.Errorf("want no pending questions, got %+v", got)
	}
	if strings.Contains(stdout, "PROVISIONAL") {
		t.Errorf("a complete run must not be marked provisional:\n%s", stdout)
	}

	baseCode, baseOut := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy", "--data", data)
	if baseCode != 0 {
		t.Fatalf("baseline exit %d", baseCode)
	}
	want, got := resultLines(baseOut), resultLines(stdout)
	if len(want) == 0 || strings.Join(want, "\n") != strings.Join(got, "\n") {
		t.Errorf("result differs from a run without --pending:\nwithout:\n%s\nwith:\n%s",
			strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
}

// Vector 2: one collect field removed. The run completes on the substituted
// default, records exactly that question, exits 3, and marks the result
// provisional.
//
// The issue names patient.pcr here. pcr's default is 0.0 and SinusitisTherapy
// divides by it (Cockcroft-Gault), so a run without it aborts — see
// TestRunPending_RecordsBeforeAnExecutionError, which asserts the pcr
// question's published content. penicillin_allergic is the same vector on a
// field whose default the rules survive.
func TestRunPending_OneQuestionIsProvisional(t *testing.T) {
	data := writeData(t, "noallergy.xml", "penicillin_allergic")
	out := filepath.Join(t.TempDir(), "pending.json")

	code, stdout := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
		"--data", data, "--pending", out)
	if code != exitPending {
		t.Fatalf("exit %d, want %d (provisional)\n%s", code, exitPending, stdout)
	}
	got := readPending(t, out)
	if len(got) != 1 {
		t.Fatalf("want exactly 1 pending question, got %d: %+v", len(got), got)
	}
	q := got[0]
	if q.Entity != "patient" || q.Field != "penicillin_allergic" {
		t.Errorf("identity: got %s.%s", q.Entity, q.Field)
	}
	if q.QuestionType != "multiple_choice" || len(q.Options) != 2 {
		t.Errorf("question: type %q, %d options", q.QuestionType, len(q.Options))
	}
	if q.Default != "false" {
		t.Errorf("substituted default: got %q, want %q", q.Default, "false")
	}
	if q.Instance == 0 {
		t.Errorf("instance identity missing: %+v", q)
	}
	if !strings.Contains(stdout, "PROVISIONAL") {
		t.Errorf("result was not marked provisional:\n%s", stdout)
	}
	// The run still finished and printed a result — provisional, not absent.
	if resultLines(stdout) == nil {
		t.Errorf("provisional run printed no result:\n%s", stdout)
	}
}

// Vector 3: answer the pending question, re-run, and the result stands —
// exit 0, nothing pending, the same result as vector 1.
func TestRunPending_AnsweredRunStands(t *testing.T) {
	partial := writeData(t, "noallergy.xml", "penicillin_allergic")
	full := writeData(t, "full.xml")
	dir := t.TempDir()

	code, _ := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
		"--data", partial, "--pending", filepath.Join(dir, "first.json"))
	if code != exitPending {
		t.Fatalf("first run exit %d, want %d", code, exitPending)
	}

	out := filepath.Join(dir, "second.json")
	code, stdout := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
		"--data", full, "--pending", out)
	if code != 0 {
		t.Fatalf("answered run exit %d, want 0\n%s", code, stdout)
	}
	if got := readPending(t, out); len(got) != 0 {
		t.Errorf("answered run still pending: %+v", got)
	}

	_, wantOut := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
		"--data", full, "--pending", filepath.Join(dir, "third.json"))
	if strings.Join(resultLines(wantOut), "\n") != strings.Join(resultLines(stdout), "\n") {
		t.Errorf("answered result differs from the fully-supplied run")
	}
}

// Vector 4: with no data at all, every collect field the run reaches is
// recorded once each, in the order reached — never twice, however often the
// rules read it.
//
// SinusitisTherapy cannot finish this run: with no data, pcr defaults to 0.0
// and Determine_Creatinine_Clearance divides by it. The exit code is
// therefore 1 (the run failed), not 3 (the run is provisional) — but the
// questions reached before the failure are still published, which is the
// whole point of recording them.
func TestRunPending_RecordsBeforeAnExecutionError(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pending.json")

	code, _ := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy", "--pending", out)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (the sample divides by pcr's zero default)", code)
	}
	got := readPending(t, out)

	var order []string
	seen := map[string]bool{}
	for _, q := range got {
		if seen[q.Field] {
			t.Errorf("field %s recorded twice", q.Field)
		}
		seen[q.Field] = true
		order = append(order, q.Field)
	}
	want := "diagnosis,age,lean_body_weight,pcr"
	if strings.Join(order, ",") != want {
		t.Errorf("order reached: got %q, want %q", strings.Join(order, ","), want)
	}
	// The pcr question the issue's vector 2 names, published in full.
	var pcr *collect.Pending
	for i := range got {
		if got[i].Field == "pcr" {
			pcr = &got[i]
		}
	}
	if pcr == nil {
		t.Fatalf("patient.pcr not recorded: %+v", got)
	}
	if pcr.QuestionType != "number" || pcr.RefLow != "0.7" || pcr.RefHigh != "1.3" || pcr.Units != "mg/dL" {
		t.Errorf("pcr question: %+v", *pcr)
	}
	if pcr.Default != "0" {
		t.Errorf("pcr substituted default: got %q, want %q", pcr.Default, "0")
	}
}

// Vector 7: --pending is the non-blocking opposite of an interview. Pairing
// it with a front end that asks is a usage error, not a preference.
func TestRunPending_MutuallyExclusiveWithAskingFrontEnds(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pending.json")
	for _, flag := range []string{"--interactive", "-i", "--web"} {
		code, _ := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
			"--pending", out, flag)
		if code != 1 {
			t.Errorf("--pending %s: exit %d, want 1 (usage error)", flag, code)
		}
		if _, err := os.Stat(out); err == nil {
			t.Errorf("--pending %s: refused runs must not write %s", flag, out)
		}
	}
}

// Vector 8: --pending never reads stdin, so it completes with stdin closed —
// the daemon case. Without it, the same run under --interactive would park on
// a read.
func TestRunPending_NeverReadsStdin(t *testing.T) {
	data := writeData(t, "noallergy.xml", "penicillin_allergic")
	out := filepath.Join(t.TempDir(), "pending.json")

	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	saved := os.Stdin
	os.Stdin = devnull
	t.Cleanup(func() { os.Stdin = saved; devnull.Close() })

	code, _ := runCapture(t, sinusitisProject, "--entry", "Determine_Therapy",
		"--data", data, "--pending", out)
	if code != exitPending {
		t.Fatalf("exit %d, want %d", code, exitPending)
	}
	if got := readPending(t, out); len(got) != 1 {
		t.Errorf("want 1 pending question, got %+v", got)
	}
}
