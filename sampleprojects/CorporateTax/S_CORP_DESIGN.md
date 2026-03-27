# S Corporation (Form 1120-S) Support - Design Document

**Project**: DTRules Corporate Tax
**Document**: S Corporation Design Analysis
**Date**: 2026-03-24
**Status**: Research & Design (No Implementation)
**Complexity**: High

---

## Executive Summary

S Corporations represent a fundamentally different taxation model than C Corporations:
- **Pass-through taxation** - Income flows to shareholders, NOT taxed at entity level
- **Limited entity-level taxes** - Only applies in specific conversion/passive income scenarios
- **Shareholder-centric** - Requires per-shareholder tracking and Schedule K-1 generation
- **Basis complexity** - Stock basis and debt basis calculations drive distribution taxation

**Key Finding**: S Corporation support requires significant architectural changes to the existing C Corporation implementation. The current entity model is entity-centric; S Corporations require a shareholder-centric model with per-share-per-day allocation logic.

**Recommendation**: Treat S Corporation as a **separate sub-project** rather than an extension of the C Corporation tables. Estimated 6-8 weeks for full implementation.

---

## Table of Contents

1. [Key Differences from C Corporations](#1-key-differences-from-c-corporations)
2. [Entity Architecture Requirements](#2-entity-architecture-requirements)
3. [Tax Calculations & Special Taxes](#3-tax-calculations--special-taxes)
4. [Forms & Schedules](#4-forms--schedules)
5. [Integration with Existing System](#5-integration-with-existing-system)
6. [State-Level Treatment](#6-state-level-treatment)
7. [Decision Table Design](#7-decision-table-design)
8. [Complexity Assessment](#8-complexity-assessment)
9. [Implementation Recommendations](#9-implementation-recommendations)
10. [Alternative Approaches](#10-alternative-approaches)

---

## 1. Key Differences from C Corporations

### 1.1 Fundamental Tax Treatment

| Aspect | C Corporation (Form 1120) | S Corporation (Form 1120-S) |
|--------|---------------------------|----------------------------|
| **Entity-level tax** | Yes - 21% federal tax | **No** - pass-through only |
| **Shareholder taxation** | Dividends taxed at individual level | **All income** taxed at individual level |
| **Double taxation** | Yes (entity + dividend) | No - single layer |
| **Tax rate** | 21% flat corporate rate | Individual rates (10%-37%) |
| **Loss utilization** | NOL carryforward (80% limit) | Passed to shareholders (subject to basis) |

**Impact**: Most of the existing federal tax calculation tables (4000-4999 series) **do NOT apply** to S Corporations.

### 1.2 When Entity-Level Tax DOES Apply

S Corporations are subject to entity-level tax in only **three specific scenarios**:

#### A. Built-In Gains (BIG) Tax - IRC § 1374
**Trigger**: C Corp converts to S Corp, sells appreciated assets within 5 years
**Rate**: 21% (highest corporate rate)
**Base**: Lesser of (a) recognized built-in gain, or (b) taxable income

```
Example:
- C Corp has machinery with $100K fair value, $40K basis
- Converts to S Corp on 1/1/2025
- Sells machinery on 6/1/2026 for $110K
- Built-in gain at conversion: $100K - $40K = $60K
- Recognized BIG: $60K (within 5-year period)
- BIG tax: $60K × 21% = $12,600
```

#### B. Excess Net Passive Income (ENPI) Tax - IRC § 1375
**Trigger**: S Corp has C Corp E&P AND passive income > 25% of gross receipts
**Rate**: 21% (highest corporate rate)
**Base**: Excess net passive income

```
Formula:
ENPI = Net Passive Income × (Passive Investment Income - 25% × Gross Receipts) / Passive Investment Income

Example:
- Gross receipts: $1,000,000
- Passive investment income: $400,000 (40% of gross receipts)
- Expenses allocable to passive income: $50,000
- Net passive income: $400,000 - $50,000 = $350,000
- 25% threshold: $1,000,000 × 25% = $250,000
- Excess: $400,000 - $250,000 = $150,000
- ENPI: $350,000 × ($150,000 / $400,000) = $131,250
- ENPI tax: $131,250 × 21% = $27,563
```

**Passive Income Includes**: Rents, royalties, dividends, interest, annuities, gains from stock/securities sales

#### C. LIFO Recapture Tax - IRC § 1363(d)
**Trigger**: C Corp using LIFO converts to S Corp
**Rate**: 21% (highest corporate rate)
**Base**: LIFO reserve (difference between LIFO and FIFO inventory value)
**Payment**: Can be spread over 4 years

```
Example:
- C Corp using LIFO has $200K LIFO basis inventory
- FIFO value: $350K
- LIFO reserve: $350K - $200K = $150K
- LIFO recapture tax: $150K × 21% = $31,500
- Payment: $7,875/year for 4 years
```

**Impact**: Need three specialized decision tables for these entity-level taxes.

### 1.3 Pass-Through Income Allocation

**Core Principle**: S Corporation income, losses, deductions, and credits **flow through** to shareholders in proportion to stock ownership.

**Allocation Method**: **Per-share-per-day** basis (IRC § 1377(a)(1))

```
Each item allocated = (Total item amount / 365 days) × Days owned × (Shares owned / Total shares)

Example:
- S Corp has $100,000 ordinary income (2025)
- Shareholder A owns 60% (219 shares of 365 total)
- Shareholder A sells 50% of stock on July 1 (kept 30% for full year, 30% for half year)
- Allocation to A:
  - Jan 1 - Jun 30 (181 days): $100,000 × (181/365) × 60% = $29,753
  - Jul 1 - Dec 31 (184 days): $100,000 × (184/365) × 30% = $15,123
  - Total: $44,876
```

**Alternative**: Closing-of-books election (IRC § 1377(a)(2)) - when shareholder disposes of entire interest or > 20% interest, can elect to close books as of disposition date.

**Impact**: Need shareholder entity with daily ownership tracking.

---

## 2. Entity Architecture Requirements

### 2.1 New Entities Required

#### A. Shareholder Entity
**Purpose**: Track individual shareholders and their ownership

```xml
<entity name="shareholder">
  <!-- Identification -->
  <field name='shareholder_id' type='string' comment='Unique shareholder ID'/>
  <field name='name' type='string' comment='Shareholder name'/>
  <field name='ssn_ein' type='string' comment='SSN (individual) or EIN (entity)'/>
  <field name='shareholder_type' type='string' comment='individual, trust, estate, corporation'/>

  <!-- Ownership -->
  <field name='shares_owned_beginning' type='integer' comment='Shares at beginning of year'/>
  <field name='shares_owned_ending' type='integer' comment='Shares at end of year'/>
  <field name='ownership_percentage' type='double' comment='Average ownership % for year'/>
  <field name='days_owned' type='integer' comment='Days owned during tax year (for partial-year shareholders)'/>

  <!-- Stock Basis Tracking (Form 7203) -->
  <field name='stock_basis_beginning' type='double' comment='Stock basis at beginning of year'/>
  <field name='stock_basis_ending' type='double' comment='Stock basis at end of year'/>
  <field name='stock_basis_increases' type='double' comment='Increases: capital contributions, income items'/>
  <field name='stock_basis_decreases' type='double' comment='Decreases: distributions, losses, deductions'/>

  <!-- Debt Basis Tracking (Form 7203) -->
  <field name='debt_basis_beginning' type='double' comment='Debt basis at beginning of year'/>
  <field name='debt_basis_ending' type='double' comment='Debt basis at end of year'/>
  <field name='shareholder_loans_outstanding' type='double' comment='Loans from shareholder to S Corp'/>

  <!-- Allocated Income/Loss Items (Schedule K-1) -->
  <field name='ordinary_income_allocated' type='double' comment='Ordinary business income/loss'/>
  <field name='rental_income_allocated' type='double' comment='Rental real estate income/loss'/>
  <field name='interest_income_allocated' type='double' comment='Interest income'/>
  <field name='dividend_income_allocated' type='double' comment='Dividend income'/>
  <field name='royalty_income_allocated' type='double' comment='Royalty income'/>
  <field name='capital_gain_allocated' type='double' comment='Net short-term/long-term capital gain/loss'/>
  <field name='section_1231_gain_allocated' type='double' comment='Section 1231 gain/loss'/>
  <field name='charitable_contribution_allocated' type='double' comment='Charitable contributions'/>
  <field name='section_179_allocated' type='double' comment='Section 179 deduction'/>

  <!-- Credits Allocated -->
  <field name='credits_allocated' type='double' comment='Total credits allocated to shareholder'/>

  <!-- Distributions -->
  <field name='cash_distributions' type='double' comment='Cash distributions received'/>
  <field name='property_distributions_fmv' type='double' comment='Property distributions (FMV)'/>

  <!-- Schedule K-1 Output -->
  <field name='k1_box_1_ordinary_income' type='double' comment='K-1 Box 1 - Ordinary business income/loss'/>
  <field name='k1_box_2_net_rental_income' type='double' comment='K-1 Box 2 - Net rental real estate income/loss'/>
  <field name='k1_box_3_interest' type='double' comment='K-1 Box 3 - Interest income'/>
  <!-- ... additional K-1 boxes ... -->
</entity>
```

**Rationale**: S Corporation taxation is shareholder-centric. Each shareholder needs:
1. Income/loss allocation (per-share-per-day)
2. Basis tracking (limits loss deductions and determines distribution taxation)
3. Schedule K-1 generation

#### B. S Corporation Entity Extensions
**Purpose**: Extend existing corporation entity with S Corp-specific fields

```xml
<entity name="s_corporation">
  <!-- S Election -->
  <field name='s_election_date' type='date' comment='Date S election became effective'/>
  <field name='total_shares_outstanding' type='integer' comment='Total shares outstanding (all classes combined)'/>
  <field name='share_class_count' type='integer' comment='Number of share classes (must be 1 for S Corp)'/>
  <field name='shareholder_count' type='integer' comment='Number of shareholders (max 100 for S Corp)'/>

  <!-- C Corporation History (for BIG tax) -->
  <field name='prior_c_corp' type='boolean' comment='Was corporation previously a C Corp?'/>
  <field name='c_to_s_conversion_date' type='date' comment='Date of C to S conversion'/>
  <field name='big_recognition_period_end' type='date' comment='End of 5-year BIG recognition period'/>
  <field name='net_unrealized_big_at_conversion' type='double' comment='Net unrealized BIG at conversion date'/>

  <!-- C Corporation E&P (for ENPI tax) -->
  <field name='has_c_corp_earnings_profits' type='boolean' comment='Has accumulated E&P from C Corp years?'/>
  <field name='accumulated_earnings_profits' type='double' comment='Accumulated E&P from C Corp years'/>

  <!-- LIFO Inventory (for LIFO recapture) -->
  <field name='uses_lifo_inventory' type='boolean' comment='Uses LIFO inventory method?'/>
  <field name='lifo_reserve_at_conversion' type='double' comment='LIFO reserve at S election date'/>
  <field name='lifo_recapture_tax_paid' type='double' comment='LIFO recapture tax paid (cumulative)'/>
  <field name='lifo_recapture_tax_remaining' type='double' comment='LIFO recapture tax remaining (4-year payment)'/>

  <!-- AAA Tracking (Schedule M-2) -->
  <field name='aaa_beginning' type='double' comment='AAA at beginning of year'/>
  <field name='aaa_ending' type='double' comment='AAA at end of year'/>
  <field name='aaa_ordinary_income' type='double' comment='AAA increase from ordinary income'/>
  <field name='aaa_other_additions' type='double' comment='AAA other additions'/>
  <field name='aaa_loss_deductions' type='double' comment='AAA decrease from loss/deductions'/>
  <field name='aaa_distributions' type='double' comment='AAA decrease from distributions'/>

  <!-- OAA Tracking (Other Adjustments Account - Schedule M-2) -->
  <field name='oaa_beginning' type='double' comment='OAA at beginning of year'/>
  <field name='oaa_ending' type='double' comment='OAA at end of year'/>
  <field name='oaa_tax_exempt_income' type='double' comment='Tax-exempt income (increases OAA)'/>
  <field name='oaa_nondeductible_expenses' type='double' comment='Non-deductible expenses (decreases OAA)'/>

  <!-- Previously Taxed Income (PTI - for S Corps elected pre-1983) -->
  <field name='pti_beginning' type='double' comment='PTI at beginning of year (rare - pre-1983 S Corps)'/>
  <field name='pti_ending' type='double' comment='PTI at end of year'/>
</entity>
```

**Rationale**: S Corporations have unique tracking requirements:
1. **AAA (Accumulated Adjustments Account)**: Tracks post-S-election undistributed income to determine tax-free vs taxable distributions
2. **C Corp E&P**: Determines if distributions are taxable dividends
3. **BIG tracking**: For converted C Corps within 5-year recognition period

#### C. Distribution Entity
**Purpose**: Track distributions to shareholders (tax treatment depends on AAA, basis, E&P)

```xml
<entity name="distribution">
  <field name='distribution_id' type='string' comment='Unique distribution ID'/>
  <field name='shareholder_id' type='string' comment='Recipient shareholder ID'/>
  <field name='distribution_date' type='date' comment='Date of distribution'/>
  <field name='distribution_type' type='string' comment='cash, property, stock'/>
  <field name='cash_amount' type='double' comment='Cash distributed'/>
  <field name='property_fmv' type='double' comment='Fair market value of property distributed'/>
  <field name='property_basis' type='double' comment='S Corp basis in property distributed'/>

  <!-- Tax Treatment (determined by ordering rules) -->
  <field name='aaa_portion' type='double' comment='Portion from AAA (tax-free)'/>
  <field name='pti_portion' type='double' comment='Portion from PTI (tax-free)'/>
  <field name='return_of_capital' type='double' comment='Portion as return of capital (reduces basis)'/>
  <field name='capital_gain' type='double' comment='Portion as capital gain (exceeds basis)'/>
  <field name='dividend_portion' type='double' comment='Portion as dividend (if C Corp E&P exists)'/>
</entity>
```

**Rationale**: Distribution ordering rules are complex:
1. First from **AAA** (tax-free, reduces AAA and basis)
2. Then from **PTI** (tax-free if pre-1983 S Corp, reduces PTI and basis)
3. Then from **C Corp E&P** (taxable dividend, doesn't reduce basis)
4. Then as **return of capital** (tax-free, reduces basis)
5. Finally as **capital gain** (taxable, if exceeds basis)

### 2.2 Entity Relationship Model

```
S Corporation (1)
    ├── has many Shareholders (1-100, S Corp limit)
    │   ├── has Ownership History (for per-share-per-day allocation)
    │   ├── has Stock Basis (Form 7203)
    │   ├── has Debt Basis (Form 7203)
    │   └── receives Schedule K-1 (Form 1120-S K-1)
    │
    ├── has many Distributions
    │   └── allocated to Shareholders (ordering rules apply)
    │
    ├── has AAA (Schedule M-2 Column a)
    ├── has OAA (Schedule M-2 Column b)
    ├── has PTI (Schedule M-2 Column d, if pre-1983)
    ├── may have C Corp E&P (if prior C Corp)
    │
    ├── has Revenue (same as C Corp)
    ├── has Expenses (same as C Corp)
    ├── has Assets (same as C Corp)
    │
    └── may have Entity-Level Taxes
        ├── BIG Tax (if prior C Corp within 5 years)
        ├── ENPI Tax (if passive income > 25% AND C Corp E&P exists)
        └── LIFO Recapture (if LIFO inventory at conversion)
```

**Complexity Note**: Unlike C Corporations (single result entity), S Corporations require **N shareholder entities** (N = 1 to 100) plus specialized tracking accounts.

---

## 3. Tax Calculations & Special Taxes

### 3.1 Built-In Gains (BIG) Tax - IRC § 1374

**When Applicable**:
- S Corporation was previously a C Corporation
- Within 5-year recognition period from conversion date
- Recognizes gain from sale of asset that was appreciated at conversion date

**Calculation Flow**:

```
Step 1: Determine if asset was held at conversion date
  → If yes, continue to Step 2
  → If no, no BIG tax applies

Step 2: Identify built-in gain at conversion
  BIG = FMV at conversion - Adjusted basis at conversion
  (Note: Must track per-asset at conversion date)

Step 3: Determine recognized built-in gain (RBIG)
  RBIG = Lesser of:
    (a) Gain recognized on sale, OR
    (b) Built-in gain at conversion

  Example:
  - FMV at conversion (1/1/2023): $100,000
  - Basis at conversion: $40,000
  - Built-in gain: $60,000
  - Sale date: 6/1/2025 (within 5 years)
  - Sale price: $110,000
  - Total gain: $110,000 - $40,000 = $70,000
  - RBIG: min($70,000, $60,000) = $60,000

Step 4: Apply taxable income limitation
  Net recognized BIG = Lesser of:
    (a) Total RBIG for year, OR
    (b) S Corporation taxable income (as if C Corp)

  (Prevents tax on unrealized appreciation)

Step 5: Calculate BIG tax
  BIG Tax = Net recognized BIG × 21%

Step 6: Reduce pass-through income
  Shareholder income allocation = S Corp income - BIG tax
```

**Decision Table Requirements**:
- Table 10100: Determine_BIG_Recognition_Period (check if within 5 years)
- Table 10110: Calculate_BIG_Asset_Gain (per-asset built-in gain calculation)
- Table 10120: Apply_BIG_Taxable_Income_Limitation
- Table 10130: Calculate_BIG_Tax (21% × net RBIG)

**Data Requirements**:
- Asset-level tracking of FMV at conversion date
- Asset-level tracking of basis at conversion date
- Conversion date (to determine 5-year period)

### 3.2 Excess Net Passive Income (ENPI) Tax - IRC § 1375

**When Applicable**:
- S Corporation has accumulated E&P from C Corporation years, AND
- Passive investment income > 25% of gross receipts

**Calculation Flow**:

```
Step 1: Determine if C Corp E&P exists
  → If no E&P, no ENPI tax applies
  → If E&P exists, continue to Step 2

Step 2: Calculate passive investment income percentage
  Passive % = Passive investment income / Gross receipts

  Passive investment income includes:
  - Dividends
  - Interest
  - Royalties
  - Rents
  - Annuities
  - Gains from sales of stocks/securities

Step 3: Check 25% threshold
  → If Passive % ≤ 25%, no ENPI tax applies
  → If Passive % > 25%, continue to Step 4

Step 4: Calculate net passive income
  Net passive income = Passive investment income - Directly connected expenses

Step 5: Calculate excess net passive income
  ENPI = Net passive income × [(Passive investment income - 25% × Gross receipts) / Passive investment income]

  Example:
  - Gross receipts: $1,000,000
  - Passive investment income: $400,000 (40%)
  - Expenses allocable to passive income: $50,000
  - Net passive income: $400,000 - $50,000 = $350,000
  - 25% threshold: $1,000,000 × 0.25 = $250,000
  - Excess: $400,000 - $250,000 = $150,000
  - ENPI: $350,000 × ($150,000 / $400,000) = $131,250

Step 6: Calculate ENPI tax
  ENPI Tax = ENPI × 21%

Step 7: Reduce pass-through income
  Each passive income item reduced pro-rata by ENPI tax
```

**Decision Table Requirements**:
- Table 10200: Determine_Passive_Income_Items (classify income as passive/non-passive)
- Table 10210: Calculate_Passive_Income_Percentage
- Table 10220: Calculate_Net_Passive_Income (income - directly connected expenses)
- Table 10230: Calculate_ENPI (formula application)
- Table 10240: Calculate_ENPI_Tax (21% × ENPI)
- Table 10250: Reduce_Passive_Income_Items (pro-rata reduction to pass-through)

**Warning**: If passive income > 25% for **3 consecutive years** AND S Corp has C Corp E&P, S election **automatically terminates**. Need to track consecutive years.

### 3.3 LIFO Recapture Tax - IRC § 1363(d)

**When Applicable**:
- C Corporation used LIFO inventory method
- Elects S Corporation status

**Calculation Flow**:

```
Step 1: Calculate LIFO reserve at conversion
  LIFO reserve = FIFO value - LIFO basis

  Example:
  - LIFO basis: $200,000
  - FIFO value: $350,000
  - LIFO reserve: $150,000

Step 2: Calculate LIFO recapture tax (last C Corp year)
  LIFO recapture tax = LIFO reserve × 21%

  Example: $150,000 × 21% = $31,500

Step 3: Payment schedule (4-year installments)
  - 1st installment: Due with last C Corp return
  - 2nd-4th installments: Due with next 3 S Corp returns

  Example:
  - Total tax: $31,500
  - Annual installment: $31,500 / 4 = $7,875/year
  - Years: 2025 (C Corp final), 2026-2028 (S Corp)

Step 4: Basis adjustment
  Inventory basis increased by LIFO reserve amount
  New basis = LIFO basis + LIFO reserve = FIFO value
```

**Decision Table Requirements**:
- Table 10300: Determine_LIFO_Recapture_Required (check if LIFO used at conversion)
- Table 10310: Calculate_LIFO_Reserve (FIFO - LIFO)
- Table 10320: Calculate_LIFO_Recapture_Tax (reserve × 21%)
- Table 10330: Calculate_LIFO_Installment_Payment (total tax / 4)

**Note**: This is a **one-time tax** at conversion. After conversion year, only installment tracking needed.

### 3.4 No Regular Corporate Tax

**Critical Difference**: Unlike C Corporations, S Corporations do **NOT** calculate regular corporate income tax on line 22a of Form 1120-S.

**Existing Tables That DO NOT APPLY**:
- Table 4000: Calculate_Taxable_Income (still calculated but for informational purposes only)
- Table 4100: Apply_Corporate_Tax_Rate (NOT USED - no 21% tax)
- Table 4300: Calculate_Refund_Or_Owed (different calculation for S Corp)

**Replacement Logic**:
```
S Corporation tax liability =
  + BIG tax (if applicable)
  + ENPI tax (if applicable)
  + LIFO recapture installment (if applicable)
  + State entity-level taxes (varies by state)
  - Estimated tax payments
  - Credits (limited)
```

---

## 4. Forms & Schedules

### 4.1 Form 1120-S Structure

**Form 1120-S**: U.S. Income Tax Return for an S Corporation

**Key Sections**:

| Section | Lines | Purpose | Complexity |
|---------|-------|---------|------------|
| **Income** | 1-6 | Gross receipts, returns, COGS, other income | Similar to C Corp |
| **Deductions** | 7-20 | Compensation, rent, taxes, interest, depreciation | Similar to C Corp |
| **Tax & Payments** | 22a-22c | Entity-level taxes only (BIG, ENPI, LIFO recapture) | **S Corp specific** |
| **Schedule K** | Lines 1-18 | **All** income/loss/deduction/credit items to allocate to shareholders | **New complexity** |
| **Schedule L** | Assets/Liabilities | Balance sheet | Same as C Corp |
| **Schedule M-1** | Reconciliation | Book income vs tax income | Same as C Corp |
| **Schedule M-2** | AAA/OAA/PTI | **S Corp specific** - distribution ordering | **New complexity** |

**Schedule K (Pass-Through Items)**:
```
Income/Loss:
1. Ordinary business income/loss
2. Net rental real estate income/loss
3. Other gross rental income/loss
4. Interest income
5a. Ordinary dividends
5b. Qualified dividends
6. Royalties
7. Net short-term capital gain/loss
8a. Net long-term capital gain/loss
8b. Collectibles (28%) gain/loss
8c. Unrecaptured § 1250 gain
9. Net § 1231 gain/loss
10. Other income/loss

Deductions:
11. Section 179 deduction
12. Charitable contributions
13. Investment interest expense
14. Section 59(e)(2) expenditures
15. Deductions - royalty income
16. Section 1231 gain/loss
17. Other deductions

Credits:
13. Credits
  a. Low-income housing credit
  b. Qualified rehabilitation expenditures
  c. Other rental credits
  d. Alcohol and cellulosic biofuel fuels credit
  e. Other credits

Self-Employment:
14. Net earnings (loss) from self-employment
15. Gross farming or fishing income
16. Gross nonfarm income

Foreign Transactions:
16. Foreign transactions (various codes)

AMT Items:
17. Alternative minimum tax (AMT) items

Tax-Exempt Income:
18. Tax-exempt income and nondeductible expenses

Distributions:
19. Distributions
20. Other information
```

**Each line** must be allocated to shareholders and reported on individual K-1s.

### 4.2 Schedule K-1 (Form 1120-S)

**Purpose**: Report each shareholder's share of income, deductions, credits, etc.

**Generation Requirements**:
- **One K-1 per shareholder** (1-100 shareholders)
- Per-share-per-day allocation (or closing-of-books if elected)
- Basis information (Form 7203 support)

**Key Boxes**:
```
Part I: Information About the Corporation
- Corp name, address, EIN
- IRS center where return filed

Part II: Information About the Shareholder
- Shareholder name, address, SSN/EIN
- Shareholder's % of stock ownership for tax year
- Beginning of year: % ownership, shares owned
- End of year: % ownership, shares owned
- Beginning/ending basis (if determinable)
- Loans from shareholder (debt basis)

Part III: Shareholder's Share of Current Year Income, Deductions, Credits, etc.
- Box 1: Ordinary business income/loss
- Box 2: Net rental real estate income/loss
- Box 3: Other net rental income/loss
- Box 4: Interest income
- Box 5a: Ordinary dividends
- Box 5b: Qualified dividends
- Box 6: Royalties
- Box 7: Net short-term capital gain/loss
- Box 8a: Net long-term capital gain/loss
- Box 9: Net § 1231 gain/loss
- Box 10: Other income/loss
- Box 11: Section 179 deduction
- Box 12: Other deductions
- Box 13: Credits
- Box 14: Foreign transactions
- Box 15: AMT items
- Box 16: Items affecting shareholder basis
- Box 17: Other information
```

**Computational Challenge**: For N shareholders, need to:
1. Allocate each Schedule K line item to N shareholders (per-share-per-day)
2. Generate N separate K-1 forms
3. Track basis adjustments for N shareholders

### 4.3 Schedule M-2 for S Corporations

**Critical Difference**: C Corp Schedule M-2 tracks retained earnings. S Corp Schedule M-2 tracks **AAA, OAA, PTI, and other equity accounts**.

**Columns**:
```
Column (a): Accumulated Adjustments Account (AAA)
Column (b): Other Adjustments Account (OAA)
Column (c): Shareholders' Undistributed Taxable Income Previously Taxed (PTI)
Column (d): Accumulated Earnings and Profits (C Corp E&P)
```

**AAA Tracking** (Column a):
```
Beginning balance
+ Ordinary income (Schedule K, line 1)
+ Other additions (net income from Schedule K, lines 2-10, except tax-exempt income)
- Loss and deductions (Schedule K, lines 1-12, except items that aren't deductible and weren't capitalized)
- Distributions (reduces AAA, not below zero before going to column d)
= Ending balance
```

**Distribution Ordering**:
1. Distributions come from AAA first (tax-free to shareholders, reduces AAA and basis)
2. Then from PTI (if pre-1983 S Corp - rare)
3. Then from C Corp E&P (taxable dividend if exists)
4. Then return of capital (reduces basis)
5. Then capital gain (if exceeds basis)

**Complexity**: Schedule M-2 for S Corp is fundamentally different from C Corp and drives distribution taxation.

### 4.4 Form 7203 (Shareholder Stock & Debt Basis)

**Purpose**: Calculate shareholder's stock basis and debt basis limitations on losses

**Not required on Form 1120-S**, but shareholders use it to track basis (impacts K-1 box 16).

**Stock Basis Calculation**:
```
Beginning stock basis
+ Stock purchases
+ Capital contributions
+ Income items (ordinary income, other income, tax-exempt income)
+ Excess depletion
- Distributions (non-dividend, non-capital gain)
- Non-deductible, non-capital expenses
- Depletion (oil & gas)
- Loss and deduction items
- Investment in property other than money
= Ending stock basis (cannot go below zero)
```

**Debt Basis Calculation**:
```
Beginning debt basis
+ Additional loans from shareholder to S Corp
+ Net increase in debt (shareholder guarantees, etc.)
- Loan repayments
- Net decrease in debt
= Ending debt basis
```

**Loss Limitation Rules**:
1. Losses limited to **stock basis first**
2. If losses exceed stock basis, limited to **debt basis second**
3. Excess losses **carry forward** until basis increases

**Impact on Design**: Need to track basis adjustments throughout the year and validate loss deductions against basis.

---

## 5. Integration with Existing System

### 5.1 Reusable Components

**Can Reuse** (with modifications):

| Component | C Corporation | S Corporation | Modification Needed |
|-----------|---------------|---------------|---------------------|
| **Revenue entity** | Form 1120 lines 1-11 | Form 1120-S lines 1-6 | Minor - slightly different line items |
| **Expense entity** | Form 1120 lines 12-27 | Form 1120-S lines 7-20 | Minor - similar structure |
| **Asset entity** | Form 4562 depreciation | Form 4562 depreciation | None - same depreciation rules |
| **Depreciation tables** | Tables 6000-6900 | Tables 6000-6900 | None - MACRS same for S Corp |
| **Schedule L** | Balance sheet | Balance sheet | None - same structure |
| **Schedule M-1** | Book-tax reconciliation | Book-tax reconciliation | None - same purpose |

**Cannot Reuse**:

| Component | Reason |
|-----------|--------|
| **Tax calculation** (Tables 4000-4300) | S Corp has no regular corporate tax |
| **Credits** (Tables 5000-5999) | S Corp credits pass through to shareholders, not used at entity level |
| **Apportionment** (Tables 7000-7999) | S Corp apportionment allocates state income to shareholders, not entity tax |
| **Schedule M-2** (Table 8100) | Completely different structure (AAA/OAA/PTI vs retained earnings) |
| **NOL** (Table 9000) | S Corp losses pass through, no entity NOL |

### 5.2 Architectural Decision: Shared vs Separate

**Option A: Extend Existing C Corp System**
```
Pros:
+ Reuse revenue, expense, asset entities
+ Reuse depreciation tables
+ Single codebase

Cons:
- Complex branching logic (if C Corp then... else if S Corp then...)
- Schedule M-2 completely different (requires separate tables anyway)
- Shareholder tracking doesn't apply to C Corp
- Risk of breaking C Corp functionality
```

**Option B: Separate S Corp Module**
```
Pros:
+ Clean separation of concerns
+ S Corp-specific entities (shareholder, AAA)
+ No risk to existing C Corp system
+ Easier testing and maintenance

Cons:
- Some code duplication (revenue, expense structure)
- Need separate file structure
```

**Recommendation**: **Option B - Separate Module**

**Rationale**:
1. S Corporation is fundamentally different (pass-through vs entity-level taxation)
2. Shareholder tracking is core to S Corp, irrelevant to C Corp
3. Most tax calculation logic doesn't apply to S Corp
4. Schedule M-2 is completely different
5. Can still share utility tables (depreciation) by referencing common tables

**Proposed Structure**:
```
sampleprojects/
├── CorporateTax/              # C Corporation (Form 1120)
│   ├── xml/
│   │   ├── CorporateTax_edd_core.xml
│   │   ├── CorporateTax_dt_core.xml
│   │   └── states/...
│   └── ...
│
└── SCorporationTax/           # NEW: S Corporation (Form 1120-S)
    ├── xml/
    │   ├── SCorporationTax_edd_core.xml   # Shareholder, S Corp entities
    │   ├── SCorporationTax_dt_core.xml    # S Corp-specific tables
    │   ├── shared/
    │   │   └── depreciation_reference.xml # Reference to CorporateTax tables 6000-6900
    │   └── states/
    │       ├── TEMPLATE_scorp_edd.xml     # State S Corp templates
    │       └── TEMPLATE_scorp_dt.xml
    ├── testfiles/
    └── README.md
```

### 5.3 Table Numbering for S Corporation

**Proposed Numbering**:

| Range | Purpose | Status |
|-------|---------|--------|
| **10000-10999** | S Corporation Core | New |
| 10000-10099 | Orchestration | New |
| 10100-10199 | BIG Tax | New |
| 10200-10299 | ENPI Tax | New |
| 10300-10399 | LIFO Recapture | New |
| 10400-10499 | Schedule K allocation | New |
| 10500-10599 | Schedule K-1 generation | New |
| 10600-10699 | AAA/OAA/PTI tracking (Schedule M-2) | New |
| 10700-10799 | Distribution ordering | New |
| 10800-10899 | Shareholder basis calculation | New |
| 10900-10999 | Per-share-per-day allocation | New |
| **6000-6999** | Depreciation (shared) | Reference existing C Corp tables |
| **60000-99999** | State S Corp tables | New (100 per state) |

---

## 6. State-Level Treatment

### 6.1 State S Corporation Taxation Variance

**Critical Issue**: States treat S Corporations **inconsistently**.

| State Treatment | States | Entity Tax? | Example |
|-----------------|--------|-------------|---------|
| **Full recognition** | Most states | No | Florida - no income tax, recognizes S election |
| **Entity-level tax** | CA, NH, TN, TX (franchise) | **Yes** | California: 1.5% entity-level tax + $800 minimum |
| **No recognition** | None (as of 2025) | N/A | (Louisiana eliminated non-recognition in 2024) |
| **Partial recognition** | Some | Varies | Different add-backs/subtractions |

### 6.2 California S Corporation Example

**CA Franchise Tax Board (FTB) Treatment**:

```
Entity-Level Tax:
1. S Corp pays 1.5% tax on net income
   CA S Corp Tax = CA apportioned income × 1.5%

2. Minimum franchise tax: $800/year

3. Shareholders ALSO pay personal income tax on pass-through income
   (This creates partial double taxation)

CA Apportionment:
- Single-sales factor (same as C Corp)
- Economic nexus: $610,395 (2025)

State-Specific Adjustments:
- Add back: Federal S Corp deductions not allowed
- Subtract: CA municipal bond interest
```

**Decision Table**: Table 60000-60099 (CA S Corp) - Similar to C Corp CA tables but different tax rate (1.5% vs 8.84%)

### 6.3 New York S Corporation Example

**NY Treatment**:

```
Entity-Level Tax:
1. Fixed Dollar Minimum Tax (based on gross receipts, not income)
   - $0 - $100K: $25
   - $100K - $250K: $50
   - $250K - $500K: $175
   - $500K - $1M: $300
   - $1M - $5M: $1,000
   - $5M - $25M: $3,500
   - $25M+: $4,500

2. Optional: NY can impose entity-level tax on certain types of income

NY Apportionment:
- Three-factor (property/payroll/sales)
- Economic nexus thresholds apply

Shareholder Level:
- NY resident shareholders pay full NY tax on pass-through income
- Nonresident shareholders pay only on NY-sourced income
```

**Complexity**: Need to track shareholder residency to determine state tax treatment.

### 6.4 State Template Modifications

**S Corporation state templates need**:

```xml
<entity name="state_s_corp_config">
  <!-- Entity-Level Tax -->
  <field name='has_entity_level_tax' type='boolean'
         comment='Does state impose entity-level tax on S Corps?'/>
  <field name='entity_tax_rate' type='double'
         comment='Entity-level tax rate (if applicable, e.g., CA 1.5%)'/>
  <field name='minimum_tax' type='double'
         comment='Minimum franchise/excise tax (e.g., CA $800)'/>

  <!-- S Election Recognition -->
  <field name='recognizes_s_election' type='boolean'
         comment='Does state recognize federal S election?'/>
  <field name='separate_state_election_required' type='boolean'
         comment='Requires separate state S election?'/>

  <!-- Shareholder Taxation -->
  <field name='taxes_nonresident_shareholders' type='boolean'
         comment='Taxes nonresident shareholders on pass-through income?'/>
  <field name='composite_return_allowed' type='boolean'
         comment='Allows composite return filing for nonresident shareholders?'/>
  <field name='composite_return_rate' type='double'
         comment='Tax rate for composite returns (if allowed)'/>

  <!-- Withholding Requirements -->
  <field name='requires_shareholder_withholding' type='boolean'
         comment='Requires withholding on nonresident shareholder income?'/>
  <field name='withholding_rate' type='double'
         comment='Withholding rate on nonresident shareholder income'/>
</entity>
```

**Impact**: State implementation for S Corps is as complex as for C Corps, possibly more so due to shareholder-level tracking.

---

## 7. Decision Table Design

### 7.1 Core Decision Tables (Estimated: 25-30 tables)

#### Orchestration (10000-10099)
```
Table 10000: Compute_S_Corporation_Tax_Return
  - Main orchestrator
  - Calls income, deduction, special tax, Schedule K, Schedule M-2, distribution tables
  - Different from C Corp orchestrator (no regular tax calculation)
```

#### Income & Deductions (10100-10199 - Reuse some C Corp logic)
```
(Can largely reference C Corp tables 2000-3999 with minor modifications)
```

#### Built-In Gains Tax (10200-10299)
```
Table 10200: Determine_BIG_Applicability
  - Check if prior C Corp
  - Check if within 5-year recognition period
  - Output: BIG tax applies? (boolean)

Table 10210: Identify_BIG_Assets
  - For each asset sold:
    - Was it held at conversion date?
    - Calculate built-in gain (FMV at conversion - basis at conversion)
  - Output: Per-asset BIG

Table 10220: Calculate_Recognized_BIG
  - For each BIG asset sold:
    - Recognized gain = Sale price - basis
    - RBIG = min(recognized gain, built-in gain)
  - Output: Total RBIG for year

Table 10230: Apply_BIG_Taxable_Income_Limitation
  - Calculate S Corp taxable income (as if C Corp)
  - Net RBIG = min(total RBIG, taxable income)
  - Output: Net RBIG

Table 10240: Calculate_BIG_Tax
  - BIG tax = Net RBIG × 21%
  - Output: BIG tax liability

Table 10250: Adjust_Passthrough_For_BIG_Tax
  - Reduce ordinary income by BIG tax
  - Output: Adjusted income for Schedule K
```

#### Excess Net Passive Income Tax (10300-10399)
```
Table 10300: Determine_ENPI_Applicability
  - Check if C Corp E&P exists
  - If yes, continue; if no, ENPI tax = 0

Table 10310: Classify_Passive_Income
  - For each income item:
    - Is it passive? (dividends, interest, rents, royalties, etc.)
  - Output: Total passive investment income

Table 10320: Calculate_Passive_Income_Percentage
  - Passive % = Passive investment income / Gross receipts
  - If ≤ 25%, ENPI tax = 0
  - Output: Passive %, threshold exceeded?

Table 10330: Calculate_Net_Passive_Income
  - Net passive income = Passive income - Directly connected expenses
  - Output: Net passive income

Table 10340: Calculate_ENPI
  - ENPI = Net passive income × [(Passive income - 25% × Gross receipts) / Passive income]
  - Output: ENPI

Table 10350: Calculate_ENPI_Tax
  - ENPI tax = ENPI × 21%
  - Output: ENPI tax liability

Table 10360: Reduce_Passive_Income_Passthrough
  - Pro-rata reduction of each passive income item
  - Output: Adjusted passive income for Schedule K

Table 10370: Check_3_Year_Termination_Risk
  - Track consecutive years with passive income > 25%
  - If 3 consecutive years: Warning (S election terminates)
```

#### LIFO Recapture Tax (10400-10499)
```
Table 10400: Determine_LIFO_Recapture_Applicability
  - Check if LIFO inventory used at conversion
  - If yes, one-time tax applies

Table 10410: Calculate_LIFO_Reserve
  - LIFO reserve = FIFO value - LIFO basis
  - Output: LIFO reserve

Table 10420: Calculate_LIFO_Recapture_Tax
  - Total tax = LIFO reserve × 21%
  - Output: Total LIFO recapture tax

Table 10430: Calculate_LIFO_Installment
  - Annual installment = Total tax / 4
  - Track payment year (1 of 4, 2 of 4, etc.)
  - Output: Current year installment payment

Table 10440: Adjust_Inventory_Basis
  - New basis = LIFO basis + LIFO reserve
  - Output: Adjusted inventory basis
```

#### Schedule K Allocation (10500-10599)
```
Table 10500: Allocate_Ordinary_Income
  - Total ordinary income → per-share-per-day → each shareholder
  - Output: Shareholder K-1 Box 1

Table 10510: Allocate_Rental_Income
  - Net rental income → per-share-per-day → each shareholder
  - Output: Shareholder K-1 Box 2

Table 10520: Allocate_Interest_Dividends
  - Interest, dividends → per-share-per-day → each shareholder
  - Output: Shareholder K-1 Boxes 4, 5a, 5b

Table 10530: Allocate_Capital_Gains
  - Net ST/LT capital gains → per-share-per-day → each shareholder
  - Output: Shareholder K-1 Boxes 7, 8a

Table 10540: Allocate_Section_179
  - Section 179 deduction → per-share-per-day → each shareholder
  - Output: Shareholder K-1 Box 11

Table 10550: Allocate_Charitable_Contributions
  - Charitable contributions → per-share-per-day → each shareholder
  - Output: Shareholder K-1 Box 12

Table 10560: Allocate_Other_Items
  - Other income, deductions, credits → per-share-per-day
  - Output: Shareholder K-1 remaining boxes
```

#### Per-Share-Per-Day Allocation (10600-10699)
```
Table 10600: Calculate_Daily_Allocation_Rate
  - Daily amount = Total amount / 365 days
  - Output: Daily allocation rate per item

Table 10610: Determine_Shareholder_Ownership_Days
  - For each shareholder:
    - Days owned during year (1-365)
    - Shares owned each day
  - Handle mid-year changes (sales, purchases)
  - Output: Shareholder ownership matrix

Table 10620: Apply_Per_Share_Per_Day_Formula
  - For each shareholder, each item:
    - Allocated = Daily rate × Days owned × (Shares owned / Total shares)
  - Output: Shareholder allocations

Table 10630: Apply_Closing_Of_Books_Election
  - Alternative to per-share-per-day
  - If shareholder disposes of entire interest or > 20%:
    - Can elect to close books as of disposition date
  - Output: Shareholder allocations (closing-of-books method)
```

#### Schedule K-1 Generation (10700-10799)
```
Table 10700: Generate_K1_Part_I
  - Corporation information
  - Output: K-1 Part I (corp name, EIN, address)

Table 10710: Generate_K1_Part_II
  - Shareholder information
  - Beginning/ending ownership %
  - Beginning/ending stock basis (if determinable)
  - Output: K-1 Part II

Table 10720: Generate_K1_Part_III
  - All allocated items (Boxes 1-17)
  - From Schedule K allocation tables (10500-10599)
  - Output: K-1 Part III

Table 10730: Generate_K1_Supplemental_Info
  - Basis adjustments (Box 16)
  - Other information (Box 17)
  - Output: K-1 supplemental schedules
```

#### AAA/OAA/PTI Tracking (Schedule M-2) (10800-10899)
```
Table 10800: Update_AAA_For_Income
  - AAA beginning balance
  - + Ordinary income (Schedule K line 1)
  - + Other income (lines 2-10, except tax-exempt)
  - Output: AAA after income additions

Table 10810: Update_AAA_For_Deductions
  - - Loss and deduction items (Schedule K)
  - Exclude nondeductible items
  - Output: AAA after deductions

Table 10820: Update_AAA_For_Distributions
  - - Distributions (cannot go below zero)
  - Output: AAA ending balance

Table 10830: Update_OAA
  - OAA beginning balance
  - + Tax-exempt income
  - - Nondeductible expenses
  - Output: OAA ending balance

Table 10840: Update_PTI
  - PTI beginning balance (only for pre-1983 S Corps)
  - - Distributions from PTI
  - Output: PTI ending balance

Table 10850: Track_C_Corp_E_And_P
  - E&P beginning balance
  - - Distributions from E&P (if AAA exhausted)
  - Output: E&P ending balance
```

#### Distribution Ordering (10900-10999)
```
Table 10900: Determine_Distribution_Source
  - For each distribution:
    - Step 1: From AAA (tax-free, reduces AAA and basis)
    - Step 2: From PTI (tax-free if pre-1983, reduces PTI and basis)
    - Step 3: From E&P (taxable dividend if C Corp E&P exists)
    - Step 4: Return of capital (tax-free, reduces basis)
    - Step 5: Capital gain (if exceeds basis)
  - Output: Distribution sourcing by category

Table 10910: Apply_Distribution_To_AAA
  - Reduce AAA by distribution (not below zero)
  - Output: AAA after distribution, amount from AAA

Table 10920: Apply_Distribution_To_Stock_Basis
  - Reduce stock basis by non-dividend distributions
  - Cannot go below zero
  - Output: Stock basis after distribution

Table 10930: Calculate_Capital_Gain_From_Distribution
  - If distribution exceeds AAA + PTI + basis:
    - Excess = capital gain
  - Output: Capital gain amount (K-1 Box 8a)

Table 10940: Calculate_Dividend_From_Distribution
  - If C Corp E&P exists and AAA exhausted:
    - Distribution from E&P = dividend
  - Output: Dividend amount (shareholder Form 1099-DIV)
```

#### Shareholder Basis Calculation (11000-11099)
```
Table 11000: Calculate_Stock_Basis_Increases
  - Beginning stock basis
  - + Stock purchases
  - + Capital contributions
  - + Income items (ordinary income, tax-exempt income, etc.)
  - Output: Total stock basis increases

Table 11010: Calculate_Stock_Basis_Decreases
  - - Distributions (non-dividend, non-capital gain)
  - - Loss and deduction items
  - - Nondeductible, noncapital expenses
  - - Depletion
  - Output: Total stock basis decreases

Table 11020: Calculate_Ending_Stock_Basis
  - Ending stock basis = Beginning + Increases - Decreases
  - Cannot go below zero
  - Output: Ending stock basis

Table 11030: Calculate_Debt_Basis
  - Debt basis = Shareholder loans to S Corp
  - Increases: New loans
  - Decreases: Loan repayments, losses (if stock basis exhausted)
  - Output: Ending debt basis

Table 11040: Apply_Basis_Limitation_On_Losses
  - Allowed losses = min(allocated losses, stock basis + debt basis)
  - Excess losses carried forward
  - Output: Deductible losses, suspended losses
```

### 7.2 Total Table Count Estimate

| Category | Tables | Complexity |
|----------|--------|------------|
| Orchestration | 1 | Medium |
| BIG Tax | 6 | High |
| ENPI Tax | 7 | High |
| LIFO Recapture | 4 | Medium |
| Schedule K Allocation | 7 | High |
| Per-Share-Per-Day | 4 | Very High |
| K-1 Generation | 4 | Medium |
| Schedule M-2 (AAA/OAA/PTI) | 5 | High |
| Distribution Ordering | 5 | Very High |
| Shareholder Basis | 5 | High |
| **Total Federal** | **48** | **High** |
| **State S Corp** | **~150** (50 states × 3 tables avg) | **Medium** |
| **Grand Total** | **~200 tables** | |

**Comparison**: C Corporation implementation has ~30 federal tables. S Corporation requires **60% more** due to shareholder-centric complexity.

---

## 8. Complexity Assessment

### 8.1 Complexity Drivers

| Factor | C Corporation | S Corporation | Multiplier |
|--------|---------------|---------------|------------|
| **Entities** | 1 result entity | 1-100 shareholder entities | **100×** |
| **Tax calculation** | Single entity tax (21%) | Pass-through to N shareholders | **N×** |
| **Distributions** | Simple dividend | 5-step ordering (AAA→PTI→E&P→basis→gain) | **5×** |
| **Basis tracking** | Not required | Per-shareholder stock + debt basis | **N×** |
| **Schedules** | M-1, M-2 (retained earnings), L | M-1, M-2 (AAA/OAA/PTI), L, **K-1 × N** | **N×** |
| **State complexity** | Entity tax only | Entity tax + shareholder-level tracking | **2×** |
| **Special taxes** | None (except AMT, obsolete) | BIG, ENPI, LIFO recapture | **3×** |
| **Allocation logic** | N/A | Per-share-per-day (daily tracking) | **365×** |

**Overall Complexity**: S Corporation is **5-10× more complex** than C Corporation due to shareholder-centric model.

### 8.2 Data Volume Scaling

**Example: 10 Shareholders, 1-Year Period**

| Data Point | C Corporation | S Corporation | Ratio |
|------------|---------------|---------------|-------|
| **Result entities** | 1 | 10 (1 per shareholder) | 10× |
| **K-1 forms** | 0 | 10 | ∞ |
| **Basis calculations** | 0 | 10 | ∞ |
| **Daily allocations** | 0 | 3,650 (10 shareholders × 365 days) | ∞ |
| **Distribution records** | 1 (aggregate) | 10 (per shareholder) | 10× |

**Memory Impact**: For N shareholders:
- C Corp: ~600 KB per return
- S Corp: ~(600 + 100×N) KB per return
- **100 shareholders**: ~10.6 MB per return (17× more than C Corp)

### 8.3 Performance Implications

**Bottlenecks**:
1. **Per-share-per-day allocation** - O(N × 365) complexity
2. **K-1 generation** - O(N) forms to generate
3. **Distribution ordering** - O(N) basis calculations
4. **Basis tracking** - O(N) stock basis + O(N) debt basis calculations

**Mitigation Strategies**:
- Pre-calculate daily allocation rates (amortize 365-day loop)
- Parallelize K-1 generation (independent for each shareholder)
- Cache basis calculations (only recompute on changes)

**Estimated Performance**:
- C Corp: 1,000-1,500 returns/second (single-threaded)
- S Corp (1 shareholder): 800-1,200 returns/second
- S Corp (10 shareholders): 100-150 returns/second
- S Corp (100 shareholders): 10-20 returns/second

**Conclusion**: S Corporation processing is **50-100× slower** than C Corporation for 100-shareholder S Corps.

### 8.4 Testing Complexity

**Test Scenarios Required**:

| Scenario | Purpose | Complexity |
|----------|---------|------------|
| Single shareholder, full year | Baseline | Low |
| Multiple shareholders, full year | Per-share-per-day allocation | Medium |
| Mid-year shareholder change | Closing-of-books vs per-share-per-day | High |
| BIG tax (prior C Corp) | Entity-level tax, asset tracking | High |
| ENPI tax (C Corp E&P + passive income) | Complex income classification | High |
| LIFO recapture | One-time tax, 4-year payment | Medium |
| Distribution ordering | AAA→PTI→E&P→basis→gain | Very High |
| Zero/negative basis | Loss limitation, suspended losses | High |
| State composite return | Multi-state shareholder tracking | Very High |

**Total Test Cases Required**: 50-75 (vs 20-30 for C Corp)

---

## 9. Implementation Recommendations

### 9.1 Phased Approach

**Phase 1: Single-Shareholder Foundation** (3-4 weeks)
- Entities: S Corporation, Shareholder (single), Result
- Basic Schedule K allocation (simple pass-through)
- Schedule M-2 (AAA only, no C Corp E&P)
- No entity-level taxes (BIG, ENPI, LIFO)
- **Goal**: Functional single-shareholder S Corp return

**Phase 2: Multi-Shareholder & Per-Share-Per-Day** (2-3 weeks)
- Shareholder entity (1-100 shareholders)
- Per-share-per-day allocation logic
- Closing-of-books election (optional)
- K-1 generation for N shareholders
- **Goal**: Multi-shareholder allocation working

**Phase 3: Entity-Level Taxes** (2-3 weeks)
- BIG tax (prior C Corp conversion)
- ENPI tax (passive income + C Corp E&P)
- LIFO recapture tax
- Schedule M-2 (AAA, OAA, C Corp E&P)
- **Goal**: Complex conversion scenarios handled

**Phase 4: Distributions & Basis** (2-3 weeks)
- Distribution ordering (AAA→PTI→E&P→basis→gain)
- Stock basis tracking (Form 7203 logic)
- Debt basis tracking
- Basis limitation on losses
- **Goal**: Distribution taxation correct

**Phase 5: State S Corporation** (3-4 weeks)
- State entity-level taxes (CA 1.5%, etc.)
- Shareholder residency tracking
- Composite return support (optional)
- Nonresident shareholder withholding
- **Goal**: State S Corp compliance

**Phase 6: Testing & Optimization** (2-3 weeks)
- Comprehensive test suite (50+ scenarios)
- Performance optimization (K-1 generation, allocation)
- Edge cases (negative basis, 3-year termination, etc.)
- **Goal**: Production-ready

**Total: 14-20 weeks (3.5-5 months)**

### 9.2 Recommended Tooling

**Schedule K-1 Generation**:
- Decision tables generate K-1 data (boxes 1-17)
- Separate rendering layer (PDF/HTML) for K-1 forms
- Template-based approach (similar to tax forms)

**Per-Share-Per-Day Allocation**:
- Pre-computation: Calculate daily rates once per return
- Matrix storage: Shareholder × Day × Item (sparse matrix if mid-year changes rare)
- Vectorization: Batch allocate all items to all shareholders simultaneously

**Basis Tracking**:
- Dedicated basis calculation module (reusable across shareholders)
- Incremental updates (not full recalculation each time)
- Validation: Ending basis = Beginning + Increases - Decreases

### 9.3 Decision: Build vs Skip

**Arguments FOR Implementation**:
1. **Completeness**: S Corps are common (over 5 million U.S. S Corps)
2. **Market demand**: Tax software must support both C and S Corps
3. **Intellectual challenge**: Complex allocation logic tests DTRules capabilities
4. **Architecture demonstration**: Shows DTRules handles multi-entity scenarios

**Arguments AGAINST Implementation**:
1. **High complexity**: 5-10× more complex than C Corp
2. **Limited reuse**: Most C Corp tables don't apply
3. **Performance concerns**: 50-100× slower for 100-shareholder S Corps
4. **Niche use cases**: Most S Corps have 1-5 shareholders (simpler allocation)
5. **Alternative tools exist**: Existing tax software handles S Corps well

**Recommendation**: **Implement Phase 1-2 Only (Single/Multi-Shareholder, No Special Taxes)**

**Rationale**:
- Demonstrates DTRules can handle pass-through entities
- 80% of S Corps have < 5 shareholders (manageable complexity)
- Phases 1-2 take 5-7 weeks (vs 14-20 weeks for full implementation)
- Can defer Phases 3-5 (entity-level taxes, distributions) as "advanced features"

**Alternatively**: **Skip S Corp, Focus on Foreign Tax Credit (#337)**

**Rationale**:
- Foreign tax credit (#337) is C Corp enhancement (builds on existing work)
- S Corp is entirely new project (limited synergy)
- Corporate tax users more likely to need foreign tax credit than S Corp support

### 9.4 If Implementing: Quick Wins

**Simplification Strategies**:

1. **Restrict to < 10 Shareholders Initially**
   - Covers 90%+ of S Corps
   - Reduces performance concerns
   - Simplifies testing

2. **Per-Share-Per-Day Only (No Closing-of-Books)**
   - Closing-of-books is optional election
   - Simpler to have one allocation method
   - Can add later if needed

3. **No PTI Tracking**
   - PTI only applies to pre-1983 S Corps (rare)
   - Focus on AAA and C Corp E&P only
   - Reduces Schedule M-2 complexity

4. **Defer State Entity-Level Taxes**
   - Federal S Corp first
   - Add state entity taxes later (Phase 5)
   - Shareholder-level state tax is complex (nonresident, composite returns)

5. **Assume Full-Year Ownership**
   - Mid-year shareholder changes are complex
   - Phase 1: Assume no ownership changes during year
   - Phase 2: Add mid-year change support

**With Simplifications**: **6-10 weeks** instead of 14-20 weeks.

---

## 10. Alternative Approaches

### 10.1 Hybrid: S Corp "Lite"

**Scope**: Pass-through income allocation only, no special taxes

**Include**:
- Revenue and expense tracking (reuse C Corp)
- Depreciation (reuse C Corp tables 6000-6900)
- Schedule K income allocation (simple per-share-per-day)
- Schedule K-1 generation (basic)
- Schedule M-2 (AAA only, no C Corp E&P)

**Exclude**:
- BIG tax (assume no prior C Corp)
- ENPI tax (assume no C Corp E&P)
- LIFO recapture (assume no LIFO inventory)
- Distribution ordering (assume distributions never exceed AAA)
- Debt basis (stock basis only)
- State entity-level taxes

**Benefit**: **40-50% reduction in complexity**, delivers core S Corp functionality.

**Target Users**: S Corps with no C Corp history, < 10 shareholders.

**Estimated Effort**: 8-10 weeks (vs 14-20 weeks full implementation)

### 10.2 Partnership Approach

**Observation**: Partnerships (Form 1065) share similar pass-through logic to S Corps.

**If Building Generalized Pass-Through Framework**:
```
Common Pass-Through Module:
├── Per-partner/shareholder allocation
├── Basis tracking
├── K-1 generation
├── Schedule M-2 (capital account analysis)
└── Distribution ordering

S Corporation Specific:
├── AAA/OAA tracking
├── BIG/ENPI/LIFO taxes
├── 100-shareholder limit
└── Single share class requirement

Partnership Specific:
├── Multiple capital account methods
├── Section 754 basis adjustments
├── Hot asset rules (Section 751)
└── Guaranteed payments
```

**Benefit**: Reusable framework for S Corp, Partnerships, LLCs (taxed as partnerships).

**Downside**: Even higher initial complexity (partnership rules are MORE complex than S Corp).

**Recommendation**: **Not advised** unless targeting full pass-through entity suite.

### 10.3 Shareholder-Level Focus

**Alternative Framing**: Instead of "Form 1120-S Implementation," build "Schedule K-1 Analysis Tool."

**Scope**:
- **Input**: Schedule K-1 (received by shareholder)
- **Output**: Individual Form 1040 integration (how K-1 items flow to 1040)
- **Focus**: Shareholder tax calculation, not S Corp return preparation

**Benefit**:
- Complements existing TaxReturn (Form 1040) implementation
- Useful for individual taxpayers receiving K-1s
- Avoids entity-level complexity (BIG, ENPI, AAA)

**Estimated Effort**: 4-6 weeks (much simpler than full S Corp)

**Use Case**: Individual tax software that needs to handle K-1 income (from S Corps, partnerships, trusts).

---

## Conclusion & Final Recommendation

### Summary of Findings

1. **S Corporation Fundamentally Different**: Pass-through taxation requires shareholder-centric model, not entity-centric like C Corp.

2. **High Complexity**: 5-10× more complex than C Corporation due to:
   - Per-shareholder allocation (1-100 shareholders)
   - Per-share-per-day allocation logic
   - Basis tracking (stock + debt)
   - Distribution ordering (5-step AAA→PTI→E&P→basis→gain)
   - Entity-level special taxes (BIG, ENPI, LIFO)
   - State variance (entity-level tax in some states)

3. **Limited Reuse**: Only ~30% of C Corp tables (revenue, expense, depreciation) reusable. Tax calculation, credits, apportionment, Schedule M-2 completely different.

4. **Significant Effort**: Full implementation 14-20 weeks, simplified implementation 6-10 weeks.

### Recommended Path Forward

**Option A: Defer S Corporation Support**
- **Focus on**: Foreign Tax Credit (#337) - C Corp enhancement, builds on existing work
- **Rationale**: S Corp is separate project with limited synergy to C Corp implementation
- **Timeline**: Revisit S Corp after Form 1118 (foreign tax credit) complete

**Option B: Implement S Corp "Lite" (Single-Shareholder Only)**
- **Scope**: Phase 1 only - single shareholder, basic pass-through, no special taxes
- **Benefit**: Demonstrates DTRules can handle pass-through entities
- **Effort**: 3-4 weeks
- **Future**: Can expand to multi-shareholder later if demand exists

**Option C: Full S Corporation Implementation**
- **Scope**: All 6 phases (single shareholder → multi-shareholder → special taxes → distributions → state → testing)
- **Effort**: 14-20 weeks
- **Justification**: Only if S Corp support is critical business requirement

### My Recommendation: **Option A - Defer S Corporation**

**Reasons**:
1. S Corporation requires fundamentally different architecture (shareholder-centric vs entity-centric)
2. Limited reuse of existing C Corp implementation (only depreciation tables)
3. High complexity (48 federal tables vs 27 for C Corp) with diminishing returns
4. Foreign Tax Credit (#337) is better next step - enhances existing C Corp, reuses apportionment logic
5. S Corp can be separate sub-project if business case emerges

**If Proceeding with S Corp**:
- Start with **Option B** (single shareholder, 3-4 weeks)
- Validate approach before committing to full multi-shareholder implementation
- Consider **S Corp "Lite"** (skip BIG/ENPI/LIFO) to reduce scope by 40-50%

---

## Appendix: Key IRS References

**Forms**:
- Form 1120-S - U.S. Income Tax Return for an S Corporation
- Schedule K-1 (Form 1120-S) - Shareholder's Share of Income, Deductions, Credits, etc.
- Form 7203 - S Corporation Shareholder Stock and Debt Basis Limitations
- Form 1120 - U.S. Corporation Income Tax Return (for comparison)

**IRC Sections**:
- IRC § 1361 - S Corporation defined
- IRC § 1362 - Election; revocation; termination
- IRC § 1363 - Effect of election on corporation (including LIFO recapture)
- IRC § 1366 - Pass-through of items to shareholders
- IRC § 1367 - Adjustments to basis of stock of shareholders
- IRC § 1368 - Distributions
- IRC § 1374 - Tax imposed on certain built-in gains
- IRC § 1375 - Tax imposed when passive investment income exceeds 25%
- IRC § 1377 - Definitions and special rule (per-share-per-day allocation)

**IRS Publications**:
- Publication 542 - Corporations
- Publication 589 - Tax Information on S Corporations

**Court Cases**:
- South Dakota v. Wayfair, 138 S. Ct. 2080 (2018) - Economic nexus (applies to S Corps at state level)

---

**Document Version**: 1.0
**Date**: 2026-03-24
**Author**: DTRules Design Team
**Status**: Research & Design Complete, Implementation Pending Decision

---

## Sources

- [About Form 1120-S, U.S. Income Tax Return for an S Corporation | Internal Revenue Service](https://www.irs.gov/forms-pubs/about-form-1120-s)
- [Form 1120-S, U.S. Income Tax Return for an S Corporation](https://www.irs.gov/pub/irs-pdf/f1120s.pdf)
- [Instructions for Form 1120-S (2025) | Internal Revenue Service](https://www.irs.gov/instructions/i1120s)
- [Shareholder's Instructions for Schedule K-1 (Form 1120-S) (2025) | Internal Revenue Service](https://www.irs.gov/instructions/i1120ssk)
- [S Corp Built-In Gains Tax Rules and Strategies](https://www.upcounsel.com/s-corp-built-in-gains-tax)
- [S Corporation Tax Guide 2025](https://pro.bloombergtax.com/insights/corporate-tax-planning/a-comprehensive-guide-to-s-corporation-taxes/)
- [26 U.S. Code § 1374 - Tax imposed on certain built-in gains](https://www.law.cornell.edu/uscode/text/26/1374)
- [26 U.S. Code § 1375 - Tax imposed when passive investment income exceeds 25%](https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title26-section1375&num=0&edition=prelim)
- [26 CFR § 1.1375-1 - Tax imposed when passive investment income exceeds 25%](https://www.law.cornell.edu/cfr/text/26/1.1375-1)
- [LIFO Recapture on Conversion from C to S Corporation](https://answerconnect.cch.com/document/arp28d2d646b87b6d1000a3be00237de5959c0f1/federal/irc/explanation/lifo-recapture-on-conversion-from-c-to-s-corporation)
- [26 CFR § 1.1363-2 - Recapture of LIFO benefits](https://www.law.cornell.edu/cfr/text/26/1.1363-2)
- [Understanding Schedule M-2 for S Corporations](https://www.vintti.com/blog/schedule-m-2-form-1120-s-analysis-of-accumulated-adjustments-account-other-adjustments-account-and-shareholders-undistributed-taxable-income-previously-taxed)
- [Making tax-free distributions to the extent of AAA](https://www.thetaxadviser.com/issues/2022/apr/making-tax-free-distributions-extent-aaa/)
- [California S Corp Requirements: Ultimate 2025 Guide](https://greinerlawcorp.com/california-s-corp-complete-guide/)
- [State Corporate Income Tax Rates and Brackets, 2026](https://taxfoundation.org/data/all/state/state-corporate-income-tax-rates-brackets/)
- [Allocating Income Using the Closing of the Books Method](https://pselaw.com/allocating-income-using-the-closing-of-the-books-method/)
- [Allocating Passthrough Items to S Corporation Shareholders](https://www.thetaxadviser.com/issues/2008/dec/allocatingpassthroughitemstoscorporationshareholders/)
