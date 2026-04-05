# DTRules Go Implementation - Performance Analysis

> **Note:** The canonical version of this document is at [docs/go-performance.md](../../../../docs/go-performance.md)

## Test Environment
- **CPU**: Intel Core Ultra 9 275HX (24 cores)
- **OS**: Linux 6.17.4
- **Go**: Current version
- **Date**: January 2026

---

## Executive Summary

All four optimizations have been implemented and tested:

| Optimization | Before | After | Speedup | Allocs |
|-------------|--------|-------|---------|--------|
| **Operator Lookup** | 83 ns | 0.64 ns | **130x** | 0 → 0 |
| **Value Arithmetic** | 19 ns | 0.79 ns | **24x** | 1 → 0 |
| **String Creation** | 41 ns | 11 ns | **3.7x** | 2 → 0 |
| **Bytecode Execution** | 55 ns | 59 ns | ~1x | 2 → 0 |

**Key Achievement**: Zero allocations in optimized hot paths, reducing GC pressure.

---

## Detailed Benchmark Results

### Micro-Operations

| Operation | Time (ns) | Allocs | Notes |
|-----------|----------|--------|-------|
| Name Lookup (string→RName) | 76 | 0 | Interned |
| Name Lookup (cached) | 3.5 | 0 | Direct access |
| **Operator Lookup (map)** | **83** | 0 | Original |
| **Operator Lookup (indexed)** | **0.64** | 0 | **130x faster** |
| Stack Push/Pop | 1.8 | 0 | Optimal |
| **Integer Arithmetic (Object)** | **19** | 1 | Original |
| **Integer Arithmetic (Value)** | **0.79** | 0 | **24x faster** |
| Value Complex Calc | 17 | 0 | 5 ops |
| Value Creation | 0.1 | 0 | Stack allocation |
| Value→Object | 9.5 | 1 | Conversion |
| Integer Creation | 0.1 | 0 | Cached |
| **String Creation (interned)** | **11** | 0 | **3.7x faster** |
| String Creation (unique) | 85 | 3 | Long strings |
| Boolean Operations | 6.9 | 0 | Singletons |
| Comparison | 7.9 | 0 | Return booleans |
| String Concat | 40 | 1 | Builder |

### Integration Benchmarks

| Operation | Time | Allocs | Bytes |
|-----------|------|--------|-------|
| Rule Set Load | 1.55 ms | 20,054 | 1.1 MB |
| Session Create | 30 μs | 16 | 88 KB |
| Decision Table Exec | 29 μs | 30 | 89 KB |
| Complex Scenario | 59 ns | 2 | 23 B |
| Many Operations | 387 ns | 18 | 432 B |

### Bytecode vs Object Execution

| Path | Time (ns) | Allocs | Bytes |
|------|----------|--------|-------|
| Bytecode Full | 59 | 0 | 0 |
| Object Full | 55 | 2 | 16 |
| Bytecode Compile | 1,956 | 63 | 2,712 |
| Object Compile | 1,862 | 63 | 2,440 |
| Bytecode VM Only | 44 | 0 | 0 |

**Note**: Object path is slightly faster in raw execution, but bytecode has zero allocations, which matters more for long-running applications with many rule evaluations.

---

## Memory Benchmarks

| Operation | Bytes/op | Allocs/op | Notes |
|-----------|----------|-----------|-------|
| Integer (large) | 0 | 0 | sync.Pool |
| String (interned) | 0 | 0 | Reused |
| String (unique) | 176 | 3 | Long |
| Name (new) | 48 | 1 | With parsing |
| Stack Ops | 0 | 0 | Pre-allocated |
| Bytecode Size | 1,162 | - | vs 1,600 for Object |

**Memory savings**: Bytecode is 27% smaller than Object arrays.

---

## Implementation Files

### New Files Created
| File | Purpose | Lines |
|------|---------|-------|
| `value.go` | Tagged union value type | 280 |
| `value_test.go` | Value tests | 230 |
| `bytecode.go` | Bytecode encoding | 230 |
| `bytecode_test.go` | Bytecode tests | 200 |
| `interpreter/vm.go` | Bytecode VM | 280 |
| `compiler/bytecode_compiler.go` | Bytecode compiler | 180 |
| `DESIGN_REVIEW.md` | Design documentation | 200 |
| `PERFORMANCE_ANALYSIS.md` | This document | 350 |

### Modified Files
| File | Changes |
|------|---------|
| `string.go` | Added string interning |
| `operators/registry.go` | Added indexed lookup |
| `interpreter/state.go` | Added Value stack |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     DTRules Go Runtime                          │
├─────────────────────────────────────────────────────────────────┤
│  Expression Input                                               │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────┐      ┌─────────────┐                          │
│  │  Tokenizer  │──────│  Compiler   │                          │
│  └─────────────┘      └─────────────┘                          │
│                             │                                   │
│              ┌──────────────┼──────────────┐                   │
│              ▼              ▼              ▼                   │
│       ┌───────────┐  ┌───────────┐  ┌───────────┐             │
│       │  Object   │  │ Bytecode  │  │   Both    │             │
│       │  Arrays   │  │  Chunks   │  │ (future)  │             │
│       └───────────┘  └───────────┘  └───────────┘             │
│              │              │                                   │
│              ▼              ▼                                   │
│       ┌───────────┐  ┌───────────┐                             │
│       │  Object   │  │   Value   │                             │
│       │   Stack   │  │   Stack   │                             │
│       │(dataStk)  │  │(valueStk) │                             │
│       └───────────┘  └───────────┘                             │
│              │              │                                   │
│              └──────┬───────┘                                   │
│                     ▼                                           │
│              ┌───────────┐                                      │
│              │  Entity   │                                      │
│              │   Stack   │                                      │
│              │(entityStk)│                                      │
│              └───────────┘                                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Usage Guide

### Using Indexed Operator Lookup
```go
// At initialization
import "github.com/DTRules/DTRules/pkg/dtrules/operators"

// Get index once
addIdx, _ := operators.GetIndex(dtrules.GetRName("+"))

// Fast lookup
addOp := operators.GetByIndex(addIdx)
```

### Using Value-based Arithmetic
```go
import "github.com/DTRules/DTRules/pkg/dtrules"

a := dtrules.NewValueInteger(100)
b := dtrules.NewValueInteger(50)
result := a.Add(b).Mul(dtrules.NewValueInteger(2))
// result.AsInteger() == 300, zero allocations
```

### Using String Interning
```go
// Automatic for strings ≤64 chars
s := dtrules.GetRString("hello")  // Interned
s2 := dtrules.GetRString("hello") // Returns same object
```

### Using Bytecode Execution
```go
import (
    "github.com/DTRules/DTRules/pkg/dtrules"
    "github.com/DTRules/DTRules/pkg/dtrules/compiler"
)

// Compile to bytecode
comp := compiler.NewCompiler(session, factory)
bc, _ := comp.CompileToBytecode("100 50 + 2 * 1000 <")

// Execute
state.ExecuteBytecode(bc)
result, _ := state.ValuePop()
// result.AsBoolean() == true
```

---

## Benchmark Commands

```bash
# All benchmarks
go test -bench=. ./pkg/dtrules/benchmark/... -benchmem

# Specific comparisons
go test -bench=BenchmarkOperator ./pkg/dtrules/benchmark/... -benchmem
go test -bench=BenchmarkValue ./pkg/dtrules/benchmark/... -benchmem
go test -bench=BenchmarkBytecode ./pkg/dtrules/benchmark/... -benchmem

# Multiple runs
go test -bench=. ./pkg/dtrules/benchmark/... -benchmem -count=5

# CPU profile
go test -bench=BenchmarkDecisionTable -cpuprofile=cpu.prof ./pkg/dtrules/benchmark/...
go tool pprof cpu.prof

# Memory profile
go test -bench=BenchmarkDecisionTable -memprofile=mem.prof ./pkg/dtrules/benchmark/...
go tool pprof mem.prof
```

---

## Test Coverage

```bash
# Run all tests
go test ./...

# Test specific packages
go test ./pkg/dtrules -run "TestValue|TestBytecode" -v
go test ./pkg/dtrules/interpreter -v
go test ./pkg/dtrules/operators -v
```

Total test count: 200+ tests across all packages

---

## Conclusion

The Go implementation of DTRules achieves significant performance improvements over the baseline:

1. **130x faster operator dispatch** through indexed lookup
2. **24x faster arithmetic** through tagged union values
3. **3.7x faster string handling** through interning
4. **Zero allocations** in critical paths
5. **27% smaller bytecode** than Object arrays

The optimizations maintain full backward compatibility - existing code continues to work unchanged while new code can opt into the optimized paths.

The dual-path architecture (Object and Value stacks) allows incremental adoption and provides a clear migration path for performance-critical applications.
