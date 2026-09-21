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


The supported embedding path
----------------------------
The engine packages ARE the embedding API. There is no wrapper package and
none is planned: 'pkg/dtrules/sdk' was written and removed (commit 69774f70)
because data already enters through the EDD as XML, so a parallel
programmatic entity API restated the same surface in a second, unvalidated
shape. Do not re-propose it. Both CLI binaries, pkg/dtrules/web and
pkg/dtrules/interview embed the engine exactly the way this page shows.

What you import:

  pkg/dtrules              shared types: RName, Entity, State, Session
  pkg/dtrules/session      rule sets and sessions
  pkg/dtrules/mapping      input XML whose tags are not yours to choose
  pkg/dtrules/datafile     canonical data XML (tags 1:1 with the EDD)
  pkg/dtrules/entity       *entity.REntity, for the datafile callbacks
  pkg/dtrules/trace        optional trace capture (with pkg/dtrules/interpreter)
  pkg/dtrules/interview    optional: run a table as an interactive interview

The sequence, in order:

  1. Load     session.NewRuleSet(name), then LoadFromDirectory(xmlDir) or
              LoadFromFS(fsys, root) for an embedded tree. A loaded rule set
              is immutable — load once, share it.
  2. Session  rs.NewSession() per execution; a session owns one entity stack.
  3. Data in  a mapping, or canonical data XML (both shown below).
  4. Execute  GetDecisionTable(GetRName(entry)), then dt.Execute(state).
  5. Read out state.FindEntity(...) — the EXECUTED instance on the stack.
              sess.CreateEntity() returns a fresh, empty entity; it is not
              how a result is read.
  6. Trace    optional; see "TRACES FOR DEBUGGING AND VALIDATION" below.

pkg/dtrules/embedding_example_test.go compiles and runs this sequence on
SinusitisTherapy, both ways of loading data, and pins the answers against
what 'dtrules run' produces.


Minimal end-to-end example
---------------------------
A self-contained program that embeds a compiled rules tree, loads it directly
from the embed.FS (no temp-directory round-trip), supplies input as canonical
data XML, runs the entry table, and reads a result.

    package main

    import (
        "embed"
        "fmt"
        "log"
        "os"

        "github.com/DTRules/DTRules/pkg/dtrules"
        "github.com/DTRules/DTRules/pkg/dtrules/datafile"
        "github.com/DTRules/DTRules/pkg/dtrules/entity"
        "github.com/DTRules/DTRules/pkg/dtrules/session"
    )

    //go:embed rules/xml
    var rulesFS embed.FS

    func main() {
        // --- step 1: load the rule set directly from the embedded FS ---
        rs, err := session.LoadRulesFromFS("TaxReturn", rulesFS, "rules/xml")
        if err != nil {
            log.Fatal(err)
        }

        // --- step 2: one session per execution ---
        sess, err := rs.NewSession()
        if err != nil {
            log.Fatal(err)
        }
        state := sess.GetState()

        // --- step 3: push the singletons the rules resolve bare names
        // against (the set a mapping's <initialentity> would name) ---
        for _, name := range []string{"constants", "result", "taxpayer"} {
            e, err := sess.CreateEntity(dtrules.GetRName(name))
            if err != nil {
                log.Fatal(err)
            }
            if err := state.EntityPush(e); err != nil {
                log.Fatal(err)
            }
        }

        // --- step 4: read canonical data XML into those instances ---
        find := func(name string) *entity.REntity {
            e, err := state.FindEntity(dtrules.GetRName(name))
            if err != nil || e == nil {
                return nil
            }
            re, _ := e.(*entity.REntity)
            return re
        }
        create := func(subtype string) (*entity.REntity, error) {
            e, err := sess.CreateEntity(dtrules.GetRName(subtype))
            if err != nil {
                return nil, err
            }
            re, _ := e.(*entity.REntity)
            return re, nil
        }
        data, err := os.Open("taxpayer.xml")
        if err != nil {
            log.Fatal(err)
        }
        defer data.Close()
        if err := datafile.Read(data, find, create, datafile.Authoritative); err != nil {
            log.Fatal(err)
        }

        // --- step 5: execute the entry-point decision table ---
        dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
        if err != nil || dt == nil {
            log.Fatal("entry table not found")
        }
        if err := dt.Execute(state); err != nil {
            log.Fatal(err)
        }

        // --- step 6: read a result field off the STACK ---
        result, err := state.FindEntity(dtrules.GetRName("result"))
        if err != nil || result == nil {
            log.Fatal("no result entity")
        }
        val, err := result.Get(dtrules.GetRName("federal_tax"))
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("federal_tax = %s\n", val.StringValue())
    }

To use this in your own project:
  1. Run 'dtrules build' to produce your xml/ tree.
  2. Copy the xml/ directory to rules/xml inside your Go module.
  3. Paste the code above into main.go, adjusting the entity names and
     decision table name for your rule set.
  4. 'go build -ldflags="-s -w" -trimpath' produces the final binary.


Input through a mapping instead
-------------------------------
Canonical data XML assumes you choose the tag names. When the document comes
from somewhere else, the project's *_map.xml reconciles its tags against the
EDD; replace steps 3 and 4 with:

    mf, err := os.Open(mapFile)          // the project's *_map.xml
    if err != nil {
        log.Fatal(err)
    }
    defer mf.Close()

    m := mapping.NewMapping(sess)
    if err := m.LoadMapping(mf); err != nil {
        log.Fatal(err)
    }

    in, err := os.Open(inputFile)
    if err != nil {
        log.Fatal(err)
    }
    defer in.Close()

    // Reads the document, THEN pushes the cardinality-1 entities it created,
    // so the stack holds the loaded instances rather than empty singletons.
    if err := m.LoadDataAndPushSingletons(in); err != nil {
        log.Fatal(err)
    }

With no input document, m.Initialize() pushes the <initialentity> singletons
and nothing else — use it when the values arrive some other way (interactive
collection, or a canonical data load layered on top).

A canonical data file is also what 'dtrules run --save' writes and what
'--data' / '--review' read, so a saved dataset is both the audit record of a
run and a replay fixture for an embedded one. It is written in EDD
declaration order (entities, and the fields within each), so the same rules on
the same input save byte-identical files.


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


Dumping embedded rules back to xlsx
------------------------------------
When debugging a deployed binary or handing rules back to an author without
the dev repo, use excel.ExtractExcel to reconstruct the xlsx tree from the
embedded XML.  This is useful for field debugging, compliance audits, and
letting rule authors inspect exactly what shipped in a given binary.

    // Minimal app that can dump its own embedded rules:
    func main() {
        if len(os.Args) > 2 && os.Args[1] == "--dump-rules" {
            if err := excel.ExtractExcel(rulesFS, os.Args[2]); err != nil {
                log.Fatal(err)
            }
            return
        }
        // ... normal app startup ...
    }

ExtractExcel walks the embedded fs.FS and writes one xlsx file for every
*_dt.xml, *_edd.xml, and *_map.xml it finds, preserving the subdirectory
layout.  The output directory is created if it does not exist.


See Also
--------
  dtrules docs workflow        Build and sync pipeline
  dtrules docs project-layout  Directory structure conventions
  dtrules docs xml-format      Compiled XML format
  dtrules docs edd             Entity Data Dictionary
  dtrules docs decision-tables Decision table authoring

TRACES FOR DEBUGGING AND VALIDATION
-----------------------------------

An embedded engine can record execution traces that load straight into
the visual trace debugger — step through the run, ask where a value came
from, generate outcome reports, and rerun the same inputs against a
speculative table edit. See the "TRACES FROM AN EMBEDDING GO PROGRAM"
section of ` + "`dtrules docs debug`" + ` for the exact wiring (WriteHeader ->
SetOutput/EnableTrace BEFORE data load -> Execute -> WriteFinalState).
`
