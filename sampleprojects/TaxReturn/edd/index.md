# Entity Data Dictionary Index

This folder contains entity definitions organized by functional area.

## Summary

| File | Entities | Attributes |
|------|----------|------------|
| [Core.xlsx](Core.xlsx) | 5 | 467 |
| [Credits.xlsx](Credits.xlsx) | 8 | 93 |
| [Deductions.xlsx](Deductions.xlsx) | 2 | 32 |
| [Forms.xlsx](Forms.xlsx) | 7 | 86 |
| [Income.xlsx](Income.xlsx) | 6 | 126 |
| [Property.xlsx](Property.xlsx) | 6 | 99 |
| **Total** | **34** | **903** |

## Entities by File

### Core.xlsx

| Entity | Attributes | Description |
|--------|------------|-------------|
| constants | 196 | Tax constants and rates for 2025 |
| dependent | 24 | Dependent (child or qualifying relative) |
| job | 52 | Root entity for tax return processing |
| result | 161 | Tax calculation results |
| taxpayer | 34 | Individual taxpayer |

### Credits.xlsx

| Entity | Attributes | Description |
|--------|------------|-------------|
| adoption | 14 | Form 8839 Qualified Adoption Expenses |
| clean_vehicle | 15 | Form 8936 Clean Vehicle Credit |
| credit | 9 | Tax credit |
| elderly_disabled | 13 | Schedule R Credit for Elderly or Disabled |
| energy_improvement | 7 | Residential energy improvement |
| lihtc_recapture | 12 | Form 8611 Recapture of Low-Income Housing Credit |
| mortgage_credit_certificate | 13 | Form 8396 Mortgage Credit Certificate for first-time homebuy... |
| prior_year_amt | 10 | Form 8801 Prior Year Minimum Tax Credit carryforward |

### Deductions.xlsx

| Entity | Attributes | Description |
|--------|------------|-------------|
| deduction | 9 | Itemized deduction |
| noncash_contribution | 23 | Form 8283 Noncash Charitable Contribution |

### Forms.xlsx

| Entity | Attributes | Description |
|--------|------------|-------------|
| child_unearned_income | 12 | Form 8615 Kiddie Tax |
| early_distribution | 12 | Form 5329 Early Distribution from Retirement Account |
| estimated_payment | 9 | Estimated Tax Payment for Form 2210 |
| household_employee | 13 | Schedule H Household Employment |
| ira_account | 14 | IRA account for basis tracking and conversions |
| nol_carryover | 9 | Net Operating Loss Carryover |
| partnership_audit | 17 | Form 8978 Partners Additional Reporting Year Tax (BBA audits... |

### Income.xlsx

| Entity | Attributes | Description |
|--------|------------|-------------|
| farm_income | 40 | Schedule F farm income and expenses |
| foreign_earned_income | 15 | Form 2555 Foreign Earned Income Exclusion |
| foreign_income | 19 | Foreign income for Form 1116 |
| income | 11 | Income source |
| k1_income | 27 | Schedule K-1 income from partnerships, S-corps, trusts |
| unreported_tips | 14 | Form 4137 Social Security and Medicare Tax on Unreported Tip... |

### Property.xlsx

| Entity | Attributes | Description |
|--------|------------|-------------|
| casualty_loss | 14 | Form 4684 Casualty or Theft Loss |
| installment_sale | 15 | Form 6252 Installment Sale |
| like_kind_exchange | 18 | Form 8824 Like-Kind Exchange |
| property | 22 | Real estate property |
| property_sale | 17 | Sale of business property (Form 4797) |
| vehicle | 13 | Business vehicle |

