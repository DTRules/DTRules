# Sample Repair Campaign — Plan of Record

*Started 2026-07-28. Paul's directive: "Fix all the sample projects; remove all
hand crafted postfix, and flush out all stubbed tables. All this work should be
done using the authoring API of DTRules." Later: "goal is to make all sample
projects work, demonstrate DTRules, with traces that can be loaded into an
editor."*

Parent issue: **#948**. One sub-issue per project, one PR per project, merged
only via `scripts/merge-pr.sh`. `make check` green is the definition of done.

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
| TaxReturn | done before this campaign (#520, PRs #933–#938) |
| TestProject | done — #950 / #951 |
| StateTax | done — #952 / #953, all 51 state scenarios compute correct tax |
| Poker | done — #954 / #955, #958, 12 documented decisions checked |
| SinusitisTherapy | done — #963 / #964 |
| CHIP | done — #962 / #969, #970, one open data-model question below |
| ChipApp | done — #971 |
| KidAid | done — #972 |
| KidAid_Application | done — #973 |
| DTEligibility | deleted — #959 / #960 |
| SyntaxTests | done — #975; no hand postfix, runs, traces |
| **CorporateTax** | **remaining** — never scoped |

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

## Remaining work

### CorporateTax

Scoped 2026-08-02. **Full findings are in
`sampleprojects/CorporateTax/STATUS.md` — read that, not this summary, before
touching the project.** The mechanical repairs are done; one scope decision is
open.

Nothing in this project has ever loaded. `CorporateTax_dt_core.xml` fails to
parse in every commit since it was introduced; `CorporateTax_edd.xml` parsed
only in the first Phase 1 commit. The "26 stub tables" in the earlier inventory
were an artifact of reading the merged file, which holds 24 tables (23 stubs)
because the merge never picked up the state tier — where the real work is: 52
files, 167 tables, 757 rows, 3 stubs, in the supported format.

Its tests pass by reading a Phase-1-era `repository/` mirror, swallowing 23
load errors as "simplified format", and running zero scenarios —
`Total: 0, Passed: 0, Failed: 0`.

Repaired: 130 unescaped `&`, 8 unescaped `<`, 3 `<field>` elements split by
pasted blocks, and the merged EDD's missing root close. 111 of 112 files parse.

Open: `CorporateTax_dt_core.xml` is in a `<rule>`/`<policy>` schema **the loader
has no support for**, is corrupt beyond what any revision holds, and is
referenced by nothing but `merge-states.sh`. Recommendation is to drop it and
keep the state tier; the alternative is the DTEligibility route. Paul's call.

Reference material for this work — 76 official state forms, instructions and
published regulations across 43 jurisdictions — is committed under
`sampleprojects/CorporateTax/reference/`, with a sha256 manifest and a refetch
script. It is committed rather than gitignored because state form URLs rotate
every filing season.

### CHIP's open question (#962)

CHIP executes, but its EDD models relationships as a separate `relationship`
entity — `source` / `target` / `type`, its own field comment reading *"Source
is the &lt;relationship type&gt; of the target"* — not as `client.parent`
fields. Its three `is the <R> of` rows compile and execute correctly under the
stated semantics but evaluate false against that data shape. Either the EDD
grows the fields or those rows are re-authored to search the relationship
entities. Paul's call.

## Tools

The read-only inventory scanner and the EL probe used throughout live in the
session scratchpad, not the repo. Rebuild as needed:

- **inventory** — walks a project and classifies every row as ok / stale /
  prose / hand / comment / empty; lists stub tables and load errors.
- **elc** — compiles candidate EL against a project's EDD symbols and diffs it
  against an expected postfix. This is the verification loop for authoring.
