# State Tax Performance Benchmarks

## Status

This directory contains comprehensive performance benchmarks for state tax calculations. The benchmark suite has been implemented in `state_tax_bench_test.go`.

## Current State

### ✅ Completed

1. **Benchmark Suite Implementation**
   - Single state tax benchmarks (NH test cases)
   - Multi-state resident allocation benchmarks (5 scenarios)
   - Federal + state combined benchmarks
   - Component isolation benchmarks
   - Performance target validation tests

2. **Documentation**
   - Comprehensive PERFORMANCE.md guide
   - Benchmark methodology
   - Profiling instructions
   - Optimization strategies

3. **Performance Targets**
   - Single State: <50ms
   - Multi-State: <200ms
   - Federal + State: <150ms

### ⚠️ Known Issues

The Go implementation currently has pre-existing compilation errors in the `operators` package that prevent the benchmarks from running:

```
pkg/dtrules/operators/datetime.go: undefined: dtrules.AsInterval
pkg/dtrules/operators/datetime.go: undefined: dtrules.IntervalDays
pkg/dtrules/operators/datetime.go: undefined: dtrules.IntervalMonths
pkg/dtrules/operators/datetime.go: undefined: dtrules.IntervalYears
pkg/dtrules/operators/datetime.go: undefined: dtrules.NewRInterval
pkg/dtrules/operators/control.go: state.GetEntityProvider undefined
pkg/dtrules/operators/stack.go: state.GetANode undefined
```

These appear to be incomplete implementations of interval/datetime functionality and entity provider methods in the State type.

### 🔧 Resolution

The benchmark code is structurally complete and follows Go best practices. Once the underlying compilation issues in the `operators` package are resolved, the benchmarks should run without modification.

The benchmarks are designed to work with the existing DTRules architecture:
- Uses `session.NewRuleSet()` for rule loading
- Uses `session.Session` for execution
- Uses `mapping.NewMapping()` for XML data loading
- Uses standard decision table execution via `dt.Execute(state)`

## Files

### state_tax_bench_test.go

Comprehensive benchmark suite with the following functions:

**Setup Functions:**
- `setupTaxReturn(b *testing.B)` - Loads EDD and Decision Tables
- `loadTestData(b, sess, file)` - Loads test XML data
- `executeTaxReturn(b, sess)` - Executes Compute_Tax_Return

**Benchmark Functions:**
- `BenchmarkSingleStateTax` - Tests NH single state scenarios
- `BenchmarkAllStates` - Tests all implemented states
- `BenchmarkMultiStateTax` - Tests multi-state allocation scenarios
- `BenchmarkFederalPlusState` - Tests combined federal + state
- `BenchmarkStateTaxComponents` - Tests individual components

**Validation Functions:**
- `TestPerformanceTargets` - Validates performance against targets
- Test helper functions for non-benchmark execution

**Report Function:**
- `BenchmarkReport` - Generates formatted performance report

## Test Data

The benchmarks use real test scenarios from:
```
sampleprojects/TaxReturn/testfiles/TestScenarios/
```

### Single State Tests
- `TestCase_NH_01_Single_W2.xml`
- `TestCase_NH_02_MFJ_Two_Brackets.xml`
- `TestCase_NH_03_High_Income_All_Brackets.xml`

### Multi-State Tests
- `TestCase_MultiState_01_NY_FL_Move.xml`
- `TestCase_MultiState_02_CA_TX_Move.xml`
- `TestCase_MultiState_03_Traveling_Consultant.xml`
- `TestCase_MultiState_02_MT_TX.xml`
- `TestCase_MultiState_01_NH_FL.xml`

### Federal + State Tests
- `TestCase_Family_2025.xml`
- `TestCase_NH_02_MFJ_Two_Brackets.xml`

## Usage (Once Build Issues Resolved)

### Running Benchmarks

```bash
# All state tax benchmarks
go test -bench=. ./pkg/dtrules/state_tax_bench_test.go -benchmem

# Specific categories
go test -bench=BenchmarkSingleStateTax ./pkg/dtrules -benchmem
go test -bench=BenchmarkMultiStateTax ./pkg/dtrules -benchmem
go test -bench=BenchmarkFederalPlusState ./pkg/dtrules -benchmem

# With CPU profiling
go test -bench=BenchmarkMultiStateTax -cpuprofile=cpu.prof ./pkg/dtrules
go tool pprof cpu.prof

# With memory profiling
go test -bench=BenchmarkSingleStateTax -memprofile=mem.prof ./pkg/dtrules
go tool pprof mem.prof
```

### Performance Validation

```bash
# Run performance target tests
go test -run TestPerformanceTargets ./pkg/dtrules -v

# Skip in short mode
go test -short ./pkg/dtrules
```

### Generate Report

```bash
GENERATE_REPORT=1 go test -bench=BenchmarkReport ./pkg/dtrules -v
```

## Benchmark Design

### Why Fresh Sessions?

Each benchmark iteration creates a fresh session to:
1. Avoid state pollution between iterations
2. Measure realistic performance (includes session overhead)
3. Test cache behavior under realistic conditions

### Warmup Strategy

Each benchmark runs one warmup iteration before `b.ResetTimer()` to:
1. Prime the Go runtime
2. Load decision tables into memory
3. Warm up any internal caches
4. Ensure stable measurements

### Component Isolation

The `BenchmarkStateTaxComponents` suite isolates:
1. **SessionCreation** - Pure session creation overhead (~30-50μs expected)
2. **DataLoading** - XML parsing and mapping (~5-10ms expected)
3. **Execution** - Decision table execution (~20-40ms expected)

This helps identify which component needs optimization.

## Expected Performance

Based on similar DTRules benchmarks in `runtime_benchmark_test.go`:

### Micro-operations
- Integer arithmetic: ~1 ns/op
- Operator lookup: ~1 ns/op (indexed)
- Stack operations: ~2 ns/op
- Entity attribute access: ~10-50 ns/op

### Integration
- Decision table execution: 29-60 μs (simple cases)
- Session creation: 30 μs
- Rule set load: 1.5 ms

### Tax Calculations (Projected)
- Single state: 20-40 ms (10,000-20,000 operations)
- Multi-state: 60-150 ms (allocations + multiple states)
- Federal + State: 50-120 ms (complex federal + state)

These should all meet the targets (<50ms, <200ms, <150ms respectively).

## Optimization Notes

If benchmarks exceed targets, focus on:

1. **Entity Attribute Lookups**
   - Cache frequently accessed attributes
   - Reduce indirection

2. **Decision Table Evaluation**
   - Optimize rule ordering
   - Short-circuit evaluation
   - Reduce unnecessary entity stack operations

3. **Memory Allocations**
   - Use sync.Pool for temporary objects
   - Reduce intermediate value creation
   - Optimize string operations

4. **Multi-State Allocation**
   - Parallel state calculation (if safe)
   - Cache allocation percentages
   - Optimize date calculations

## Next Steps

1. **Fix Compilation Errors**
   - Implement missing interval types
   - Add missing State methods
   - Complete datetime operators

2. **Run Baseline Benchmarks**
   ```bash
   go test -bench=. ./pkg/dtrules -benchmem -count=5 > baseline.txt
   ```

3. **Profile Hotspots**
   ```bash
   go test -bench=BenchmarkMultiStateTax -cpuprofile=cpu.prof ./pkg/dtrules
   go tool pprof -top cpu.prof
   ```

4. **Optimize as Needed**
   - Focus on functions >10% CPU time
   - Reduce allocations in hot paths
   - Cache frequently computed values

5. **Validate Targets**
   ```bash
   go test -run TestPerformanceTargets ./pkg/dtrules -v
   ```

6. **Expand Test Coverage**
   - Add more state implementations
   - Test edge cases
   - Add stress tests

## References

- **Main Documentation**: `docs/PERFORMANCE.md`
- **Runtime Benchmarks**: `runtime/runtime_benchmark_test.go`
- **Tax Return Tests**: `taxreturn_results_test.go`
- **Go Performance Guide**: `docs/go-performance.md`
- **Multi-State Design**: `sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md`

## Contributing

When adding new state tax implementations:

1. Add test case XML to `sampleprojects/TaxReturn/testfiles/TestScenarios/`
2. Add benchmark case to appropriate function in `state_tax_bench_test.go`
3. Run benchmarks to ensure performance targets are met
4. Update this README with new test cases
5. Document any state-specific optimizations

## License

Copyright 2024 Paul Snow

Licensed under the Apache License, Version 2.0 (the "License").
See LICENSE file in repository root for details.
