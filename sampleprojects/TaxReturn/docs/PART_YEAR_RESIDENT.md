# Part-Year Resident State Tax Calculation

## Overview

The TaxReturn project now supports part-year resident state tax calculations. This allows taxpayers who moved into or out of a state during the tax year to calculate their state tax liability based on their period of residency.

## Implementation

### Entity Fields

The `state_tax_result` entity has been enhanced with the following fields for part-year resident support:

| Field | Type | Description |
|-------|------|-------------|
| `residency_status` | string | Residency type: `full_year_resident`, `part_year_resident`, or `non_resident` |
| `residency_start_date` | date | Date residency began (for part-year residents) |
| `residency_end_date` | date | Date residency ended (for part-year residents) |
| `resident_months` | integer | Number of months as resident (computed, 1-12) |
| `resident_ratio` | double | Ratio of resident months to 12 (computed, 0.0-1.0) |
| `resident_income` | double | Income earned during resident period |
| `non_resident_income` | double | Income sourced to state during non-resident period |
| `allocated_deductions` | double | Deductions allocated to state based on residency period |

### Decision Tables

#### TABLE 39900: Calculate_Part_Year_Allocation

This table processes each `state_tax_result` entry and:

1. **Calculates resident months** from `residency_start_date` and `residency_end_date`
2. **Computes resident_ratio** as `resident_months / 12`
3. **Allocates income** proportionally:
   - `resident_income = federal_agi × resident_ratio`
   - `non_resident_income = federal_agi × (1 - resident_ratio)`
4. **Allocates deductions** proportionally:
   - `allocated_deductions = total_deductions × resident_ratio`

For full-year residents, the resident_ratio is set to 1.0 and no allocation is performed.

#### State-Specific Tax Tables (IL, NH, MT)

All state tax calculation tables have been updated to:

1. **Create or find `state_tax_result` entity** in the `state_tax_results` array
2. **Use `state_tax_result.state_agi`** instead of directly using federal AGI
3. **Prorate standard deductions and exemptions** by multiplying by `resident_ratio`
4. **Populate `state_tax_result` fields** with calculated values

**Illinois (TABLE 41100):**
- Personal exemptions prorated: `num_people × $2,775 × resident_ratio`
- IL retirement income subtraction prorated by resident_ratio

**New Hampshire (TABLE 42700):**
- Standard deduction prorated: `$8,000 (Single) or $16,000 (MFJ) × resident_ratio`
- Progressive brackets (3%, 5%, 7.5%) applied to allocated income

**Montana (TABLE 44900):**
- Standard deduction prorated: `$4,500 (Single) or $11,500 (MFJ) × resident_ratio`
- Progressive brackets (4.7%, 5.9%) applied to allocated income

## Usage

### Declaring Part-Year Residency

To indicate part-year residency, include a `state_tax_results` entry in your tax return XML with:

```xml
<state_tax_results>
   <state_tax_result id="1">
      <state_code>NH</state_code>
      <residency_status>part_year_resident</residency_status>
      <residency_start_date>7/1/2025</residency_start_date>
      <residency_end_date>12/31/2025</residency_end_date>
   </state_tax_result>
</state_tax_results>
```

### Calculation Method

The part-year resident calculation uses the **simplified allocation method**:

1. **Income Allocation**: Federal AGI is allocated based on the ratio of months as a resident to 12 months
2. **Deduction Allocation**: Deductions (standard or itemized) are allocated using the same ratio
3. **Tax Calculation**: State tax is computed on the allocated income and deductions

This is a simplified approach. Some states may require more complex allocation methods based on:
- Income sourcing rules
- Days present in the state
- Specific state adjustments

### Examples

#### Example 1: Moved to NH (6 months)

**Scenario:** Single filer moved from Texas (no state tax) to New Hampshire on July 1, 2025.

```
Federal AGI: $120,000
Resident months: 6 (July 1 - Dec 31)
Resident ratio: 6/12 = 0.5

NH Calculations:
- NH AGI allocated: $120,000 × 0.5 = $60,000
- NH Standard Deduction: $8,000 × 0.5 = $4,000
- NH Taxable Income: $60,000 - $4,000 = $56,000
- NH Tax (3% bracket 1): $56,000 × 3% = $1,680
```

**Test Case:** `TestCase_PartYear_NH_01_MoveIn.xml`

#### Example 2: Moved out of IL (4 months)

**Scenario:** Married filing jointly, moved from Illinois to Texas on April 30, 2025.

```
Federal AGI: $150,000
Resident months: 4 (Jan 1 - April 30)
Resident ratio: 4/12 = 0.3333

IL Calculations:
- IL AGI allocated: $150,000 × 0.3333 = $50,000
- Number of people: 2
- IL Personal Exemptions: 2 × $2,775 × 0.3333 = $1,850
- IL Taxable Income: $50,000 - $1,850 = $48,150
- IL Tax (4.95% flat): $48,150 × 4.95% = $2,383.43
```

**Test Case:** `TestCase_PartYear_IL_02_MoveOut.xml`

#### Example 3: High income part-year (4 months)

**Scenario:** Single filer moved to Montana on September 1, 2025, with high income spanning both tax brackets.

```
Federal AGI: $180,000
Resident months: 4 (Sept 1 - Dec 31)
Resident ratio: 4/12 = 0.3333

MT Calculations:
- MT AGI allocated: $180,000 × 0.3333 = $60,000
- MT Standard Deduction: $4,500 × 0.3333 = $1,500
- MT Taxable Income: $60,000 - $1,500 = $58,500
- MT Tax:
  - Bracket 1: $21,100 × 4.7% = $991.70
  - Bracket 2: ($58,500 - $21,100) × 5.9% = $2,206.60
  - Total: $3,198.30
```

**Test Case:** `TestCase_PartYear_MT_03_HighIncome.xml`

## Test Cases

Three test cases demonstrate part-year resident scenarios:

1. **TestCase_PartYear_NH_01_MoveIn.xml** - Moved into NH mid-year (6 months)
2. **TestCase_PartYear_IL_02_MoveOut.xml** - Moved out of IL early in year (4 months)
3. **TestCase_PartYear_MT_03_HighIncome.xml** - High income spanning multiple brackets (4 months)

## Technical Notes

### Month Calculation

The current implementation calculates resident months using a simple month difference:

```
resident_months = end_month - start_month + 1
```

This is bounded between 1 and 12. For more precise calculations, days could be used instead of months.

### Deduction Allocation

Both standard and itemized deductions are allocated proportionally. This assumes:
- Expenses were incurred evenly throughout the year
- No state-specific deduction rules apply

Some states may have different allocation rules for specific deductions.

### Multiple State Returns

While the `state_tax_results` array supports multiple states, this implementation focuses on single-state part-year scenarios. Moving between two states with income tax (e.g., NY to CA) would require:
- Two `state_tax_result` entries (one for each state)
- Proper income sourcing for each state
- Credit for taxes paid to other states

## Future Enhancements

Potential improvements to part-year resident support:

1. **Day-based allocation** - Use actual days resident instead of months
2. **Income sourcing** - Track which income items are sourced to which state
3. **Multi-state scenarios** - Support for moving between two tax states
4. **State-specific rules** - Implement state-specific allocation methods
5. **Credits for taxes paid** - Calculate credit for taxes paid to other states
6. **Non-resident filing** - Support for non-resident returns with state-sourced income

## References

- **Illinois Form IL-1040:** Part-year resident schedules
- **New Hampshire:** (Hypothetical) Part-year allocation rules
- **Montana Form 2:** Part-year resident instructions
- **IRS Publication 17:** State and local income taxes

## Implementation Date

February 2025 - Issue #233
