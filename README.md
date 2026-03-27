# DTRules

A Decision Table based Rules Engine with implementations in **Java** and **Go**.

## Overview

DTRules is a production-ready rules engine that allows business analysts and policy experts to define complex business logic using **Decision Tables** in Excel spreadsheets, combined with a human-readable **Domain Specific Language (DSL)**.

Decision Tables provide a tabular representation of business rules that can be understood by both policy experts and developers. The DSL ensures that conditions and actions are written in clear, readable syntax rather than code.

### Key Features

- **Excel-based Rule Definition** - Define rules in familiar spreadsheet format
- **Domain Specific Languages** - EL (Expression Language) and EBL (Entity Business Language)
- **Entity Data Model** - Define your data structures in Excel
- **Auto-mapping** - Automatic mapping between Java objects and rule entities
- **Test Harness** - Built-in testing framework for validating rules
- **Trace & Debug** - Comprehensive tracing for debugging rule execution
- **ANTLR 4 Parsers** - Modern parser infrastructure (migrated from JFlex/CUP)

### Production Use

DTRules has been used in production systems including:
- State welfare/assistance programs (Texas TIERS, Ohio OFAST)
- Insurance eligibility determination systems
- Commercial business logic applications

## Implementations

| Implementation | Status | Use Case |
|----------------|--------|----------|
| **[Java](dtrules-engine/)** | Production | Original implementation with full tooling |
| **[Go](go/)** | Production | High-performance runtime, 130x faster operator lookup |

### Java Implementation

The original Java implementation provides:
- Excel-to-XML compilers for decision tables
- Full IDE support and debugging
- Comprehensive test harness
- Production-proven stability

### Go Implementation

The Go implementation provides:
- **Full compatibility** with Java DTRules XML formats
- **High performance**: 130x faster operator lookup, 24x faster arithmetic
- **Zero-allocation** hot paths for reduced GC pressure
- **CLI tool** for rule validation and execution

See [go/README.md](go/README.md) for Go-specific documentation.

## Requirements

### Java

- **Java 8+** (built with Java 1.6 target for compatibility)
- **Apache Maven 3.x**

### Go

- **Go 1.21+**

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/PaulSnow/DTRules.git
cd DTRules
mvn clean install
```

This builds all modules:
- `dtrules-engine` - Core rules engine
- `compilerutil` - Excel to XML compiler utilities
- `dsl` - Domain Specific Language implementations (EL, EBL)
- `sampleprojects` - Example projects

### 2. Run a Sample Project

The easiest way to understand DTRules is to run one of the sample projects.

#### CHIP Example (Health Insurance Eligibility)

```bash
cd sampleprojects/CHIP

# First, compile the Excel decision tables to XML
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.CompileChip"

# Then run the tests
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestChip"
```

### 3. Explore the Sample Project Structure

```
sampleprojects/CHIP/
├── DecisionTables/           # Excel files with business rules
│   └── ChipEligibility_dt.xls
├── edd/                      # Entity Definition Documents
│   └── ChipEligibility_edd.xls
├── xml/                      # Compiled XML rule files (generated)
├── repository/               # Packaged rules for deployment
├── testfiles/                # Test case input files
├── DTRules.xml               # Project configuration
└── src/main/java/            # Java code
    └── CompileChip.java      # Compiles Excel → XML
    └── TestChip.java         # Runs test cases
```

## Project Structure

```
DTRules/
├── dtrules-engine/     # Core Java rules execution engine
├── compilerutil/       # Excel-to-XML compiler utilities
├── dsl/                # Domain Specific Languages
│   ├── el/             # Expression Language (EL) - ANTLR 4
│   ├── ebl/            # Entity Business Language (EBL) - ANTLR 4
│   └── sudoku_language/ # Sudoku DSL (JFlex/CUP)
├── go/                 # Go implementation
│   ├── cmd/dtrules/    # CLI tool
│   ├── cmd/api/        # REST API server for UI
│   └── pkg/dtrules/    # Core library
├── ui/                 # React-based visual UI
├── sampleprojects/     # Example implementations
│   ├── CHIP/           # Health insurance eligibility
│   ├── ChipApp/        # CHIP application wrapper
│   ├── KidAid/         # Child assistance program
│   ├── KidAid_Application/ # KidAid application wrapper
│   ├── SyntaxTests/    # Language feature examples
│   └── TestProject/    # Minimal starter template
├── docs/               # Documentation
└── pom.xml             # Maven parent configuration
```

## Sample Projects

| Project | Description | Type |
|---------|-------------|------|
| **TestProject** | Minimal template for new projects | Rule Set |
| **SyntaxTests** | Comprehensive EL language feature examples | Reference |
| **CHIP** | Health insurance eligibility determination | Rule Set |
| **KidAid** | Child assistance program eligibility | Rule Set |
| **ChipApp** | Multi-threaded CHIP application wrapper | Application |
| **KidAid_Application** | Simple KidAid integration example | Application |

### Recommended Learning Path

1. **[TestProject](sampleprojects/TestProject/)** - Understand the minimal project structure
2. **[SyntaxTests](sampleprojects/SyntaxTests/)** - Learn EL language features
3. **[CHIP](sampleprojects/CHIP/)** - See a real-world eligibility example
4. **[ChipApp](sampleprojects/ChipApp/)** - Learn application integration patterns

## How It Works

### 1. Define Your Data Model (EDD)

Create an Entity Definition Document in Excel (`*_edd.xls`) that defines:
- Entities (data objects)
- Attributes and their types
- Relationships between entities

### 2. Write Decision Tables

Create Decision Tables in Excel (`*_dt.xls`) with:
- **Conditions** - Boolean expressions in EL syntax
- **Actions** - Operations to perform when conditions match
- **Columns** - Each column represents a rule

### 3. Compile to XML

```java
Excel2XML compiler = new Excel2XML(path, "DTRules.xml", "RuleSetName");
compiler.compileRuleSet(path, "DTRules.xml", "RuleSetName", "repository", maps, 5);
```

### 4. Execute Rules

```java
// Load the rules
RulesDirectory rd = new RulesDirectory(path, "DTRules.xml");
RuleSet rs = rd.getRuleSet("RuleSetName");
IRSession session = rs.newSession();

// Load input data
session.loadXml(inputStream, "mapping");

// Execute a decision table
session.execute("DecisionTableName");

// Get results
session.printEntityReport(output, session.getState(), "results", ...);
```

## Configuration

Each project has a `DTRules.xml` configuration file:

```xml
<DTRules>
  <compiler>EL</compiler>

  <RuleSet name="MyRules" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>
    <WorkingDirectory>/temp</WorkingDirectory>

    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>

    <Entities        name="MyRules_edd.xml" />
    <Decisiontables  name="MyRules_dt.xml"  />
    <Map             name="MyRules_map.xml" />
  </RuleSet>
</DTRules>
```

## Documentation

### Getting Started
- [UI Quick Start](QUICKSTART-UI.md) - Get the visual UI running in 5 minutes
- [Quick Start Guide](docs/QUICKSTART.md) - Step-by-step tutorial (Java)
- [Building from Source](docs/BUILDING.md) - Detailed build instructions

### Java Reference
- [Expression Language Reference](docs/EL-REFERENCE.md) - Complete EL syntax and operators
- [Architecture Guide](docs/ARCHITECTURE.md) - System design and components
- [API Guide](docs/API-GUIDE.md) - Java integration patterns and examples
- [Documentation Index](docs/README.md) - Full documentation overview

### Go Reference
- [Go README](go/README.md) - Installation, CLI usage, and quick start
- [Design Review](go/pkg/dtrules/DESIGN_REVIEW.md) - Architecture and design decisions
- [Performance Analysis](go/pkg/dtrules/benchmark/PERFORMANCE_ANALYSIS.md) - Detailed benchmarks

### Development
- [ANTLR Migration Guide](dsl/ANTLR_MIGRATION.md) - Parser modernization details
- [Changelog](CHANGELOG.md) - Version history

### Legacy Documentation
- `dtrules-engine/docs/` - Core engine documentation (PDF/DOC)

## Creating a New Project

1. Copy `sampleprojects/TestProject/` as a template
2. Rename and update `pom.xml` with your project details
3. Create your Entity Definition Document in `edd/`
4. Create your Decision Tables in `DecisionTables/`
5. Update `DTRules.xml` with your file names
6. Create compile and test Java classes

## Running Tests

```bash
# Run all tests
mvn test

# Run EL compiler tests (128 parameterized tests)
cd dsl/el
mvn test

# Run sample project tests
cd sampleprojects/CHIP
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestChip"
```

## License

Apache License, Version 2.0

Copyright 2004-2012 DTRules.com

## Go Implementation & Visual UI

In addition to the Java implementation, DTRules includes:

### Go Rules Engine

A high-performance Go implementation with the same XML format compatibility:

```bash
cd go

# Run tests
go test ./...

# Build CLI
go build -o dtrules ./cmd/dtrules

# Execute rules
./dtrules -rules ../sampleprojects/CHIP/xml -entry Compute_Eligibility -trace
```

See [go/README.md](go/README.md) for full documentation.

### Visual UI (React + Go Backend)

A modern web-based UI for editing decision tables, entities, and testing rules:

```bash
# Terminal 1: Start the Go API backend
cd go
go run ./cmd/api

# Terminal 2: Start the React frontend
cd ui
npm install
npm run dev
```

Then open http://localhost:5173 in your browser.

**Quick Start:**
1. Click "Open CHIP Sample Project" on the welcome screen
2. Enter the path: `/path/to/DTRules/sampleprojects/CHIP/xml`
3. Explore entities, decision tables, and run tests

See [ui/README.md](ui/README.md) for full documentation.

## Links

- Website: https://dtrules.com
- GitHub: https://github.com/PaulSnow/DTRules
