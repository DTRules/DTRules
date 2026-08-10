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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The debug endpoints were exercised only by session driver scripts, so
// nothing in the repo pinned their HTTP contract: a handler could change its
// method, its required parameters, or the shape of what it returns, and the
// build stayed green (#931).
//
// These tests drive the real router over a real trace. They assert the
// contract a UI depends on -- status codes, required parameters, and the keys
// each payload must carry -- and deliberately not the values the KidAid trace
// happens to hold, which would make the suite a tripwire for every rule edit.

const (
	// kidaidTrace records no finalState; syntheticTrace does. Report
	// generation needs one, so both are here rather than one.
	//
	// They live in testdata/ rather than beside the sample that produced
	// them: .gitignore excludes `output/`, so the copies under
	// sampleprojects/KidAid/testfiles/output/ are never committed. These
	// tests passed on the machine that wrote them and failed on every fresh
	// checkout, which is the whole failure mode `dtrules verify` exists to
	// prevent for rules and which test fixtures are just as prone to.
	kidaidTrace    = "testdata/kidaid.trace.xml"
	syntheticTrace = "testdata/big-synthetic.trace.xml"
)

// debugServer returns a router with the KidAid trace already loaded.
func debugServer(t *testing.T, readOnly bool) http.Handler {
	t.Helper()
	return debugServerWith(t, kidaidTrace, readOnly)
}

// debugServerWith assembles a throwaway project from KidAid's committed rules
// and the named trace fixture, and returns a router over it.
//
// It builds the project rather than pointing at sampleprojects/KidAid because
// trace paths are validated against the project root: a fixture in testdata/
// is outside any sample, and that refusal is correct. Copying both into one
// temp directory exercises the real validation instead of disabling it.
func debugServerWith(t *testing.T, tracePath string, readOnly bool) http.Handler {
	t.Helper()
	root := t.TempDir()

	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcXML := filepath.Join("..", "..", "..", "sampleprojects", "KidAid", "xml")
	entries, err := os.ReadDir(srcXML)
	if err != nil {
		t.Skipf("KidAid rules not present: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(srcXML, e.Name()))
		if rerr != nil {
			t.Fatal(rerr)
		}
		if werr := os.WriteFile(filepath.Join(xmlDir, e.Name()), data, 0o644); werr != nil {
			t.Fatal(werr)
		}
	}

	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace fixture %s: %v", tracePath, err)
	}
	tracePathInProject := filepath.Join(root, filepath.Base(tracePath))
	if err := os.WriteFile(tracePathInProject, trace, 0o644); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ProjectRoot: root, ReadOnly: readOnly})
	// A trace is read in the context of the rules it came from, so the
	// project has to be open first -- the same order dtrules debug uses.
	if err := s.LoadProject(xmlDir); err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if err := s.LoadDebugTrace(tracePathInProject); err != nil {
		t.Fatalf("LoadDebugTrace(%s): %v", tracePathInProject, err)
	}
	return s.Routes()
}

// do issues a request and returns the status and decoded body. A body that is
// not JSON is a failure in itself -- every one of these endpoints promises
// JSON, including on error.
func do(t *testing.T, h http.Handler, method, target, body string) (int, map[string]any) {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	raw, _ := io.ReadAll(w.Body)
	var decoded map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("%s %s returned %d with non-JSON body: %s",
				method, target, w.Code, truncate(string(raw)))
		}
	}
	return w.Code, decoded
}

func truncate(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

func requireKeys(t *testing.T, what string, got map[string]any, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := got[k]; !ok {
			present := make([]string, 0, len(got))
			for k := range got {
				present = append(present, k)
			}
			t.Errorf("%s: response has no %q (keys present: %v)", what, k, present)
		}
	}
}

func TestDebugStatusReportsTheLoadedTrace(t *testing.T) {
	h := debugServer(t, false)

	code, body := do(t, h, "GET", "/api/debug/status", "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", code, body)
	}
	requireKeys(t, "status", body, "loaded", "nodes")
	if loaded, _ := body["loaded"].(bool); !loaded {
		t.Errorf("loaded = false after LoadDebugTrace; a UI mounting after a "+
			"server-side preload would show an empty debugger (body %v)", body)
	}
	if n, _ := body["nodes"].(float64); n <= 0 {
		t.Errorf("nodes = %v, want a positive count", body["nodes"])
	}
}

func TestDebugTreeAndPosition(t *testing.T) {
	h := debugServer(t, false)

	code, tree := do(t, h, "GET", "/api/debug/tree", "")
	if code != http.StatusOK {
		t.Fatalf("tree = %d, want 200 (body %v)", code, tree)
	}

	// Node 1 exists in any non-empty trace; the point is the shape of the
	// reply, not where it lands.
	code, pos := do(t, h, "POST", "/api/debug/position", `{"node": 1}`)
	if code != http.StatusOK {
		t.Fatalf("position = %d, want 200 (body %v)", code, pos)
	}
	if len(pos) == 0 {
		t.Error("position returned an empty payload; the debugger has nothing to render")
	}
}

// find is the why-chain join. Without an attr it must say so rather than
// returning an empty result that reads like "nothing caused this".
func TestDebugFindRequiresAttr(t *testing.T) {
	h := debugServer(t, false)

	code, body := do(t, h, "GET", "/api/debug/find", "")
	if code != http.StatusBadRequest {
		t.Fatalf("find with no attr = %d, want 400 (body %v)", code, body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, "attr") {
		t.Errorf("the error should name the missing parameter, got %q", msg)
	}
}

func TestDebugFindAcceptsAnAttr(t *testing.T) {
	h := debugServer(t, false)

	// Whether this attribute was written in the trace is not the contract --
	// answering without a 500 is.
	code, body := do(t, h, "GET", "/api/debug/find?attr=eligible", "")
	if code != http.StatusOK {
		t.Fatalf("find = %d, want 200 (body %v)", code, body)
	}
}

func TestDebugEntityRequiresID(t *testing.T) {
	h := debugServer(t, false)

	code, body := do(t, h, "GET", "/api/debug/entity", "")
	if code != http.StatusBadRequest {
		t.Fatalf("entity with no id = %d, want 400 (body %v)", code, body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, "id") {
		t.Errorf("the error should name the missing parameter, got %q", msg)
	}
}

func TestDebugArrayRequiresID(t *testing.T) {
	h := debugServer(t, false)

	if code, body := do(t, h, "GET", "/api/debug/array", ""); code != http.StatusBadRequest {
		t.Fatalf("array with no id = %d, want 400 (body %v)", code, body)
	}
}

func TestDebugReportGenerates(t *testing.T) {
	h := debugServerWith(t, syntheticTrace, false)

	code, body := do(t, h, "POST", "/api/debug/report", `{}`)
	if code != http.StatusOK {
		t.Fatalf("report = %d, want 200 (body %v)", code, body)
	}
	if len(body) == 0 {
		t.Error("report returned an empty payload")
	}
}

// A report is built from the trace's finalState. Asked for one over a trace
// that has none, the endpoint has to say which thing is missing -- an empty
// report would read as "the run produced nothing".
func TestDebugReportSaysWhyItCannotBuildOne(t *testing.T) {
	h := debugServerWith(t, kidaidTrace, false)

	code, body := do(t, h, "POST", "/api/debug/report", `{}`)
	if code == http.StatusOK {
		t.Fatalf("report succeeded over a trace with no finalState (body %v)", body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, "finalState") {
		t.Errorf("the error should name what is missing, got %q", msg)
	}
}

// Every one of these rejects the wrong verb rather than falling through to a
// nil-pointer path.
func TestDebugEndpointsRejectTheWrongMethod(t *testing.T) {
	h := debugServer(t, false)

	cases := []struct{ method, target string }{
		{"POST", "/api/debug/status"},
		{"POST", "/api/debug/tree"},
		{"POST", "/api/debug/find?attr=eligible"},
		{"POST", "/api/debug/entity?id=1"},
		{"POST", "/api/debug/array?id=1"},
		{"GET", "/api/debug/position"},
		{"GET", "/api/debug/console"},
		{"GET", "/api/debug/report"},
		{"GET", "/api/debug/speculate"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.target, func(t *testing.T) {
			code, body := do(t, h, c.method, c.target, "")
			if code != http.StatusMethodNotAllowed {
				t.Errorf("= %d, want 405 (body %v)", code, body)
			}
		})
	}
}

// A server with no trace must answer "no trace" on every read, not panic and
// not report success over a nil session.
func TestDebugEndpointsWithNoTraceLoaded(t *testing.T) {
	root, err := filepath.Abs("../../../sampleprojects/KidAid")
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{ProjectRoot: root}).Routes()

	code, body := do(t, h, "GET", "/api/debug/status", "")
	if code != http.StatusOK {
		t.Fatalf("status with no trace = %d, want 200 saying loaded:false", code)
	}
	if loaded, _ := body["loaded"].(bool); loaded {
		t.Errorf("loaded = true with no trace (body %v)", body)
	}

	for _, target := range []string{
		"/api/debug/find?attr=eligible",
		"/api/debug/entity?id=1",
		"/api/debug/array?id=1",
	} {
		if code, body := do(t, h, "GET", target, ""); code != http.StatusBadRequest {
			t.Errorf("%s with no trace = %d, want 400 (body %v)", target, code, body)
		}
	}
}

// Read-only mode exists to publish rules for review. Execution and reads
// persist nothing and must stay available, or the published debugger is inert.
func TestDebugReadsStayAvailableInReadOnlyMode(t *testing.T) {
	h := debugServer(t, true)

	for _, target := range []string{"/api/debug/status", "/api/debug/tree"} {
		if code, body := do(t, h, "GET", target, ""); code != http.StatusOK {
			t.Errorf("%s in read-only mode = %d, want 200 (body %v)", target, code, body)
		}
	}
}

// Trace paths are validated against the project root; a path outside it must
// be refused rather than read.
func TestDebugLoadRefusesAPathOutsideTheProject(t *testing.T) {
	h := debugServer(t, false)

	code, body := do(t, h, "POST", "/api/debug/load", `{"path": "/etc/passwd"}`)
	if code == http.StatusOK {
		t.Fatalf("load accepted a path outside the project root (body %v)", body)
	}
}
