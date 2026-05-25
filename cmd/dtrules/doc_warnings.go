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

const docWarnings = `Advisory Warnings
=================

DTRules reports advisory warnings about decision tables and EDDs through
` + "`decisiontable.Analyze`" + `, the canonical advisory pass. Warnings are
*advisory* — they never gate deployment. Authors are expected to read
them, decide which to fix, and which to keep (sometimes a "redundant"
condition is documentation, or an "assignment-only table" is intentional
for audit trail purposes).

This topic catalogues every warning kind, what triggers it, and what
to do about it.

Where you see them
------------------

After v1.14.x, advisory warnings surface on every authoring/build
surface:

  dtrules build                  After import; from "Nothing to sync" branch too (#787)
  dtrules compile <dir>          Default mode runs the advisory pass (--no-analyze to skip)
  dtrules table get <name>       Embedded in the JSON response
  dtrules table put <name>       Embedded in the response after the edit lands
  dtrules table patch <name>     Same
  dtrules table warnings <name>  Read-only fetch
  dtrules review                 Full project report; persisted to .dtrules/last-review.json
  MCP tools (table_get, table_put, table_patch, table_warnings, project_full_review)

All surfaces share one source of truth: ` + "`decisiontable.Analyze(Inputs{...})`" + `.
Adding a new authoring surface that doesn't go through that call is a
contract violation.

Warning kinds
-------------

The set below is exhaustive for v1.14.x. Each entry follows the same
shape: a one-line description, a minimal repro, and the typical action.

1. no-op column
~~~~~~~~~~~~~~~

Column N has no actions marked X — reaching it produces no behavior.

  Repro:
    column 1: condition Y, action 1 X
    column 2: condition N, (no actions)

  Action: delete the column unless you're keeping it as a "fall-through
  matches but does nothing" placeholder. If the latter, the warning is
  noise; reorganize so the same effect comes from a default ALL row or
  from making the table FIRST-policy with an implicit terminal column.

2. subsumed column
~~~~~~~~~~~~~~~~~~

Column A is redundant because column B is more permissive (B has fewer
constraints) AND has all of A's actions. Anything that matches A also
matches B; reaching A is impossible under FIRST policy.

  Action: delete column A. The optimization is purely a cleanup — table
  semantics under FIRST stay the same because B fires first.

3. redundant condition (#762; FIRST-policy only)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In a FIRST table, reaching column N means every prior column failed.
A Y/N entry in column N is *redundant* if some prior column M's
failure already forces that exact value. Two patterns:

  (a) M constrains the same row R with the opposite value AND every
      other Y/N constraint in M also matches N's value for that row.
  (b) M constrains a different row R' whose DSL is the syntactic
      negation of N's row R DSL.

  Repro:
    table policy: FIRST
    column 1: row1=Y row2=Y row3=Y
    column 2: row1=N row2=- row3=-
    column 3: row1=Y row2=N row3=-      ← row1=Y is redundant: column 2 failed → row1 must be Y

  Action: replace the redundant entry with ` + "`-`" + ` (don't care). Same
  semantics, less to read.

4. unreachable column (DSL-negation; matrix-driven)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Column N requires two conditions to both be true, but the conditions
are syntactic negations of each other in DSL — at most one can hold.
The column can never fire.

  Recognized negation patterns:
    " is equal to " ↔ " is not equal to "
    " > " ↔ " <= "
    " < " ↔ " >= "
    "not <expr>" ↔ "<expr>"

  Action: review the column. Either the conditions are wrong (typo) or
  the matrix has a Y where it should be N.

5. assignment-only table (#763)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Every column's actions are pure ` + "`set X = Y`" + ` statements, and every
column assigns the same set of variables. The table is just picking a
value for those variables based on conditions — it can usually be
inlined at the call site (a context statement, local, or conditional
expression).

  Action: consider inlining. Sometimes the table is kept on purpose
  for audit-trail visibility or tool integration; the warning text
  says "consider", not "must".

6. hand-coded postfix
~~~~~~~~~~~~~~~~~~~~~

A table element has non-empty ` + "`<*_postfix>`" + ` but empty ` + "`<*_dsl>`" + `.
DTRules expects EL DSL as the authoring source of truth; postfix is
the compiled artifact. Hand-edited postfix bypasses the compile
pipeline and risks the next ` + "`dtrules build`" + ` re-emitting empty postfix
from the empty DSL.

  Action: author EL DSL for the element. If the postfix encodes
  something the EL grammar can't express, that's a grammar gap worth
  filing. Tables with ANY hand-coded postfix are blocked from
  execution by the runtime (the loader sets ` + "`HasHandCodedPostfix=true`" + `).

7. dead condition row (#766; tree-based, from review only)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

A condition row that no CNode in the compiled decision tree
branches on — the row contributes nothing to column selection. Often
appears when every column has the same value for that row (so the
optimizer collapsed it), or when the row is all ` + "`-`" + ` (don't care
across the board).

  Action: delete the row. Tree-based; only surfaces from
  ` + "`AnalyzeCompiledTable`" + ` which the review/MCP path runs after compile.

8. unreachable column (tree-based; #765, from review only)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

No ANode leaf in the compiled tree references column N. The compiler
proved it unreachable via subsumption, exact-duplicate matching, or
some other path the matrix-driven kind 4 above couldn't see.

  Action: delete the column.

Severity and exit codes
-----------------------

All warnings exit 0. The advisory contract from #761 is:

  Errors gate deployment. Warnings never do.

If you want to make a specific warning fatal in CI, grep for it after
` + "`dtrules build`" + ` / ` + "`dtrules compile`" + ` and fail your own pipeline.

Filtering on a project
----------------------

Per-file (compile): the warning lines go to stderr. The
` + "`advisory: N warning(s)`" + ` summary goes to stdout.

  dtrules compile rules/ 2> warnings.log
  grep "redundant condition" warnings.log

Per-table (table warnings): JSON output, structured.

  dtrules table warnings Calculate_Withholding --project pkg/dtrules \
    | jq '.warnings[] | select(.kind=="redundant condition")'

Project-wide (review): grouped and persisted.

  dtrules review --project .
  jq '.warnings | group_by(.kind) | map({kind: .[0].kind, count: length})' \
    .dtrules/last-review.json

Related topics: ` + "`dtrules docs compile`" + `, ` + "`dtrules docs workflow`" + `,
` + "`dtrules docs decision-tables`" + `.
`
