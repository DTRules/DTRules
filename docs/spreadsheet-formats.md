# Spreadsheet Formats Guide

DTRules supports multiple spreadsheet formats for defining Entity Definition Documents (EDD) and Decision Tables. This guide covers all supported formats, their features, and usage patterns.

## Table of Contents

1. [Supported Formats Overview](#supported-formats-overview)
2. [Excel Formats (.xls, .xlsx, .xlsm)](#excel-formats)
3. [OpenDocument Format (.ods)](#opendocument-format)
4. [Google Sheets](#google-sheets)
5. [Format Comparison](#format-comparison)
6. [Migration Guide](#migration-guide)
7. [Troubleshooting](#troubleshooting)

---

## Supported Formats Overview

| Format | Extension | Read | Write | Cloud | Notes |
|--------|-----------|------|-------|-------|-------|
| Excel 97-2003 | `.xls` | ✅ | ❌ | ❌ | Legacy, max 65,536 rows |
| Excel 2007+ | `.xlsx` | ✅ | ✅ | ❌ | Recommended for new projects |
| Excel with Macros | `.xlsm` | ✅ | ❌ | ❌ | Macros ignored during conversion |
| OpenDocument | `.ods` | ✅ | ❌ | ❌ | LibreOffice/OpenOffice |
| Google Sheets | URL | ✅ | ❌ | ✅ | Requires API credentials |

---

## Excel Formats

### Excel 2007+ (.xlsx) - Recommended

The modern Excel format based on Open XML. This is the recommended format for new projects.

**Advantages:**
- Large file support (1,048,576 rows × 16,384 columns)
- Smaller file sizes (compressed XML)
- Better compatibility with modern tools
- Full formatting support

**Usage:**
```java
// Automatic format detection
ImportRuleSets importer = new ImportRuleSets();
importer.convertEDDs(ruleSet, "definitions.xlsx", "output_edd.xml");
```

### Excel 97-2003 (.xls) - Legacy

The binary Excel format. Supported for backward compatibility.

**Limitations:**
- Maximum 65,536 rows × 256 columns
- Larger file sizes
- Some modern features unavailable

**Usage:**
```java
// Same API - format auto-detected
importer.convertEDDs(ruleSet, "definitions.xls", "output_edd.xml");
```

### Excel with Macros (.xlsm)

Excel files containing VBA macros. Macros are ignored during conversion.

**Note:** If your spreadsheet uses macros for data validation or generation, ensure the final data is present in the cells before conversion.

---

## OpenDocument Format

### ODS (.ods)

The OpenDocument Spreadsheet format used by LibreOffice, OpenOffice, and Google Sheets (export).

**Advantages:**
- Open standard (ISO/IEC 26300)
- Free software compatibility
- Good interoperability

**Usage:**
```java
ImportRuleSets importer = new ImportRuleSets();
importer.convertEDDs(ruleSet, "definitions.ods", "output_edd.xml");
```

**Creating ODS Files:**
- LibreOffice Calc: File → Save As → ODS
- Google Sheets: File → Download → OpenDocument format (.ods)
- Microsoft Excel: File → Save As → OpenDocument Spreadsheet

---

## Google Sheets

Cloud-based spreadsheets with real-time collaboration support.

### Setup Options

#### Option 1: Service Account (Recommended for Production)

Best for automated systems and CI/CD pipelines.

1. **Create Google Cloud Project:**
   ```
   https://console.cloud.google.com/
   ```

2. **Enable Google Sheets API:**
   ```
   APIs & Services → Enable APIs → Google Sheets API
   ```

3. **Create Service Account:**
   ```
   IAM & Admin → Service Accounts → Create Service Account
   ```

4. **Download JSON Key:**
   ```
   Service Account → Keys → Add Key → JSON
   ```

5. **Share Spreadsheet:**
   - Open your Google Sheet
   - Click Share
   - Add the service account email (from JSON file)
   - Grant "Viewer" or "Editor" access

6. **Use in Code:**
   ```java
   ImportRuleSets importer = new ImportRuleSets();
   importer.setGoogleCredentialsPath("/path/to/service-account.json");

   String sheetsUrl = "https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit";
   importer.convertEDDs(entityFactory, ruleSet, sheetsUrl);
   ```

#### Option 2: Application Default Credentials (Development)

Best for local development and testing.

1. **Install Google Cloud CLI:**
   ```bash
   # macOS
   brew install google-cloud-sdk

   # Linux
   curl https://sdk.cloud.google.com | bash

   # Windows
   # Download installer from https://cloud.google.com/sdk/docs/install
   ```

2. **Authenticate:**
   ```bash
   gcloud auth application-default login
   ```

3. **Use in Code:**
   ```java
   // No credentials path needed - uses ADC automatically
   ImportRuleSets importer = new ImportRuleSets();
   importer.convertEDDs(entityFactory, ruleSet, sheetsUrl);
   ```

#### Option 3: Environment Variable

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
```

### Google Sheets URL Format

DTRules accepts standard Google Sheets URLs:

```
https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit
https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit#gid=SHEET_ID
https://docs.google.com/spreadsheets/d/SPREADSHEET_ID
```

The `SPREADSHEET_ID` is extracted automatically.

### Direct API Access

For advanced use cases, use `GoogleSheetsReader` directly:

```java
GoogleSheetsReader reader = new GoogleSheetsReader("/path/to/credentials.json");

// Get spreadsheet metadata
Spreadsheet spreadsheet = reader.getSpreadsheet(spreadsheetId);
List<Sheet> sheets = reader.getSheets(spreadsheetId);

// Read data
List<List<Object>> data = reader.readSheet(spreadsheetId, "Sheet1");
List<List<Object>> range = reader.readRange(spreadsheetId, "Sheet1!A1:D10");

// Helper methods
String value = GoogleSheetsReader.getCellValue(data, row, col);
int rows = GoogleSheetsReader.getRowCount(data);
int cols = GoogleSheetsReader.getColumnCount(data, row);
```

---

## Format Comparison

### Feature Matrix

| Feature | XLS | XLSX | XLSM | ODS | Google Sheets |
|---------|-----|------|------|-----|---------------|
| Max Rows | 65,536 | 1M+ | 1M+ | 1M+ | 10M cells |
| Max Columns | 256 | 16,384 | 16,384 | 1,024 | 18,278 |
| Formulas | ✅ | ✅ | ✅ | ✅ | ✅ |
| Cell Formatting | ✅ | ✅ | ✅ | ✅ | ✅ |
| Multiple Sheets | ✅ | ✅ | ✅ | ✅ | ✅ |
| Real-time Collab | ❌ | ❌ | ❌ | ❌ | ✅ |
| Offline Access | ✅ | ✅ | ✅ | ✅ | Limited |
| Version Control | Manual | Manual | Manual | Manual | Built-in |

### Recommended Use Cases

| Use Case | Recommended Format |
|----------|-------------------|
| New projects | `.xlsx` |
| Team collaboration | Google Sheets |
| Open source projects | `.ods` |
| Legacy system integration | `.xls` |
| Automated pipelines | `.xlsx` or Google Sheets |
| Large datasets (>65K rows) | `.xlsx` |

---

## Migration Guide

### From .xls to .xlsx

1. Open the .xls file in Excel or LibreOffice
2. Save As → Excel Workbook (.xlsx)
3. Update your code paths (format auto-detected)

### From Excel to Google Sheets

1. Upload Excel file to Google Drive
2. Open with Google Sheets
3. Share with service account (if using automation)
4. Update code to use Google Sheets URL

### From Google Sheets to Local File

1. File → Download → Microsoft Excel (.xlsx) or OpenDocument (.ods)
2. Update code to use local file path

---

## Troubleshooting

### Common Issues

#### "Could not find artifact" for Google Sheets API

Ensure you have the correct dependency version:
```xml
<dependency>
    <groupId>com.google.apis</groupId>
    <artifactId>google-api-services-sheets</artifactId>
    <version>v4-rev20251110-2.0.0</version>
</dependency>
```

#### "The caller does not have permission" (Google Sheets)

1. Verify the spreadsheet is shared with your service account email
2. Check that the Google Sheets API is enabled in your project
3. Ensure credentials file path is correct

#### "Invalid Google Sheets URL"

URL must match pattern:
```
https://docs.google.com/spreadsheets/d/[SPREADSHEET_ID]...
```

#### "Couldn't find column headers" (EDD)

Ensure your EDD spreadsheet has these required headers in row 1:
- Entity
- Attribute
- Type
- Default Value (or DefaultValue)
- Input
- Access

#### Formula cells showing wrong values

Formulas are evaluated at the time the spreadsheet was last saved. For Google Sheets, values are retrieved as displayed. Ensure formulas are calculated before conversion.

### Debug Logging

Enable verbose logging to diagnose issues:

```java
Excel2XML compiler = new Excel2XML(path, "DTRules.xml", "RuleSetName");
compiler.verbose = true;  // Enable detailed output
```

---

## API Reference

### ImportRuleSets

```java
// Constructor
ImportRuleSets importer = new ImportRuleSets();

// Google Sheets credentials
void setGoogleCredentialsPath(String path)

// Format detection
static boolean isGoogleSheetsSource(String source)

// Conversion methods
void convertEDDs(RuleSet rs, String source, String outputXml)
void convertDecisionTables(RuleSet rs, String outputXml)
boolean convertDecisionTable(StringBuffer data, File file, XMLPrinter out, int depth)
boolean convertDecisionTableFromGoogleSheets(StringBuffer data, String url, XMLPrinter out, int depth)
```

### GoogleSheetsReader

```java
// Constructors
GoogleSheetsReader()  // Uses Application Default Credentials
GoogleSheetsReader(String credentialsPath)  // Uses service account
GoogleSheetsReader(GoogleCredentials credentials)  // Uses provided credentials

// URL utilities
static String extractSpreadsheetId(String url)
static boolean isGoogleSheetsUrl(String url)

// Read operations
Spreadsheet getSpreadsheet(String spreadsheetId)
List<Sheet> getSheets(String spreadsheetId)
List<List<Object>> readSheet(String spreadsheetId, String sheetName)
List<List<Object>> readRange(String spreadsheetId, String range)
List<List<Object>> readFirstSheet(String spreadsheetId)

// Data helpers
static String getCellValue(List<List<Object>> data, int row, int col)
static int getRowCount(List<List<Object>> data)
static int getColumnCount(List<List<Object>> data, int row)
```

---

## See Also

- [Compiler Utilities README](../compilerutil/README.md) - Module overview
- [Quick Start Guide](quickstart.md) - Getting started
- [API Guide](api-guide.md) - Java integration patterns
- [Architecture Guide](architecture.md) - System design
