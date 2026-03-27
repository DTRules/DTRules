# Multi-File XML Extraction Guide

## Overview

This document explains the multi-file XML structure for the TaxReturn project, which separates monolithic `TaxReturn_dt.xml` and `TaxReturn_edd.xml` files into smaller, modular files organized by function and state.

## Why Multi-File Structure?

### Problems with Monolithic Files

1. **Merge Conflicts**: 40+ states all editing the same files creates guaranteed conflicts
2. **Difficult to Navigate**: 10,000+ line XML files are hard to work with
3. **Slow to Load**: Large files slow down editors and tools
4. **Tight Coupling**: All development must coordinate around shared files
5. **Testing Complexity**: Can't test individual states in isolation

### Benefits of Multi-File Structure

1. **Parallel Development**: Teams can work on different states simultaneously
2. **Easier Review**: State-specific changes are in state-specific files
3. **Clearer Organization**: Files organized by domain (federal core, states, forms)
4. **Faster Builds**: Only changed files need reprocessing
5. **Better Testing**: Can load and test individual states independently

## Directory Structure

```
sampleprojects/TaxReturn/xml/
├── TaxReturn_dt.xml              # Monolithic DT file (for backward compatibility)
├── TaxReturn_edd.xml             # Monolithic EDD file (for backward compatibility)
├── TaxReturn_map.xml             # Mapping configuration (unchanged)
├── 084_Calculate_State_Tax.xml   # State dispatcher table
├── TaxReturn_dt_core.xml         # Federal core decision tables
├── TaxReturn_edd_core.xml        # Federal core entity definitions
├── states/                       # State-specific files
│   ├── TEMPLATE_dt.xml           # Template for new states (DT)
│   ├── TEMPLATE_edd.xml          # Template for new states (EDD)
│   ├── AL_dt.xml                 # Alabama decision tables
│   ├── AL_edd.xml                # Alabama entity definitions
│   ├── CA_dt.xml                 # California decision tables
│   ├── CA_edd.xml                # California entity definitions
│   └── ...                       # (all 50 states)
└── forms/                        # IRS form-specific files (future)
    ├── 5695_dt.xml               # Form 5695 (Energy Credits)
    ├── 8615_dt.xml               # Form 8615 (Kiddie Tax)
    └── ...
```

## File Naming Conventions

### Decision Tables (`*_dt.xml`)

- **States**: `{STATE_CODE}_dt.xml` (e.g., `CA_dt.xml`, `NY_dt.xml`)
- **Forms**: `{FORM_NUMBER}_dt.xml` (e.g., `5695_dt.xml`, `8615_dt.xml`)
- **Core**: `TaxReturn_dt_core.xml` (federal tax logic)
- **Standalone**: `{NUMBER}_{TABLE_NAME}.xml` (e.g., `084_Calculate_State_Tax.xml`)

### Entity Definitions (`*_edd.xml`)

- **States**: `{STATE_CODE}_edd.xml` (e.g., `CA_edd.xml`, `NY_edd.xml`)
- **Forms**: `{FORM_NUMBER}_edd.xml` (e.g., `5695_edd.xml`)
- **Core**: `TaxReturn_edd_core.xml` (base entities)

## Table Numbering Scheme

Tables are numbered to group related functionality:

- **1000-1999**: Core federal tax calculation
- **2000-2999**: Income and AGI calculations
- **3000-3999**: Deductions, credits, and other taxes
- **4000-4999**: State tax calculations
  - **40000-40999**: States A-L
  - **41000-41999**: States M-Z
  - **42000-49999**: Reserved for expansion
- **5000+**: IRS forms and schedules

### State Table Numbers

Each state gets a unique range based on alphabetical order:

```
AL (Alabama)     = 40000-40099
AK (Alaska)      = 40100-40199
AZ (Arizona)     = 40200-40299
AR (Arkansas)    = 40300-40399
CA (California)  = 40400-40499
CO (Colorado)    = 40500-40599
...
WY (Wyoming)     = 44900-44999
```

## FILE_PATH Attribute

Each decision table has a `FILE_PATH` attribute that indicates its logical location:

```xml
<decision_table>
  <table_name>Calculate_CA_Tax</table_name>
  <attribute_fields>
    <TABLE_NUMBER>40400</TABLE_NUMBER>
    <FILE_PATH>states/CA/40400_Calculate_CA_Tax</FILE_PATH>
  </attribute_fields>
  ...
</decision_table>
```

**Rules:**
- State tables: `states/{STATE_CODE}/{TABLE_NUMBER}_{TABLE_NAME}`
- Form tables: `forms/{FORM_NUMBER}/{TABLE_NUMBER}_{TABLE_NAME}`
- Core tables: `core/{TABLE_NUMBER}_{TABLE_NAME}`

The `FILE_PATH` is used for:
- Documentation and navigation
- Trace output (shows which logical file made a decision)
- Future tooling (rule editors, debuggers)

## Workflow

### For Developers: Working with Multi-File Structure

#### Adding a New State

1. **Copy templates:**
   ```bash
   cd sampleprojects/TaxReturn/xml/states
   cp TEMPLATE_dt.xml CO_dt.xml
   cp TEMPLATE_edd.xml CO_edd.xml
   ```

2. **Edit state files:**
   - Update `CO_dt.xml` with Colorado's tax logic
   - Update `CO_edd.xml` with Colorado-specific constants
   - Use appropriate TABLE_NUMBER from the numbering scheme

3. **Validate extraction:**
   ```bash
   cd sampleprojects/TaxReturn
   python3 scripts/validate_extraction.py
   ```

4. **Merge for testing:**
   ```bash
   python3 scripts/merge_files.py
   ```

5. **Test:**
   ```bash
   cd ../../go
   go test ./pkg/dtrules/... -run TestTaxReturn
   ```

6. **Commit only state files:**
   ```bash
   git add xml/states/CO_dt.xml xml/states/CO_edd.xml
   git commit -m "feat: implement CO state tax (#180)"
   ```

#### Modifying Existing State

1. **Edit the state file directly:**
   ```bash
   vim xml/states/CA_dt.xml
   ```

2. **Merge and test:**
   ```bash
   python3 scripts/merge_files.py
   cd ../../go && go test ./pkg/dtrules/... -run TestTaxReturn
   ```

3. **Validate changes:**
   ```bash
   python3 scripts/validate_extraction.py
   ```

4. **Commit:**
   ```bash
   git add xml/states/CA_dt.xml
   git commit -m "fix: update CA tax brackets for 2025"
   ```

### For Build Systems: Merging Files

The build process should merge files before running tests:

```bash
# In CI/CD pipeline
cd sampleprojects/TaxReturn
python3 scripts/merge_files.py
cd ../../go
go test ./pkg/dtrules/...
```

### For Releases: Creating Monolithic Files

Before release, create final merged files:

```bash
cd sampleprojects/TaxReturn
python3 scripts/merge_files.py --verbose
python3 scripts/validate_extraction.py
```

The merged `TaxReturn_dt.xml` and `TaxReturn_edd.xml` can be distributed as single-file packages for users who prefer the monolithic format.

## Validation

### Running Validation

```bash
cd sampleprojects/TaxReturn
python3 scripts/validate_extraction.py > /tmp/validate.log 2>&1
tail -50 /tmp/validate.log
```

### What Validation Checks

1. **XML Well-Formedness**: All XML files parse correctly
2. **Table Completeness**: All tables from monolithic file are in extracted files
3. **No Duplicates**: Each TABLE_NUMBER appears exactly once
4. **FILE_PATH Consistency**: Paths match directory structure
5. **Entity Completeness**: All entities from monolithic file are present
6. **Field Counts**: Entity field counts match

### Common Validation Errors

#### Missing Tables

```
ERROR: Missing 3 tables in extracted files: ['40400', '40500', '40600']
```

**Fix**: Extract the missing tables from the monolithic file into appropriate state files.

#### Duplicate Tables

```
ERROR: Table 40400 appears in multiple files: ['CA_dt.xml', 'TaxReturn_dt_core.xml']
```

**Fix**: Remove the duplicate. Tables should only appear in one file.

#### FILE_PATH Mismatch

```
WARNING: Table 40400 in CA_dt.xml has FILE_PATH 'states/NY/40400_Calculate_Tax'
         but expected prefix 'states/CA/'
```

**Fix**: Update the FILE_PATH to match the state code.

#### Entity Field Count Mismatch

```
WARNING: Entity 'result' has 45 fields in monolithic but 42 fields in result_edd.xml
```

**Fix**: Check that all fields were extracted. Some fields may be state-specific and should be in state EDD files.

## Merge Script

### Basic Usage

```bash
# Merge all files into xml/TaxReturn_dt.xml and xml/TaxReturn_edd.xml
python3 scripts/merge_files.py
```

### Options

```bash
# Dry run (show what would be merged)
python3 scripts/merge_files.py --dry-run

# Verbose output
python3 scripts/merge_files.py --verbose

# Custom output directory
python3 scripts/merge_files.py --output-dir /tmp/merged
```

### How Merging Works

1. **Find Files**: Locates all `*_dt.xml` and `*_edd.xml` files (except monolithic)
2. **Extract Elements**: Parses each file and extracts decision tables / entities
3. **Sort**:
   - Decision tables sorted by TABLE_NUMBER (numeric)
   - Entities sorted alphabetically by name
4. **Combine**: Creates new XML tree with all elements
5. **Write**: Outputs merged `TaxReturn_dt.xml` and `TaxReturn_edd.xml`

### Merge Output Example

```
=== Merging Decision Tables ===
Found 52 decision table files to merge
Extracted 287 decision tables
✅ Created xml/TaxReturn_dt.xml

=== Merging Entity Definitions ===
Found 52 entity definition files to merge
Extracted 94 entity definitions
✅ Created xml/TaxReturn_edd.xml

✅ Merge completed successfully
```

## When to Use Multi-File vs Monolithic

### Use Multi-File Structure When:

- ✅ Developing new states or forms
- ✅ Multiple developers working simultaneously
- ✅ Need to review state-specific changes
- ✅ Testing individual states in isolation
- ✅ Version controlling changes

### Use Monolithic Files When:

- ✅ Deploying to production (backward compatibility)
- ✅ Distributing complete rule set as single artifact
- ✅ Loading entire ruleset in DTRules engine (currently)
- ✅ Using legacy tools that expect monolithic format

## Migration Strategy

### Phase 1: Extraction (Current)

- Extract decision tables and entities into separate files
- Keep monolithic files for backward compatibility
- Validate extraction is complete and correct

### Phase 2: Development (Now)

- Developers work with multi-file structure
- Build scripts merge files automatically
- CI/CD runs validation on every commit

### Phase 3: Runtime Loading (Future)

- Update DTRules engine to load multi-file structure directly
- Remove need to merge files for testing
- Faster loading (only load needed states/forms)

### Phase 4: Distribution (Future)

- Distribute either multi-file or merged monolithic
- Tools to convert between formats
- Support both formats indefinitely

## Best Practices

### For State Developers

1. **Copy Templates**: Always start from `TEMPLATE_dt.xml` and `TEMPLATE_edd.xml`
2. **Use Correct Numbers**: Follow the table numbering scheme
3. **Validate Early**: Run validation frequently during development
4. **Test After Merge**: Always test after merging
5. **Commit Only State Files**: Don't commit merged monolithic files

### For Reviewers

1. **Check FILE_PATH**: Verify paths match directory structure
2. **Verify Table Numbers**: Ensure numbers don't conflict
3. **Run Validation**: Validate before approving PR
4. **Test Coverage**: Require test cases for each state

### For CI/CD

1. **Always Validate**: Run validation on every commit
2. **Merge Before Test**: Merge files before running tests
3. **Check Both**: Test both multi-file and merged versions
4. **Fail Fast**: Fail build if validation errors exist

## Troubleshooting

### "ERROR: XML parse error"

**Cause**: Malformed XML syntax in one of the files.

**Fix**: Check the specific file mentioned. Common issues:
- Unclosed tags
- Missing quotes in attributes
- Special characters not escaped (`&`, `<`, `>`)

### "ERROR: Duplicate table numbers"

**Cause**: Same TABLE_NUMBER appears in multiple files.

**Fix**: Assign a unique TABLE_NUMBER to each table per the numbering scheme.

### "WARNING: Entity field count mismatch"

**Cause**: Entity definition differs between monolithic and extracted files.

**Fix**: Verify all fields were extracted. State-specific fields should be in state EDD files, not core.

### Tests Fail After Merge

**Cause**: Merge didn't include all necessary tables or entities.

**Fix**:
1. Run validation to identify missing pieces
2. Check that all files are being found by glob patterns
3. Verify TABLE_NUMBER and entity names are correct

## Future Enhancements

1. **Direct Multi-File Loading**: Update DTRules engine to load from multiple files
2. **Incremental Compilation**: Only reprocess changed files
3. **State Templates**: Generate state files from templates automatically
4. **Visual Editor**: GUI for editing decision tables with multi-file awareness
5. **Dependency Tracking**: Automatic detection of table dependencies
6. **Schema Validation**: XSD schema validation for all XML files

## Questions and Support

For questions about the multi-file structure:
1. Check this guide first
2. Run validation to identify specific issues
3. Check git history for examples of state implementations
4. Open an issue if you find bugs or need clarification

## See Also

- `scripts/validate_extraction.py` - Validation script source
- `scripts/merge_files.py` - Merge script source
- `xml/states/TEMPLATE_dt.xml` - Decision table template
- `xml/states/TEMPLATE_edd.xml` - Entity definition template
- `.claude/CLAUDE.md` - Project development guidelines
