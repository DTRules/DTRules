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

# Analyze a trace file (print tree)
./dtrules -trace-file trace.xml

# Reconstruct state at a specific trace node
./dtrules -rules /path/to/rules -trace-file trace.xml -trace-node 428

# Generate coverage report from trace files
./dtrules -rules /path/to/rules -coverage ./output/

# Run test harness
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

## Runtime Architecture

Each runtime owns its own VMState (stacks, frames, session reference). The `Runtime` interface
combines three concerns:

- **`RuntimeInit`** — push entities, values, and data onto the runtime's stacks before execution.
- **`RuntimeQuery`** — read entities, values, and data from the stacks after execution.
- **`Runtime`** — execute bytecode, and bridge back to the `State` interface for existing code.

The Go interpreter's `DTState` implements `Runtime`. Alternative runtimes (e.g. native assembly)
can implement the same interface. Use `RuntimeFactory` to create runtime instances:

```go
factory := &interpreter.GoRuntimeFactory{}
rt, err := factory.CreateRuntime(session)

// Push initial state
rt.PushEntity(myEntity)

// Execute
rt.ExecuteBytecode(compiled)

// Read results
value, _ := rt.PopValue()
```

## Package Structure

```
go/
├── cmd/
│   ├── dtrules/          # CLI tool
│   └── api/              # REST API server for UI
└── pkg/dtrules/
    ├── compiler/         # Postfix expression compiler
    ├── interpreter/      # State, bytecode VM, and Runtime implementation
    ├── entity/           # Entity system
    ├── operators/        # 179+ built-in operators
    ├── session/          # Session and RuleSet management
    ├── decisiontable/    # Decision table execution
    ├── loader/           # XML loaders (EDD, DT)
    ├── mapping/          # Data mapping
    ├── trace/            # Trace file analysis and state reconstruction
    ├── testsupport/      # Test harness, coverage, and change reports
    └── benchmark/        # Performance benchmarks
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

## Trace Analysis

The `trace` package parses XML trace files produced during rule execution and reconstructs
the engine state at any point. This is useful for debugging which rules fired, what values
were assigned, and how entities changed over time.

```go
t := trace.NewTrace()
root, _ := t.Load("trace.xml")

// Print the trace tree
t.Print(os.Stdout)

// Reconstruct state at node 428
sess, _ := t.SetState(ruleSet, t.Find(428))

// Inspect changes
for _, change := range t.GetChanges() {
    fmt.Println(change)
}
```

## Test Support

The `testsupport` package provides:

- **TestHarness** — runs XML test files against a rule set, producing result and trace output files.
- **Coverage** — analyzes trace files to compute decision-table column coverage.
- **ChangeReport** — compares two versions of a rule set (EDD, decision tables, mappings) and reports structural and execution differences.

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
