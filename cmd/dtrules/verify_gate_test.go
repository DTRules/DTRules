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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testProjectDir is the one sample that is genuinely contract-clean, so it is
// the right fixture for asserting that a clean project passes. It is also
// small: verify runs on it in under a tenth of a second, against two minutes
// for TaxReturn.
var testProjectDir = filepath.Join("..", "..", "sampleprojects", "TestProject")

func skipIfNoTestProject(t *testing.T) {
	t.Helper()
	for _, sub := range []string{"xml", "excel"} {
		if _, err := os.Stat(filepath.Join(testProjectDir, sub)); err != nil {
			t.Skipf("TestProject %s/ not found: %v", sub, err)
		}
	}
}

// TestVerifyGatePassesCleanProject is the positive half of the gate.
func TestVerifyGatePassesCleanProject(t *testing.T) {
	skipIfNoTestProject(t)

	tmp := t.TempDir()
	if err := copyDir(testProjectDir, tmp); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	proj := filepath.Join(tmp, "TestProject")

	stderr := captureStderr(t, func() {
		if code := NewCLI().runVerify([]string{proj}); code != 0 {
			t.Errorf("clean project failed verify (exit %d)", code)
		}
	})
	if stderr != "" {
		t.Logf("stderr:\n%s", stderr)
	}
}

// TestVerifyGateCatchesHandEditedXML is the negative half, and the one that
// matters: it is the behaviour the authoring contract's first invariant claims
// ("verify rebuilds XML from Excel and asserts byte-equality").
//
// Nothing tested it. There was a TestVerifyModifiedXML, but three separate
// things made it inert (#1010):
//
//  1. It was guarded by a hardcoded absolute path under one developer's
//     GOPATH, so it skipped everywhere else.
//  2. It asserted nothing — it logged the exit code and said "the test
//     verifies no panic occurs".
//  3. The gate itself never ran: verify copied the project to a temp dir and
//     then looked for the copy one directory above where it landed, so the
//     rebuild was skipped and the comparison loop skipped both trees.
//
// So a hand-edited decision table survived build, passed verify, and shipped —
// exactly the state the contract says is prevented. This asserts the failure
// names the tampered file, not merely that the exit code is non-zero: on a
// project with any other finding, non-zero would pass no matter what.
func TestVerifyGateCatchesHandEditedXML(t *testing.T) {
	skipIfNoTestProject(t)

	tmp := t.TempDir()
	if err := copyDir(testProjectDir, tmp); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	proj := filepath.Join(tmp, "TestProject")
	xmlDir := filepath.Join(proj, "xml")

	entries, err := os.ReadDir(xmlDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var target string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_dt.xml") {
			target = filepath.Join(xmlDir, e.Name())
			break
		}
	}
	if target == "" {
		t.Skip("no _dt.xml in TestProject")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	// Edit a rule, the way a person bypassing Excel would: change the DSL,
	// not the whitespace. Whitespace could plausibly be normalised away by a
	// rebuild; a changed condition cannot be.
	const marker = "<condition_dsl>"
	idx := strings.Index(string(data), marker)
	if idx < 0 {
		t.Skip("no condition_dsl to tamper with")
	}
	end := strings.Index(string(data[idx:]), "</condition_dsl>")
	if end < 0 {
		t.Skip("malformed condition_dsl")
	}
	tampered := string(data[:idx+len(marker)]) +
		"tampered.field == 12345" +
		string(data[idx+end:])
	if err := os.WriteFile(target, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}

	var code int
	stderr := captureStderr(t, func() {
		code = NewCLI().runVerify([]string{proj})
	})

	if code == 0 {
		t.Fatalf("verify passed a hand-edited decision table — the gate is inert\n%s", stderr)
	}
	if !strings.Contains(stderr, filepath.Base(target)) {
		t.Errorf("failure does not name the tampered file %q; got:\n%s",
			filepath.Base(target), stderr)
	}
	if !strings.Contains(stderr, "differs from build output") {
		t.Errorf("failure is not the build-idempotency finding; got:\n%s", stderr)
	}
}
