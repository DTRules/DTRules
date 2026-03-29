# Multi-File XML Structure Migration Guide

## Overview

DTRules now supports loading rules from a directory containing multiple XML files, in addition to the traditional monolithic file approach. This allows for better organization, reduced merge conflicts, and modular rule development.

## Quick Start

### Before (Monolithic Files)

```go
rs := session.NewRuleSet("TaxReturn")

// Load EDD
eddFile, err := os.Open("TaxReturn_edd.xml")
if err != nil {
    return err
}
defer eddFile.Close()
err = rs.LoadEDD(eddFile)
if err != nil {
    return err
}

// Load Decision Tables
dtFile, err := os.Open("TaxReturn_dt.xml")
if err != nil {
    return err
}
defer dtFile.Close()
err = rs.LoadDecisionTables(dtFile)
if err != nil {
    return err
}
```

### After (Multi-File Directory)

```go
rs := session.NewRuleSet("TaxReturn")

// Load all XML files from directory
err := rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")
if err != nil {
    return err
}
```

## API Methods

### LoadFromDirectory

```go
func (rs *RuleSet) LoadFromDirectory(dirPath string) error
```

Loads all XML files from a directory recursively:
- Scans for `*.xml` files
- Separates EDD and DT files
- Sorts by numbering (file_path for EDDs, TABLE_NUMBER for DTs)
- Loads EDDs first, then decision tables
- Returns aggregated errors if any files fail to load

### LoadFromPath

```go
func (rs *RuleSet) LoadFromPath(path, eddPath, dtPath string) error
```

Flexible loading that supports:
- Directory path: `rs.LoadFromPath("./xml", "", "")`
- Individual files: `rs.LoadFromPath("", "./xml/TaxReturn_edd.xml", "./xml/TaxReturn_dt.xml")`

### Package-level Convenience Functions

```go
// Load from directory
rs, err := session.LoadRulesFromDirectory("TaxReturn", "./sampleprojects/TaxReturn/xml")

// Load from files
rs, err := session.LoadRulesFromFiles("TaxReturn",
    "./xml/TaxReturn_edd.xml",
    "./xml/TaxReturn_dt.xml")
```

## File Structure

### Multi-File Organization

```
sampleprojects/TaxReturn/xml/
├── TaxReturn_edd.xml          # Core EDD (optional - can be split)
├── TaxReturn_dt.xml           # Core DT (optional - can be split)
├── TaxReturn_map.xml          # Mapping file (unchanged)
└── states/
    ├── AL_edd.xml             # Alabama constants
    ├── AL_dt.xml              # Alabama decision tables
    ├── CA_edd.xml             # California constants
    ├── CA_dt.xml              # California decision tables
    └── ...
```

### File Numbering

Files are loaded in numerical order based on:

**EDD files**: `file_path` in `<file_metadata>`:
```xml
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/AL/40100_AL_constants</file_path>
  </file_metadata>
  ...
</entity_data_dictionary>
```

**DT files**: `TABLE_NUMBER` in `<attribute_fields>`:
```xml
<decision_table>
  <table_name>AL_Calculate_State_Tax</table_name>
  <attribute_fields>
    <TABLE_NUMBER>40100</TABLE_NUMBER>
    <FILE_PATH>states/AL/40100_AL_Calculate_State_Tax</FILE_PATH>
  </attribute_fields>
  ...
</decision_table>
```

### Recommended Numbering Scheme

- `00000-09999`: Core framework files
- `10000-39999`: Federal/national rules
- `40000-69999`: State-specific rules (by state code)
- `70000-89999`: Extended features
- `90000-99999`: Test/experimental files

Example for state files:
- Alabama: 40100-40199
- Alaska: 40200-40299
- Arizona: 40300-40399
- etc.

## Backward Compatibility

The system maintains full backward compatibility:

1. **Monolithic files still work**: Existing code using `LoadEDD()` and `LoadDecisionTables()` continues to function
2. **Hybrid approach supported**: Mix directory loading with individual file loading
3. **Automatic fallback**: Tests can try directory loading first, fall back to monolithic files

Example fallback pattern:
```go
rs := session.NewRuleSet("TaxReturn")

// Try multi-file structure first
err := rs.LoadFromDirectory(xmlDir)
if err != nil {
    // Fall back to monolithic files
    eddFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_edd.xml"))
    defer eddFile.Close()
    rs.LoadEDD(eddFile)

    dtFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_dt.xml"))
    defer dtFile.Close()
    rs.LoadDecisionTables(dtFile)
}
```

## Migration Steps

### For New Projects

Use multi-file structure from the start:

1. Create numbered XML files for each module
2. Add `<file_metadata>` to EDDs with `<file_path>`
3. Add `TABLE_NUMBER` and `FILE_PATH` to decision tables
4. Use `LoadFromDirectory()` in your code

### For Existing Projects

Two approaches:

#### Approach 1: Extract to Multi-File (Recommended)

1. Split monolithic files into logical modules
2. Add numbering to each file
3. Update code to use `LoadFromDirectory()`
4. Keep monolithic files as backup during transition

#### Approach 2: Hybrid (Gradual Migration)

1. Keep core files monolithic
2. Add new features as separate numbered files
3. Use `LoadFromDirectory()` which will load both
4. Gradually migrate core files when ready

## Testing

### Updated Test Pattern

```go
func TestTaxReturn(t *testing.T) {
    rs := session.NewRuleSet("TaxReturn")

    // Multi-file loading
    err := rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")
    if err != nil {
        t.Fatalf("Failed to load rules: %v", err)
    }

    // Rest of test...
}
```

### Backward Compatibility Test Pattern

```go
func TestBackwardCompatibility(t *testing.T) {
    rs := session.NewRuleSet("TaxReturn")

    // Try new approach
    err := rs.LoadFromDirectory(xmlDir)
    if err != nil {
        // Fall back to old approach
        eddFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_edd.xml"))
        defer eddFile.Close()
        rs.LoadEDD(eddFile)

        dtFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_dt.xml"))
        defer dtFile.Close()
        rs.LoadDecisionTables(dtFile)
    }

    // Verify loaded correctly
    if len(rs.GetDecisionTableNames()) == 0 {
        t.Fatal("No decision tables loaded")
    }
}
```

## Benefits

### Development

- **Modularity**: Each feature/state in its own file
- **No merge conflicts**: Independent files don't conflict
- **Easier navigation**: Find rules by filename
- **Incremental loading**: Load only what you need (future feature)

### Maintenance

- **Clear ownership**: Files map to team/feature boundaries
- **Version control**: Better diffs and blame tracking
- **Selective updates**: Change one module without affecting others
- **Testing**: Test individual modules in isolation

### Collaboration

- **Parallel development**: Multiple developers on different files
- **Code review**: Smaller, focused changes
- **Feature flags**: Enable/disable by file presence
- **A/B testing**: Swap alternative implementation files

## Common Patterns

### Pattern 1: Core + Extensions

```
xml/
├── 10000_core_edd.xml        # Core entities
├── 10001_core_dt.xml         # Core tables
├── 20000_extensions_edd.xml  # Extension entities
└── 20001_extensions_dt.xml   # Extension tables
```

### Pattern 2: Feature Modules

```
xml/
├── 10000_authentication_edd.xml
├── 10001_authentication_dt.xml
├── 20000_authorization_edd.xml
├── 20001_authorization_dt.xml
├── 30000_reporting_edd.xml
└── 30001_reporting_dt.xml
```

### Pattern 3: Jurisdictional (e.g., states)

```
xml/
├── core/
│   ├── 10000_federal_edd.xml
│   └── 10001_federal_dt.xml
└── states/
    ├── AL_edd.xml
    ├── AL_dt.xml
    ├── CA_edd.xml
    └── CA_dt.xml
```

## Troubleshooting

### Problem: Files not loading in correct order

**Solution**: Check numbering in `file_path` (EDDs) and `TABLE_NUMBER` (DTs). Files load in numerical order.

### Problem: Duplicate entity/table errors

**Solution**: Ensure no duplicate names across files. Each entity/table name must be unique.

### Problem: Missing dependencies

**Solution**: Check file numbering. Dependencies should be in lower-numbered files.

### Problem: Directory loading fails

**Solution**: Ensure at least one valid EDD or DT file exists. Check file permissions.

## Performance Considerations

- **Load time**: Slightly slower than monolithic due to multiple file opens
- **Memory**: Same as monolithic (all rules loaded into memory)
- **Runtime**: No difference (same execution engine)

Typical load times:
- Monolithic: ~50ms for 50K LOC
- Multi-file (50 files): ~75ms for 50K LOC
- Difference negligible for most applications

## Future Enhancements

Planned features:
- **Lazy loading**: Load files on-demand
- **Caching**: Cache parsed files
- **Parallel loading**: Load files concurrently
- **Selective loading**: Load only specified modules
- **Hot reload**: Reload changed files without restart

## Examples

See test files for working examples:
- `/go/pkg/dtrules/taxreturn_results_test.go` - Multi-file loading
- `/go/pkg/dtrules/sampleprojects_test.go` - Backward compatible loading
- `/go/pkg/dtrules/loader/directory_test.go` - Directory loading tests

## Support

For questions or issues:
- Check existing tests for examples
- Review error messages (they include file paths)
- Verify file numbering and metadata
- Ensure XML is well-formed

## Summary

**Recommended approach for new code:**
```go
rs := session.NewRuleSet("MyRules")
err := rs.LoadFromDirectory("./xml")
```

**Backward compatible approach:**
```go
rs := session.NewRuleSet("MyRules")
err := rs.LoadFromDirectory(xmlDir)
if err != nil {
    // Fallback to monolithic files
    rs.LoadEDD(eddFile)
    rs.LoadDecisionTables(dtFile)
}
```

The multi-file structure is the recommended approach going forward, but monolithic files remain fully supported for backward compatibility.
