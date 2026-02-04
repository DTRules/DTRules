# DTRules Benchmark Suite

Performance comparison between Assembly, Go, and Java implementations.

## Quick Start

```bash
# Quick benchmark (ASM vs Go)
./quick_bench.sh

# Full benchmark suite
./run_benchmarks.sh

# Visualize results
python3 plot_results.py results/benchmark_*.csv
```

## Benchmarks

### 1. Cold Start Latency

Measures time from process start to completion, including:
- Binary loading
- Runtime initialization (Go/Java)
- XML parsing
- Rule execution

**Why it matters**: Critical for serverless, CLI tools, short-lived processes.

### 2. Warm Throughput

Measures operations per second after warmup, testing:
- Core execution speed
- Operator performance
- Memory allocation efficiency

**Why it matters**: Important for long-running servers, batch processing.

### 3. Memory Usage

Measures peak RSS (Resident Set Size):
- Baseline memory footprint
- Per-operation memory growth
- GC overhead (Go/Java)

**Why it matters**: Affects container sizing, multi-tenant deployments.

### 4. Binary Size

Compares deployment artifact sizes:
- ASM: Statically linked binary
- Go: Statically linked binary
- Java: JAR file (excludes JVM)

**Why it matters**: Affects container images, download times, disk usage.

## Test Rules

| File | Description | Complexity |
|------|-------------|------------|
| `bench_simple.xml` | Basic arithmetic | Minimal |
| `bench_medium.xml` | Stack + comparisons | Moderate |
| `bench_large.xml` | Many operations | High |
| `bench_compute.xml` | Computation-heavy | Stress test |

## Expected Results

### Assembly Advantages

- **Cold start**: 10-100x faster (no runtime)
- **Memory**: 10-50x less (no GC, minimal runtime)
- **Binary size**: 50-100x smaller

### Go/Java Advantages

- **Complex strings**: Optimized standard library
- **JIT (Java)**: Can optimize hot paths
- **Dynamic limits**: No fixed memory caps

## Sample Output

```
COLD START LATENCY (lower is better)
------------------------------------------------------------
Implementation        Min       Mean        P99
------------------------------------------------------------
asm                0.50ms     0.65ms     1.20ms
go                 8.50ms    10.20ms    15.50ms
java              85.00ms   120.50ms   180.00ms

Speedup vs ASM (cold start):
  go: ASM is 15.7x faster
  java: ASM is 185.4x faster
```

## Configuration

Edit `run_benchmarks.sh` to adjust:

```bash
WARMUP_ITERATIONS=10      # Iterations before measuring
BENCHMARK_ITERATIONS=100  # Measured iterations
COLD_START_ITERATIONS=50  # Cold start samples
```

## Requirements

- `bc` - for floating point math
- `time` (GNU) - for memory measurement
- Python 3 - for visualization (optional)

## Output Files

Results are saved to `results/`:
- `benchmark_YYYYMMDD_HHMMSS.txt` - Human readable
- `benchmark_YYYYMMDD_HHMMSS.csv` - Machine readable

CSV format:
```csv
implementation,benchmark,metric,value,unit
asm,cold_start,mean,0.65,ms
go,cold_start,mean,10.20,ms
```

## Adding Implementations

To benchmark additional implementations, add detection in `run_benchmarks.sh`:

```bash
# Example: Add Rust implementation
if [ -f "$PROJECT_DIR/../rust/target/release/dtrules-rs" ]; then
    RUST_BIN="$PROJECT_DIR/../rust/target/release/dtrules-rs"
    RUST_AVAILABLE=1
fi
```

Then add benchmark calls:
```bash
if [ $RUST_AVAILABLE -eq 1 ]; then
    run_cold_start_benchmark "rust" "$RUST_BIN" "" "$rule"
fi
```
