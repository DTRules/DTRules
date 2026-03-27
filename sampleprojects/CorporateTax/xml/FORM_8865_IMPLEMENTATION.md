# Form 8865 Implementation - Foreign Partnership Reporting

## Overview

This implementation adds Form 8865 (Return of U.S. Persons With Respect to Certain Foreign Partnerships) support to the CorporateTax project.

## Files Created

1. **form_8865_edd.xml** - Entity definition for `foreign_partnership`
2. **form_8865_dt.xml** - Decision tables 14000-14200

## Integration Instructions

### Step 1: Add Entity to CorporateTax_edd_core.xml

Insert the contents of `form_8865_edd.xml` into `/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/CorporateTax/xml/CorporateTax_edd_core.xml`

**Location**: Before the `<entity name="job">` definition (around line 1227)

### Step 2: Add Tables to CorporateTax_dt_core.xml

Insert the contents of `form_8865_dt.xml` into `/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/CorporateTax/xml/CorporateTax_dt_core.xml`

**Location**: At the end of the file, before the closing `</decision_tables>` tag

## Entity: foreign_partnership

The `foreign_partnership` entity tracks all aspects of U.S. person interests in foreign partnerships:

### Key Fields

#### Filing Categories
- `filing_category_1` - U.S. person controlling (>50%)
- `filing_category_2` - Acquisition/disposition of 10%+ interest
- `filing_category_3` - 10%+ owner of controlled foreign partnership
- `filing_category_4` - Reportable events
- `form_8865_required` - Whether filing is required

#### Ownership
- `ownership_percentage_profits` - % by profits interest
- `ownership_percentage_capital` - % by capital interest
- `ownership_percentage_value` - % by value
- `total_us_persons_ownership` - Total U.S. ownership
- `is_controlled_foreign_partnership` - Controlled by U.S. persons (>50%)

#### Schedule K (Partnership Income/Loss)
- `k_ordinary_business_income` - Ordinary business income (loss)
- `k_guaranteed_payments` - Guaranteed payments
- `k_interest_income` - Interest income
- `k_dividends` - Dividend income
- `k_royalties` - Royalty income
- `k_net_short_term_capital_gain` - Short-term capital gain (loss)
- `k_net_long_term_capital_gain` - Long-term capital gain (loss)
- `k_section_179_deduction` - Section 179 deduction
- `k_foreign_taxes_paid` - Foreign taxes paid
- `k_foreign_source_income_general` - Foreign source income - general category
- `k_foreign_source_income_passive` - Foreign source income - passive category

#### Schedule K-1 (Partner's Share)
- `k1_ordinary_business_income` - Partner's share of ordinary income
- `k1_total_income` - Total income allocated to partner
- `k1_distributive_share_pct` - Partner's distributive share %

#### Schedule K-2/K-3 (International Items)
- `k2_foreign_taxes_paid` - Foreign taxes paid (partnership level)
- `k2_foreign_source_income_general` - Foreign source income - general
- `k2_foreign_source_income_passive` - Foreign source income - passive
- `k3_foreign_taxes_paid` - Partner's share of foreign taxes
- `k3_foreign_source_income_general` - Partner's share - general income
- `k3_foreign_source_income_passive` - Partner's share - passive income

#### Distributions
- `distributions_cash` - Cash distributions received
- `distributions_property_fmv` - FMV of property distributions
- `distributions_total` - Total distributions
- `distributions_in_excess_of_basis` - Distributions > basis (taxable)

#### Capital Account
- `capital_account_beginning` - Beginning balance
- `capital_contributions_current_year` - Current year contributions
- `capital_account_share_of_income` - Share of income
- `capital_account_share_of_loss` - Share of loss
- `capital_account_distributions` - Distributions
- `capital_account_ending` - Ending balance

#### Partner Basis (IRC § 705)
- `basis_beginning` - Beginning basis
- `basis_increase_contributions` - Increases from contributions
- `basis_increase_income` - Increases from income share
- `basis_decrease_distributions` - Decreases from distributions
- `basis_decrease_losses` - Decreases from losses
- `basis_ending` - Ending basis
- `basis_limitation_applied` - Whether basis limitation applies
- `suspended_losses` - Losses suspended due to insufficient basis (§ 704(d))

#### Acquisition/Disposition
- `acquisition_date` - Date of acquisition
- `acquisition_cost` - Cost of acquired interest
- `disposition_date` - Date of disposition
- `disposition_amount_realized` - Amount realized
- `disposition_gain_loss` - Gain or loss on disposition

## Decision Tables

### Table 14000: Identify_Foreign_Partnership_Filing_Requirement

**Purpose**: Determines which Form 8865 filing category applies and sets the filing requirement flag.

**Categories**:
- **Category 1**: U.S. person controlling foreign partnership (>50% by profits, capital, or value)
- **Category 2**: U.S. person acquiring or disposing of 10%+ interest
- **Category 3**: U.S. person owning 10%+ of controlled foreign partnership
- **Category 4**: Reportable events (changes in proportional interest, etc.)

**Key Logic**:
1. Check if partnership is foreign (non-U.S. country code)
2. Determine if U.S. person controls partnership (>50%)
3. Check for acquisition/disposition of 10%+ interest
4. Check if partnership is controlled and person owns 10%+
5. Check for reportable events
6. Set appropriate category flags and filing requirement

**References**:
- IRC § 6038(a) - Category 1 and 3 requirements
- IRC § 6046A(b) - Category 2 requirements
- IRC § 6038(a)(5) - Category 4 reportable events

### Table 14100: Calculate_Foreign_Partnership_Income

**Purpose**: Calculates partner's distributive share of partnership income and allocates to appropriate income categories.

**Key Calculations**:
1. **Ordinary Business Income**: Partner's share = Partnership income × Distributive share %
2. **Guaranteed Payments**: Always 100% to receiving partner, treated as ordinary income
3. **Investment Income**: Interest, dividends, royalties flow through with character preserved
4. **Capital Gains**: Short-term and long-term capital gains flow through
5. **Foreign Source Income**: Allocated by category (general, passive) for FTC purposes

**Income Allocation**:
- Ordinary business income → `result.other_income`
- Interest income → `result.interest_income`
- Dividend income → `result.dividends_received`
- Royalty income → `result.gross_royalties`
- Capital gains → `result.capital_gain_net_income`
- Foreign taxes → `result.foreign_taxes_paid`
- Foreign source income → `result.ftc_general_foreign_income` / `result.ftc_passive_foreign_income`

**References**:
- IRC § 702 - Income and credits of partner
- IRC § 704(b) - Partner's distributive share
- IRC § 707(c) - Guaranteed payments

### Table 14200: Complete_Form_8865_Schedules

**Purpose**: Completes Form 8865 schedules K-1, K-2, and K-3, and tracks capital account and partner basis.

**Key Calculations**:

1. **Capital Account** (Form 8865 Schedule K-1 Part II):
   ```
   Ending Capital = Beginning + Contributions + Income - Losses - Distributions
   ```

2. **Partner Basis** (IRC § 705):
   ```
   Ending Basis = Beginning + Contributions + Income - Distributions - Losses
   ```

   **Basis Limitation** (IRC § 704(d)):
   - Partner cannot deduct losses in excess of basis
   - Excess losses suspended and carried forward
   - Applied after distributions

3. **Distributions** (IRC § 731, § 733):
   - Generally non-taxable
   - Cash distributions > basis = capital gain
   - Basis reduced by distributions (not below zero)

4. **Schedule K-2/K-3**:
   - Partnership-level foreign information (K-2)
   - Partner-level share of foreign items (K-3)
   - Required for foreign tax credit calculations

**References**:
- IRC § 705 - Determination of basis
- IRC § 704(d) - Limitation on losses
- IRC § 731 - Distributions
- IRC § 733 - Basis adjustments

## Filing Requirements

### Category 1: Controlling U.S. Person (>50%)

**Who Must File**: U.S. person who controls foreign partnership
- Own >50% by profits, capital, or value
- Direct or indirect ownership

**What to File**:
- Form 8865 with all schedules
- Schedule K-1 (Partner's Share)
- Schedule K-2/K-3 (International Items)
- Balance sheets and income statements

**Penalties** (IRC § 6038(b)):
- $10,000 per annual period for failure to file
- Additional $10,000 per 30-day period after notice (max $50,000)
- Reduction of foreign tax credits

### Category 2: Acquisition/Disposition of 10%+ Interest

**Who Must File**: U.S. person who acquires or disposes of 10%+ interest
- Acquire interest causing 10%+ ownership
- Dispose of interest when 10%+ owned

**What to File**:
- Form 8865 for year of transaction
- Schedule O (Transfer of Property)

**Penalties** (IRC § 6679):
- 10% of value of property contributed
- Basis reduction in contributed property
- Potential gain recognition

### Category 3: 10%+ Owner of Controlled Partnership

**Who Must File**: U.S. person owning 10%+ of controlled foreign partnership
- Partnership controlled by U.S. persons (collectively >50%)
- Person owns 10%+ (individually)

**What to File**:
- Form 8865 annually
- Schedule K-1, K-2, K-3
- Limited financial information

**Penalties**: Same as Category 1

### Category 4: Reportable Events

**Who Must File**: U.S. person experiencing reportable events
- Change in proportional interest
- Certain contributions or acquisitions

**What to File**:
- Form 8865 for year of event
- Limited schedules

## Integration with Foreign Tax Credit

Partnership foreign source income and foreign taxes flow through to partners for FTC purposes:

1. **Schedule K-2/K-3** provides:
   - Foreign source income by category (general, passive)
   - Foreign taxes paid
   - Information for Form 1116/1118

2. **Separate Limitation Baskets**:
   - General category income (`k3_foreign_source_income_general`)
   - Passive category income (`k3_foreign_source_income_passive`)

3. **Foreign Tax Credit Calculation**:
   - Partner's share of foreign taxes → `result.foreign_taxes_paid`
   - Foreign source income → `result.ftc_general_foreign_income` / `result.ftc_passive_foreign_income`
   - Used in Table 9100 (Calculate_Foreign_Tax_Credit)

## Testing Recommendations

Create test cases for:

1. **Category 1 Filer**: U.S. corporation owning 60% of foreign partnership
2. **Category 2 Filer**: Acquisition of 15% interest mid-year
3. **Category 3 Filer**: 12% owner of partnership controlled by U.S. persons
4. **Basis Limitation**: Partner with insufficient basis to deduct losses
5. **Distributions Exceeding Basis**: Cash distribution > partner basis (capital gain)
6. **Foreign Tax Credit**: Partnership with foreign source income and foreign taxes

## References

### Internal Revenue Code
- IRC § 702 - Income and credits of partner
- IRC § 704 - Partner's distributive share
- IRC § 704(d) - Limitation on losses
- IRC § 705 - Basis of partner's interest
- IRC § 707(c) - Guaranteed payments
- IRC § 731 - Distributions
- IRC § 733 - Basis adjustments for distributions
- IRC § 6038 - Information returns - foreign partnerships (Categories 1, 3, 4)
- IRC § 6038B - Transfer of property to foreign partnership
- IRC § 6046A - Returns as to interests in foreign partnerships (Category 2)
- IRC § 6679 - Penalties for Category 2 failures

### Forms and Instructions
- Form 8865 - Return of U.S. Persons With Respect to Certain Foreign Partnerships
- Form 8865 Instructions
- Schedule K-1 (Form 8865) - Partner's Share of Income, Deductions, Credits, etc.
- Schedule K-2 (Form 8865) - Partners' Distributive Share Items - International
- Schedule K-3 (Form 8865) - Partner's Share of Income, Deductions, Credits, etc. - International

### Treasury Regulations
- Treas. Reg. § 1.704-1(b) - Partner's distributive share
- Treas. Reg. § 1.6038-3 - Information returns
