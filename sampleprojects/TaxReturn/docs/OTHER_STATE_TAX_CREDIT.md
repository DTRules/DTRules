# Credit for Taxes Paid to Other States

## Overview

This implementation adds support for calculating the credit for taxes paid to other states, preventing double taxation when a taxpayer is a resident of one state but earns income in another state.

## Implementation Details

### Entity Changes

#### 1. Income Entity (`income`)
Added fields to track which state income was earned in:
- `state_earned` (string): State where the income was earned (for non-resident state tax)
- `state_tax_withheld` (double): State tax withheld at source

#### 2. Property Entity (`property`)
Added field to track property location:
- `state` (string): State where property is located (for state tax allocation)

#### 3. State Tax Result Entity (`state_tax_result`)
Added fields to track credit calculation:
- `income_taxed_by_other_states` (double): Income amount taxed by other states
- `taxes_paid_to_other_states` (double): Total taxes paid to other states
- `other_state_tax_credit` (double): Credit for taxes paid to other states

### Decision Tables

#### TABLE 45000: Calculate_Flat_Rate_State_Tax
A generic decision table that calculates state income tax using a simple flat rate and standard deduction. Used for states where full progressive calculations haven't been implemented yet.

**Parameters:**
- `state_code`: Two-letter state code
- `tax_rate`: Flat tax rate for the state
- `std_deduction_single`: Standard deduction for single filers
- `std_deduction_mfj`: Standard deduction for married filing jointly

**Process:**
1. Creates a `state_tax_result` entity for the state
2. Applies appropriate standard deduction based on filing status
3. Calculates tax as: (AGI - Standard Deduction) × Tax Rate
4. Stores result in `job.state_tax_results` array

#### TABLE 45100: Calculate_Other_State_Tax_Credit
Calculates the credit for taxes paid to other states to prevent double taxation.

**How It Works:**
1. Identifies the resident state (from `job.state`)
2. Sums income and taxes from all non-resident states in `job.state_tax_results`
3. Calculates resident state tax on other-state income:
   ```
   resident_state_tax_on_other_income = (other_state_income / total_taxable_income) × resident_state_tax
   ```
4. Credit is the **lesser of**:
   - Taxes actually paid to other states, OR
   - Resident state tax on that same income
5. Updates resident state's `state_tax_result` with the credit
6. Reduces resident state's tax liability by the credit amount

**Integration:**
Called automatically after `Dispatch_State_Tax` in the main tax calculation flow (TABLE 1: Compute_Tax_Return, Action 12).

### State Tax Support

#### Fully Implemented States (Progressive Brackets)
- **Illinois (IL)**: Flat 4.95% rate with personal exemptions (TABLE 41100)
- **New Hampshire (NH)**: Progressive brackets 3%, 5%, 7.5% (TABLE 42700)
- **Montana (MT)**: Progressive brackets 4.7%, 5.9% (TABLE 44900)

#### Simplified Implementation States (Flat Rate)
These states use TABLE 45000 with simplified flat rates for demonstration:
- **New York (NY)**: 6.25% rate (middle bracket approximation)
- **New Jersey (NJ)**: 5.525% rate (middle bracket approximation)
- **California (CA)**: 9.3% rate (middle bracket approximation)
- **Arizona (AZ)**: 2.5% flat rate (actual 2025 rate)

### Constants Added

Located in the `constants` entity in TaxReturn_edd.xml:

#### New York
- `ny_standard_deduction_single`: $8,000
- `ny_standard_deduction_mfj`: $16,050
- `ny_tax_rate_1` through `ny_tax_rate_9`: Progressive rates from 4% to 10.9%

#### New Jersey
- `nj_tax_rate_1` through `nj_tax_rate_7`: Progressive rates from 1.4% to 10.75%

#### California
- `ca_standard_deduction_single`: $5,363
- `ca_standard_deduction_mfj`: $10,726
- `ca_tax_rate_1` through `ca_tax_rate_9`: Progressive rates from 1% to 12.3%

#### Arizona
- `az_tax_rate`: 2.5% flat rate
- `az_standard_deduction_single`: $14,600 (follows federal)
- `az_standard_deduction_mfj`: $29,200 (follows federal)

## Test Cases

### Test Case 1: NY Resident Working in NJ
**File**: `TestCase_MultiState_NY_NJ.xml`

**Scenario:**
- NY resident earns $80,000 working in NJ
- NJ taxes the $80,000 as nonresident income (~$4,420 tax)
- NY taxes all income but provides credit for NJ taxes paid
- Credit = lesser of: NJ tax paid OR NY tax on that income

**Expected Behavior:**
1. NJ calculates nonresident tax on $80,000
2. NY calculates resident tax on full $80,000
3. NY credit reduces double taxation
4. Net result: Taxpayer pays approximately the higher of the two state rates

### Test Case 2: CA Resident with AZ Rental Property
**File**: `TestCase_MultiState_CA_AZ.xml`

**Scenario:**
- CA resident earns $100,000 W-2 in CA
- Owns rental property in AZ generating $30,000 net income
- AZ taxes the $30,000 rental income (~$750 at 2.5%)
- CA taxes all $130,000 but provides credit for AZ taxes

**Expected Behavior:**
1. AZ calculates nonresident tax on $30,000 rental income
2. CA calculates resident tax on full $130,000 (W-2 + rental)
3. CA credit = lesser of AZ tax OR CA tax on the $30,000
4. Multi-state allocation prevents double taxation on rental income

### Test Case 3: Complex Multi-State Scenario
**File**: `TestCase_MultiState_Complex.xml`

**Scenario:**
- CA resident with income from 3 states
- $70,000 W-2 in CA
- $40,000 W-2 in AZ (remote work)
- $20,000 rental income in NJ
- Both AZ and NJ tax their respective income
- CA taxes all income but provides credits for both states

**Expected Behavior:**
1. AZ taxes $40,000 remote work income
2. NJ taxes $20,000 rental income
3. CA taxes full $130,000
4. CA credit = sum of (lesser values) for each state
5. Demonstrates handling of multiple simultaneous state credits

## Credit Calculation Formula

The credit for taxes paid to other states prevents double taxation while ensuring the resident state collects at least its share:

```
For each non-resident state:
  other_state_income = income earned in that state
  other_state_tax_paid = tax paid to that state

Total other-state income = sum of all other_state_income
Total other-state taxes = sum of all other_state_tax_paid

Resident state tax on other income =
  (total_other_state_income / resident_state_taxable_income) × resident_state_tax

Credit = MIN(total_other_state_taxes, resident_state_tax_on_other_income)

Final resident state tax = resident_state_tax - credit
```

This ensures:
1. Taxpayer never pays more than the higher state's rate
2. Resident state always collects at least its share
3. No state tax is completely eliminated (unless other state rate ≥ resident rate)

## Limitations

### Current Implementation
- Simplified flat rates for NY, NJ, and CA (not full progressive brackets)
- Does not handle:
  - Part-year resident scenarios
  - City/local taxes
  - Special allocations for S-corps, partnerships, trusts
  - Reciprocal agreements between states
  - State-specific additions/subtractions to federal AGI

### Future Enhancements
1. Implement full progressive brackets for NY, NJ, CA
2. Add support for part-year resident calculations
3. Handle state-specific income modifications
4. Support for reciprocal state agreements
5. City and local tax calculations
6. State-specific credits beyond other-state credit

## References

### Tax Law
- IRC Section 164(b)(6): Limitation on state and local tax deduction
- State-specific credit forms:
  - New York: Form IT-112-R (Resident Credit)
  - New Jersey: Form NJ-1040, Schedule A (Resident Credit)
  - California: Schedule S (Other State Tax Credit)
  - Arizona: Form 309 (Credit for Taxes Paid to Another State)

### Implementation Files
- **Entity Definitions**: `xml/TaxReturn_edd.xml`
  - Lines 163-168: Income entity multi-state fields
  - Lines 179: Property entity state field
  - Lines 1484-1487: State tax result credit fields
  - Lines 1028-1068: State tax constants

- **Decision Tables**: `xml/TaxReturn_dt.xml`
  - Lines 8685-8935: TABLE 40000 (Dispatch_State_Tax) - Updated with NY, NJ, CA, AZ
  - Lines 9151-9250: TABLE 45000 (Calculate_Flat_Rate_State_Tax) - Generic flat rate calculator
  - Lines 9252-9350: TABLE 45100 (Calculate_Other_State_Tax_Credit) - Credit calculation

- **Test Files**: `testfiles/TestScenarios/`
  - `TestCase_MultiState_NY_NJ.xml`: NY/NJ test case
  - `TestCase_MultiState_CA_AZ.xml`: CA/AZ rental property test
  - `TestCase_MultiState_Complex.xml`: Multiple state scenario

## Usage Example

```xml
<!-- Input: NY resident working in NJ -->
<job>
  <state>NY</state>
  <filing_status>single</filing_status>

  <incomes>
    <income id="1">
      <gross_amount>80000</gross_amount>
      <state_earned>NJ</state_earned>
      <state_tax_withheld>3500</state_tax_withheld>
    </income>
  </incomes>
</job>

<!-- Processing Flow: -->
<!-- 1. Compute_Tax_Return calls Dispatch_State_Tax -->
<!-- 2. Dispatch recognizes NY as resident state -->
<!-- 3. Calculates NY tax on $80,000 -->
<!-- 4. Dispatch also processes NJ nonresident return (if implemented) -->
<!-- 5. Calculate_Other_State_Tax_Credit runs -->
<!-- 6. Credit = MIN(NJ tax paid, NY tax on $80k) -->
<!-- 7. NY tax reduced by credit -->

<!-- Output: -->
<state_tax_results>
  <state_tax_result>
    <state_code>NY</state_code>
    <state_tax_before_credits>5000</state_tax_before_credits>
    <other_state_tax_credit>3500</other_state_tax_credit>
    <state_tax_liability>1500</state_tax_liability>
  </state_tax_result>
  <state_tax_result>
    <state_code>NJ</state_code>
    <state_tax_liability>3500</state_tax_liability>
  </state_tax_result>
</state_tax_results>
```

## Testing

To test the implementation:

1. **Single State (baseline)**:
   ```bash
   # Test existing state calculations still work
   # Run existing NH, IL, MT test cases
   ```

2. **Multi-State Credit**:
   ```bash
   # Run the three multi-state test cases
   java -cp [...] com.dtrules.samples.taxreturn.TestTaxReturn \
     testfiles/TestScenarios/TestCase_MultiState_NY_NJ.xml
   ```

3. **Verify**:
   - Check audit trail shows credit calculation
   - Verify credit = lesser of taxes paid vs. resident tax
   - Confirm resident state tax is reduced by credit
   - Ensure no double taxation occurs

## Acceptance Criteria (from Issue #235)

- [x] Credit calculation per state
- [x] Lesser of: tax paid or resident state tax on same income
- [x] Handles multiple states
- [x] Credit limitations (cannot exceed resident state tax)
- [x] Calculate_Other_State_Tax_Credit table (TABLE 45100)
- [x] 3 test cases: NY resident working NJ, CA resident rental AZ, multi-state
- [x] Documentation (this file)

## Notes

- This implementation provides the foundation for multi-state tax credit calculations
- The simplified flat-rate approach for NY, NJ, and CA allows demonstration of the credit mechanism without implementing full progressive brackets
- Full implementation of progressive brackets for these states can be added following the pattern established in NH (TABLE 42700) and MT (TABLE 44900)
- The credit calculation follows the standard state tax credit methodology used across all states
