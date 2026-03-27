# Multi-File XML Structure Roadmap

This document outlines the plan to transition DTRules from monolithic XML files to a multi-file directory structure.

## Current State (As of 2025-03-23)

### Monolithic Structure
```
sampleprojects/TaxReturn/
  xml/
    TaxReturn_dt.xml      # 83+ decision tables in one file
    TaxReturn_edd.xml     # 17 entities in one file
    states/
      CO_dt.xml           # Separate state files (not loaded)
      CO_edd.xml
      CA_dt.xml
      ...
```

### Problems
- Single files become very large (TaxReturn_dt.xml > 50,000 lines)
- Merge conflicts when multiple developers work on different tables
- State tax tables exist but are not integrated
- Difficult to find specific tables

## Target State

### Multi-File Structure
```
sampleprojects/TaxReturn/
  xml/
    000_core_entities.xml             # EDD (even numbers)
    001_Compute_Tax_Return.xml        # DT (odd numbers)
    002_result_entity.xml             # EDD
    003_Calculate_Gross_Income.xml    # DT
    ...
    083_Calculate_Schedule_H.xml
    084_Calculate_State_Tax.xml       # NEW - state dispatcher
    states/
      AL/
        40100_AL_constants.xml        # EDD
        40101_Calculate_AL_Tax.xml    # DT
      CA/
        40500_CA_constants.xml
        40501_Calculate_CA_Tax.xml
      CO/
        40600_CO_constants.xml
        40601_Calculate_CO_Tax.xml
      ...
  excel/                              # Generated documentation
    (mirrors xml/ structure)
```

### Numbering Convention
- **Even numbers** (0, 2, 4, ...) = EDD files
- **Odd numbers** (1, 3, 5, ...) = DT files
- **Core tables**: 000-999
- **State tables**: 40000-49999
  - Each state gets 100 numbers
  - State index = alphabetical position (AL=1, AK=2, ..., CO=6, ...)
  - State base = 40000 + (state_index × 100)
  - Example: CO (6th) = 40600-40699

### File Path Metadata

Each XML file contains its canonical path for Excel generation:

**Decision Tables:**
```xml
<decision_table>
  <table_name>Calculate_CO_Tax</table_name>
  <attribute_fields>
    <TABLE_NUMBER>40601</TABLE_NUMBER>
    <FILE_PATH>states/CO/40601_Calculate_CO_Tax</FILE_PATH>
  </attribute_fields>
  ...
</decision_table>
```

**Entity Data Dictionary:**
```xml
<entity_data_dictionary>
  <file_metadata>
    <file_path>states/CO/40600_CO_constants</file_path>
  </file_metadata>
  <entity name="result">
    ...
  </entity>
</entity_data_dictionary>
```

## Implementation Plan

### Phase 1: Infrastructure (#340, #342)
- [x] Add FILE_PATH metadata to XML schema
- [ ] Implement LoadRulesFromDirectory() function
- [ ] Support recursive directory scanning
- [ ] Parse even/odd numbering to determine file type
- [ ] Maintain backward compatibility with monolithic files

### Phase 2: TaxReturn Migration (#343)
- [ ] Extract each decision table to separate numbered file
- [ ] Extract each entity to separate file or keep grouped
- [ ] Create state subdirectories (AL/, CA/, CO/, ...)
- [ ] Move state files to subdirectories with proper numbering
- [ ] Add FILE_PATH metadata to all files
- [ ] Keep old monolithic files temporarily (deprecated)

### Phase 3: State Tax Integration (#344, #345)
- [ ] Create Calculate_State_Tax dispatcher (Table #084)
- [ ] Route based on residence_state field
- [ ] Call appropriate state table (Calculate_AL_Tax, Calculate_CO_Tax, etc.)
- [ ] Integrate into main Compute_Tax_Return flow
- [ ] Update Calculate_Final_Balance to include state tax
- [ ] Target: 100% test pass rate (currently 24%)

### Phase 4: Excel Extraction (#346)
- [x] Create dt2excel tool
- [x] Create edd2excel tool
- [ ] Update to use FILE_PATH metadata
- [ ] Mirror XML directory structure in excel/
- [ ] Update extraction scripts

### Phase 5: Other Sample Projects (#347)
- [ ] Apply same structure to CDSExample
- [ ] Apply to CHIP
- [ ] Apply to Invoicing
- [ ] Document patterns and best practices
- [ ] Create migration guide for users

## Benefits

### Development
- ✅ **No merge conflicts** - Each table in separate file
- ✅ **Easy to find** - Numbered files, clear organization
- ✅ **State isolation** - States in separate directories
- ✅ **Parallel development** - Multiple developers on different states

### Deployment
- ✅ **Clean separation** - xml/ deploys, excel/ doesn't
- ✅ **Smaller files** - Easier to review, faster to load
- ✅ **Modular** - Load only what you need

### Documentation
- ✅ **Excel alongside XML** - Easy to find documentation
- ✅ **Directory structure** - Self-documenting organization
- ✅ **Version control friendly** - Clear diffs, easy to review

## Current Test Status

- **Total tests**: 154 (51 states × 3 tests each)
- **Passing**: 37 (24%)
- **Failing**: 117 (76%)

### Why Tests Are Failing
State tax calculations are not integrated into main flow. Only no-income-tax states pass because they expect $0 state tax.

**Passing states** (no income tax):
- AK, FL, SD, TX, WY (+ partial: NV, TN, NH, WA)

**Expected after integration**: 100% pass rate

## Issues Tracking

All work tracked in GitHub issues:

- **#340** - Add FILE_PATH metadata to XML schemas
- **#342** - Implement multi-file XML loader
- **#343** - Restructure TaxReturn XML files
- **#344** - Create Calculate_State_Tax dispatcher
- **#345** - Integrate state tax into main flow
- **#346** - Update Excel extraction
- **#347** - Apply to all sample projects

See: https://github.com/DTRules/DTRules/issues

## Timeline

**Estimated effort**: 3-4 weeks

- Week 1: Infrastructure & loader (#340, #342)
- Week 2: TaxReturn restructuring (#343)
- Week 3: State tax integration (#344, #345)
- Week 4: Excel & other projects (#346, #347)

## Migration Strategy

### Backward Compatibility
- Keep monolithic files during transition
- Support both LoadDecisionTables(file) and LoadRulesFromDirectory(dir)
- Mark monolithic approach as deprecated
- Remove after multi-file stable

### Testing Strategy
- All existing tests must pass
- Add tests for multi-file loading
- Verify table execution order preserved
- Validate Excel extraction accuracy

## Questions & Decisions

### Decided
- ✅ Even=EDD, Odd=DT numbering convention
- ✅ Full FILE_PATH in metadata (not derived)
- ✅ Separate xml/ and excel/ directory trees
- ✅ State subdirectories (states/CO/, states/CA/, ...)

### To Decide
- [ ] EDD granularity: One big file or one per entity?
- [ ] Backward compatibility timeline: When to remove monolithic files?
- [ ] Excel file format: One sheet per table or multiple sheets?

## Resources

- **Excel extraction tools**: go/cmd/dt2excel/, go/cmd/edd2excel/
- **Extraction script**: sampleprojects/TaxReturn/scripts/extract-to-excel.sh
- **Current state files**: sampleprojects/TaxReturn/xml/states/
- **Test scenarios**: sampleprojects/TaxReturn/testfiles/TestScenarios/State/

---

**Last Updated**: 2025-03-23
**Status**: Planning Phase
**Next Step**: Issue #340 (FILE_PATH metadata)
