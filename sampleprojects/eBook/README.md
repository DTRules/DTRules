# eBook - Multi-Ruleset Business Logic

A sample DTRules project demonstrating multiple rule sets and the EBL (Entity Business Language).

## Overview

This project implements business logic for an eBook platform, including book previews, returns, and sales. It demonstrates how to organize multiple related rule sets within a single project and uses the enhanced EBL language instead of the standard EL.

## What This Demonstrates

- **Multiple rule sets** in one project (BookPreview, BookReturns, BookSells)
- **EBL (Entity Business Language)** - enhanced DSL
- Complex domain modeling (books, chapters, pages, customers, publishers)
- Separate compilation and testing for each rule set
- Real-world business scenario implementation

## Project Structure

```
eBook/
├── DTRules_eBooks.xml             # Main configuration (multi-ruleset)
├── rs_preview/                    # BookPreview ruleset
│   ├── DecisionTables/
│   │   └── BookPreview_dt.xls
│   ├── edd/
│   │   └── BookPreview_edd.xls
│   └── xml/
├── rs_returns/                    # BookReturns ruleset
│   ├── DecisionTables/
│   │   └── BookReturns_dt.xls
│   ├── edd/
│   │   └── BookReturns_edd.xls
│   └── xml/
├── rs_sells/                      # BookSells ruleset
│   ├── DecisionTables/
│   │   └── BookSells_dt.xls
│   ├── edd/
│   │   └── BookSells_edd.xls
│   └── xml/
├── testfiles/                     # Test files for each ruleset
└── src/main/java/
    └── com/dtrules/samples/bookpreview/
        ├── Compile_BookPreview.java
        ├── Test_BookPreview.java
        ├── TestCaseGen_BookPreview.java
        ├── Compile_BookReturns.java
        ├── Test_BookReturns.java
        ├── TestCaseGen_BookReturns.java
        ├── Compile_BookSells.java
        ├── Test_BookSells.java
        └── TestCaseGen_BookSells.java
```

## Rule Sets

### BookPreview
Determines book preview eligibility:
- Which chapters/pages can be previewed
- Customer access levels
- Preview limitations

### BookReturns
Handles book return processing:
- Return eligibility
- Refund calculations
- Return policy enforcement

### BookSells
Manages book sales logic:
- Pricing rules
- Discounts
- Purchase validation

## Entities

| Entity | Description |
|--------|-------------|
| `Book` | Book with publisher, title, chapters |
| `Chapter` | Chapter with pages |
| `Page` | Individual page content |
| `Publisher` | Publisher information |
| `Customer` | Customer with demographics and purchase history |
| `Request` | Book request from customer |
| `Open_Book` | Currently opened book state |
| `ABookObj` | Abstract base for book objects |
| `DataObj` | Data wrapper entity |

## EBL (Entity Business Language)

This project uses **EBL** instead of the standard EL. EBL provides:
- Enhanced syntax for entity operations
- More expressive business logic constructs
- Better support for complex domain models

Configuration in `DTRules_eBooks.xml`:
```xml
<compileralias name="EBL">com.dtrules.compiler.ebl.EBL</compileralias>
<compiler>EBL</compiler>
```

## Running This Sample

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile BookPreview

```bash
cd sampleprojects/eBook
mvn exec:java -Dexec.mainClass="com.dtrules.samples.bookpreview.Compile_BookPreview"
```

### Run BookPreview Tests

```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.bookpreview.Test_BookPreview"
```

### Other Rule Sets

Replace `BookPreview` with `BookReturns` or `BookSells` in the class names above.

## Configuration

`DTRules_eBooks.xml` (excerpt):
```xml
<DTRules>
  <compileralias name="EBL">com.dtrules.compiler.ebl.EBL</compileralias>
  <compiler>EBL</compiler>

  <RuleSet name="BookPreview" source="file">
    <RuleSetFilePath>/rs_preview/xml/</RuleSetFilePath>
    <WorkingDirectory>/rs_preview/temp/</WorkingDirectory>
    <DTExcelFolder>/rs_preview/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/rs_preview/edd/</EDDExcelFolder>
    <Entities        name="BookPreview_edd.xml" />
    <Decisiontables  name="BookPreview_dt.xml"  />
    <Map             name="BookPreview_map.xml" />
  </RuleSet>

  <!-- Additional rulesets for BookReturns, BookSells -->
</DTRules>
```

## Multi-Ruleset Architecture

This project demonstrates how to:

1. **Organize rule sets** - Each in its own directory (`rs_preview/`, `rs_returns/`, `rs_sells/`)
2. **Share configuration** - Single `DTRules_eBooks.xml` defines all rule sets
3. **Independent compilation** - Each rule set compiled separately
4. **Selective execution** - Load and execute specific rule sets as needed

## Related Projects

- **eBookApp** - Standalone application wrapper for eBook rules

## DSL

Uses **EBL (Entity Business Language)** - an enhanced version of EL.
