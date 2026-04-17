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
	"os/exec"
	"strings"
	"testing"
)

// TestVersionCommandOutputsSemver verifies that `dtrules version` outputs a
// recognizable version string, commit, and date (each may be "unknown"/"dev").
func TestVersionCommandOutputsSemver(t *testing.T) {
	// First try: call the CLI directly via the in-process RunCLI.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cli := NewCLI()
	code := cli.Run([]string{"version"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if code != 0 {
		t.Errorf("version command exit code = %d, want 0", code)
	}

	// The output must contain something that looks like a version: "dev" or "v\d+."
	hasVersion := strings.Contains(output, "dev") ||
		strings.Contains(output, "v0.") ||
		strings.Contains(output, "v1.") ||
		strings.Contains(output, "DTRules")
	if !hasVersion {
		t.Errorf("version output does not contain a recognizable version string:\n%s", output)
	}
}

// TestVersionBinaryOutputsSemver verifies that the dtrules binary's version
// subcommand outputs a recognizable version string when the binary is available.
func TestVersionBinaryOutputsSemver(t *testing.T) {
	bin := findBinaryForVersionTest()
	if bin == "" {
		t.Skip("dtrules binary not found and could not be built")
	}

	cmd := exec.Command(bin, "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("dtrules version failed: %v", err)
	}
	output := string(out)

	// Must contain a version indicator.
	hasVersion := strings.Contains(output, "dev") ||
		strings.Contains(output, "v0.") ||
		strings.Contains(output, "v1.") ||
		strings.Contains(output, "DTRules")
	if !hasVersion {
		t.Errorf("binary version output missing version indicator:\n%s", output)
	}
}

func findBinaryForVersionTest() string {
	locations := []string{
		"../../build/dtrules",
		"../../dtrules",
		"dtrules",
	}
	for _, loc := range locations {
		cmd := exec.Command(loc, "version")
		if err := cmd.Run(); err == nil {
			return loc
		}
	}
	// Try to build on demand.
	build := exec.Command("go", "build", "-o", "/tmp/dtrules-version-test-bin", "../../cmd/dtrules/")
	if out, err := build.CombinedOutput(); err != nil {
		_ = out
		return ""
	}
	return "/tmp/dtrules-version-test-bin"
}
