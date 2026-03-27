# DTRules Go Implementation

A high-performance Go implementation of the DTRules decision table rules engine.

## Overview

DTRules is a stack-based rules engine that executes decision tables. This Go implementation provides:

- **Full compatibility** with DTRules XML formats (EDD and Decision Tables)
- **JSON support** for EDD definitions and data loading
- **High performance** with zero-allocation hot paths
- **Dual execution modes**: Object-based (compatible) and Value-based (optimized)
- **179+ operators** covering math, boolean, string, array, date, and control flow operations
- **Trace analysis** for debugging and state reconstruction
- **Test harness** with coverage analysis and change reporting

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
./dtrules -rules /path/to/rules -trace-file trace.xml -trace-node 428

# Generate coverage report from trace files
./dtrules -rules /path/to/rules -coverage /path/to/trace/output/

# Run tests from a test directory
./dtrules -rules /path/to/rules -test /path/to/testfiles/ -entry Main
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

    // Load EDD (Entity Data Dictionary) from XML
    eddFile, _ := os.Open("EDD.xml")
    rs.LoadEDD(eddFile)
    eddFile.Close()

    // Or load EDD from JSON
    // eddJSON, _ := os.Open("EDD.json")
    // rs.LoadEDDJSON(eddJSON)
    // eddJSON.Close()

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
    ├── loader/           # XML and JSON loaders (EDD, DT)
    ├── mapping/          # Data mapping (XML and JSON)
    ├── trace/            # Trace analysis and state reconstruction
    ├── testsupport/      # Test harness, coverage, and change reports
    └── benchmark/        # Performance benchmarks
```

## JSON Support

In addition to the traditional XML format, DTRules supports JSON for entity definitions and data loading.

### JSON EDD Format

Entity Data Dictionary definitions can be loaded from JSON:

```json
{
  "entities": [
    {
      "name": "person",
      "access": "rw",
      "comment": "A person entity",
      "fields": [
        {"name": "name", "type": "string", "access": "rw"},
        {"name": "age", "type": "integer", "access": "rw", "defaultValue": "0"},
        {"name": "active", "type": "boolean", "access": "r"}
      ]
    }
  ]
}
```

Supported field types: `string`, `integer`, `double`, `boolean`, `date`, `entity`, `array`.

### JSON Data Loading

JSON data can be loaded into entities via the mapping system:

```json
{
  "person": {"name": "John", "age": 30, "active": true},
  "orders": [
    {"order_id": 1, "total": 99.99},
    {"order_id": 2, "total": 49.50}
  ]
}
```

Top-level keys map to entity names. Values can be single objects or arrays of objects. Plural keys (e.g., `orders`) are automatically singularized to match entity names (e.g., `order`).

## Trace Analysis

The `trace` package provides post-execution analysis of DTRules trace files:

- Load and parse trace XML files into a navigable tree structure
- Reconstruct rules engine state at any point in the trace
- Track attribute changes during rule execution
- Walk trace trees depth-first with visitor functions

```bash
# Print trace tree
./dtrules -trace-file output/test_trace.xml

# Reconstruct state at a specific node
./dtrules -rules ./rules -trace-file output/test_trace.xml -trace-node 428 -v
```

## Test Harness

The `testsupport` package provides test infrastructure:

- **TestHarness**: Execute test files against rule sets, compare results with reference outputs
- **Coverage**: Analyze trace files to compute decision table column coverage
- **ChangeReport**: Compare two versions of a rule set to detect structural and execution changes

```bash
# Run tests and generate output
./dtrules -rules ./rules -test ./testfiles/ -entry Main -trace

# Generate coverage report
./dtrules -rules ./rules -coverage ./output/
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
