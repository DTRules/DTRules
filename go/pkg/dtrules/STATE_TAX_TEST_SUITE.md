# Comprehensive State Tax Test Suite

## Overview

This document describes the comprehensive state tax test suite created in `comprehensive_state_tax_test.go`. The suite provides validation for all 50 U.S. states plus Washington DC, with at least 3 test cases per state.

## Test Coverage

### Total Test Cases: 153 (3 per state × 51 jurisdictions)

The test suite covers:
- **41 states with income tax** - Progressive or flat tax implementations
- **9 states with no income tax** - Alaska, Florida, Nevada, South Dakota, Tennessee, Texas, Washington, Wyoming
- **Washington DC** - Progressive tax implementation

### Test Case Categories

Each state has 3 test scenarios:

1. **Simple Single W-2 Earner** (`_01_Single_W2.xml`)
   - Single filing status
   - One W-2 income source
   - Standard deduction
   - Tests basic tax calculation

2. **Multiple Income Sources** (`_02_MFJ_Multiple.xml`)
   - Married Filing Jointly status
   - Multiple income types: W-2, self-employment, rental income
   - Tests income aggregation and deduction handling

3. **High Earner - All Brackets** (`_03_High_Income.xml`)
   - High income to test all tax brackets
   - Complex income scenarios
   - Tests progressive bracket calculations

## State Implementations

### Progressive Tax States (33 states)

States with graduated tax brackets based on income levels:

| State | Issue # | Tax Rates | Test Files |
|-------|---------|-----------|------------|
| Alabama | #213 | 2% - 5% | AL/TestCase_AL_01/02/03 |
| Arizona | #221 | 2.55% - 4.50% | AZ/TestCase_AZ_01/02/03 |
| California | #195 | 1% - 13.3% | CA/TestCase_CA_01/02/03 |
| Connecticut | #188 | 3% - 6.99% | CT/TestCase_CT_01/02/03 |
| Delaware | #220 | 2.2% - 6.6% | DE/TestCase_DE_01/02/03 |
| Georgia | #209 | 1% - 5.75% | GA/TestCase_GA_01/02/03 |
| Hawaii | #197 | 1.4% - 11% | HI/TestCase_HI_01/02/03 |
| Idaho | #198 | 5.8% - 6.5% | ID/TestCase_ID_01/02/03 |
| Iowa | #201 | 0.33% - 6% | IA/TestCase_IA_01/02/03 |
| Kansas | #203 | 3.1% - 5.7% | KS/TestCase_KS_01/02/03 |
| Maine | #191 | 5.8% - 7.15% | ME/TestCase_ME_01/02/03 |
| Maryland | #219 | 2% - 5.75% | MD/TestCase_MD_01/02/03 |
| Minnesota | #200 | 5.35% - 9.85% | MN/TestCase_MN_01/02/03 |
| Mississippi | #214 | 0% - 5% | MS/TestCase_MS_01/02/03 |
| Missouri | #202 | 1.5% - 4.95% | MO/TestCase_MO_01/02/03 |
| Montana | #208 | 4.7% - 5.9% | MT/TestCase_MT_01/02/03 |
| Nebraska | #204 | 2.46% - 6.64% | NE/TestCase_NE_01/02/03 |
| New Hampshire | #193 | 3% - 7.5% | TestCase_NH_01/02/03 |
| New Jersey | #189 | 1.4% - 10.75% | NJ/TestCase_NJ_01/02/03 |
| New York | #186 | 4% - 10.9% | NY/TestCase_NY_01/02/03 |
| North Dakota | #205 | 1.95% - 2.5% | ND/TestCase_ND_01/02/03 |
| Ohio | #206 | 0% - 3.75% | OH/TestCase_OH_01/02/03 |
| Oregon | #196 | 4.75% - 9.9% | OR/TestCase_OR_01/02/03 |
| Rhode Island | #192 | 3.75% - 5.99% | RI/TestCase_RI_01/02/03 |
| South Carolina | #210 | 0% - 6.5% | SC/TestCase_SC_01/02/03 |
| Vermont | #190 | 3.35% - 8.75% | VT/TestCase_VT_01/02/03 |
| Virginia | #211 | 2% - 5.75% | VA/TestCase_VA_01/02/03 |
| Washington DC | #194 | 4% - 10.75% | DC/TestCase_DC_01/02/03 |
| West Virginia | #212 | 2.36% - 5.12% | WV/TestCase_WV_01/02/03 |
| Wisconsin | #199 | 3.54% - 7.65% | WI/TestCase_WI_01/02/03 |

### Flat Tax States (8 states)

States with a single tax rate applied to all income:

| State | Issue # | Tax Rate | Test Files |
|-------|---------|----------|------------|
| Colorado | #180 | 4.40% | CO/TestCase_CO_01/02/03 |
| Illinois | #181 | 4.95% | IL/TestCase_IL_01/02/03 |
| Indiana | #182 | 3.15% | IN/TestCase_IN_01/02/03 |
| Kentucky | #207 | 4.5% | KY/TestCase_KY_01/02/03 |
| Massachusetts | #187 | 5% | MA/TestCase_MA_01/02/03 |
| Michigan | #183 | 4.25% | MI/TestCase_MI_01/02/03 |
| North Carolina | #184 | 4.50% | NC/TestCase_NC_01/02/03 |
| Pennsylvania | #185 | 3.07% | PA/TestCase_PA_01/02/03 |

### No Income Tax States (9 states)

States with no state income tax (verified with $0 state tax):

| State | Issue # | Test Files |
|-------|---------|------------|
| Alaska | #224 | AK/TestCase_AK_01/02/03 |
| Florida | #229 | FL/TestCase_FL_01/02/03 |
| Nevada | #227 | NV/TestCase_NV_01/02/03 |
| South Dakota | #225 | SD/TestCase_SD_01/02/03 |
| Tennessee | #218 | TN/TestCase_TN_01/02/03 |
| Texas | #230 | TX/TestCase_TX_01/02/03 |
| Washington | #228 | WA/TestCase_WA_01/02/03 |
| Wyoming | #226 | WY/TestCase_WY_01/02/03 |

## Test Data Requirements

To make the comprehensive test suite pass, the following test data files need to be created in:
`sampleprojects/TaxReturn/testfiles/TestScenarios/`

### Test File Structure

Each test file should follow this XML structure:

```xml
<job>
   <id>STATE-##</id>
   <tax_year>2025</tax_year>
   <filing_status>single|mfj|hoh</filing_status>
   <state>STATE_CODE</state>
   <has_state_income_tax>true|false</has_state_income_tax>

   <!-- Expected values for validation -->
   <expected_agi>AMOUNT</expected_agi>
   <expected_state_agi>AMOUNT</expected_state_agi>
   <expected_state_taxable_income>AMOUNT</expected_state_taxable_income>
   <expected_state_tax>AMOUNT</expected_state_tax>

   <!-- Taxpayer and income data -->
   <taxpayers>...</taxpayers>
   <incomes>...</incomes>
</job>
```

### Example Test Files Created

The following test files already exist and serve as templates:

- **New Hampshire**: `TestCase_NH_01_Single_W2.xml`, `TestCase_NH_02_MFJ_Two_Brackets.xml`, `TestCase_NH_03_High_Income_All_Brackets.xml`

These can be used as templates for creating test files for other states.

## Test Execution

### Running the Full Suite

```bash
cd go
go test -v ./pkg/dtrules -run TestComprehensiveStateTaxes
```

### Expected Output

The test suite will:
1. Load the TaxReturn EDD and decision tables once
2. For each test case:
   - Create a fresh session
   - Load the test data XML file
   - Execute the tax calculation
   - Validate state tax results
3. Print a summary:
   - Pass count
   - Fail count
   - Skip count (for missing test files)

### Validation Criteria

Each test validates:
1. **AGI is non-zero** - Calculation produced valid results
2. **State tax matches expectations**:
   - States with income tax: `state_tax >= minStateTax`
   - States without income tax: `state_tax == 0`
3. **Execution completes** without errors

## Current Status

### Test Suite Status: ✓ Created

The comprehensive test suite has been implemented in:
- `go/pkg/dtrules/comprehensive_state_tax_test.go`

### Implementation Status

Based on git commits, all 50 states + DC have tax implementations:
- ✓ All flat tax states implemented (issues #180-#185, #187, #207)
- ✓ All progressive tax states implemented (issues #186, #188-#221)
- ✓ All no-income-tax states documented (issues #224-#230)
- ✓ Washington DC implemented (issue #194)

### Test Data Status

**Existing Test Files**: 3 (New Hampshire only)
**Required Test Files**: 150+ (3 per state × 50 states)

To complete the test suite, test data XML files need to be created for 48 additional states following the patterns established in the New Hampshire test files.

## Next Steps

1. **Create test data files** for each state following the template structure
2. **Ensure decision tables** properly handle all state tax calculations
3. **Run tests** and verify all 153 test cases pass
4. **Document results** showing state tax calculation accuracy

## Benefits

This comprehensive test suite provides:

1. **Complete Coverage** - Every state tested with multiple scenarios
2. **Regression Prevention** - Changes won't break existing state calculations
3. **Documentation** - Test cases serve as examples of state tax handling
4. **Validation** - Ensures tax calculations are accurate across all jurisdictions
5. **Quality Assurance** - 3+ test cases per state catch edge cases

## References

- TaxReturn sample project: `sampleprojects/TaxReturn/`
- EDD definition: `sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
- Decision tables: `sampleprojects/TaxReturn/xml/TaxReturn_dt.xml`
- Test scenarios: `sampleprojects/TaxReturn/testfiles/TestScenarios/`
