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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDupXMLProject(t *testing.T, layout map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	xmlDir := filepath.Join(root, "xml")
	if err := os.MkdirAll(xmlDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for name, tables := range layout {
		var sb strings.Builder
		sb.WriteString("<decision_tables>\n")
		for _, n := range tables {
			fmt.Fprintf(&sb, "  <decision_table><table_name>%s</table_name>"+
				"<attribute_fields><type>FIRST</type></attribute_fields>"+
				"</decision_table>\n", n)
		}
		sb.WriteString("</decision_tables>\n")
		if err := os.WriteFile(filepath.Join(xmlDir, name), []byte(sb.String()), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return root
}

func runProjectCmd(t *testing.T, projectPath string, args []string) (stdout, stderr string, exit int) {
	t.Helper()
	var so, se bytes.Buffer
	cmd := &tableCmdCtx{
		stdin:       strings.NewReader(""),
		stdout:      &so,
		stderr:      &se,
		projectPath: projectPath,
	}
	switch args[0] {
	case "diagnostics":
		exit = cmd.projectDiagnostics()
	default:
		t.Fatalf("unknown project subcommand %q", args[0])
	}
	return so.String(), se.String(), exit
}

func TestProjectDiagnosticsCLI_WithDuplicates(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"001_a_dt.xml": {"Foo"},
		"002_b_dt.xml": {"Foo"},
	})

	out, _, code := runProjectCmd(t, root, []string{"diagnostics"})
	if code != 0 {
		t.Fatalf("exit %d, stdout=%s", code, out)
	}

	var got struct {
		Diagnostics []struct {
			Kind         string `json:"kind"`
			OriginalName string `json:"original_name"`
			AssignedName string `json:"assigned_name"`
			File         string `json:"file"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse: %v\nstdout=%s", err, out)
	}
	if len(got.Diagnostics) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(got.Diagnostics), got.Diagnostics)
	}
	d := got.Diagnostics[0]
	if d.Kind != "duplicate_table" || d.OriginalName != "Foo" || d.AssignedName != "Foo-1" {
		t.Errorf("unexpected diagnostic: %+v", d)
	}
	if filepath.Base(d.File) != "002_b_dt.xml" {
		t.Errorf("File base = %q, want 002_b_dt.xml", filepath.Base(d.File))
	}
}

func TestProjectDiagnosticsCLI_Clean(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo"},
		"b_dt.xml": {"Bar"},
	})

	out, _, code := runProjectCmd(t, root, []string{"diagnostics"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Diagnostics []interface{} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got.Diagnostics) != 0 {
		t.Errorf("want 0 diagnostics, got %d", len(got.Diagnostics))
	}
}

// --- compile-time gate (Layer 3) ---

func TestDupCheck_SuffixRejected(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo", "Foo-1"},
	})
	findings, err := checkNoDupMarkers(filepath.Join(root, "xml"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if !findings[0].Suffix || findings[0].Name != "Foo-1" {
		t.Errorf("unexpected finding: %+v", findings[0])
	}
	if !strings.Contains(findings[0].Message(), "`-N` suffix") {
		t.Errorf("message missing hint:\n%s", findings[0].Message())
	}
}

func TestDupCheck_OnDiskDuplicateRejected(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo"},
		"b_dt.xml": {"Foo"},
	})
	findings, err := checkNoDupMarkers(filepath.Join(root, "xml"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Suffix {
		t.Errorf("expected on-disk duplicate, got suffix finding: %+v", f)
	}
	if len(f.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(f.Files))
	}
}

func TestDupCheck_CleanProjectPasses(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo"},
		"b_dt.xml": {"Bar"},
	})
	findings, err := checkNoDupMarkers(filepath.Join(root, "xml"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("want 0 findings, got %+v", findings)
	}
}

func TestValidateCLI_FailsOnSuffix(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo-1"},
	})
	cli := NewCLI()
	// Silence "validate" stdout by redirecting — the implementation prints
	// directly via fmt. We just care about the exit code and the stderr
	// containing the `-N` reject.
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
	})

	code := cli.runValidate([]string{"--project", root})

	wOut.Close()
	wErr.Close()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)

	if code == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s stderr=%s", outBuf.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Foo-1") {
		t.Errorf("stderr missing Foo-1:\n%s", errBuf.String())
	}
}

func TestBuildCLI_FailsOnSuffix(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo-1"},
	})
	cli := NewCLI()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
	})

	code := cli.runBuild([]string{root})

	wOut.Close()
	wErr.Close()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)

	if code == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s stderr=%s", outBuf.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Foo-1") {
		t.Errorf("stderr missing Foo-1:\n%s", errBuf.String())
	}
}

func TestVerifyCLI_FailsOnSuffix(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo-1"},
	})
	cli := NewCLI()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
	})

	code := cli.runVerify([]string{root})

	wOut.Close()
	wErr.Close()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)

	if code == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s stderr=%s", outBuf.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Foo-1") {
		t.Errorf("stderr missing Foo-1:\n%s", errBuf.String())
	}
}

func TestValidateCLI_PassesWithoutSuffix(t *testing.T) {
	root := writeDupXMLProject(t, map[string][]string{
		"a_dt.xml": {"Foo"},
	})
	cli := NewCLI()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
	})

	code := cli.runValidate([]string{"--project", root})

	wOut.Close()
	wErr.Close()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)

	// The validate command may still fail for other reasons (missing Excel, no
	// edd, …) but the dup check itself must not contribute to the failure.
	if strings.Contains(errBuf.String(), "`-N` suffix") {
		t.Errorf("unexpected dup-check failure: code=%d stderr=%s", code, errBuf.String())
	}
}
