# Sample Repair Campaign — Plan of Record

*Started 2026-07-28. Paul's directive: "Fix all the sample projects; remove all
hand crafted postfix, and flush out all stubbed tables. All this work should be
done using the authoring API of DTRules." Later: "goal is to make all sample
projects work, demonstrate DTRules, with traces that can be loaded into an
editor."*

Parent issue: **#948** — **closed 2026-08-07**. One sub-issue per project, one
PR per project, merged only via `scripts/merge-pr.sh`. `make check` green is the
definition of done.

What remains open is listed under "Still open" at the end; none of it is repair
work.

## Definition of done

A sample is repaired when all four hold:

1. **No hand-coded postfix** — every row's postfix compiles from its EL DSL.
2. **No stub tables** — every table has real, executable content.
3. **Excel and XML agree** — all edits through `dtrules table put/patch`,
   `dtrules edd put`, or the `dtrules build` funnel. Never hand-edit rule XML.
4. **It runs and leaves a trace** — `dtrules run --entry <t> --input <scenario>
   --trace <file>` succeeds, the trace loads, and it records real fired
   columns. Enforced by `TestSampleProjectsProduceLoadableTraces`.

Point 4 was added late and matters most. See "The pattern".

## Status

| Project | State |
|---|---|
| TaxReturn | done before this campaign (#520, PRs #933–#938); expected-value drift closed by #935 / #998 |
| TestProject | done — #950 / #951 |
| StateTax | done — #952 / #953, all 51 state scenarios compute correct tax |
| Poker | done — #954 / #955, #958, 12 documented decisions checked |
| SinusitisTherapy | done — #963 / #964 |
| CHIP | done — #962 / #969, #970, #997 |
| ChipApp | done — #971 |
| KidAid | done — #972 |
| KidAid_Application | done — #973 |
| DTEligibility | deleted — #959 / #960 |
| SyntaxTests | done — #975; no hand postfix, runs, traces |
| CorporateTax | done — #986 / #988, #992; executes, CA scenario computes correct tax |

DTEligibility was deleted rather than repaired. It arrived on 2026-02-05 inside
a commit titled "docs: Consolidate documentation into /docs directory", unnamed
in that commit message, and had never worked: its map used a format the Go
loader has never supported and its postfix a calling convention predating the
Go compiler. Nothing executed it. **Check provenance
(`git log --diff-filter=A -- <path>`) before assuming a sample was intended.**

## The pattern

**Every project examined had a test that passed while executing nothing.** Six
for six, a different cause each time:

| project | why it executed nothing |
|---|---|
| TestProject | test read a stale `repository/` mirror whose map lacked its `createentity` lines |
| StateTax | test logged its own runtime failure as "may be expected" |
| Poker | map never attached players to `game.players` |
| SinusitisTherapy | map never declared the `<patient>` root, so the singleton loaded empty |
| CHIP | same as SinusitisTherapy |
| KidAid | tables loaded typeless after a recompile erased `<TYPE>` |
| SyntaxTests | test skipped as "archived"; no `<entry>`, no orchestrator, and 312 initial actions the loader could not see |

A run that does nothing still exits 0 and still writes a well-formed trace. The
**fired-column floor** in `TestSampleProjectsProduceLoadableTraces` is the only
check that catches all six. Every sample added to that test gets one.

## Method

Rule edits go through the authoring API — `dtrules table put/patch`,
`dtrules edd put` — or the `dtrules build` funnel. For bulk work, script
against the API, not the XML.

**Authoring a hand-coded row.** Read the stored postfix, write the EL you
believe produces it, compile it, diff against what was there, and apply only on
a match. `dtrules docs operators` carries a postfix→EL reverse index. This is
how StateTax's and Poker's rows were done, and it is the only safe way — #974
is what happens without it.

**Recompiling a project** (how a project picks up compiler fixes): re-apply a
row's own DSL through `update-condition-dsl` / `update-action-dsl`, which
re-syncs the whole table. Two hazards:

- **`set-policy` does not recompile.** It reports success and re-syncs
  nothing; only the real mutators call `syncToXML`.
- **It deletes hand-coded rows.** `syncToXML` regenerates every postfix from
  its DSL, so a row whose only content *is* postfix comes back empty. Check
  `Table.HandCodedRows()` first and author the EL for anything it names.

**Always diff before committing a recompile.** `+<Type></Type>` caught the
type-erasure bug (#972); missing `<*_postfix>` content caught the row deletion
(#974); an emptied `<initial_actions>` caught 312 rows being deleted in
SyntaxTests. All three were mine, and all three were invisible in the tool
output — only the diff showed them.

**Key rows by position, not by number.** `table get` renumbers every row to its
position on load. A patch keyed by the number written in the XML lands on
whatever row now holds that number, which is one row off wherever the stored
numbering had a gap. That silently overwrote two live rows in SyntaxTests before
the diff caught it. Verify each patch against the row it claims to target —
a hand row should currently be empty, a repair should hold the exact broken DSL.

## Scope decisions (Paul, 2026-07-28)

- Legacy `repository/xml/` mirrors get deleted; each project single-sources on
  `xml/` and tests reading the mirror are repointed. **Exception:**
  `KidAid_Application/repository` *is* the project.
- `lib/*.jar` and `.classpath` get deleted. `DecisionTables/*.xls` and
  `edd/*.xls` stay as the historical authoring source.

## Engine defects this campaign uncovered

All fixed unless noted.

**Policy statements.** No runtime implementation at all (#949) — the compiler
emitted `policystatements`, ten samples authored it, nothing implemented it.
Column 1's statement was eaten on every Excel→XML build. `{expr}` interpolation
was emitted as literal braces. They were not authorable, because `syncToXML`
preserved their hand-written postfix, so #817 stopped at the table's last
section. Then, to Paul's design (#956): statements collect automatically, drain
with `add the policy statements to <array>`, reset with `clear the policy
statements` — and that add was appending the whole accumulator as one blob.

**A whole section nobody read.** `<initial_actions>` was parsed only as
`<initial_action>`, but every other section of a DT file is spelled
`<*_details>` and SyntaxTests spells this one `<initial_action_details>`
throughout. Its 312 initial-action rows were invisible: not loaded, not
compiled, not executed, unreachable from the authoring API. Worse, the writer
emitted `<initial_actions></initial_actions>` over them, so the first
`table put` deleted all 312 — caught in the diff, reverted. Reader, writer and
runtime loader now take both spellings and normalise to the canonical one.
**The same class of bug as `<TYPE>` (#972); look for it in any section whose
element name has two spellings.**

**Comment-only rows were not authorable.** `CompileCondition` skipped a row
that is only a comment; `CompileContext` and `CompileAction` did not. A table
containing one commented-out documentation row could not be written through the
authoring API at all — `table put` rejected the whole table on a parse error
for a row that was never meant to execute.

**Sync and round-trip.** `<createentity list='…'>` dropped on every map round
trip, which emptied StateTax's bracket schedules so 29 states computed zero.
Context comments erased one build at a time. Column elements written in Go map
order. The legacy `<TYPE>` spelling erased on write, leaving tables typeless
and unloadable (#972).

**Loading.** `Initialize`+`LoadData` built a second instance of any
cardinality-1 entity that also had a `createentity`, so loaded values landed on
a copy nobody held.

**Compiler and EL.** Rows compiled blind to their own table's context locals
(#965). `is the <R> of` emitted a runtime error stub after findmatch was removed
(#927) — it means only "the entity is held by that field of the other entity",
which `getrelationship` already did. `allowing array to be removed` emitted the
forward iteration that makes removal unsafe. `forallr` had no EL surface at
all, in context cells (#977) or action cells (#978). `CompileAction` reported
the wrong parse error, pointing at the entry rule instead of the real mistake.

**Tooling and docs.** `dtrules verify` rejected `{}`, so every table containing
a bare `if` failed. `docs/el-reference.md` documented postfix the compiler does
not emit in 70 of its 116 examples (#961) — now regenerated from the compiler
and gated by `docs/el_reference_postfix_test.go` against a fixture EDD the
reference owns.

## How it went, project by project

### CorporateTax (#986)

Done, 2026-08-05, all four phases of `sampleprojects/CorporateTax/PLAN.md` —
that file and `STATUS.md` carry the full record. The short version: nothing in
it had ever loaded (federal core unparseable in every commit, merge script dead
on its first state file, 173 of 240 state EDD field declarations dropped by a
root-spelling mismatch, 145 of 413 hand rows using operators that do not
exist). Now: single-sourced on `xml/states/`, every row authored in EL
(elcheck `hand=0 diff=0 err=0` across 172 tables), one naming convention with
wrappers for the four gross-receipts states, `Run_Corporate_Tax` dispatching
on `apportionment.state_code`, and a CA scenario computing 88,400 tax / 1,600
refund, enforced by trace floor + arithmetic + no-hand-postfix tests.

The hand postfix predated the authoring API and was never an oracle — it was
read as intent (Paul's framing), decompiled to EL by
`tools/elcheck/decompile_postfix.py`, verified byte-identical where its
operators were real, and the originals retained in git as reference.

Engine bugs it flushed out, both fixed with tests: the authoring `set-name`
patch was a silent no-op (view assignment never synced — `Project.RenameTable`
now exists), and `<initial_action_details>` was invisible to every reader
(312 rows in SyntaxTests; fixed on the SyntaxTests branch).

Still open, deliberately: map tags for the `result.XX_*` inputs the 15
renamed states read; scenarios for graduated and gross-receipts states (the
ME/MS/VT bracket rows are verified by compilation, not yet by execution);
the Excel-bootstrap decision (this is the only sample with no workbooks);
and whether Form 1120 is rebuilt at all.

### CHIP (#962) — resolved

The question was which of two changes to make: grow `client.parent`-style
fields on the EDD, or re-author the three `is the <R> of` rows to search the
relationship entities. **Neither would have worked**, because the relationship
entities never had endpoints to search.

CHIP's map has tags that both create an entity and set an attribute —
`<source id="1001"/>` inside a `<relationship>` creates a `client` *and* sets
`relationship.source` to it. The loader resolved the enclosing entity from the
top of the entity stack, but the tag had already pushed the client it just
created, so the attribute landed on the client and `relationship.source` /
`.target` stayed null on every relationship CHIP ever loaded. Fixed in #997 by
reading one below the top when the tag created an entity. Household size went
from 1 to 2 — it had been counting the applicant and nobody else.

**The lesson is the shape, not the bug:** a question framed as "which of these
two designs" was really "neither, because the data the designs disagree about
was never there". Check that the thing being reasoned about actually loads
before choosing between models of it.

### Two more of the pattern, found last

**A test whose input was not in the repository (#998).** `.gitignore` carried a
blanket `testfiles/`. 577 sample test files predate the rule and stayed tracked
— which is why nobody noticed, since git ignores nothing about a file already
added — but every scenario written *during this campaign* was silently never
committed: CorporateTax's CA/ME/MT, ChipApp's, KidAid_Application's. Those
tests passed on the authoring machine and could not have passed on any other
checkout. **Run `make check` in a clean worktree, not just in yours.**

**Three sets of expected values, no two agreeing (#935).** TaxReturn's
`Family_2025` figures existed hardcoded in the test *and* in the scenario, with
a third answer from the rules. The scenario's copy was even commented "updated
to match actual calculations" — it had been a capture of rule output once, and
then drifted, because the test kept its own copy. The test now reads the
scenario. Two copies of a number that must agree will not stay agreeing.

## Still open

None of this is repair work; it is scope the campaign deliberately did not take.

- **CorporateTax** — scenarios for the gross-receipts states (OH CAT, WA B&O,
  TX margin, TN franchise/excise; those wrapper paths have never fired); MA and
  MS rates, whose forms their sites blocked us from downloading; whether the
  project gets an Excel bootstrap (it is the only sample with no workbooks);
  whether Form 1120 is rebuilt at all. Full detail in
  `sampleprojects/CorporateTax/STATUS.md`.
- **TaxReturn** — the ~8.5k total-tax disagreement with the external reference
  model, and the SE-tax and QBI treatment behind it. Needs a tax reference, not
  a code change. #935 closed on the drift between copies, not on this.
- **SinusitisTherapy** — the shipped PDF is guarded against rule drift, but not
  against changes visible only in the PDF (styling, print flags).

## Tools

The read-only inventory scanner and the EL probe used throughout live in the
session scratchpad, not the repo. Rebuild as needed:

- **inventory** — walks a project and classifies every row as ok / stale /
  prose / hand / comment / empty; lists stub tables and load errors.
- **elc** — compiles candidate EL against a project's EDD symbols and diffs it
  against an expected postfix. This is the verification loop for authoring.
