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

# Analyze a trace file
./dtrules -trace-file trace.xml

# Reconstruct state at a specific trace node
./dtrules -rules /path/to/rules -trace-file trace.xml -trace-node 428

# Generate coverage report from trace files
./dtrules -rules /path/to/rules -coverage ./output/

# Run tests from a test directory
./dtrules -rules /path/to/rules -test ./testfiles/ -test-output ./output/ -entry Main
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
├── cmd/
│   ├── dtrules/          # CLI tool
│   └── api/              # REST API server for UI
└── pkg/dtrules/
    ├── compiler/         # Postfix expression compiler
    ├── interpreter/      # State and bytecode VM
    ├── entity/           # Entity system
    ├── operators/        # 179+ built-in operators
    ├── session/          # Session and RuleSet management
    ├── decisiontable/    # Decision table execution
    ├── loader/           # XML loaders (EDD, DT)
    ├── mapping/          # Data mapping
    ├── benchmark/        # Performance benchmarks
    ├── trace/            # Trace file analysis and state reconstruction
    └── testsupport/      # Test harness, coverage analysis, change reports
```

## REST API Server

The API server provides HTTP endpoints for the DTRules UI:

```bash
# Start the API server (default port 8080)
go run ./cmd/api

# Start on a different port
go run ./cmd/api -port 9000

# With verbose logging
go run ./cmd/api -v
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/project/open` | POST | Open project directory |
| `/api/project/save` | POST | Save modified files |
| `/api/project/files` | GET | List project files |
| `/api/edd` | GET | Get all entities |
| `/api/edd/entity/{name}` | GET/POST/PUT/DELETE | Entity CRUD |
| `/api/dt` | GET/POST | List/create decision tables |
| `/api/dt/{name}` | GET/PUT/DELETE | Decision table CRUD |
| `/api/dt/{name}/tree` | GET | Get decision tree for visualization |
| `/api/compile/expression` | POST | Validate postfix expression |
| `/api/compile/operators` | GET | Get available operators |
| `/api/compile/fields` | GET | Get entity fields |
| `/api/execute` | POST | Execute rules |

## Trace Analysis

The `trace` package can parse XML trace files generated during rule execution and reconstruct the engine state at any point in the trace. This is useful for debugging and understanding how rules arrived at their results.

```go
import "github.com/PaulSnow/DTRules/go/pkg/dtrules/trace"

// Load and inspect a trace file
t := trace.NewTrace()
root, _ := t.Load("trace.xml")

// Find a specific node
node := t.Find(428)

// Reconstruct state at that point (requires a loaded RuleSet)
sess, _ := t.SetState(ruleSet, node)

// Check what changed
for _, change := range t.GetChanges() {
    fmt.Printf("Entity %d, attr %s\n", change.EntityID, change.AttributeKey)
}
```

## Test Support

The `testsupport` package provides infrastructure for testing rule sets:

- **TestHarness**: Executes test files against a rule set, generates trace and result files, and compares results against reference output.
- **Coverage**: Analyzes trace files to compute decision table column coverage, identifying which columns were exercised and which test files contribute to coverage.
- **ChangeReport**: Compares two versions of a rule set (EDD, decision tables, mappings) to identify structural and execution changes.

## Value and DTState Memory Layout

The `Value` type and `DTState` struct have documented memory layouts with field offsets, enabling low-level access from assembly or FFI code. Each type's layout is verified by tests (`value_layout_test.go` and `state_layout_test.go`) that assert field offsets and struct sizes match the documentation.

See the doc comments on `Value` in `pkg/dtrules/value.go` and `DTState` in `pkg/dtrules/interpreter/state.go` for the full offset tables.

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
