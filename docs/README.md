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
| [Spreadsheet Formats Guide](SPREADSHEET-FORMATS.md) | Excel, ODS, and Google Sheets support |

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

## Visual UI

A modern React-based UI for editing decision tables and testing rules:

```bash
# Terminal 1: Start the Go API backend
cd go && go run ./cmd/api

# Terminal 2: Start the React frontend
cd ui && npm install && npm run dev
```

Then open http://localhost:5173 in your browser. See [ui/README.md](../ui/README.md) for details.

## Development Documentation

| Document | Location | Description |
|----------|----------|-------------|
| Compiler Utilities | [compilerutil/README.md](../compilerutil/README.md) | Spreadsheet conversion tools |
| ANTLR 4 Migration | [dsl/ANTLR_MIGRATION.md](../dsl/ANTLR_MIGRATION.md) | Parser modernization guide |
| Changelog | [CHANGELOG.md](../CHANGELOG.md) | Version history and changes |

## Legacy Documentation

Additional documentation is available in legacy formats:

| Document | Location | Description |
|----------|----------|-------------|
| DTRules Overview | `dtrules-engine/docs/DTRules.doc` | Core engine documentation |
| Operator Reference | `dtrules-engine/docs/OperatorList.doc` | List of available operators |
| EL Overview | `dtrules-engine/docs/Overview_of_DTRules_and_EL.pdf` | Expression Language overview |
| EL Examples | `sampleprojects/SyntaxTests/EL Documentation-1.odt` | Comprehensive EL syntax examples |

## Sample Projects

### Recommended Learning Path

1. **[TestProject](../sampleprojects/TestProject/)** - Understand the minimal project structure
2. **[SyntaxTests](../sampleprojects/SyntaxTests/)** - Learn EL language features
3. **[CHIP](../sampleprojects/CHIP/)** - See a real-world eligibility example
4. **[ChipApp](../sampleprojects/ChipApp/)** - Learn application integration patterns

### All Sample Projects

#### Rule Set Projects

| Project | Purpose | Complexity |
|---------|---------|------------|
| [TestProject](../sampleprojects/TestProject/) | Minimal template for new projects | Low |
| [SyntaxTests](../sampleprojects/SyntaxTests/) | EL language feature examples | Reference |
| [CHIP](../sampleprojects/CHIP/) | Health insurance eligibility | High |
| [KidAid](../sampleprojects/KidAid/) | Child assistance eligibility | High |

#### Application Wrappers

| Project | Purpose | Features |
|---------|---------|----------|
| [ChipApp](../sampleprojects/ChipApp/) | CHIP integration example | Multi-threaded, performance testing |
| [KidAid_Application](../sampleprojects/KidAid_Application/) | KidAid integration example | Simple integration pattern |

See the [Sample Projects Overview](../sampleprojects/README.md) for the complete guide.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Spreadsheets   ──compile──▶   XML Rules   ──load──▶  Engine  ──▶  Results
│  (EDD + DT)                    (*_edd.xml,                │
│                                 *_dt.xml)            Your Application
│  Supported:                                               │
│  • Excel (.xls, .xlsx, .xlsm)                             │
│  • OpenDocument (.ods)                                    │
│  • Google Sheets (URL)                                    │
└─────────────────────────────────────────────────────────────────────────┘
```

For detailed architecture diagrams and component descriptions, see the [Architecture Guide](ARCHITECTURE.md).

## Key Concepts

### Entity Definition Document (EDD)
Defines the data model used by decision tables. Contains entities, attributes, types, and relationships. Created in spreadsheets (Excel, ODS, or Google Sheets). See [Spreadsheet Formats Guide](SPREADSHEET-FORMATS.md).

### Decision Tables
Tabular representation of business rules. Each column is a rule with conditions to match and actions to execute. Uses EL syntax. Can be authored in Excel, LibreOffice, or Google Sheets.

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

## DSL Implementations

| DSL | Description | Parser | Sample Projects |
|-----|-------------|--------|-----------------|
| **EL** | Expression Language - standard DSL | ANTLR 4 | CHIP, KidAid, SyntaxTests, TestProject |
| **EBL** | Entity Business Language - enhanced DSL | ANTLR 4 | (available for advanced use cases) |

## Contributing

See the [main README](../README.md) for repository information and licensing.
