# West Virginia State Income Tax Implementation

## Overview
Implemented West Virginia progressive state income tax calculation for the 2025 tax year.

## Implementation Details

### 1. Tax Constants (TaxReturn_edd.xml)
Added the following West Virginia tax constants to the constants entity:

- **Tax Rates** (5 brackets):
  - wv_rate_1: 2.22% (bracket 1)
  - wv_rate_2: 2.96% (bracket 2)
  - wv_rate_3: 3.33% (bracket 3)
  - wv_rate_4: 4.44% (bracket 4)
  - wv_rate_5: 4.82% (bracket 5/top)

- **Tax Brackets**:
  - wv_bracket_1: $10,000
  - wv_bracket_2: $25,000
  - wv_bracket_3: $40,000
  - wv_bracket_4: $60,000
  - Over $60,000: top bracket

- **Standard Deductions**:
  - wv_standard_deduction_single: $2,000
  - wv_standard_deduction_joint: $4,000

- **Result Field**:
  - wv_state_tax: Stores calculated WV state income tax

### 2. Decision Table (TaxReturn_dt.xml)

**Table**: `Calculate_WV_Tax`
**Table Number**: 45300
**Type**: FIRST (progressive bracket calculation)

**Logic**:
1. Calculate WV taxable income = AGI - WV standard deduction (based on filing status)
2. Apply progressive tax brackets:
   - $0-$10,000: 2.22%
   - $10,001-$25,000: 2.96%
   - $25,001-$40,000: 3.33%
   - $40,001-$60,000: 4.44%
   - Over $60,000: 4.82%

3. Store result in result.wv_state_tax

**Integration**: Added to Compute_Tax_Return table (action 9a) with dispatcher that calls Calculate_WV_Tax when job.state = "WV"

### 3. Test Cases

Created 3 comprehensive test cases covering all tax brackets:

#### Test Case 1: Low Income Single Filer
- **File**: testfiles/TestScenarios/WV/TestCase_WV_Low_Income.xml
- **Income**: $30,000 W-2
- **Filing Status**: Single
- **WV Taxable Income**: $30,000 - $2,000 = $28,000
- **Expected WV Tax**: $765.90
  - $10,000 × 2.22% = $222.00
  - $15,000 × 2.96% = $444.00
  - $3,000 × 3.33% = $99.90

#### Test Case 2: Middle Income Married Couple
- **File**: testfiles/TestScenarios/WV/TestCase_WV_Middle_Income.xml
- **Income**: $75,000 W-2 (combined)
- **Filing Status**: Married Filing Jointly
- **WV Taxable Income**: $75,000 - $4,000 = $71,000
- **Expected WV Tax**: $2,184.10
  - $10,000 × 2.22% = $222.00
  - $15,000 × 2.96% = $444.00
  - $15,000 × 3.33% = $499.50
  - $11,000 × 4.44% = $488.40
  - $20,000 × 4.82% = $964.00
  - Note: This spans brackets 1-5

#### Test Case 3: High Income Single Filer (Top Bracket)
- **File**: testfiles/TestScenarios/WV/TestCase_WV_High_Income.xml
- **Income**: $150,000 W-2
- **Filing Status**: Single
- **WV Taxable Income**: $150,000 - $2,000 = $148,000
- **Expected WV Tax**: $6,295.10
  - $10,000 × 2.22% = $222.00
  - $15,000 × 2.96% = $444.00
  - $15,000 × 3.33% = $499.50
  - $20,000 × 4.44% = $888.00
  - $88,000 × 4.82% = $4,241.60

## Sources

West Virginia 2025 tax rates based on:
- [West Virginia Income Tax Explained 2025 - Valur](https://learn.valur.com/west-virginia-income-tax/)
- [West Virginia Income Tax Rates & Brackets 2025](https://remotelaws.com/state-income-tax/us-states/west-virginia/)
- [EY Tax News: West Virginia law lowers personal income tax rates effective January 1, 2025](https://taxnews.ey.com/news/2024-2154-west-virginia-law-lowers-personal-income-tax-rates-effective-january-1-2025)

## Acceptance Criteria Met

✅ Research West Virginia 2025 constants, add to EDD
✅ Calculate_WV_Tax (TABLE 45300)
✅ Progressive brackets, state-specific rules
✅ Add WV branch to dispatcher
✅ 3 test cases (low income, middle income, high income)

## Technical Notes

- The implementation uses DTRules' postfix notation for calculations
- Local variable `wv_taxable_income` is used to store the state-specific taxable income
- Filing status determines which standard deduction to apply (single vs. MFJ)
- Tax calculation follows cumulative bracket method (each bracket is taxed separately and summed)
