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

const docCompile = `dtrules compile — surgical EL → postfix backfill
===================================================

` + "`dtrules compile <dir>`" + ` reads every ` + "`*_dt.xml`" + ` under <dir>,
runs the EL compiler on each non-comment DSL element whose
` + "`<*_postfix>`" + ` is empty, and writes the compiled postfix in place. No
Excel round-trip — bytes outside the targeted postfix elements stay
untouched.

It also runs ` + "`decisiontable.Analyze`" + ` over every table after the fill
loop (v1.14.1) and prints advisory warnings to stderr. ` + "`--no-analyze`" + `
disables this if you only want the postfix-fill.

Introduced in v1.14.0; advisory-on-by-default added in v1.14.1; EDD
symbol table wiring (so ` + "`fixed`" + ` and ` + "`double`" + ` operands get the right
operator family) in v1.14.2.

When to use it
--------------

  • You authored EL DSL and want the XML to ship with pre-compiled
    postfix. Required by the v1.14.0 strict loader — see
    ` + "`dtrules docs workflow`" + `.

  • You want advisory warnings on a project layout that ` + "`dtrules build`" + `
    and ` + "`dtrules review`" + ` don't accept (no ` + "`xml/`" + ` subdir). Library
    consumers whose rules live at e.g. ` + "`pkg/dtrules/rules/*.xml`" + ` flat
    use this as their one-stop authoring check.

  • You're refreshing postfix after a compiler bug fix and want to
    overwrite the stored output. Use ` + "`--force`" + ` (v1.14.2).

How it differs from ` + "`dtrules build`" + `
----------------------------------------

  dtrules build       Full Excel/XML sync + EL compile pipeline.
                      Requires the canonical project layout
                      (<project>/xml/ and <project>/excel/).
                      XML-authored mode round-trips XML → Excel → XML,
                      which canonicalizes whitespace and attribute
                      ordering — lossy for stored postfix the EL
                      grammar can't express.

  dtrules compile     Just the EL → postfix step. Surgical: only
                      <*_postfix> bytes change. Works on any
                      directory layout, including flat ones. The
                      canonical backfill tool when XML authoring
                      outpaces the build pipeline.

Flags
-----

  --dry-run     Report what would change without writing files.

  --strict      Atomic-or-nothing per file: any compile error blocks
                the file's write. Default writes the successful fills
                and reports errors as a to-do list.

  --force       Rewrite existing postfix from a fresh compile.
                Default only fills empty postfix. Use after a compiler
                bug fix to refresh stored postfix produced by an
                earlier buggy version.

  --no-analyze  Skip the advisory pass after compile. Default runs it.

  -v, --verbose Per-file compiled/skipped/error/warning counts.

Exit codes
----------

  0   Every DSL element compiled cleanly (or was already populated).
      Advisory warnings do NOT change exit code — they exit 0.
  1   At least one DSL element failed to compile, or an I/O error
      occurred.

Examples
--------

Backfill postfix for a staking-like flat layout:

  dtrules compile pkg/dtrules/rules

Force-refresh after a compiler bug fix:

  dtrules compile --force --no-analyze pkg/dtrules/rules
  git diff pkg/dtrules/rules

Compile a single file:

  dtrules compile pkg/dtrules/rules/staking_dt.xml

Dry-run + verbose to preview a backfill:

  dtrules compile --dry-run --verbose sampleprojects/MyProject

CI gate (atomic per file, no advisory noise):

  dtrules compile --strict --no-analyze rules/ || exit 1

Behavior notes
--------------

  • TEMPLATE_*.xml is skipped. Template files intentionally contain
    placeholder text ([STATE], [Filing Status], etc.) that isn't
    valid EL.

  • Comment-only DSL (#, //, or /* ... */) is recognized and the
    postfix is left empty — that's a no-op element by convention.

  • EDD discovery: all *_edd.xml under the target directory are
    parsed for field types. The type map is passed to the EL compiler
    via SetSymbols before the compile loop, so 'fixed' operands emit
    fp+/fp-/fpmin etc. instead of integer ops. With no EDD found,
    the compiler defaults to integer-typed operands — fine for
    int-only projects but wrong for fixed/double-typed math.

  • Strict-mode write semantics: any error in a file blocks that
    file's write. Other files are unaffected.

  • Default-mode write semantics: successful fills are written even
    if other elements in the same file fail to compile. The file is
    improved incrementally and errors surface as a to-do list.

Related topics: ` + "`dtrules docs warnings`" + ` (every warning kind),
` + "`dtrules docs workflow`" + `, ` + "`dtrules docs cli`" + `.
`
