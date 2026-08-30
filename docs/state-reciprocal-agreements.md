# State Reciprocal Agreements

A reciprocal agreement lets a resident of state A who works in state B pay
income tax only to A. B does not tax the wages, and — once the employee files
B's exemption certificate — does not withhold on them either.

Implemented for #234. This document is the reference for what is modelled, what
is deliberately not, and where each figure comes from.

## What the rules do

Reciprocity is a **work-state relief on wage income**. It never changes the
resident state's return: the resident state taxes its residents on everything,
agreement or no agreement. What it changes is the *other* state's return.

The chain, in execution order inside `Dispatch_State_Tax` (table 40000):

| Table | Number | Does |
|-------|--------|------|
| `Log_State_Tax_Start` | 39998 | Multi-state when `job.state_periods` is non-empty; runs `Allocate_Income_By_State` |
| `Allocate_Income_By_State` | 45000 | Per period: day count, allocation percentage, resets the accumulators |
| `Allocate_Income_To_State_Period` | 45010 | Per income: accumulates `allocated_income` and, for wages only, `allocated_wage_income` |
| `Calculate_State_Source_Income` | 40500 | Builds one `state_tax_result` per period via `Build_State_Tax_Result_For_Period` |
| `Determine_State_Reciprocity` | 40520 | **The matrix.** Sets `reciprocity_applies` and removes the exempt wages |

`Determine_State_Reciprocity` runs over `for all job.state_tax_results`. For a
matching work/resident pair it does this:

```
set state_tax_result.reciprocity_applies = true
set state_tax_result.reciprocity_partner = job.state
set state_tax_result.state_source_income =
        state_tax_result.state_source_income - state_tax_result.state_wage_income
set state_tax_result.state_agi           = state_tax_result.state_source_income
set state_tax_result.state_taxable_income = state_tax_result.state_source_income
set state_tax_result.state_wage_income   = 0
```

Only the **wage share** is removed. Anything else sourced to that state — rent,
self-employment, capital gains — stays taxable there. That is why the allocator
tracks `allocated_wage_income` separately from `allocated_income`: without the
split there is nothing to exempt selectively.

### Why the non-resident return is not deleted

The relief zeroes the work state's taxable wages; it does not remove the
`state_tax_result`. That is deliberate. The exemption certificate governs
**withholding**, not liability. An employee who never filed one has tax withheld
by the work state anyway, and recovers it by filing a non-resident return
showing zero taxable wages. Deleting the result would model a filing that still
has to happen.

`state_period.certificate_filed` records whether the certificate was filed.

## The matrix

30 bilateral agreements across 16 jurisdictions, tax year 2025. Every row was
verified against the work state's own revenue department publication.

| Work state | Exempts residents of | Certificate |
|---|---|---|
| District of Columbia | **any** nonresident | D-4A (refund on D-40B) |
| Illinois | IA, KY, MI, WI | IL-W-5-NR |
| Indiana | KY, MI, OH, PA, WI | WH-47 |
| Iowa | IL | IA 44-016 |
| Kentucky | IL, IN, MI, OH, VA, WV, WI | 42A809 (refund on 740-NP-R) |
| Maryland | DC, PA, VA, WV | MW507 |
| Michigan | IL, IN, KY, MN, OH, WI | MI-W4 line 8 |
| Minnesota | MI, ND | MWR |
| Montana | ND | MW-4 |
| New Jersey | PA | NJ-165 |
| North Dakota | MN, MT | NDW-R |
| Ohio | IN, KY, MI, PA, WV | IT 4 |
| Pennsylvania | IN, MD, NJ, OH, VA, WV | REV-419 |
| Virginia | DC, KY, MD, PA, WV | VA-4 (refund on 763-S) |
| West Virginia | KY, MD, OH, PA, VA | WV/IT-104NR |
| Wisconsin | IL, IN, KY, MI | W-220 |

The per-state counts sum to 60 = 2 × 30, with every agreement confirmed from
both sides.

### District of Columbia is not a bilateral pact

DC is barred by the Home Rule Act (DC Code § 1-206.02(a)(5)) from taxing the
income of **any** nonresident. So its row carries no partner list, and Form
D-4A asks only that the filer is not a DC resident. Maryland and Virginia do
hold written agreements with DC, but DC's side of them is redundant with
federal law.

This is why the DC row — and only the DC row — carries an explicit
`is_resident is equal to false` guard. Every other row implies non-residency
because no state is its own reciprocity partner; DC's would otherwise exempt a
DC resident's own wages.

## What is deliberately excluded

**Minnesota–Wisconsin is terminated.** Minnesota ended it effective
2010-01-01. Wisconsin Pub 121 (01/26) lists only IL, IN, KY and MI; Minnesota
lists only MI and ND. 2023 Wisconsin Act 147 required a joint reinstatement
study, delivered December 2024; no agreement resulted. Listing it would exempt
wages that are owed.

**Arizona is not a reciprocity state**, though it is widely published as one.
A.R.S. § 43-1096 is a *withholding exception backed by a credit*: a resident of
CA, IN, OR or VA may stop Arizona withholding on Form WEC, but the wages remain
Arizona-source and Arizona-taxable, a Form 140NR is still required, and the
liability is offset by a credit on Form 309. That is the inverse of reciprocity,
where the work state's tax never attaches at all.

**Maryland's pre-1992 statutory reciprocity** with AZ, CA, IN, MI and WI ended
with Chapter 1, Acts of 1992. Maryland's only current agreements are DC, PA, VA
and WV.

**Convenience-of-the-employer rules are a different mechanism** and are not
modelled here. Reciprocity asks whose tax applies to work performed in the work
state; the convenience rule asks whether remote days are sourced back to the
employer's state at all. NY, DE, NE, PA and AL apply one unconditionally; CT and
NJ apply one only when the taxpayer's resident state does. Where both could
apply, reciprocity wins — New Jersey states its convenience rule "does not apply
to Pennsylvania residents who work in New Jersey, since there is a Reciprocal
Agreement in place with that state."

## Conditions not yet modelled

The agreements carry per-pair conditions that the matrix does not yet encode.
Each would narrow the relief, so the current model is the permissive case:

- **Daily-commuter requirement** — Kentucky ↔ Virginia only. Both sides require
  the taxpayer to commute daily.
- **Monthly-return-home requirement** — Minnesota (for both MI and ND) and North
  Dakota (for MN residents only) require returning to the permanent residence at
  least monthly. Form MWR stops the filer outright if they do not.
- **183-day / place-of-abode tests** — MD, VA, WV and KY convert a long-staying
  commuter into a statutory resident, which defeats reciprocity. Maryland's WV
  agreement is the exception: unconditional.
- **Full-year residency** — Kentucky's 740-NP-R is for full-year nonresidents
  only.
- **Local income taxes generally survive reciprocity** — Indiana county LIT is
  charged to reciprocal-state residents exactly as to anyone else, and a PA
  resident exempt from Maryland *state* tax still owes the Maryland *county*
  rate unless they live in York or Adams County.
- **Ohio ↔ Kentucky ≥20% S-corp carve-out** — compensation paid to a
  shareholder-employee holding 20% or more of a pass-through entity is
  reclassified as a distributive share and falls outside the agreement. Form
  42A809 makes the filer certify against it.

## Scope: wages only

Every state states this explicitly. Wisconsin Pub 121 is the clearest:

> "Reciprocity applies only to income earned as an employee. It does not apply
> to other types of income, such as gains on the sale of property, rental
> income, lottery winnings, self-employment income, income from pass-through
> entities, etc."

The rules implement this through `income.type`: only `w2_wages`, `w2` and
`wages` accumulate into `allocated_wage_income`. `self_employment`, `rental`,
`capital_gain`, `dividend` and the rest never do, so they are never exempted.

## Writing a scenario

Set the resident state on `job.state`, give the taxpayer a `state_period` for
each state, and tag the income with `state_code`. Do **not** put a
`reciprocal_agreement` flag on the input — the agreement is derived from the
pair, and asserting it in the data would test the scenario rather than the
rules.

Two expectation tags drive `Validate_Reciprocity` (table 8050):

```xml
<expected_reciprocity_state>KY</expected_reciprocity_state>
<expected_nonresident_source_income>0</expected_nonresident_source_income>
```

`expected_reciprocity_state` names the work state that should be exempted; an
empty value means the scenario makes no reciprocity claim and the validation
stays silent. `expected_nonresident_source_income` is what that state may still
tax after the wage exemption — 0 when the taxpayer earned only wages there.

Committed scenarios live in `testfiles/TestScenarios/MultiState/`:
`..._05_OH_KY_Reciprocal`, `..._09_IL_WI_Reciprocal`,
`..._10_NJ_PA_Reciprocal`, `..._11_WI_MN_No_Agreement` (a negative) and
`..._12_OH_KY_Wages_Only` (wages exempt, rent still taxed).

All 30 pairs are exercised from both directions by `TestReciprocityMatrix` in
`pkg/dtrules/reciprocity_test.go`, alongside the negatives for the terminated
and misattributed agreements.

## Sources

Each figure traces to the work state's own publication: DC Code § 1-206.02(a)(5)
and Form D-4A; Illinois IL-W-5-NR; Indiana Information Bulletin #33 (Dec 2024)
and WH-47; Iowa's Iowa–Illinois Reciprocal Agreement guidance; Kentucky Forms
42A809 and 740-NP-R; Maryland Administrative Release No. 3 and MW507; Michigan
Form 446 (2026 Withholding Guide); Minnesota DOR reciprocity guidance and Form
MWR; Montana DOR North Dakota reciprocity page; New Jersey njit25; North Dakota
Form NDW-R; Ohio IT 4 (R.C. 5747.05(A)(2)); Pennsylvania PIT Guide; Virginia Tax
reciprocity page; West Virginia TSD-381; Wisconsin Publication 121 (01/26).
