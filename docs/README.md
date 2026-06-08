# DTRules Documentation

DTRules is a Go-primary rules engine. The authoritative CLI-usage
walkthrough lives inside the `dtrules` binary itself — run `dtrules
docs cli` after installing. The Java implementation is archived under
`legacy/java/` and no longer maintained.

## Quick Links

| I want to... | Go here |
|--------------|---------|
| Get started with the CLI | `dtrules docs cli` (embedded) — or [`quickstart.md`](quickstart.md) for pointers |
| Build from source | [Building Guide](building.md) |
| Learn the expression language | `dtrules docs el` or [EL Reference](el-reference.md) |
| Write decision tables in XML | [Decision Table XML Format](decision-table-xml-format.md) |
| Understand the architecture | [Architecture Guide](architecture.md) |

---

## Getting Started

The in-binary topics are the freshest — prefer them over the .md files
for day-to-day use:

| Source | Topic |
|--------|-------|
| `dtrules docs cli` | CLI install, init, build, validate, verify walkthrough |
| `dtrules docs workflow` | `dtrules build` pipeline deep-dive |
| `dtrules docs el` | Expression Language reference |
| `dtrules docs edd` | Entity Data Dictionary |
| `dtrules docs operators` | Every runtime operator with examples |
| [`building.md`](building.md) | Build the binary from source (contributors) |

---

## Core Concepts

| Document | Description |
|----------|-------------|
| [Architecture Guide](architecture.md) | System design, components, and execution flow |
| [Expression Language Reference](el-reference.md) | Complete EL syntax, operators, and functions |
| [Decision Table XML Format](decision-table-xml-format.md) | **Required** EL format for decision table XML |
| [EL Compiler](../pkg/dtrules/compiler/el/README.md) | Compiles EL descriptions to postfix notation |
| [Spreadsheet Formats Guide](spreadsheet-formats.md) | Excel, ODS, and Google Sheets support |

---

## Feature Narratives

Long-form companions to the embedded `dtrules docs` topics, focused on
*why* and *when* rather than syntax. The embedded topics remain the
authoritative usage walkthrough.

| Document | Feature | First shipped |
|----------|---------|---------------|
| [EDD Usage Analyzer](edd-usage-analyzer.md) | Entity-stack-aware `unused` / `write-only` detection | v1.13.0 |
| [Redundant Action-Set Column Advisory](advisory-redundant-action-set.md) | Column-level redundancy check on the compiled decision tree | v1.14.6 |
| [`first pass` Predicate](first-pass-predicate.md) | EL predicate for the first iteration of the innermost active loop | v1.12.0 |
| [`for all <type> entities` Loop](for-all-type-entities.md) | Type-driven iteration via the EDD `owns` declaration | [#703](https://github.com/DTRules/DTRules/issues/703) |
| [Multiple Entry Points](multi-entry-points.md) | Run several decision tables against one loaded session | Always supported (pinned + documented in v1.15.1) |

For features without a narrative companion, see the embedded `dtrules
docs` topics (`dtrules docs operators` covers `fphalfup/` and
`fpdivr/`; `dtrules docs el` covers `create T as alias`).

---

## Java Implementation

| Document | Description |
|----------|-------------|
| [API Guide](api-guide.md) | Java integration patterns and code examples |
| [ANTLR Migration Guide](antlr-migration.md) | EL/EBL parser modernization (JFlex/CUP → ANTLR 4) |

### Java Quick Start

```bash
git clone https://github.com/DTRules/DTRules.git
cd DTRules
mvn clean install
```

---

## Go Implementation

| Document | Description |
|----------|-------------|
| [Root README](../README.md) | Installation, CLI usage |
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
| [Legacy ASM](../legacy/) | Original Plan 9 assembly tree (archived; superseded by the Go-native runtime) |

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
| [UI Architecture](../ui/architecture.md) | Component structure and design |

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
| [Compiler Internals](compiler-internals.md) | EL compiler architecture and ANTLR plumbing |
| [Building](building.md) | Build, test, and packaging instructions |

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
Domain-specific language for writing conditions and actions. Designed to be readable by business analysts and policy experts. The [EL Compiler](../pkg/dtrules/compiler/el/README.md) converts EL descriptions to postfix notation for runtime execution.

### Session
Runtime execution context. Each evaluation creates a new session, loads data, executes decision tables, and extracts results.

---

## Contributing

See the [main README](../README.md) for repository information and licensing.
