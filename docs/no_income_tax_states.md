# No Income Tax States

## Overview

Nine U.S. states do not impose a state income tax on wages and salaries. This document describes how the DTRules TaxReturn system handles these states.

## The Nine States

The following states have no state income tax:

1. **Alaska (AK)** - No state income tax
2. **Florida (FL)** - No state income tax
3. **Nevada (NV)** - No state income tax
4. **New Hampshire (NH)** - No state income tax on wages (Note: NH does tax interest and dividend income above certain thresholds, but this is not implemented in the current system)
5. **South Dakota (SD)** - No state income tax
6. **Tennessee (TN)** - No state income tax
7. **Texas (TX)** - No state income tax
8. **Washington (WA)** - No state income tax
9. **Wyoming (WY)** - No state income tax

## Implementation

### Entity Definition

The no-income-tax states are defined in the `TaxReturn_edd.xml` file as a global array:

```xml
<field name='no_income_tax_states' type='array' subtype='' access='r' input=''
       default_value='{ "TX" "FL" "WA" "NV" "SD" "WY" "AK" "TN" "NH" }'
       comment='States with no income tax'/>
```

### State Tax Determination

The system determines whether a state has income tax during the initial actions phase of the TaxReturn decision table:

**Location:** `TaxReturn_dt.xml`, line 22

```
no_income_tax_states job.state memberof not /job.has_state_income_tax xdef
```

This expression:
1. Checks if `job.state` is a member of the `no_income_tax_states` array
2. Applies logical NOT to invert the result
3. Stores the boolean result in `job.has_state_income_tax`

**Result:**
- If the state is in the no-income-tax list: `job.has_state_income_tax = false`
- If the state has income tax: `job.has_state_income_tax = true`

### State Tax Calculation Dispatch

The `Dispatch_State_Tax` decision table (TABLE 40000) only executes state-specific tax calculations when `job.has_state_income_tax` is `true`. This effectively returns zero tax liability for all no-income-tax states without requiring explicit handling for each state.

**Location:** `TaxReturn_dt.xml`, lines 8689-8799

The dispatch table uses conditions like:
- `job.has_state_income_tax is true AND job.state is "IL"` → Calculate_IL_Tax
- `job.has_state_income_tax is true AND job.state is "NH"` → Calculate_NH_Tax
- `job.has_state_income_tax is true AND job.state is "MT"` → Calculate_MT_Tax

If `job.has_state_income_tax` is `false`, no state tax calculation tables execute, resulting in zero state tax liability.

## Audit Trail

When processing a tax return for a no-income-tax state, the audit trail will show:

1. Initial setup logging the tax year and filing status
2. Federal tax calculations
3. No state tax calculation entries (because `has_state_income_tax = false` prevents dispatch)
4. Final results with zero state tax liability

## Testing

Test cases for no-income-tax states should verify:

1. **Correct state identification:** Each of the 9 states is properly recognized as having no income tax
2. **Zero tax calculation:** State tax liability is zero for all income levels
3. **No state tax tables executed:** Audit trail confirms no state-specific calculation tables ran
4. **Flag setting:** `job.has_state_income_tax` is correctly set to `false`

Example test scenarios:
- Texas resident with $50,000 income → $0 state tax
- Florida resident with $150,000 income → $0 state tax
- Alaska resident with $200,000 income → $0 state tax
- Washington resident with $75,000 income → $0 state tax

## Special Considerations

### New Hampshire

While NH is listed as a no-income-tax state for wages, it historically taxed interest and dividend income (the "Interest and Dividends Tax"). As of January 1, 2025, New Hampshire fully repealed its interest and dividends tax, making it a true no-income-tax state.

For the purposes of this wage-based tax calculation system, NH is treated as having no state income tax.

### Future Expansions

If states change their tax policies (e.g., implementing new income taxes or eliminating existing ones), updates required:

1. **Adding a state to no-tax list:**
   - Update `no_income_tax_states` array in `TaxReturn_edd.xml`
   - Remove any existing state-specific calculation tables
   - Remove state dispatch conditions from `Dispatch_State_Tax`

2. **Removing a state from no-tax list:**
   - Remove state code from `no_income_tax_states` array
   - Create new state-specific calculation table
   - Add dispatch condition to `Dispatch_State_Tax`
   - Add state tax parameters to globals section of EDD
   - Add state-specific result fields to `result` entity

## References

- Entity Definitions: `sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
- Decision Tables: `sampleprojects/TaxReturn/xml/TaxReturn_dt.xml`
- State dispatch logic: TABLE 40000 (Dispatch_State_Tax)
- Initial actions: TaxReturn_dt.xml, line 22
