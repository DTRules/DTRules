# Sample Repair Campaign — Plan of Record

*Written 2026-07-28, at the start of the work. Paul's directive: "Fix all the
sample projects; remove all hand crafted postfix, and flush out all stubbed
tables. All this work should be done using the authoring API of DTRules."
Sub-issues per project, with progress documented in each issue.*

## Goal

Every sample project loads clean, executes, and satisfies the authoring
contract: **no hand-coded postfix anywhere** (every row's postfix is compiled
from its EL DSL), **no stub tables** (every table has real, executable
content), and **Excel/XML in agreement** (all edits through the authoring
API / build funnel — never direct XML edits; see the 2026-07-26 lesson where
direct edits nearly got clobbered by a stale-workbook mtime race).

## Non-negotiable method (Paul's correction, twice)

- Rule edits go through **`dtrules table put/patch` / `dtrules edd put`**
  (writes XML + compiles postfix + updates Excel atomically), or through
  **`dtrules build`** (XML-authored path exports Excel; run `--dry-run`
  first and READ THE DIRECTION LINE — conflicts now fail loudly per #938/#946).
- For bulk work, **script against the authoring API**, not against the XML.
- Before any risky build: everything committed + tarball outside the repo.
- Branch per project, PR per project ("Closes #<sub-issue>"), merge only via
  `scripts/merge-pr.sh`. `make check` green is the definition of done.

## Worklist (from `apiserver.DiscoverProjects` — 13 projects + TaxReturn done)

| Project | Marker | Known state (pre-inventory) |
|---|---|---|
| sampleprojects/TaxReturn | DTRules.xml | ✅ DONE (#520 campaign, PRs #933–#938): 0 hand rows, Excel regenerated |
| sampleprojects/CHIP | DTRules.xml | Known hand-postfix warnings (Evaluate_MEDICAID/FOODSTAMPS_Eligibility); CHIP_dt_strip.xml is unparseable junk |
| sampleprojects/ChipApp | DTRules.xml | Unknown; likely CHIP sibling |
| sampleprojects/CorporateTax | xml/ | WORST: EDD uses wrong root element `<entity_dictionary>` (loader expects `entity_data_dictionary`); "condition table isn't balanced" load errors; `*_dt_core.xml` has XML syntax errors; state files carry test-first ifelse hand postfix (#947) |
| sampleprojects/KidAid | DTRules.xml | #927: findmatch-era rules (`relationship-is-of` uses REMOVED op) — migration needs Paul's semantics call; map Excel round-trip formerly lossy; checked-in postfix predates numeric-add fix |
| sampleprojects/KidAid_Application/repository | DTRules.xml | Unknown; nested project |
| sampleprojects/Poker | xml/ | Unknown; no DTRules.xml |
| sampleprojects/SinusitisTherapy | xml/ | Unknown; no DTRules.xml |
| sampleprojects/StateTax | DTRules.xml | Unknown |
| sampleprojects/SyntaxTests | DTRules.xml (+ nested repository/) | Test-first ifelse hand postfix (#947); `syntaxexample_edd.xml` has type conflict (totalIncome integer vs double) |
| sampleprojects/TestProject | DTRules.xml | Unknown; probably small |
| cmd/sinusitis-web/rules | xml/ | Embedded project; also a candidate to relocate under sampleprojects/ |

Projects without DTRules.xml get one created (one per project — it is the
project marker; #941/#942 established this model).

## Execution order

Small/easy first to bed the workflow in, hardest last:
TestProject → StateTax → Poker → SinusitisTherapy (+
cmd/sinusitis-web/rules) → SyntaxTests → CHIP → ChipApp →
KidAid_Application → KidAid (partial; #927 items need Paul) → CorporateTax.

## Per-project procedure

1. **Inventory** (read-only scanner, scratchpad `inventory/main.go` — has a
   syntax error at line 36 to fix: `if d := ...; st, err := os.Stat(d)` is
   illegal Go; restructure): classify every row as ok / stale (DSL compiles,
   differs from stored postfix) / prose (DSL doesn't compile) / hand
   (postfix without DSL) / empty; list stub tables and load errors.
2. **File the sub-issue** with the findings table ("Part of the sample repair
   campaign; parent #947 for ifelse rows"). Keep updating it as work lands.
3. **Repair via authoring API**:
   - stale rows → `dtrules build` recompiles (its import step compiles all DSL);
   - prose/hand rows → author EL from the stored postfix (reverse index in
     `dtrules docs operators` maps postfix ops → EL phrases), applied with
     `dtrules table put/patch`;
   - stub tables → implement real logic (domain judgment — document the
     chosen semantics in the sub-issue);
   - broken EDDs (CorporateTax root element, SyntaxTests type conflict) →
     fix via `dtrules edd put` where possible;
   - create DTRules.xml for the xml/-convention projects.
4. **Verify**: project's own tests if any (`sampleprojects_test.go`,
   corporate_tax_test etc. — some behind `-tags archive`), then `make check`.
5. **PR** ("Closes #<sub-issue>"), merge via `scripts/merge-pr.sh`, post the
   outcome summary as a final issue comment.

## Landmines (learned the hard way this session)

- `ifelse`/`if` pop the TEST from the TOP; legacy test-first postfix fails at
  runtime (#943). Recompiling from DSL fixes it — that's the point.
- A multi-entity EDD without `<file_metadata><file_path>` is SKIPPED by the
  directory loader (#946) — regenerated EDDs must carry it (fixed in the
  writer, but watch for it in hand-repairs).
- Map workbooks have no sync support (#929) — sync skips `*_map.xlsx`.
- Comment-only `/* ... */` DSL rows are valid no-ops (since #933/#944).
- The excel package's `EffectiveDSL()` does NOT fall back to
  `action_description` but the runtime loader's `GetDSL()` DOES — a scanner
  must check all three fields or it under-reports prose rows (this bit the
  TaxReturn campaign: ~20 rows hid until the build's drop report).
- KidAid: do the mechanical parts; `relationship-is-of`/findmatch semantics
  are Paul's call (#927) — document, don't guess.

## State at handoff

- Everything through PR #941 is merged; main = `ef7dca44`; installed
  `~/go/bin/dtrules` = v1.21.0-7. Working tree clean.
- Issues: #942–#946 filed+closed (retrospective); #947 OPEN (ifelse legacy
  samples — this campaign supersedes/absorbs it); #935 OPEN (Family_2025
  expected values — Paul's decision); #927–#931 open backlog.
- Task list: task #8 "Inventory all sample projects" created, in progress.
- Nothing for this campaign has been committed yet; the only artifact is the
  scratchpad inventory scanner (broken, see above) and this plan.

## Recompiling a project (learned the hard way, 2026-07-29)

Re-applying a row's own DSL through `dtrules table patch update-condition-dsl`
/ `update-action-dsl` recompiles the WHOLE table — that is how a project picks
up compiler fixes. Two things to know before doing it:

- **`set-policy` does not recompile.** It reports success and re-syncs
  nothing; only the real mutators call `syncToXML`.
- **It deletes hand-coded rows.** `syncToXML` regenerates every postfix from
  its DSL, so a row whose only content IS postfix comes back empty. Check
  `Table.HandCodedRows()` first and author the EL for anything it names —
  read the stored postfix, write the DSL, confirm it compiles to the same
  thing. CHIP, ChipApp, KidAid and KidAid_Application each lost rows this way
  before anyone noticed, because the emptied rows were in no scenario.

Always diff before committing a recompile. `+<Type></Type>` is what caught
the type-erasure bug; missing `<*_postfix>` content is what catches this one.
