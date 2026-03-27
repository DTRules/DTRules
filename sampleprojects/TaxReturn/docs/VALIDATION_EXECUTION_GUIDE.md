# State Tax Validation Execution Guide

## Overview

This guide provides step-by-step instructions for executing the validation tests for the top 10 US states by population. Each test case should be run through DTRules and validated against external tax calculators.

## Test Case Inventory

### Created Test Cases (30 total)

| State | Test 1 | Test 2 | Test 3 |
|-------|--------|--------|--------|
| CA | TestCase_CA_01_Single_W2.xml | TestCase_CA_02_MFJ_Dual_Income.xml | TestCase_CA_03_High_SE_Income.xml |
| TX | TestCase_TX_01_Single_W2.xml | TestCase_TX_02_MFJ_Dual_Income.xml | TestCase_TX_03_High_SE_Income.xml |
| FL | TestCase_FL_01_Single_W2.xml | TestCase_FL_02_MFJ_Dual_Income.xml | TestCase_FL_03_High_SE_Income.xml |
| NY | TestCase_NY_01_Single_W2.xml | TestCase_NY_02_MFJ_Dual_Income.xml | TestCase_NY_03_High_SE_Income.xml |
| PA | TestCase_PA_01_Single_W2.xml | TestCase_PA_02_MFJ_Dual_Income.xml | TestCase_PA_03_High_SE_Income.xml |
| IL | TestCase_IL_01_Single_W2.xml | TestCase_IL_02_MFJ_Dual_Income.xml | TestCase_IL_03_High_SE_Income.xml |
| OH | TestCase_OH_01_Single_W2.xml | TestCase_OH_02_MFJ_Dual_Income.xml | TestCase_OH_03_High_SE_Income.xml |
| GA | TestCase_GA_01_Single_W2.xml | TestCase_GA_02_MFJ_Dual_Income.xml | TestCase_GA_03_High_SE_Income.xml |
| NC | TestCase_NC_01_Single_W2.xml | TestCase_NC_02_MFJ_Dual_Income.xml | TestCase_NC_03_High_SE_Income.xml |
| MI | TestCase_MI_01_Single_W2.xml | TestCase_MI_02_MFJ_Dual_Income.xml | TestCase_MI_03_High_SE_Income.xml |

## Prerequisites

### Software Requirements
1. Java 8 or higher
2. Maven 3.x
3. Access to online tax calculators (internet connection)

### Build the Project

```bash
cd /tmp/DTRules-worktrees/issue-237
mvn clean install -DskipTests > /tmp/build.log 2>&1
```

### Compile TaxReturn Decision Tables

```bash
cd sampleprojects/TaxReturn
mvn exec:java -Dexec.mainClass="com.dtrules.samples.taxreturn.CompileTaxReturn" \
  > /tmp/compile_tax.log 2>&1
```

## Validation Process

### Step 1: Run DTRules Calculation

For each test case:

```bash
cd sampleprojects/TaxReturn

# Example for Texas Single Filer
mvn exec:java \
  -Dexec.mainClass="com.dtrules.samples.taxreturn.TestTaxReturn" \
  -Dexec.args="testfiles/TestScenarios/TestCase_TX_01_Single_W2.xml" \
  > /tmp/test_tx_01.log 2>&1

# Review output
tail -100 /tmp/test_tx_01.log
```

### Step 2: Extract DTRules Results

Look for these values in the output:
- `total_income` - Gross income
- `agi` - Adjusted Gross Income
- `taxable_income` - Taxable income
- `total_tax` - Total federal tax
- `state_tax` - State income tax (if applicable)
- `amount_owed` or `refund_amount`

### Step 3: Validate Against External Calculators

#### Online Tax Calculators

**SmartAsset Tax Calculator**
- URL: https://smartasset.com/taxes/income-taxes
- Enter: State, filing status, income, deductions
- Compare: Federal tax, state tax, total tax

**State-Specific Calculators**
- California: https://www.ftb.ca.gov/tax-calculator/
- New York: https://www.tax.ny.gov/pit/file/tax-calculator.htm
- Pennsylvania: https://www.revenue.pa.gov/
- Illinois: https://www2.illinois.gov/rev/
- Others: Search "[State] income tax calculator"

#### Manual Calculation Verification

For flat-rate states, validation is straightforward:

**Texas/Florida:** No income tax, only federal
**Pennsylvania:** AGI × 3.07%
**Illinois:** (AGI - exemptions) × 4.95%
**North Carolina:** (AGI - std deduction) × 4.5%
**Michigan:** (AGI - exemptions) × 4.25%

### Step 4: Document Results

Record in validation spreadsheet or tracking document:

| Test Case | DTRules Federal | Calculator Federal | Δ | DTRules State | Calculator State | Δ | Status |
|-----------|----------------|-------------------|---|---------------|-----------------|---|--------|
| TX_01 | $9,104 | $9,100 | $4 | $0 | $0 | $0 | ✅ Pass |
| ... | ... | ... | ... | ... | ... | ... | ... |

**Acceptance Criteria:**
- ✅ Pass: Difference ≤ $10 total tax
- ⚠️ Review: Difference $11-50 (investigate)
- ❌ Fail: Difference > $50 (requires fix)

## Test Execution Priority

### Phase 1: No-Tax States (Easiest)
1. TX - Texas (3 test cases)
2. FL - Florida (3 test cases)

**Validation:** Only federal tax, easy to verify

### Phase 2: Implemented State (Validation Only)
3. IL - Illinois (3 test cases)

**Validation:** Test existing implementation

### Phase 3: Flat-Rate States (Next Priority)
4. PA - Pennsylvania 3.07% (3 test cases)
5. NC - North Carolina 4.5% (3 test cases)
6. MI - Michigan 4.25% (3 test cases)

**Validation:** Simple calculation, easy to verify manually

### Phase 4: Progressive States (Medium Complexity)
7. OH - Ohio (3 test cases)
8. GA - Georgia (3 test cases)

**Validation:** Multiple brackets, requires bracket lookup

### Phase 5: Complex Progressive States
9. NY - New York (3 test cases)
10. CA - California (3 test cases)

**Validation:** Many brackets, highest complexity

## Detailed Test Scenarios

### Scenario 1: Single W2 Employee ($75k)

**Test Profile:**
- Filing Status: Single
- Income: $75,000 W-2 wages
- Withholding: $9,000
- Deductions: Standard ($14,800 federal)

**Expected Federal Results:**
- AGI: $75,000
- Standard Deduction: $14,800
- Taxable Income: $60,200
- Tax Calculation:
  - 10% on first $11,600 = $1,160
  - 12% on next $35,550 ($11,600 to $47,150) = $4,266
  - 22% on remaining $13,050 ($47,150 to $60,200) = $2,871
  - **Total Federal Tax: $8,297**
- Refund/Owed: $9,000 withholding - $8,297 tax = **$703 refund**

**State Tax by State:**
- TX, FL: $0
- PA: $75,000 × 3.07% = $2,303
- IL: ($75,000 - $2,650) × 4.95% = $3,581
- NC: ($75,000 - $12,750) × 4.5% = $2,801
- MI: ($75,000 - $5,400) × 4.25% = $2,958
- OH: ~$1,500 (progressive brackets)
- GA: ~$4,100 (progressive brackets)
- NY: ~$4,115 (progressive brackets)
- CA: ~$3,908 (progressive brackets)

### Scenario 2: MFJ Dual Income ($155k, 2 children)

**Test Profile:**
- Filing Status: Married Filing Jointly
- Spouse 1: $90,000 W-2, $14,000 withholding
- Spouse 2: $65,000 W-2, $8,000 withholding
- Total Income: $155,000
- Dependents: 2 children (CTC eligible)

**Expected Federal Results:**
- AGI: $155,000
- Standard Deduction: $30,000
- Taxable Income: $125,000
- Tax Calculation:
  - 10% on first $23,200 = $2,320
  - 12% on next $71,100 ($23,200 to $94,300) = $8,532
  - 22% on remaining $30,700 ($94,300 to $125,000) = $6,754
  - **Subtotal: $17,606**
- Child Tax Credit: -$4,000 (2 × $2,000)
- **Net Federal Tax: $13,606**
- Total Withholding: $22,000
- **Refund: $8,394**

**State Tax by State:**
- TX, FL: $0
- PA: $155,000 × 3.07% = $4,759
- IL: ($155,000 - $5,300) × 4.95% = $7,410
- NC: ($155,000 - $25,500) × 4.5% = $5,828
- MI: ($155,000 - $10,800) × 4.25% = $6,129
- OH: ~$3,700 (progressive)
- GA: ~$8,600 (progressive)
- NY: ~$9,200 (progressive)
- CA: ~$9,500 (progressive)

### Scenario 3: High SE Income ($215k net)

**Test Profile:**
- Filing Status: Single
- Self-Employment: $250,000 gross - $35,000 expenses = $215,000 net
- Estimated Payments: $50,000
- Itemized Deductions: $28,000 (SALT $10k + mortgage $15k + charity $3k)

**Expected Federal Results:**
- SE Income: $215,000
- SE Tax: $215,000 × 92.35% × 15.3% = $30,374
- SE Tax Deduction: $30,374 / 2 = $15,187
- AGI: $215,000 - $15,187 = $199,813
- Itemized Deduction: $28,000
- QBI Deduction: ~$34,323 (20% of qualified income)
- Taxable Income: ~$137,490
- Regular Tax: ~$26,000
- SE Tax: $30,374
- **Total Federal Tax: ~$56,374**
- Estimated Payments: $50,000
- **Amount Owed: ~$6,374**

**State Tax by State:**
- TX, FL: $0
- PA: $215,000 × 3.07% = $6,601
- IL: ($215,000 - $2,650) × 4.95% = $10,511
- NC: ($215,000 - $12,750) × 4.5% = $9,101
- MI: ($215,000 - $5,400) × 4.25% = $8,908
- OH: ~$7,300 (progressive)
- GA: ~$11,800 (progressive)
- NY: ~$15,500 (progressive)
- CA: ~$18,000 (progressive)

## Common Issues and Troubleshooting

### Issue 1: Test Won't Run

**Symptom:** Maven execution fails
**Solution:**
```bash
# Ensure project is built
cd /tmp/DTRules-worktrees/issue-237
mvn clean install

# Ensure decision tables are compiled
cd sampleprojects/TaxReturn
mvn exec:java -Dexec.mainClass="com.dtrules.samples.taxreturn.CompileTaxReturn"
```

### Issue 2: State Tax Not Implemented

**Symptom:** No state tax calculated, or error about missing table
**Expected:** States CA, NY, PA, OH, GA, NC, MI not yet implemented
**Solution:**
- Document as "State implementation needed"
- Validate federal tax only
- Mark state tax as "N/A - Not Implemented"

### Issue 3: Results Don't Match

**Symptom:** DTRules tax differs from calculator by >$50

**Investigation Steps:**
1. Check audit trail in DTRules output
2. Verify all income sources entered correctly
3. Check deduction amounts
4. Verify filing status matches
5. Check calculator year matches (2025)
6. Try different calculator for second opinion

**Common Causes:**
- Wrong standard deduction year
- Missing or incorrect exemption amounts
- Rounding differences
- State-specific rules not implemented

### Issue 4: Missing Output Values

**Symptom:** Can't find AGI or taxable income in output

**Solution:**
```bash
# Look in the trace output
grep -E "agi|taxable_income|total_tax" /tmp/test_output.log

# Check the results entity
grep -A 20 "result entity" /tmp/test_output.log
```

## Validation Tracking

### Create Validation Spreadsheet

Track results in a spreadsheet with columns:
1. Test Case ID
2. State
3. Scenario
4. DTRules Federal Tax
5. Calculator Federal Tax
6. Federal Δ
7. DTRules State Tax
8. Calculator State Tax
9. State Δ
10. Total Δ
11. Status (Pass/Fail/Review)
12. Notes
13. Validator Name
14. Date Validated
15. Screenshot Link

### Screenshots

For audit purposes, take screenshots of:
1. Online calculator results
2. DTRules output (key values)
3. Any discrepancy investigations

Store in: `sampleprojects/TaxReturn/testfiles/ValidationScreenshots/`

## Validation Report Template

After completing all tests, create summary report:

```markdown
# State Tax Validation Results

## Summary
- Total Tests: 30
- Passed: X
- Failed: Y
- Needs Implementation: Z

## By State
### Texas (3 tests)
- Status: ✅ All Passed
- Max Discrepancy: $4
- Notes: No state tax, federal only

### California (3 tests)
- Status: ❌ Failed - Not Implemented
- Notes: State tax calculation not available

[Continue for all states...]

## Issues Found
1. [Description of any issues]
2. [Discrepancies that need investigation]

## Recommendations
1. [Priority implementation items]
2. [Suggested improvements]
```

## Next Steps

After validation:

1. **Document Results** - Complete validation report
2. **Create Issues** - File tickets for failed tests
3. **Prioritize Implementation** - Based on validation findings
4. **Update Documentation** - Record any clarifications needed
5. **Share Findings** - Present results to team

## Contact

For questions about validation process:
- Review STATE_TAX_VALIDATION.md
- Check existing test cases in testfiles/TestScenarios/
- Refer to TaxReturn_dt.xml for decision table logic

---

**Last Updated:** 2025-03-22
**Status:** Ready for validation execution
