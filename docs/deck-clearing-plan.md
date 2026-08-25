# Deck-clearing plan: the eight open issues

Goal: every issue open on 2026-08-25 either merged-and-closed or closed with
a stated reason. Sequenced cheapest-verification-first, so the board shrinks
early and the expensive work is done against a short list.

Status of record. Update as phases land.

| # | Title | Disposition | Phase |
|---|---|---|---|
| 443 | EL expressions: Northeast states | **Verify → close** | 0 |
| 447 | EL expressions: West states | **Verify → close** | 0 |
| 178 | Full 41-state coverage | **Verify → close or re-scope** | 0 |
| 1161 | Itemized deductions performed by nothing | **Fix** | 1 |
| 776 | Static EDD usage analysis | **Finish narrow, descope rest** | 2 |
| 849 | Competing platforms | **Finish tranches** | 3 |
| 930 | Trace debugger roadmap | **Decide** | 4 |
| 234 | Reciprocal state agreements | **Build, or descope explicitly** | 5 |

---

## Phase 0 — verification sweep (closes up to three issues, ~half a session)

The state-coverage family was filed in March, before the work that may have
already satisfied it. Measured 2026-08-25:

- **51 state `_dt.xml` files carry real EL. Zero contain placeholder prose.**
- The 6 with no DSL are AK, GU, TX, VI, WA, WY — no-income-tax jurisdictions
  and territories, where empty is the correct answer.

#443 and #447 exist to "convert placeholder text to valid EL expressions".
There is no placeholder text left. The work to do is therefore *verification,
not authoring*:

1. For each state the two issues name, confirm the table compiles, executes
   against its scenarios, and its rate/bracket figures cite a source.
2. Where a citation is missing, add it through the authoring API (never by
   hand — [authoring contract](authoring-contract.md)).
3. Close each issue quoting the per-state evidence.

#178 ("full 41-state coverage") is then arithmetic: 51 ≥ 41. Close it on the
same evidence, **unless** the verification finds states whose EL executes but
whose figures are wrong — in which case re-scope it to name exactly those and
keep it open for Phase 5.

**Exit criteria**: #443 and #447 closed with evidence; #178 closed or reduced
to a named list.

**Known caveat, stated not hidden**: per-state scenario folders run about 2
of 3 clean, and `MultiState` is 0 of 38. That is scenario debt (Phase 1's
sibling), not missing coverage — do not let it block Phase 0 closes, and do
not claim it is fixed.

## Phase 1 — #1161, the itemized deduction subtree (1–2 sessions)

An entire deduction subtree is dead code: `Calculate_Itemized_Deductions` is
performed by nothing, so every return takes the standard deduction. Measured
on the issue: the obvious one-line fix repairs **zero** scenarios and breaks
three, because the subtree has component defects underneath. Order matters —
do not lead with the one-liner.

1. **Medical** — `Calculate_Medical_Expense_Deduction` contributes nothing
   even above the 7.5%-of-AGI floor (`ScheduleA_01` does not move when the
   subtree is wired in). Find and fix.
2. **SALT collection** — `Sum_SALT_Deduction` iterates a `deductions` array
   that scenario property data does not fully populate: `Family_2025`
   declares two properties (8,500 and 4,200 in tax) and collects 8,500.
   Fix the mapping/collection so every property contributes.
3. **Mortgage / charitable residuals** — after 1 and 2, re-measure
   `ScheduleA_02/03`; the earlier probe left ±2,500–3,000 unexplained.
4. **Then** apply `perform Calculate_Itemized_Deductions`, reconcile the
   scenarios that flip (each gap must equal itemized − standard exactly, the
   #1159 substitution rule), and raise `scenariosCleanFloor` past 280.

**Exit criteria**: `dtrules verify` clean, `make check` green, floor raised,
#1161 closed. **Never** substitute a computed value that cannot be derived
from the scenario's own declared inputs — that is what stopped the first
attempt and the rule holds.

## Phase 2 — #776, finish narrow and descope the rest (1 session)

Dispatch bounding is complete (derived bounds #1155, explicit `among` #1156,
`with default` #1137, unbounded refusal). Two items remain, and they are not
equal:

- **`possibly_used`** — the category exists and is emitted nowhere. Wire it:
  a field reachable only through a dynamic dispatch's possible targets is
  neither definitely-used nor unused. Small, and it completes the issue's
  stated deliverable.
- **AST-walk replacement of the regex pass** — the issue's original scope.
  **Descope.** The regex pass survives hand-verification on the whole corpus;
  rewriting it is large engineering for no observed defect. Close #776 with
  this stated, and open a narrow successor *only if* a false finding is ever
  observed in practice. Do not carry an unbuilt rewrite as an open issue.

**Exit criteria**: `possibly_used` emitted and tested; #776 closed with the
descope reasoned on the thread.

## Phase 3 — #849, finish the tranches (1 session)

[`competitive-landscape.md`](competitive-landscape.md) has five platforms.
Remaining: Blue Polaris, RapidGen, Aletyx, Rules Matix, plus KU Leuven and
RuleML evaluated as research/standards rather than products. Same question
set, same citation discipline (vendor docs, not memory).

**Exit criteria**: all listed platforms profiled; the synthesis re-read as a
whole; #849 closed as "living doc established" — the doc keeps living, the
issue does not need to.

## Phase 4 — #930, decide (no code, one comment)

Trace debugger roadmap: watch breakpoints, EL console, cross-trace diff.
**Recommended disposition: close as captured-not-scheduled.** The design
decisions are recorded; roadmaps written ahead of users go stale, and this
one has no user waiting on it. Reopen the day someone debugging rules hits
the gap. If Paul wants a slice built instead, the smallest useful one is
watch breakpoints on a single field.

## Phase 5 — #234, reciprocal agreements (the real remainder)

Measured: **no reciprocity implementation exists anywhere in the project** —
no fields, no tables, no scenarios. This is the one genuinely unbuilt item on
the board.

Roughly 30 bilateral agreements (IL–IA/KY/MI/WI, MD–DC/PA/VA/WV, etc.). Each
says: a resident of A working in B pays A, and B withholds nothing, on filing
a certificate. Work: research the pairs from state revenue departments
(citation discipline as with the CorporateTax statutes), model the pair table
plus the certificate flag, implement, and write scenarios — a resident of each
side of at least the large agreements.

Sequenced last because it is the largest and least entangled. If it is not
worth building, **close it explicitly** with that reasoning rather than
leaving it open for another five months.

**Exit criteria**: implemented with scenarios and `MultiState` improving, or
closed with a stated decision.

---

## After the deck is clear

The scenario corpus is the honest remainder: 505 scenarios, 280 clean.
Phases 1 and 5 will move it. Once the board is empty, file **one** tracking
issue for the rest with the folder-level breakdown, so the debt is visible
without eight stale issues standing in for it.
