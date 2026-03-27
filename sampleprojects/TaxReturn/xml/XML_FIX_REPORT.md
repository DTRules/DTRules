# XML Corruption Fix Summary

## Files Fixed (2/4)

### ✓ DC_dt.xml - DC State Decision Tables
**Status:** FIXED
**Issues:**
- Nested decision_table tags (two tables merged into one)
- Content from Process_Foreign_Earned_Income table embedded in Calculate_DC_Homebuyer_Credit table
- Actions from both tables mixed together

**Changes Made:**
1. Properly closed first table (Calculate_DC_Homebuyer_Credit) before second table starts
2. Separated action content between the two tables:
   - Action 1 (DC homebuyer): Calculate credit (lesser of $5,000 or purchase price)
   - Action 2 (DC homebuyer): No DC home purchase
3. Created proper second table (Process_Foreign_Earned_Income) with its own contexts and actions

**Validation:** ✓ Parses correctly - 2 decision tables

### ✓ MT_dt.xml - Montana State Decision Tables
**Status:** FIXED
**Issues:**
- Action 1 had content from TWO different tables mixed together:
  - Energy Efficient Appliance Credit (dishwashers) - WRONG
  - Montana state tax calculation (MFJ) - CORRECT
- Action 4 had irrelevant energy appliance content

**Changes Made:**
1. Removed dishwasher energy credit content from Action 1
2. Kept only MT tax calculation for MFJ in Action 1
3. Removed Action 4 (energy appliance credit - not applicable to MT state tax)

**Validation:** ✓ Parses correctly - 1 decision table

## Files Still Broken (2/4)

### ✗ TaxReturn_dt_core.xml - Core Decision Tables
**Status:** STILL HAS ERRORS
**Error:** Parse error at line 8435

**Known Issues:**
1. Line 7541-7551: Context content from Process_Installment_Sales embedded in Calculate_Mortgage_Interest_Credit conditions - FIXED
2. Line 8361-8376: Mortgage credit content in Calculate_Partnership_Audit_Tax actions - FIXED  
3. Line 10016-10034: MT tax HOH content embedded in kiddie tax action - FIXED
4. Line 8381-8389: Empty Calculate_Energy_Appliance_Credit table (only headers, no body) - REMOVED
5. Line 8427-8435: Unreported tips context embedded in energy appliance initial_action - STILL PRESENT

**Partial Fixes Applied:**
- Fixed 3 instances of cross-table content contamination
- Removed 1 incomplete table stub
- Still has at least 1 more corruption at line 8435

### ✗ TaxReturn_dt.xml - Full Decision Tables  
**Status:** STILL HAS ERRORS
**Error:** Parse error at line 7798

**Known Issues:**
- Similar corruption patterns as core file
- Likely has content from one table embedded in another around line 7798

## Root Cause Analysis

The corruption follows a consistent pattern:
1. **Content Merger:** XML content from one decision table is incorrectly embedded within another table's structure
2. **Missing Close Tags:** action_postfix, condition_postfix, or context_postfix tags are not properly closed
3. **Duplicate Tags:** Multiple action_description or action_postfix tags within a single action_details block

This appears to be a systematic issue with the extraction script (`extract_dt_tables.py`) that generates these files from the source XML. The script likely has bugs in:
- Tracking table boundaries
- Properly closing XML tags
- Preventing content from bleeding between tables

## Recommendations

1. **Short-term:** The state files (DC, MT) are now usable
2. **Medium-term:** Manually fix remaining corruptions in core and full files (labor intensive)
3. **Long-term:** Fix the extraction script to prevent this corruption in future generations

## Files Modified

- `/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/TaxReturn/xml/states/DC_dt.xml` - FIXED
- `/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/TaxReturn/xml/states/MT_dt.xml` - FIXED
- `/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/TaxReturn/xml/TaxReturn_dt_core.xml` - PARTIALLY FIXED
- `/home/paul/go/src/github.com/DTRules/DTRules/sampleprojects/TaxReturn/xml/TaxReturn_dt.xml` - NOT FIXED

## Backups Created

- `DC_dt.xml.backup`
- `MT_dt.xml.backup`
