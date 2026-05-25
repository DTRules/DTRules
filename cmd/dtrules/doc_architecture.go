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

const docArchitecture = `DTRules Architecture: Dev-Time vs. Deploy-Time
===============================================

DTRules has one packaging asymmetry: at dev-time you work with many files on
disk; at deploy-time you ship a single statically-linked binary. There are not
two runtime paths — there is one binary, and all the artifacts that produced it
stay behind on the developer machine.

Since v1.14.0: the loader is strictly a postfix consumer. EL is compiled
at authoring time ('dtrules build' or 'dtrules compile'); the runtime
binary no longer pulls in the compiler/el package or its grammar tables.
This is what makes the deploy-time binary as small as it is — the
authoring toolchain stays on the dev side. The XML that ships must have
compiled postfix; DSL with no postfix is a load error directing the
operator to 'dtrules build'. See 'dtrules docs workflow' for the
migration steps, 'dtrules docs compile' for the surgical backfill path.


Dev-Time Layout
---------------
During development and CI, every artifact lives on disk as an inspectable file.
The full structure is:

  project/
  ├── excel/                      # Source of truth (human-editable)
  │   ├── .sync-manifest.json     # Tracks export timestamps — do not commit
  │   ├── Rules.xlsx
  │   └── states/
  │       ├── CO.xlsx
  │       └── CA.xlsx
  ├── xml/                        # Compiled from Excel; AI authors *_dsl tags here
  │   ├── Rules_edd.xml
  │   ├── Rules_dt.xml
  │   └── states/
  │       ├── CO_edd.xml
  │       ├── CO_dt.xml
  │       ├── CA_edd.xml
  │       └── CA_dt.xml
  └── testfiles/                  # Test scenarios
      └── TestScenarios/

Every transformation is persisted as an artifact — excel <-> xml <-> testfiles.
This gives maximum visibility: humans read Excel, AI agents read XML, CI runs
test scenarios. 'dtrules build' round-trips between them.

Dev-time artifact graph:

  +-----------------------------------------------------+
  |                  Developer Machine / CI              |
  |                                                     |
  |   excel/*.xlsx  <------------------------------>  xml/*.xml     |
  |         |           dtrules build                    |         |
  |         |                                            |         |
  |         +-------------> testfiles/ <-----------------+         |
  |                        (test scenarios)                        |
  +-----------------------------------------------------+

The dev-time layout lives on developer machines and CI runners only.
It is never deployed to production.


Deploy-Time Layout
------------------
A deployment is a single statically-linked Go binary. Nothing else.

  production host:
  └── myapp          (one binary — no .xlsx, no .xml, no testfiles)

All EDD and decision-table XML is embedded into the binary at compile time via
Go's //go:embed directive. When the process starts, rules are loaded directly
from the binary — no file paths, no workbook names, no sync manifest.

The surface area the application touches is purely DTRules types:

  import "github.com/DTRules/DTRules/pkg/dtrules/sdk"

  rs, err := sdk.LoadEmbedded(embeddedFS)  // rules come from binary
  session := rs.NewSession()
  session.SetEntity("input", map[string]any{"income": 85000.0})
  session.Execute("Compute_Tax_Return")
  result := session.GetEntity("result")

The application never sees file paths, workbook names, or XML content directly.

Deploy-time graph:

  +------------------------------------------------------+
  |                   Production Host                    |
  |                                                      |
  |   +----------------------------------------------+   |
  |   |              myapp binary                    |   |
  |   |                                              |   |
  |   |   embedded xml/   -->  RuleSet               |   |
  |   |                            |                 |   |
  |   |   consumer app  ------->  Session            |   |
  |   |                          (execute tables)    |   |
  |   +----------------------------------------------+   |
  +------------------------------------------------------+


The Key Rule
------------
If an app reads an xlsx or xml file at runtime, the deployment is wrong.

Production code must not open any DTRules file at runtime. The //go:embed
directive is what converts the xml/ tree into part of the binary — that
happens at compile time, not at process startup. A file read at startup means
the binary was not built correctly, or an old pattern is being used.


Embedding Recipe (Go)
---------------------
In the package that owns your rule set:

  //go:embed xml/*.xml xml/states/*.xml
  var rulesFS embed.FS

  func LoadRules() (*sdk.RuleSet, error) {
      return sdk.LoadEmbedded(rulesFS)
  }

See 'dtrules docs embedding' for the full embedding recipe with error handling.


CI Build Guidance
-----------------
Build the deployment binary from the committed xml/ tree only, after
'dtrules verify' passes. Do not include xlsx files in the production build
context.

Recommended CI pipeline:

  # 1. Verify rule consistency (EL compliance, sync status)
  dtrules verify

  # 2. Run test scenarios
  go test ./...

  # 3. Build the deployment binary
  go build -o build/myapp ./cmd/myapp/

The build is reproducible: the same commit produces the same binary, which
embeds the same rules. There is no runtime file loading to introduce variance.

What NOT to do in CI:

  # Wrong: copying xlsx into build context
  COPY excel/ /build/excel/

  # Wrong: reading xml at runtime
  rs, _ := sdk.LoadDir("./xml")   // do not do this in production code


Summary
-------
  dev-time:    Many files on disk — Excel, XML, testfiles, sync manifest.
               Lives on developer machines and CI runners only.
               Never deployed.

  deploy-time: One binary. All XML embedded via //go:embed.
               No xlsx, no xml files, no testfiles on the deployment host.
               The app never sees file paths or workbook names.

The asymmetry is intentional. dev-time visibility (inspectable artifacts,
human-editable Excel) enables confident rule authoring. deploy-time simplicity
(one binary) removes an entire class of deployment bugs.


See Also
--------
  dtrules docs embedding      Concrete //go:embed recipe and SDK usage (#535)
  dtrules docs project-layout Dev-time folder conventions (#531)
  dtrules docs workflow       Development workflow with Excel and XML
  dtrules docs edd            Entity Data Dictionary reference
`
