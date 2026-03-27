# State Tax Implementation Recommendations

**Issue:** #237
**Purpose:** Prioritized roadmap for implementing remaining 7 states

## Executive Summary

Based on the validation work for issue #237, we recommend implementing the remaining 7 states (of the top 10 by population) in the following priority order. This order balances population size, implementation complexity, and validation readiness.

---

## Implementation Priority

### Phase 1: Flat Tax States (High Priority)

These states are simple to implement (single tax rate) and cover 4 of the top 10 states by population.

#### 1. Pennsylvania (PA) - Rank #5
- **Tax Type:** Flat 3.07%
- **Complexity:** ⭐ Very Simple
- **Population:** 12.9M (#5 largest)
- **Unique Features:**
  - No standard or itemized deductions
  - Uses PA-specific taxable income definition
  - Simplest state to implement

**Implementation Estimate:** 1-2 days
**Testing Estimate:** 1 day
**Total:** 2-3 days

**Decision Table:**
```
Calculate_PA_Tax
- No deductions
- No exemptions
- Flat 3.07% rate on PA taxable income
```

#### 2. North Carolina (NC) - Rank #9
- **Tax Type:** Flat 4.25%
- **Complexity:** ⭐ Very Simple
- **Population:** 10.7M (#9 largest)
- **Unique Features:**
  - Standard deduction: Single $14,125, MFJ $28,250
  - Rate recently reduced from 4.5% to 4.25% (2025)

**Implementation Estimate:** 1-2 days
**Testing Estimate:** 1 day
**Total:** 2-3 days

**Decision Table:**
```
Calculate_NC_Tax
- Apply standard deduction based on filing status
- Flat 4.25% rate
```

#### 3. Michigan (MI) - Rank #10
- **Tax Type:** Flat 4.25%
- **Complexity:** ⭐ Simple
- **Population:** 10.0M (#10 largest)
- **Unique Features:**
  - Personal exemption: $5,600 per person
  - Uses federal AGI with adjustments

**Implementation Estimate:** 1-2 days
**Testing Estimate:** 1 day
**Total:** 2-3 days

**Decision Table:**
```
Calculate_MI_Tax
- Start with federal AGI
- Apply personal exemptions ($5,600 per person)
- Flat 4.25% rate
```

#### 4. Georgia (GA) - Rank #8
- **Tax Type:** Flat 5.19%
- **Complexity:** ⭐⭐ Simple-Moderate
- **Population:** 10.9M (#8 largest)
- **Unique Features:**
  - Standard deduction: Single $5,400, MFJ $7,100
  - Personal exemption: $2,700 per person
  - Both deduction and exemption apply

**Implementation Estimate:** 2 days
**Testing Estimate:** 1 day
**Total:** 3 days

**Decision Table:**
```
Calculate_GA_Tax
- Apply standard deduction based on filing status
- Apply personal exemptions ($2,700 per person)
- Flat 5.19% rate
```

**Phase 1 Total:** 10-12 days for all 4 flat-tax states

---

### Phase 2: Moderate Complexity Progressive Tax

#### 5. Ohio (OH) - Rank #7
- **Tax Type:** Progressive (3 brackets)
- **Complexity:** ⭐⭐⭐ Moderate
- **Population:** 11.8M (#7 largest)
- **Unique Features:**
  - 3 brackets: 0%, 2.75%, 3.125%
  - First $26,050 tax-free
  - Transitioning to flat 2.75% in 2026

**Implementation Estimate:** 3-4 days
**Testing Estimate:** 2 days
**Total:** 5-6 days

**Decision Table:**
```
Calculate_OH_Tax
- Bracket 1: $0 - $26,050 at 0%
- Bracket 2: $26,051 - $100,000 at 2.75%
- Bracket 3: $100,001+ at 3.125%
- Progressive calculation similar to NH
```

**Tax Brackets (2025):**
| Income Range | Rate |
|--------------|------|
| $0 - $26,050 | 0% |
| $26,051 - $100,000 | 2.75% |
| $100,001+ | 3.125% |

---

### Phase 3: Complex Progressive Tax States

These states have 9 tax brackets and complex rules. Implement after gaining experience with simpler states.

#### 6. New York (NY) - Rank #4
- **Tax Type:** Progressive (9 brackets)
- **Complexity:** ⭐⭐⭐⭐ Complex
- **Population:** 19.5M (#4 largest)
- **Unique Features:**
  - 9 tax brackets: 4% to 10.9%
  - Top rate applies to income over $25 million
  - Standard deduction: Single $8,000, MFJ $16,050
  - NYC and Yonkers have additional local taxes (not in scope)

**Implementation Estimate:** 5-7 days
**Testing Estimate:** 3 days
**Total:** 8-10 days

**Decision Table:**
```
Calculate_NY_Tax
- Apply standard deduction
- 9-bracket progressive calculation
- May require tax table lookup for income ≤ $107,650
```

**Tax Brackets (Single, 2025):**
| Income Range | Rate |
|--------------|------|
| $0 - $8,500 | 4% |
| $8,501 - $11,700 | 4.5% |
| $11,701 - $13,900 | 5.25% |
| $13,901 - $80,650 | 5.5% |
| $80,651 - $215,400 | 6% |
| $215,401 - $1,077,550 | 6.85% |
| $1,077,551 - $5,000,000 | 9.65% |
| $5,000,001 - $25,000,000 | 10.3% |
| $25,000,001+ | 10.9% |

#### 7. California (CA) - Rank #1
- **Tax Type:** Progressive (9 brackets + surcharge)
- **Complexity:** ⭐⭐⭐⭐⭐ Very Complex
- **Population:** 39.0M (#1 largest)
- **Unique Features:**
  - 9 tax brackets: 1% to 12.3%
  - Mental health surcharge: +1% on income over $1M (top rate 13.3%)
  - Standard deduction: Single $5,706, MFJ $11,412
  - Most complex state tax system

**Implementation Estimate:** 7-10 days
**Testing Estimate:** 4 days
**Total:** 11-14 days

**Decision Table:**
```
Calculate_CA_Tax
- Apply standard deduction
- 9-bracket progressive calculation
- Mental health surcharge for income > $1M
- May require separate calculation for capital gains
```

**Tax Brackets (Single, 2025):**
| Income Range | Rate |
|--------------|------|
| $0 - $10,754 | 1% |
| $10,755 - $25,489 | 2% |
| $25,490 - $40,224 | 4% |
| $40,225 - $55,886 | 6% |
| $55,887 - $70,621 | 8% |
| $70,622 - $361,588 | 9.3% |
| $361,589 - $434,705 | 10.3% |
| $434,706 - $724,509 | 11.3% |
| $724,510+ | 12.3% (+1% if >$1M) |

---

## Overall Implementation Timeline

| Phase | States | Days | Cumulative |
|-------|--------|------|------------|
| Phase 1 | PA, NC, MI, GA | 10-12 | 10-12 |
| Phase 2 | OH | 5-6 | 15-18 |
| Phase 3a | NY | 8-10 | 23-28 |
| Phase 3b | CA | 11-14 | 34-42 |

**Total Implementation Time:** 34-42 working days (7-9 weeks)

With parallel work or experienced developers, this could be reduced to 4-6 weeks.

---

## Implementation Pattern

### 1. Entity Definitions

All states use existing entity definitions. No new entities required.

### 2. Decision Table Structure

Follow the pattern established in `Calculate_IL_Tax`, `Calculate_NH_Tax`, and `Calculate_MT_Tax`:

```xml
<decision_table>
  <table_name>Calculate_[STATE]_Tax</table_name>
  <Type>FIRST</Type>
  <COMMENTS>State-specific description</COMMENTS>
  <TABLE_NUMBER>[Next available number]</TABLE_NUMBER>

  <initial_actions>
    <!-- Initialize state-specific fields -->
    <!-- Log start of calculation -->
  </initial_actions>

  <conditions>
    <!-- Filing status conditions -->
    <!-- Income range conditions (for progressive) -->
  </conditions>

  <actions>
    <!-- Calculate deductions -->
    <!-- Calculate exemptions -->
    <!-- Apply tax rate(s) -->
    <!-- Store results in result.[state]_* fields -->
  </actions>
</decision_table>
```

### 3. Update Dispatch_State_Tax

Add new condition and action for each state:

```xml
<condition_details>
  <condition_number>X</condition_number>
  <condition_comment>[State] state</condition_comment>
  <condition_postfix>
    state_period isnull not
    if
      state_period.state_code [STATE] streq
    else
      job.state [STATE] streq
    then
  </condition_postfix>
</condition_details>

<action_details>
  <action_number>X</action_number>
  <action_description>Calculate_[STATE]_Tax</action_description>
  <action_postfix>
    Calculate_[STATE]_Tax
  </action_postfix>
</action_details>
```

### 4. Result Fields

Add to result entity in EDD:

```xml
<!-- [STATE] Tax Calculations -->
<field name='[state]_agi' type='double' comment='[State] Adjusted Gross Income'/>
<field name='[state]_standard_deduction' type='double' comment='[State] standard deduction'/>
<field name='[state]_exemption_total' type='double' comment='[State] personal exemptions'/>
<field name='[state]_taxable_income' type='double' comment='[State] taxable income'/>
<field name='[state]_tax' type='double' comment='[State] income tax'/>
```

For progressive states, add bracket fields:
```xml
<field name='[state]_bracket_1_tax' type='double'/>
<field name='[state]_bracket_2_tax' type='double'/>
<!-- etc. -->
```

### 5. Test Cases

Create 3 test cases per state:
- Low income (tests first bracket or exemption coverage)
- Medium income (tests middle brackets)
- High income (tests top bracket)

Follow pattern in `/testfiles/Validation/TestCase_IL_Validation_*.xml`

### 6. Validation

- Run test cases through DTRules
- Validate against online calculators
- Compare with official state tax tables
- Achieve < $1 difference in all test cases

---

## Technical Considerations

### Rounding

- **Illinois:** Rounds to nearest cent
- **Most states:** Round to nearest dollar
- **Decision:** Document rounding behavior for each state

### Tax Tables vs. Calculation

Some states provide tax tables for income below a threshold:
- **New York:** Tax table for income ≤ $107,650
- **California:** Tax table for income ≤ $100,000

**Recommendation:** Implement formula calculation for all income levels. Tax tables are for manual filing convenience, not programmatic calculation.

### Filing Status

Implement support for:
- Single
- Married Filing Jointly
- Married Filing Separately
- Head of Household (where applicable)

Most states follow federal filing status definitions.

### Special Rules

Document state-specific rules in table comments:

**Pennsylvania:**
- No standard deduction
- No personal exemption
- Uses PA-specific income definition (not federal AGI)

**Illinois:**
- Subtracts retirement income (pensions, Social Security, IRA distributions)
- Uses federal AGI as starting point

**California:**
- Mental health surcharge on income > $1M
- Different rates for capital gains (in future)

---

## Testing Strategy

### Unit Tests
For each state:
1. Test each filing status
2. Test edge cases (zero income, exactly at bracket boundary)
3. Test with/without deductions and exemptions

### Integration Tests
1. Multi-state scenarios (using existing multi-state allocation)
2. Moving between states mid-year
3. Non-resident working in multiple states

### Validation Tests
1. Compare with online calculators (3 scenarios minimum)
2. Compare with official state tax forms (manual calculation)
3. Document all test results

### Acceptance Criteria
- All calculations within $1 of expected values
- All test cases pass
- Code reviewed and documented
- Validation report updated

---

## Risks and Mitigation

### Risk: Tax Law Changes

**Impact:** Rates, brackets, or rules change after implementation

**Mitigation:**
- Document tax year (2025) in all code comments
- Create configuration for rates and brackets
- Annual review and update process

### Risk: Complex State Rules

**Impact:** Some states have many special cases

**Mitigation:**
- Start with simple flat-tax states
- Thoroughly research and document special rules
- Validate with multiple sources
- Consider "not supported" for edge cases

### Risk: Testing Coverage

**Impact:** Edge cases not tested

**Mitigation:**
- Minimum 3 test scenarios per state
- Test all filing statuses
- Use online calculators for validation
- Manual verification with state forms

---

## Success Metrics

### Code Quality
- ✅ All decision tables follow established patterns
- ✅ Comprehensive comments and documentation
- ✅ Consistent field naming conventions

### Testing
- ✅ 3+ test cases per state
- ✅ All calculations within $1 of expected
- ✅ Validated against 2+ online calculators
- ✅ Test coverage for all filing statuses

### Documentation
- ✅ Implementation documented in table comments
- ✅ Special rules clearly documented
- ✅ Validation results documented
- ✅ User-facing documentation updated

### Timeline
- ✅ Flat-tax states completed within 2 weeks
- ✅ Ohio completed within 3 weeks
- ✅ New York completed within 5 weeks
- ✅ California completed within 8 weeks

---

## Resources Required

### Developer Skills
- DTRules XML syntax (EL/EBL)
- State tax law knowledge
- Excel (for optional spreadsheet decision tables)
- Testing and validation

### Tools
- DTRules development environment
- Maven for building and testing
- Online tax calculators
- State tax forms and instructions

### Time Commitment
- **Phase 1 (Flat-tax states):** 2-3 weeks (one developer)
- **Phase 2 (Ohio):** 1 week
- **Phase 3 (NY, CA):** 4-5 weeks

**Total:** 7-9 weeks for complete implementation

---

## Deliverables

For each implemented state:

1. **Decision Table:** `Calculate_[STATE]_Tax` in TaxReturn_dt.xml
2. **Entity Updates:** Result fields in TaxReturn_edd.xml
3. **Dispatch Update:** Condition and action in Dispatch_State_Tax
4. **Test Cases:** 3+ XML test files in /testfiles/Validation/
5. **Documentation:**
   - Implementation notes in table comments
   - Validation results in VALIDATION_REPORT.md
   - User documentation updates
6. **Validation:**
   - Online calculator results
   - Screenshots
   - Acceptance sign-off

---

## Next Steps

1. **Prioritize Phase 1** - Implement PA, NC, MI, GA (flat-tax states)
2. **Create implementation tickets** - One per state with detailed specs
3. **Assign resources** - Developer(s) to work on implementation
4. **Set milestones** - Bi-weekly progress reviews
5. **Continuous validation** - Test as you build
6. **Documentation** - Update as you go, not at the end

---

## Appendix: Rate Reference Table

Quick reference for all state rates:

| State | Type | Primary Rate(s) | Deduction (Single) | Exemption | Notes |
|-------|------|-----------------|-------------------|-----------|-------|
| PA | Flat | 3.07% | None | None | Simplest |
| NC | Flat | 4.25% | $14,125 | None | New 2025 rate |
| MI | Flat | 4.25% | None | $5,600/person | |
| GA | Flat | 5.19% | $5,400 | $2,700/person | |
| IL | Flat | 4.95% | None | $2,775/person | Implemented |
| OH | Progressive | 0%, 2.75%, 3.125% | None | None | 3 brackets |
| NY | Progressive | 4%-10.9% | $8,000 | None | 9 brackets |
| CA | Progressive | 1%-13.3% | $5,706 | None | 9 brackets + surcharge |

**Legend:**
- Flat: Single tax rate
- Progressive: Multiple tax brackets
- Deduction: Standard deduction amount
- Exemption: Personal exemption per person

---

**Document Version:** 1.0
**Last Updated:** March 22, 2026
**Author:** DTRules Development Team (Issue #237)
