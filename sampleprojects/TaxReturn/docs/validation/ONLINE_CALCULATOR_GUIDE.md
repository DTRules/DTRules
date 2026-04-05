# Online Tax Calculator Validation Guide

**Issue:** #237
**Purpose:** Manual validation of state tax calculations using online calculators

## Overview

This guide provides step-by-step instructions for validating DTRules tax calculations against online tax calculators. Since some states are not yet implemented in DTRules, we use online calculators to establish baseline "correct" values.

---

## Recommended Online Calculators

### General Multi-State Calculators
1. **TurboTax TaxCaster**
   - URL: https://turbotax.intuit.com/tax-tools/calculators/taxcaster/
   - Best for: All states, comprehensive
   - Features: State-by-state breakdown, filing status options

2. **H&R Block Tax Calculator**
   - URL: https://www.hrblock.com/tax-calculator/
   - Best for: Quick estimates, simple scenarios
   - Features: Federal and state combined

3. **NerdWallet State Tax Calculator**
   - URL: https://www.nerdwallet.com/taxes/tax-calculator
   - Best for: State comparisons
   - Features: Side-by-side state comparisons

### State-Specific Calculators

#### California
- **Franchise Tax Board Calculator**
  - URL: https://www.ftb.ca.gov/file/personal/tax-calculator-tables-rates.asp
  - Official state calculator

#### New York
- **NYS Tax Calculator**
  - URL: https://www.tax.ny.gov/pit/file/tax-calculator.htm
  - Official state calculator

#### Illinois
- **Illinois Department of Revenue**
  - URL: https://www2.illinois.gov/rev/
  - Tax tables available

#### Pennsylvania
- **PA Department of Revenue**
  - URL: https://www.revenue.pa.gov/
  - Simple flat rate calculator

#### Ohio
- **Ohio Department of Taxation**
  - URL: https://tax.ohio.gov/
  - Tax tables and worksheets

---

## Validation Procedure

### Step 1: Prepare Test Scenario

For each state, use the standardized test scenarios from VALIDATION_REPORT.md:

- **Scenario A:** Single, $65,000 W-2
- **Scenario B:** Married Filing Jointly, $120,000 W-2
- **Scenario C:** Single, $200,000 W-2

### Step 2: Input Data into Calculator

Enter the following information consistently:
- **Tax Year:** 2025
- **Filing Status:** Single or Married Filing Jointly
- **State:** [Current state being tested]
- **W-2 Income:** Per scenario
- **Federal Withholding:** Per test case
- **State Withholding:** Per test case
- **Standard Deduction:** Use calculator default
- **No other income or deductions**

### Step 3: Record Results

For each calculation, record:
- State AGI (if shown)
- Standard deduction (if shown)
- Personal exemptions (if applicable)
- State taxable income
- **State tax calculated**
- Calculator URL and date

### Step 4: Screenshot Documentation

Capture screenshots showing:
- Input values entered
- Final tax calculation result
- Calculator name and URL (visible in browser)

Save screenshots as:
- `screenshots/[STATE]_[SCENARIO]_[CALCULATOR].png`

Example: `screenshots/CA_ScenarioA_TurboTax.png`

### Step 5: Compare with Expected Values

Compare calculator results with expected values in VALIDATION_REPORT.md:
- If within $1: ✅ Validation passed
- If difference > $1: ⚠️ Investigate discrepancy

### Step 6: Document Findings

Update VALIDATION_REPORT.md with:
- Actual calculator result
- Calculator name and date
- Any discrepancies found
- Investigation notes

---

## Validation Checklist

### Illinois (IL) - Implemented State

- [ ] **Scenario A - Single, $65,000**
  - [ ] Run through DTRules engine
  - [ ] Validate with TurboTax
  - [ ] Validate with H&R Block
  - [ ] Validate with IL state calculator
  - [ ] Screenshot all results
  - [ ] Expected: $3,080.14
  - [ ] DTRules result: _______
  - [ ] TurboTax result: _______
  - [ ] H&R Block result: _______
  - [ ] IL official result: _______
  - [ ] Match within $1? ___

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Run through DTRules engine
  - [ ] Validate with online calculators
  - [ ] Expected: $5,665.28
  - [ ] DTRules result: _______
  - [ ] Calculator results: _______
  - [ ] Match within $1? ___

- [ ] **Scenario C - Single, $200,000**
  - [ ] Run through DTRules engine
  - [ ] Validate with online calculators
  - [ ] Expected: $9,762.64
  - [ ] DTRules result: _______
  - [ ] Calculator results: _______
  - [ ] Match within $1? ___

### California (CA) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with TurboTax
  - [ ] Validate with CA FTB calculator
  - [ ] Expected: ~$2,915
  - [ ] TurboTax result: _______
  - [ ] CA FTB result: _______
  - [ ] Screenshot captured? ___

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Validate with online calculators
  - [ ] Expected: ~$5,850
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Validate with online calculators
  - [ ] Expected: ~$14,250
  - [ ] Results: _______

### Texas (TX) - No State Tax

- [x] All scenarios: $0 (no calculation needed)
- [x] DTRules correctly handles TX

### Florida (FL) - No State Tax

- [x] All scenarios: $0 (no calculation needed)
- [x] DTRules correctly handles FL

### New York (NY) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with TurboTax
  - [ ] Validate with NY state calculator
  - [ ] Expected: ~$2,970
  - [ ] Results: _______

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Validate with online calculators
  - [ ] Expected: ~$5,600
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Validate with online calculators
  - [ ] Expected: ~$11,500
  - [ ] Results: _______

### Pennsylvania (PA) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with online calculators
  - [ ] Expected: $1,995.50
  - [ ] Results: _______

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Expected: $3,684.00
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Expected: $6,140.00
  - [ ] Results: _______

### Ohio (OH) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with online calculators
  - [ ] Expected: $1,071.13
  - [ ] Results: _______

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Expected: $2,658.63
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Expected: $5,158.63
  - [ ] Results: _______

### Georgia (GA) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with online calculators
  - [ ] Expected: $2,953.11
  - [ ] Results: _______

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Expected: $5,579.25
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Expected: $9,959.61
  - [ ] Results: _______

### North Carolina (NC) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with online calculators
  - [ ] Expected: $2,162.19
  - [ ] Results: _______

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Expected: $3,899.38
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Expected: $7,899.69
  - [ ] Results: _______

### Michigan (MI) - Not Implemented

- [ ] **Scenario A - Single, $65,000**
  - [ ] Validate with online calculators
  - [ ] Expected: $2,524.50
  - [ ] Results: _______

- [ ] **Scenario B - MFJ, $120,000**
  - [ ] Expected: $4,624.00
  - [ ] Results: _______

- [ ] **Scenario C - Single, $200,000**
  - [ ] Expected: $8,262.00
  - [ ] Results: _______

---

## Common Issues and Solutions

### Issue: Calculator gives different result than expected

**Possible Causes:**
1. Different tax year (ensure 2025)
2. Different deduction amounts
3. Calculator using different state rules
4. Expected value calculation error

**Solution:**
1. Verify all inputs match test scenario exactly
2. Check if calculator shows intermediate values (AGI, deductions)
3. Try a different calculator
4. Document the discrepancy in VALIDATION_REPORT.md
5. Consult official state tax forms for verification

### Issue: Calculator doesn't support 2025 tax year

**Solution:**
1. Use 2024 tax year as proxy (note this in documentation)
2. Adjust for known rate changes (e.g., NC rate changed from 4.5% to 4.25%)
3. Note limitations in VALIDATION_REPORT.md

### Issue: State-specific rules not captured

**Examples:**
- Pennsylvania: No standard deduction
- Illinois: Retirement income subtraction
- California: Mental health surcharge above $1M

**Solution:**
1. Use state-specific calculators when possible
2. Consult official state tax forms
3. Perform manual calculation as verification
4. Document special rules in VALIDATION_REPORT.md

---

## Manual Calculation Verification

For critical scenarios, perform manual calculations using official state tax forms:

### Example: Illinois Scenario A

**Using IL Form IL-1040 Instructions:**

1. **Federal AGI:** $65,000 (from federal Form 1040 Line 11)
2. **Subtractions:** $0 (no retirement income)
3. **Illinois Base Income:** $65,000
4. **Personal Exemption:** $2,775 × 1 person = $2,775
5. **Illinois Net Income:** $65,000 - $2,775 = $62,225
6. **Illinois Tax:** $62,225 × 0.0495 = $3,080.1375
7. **Rounded:** $3,080.14

**Verification:**
- Online calculator: _______
- DTRules result: _______
- Manual calculation: $3,080.14
- Match? ___

---

## Screenshot Organization

Create a `screenshots` subdirectory:

```
Validation/
├── VALIDATION_REPORT.md
├── ONLINE_CALCULATOR_GUIDE.md
├── README.md
├── TestCase_IL_Validation_A.xml
├── TestCase_IL_Validation_B.xml
├── TestCase_IL_Validation_C.xml
└── screenshots/
    ├── CA_ScenarioA_TurboTax.png
    ├── CA_ScenarioA_CAFTB.png
    ├── IL_ScenarioA_DTRules.png
    ├── IL_ScenarioA_TurboTax.png
    ├── NY_ScenarioA_TurboTax.png
    ├── NY_ScenarioA_NYState.png
    └── ... (more screenshots)
```

---

## Reporting Results

### Summary Table Format

Update VALIDATION_REPORT.md with this table for each scenario:

| State | Scenario | Expected | DTRules | TurboTax | H&R Block | State Calc | Match? | Notes |
|-------|----------|----------|---------|----------|-----------|------------|--------|-------|
| IL | A | $3,080.14 | - | - | - | - | - | Pending |
| IL | B | $5,665.28 | - | - | - | - | - | Pending |
| IL | C | $9,762.64 | - | - | - | - | - | Pending |
| CA | A | ~$2,915 | N/A | - | - | - | - | Not impl |

### Discrepancy Documentation Format

If results don't match:

```markdown
#### Discrepancy: Illinois Scenario A

- **Expected:** $3,080.14
- **DTRules:** $3,082.50
- **TurboTax:** $3,080.00
- **Difference:** $2.36 (DTRules), $0.14 (TurboTax)

**Investigation:**
- DTRules appears to be rounding differently
- TurboTax rounds down to nearest dollar
- Expected calculation uses exact cents

**Resolution:**
- Acceptable difference (< $3)
- Consider updating DTRules rounding logic
- Document rounding behavior
```

---

## Timeline

Estimated time for complete validation:

- **Illinois (3 scenarios, 3 calculators):** 2 hours
- **California (3 scenarios, 2 calculators):** 1.5 hours
- **New York (3 scenarios, 2 calculators):** 1.5 hours
- **Pennsylvania (3 scenarios, 1 calculator):** 1 hour
- **Ohio (3 scenarios, 1 calculator):** 1 hour
- **Georgia (3 scenarios, 1 calculator):** 1 hour
- **North Carolina (3 scenarios, 1 calculator):** 1 hour
- **Michigan (3 scenarios, 1 calculator):** 1 hour
- **Documentation and screenshots:** 2 hours

**Total:** ~12-15 hours of manual validation work

---

## Next Steps After Validation

1. Update VALIDATION_REPORT.md with all results
2. Create GitHub issues for any discrepancies found
3. Prioritize state implementations based on:
   - Population (larger states first)
   - Complexity (simple flat-tax states first)
   - Validation results (states with verified calculations)
4. Begin implementation of non-implemented states
5. Create comprehensive test suites for each new state

---

## Resources

### Tax Forms (2025)
- [IRS Form 1040](https://www.irs.gov/forms-pubs/about-form-1040)
- [CA Form 540](https://www.ftb.ca.gov/forms/2025/2025-540.html)
- [NY Form IT-201](https://www.tax.ny.gov/forms/income_cur_forms.htm)
- [IL Form IL-1040](https://www2.illinois.gov/rev/forms/incometax/Pages/CurrentYear.aspx)
- [PA Form PA-40](https://www.revenue.pa.gov/FormsandPublications/FormsforIndividuals/PIT/Pages/default.aspx)
- [OH Form IT-1040](https://tax.ohio.gov/forms/ohio_individual_income_tax_forms)
- [GA Form 500](https://dor.georgia.gov/taxes/individual-income-tax/current-individual-income-tax-forms)
- [NC Form D-400](https://www.ncdor.gov/taxes-forms/individual-income-tax)
- [MI Form MI-1040](https://www.michigan.gov/taxes/iit/forms)

### Technical Documentation
- DTRules GitHub: https://github.com/DTRules/DTRules
- Issue #237: Cross-validate top 10 states
