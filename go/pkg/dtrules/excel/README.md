# DTRules Excel Package

This package provides bidirectional Excel import/export for DTRules decision tables and Entity Data Dictionaries (EDD).

## Overview

The Excel package enables a round-trip workflow where business analysts can:
1. Export rules to Excel for review and editing
2. Make changes in a familiar spreadsheet interface
3. Import changes back to XML

## Components

### Exporters (XML → Excel)

- **`exporter.go`** - Exports decision tables and EDD to Excel format
  - `ExportDecisionTables()` - Export all DTs to a single workbook
  - `ExportDecisionTablesToDir()` - Export DTs grouped by source file
  - `ExportEDD()` - Export entity definitions to Excel
  - `ExportEDDToDir()` - Export EDD grouped by source file

### Importers (Excel → XML)

- **`dt_importer.go`** - Import decision tables from Excel
  - `ImportDecisionTables()` - Import from a single Excel file
  - `ImportDecisionTablesFromDir()` - Recursively import from directory

- **`edd_importer.go`** - Import EDD from Excel
  - `ImportEDD()` - Import from a single Excel file
  - `ImportEDDFromDir()` - Recursively import from directory

- **`workbook_importer.go`** - Combined workbook import
  - `ImportWorkbook()` - Import both DT and EDD from a single Excel file
  - `ImportDirectory()` - Recursively process all Excel files
  - `WriteAll()` - Write results mirroring Excel directory structure

## Directory Structure

Excel directory structure drives XML organization:

```
excel/                          xml/
├── TaxReturn.xlsx         →    ├── TaxReturn_dt.xml
│   (DT + EDD sheets)           ├── TaxReturn_edd.xml
└── states/                     └── states/
    ├── CO.xlsx            →        ├── CO_dt.xml
    │                               ├── CO_edd.xml
    ├── CA.xlsx            →        ├── CA_dt.xml
    │                               ├── CA_edd.xml
    └── ...                         └── ...
```

## Sheet Detection

The workbook importer automatically detects sheet types:

- **EDD sheets**: Named "EDD" or contain entity/attribute headers
- **DT sheets**: Contain "Decision Table", "Name:", or decision table structure

## CLI Tools

### excel2dt

Import decision tables from Excel:

```bash
excel2dt -input excel/ -output xml/TaxReturn_dt.xml
excel2dt -input tables.xlsx -output xml/Tables_dt.xml -v
```

### excel2edd

Import EDD from Excel:

```bash
excel2edd -input excel/TaxReturn_edd.xlsx -output xml/TaxReturn_edd.xml
excel2edd -dir excel/ -output xml/TaxReturn_edd.xml
```

## Usage Examples

### Import Combined Workbook

```go
importer := excel.NewWorkbookImporter()
importer.SetVerbose(true)

// Import all Excel files recursively
results, err := importer.ImportDirectory("excel/")
if err != nil {
    log.Fatal(err)
}

// Write XML files mirroring directory structure
err = importer.WriteAll(results, "xml/")
```

### Export to Excel

```go
exporter := excel.NewExporter(ruleSet)

// Export decision tables
err := exporter.ExportDecisionTablesToDir("excel/")

// Export EDD
err = exporter.ExportEDDToDir("excel/")
```

## File Metadata

When importing, the `xls_file` attribute is set to the relative path:
- `excel/states/CO.xlsx` → `xls_file="states/CO.xlsx"`

This metadata is preserved through round-trips.
