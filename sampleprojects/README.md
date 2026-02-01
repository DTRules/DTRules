# DTRules Sample Projects

This directory contains sample projects demonstrating various aspects of DTRules.

## Project Categories

### Rule Set Projects

These projects define decision tables and entity models:

| Project | Description | Test Cases | Complexity |
|---------|-------------|------------|------------|
| [TestProject](TestProject/) | Minimal starter template | Compile only | Low |
| [SyntaxTests](SyntaxTests/) | EL language feature examples | 1 | Reference |
| [CHIP](CHIP/) | Health insurance eligibility | 2 | High |
| [KidAid](KidAid/) | Child assistance eligibility | 2 | High |

### Application Wrappers

These projects demonstrate how to integrate DTRules into applications:

| Project | Wraps | Features |
|---------|-------|----------|
| [ChipApp](ChipApp/) | CHIP | Multi-threaded execution, performance benchmarking |
| [KidAid_Application](KidAid_Application/) | KidAid | Simple integration example |

## Getting Started

### Recommended Learning Path

1. **[TestProject](TestProject/)** - Understand the minimal structure
2. **[SyntaxTests](SyntaxTests/)** - Learn EL language features
3. **[CHIP](CHIP/)** - See a real-world example
4. **[ChipApp](ChipApp/)** - Learn application integration

### Quick Start

```bash
# Build everything
cd /path/to/DTRules
mvn clean install

# Try the CHIP example
cd sampleprojects/CHIP
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.CompileChip"
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestChip"
```

## Project Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                    Rule Set Projects                        │
│  ┌──────────┐  ┌──────────┐                                 │
│  │   CHIP   │  │  KidAid  │                                 │
│  └────┬─────┘  └────┬─────┘                                 │
│       │             │                                        │
│       ▼             ▼                                        │
│  ┌──────────┐  ┌──────────────┐                             │
│  │ ChipApp  │  │ KidAid_App   │                             │
│  │          │  │              │                             │
│  └──────────┘  └──────────────┘                             │
│           Application Wrappers                              │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Reference Projects                       │
│  ┌──────────────────────┐  ┌──────────────────────────────┐│
│  │     TestProject      │  │        SyntaxTests           ││
│  │   (minimal template) │  │    (language reference)      ││
│  └──────────────────────┘  └──────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Common Structure

Each rule set project follows this structure:

```
ProjectName/
├── DTRules.xml               # Configuration
├── DecisionTables/           # Excel decision tables
│   └── *_dt.xls
├── edd/                      # Entity Definition Documents
│   └── *_edd.xls
├── xml/                      # Compiled XML (generated)
├── repository/               # Packaged for deployment
├── testfiles/                # Test inputs
│   └── TestScenarios/        # Test case XML files
├── pom.xml                   # Maven configuration
└── src/main/java/            # Compile/Test classes
```

## Running Tests

### Individual Projects

```bash
# CHIP
cd CHIP
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestChip"
# Output: ALL PASS

# KidAid
cd KidAid
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sampleproject2.TestKidAid"
# Output: ALL PASS

# SyntaxTests
cd SyntaxTests
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sampleproject2.TestSyntaxExamples"
# Output: ALL PASS

# TestProject (compile only)
cd TestProject
mvn exec:java -Dexec.mainClass="com.dtrules.samples.test.Compile_Test"
```

### All Projects

```bash
cd sampleprojects
mvn clean install
```

## Creating Your Own Project

1. Copy [TestProject](TestProject/) as a template
2. Follow the README instructions in TestProject
3. Refer to [SyntaxTests](SyntaxTests/) for EL syntax examples
4. Use [ChipApp](ChipApp/) as a pattern for application integration

## Test Results Summary

| Project | Tests | Status |
|---------|-------|--------|
| CHIP | 2 cases | PASS |
| KidAid | 2 cases | PASS |
| SyntaxTests | 1 case | PASS |
| TestProject | Compile | PASS |

## Documentation

- [Main Documentation](../docs/README.md)
- [EL Reference](../docs/EL-REFERENCE.md)
- [API Guide](../docs/API-GUIDE.md)
