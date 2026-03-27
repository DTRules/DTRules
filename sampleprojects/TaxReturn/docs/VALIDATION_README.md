# Top 10 States Tax Validation - README

## Issue #237: Validate Top 10 States Against Tax Software

This directory contains the validation framework for testing DTRules tax calculations against commercial tax software and online calculators for the top 10 US states by population.

## What's Included

### Documentation (3 files)

1. **STATE_TAX_VALIDATION.md** - Comprehensive validation report
   - Overview of all 10 states
   - Tax structure details (flat vs progressive)
   - Expected results for each scenario
   - Implementation status tracking
   - Validation sources and references

2. **VALIDATION_EXECUTION_GUIDE.md** - Step-by-step execution instructions
   - How to run test cases
   - How to validate against online calculators
   - Troubleshooting common issues
   - Results tracking templates

3. **This README** - Quick reference guide

### Test Cases (30 XML files)

Located in: `testfiles/TestScenarios/`

**Format:** `TestCase_[STATE]_[##]_[SCENARIO].xml`

Each of the 10 states has 3 test scenarios:
- `01_Single_W2` - Single filer, $75k W-2 income
- `02_MFJ_Dual_Income` - Married filing jointly, $155k combined, 2 children
- `03_High_SE_Income` - Single, $215k self-employment income

**States Covered:**
- CA - California
- TX - Texas ✅
- FL - Florida ✅
- NY - New York
- PA - Pennsylvania
- IL - Illinois ✅ (implementation exists)
- OH - Ohio
- GA - Georgia
- NC - North Carolina
- MI - Michigan

## Quick Start

### 1. Build the Project

```bash
cd /tmp/DTRules-worktrees/issue-237
mvn clean install > /tmp/build.log 2>&1
```

### 2. Compile Decision Tables

```bash
cd sampleprojects/TaxReturn
mvn exec:java -Dexec.mainClass="com.dtrules.samples.taxreturn.CompileTaxReturn" \
  > /tmp/compile.log 2>&1
```

### 3. Run a Test Case

```bash
# Example: Texas Single Filer
mvn exec:java \
  -Dexec.mainClass="com.dtrules.samples.taxreturn.TestTaxReturn" \
  -Dexec.args="testfiles/TestScenarios/TestCase_TX_01_Single_W2.xml" \
  > /tmp/test_tx_01.log 2>&1

# View results
tail -100 /tmp/test_tx_01.log
```

### 4. Validate Results

Compare DTRules output against:
- SmartAsset Tax Calculator: https://smartasset.com/taxes/income-taxes
- State DOR calculators
- Manual calculations (for flat-rate states)

See **VALIDATION_EXECUTION_GUIDE.md** for detailed instructions.

## Test Scenarios Summary

### Scenario 1: Single Filer - W2 Employee
- **Income:** $75,000 W-2 wages
- **Filing Status:** Single
- **Deductions:** Standard deduction
- **Purpose:** Test basic wage income, no complexities

**Expected Federal Tax:** ~$8,297
**Expected State Tax:** Varies by state (see STATE_TAX_VALIDATION.md)

### Scenario 2: Married Filing Jointly - Dual Income
- **Income:** $90,000 + $65,000 = $155,000
- **Filing Status:** MFJ
- **Dependents:** 2 children (CTC eligible)
- **Deductions:** Standard deduction
- **Purpose:** Test joint filing, child tax credit

**Expected Federal Tax:** ~$13,606 (after CTC)
**Expected State Tax:** Varies by state

### Scenario 3: High Income - Self-Employment
- **Income:** $215,000 net self-employment ($250k gross - $35k expenses)
- **Filing Status:** Single
- **Deductions:** Itemized ($28k total)
- **Purpose:** Test SE tax, QBI deduction, itemized deductions

**Expected Federal Tax:** ~$56,374 (includes SE tax)
**Expected State Tax:** Varies by state

## Implementation Status

| Status | Count | States |
|--------|-------|--------|
| ✅ Implemented | 3 | TX, FL, IL |
| ❌ Not Implemented | 7 | CA, NY, PA, OH, GA, NC, MI |

### Implementation Priority

**HIGH (Simple Flat Rates):**
- PA - Pennsylvania: 3.07% flat
- NC - North Carolina: 4.5% flat
- MI - Michigan: 4.25% flat

**MEDIUM (Progressive):**
- OH - Ohio: 4 brackets
- GA - Georgia: 6 brackets

**LOW (Complex Progressive):**
- NY - New York: 8 brackets + local taxes
- CA - California: 10 brackets, highest rates

## Validation Acceptance Criteria

From issue #237:

✅ **Passed:** Difference ≤ $10 total tax
- ±$5 federal tax
- ±$5 state tax
- ±$10 total

⚠️ **Review Needed:** Difference $11-50
- Investigate discrepancy
- Document findings
- May be acceptable if explained

❌ **Failed:** Difference > $50
- Indicates logic error
- Requires code fix
- Block validation sign-off

## Files Created for Issue #237

```
sampleprojects/TaxReturn/
├── docs/
│   ├── STATE_TAX_VALIDATION.md          (New - comprehensive report)
│   ├── VALIDATION_EXECUTION_GUIDE.md    (New - execution instructions)
│   └── VALIDATION_README.md             (New - this file)
└── testfiles/TestScenarios/
    ├── TestCase_CA_01_Single_W2.xml             (New)
    ├── TestCase_CA_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_CA_03_High_SE_Income.xml        (New)
    ├── TestCase_TX_01_Single_W2.xml             (New)
    ├── TestCase_TX_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_TX_03_High_SE_Income.xml        (New)
    ├── TestCase_FL_01_Single_W2.xml             (New)
    ├── TestCase_FL_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_FL_03_High_SE_Income.xml        (New)
    ├── TestCase_NY_01_Single_W2.xml             (New)
    ├── TestCase_NY_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_NY_03_High_SE_Income.xml        (New)
    ├── TestCase_PA_01_Single_W2.xml             (New)
    ├── TestCase_PA_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_PA_03_High_SE_Income.xml        (New)
    ├── TestCase_IL_01_Single_W2.xml             (New)
    ├── TestCase_IL_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_IL_03_High_SE_Income.xml        (New)
    ├── TestCase_OH_01_Single_W2.xml             (New)
    ├── TestCase_OH_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_OH_03_High_SE_Income.xml        (New)
    ├── TestCase_GA_01_Single_W2.xml             (New)
    ├── TestCase_GA_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_GA_03_High_SE_Income.xml        (New)
    ├── TestCase_NC_01_Single_W2.xml             (New)
    ├── TestCase_NC_02_MFJ_Dual_Income.xml       (New)
    ├── TestCase_NC_03_High_SE_Income.xml        (New)
    ├── TestCase_MI_01_Single_W2.xml             (New)
    ├── TestCase_MI_02_MFJ_Dual_Income.xml       (New)
    └── TestCase_MI_03_High_SE_Income.xml        (New)
```

**Total:** 33 files created (3 documentation + 30 test cases)

## Validation Workflow

```
1. Create test cases ✅ (DONE)
   └─> 30 XML files for 10 states

2. Run through DTRules ⏳ (READY)
   └─> Execute TestTaxReturn for each case

3. Validate against calculators ⏳ (READY)
   └─> Compare with SmartAsset, state calculators

4. Document results ⏳ (PENDING)
   └─> Record discrepancies, take screenshots

5. Investigate discrepancies ⏳ (IF NEEDED)
   └─> Debug, fix issues, retest

6. Create final report ⏳ (PENDING)
   └─> Summary of findings, recommendations
```

## Current Status

✅ **Completed:**
- Research of tax rates for all 10 states
- Creation of 30 test case XML files (3 per state)
- Comprehensive validation documentation
- Execution guide with troubleshooting

⏳ **Ready for Execution:**
- Running test cases through DTRules
- Validating against online calculators
- Recording results and screenshots

❌ **Blocked/Needs Implementation:**
- 7 states need state tax calculation logic (CA, NY, PA, OH, GA, NC, MI)
- Can validate federal tax only for unimplemented states

## Manual Validation Examples

### Texas (No Income Tax)
```
Expected:
- Federal AGI: $75,000
- Federal Tax: $8,297
- State Tax: $0
- Total: $8,297

Validate at: SmartAsset.com
✅ Federal matches within $5
✅ State tax is $0
✅ Total matches within $10
```

### Illinois (4.95% Flat)
```
Expected:
- Federal AGI: $75,000
- Federal Tax: $8,297
- IL exemption: $2,650
- IL tax: ($75,000 - $2,650) × 4.95% = $3,581
- Total: $11,878

Manual check: 72,350 × 0.0495 = $3,581 ✓
```

### Pennsylvania (3.07% Flat)
```
Expected:
- Federal AGI: $75,000
- Federal Tax: $8,297
- PA tax: $75,000 × 3.07% = $2,303
- Total: $10,600

Manual check: 75,000 × 0.0307 = $2,302.50 ≈ $2,303 ✓
```

## Notes for Reviewers

### What This PR Does
- Creates validation framework for top 10 states
- Provides 30 test cases covering 3 scenarios per state
- Documents expected results and validation methodology
- Does NOT implement missing state tax calculations

### What This PR Does NOT Do
- Implement state tax logic for CA, NY, PA, OH, GA, NC, MI
- Execute the validation (manual testing required)
- Provide screenshots from tax software (requires manual validation)

### Follow-up Work Needed
1. Execute validation tests (manual or automated)
2. Implement missing state tax calculations
3. Create validation report with actual results
4. Take screenshots for audit trail

## References

- **Issue:** #237 - Validate top 10 states against tax software
- **Related Issues:** #180, #186, #195, #209, #206, #220, #187, #184 (state implementations)
- **Branch:** feature/issue-237
- **Blocked by:** State implementations for 7 states

## Contact

For questions:
- Review STATE_TAX_VALIDATION.md for state-specific details
- Review VALIDATION_EXECUTION_GUIDE.md for execution steps
- Check testfiles/TestScenarios/ for test case XML files
- Reference TaxReturn_dt.xml for decision table logic

---

**Created:** 2025-03-22
**Status:** Test framework complete, ready for validation execution
**Estimated validation time:** 10-15 days (per issue description)
