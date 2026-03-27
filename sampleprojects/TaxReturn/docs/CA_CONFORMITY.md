# California Federal Conformity and AGI Calculation

## Overview

California partially conforms to federal tax law as of January 1, 2025 (SB 711). This means California uses federal Adjusted Gross Income (AGI) as a starting point but makes several state-specific adjustments via Form 540 Schedule CA.

## Implementation Status

**Part 1: CA AGI Calculation - COMPLETE** (Issue #195)
- TABLE 43001: Calculate_CA_AGI implemented
- EDD constants added for CA conformity
- Result fields added for CA AGI tracking

**Part 2: CA Tax Calculation - PENDING** (Future implementation)
- Will implement progressive tax brackets (1% to 13.3%)
- Will implement Mental Health Services Tax (1% on income over $1M)
- Will use CA AGI from Part 1

## California AGI Conformity (Schedule CA)

### IRC Conformity Date
- California conforms to IRC as of: **January 1, 2025** (SB 711)
- Constant: `ca_conformity_date = "2025-01-01"`

### Major Federal-to-California AGI Adjustments

California makes the following major adjustments to federal AGI:

#### 1. Social Security Benefits
- **Federal**: Partially taxable (0%, 50%, or 85% based on provisional income)
- **California**: 100% excludable from gross income
- **Schedule CA**: Subtraction adjustment
- **Impact**: CA AGI is lower than federal AGI for taxpayers with SS benefits

#### 2. Unemployment Compensation
- **Federal**: Fully taxable
- **California**: 100% excludable from gross income
- **Schedule CA**: Subtraction adjustment
- **Impact**: CA AGI is lower than federal AGI for taxpayers with unemployment

#### 3. Military Retirement Pay
- **Federal**: Fully taxable
- **California**: Up to $20,000 excludable (2025-2029 per AB 1786)
- **Schedule CA**: Subtraction adjustment (capped at $20k)
- **Constant**: `ca_military_retirement_exclusion = 20000`
- **Impact**: CA AGI is lower than federal AGI for military retirees
- **Note**: Exclusion applies to uniformed services retirement pay

### Other CA Conformity Differences (Not Currently Implemented)

The following differences exist but are not included in the simple implementation:

#### Additions to Federal AGI
- Foreign earned income exclusion (Form 2555) - added back
- Combat pay election - added back
- State/local tax refunds if itemized federally - added back

#### Subtractions from Federal AGI
- State disability insurance (SDI) payments
- CA lottery winnings
- Certain federal tax credits that reduce federal AGI

## Implementation Details

### TABLE 43001: Calculate_CA_AGI

**Type**: FIRST (executes first matching column)

**Input**:
- `result.agi` - Federal AGI from Form 1040 Line 11
- `job.incomes` - Array of income items with types and amounts

**Output**:
- `result.ca_agi` - California AGI after Schedule CA adjustments

**Logic**:
1. Initialize CA AGI = Federal AGI
2. Check for Social Security income (type = "social_security")
   - If found, subtract entire amount from CA AGI
3. Check for unemployment compensation (type = "unemployment")
   - If found, subtract entire amount from CA AGI
4. Check for military retirement (type = "pension", description contains "military" or "uniformed services")
   - If found, subtract up to $20,000 from CA AGI
5. Log all adjustments to audit trail

### Constants Added to EDD

**AGI Conformity Constants**:
- `ca_conformity_date` - "2025-01-01"
- `ca_military_retirement_exclusion` - 20000

**Future Tax Calculation Constants** (already added for Part 2):
- Standard deductions by filing status
- Progressive tax brackets (9 brackets)
- Tax rates (1% to 13.3%)
- Mental Health Services Tax rate (1%) and threshold ($1M)

### Result Fields Added

**CA AGI Field**:
- `ca_agi` - California AGI after Schedule CA adjustments

**Future CA Tax Fields** (for Part 2):
- `ca_standard_deduction`
- `ca_taxable_income`
- `ca_regular_tax`
- `ca_mhst` (Mental Health Services Tax)
- `ca_tax` (total)

## Test Cases

To verify CA AGI differs from federal AGI:

### Test Case 1: Social Security Income
```xml
<job>
  <state>CA</state>
  <incomes>
    <income type="social_security" gross_amount="30000"/>
  </incomes>
</job>
```
Expected: CA AGI = Federal AGI - $30,000

### Test Case 2: Unemployment Compensation
```xml
<job>
  <state>CA</state>
  <incomes>
    <income type="unemployment" gross_amount="15000"/>
  </incomes>
</job>
```
Expected: CA AGI = Federal AGI - $15,000

### Test Case 3: Military Retirement
```xml
<job>
  <state>CA</state>
  <incomes>
    <income type="pension" description="military retirement" gross_amount="50000"/>
  </incomes>
</job>
```
Expected: CA AGI = Federal AGI - $20,000 (capped at exclusion limit)

### Test Case 4: Multiple Adjustments
```xml
<job>
  <state>CA</state>
  <incomes>
    <income type="social_security" gross_amount="25000"/>
    <income type="unemployment" gross_amount="8000"/>
    <income type="pension" description="uniformed services retirement" gross_amount="30000"/>
  </incomes>
</job>
```
Expected: CA AGI = Federal AGI - $25,000 - $8,000 - $20,000 = Federal AGI - $53,000

## References

- **Form 540**: California Resident Income Tax Return
- **Schedule CA**: California Adjustments - Residents
- **FTB Pub 1001**: Supplemental Guidelines to California Adjustments
- **SB 711**: IRC conformity as of January 1, 2025
- **AB 1786**: Military retirement exclusion ($20k for 2025-2029)
- **Rev. & Tax. Code 17041**: California tax rates (for Part 2)
- **Rev. & Tax. Code 17043**: Mental Health Services Tax (for Part 2)

## Simplifications

This implementation focuses on the **3-5 major differences** as specified:

**Implemented**:
1. Social Security exclusion (100%)
2. Unemployment compensation exclusion (100%)
3. Military retirement exclusion (up to $20k)

**Not Implemented** (would add complexity beyond scope):
- Foreign earned income add-backs
- Combat pay add-backs
- State tax refund add-backs
- Lottery winnings
- SDI exclusions
- Itemized deduction differences
- Capital gains treatment differences (CA has no preferential rate)

## Next Steps (Part 2)

Part 2 will implement full California tax calculation:
1. Apply CA standard deduction or itemized deductions
2. Calculate CA taxable income
3. Apply progressive tax brackets (1% to 13.3%)
4. Calculate Mental Health Services Tax (1% on income over $1M)
5. Handle CA-specific credits
6. Store result in `state_tax_result` entity

Part 2 will use the CA AGI calculated by TABLE 43001 as its starting point.
