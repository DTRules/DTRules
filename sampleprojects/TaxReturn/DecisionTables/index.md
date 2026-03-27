# Decision Tables Index

This folder contains decision tables organized by functional area.

## Summary

| File | Tables | Description |
|------|--------|-------------|
| [Deductions.xlsx](Deductions.xlsx) | 12 | Deduction calculations |
| [FinalBalance.xlsx](FinalBalance.xlsx) | 1 |  |
| [Helpers.xlsx](Helpers.xlsx) | 5 | Helper and utility tables |
| [Income.xlsx](Income.xlsx) | 9 | Income processing tables |
| [MajorCredits.xlsx](MajorCredits.xlsx) | 8 |  |
| [Other.xlsx](Other.xlsx) | 2 | Uncategorized tables |
| [SpecialForms.xlsx](SpecialForms.xlsx) | 41 | Special IRS forms (K-1, foreign, etc.) |
| [TaxCredits.xlsx](TaxCredits.xlsx) | 12 |  |
| [TaxableIncome.xlsx](TaxableIncome.xlsx) | 2 |  |
| [Validation.xlsx](Validation.xlsx) | 5 | Test validation tables |
| **Total** | **97** | |

## Tables by File

### Deductions.xlsx

| Table | Description |
|-------|-------------|
| Calculate_Charitable_Deduction | Calculates charitable contribution deduction per Schedule A Lines 11-14, Publica... |
| Calculate_Deductions | Determines whether to use standard or itemized deduction per Form 1040 Line 12, ... |
| Calculate_Itemized_Deductions | Calculates total itemized deductions per Schedule A including SALT (capped at $1... |
| Calculate_Mortgage_Interest | Calculates mortgage interest deduction for qualified residences per Schedule A L... |
| Calculate_Noncash_Charitable | Calculates noncash charitable contribution deductions per Form 8283, Publication... |
| Calculate_SALT_Deduction | Calculates State and Local Tax (SALT) deduction including state income tax, real... |
| Calculate_Standard_Deduction | Determines standard deduction amount based on filing status per IRC Section 63(c... |
| Filter_Charity_Deduction | Helper table that processes deduction if category is charity. |
| Filter_Primary_Residence | Helper table that processes property if it is a primary residence. |
| Process_Noncash_Contribution | Processes individual noncash charitable contribution. Applies appropriate valuat... |
| Sum_Primary_Residence_Tax | Helper table for SALT calculation. Adds property tax to running total if propert... |
| Sum_SALT_Deduction | Helper table for SALT calculation. Adds state and local tax deductions to runnin... |

### FinalBalance.xlsx

| Table | Description |
|-------|-------------|
| Calculate_Final_Balance | Calculates final tax balance - total tax minus credits and payments to determine... |

### Helpers.xlsx

| Table | Description |
|-------|-------------|
| Calculate_Mortgage_Interest_For_Property | Processes mortgage interest for a single property. Called by filter table. |
| Count_EITC_Qualifying_Child | Helper table that increments counter if dependent is qualifying child for EITC. |
| Filter_Rental_Property | Helper table that processes property if it is a rental. |
| Filter_Self_Employed_Taxpayer | Helper table that processes taxpayer if self-employed. |
| Filter_Taxpayer_Vehicle | Helper table that processes vehicle if it belongs to current taxpayer. |

### Income.xlsx

| Table | Description |
|-------|-------------|
| Calculate_AGI_Adjustments | Calculates above-the-line deductions (adjustments to income) per Schedule 1 Part... |
| Calculate_Gross_Income | Calculates total gross income from all sources including W-2 wages, self-employm... |
| Calculate_SE_Tax | Calculates self-employment tax per Schedule SE and IRC 1401. Called after net pr... |
| Calculate_Vehicle_Deduction | Calculates vehicle expense deduction using either standard mileage rate or actua... |
| Compute_Tax_Return | Entry point for tax return calculation. Orchestrates the complete tax calculatio... |
| Process_Other_Income | Processes other income sources (pension, interest, dividends, etc.) from income ... |
| Process_Rental_Income | Processes rental property income and expenses per Schedule E Part I, Publication... |
| Process_Self_Employment | Processes self-employment income per Schedule C, calculates vehicle deductions p... |
| Process_W2_Income | Processes W-2 wage income for all taxpayers. Records wages per Form 1040 Line 1a... |

### MajorCredits.xlsx

| Table | Description |
|-------|-------------|
| Calculate_AMT | Calculate Alternative Minimum Tax per Form 6251, IRC 55-59. AMTI = taxable incom... |
| Calculate_CTC_Phase_Out | Applies CTC phase-out based on filing status and AGI. Phase-out threshold is $40... |
| Calculate_Child_Tax_Credit | Calculates Child Tax Credit (CTC) and Other Dependent Credit (ODC) per Schedule ... |
| Calculate_Credits | Calculates tax credits including Child Tax Credit (CTC), Other Dependent Credit ... |
| Calculate_EITC | Calculates Earned Income Tax Credit per Schedule EIC, Publication 596, IRC 32. E... |
| Calculate_Education_Credit_Per_Dependent | Calculates education credit for a single dependent. Determines AOTC vs LLC eligi... |
| Calculate_Education_Credits | Calculates education credits (AOTC and LLC) per Form 8863, Publication 970, IRC ... |
| Calculate_Premium_Tax_Credit | Calculate Premium Tax Credit per Form 8962, IRC 36B |

### Other.xlsx

| Table | Description |
|-------|-------------|
| Calculate_Educator_Expenses |  |
| Calculate_Medical_Expense_Deduction |  |

### SpecialForms.xlsx

| Table | Description |
|-------|-------------|
| Apply_CG_Brackets_MFJ | Applies MFJ capital gains brackets (0%/15%/20%) based on taxable income level. |
| Apply_CG_Brackets_Single | Applies Single capital gains brackets (0%/15%/20%) based on taxable income level... |
| Calculate_Additional_Medicare_Tax | Calculates Additional Medicare Tax (0.9%) per Form 8959, IRC 3101(b)(2). Applies... |
| Calculate_Capital_Gains_Tax | Calculates capital gains tax using preferential rates (0%/15%/20%) per Schedule ... |
| Calculate_Elderly_Disabled_Credit | Calculate Schedule R Credit for Elderly or Disabled per IRC 22. 15% credit on ba... |
| Calculate_Estimated_Tax_Penalty | Calculate Form 2210 underpayment penalty for estimated taxes per IRC 6654. Check... |
| Calculate_Farm_SE_Tax | Calculate self-employment tax on farm income. Farm net profit is subject to SE t... |
| Calculate_Foreign_Tax_Credit | Calculate Foreign Tax Credit with IRC 904 limitation. Credit cannot exceed US ta... |
| Calculate_HSA_Deduction | Calculates HSA deduction per Form 8889, Publication 969, IRC 223. Limits based o... |
| Calculate_IRA_Deduction | Calculates Traditional IRA deduction per Publication 590-A, IRC 219. Applies pha... |
| Calculate_K1_SE_Tax | Calculate self-employment tax on K-1 SE earnings (Box 14 and guaranteed payments... |
| Calculate_LIHTC_Recapture | Calculate Recapture of Low-Income Housing Credit per Form 8611 and IRC 42(j). Re... |
| Calculate_Mortgage_Interest_Credit | Calculate Mortgage Interest Credit for first-time homebuyers with Mortgage Credi... |
| Calculate_NIIT | Calculates Net Investment Income Tax (3.8%) per Form 8960, IRC 1411. NIIT applie... |
| Calculate_OBBBA_Deductions | Calculates OBBBA 2025 deductions: No Tax on Tips, No Tax on Overtime, and Senior... |
| Calculate_Overtime_Deduction | No Tax on Overtime deduction per OBBBA 2025. Maximum $12,500 (Single/HOH) or $25... |
| Calculate_Partnership_Audit_Tax | Calculate Partner's Additional Reporting Year Tax per Form 8978 and IRC 6226. Fo... |
| Calculate_Prior_Year_AMT_Credit | Calculate Prior Year Minimum Tax Credit per Form 8801 and IRC 53. Credits carryf... |
| Calculate_Roth_Conversion_Tax | Calculate taxable portion of Roth conversions using the pro-rata rule. When trad... |
| Calculate_SS_Taxability | Calculates taxable portion of Social Security benefits per Publication 915. Uses... |
| Calculate_Section_1231 | Determine treatment of net Section 1231 gain or loss. Net gains are treated as l... |
| Calculate_Senior_Deduction | Senior Additional Deduction per OBBBA 2025. $6,000 per senior (age 65+). Phases ... |
| Calculate_Student_Loan_Deduction | Calculates student loan interest deduction per Publication 970, IRC 221. Maximum... |
| Calculate_Tips_Deduction | No Tax on Tips deduction per OBBBA 2025. Maximum deduction is $25,000. Phases ou... |
| Calculate_Unreported_Tips_Tax | Calculate Social Security and Medicare Tax on Unreported Tip Income per Form 413... |
| Process_Adoption_Credit | Process Form 8839 Adoption Credit per IRC 23. Up to $16,810 per child for qualif... |
| Process_Capital_Gains | Processes capital gains and qualified dividends from income entities. Categorize... |
| Process_Casualty_Losses | Process Form 4684 Casualty and Theft Losses per IRC 165. Personal casualty losse... |
| Process_Clean_Vehicle_Credit | Process Form 8936 Clean Vehicle Credit per IRC 30D. Up to $7,500 for new EVs mee... |
| Process_Early_Distributions | Process Form 5329 Early Distribution penalties. 10% penalty on early withdrawals... |
| Process_Farm_Income | Process Schedule F farm income and expenses. Calculates net farm profit/loss for... |
| Process_Foreign_Earned_Income | Process Form 2555 Foreign Earned Income Exclusion. Allows US citizens/residents ... |
| Process_Foreign_Income | Process foreign income for Form 1116 Foreign Tax Credit. Categorizes income by t... |
| Process_Household_Employment | Process Schedule H Household Employment Taxes. FICA and FUTA taxes for household... |
| Process_IRA_Accounts | Process IRA accounts for Form 8606. Tracks non-deductible contributions (basis),... |
| Process_Installment_Sales | Process Form 6252 Installment Sale Income. Allows gain recognition to be spread ... |
| Process_K1_Income | Process Schedule K-1 income from partnerships, S-corps, and trusts. Categorizes ... |
| Process_Kiddie_Tax | Process Form 8615 Kiddie Tax. Child unearned income over threshold taxed at pare... |
| Process_Like_Kind_Exchanges | Process Form 8824 Like-Kind Exchanges (Section 1031). Allows deferral of gain on... |
| Process_NOL_Carryover | Process Net Operating Loss carryovers per IRC 172. NOLs limited to 80% of taxabl... |
| Process_Property_Sales | Process Form 4797 sales of business property. Calculates gain/loss, depreciation... |

### TaxCredits.xlsx

| Table | Description |
|-------|-------------|
| Apply_Tax_Brackets | Applies marginal tax brackets to calculate regular tax per Form 1040 Line 16, IR... |
| Apply_Tax_Brackets_MFJ | Calculates regular tax using MFJ brackets. Each column represents a bracket rang... |
| Apply_Tax_Brackets_Single | Calculates regular tax using Single brackets. Each column represents a bracket r... |
| Calculate_CDCC | Calculate Child and Dependent Care Credit per Form 2441, IRC 21 |
| Calculate_Energy_Credit_Per_Improvement | Calculate credit for individual energy improvement per IRC 25C/25D |
| Calculate_Energy_Credits | Residential Energy Credits per Form 5695, IRC 25C/25D |
| Calculate_Passive_Activity_Loss | Passive Activity Loss Rules per Form 8582, IRC 469 |
| Calculate_Savers_Credit | Saver's Credit (Form 8880, IRC 25B) - Retirement Savings Contributions Credit. R... |
| Calculate_Tax_Liability | Calculates total tax liability including regular tax (using tax brackets per IRC... |
| Count_CDCC_Qualifying_Person | Count qualifying persons and care expenses for CDCC per IRC 21 |
| Sum_Rental_Loss | Sum rental losses for PAL calculation per IRC 469 |
| Sum_Savers_Contribution | Sum eligible retirement contributions for Saver's Credit per IRC 25B |

### TaxableIncome.xlsx

| Table | Description |
|-------|-------------|
| Calculate_QBI_Deduction | Calculates Qualified Business Income deduction per Form 8995/8995-A, Publication... |
| Calculate_Taxable_Income | Calculates taxable income per Form 1040 Line 15, IRC Section 63. Subtracts deduc... |

### Validation.xlsx

| Table | Description |
|-------|-------------|
| Validate_AGI | Validates AGI against expected value. Sets validation_passed to false on mismatc... |
| Validate_Results | Validates calculated results against expected values from test case. Logs any di... |
| Validate_Summary | Logs overall validation summary. |
| Validate_Taxable_Income | Validates taxable income against expected value. |
| Validate_Total_Tax | Validates total tax against expected value. |

