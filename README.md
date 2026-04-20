# DTRules

A high-performance Decision Table Rules Engine written in **Go**.

## Overview

DTRules is a production-ready rules engine that allows business analysts and policy experts to define complex business logic using **Decision Tables** in Excel spreadsheets. Decision tables provide a tabular representation of business rules that can be understood by both policy experts and developers.

### Key Features

- **Excel-based Rule Definition** - Define rules in familiar spreadsheet format
- **Embeddable SDK** - Compile rules into your Go binary
- **High Performance** - 130x faster operator lookup, 24x faster arithmetic vs Java
- **CLI Tool** - Validate, test, and execute rules from command line
- **Bidirectional Sync** - Excel ↔ XML synchronization with change tracking
- **Fixed-Point Decimals** - 256-bit `fixed` type for token, staking, and blockchain math without float drift (`dtrules docs fixed`)
- **Embedded Documentation** - `dtrules docs` for AI and developers

### Production Use

DTRules has been used in production systems including:
- State welfare/assistance programs (Texas TIERS, Ohio OFAST)
- Insurance eligibility determination systems
- Commercial business logic applications

## Install

**Option 1 — go install (requires Go 1.21+):**

```bash
go install github.com/DTRules/DTRules/cmd/dtrules@latest
```

**Option 2 — Prebuilt binaries:**

Download from [GitHub Releases](https://github.com/DTRules/DTRules/releases) for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, and windows-amd64.

**Verify install:**

```bash
dtrules version && dtrules docs
```

<!-- TODO: Homebrew tap -->

## Quick Start

### Build from Source

```bash
git clone https://github.com/DTRules/DTRules.git
cd DTRules
make build
```

### Using the CLI

```bash
# List decision tables
dtrules -rules ./sampleprojects/CHIP/xml -list

# Execute rules
dtrules -rules ./sampleprojects/CHIP/xml -entry Compute_Eligibility

# View embedded documentation
dtrules docs                     # List topics
dtrules docs decision-tables     # How to write decision tables
dtrules docs operators           # All operators
dtrules docs sdk                 # Embedding in applications

# Sync Excel and XML
dtrules sync status              # Show sync status
dtrules sync import              # Import Excel → XML
dtrules sync export              # Export XML → Excel
```

### Embedding in Applications

```go
package main

import (
    "embed"
    "log"
    "github.com/DTRules/DTRules/pkg/dtrules/sdk"
)

//go:embed rules/*.xml
var rulesFS embed.FS

func main() {
    // Load rules from embedded filesystem
    engine, err := sdk.NewEngine("MyRules", sdk.WithFS(rulesFS, "rules"))
    if err != nil {
        log.Fatal(err)
    }

    // Execute decision table
    ctx := engine.NewContext()
    ctx.SetEntity("input", "income", 50000)
    ctx.SetEntity("input", "age", 35)

    result, err := engine.Execute("Calculate_Eligibility", ctx)
    if err != nil {
        log.Fatal(err)
    }

    eligible := result.GetBool("eligible")
}
```

## Project Structure

```
DTRules/
├── cmd/
│   ├── dtrules/        # CLI tool
│   ├── api/            # REST API server for UI
│   └── ...             # Other commands
├── pkg/dtrules/
│   ├── sdk/            # Embeddable SDK
│   ├── sync/           # Excel/XML synchronization
│   ├── excel/          # Excel import/export
│   ├── session/        # Rule execution sessions
│   ├── operators/      # 179+ built-in operators
│   └── ...             # Core engine packages
├── examples/
│   └── embedded-app/   # Example embedded application
├── sampleprojects/     # Example rule sets
│   ├── CHIP/           # Health insurance eligibility
│   ├── TaxReturn/      # Tax calculation
│   └── ...
├── ui/                 # React-based visual UI
├── website/            # DTRules.com website
└── legacy/             # Java implementation (archived)
```

## Documentation

### Embedded (in binary)

```bash
dtrules docs                     # List all topics
dtrules docs xml-format          # XML file format specification
dtrules docs decision-tables     # How to write decision tables
dtrules docs operators           # All operators with examples
dtrules docs sdk                 # Embedding in applications
dtrules docs examples            # Complete working examples
dtrules docs workflow            # Development workflow
```

### Online

- [DTRules.com](https://dtrules.com) - Official website
- [UI Quick Start](QUICKSTART-UI.md) - Visual UI setup

### In Repository

- [EL Reference](docs/EL-REFERENCE.md) - Expression Language syntax
- [XML Format](docs/decision-table-xml-format.md) - Decision table XML structure
- [EL Compiler](pkg/dtrules/compiler/el/README.md) - How EL expressions are compiled

## Sample Projects

| Project | Description |
|---------|-------------|
| **CHIP** | Health insurance eligibility determination |
| **TaxReturn** | State income tax calculation (all 50 states) |
| **StateTax** | Corporate tax calculation |
| **KidAid** | Child assistance program eligibility |
| **TestProject** | Minimal starter template |

## Visual UI

A modern web-based UI for editing decision tables and testing rules:

```bash
# Start the Go API backend
go run ./cmd/api

# Start the React frontend (in another terminal)
cd ui
npm install && npm run dev
```

Open http://localhost:5173 in your browser.

## Performance

Key optimizations achieve significant speedups vs Java:

| Operation | Speedup |
|-----------|---------|
| Operator Lookup | **130x** |
| Integer Arithmetic | **24x** |
| String Creation | **3.7x** |

## Expression Language (EL)

Decision tables use **EL (Expression Language)** for conditions and actions. EL is the only language to author rules in — write all conditions and actions in EL.

### EL Syntax Examples

**Conditions:**
```
taxpayer.filing_status == "SINGLE"
taxpayer.income > 50000.0
result.has_nexus is true
```

**Actions:**
```
set result.tax_liability = income * rate
set taxpayer.exemptions = taxpayer.exemptions + 1
perform Calculate_Deductions
```

### Why EL is Required

- **Readable** - Business analysts can understand and modify rules
- **Validated** - Syntax errors caught at compile time, not runtime
- **Consistent** - Eliminates hand-coded expression errors

See [EL Reference](docs/EL-REFERENCE.md) for complete syntax and [XML Format](docs/decision-table-xml-format.md) for file structure.

## Development Workflow

### Excel-first (Business Analysts)

1. Edit rules in Excel spreadsheets
2. Run `dtrules sync import` to generate XML with EL
3. Test with `dtrules -rules ./xml -test ./testfiles`

### XML-first (Developers/AI)

1. Write EL expressions in XML files
2. Test with `dtrules -rules ./xml -test ./testfiles`
3. Run `dtrules sync export` to update Excel

## Requirements

- **Go 1.21+**

## Legacy Java Implementation

The original Java implementation is archived in `legacy/java/`. It remains functional but is no longer the primary implementation.

## License

Apache License, Version 2.0

## Links

- Website: https://dtrules.com
- GitHub: https://github.com/DTRules/DTRules
