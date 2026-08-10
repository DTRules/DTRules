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

package authoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Everything that writes a rule goes through the authoring API. Excel is the
// system of record and `postfix` is compiled, never authored, so a writer that
// emits rule XML on its own can produce files no `dtrules build` would
// reproduce -- uncompiled EL, no advisory pass, no sync manifest, and a
// `verify` that is red for reasons nobody can act on.
//
// That is not hypothetical. `dtrules compile` was removed for exactly this;
// `excel2dt`, `excel2edd` and `edd2excel` were removed for it; and the HTTP
// surface silently skipped the Excel half of the contract on every save until
// #804. Each was found by someone tripping over the damage rather than by the
// build, so this test exists to fail first instead.
//
// If a new writer is genuinely needed, the answer is to build it through the
// authoring API and add its package here with a note saying why.
var writeFunnelAllowed = map[string]string{
	// The authoring API itself -- Project.Save, SaveEDD, and the Excel refresh
	// and mtime guard they wrap.
	"pkg/dtrules/authoring": "is the authoring API",

	// The single XML emission funnel every sanctioned path calls into. Owning
	// the serialisation is the point; it is not a way around the contract.
	"pkg/dtrules/excel": "is the emission funnel the authoring API uses",

	// The build pipeline: Excel -> XML, the documented human path.
	"cmd/dtrules": "is the build funnel and the CLI authoring surface",

	// Writes XML, but only into a MkdirTemp sandbox for speculative execution,
	// and only inside the apiserver's own scratch overlay -- never the project.
	// The project-facing save in this package is guarded and refreshes Excel.
	"pkg/dtrules/apiserver": "writes the project only through a guarded save; speculation is sandboxed",
}

// TestNoRuleWriterOutsideTheFunnel walks the tree and fails when a package
// that is not accounted for emits rule XML.
func TestNoRuleWriterOutsideTheFunnel(t *testing.T) {
	root := repoRoot(t)

	// Signatures of "this code writes a rule file", not "this code knows what
	// one is called". Globbing for *_dt.xml is what the loader and the
	// analysis passes do all day; the two things that actually emit are a
	// call into the XML funnel, and hand-rolled markup written straight out.
	emitsViaFunnel := func(body string) bool {
		return strings.Contains(body, "WriteXML(")
	}
	emitsByHand := func(body string) bool {
		if !strings.Contains(body, "os.WriteFile") {
			return false
		}
		for _, markup := range []string{"<decision_table", "<entity_data_dictionary", "<decision_tables"} {
			if strings.Contains(body, markup) {
				return true
			}
		}
		return false
	}

	var offenders []string
	for _, dir := range []string{"cmd", "pkg"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			pkgDir := filepath.ToSlash(filepath.Dir(rel))
			if _, ok := writeFunnelAllowed[pkgDir]; ok {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			body := string(data)
			switch {
			case emitsViaFunnel(body):
				offenders = append(offenders, filepath.ToSlash(rel)+" calls WriteXML")
			case emitsByHand(body):
				offenders = append(offenders, filepath.ToSlash(rel)+" writes rule XML by hand")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(offenders) > 0 {
		t.Errorf("these write rule files outside the authoring API:\n  %s\n\n"+
			"Rules change two ways and no others: edit Excel and run `dtrules build`, "+
			"or call the authoring API, which writes the XML, compiles the postfix and "+
			"updates Excel in one operation. A new writer needs a reason and an entry "+
			"in writeFunnelAllowed saying what it is.",
			strings.Join(offenders, "\n  "))
	}
}

// repoRoot walks up from the test's working directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find the module root")
	return ""
}
