# Directory Loading Implementation

## Overview

The `LoadFromDirectory()` method enables loading DTRules from a directory containing multiple XML files, automatically handling load order based on file numbering.

## Quick Start

```go
rs := session.NewRuleSet("TaxReturn")
err := rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")
```

## How It Works

1. Scans directory recursively for `*.xml` files
2. Parses metadata to determine file type and load order
3. Sorts by numbering (EDDs by file_path, DTs by TABLE_NUMBER)
4. Loads EDDs first (in order), then Decision Tables (in order)

## File Requirements

### EDD Files
```xml
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/AL/40100_AL_constants</file_path>
  </file_metadata>
  ...
</entity_data_dictionary>
```

### DT Files
```xml
<decision_table>
  <attribute_fields>
    <TABLE_NUMBER>40100</TABLE_NUMBER>
    <FILE_PATH>states/AL/40100_Calculate_AL_Tax</FILE_PATH>
  </attribute_fields>
  ...
</decision_table>
```

## See Also

- [Multi-File Migration Guide](../MULTI_FILE_MIGRATION.md)
- [Tests](../multifile_quick_test.go)
