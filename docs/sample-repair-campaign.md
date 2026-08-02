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
| **SyntaxTests** | **remaining** — #975 |
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
(#974). Both were mine.

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

### SyntaxTests (#975)

23 tables: 48 hand rows, 30 prose rows, 3 stub tables, plus an EDD type
conflict (#783).

The 48 are mostly reverse iteration, and since #978 they have an authorable
form producing byte-identical postfix:

```
for all clients in reverse { set eligible = true; }
  ->  { true cvb /eligible xdef } clients forallr
```

Transcribing them is mechanical: compile, diff, apply on exact match. The 30
prose rows need reading individually. **Do not run the recompile recipe here** —
those 48 rows are exactly what it deletes.

Two gotchas: statements inside a block need their own semicolons
(`{ set x = 1; }`), and the inventory scanner reports rows referencing context
locals as "stale" because it compiles each row independently. Those are false
positives.

### CorporateTax

Never scoped. From the initial inventory: 26 stub tables; the EDD uses root
element `<entity_dictionary>` where the loader expects
`<entity_data_dictionary>`; `*_dt_core.xml` and `states/DC_corp_dt.xml` have
XML syntax errors; `CorporateTax_edd.xml` has an unescaped `<` inside a quoted
string. Its `PHASE1_COMPLETE.md`-style status files come from a real feature
effort (#316–#320), not an accident.

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
