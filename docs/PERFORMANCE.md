# State Tax Calculation Performance

## Overview

This document describes the performance testing and optimization work for state tax calculations in DTRules, focusing on benchmarking single-state, multi-state, and combined federal+state tax computations.

## Performance Targets

The following performance targets have been established based on typical tax calculation workloads:

| Scenario | Target | Rationale |
|----------|--------|-----------|
| **Single State Tax** | <50ms | Individual state calculation should be near-instantaneous for user-facing applications |
| **Multi-State Tax** | <200ms | Multiple state allocations involve additional complexity but must remain responsive |
| **Federal + State** | <150ms | Combined federal and state calculations are the most common use case |
| **All States** | <200ms | Batch processing across all implemented states |

## Benchmark Suite

### Location

The comprehensive benchmark suite is located at:
```
go/pkg/dtrules/state_tax_bench_test.go
```

### Benchmark Categories

#### 1. Single State Tax Benchmarks

Tests individual state tax calculations using real test scenarios:

```go
BenchmarkSingleStateTax/NH_Single_W2
BenchmarkSingleStateTax/NH_MFJ_Two_Brackets
BenchmarkSingleStateTax/NH_High_Income
```

**Test Cases:**
- `TestCase_NH_01_Single_W2.xml` - Single filer with W-2 income
- `TestCase_NH_02_MFJ_Two_Brackets.xml` - Married filing jointly across two tax brackets
- `TestCase_NH_03_High_Income_All_Brackets.xml` - High earner hitting all tax brackets

**What's Measured:**
- Session creation
- Test data loading (mapping XML to entities)
- Decision table execution
- Tax calculation logic

#### 2. Multi-State Tax Benchmarks

Tests multi-state resident allocation scenarios:

```go
BenchmarkMultiStateTax/NY_FL_Move
BenchmarkMultiStateTax/CA_TX_Move
BenchmarkMultiStateTax/Traveling_Consultant
BenchmarkMultiStateTax/MT_TX_Move
BenchmarkMultiStateTax/NH_FL
```

**Test Cases:**
- Part-year residents who moved between states
- Traveling consultants working in multiple jurisdictions
- States with and without income tax

**What's Measured:**
- Income allocation across multiple states
- State period processing
- Date-based allocation calculations
- Dispatch to multiple state tax calculators

#### 3. Federal + State Combined Benchmarks

Tests the most common scenario of combined federal and state tax calculation:

```go
BenchmarkFederalPlusState/Family_2025
BenchmarkFederalPlusState/NH_MFJ
```

**Test Cases:**
- `TestCase_Family_2025.xml` - Family scenario with complex federal tax (OBBBA deductions, credits, etc.)
- `TestCase_NH_02_MFJ_Two_Brackets.xml` - Married couple with federal and state tax

**What's Measured:**
- Federal tax calculation (AGI, deductions, credits)
- State tax calculation
- Combined end-to-end execution time

#### 4. Component Benchmarks

Isolates individual components to identify bottlenecks:

```go
BenchmarkStateTaxComponents/SessionCreation
BenchmarkStateTaxComponents/DataLoading
BenchmarkStateTaxComponents/Execution
```

**What's Measured:**
- Pure session creation overhead
- XML mapping and data loading time
- Decision table execution time (isolated)

#### 5. All States Benchmark

Tests batch processing across all implemented states:

```go
BenchmarkAllStates
```

Iterates through all state test files to measure aggregate performance.

### Performance Validation Tests

In addition to benchmarks, we have validation tests that enforce performance targets:

```go
TestPerformanceTargets/SingleState_Target_50ms
TestPerformanceTargets/MultiState_Target_200ms
TestPerformanceTargets/FederalPlusState_Target_150ms
```

These tests:
- Run 20 iterations to get stable averages
- Compare against target thresholds
- Log warnings (not failures) when targets are exceeded
- Provide actionable performance data

## Running Benchmarks

### Basic Benchmark Execution

```bash
cd go

# Run all state tax benchmarks
go test -bench=. ./pkg/dtrules/state_tax_bench_test.go -benchmem

# Run specific benchmark category
go test -bench=BenchmarkSingleStateTax ./pkg/dtrules -benchmem
go test -bench=BenchmarkMultiStateTax ./pkg/dtrules -benchmem
go test -bench=BenchmarkFederalPlusState ./pkg/dtrules -benchmem

# Run with more iterations for stable results
go test -bench=BenchmarkSingleStateTax ./pkg/dtrules -benchmem -benchtime=10s

# Run specific test case
go test -bench=BenchmarkSingleStateTax/NH_Single_W2 ./pkg/dtrules -benchmem
```

### Performance Target Validation

```bash
# Run performance validation tests
go test -run TestPerformanceTargets ./pkg/dtrules -v

# Skip performance tests in short mode
go test -short ./pkg/dtrules
```

### Profiling

Generate CPU and memory profiles to identify bottlenecks:

```bash
# CPU profiling
go test -bench=BenchmarkMultiStateTax \
  -cpuprofile=cpu.prof \
  ./pkg/dtrules

# Analyze CPU profile
go tool pprof cpu.prof
# Interactive commands:
# - top10: Show top 10 functions by CPU time
# - list functionName: Show annotated source
# - web: Generate visual graph (requires graphviz)

# Memory profiling
go test -bench=BenchmarkSingleStateTax \
  -memprofile=mem.prof \
  ./pkg/dtrules

# Analyze memory profile
go tool pprof mem.prof

# Allocation profiling (more detailed)
go test -bench=BenchmarkSingleStateTax \
  -memprofile=mem.prof \
  -memprofilerate=1 \
  ./pkg/dtrules
```

### Benchmark Report Generation

```bash
# Generate formatted report
GENERATE_REPORT=1 go test -bench=BenchmarkReport ./pkg/dtrules -v

# Save benchmark results for comparison
go test -bench=. ./pkg/dtrules -benchmem > benchmark_baseline.txt

# Compare before/after optimization
go test -bench=. ./pkg/dtrules -benchmem > benchmark_optimized.txt
benchcmp benchmark_baseline.txt benchmark_optimized.txt
```

## Benchmark Results Interpretation

### Output Format

```
BenchmarkSingleStateTax/NH_Single_W2-24    100   12345678 ns/op   12345 B/op   123 allocs/op
```

This means:
- **100**: Number of iterations run
- **12345678 ns/op**: 12.3ms per operation (average)
- **12345 B/op**: 12KB allocated per operation
- **123 allocs/op**: 123 allocations per operation

### Key Metrics

1. **ns/op (nanoseconds per operation)**
   - Primary performance metric
   - Compare against targets (50ms = 50,000,000 ns)
   - Lower is better

2. **B/op (bytes per operation)**
   - Memory allocation per tax calculation
   - Indicates memory pressure
   - Watch for excessive allocations

3. **allocs/op (allocations per operation)**
   - Number of heap allocations
   - High allocation counts can impact GC performance
   - Optimization target for hot paths

## Performance Optimization Strategy

### 1. Establish Baseline

```bash
go test -bench=. ./pkg/dtrules -benchmem -count=5 > baseline.txt
```

Run multiple times to account for variance.

### 2. Profile to Find Bottlenecks

```bash
go test -bench=BenchmarkMultiStateTax -cpuprofile=cpu.prof ./pkg/dtrules
go tool pprof -top cpu.prof
```

Look for:
- Functions consuming >10% CPU time
- Unexpected function calls
- Excessive allocations

### 3. Optimize Hot Paths

Common optimizations:
- **Reduce allocations**: Use sync.Pool for frequently allocated objects
- **Avoid reflection**: Cache type information
- **Optimize loops**: Reduce iterations, avoid nested loops
- **String operations**: Use strings.Builder, avoid concatenation
- **Map lookups**: Cache frequently accessed values

### 4. Validate Optimizations

```bash
go test -bench=. ./pkg/dtrules -benchmem -count=5 > optimized.txt
benchcmp baseline.txt optimized.txt
```

### 5. Run Correctness Tests

```bash
go test ./pkg/dtrules -v
```

Ensure optimizations don't break functionality.

## Test Data

### Test Files Location

```
sampleprojects/TaxReturn/testfiles/TestScenarios/
```

### Available Test Cases

#### Single State Tests
- `TestCase_NH_01_Single_W2.xml` - New Hampshire single filer
- `TestCase_NH_02_MFJ_Two_Brackets.xml` - NH married filing jointly
- `TestCase_NH_03_High_Income_All_Brackets.xml` - NH high earner

#### Multi-State Tests
- `TestCase_MultiState_01_NY_FL_Move.xml` - NY to FL move (212 days / 153 days)
- `TestCase_MultiState_02_CA_TX_Move.xml` - CA to TX move (273 days / 92 days)
- `TestCase_MultiState_03_Traveling_Consultant.xml` - Three states (NH/MA/NY)
- `TestCase_MultiState_02_MT_TX.xml` - Montana to Texas move
- `TestCase_MultiState_01_NH_FL.xml` - NH to FL part-year

#### Federal + State Tests
- `TestCase_Family_2025.xml` - Complex family scenario with OBBBA provisions
- All NH test cases include federal calculations

## Implementation Details

### Architecture

Each benchmark follows this pattern:

1. **Setup Phase** (not measured)
   - Load TaxReturn EDD (Entity Data Dictionary)
   - Load Decision Tables
   - Create RuleSet

2. **Warmup Iteration** (not measured)
   - Create session
   - Load test data
   - Execute once to warm caches

3. **Benchmark Loop** (measured)
   - Create fresh session
   - Load test data via XML mapping
   - Execute `Compute_Tax_Return` decision table
   - Measure total time

### Session Lifecycle

Each benchmark iteration creates a fresh session to ensure:
- No state pollution between iterations
- Realistic performance measurement
- Cache behavior matches production

### Test Data Loading

Uses the DTRules mapping system to load XML test data:
1. Load mapping definition (`TaxReturn_map.xml`)
2. Initialize mapping
3. Load test case XML
4. Map XML elements to entities

This reflects real-world usage where tax data comes from external sources.

## Known Performance Characteristics

### Session Creation

**Typical Performance**: 30-50μs
**Components**:
- Entity factory initialization
- State stack allocation
- Decision table registration

**Optimization**: Session creation is fast; focus optimization elsewhere.

### Data Loading

**Typical Performance**: 5-10ms
**Components**:
- XML parsing
- Entity creation
- Attribute mapping
- Array population

**Optimization**: Could cache parsed structures, but loading is not the bottleneck.

### Decision Table Execution

**Typical Performance**: 20-40ms (depends on complexity)
**Components**:
- Rule evaluation
- Entity attribute lookup
- Arithmetic operations
- Control flow (if/then/else)

**Optimization**: This is the primary optimization target.

## Future Work

### Additional Benchmarks

1. **State Coverage**
   - Add benchmarks for all 50 states as they're implemented
   - Test edge cases (no income tax states, local taxes)

2. **Scaling Tests**
   - Batch processing (1000s of returns)
   - Concurrent execution
   - Memory usage under load

3. **Regression Testing**
   - Automated performance monitoring
   - CI/CD integration
   - Alert on performance degradation

### Optimization Opportunities

Based on initial profiling, potential optimizations include:

1. **Entity Attribute Caching**
   - Cache frequently accessed attributes
   - Reduce entity lookup overhead

2. **Decision Table Optimization**
   - Precompile rule conditions
   - Optimize rule ordering
   - Short-circuit evaluation

3. **Memory Pool**
   - Reuse entity objects
   - Pool temporary calculation values
   - Reduce GC pressure

4. **Parallel Execution**
   - Parallel state tax calculation for multi-state returns
   - Concurrent decision table evaluation (where safe)

## References

### Related Documentation

- `go/pkg/dtrules/runtime/runtime_benchmark_test.go` - Runtime operation benchmarks
- `docs/go-performance.md` - Go implementation performance analysis
- `sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md` - Multi-state allocation design

### Tax Calculation Test Cases

All test cases include expected results and are validated for correctness before benchmarking.

See:
- `go/pkg/dtrules/taxreturn_results_test.go` - Correctness tests
- `sampleprojects/TaxReturn/testfiles/TestScenarios/` - Test data

## Troubleshooting

### Benchmark Variance

If benchmark results vary significantly between runs:

```bash
# Run with more iterations
go test -bench=. ./pkg/dtrules -benchmem -benchtime=30s

# Run multiple times and average
for i in {1..5}; do
  go test -bench=BenchmarkSingleStateTax ./pkg/dtrules -benchmem >> results.txt
done
```

### Out of Memory

If benchmarks cause OOM errors:

```bash
# Increase available memory
GOGC=50 go test -bench=. ./pkg/dtrules -benchmem
```

### Build Failures

Ensure prerequisites are met:

```bash
# Check Go version
go version  # Should be 1.19+

# Update dependencies
go mod tidy
go mod download
```

## Summary

This benchmark suite provides comprehensive performance testing for state tax calculations:

- ✅ **Single state**: NH progressive tax (3 scenarios)
- ✅ **Multi-state**: Part-year residents and traveling workers (5 scenarios)
- ✅ **Federal + state**: Combined calculations (2 scenarios)
- ✅ **Component isolation**: Session, loading, execution
- ✅ **Performance targets**: 50ms / 200ms / 150ms
- ✅ **Profiling support**: CPU and memory profiles
- ✅ **Validation tests**: Automated target checking

The benchmarks use real test data and decision tables to ensure results reflect actual production performance.
