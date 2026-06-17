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

const docAuthoringContract = `DTRules Authoring Contract
==========================

READ THIS FIRST if you are an LLM or tool changing rules. It is the one
contract every authoring surface obeys. Full spec: docs/authoring-contract.md.

The rule, in one sentence
-------------------------

  Excel is the system of record for DSL. Every tool that writes XML must
  write the same DSL back to Excel in the same operation. Nothing writes
  rule content any other way.


The three invariants
--------------------

  1. Excel is the system of record — for DSL only.
     The authoritative content of a rule is its EL DSL (condition_dsl,
     action_dsl, context_dsl, initial_action_dsl). Excel carries DSL and
     table structure. It does NOT carry postfix.

  2. postfix is a compiled artifact, never authored.
     postfix is the compiled output of DSL. No surface writes postfix
     independently of the DSL it comes from. Never hand-write postfix or
     bytecode.

  3. The authoring API is the only programmatic writer, and it is
     write-through. Every API write atomically (a) updates the XML DSL,
     (b) compiles DSL -> postfix into the XML, and (c) updates Excel to
     match. Step (c) is mandatory and fail-closed; if Excel is absent the
     API bootstraps it from the XML.


Exactly two ways to change a rule
---------------------------------

  (a) Edit Excel, then 'dtrules build'        (the human path)
      build extracts DSL to XML and compiles postfix. Excel is the input;
      XML is generated.

  (b) The authoring API                        (the programmatic path)
      'dtrules table' / 'dtrules edd' (and the MCP write tools) update the
      XML DSL, compile postfix, AND update Excel in one operation.

There is no third way. XML is a generated artifact; postfix is compiled.


Hard rules
----------

  - NEVER hand-edit XML. It is generated. Edits go through Excel or the
    authoring API.
  - NEVER hand-write postfix. It is the compiled output of DSL.
  - There is no command that writes rule content into XML alone. The bypass
    writers ('dtrules compile' and 'dtrules build --from-xml') were removed
    in v1.16.0.


Enforcement (in the tools, not just prose)
------------------------------------------

An improperly-authored rule set does not produce a working, committable
artifact:

  - The strict loader refuses DSL with missing/empty postfix.
  - 'dtrules verify' fails when:
      * build(committed Excel) would change the committed XML/Excel (drift);
      * a rule project has decision-table/EDD XML but no Excel workbook
        (Excel-presence gate — you skipped building the system of record);
      * a table performs an undefined table, reads an EDD field its entity
        does not declare, or uses an undefined operator (external-reference
        gate — rules must be self-contained).

So an agent that scribbles directly into XML produces a state the next
build overwrites and that verify rejects in CI. The improper path is
self-defeating.


See also
--------

    dtrules docs cli           the two authoring paths, end to end
    dtrules docs workflow      the build pipeline
    dtrules docs authoring     the Go SDK / JSON CLI behind the API
    dtrules docs warnings      advisory + reference-existence findings
`
