# XML File Filtering Rules for LoadFromDirectory

## Overview

The `LoadFromDirectory` function recursively scans a directory for DTRules XML files and loads them in proper order. To prevent issues with template files, merged files, and other non-loadable XML files, several filtering rules are applied.

## Filtering Rules

### 1. File Extension Filter
- Only files ending in `.xml` (case-insensitive) are considered
- Applied in: `CollectXMLFiles()`

### 2. Early Path-Based Filters
Applied before parsing to quickly skip known problematic files:

- **Template Files**: Skip files with "TEMPLATE" in the filename
  - Example: `TEMPLATE_dt.xml`, `TEMPLATE_edd.xml`
  - Reason: These contain placeholder values like `TABLE_NUMBER="4XXXX"`

- **Mapping Files**: Skip files ending with `_map.xml`
  - Example: `TaxReturn_map.xml`
  - Reason: These are mapping configuration files, not DT/EDD files

- **Test Files**: Skip files in `/testfiles/` or `\testfiles\` directories
  - Reason: Test data files, not rule definitions

- **Schema Files**: Skip files in `/schemas/` or `\schemas\` directories
  - Reason: XML schema definitions, not rule files

Applied in: `shouldSkipFile()`

### 3. Content-Based Filters
Applied after reading file contents:

- **Non-DTRules Files**: Skip files that don't contain `<decision_tables` or `<entity_data_dictionary`
  - Reason: Not a DTRules XML file format

- **Template Placeholders**: Skip DT files with placeholders in TABLE_NUMBER
  - Example: TABLE_NUMBER containing "X" or "?"
  - Reason: Templates, not actual tables

- **Missing FILE_PATH (Decision Tables)**: Skip DT files where the first table has empty FILE_PATH
  - Example: Merged files like `TaxReturn_dt.xml`
  - Reason: These are generated/merged files containing many tables; individual table files should have FILE_PATH

- **Missing file_path (EDD)**: Skip EDD files without file_metadata/file_path
  - Example: Merged files like `TaxReturn_edd.xml`
  - Reason: These are generated/merged files; individual EDD files should have file_path

Applied in: `ParseFileMetadata()`

## Why These Filters?

### Problem: Test Timeouts
Without proper filtering, `LoadFromDirectory` would attempt to parse:
- Template files with invalid TABLE_NUMBER values
- Large merged files containing 100+ tables
- Mapping configuration files with different XML schemas
- Test scenario files

This caused:
- XML parsing errors and infinite retry loops
- Extremely slow performance (140-220 second timeouts)
- Out-of-memory conditions from loading duplicate data

### Solution: Multi-Layer Filtering
By applying filters at multiple stages:
1. **Path check** (fast, no I/O beyond directory walk)
2. **Quick content scan** (string contains check)
3. **Validation during parse** (catch placeholder values)

We ensure only valid, individual rule files are loaded.

## Expected File Structure

### Valid Decision Table File
```xml
<decision_tables>
  <decision_table>
    <table_name>Calculate_CO_Tax</table_name>
    <attribute_fields>
      <TABLE_NUMBER>40600</TABLE_NUMBER>
      <FILE_PATH>states/CO/40600_Calculate_CO_Tax</FILE_PATH>
      ...
    </attribute_fields>
    ...
  </decision_table>
</decision_tables>
```

### Valid EDD File
```xml
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/CO/40600_CO_constants</file_path>
  </file_metadata>
  <entity name="result">
    ...
  </entity>
</entity_data_dictionary>
```

## State Tax Implementation Pattern

For state tax implementations:
- Each state gets 2 files: `XX_dt.xml` and `XX_edd.xml`
- Both files have unique numbering (FILE_PATH and file_path)
- Files are loaded by `LoadFromDirectory` automatically
- Merged files (`TaxReturn_dt.xml`, `TaxReturn_edd.xml`) are generated for backwards compatibility but NOT loaded by directory loader

## Testing

To verify filtering is working:
```bash
cd go && go test -v ./pkg/dtrules -run TestMultiFileLoading
```

Expected warnings (these are GOOD - they show filtering is working):
```
Warning: skipping file .../TaxReturn_map.xml: file ... does not appear to be a DTRules XML file
Warning: skipping file .../TEMPLATE_dt.xml: template file with placeholder TABLE_NUMBER: 4XXXX
Warning: skipping file .../TaxReturn_dt.xml: decision table file missing FILE_PATH (likely a merged/generated file)
Warning: skipping file .../TaxReturn_edd.xml: EDD file missing file_path (likely a merged/generated file)
```

No warnings for individual state files - they should all load successfully.
