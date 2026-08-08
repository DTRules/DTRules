// Copyright 2026 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

const docDebug = `
DEBUG — TRACES, THE DEBUGGER, REPORTS, AND SPECULATIVE RERUNS
=============================================================

DTRules can record a complete trace of a run — every input value, every
condition result, every fired column, every action, every value written —
and then act as a post-mortem debugger over that recording: step through
the execution table by table, inspect entity state at any point, ask where
a value came from, generate reports over the outcome, and rerun the same
inputs against a speculatively edited table to see exactly what would
change. Project files are never touched by any of it.

THE TRUST MODEL
---------------

A trace embeds provenance: the DTRules version and a fingerprint (sha256)
of the rules that produced it. On load the debugger:

  1. compares the trace's rules fingerprint against the open project
     ("rules match" / "rules DIFFER from workspace" in the trust strip), and
  2. ALWAYS replays the trace to its end and verifies the reconstructed
     state against the recorded final state ("end-state verified").

If verification reports mismatches, do not trust fine-grained state at
intermediate positions — something about the trace or the replay is off.

PRODUCING A TRACE
-----------------

    dtrules run . --input <data.xml> --trace          # writes traces/<input>.trace.xml
    dtrules run . --input <data.xml> --trace out.xml  # explicit path

The entry decision table comes from --entry, or from <entry> in the
project's DTRules.xml:

    <DTRules>
      <xml_dir>pkg/dtrules/rules</xml_dir>   <!-- where the rules live -->
      <entry>Staking_Distribution</entry>    <!-- default entry table -->
    </DTRules>

With that config in place, the whole debugging workflow is one command:

    dtrules debug <input.xml>

which runs the entry table over the input with tracing, writes
traces/<input>.trace.xml, and opens the editor with the trace already
loaded in the Debug tab. Bare "dtrules" in a project directory prints an
overview of the rules, discovered inputs, and existing traces.

A trace of a few hundred bytes means the run died before executing —
the file browser shows sizes so a real trace (usually megabytes) is
recognizable at a glance.

THE DEBUG TAB
-------------

Open a project, then Debug > pick a trace file. (dtrules edit --trace
<file> preloads one; dtrules debug does this automatically.)

The TABLE VIEW is the primary surface: the decision table under execution
rendered as the authored sheet — CONTEXTS, INITIAL ACTIONS, CONDITIONS,
ACTIONS bands — with the fired column highlighted. Conditions and actions
are numbered by position (1, 2, 3, ...), the same coordinates the engine
and the editor use.

Focus drives the entity stack panel on the right:

    Entering row     state coming INTO this pass
    an action row    state just AFTER that action executed
    Leaving row      state leaving this pass

Hovering any field name in a DSL cell shows its value on the entity stack
at the current position — the same field shows different values as the
focus moves. The stack marks its top ("top of stack" = the current
context); frames collapse behind triangles.

Navigation:

  - A performed table name in an executed action is a link. It drills into
    that call's execution (Entering = state going into the call). A table
    that executed elsewhere in the trace jumps there instead (backward or
    forward — the position moves). A table that never ran opens its static
    definition in the editor; Esc returns to the debugger.
  - A call whose "for all" context had nothing to iterate still drills —
    the view shows the table with a banner and the state going into the
    call.
  - Multi-call actions sub-number their calls: action 4's calls are 4.1,
    4.2, ... and a drilled frame is titled with its call position
    ("4.2 Accumulate_Computed_Balance").
  - Iterating contexts show "Iteration <n> of N" — arrows step, typing a
    number jumps.
  - Up / Out finishes the current table and lands at the NEXT decision
    table call: the next call of the same action (4.1 -> 4.2), then the
    next action's call, then the caller's Leaving row.

Verbs: Run (to breakpoint), Step (next action), Into (descend into a
performed table), Pass (finish this pass), Up (out to the next call).
Marks: one drops automatically at each table entry; Mark drops one
anywhere; "To mark" walks back (step-back via replay).

The TREE view (expert) is the raw execution structure — structural nodes
only, children grouped in ranges (25/100/1000...) so thousands of passes
stay navigable. Click a node to run to it; right-click toggles a
breakpoint; Run advances to the next breakpoint.

The POSTFIX CONSOLE evaluates read-only postfix at the current position;
whatever is left on the data stack is printed. Mutating operators are
refused.

FIND WRITES — "WHERE WAS THIS SET, AND WHY?"
--------------------------------------------

The Find panel searches the trace for writes of a field. Query forms
(all names case-insensitive, per EL):

    field                                   every write of the field
    entity.field                            scoped to an entity type
    entity[key=value].field                 ONE instance, picked by a key
    entity#1234.field                       ONE instance, by id
    ... = value                             only writes of that value

The instance forms are the point: "why is THIS account ineligible?"

    staking_account[account_url=acc://fahad.acme/fahad].is_eligible

returns that account's writes — typically one — and clicking it jumps to
the exact def, in that account's own iteration. Expanding a hit shows the
WHY-CHAIN: for the writing table and every caller up the stack, the fired
column's conditions with the cell each required (Y/N) joined to the
actual recorded result — all the conditions that had to be met.

REPORTS — OUTCOMES, EDD-DRIVEN
------------------------------

The Report view composes reports from the EDD: pick an entity (every
instance the run created) or the elements of an array attribute
(e.g. staking_transaction.to), click the fields you want as columns,
add filters, sort, and choose a key field (row identity for diffs).
Specs save into the project as reports/<name>.report.json:

    {"name": "Payouts", "sections": [
      {"title": "Recipients", "source": "staking_transaction.to",
       "fields": ["url", "amount"], "sort": "amount", "key": "url"},
      {"title": "Eligible", "entity": "staking_account",
       "fields": ["account_url", "reward"],
       "where": [{"field": "is_eligible", "op": "==", "value": "true"}],
       "sort": "reward", "key": "account_url"}]}

Filter ops: == != > >= < <= contains (numeric when both sides parse,
else case-insensitive text). From the CLI:

    dtrules report traces/input-184.trace.xml --spec reports/payouts.report.json
    dtrules report new.trace.xml --spec spec.json --baseline old.trace.xml

The --baseline form runs the same spec on both traces and appends the
row-level diff (added / removed / changed, aligned by the key field).

SPECULATIVE RERUNS — "WHAT IF?"
------------------------------

The "What if..." button on any table in the debug view opens it in the
editor under a SPECULATIVE banner. "Run speculation":

  1. copies the rules to a scratch overlay and applies your edit there
     (the DSL is recompiled — compile errors come straight back);
  2. seeds a fresh session from the trace's RECORDED initial data — the
     same inputs, no input file needed;
  3. re-executes the entry table with tracing.

The speculative trace becomes the active session (SPECULATIVE chip;
"Restore baseline" discards it). Every report now runs against BOTH
traces and shows "Changes vs baseline" — rows added, removed, and
field-level changes keyed by the section's key field. Project files are
never modified; re-speculating replaces the previous speculation (edits
do not stack).

The full loop: a report row looks wrong -> Find pins where that instance's
field was set -> the why-chain shows the conditions that fired -> What if
edits the table -> the diff shows exactly which outcomes change.

TRACE FORMAT (brief)
--------------------

XML, format="2" on the <DTRulesTrace> root (dtrules_version and
rules_fingerprint attributes carry provenance). Events: value-carrying
<def> (entity/name/id attrs, value postfix body), <entitypush>/<entitypop>,
<arraybind> (attr <-> arrayId), <addto>/<remove>/<addat>/<removeat>,
<decisiontable> > <execute_table> (one per pass) > <condition n= result=>,
<column n=> > <action n=> with performed tables nested inside their
action, and a <finalState> dump used for verification. Condition/action
"n" is the 1-based POSITION in the table's section — the same ordinal
every view displays.

TRACES FROM AN EMBEDDING GO PROGRAM
-----------------------------------

A host application that embeds the engine (see ` + "`dtrules docs embedding`" + `)
produces the same debugger-ready traces the CLI does — the staking system
is the reference example. The wiring, in order:

    import (
        "github.com/DTRules/DTRules/pkg/dtrules/interpreter"
        "github.com/DTRules/DTRules/pkg/dtrules/mapping"
        "github.com/DTRules/DTRules/pkg/dtrules/operators"
        "github.com/DTRules/DTRules/pkg/dtrules/session"
        "github.com/DTRules/DTRules/pkg/dtrules/trace"
    )

    // 1. Load rules (files, embed.FS, whatever the host uses).
    rs := session.NewRuleSet("Staking")
    rs.LoadEDD(eddReader)
    rs.LoadDecisionTables(dtReader)

    // 2. Fresh session per run.
    sess, _ := session.NewSession(rs)
    state := sess.GetState().(*interpreter.DTState)
    state.SetOperatorTable(operators.GetOperatorTable())

    // 3. Tracing — BEFORE loading data, so the input values are
    //    recorded as def events. A trace without its initial data
    //    cannot be replayed or verified.
    f, _ := os.Create("traces/period-184.trace.xml")
    fingerprint, _ := trace.FingerprintRules(rulesDir) // dir the XML came from
    trace.WriteHeader(f, trace.Provenance{
        DTRulesVersion:   yourEngineVersion,
        RulesFingerprint: fingerprint,
    })
    state.SetOutput(f, nil)
    state.EnableTrace()

    // 4. Load mapping + input data (now recorded into the trace).
    m := mapping.NewMapping(sess)
    m.LoadMapping(mapReader)
    m.LoadDataAndPushSingletons(inputReader)

    // 5. Execute, then close out the trace.
    execErr := sess.(*session.RSession).Execute("Staking_Distribution")
    trace.WriteFinalState(f, state)
    trace.WriteFooter(f)
    f.Close()
    // On execErr the partial trace is still valuable post-mortem data.

Notes:

  - Fingerprint: compute it from the same XML directory the rules were
    loaded from. When rules ship inside the binary via go:embed, compute
    it at build time (or from the source tree) and pass it through; an
    empty fingerprint still verifies end-state but shows "fingerprint
    unknown" in the trust strip.
  - Tracing costs one buffered write per event — enable it per run
    (a flag, a failed-run retry, a weekly audit run), not permanently,
    for hot paths.
  - Keep the trace next to the inputs that produced it; the pair is the
    reproducible artifact.

Once the file exists, everything in this document applies to it:

    dtrules edit <project-with-those-rules>    # Debug tab -> load the trace
    dtrules report <trace> --spec <spec.json>  # outcome tables, --json for machines

For AI agents and scripts: traces are plain XML (format documented above),
` + "`dtrules report --json`" + ` emits machine-readable outcomes, and the editor's
HTTP API exposes /api/debug/load, /position, /find (field provenance with
the why-chain), /report, and /speculate — everything the UI does. This is
how an agent can debug and validate decision tables against a recorded
production run without touching the host system.

SERVER NOTES
------------

dtrules edit --host 0.0.0.0 --read-only serves a review instance; the
debug endpoints work in read-only mode (replay touches only in-memory
sandbox sessions, and speculation writes only to scratch overlays). A
debug session is shared by all viewers of a server — on a call, whoever
clicks drives.

Related topics: ` + "`dtrules docs workflow`" + `, ` + "`dtrules docs decision-tables`" + `,
` + "`dtrules docs mapping`" + `, ` + "`dtrules docs el`" + `
`
