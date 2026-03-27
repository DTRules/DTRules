# TaxReturn Comprehensive Test Coverage Plan

Successfully created 18 sub-issues for organized test development.

## Issue Numbers and Organization

### HIGH Priority (5 groups) - Implement First
- **Issue #351**: Test Group 1: Federal Core & Credits (~15 tests)
  - Form 1040, CTC, EITC, CDCC, Premium Tax Credit, Energy Credits, Education Credits
  - Location: `testfiles/TestScenarios/Credits/`

- **Issue #352**: Test Group 2: Schedules A, C, D, E (~20 tests)
  - Schedule A (itemized), C (self-employment), D (capital gains), E (rental)
  - Location: `testfiles/TestScenarios/Schedules/`

- **Issue #359**: Test Group 9: Military Tax Provisions (~10 tests)
  - Combat zone exclusion, PCS moves, death benefits, MSRRA scenarios
  - Location: `testfiles/TestScenarios/Military/`

- **Issue #365**: Test Group 15: State Tax - Partial Military Exemption (14 tests)
  - CA, CO, DE, DC, GA, ID, KY, MD, MT, NM, OR, UT, VT, VA
  - Complex state-specific rules (age-based, income-based, date-based)
  - Location: `testfiles/TestScenarios/State/PartialExemption/`

- **Issue #368**: Test Group 18: Integration & Real-World Scenarios (~10 tests)
  - End-to-end validation combining multiple forms
  - Location: `testfiles/TestScenarios/Integration/`

### MEDIUM Priority (10 groups)
- **Issue #353**: Test Group 3: Above-the-Line Deductions (~10 tests)
- **Issue #354**: Test Group 4: Self-Employment & Additional Taxes (~12 tests)
- **Issue #355**: Test Group 5: Retirement & Social Security (~10 tests)
- **Issue #356**: Test Group 6: Business & Partnership Forms (~15 tests)
- **Issue #358**: Test Group 8: OBBBA 2025 & Special Deductions (~12 tests)
- **Issue #360**: Test Group 10: Foreign Income & Tax (~8 tests)
- **Issue #361**: Test Group 11: Kiddie Tax & Dependent Returns (~6 tests)
- **Issue #362**: Test Group 12: Special Situations & Edge Cases (~10 tests)
- **Issue #364**: Test Group 14: State Tax - Full Military Exemption (5 tests)
- **Issue #366**: Test Group 16: Multi-State Scenarios (~8 tests)

### LOW Priority (3 groups) - Implement Last
- **Issue #357**: Test Group 7: Household Employment & Schedule H (~5 tests)
- **Issue #363**: Test Group 13: State Tax - No Income Tax States (9 tests)
- **Issue #367**: Test Group 17: Estimated Tax & Penalties (~6 tests)

## Summary Statistics

**Total Test Groups:** 18
**Estimated Test Files:** ~190 test cases
**Issue Range:** #351 - #368
**All issues linked to:** Parent issue #88

## Recommended Implementation Order

### Phase 1: Foundation (Issues #351, #352)
Start with federal core forms and common schedules to establish baseline test infrastructure.
- Federal core credits and Form 1040 variants
- Common schedules (A, C, D, E)

### Phase 2: Recent Features (Issues #359, #365)
Test recently implemented military provisions and complex state tax logic.
- Military combat zone, PCS, MSRRA
- Complex state rules (CA, MD, OR, MT, etc.)

### Phase 3: Federal Forms (Issues #353-358, #360-362)
Complete federal form coverage.
- Adjustments, additional taxes, retirement
- Business forms, foreign income, special situations

### Phase 4: State Tax (Issues #363, #364, #366)
Complete remaining state tax scenarios.
- No-tax states verification
- Full exemption states (representative tests)
- Multi-state allocation scenarios

### Phase 5: Integration (Issue #368)
Real-world end-to-end scenarios combining multiple forms.
- Family scenarios (all credits)
- Business owner scenarios
- Military family scenarios
- Foreign worker scenarios

### Phase 6: Polish (Issue #367)
Final low-priority items.
- Estimated tax penalties
- Underpayment calculations

## Implementation Guidelines

### Test File Naming Convention
```
TestCase_[Category]_[Number]_[Description].xml
```

Examples:
- `TestCase_Credits_01_CTC_Single_2_Children.xml`
- `TestCase_Military_05_Combat_Zone_Enlisted.xml`
- `TestCase_State_MD_01_Military_Retirement_Age_60.xml`

### Test Structure
Each test case should include:
1. Clear scenario description in XML comments
2. Input data (taxpayer info, income, deductions)
3. Expected results (AGI, taxable income, tax liability)
4. Form-specific outputs (credit amounts, state tax, etc.)

### Development Process
1. Create issue branch: `git checkout -b feature/issue-XXX`
2. Create test directory if needed
3. Develop test cases
4. Run tests: `cd go && go test ./pkg/dtrules/... -run TestTaxReturn`
5. Commit and create PR
6. Link PR to issue

### Parallel Development
Multiple groups can be developed in parallel:
- Federal groups (1-12) are independent
- State groups (13-16) are independent
- Integration group (18) depends on others

## View All Issues

```bash
# List all test group issues
gh issue list --label "tax-rules" --state open

# View specific issue
gh issue view 351
```

## Current Status

- [x] Test framework exists (Comprehensive/, Level1_Simple/, etc.)
- [x] 18 sub-issues created (#351-368)
- [ ] Begin Phase 1 implementation
- [ ] Track progress in parent issue #88
