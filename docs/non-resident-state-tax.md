# Non-Resident State Tax Implementation

## Overview

This feature implements non-resident state income tax calculations for the DTRules tax return system. When taxpayers earn income in states where they are not residents, they must file non-resident tax returns in those states and only pay tax on the income earned ("sourced") in that state.

## Architecture

### Entity Structure

The `state_tax_result` entity (TaxReturn_edd.xml:1476-1489) contains:
- `state_code`: Two-letter state code
- `is_resident`: Boolean indicating resident vs non-resident status
- `state_source_income`: Income earned in this state (for non-residents)
- `state_agi`: State adjusted gross income
- `state_taxable_income`: Taxable income after deductions
- `state_tax_before_credits`: Tax before credits
- `state_credits`: State tax credits
- `state_tax_liability`: Final tax liability
- `state_withholding`: State tax withheld
- `state_refund_or_owed`: Refund or amount owed

### Income Allocation

Income sources specify their state via:
- `taxpayer.w2_state_source`: State where W-2 wages were earned
- `taxpayer.se_state_source`: State where self-employment income was earned
- `income.state_source`: State where other income was earned
- `property.state_location`: State where rental property is located

### Tables

#### TABLE 40500: Calculate_State_Source_Income
Allocates income to states based on source location:
1. Iterates through all taxpayers, incomes, and properties
2. Accumulates income by state in a hash map
3. Creates `state_tax_result` entities for each state with income
4. Marks the taxpayer's resident state (job.state) as `is_resident=true`
5. All other states are marked as `is_resident=false`

#### TABLE 40000: Dispatch_State_Tax
Multi-state tax dispatcher:
1. Calls `Calculate_State_Source_Income` to allocate income
2. Iterates through each `state_tax_result`
3. Pushes state_tax_result entity onto stack
4. Dispatches to state-specific calculation based on `state_code`:
   - Illinois: Calculate_IL_Tax
   - New Hampshire: Calculate_NH_Tax
   - Montana: Calculate_MT_Tax

#### State-Specific Tables (41100, 42700, 44900)

Each state calculation table now:
1. Checks `current_state.is_resident` (state_tax_result on entity stack)
2. **For Residents**: Uses full federal AGI (`result.agi`)
3. **For Non-Residents**: Uses only state-source income (`current_state.state_source_income`)
4. Applies deductions:
   - **Residents**: Full deductions/exemptions
   - **Non-Residents**: Proportional deductions based on ratio of state income to total AGI
5. Calculates tax using state-specific rates and brackets
6. Stores results in `current_state` (state_tax_result entity)

## Implementation Details

### Illinois (Calculate_IL_Tax)
- **Tax Rate**: Flat 4.95%
- **Personal Exemption**: $2,775 per person
- **Non-Resident Logic**:
  - Uses IL source income only
  - Exemptions prorated by ratio: (IL AGI / Federal AGI) × Total Exemptions
  - Subtracts retirement income only for residents

### New Hampshire (Calculate_NH_Tax)
- **Tax Brackets**: Progressive (3%, 5%, 7.5%)
- **Standard Deduction**: $8,000 (Single), $16,000 (MFJ)
- **Non-Resident Logic**:
  - Uses NH source income only
  - Standard deduction prorated by ratio
  - Applies progressive brackets to taxable income

### Montana (Calculate_MT_Tax)
- **Tax Brackets**: 4.7% / 5.9% (thresholds vary by filing status)
- **Standard Deduction**: $4,500 (Single), $11,500 (MFJ)
- **Non-Resident Logic**:
  - Uses MT source income only
  - Standard deduction prorated by ratio
  - Applies progressive brackets to taxable income

## Test Cases

### TestCase_NonResident_01_CA_Remote_IL.xml
- **Scenario**: California resident working remotely for Illinois employer
- **Expected**: IL tax = $0 (no IL-source income, work performed in CA)

### TestCase_NonResident_02_TX_MT_Rental.xml
- **Scenario**: Texas resident with rental property in Montana
- **Expected**: MT tax on $18,000 rental income = $634.50
- **Calculation**:
  - MT AGI: $18,000
  - Standard deduction: $4,500
  - Taxable: $13,500
  - Tax: $13,500 × 4.7% = $634.50

### TestCase_NonResident_03_Multistate_Business.xml
- **Scenario**: Multi-state business income
- **Expected**: Multiple state tax returns with proper allocation

## Usage

To specify non-resident income, set the state_source fields in test data:

```xml
<taxpayer id="1">
  <w2_wages>75000</w2_wages>
  <w2_state_source>IL</w2_state_source>  <!-- Earned in IL -->
</taxpayer>

<property id="1">
  <type>rental</type>
  <state_location>MT</state_location>  <!-- Property in MT -->
  <rental_net_income>18000</rental_net_income>
</property>
```

The system automatically:
1. Allocates income to states
2. Determines resident vs non-resident status
3. Calculates appropriate tax for each state
4. Stores results in separate `state_tax_result` entities

## Limitations

- Only IL, NH, and MT are currently implemented
- State credits not yet implemented
- Part-year resident scenarios not yet supported
- Reciprocal agreements between states not modeled

## Future Enhancements

1. Add more states (CA, NY, etc.)
2. Implement state tax credits
3. Handle part-year residents
4. Model reciprocal agreements
5. Add state withholding calculations
6. Support amended returns for multi-state scenarios
