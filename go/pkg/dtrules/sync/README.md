# DTRules Sync Package

This package provides bidirectional synchronization between Excel and XML files for DTRules, integrated into the compilation workflow.

## Overview

The sync package ensures Excel and XML stay synchronized:
- **Pre-compile**: If Excel is newer, import changes to XML
- **Post-compile**: Regenerate Excel to maintain consistent formatting

## Key Features

- **Timestamp-based detection** - Compares file modification times
- **Recursive directory support** - Handles subdirectories like `states/`
- **Combined workbooks** - Single Excel file can contain both DT and EDD
- **Conflict handling** - Configurable resolution strategies
- **Dry run mode** - Preview changes without modifying files

## Usage

### Basic Sync

```go
import "github.com/DTRules/DTRules/go/pkg/dtrules/sync"

// Simple compilation with sync
result, err := sync.CompileWithSync("TaxReturn", xmlDir, excelDir)
if err != nil {
    log.Fatal(err)
}
ruleSet := result.RuleSet
```

### With Options

```go
opts := &sync.SyncOptions{
    ConflictResolution: "prefer-excel",  // or "prefer-xml", "error", "skip"
    DryRun:             false,
    GracePeriod:        2 * time.Second,
}
result, err := sync.CompileWithSyncOptions("TaxReturn", xmlDir, excelDir, opts)
```

### Manual Sync Operations

```go
syncer := sync.NewSyncer(xmlDir, excelDir)

// Check what needs syncing
direction, files, err := syncer.CheckSync()

// Sync Excel to XML (when Excel is newer)
err = syncer.SyncExcelToXML()

// Sync XML to Excel (after compilation)
err = syncer.SyncXMLToExcel()
```

## Directory Structure

The sync package preserves directory structure:

```
excel/                          xml/
├── TaxReturn.xlsx         ↔    ├── TaxReturn_dt.xml
│                               ├── TaxReturn_edd.xml
└── states/                     └── states/
    ├── CO.xlsx            ↔        ├── CO_dt.xml
    │                               ├── CO_edd.xml
    └── ...                         └── ...
```

## Sync Directions

| Direction | Meaning |
|-----------|---------|
| `NoSync` | Files are in sync |
| `ExcelToXML` | Excel is newer, import to XML |
| `XMLToExcel` | XML is newer, export to Excel |
| `Conflict` | Both modified, requires resolution |

## Conflict Resolution

| Strategy | Behavior |
|----------|----------|
| `"error"` | Return error on conflict (default) |
| `"prefer-excel"` | Always use Excel version |
| `"prefer-xml"` | Always use XML version |
| `"skip"` | Skip conflicting files |

## Combined Workbook Mode

When enabled, a single Excel file maps to multiple XML files:

```go
syncer := sync.NewSyncer(xmlDir, excelDir)
syncer.SetUseCombinedWorkbooks(true)

// excel/CO.xlsx ↔ xml/CO_dt.xml + xml/CO_edd.xml
```

## Integration with Compilation

The `SyncCompiler` wraps the compilation workflow:

```go
compiler := sync.NewSyncCompiler("TaxReturn", xmlDir, excelDir, nil)

// Pre-compile sync
err := compiler.PreCompileSync()

// ... compile XML ...

// Post-compile sync
err = compiler.PostCompileSync()
```

## Skipped Files

The following are automatically skipped:
- Test files (`testfiles/`)
- Schema files (`*_schema.xml`)
- Template files (`TEMPLATE_*.xml`)
- Mapping files (`*_map.xml`)
