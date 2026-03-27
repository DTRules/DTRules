# DTRules Excel Exports

This directory contains Excel (.xlsx) exports of the DTRules decision tables and entity definitions for easier viewing and documentation.

## Files

- **TaxReturn_edd.xlsx** - Entity Data Dictionary with all 17 entities and their fields
- **001-083_*.xlsx** - Individual decision table files (83 tables total)

## Benefits of Excel Format

✅ **Easy viewing** - Open in Excel, LibreOffice, or Google Sheets
✅ **Search & filter** - Use Excel's built-in search to find specific rules
✅ **Print documentation** - Generate PDF reports for stakeholders
✅ **Share with non-technical users** - Business analysts can review rules
✅ **Track changes** - Compare versions using Excel's diff tools

## Usage

### View Decision Tables
```bash
# Open specific table
open excel/001_Compute_Tax_Return.xlsx

# Open EDD
open excel/TaxReturn_edd.xlsx
```

### Regenerate Excel Files

Excel files are automatically generated when you run:
```bash
./scripts/merge-states.sh
```

Or manually:
```bash
./scripts/extract-to-excel.sh
```

## File Naming Convention

Decision table files are numbered sequentially:
- `001_Compute_Tax_Return.xlsx` - Main entry point
- `002_Calculate_Gross_Income.xlsx` - Income calculation
- `003_Process_W2_Income.xlsx` - W-2 processing
- etc.

The numbers correspond to the order of tables in `TaxReturn_dt.xml`.

## Entity Data Dictionary (EDD) Structure

The **TaxReturn_edd.xlsx** file contains separate sheets for each entity:

- **job** - Main processing job entity
- **taxpayer** - Individual taxpayer information
- **dependent** - Dependent/child information
- **property** - Real estate and property
- **vehicle** - Business vehicle information
- **deduction** - Itemized deductions
- **income** - Income sources
- **energy_improvement** - Energy credits (Form 5695)
- **k1_income** - Partnership/S-corp income (Schedule K-1)
- **ira_account** - IRA accounts (Form 8606)
- **farm_income** - Farm income (Schedule F)
- **property_sale** - Business property sales (Form 4797)
- **foreign_income** - Foreign income (Form 1116)
- **household_employee** - Household employment (Schedule H)
- **result** - Tax calculation results
- **constants** - System constants
- **credit** - Tax credit information

Each sheet shows:
- Field name
- Data type (string, double, integer, boolean, array, entity)
- Access level (r, rw)
- Default values
- Comments explaining the field's purpose

## Searching for Specific Rules

To find rules about a specific topic:

1. **Use file explorer search**:
   - Search for "AMT" to find Alternative Minimum Tax files
   - Search for "EITC" to find Earned Income Tax Credit

2. **Use Excel's Find function** (Ctrl+F or Cmd+F):
   - Open a table file
   - Search for keywords in conditions/actions

3. **Use grep on the entire directory**:
   ```bash
   grep -r "premium_tax_credit" excel/
   ```

## Integration with Development

These Excel files are **generated outputs** - do not edit them directly!

To modify decision tables:
1. Edit the XML source files in `xml/` or `xml/states/`
2. Run `./scripts/merge-states.sh`
3. Excel files will be automatically regenerated

## Technical Details

**Extraction Tools**:
- `dt2excel` - Converts TaxReturn_dt.xml → Excel files
- `edd2excel` - Converts TaxReturn_edd.xml → Excel file

**Source Code**:
- `go/cmd/dt2excel/main.go`
- `go/cmd/edd2excel/main.go`

**Total Size**: ~700 KB (all files)
**Last Generated**: Automatically updated on merge
