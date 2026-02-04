# DTRules Compiler Utilities

Excel-to-XML compilation utilities and test support for DTRules.

## Overview

This module provides:
- **Spreadsheet to XML converters** for entity definitions and decision tables
- **Multi-format support**: Excel (.xls, .xlsx, .xlsm), OpenDocument (.ods), Google Sheets
- **Excel output**: Export rules back to Excel format
- Test harness base classes
- Execution tracing support
- Deployment utilities

## Supported Spreadsheet Formats

| Format | Extension | Library | Description |
|--------|-----------|---------|-------------|
| Excel 97-2003 | `.xls` | Apache POI (HSSF) | Legacy binary format |
| Excel 2007+ | `.xlsx` | Apache POI (XSSF) | Modern XML-based format |
| Excel with Macros | `.xlsm` | Apache POI (XSSF) | XML format with macro support |
| OpenDocument | `.ods` | Simple ODF | LibreOffice/OpenOffice format |
| Google Sheets | URL | Google Sheets API | Cloud-based spreadsheets |

## Quick Start

### Reading from Excel Files

```java
// Standard compilation from Excel
Excel2XML compiler = new Excel2XML(path, "DTRules.xml", "RuleSetName");
compiler.compileRuleSet(path, "DTRules.xml", "RuleSetName",
                        "repository", maps, 5);
```

### Reading from Google Sheets

```java
ImportRuleSets importer = new ImportRuleSets();

// Option 1: Service account credentials
importer.setGoogleCredentialsPath("/path/to/service-account.json");

// Option 2: Application Default Credentials
// Set GOOGLE_APPLICATION_CREDENTIALS environment variable
// Or run: gcloud auth application-default login

// Convert from Google Sheets URL
importer.convertEDD(entityFactory, ruleSet,
    "https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit");
```

### Writing to Excel

```java
// Create Excel output (XLSX format)
Rules2Excel r2e = new Rules2Excel(false); // false = non-balanced mode
r2e.writeExcel(admin, ruleset, "output", sortFields, ascending, limit);
```

## Key Classes

### ImportRuleSets

Imports spreadsheets and converts them to XML. Supports all formats.

```java
ImportRuleSets importer = new ImportRuleSets();

// For Google Sheets, set credentials first
importer.setGoogleCredentialsPath("/path/to/credentials.json");

// Convert EDD from any supported format
importer.convertEDDs(ruleSet, "edd_source", "output.xml");

// Convert Decision Tables
importer.convertDecisionTables(ruleSet, "output_dt.xml");
```

**Supported Sources:**
- Local file paths: `/path/to/file.xlsx`
- Google Sheets URLs: `https://docs.google.com/spreadsheets/d/ID/edit`

### Excel2XML

High-level compiler that orchestrates the full conversion process.

```java
Excel2XML compiler = new Excel2XML(path, "DTRules.xml", "RuleSetName");
compiler.convertRuleset();  // Convert Excel to XML
compiler.compile(10, System.out);  // Compile with error reporting
```

### Rules2Excel

Exports rules back to Excel format (XLSX).

```java
Rules2Excel exporter = new Rules2Excel(balanced);
exporter.writeExcel(admin, ruleset, "output", fields, ascending, limit);
```

### GoogleSheetsReader

Direct access to Google Sheets API.

```java
// With service account
GoogleSheetsReader reader = new GoogleSheetsReader("/path/to/credentials.json");

// With Application Default Credentials
GoogleSheetsReader reader = new GoogleSheetsReader();

// Read data
String spreadsheetId = GoogleSheetsReader.extractSpreadsheetId(url);
List<List<Object>> data = reader.readSheet(spreadsheetId, "Sheet1");

// Helper methods
String value = GoogleSheetsReader.getCellValue(data, row, col);
int rowCount = GoogleSheetsReader.getRowCount(data);
```

### ATestHarness

Base class for running decision table tests:

```java
public class TestMyRules extends ATestHarness {
    public static void main(String[] args) throws Exception {
        TestMyRules t = new TestMyRules();
        t.load(path + "/xml/testParms.xml");
        t.runTests();
    }
}
```

## Excel File Formats

### Entity Definition Document (EDD)

Defines the data model. Required columns:

| Column | Required | Description |
|--------|----------|-------------|
| Entity | Yes | Entity name |
| Attribute | Yes | Attribute name |
| Type | Yes | Data type (string, integer, boolean, etc.) |
| SubType | No | Additional type information |
| Default Value | Yes | Default value for attribute |
| Input | Yes | Input source identifier |
| Access | Yes | Access mode (r, w, rw) |
| Comment | No | Documentation |
| Source | No | Source reference |

### Decision Tables

Defines business rules. Structure:

```
Name: Table_Name
Type: All|First

Contexts:
  [context expressions]

Initial_Actions:
  [setup actions]

Conditions:
  # | Comment | Condition Expression | 1 | 2 | 3 | ...
  1 | desc    | x > 0               | Y | N | - |

Actions:
  # | Comment | Action Expression    | 1 | 2 | 3 | ...
  1 | desc    | result = true       | X |   | X |

Policy_Statements:
  [column descriptions]
```

## Compilation Process

```
Spreadsheet Files              XML Files
┌──────────────────┐          ┌──────────────┐
│  EDD.xlsx        │  ──────▶ │  *_edd.xml   │
│  EDD.ods         │          └──────────────┘
│  Google Sheet    │
└──────────────────┘
┌──────────────────┐          ┌──────────────┐
│  DT_*.xlsx       │  ──────▶ │  *_dt.xml    │
│  DT_*.ods        │          └──────────────┘
│  Google Sheets   │
└──────────────────┘          ┌──────────────┐
                              │  *_map.xml   │
                              └──────────────┘
```

## Google Sheets Setup

### Option 1: Service Account (Recommended for Automation)

1. Create a Google Cloud project
2. Enable the Google Sheets API
3. Create a service account and download JSON key
4. Share your spreadsheet with the service account email
5. Use the JSON key path in your code

### Option 2: Application Default Credentials (Development)

```bash
# Install gcloud CLI, then:
gcloud auth application-default login
```

### Option 3: Environment Variable

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Apache POI | 5.2.5 | Excel file support (.xls, .xlsx, .xlsm) |
| Apache POI OOXML | 5.2.5 | Modern Excel format support |
| Simple ODF | 0.9.0 | OpenDocument format support (.ods) |
| Google API Client | 2.2.0 | Google Sheets API |
| Google Sheets API | v4 | Google Sheets access |
| Google Auth | 1.20.0 | Google authentication |

## Building

```bash
mvn clean install
```

## Testing

```bash
# Run all tests
mvn test -pl compilerutil

# Run specific test class
mvn test -pl compilerutil -Dtest=SpreadsheetReadWriteTest

# Run with Google Sheets integration tests
GOOGLE_APPLICATION_CREDENTIALS=/path/to/creds.json mvn test -pl compilerutil
```

## See Also

- [Spreadsheet Formats Guide](../docs/SPREADSHEET-FORMATS.md) - Detailed format documentation
- [API Guide](../docs/API-GUIDE.md) - Java integration patterns
- [Quick Start](../docs/QUICKSTART.md) - Getting started guide
