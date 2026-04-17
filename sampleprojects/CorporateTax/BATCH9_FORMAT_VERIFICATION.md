# Batch 9 Corporate Tax Files - Format Update Report

## Executive Summary

**Status: NO UPDATES NEEDED**

All Batch 9 corporate tax files (SC, SD, UT, VT, WV, WY, DC) are **already in the new DTRules standard format**. No changes were required.

## States Reviewed

| State | Code | Table Range | Tax Type | Tables |
|-------|------|-------------|----------|--------|
| South Carolina | SC | 54400-54499 | 5% flat rate | 3 |
| South Dakota | SD | 54500-54599 | No corporate tax | 3 |
| Utah | UT | 54600-54699 | 4.5% flat rate | 3 |
| Vermont | VT | 54700-54799 | Graduated (6%, 7%, 8.5%) | 3 |
| West Virginia | WV | 54800-54899 | 6.5% flat rate | 3 |
| Wyoming | WY | 54900-54999 | No corporate tax | 3 |
| District of Columbia | DC | 55000-55099 | 8.25% + minimum tax | 4 |

## Files Verified

For each state, two files were verified:
- `XX_corp_edd.xml` - Entity Data Dictionary with state-specific constants
- `XX_corp_dt.xml` - Decision Tables with tax calculation logic

**Total files verified: 14 (7 states × 2 files each)**

## Format Compliance Verification

### Entity Data Dictionary Files (EDD)

All `_corp_edd.xml` files comply with the new standard format:

✅ **Root Structure:**
- Uses `<entity_dictionary name="CorporateTax_XX_Constants">` root tag
- Properly defines `apportionment` entity with state-specific configuration
- Properly defines `result` entity for state-specific extensions

✅ **Required Fields in apportionment entity:**
- `state_code` - Two-letter state code (SC, SD, UT, VT, WV, WY, DC)
- `state_tax_rate` - Corporate income tax rate (or 0.0 for no-tax states)
- `apportionment_formula` - Method used (typically 'single_sales')
- `economic_nexus_threshold` - Post-Wayfair nexus threshold
- `transaction_threshold` - Transaction count threshold
- `state_additions` - Additions to federal taxable income
- `state_subtractions` - Subtractions from federal taxable income
- `state_credits` - Available state tax credits

✅ **Comprehensive Documentation:**
- Each file includes detailed header comments
- Field definitions include descriptive comments
- References to state statutes and forms included

### Decision Table Files (DT)

All `_corp_dt.xml` files comply with the new standard format:

✅ **Root Structure:**
- Uses `<decision_tables name="CorporateTax_XX_Tables">` root tag
- Tables properly numbered within allocated ranges

✅ **Standard Table Structure:**
```xml
<decision_table name="..." number="...">
  <description>...</description>
  <conditions>
    <condition name="...">
      <expression>...</expression>
      <comment>...</comment>
    </condition>
  </conditions>
  <rules>
    <rule number="...">
      <conditions>...</conditions>
      <actions>
        <action>Java-like pseudocode</action>
      </actions>
      <policy>Detailed explanation with references</policy>
    </rule>
  </rules>
</decision_table>
```

✅ **Key Features:**
- Each table has descriptive `<description>` tag
- Conditions use named `<condition>` elements with expressions
- Rules organized in `<rules>` section with numbered `<rule>` elements
- Each rule includes comprehensive `<policy>` statement
- Actions use EL (Expression Language) syntax
- Includes references to state statutes, forms, and regulations

## Table Numbering Verification

All decision tables are correctly numbered within allocated ranges:

| State | Range | Tables Defined | Status |
|-------|-------|----------------|--------|
| SC | 54400-54499 | 54400, 54420, 54480 | ✅ Correct |
| SD | 54500-54599 | 54500, 54520, 54580 | ✅ Correct |
| UT | 54600-54699 | 54600, 54620, 54680 | ✅ Correct |
| VT | 54700-54799 | 54700, 54720, 54780 | ✅ Correct |
| WV | 54800-54899 | 54800, 54820, 54880 | ✅ Correct |
| WY | 54900-54999 | 54900, 54920, 54980 | ✅ Correct |
| DC | 55000-55099 | 55000, 55020, 55080, 55090 | ✅ Correct |

## Standard Decision Table Pattern

Each state follows the standard 3-table pattern:

1. **XX000: Determine Filing Requirement**
   - Checks for physical presence nexus
   - Checks for economic nexus (post-Wayfair)
   - Determines if state tax return filing required

2. **XX020: Calculate Income Adjustments**
   - Calculates state-specific additions to federal taxable income
   - Calculates state-specific subtractions from federal taxable income
   - Applies state conformity rules

3. **XX080: Calculate State Tax**
   - Applies state tax rate (flat or graduated)
   - Subtracts state credits
   - Calculates final tax liability and refund/owed amount

**Note:** DC has additional table 55090 for minimum tax calculation.

## Logic Preservation

All original tax calculation logic has been preserved:

✅ **South Carolina (SC):**
- 5% flat rate correctly implemented
- License fee calculation preserved
- Single-sales apportionment maintained

✅ **South Dakota (SD):**
- No corporate income tax status correctly implemented
- Bank franchise tax and insurance premium tax distinctions maintained
- Wayfair case reference preserved

✅ **Utah (UT):**
- 4.5% flat rate (reduced from 4.55% in 2025) correctly implemented
- Transaction threshold removal (S.B. 47, 2025) reflected
- Single-sales apportionment maintained

✅ **Vermont (VT):**
- Graduated brackets (6%, 7%, 8.5%) correctly implemented
- Bracket thresholds ($10,000, $25,000) preserved
- Progressive rate calculation logic maintained

✅ **West Virginia (WV):**
- 6.5% flat rate correctly implemented
- Single-sales apportionment (effective 2022+) maintained
- Market-based sourcing references preserved

✅ **Wyoming (WY):**
- No corporate income tax status correctly implemented
- Annual report fee distinction (domestic $60, foreign $120) preserved
- No franchise/gross receipts tax status maintained

✅ **District of Columbia (DC):**
- 8.25% franchise tax rate correctly implemented
- Minimum tax calculation ($250 or $1,000 based on receipts) preserved
- Single-sales apportionment (effective 2015) maintained
- Greater-of calculation (income tax vs. minimum tax) logic preserved

## XML Validation

All files are well-formed XML:
- Root tags properly balanced (`<entity_dictionary>...</entity_dictionary>`)
- Root tags properly balanced (`<decision_tables>...</decision_tables>`)
- All nested elements properly closed
- No syntax errors detected

## Conclusion

**All Batch 9 corporate tax files (SC, SD, UT, VT, WV, WY, DC) are already in full compliance with the new DTRules standard format.**

No updates, changes, or modifications were needed. All files:
- Use the correct structural format
- Have proper table numbering
- Include comprehensive documentation
- Preserve all original tax calculation logic
- Are well-formed and valid XML

The files are ready for use and require no further action.

---
**Verification Date:** 2026-03-24
**Files Verified:** 14 files (7 states × 2 files)
**Status:** ✅ COMPLIANT - No changes needed
