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

const docEmbedding = `Embedding DTRules in a Go Binary
=================================

Goal: ship ONE statically-linked binary that contains every decision table,
EDD entry, and mapping.  No xlsx files.  No xml/ directory at runtime.
Copy the binary — it just works.


What goes into the embed
------------------------
After running 'dtrules build', the xml/ directory in your project contains
the compiled XML artifacts that the runtime reads.  That is the only thing
you embed.

    //go:embed rules/xml
    var rulesFS embed.FS

'rules/xml' is a copy of (or symlink to) your project's xml/ directory,
placed inside your Go module.  The embed directive captures every file in
that subtree into the binary at compile time.

Do NOT embed:
  - xlsx files (they are the authoring source, not the runtime artifact)
  - testfiles/
  - excel/.sync-manifest.json
  - Any file outside xml/

A typical layout inside your Go module:

    myapp/
    ├── main.go                 # //go:embed rules/xml
    ├── rules/
    │   └── xml/                # copy of your project's compiled xml/ tree
    │       ├── TaxReturn_edd.xml
    │       ├── TaxReturn_dt.xml
    │       └── states/
    │           ├── CO_edd.xml
    │           └── CO_dt.xml
    └── go.mod


Current limitation: loader requires a filesystem path
------------------------------------------------------
The session loader (session.LoadFromDirectory) currently accepts an
os-filesystem directory path, not an fs.FS.  Until the API is updated
(see follow-up issue #536), copy the embedded files to a temp directory
at startup, then load by path.

A follow-up issue has been filed:
  https://github.com/DTRules/DTRules/issues/536
  "Loader should accept fs.FS directly (no tempdir round-trip)"

The minimal end-to-end example below uses the tempdir workaround.
When issue #536 is resolved the tempdir block can be replaced with a
direct fs.FS call.


Minimal end-to-end example
---------------------------
The following self-contained program embeds a compiled rules tree, loads it,
sets one input field, calls a decision table, and reads a result.

    package main

    import (
        "embed"
        "fmt"
        "io/fs"
        "log"
        "os"
        "path/filepath"

        "github.com/DTRules/DTRules/pkg/dtrules"
        "github.com/DTRules/DTRules/pkg/dtrules/session"
    )

    //go:embed rules/xml
    var rulesFS embed.FS

    func main() {
        // --- step 1: copy embedded xml to a temp directory ---
        // (workaround until loader accepts fs.FS; see issue #536)
        tmpDir, err := os.MkdirTemp("", "dtrules-*")
        if err != nil {
            log.Fatal(err)
        }
        defer os.RemoveAll(tmpDir)

        if err := copyEmbedToDir(rulesFS, "rules/xml", tmpDir); err != nil {
            log.Fatal(err)
        }

        // --- step 2: load the rule set ---
        rs, err := session.LoadRulesFromDirectory("TaxReturn", tmpDir)
        if err != nil {
            log.Fatal(err)
        }

        // --- step 3: create a session and build input entities ---
        sess, err := rs.NewSession()
        if err != nil {
            log.Fatal(err)
        }

        taxpayer, err := sess.CreateEntity(dtrules.GetRName("taxpayer"))
        if err != nil {
            log.Fatal(err)
        }
        _ = taxpayer.Put(dtrules.GetRName("w2_wages"), dtrules.GetRDoubleValue(50000))
        _ = taxpayer.Put(dtrules.GetRName("filing_status"), dtrules.NewRString("SINGLE"))

        state := sess.GetState()
        state.EntityPush(taxpayer)

        // --- step 4: execute the entry-point decision table ---
        if err := sess.Execute("Compute_Tax_Return"); err != nil {
            log.Fatal(err)
        }

        // --- step 5: read a result field ---
        result, err := sess.CreateEntity(dtrules.GetRName("result"))
        if err != nil {
            log.Fatal(err)
        }
        val, err := result.Get(dtrules.GetRName("federal_tax"))
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("federal_tax = %s\n", val.StringValue())
    }

    // copyEmbedToDir copies all files from an embed.FS subtree to a real directory.
    func copyEmbedToDir(fsys embed.FS, src, dst string) error {
        return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
            if err != nil {
                return err
            }
            rel, _ := filepath.Rel(src, path)
            target := filepath.Join(dst, rel)
            if d.IsDir() {
                return os.MkdirAll(target, 0755)
            }
            data, err := fsys.ReadFile(path)
            if err != nil {
                return err
            }
            return os.WriteFile(target, data, 0644)
        })
    }

To use this in your own project:
  1. Run 'dtrules build' to produce your xml/ tree.
  2. Copy the xml/ directory to rules/xml inside your Go module.
  3. Paste the code above into main.go, adjusting the entity names and
     decision table name for your rule set.
  4. 'go build -ldflags="-s -w" -trimpath' produces the final binary.


Build pipeline
--------------

Development:
    dtrules build                   # xlsx <-> xml round-trip on disk

CI gate (committed xml must match xlsx):
    dtrules verify                  # fail if xml/ is out of sync with xlsx

Embedding step (before 'go build'):
    cp -r sampleprojects/TaxReturn/xml myapp/rules/xml

Final binary:
    go build -ldflags="-s -w" -trimpath -o myapp ./cmd/myapp/

Deploy:
    scp myapp server:/usr/local/bin/   # one file, no dependencies

The resulting binary is fully self-contained.  No xlsx, no xml/, no
filesystem dependencies.


Anti-patterns
-------------
Do NOT ship xlsx files alongside the binary.
Do NOT ship the xml/ directory separately from the binary.
Do NOT read xlsx at runtime.
Do NOT use os.ReadFile or os.Open on rule artifacts — they are embedded.
Do NOT embed .sync-manifest.json (it is machine-local and meaningless elsewhere).


Versioning
----------
Rule updates require a new binary build.  There is no hot-reload mechanism
and no remote rules support in this configuration — both are out of scope.

To track which rules are in a given binary:
  - Use 'dtrules version' inside the binary: it reports the binary version.
  - Store a short git SHA or semver string in a constant and print it on startup.
  - Tag the git commit that produced the xml/ snapshot you embedded.

Rule updates are just binary updates: build, test, deploy the new binary.


Binary size
-----------
Embedded XML rule files are plain text, so they compress well into the binary.
For reference, the TaxReturn sample project's compiled xml/ tree is
approximately 3.8 MB on disk.  After Go's build toolchain processes it, the
contribution to the final binary is typically 1–3 MB (varies with rule set
complexity and -ldflags="-s -w" stripping).

For a large production rule set (dozens of states, hundreds of tables), expect
5–20 MB of embedded rule content.  This is well within normal binary size
budgets for server applications.


See Also
--------
  dtrules docs workflow        Build and sync pipeline
  dtrules docs project-layout  Directory structure conventions
  dtrules docs xml-format      Compiled XML format
  dtrules docs edd             Entity Data Dictionary
  dtrules docs decision-tables Decision table authoring
`
