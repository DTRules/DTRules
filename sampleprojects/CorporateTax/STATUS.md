# CorporateTax — status

Scoped 2026-08-02 as the last project in the sample repair campaign (#948).
This file exists so the scoping does not have to be redone. **Read it before
touching this project.**

## Where it stands

| | |
|---|---|
| XML files that parse | **111 of 112** |
| `xml/states/*_corp_dt.xml` | 52 files, **167 tables**, 757 rows with content, only 3 stubs — in the supported format |
| …of those rows, hand-coded postfix | **413 of 757 (55%)** |
| `xml/CorporateTax_dt.xml` (merged) | 24 tables, **23 of them stubs** — the merge never picked up the state content |
| `xml/CorporateTax_dt_core.xml` | **does not parse**, and is in a format the loader cannot read |
| `DTRules.xml` | **does not exist** — no project marker, no `<entry>` |
| Excel authoring source | **none** — the only sample with no `edd/` or `DecisionTables/` |
| Reference material | `reference/` — 76 official state forms and instructions, committed |

## Nothing here has ever loaded

`CorporateTax_dt_core.xml` fails to parse in **every commit since it was
introduced** — all five revisions checked. `CorporateTax_edd.xml` parsed only
in the first Phase 1 commit and broke on the next one. There is nothing to
recover from history; the content that is missing was never committed.

Meanwhile the tests pass. They read a small Phase-1-era `repository/` mirror,
hit **23 decision-table load errors**, swallow them as
`"Note: Decision table loading had errors (simplified format)"`, and skip every
execution test. `TestCorporateTaxAllScenarios` finds **0 scenarios** and reports
`Total: 0, Passed: 0, Failed: 0` — passing. Alongside sit `PHASE1_COMPLETE.md`,
`PHASE2_COMPLETE.md`, `FINAL_STATUS.md` and three more files attesting to
completion of a 30-issue, three-phase effort (#316–#332).

This is the campaign's pattern at its limit: not one test executing nothing,
but an entire feature programme whose primary artifact the engine has never
been able to read.

## What has been repaired

Mechanical only — the file-level damage that stopped any tool from reading the
project:

- **130 unescaped `&`** (from `E&P` throughout the international-tax fields).
- **8 unescaped `<`** (`gross receipts <= $1M`, `<20% DRD pool`).
- **3 `<field>` elements split by pasted blocks.** These looked like truncation
  but were not: the element's opening line and its `comment=…/>` close were both
  intact with an unrelated block of 60–80 lines wedged between them. Rejoined,
  nothing deleted.
- **The merged EDD's missing `</entity_dictionary>`.**

No rule content was changed. No decision-table logic was touched.

## What is still open — needs a decision

`CorporateTax_dt_core.xml` is the federal core, and three things are true of it
at once:

1. It is written in a `<rule>` / `<actions><action>` / `<policy>` schema. **The
   loader has no support for this format** — grep for `xml:"rule` in
   `pkg/dtrules/loader`; there is nothing. Only 2 of 54 files use it (this one
   and `states/TEMPLATE_corp_dt.xml`); the other 51 use the standard
   `<*_details>` form.
2. It is structurally corrupt: 91 `<decision_table` opens against 89 closes,
   274 `<rule ` against 273 `</rule>`, and the Table 15000 block's opening tags
   were overwritten by a comment. The lost content is in no revision.
3. It is referenced by nothing except `scripts/merge-states.sh`.

**The recommendation is: keep the state tier, drop the federal core.** Delete
`CorporateTax_dt_core.xml`, write a real `DTRules.xml` with an entry table, fix
the merge so the 167 state tables actually reach the merged file, then
transcribe the 413 hand-coded rows the way SyntaxTests' 48 were. That last part
is the bulk of the work and needs a federal core rebuilt in the supported
format to hang the states off — which is new content, not repair.

The alternative is the DTEligibility route (#959/#960): delete the project.
Against that, unlike DTEligibility this was deliberate and there are 167 tables
of genuine work in the right format.

**Paul's call.** Nothing beyond the mechanical repairs should proceed until it
is made.

## Traps

- **`table get` renumbers rows to their position on load.** A patch keyed by the
  number written in the XML lands on whatever row now holds that number — one
  row off wherever the stored numbering has a gap. Key authoring patches by
  position. This silently overwrote two live rows in SyntaxTests before a diff
  caught it.
- **Diff every recompile.** All three near-misses in this campaign were
  invisible in tool output and showed only in the diff.
- The status files (`PHASE1_COMPLETE.md` and friends) describe intent, not
  state. Do not trust them.
