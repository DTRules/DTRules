# TaxReturn Implementation Gaps

This document details the tax forms and situations not yet supported by the TaxReturn decision tables.

## Current Coverage

The TaxReturn project fully supports:
- **Income:** W-2 wages, self-employment (Schedule C/SE), rental (Schedule E), capital gains (Schedule D), Social Security, interest, dividends, other income
- **Deductions:** Standard, itemized (SALT, mortgage interest, charitable, medical), IRA, HSA, student loan, educator expenses, QBI
- **Credits:** CTC, ODC, EITC, education (AOTC/LLC), CDCC, Saver's, energy (25C/25D), PTC
- **Taxes:** Regular tax, AMT, capital gains tax, NIIT, Additional Medicare Tax, SE tax
- **Special:** OBBBA 2025 provisions (tips, overtime, senior deductions)

---

## Gap 1: Schedule K-1 Income

### What It Is
Schedule K-1 reports a partner's, shareholder's, or beneficiary's share of income, deductions, and credits from:
- **Form 1065 K-1:** Partnerships
- **Form 1120-S K-1:** S Corporations
- **Form 1041 K-1:** Trusts and Estates

### Who Needs It
- Partners in LLCs, LLPs, general partnerships
- S corporation shareholders
- Beneficiaries of trusts or estates

### IRS Forms
- Schedule K-1 (Form 1065)
- Schedule K-1 (Form 1120-S)
- Schedule K-1 (Form 1041)
- Schedule E Part II (reporting)

### K-1 Box Categories

| Box | Description | Where Reported |
|-----|-------------|----------------|
| 1 | Ordinary business income/loss | Schedule E |
| 2 | Net rental real estate income/loss | Schedule E |
| 3 | Other net rental income/loss | Schedule E |
| 4a | Guaranteed payments for services | Schedule E, SE tax |
| 4b | Guaranteed payments for capital | Schedule E |
| 5 | Interest income | Schedule B |
| 6a | Ordinary dividends | Schedule B |
| 6b | Qualified dividends | Schedule B, preferential rate |
| 7 | Royalties | Schedule E |
| 8 | Net short-term capital gain/loss | Schedule D |
| 9a | Net long-term capital gain/loss | Schedule D |
| 10 | Net section 1231 gain/loss | Form 4797 |
| 11 | Other income | Various |
| 12 | Section 179 deduction | Form 4562 |
| 13 | Other deductions | Schedule A or E |
| 14 | Self-employment earnings | Schedule SE |
| 15 | Credits | Various |
| 16 | Foreign transactions | Form 1116 |
| 17 | AMT items | Form 6251 |
| 20 | Section 199A QBI information | Form 8995 |

### Implementation Required

**Entity: `k1_income`**
```xml
<entity name='k1_income'>
  <field name='source_type' type='string' comment='partnership, s_corp, trust'/>
  <field name='ein' type='string' comment='Entity EIN'/>
  <field name='entity_name' type='string' comment='Partnership/S-corp/Trust name'/>
  <field name='ordinary_income' type='double' comment='Box 1'/>
  <field name='rental_income' type='double' comment='Box 2'/>
  <field name='guaranteed_payments' type='double' comment='Box 4a'/>
  <field name='interest_income' type='double' comment='Box 5'/>
  <field name='ordinary_dividends' type='double' comment='Box 6a'/>
  <field name='qualified_dividends' type='double' comment='Box 6b'/>
  <field name='short_term_gain' type='double' comment='Box 8'/>
  <field name='long_term_gain' type='double' comment='Box 9a'/>
  <field name='section_1231_gain' type='double' comment='Box 10'/>
  <field name='section_179' type='double' comment='Box 12'/>
  <field name='se_earnings' type='double' comment='Box 14'/>
  <field name='qbi_income' type='double' comment='Box 20 Code Z'/>
  <field name='w2_wages' type='double' comment='Box 20 for QBI'/>
  <field name='ubia' type='double' comment='Box 20 UBIA for QBI'/>
</entity>
```

**Decision Tables:**
- `Process_K1_Income` - Categorize and sum K-1 amounts by type
- `Calculate_K1_SE_Tax` - SE tax on guaranteed payments and partnership SE earnings

---

## Gap 2: Form 8606 - Roth Conversions

### What It Is
Form 8606 tracks:
- Non-deductible Traditional IRA contributions (basis)
- Taxable portion of distributions from Traditional IRAs with basis
- Roth conversions and the taxable amount

### Who Needs It
- Anyone who made non-deductible IRA contributions
- Anyone converting Traditional IRA to Roth IRA
- Anyone with IRA basis taking distributions

### The Pro-Rata Rule
When you have both deductible and non-deductible contributions in Traditional IRAs, you cannot selectively convert just the non-deductible portion. The taxable percentage is:

```
Taxable % = (Total IRA Balance - Total Basis) / Total IRA Balance
```

### Example
- Total Traditional IRA balance: $100,000
- Non-deductible basis: $20,000
- Converting: $50,000 to Roth

Taxable amount = $50,000 × ($100,000 - $20,000) / $100,000 = $40,000

### Implementation Required

**Entity: `ira_account`**
```xml
<entity name='ira_account'>
  <field name='account_type' type='string' comment='traditional, roth, sep, simple'/>
  <field name='year_end_balance' type='double' comment='12/31 balance'/>
  <field name='nondeductible_basis' type='double' comment='Cumulative basis'/>
  <field name='current_year_contribution' type='double'/>
  <field name='conversion_amount' type='double' comment='Amount converted to Roth'/>
  <field name='distribution_amount' type='double' comment='Non-conversion distributions'/>
</entity>
```

**Decision Tables:**
- `Calculate_IRA_Basis` - Track cumulative non-deductible contributions
- `Calculate_Conversion_Tax` - Apply pro-rata rule to conversions
- `Process_IRA_Distribution` - Calculate taxable portion of distributions

---

## Gap 3: Schedule F - Farm Income

### What It Is
Schedule F reports profit or loss from farming operations.

### Who Needs It
- Farmers (crops, livestock, dairy, poultry, fish, fruit, etc.)
- Agricultural partnerships (flows to K-1)

### Key Differences from Schedule C
- Farm income averaging (Schedule J) available
- Different depreciation rules (Section 179, bonus)
- Conservation Reserve Program payments
- Commodity Credit Corporation loans
- Crop insurance proceeds (can elect deferral)

### Implementation Required

**Entity: `farm_income`**
```xml
<entity name='farm_income'>
  <field name='sales_livestock' type='double'/>
  <field name='sales_crops' type='double'/>
  <field name='cooperative_distributions' type='double'/>
  <field name='agricultural_payments' type='double'/>
  <field name='ccc_loans' type='double'/>
  <field name='crop_insurance' type='double'/>
  <field name='custom_hire' type='double'/>
  <field name='other_income' type='double'/>
  <!-- Expenses -->
  <field name='car_truck' type='double'/>
  <field name='chemicals' type='double'/>
  <field name='conservation' type='double'/>
  <field name='depreciation' type='double'/>
  <field name='feed' type='double'/>
  <field name='fertilizers' type='double'/>
  <field name='fuel' type='double'/>
  <field name='insurance' type='double'/>
  <field name='interest' type='double'/>
  <field name='labor' type='double'/>
  <field name='rent_lease' type='double'/>
  <field name='repairs' type='double'/>
  <field name='seeds_plants' type='double'/>
  <field name='supplies' type='double'/>
  <field name='taxes' type='double'/>
  <field name='utilities' type='double'/>
  <field name='veterinary' type='double'/>
</entity>
```

**Decision Tables:**
- `Process_Farm_Income` - Calculate Schedule F profit/loss
- `Calculate_Farm_SE_Tax` - SE tax on farm income

---

## Gap 4: Form 4797 - Business Property Sales

### What It Is
Form 4797 reports sales of business property, including:
- Section 1231 property (depreciable business property held > 1 year)
- Depreciation recapture (Section 1245 and 1250)
- Involuntary conversions

### Who Needs It
- Business owners selling equipment, vehicles, machinery
- Real estate investors selling rental property
- Anyone with Section 1231 gains/losses

### Section 1231 Rules
- Net Section 1231 gains: Treated as long-term capital gains (preferential rates)
- Net Section 1231 losses: Treated as ordinary losses (full deduction)
- 5-year lookback: If ordinary losses in prior 5 years, gains recaptured as ordinary

### Depreciation Recapture
- **Section 1245:** All depreciation on personal property recaptured as ordinary income
- **Section 1250:** Excess depreciation on real property recaptured as ordinary income (rare after 1986)
- **Unrecaptured Section 1250:** Depreciation on real property taxed at max 25%

### Implementation Required

**Entity: `property_sale`**
```xml
<entity name='property_sale'>
  <field name='description' type='string'/>
  <field name='date_acquired' type='date'/>
  <field name='date_sold' type='date'/>
  <field name='sales_price' type='double'/>
  <field name='cost_basis' type='double'/>
  <field name='depreciation_allowed' type='double'/>
  <field name='property_type' type='string' comment='1245, 1250, 1231'/>
  <field name='is_real_property' type='boolean'/>
</entity>
```

**Decision Tables:**
- `Calculate_Section_1231` - Determine 1231 gain/loss treatment
- `Calculate_Depreciation_Recapture` - Compute ordinary income from recapture
- `Calculate_Unrecaptured_1250` - Compute 25% gain on real property

---

## Gap 5: Form 1116 - Foreign Tax Credit

### What It Is
Form 1116 calculates the credit for taxes paid to foreign countries.

### Who Needs It
- Anyone with foreign income (wages, investments, business)
- Anyone who paid foreign taxes (directly or through mutual funds)

### Credit Limitation
The credit is limited to the US tax attributable to foreign source income:

```
Credit Limit = US Tax × (Foreign Source Taxable Income / Total Taxable Income)
```

### Categories of Income
- Passive (dividends, interest, royalties, rents)
- General (wages, business income)
- Section 901(j) (sanctioned countries - no credit)

### Implementation Required

**Entity: `foreign_income`**
```xml
<entity name='foreign_income'>
  <field name='country' type='string'/>
  <field name='category' type='string' comment='passive, general'/>
  <field name='gross_income' type='double'/>
  <field name='deductions' type='double'/>
  <field name='foreign_taxes_paid' type='double'/>
  <field name='foreign_taxes_accrued' type='double'/>
</entity>
```

**Decision Tables:**
- `Calculate_Foreign_Source_Income` - Sum by category
- `Calculate_FTC_Limitation` - Compute credit limit
- `Apply_Foreign_Tax_Credit` - Lesser of taxes paid or limit

---

## Implementation Priority

| Form | Priority | Complexity | Users Affected |
|------|----------|------------|----------------|
| Schedule K-1 | High | High | Business owners, investors |
| Form 1116 | Medium | Medium | International investors |
| Form 8606 | Medium | Medium | IRA holders |
| Form 4797 | Low | High | Business property sellers |
| Schedule F | Low | Medium | Farmers |

## Testing Strategy

Each implementation should include test cases for:
1. Basic scenarios (single K-1, simple conversion, etc.)
2. Multiple sources (3+ K-1s, multiple foreign countries)
3. Edge cases (losses, phase-outs, carryovers)
4. Integration with existing tables (QBI, capital gains, SE tax)
