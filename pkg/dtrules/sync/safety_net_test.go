package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findRepoRoot walks up from the test binary's working directory until it finds go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod found)")
		}
		dir = parent
	}
}

// TestSafetyNet_MakeCheckCoversFullBuild asserts the Makefile 'check' target
// invokes `go build ./...` (unscoped) so that an unused-but-broken helper in
// any package fails the build -- the exact class of regression that landed
// in #515 and #527 when scoped tests passed but the module didn't build.
func TestSafetyNet_MakeCheckCoversFullBuild(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(findRepoRoot(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "go build ./...") {
		t.Fatal("Makefile does not contain 'go build ./...' -- the check target must exercise the full module")
	}
}
