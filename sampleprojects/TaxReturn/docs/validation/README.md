# State Tax Validation Test Cases

This directory contains validation test cases for the top 10 U.S. states by population as part of Issue #237.

## Purpose

Cross-validate DTRules tax calculations against:
- Online tax calculators (TurboTax, H&R Block, etc.)
- Official state tax forms and tables
- Manual calculations

## Test Case Structure

Each state has 3 validation scenarios:
- **Scenario A:** Single filer, $65,000 W-2 income
- **Scenario B:** Married filing jointly, $120,000 W-2 income
- **Scenario C:** Single filer, $200,000 W-2 income

## Test Files

### Implemented States (DTRules)

#### Illinois (IL)
- `TestCase_IL_Validation_A.xml` - Single, $65,000 → Expected tax: $3,080.14
- `TestCase_IL_Validation_B.xml` - MFJ, $120,000 → Expected tax: $5,665.28
- `TestCase_IL_Validation_C.xml` - Single, $200,000 → Expected tax: $9,762.64

**Status:** ✅ Ready to test with DTRules engine

### Not Yet Implemented (Reference Only)

The following states are documented in VALIDATION_REPORT.md with expected tax calculations:

#### California (CA) - Progressive 9 brackets (1%-13.3%)
- Scenario A: $65,000 → Expected: ~$2,915
- Scenario B: $120,000 → Expected: ~$5,850
- Scenario C: $200,000 → Expected: ~$14,250

#### Texas (TX) - No state income tax
- All scenarios: $0

#### Florida (FL) - No state income tax
- All scenarios: $0

#### New York (NY) - Progressive 9 brackets (4%-10.9%)
- Scenario A: $65,000 → Expected: ~$2,970
- Scenario B: $120,000 → Expected: ~$5,600
- Scenario C: $200,000 → Expected: ~$11,500

#### Pennsylvania (PA) - Flat 3.07%
- Scenario A: $65,000 → Expected: $1,995.50
- Scenario B: $120,000 → Expected: $3,684.00
- Scenario C: $200,000 → Expected: $6,140.00

#### Ohio (OH) - Progressive 3 brackets (0%, 2.75%, 3.125%)
- Scenario A: $65,000 → Expected: $1,071.13
- Scenario B: $120,000 → Expected: $2,658.63
- Scenario C: $200,000 → Expected: $5,158.63

#### Georgia (GA) - Flat 5.19%
- Scenario A: $65,000 → Expected: $2,953.11
- Scenario B: $120,000 → Expected: $5,579.25
- Scenario C: $200,000 → Expected: $9,959.61

#### North Carolina (NC) - Flat 4.25%
- Scenario A: $65,000 → Expected: $2,162.19
- Scenario B: $120,000 → Expected: $3,899.38
- Scenario C: $200,000 → Expected: $7,899.69

#### Michigan (MI) - Flat 4.25%
- Scenario A: $65,000 → Expected: $2,524.50
- Scenario B: $120,000 → Expected: $4,624.00
- Scenario C: $200,000 → Expected: $8,262.00

## Running Validation Tests

### For Illinois (Implemented)

1. Build the TaxReturn project:
   ```bash
   cd /tmp/DTRules-worktrees/issue-237/sampleprojects/TaxReturn
   mvn clean package
   ```

2. Run each test case through the DTRules engine and compare output

3. Document results in VALIDATION_REPORT.md

### For Other States (Not Implemented)

1. Use online tax calculators:
   - [TurboTax TaxCaster](https://turbotax.intuit.com/tax-tools/calculators/taxcaster/)
   - [H&R Block Tax Calculator](https://www.hrblock.com/tax-calculator/)
   - State-specific calculators

2. Input test scenario data

3. Compare with expected values in VALIDATION_REPORT.md

4. Capture screenshots for documentation

5. Update VALIDATION_REPORT.md with actual results

## Expected Accuracy

All calculations should match expected values within $1.

## Discrepancy Investigation

If results don't match:
1. Verify input data (AGI, deductions, exemptions)
2. Check tax rate and bracket values
3. Review state-specific rules (retirement income, etc.)
4. Consult official state tax forms
5. Document findings in VALIDATION_REPORT.md

## State Tax Resources

- [Tax Foundation - 2025 State Tax Rates](https://taxfoundation.org/data/all/state/state-income-tax-rates/)
- [California FTB](https://www.ftb.ca.gov/)
- [New York Tax Department](https://www.tax.ny.gov/)
- [Illinois Department of Revenue](https://www2.illinois.gov/rev/)
- [Ohio Department of Taxation](https://tax.ohio.gov/)
- [Pennsylvania Department of Revenue](https://www.revenue.pa.gov/)
- [Georgia Department of Revenue](https://dor.georgia.gov/)
- [North Carolina DOR](https://www.ncdor.gov/)
- [Michigan Treasury](https://www.michigan.gov/treasury)

## Implementation Status

See VALIDATION_REPORT.md for:
- Detailed test results
- Implementation recommendations
- Priority order for adding new states
- Technical specifications for each state

## Related Documentation

- `/sampleprojects/TaxReturn/docs/GAPS.md` - Known implementation gaps
- `/sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md` - Multi-state support
- `/sampleprojects/TaxReturn/xml/TaxReturn_dt.xml` - Decision tables
