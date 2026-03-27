# Consolidated Returns Design Document

**Project**: DTRules Corporate Tax - Consolidated Returns Support
**Date**: 2026-03-24
**Status**: Design Phase - Implementation Not Started
**Related Issue**: #339

---

## Executive Summary

This document provides a comprehensive design for implementing consolidated corporate tax returns in DTRules. Consolidated returns allow affiliated groups of corporations (80%+ ownership) to file a single combined tax return, requiring complex intercompany eliminations, basis adjustments, and multi-entity coordination.

**Key Challenge**: Consolidated returns represent the most complex feature in corporate taxation, requiring fundamental architectural changes to support multiple related entities, intercompany transaction tracking, and sophisticated elimination logic.

**Recommendation**: **Defer implementation** until Phase 5 or later. This is a major undertaking requiring 8-12 weeks of development and represents a distinct product offering beyond basic corporate tax compliance.

---

## Table of Contents

1. [Regulatory Framework](#regulatory-framework)
2. [Eligibility Requirements](#eligibility-requirements)
3. [Consolidation Mechanics](#consolidation-mechanics)
4. [Entity Architecture Requirements](#entity-architecture-requirements)
5. [Decision Table Design](#decision-table-design)
6. [Forms and Schedules](#forms-and-schedules)
7. [Complexity Factors](#complexity-factors)
8. [Implementation Phases](#implementation-phases)
9. [Integration Challenges](#integration-challenges)
10. [Recommendations](#recommendations)

---

## 1. Regulatory Framework

### Primary Code Sections

- **IRC § 1501** - Privilege to file consolidated returns
- **IRC § 1502** - Consolidated return regulations (Treasury has broad authority)
- **IRC § 1503** - Computation and payment of tax
- **IRC § 1504** - Definitions of affiliated group

### Key Treasury Regulations

- **Reg. § 1.1502-13** - Intercompany transactions (matching and acceleration rules)
- **Reg. § 1.1502-19** - Excess loss accounts (ELA)
- **Reg. § 1.1502-21** - Net operating losses (SRLY rules)
- **Reg. § 1.1502-32** - Investment basis adjustments
- **Reg. § 1.1502-75** - Filing requirements
- **Reg. § 1.1502-76** - Taxable year of members

### Recent Updates

- **December 2024**: Treasury finalized major revisions to consolidated return regulations, modernizing language and enhancing clarity
- **August 2024**: New dual consolidated loss regulations addressing Pillar Two OECD rules
- **2020**: Consolidated NOL rules updated post-TCJA

### References

- [26 USC § 1504 - Affiliated Group Defined](https://www.law.cornell.edu/uscode/text/26/1504)
- [26 CFR § 1.1502-13 - Intercompany Transactions](https://www.law.cornell.edu/cfr/text/26/1.1502-13)
- [26 CFR § 1.1502-32 - Investment Adjustments](https://www.law.cornell.edu/cfr/text/26/1.1502-32)

---

## 2. Eligibility Requirements

### 2.1 Affiliated Group Definition (IRC § 1504)

An **affiliated group** consists of:

1. **Common parent corporation** that directly owns stock meeting both:
   - At least **80% of total voting power**, AND
   - At least **80% of total fair market value**

2. **Chain of ownership**: Each subsidiary must be 80%+ owned (voting and value) by one or more group members

3. **Continuous ownership**: 80% threshold must be maintained throughout the tax year

### 2.2 Includible Corporations

**Eligible** (can be included):
- Domestic C corporations
- Domestic insurance companies (with special 5-year waiting period under IRC § 1504(c)(2))

**Excluded** (cannot be included):
- Tax-exempt organizations (IRC § 501)
- REITs (Real Estate Investment Trusts)
- RICs (Regulated Investment Companies)
- S Corporations
- Foreign corporations (even if domestic parent owns 80%+)
- Certain insurance companies during waiting period

### 2.3 Stock Ownership Rules

**What counts as "stock"**:
- Common stock (voting and non-voting)
- Non-participating preferred stock with limited dividend rights

**What doesn't count**:
- Certain preferred stock that:
  - Is limited and preferred as to dividends
  - Does not participate in corporate growth
  - Has redemption/liquidation rights not exceeding issue price

### 2.4 Affiliation Testing Requirements

The system must test affiliation at multiple points:

1. **Beginning of year** - Determine initial group membership
2. **Mid-year acquisitions** - Test when new member joins
3. **Mid-year dispositions** - Determine when member leaves
4. **Year-end** - Validate continuous ownership throughout year

### Decision Table Requirements

**Table 10000**: `Test_Affiliation_Eligibility`
- Input: Ownership percentage (voting), ownership percentage (value), corporation type
- Output: Eligible for consolidation (boolean), exclusion reason
- Rules:
  - If voting < 80% OR value < 80% → Not eligible
  - If foreign corporation → Excluded
  - If REIT or RIC → Excluded
  - If tax-exempt → Excluded
  - If S Corporation → Excluded
  - Otherwise → Eligible

**Table 10100**: `Determine_Common_Parent`
- Input: Group member entities
- Output: Common parent entity ID
- Rules:
  - Parent is not 80%+ owned by any other domestic corporation
  - Parent owns at least one subsidiary meeting 80% test

**Table 10200**: `Build_Affiliated_Group_Chain`
- Input: Common parent, potential subsidiaries
- Output: List of included members, ownership percentages
- Rules:
  - Direct ownership ≥ 80% (voting and value) → Included
  - Indirect ownership through chain ≥ 80% → Included
  - Tiered structures require cumulative 80% test

### References

- [26 USC § 1504 - Definitions](https://www.law.cornell.edu/uscode/text/26/1504)
- [Accounting Insights - Section 1504 Affiliated Groups](https://accountinginsights.org/what-is-section-1504-and-how-does-it-define-affiliated-groups/)

---

## 3. Consolidation Mechanics

### 3.1 Core Principle: Single Entity Treatment

The consolidated group is treated as a **single economic entity** for tax purposes. The fundamental goal is to prevent:
- **Double counting** of income/gains within the group
- **Double deduction** of expenses/losses
- **Artificial income** from intercompany transactions

### 3.2 Intercompany Eliminations (Reg. § 1.1502-13)

Intercompany transactions are treated using a **matching and acceleration** system:

#### Matching Rule
The selling member's (S) gain/loss is deferred until the buying member (B) takes the property outside the group (sale to third party, depreciation, etc.). The timing and character of S's deferred gain/loss matches B's income recognition.

#### Acceleration Rule
If the matching rule cannot apply (e.g., B leaves the group), S's deferred gain/loss is accelerated and recognized immediately.

#### Types of Intercompany Transactions

1. **Inventory Sales**
   - S sells inventory to B for $100 (cost basis $60)
   - S's $40 gain is deferred
   - When B sells to third party for $120:
     - B recognizes $20 gain ($120 - $100)
     - S's $40 deferred gain is taken into account
     - Consolidated group reports $60 total gain ($120 - $60 original cost)

2. **Fixed Asset Sales**
   - S sells equipment to B for $50 (basis $30, accumulated depreciation $10)
   - S's $30 gain ($50 - $20 adjusted basis) is deferred
   - B depreciates over 5 years → S's gain is recognized proportionally
   - If B later sells to third party, remaining S gain is accelerated

3. **Service Transactions**
   - Generally taken into account immediately (no deferral)
   - Example: S provides IT services to B for $100
   - S recognizes $100 income, B deducts $100 expense
   - Net effect on consolidated income: $0

4. **Dividend Payments**
   - Dividends between group members are eliminated
   - S pays $10,000 dividend to M (parent)
   - S has dividend paid deduction: $10,000
   - M has dividend received income: $10,000
   - Consolidated elimination: Both offset to $0

5. **Interest Payments**
   - S pays $5,000 interest to M on intercompany loan
   - S deducts $5,000, M includes $5,000 income
   - Net consolidated effect: $0
   - Subject to IRC § 163(j) limitations at consolidated level

### 3.3 Investment Basis Adjustments (Reg. § 1.1502-32)

Parent's basis in subsidiary stock is adjusted to reflect subsidiary's economic performance:

#### Positive Adjustments (Increase Basis)
- Subsidiary's taxable income
- Subsidiary's tax-exempt income
- Subsidiary's excess of deductions over losses

#### Negative Adjustments (Decrease Basis)
- Subsidiary's losses
- Subsidiary's noncapital, nondeductible expenses
- Subsidiary's distributions to parent

#### Purpose
Prevent double taxation or double benefit on later stock sale.

**Example**:
- M forms S with $100 contribution
- S earns $50 taxable income (consolidated group pays tax)
- M's basis in S stock increases to $150
- If M sells S stock for $150, no additional gain recognized
- Without adjustment, M would pay tax twice on S's $50 income

#### Tiering Adjustments
In multi-tier structures (M → S → T):
1. T's income increases S's basis in T stock
2. S's basis increase in T stock increases M's basis in S stock
3. Adjustments "tier up" through ownership chain

### 3.4 Excess Loss Accounts (ELA) (Reg. § 1.1502-19)

When basis adjustments reduce stock basis below zero, an **excess loss account** is created.

#### How ELAs Arise
- Subsidiary generates NOLs absorbed by group
- Subsidiary makes distributions exceeding basis
- Most common: Debt-financed subsidiary operations with losses

#### ELA Triggers (Income Recognition)
ELA is recaptured as ordinary income when:
1. Parent disposes of subsidiary stock (sale, exchange)
2. Subsidiary is deconsolidated (leaves the group)
3. Subsidiary stock becomes worthless
4. Subsidiary makes certain distributions

#### Example
- M capitalizes S with $100
- S borrows $200 and makes $200 distribution to M
- S generates $150 NOL absorbed by group
- Basis adjustments:
  - Distribution: $100 - $200 = ($100) → ELA of $100
  - NOL: ELA of $100 - $150 = ($50) → ELA increases to $250
- If M sells S stock:
  - ELA of $250 is recaptured as ordinary income
  - Plus any actual gain/loss on stock sale

### 3.5 Consolidated NOL Rules (Reg. § 1.1502-21)

#### SRLY (Separate Return Limitation Year) Rules

**Purpose**: Prevent "trafficking in losses" by limiting use of losses generated before a corporation joined the group.

**SRLY Limitation**:
- Losses generated by a member in separate return years (before joining group)
- Can only offset that member's contribution to consolidated taxable income
- Tracked using a "SRLY register" for each member

**Example**:
- Corporation T has $1,000 NOL carryforward from pre-acquisition years
- M acquires T on Jan 1, 2025 (T joins consolidated group)
- In 2025:
  - T contributes $200 to consolidated taxable income
  - Other members contribute $800 (total CTI = $1,000)
- T's SRLY NOL can offset only $200 (T's contribution)
- Remaining $800 NOL carries forward under SRLY limitation

#### Interaction with IRC § 382

IRC § 382 limits NOL usage following an ownership change (50%+ shift over 3 years).

**Ordering Rule**:
- Apply § 382 limitation first (annual limitation amount)
- Then apply SRLY limitation
- Use the more restrictive limitation

**SRLY Exception**:
If the ownership change and SRLY event occur simultaneously (within 6 months), SRLY rules don't apply - only § 382 applies.

#### Consolidated NOL Carryforward
- Group-level NOLs (generated after consolidation) can offset any member's income
- No SRLY limitation for consolidated return year losses
- Subject to consolidated § 382 limitation if ownership change occurs

### 3.6 Consolidated Taxable Income Calculation

**Formula**:
```
Consolidated Taxable Income =
  Sum of each member's separate taxable income/loss
  - Intercompany eliminations
  + Restored intercompany items
  + Consolidated items (charitable contributions, dividends received deduction)
```

**Multi-Step Process**:

1. **Compute Separate Taxable Income** for each member
   - As if filing separate return
   - Before intercompany eliminations

2. **Apply Intercompany Eliminations**
   - Defer intercompany gains on inventory/asset sales
   - Eliminate intercompany dividends
   - Eliminate intercompany interest/royalties/rents

3. **Calculate Consolidated Items**
   - Charitable contribution limitation (10% of consolidated TI)
   - Dividends received deduction (applied at consolidated level)
   - IRC § 163(j) interest limitation (30% of consolidated ATI)

4. **Apply Consolidated Deductions**
   - Consolidated NOL carryforwards (subject to SRLY)
   - Consolidated § 199A deduction (if applicable)

5. **Result**: Consolidated Taxable Income

### Decision Table Requirements

**Table 10300**: `Identify_Intercompany_Transactions`
- Input: Transaction type, seller entity, buyer entity
- Output: Intercompany flag, elimination treatment
- Rules:
  - If seller and buyer both in group → Intercompany = true
  - If transaction type = inventory sale → Treatment = deferred gain/loss
  - If transaction type = dividend → Treatment = complete elimination
  - If transaction type = interest/rent/royalty → Treatment = complete elimination
  - If transaction type = service → Treatment = immediate (no deferral)

**Table 10400**: `Calculate_Intercompany_Deferred_Gain_Loss`
- Input: Sale price, seller's basis, buyer entity, asset type
- Output: Deferred amount, restoration schedule
- Rules:
  - Deferred gain/loss = Sale price - Seller's adjusted basis
  - If asset = inventory → Restore when buyer sells to third party
  - If asset = depreciable → Restore pro-rata with buyer's depreciation
  - If asset = non-depreciable → Restore when buyer disposes

**Table 10500**: `Compute_Investment_Basis_Adjustment`
- Input: Subsidiary income/loss, distributions, tax-exempt income, nondeductible expenses
- Output: Basis adjustment amount, new basis, ELA amount
- Rules:
  - Positive adjustments = taxable income + tax-exempt income
  - Negative adjustments = losses + distributions + nondeductible expenses
  - If new basis < 0 → Create/increase ELA
  - If ELA exists and new basis > 0 → Reduce ELA first

**Table 10600**: `Tier_Up_Basis_Adjustments`
- Input: Lower-tier entity, middle-tier entity, top-tier entity
- Output: Cascaded basis adjustments
- Rules:
  - T's adjustment increases S's basis in T
  - S's increased T basis increases M's basis in S
  - Continue up ownership chain to common parent

**Table 10700**: `Apply_SRLY_Limitation`
- Input: Member's SRLY NOL, member's contribution to CTI, years in group
- Output: Usable SRLY NOL, SRLY register balance
- Rules:
  - If years in group = 0 (first year) → All NOL is SRLY
  - Maximum SRLY usage = Member's positive contribution to CTI
  - Remaining SRLY NOL carries forward with limitation

**Table 10800**: `Calculate_Consolidated_Taxable_Income`
- Input: All member separate TI, intercompany eliminations, consolidated items
- Output: Consolidated TI
- Rules:
  - Sum all member separate TI
  - Subtract deferred intercompany gains
  - Add restored intercompany gains
  - Apply consolidated limitations (charitable, interest)
  - Subtract consolidated NOL carryforwards

### References

- [26 CFR § 1.1502-13 - Intercompany Transactions](https://www.law.cornell.edu/cfr/text/26/1.1502-13)
- [CPA Exams Mastery - Intercompany Eliminations](https://cpaexamsmastery.com/tcp/3/9/2/)
- [26 CFR § 1.1502-32 - Investment Adjustments](https://www.ecfr.gov/current/title-26/chapter-I/subchapter-A/part-1/subject-group-ECFRbf74180a0ad35b8/section-1.1502-32)
- [Tax Adviser - Managing Excess Loss Accounts](https://www.thetaxadviser.com/issues/2019/may/managing-excess-loss-accounts/)
- [26 CFR § 1.1502-21 - Net Operating Losses](https://www.law.cornell.edu/cfr/text/26/1.1502-21)
- [CPA Exams Mastery - SRLY Rules](https://cpaexamsmastery.com/tcp/3/9/3/)

---

## 4. Entity Architecture Requirements

### 4.1 Current Architecture Limitations

The existing CorporateTax implementation supports **single-corporation** returns only:

**Current Entities** (7):
- `corporation` - Single company information
- `revenue` - Single company income
- `expense` - Single company deductions
- `asset` - Depreciation tracking
- `apportionment` - State tax allocation
- `result` - Tax calculation results
- `job` - Test metadata

**Problem**: No support for:
- Multiple related corporations
- Parent-subsidiary relationships
- Intercompany transactions
- Ownership tracking
- Consolidated eliminations

### 4.2 Required Entity Model for Consolidated Returns

#### New Entities Needed

**1. `affiliated_group`** (Group-level information)

```xml
<entity name="affiliated_group">
  <field name='group_id' type='string' comment='Unique group identifier'/>
  <field name='group_name' type='string' comment='Common parent name'/>
  <field name='common_parent_ein' type='string' comment='Parent EIN'/>
  <field name='tax_year' type='integer' comment='Consolidation tax year'/>
  <field name='election_year' type='integer' comment='Year consolidation elected'/>
  <field name='filing_status' type='string' comment='consolidated or separate'/>

  <!-- Consolidated Results -->
  <field name='consolidated_taxable_income' type='double' default_value='0.0'/>
  <field name='consolidated_tax' type='double' default_value='0.0'/>
  <field name='consolidated_credits' type='double' default_value='0.0'/>
  <field name='consolidated_payments' type='double' default_value='0.0'/>
  <field name='consolidated_refund_or_owed' type='double' default_value='0.0'/>
</entity>
```

**2. `group_member`** (Individual corporation within group)

```xml
<entity name="group_member">
  <field name='member_id' type='string' comment='Unique member identifier'/>
  <field name='ein' type='string' comment='Employer Identification Number'/>
  <field name='corporation_name' type='string' comment='Legal name'/>
  <field name='parent_member_id' type='string' comment='Direct parent member ID (null if common parent)'/>
  <field name='ownership_percentage_voting' type='double' comment='% voting stock owned by parent'/>
  <field name='ownership_percentage_value' type='double' comment='% FMV stock owned by parent'/>
  <field name='date_joined_group' type='date' comment='Date became affiliated'/>
  <field name='date_left_group' type='date' comment='Date deconsolidated (null if still member)'/>

  <!-- Separate Company Information -->
  <field name='separate_taxable_income' type='double' default_value='0.0' comment='Before consolidation'/>
  <field name='contribution_to_consolidated_income' type='double' default_value='0.0' comment='After eliminations'/>

  <!-- Stock Basis Tracking (for parent's investment) -->
  <field name='stock_basis_beginning' type='double' default_value='0.0' comment='Parent basis at year start'/>
  <field name='stock_basis_adjustments' type='double' default_value='0.0' comment='Current year adjustments'/>
  <field name='stock_basis_ending' type='double' default_value='0.0' comment='Parent basis at year end'/>
  <field name='excess_loss_account' type='double' default_value='0.0' comment='ELA if basis < 0'/>

  <!-- SRLY Tracking -->
  <field name='srly_nol_carryforward' type='double' default_value='0.0' comment='Pre-acquisition NOLs'/>
  <field name='srly_register_balance' type='double' default_value='0.0' comment='Cumulative contribution to CTI'/>
  <field name='srly_nol_utilized_current_year' type='double' default_value='0.0'/>
</entity>
```

**3. `intercompany_transaction`** (Transactions between members)

```xml
<entity name="intercompany_transaction">
  <field name='transaction_id' type='string' comment='Unique transaction identifier'/>
  <field name='transaction_type' type='string' comment='sale_inventory, sale_asset, dividend, interest, rent, royalty, service'/>
  <field name='transaction_date' type='date' comment='Date of transaction'/>

  <!-- Parties -->
  <field name='selling_member_id' type='string' comment='Selling/paying member'/>
  <field name='buying_member_id' type='string' comment='Buying/receiving member'/>

  <!-- Financial Details -->
  <field name='transaction_amount' type='double' comment='Sale price / payment amount'/>
  <field name='seller_basis' type='double' comment='Seller adjusted basis (for asset sales)'/>
  <field name='gain_or_loss' type='double' comment='Seller gain/loss'/>

  <!-- Deferral/Elimination -->
  <field name='requires_deferral' type='boolean' default_value='false' comment='Gain/loss deferred?'/>
  <field name='deferred_amount' type='double' default_value='0.0' comment='Deferred gain/loss'/>
  <field name='deferred_balance' type='double' default_value='0.0' comment='Remaining deferred amount'/>
  <field name='restored_current_year' type='double' default_value='0.0' comment='Amount restored to income'/>
  <field name='restoration_trigger' type='string' comment='third_party_sale, depreciation, deconsolidation'/>

  <!-- Asset Tracking (for asset sales) -->
  <field name='asset_id' type='string' comment='Asset identifier if applicable'/>
  <field name='asset_type' type='string' comment='inventory, equipment, building, intangible'/>
  <field name='buyer_depreciable' type='boolean' default_value='false' comment='Buyer depreciates?'/>
  <field name='buyer_recovery_period' type='integer' comment='Depreciation years'/>
</entity>
```

**4. `elimination_entry`** (Consolidation eliminations)

```xml
<entity name="elimination_entry">
  <field name='elimination_id' type='string' comment='Unique identifier'/>
  <field name='elimination_type' type='string' comment='intercompany_gain, dividend, interest, inventory, other'/>
  <field name='related_transaction_id' type='string' comment='Link to intercompany_transaction'/>

  <!-- Amounts -->
  <field name='elimination_amount' type='double' comment='Amount eliminated from consolidated income'/>
  <field name='income_reduction' type='double' comment='Reduction to selling member income'/>
  <field name='expense_reduction' type='double' comment='Reduction to buying member expense'/>

  <!-- Description -->
  <field name='description' type='string' comment='Elimination description for audit trail'/>
</entity>
```

**5. Enhanced `corporation`** (Modified for consolidation)

Existing `corporation` entity needs additional fields:

```xml
<!-- Add to existing corporation entity -->
<field name='is_group_member' type='boolean' default_value='false' comment='Part of affiliated group?'/>
<field name='affiliated_group_id' type='string' comment='Group ID if member'/>
<field name='member_id' type='string' comment='Member identifier within group'/>
```

### 4.3 Entity Relationships

```
affiliated_group (1)
    |
    +-- group_member (N) [one-to-many]
    |        |
    |        +-- corporation (1) [one-to-one, extended]
    |        +-- revenue (1)
    |        +-- expense (N)
    |        +-- asset (N)
    |
    +-- intercompany_transaction (N) [many-to-many through members]
    |        |
    |        +-- selling_member → group_member
    |        +-- buying_member → group_member
    |
    +-- elimination_entry (N) [one-to-many]
             |
             +-- related_transaction → intercompany_transaction
```

### 4.4 Data Flow Architecture

**Step 1: Group Formation**
```
Input: Corporation entities with ownership data
  ↓
Table 10000-10200: Test affiliation, build group
  ↓
Output: affiliated_group entity + group_member entities
```

**Step 2: Separate Company Calculations**
```
For each group_member:
  Input: Member's revenue, expense, asset data
    ↓
  Tables 2000-4000: Calculate separate taxable income (current logic)
    ↓
  Output: member.separate_taxable_income
```

**Step 3: Intercompany Transaction Processing**
```
Input: intercompany_transaction entities
  ↓
Table 10300-10400: Identify transactions, calculate deferrals
  ↓
Output: Updated deferred_amount, elimination_entry entities
```

**Step 4: Investment Basis Adjustments**
```
For each group_member (bottom-up in tiered structures):
  Input: Member income/loss, distributions, tax-exempt income
    ↓
  Table 10500-10600: Calculate basis adjustments, tier up
    ↓
  Output: Updated stock_basis_ending, excess_loss_account
```

**Step 5: Consolidated Income Calculation**
```
Input: All member separate TI + eliminations + SRLY NOLs
  ↓
Table 10700-10800: Apply SRLY, calculate consolidated TI
  ↓
Output: affiliated_group.consolidated_taxable_income
```

**Step 6: Consolidated Tax and Credits**
```
Input: Consolidated TI, consolidated credits
  ↓
Tables 4100, 5300: Calculate tax, apply credits (existing logic)
  ↓
Output: affiliated_group.consolidated_tax, refund_or_owed
```

### 4.5 Architectural Challenges

**Challenge 1: Multi-Entity Context**
- Current DTRules assumes single entity context
- Consolidated returns require iterating over member collection
- Need "for each member" looping constructs in decision tables

**Challenge 2: Relationship Traversal**
- Parent-subsidiary relationships require hierarchical queries
- Tiered structures (M → S → T) need recursive basis adjustments
- DTRules may not support graph traversal natively

**Challenge 3: Transaction Matching**
- Intercompany transactions require matching seller and buyer entities
- Need to track transaction lifecycle (deferral → restoration)
- Temporal tracking across multiple tax years

**Challenge 4: State Separation**
- Each member has independent state tax calculations
- State apportionment factors vary by member
- Consolidated returns are federal-only (most states require separate/combined)

---

## 5. Decision Table Design

### 5.1 Table Numbering Allocation

Building on existing Corporate Tax numbering:

| Range | Purpose | Status |
|-------|---------|--------|
| 1000-9999 | Single corporation (current) | ✅ Implemented |
| **10000-10999** | **Affiliation & Group Formation** | ⏳ New |
| **11000-11999** | **Intercompany Transactions** | ⏳ New |
| **12000-12999** | **Basis Adjustments & ELA** | ⏳ New |
| **13000-13999** | **SRLY & Consolidated NOL** | ⏳ New |
| **14000-14999** | **Consolidated Income & Tax** | ⏳ New |
| **15000-15999** | **Form 851 & Schedules** | ⏳ New |
| 50000-99999 | State tables | 🚧 Partial |

### 5.2 Detailed Table Specifications

#### 10000 Series: Affiliation & Group Formation

**Table 10000**: `Test_Affiliation_Eligibility`
- **Purpose**: Determine if corporation can be included in affiliated group
- **Inputs**:
  - `ownership_percentage_voting` (double)
  - `ownership_percentage_value` (double)
  - `corporation_type` (string: domestic_c_corp, foreign, reit, ric, s_corp, tax_exempt)
- **Outputs**:
  - `eligible_for_consolidation` (boolean)
  - `exclusion_reason` (string)
- **Rules**:
  1. If `corporation_type` = foreign → Not eligible (exclusion: foreign corporation)
  2. If `corporation_type` = reit OR ric → Not eligible (exclusion: REIT/RIC)
  3. If `corporation_type` = s_corp → Not eligible (exclusion: S Corporation)
  4. If `corporation_type` = tax_exempt → Not eligible (exclusion: tax-exempt under 501)
  5. If `ownership_percentage_voting` < 80.0 → Not eligible (exclusion: insufficient voting power)
  6. If `ownership_percentage_value` < 80.0 → Not eligible (exclusion: insufficient FMV ownership)
  7. Otherwise → Eligible
- **Policy Statement**: "IRC § 1504(a) requires 80% ownership of voting power and value. Certain entities are excluded per IRC § 1504(b)."

**Table 10100**: `Determine_Common_Parent`
- **Purpose**: Identify the common parent of the affiliated group
- **Inputs**:
  - Collection of `group_member` entities with ownership relationships
- **Outputs**:
  - `common_parent_member_id` (string)
  - `common_parent_ein` (string)
- **Rules**:
  1. Common parent has `parent_member_id` = null (not owned by another domestic corporation)
  2. Common parent owns ≥80% (voting and value) of at least one other includible corporation
  3. All other members trace ownership chain back to common parent
- **Policy Statement**: "Common parent defined per IRC § 1504(a)(1). Must not be 80%+ owned by another domestic corporation."

**Table 10200**: `Build_Affiliated_Group_Chain`
- **Purpose**: Build complete list of affiliated group members
- **Inputs**:
  - `common_parent_member_id` (string)
  - Collection of corporations with ownership data
- **Outputs**:
  - Collection of `group_member` entities
  - `total_members` (integer)
- **Rules**:
  1. Include common parent
  2. Include all corporations with direct 80%+ ownership by common parent
  3. Include all corporations with indirect 80%+ ownership through chain
  4. Exclude any corporation failing affiliation test (Table 10000)
- **Policy Statement**: "Affiliated group includes all corporations meeting IRC § 1504 ownership test through direct or indirect chain."

**Table 10250**: `Handle_Mid_Year_Acquisition`
- **Purpose**: Determine tax treatment when member joins mid-year
- **Inputs**:
  - `acquisition_date` (date)
  - `tax_year_end` (date)
  - `member_separate_income_pre_acquisition` (double)
  - `member_separate_income_post_acquisition` (double)
- **Outputs**:
  - `includible_in_consolidated_return` (boolean)
  - `included_income_amount` (double)
  - `separate_return_required` (boolean)
- **Rules**:
  1. Member joins group on `acquisition_date`
  2. Pre-acquisition income filed on separate return (pro-rated to acquisition date)
  3. Post-acquisition income included in consolidated return
  4. Reg. § 1.1502-76 governs closing of books or ratable allocation
- **Policy Statement**: "Reg. § 1.1502-76 requires allocation of items between consolidated and separate return periods."

#### 11000 Series: Intercompany Transactions

**Table 11000**: `Identify_Intercompany_Transaction`
- **Purpose**: Determine if transaction is intercompany and requires special treatment
- **Inputs**:
  - `selling_member_id` (string)
  - `buying_member_id` (string)
  - `transaction_type` (string: inventory_sale, asset_sale, dividend, interest, rent, royalty, service)
- **Outputs**:
  - `is_intercompany` (boolean)
  - `elimination_treatment` (string: defer, eliminate, immediate)
- **Rules**:
  1. If seller and buyer both in same consolidated group → `is_intercompany` = true
  2. If intercompany AND transaction_type = dividend → `elimination_treatment` = eliminate (100%)
  3. If intercompany AND transaction_type = interest, rent, royalty → `elimination_treatment` = eliminate (100%)
  4. If intercompany AND transaction_type = inventory_sale, asset_sale → `elimination_treatment` = defer (until outside event)
  5. If intercompany AND transaction_type = service → `elimination_treatment` = immediate (no deferral)
  6. Otherwise → `is_intercompany` = false
- **Policy Statement**: "Reg. § 1.1502-13 requires matching and acceleration treatment for intercompany transactions to reflect single entity concept."

**Table 11100**: `Calculate_Intercompany_Gain_Loss`
- **Purpose**: Calculate deferred gain/loss on intercompany asset/inventory sales
- **Inputs**:
  - `sale_price` (double)
  - `seller_adjusted_basis` (double)
  - `asset_type` (string: inventory, depreciable, non_depreciable)
- **Outputs**:
  - `intercompany_gain_loss` (double)
  - `deferred_amount` (double)
- **Rules**:
  1. `intercompany_gain_loss` = `sale_price` - `seller_adjusted_basis`
  2. If gain/loss ≠ 0 → `deferred_amount` = `intercompany_gain_loss` (full deferral)
  3. Otherwise → `deferred_amount` = 0
- **Policy Statement**: "Seller's gain or loss is computed but taken into account when buyer's corresponding item is taken into account (matching rule)."

**Table 11200**: `Determine_Restoration_Trigger`
- **Purpose**: Identify when deferred intercompany gain/loss should be restored
- **Inputs**:
  - `buyer_action` (string: sold_to_third_party, depreciated, distributed, member_left_group)
  - `asset_type` (string)
  - `buyer_depreciable` (boolean)
- **Outputs**:
  - `trigger_restoration` (boolean)
  - `restoration_amount` (double)
  - `restoration_type` (string: matching, acceleration)
- **Rules**:
  1. If `buyer_action` = sold_to_third_party → Restore full deferred amount (matching rule)
  2. If `buyer_action` = depreciated AND `buyer_depreciable` = true → Restore pro-rata based on depreciation (matching rule)
  3. If `buyer_action` = member_left_group → Restore full remaining deferred amount (acceleration rule)
  4. Otherwise → No restoration
- **Policy Statement**: "Deferred items are taken into account to produce the same effect on consolidated taxable income as if the seller and buyer were divisions of a single corporation (Reg. § 1.1502-13(c))."

**Table 11300**: `Create_Elimination_Entry`
- **Purpose**: Generate accounting elimination for intercompany transaction
- **Inputs**:
  - `transaction_type` (string)
  - `selling_member_income` (double)
  - `buying_member_expense` (double)
- **Outputs**:
  - `elimination_amount` (double)
  - `income_reduction` (double)
  - `expense_reduction` (double)
  - `net_effect_on_cti` (double)
- **Rules**:
  1. If transaction_type = dividend, interest, rent, royalty:
     - `income_reduction` = `selling_member_income`
     - `expense_reduction` = `buying_member_expense`
     - `elimination_amount` = MIN(income_reduction, expense_reduction)
     - `net_effect_on_cti` = 0 (complete elimination)
  2. If transaction_type = inventory_sale, asset_sale:
     - Defer gain/loss (no immediate elimination, handle in Table 11100-11200)
- **Policy Statement**: "Intercompany dividends, interest, rents, and royalties are eliminated to prevent double counting within consolidated group."

#### 12000 Series: Basis Adjustments & ELA

**Table 12000**: `Compute_Investment_Basis_Adjustment`
- **Purpose**: Calculate parent's basis adjustment for subsidiary stock under Reg. § 1.1502-32
- **Inputs**:
  - `subsidiary_taxable_income` (double)
  - `subsidiary_tax_exempt_income` (double)
  - `subsidiary_loss` (double)
  - `subsidiary_nondeductible_expenses` (double)
  - `subsidiary_distributions` (double)
  - `current_stock_basis` (double)
- **Outputs**:
  - `positive_adjustments` (double)
  - `negative_adjustments` (double)
  - `net_adjustment` (double)
  - `new_stock_basis` (double)
  - `excess_loss_account` (double)
- **Rules**:
  1. `positive_adjustments` = `subsidiary_taxable_income` + `subsidiary_tax_exempt_income`
  2. `negative_adjustments` = `subsidiary_loss` + `subsidiary_nondeductible_expenses` + `subsidiary_distributions`
  3. `net_adjustment` = `positive_adjustments` - `negative_adjustments`
  4. `new_stock_basis` = `current_stock_basis` + `net_adjustment`
  5. If `new_stock_basis` < 0:
     - `excess_loss_account` = ABS(`new_stock_basis`)
     - `new_stock_basis` = 0
  6. Otherwise:
     - `excess_loss_account` = 0
- **Policy Statement**: "Reg. § 1.1502-32 requires basis adjustments to reflect subsidiary's economic performance and prevent double taxation or double benefit."

**Table 12100**: `Tier_Up_Basis_Adjustments`
- **Purpose**: Cascade basis adjustments up multi-tier ownership structures
- **Inputs**:
  - Lower-tier subsidiary basis adjustment
  - Middle-tier parent entity
  - Top-tier grandparent entity
- **Outputs**:
  - Cascaded adjustments at each ownership level
- **Rules**:
  1. Lower-tier (T) adjustment increases middle-tier's (S) basis in T stock
  2. S's basis increase becomes S's own income adjustment
  3. S's adjustment increases top-tier's (M) basis in S stock
  4. Continue up chain to common parent
- **Policy Statement**: "Multi-tier structures require cascading basis adjustments to prevent distortions in stock basis throughout ownership chain."

**Table 12200**: `Determine_ELA_Recapture`
- **Purpose**: Identify when excess loss account is triggered and recaptured as income
- **Inputs**:
  - `excess_loss_account_balance` (double)
  - `triggering_event` (string: stock_sale, deconsolidation, worthless_stock, distribution)
  - `stock_sale_gain_loss` (double) [if applicable]
- **Outputs**:
  - `ela_recapture_required` (boolean)
  - `ela_income_inclusion` (double)
  - `character` (string: ordinary_income)
- **Rules**:
  1. If `excess_loss_account_balance` > 0 AND (`triggering_event` = stock_sale OR deconsolidation OR worthless_stock):
     - `ela_recapture_required` = true
     - `ela_income_inclusion` = `excess_loss_account_balance` (full recapture)
     - `character` = ordinary_income
  2. Total gain on disposition = `stock_sale_gain_loss` + `ela_income_inclusion`
  3. Otherwise → No recapture
- **Policy Statement**: "Reg. § 1.1502-19 requires ELA recapture as ordinary income to prevent permanent exclusion of previously deducted losses."

#### 13000 Series: SRLY & Consolidated NOL

**Table 13000**: `Identify_SRLY_Attributes`
- **Purpose**: Classify NOL carryforwards as SRLY (separate return) or non-SRLY (consolidated)
- **Inputs**:
  - `nol_carryforward` (double)
  - `year_nol_generated` (integer)
  - `year_joined_group` (integer)
  - `current_tax_year` (integer)
- **Outputs**:
  - `srly_nol_amount` (double)
  - `non_srly_nol_amount` (double)
- **Rules**:
  1. If `year_nol_generated` < `year_joined_group` → `srly_nol_amount` = `nol_carryforward` (pre-acquisition loss)
  2. If `year_nol_generated` >= `year_joined_group` → `non_srly_nol_amount` = `nol_carryforward` (consolidated loss)
- **Policy Statement**: "SRLY rules (Reg. § 1.1502-21(c)) apply to losses generated in separate return years before member joined consolidated group."

**Table 13100**: `Calculate_SRLY_Register`
- **Purpose**: Track cumulative contribution to consolidated taxable income for SRLY limitation
- **Inputs**:
  - `member_separate_taxable_income_current_year` (double)
  - `member_srly_register_prior_balance` (double)
- **Outputs**:
  - `member_srly_register_new_balance` (double)
  - `member_cumulative_contribution_to_cti` (double)
- **Rules**:
  1. `member_cumulative_contribution_to_cti` = `member_srly_register_prior_balance` + `member_separate_taxable_income_current_year`
  2. `member_srly_register_new_balance` = `member_cumulative_contribution_to_cti`
  3. SRLY register can be negative if member has cumulative losses
- **Policy Statement**: "SRLY register measures member's cumulative contribution to consolidated taxable income for purposes of limiting SRLY NOL usage."

**Table 13200**: `Apply_SRLY_Limitation`
- **Purpose**: Limit use of SRLY NOL to member's contribution to consolidated income
- **Inputs**:
  - `member_srly_nol_carryforward` (double)
  - `member_contribution_to_cti_current_year` (double)
  - `member_srly_register_balance` (double)
- **Outputs**:
  - `srly_nol_allowed_current_year` (double)
  - `srly_nol_remaining_carryforward` (double)
- **Rules**:
  1. If `member_contribution_to_cti_current_year` <= 0 → `srly_nol_allowed_current_year` = 0 (no positive income to offset)
  2. Otherwise:
     - `srly_nol_allowed_current_year` = MIN(`member_srly_nol_carryforward`, `member_contribution_to_cti_current_year`)
  3. `srly_nol_remaining_carryforward` = `member_srly_nol_carryforward` - `srly_nol_allowed_current_year`
- **Policy Statement**: "SRLY NOL can only offset income contributed by the SRLY member to prevent trafficking in losses (Reg. § 1.1502-21(c))."

**Table 13300**: `Apply_Section_382_Limitation`
- **Purpose**: Apply IRC § 382 ownership change limitations to NOL usage
- **Inputs**:
  - `ownership_change_occurred` (boolean)
  - `section_382_limitation_amount` (double) [annual limit]
  - `available_nol` (double)
  - `consolidated_taxable_income` (double)
- **Outputs**:
  - `nol_allowed_after_382` (double)
  - `nol_disallowed_by_382` (double)
- **Rules**:
  1. If `ownership_change_occurred` = false → No limitation, use full NOL
  2. If `ownership_change_occurred` = true:
     - `nol_allowed_after_382` = MIN(`available_nol`, `section_382_limitation_amount`, `consolidated_taxable_income`)
     - `nol_disallowed_by_382` = `available_nol` - `nol_allowed_after_382`
  3. Apply § 382 limitation before SRLY limitation (ordering rule)
- **Policy Statement**: "IRC § 382 limits NOL usage following ownership change (50%+ shift in value over 3 years). Annual limit = FMV × long-term tax-exempt rate."

**Table 13400**: `Aggregate_Consolidated_NOL`
- **Purpose**: Combine all usable NOLs (SRLY and non-SRLY) for current year
- **Inputs**:
  - Collection of member SRLY NOLs (after limitations)
  - Consolidated group NOL carryforwards (non-SRLY)
  - `consolidated_taxable_income_before_nol` (double)
- **Outputs**:
  - `total_nol_utilized_current_year` (double)
  - `remaining_nol_carryforward` (double)
  - `consolidated_taxable_income_after_nol` (double)
- **Rules**:
  1. Sum all SRLY NOLs allowed (from Table 13200)
  2. Add non-SRLY NOLs (no limitation except § 382)
  3. `total_nol_utilized_current_year` = MIN(total available NOL, `consolidated_taxable_income_before_nol`)
  4. `consolidated_taxable_income_after_nol` = `consolidated_taxable_income_before_nol` - `total_nol_utilized_current_year`
- **Policy Statement**: "Consolidated NOL carryforwards offset any member's income; SRLY NOLs subject to limitation (Reg. § 1.1502-21)."

#### 14000 Series: Consolidated Income & Tax

**Table 14000**: `Aggregate_Member_Separate_Income`
- **Purpose**: Sum all member separate taxable incomes before eliminations
- **Inputs**:
  - Collection of `group_member` entities with `separate_taxable_income`
- **Outputs**:
  - `total_separate_income` (double)
  - `member_count` (integer)
- **Rules**:
  1. Sum `separate_taxable_income` for all members
  2. Include both positive income and losses (net)
  3. This is before any intercompany eliminations
- **Policy Statement**: "Starting point is sum of each member's separate taxable income as if filing separate returns (Reg. § 1.1502-11)."

**Table 14100**: `Apply_Consolidated_Eliminations`
- **Purpose**: Apply all intercompany eliminations and deferrals
- **Inputs**:
  - `total_separate_income` (double)
  - Collection of `elimination_entry` entities
  - Deferred intercompany gains/losses
- **Outputs**:
  - `total_eliminations` (double)
  - `income_after_eliminations` (double)
- **Rules**:
  1. Sum all current year elimination amounts
  2. Subtract deferred gains (reduce income)
  3. Add restored gains from prior years (increase income)
  4. `income_after_eliminations` = `total_separate_income` - deferred + restored - eliminated
- **Policy Statement**: "Intercompany transactions are eliminated or deferred to reflect single entity treatment (Reg. § 1.1502-13)."

**Table 14200**: `Calculate_Consolidated_Charitable_Limitation`
- **Purpose**: Apply 10% charitable contribution limitation at consolidated level
- **Inputs**:
  - Sum of all member charitable contributions
  - `consolidated_taxable_income_before_charitable` (double)
- **Outputs**:
  - `charitable_deduction_allowed` (double)
  - `charitable_carryforward` (double)
- **Rules**:
  1. Aggregate all member contributions
  2. Limitation = 10% × `consolidated_taxable_income_before_charitable`
  3. `charitable_deduction_allowed` = MIN(total contributions, limitation)
  4. `charitable_carryforward` = Excess contributions (5-year carryforward)
- **Policy Statement**: "IRC § 170(b)(2) 10% limitation applied at consolidated level, not member level (Reg. § 1.1502-24)."

**Table 14300**: `Calculate_Consolidated_163j_Limitation`
- **Purpose**: Apply IRC § 163(j) business interest limitation at consolidated level
- **Inputs**:
  - Sum of all member business interest expenses
  - `consolidated_adjusted_taxable_income` (double) [ATI]
- **Outputs**:
  - `interest_deduction_allowed` (double)
  - `interest_disallowed_carryforward` (double)
- **Rules**:
  1. Aggregate all member business interest expense
  2. Calculate consolidated ATI (taxable income before interest, NOL, certain deductions)
  3. Limitation = 30% × consolidated ATI
  4. `interest_deduction_allowed` = MIN(total interest, limitation)
  5. `interest_disallowed_carryforward` = Excess interest (indefinite carryforward)
- **Policy Statement**: "IRC § 163(j) business interest limitation applies at consolidated level. 30% of consolidated ATI limit (Reg. § 1.1502-21(h))."

**Table 14400**: `Calculate_Consolidated_Taxable_Income`
- **Purpose**: Final consolidated taxable income calculation
- **Inputs**:
  - `total_separate_income` (double)
  - `total_eliminations` (double)
  - `charitable_deduction` (double)
  - `interest_deduction` (double)
  - `nol_deduction` (double)
  - Other consolidated adjustments
- **Outputs**:
  - `consolidated_taxable_income` (double)
- **Rules**:
  1. Start with sum of member separate income
  2. Apply intercompany eliminations
  3. Apply consolidated charitable limitation
  4. Apply consolidated interest limitation
  5. Subtract consolidated NOL
  6. Result = Consolidated taxable income (Form 1120 line 28)
- **Policy Statement**: "Consolidated taxable income computed per Reg. § 1.1502-11, applying limitations at group level."

**Table 14500**: `Calculate_Consolidated_Tax`
- **Purpose**: Apply 21% rate to consolidated taxable income
- **Inputs**:
  - `consolidated_taxable_income` (double)
- **Outputs**:
  - `consolidated_tax_before_credits` (double)
- **Rules**:
  1. If `consolidated_taxable_income` <= 0 → Tax = $0
  2. Otherwise → Tax = `consolidated_taxable_income` × 21%
  3. Same flat rate as separate corporations
- **Policy Statement**: "IRC § 11(b) flat 21% rate applies to consolidated taxable income (TCJA 2017)."

**Table 14600**: `Aggregate_Consolidated_Credits`
- **Purpose**: Combine all member tax credits at group level
- **Inputs**:
  - Collection of member credits (R&D, general business, foreign tax, etc.)
- **Outputs**:
  - `total_consolidated_credits` (double)
  - `consolidated_tax_after_credits` (double)
- **Rules**:
  1. Aggregate all member credits
  2. Apply credit limitations at consolidated level
  3. `consolidated_tax_after_credits` = `consolidated_tax_before_credits` - `total_consolidated_credits`
  4. Cannot reduce tax below zero
- **Policy Statement**: "Tax credits computed and limited at consolidated level (Reg. § 1.1502-3)."

#### 15000 Series: Form 851 & Schedules

**Table 15000**: `Prepare_Form_851_Affiliations_Schedule`
- **Purpose**: Generate Form 851 data for affiliated group members
- **Inputs**:
  - `affiliated_group` entity
  - Collection of `group_member` entities
- **Outputs**:
  - Form 851 data structure with member details
- **Rules**:
  1. List common parent (Part I)
  2. List all subsidiaries with EIN, name, ownership % (Part II)
  3. Report estimated tax payments by member (Part III)
  4. Report overpayment credits by member (Part IV)
- **Policy Statement**: "Form 851 required with consolidated return to identify group members and allocate payments (Reg. § 1.1502-75(h))."

**Table 15100**: `Prepare_Schedule_M3_Consolidated`
- **Purpose**: Generate Schedule M-3 consolidated reconciliation (required if total assets ≥ $10M)
- **Inputs**:
  - Consolidated financial statement income
  - Consolidated taxable income
  - Book-tax differences
- **Outputs**:
  - Schedule M-3 Parts I, II, III
- **Rules**:
  1. Part I: Reconcile worldwide accounting income to consolidated taxable income
  2. Part II: Reconcile income/loss items
  3. Part III: Reconcile expense/deduction items
  4. Parent must also complete separate Schedule M-3 for parent-only activity
- **Policy Statement**: "Schedule M-3 required for consolidated groups with total assets ≥ $10 million. Group files consolidated M-3 plus parent-only M-3 (Instructions for Schedule M-3)."

### 5.3 Execution Flow

**Orchestration Table**: `Table 10000 - Compute_Consolidated_Tax_Return`

```
1. Test affiliation (Tables 10000-10200)
   ↓
2. Calculate each member's separate TI (Tables 2000-4000, existing)
   ↓
3. Process intercompany transactions (Tables 11000-11300)
   ↓
4. Compute investment basis adjustments (Tables 12000-12200)
   ↓
5. Apply SRLY and § 382 limitations (Tables 13000-13400)
   ↓
6. Calculate consolidated TI (Tables 14000-14400)
   ↓
7. Calculate consolidated tax and credits (Tables 14500-14600)
   ↓
8. Prepare Form 851 and schedules (Tables 15000-15100)
   ↓
9. Generate consolidated return output
```

### References

- [IRS Form 851 - Affiliations Schedule](https://www.irs.gov/forms-pubs/about-form-851)
- [Instructions for Schedule M-3 Form 1120](https://www.irs.gov/instructions/i1120sm3)
- [26 CFR § 1.1502-75 - Filing Requirements](https://www.law.cornell.edu/cfr/text/26/1.1502-75)

---

## 6. Forms and Schedules

### 6.1 Form 1120 (Consolidated)

**Changes from Separate Return**:

- **Page 1, Top**: Check box "Consolidated return" (box next to final return)
- **Name/EIN**: Common parent's name and EIN
- **Line 11**: Total income = Consolidated total income (sum of members)
- **Line 27**: Total deductions = Consolidated total deductions
- **Line 28**: Taxable income = Consolidated taxable income
- **Line 31**: Total tax = Consolidated tax liability
- **Lines 32-34**: Credits, payments (allocated per Form 851)
- **Line 35**: Tax due or refund

### 6.2 Form 851 - Affiliations Schedule

**Required Attachment**: Must be filed with every consolidated return.

**Content**:

- **Part I - Common Parent Corporation**
  - Name, EIN, address
  - Date of incorporation
  - Stock information

- **Part II - Subsidiary Corporations**
  - For each subsidiary:
    - Name, EIN, address
    - Percentage of voting stock owned by parent/other members
    - Percentage of value stock owned
    - Taxable income/loss

- **Part III - Changes in Stock Ownership**
  - Acquisitions during year
  - Dispositions during year
  - Changes in ownership percentage

- **Part IV - Principal Business Activity**
  - For each member

- **Part V - Estimated Tax Payments**
  - Allocation of estimated tax payments by member
  - Overpayments from prior year by member

### 6.3 Schedule M-3 (Consolidated)

**Required If**: Consolidated group total assets ≥ $10 million

**Two Schedules Required**:

1. **Consolidated Schedule M-3** (Parts I, II, III)
   - Reconcile consolidated financial statement income to consolidated taxable income
   - Report all book-tax differences at consolidated level

2. **Parent-Only Schedule M-3** (Parts II, III only)
   - Reconcile parent's separate activity
   - Required even though parent is included in consolidated M-3

**Parts**:

- **Part I**: Financial Information and Net Income Reconciliation
  - Worldwide consolidated net income per books
  - Income statement period
  - Type of income statement (GAAP, IFRS, tax basis, etc.)
  - Adjustments to reconcile to consolidated taxable income

- **Part II**: Reconciliation of Net Income (Loss) per Income Statement With Taxable Income
  - Income items (interest, dividends, royalties, capital gains, etc.)
  - Book-tax differences for each item
  - Temporary vs. permanent differences

- **Part III**: Reconciliation of Expense per Income Statement With Deductions
  - Expense items (compensation, interest, depreciation, etc.)
  - Book-tax differences
  - Temporary vs. permanent differences

### 6.4 Additional Schedules

**Schedule C** - Dividends and Special Deductions
- Consolidated dividends received deduction
- Applied at group level

**Schedule J** - Tax Computation
- May be required for certain groups
- Reports consolidated tax calculation

**Schedule K** - Other Information
- Questions about group's activities
- Foreign operations, transfers, etc.

**Schedule L** - Balance Sheet
- Consolidated balance sheet per books
- Beginning and end of year

**Schedule M-1** - Reconciliation of Income (Loss) per Books With Income per Return
- Only if not required to file Schedule M-3
- Simpler book-tax reconciliation

**Schedule M-2** - Analysis of Unappropriated Retained Earnings per Books
- Consolidated retained earnings
- Beginning balance, additions, distributions, ending balance

### 6.5 State Consolidated/Combined Returns

**Critical Note**: Most states do NOT allow federal consolidated returns.

**State Approaches**:

1. **Separate Filing** (Most Common)
   - Each member files separate state return
   - Apportion federal taxable income separately
   - Examples: New Jersey, Pennsylvania

2. **Combined Reporting** (Unitary Business)
   - Combine income of unitary business
   - Different from federal consolidation (uses water's edge/worldwide approach)
   - Examples: California, Illinois, New York

3. **Consolidated Filing** (Rare)
   - Follow federal consolidation rules
   - Examples: Very few states allow this

**Implementation Implication**: State tax calculation in consolidated returns is complex and varies by state. May require separate state returns even if filing federal consolidated.

---

## 7. Complexity Factors

### 7.1 Mid-Year Acquisitions and Dispositions

**Challenge**: When a corporation joins or leaves the group mid-year, income must be allocated between consolidated and separate return periods.

#### Mid-Year Acquisition

**Regulatory Approach** (Reg. § 1.1502-76):

1. **Closing-of-Books Method** (Default)
   - Close subsidiary's books on acquisition date
   - Pre-acquisition income → Separate return (short year)
   - Post-acquisition income → Consolidated return

2. **Ratable Allocation Method** (Optional)
   - Allocate income/loss pro-rata based on days in each period
   - Simpler but less accurate

**Example**:
- M acquires S on July 1, 2025 (mid-year)
- S earned $120,000 for calendar year
- **Closing-of-books**:
  - Jan 1 - Jun 30: $60,000 → S separate return
  - Jul 1 - Dec 31: $60,000 → Consolidated return
- **Ratable allocation**:
  - 181 days separate / 365 total = 49.6% → $59,520 separate
  - 184 days consolidated / 365 total = 50.4% → $60,480 consolidated

**Complexity**:
- Requires tracking acquisition dates for each member
- Different treatment of capital vs. ordinary items
- Intercompany transactions straddling acquisition require special rules

#### Mid-Year Disposition

**When Member Leaves**:
- Deconsolidation occurs (stock sold, member leaves group)
- Triggers ELA recapture if basis is negative
- Deferred intercompany items may accelerate
- Must allocate income between consolidated period and post-deconsolidation

**Example**:
- M sells S stock on August 1, 2025
- S has deferred gain of $50,000 from prior intercompany sale
- M has ELA of $30,000 in S stock
- **Tax Consequences**:
  - Accelerate $50,000 deferred gain (include in consolidated income)
  - Recapture $30,000 ELA as ordinary income
  - M's gain/loss on S stock sale calculated separately

### 7.2 Multiple-Tier Structures

**Challenge**: Ownership chains (M → S → T → U) create cascading basis adjustments and complex intercompany transaction tracking.

#### Example Structure

```
M (Common Parent)
 |
 +-- S1 (80% owned by M)
 |    |
 |    +-- T1 (90% owned by S1)
 |    |
 |    +-- T2 (85% owned by S1)
 |
 +-- S2 (100% owned by M)
      |
      +-- T3 (80% owned by S2)
```

**Affiliation Testing**:
- Direct ownership: M → S1, M → S2 (both 80%+) ✓
- Indirect ownership through chain:
  - M → S1 → T1: 80% × 90% = 72% (fails 80% test) ✗
  - Wait! IRC § 1504 uses "direct" test at each level, not cumulative
  - S1 owns 90% of T1 directly → T1 is affiliated ✓

**Key Rule**: Each link in chain must be 80%+, but test is direct ownership at each level, not cumulative.

#### Tiered Basis Adjustments

**Scenario**:
- T1 earns $100,000 (bottom tier)
- S1 owns T1 stock (middle tier)
- M owns S1 stock (top tier)

**Adjustment Cascade**:
1. **T1 level**: T1's $100K income included in consolidated return
2. **S1 level**: S1's basis in T1 stock increases by $100K (Reg. § 1.1502-32)
3. **M level**: S1's $100K basis increase is treated as income adjustment → M's basis in S1 stock increases by $100K

**Why It Matters**:
- If M later sells S1 stock, basis must reflect T1's accumulated earnings
- Prevents double taxation on multiple tiers
- Complex to track in structures with 5+ tiers

#### Intercompany Transactions Across Tiers

**Scenario**:
- T1 sells inventory to S1 for $200 (cost basis $120)
- S1 sells to S2 for $250
- S2 sells to third party for $300

**Analysis**:
1. T1 → S1: Intercompany sale, $80 gain deferred
2. S1 → S2: Intercompany sale, $50 gain deferred
3. S2 → Third party: External sale, triggers restoration
   - T1's $80 deferred gain restored
   - S1's $50 deferred gain restored
   - S2 recognizes $50 gain ($300 - $250)
   - Total consolidated gain: $80 + $50 + $50 = $180 = $300 - $120 ✓

**Complexity**:
- Must track deferred gains through multiple intercompany legs
- Matching rule applies to ultimate external sale
- Cannot "leak" income through intercompany chains

### 7.3 Foreign Subsidiaries

**Key Rule**: Foreign corporations **cannot** be members of consolidated group (IRC § 1504(b)(3)).

**Consequences**:

1. **Separate Entity Treatment**
   - Foreign subs file their own returns (if U.S. income)
   - Not included in consolidated group

2. **Controlled Foreign Corporation (CFC) Rules**
   - If U.S. parent owns 50%+ of foreign sub (by vote or value)
   - Subpart F income inclusion (taxed currently)
   - GILTI (Global Intangible Low-Taxed Income) inclusion

3. **Foreign Tax Credit**
   - Parent gets foreign tax credit for taxes paid by CFC
   - Complex limitation calculations

4. **Dual Consolidated Losses (DCL)**
   - If foreign hybrid entity generates loss
   - Cannot use loss to offset U.S. consolidated income if also usable in foreign country
   - Prevents double-dipping

**Example**:
- M (U.S. parent)
- S (U.S. sub, 100% owned) → Consolidated with M
- F (Foreign sub, 100% owned) → NOT consolidated
  - F has Subpart F income of $500K
  - M must include $500K in consolidated income
  - M gets foreign tax credit for F's foreign taxes paid

**Implementation Challenge**:
- Must track foreign subs separately
- Include Subpart F and GILTI in consolidated income
- Calculate foreign tax credit at consolidated level
- Apply DCL limitations

### 7.4 Dual Consolidated Losses

**Definition**: Loss generated by a "dual resident corporation" (taxable in both U.S. and foreign country) or a "separate unit" of a domestic corporation.

**IRC § 1503(d) Rule**: DCL cannot offset income of any domestic affiliate unless taxpayer certifies loss will not be used to offset foreign income.

**Recent Update**: August 2024 proposed regulations address interaction with OECD Pillar Two.

**Example**:
- M (U.S. parent)
- S (U.S. subsidiary)
- DRC (Dual resident corporation - incorporated in U.S. and Country X)
- DRC generates $100K loss
- If DRC's loss can offset income in Country X:
  - Cannot use loss in U.S. consolidated return
  - Must be "parked" unless certification provided

**Complexity**:
- Requires analysis of foreign tax law
- Tracking of dual resident entities
- Certification requirements
- Recapture if certification violated

### 7.5 Intercompany Debt Instruments

**Challenge**: Interest on intercompany loans is eliminated, but creates basis adjustments and potential debt/equity recharacterization.

**Scenario**:
- M lends $1,000,000 to S at 5% interest
- S pays $50,000 interest to M
- **Consolidated Treatment**:
  - M's $50K interest income eliminated
  - S's $50K interest deduction eliminated
  - Net effect on CTI: $0

**Complications**:

1. **IRC § 163(j) Limitation**
   - Business interest deduction limited to 30% of consolidated ATI
   - Intercompany interest eliminated BEFORE applying limitation
   - Only third-party interest subject to limitation

2. **Debt vs. Equity**
   - If "loan" is recharacterized as equity, "interest" becomes dividend
   - Dividends between members also eliminated
   - But affects basis adjustments differently

3. **Original Issue Discount (OID)**
   - If intercompany debt issued at discount
   - OID accruals eliminated under intercompany transaction rules
   - Complex timing issues

### 7.6 Consolidated Return Change of Accounting Method

**Challenge**: Changing accounting method (cash to accrual, inventory method, depreciation) requires IRC § 481(a) adjustment.

**In Consolidated Context**:
- § 481(a) adjustment computed for each member
- Adjustments netted at consolidated level
- Can create income or deductions in year of change

**Example**:
- S1 changes from cash to accrual method
- Creates $200K § 481(a) positive adjustment (income acceleration)
- S2 changes inventory method (FIFO to weighted average)
- Creates $50K § 481(a) negative adjustment
- Net consolidated § 481(a) adjustment: $150K income

---

## 8. Implementation Phases

### Phase 1: Foundation (4-6 weeks)

**Objective**: Build multi-entity architecture and affiliation testing.

**Deliverables**:

1. **Entity Model**
   - Create `affiliated_group` entity
   - Create `group_member` entity
   - Extend `corporation` entity for consolidation
   - Update DTRules schema to support entity collections

2. **Affiliation Testing Tables** (10000-10200)
   - Table 10000: Test affiliation eligibility
   - Table 10100: Determine common parent
   - Table 10200: Build affiliated group chain
   - Table 10250: Handle mid-year acquisition

3. **Test Cases**
   - Simple two-member group (parent + subsidiary)
   - Three-tier structure (M → S → T)
   - Mid-year acquisition scenario
   - Foreign subsidiary exclusion test

4. **Documentation**
   - Architecture guide
   - Entity relationship diagrams
   - Table specifications

**Success Criteria**:
- Can identify affiliated group from ownership data
- Can build member list excluding ineligible entities
- Can handle mid-year membership changes

---

### Phase 2: Intercompany Transactions (3-4 weeks)

**Objective**: Implement intercompany transaction tracking and elimination logic.

**Deliverables**:

1. **Entity Model**
   - Create `intercompany_transaction` entity
   - Create `elimination_entry` entity

2. **Intercompany Transaction Tables** (11000-11300)
   - Table 11000: Identify intercompany transactions
   - Table 11100: Calculate intercompany gain/loss
   - Table 11200: Determine restoration triggers
   - Table 11300: Create elimination entries

3. **Transaction Types**
   - Inventory sales (deferred until buyer sells)
   - Asset sales (deferred, restored with depreciation)
   - Dividends (eliminated completely)
   - Interest (eliminated completely)

4. **Test Cases**
   - Inventory sale chain (S → B → third party)
   - Depreciable asset sale with restoration
   - Intercompany dividend elimination
   - Intercompany interest elimination

**Success Criteria**:
- Can identify intercompany transactions from buyer/seller IDs
- Can defer gains on inventory/asset sales
- Can restore deferred gains on triggering events
- Can eliminate dividends and interest

---

### Phase 3: Investment Basis & ELA (3-4 weeks)

**Objective**: Implement basis adjustments and excess loss account calculations.

**Deliverables**:

1. **Basis Adjustment Tables** (12000-12200)
   - Table 12000: Compute investment basis adjustment
   - Table 12100: Tier up basis adjustments
   - Table 12200: Determine ELA recapture

2. **Tracking Mechanisms**
   - Stock basis beginning/ending balances
   - Annual adjustment calculations
   - ELA creation and recapture
   - Tiered adjustment cascading

3. **Test Cases**
   - Simple basis adjustment (subsidiary income increases parent basis)
   - ELA creation (subsidiary loss + distribution exceeds basis)
   - ELA recapture on stock sale
   - Three-tier basis adjustment cascade

**Success Criteria**:
- Parent basis adjusts for subsidiary income/loss
- Distributions reduce basis correctly
- ELA created when basis goes negative
- ELA recaptured on triggering events
- Multi-tier adjustments cascade correctly

---

### Phase 4: SRLY and NOL (2-3 weeks)

**Objective**: Implement SRLY limitations and consolidated NOL rules.

**Deliverables**:

1. **SRLY Tables** (13000-13400)
   - Table 13000: Identify SRLY attributes
   - Table 13100: Calculate SRLY register
   - Table 13200: Apply SRLY limitation
   - Table 13300: Apply Section 382 limitation
   - Table 13400: Aggregate consolidated NOL

2. **NOL Tracking**
   - Separate SRLY vs. non-SRLY carryforwards
   - SRLY register by member
   - Consolidated group NOL carryforwards

3. **Test Cases**
   - Acquired subsidiary with pre-acquisition NOL (SRLY)
   - SRLY limitation (NOL limited to member's contribution)
   - Section 382 ownership change
   - Consolidated NOL usage (no SRLY limit)

**Success Criteria**:
- Can identify SRLY NOLs vs. consolidated NOLs
- SRLY register tracks member contributions correctly
- SRLY NOL limited to member's positive income
- Section 382 limits applied before SRLY limits

---

### Phase 5: Consolidated Income Calculation (2-3 weeks)

**Objective**: Calculate consolidated taxable income with all adjustments.

**Deliverables**:

1. **Consolidation Tables** (14000-14600)
   - Table 14000: Aggregate member separate income
   - Table 14100: Apply consolidated eliminations
   - Table 14200: Calculate consolidated charitable limitation
   - Table 14300: Calculate consolidated 163(j) limitation
   - Table 14400: Calculate consolidated taxable income
   - Table 14500: Calculate consolidated tax
   - Table 14600: Aggregate consolidated credits

2. **Integration**
   - Combine all prior phases
   - Apply consolidated-level limitations
   - Calculate final tax liability

3. **Test Cases**
   - Full consolidated return (2 members)
   - Consolidated charitable limitation
   - Consolidated interest limitation
   - Intercompany eliminations + SRLY + basis adjustments

**Success Criteria**:
- Can calculate consolidated TI from member separate TI
- Consolidated limitations applied correctly
- Tax calculated on consolidated income
- Credits aggregated at group level

---

### Phase 6: Forms and Reporting (2-3 weeks)

**Objective**: Generate Form 1120 consolidated, Form 851, Schedule M-3.

**Deliverables**:

1. **Form Generation Tables** (15000-15100)
   - Table 15000: Prepare Form 851
   - Table 15100: Prepare Schedule M-3 consolidated

2. **Output Formats**
   - Form 1120 with consolidated data
   - Form 851 affiliations schedule
   - Schedule M-3 (if total assets ≥ $10M)
   - Supporting schedules (M-1, M-2, L)

3. **Test Cases**
   - Complete consolidated return package
   - Form 851 with 3 members
   - Schedule M-3 consolidated reconciliation

**Success Criteria**:
- Form 1120 populated with consolidated amounts
- Form 851 lists all members with correct data
- Schedule M-3 reconciles book to tax income
- All supporting schedules accurate

---

### Phase 7: Advanced Features (4-6 weeks)

**Objective**: Handle complex scenarios (mid-year changes, dual consolidated losses, foreign subs).

**Deliverables**:

1. **Advanced Tables**
   - Mid-year acquisition pro-ration
   - Mid-year disposition and deconsolidation
   - Dual consolidated loss limitations
   - Subpart F and GILTI inclusions (foreign subs)

2. **Edge Cases**
   - Member joins mid-year with pre-acquisition NOL
   - Member leaves mid-year with ELA
   - Dual resident corporation with foreign loss usage
   - Foreign subsidiary CFC inclusions

3. **Test Cases**
   - Mid-year acquisition with short-year allocation
   - Deconsolidation triggering ELA recapture
   - DCL certification and limitation
   - CFC Subpart F inclusion in consolidated return

**Success Criteria**:
- Mid-year changes handled correctly
- Deconsolidation triggers all acceleration events
- DCL limitations prevent double-dipping
- Foreign subsidiary income inclusions work

---

### Total Implementation Estimate

**Timeline**: 20-29 weeks (5-7 months)

**Effort by Phase**:

| Phase | Weeks | Deliverables | Complexity |
|-------|-------|--------------|------------|
| 1. Foundation | 4-6 | Entity model, affiliation testing | Medium |
| 2. Intercompany Transactions | 3-4 | Transaction tracking, eliminations | High |
| 3. Basis & ELA | 3-4 | Basis adjustments, ELA | High |
| 4. SRLY & NOL | 2-3 | NOL limitations | Medium |
| 5. Consolidated Income | 2-3 | Final calculation | Medium |
| 6. Forms | 2-3 | Form 1120, 851, M-3 | Low |
| 7. Advanced | 4-6 | Edge cases, foreign subs | Very High |
| **Total** | **20-29** | **Full implementation** | **Very High** |

**Risk Factors**:
- DTRules may not support required multi-entity operations
- Graph traversal for tiered structures may be difficult
- Transaction lifecycle tracking across years requires persistence
- Testing complexity high (combinatorial explosion of scenarios)

---

## 9. Integration Challenges

### 9.1 DTRules Architecture Limitations

**Challenge 1: Single-Entity Assumption**

Current DTRules appears designed for single-entity context:
- Decision tables operate on single `corporation`, `revenue`, `expense` entities
- No built-in collection iteration (for each member)
- No relationship traversal (parent → child → grandchild)

**Potential Solutions**:
1. **Pre-process collections**: Use external code to iterate members, invoke DTRules for each
2. **Extend DTRules**: Add native support for entity collections and loops
3. **Flatten structure**: Create consolidated entities with member1_, member2_ fields (doesn't scale)

**Recommendation**: Evaluate DTRules capability for collection iteration. If not supported, this is a **blocking issue** requiring engine enhancements.

---

**Challenge 2: Inter-Year State Tracking**

Consolidated returns require tracking across years:
- Deferred intercompany gains (may restore 3-5 years later)
- SRLY register (cumulative from year member joined)
- Stock basis (accumulated adjustments since acquisition)
- ELA (grows over multiple years, triggered later)

**Current Issue**: DTRules decision tables appear stateless (single tax year execution).

**Potential Solutions**:
1. **External persistence**: Store deferred gains, SRLY register in database; load at start of each year
2. **Input carryover entities**: Accept prior year results as input entities
3. **State management layer**: Build wrapper to manage multi-year state

**Recommendation**: Consolidated returns require state management beyond single-year execution. Need persistence layer or carryover mechanism.

---

**Challenge 3: Circular References**

Basis adjustments can create circular dependencies:
- Parent's basis in subsidiary depends on subsidiary income
- Subsidiary income may include gain from asset sale to parent
- Sale gain depends on parent's basis in asset
- Parent's asset basis depends on prior year subsidiary distributions
- Distributions depend on subsidiary income

**Example**:
```
M's basis in S stock
  ↓ (depends on)
S's income
  ↓ (includes)
S's gain on sale to M
  ↓ (depends on)
S's basis in asset
  ↓ (may depend on)
Prior year distributions from S to M
  ↓ (which affect)
M's basis in S stock (circular!)
```

**Potential Solutions**:
1. **Iterative calculation**: Compute multiple passes until convergence
2. **Topological sort**: Order calculations to break cycles
3. **Simplifying assumptions**: Disallow certain circular scenarios

**Recommendation**: Treasury regulations have anti-circular rules (Reg. § 1.1502-32(e)). Implement these to prevent computational loops.

---

### 9.2 Testing Complexity

**Challenge**: Combinatorial explosion of test scenarios.

**Scenario Dimensions**:
- Number of members: 2, 3, 5, 10+
- Tiers: Flat (M → S), two-tier (M → S → T), three-tier (M → S → T → U)
- Ownership %: 80%, 90%, 100%
- Transaction types: Inventory, assets, dividends, interest (4 types)
- Timing: Full year, mid-year join, mid-year leave
- NOL: SRLY, non-SRLY, § 382
- Special cases: ELA, dual resident, foreign subs

**Test Case Estimate**:
- Basic scenarios: 20-30 test cases
- Standard scenarios: 50-75 test cases
- Edge cases: 100+ test cases
- **Total**: 170-200+ test cases

**Recommendation**:
- Start with 5-10 core scenarios in Phase 1
- Add 10-15 tests per phase
- Use test generation tools for combinatorial coverage
- Focus on IRS examples from regulations

---

### 9.3 State Tax Integration

**Problem**: Consolidated returns are federal-only. States have three approaches:

1. **Separate Filing** (PA, NJ, etc.)
   - Each member files separate state return
   - Start with federal taxable income (pre-consolidation)
   - Apply state apportionment separately

2. **Combined Reporting** (CA, IL, NY, etc.)
   - Combine unitary business income
   - Different from federal consolidation
   - Uses different elimination rules

3. **Consolidated (Rare)**
   - Few states follow federal consolidation

**Challenge**: State tax calculation in CorporateTax project assumes separate corporation. Consolidated returns require mapping federal consolidated to state separate/combined.

**Potential Solutions**:
1. **Hybrid approach**:
   - Federal: Use consolidated return tables (10000-15000)
   - State: Use separate return tables (7000-7999) for each member
2. **Dual calculation**:
   - Calculate both consolidated (federal) and separate (state) TI
   - Maintain separate entity sets for each
3. **State-specific combined reporting** (future phase):
   - Implement CA/IL/NY combined reporting rules
   - Different from federal consolidation

**Recommendation**:
- Phase 1-6: Federal consolidated only
- Phase 7: Add state separate filing for consolidated members
- Future: State combined reporting (separate project)

---

### 9.4 Performance Considerations

**Concern**: Consolidated returns are computationally intensive.

**Complexity Factors**:
- N members × M transactions = O(N×M) intercompany eliminations
- Tiered structures: O(N×depth) basis adjustments
- SRLY tracking: O(N×years) register calculations
- Transaction lifecycle: O(transactions×restoration_years)

**Example Workload**:
- 10-member group
- 50 intercompany transactions/year
- 3-tier structure
- 5-year SRLY carryforward
- **Calculations**: 10 separate TI + 50 eliminations + 10 basis adjustments + 10 SRLY + consolidation
- **Estimate**: 5-10x slower than single corporation return

**Scaling Concern**:
- Large consolidated groups (100+ members, like GE) may have performance issues
- Decision table evaluation may not be optimized for this workload

**Recommendation**:
- Benchmark with 2-member group first (Phase 1)
- Test 10-member group (Phase 5)
- Optimize hot paths if needed
- Consider limiting initial release to groups <50 members

---

### 9.5 Documentation and Maintainability

**Challenge**: Consolidated return regulations are the most complex area of tax law.

**Regulation Volume**:
- **Reg. § 1.1502**: Over 100 sections (1.1502-1 through 1.1502-100)
- **1,000+ pages** of dense regulatory text
- Constantly updated (December 2024 overhaul)

**Knowledge Requirements**:
- Expert-level understanding of corporate tax
- Deep knowledge of consolidated return regulations
- Familiarity with accounting for business combinations
- Understanding of multi-entity relationships

**Maintenance Burden**:
- Regulatory changes require significant updates
- Complex interactions mean changes have ripple effects
- Testing burden increases exponentially with features

**Recommendation**:
- **Hire a specialist**: Consolidated returns require expert consultation
- **Incremental approach**: Don't implement everything at once
- **Focus on common cases**: 80% of consolidated groups are simple (parent + 2-3 subs)
- **Defer rare scenarios**: Dual consolidated losses, etc. are edge cases

---

## 10. Recommendations

### 10.1 Strategic Recommendation: DEFER

**Verdict**: Defer consolidated returns to Phase 5 or later (after completing S Corporation support, foreign tax credit, and state tax implementations).

**Reasoning**:

1. **Complexity**: Consolidated returns are the single most complex feature in corporate taxation
   - 6-7 months development effort
   - Requires fundamental architecture changes
   - High testing burden (200+ test cases)
   - Expert knowledge required

2. **Market Demand**: Consolidated returns serve a niche market
   - Only ~10-15% of corporations file consolidated returns
   - Most are large enterprises with in-house tax teams
   - Less demand from SMB market (DTRules primary audience)

3. **Prerequisites**: Other features provide more value
   - S Corporation support (Form 1120-S): 70% of corporations
   - State tax (all 50 states): 100% of multi-state corporations
   - Foreign tax credit: Growing importance with globalization
   - NOL tracking: Applies to separate and consolidated returns

4. **Architecture Risk**: May require DTRules engine enhancements
   - Multi-entity collection iteration
   - Relationship traversal (graph queries)
   - Inter-year state management
   - If engine doesn't support these, project is blocked

5. **Alternative Approach**: Start with simpler multi-entity features
   - Affiliated service groups (IRC § 414)
   - Controlled groups (IRC § 1563)
   - Brother-sister corporations
   - These build multi-entity capability without full consolidation complexity

---

### 10.2 Phased Approach If Proceeding

**If consolidated returns must be implemented**, follow strict phasing:

#### Minimal Viable Product (MVP) - Phases 1-3 Only

**Scope**: Simple two-member consolidated return (parent + subsidiary)
- Affiliation testing (Table 10000-10200)
- Basic intercompany eliminations (dividends, interest only)
- Investment basis adjustments (no ELA)
- Consolidated income calculation (sum members, apply eliminations)
- Form 1120 consolidated + Form 851

**Excludes**:
- Mid-year acquisitions/dispositions
- Multi-tier structures (limit to 2 tiers)
- SRLY NOLs (assume all members joined at formation)
- ELA (assume basis never goes negative)
- Intercompany asset/inventory sales (deferred gains)
- Dual consolidated losses
- Foreign subsidiaries

**Timeline**: 10-14 weeks

**Value**: Covers 60-70% of consolidated return filings (simple parent + sub structures)

---

#### Standard Product - Phases 1-6

**Scope**: Full consolidation for domestic groups
- All MVP features
- Plus: Multi-tier structures
- Plus: Intercompany asset/inventory sales with deferral
- Plus: Investment basis adjustments with ELA
- Plus: SRLY NOL limitations
- Plus: Schedule M-3 consolidated

**Excludes**:
- Mid-year acquisitions/dispositions
- Dual consolidated losses
- Foreign subsidiaries (CFC inclusions)

**Timeline**: 20-24 weeks

**Value**: Covers 85-90% of consolidated returns

---

#### Enterprise Product - All Phases

**Scope**: Complete consolidation support
- All standard product features
- Plus: Mid-year membership changes
- Plus: Dual consolidated loss limitations
- Plus: Foreign subsidiary CFC inclusions
- Plus: Complex multi-tier scenarios (5+ tiers)

**Timeline**: 28-32 weeks

**Value**: 95%+ coverage

---

### 10.3 Alternative: Partner Integration

**Recommendation**: Consider integrating with existing consolidated return software instead of building from scratch.

**Rationale**:
- Major tax software (CCH ProSystem fx, Thomson Reuters UltraTax, Intuit Lacerte) have mature consolidated return modules
- Development cost: $500K-$1M (6-7 months × senior developer)
- Maintenance cost: $100K-$200K/year (regulatory updates)
- Alternative: Integration/API layer may be more cost-effective

**Partnership Model**:
1. DTRules calculates separate return for each member
2. Export to consolidated return software for:
   - Intercompany eliminations
   - Basis adjustments
   - Consolidation
3. Import consolidated results back to DTRules

**Benefits**:
- Leverage existing expertise
- Avoid 6-7 month development
- Reduce maintenance burden
- Focus DTRules on core strengths (decision tables for tax logic)

---

### 10.4 Architecture Recommendations If Proceeding

**If implementing consolidated returns in DTRules**:

#### 1. Enhance DTRules Engine (Pre-work)

Before starting Phase 1, ensure DTRules supports:

a. **Entity Collections**
```xml
<entity name="group_members" type="collection">
  <member_entity ref="group_member"/>
</entity>
```

b. **Collection Iteration**
```
FOR EACH member IN group_members:
  Calculate member.separate_taxable_income
  Add to total_separate_income
```

c. **Relationship Traversal**
```
Find all entities where parent_member_id = current_member.member_id
(recursive query for tiered structures)
```

d. **Conditional Logic on Collections**
```
IF EXISTS transaction WHERE seller = memberA AND buyer = memberB:
  Create elimination_entry
```

**If DTRules doesn't support these**: Building consolidated returns is **not feasible** without major engine work.

---

#### 2. External State Management

Implement persistence layer for:
- Deferred intercompany gains (year-over-year tracking)
- SRLY registers (cumulative from acquisition)
- Stock basis (accumulated adjustments)
- ELA balances

**Options**:
- Database (PostgreSQL, MySQL)
- File-based (JSON, XML)
- In-memory cache (Redis) for performance

**Interface**:
```go
// Load prior year state
func LoadPriorYearConsolidatedState(groupID string, taxYear int) (*ConsolidatedState, error)

// Save current year state
func SaveConsolidatedState(state *ConsolidatedState) error
```

---

#### 3. Modular Design

Separate consolidation logic into independent modules:

**Module 1**: Affiliation Testing
- Input: Corporation ownership data
- Output: affiliated_group entity with members

**Module 2**: Intercompany Elimination Engine
- Input: Transactions, member list
- Output: Elimination entries, deferred amounts

**Module 3**: Basis Adjustment Engine
- Input: Member income/loss, distributions
- Output: Stock basis adjustments, ELA

**Module 4**: Consolidation Aggregator
- Input: Member separate TI, eliminations, adjustments
- Output: Consolidated TI

**Benefits**:
- Testable in isolation
- Reusable across phases
- Easier to maintain

---

#### 4. Comprehensive Logging and Audit Trail

Consolidated returns require extensive documentation for IRS audits:

**Log Every**:
- Affiliation determination (why each member is included)
- Intercompany transaction identification
- Deferral/elimination calculations
- Basis adjustment computations
- SRLY register changes
- Consolidation steps

**Audit Trail Format**:
```
[Affiliation Test - Member S1]
  Ownership by M: 85% voting, 90% value
  Result: INCLUDED (meets 80% test)
  IRC Reference: § 1504(a)(2)

[Intercompany Transaction - ID 12345]
  Seller: S1, Buyer: S2
  Type: Inventory sale
  Amount: $50,000, Basis: $30,000
  Gain: $20,000
  Treatment: DEFERRED (Reg. § 1.1502-13)
  Restoration trigger: Buyer sale to third party
```

---

#### 5. Validation and Error Checking

Implement extensive validation:

**Affiliation**:
- ✓ Ownership ≥ 80% (voting and value)
- ✓ No circular ownership
- ✓ No foreign corporations
- ✓ Continuous ownership throughout year

**Intercompany Transactions**:
- ✓ Both parties in same group
- ✓ Transaction type valid
- ✓ Basis data available for asset sales
- ✓ No third-party involvement

**Basis Adjustments**:
- ✓ Parent owns subsidiary
- ✓ Subsidiary income/loss reconciles
- ✓ ELA created only when basis < 0
- ✓ Tier-up chain complete

**Consolidation**:
- ✓ Sum of member TI reconciles to consolidated TI
- ✓ All eliminations accounted for
- ✓ No orphaned deferred gains
- ✓ SRLY limitations applied correctly

---

### 10.5 Success Criteria

If implementing consolidated returns, define success as:

**Phase 1 Success**:
- [ ] Can identify 2-member affiliated group from ownership data
- [ ] Can exclude ineligible entities (foreign, REIT, etc.)
- [ ] Can handle simple mid-year acquisition (pro-ration)

**Phase 2 Success**:
- [ ] Can identify and eliminate intercompany dividends
- [ ] Can identify and eliminate intercompany interest
- [ ] Can defer intercompany inventory sale gains
- [ ] Can restore deferred gains on third-party sale

**Phase 3 Success**:
- [ ] Parent basis increases for subsidiary income
- [ ] Parent basis decreases for subsidiary distributions
- [ ] ELA created when basis goes negative
- [ ] ELA recaptured on stock sale

**Phase 4 Success**:
- [ ] SRLY NOLs identified correctly
- [ ] SRLY limitation applied (NOL limited to member income)
- [ ] Consolidated NOLs offset any member income
- [ ] Section 382 limitation applied before SRLY

**Phase 5 Success**:
- [ ] Consolidated TI = sum of member TI - eliminations + restorations
- [ ] Charitable limitation applied at consolidated level (10% CTI)
- [ ] Interest limitation applied at consolidated level (30% ATI)
- [ ] Tax calculated on consolidated TI at 21%

**Phase 6 Success**:
- [ ] Form 1120 populated with consolidated amounts
- [ ] Form 851 lists all members with correct ownership %
- [ ] Schedule M-3 reconciles consolidated book to tax income
- [ ] All amounts reconcile to member-level detail

**Overall Success**:
- [ ] Complete consolidated return can be generated
- [ ] Audit trail documents all calculations
- [ ] Results match IRS examples from regulations
- [ ] Test coverage ≥ 90% of code paths

---

## Conclusion

Consolidated corporate tax returns represent the **apex of complexity** in corporate taxation. While technically feasible to implement in DTRules, the effort required (6-7 months, 20-29 weeks) is substantial, and the architectural challenges are significant.

**Primary Recommendation**: **DEFER** consolidated returns until after S Corporation support, state tax implementations, and foreign tax credit are complete. These features serve broader markets and provide more immediate value.

**If Proceeding**:
1. Validate DTRules supports multi-entity collections and relationship traversal (BLOCKING prerequisite)
2. Implement MVP (Phases 1-3 only) to prove viability (10-14 weeks)
3. Evaluate market demand before committing to full implementation
4. Consider partnership/integration with existing tax software as alternative

**Alternative Approach**: Focus DTRules on decision table strengths (tax logic, rule evaluation) and integrate with existing consolidation software for multi-entity mechanics.

Consolidated returns are a **specialized enterprise feature** that may not align with DTRules' core value proposition of accessible, rule-based tax automation for small-to-medium businesses.

---

## References

### IRS Resources
- [Form 1120 - U.S. Corporation Income Tax Return](https://www.irs.gov/forms-pubs/about-form-1120)
- [Form 851 - Affiliations Schedule](https://www.irs.gov/forms-pubs/about-form-851)
- [Instructions for Schedule M-3 (Form 1120)](https://www.irs.gov/instructions/i1120sm3)
- [IRM 4.61.13 - Dual Consolidated Losses](https://www.irs.gov/irm/part4/irm_04-061-013)

### U.S. Code
- [26 USC § 1501 - Privilege to File Consolidated Returns](https://uscode.house.gov/view.xhtml?path=/prelim@title26/subtitleA/chapter6&edition=prelim)
- [26 USC § 1502 - Regulations](https://uscode.house.gov/view.xhtml?path=/prelim@title26/subtitleA/chapter6&edition=prelim)
- [26 USC § 1503 - Computation and Payment of Tax](https://uscode.house.gov/view.xhtml?path=/prelim@title26/subtitleA/chapter6&edition=prelim)
- [26 USC § 1504 - Definitions](https://www.law.cornell.edu/uscode/text/26/1504)

### Treasury Regulations
- [26 CFR § 1.1502-13 - Intercompany Transactions](https://www.law.cornell.edu/cfr/text/26/1.1502-13)
- [26 CFR § 1.1502-19 - Excess Loss Accounts](https://www.law.cornell.edu/cfr/text/26/1.1502-19)
- [26 CFR § 1.1502-21 - Net Operating Losses](https://www.law.cornell.edu/cfr/text/26/1.1502-21)
- [26 CFR § 1.1502-32 - Investment Adjustments](https://www.law.cornell.edu/cfr/text/26/1.1502-32)
- [26 CFR § 1.1502-75 - Filing Requirements](https://www.law.cornell.edu/cfr/text/26/1.1502-75)
- [26 CFR § 1.1502-76 - Taxable Year of Members](https://www.law.cornell.edu/cfr/text/26/1.1502-76)

### Federal Register
- [December 2024 - Consolidated Return Regulations Modernization](https://www.federalregister.gov/documents/2024/12/30/2024-29480/revising-consolidated-return-regulations-and-controlled-group-of-corporations-regulations-to-reflect)
- [August 2024 - Dual Consolidated Loss Regulations](https://www.federalregister.gov/documents/2024/08/07/2024-16665/rules-regarding-dual-consolidated-losses-and-the-treatment-of-certain-disregarded-payments)
- [October 2020 - Consolidated NOL Rules](https://www.federalregister.gov/documents/2020/10/27/2020-22974/consolidated-net-operating-losses)

### Professional Resources
- [CPA Exams Mastery - Intercompany Eliminations](https://cpaexamsmastery.com/tcp/3/9/2/)
- [CPA Exams Mastery - SRLY Rules & NOL Interactions](https://cpaexamsmastery.com/tcp/3/9/3/)
- [Tax Adviser - Managing Excess Loss Accounts](https://www.thetaxadviser.com/issues/2019/may/managing-excess-loss-accounts/)
- [Tax Adviser - Complying with SRLY Rules](https://www.thetaxadviser.com/issues/2024/sep/complying-with-the-srly-rules/)
- [RSM - Unified Loss Rules](https://rsmus.com/insights/services/business-tax/unified-loss-rules.html)
- [RSM - M&A Perspective on Filing Requirements](https://rsmus.com/insights/tax-alerts/2024/ma-perspective-who-should-be-filing-that-return-and-when.html)

### Academic/Legal
- [Accounting Insights - Section 1504 Affiliated Groups](https://accountinginsights.org/what-is-section-1504-and-how-does-it-define-affiliated-groups/)
- [CPA Journal - Excess Loss Accounts](http://archives.cpajournal.com/2001/0800/dept/d085001a.htm)

---

**Document Version**: 1.0
**Last Updated**: 2026-03-24
**Author**: DTRules Team (Claude Code Research)
**Next Review**: Upon decision to proceed or defer (Q2 2026)
