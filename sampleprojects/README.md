# DTRules Sample Projects

This directory contains sample projects demonstrating various aspects of DTRules.

## Project Categories

### Rule Set Projects

These projects define decision tables and entity models:

| Project | Description | DSL | Complexity |
|---------|-------------|-----|------------|
| [CHIP](CHIP/) | Health insurance eligibility | EL | High |
| [KidAid](KidAid/) | Child assistance eligibility | EL | High |
| [Sudoku](Sudoku/) | Puzzle solver with custom DSL | Custom | High |
| [SyntaxTests](SyntaxTests/) | EL language feature examples | EL | Reference |
| [TestProject](TestProject/) | Minimal starter template | EL | Low |
| [eBook](eBook/) | Multi-ruleset business logic | EBL | High |

### Application Wrappers

These projects wrap rule sets in executable applications:

| Project | Wraps | Features |
|---------|-------|----------|
| [ChipApp](ChipApp/) | CHIP | Multi-threaded, performance testing |
| [KidAid_Application](KidAid_Application/) | KidAid | Simple integration example |
| [eBookApp](eBookApp/) | eBook | EBL-based application |

## Getting Started

### Recommended Learning Path

1. **[TestProject](TestProject/)** - Understand the minimal structure
2. **[SyntaxTests](SyntaxTests/)** - Learn EL language features
3. **[CHIP](CHIP/)** - See a real-world example
4. **[ChipApp](ChipApp/)** - Learn application integration
5. **[eBook](eBook/)** - Explore multi-ruleset projects
6. **[Sudoku](Sudoku/)** - Understand custom DSL creation

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
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │   CHIP   │  │  KidAid  │  │  eBook   │  │   Sudoku   │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────────┘  │
│       │             │             │                         │
│       ▼             ▼             ▼                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │ ChipApp  │  │ KidAid   │  │ eBookApp │                  │
│  │          │  │ _App     │  │          │                  │
│  └──────────┘  └──────────┘  └──────────┘                  │
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
├── pom.xml                   # Maven configuration
└── src/main/java/            # Compile/Test classes
```

## DSL Comparison

| DSL | Projects | Description |
|-----|----------|-------------|
| **EL** | CHIP, KidAid, SyntaxTests, TestProject | Standard Expression Language |
| **EBL** | eBook, eBookApp | Enhanced Business Language |
| **Custom** | Sudoku | Domain-specific (SudokuLanguage) |

## Creating Your Own Project

1. Copy [TestProject](TestProject/) as a template
2. Follow the README instructions in TestProject
3. Refer to [SyntaxTests](SyntaxTests/) for EL syntax examples
4. Use [ChipApp](ChipApp/) as a pattern for application integration

## Building All Samples

```bash
cd sampleprojects
mvn clean install
```

Or build a specific project:

```bash
mvn clean install -pl CHIP
```
