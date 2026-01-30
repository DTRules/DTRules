# DTRules

A Decision Table based Rules Engine for Java.

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

### Production Use

DTRules has been used in production systems including:
- State welfare/assistance programs (Texas TIERS, Ohio OFAST)
- Insurance eligibility determination systems
- Commercial business logic applications

## Requirements

- **Java 8+** (built with Java 1.6 target for compatibility)
- **Apache Maven 3.x**

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
- `dsl` - Domain Specific Language implementations
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
├── dtrules-engine/     # Core rules execution engine
├── compilerutil/       # Excel-to-XML compiler utilities
├── dsl/                # Domain Specific Languages
│   ├── el/             # Expression Language (EL) - JFlex/CUP + ANTLR 4
│   ├── ebl/            # Entity Business Language (EBL) - JFlex/CUP + ANTLR 4
│   └── sudoku_language/ # Sudoku DSL - JFlex/CUP + ANTLR 4
├── sampleprojects/     # Example implementations
│   ├── CHIP/           # Health insurance eligibility
│   ├── KidAid/         # Child assistance program
│   ├── Sudoku/         # Sudoku solver (custom DSL demo)
│   ├── SyntaxTests/    # Language feature examples
│   └── TestProject/    # Minimal starter template
└── pom.xml             # Maven parent configuration
```

## Sample Projects

| Project | Description | DSL | Complexity |
|---------|-------------|-----|------------|
| **TestProject** | Minimal template for new projects | EL | Low |
| **SyntaxTests** | Comprehensive EL language feature examples | EL | Reference |
| **CHIP** | Health insurance eligibility determination | EL | High |
| **KidAid** | Child assistance program eligibility | EL | High |
| **Sudoku** | Sudoku puzzle solver demonstrating custom DSL | Custom | High |
| **eBook** | Multi-ruleset business logic example | EBL | High |
| **ChipApp** | Standalone CHIP application wrapper | EL | Medium |
| **KidAid_Application** | Standalone KidAid wrapper | EL | Low |
| **eBookApp** | Standalone eBook wrapper | EBL | Medium |

### Recommended Learning Path

1. **[TestProject](sampleprojects/TestProject/)** - Understand the minimal project structure
2. **[SyntaxTests](sampleprojects/SyntaxTests/)** - Learn EL language features
3. **[CHIP](sampleprojects/CHIP/)** - See a real-world eligibility example
4. **[ChipApp](sampleprojects/ChipApp/)** - Learn application integration patterns
5. **[eBook](sampleprojects/eBook/)** - Explore multi-ruleset projects
6. **[Sudoku](sampleprojects/Sudoku/)** - Understand custom DSL creation

## How It Works

### 1. Define Your Data Model (EDD)

Create an Entity Definition Document in Excel (`*_edd.xls`) that defines:
- Entities (data objects)
- Attributes and their types
- Relationships between entities

### 2. Write Decision Tables

Create Decision Tables in Excel (`*_dt.xls`) with:
- **Conditions** - Boolean expressions in EL/EBL syntax
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
- [Quick Start Guide](docs/QUICKSTART.md) - Step-by-step tutorial
- [Building from Source](docs/BUILDING.md) - Detailed build instructions

### Reference
- [Expression Language Reference](docs/EL-REFERENCE.md) - Complete EL syntax and operators
- [Architecture Guide](docs/ARCHITECTURE.md) - System design and components
- [API Guide](docs/API-GUIDE.md) - Java integration patterns and examples
- [Documentation Index](docs/README.md) - Full documentation overview

### Legacy Documentation
- `dtrules-engine/docs/` - Core engine documentation (PDF/DOC)
- `dsl/el/docs/` - Expression Language overview (PDF)

## Creating a New Project

1. Copy `sampleprojects/TestProject/` as a template
2. Rename and update `pom.xml` with your project details
3. Create your Entity Definition Document in `edd/`
4. Create your Decision Tables in `DecisionTables/`
5. Update `DTRules.xml` with your file names
6. Create compile and test Java classes

## License

Apache License, Version 2.0

Copyright 2004-2012 DTRules.com

## Links

- Website: http://DTRules.com
- GitHub: https://github.com/PaulSnow/DTRules
