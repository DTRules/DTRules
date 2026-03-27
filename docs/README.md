# DTRules Documentation

This directory contains comprehensive documentation for DTRules. The project has implementations in **Java**, **Go**, and **Assembly**.

## Quick Links

| I want to... | Go here |
|--------------|---------|
| Get started quickly | [Quick Start (Java)](QUICKSTART.md) or [Quick Start (UI)](../QUICKSTART-UI.md) |
| Build from source | [Building Guide](BUILDING.md) |
| Learn the expression language | [EL Reference](EL-REFERENCE.md) |
| Understand the architecture | [Architecture Guide](ARCHITECTURE.md) |
| Use the Go implementation | [Go README](../go/README.md) |

---

## Getting Started

| Document | Description |
|----------|-------------|
| [Quick Start (Java)](QUICKSTART.md) | Step-by-step guide to running your first DTRules project |
| [Quick Start (UI)](../QUICKSTART-UI.md) | Get the visual UI running in 5 minutes |
| [Building from Source](BUILDING.md) | Detailed build instructions and IDE setup |

---

## Core Concepts

| Document | Description |
|----------|-------------|
| [Architecture Guide](ARCHITECTURE.md) | System design, components, and execution flow |
| [Expression Language Reference](EL-REFERENCE.md) | Complete EL syntax, operators, and functions |
| [Spreadsheet Formats Guide](SPREADSHEET-FORMATS.md) | Excel, ODS, and Google Sheets support |
| [Bytecode Specification](bytecode-spec.md) | Portable bytecode format for cross-runtime execution |

---

## Java Implementation

| Document | Description |
|----------|-------------|
| [API Guide](API-GUIDE.md) | Java integration patterns and code examples |
| [ANTLR Migration Guide](antlr-migration.md) | EL/EBL parser modernization (JFlex/CUP → ANTLR 4) |

### Java Quick Start

```bash
git clone https://github.com/PaulSnow/DTRules.git
cd DTRules
mvn clean install
```

---

## Go Implementation

| Document | Description |
|----------|-------------|
| [Go README](../go/README.md) | Installation, CLI usage, REST API |
| [Design Review](go-design-review.md) | Architecture, design decisions, security review |
| [Performance Analysis](go-performance.md) | Detailed benchmarks and optimizations |
| [Native ASM Runtime](nativeasm-runtime.md) | Plan 9 assembly runtime (20-50x faster) |

### Go Quick Start

```bash
cd go && go build -o dtrules ./cmd/dtrules
./dtrules -rules /path/to/xml -list
./dtrules -rules /path/to/xml -entry Main
```

### Performance Highlights

| Optimization | Speedup |
|-------------|---------|
| Operator Lookup | 130x faster |
| Value Arithmetic | 24x faster |
| String Interning | 3.7x faster |
| Native ASM Push/Pop | 20x vs CGO |

---

## Assembly Implementation

| Document | Description |
|----------|-------------|
| [Assembly Guide](asm-guide.md) | x86-64 NASM implementation - architecture, compatibility, testing |
| [ASM README](../asm/README.md) | Build instructions and overview |

The assembly implementation is educational, demonstrating how a rules engine can be built at the lowest level with no libc dependencies.

---

## Visual UI

A modern React-based UI for editing decision tables and testing rules:

```bash
# Terminal 1: Start Go API backend
cd go && go run ./cmd/api

# Terminal 2: Start React frontend
cd ui && npm install && npm run dev
```

Then open http://localhost:5173

| Document | Description |
|----------|-------------|
| [UI README](../ui/README.md) | Setup, features, and usage |
| [UI Architecture](../ui/ARCHITECTURE.md) | Component structure and design |

---

## Sample Projects

### Recommended Learning Path

1. **[TestProject](../sampleprojects/TestProject/)** - Understand the minimal project structure
2. **[SyntaxTests](../sampleprojects/SyntaxTests/)** - Learn EL language features
3. **[CHIP](../sampleprojects/CHIP/)** - See a real-world eligibility example
4. **[ChipApp](../sampleprojects/ChipApp/)** - Learn application integration patterns

### All Sample Projects

| Project | Purpose | Complexity |
|---------|---------|------------|
| [TestProject](../sampleprojects/TestProject/) | Minimal template for new projects | Low |
| [SyntaxTests](../sampleprojects/SyntaxTests/) | EL language feature examples | Reference |
| [CHIP](../sampleprojects/CHIP/) | Health insurance eligibility | High |
| [KidAid](../sampleprojects/KidAid/) | Child assistance eligibility | High |
| [ChipApp](../sampleprojects/ChipApp/) | CHIP integration example | Medium |
| [KidAid_Application](../sampleprojects/KidAid_Application/) | KidAid integration example | Medium |

See the [Sample Projects Overview](../sampleprojects/README.md) for the complete guide.

---

## Development Documentation

| Document | Description |
|----------|-------------|
| [Changelog](../CHANGELOG.md) | Version history and changes |
| [Compiler Utilities](../compilerutil/README.md) | Spreadsheet conversion tools |
| [DSL Overview](../dsl/README.md) | Domain Specific Language modules |

---

## Legacy Documentation

Additional documentation is available in legacy formats:

| Document | Location | Description |
|----------|----------|-------------|
| DTRules Overview | `dtrules-engine/docs/DTRules.doc` | Core engine documentation |
| Operator Reference | `dtrules-engine/docs/OperatorList.doc` | List of available operators |
| EL Overview | `dtrules-engine/docs/Overview_of_DTRules_and_EL.pdf` | Expression Language overview |
| EL Examples | `sampleprojects/SyntaxTests/EL Documentation-1.odt` | Comprehensive EL syntax examples |

---

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

---

## Key Concepts

### Entity Definition Document (EDD)
Defines the data model used by decision tables. Contains entities, attributes, types, and relationships. Created in spreadsheets.

### Decision Tables
Tabular representation of business rules. Each column is a rule with conditions to match and actions to execute. Uses EL syntax.

### Expression Language (EL)
Domain-specific language for writing conditions and actions. Designed to be readable by business analysts and policy experts.

### Session
Runtime execution context. Each evaluation creates a new session, loads data, executes decision tables, and extracts results.

---

## Contributing

See the [main README](../README.md) for repository information and licensing.
