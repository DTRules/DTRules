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

package apiserver

import (
	"path/filepath"
	"strings"
	"testing"
)

// Reports have always diffed against a baseline — but the only way to get one
// was to start a speculation, which asks a different question. "What changed
// between period N and period N-1" compares two runs that both really
// happened, and needed no speculation at all (#930).

const aReportSpec = `{"title":"t","sections":[{"title":"clients","entity":"client",` +
	`"fields":["age"]}]}`

func TestBaselineFromASecondTraceMakesReportsDiff(t *testing.T) {
	// The synthetic fixture is the one carrying a finalState, and a report
	// replays to it — the KidAid trace records none, so reports cannot run on
	// it at all.
	h, root := debugServerAndRoot(t, syntheticTrace, true)

	// Without a baseline, a report is just a report.
	status, body := do(t, h, "POST", "/api/debug/report", aReportSpec)
	if status != 200 {
		t.Fatalf("report: %d %v", status, body)
	}
	if _, ok := body["diff"]; ok {
		t.Fatal("a report with no baseline should carry no diff")
	}

	status, body = do(t, h, "POST", "/api/debug/baseline",
		`{"path":`+quote(filepath.Join(root, baselineTraceName))+`}`)
	if status != 200 {
		t.Fatalf("setting a baseline: %d %v", status, body)
	}
	bl, _ := body["baseline"].(map[string]any)
	if bl == nil {
		t.Fatal("no baseline in the response")
	}
	if bl["speculative"] != false {
		t.Errorf("speculative = %v, want false — this is a second real trace", bl["speculative"])
	}

	status, body = do(t, h, "POST", "/api/debug/report", aReportSpec)
	if status != 200 {
		t.Fatalf("report with baseline: %d %v", status, body)
	}
	if _, ok := body["diff"]; !ok {
		t.Error("a report with a baseline set should carry a diff")
	}
	if _, ok := body["baseline"]; !ok {
		t.Error("the baseline report itself should come back, so a client can show both sides")
	}
}

func TestBaselineCanBeCleared(t *testing.T) {
	h, root := debugServerAndRoot(t, syntheticTrace, true)

	if st, body := do(t, h, "POST", "/api/debug/baseline",
		`{"path":`+quote(filepath.Join(root, baselineTraceName))+`}`); st != 200 {
		t.Fatalf("set: %d %v", st, body)
	}
	st, body := do(t, h, "POST", "/api/debug/baseline", `{"path":""}`)
	if st != 200 {
		t.Fatalf("clear: %d %v", st, body)
	}
	if body["baseline"] != nil {
		t.Errorf("baseline = %v, want nil after clearing", body["baseline"])
	}
	_, rep := do(t, h, "POST", "/api/debug/report", aReportSpec)
	if _, ok := rep["diff"]; ok {
		t.Error("a cleared baseline should stop reports diffing")
	}
}

// A run does not differ from itself, and offering the comparison invites
// reading an empty diff as "nothing changed between the two periods".
func TestBaselineRefusesTheTraceAlreadyLoaded(t *testing.T) {
	h, root := debugServerAndRoot(t, syntheticTrace, true)

	_, st := do(t, h, "GET", "/api/debug/status", "")
	path, _ := st["tracePath"].(string)
	if path == "" {
		t.Skip("status carries no tracePath")
	}
	status, body := do(t, h, "POST", "/api/debug/baseline",
		`{"path":`+quote(filepath.Join(root, baseName(path)))+`}`)
	if status == 200 {
		t.Fatalf("a trace was accepted as its own baseline: %v", body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, "does not differ from itself") {
		t.Errorf("refused, but not for the right reason: %v", body)
	}
}

// The session payload has to say what it is comparing against, or a client
// showing a diff cannot label which side is which.
func TestStatusReportsTheBaseline(t *testing.T) {
	h, root := debugServerAndRoot(t, syntheticTrace, true)

	_, before := do(t, h, "GET", "/api/debug/status", "")
	if before["baseline"] != nil {
		t.Errorf("baseline = %v before one is set, want nil", before["baseline"])
	}

	do(t, h, "POST", "/api/debug/baseline", `{"path":`+quote(filepath.Join(root, baselineTraceName))+`}`)

	_, after := do(t, h, "GET", "/api/debug/status", "")
	bl, _ := after["baseline"].(map[string]any)
	if bl == nil {
		t.Fatal("status does not report the baseline it is comparing against")
	}
	if bl["speculative"] != false {
		t.Errorf("speculative = %v, want false", bl["speculative"])
	}
}

// "Reset the speculation" must not fire on a comparison baseline, or it swaps
// the active session over to a trace the user only wanted to compare against.
func TestSpeculateResetIgnoresAComparisonBaseline(t *testing.T) {
	h, root := debugServerAndRoot(t, syntheticTrace, true)

	do(t, h, "POST", "/api/debug/baseline", `{"path":`+quote(filepath.Join(root, baselineTraceName))+`}`)

	status, body := do(t, h, "POST", "/api/debug/speculate/reset", `{}`)
	if status == 200 {
		t.Fatalf("reset acted on a comparison baseline and would have swapped the session: %v", body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(strings.ToLower(msg), "no speculation") {
		t.Errorf("refused, but not as 'no speculation active': %v", body)
	}
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}
