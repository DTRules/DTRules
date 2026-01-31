# DTRules Documentation

DTRules has implementations in both **Java** and **Go**. This documentation primarily covers the Java implementation. For Go-specific documentation, see the [Go README](../go/README.md).

## Implementations

| Implementation | Documentation | Best For |
|----------------|---------------|----------|
| **Java** | This directory | Full tooling, Excel compilers, IDE integration |
| **Go** | [go/README.md](../go/README.md) | High-performance runtime, microservices |

## Getting Started (Java)

| Document | Description |
|----------|-------------|
| [Quick Start Guide](QUICKSTART.md) | Step-by-step guide to running your first DTRules project |
| [Building from Source](BUILDING.md) | Detailed build instructions and IDE setup |

## Reference Documentation (Java)

| Document | Description |
|----------|-------------|
| [Expression Language Reference](EL-REFERENCE.md) | Complete EL syntax, operators, and functions |
| [Architecture Guide](ARCHITECTURE.md) | System design, components, and execution flow |
| [API Guide](API-GUIDE.md) | Java integration patterns and code examples |

## Go Implementation

| Document | Description |
|----------|-------------|
| [Go README](../go/README.md) | Installation, CLI usage, quick start |
| [Design Review](../go/pkg/dtrules/DESIGN_REVIEW.md) | Architecture, design decisions, security review |
| [Performance Analysis](../go/pkg/dtrules/benchmark/PERFORMANCE_ANALYSIS.md) | Detailed benchmarks and optimizations |

### Go Quick Start

```bash
# Build CLI
cd go && go build -o dtrules ./cmd/dtrules

# Run with existing Java-compiled rules
./dtrules -rules /path/to/xml -list
./dtrules -rules /path/to/xml -entry Main
```

### Legacy Documentation

Additional documentation is available in legacy formats:

| Document | Location | Description |
|----------|----------|-------------|
| DTRules Overview | `dtrules-engine/docs/DTRules.doc` | Core engine documentation |
| Operator Reference | `dtrules-engine/docs/OperatorList.doc` | List of available operators |
| EL Overview | `dsl/el/docs/Overview_of_DTRules_and_EL.pdf` | Expression Language overview |
| EL Examples | `sampleprojects/SyntaxTests/EL Documentation-1.odt` | Comprehensive EL syntax examples |

## Sample Projects

### Recommended Learning Path

1. **[TestProject](../sampleprojects/TestProject/)** - Understand the minimal project structure
2. **[SyntaxTests](../sampleprojects/SyntaxTests/)** - Learn EL language features
3. **[CHIP](../sampleprojects/CHIP/)** - See a real-world eligibility example
4. **[ChipApp](../sampleprojects/ChipApp/)** - Learn application integration patterns
5. **[eBook](../sampleprojects/eBook/)** - Explore multi-ruleset projects
6. **[Sudoku](../sampleprojects/Sudoku/)** - Understand custom DSL creation

### All Sample Projects

| Project | Purpose | Complexity |
|---------|---------|------------|
| [TestProject](../sampleprojects/TestProject/) | Minimal template for new projects | Low |
| [SyntaxTests](../sampleprojects/SyntaxTests/) | EL language feature examples | Reference |
| [CHIP](../sampleprojects/CHIP/) | Health insurance eligibility | High |
| [KidAid](../sampleprojects/KidAid/) | Child assistance eligibility | High |
| [Sudoku](../sampleprojects/Sudoku/) | Puzzle solver with custom DSL | High |
| [eBook](../sampleprojects/eBook/) | Multi-ruleset business logic (EBL) | High |
| [ChipApp](../sampleprojects/ChipApp/) | Application integration pattern | Medium |
| [KidAid_Application](../sampleprojects/KidAid_Application/) | Simple integration example | Low |
| [eBookApp](../sampleprojects/eBookApp/) | EBL application wrapper | Medium |

See the [Sample Projects Overview](../sampleprojects/README.md) for the complete guide.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Excel Files    ──compile──▶   XML Rules   ──load──▶  Engine  ──▶  Results
│  (EDD + DT)                    (*_edd.xml,              │
│                                 *_dt.xml)          Your Application
└─────────────────────────────────────────────────────────────┘
```

For detailed architecture diagrams and component descriptions, see the [Architecture Guide](ARCHITECTURE.md).

## Key Concepts

### Entity Definition Document (EDD)
Defines the data model used by decision tables. Contains entities, attributes, types, and relationships. Created in Excel.

### Decision Tables
Tabular representation of business rules. Each column is a rule with conditions to match and actions to execute. Uses EL syntax.

### Expression Language (EL)
Domain-specific language for writing conditions and actions. Designed to be readable by business analysts and policy experts.

### Session
Runtime execution context. Each evaluation creates a new session, loads data, executes decision tables, and extracts results.

### Mapping
Configuration for converting between external data formats (XML, Java objects) and internal entity representation.

## Quick Code Example

```java
// Initialize (once at startup)
RulesDirectory rd = new RulesDirectory(path, "DTRules.xml");
RuleSet rs = rd.getRuleSet("MyRules");

// Execute (per evaluation)
IRSession session = rs.newSession();
session.loadXml(inputStream, "main");
session.execute("Main_Decision_Table");

// Get results
DTState state = session.getState();
IREntity results = state.find("results");
String status = results.get("status").stringValue();
```

See the [API Guide](API-GUIDE.md) for comprehensive examples.

## DSL Comparison

| DSL | Description | Sample Projects |
|-----|-------------|-----------------|
| **EL** | Expression Language - standard DSL | CHIP, KidAid, SyntaxTests, TestProject |
| **EBL** | Entity Business Language - enhanced DSL | eBook, eBookApp |
| **Custom** | Domain-specific languages | Sudoku (SudokuLanguage) |

## Contributing

See the [main README](../README.md) for repository information and licensing.
