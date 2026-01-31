# eBookApp - eBook Application Wrapper

A standalone application wrapping the eBook BookPreview rules.

## Overview

eBookApp wraps the eBook project's BookPreview rule set in a complete application framework. Similar to ChipApp, it demonstrates multi-threaded execution, configuration management, and result aggregation, but uses the EBL (Entity Business Language) instead of EL.

## What This Demonstrates

- **EBL integration** - Using enhanced business language rules
- **Application embedding** - Production-ready wrapper
- **Multi-threaded execution** - Parallel request processing
- **Domain modeling** - Book/Publisher/Customer entities
- **Configuration management** - External settings

## Project Structure

```
eBookApp/
├── DTRules_BookPreview.xml        # Rules configuration
├── settings.xml                   # Application settings
├── DecisionTables/                # Decision tables (if local)
├── edd/                           # Entity definitions (if local)
├── xml/                           # Compiled rules
│   ├── BookPreview_edd.xml
│   ├── BookPreview_dt.xml
│   └── BookPreview_map.xml
└── src/main/java/
    └── com/dtrules/samples/bookpreview/app/
        ├── BookPreviewApp.java    # Main application
        ├── EvaluateJob.java       # Job interface
        ├── EvaluateJobDTRules.java # DTRules implementation
        ├── RunThread.java         # Thread management
        ├── GenCase.java           # Test case generator
        ├── LoadSettings.java      # Settings loader
        └── dataobjects/           # Domain objects
            ├── DataObj.java
            ├── Request.java
            ├── Book.java
            ├── Customer.java
            └── Publisher.java
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    BookPreviewApp                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │  Settings   │  │  GenCase    │  │   RunThread     │ │
│  │  Loader     │  │  (Requests) │  │   Pool          │ │
│  └─────────────┘  └─────────────┘  └─────────────────┘ │
│                          │                   │          │
│                          ▼                   ▼          │
│                    ┌─────────────────────────────────┐  │
│                    │     EvaluateJobDTRules         │  │
│                    │  ┌────────────────────────┐    │  │
│                    │  │   EBL Rules Engine     │    │  │
│                    │  │   (BookPreview rules)  │    │  │
│                    │  └────────────────────────┘    │  │
│                    └─────────────────────────────────┘  │
│                                   │                      │
│                                   ▼                      │
│                         ┌─────────────────┐             │
│                         │  Preview        │             │
│                         │  Approval/Denial│             │
│                         └─────────────────┘             │
└─────────────────────────────────────────────────────────┘
```

## Domain Model

### Request Flow
```
Customer → Request → Book → Chapters → Pages
                       ↓
                   Publisher
```

### Entities

| Entity | Description |
|--------|-------------|
| `Request` | Customer's book preview request |
| `Customer` | Customer demographics and history |
| `Book` | Book with title and chapters |
| `Chapter` | Chapter containing pages |
| `Page` | Individual page content |
| `Publisher` | Book publisher information |

## Running This Sample

### Prerequisites

1. Build DTRules:
   ```bash
   cd /path/to/DTRules
   mvn clean install
   ```

2. Compile eBook rules (if not already done):
   ```bash
   cd sampleprojects/eBook
   mvn exec:java -Dexec.mainClass="com.dtrules.samples.bookpreview.Compile_BookPreview"
   ```

### Run the Application

```bash
cd sampleprojects/eBookApp
mvn exec:java -Dexec.mainClass="com.dtrules.samples.bookpreview.app.BookPreviewApp"
```

## Configuration

### settings.xml
```xml
<settings>
  <threads>4</threads>           <!-- Worker thread count -->
  <requests>1000</requests>      <!-- Number of requests to process -->
  <trace>false</trace>           <!-- Enable tracing -->
</settings>
```

### DTRules_BookPreview.xml
```xml
<DTRules>
  <compileralias name="EBL">com.dtrules.compiler.ebl.EBL</compileralias>
  <compiler>EBL</compiler>

  <RuleSet name="BookPreview" source="file">
    <RuleSetFilePath>/xml/</RuleSetFilePath>
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>
    <Entities        name="BookPreview_edd.xml" />
    <Decisiontables  name="BookPreview_dt.xml"  />
    <Map             name="BookPreview_map.xml" />
  </RuleSet>
</DTRules>
```

## EBL vs EL

This project uses **EBL (Entity Business Language)** which provides:
- Enhanced entity navigation syntax
- More expressive business logic constructs
- Better support for complex domain relationships

Compare the configuration:
```xml
<!-- EL (Expression Language) -->
<compiler>EL</compiler>

<!-- EBL (Entity Business Language) -->
<compileralias name="EBL">com.dtrules.compiler.ebl.EBL</compileralias>
<compiler>EBL</compiler>
```

## Business Logic

The BookPreview rules determine:
- Whether a customer can preview a book
- Which chapters/pages are accessible
- Preview duration and limitations
- Customer eligibility based on history

## Dependencies

This project depends on the **eBook** artifact:
```xml
<dependency>
  <groupId>com.dtrules</groupId>
  <artifactId>eBook</artifactId>
  <version>${project.version}</version>
</dependency>
```

## Related Projects

- **eBook** - The underlying multi-ruleset project
- **ChipApp** - Similar application pattern with EL

## DSL

Uses rules compiled with **EBL (Entity Business Language)**.
