# DTRules Go Implementation

A high-performance Go implementation of the DTRules decision table rules engine.

## Overview

DTRules is a stack-based rules engine that executes decision tables. This Go implementation provides:

- **Full compatibility** with DTRules XML formats (EDD and Decision Tables)
- **High performance** with zero-allocation hot paths
- **Dual execution modes**: Object-based (compatible) and Value-based (optimized)
- **179+ operators** covering math, boolean, string, array, date, and control flow operations

## Performance

Key optimizations achieve significant speedups:

| Operation | Baseline | Optimized | Speedup |
|-----------|----------|-----------|---------|
| Operator Lookup | 83 ns | 0.64 ns | **130x** |
| Integer Arithmetic | 19 ns | 0.79 ns | **24x** |
| String Creation | 41 ns | 11 ns | **3.7x** |

See [PERFORMANCE_ANALYSIS.md](pkg/dtrules/benchmark/PERFORMANCE_ANALYSIS.md) for detailed benchmarks.

## Installation

```bash
go get github.com/PaulSnow/DTRules/go
```

## Quick Start

### Using the CLI

```bash
# Build the CLI
go build -o dtrules ./cmd/dtrules

# List decision tables
./dtrules -rules /path/to/rules -list

# Validate rules
./dtrules -rules /path/to/rules -validate

# Execute a decision table
./dtrules -rules /path/to/rules -entry Compute_Eligibility

# Execute with tracing
./dtrules -rules /path/to/rules -entry Main -trace
```

### Using as a Library

```go
package main

import (
    "os"
    "github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

func main() {
    // Create a rule set
    rs := session.NewRuleSet("MyRules")

    // Load EDD (Entity Data Dictionary)
    eddFile, _ := os.Open("EDD.xml")
    rs.LoadEDD(eddFile)
    eddFile.Close()

    // Load Decision Tables
    dtFile, _ := os.Open("DecisionTables.xml")
    rs.LoadDecisionTables(dtFile)
    dtFile.Close()

    // Create a session and execute
    sess, _ := rs.NewSession()
    rsess := sess.(*session.RSession)
    rsess.Execute("Compute_Eligibility")
}
```

## Architecture

DTRules uses a three-stack interpreter similar to PostScript:

- **Data Stack**: Holds operands and intermediate values
- **Entity Stack**: Provides scoped dictionary lookup (context)
- **Control Stack**: Manages stack frames and local variables

```
┌─────────────────────────────────────────────┐
│              DTRules Runtime                │
├─────────────────────────────────────────────┤
│  Expression → Compiler → Executable Code    │
│                              │              │
│              ┌───────────────┼───────────┐  │
│              ▼               ▼           │  │
│       ┌───────────┐   ┌───────────┐      │  │
│       │  Object   │   │   Value   │      │  │
│       │   Stack   │   │   Stack   │      │  │
│       └───────────┘   └───────────┘      │  │
│              │               │           │  │
│              └───────┬───────┘           │  │
│                      ▼                   │  │
│               ┌───────────┐              │  │
│               │  Entity   │              │  │
│               │   Stack   │              │  │
│               └───────────┘              │  │
└─────────────────────────────────────────────┘
```

## Package Structure

```
go/
├── cmd/dtrules/          # CLI tool
└── pkg/dtrules/
    ├── compiler/         # Postfix expression compiler
    ├── interpreter/      # State and bytecode VM
    ├── entity/           # Entity system
    ├── operators/        # 179+ built-in operators
    ├── session/          # Session and RuleSet management
    ├── decisiontable/    # Decision table execution
    ├── loader/           # XML loaders (EDD, DT)
    ├── mapping/          # Data mapping
    └── benchmark/        # Performance benchmarks
```

## Decision Table Types

- **BALANCED**: All condition branches must be defined
- **FIRST**: Executes only the first matching column
- **ALL**: Executes all matching columns in order

## Testing

```bash
# Run all tests
go test ./...

# Run benchmarks
go test -bench=. ./pkg/dtrules/benchmark/... -benchmem

# Run specific tests
go test -v ./pkg/dtrules -run TestValue
```

## Documentation

- [Design Review](pkg/dtrules/DESIGN_REVIEW.md) - Architecture and design decisions
- [Performance Analysis](pkg/dtrules/benchmark/PERFORMANCE_ANALYSIS.md) - Detailed benchmarks

## License

Apache License 2.0 - See [LICENSE](../LICENSE) for details.

## Contributing

Contributions welcome. Please ensure:
- All tests pass (`go test ./...`)
- Code follows Go conventions (`go fmt`, `go vet`)
- New features include tests
