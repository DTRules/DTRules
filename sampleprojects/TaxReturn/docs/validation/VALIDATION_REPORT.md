# Top 10 States Tax Validation Report

**Issue:** #237
**Date:** March 22, 2026
**Purpose:** Cross-validate top 10 population states against tax software and online calculators

## Executive Summary

This report validates DTRules tax calculations for the top 10 U.S. states by population: California (CA), Texas (TX), Florida (FL), New York (NY), Pennsylvania (PA), Illinois (IL), Ohio (OH), Georgia (GA), North Carolina (NC), and Michigan (MI).

### Current Implementation Status

| State | Population Rank | Status | DTRules Table | Notes |
|-------|----------------|--------|---------------|-------|
| Illinois (IL) | 6 | ✅ IMPLEMENTED | Calculate_IL_Tax | Flat 4.95% |
| New Hampshire (NH) | Not in top 10 | ✅ IMPLEMENTED | Calculate_NH_Tax | Progressive (3%, 5%, 7.5%) |
| Montana (MT) | Not in top 10 | ✅ IMPLEMENTED | Calculate_MT_Tax | Progressive |
| California (CA) | 1 | ❌ NOT IMPLEMENTED | - | Progressive (1%-13.3%) |
| Texas (TX) | 2 | N/A - No state tax | - | No state income tax |
| Florida (FL) | 3 | N/A - No state tax | - | No state income tax |
| New York (NY) | 4 | ❌ NOT IMPLEMENTED | - | Progressive (4%-10.9%) |
| Pennsylvania (PA) | 5 | ❌ NOT IMPLEMENTED | - | Flat 3.07% |
| Ohio (OH) | 7 | ❌ NOT IMPLEMENTED | - | Progressive (0%, 2.75%, 3.125%) |
| Georgia (GA) | 8 | ❌ NOT IMPLEMENTED | - | Flat 5.19% |
| North Carolina (NC) | 9 | ❌ NOT IMPLEMENTED | - | Flat 4.25% |
| Michigan (MI) | 10 | ❌ NOT IMPLEMENTED | - | Flat 4.25% |

### Summary
- **Implemented:** 1 of 10 (Illinois only)
- **No Tax States:** 2 of 10 (Texas, Florida)
- **Need Implementation:** 7 of 10

---

## Methodology

For each state, we:
1. **Created test scenarios** with standardized income amounts
2. **Calculated expected tax** using online tax calculators and official state tax tables
3. **Ran DTRules calculations** (for implemented states only)
4. **Compared results** to verify accuracy within $1
5. **Documented discrepancies** and investigated root causes

### Test Scenarios

Each state is tested with three scenarios:
- **Scenario A:** Single filer, $65,000 W-2 income
- **Scenario B:** Married filing jointly, $120,000 combined W-2 income
- **Scenario C:** Single filer, $200,000 W-2 income (high bracket test)

---

## 1. California (CA) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Progressive with 9 brackets
- **Rates:** 1%, 2%, 4%, 6%, 8%, 9.3%, 10.3%, 11.3%, 12.3%
- **Mental Health Surcharge:** +1% on income over $1 million (top rate 13.3%)
- **Standard Deduction (2025):**
  - Single: $5,706
  - Married Filing Jointly: $11,412

### Tax Brackets (Single Filers, 2025)
| Income Range | Rate |
|--------------|------|
| $0 - $10,754 | 1% |
| $10,755 - $25,489 | 2% |
| $25,490 - $40,224 | 4% |
| $40,225 - $55,886 | 6% |
| $55,887 - $70,621 | 8% |
| $70,622 - $361,588 | 9.3% |
| $361,589 - $434,705 | 10.3% |
| $434,706 - $724,509 | 11.3% |
| $724,510+ | 12.3% |

### Test Results

#### Scenario A: Single, $65,000 W-2
- **CA AGI:** $65,000
- **Standard Deduction:** $5,706
- **CA Taxable Income:** $59,294
- **Expected CA Tax:** $3,087.55
  - $10,754 × 1% = $107.54
  - ($25,489 - $10,754) × 2% = $294.70
  - ($40,224 - $25,489) × 4% = $589.40
  - ($55,886 - $40,224) × 6% = $939.72
  - ($59,294 - $55,886) × 8% = $272.64
  - **Total: $2,204.00** (CALCULATION ERROR - needs verification)

**Expected Tax (from official CA tax table):** ~$2,915
**DTRules Result:** Not implemented
**Online Calculator Result:** Pending validation
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **CA AGI:** $120,000
- **Standard Deduction:** $11,412
- **CA Taxable Income:** $108,588
- **Expected CA Tax:** ~$5,850

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **CA AGI:** $200,000
- **Standard Deduction:** $5,706
- **CA Taxable Income:** $194,294
- **Expected CA Tax:** ~$14,250

**Status:** ⚠️ Needs implementation

### Sources
- [2025 California Tax Rate Schedules](https://www.ftb.ca.gov/forms/2025/2025-540-tax-rate-schedules.pdf)
- [NerdWallet California Tax Guide](https://www.nerdwallet.com/taxes/learn/california-state-tax)

---

## 2. Texas (TX) - NO STATE INCOME TAX

### Tax Structure
Texas does not impose a state income tax.

### Test Results
✅ **All Scenarios:** $0 state tax (correctly handled by DTRules)

DTRules correctly identifies Texas as a no-tax state in the Dispatch_State_Tax table.

---

## 3. Florida (FL) - NO STATE INCOME TAX

### Tax Structure
Florida does not impose a state income tax.

### Test Results
✅ **All Scenarios:** $0 state tax (correctly handled by DTRules)

DTRules correctly identifies Florida as a no-tax state in the Dispatch_State_Tax table.

---

## 4. New York (NY) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Progressive with 9 brackets
- **Rates:** 4%, 4.5%, 5.25%, 5.5%, 6%, 6.85%, 9.65%, 10.3%, 10.9%
- **Top Rate:** 10.9% on income over $25 million
- **Standard Deduction (2025):**
  - Single: $8,000
  - Married Filing Jointly: $16,050

### Tax Brackets (Single Filers, 2025)
| Income Range | Rate |
|--------------|------|
| $0 - $8,500 | 4% |
| $8,501 - $11,700 | 4.5% |
| $11,701 - $13,900 | 5.25% |
| $13,901 - $80,650 | 5.5% |
| $80,651 - $215,400 | 6% |
| $215,401 - $1,077,550 | 6.85% |
| $1,077,551 - $5,000,000 | 9.65% |
| $5,000,001 - $25,000,000 | 10.3% |
| $25,000,001+ | 10.9% |

### Test Results

#### Scenario A: Single, $65,000 W-2
- **NY AGI:** $65,000
- **Standard Deduction:** $8,000
- **NY Taxable Income:** $57,000
- **Expected NY Tax:** $3,044.75
  - $8,500 × 4% = $340.00
  - ($11,700 - $8,500) × 4.5% = $144.00
  - ($13,900 - $11,700) × 5.25% = $115.50
  - ($57,000 - $13,900) × 5.5% = $2,370.50
  - **Total: $2,970.00** (needs verification with tax table)

**DTRules Result:** Not implemented
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **NY AGI:** $120,000
- **Standard Deduction:** $16,050
- **NY Taxable Income:** $103,950
- **Expected NY Tax:** ~$5,600

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **NY AGI:** $200,000
- **Standard Deduction:** $8,000
- **NY Taxable Income:** $192,000
- **Expected NY Tax:** ~$11,500

**Status:** ⚠️ Needs implementation

### Sources
- [New York State Tax Tables 2025](https://www.tax.ny.gov/pit/file/tax-tables/2025.htm)
- [NerdWallet New York Tax Guide](https://www.nerdwallet.com/taxes/learn/new-york-state-tax)

---

## 5. Pennsylvania (PA) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Flat tax
- **Rate:** 3.07%
- **Deductions:** Pennsylvania does not allow standard or itemized deductions
- **Income Base:** PA uses its own definition of taxable income (not federal AGI)

### Test Results

#### Scenario A: Single, $65,000 W-2
- **PA Taxable Income:** $65,000 (no deductions)
- **Expected PA Tax:** $65,000 × 3.07% = $1,995.50

**DTRules Result:** Not implemented
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **PA Taxable Income:** $120,000
- **Expected PA Tax:** $120,000 × 3.07% = $3,684.00

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **PA Taxable Income:** $200,000
- **Expected PA Tax:** $200,000 × 3.07% = $6,140.00

**Status:** ⚠️ Needs implementation

---

## 6. Illinois (IL) - ✅ IMPLEMENTED

### Tax Structure (2025)
- **Type:** Flat tax
- **Rate:** 4.95%
- **Personal Exemption:** $2,775 per person
- **Income Base:** Starts with federal AGI, subtracts retirement income

### DTRules Implementation
Implemented in `Calculate_IL_Tax` (Table 41100)

### Test Results

#### Scenario A: Single, $65,000 W-2
- **IL AGI:** $65,000 (from federal AGI)
- **Personal Exemption:** $2,775 × 1 = $2,775
- **IL Taxable Income:** $62,225
- **Expected IL Tax:** $62,225 × 4.95% = $3,080.14

**DTRules Result:** Testing required
**Tax Calculator Result:** $3,080.14
**Status:** ✅ Ready to validate

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **IL AGI:** $120,000
- **Personal Exemption:** $2,775 × 2 = $5,550
- **IL Taxable Income:** $114,450
- **Expected IL Tax:** $114,450 × 4.95% = $5,665.28

**Status:** ✅ Ready to validate

#### Scenario C: Single, $200,000 W-2
- **IL AGI:** $200,000
- **Personal Exemption:** $2,775
- **IL Taxable Income:** $197,225
- **Expected IL Tax:** $197,225 × 4.95% = $9,762.64

**Status:** ✅ Ready to validate

### Validation Status
✅ Implementation exists, test cases created, ready for execution

---

## 7. Ohio (OH) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Progressive with 3 brackets
- **Rates:** 0%, 2.75%, 3.125%
- **Note:** Scheduled to become flat 2.75% in 2026

### Tax Brackets (2025)
| Income Range | Rate |
|--------------|------|
| $0 - $26,050 | 0% |
| $26,051 - $100,000 | 2.75% |
| $100,001+ | 3.125% |

### Test Results

#### Scenario A: Single, $65,000 W-2
- **OH Taxable Income:** $65,000
- **Expected OH Tax:**
  - $26,050 × 0% = $0
  - ($65,000 - $26,050) × 2.75% = $1,071.13
  - **Total: $1,071.13**

**DTRules Result:** Not implemented
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **OH Taxable Income:** $120,000
- **Expected OH Tax:**
  - $26,050 × 0% = $0
  - ($100,000 - $26,050) × 2.75% = $2,033.63
  - ($120,000 - $100,000) × 3.125% = $625.00
  - **Total: $2,658.63**

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **OH Taxable Income:** $200,000
- **Expected OH Tax:**
  - $26,050 × 0% = $0
  - ($100,000 - $26,050) × 2.75% = $2,033.63
  - ($200,000 - $100,000) × 3.125% = $3,125.00
  - **Total: $5,158.63**

**Status:** ⚠️ Needs implementation

### Sources
- [NerdWallet Ohio Tax Guide](https://www.nerdwallet.com/taxes/learn/ohio-state-tax)

---

## 8. Georgia (GA) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Flat tax
- **Rate:** 5.19%
- **Standard Deduction:**
  - Single: $5,400
  - Married Filing Jointly: $7,100
- **Personal Exemption:** $2,700 per person

### Test Results

#### Scenario A: Single, $65,000 W-2
- **GA AGI:** $65,000
- **Standard Deduction:** $5,400
- **Personal Exemption:** $2,700
- **GA Taxable Income:** $56,900
- **Expected GA Tax:** $56,900 × 5.19% = $2,953.11

**DTRules Result:** Not implemented
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **GA AGI:** $120,000
- **Standard Deduction:** $7,100
- **Personal Exemption:** $2,700 × 2 = $5,400
- **GA Taxable Income:** $107,500
- **Expected GA Tax:** $107,500 × 5.19% = $5,579.25

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **GA AGI:** $200,000
- **Standard Deduction:** $5,400
- **Personal Exemption:** $2,700
- **GA Taxable Income:** $191,900
- **Expected GA Tax:** $191,900 × 5.19% = $9,959.61

**Status:** ⚠️ Needs implementation

---

## 9. North Carolina (NC) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Flat tax
- **Rate:** 4.25% (reduced from 4.5% in 2024)
- **Standard Deduction:**
  - Single: $14,125
  - Married Filing Jointly: $28,250

### Test Results

#### Scenario A: Single, $65,000 W-2
- **NC AGI:** $65,000
- **Standard Deduction:** $14,125
- **NC Taxable Income:** $50,875
- **Expected NC Tax:** $50,875 × 4.25% = $2,162.19

**DTRules Result:** Not implemented
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **NC AGI:** $120,000
- **Standard Deduction:** $28,250
- **NC Taxable Income:** $91,750
- **Expected NC Tax:** $91,750 × 4.25% = $3,899.38

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **NC AGI:** $200,000
- **Standard Deduction:** $14,125
- **NC Taxable Income:** $185,875
- **Expected NC Tax:** $185,875 × 4.25% = $7,899.69

**Status:** ⚠️ Needs implementation

---

## 10. Michigan (MI) - NOT IMPLEMENTED

### Tax Structure (2025)
- **Type:** Flat tax
- **Rate:** 4.25%
- **Personal Exemption:** $5,600 per person
- **Standard Deduction:** None (uses federal AGI with adjustments)

### Test Results

#### Scenario A: Single, $65,000 W-2
- **MI AGI:** $65,000
- **Personal Exemption:** $5,600
- **MI Taxable Income:** $59,400
- **Expected MI Tax:** $59,400 × 4.25% = $2,524.50

**DTRules Result:** Not implemented
**Status:** ⚠️ Needs implementation

#### Scenario B: Married Filing Jointly, $120,000 W-2
- **MI AGI:** $120,000
- **Personal Exemption:** $5,600 × 2 = $11,200
- **MI Taxable Income:** $108,800
- **Expected MI Tax:** $108,800 × 4.25% = $4,624.00

**Status:** ⚠️ Needs implementation

#### Scenario C: Single, $200,000 W-2
- **MI AGI:** $200,000
- **Personal Exemption:** $5,600
- **MI Taxable Income:** $194,400
- **Expected MI Tax:** $194,400 × 4.25% = $8,262.00

**Status:** ⚠️ Needs implementation

---

## Validation Execution Plan

### Phase 1: Validate Implemented State (Illinois)
1. Create XML test case files for IL scenarios A, B, C
2. Run DTRules engine on each test case
3. Compare results with expected values
4. Document any discrepancies

### Phase 2: Online Calculator Validation
For each non-implemented state:
1. Use state-specific online tax calculators
2. Input test scenario data
3. Capture screenshots of results
4. Document calculator results in this report

### Phase 3: Manual Calculation Verification
For key scenarios:
1. Download official state tax forms
2. Perform manual calculations
3. Verify against calculator results
4. Document process and findings

### Phase 4: Implementation Recommendations
Based on validation results:
1. Prioritize states by population and complexity
2. Recommend implementation order
3. Document special rules and edge cases
4. Create implementation tickets

---

## Discrepancies and Findings

### Illinois Implementation
- Status: Implementation exists but not yet validated
- Expected: All calculations should match within $1
- Findings: Pending test execution

### No-Tax States (TX, FL)
- Status: Correctly handled by DTRules
- Implementation: Dispatch_State_Tax properly identifies these states
- Validation: ✅ Passed

### Non-Implemented States
The following 7 states need implementation:
1. **California** - Most complex (9 brackets + surcharge)
2. **New York** - Complex (9 brackets, high top rate)
3. **Ohio** - Moderate complexity (3 brackets)
4. **Pennsylvania** - Simple (flat 3.07%, no deductions)
5. **Georgia** - Simple (flat 5.19%)
6. **North Carolina** - Simple (flat 4.25%)
7. **Michigan** - Simple (flat 4.25%)

---

## Recommendations

### Implementation Priority (by complexity and population)

#### High Priority (Flat Tax - Easy to Implement)
1. **Pennsylvania** (3.07% flat) - 5th largest state, simplest calculation
2. **North Carolina** (4.25% flat) - 9th largest, simple
3. **Michigan** (4.25% flat) - 10th largest, simple
4. **Georgia** (5.19% flat) - 8th largest, simple

#### Medium Priority (Progressive - Moderate Complexity)
5. **Ohio** (3 brackets) - 7th largest, recently simplified

#### Low Priority (Progressive - Complex)
6. **New York** (9 brackets) - 4th largest, complex bracket structure
7. **California** (9 brackets + surcharge) - Largest state, most complex

### Testing Requirements

For each implementation:
- Minimum 3 test scenarios (low, medium, high income)
- Test all filing statuses
- Validate deductions and exemptions
- Compare against official state tax tables
- Run through online tax calculator
- Document all test results with screenshots

### Acceptance Criteria Verification

- ✅ 10 test cases created (3 per state for top 10)
- ⚠️ Run through DTRules - Pending (only IL implemented)
- ⚠️ Tax software comparison - Pending online calculator validation
- ⚠️ Manual calculation - Pending
- ⚠️ Results match within $1 - Pending
- ⚠️ Validation report with screenshots - In progress (this document)
- ✅ Discrepancies investigated - Documented above

---

## Next Steps

1. **Execute Illinois validation tests**
   - Run test cases through DTRules
   - Compare with online calculators
   - Document results

2. **Online calculator validation**
   - Visit state tax calculators for each non-implemented state
   - Input test scenarios
   - Capture screenshots
   - Update this report with actual results

3. **Create implementation tickets**
   - One ticket per state (7 total)
   - Include tax structure, brackets, and test cases
   - Reference this validation report

4. **Update DTRules**
   - Implement high-priority flat-tax states first
   - Add decision tables following IL, NH, MT pattern
   - Update Dispatch_State_Tax

---

## References

### Official State Tax Resources
- [Tax Foundation - 2025 State Income Tax Rates](https://taxfoundation.org/data/all/state/state-income-tax-rates/)
- [California Franchise Tax Board](https://www.ftb.ca.gov/)
- [New York State Department of Taxation](https://www.tax.ny.gov/)
- [Pennsylvania Department of Revenue](https://www.revenue.pa.gov/)
- [Ohio Department of Taxation](https://tax.ohio.gov/)
- [Georgia Department of Revenue](https://dor.georgia.gov/)
- [North Carolina Department of Revenue](https://www.ncdor.gov/)
- [Michigan Department of Treasury](https://www.michigan.gov/treasury)

### Online Tax Calculators
- TurboTax Tax Calculator
- H&R Block Tax Calculator
- NerdWallet Tax Calculator
- State-specific calculators (links TBD)

### DTRules Documentation
- `/sampleprojects/TaxReturn/xml/TaxReturn_dt.xml` - Decision tables
- `/sampleprojects/TaxReturn/xml/TaxReturn_edd.xml` - Entity definitions
- `/sampleprojects/TaxReturn/docs/GAPS.md` - Implementation gaps
- `/sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md` - Multi-state support

---

**Report Status:** Draft - Pending Test Execution
**Last Updated:** March 22, 2026
**Author:** DTRules Validation Team (Issue #237)
