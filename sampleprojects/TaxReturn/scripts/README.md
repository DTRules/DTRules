# TaxReturn XML Processing Scripts

This directory contains scripts for managing the multi-file XML structure of the TaxReturn project.

## Scripts

### validate_extraction.py

Validates that decision tables and entity definitions have been correctly extracted from monolithic XML files into the multi-file structure.

**Usage:**
```bash
cd sampleprojects/TaxReturn
python3 scripts/validate_extraction.py
```

**What it checks:**
- All XML files are well-formed
- All decision tables from monolithic file are in extracted files
- No duplicate TABLE_NUMBERs across files
- FILE_PATH attributes match directory structure
- All entity definitions are present
- Entity field counts match between monolithic and extracted files

**Exit codes:**
- `0` - All validations passed
- `1` - Validation errors found

**Output example:**
```
=== Validating XML Well-Formedness ===
Checked 67 XML files for well-formedness

=== Validating Decision Tables ===
Loading monolithic file: xml/TaxReturn_dt.xml
Found 145 tables in monolithic file
Found 34 extracted DT files
Total extracted tables: 145

=== Validating Entity Definitions ===
Loading monolithic file: xml/TaxReturn_edd.xml
Found 94 entities in monolithic file
Found 34 extracted EDD files
Total extracted entities: 94

✅ Validation PASSED
```

### merge_files.py

Merges multi-file XML structure back into monolithic `TaxReturn_dt.xml` and `TaxReturn_edd.xml` files.

**Usage:**
```bash
cd sampleprojects/TaxReturn

# Basic merge
python3 scripts/merge_files.py

# Dry run (don't write files)
python3 scripts/merge_files.py --dry-run

# Verbose output
python3 scripts/merge_files.py --verbose

# Custom output directory
python3 scripts/merge_files.py --output-dir /tmp/merged
```

**How it works:**
1. Finds all `*_dt.xml` and `*_edd.xml` files (excluding monolithic files)
2. Parses and extracts decision tables and entities
3. Sorts decision tables by TABLE_NUMBER, entities alphabetically
4. Generates merged XML files with proper formatting
5. Writes `TaxReturn_dt.xml` and `TaxReturn_edd.xml`

**Output example:**
```
=== Merging Decision Tables ===
Found 34 decision table files to merge
Extracted 145 decision tables
✅ Created xml/TaxReturn_dt.xml

=== Merging Entity Definitions ===
Found 34 entity definition files to merge
Extracted 94 entity definitions
✅ Created xml/TaxReturn_edd.xml

✅ Merge completed successfully
```

### test_validation.sh

Test suite that demonstrates validation and merge functionality with various error cases.

**Usage:**
```bash
cd sampleprojects/TaxReturn
bash scripts/test_validation.sh
```

**What it tests:**
- Well-formed XML files (should pass)
- Malformed XML (should detect)
- Duplicate TABLE_NUMBERs (should detect)
- FILE_PATH mismatches (should warn)
- Real project validation

## Workflow Examples

### After Modifying State Files

```bash
cd sampleprojects/TaxReturn

# Validate your changes
python3 scripts/validate_extraction.py

# Merge for testing
python3 scripts/merge_files.py

# Run tests
cd ../../go
go test ./pkg/dtrules/... -run TestTaxReturn
```

### Before Committing

```bash
cd sampleprojects/TaxReturn

# Validate everything is correct
python3 scripts/validate_extraction.py

# DON'T commit the merged monolithic files
# Only commit your state-specific files
git add xml/states/CO_dt.xml xml/states/CO_edd.xml
git commit -m "feat: implement CO state tax"
```

### In CI/CD Pipeline

```bash
cd sampleprojects/TaxReturn

# Validate extraction
python3 scripts/validate_extraction.py || exit 1

# Merge files for testing
python3 scripts/merge_files.py || exit 1

# Run tests
cd ../../go
go test ./pkg/dtrules/... || exit 1
```

### Troubleshooting Validation Errors

**Error: XML parse error**
```
ERROR: XML parse error in CA_dt.xml: mismatched tag: line 45, column 2
```
Fix: Open the file and fix the XML syntax error at the indicated line.

**Error: Duplicate table numbers**
```
ERROR: Table 40400 appears in multiple files: ['CA_dt.xml', 'CO_dt.xml']
```
Fix: Assign unique TABLE_NUMBERs per the numbering scheme (see EXTRACTION_GUIDE.md).

**Warning: FILE_PATH mismatch**
```
WARNING: Table 40400 in CA_dt.xml has FILE_PATH 'states/CO/40400_Tax'
         but expected prefix 'states/CA/'
```
Fix: Update the FILE_PATH to match the state code in the filename.

**Error: Missing tables**
```
ERROR: Missing 5 tables in extracted files: ['1000', '2000', '3000', ...]
```
Fix: Extract the missing tables from TaxReturn_dt.xml into appropriate files.

## Redirecting Output

As per project guidelines, always redirect output to log files:

```bash
# Validation
python3 scripts/validate_extraction.py > /tmp/validate.log 2>&1
tail -50 /tmp/validate.log

# Merge
python3 scripts/merge_files.py > /tmp/merge.log 2>&1
tail -30 /tmp/merge.log
```

## Requirements

- Python 3.7 or later
- No external dependencies (uses only standard library)

## See Also

- `../EXTRACTION_GUIDE.md` - Complete guide to multi-file XML structure
- `../.claude/CLAUDE.md` - Project development guidelines
- `../xml/states/TEMPLATE_dt.xml` - Template for new state decision tables
- `../xml/states/TEMPLATE_edd.xml` - Template for new state entity definitions
