# CorporateTax — plan to clear the blocked work

Companion to [STATUS.md](STATUS.md), which records what was found and why.
This is what to do about it, in the order it has to happen.

## Two decisions first

**D1 — Does CorporateTax stay?** Everything below assumes yes. The case for:
164 tables load, the state tier is in the supported schema, and the reference
material is now committed. The case against: no federal computation survives,
190 fields were never declared, 551 rows need authoring, and no test has ever
executed it. The comparable precedent is DTEligibility, which was deleted
(#959/#960) — but that had never been intended, and this was.

*If the answer is no, stop here.* Phase 0 still lands (the engine fixes and the
tooling are independent of this project), and the rest is `git rm`.

**D2 — Do the two finished branches get PRs now?** `fix/syntaxtests-el-948` is
complete and independent. `fix/corporatetax-parse-and-reference` is a coherent
increment — the project went from unparseable to loading — and holding it back
until the whole campaign finishes makes the eventual diff much harder to
review. Recommend merging both now via `scripts/merge-pr.sh`.

## Sequencing, and why this order

The EL is the visible debt, but it is not the first constraint. Writing to a
field the EDD does not declare is a runtime error, so **perfecting the EL of a
table whose fields do not exist produces expressions that fail on their first
assignment.** The EDD comes first. It is also independent: `SaveEDD` writes EDD
XML only and never recompiles a decision table, so unlike `table put` it is
safe to use today.

```
Phase 1  EDD          190 fields          unblocks execution      ← start here
Phase 2  EL           551 rows            unblocks the API
Phase 3  Entry point  1 table + decisions unblocks run/trace
Phase 4  Enforcement  tests               makes it stay fixed
```

---

## Phase 1 — Complete the EDD (190 fields)

Mechanical, and smaller than the number suggests. 165 of the 190 are
`<state>_`-prefixed and follow only **28 distinct suffixes**; 490 of the 504
references are on the `result` entity; only **21 states** are affected.

| suffix | states missing it |
|---|---|
| `gross_receipts`, `apportioned_income`, `estimated_payments`, `refund_or_owed`, `municipal_interest` | 15 each |
| `us_interest_income` | 14 |
| `has_nexus`, `tax_liability`, `taxable_income` | 8 each |
| `economic_nexus_threshold`, `state_additions`, `state_subtractions` | 7 each |

Worst states: MT 54 refs, ME 52, MS 47, MA 43, MO 42, MI 41, MN 41, MD 30.

**How.** Add the fields to the per-state `xml/states/XX_corp_edd.xml` source
files, not to the merged `CorporateTax_edd.xml` — the merged file is generated
and the next `merge-states.sh` would discard the edit. Type them from the
suffix (`_income`, `_tax`, `_receipts`, `_payments` → `double`; `has_` →
`boolean`; `_threshold` → `double`), matching the declarations the states that
*do* have them already use, so the compiler emits `cvd` rather than falling
back to `cvi`.

**Verify.** Re-run the undeclared-field count (the script is in STATUS.md's
history) until it reaches zero, then `./scripts/merge-states.sh` and confirm the
rule set still loads with 164 tables.

**Size.** One scripted pass plus review. The types are inferable but should be
eyeballed per state against `reference/forms/<STATE>/` — that is what the forms
were downloaded for.

---

## Phase 2 — Author the EL (551 rows)

413 rows carry postfix with no DSL; 138 more have DSL that no longer compiles.

**The postfix is intent, not an oracle.** 145 of the 413 use operators that do
not exist (`sub` ×132, `add` ×78, `mul` ×69) or have `xdef` operands reversed;
15 have no code at all. For those, a recompile *should* produce different bytes
— the original never ran. For the 253 whose operators are all real, a
byte-identical recompile is the proof the reading was right.

**The loop**, per state:

```bash
python3 tools/elcheck/derive_from_postfix.py sampleprojects/CorporateTax cand.json
go run ./tools/elcheck -project sampleprojects/CorporateTax \
    -exclude CorporateTax_dt.xml -overrides cand.json
```

`RESOLVED` = byte-identical, accept. `DIFF` = read it: the expected shapes are
`>` becoming the type-correct `f>`, and test-first `ifelse` becoming test-last
(#943). Anything else is a mistranslation. `HAND`/`ERR` = not yet derived; extend
the rule set in `derive_from_postfix.py` or write the row by hand.

**The gate.** A table can only be written once **every** one of its rows is
resolved — `syncToXML` regenerates all postfix from DSL, so one unresolved row
in a table is one deleted row. Work table-by-table; apply with a whole-table
`table put`; key patches by **row position**, never the number in the XML.

**After each apply**, diff positionally against the original: same row count in
and out, no row that had postfix left without it. Every near-miss in this
campaign was invisible in tool output and showed only in the diff.

**Order.** Start with the states whose rows are already fully derived (cheapest
proof the loop works end to end), then the 253-row "real operator" group where
byte-identical matching gives certainty, then the 145 fabricated rows where
judgement is needed and the state forms in `reference/` are the authority.

**Size.** The largest remaining item by far. 127 of 413 derive today with 64
byte-identical; the tail needs rule extensions and hand work.

---

## Phase 3 — Entry point and execution

Two content decisions, then one small table:

- **Naming.** 36 states use `Determine_XX_Filing_Requirement` /
  `Calculate_XX_Income_Adjustments` / `Calculate_XX_State_Tax`; 15 (CO IA ID IN
  KS KY LA MA MD ME MI MN MO MS MT) use the `_Corporate_` spelling. Dynamic
  dispatch needs one. Renaming is safe in itself — nothing performs anything —
  but it goes through the authoring API, so it is gated behind Phase 2.
- **States with no corporate income tax.** NV, SD, WA, WY have no equivalent
  trio and need an agreed shape (most likely a `Determine_XX_...` that sets tax
  to zero, which is what the template already suggests).

Then the orchestrator, in the supported schema:

```
context:   (none — flat singletons)
condition: apportionment.state_code is not null
actions:   perform table named ("Determine_"  + apportionment.state_code + "_Filing_Requirement")
           perform table named ("Calculate_"  + apportionment.state_code + "_Income_Adjustments")
           perform table named ("Calculate_"  + apportionment.state_code + "_State_Tax")
```

`perform table named (...)` is a supported, tested EL form. Add `<entry>` to
`DTRules.xml`, and a scenario under `testfiles/`. The map also needs checking —
it has `<setattribute>` entries but no `<entities>` or `<initialization>`
block, which is how Poker and SinusitisTherapy loaded empty.

**Note the federal gap.** Form 1120 went with the removed core. The state
tables apportion from federal taxable income, which now has to arrive as input
rather than being computed. Rebuilding it is a separate scope question.

---

## Phase 4 — Make it stay fixed

- Add CorporateTax to `TestSampleProjectsProduceLoadableTraces` with a
  fired-column floor. This is the check that catches a run that loads
  everything and decides nothing — the failure mode six of six projects had.
- Un-skip the CorporateTax tests in `pkg/dtrules/corporate_tax_test.go`. They
  currently pass while swallowing 23 load errors as "simplified format" and
  running zero scenarios; that must not survive.
- Delete `sampleprojects/CorporateTax/repository/` per the campaign's scope
  decision and repoint those tests at `xml/`.
- Add an `elcheck` gate so `hand` and `err` cannot regress above zero once
  Phase 2 lands.

---

## Unrelated loose end

Seven reference documents could not be fetched: MA and NH return 403 to
scripted requests, MS fails TLS chain verification. The URLs in
`reference/sources.tsv` are correct — they need a browser, not a code change.
SD, WY (no corporate income tax) and OH (e-file only) have nothing to fetch.
