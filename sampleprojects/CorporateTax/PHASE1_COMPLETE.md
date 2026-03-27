# Phase 1 Implementation Complete

**Date**: 2026-03-23
**Issues Completed**: #316, #317, #318, #319, #320, #321, #322

## Summary

Phase 1 of the Corporate Tax implementation is complete. This includes all core federal tax calculation logic for Form 1120.

## Components Implemented

### 1. Entity Data Dictionary (Issue #316)

**File**: `xml/CorporateTax_edd_core.xml`

Entities defined:
- **corporation** - 8 fields (EIN, name, incorporation date, business code, accounting method, tax year, total assets, gross receipts)
- **revenue** - 13 fields (Form 1120 lines 1-11: gross receipts, COGS, dividends, interest, rents, royalties, capital gains, other income)
- **expense** - 4 fields (category, amount, deductible flag, limitation applied)
- **result** - 23 fields (all deductions lines 12-27, tax calculation lines 28-35, audit trail)
- **job** - 3 fields (test metadata)

**Total**: 51 fields covering all Form 1120 page 1 requirements

### 2. XML Mapping Layer (Issue #317)

**File**: `xml/CorporateTax_map.xml`

Complete bidirectional mapping (XMLtoEDD and EDDtoXML) for all entities:
- Job mappings (test case metadata)
- Corporation mappings (company information)
- Revenue mappings (Form 1120 lines 1-11)
- Expense mappings (business deductions)
- Result mappings (Form 1120 lines 12-35, audit trail)

### 3. Income Calculation Tables (Issue #318)

**Tables Implemented**:
- **Table 2000**: Calculate_Net_Receipts (Form 1120 line 1c = 1a - 1b)
- **Table 2100**: Calculate_Gross_Profit (Form 1120 line 3 = 1c - 2)
- **Table 2600**: Calculate_Total_Income (Form 1120 line 11)

All tables include:
- Complete conditions and rules
- Policy statements with IRS form references
- Proper handling of edge cases (zero income, negative values)

### 4. Deduction Calculation Tables (Issue #319)

**Tables Implemented**:
- **Table 3000**: Calculate_Compensation_Deduction (Form 1120 line 12)
- **Table 3200**: Calculate_Interest_Deduction with IRC 163(j) limitation (30% ATI cap)
- **Table 3500**: Calculate_Charitable_Contributions with 10% limitation (IRC 170(b)(2))
- **Table 3600**: Calculate_Total_Deductions (Form 1120 line 27)

Key features:
- IRC Section 163(j) business interest limitation
- IRC Section 170(b)(2) charitable contribution cap
- Full policy documentation citing IRS forms and tax code sections

### 5. Tax Calculation Tables (Issue #320)

**Tables Implemented**:
- **Table 4000**: Calculate_Taxable_Income (Form 1120 line 28 = 11 - 27)
- **Table 4100**: Apply_Corporate_Tax_Rate (21% flat rate per IRC Section 11)
- **Table 4300**: Calculate_Refund_Or_Owed

Key features:
- Flat 21% corporate tax rate (TCJA 2017)
- Net Operating Loss (NOL) placeholder for Phase 4
- No Alternative Minimum Tax (repealed for corporations)
- Complete payment and credit handling

### 6. Orchestration Table (Issue #321-#322)

**Table Implemented**:
- **Table 1000**: Compute_Corporate_Tax_Return

Execution flow:
1. Income calculation (calls tables 2000, 2100, 2600)
2. Deduction calculation (calls tables 3000, 3200, 3500, 3600)
3. Tax calculation (calls tables 4000, 4100, 4300)

## Test Cases

**File**: `testfiles/TestScenarios/SimpleManufacturing.xml`

Initial test case:
- Simple manufacturing corporation
- $5M gross receipts, $3M COGS
- Standard business deductions
- Expected taxable income: $2M
- Expected federal tax: $420K (21% of $2M)

## Build Process

**Merge Script**: `scripts/merge-states.sh`

The merge script successfully:
- Combines core federal tables with state-specific files
- Produces final `CorporateTax_edd.xml` and `CorporateTax_dt.xml`
- Excludes TEMPLATE files
- Reports merge statistics

Usage:
```bash
cd sampleprojects/CorporateTax
./scripts/merge-states.sh
```

## IRS Forms Covered

Phase 1 implements core calculations from:
- **Form 1120** - U.S. Corporation Income Tax Return (lines 1-35)
- **Schedule A** - Cost of Goods Sold (referenced in table 2100)
- **Schedule E** - Compensation of Officers (referenced in table 3000)
- **Schedule J** - Tax Computation (table 4100)
- **Form 8990** - Limitation on Business Interest Expense (table 3200)

## Tax Code References

Decision tables cite:
- **IRC Section 11(b)** - Corporate tax rate (21%)
- **IRC Section 162(a)(1)** - Ordinary and necessary business expenses
- **IRC Section 163(j)** - Business interest expense limitation
- **IRC Section 170(b)(2)** - Charitable contribution limitation (10% of taxable income)
- **IRC Section 172** - Net Operating Loss (NOL) - placeholder for Phase 4

## Table Numbering

Federal tables use ranges 1000-9999:
- **1000-1999**: Orchestration (table 1000 implemented)
- **2000-2999**: Income calculation (tables 2000, 2100, 2600 implemented)
- **3000-3999**: Deduction calculation (tables 3000, 3200, 3500, 3600 implemented)
- **4000-4999**: Tax calculation (tables 4000, 4100, 4300 implemented)
- **5000-5999**: Credits (Phase 2 - not yet implemented)
- **6000-6999**: Depreciation/Assets (Phase 2 - not yet implemented)
- **7000-7999**: Apportionment (Phase 3 - not yet implemented)
- **8000-8999**: Schedules M-1, M-2, L (Phase 4 - not yet implemented)
- **9000-9999**: Advanced (Phase 4 - not yet implemented)

State tables will use ranges 50000-99999 (100 tables per state).

## Known Limitations (Phase 1)

The following are documented but not yet implemented:

1. **Depreciation**: Depreciation field exists but no MACRS calculation (Phase 2)
2. **Credits**: Total credits field exists but no credit calculation tables (Phase 2)
3. **Interest carryforward**: IRC 163(j) excess interest not tracked for carryforward
4. **Charitable carryforward**: Excess contributions not tracked for 5-year carryforward
5. **NOL**: Net Operating Loss identified but not calculated or carried forward (Phase 4)
6. **State taxes**: No state apportionment or state tax calculation (Phase 3)
7. **Schedules M-1, M-2, L**: Book/tax reconciliation not implemented (Phase 4)
8. **Foreign tax credit**: Not implemented (Phase 4)

These limitations are acceptable for Phase 1 and are tracked in subsequent GitHub issues.

## Next Steps

### Phase 2 - Depreciation & Credits (Issues #323-#328)
- Table 6000-6999: MACRS depreciation
- Table 5000-5999: R&D credit, Work Opportunity Tax Credit, General Business Credit

### Phase 3 - State Tax (Issues #329-#332)
- Table 7000-7999: Apportionment calculation
- State files: 51 jurisdictions (50 states + DC)
- Each state gets 100-table range (50000-99999)

### Phase 4 - Advanced (Issues #333-#339)
- Schedules M-1, M-2, L (book/tax reconciliation)
- Net Operating Loss (NOL) carryforward
- Foreign tax credit
- S Corporation support

## Files Created

Core XML files:
- `xml/CorporateTax_edd_core.xml` (6.6 KB)
- `xml/CorporateTax_dt_core.xml` (22 KB)
- `xml/CorporateTax_map.xml` (14 KB)

Merged files (generated by script):
- `xml/CorporateTax_edd.xml` (6.4 KB)
- `xml/CorporateTax_dt.xml` (22 KB)

Supporting files:
- `scripts/merge-states.sh` (3.3 KB)
- `testfiles/TestScenarios/SimpleManufacturing.xml` (2.5 KB)

Templates (from Issue #315):
- `xml/states/TEMPLATE_corp_edd.xml`
- `xml/states/TEMPLATE_corp_dt.xml`

Documentation (from Issue #315):
- `README.md` (comprehensive project documentation)

## Validation

Phase 1 implementation includes:
- ✅ Complete entity definitions with 51+ fields
- ✅ Bidirectional XML mapping
- ✅ 10 decision tables (3 income, 4 deduction, 3 tax, 1 orchestration)
- ✅ Policy statements citing IRS forms and IRC sections
- ✅ Merge script tested and working
- ✅ Initial test case created

Ready for Go implementation and testing.

---

**Implementation Time**: ~4 hours
**Lines of Code**: ~900 (XML)
**Decision Tables**: 10
**Test Cases**: 1 (more to be added)
