# State Income Tax Implementation Issues

This document contains the issues to create on GitHub for implementing comprehensive state income tax support in DTRules TaxReturn.

**Revised**: 2026-03-22 - Minimal foundation approach (states start immediately)

---

## EPIC ISSUE: State Income Tax Implementation

Create this as the epic issue on GitHub:

### Title
State Income Tax Implementation - Full 41-State Coverage

### Description
Implement comprehensive state income tax calculation for all 41 states (plus DC) that have income tax. This includes state-specific brackets, deductions, credits, and conformity rules with federal AGI.

### Scope
- Minimal foundation (state result entity only)
- State entity definitions and decision tables (self-contained)
- 41 states + DC with income tax
- Multi-state resident/non-resident scenarios
- Comprehensive test coverage
- Integration with existing federal tax engine

### Architecture Approach
**Minimal Foundation**: Each state issue is self-contained and includes:
- Researching state-specific constants
- Adding constants to EDD
- Creating decision table
- Creating test cases
- Adding branch to state dispatcher

**Benefits**: Maximum parallelism, no coordination overhead, states start immediately.

### Tasks

**Phase 1: Minimal Foundation (Issue #200)**
- [ ] #200 - Add state tax result entity (1-2 hours)

**Phase 2: Flat Tax States (Issues #210-215) - depends on #200**
- [ ] #210 - Implement Colorado flat tax (4.40%)
- [ ] #211 - Implement Illinois flat tax (4.95%)
- [ ] #212 - Implement Indiana flat tax (3.15%)
- [ ] #213 - Implement Michigan flat tax (4.25%)
- [ ] #214 - Implement North Carolina flat tax (4.50%)
- [ ] #215 - Implement Pennsylvania flat tax (3.07%)

**Phase 3: Northeast Progressive States (Issues #220-228) - depends on #200**
- [ ] #220 - Implement New York progressive tax
- [ ] #221 - Implement Massachusetts progressive tax
- [ ] #222 - Implement Connecticut progressive tax
- [ ] #223 - Implement New Jersey progressive tax
- [ ] #224 - Implement Vermont progressive tax
- [x] #225 - Implement Maine progressive tax
- [x] #226 - Implement Rhode Island progressive tax
- [ ] #227 - Implement New Hampshire (interest/dividends only)
- [ ] #228 - Implement Washington DC progressive tax

**Phase 4: West Coast States (Issues #230-233) - depends on #200**
- [ ] #230 - Implement California progressive tax (complex)
- [ ] #231 - Implement Oregon progressive tax
- [ ] #232 - Implement Hawaii progressive tax
- [ ] #233 - Implement Idaho progressive tax

**Phase 5: Midwest/Great Plains (Issues #240-250) - depends on #200**
- [ ] #240 - Implement Wisconsin progressive tax
- [ ] #241 - Implement Minnesota progressive tax
- [ ] #242 - Implement Iowa progressive tax
- [ ] #243 - Implement Missouri progressive tax
- [ ] #244 - Implement Kansas progressive tax
- [ ] #245 - Implement Nebraska progressive tax
- [ ] #246 - Implement North Dakota progressive tax
- [ ] #247 - Implement Ohio progressive tax
- [ ] #248 - Implement Kentucky progressive tax
- [ ] #249 - Implement Montana progressive tax

**Phase 6: South/Southeast (Issues #260-269) - depends on #200**
- [ ] #260 - Implement Georgia progressive tax
- [ ] #261 - Implement South Carolina progressive tax
- [ ] #262 - Implement Virginia progressive tax
- [ ] #263 - Implement West Virginia progressive tax
- [ ] #264 - Implement Alabama progressive tax
- [ ] #265 - Implement Mississippi progressive tax
- [ ] #266 - Implement Louisiana progressive tax
- [ ] #267 - Implement Arkansas progressive tax
- [ ] #268 - Implement Oklahoma progressive tax
- [ ] #269 - Implement Tennessee progressive tax

**Phase 7: Additional States (Issues #270-280) - depends on #200**
- [ ] #270 - Implement Maryland progressive tax
- [ ] #271 - Implement Delaware progressive tax
- [ ] #272 - Implement Arizona progressive tax
- [x] #273 - Implement New Mexico progressive tax
- [ ] #274 - Implement Utah progressive tax
- [ ] #275 - Implement Alaska progressive tax
- [ ] #276 - Implement South Dakota progressive tax
- [ ] #277 - Implement Wyoming progressive tax
- [ ] #278 - Implement Nevada progressive tax
- [ ] #279 - Implement Washington progressive tax
- [ ] #280 - Implement Florida progressive tax

**Phase 8: No-Tax States Documentation (Issue #285) - depends on #200**
- [ ] #285 - Document no-income-tax states handling

**Phase 9: Multi-State Scenarios (Issues #290-294) - depends on #210, #220, #230**
- [ ] #290 - Implement multi-state resident allocation
- [ ] #291 - Implement non-resident state tax calculation
- [ ] #292 - Implement part-year resident scenarios
- [ ] #293 - Implement reciprocal state agreements
- [ ] #294 - Implement credit for taxes paid to other states

**Phase 10: Validation and Documentation (Issues #300-304)**
- [ ] #300 - Create comprehensive state tax test suite (depends on #210-#280)
- [ ] #301 - Validate top 10 states against tax software (depends on #210-#280)
- [ ] #302 - Document state tax conformity matrix (depends on #210-#280)
- [ ] #303 - Create state tax API documentation (depends on #210-#280)
- [ ] #304 - Performance testing and optimization (depends on #210-#280)

---

## Individual Issue Templates

Copy each section below into a separate GitHub issue.

---

### Issue #200: Add State Tax Result Entity

**Description**

Add minimal state_tax_result entity to EDD to capture state tax calculation results. This is the only foundation requirement - states can then start implementing immediately.

**Acceptance Criteria**

- [ ] `state_tax_result` entity added to `TaxReturn_edd.xml` with fields:
  - `state_code` (string) - Two-letter state code
  - `state_agi` (double) - State AGI if different from federal
  - `state_taxable_income` (double) - State taxable income
  - `state_tax_before_credits` (double) - Tax before credits
  - `state_credits` (double) - Total state credits
  - `state_tax_liability` (double) - Final state tax liability
  - `state_withholding` (double) - State tax withheld
  - `state_refund_or_owed` (double) - Refund (positive) or owed (negative)
- [ ] Output mapping added to `TaxReturn_map.xml`
- [ ] Entity loads correctly (smoke test)

**Technical Notes**

- Files to modify:
  - `sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
  - `sampleprojects/TaxReturn/xml/TaxReturn_map.xml`
- Keep minimal - states add additional fields as needed
- Pattern after existing `result` entity
- Estimate: 1-2 hours

---

### Issue #210: Implement Colorado Flat Tax (4.40%)

**Description**

Implement Colorado state income tax calculation. Colorado has a flat 4.40% rate on federal taxable income with some Colorado-specific deductions and credits.

**This is the first state implementation** and establishes patterns for other states. As part of this issue, create the simple `Calculate_State_Tax` dispatcher table that other states will add to.

**Acceptance Criteria**

**Colorado-Specific**:
- [ ] Research Colorado 2025 tax constants from official sources
- [ ] Add CO constants to `TaxReturn_edd.xml`:
  - CO tax rate: 4.40%
  - Capital gains deduction rate: 39%
  - Standard deduction: uses federal
- [ ] `Calculate_CO_Tax` decision table created (TABLE 41000)
- [ ] Colorado starts with federal taxable income
- [ ] Colorado-specific additions/subtractions applied:
  - State tax refund (if itemized federally)
  - Capital gains deduction (39% of net LTCG)
- [ ] 4.40% flat rate applied to adjusted taxable income
- [ ] Results stored in state_tax_result entity

**Dispatcher (First State Creates This)**:
- [ ] `Calculate_State_Tax` orchestration table created (TABLE 40000)
- [ ] Called from `Compute_Tax_Return` after federal tax calculation
- [ ] Branches based on `job.state`:
  - If "CO": calls Calculate_CO_Tax
  - Else: sets state_tax_liability to 0
- [ ] Audit trail logging for state calculations

**Testing**:
- [ ] Create test case directory: `testfiles/TestScenarios/State/CO/`
- [ ] At least 3 test cases created and passing:
  - `CO-01`: Single W-2 simple ($50k income)
  - `CO-02`: MFJ with capital gains ($100k + $20k LTCG)
  - `CO-03`: Self-employed with QBI ($75k)
- [ ] Test cases include expected state tax values
- [ ] Results match Colorado Form 104 calculation

**Technical Notes**

- Files to modify:
  - `sampleprojects/TaxReturn/xml/TaxReturn_dt.xml` (add tables)
  - `sampleprojects/TaxReturn/xml/TaxReturn_edd.xml` (add CO constants)
- Reference: Colorado Form 104 Instructions (2025)
- Colorado website: https://tax.colorado.gov/
- Colorado uses federal taxable income as starting point (rolling conformity)
- Capital gains deduction: 39% of net long-term capital gains (qualifying assets)
- Pattern this implementation establishes will be followed by other states
- Estimate: 3-4 days (includes creating dispatcher)

---

### Issue #211: Implement Illinois Flat Tax (4.95%)

**Description**

Implement Illinois state income tax calculation. Illinois has a flat 4.95% rate on federal AGI with Illinois-specific exemptions.

**Acceptance Criteria**

**Illinois-Specific**:
- [ ] Research Illinois 2025 tax constants from official sources
- [ ] Add IL constants to `TaxReturn_edd.xml`:
  - IL tax rate: 4.95%
  - Personal exemption: $2,775 per person (2025 estimate)
  - Retirement income subtraction rules
- [ ] `Calculate_IL_Tax` decision table created (TABLE 41100)
- [ ] Illinois starts with federal AGI
- [ ] Illinois-specific modifications applied:
  - Add back: state/municipal bond interest
  - Subtract: retirement income (qualifying pensions/Social Security)
- [ ] Personal exemption applied ($2,775 × number of people)
- [ ] 4.95% flat rate applied to net income
- [ ] Results stored in state_tax_result entity
- [ ] Add IL branch to `Calculate_State_Tax` dispatcher

**Testing**:
- [ ] Create test case directory: `testfiles/TestScenarios/State/IL/`
- [ ] At least 3 test cases created and passing:
  - `IL-01`: Single W-2 simple ($60k)
  - `IL-02`: MFJ with retirement income ($80k + $30k pension)
  - `IL-03`: Family with dependents (2 adults, 3 children, $90k)
- [ ] Results match Illinois Form IL-1040 calculation

**Technical Notes**

- Files to modify:
  - `sampleprojects/TaxReturn/xml/TaxReturn_dt.xml` (add Calculate_IL_Tax)
  - `sampleprojects/TaxReturn/xml/TaxReturn_edd.xml` (add IL constants)
- Reference: Illinois Form IL-1040 Instructions (2025)
- Illinois website: https://tax.illinois.gov/
- Illinois uses federal AGI as starting point (rolling conformity)
- Personal exemption: $2,775 per person (taxpayer, spouse, dependents)
- Estimate: 2-3 days

---

### Issue #212: Implement Indiana Flat Tax (3.15%)

**Description**

Implement Indiana state income tax calculation. Indiana has a flat 3.15% rate on federal AGI with Indiana-specific deductions and exemptions.

**Acceptance Criteria**

**Indiana-Specific**:
- [ ] Research Indiana 2025 tax constants
- [ ] Add IN constants to `TaxReturn_edd.xml`:
  - IN tax rate: 3.15%
  - Personal exemption: $1,500 per person, $3,500 per dependent
  - Additional exemption: $1,000 for 65+ or blind
  - Qualified retirement deduction: $6,250 (age 62+)
  - Renter's deduction: $3,000
- [ ] `Calculate_IN_Tax` decision table created (TABLE 41200)
- [ ] Indiana starts with federal AGI
- [ ] Indiana-specific modifications:
  - Military pay deduction
  - Qualified retirement income deduction
  - Renter's deduction (if applicable)
- [ ] Personal exemptions: $1,500 per person, $3,500 per dependent
- [ ] Additional exemption: $1,000 for 65+ or blind
- [ ] 3.15% flat rate applied
- [ ] Add IN branch to dispatcher
- [ ] Note in audit trail: county tax exists but not calculated

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/IN/`
- [ ] At least 3 test cases:
  - `IN-01`: Single W-2 ($55k)
  - `IN-02`: Senior with retirement income (age 65, $70k)
  - `IN-03`: Family with renter's deduction ($85k, renting)
- [ ] Results match Indiana Form IT-40 calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: Indiana Form IT-40 Instructions (2025)
- Indiana website: https://www.in.gov/dor/
- Note: Indiana also has county taxes (0.25%-3.38%) but those are out of scope
- Estimate: 2-3 days

---

### Issue #213: Implement Michigan Flat Tax (4.25%)

**Description**

Implement Michigan state income tax calculation. Michigan has a 4.25% flat rate on federal AGI with Michigan-specific deductions.

**Acceptance Criteria**

**Michigan-Specific**:
- [ ] Research Michigan 2025 tax constants
- [ ] Add MI constants to `TaxReturn_edd.xml`:
  - MI tax rate: 4.25%
  - MI standard deduction: $5,600 single, $11,200 MFJ
  - Personal exemption: $5,600 per person
  - Retirement income rules by birth year
- [ ] `Calculate_MI_Tax` decision table created (TABLE 41300)
- [ ] Michigan starts with federal AGI
- [ ] Michigan-specific modifications:
  - Social Security/military retirement subtraction
  - Private pension subtraction (with age-based limits)
- [ ] Michigan standard deduction applied
- [ ] Personal exemption: $5,600 per person
- [ ] 4.25% flat rate applied
- [ ] Add MI branch to dispatcher

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/MI/`
- [ ] At least 3 test cases:
  - `MI-01`: Single W-2 ($58k)
  - `MI-02`: MFJ with pension income ($95k)
  - `MI-03`: Family with dependents ($105k)
- [ ] Results match Michigan Form MI-1040 calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: Michigan Form MI-1040 Instructions (2025)
- Michigan website: https://www.michigan.gov/taxes/
- Michigan has complex retirement income rules based on birth year
- Estimate: 2-3 days

---

### Issue #214: Implement North Carolina Flat Tax (4.50%)

**Description**

Implement North Carolina state income tax calculation. North Carolina has a 4.50% flat rate with NC-specific standard deduction.

**Acceptance Criteria**

**North Carolina-Specific**:
- [ ] Research NC 2025 tax constants
- [ ] Add NC constants to `TaxReturn_edd.xml`:
  - NC tax rate: 4.50%
  - NC standard deduction: $12,750 single, $25,500 MFJ (2025 estimate)
- [ ] `Calculate_NC_Tax` decision table created (TABLE 41400)
- [ ] North Carolina starts with federal AGI
- [ ] NC-specific additions:
  - State/local tax refund (if itemized federally)
- [ ] NC-specific deductions:
  - NC standard deduction OR
  - NC itemized deductions (medical, mortgage interest, charitable)
- [ ] 4.50% flat rate applied
- [ ] Add NC branch to dispatcher

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/NC/`
- [ ] At least 3 test cases:
  - `NC-01`: Single W-2 ($62k)
  - `NC-02`: MFJ with itemized deductions ($110k)
  - `NC-03`: Self-employed ($80k)
- [ ] Results match NC Form D-400 calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: North Carolina Form D-400 Instructions (2025)
- NC website: https://www.ncdor.gov/
- NC has its own standard deduction (different from federal)
- Estimate: 2-3 days

---

### Issue #215: Implement Pennsylvania Flat Tax (3.07%)

**Description**

Implement Pennsylvania state income tax calculation. Pennsylvania has a 3.07% flat rate on specific income categories with NO deductions or exemptions.

**Acceptance Criteria**

**Pennsylvania-Specific**:
- [ ] Research PA 2025 tax constants
- [ ] Add PA constants to `TaxReturn_edd.xml`:
  - PA tax rate: 3.07%
  - PA income classes (8 categories)
- [ ] `Calculate_PA_Tax` decision table created (TABLE 41500)
- [ ] Pennsylvania taxes 8 income classes at 3.07%:
  - Compensation (W-2 wages)
  - Interest
  - Dividends
  - Net profits (business income)
  - Net gains from sale of property
  - Rents and royalties
  - Estate/trust income
  - Gambling/lottery winnings
- [ ] Exclusions: retirement income (pensions, Social Security, IRA distributions)
- [ ] No deductions allowed (except for business expenses)
- [ ] No personal exemptions
- [ ] Add PA branch to dispatcher

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/PA/`
- [ ] At least 3 test cases:
  - `PA-01`: Single W-2 ($65k)
  - `PA-02`: Retiree with pension and Social Security ($40k pension + $25k SS = $0 PA tax)
  - `PA-03`: Self-employed with rental income ($70k + $15k rental)
- [ ] Results match PA Form PA-40 calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: Pennsylvania Form PA-40 Instructions (2025)
- PA website: https://www.revenue.pa.gov/
- PA is unique - does not start with federal AGI
- PA excludes all retirement income (major benefit for retirees)
- Estimate: 2-3 days

---

### Issue #220: Implement New York Progressive Tax

**Description**

Implement New York state income tax calculation. New York has progressive tax brackets ranging from 4% to 10.9% on federal AGI with NY-specific modifications.

**Acceptance Criteria**

**New York-Specific**:
- [ ] Research NY 2025 tax constants
- [ ] Add NY constants to `TaxReturn_edd.xml`:
  - NY tax brackets (Single and MFJ, 9 brackets each)
  - NY standard deduction: $8,000 single, $16,050 MFJ
  - NY dependent exemption: $1,000 each
  - Public pension subtraction limit: $20,000
- [ ] `Calculate_NY_Tax` decision table created (TABLE 42000)
- [ ] `Apply_NY_Tax_Brackets_Single` helper table (TABLE 42001)
- [ ] `Apply_NY_Tax_Brackets_MFJ` helper table (TABLE 42002)
- [ ] NY starts with federal AGI
- [ ] NY-specific additions/subtractions:
  - Add: state/local tax refund (if itemized federally)
  - Subtract: NY state/local income taxes paid
  - Subtract: public pension subtraction ($20,000 max)
- [ ] NY standard deduction applied
- [ ] NY dependent exemption ($1,000 each)
- [ ] Progressive brackets applied (4% to 10.9%)
- [ ] Add NY branch to dispatcher

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/NY/`
- [ ] At least 3 test cases:
  - `NY-01`: Single W-2 ($50k income, ~4-5% effective rate)
  - `NY-02`: MFJ with pension ($100k W-2 + $30k pension)
  - `NY-03`: High earner ($500k+, top bracket)
- [ ] Results match NY Form IT-201 calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: New York Form IT-201 Instructions (2025)
- NY website: https://www.tax.ny.gov/
- NY 2025 brackets (Single):
  - 4%: $0 - $8,500
  - 4.5%: $8,501 - $11,700
  - 5.25%: $11,701 - $13,900
  - 5.85%: $13,901 - $80,650
  - 6.25%: $80,651 - $215,400
  - 6.85%: $215,401 - $1,077,550
  - 9.65%: $1,077,551 - $5,000,000
  - 10.3%: $5,000,001 - $25,000,000
  - 10.9%: $25,000,001+
- Pattern after federal bracket tables (Calculate_Tax_Single, etc.)
- Estimate: 4-5 days

---

### Issue #230: Implement California Progressive Tax (Complex)

**Description**

Implement California state income tax calculation. California has its own AGI calculation (partial federal conformity) and progressive brackets from 1% to 13.3%.

**WARNING**: California is the most complex state due to limited federal conformity. Consider allocating extra time or splitting into sub-issues.

**Acceptance Criteria**

**California-Specific**:
- [ ] Research CA 2025 tax constants and conformity rules
- [ ] Add CA constants to `TaxReturn_edd.xml`:
  - CA tax brackets (10 brackets × 4 filing statuses)
  - CA standard deduction: $5,363 single, $10,726 MFJ
  - Mental health services tax threshold: $1M
  - CA conformity deviations documented
- [ ] `Calculate_CA_AGI` sub-table created (TABLE 43001)
  - CA has unique AGI calculation
  - Different itemized deduction rules
  - State/local tax not deductible (no SALT)
  - Different depreciation rules
  - CA-specific adjustments documented
- [ ] `Calculate_CA_Tax` decision table created (TABLE 43000)
- [ ] `Apply_CA_Tax_Brackets` helper tables (per filing status)
- [ ] CA standard deduction applied
- [ ] Mental health services tax (1% surtax on income > $1M)
- [ ] Progressive brackets: 1% to 13.3%
- [ ] Add CA branch to dispatcher

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/CA/`
- [ ] At least 5 test cases (due to complexity):
  - `CA-01`: Single W-2 simple ($70k)
  - `CA-02`: MFJ with itemized deductions ($120k)
  - `CA-03`: Self-employed with CA-specific adjustments ($90k)
  - `CA-04`: High earner testing mental health surtax ($1.5M)
  - `CA-05`: Complex with multiple income sources
- [ ] Results match CA Form 540 calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: California Form 540 Instructions (2025)
- CA website: https://www.ftb.ca.gov/
- CA is most complex state due to limited federal conformity
- CA has its own versions of many federal provisions
- Highest state marginal rate in nation (13.3%)
- Consider: May want to split this into 2-3 issues if too large
- Estimate: 5-7 days

---

### Template for Other Progressive States (#221-280)

**Copy this template for each remaining state, filling in state-specific details:**

```markdown
### Issue #XXX: Implement [State Name] Progressive Tax

**Description**

Implement [State Name] state income tax calculation. [State] has progressive tax brackets ranging from X% to Y% on [federal AGI / federal taxable income / other].

**Acceptance Criteria**

**[State]-Specific**:
- [ ] Research [State] 2025 tax constants
- [ ] Add [STATE] constants to `TaxReturn_edd.xml`:
  - Tax brackets (list them)
  - Standard deduction amounts
  - Personal/dependent exemptions
  - State-specific rules
- [ ] `Calculate_[ST]_Tax` decision table created (TABLE 4XXXX)
- [ ] `Apply_[ST]_Tax_Brackets` helper tables (if progressive)
- [ ] [State] starts with [federal AGI / taxable income]
- [ ] [State]-specific additions/subtractions:
  - [List state-specific adjustments]
- [ ] Standard deduction or itemized deductions applied
- [ ] Tax brackets applied
- [ ] Add [ST] branch to dispatcher

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/[ST]/`
- [ ] At least 3 test cases:
  - `[ST]-01`: Single W-2 simple
  - `[ST]-02`: MFJ with [state-specific scenario]
  - `[ST]-03`: [Edge case or high earner]
- [ ] Results match [State] Form [XXX] calculation

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reference: [State] Form [XXX] Instructions (2025)
- [State] website: [URL]
- [State-specific notes]
- Estimate: 3-4 days
```

**States using this template:**
- #221: Massachusetts (5% + 4% surtax on $1M+, special Part B for STCG)
- #222: Connecticut (2% to 6.99%, phase-outs for high earners)
- #223: New Jersey (1.4% to 10.75%, starts with gross income)
- #224-228: VT, ME, RI, NH (dividends/interest only), DC
- #231-233: OR, HI, ID
- #240-249: WI, MN, IA, MO, KS, NE, ND, OH, KY, MT
- #260-280: Remaining states

---

### Issue #285: Document No-Income-Tax States Handling

**Description**

Document and implement handling for the 9 states with no personal income tax. These states should return zero tax liability.

**Acceptance Criteria**

- [ ] Document in `docs/no_income_tax_states.md`:
  - List of 9 states: TX, FL, WA, NV, SD, WY, AK, TN, NH
  - Note: TN and NH tax dividend/interest income only (out of scope)
  - Implementation approach
- [ ] Update `Calculate_State_Tax` dispatcher to return zero for these states
- [ ] Add to `TaxReturn_edd.xml`:
  - `no_income_tax_states` array constant (already exists, verify)
- [ ] Test cases created:
  - `testfiles/TestScenarios/State/NoTax/`
  - Test that TX, FL, WA, NV, SD, WY, AK residents get zero state tax
  - Test that TN, NH return zero (even though they technically tax div/int)
- [ ] Audit trail notes for no-tax states

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml, docs/no_income_tax_states.md
- Simple issue - just documentation and test validation
- May already be partially implemented (no_income_tax_states array exists)
- TN: Repealed Hall Tax (interest/dividends) effective 2021
- NH: Interest/dividends tax repealed effective 2025 (verify)
- Estimate: 1-2 days

---

### Issue #290: Implement Multi-State Resident Allocation

**Description**

Implement calculation logic for taxpayers who moved between states during the tax year (part-year resident in multiple states).

**Acceptance Criteria**

- [ ] Add `state_period` entity to EDD:
  - `state_code` (string)
  - `start_date` (date)
  - `end_date` (date)
  - `resident_status` (string: "resident", "non-resident", "part-year")
- [ ] Support multiple state_period entities per job
- [ ] Income allocation logic based on:
  - Dates of residency in each state
  - Source of income (W-2 location, business location, rental property location)
  - Days in each state (physical presence)
- [ ] `Allocate_Income_By_State` decision table (TABLE 45000)
- [ ] Each applicable state calculates tax on its allocated portion
- [ ] Update `Calculate_State_Tax` to handle multiple states

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/MultiState/`
- [ ] At least 3 test cases:
  - `MS-01`: Moved from NY to FL mid-year (Jan-Jun NY resident, Jul-Dec FL resident)
  - `MS-02`: Moved from CA to TX (high-tax to no-tax state)
  - `MS-03`: Traveling consultant working in multiple states
- [ ] Document allocation methodology

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml, docs/multi_state_allocation.md
- Complex logic - income allocation varies by type
- W-2 income: allocated by work location
- Business income: allocated by business location / days
- Rental income: allocated by property location
- Investment income: usually follows resident state
- Estimate: 5-7 days

---

### Issue #291: Implement Non-Resident State Tax Calculation

**Description**

Implement non-resident state tax calculation for income earned in a state where taxpayer is not a resident.

**Acceptance Criteria**

- [ ] Non-resident calculation logic for each state
- [ ] Only state-source income is taxed:
  - W-2 wages earned in state
  - Business income attributable to state
  - Rental property located in state
  - Capital gains generally excluded (unless real estate in state)
- [ ] Non-resident allocation formula:
  - Some states: (State source income / Total income) × State tax
  - Other states: Calculate tax directly on state-source income
- [ ] Update each state's Calculate_XX_Tax to handle resident_status
- [ ] Document non-resident rules per state

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/NonResident/`
- [ ] At least 3 test cases:
  - `NR-01`: CA resident working remotely for NY company (NY doesn't tax remote workers)
  - `NR-02`: TX resident with rental property in CO (CO taxes rental income)
  - `NR-03`: FL resident with business income in multiple states
- [ ] Results match state non-resident forms

**Technical Notes**

- Files: TaxReturn_dt.xml, docs/non_resident_rules.md
- Each state has different non-resident rules
- Some states don't tax non-residents on certain income types
- Reciprocity agreements affect this (see #293)
- Estimate: 4-5 days

---

### Issue #292: Implement Part-Year Resident Scenarios

**Description**

Implement part-year resident tax calculation where taxpayer is treated as resident for part of year and non-resident for the rest.

**Acceptance Criteria**

- [ ] Part-year resident calculation for each state
- [ ] Two calculations performed:
  - Resident calculation (all income during resident period)
  - Non-resident calculation (state-source income during non-resident period)
- [ ] Combined to produce final state liability
- [ ] Handles both moving into and out of state
- [ ] Handles domicile change vs physical presence
- [ ] Deduction allocation:
  - Some states pro-rate deductions by months
  - Other states allow full deductions

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/PartYear/`
- [ ] At least 3 test cases:
  - `PY-01`: Moved to CA in July (non-resident Jan-Jun, resident Jul-Dec)
  - `PY-02`: Moved from NY in March (resident Jan-Mar, non-resident Apr-Dec)
  - `PY-03`: Multiple moves during year (complex)
- [ ] Results match state part-year resident forms

**Technical Notes**

- Files: TaxReturn_dt.xml, docs/part_year_resident.md
- Most states have specific part-year resident forms
- Part-year is combination of resident and non-resident calculations
- Income must be allocated by period and source
- Estimate: 4-5 days

---

### Issue #293: Implement Reciprocal State Agreements

**Description**

Implement reciprocal agreement handling where certain states don't tax non-residents from partner states.

**Acceptance Criteria**

- [ ] Reciprocity matrix documented in EDD:
  - IL: IA, KY, MI, WI
  - IN: KY, MI, OH, PA, WI
  - MD: DC, PA, VA, WV
  - MT: ND
  - NJ: PA
  - VA: DC, KY, MD, PA, WV
  - WV: KY, MD, OH, PA, VA
- [ ] Logic to skip non-resident tax when reciprocity applies
- [ ] Only applies to wage income (W-2), not business/rental/investment
- [ ] Add reciprocity check to non-resident calculation

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/Reciprocity/`
- [ ] Test cases for each reciprocal relationship:
  - `RC-01`: IL resident working in WI (no WI tax due to reciprocity)
  - `RC-02`: PA resident working in NJ (no NJ tax, only PA tax)
  - `RC-03`: MD resident working in VA (no VA tax)
- [ ] Verify non-wage income still taxed

**Technical Notes**

- Files: TaxReturn_dt.xml, TaxReturn_edd.xml
- Reciprocity only applies to wage income
- Must verify agreements are still active (check annually)
- Employee should file exemption form in work state
- Estimate: 2-3 days

---

### Issue #294: Implement Credit for Taxes Paid to Other States

**Description**

Implement credit calculation where resident state provides credit for taxes paid to non-resident state, preventing double taxation.

**Acceptance Criteria**

- [ ] Credit calculation for each state that allows it (most do)
- [ ] Credit formula:
  - Lesser of: (1) Tax paid to other state, or (2) Resident state tax on same income
- [ ] Handles multiple non-resident states (credit allocated proportionally)
- [ ] Credit limitations:
  - Cannot exceed resident state tax liability
  - Only for actual taxes paid (not assessed but unpaid)
  - Only for income also taxed by resident state
- [ ] `Calculate_Other_State_Tax_Credit` decision table (TABLE 45100)
- [ ] Add credit calculation to each state's Calculate_XX_Tax

**Testing**:
- [ ] Test directory: `testfiles/TestScenarios/State/Credit/`
- [ ] At least 3 test cases:
  - `CR-01`: NY resident working in NJ (both tax, NY gives credit)
  - `CR-02`: CA resident with rental in AZ (CA gives credit for AZ tax on rental)
  - `CR-03`: IL resident with income in 3 states (proportional credit)
- [ ] Results match state credit for taxes paid forms (Schedule CR, etc.)

**Technical Notes**

- Files: TaxReturn_dt.xml
- Most states have Form "Schedule CR" or similar
- Credit prevents double taxation but doesn't eliminate it (state rate differences)
- Some states don't give credit (NH, WA - but they have no income tax anyway)
- Estimate: 4-5 days

---

### Issue #300: Create Comprehensive State Tax Test Suite

**Description**

Create a comprehensive test suite that validates all state tax calculations against known good results.

**Acceptance Criteria**

- [ ] At least 3 test cases per state (123+ test cases minimum)
- [ ] Test case coverage for each state:
  - Single W-2 (simple case)
  - MFJ with multiple income sources
  - High earner testing top bracket (if progressive)
- [ ] Test cases validated against:
  - State tax software, or
  - Hand calculation using state forms, or
  - Online state tax calculators
- [ ] `comprehensive_state_tax_test.go` created
- [ ] All tests pass (100% success rate)
- [ ] Test results documented with expected values
- [ ] Any known discrepancies documented

**Technical Notes**

- Files to create:
  - `go/pkg/dtrules/comprehensive_state_tax_test.go`
  - Test case XMLs (already created by individual state issues)
- This issue compiles all state test cases into comprehensive suite
- Runs all 123+ tests in one go
- Consider using official state test scenarios where available
- Estimate: 5-7 days

---

### Issue #301: Validate Top 10 States Against Tax Software

**Description**

Cross-validate DTRules state tax calculations against commercial tax software (TurboTax, TaxAct, or H&R Block) for the 10 highest-population states.

**Scope limited to**: CA, TX, FL, NY, PA, IL, OH, GA, NC, MI (54% of US population)

**Acceptance Criteria**

- [ ] 10 representative test cases prepared (1 per state)
- [ ] Each test case run through:
  - DTRules implementation
  - Commercial tax software (TurboTax or TaxAct)
  - Manual calculation using state forms
- [ ] Results compared (must match within $1)
- [ ] Any discrepancies investigated and resolved
- [ ] Validation report documenting:
  - Test cases used
  - Software versions
  - Results comparison table
  - Any known differences
  - Screenshots or exports from tax software

**Technical Notes**

- May require purchasing tax software or using free trials
- Focus on states with highest populations (54% coverage with 10 states)
- Document any cases where software differs from official forms
- This may require human assistance (AI can't run GUI software)
- Consider: Use online state tax calculators instead of installed software
- Estimate: 10-15 days (manual work intensive)

---

### Issue #302: Document State Tax Conformity Matrix

**Description**

Create comprehensive documentation of each state's conformity to federal tax law.

**Acceptance Criteria**

- [ ] Documentation created: `docs/state_tax_conformity.md`
- [ ] Matrix includes for each of 41 states:
  - Federal conformity type (rolling, static, custom)
  - Federal conformity date (if static conformity)
  - Starting point (federal AGI, federal taxable income, gross income)
  - Major federal provisions NOT adopted by state
  - State-specific additions to income
  - State-specific subtractions from income
  - Last verification date
- [ ] Source citations for each state
- [ ] Summary table comparing all states
- [ ] Notes on states with unique approaches (CA, PA, etc.)

**Technical Notes**

- Research from:
  - Federation of Tax Administrators (taxadmin.org)
  - State revenue department guidance documents
  - Tax Foundation publications
- This is critical for maintaining accuracy as federal law changes
- If state has static conformity, future federal changes won't automatically apply
- Estimate: 3-4 days

---

### Issue #303: Create State Tax API Documentation

**Description**

Create comprehensive API documentation for the state tax calculation system to help future developers add states or modify logic.

**Acceptance Criteria**

- [ ] Documentation created: `docs/state_tax_api.md`
- [ ] Documentation includes:
  - Architecture overview (entity structure, table organization)
  - How to add a new state (step-by-step guide)
  - Entity structure for state tax
  - Decision table naming conventions
  - Test case format and validation
  - Output structure
  - Multi-state handling
- [ ] Examples for:
  - Adding a flat tax state (with code snippets)
  - Adding a progressive tax state (with bracket logic)
  - Adding state-specific credits
  - Multi-state scenarios
- [ ] Postfix notation examples for common state tax patterns
- [ ] Architecture diagrams
- [ ] Common pitfalls and troubleshooting

**Technical Notes**

- Include code examples in DTRules postfix notation
- Provide template decision tables
- Document lessons learned from implementation
- Should be written after implementing many states (better informed)
- Estimate: 3-4 days

---

### Issue #304: Performance Testing and Optimization

**Description**

Test and optimize performance of state tax calculations, especially for multi-state scenarios.

**Acceptance Criteria**

- [ ] Benchmark suite created in `go/pkg/dtrules/state_tax_bench_test.go`
- [ ] Performance tested for:
  - Single state calculation (baseline)
  - All states calculation (50-state run)
  - Multi-state scenarios (2-4 states)
  - Federal + state calculation (realistic scenario)
- [ ] Performance targets:
  - Single state: < 50ms per calculation
  - Multi-state (3 states): < 200ms
  - Federal + state: < 150ms total
- [ ] Profiling performed with `pprof`:
  - Identify hotspots
  - Identify redundant calculations
  - Identify inefficient table calls
- [ ] Optimization opportunities documented
- [ ] Optimizations implemented if low-hanging fruit found
- [ ] Performance results documented in `docs/PERFORMANCE.md`

**Technical Notes**

- Use Go benchmarking framework
- Profile with `pprof` to identify bottlenecks
- Consider caching intermediate results (federal AGI reused by states)
- Consider parallel state calculation (states are independent)
- Measure memory usage as well as CPU time
- Estimate: 2-3 days

---

## Summary Statistics

**Total Issues**: ~74 issues (revised from 95)
- Foundation: 1 issue (#200) - 1-2 hours
- Flat Tax States: 6 issues (#210-215) - 2-3 days each
- Progressive States: ~55 issues (#220-280) - 3-5 days each
- No-Tax Documentation: 1 issue (#285) - 1-2 days
- Multi-State: 5 issues (#290-294) - 2-7 days each
- Validation/Docs: 5 issues (#300-304) - 2-15 days each

**Estimated Effort**:
- Foundation: 1-2 hours (not weeks!)
- Flat tax states: 12-18 days (6 states × 2-3 days)
- Progressive states: 165-275 days (55 states × 3-5 days)
- No-tax doc: 1-2 days
- Multi-state: 19-25 days
- Validation: 23-33 days
- **Total**: ~220-355 worker-days
- **With 5 workers**: ~44-71 calendar days = **10-16 weeks = 2.5-4 months**

**Revised Timeline (Minimal Foundation)**:
```
Foundation:      1-2 hours (just entity)
States:          10-16 weeks (fully parallel from day 1, 5 workers)
Multi-state:     3-4 weeks (overlaps with states)
Validation:      3-5 weeks
─────────────────────────────────────────────────────
Total:           13-20 weeks = 3-5 months
```

**Key Improvement**: No foundation bottleneck - states start immediately!

**Priority Order**:
1. Foundation (#200) - 1-2 hours
2. High-population states first (CA, NY, TX, FL, IL, PA, OH, GA, NC, MI)
3. Flat tax states (good for learning patterns)
4. Remaining states
5. Multi-state scenarios (can start after ~10 states implemented)
6. Validation and documentation

---

## Dependencies Summary for Orchestrator

```
#200: No dependencies (foundation)

#210-#280: All depend on #200 only (states are independent)

#285: Depends on #200

#290-#294: Depends on #210, #220, #230 (need sample states)

#300: Depends on #210-#280 (all states)
#301: Depends on #210, #220, #230, #260, #240, #270, #213, #247, #214 (top 10 population)
#302: Depends on #210-#280 (all states)
#303: Depends on #210-#280 (all states)
#304: Depends on #210-#280 (all states)
```

---

## How to Use With Orchestrator

1. **Create the Epic Issue** on GitHub with the task list from above
2. **Create Individual Issues** (#200-#304) by copying each template
3. **Mark dependencies** using "(depends on #N)" syntax in epic
4. **Launch orchestrator**:
   ```bash
   cd /home/paul/go/src/github.com/DTRules/DTRules
   orchestrator launch <epic-number> --workers 5
   ```

5. **Monitor progress**:
   - Web dashboard: http://localhost:8123
   - CLI: `orchestrator status`
   - tmux: `tmux attach -t DTRules`

6. **Prioritize high-population states** by editing issues first

---

*Document Version: 2.0 - Minimal Foundation Approach (2026-03-22)*
