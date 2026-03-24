# Directory-Based XML Loading

## Overview

The RuleSet API now supports loading XML files from directories, in addition to the existing reader-based API. This makes it easier to work with multi-file rule sets, particularly for complex applications like state tax calculations.

## API Reference

### RuleSet Methods

#### LoadFromDirectory
```go
func (rs *RuleSet) LoadFromDirectory(dirPath string) error
```

Loads all XML files from a directory recursively. Files are automatically categorized as EDD or Decision Table files and loaded in the correct order based on:
- **EDDs**: Ordered by the leading number in the `file_path` field (e.g., `40100_CA_constants`)
- **Decision Tables**: Ordered by `TABLE_NUMBER` attribute

**Example:**
```go
rs := session.NewRuleSet("TaxReturn")
err := rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")
if err != nil {
    log.Fatal(err)
}
```

#### LoadEDDFile
```go
func (rs *RuleSet) LoadEDDFile(filePath string) error
```

Loads an EDD from a file path (convenience wrapper for `LoadEDD`).

**Example:**
```go
err := rs.LoadEDDFile("./xml/TaxReturn_edd.xml")
```

#### LoadDecisionTablesFile
```go
func (rs *RuleSet) LoadDecisionTablesFile(filePath string) error
```

Loads decision tables from a file path (convenience wrapper for `LoadDecisionTables`).

**Example:**
```go
err := rs.LoadDecisionTablesFile("./xml/TaxReturn_dt.xml")
```

#### LoadFromPath
```go
func (rs *RuleSet) LoadFromPath(path, eddPath, dtPath string) error
```

Flexible loader that can handle both directory and individual file paths:
- If `eddPath` and `dtPath` are provided, loads individual files
- If only `path` is provided and it's a directory, loads all XML files
- Auto-detects the appropriate loading method

**Examples:**
```go
// Load from directory
rs.LoadFromPath("./xml", "", "")

// Load from individual files
rs.LoadFromPath("", "./xml/TaxReturn_edd.xml", "./xml/TaxReturn_dt.xml")
```

### Package-Level Convenience Functions

#### LoadRulesFromDirectory
```go
func LoadRulesFromDirectory(name, dirPath string) (*RuleSet, error)
```

Creates a new RuleSet and loads all XML files from a directory.

**Example:**
```go
rs, err := session.LoadRulesFromDirectory("TaxReturn", "./xml")
if err != nil {
    log.Fatal(err)
}
sess, _ := rs.NewSession()
```

#### LoadRulesFromFiles
```go
func LoadRulesFromFiles(name, eddPath, dtPath string) (*RuleSet, error)
```

Creates a new RuleSet and loads EDD and DT from individual files.

**Example:**
```go
rs, err := session.LoadRulesFromFiles("TaxReturn",
    "./xml/TaxReturn_edd.xml",
    "./xml/TaxReturn_dt.xml")
```

## Migration Guide

### Before: Individual Files

```go
rs := session.NewRuleSet("TaxReturn")

// Load EDD
eddFile, _ := os.Open("./xml/TaxReturn_edd.xml")
defer eddFile.Close()
rs.LoadEDD(eddFile)

// Load DT
dtFile, _ := os.Open("./xml/TaxReturn_dt.xml")
defer dtFile.Close()
rs.LoadDecisionTables(dtFile)

sess, _ := rs.NewSession()
```

### After: Directory Loading

```go
// Option 1: Using method
rs := session.NewRuleSet("TaxReturn")
rs.LoadFromDirectory("./xml")
sess, _ := rs.NewSession()

// Option 2: Using convenience function
rs, _ := session.LoadRulesFromDirectory("TaxReturn", "./xml")
sess, _ := rs.NewSession()

// Option 3: Still supports individual files
rs, _ := session.LoadRulesFromFiles("TaxReturn",
    "./xml/TaxReturn_edd.xml",
    "./xml/TaxReturn_dt.xml")
```

## File Organization

The directory loader expects XML files with specific metadata:

### EDD Files
Must contain `<entity_data_dictionary>` root element with optional metadata:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <file_metadata>
    <file_path>states/CA/40100_CA_constants</file_path>
  </file_metadata>
  <!-- entities -->
</entity_data_dictionary>
```

The leading number in `file_path` (40100) determines loading order.

### Decision Table Files
Must contain `<decision_tables>` root element with tables containing `TABLE_NUMBER`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
  <decision_table>
    <table_name>Calculate_CA_Tax</table_name>
    <attribute_fields>
      <TABLE_NUMBER>40200</TABLE_NUMBER>
    </attribute_fields>
    <!-- table content -->
  </decision_table>
</decision_tables>
```

The `TABLE_NUMBER` determines loading order.

## Directory Structure Example

```
sampleprojects/TaxReturn/xml/
├── TaxReturn_edd.xml          # Core EDD (loaded first, no number = 0)
├── TaxReturn_dt.xml           # Core DTs (tables with TABLE_NUMBER)
└── states/
    ├── CA_edd.xml             # California constants (40100)
    ├── CA_dt.xml              # California tables (40200+)
    ├── NY_edd.xml             # New York constants (41100)
    └── NY_dt.xml              # New York tables (41200+)
```

All files are loaded automatically in the correct order when you call:
```go
rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")
```

## Ordering and Dependencies

Files are loaded in this order:
1. All EDD files, sorted by their `file_path` number (lowest to highest)
2. All Decision Table files, sorted by `TABLE_NUMBER` (lowest to highest)

This ensures:
- Constants and entities are defined before tables that use them
- Core definitions load before state-specific extensions
- Tables with dependencies load after their prerequisites

## Error Handling

The loader will:
- Skip files that can't be parsed (with a warning)
- Return an error if no valid XML files are found
- Aggregate errors from multiple files into a single error message

```go
err := rs.LoadFromDirectory("./xml")
if err != nil {
    // err contains details about all files that failed to load
    log.Fatalf("Failed to load rules: %v", err)
}
```

## Backward Compatibility

All existing reader-based APIs remain unchanged:
- `LoadEDD(r io.Reader)` - still works
- `LoadDecisionTables(r io.Reader)` - still works

The new file/directory APIs are pure additions and don't affect existing code.

## Implementation Details

The directory loading is implemented in:
- `/go/pkg/dtrules/loader/directory.go` - Core directory scanning and loading
- `/go/pkg/dtrules/session/ruleset.go` - RuleSet integration methods

The `LoadRulesFromDirectory` function in the loader package handles:
- Recursive directory scanning
- XML file detection and metadata parsing
- Ordering by TABLE_NUMBER and file_path numbering
- Sequential loading (EDDs first, then DTs)
