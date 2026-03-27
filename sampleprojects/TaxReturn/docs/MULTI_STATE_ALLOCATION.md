# Multi-State Resident Allocation

## Overview

This feature implements income allocation for taxpayers who moved between states or worked in multiple states during a tax year. It correctly allocates income, deductions, and calculates state tax for part-year residents and non-residents working in multiple jurisdictions.

## New Entities

### state_period

Represents a period of time during the tax year when a taxpayer resided or worked in a specific state.

**Fields:**
- `id` (integer): Unique identifier for the state period
- `state_code` (string): Two-letter state code (e.g., "NY", "FL", "CA", "TX")
- `start_date` (date): Start date of residence/work in this state
- `end_date` (date): End date of residence/work in this state
- `resident_status` (string): Residency status - "full_year", "part_year", or "nonresident"
- `days_in_state` (integer, computed): Number of days in this state during this period
- `allocation_percentage` (double, computed): Percentage of income allocated to this state
- `allocated_income` (double, computed): Income allocated to this state
- `allocated_withholding` (double, computed): Withholding allocated to this state
- `notes` (array): Notes about this state period

## Modified Entities

### job
- Added `state_periods` (array of state_period): Array of periods in different states

### income
- Added `state_code` (string): State where income was earned (for multi-state allocation)
- Added `state_period_id` (integer): Reference to state_period for date-based allocation

## New Decision Tables

### TABLE 45000: Allocate_Income_By_State

**Type:** ITERATIVE
**Context:** state_period

**Purpose:** Allocates income across multiple states for taxpayers who moved between states during the tax year. Processes each state_period and allocates income based on dates and source state.

**Logic:**
1. Calculates days in each state (end_date - start_date)
2. Computes allocation percentage (days_in_state / 365)
3. Allocates income to each state based on:
   - If income.state_code matches state_period.state_code, apply allocation percentage
   - If income.state_code is null, apply allocation percentage to all income
4. Allocates withholding proportionally with income

**Example:**
```
Taxpayer moved from NY to FL on August 1
NY period: Jan 1 - July 31 = 212 days = 58.1%
FL period: Aug 1 - Dec 31 = 153 days = 41.9%
Total income: $90,000
NY allocated: $90,000 × 58.1% = $52,290
FL allocated: $90,000 × 41.9% = $37,710
```

## Modified Decision Tables

### TABLE 40000: Dispatch_State_Tax

**Changes:**
- Changed from FIRST to ITERATIVE execution
- Now processes each state_period in the job.state_periods array
- Falls back to job.state for single-state returns (backward compatible)
- Automatically calls Allocate_Income_By_State before state tax calculations
- Supports states with no income tax (TX, FL, WA, NV)

**Logic:**
1. If job.state_periods is empty, uses job.state (single-state, backward compatible)
2. If job.state_periods has entries, calls Allocate_Income_By_State first
3. Iterates through each state_period and dispatches to appropriate state tax calculation
4. For states with no income tax, logs accordingly and skips calculation

## Test Cases

### TestCase_MultiState_01_NY_FL_Move.xml
**Scenario:** Single taxpayer moved from New York to Florida mid-year

- **Period:** NY (Jan 1 - July 31, 212 days), FL (Aug 1 - Dec 31, 153 days)
- **Income:** $90,000 total
- **Allocation:** NY: $52,290 (58.1%), FL: $37,710 (41.9%)
- **Tax:** NY state tax on $52,290, FL has no state income tax

### TestCase_MultiState_02_CA_TX_Move.xml
**Scenario:** Married filing jointly, moved from California to Texas

- **Period:** CA (Jan 1 - Sept 30, 273 days), TX (Oct 1 - Dec 31, 92 days)
- **Income:** $145,000 total (two taxpayers)
- **Allocation:** CA: $108,460 (74.8%), TX: $36,540 (25.2%)
- **Tax:** CA state tax on $108,460, TX has no state income tax

### TestCase_MultiState_03_Traveling_Consultant.xml
**Scenario:** Self-employed consultant working in three states

- **Periods:**
  - NH (Jan 1 - Apr 30, 120 days, 32.9%)
  - IL (May 1 - Aug 31, 123 days, 33.7%)
  - MT (Sept 1 - Dec 31, 122 days, 33.4%)
- **Income:** $120,000 SE net profit
- **Allocation:**
  - NH: $39,480
  - IL: $40,440
  - MT: $40,080
- **Tax:** Each state calculates tax on allocated income using nonresident rules

## Usage

### Single-State Return (Backward Compatible)
No changes needed. Simply specify `job.state` as before:
```xml
<job>
  <state>NH</state>
  ...
</job>
```

### Multi-State Return
Specify state_periods and optionally mark income by state:
```xml
<job>
  <state>FL</state>  <!-- Current/final state of residence -->

  <state_periods>
    <state_period id="1">
      <state_code>NY</state_code>
      <start_date>1/1/2025</start_date>
      <end_date>7/31/2025</end_date>
      <resident_status>part_year</resident_status>
    </state_period>
    <state_period id="2">
      <state_code>FL</state_code>
      <start_date>8/1/2025</start_date>
      <end_date>12/31/2025</end_date>
      <resident_status>part_year</resident_status>
    </state_period>
  </state_periods>

  <incomes>
    <income id="1">
      <type>w2_wages</type>
      <gross_amount>52290</gross_amount>
      <state_code>NY</state_code>
      <state_period_id>1</state_period_id>
    </income>
    <income id="2">
      <type>w2_wages</type>
      <gross_amount>37710</gross_amount>
      <state_code>FL</state_code>
      <state_period_id>2</state_period_id>
    </income>
  </incomes>
</job>
```

## Resident Status Values

- **full_year**: Taxpayer was a resident of this state for the entire tax year
- **part_year**: Taxpayer moved into or out of this state during the tax year
- **nonresident**: Taxpayer worked in this state but was not a resident

## State Tax Rules

### States with No Income Tax
The following states have no income tax and are handled automatically:
- Texas (TX)
- Florida (FL)
- Washington (WA)
- Nevada (NV)
- Wyoming (WY)
- South Dakota (SD)
- Alaska (AK)
- Tennessee (TN) - Note: Has tax on interest/dividends only
- New Hampshire (NH) - Note: Has tax on interest/dividends only (but progressive rates implemented)

When a state_period references one of these states, the system logs "No state income tax for [STATE]" and no calculation is performed.

### Implemented State Tax Calculations
- Illinois (IL) - Flat 4.95% tax
- New Hampshire (NH) - Progressive brackets (3%, 5%, 7.5%)
- Montana (MT) - Progressive brackets

### Future State Implementations
Additional states can be added by:
1. Creating a Calculate_[STATE]_Tax decision table
2. Adding a condition in Dispatch_State_Tax for the new state
3. Adding an action to call the new calculation table

## Implementation Notes

### Date Calculations
- Days are calculated using the `daysbetween` operator
- Allocation percentages use 365 days as the denominator
- For leap years, this approach slightly under-allocates (consider adjusting to 366)

### Income Allocation Logic
- If income has a `state_code`, it's allocated only to matching state_periods
- If income has no `state_code`, it's allocated proportionally to all state_periods
- Withholding follows the same allocation as income

### Backward Compatibility
- Existing single-state returns work without modification
- If `job.state_periods` is empty, system uses `job.state`
- All existing test cases continue to pass

## Related Files

- **EDD:** `/sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
- **Decision Tables:** `/sampleprojects/TaxReturn/xml/TaxReturn_dt.xml`
- **Test Cases:** `/sampleprojects/TaxReturn/testfiles/TestScenarios/TestCase_MultiState_*.xml`

## Future Enhancements

1. **Reciprocal Agreements:** Some states have reciprocal agreements (e.g., IL/WI, MD/DC)
2. **Apportionment Rules:** Some states use formulary apportionment for business income
3. **Credit for Taxes Paid:** Implement credit for taxes paid to other states
4. **City/Local Taxes:** Extend to support city-level income taxes
5. **Military Rules:** Special rules for military members stationed in other states
6. **Convenience of Employer:** Handle telecommuting rules (e.g., NY convenience rule)
