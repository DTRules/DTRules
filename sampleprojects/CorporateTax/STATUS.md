# CorporateTax — status

Scoped 2026-08-02 as the last project in the sample repair campaign (#948).
This file exists so the scoping does not have to be redone. **Read it before
touching this project.**

## Where it stands

| | |
|---|---|
| XML files that parse | **all of them** (111; the corrupt core was removed) |
| Does the rule set load? | **yes — 19 entities, 164 decision tables.** First time ever |
| `xml/states/*_corp_dt.xml` | 52 files, **167 tables**, 757 rows with content, only 3 stubs — in the supported format |
| …of those rows, hand-coded postfix | **413 of 757 (55%)** |
| `xml/CorporateTax_dt.xml` (merged) | **164 tables, 0 stubs** — was 24/23 until the merge bug was fixed |
| `xml/CorporateTax_dt_core.xml` | **removed** — never parsed, unsupported schema |
| `DTRules.xml` | present; **no `<entry>` yet** — needs the orchestrator decision |
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

## The merge had never run past its first file

`merge-states.sh` incremented its counter with `((count++))`, which evaluates to
the *pre-increment* value — so on the very first state file it returned 0, a
non-zero exit status, and `set -e` aborted the script. Every run it had ever
had died after one state.

That is the whole explanation for two things that looked like separate
mysteries: the merged DT holding 24 tables (the core's) instead of 164, and the
merged EDD missing its closing tag (the script never reached the line that
writes it). The "26 stub tables" in the original campaign inventory was this
bug, not missing content.

Fixed, along with making the federal core optional so the merge can run without
one. The merged file now carries all 164 state tables and no stubs.

## What has been repaired

Mechanical only — the file-level damage that stopped any tool from reading the
project:

- **130 unescaped `&`** (from `E&P` throughout the international-tax fields).
- **8 unescaped `<`** (`gross receipts <= $1M`, `<20% DRD pool`).
- **3 `<field>` elements split by pasted blocks.** These looked like truncation
  but were not: the element's opening line and its `comment=…/>` close were both
  intact with an unrelated block of 60–80 lines wedged between them. Rejoined,
  nothing deleted.
- **The merged EDD's missing `</entity_dictionary>`** — a symptom of the merge
  bug above, now produced correctly by the script.
- **6 prose action rows marked as comments.** Each held an English sentence in
  `<action_dsl>` against a comment-only `<action_postfix>` ("Apportionment will
  be calculated by tables 7000-7500"), which the loader reads as EL somebody
  forgot to compile and refuses the file over. In HI, NJ, NY, OR.

No decision-table logic was changed.

## The hand-coded postfix is not an oracle

The postfix predates the authoring API. Written directly, never compiled from
anything, and — critically — **never executed**, so nothing ever forced it to be
correct. A census of the 413 hand rows:

| | rows |
|---|---|
| operators all real — postfix *is* an exact oracle | **253** |
| uses operators that do not exist, or `xdef` with operands reversed | **145** |
| comment-only: no executable code at all | **15** |

`add` (78 uses), `sub` (132) and `mul` (69) are not registered operators;
`xdef` appears 258 times with its operands the wrong way round
(`ga_tax apportionment.state_tax xdef` instead of
`ga_tax cvd /apportionment.state_tax xdef`). 39 states are affected.

Even among the 253 "real operator" rows the postfix is not trustworthy as
*behaviour*: Alaska's tax calculation is test-first `ifelse`, which the Go
runtime cannot execute correctly (#943/#947), and untyped `-` is used on
double operands throughout.

So the method is: **read the postfix as intent, write the EL, and recompile.**
Where the operators are real, a byte-identical recompile confirms the reading.
Where they are not, the new postfix will and should differ — the original never
ran. `tools/elcheck/derive_from_postfix.py` mechanises the repeating shapes;
`tools/elcheck` is the check.

Worked example, Alaska — 7 of 8 rows recompile byte-identical, and the eighth
is a correction:

```
stored: … f> { then } { else } ifelse     test first — never executed on Go
got:    { then } { else } … f> ifelse     test last — correct, and f- not -
```

## 504 references to fields the EDD never declares

The state tables read and write 190 distinct fields that are declared nowhere,
504 references in all — `co_refund_or_owed`, `has_physical_presence_co`,
`federal_tax_liability`. Writing to an undeclared field is a runtime error, so
these tables cannot execute regardless of how good their EL is.

That number was 304 fields / 910 references until the EDD merge was fixed (see
below); the remainder is genuine content that was never authored.

## A second merge bug: the EDD root spelling

`merge-states.sh` stripped each file's root element by name, using
`entity_dictionary`. The core EDD spells it that way — but **every state EDD
spells it `entity_data_dictionary`**, so the pattern never matched, and
`sed '1,/pattern/d'` with no delimiter deletes the file to its end.

**173 of 240 state field declarations were dropped on every merge.** That is
where most of the undeclared-field references came from. Fixed by accepting
either spelling and emitting the canonical `entity_data_dictionary`, which is
also the root the loader looks for first. Merged fields went 1170 → 1340, state
fields present 67 → 237.

## The authoring debt, measured

`go run ./tools/elcheck -project sampleprojects/CorporateTax -exclude CorporateTax_dt.xml`

```
TOTAL ok=176 prose=30 resolved=0 hand=413 diff=0 err=138
```

**551 rows need authoring: 413 carry postfix with no DSL, 138 more have DSL
that no longer compiles.** Both kinds are emptied by `syncToXML`, so:

> **The authoring API cannot be used on this project yet.** Any `table put`, or
> any patch — including a rename — regenerates every postfix in that table from
> its DSL and deletes those rows. A table becomes safe to edit only once every
> one of its rows is resolved.

That is the gate on all remaining work, including the table renames the
orchestrator would want.

Feasibility is measured, not guessed. Every one of the 413 hand rows carries a
comment shaped `<description>; <the EL>`, so candidates can be bulk-extracted
(`tools/elcheck/seed_from_comments.py`). Of 169 candidates that survive a
prose filter: **65 compile byte-identical to the stored postfix**, 52 compile
but differ, 52 do not compile. The remaining 244 hand rows have prose-only
comments and need their EL derived from the postfix — repetitive work, since
the patterns are few (`0.0 cvd /apportionment.state_tax xdef …`), but manual.

## The federal core, and why it was removed

`CorporateTax_dt_core.xml` was removed rather than repaired. Three things were
true of it at once:

1. It was written in a `<rule>` / `<actions><action>` / `<policy>` schema. **The
   loader has no support for that format** — grep `xml:"rule` in
   `pkg/dtrules/loader`; there is nothing. Only 2 of 54 files used it (this one
   and `states/TEMPLATE_corp_dt.xml`, which is authoring documentation and
   excluded from the merge); the other 51 use the standard `<*_details>` form.
2. It was structurally corrupt: 91 `<decision_table` opens against 89 closes,
   274 `<rule ` against 273 `</rule>`, and the Table 15000 block's opening tags
   overwritten by a comment. The lost content is in no revision.
3. Nothing referenced it but `scripts/merge-states.sh`.

Removing it costs nothing that worked: no state table performs any other table,
and only two state rows reference a federal result at all. It is recoverable
from git if that judgement turns out wrong.

## What is still open

**1. The entry point.** There is no `<entry>` and no orchestrator. The shape is
clear — dispatch on `apportionment.state_code` with
`perform table named ("Determine_" + apportionment.state_code + "_Filing_Requirement")`,
a supported and tested EL form — but two content decisions come first:

- **Naming.** 36 states use `Determine_XX_Filing_Requirement` /
  `Calculate_XX_Income_Adjustments` / `Calculate_XX_State_Tax`; 15 (CO IA ID IN
  KS KY LA MA MD ME MI MN MO MS MT — clearly one authoring batch) use
  `Determine_XX_Corporate_Filing_Requirement` and friends. Dynamic dispatch
  needs one convention. Renaming is safe in itself — nothing performs anything
  — but it goes through the authoring API, which is gated on the debt above.
- **States with no corporate income tax.** NV, SD, WA and WY have no
  equivalent table trio and need an agreed shape.

**2. The federal computation.** Form 1120 itself is gone with the core. The
states apportion from federal taxable income, which now has to come from the
input rather than being computed. Whether to rebuild it is a scope question,
not a repair.

**3. Whether the project stays at all.** The alternative remains the
DTEligibility route (#959/#960). Against it: unlike DTEligibility this was
deliberate, and 164 tables of genuine work now load.

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
