#!/bin/bash
# DTRules Implementation Benchmark Suite
# Compares Assembly, Go, and Java implementations

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RULES_DIR="$SCRIPT_DIR/rules"
RESULTS_DIR="$SCRIPT_DIR/results"

# Configuration
WARMUP_ITERATIONS=10
BENCHMARK_ITERATIONS=100
COLD_START_ITERATIONS=50

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#------------------------------------------------------------------------------
# Header
#------------------------------------------------------------------------------

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║           DTRules Implementation Benchmark Suite                 ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

#------------------------------------------------------------------------------
# Detect available implementations
#------------------------------------------------------------------------------

ASM_BIN="$PROJECT_DIR/dtrules-asm"
GO_BIN=""
JAVA_JAR=""

echo "Detecting implementations..."
echo ""

# Assembly
if [ -f "$ASM_BIN" ]; then
    echo -e "  ${GREEN}✓${NC} Assembly: $ASM_BIN"
    ASM_AVAILABLE=1
else
    echo -e "  ${RED}✗${NC} Assembly: not found (run 'make' in asm/)"
    ASM_AVAILABLE=0
fi

# Go
if command -v dtrules-go &> /dev/null; then
    GO_BIN="dtrules-go"
    echo -e "  ${GREEN}✓${NC} Go: $(which dtrules-go)"
    GO_AVAILABLE=1
elif [ -f "$PROJECT_DIR/../go/dtrules-go" ]; then
    GO_BIN="$PROJECT_DIR/../go/dtrules-go"
    echo -e "  ${GREEN}✓${NC} Go: $GO_BIN"
    GO_AVAILABLE=1
else
    echo -e "  ${YELLOW}○${NC} Go: not found"
    GO_AVAILABLE=0
fi

# Java
if [ -f "$PROJECT_DIR/../java/target/dtrules.jar" ]; then
    JAVA_JAR="$PROJECT_DIR/../java/target/dtrules.jar"
    echo -e "  ${GREEN}✓${NC} Java: $JAVA_JAR"
    JAVA_AVAILABLE=1
elif [ -f "$PROJECT_DIR/../DTRules/target/dtrules.jar" ]; then
    JAVA_JAR="$PROJECT_DIR/../DTRules/target/dtrules.jar"
    echo -e "  ${GREEN}✓${NC} Java: $JAVA_JAR"
    JAVA_AVAILABLE=1
else
    echo -e "  ${YELLOW}○${NC} Java: not found"
    JAVA_AVAILABLE=0
fi

echo ""

if [ $ASM_AVAILABLE -eq 0 ]; then
    echo "Error: Assembly implementation required for benchmarks"
    exit 1
fi

#------------------------------------------------------------------------------
# Setup
#------------------------------------------------------------------------------

mkdir -p "$RESULTS_DIR"
mkdir -p "$RULES_DIR"

# Create benchmark rules if they don't exist
if [ ! -f "$RULES_DIR/bench_simple.xml" ]; then
    echo "Creating benchmark rule files..."
    "$SCRIPT_DIR/create_rules.sh"
fi

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$RESULTS_DIR/benchmark_${TIMESTAMP}.txt"
CSV_FILE="$RESULTS_DIR/benchmark_${TIMESTAMP}.csv"

echo "Results will be saved to:"
echo "  Text: $REPORT_FILE"
echo "  CSV:  $CSV_FILE"
echo ""

# CSV header
echo "implementation,benchmark,metric,value,unit" > "$CSV_FILE"

#------------------------------------------------------------------------------
# Utility functions
#------------------------------------------------------------------------------

# Get memory usage of a process (RSS in KB)
get_memory() {
    local pid=$1
    if [ -f /proc/$pid/status ]; then
        grep VmRSS /proc/$pid/status | awk '{print $2}'
    else
        echo "0"
    fi
}

# Calculate statistics from a file of numbers
calc_stats() {
    local file=$1
    awk '
    BEGIN { min = 999999999; max = 0; sum = 0; count = 0 }
    {
        if ($1 < min) min = $1
        if ($1 > max) max = $1
        sum += $1
        values[count++] = $1
    }
    END {
        mean = sum / count

        # Sort for median and percentiles
        for (i = 0; i < count; i++) {
            for (j = i + 1; j < count; j++) {
                if (values[i] > values[j]) {
                    t = values[i]; values[i] = values[j]; values[j] = t
                }
            }
        }

        median = values[int(count/2)]
        p95 = values[int(count * 0.95)]
        p99 = values[int(count * 0.99)]

        # Standard deviation
        for (i = 0; i < count; i++) {
            variance += (values[i] - mean)^2
        }
        stddev = sqrt(variance / count)

        printf "%.3f %.3f %.3f %.3f %.3f %.3f %.3f\n", min, max, mean, median, p95, p99, stddev
    }
    ' "$file"
}

# Run a command and measure time in milliseconds
time_ms() {
    local start end
    start=$(date +%s%N)
    "$@" > /dev/null 2>&1
    end=$(date +%s%N)
    echo "scale=3; ($end - $start) / 1000000" | bc
}

#------------------------------------------------------------------------------
# Benchmark: Cold Start Latency
#------------------------------------------------------------------------------

run_cold_start_benchmark() {
    local name=$1
    local cmd=$2
    local args=$3
    local rule_file=$4

    echo -n "  $name: "

    local times_file=$(mktemp)

    for i in $(seq 1 $COLD_START_ITERATIONS); do
        # Clear filesystem cache (requires sudo, skip if not available)
        sync
        echo 3 | sudo tee /proc/sys/vm/drop_caches > /dev/null 2>&1 || true

        time_ms $cmd $args "$rule_file" >> "$times_file"
    done

    local stats=$(calc_stats "$times_file")
    local min=$(echo $stats | cut -d' ' -f1)
    local max=$(echo $stats | cut -d' ' -f2)
    local mean=$(echo $stats | cut -d' ' -f3)
    local median=$(echo $stats | cut -d' ' -f4)
    local p95=$(echo $stats | cut -d' ' -f5)
    local p99=$(echo $stats | cut -d' ' -f6)

    printf "min=%.1fms mean=%.1fms p99=%.1fms\n" $min $mean $p99

    echo "$name,cold_start,min,$min,ms" >> "$CSV_FILE"
    echo "$name,cold_start,mean,$mean,ms" >> "$CSV_FILE"
    echo "$name,cold_start,median,$median,ms" >> "$CSV_FILE"
    echo "$name,cold_start,p95,$p95,ms" >> "$CSV_FILE"
    echo "$name,cold_start,p99,$p99,ms" >> "$CSV_FILE"

    rm "$times_file"
}

#------------------------------------------------------------------------------
# Benchmark: Warm Throughput
#------------------------------------------------------------------------------

run_throughput_benchmark() {
    local name=$1
    local cmd=$2
    local args=$3
    local rule_file=$4

    echo -n "  $name: "

    # Warmup
    for i in $(seq 1 $WARMUP_ITERATIONS); do
        $cmd $args "$rule_file" > /dev/null 2>&1
    done

    local times_file=$(mktemp)

    for i in $(seq 1 $BENCHMARK_ITERATIONS); do
        time_ms $cmd $args "$rule_file" >> "$times_file"
    done

    local stats=$(calc_stats "$times_file")
    local min=$(echo $stats | cut -d' ' -f1)
    local mean=$(echo $stats | cut -d' ' -f3)
    local p99=$(echo $stats | cut -d' ' -f6)

    # Calculate ops/sec
    local ops_sec=$(echo "scale=1; 1000 / $mean" | bc)

    printf "%.1f ops/sec (mean=%.1fms p99=%.1fms)\n" $ops_sec $mean $p99

    echo "$name,throughput,ops_per_sec,$ops_sec,ops/s" >> "$CSV_FILE"
    echo "$name,throughput,mean_latency,$mean,ms" >> "$CSV_FILE"
    echo "$name,throughput,p99_latency,$p99,ms" >> "$CSV_FILE"

    rm "$times_file"
}

#------------------------------------------------------------------------------
# Benchmark: Memory Usage
#------------------------------------------------------------------------------

run_memory_benchmark() {
    local name=$1
    local cmd=$2
    local args=$3
    local rule_file=$4

    echo -n "  $name: "

    # Run with a sleep to capture memory
    if [ "$name" = "asm" ]; then
        # For ASM, just run and check /proc
        $cmd $args "$rule_file" &
        local pid=$!
        sleep 0.1
        local mem=$(get_memory $pid)
        wait $pid 2>/dev/null || true
    else
        # For Go/Java, run and capture peak memory
        /usr/bin/time -v $cmd $args "$rule_file" 2>&1 | grep "Maximum resident" | awk '{print $6}' > /tmp/mem_$$.txt &
        wait
        local mem=$(cat /tmp/mem_$$.txt 2>/dev/null || echo "0")
        rm -f /tmp/mem_$$.txt
    fi

    if [ -z "$mem" ] || [ "$mem" = "0" ]; then
        # Fallback: use /usr/bin/time
        mem=$(/usr/bin/time -v $cmd $args "$rule_file" 2>&1 | grep "Maximum resident" | awk '{print $6}')
    fi

    printf "%s KB\n" "$mem"

    echo "$name,memory,peak_rss,$mem,KB" >> "$CSV_FILE"
}

#------------------------------------------------------------------------------
# Benchmark: Binary/Startup Size
#------------------------------------------------------------------------------

run_size_benchmark() {
    echo ""
    echo "Binary Sizes:"

    if [ $ASM_AVAILABLE -eq 1 ]; then
        local asm_size=$(stat -c%s "$ASM_BIN" 2>/dev/null || stat -f%z "$ASM_BIN")
        local asm_size_kb=$((asm_size / 1024))
        echo "  asm:  ${asm_size_kb} KB"
        echo "asm,size,binary,$asm_size_kb,KB" >> "$CSV_FILE"
    fi

    if [ $GO_AVAILABLE -eq 1 ]; then
        local go_size=$(stat -c%s "$GO_BIN" 2>/dev/null || stat -f%z "$GO_BIN")
        local go_size_kb=$((go_size / 1024))
        echo "  go:   ${go_size_kb} KB"
        echo "go,size,binary,$go_size_kb,KB" >> "$CSV_FILE"
    fi

    if [ $JAVA_AVAILABLE -eq 1 ]; then
        local java_size=$(stat -c%s "$JAVA_JAR" 2>/dev/null || stat -f%z "$JAVA_JAR")
        local java_size_kb=$((java_size / 1024))
        echo "  java: ${java_size_kb} KB (JAR only, excludes JVM)"
        echo "java,size,binary,$java_size_kb,KB" >> "$CSV_FILE"
    fi
}

#------------------------------------------------------------------------------
# Run All Benchmarks
#------------------------------------------------------------------------------

echo "═══════════════════════════════════════════════════════════════════"
echo "BENCHMARK: Cold Start Latency ($COLD_START_ITERATIONS iterations)"
echo "═══════════════════════════════════════════════════════════════════"

for rule in "$RULES_DIR"/*.xml; do
    rule_name=$(basename "$rule" .xml)
    echo ""
    echo "Rule: $rule_name"

    if [ $ASM_AVAILABLE -eq 1 ]; then
        run_cold_start_benchmark "asm" "$ASM_BIN" "" "$rule"
    fi

    if [ $GO_AVAILABLE -eq 1 ]; then
        run_cold_start_benchmark "go" "$GO_BIN" "-rules" "$rule"
    fi

    if [ $JAVA_AVAILABLE -eq 1 ]; then
        run_cold_start_benchmark "java" "java" "-jar $JAVA_JAR" "$rule"
    fi
done

echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "BENCHMARK: Warm Throughput ($BENCHMARK_ITERATIONS iterations)"
echo "═══════════════════════════════════════════════════════════════════"

for rule in "$RULES_DIR"/*.xml; do
    rule_name=$(basename "$rule" .xml)
    echo ""
    echo "Rule: $rule_name"

    if [ $ASM_AVAILABLE -eq 1 ]; then
        run_throughput_benchmark "asm" "$ASM_BIN" "" "$rule"
    fi

    if [ $GO_AVAILABLE -eq 1 ]; then
        run_throughput_benchmark "go" "$GO_BIN" "-rules" "$rule"
    fi

    if [ $JAVA_AVAILABLE -eq 1 ]; then
        run_throughput_benchmark "java" "java" "-jar $JAVA_JAR" "$rule"
    fi
done

echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "BENCHMARK: Peak Memory Usage"
echo "═══════════════════════════════════════════════════════════════════"

# Use the largest rule file
LARGE_RULE=$(ls -S "$RULES_DIR"/*.xml | head -1)
echo ""
echo "Rule: $(basename "$LARGE_RULE" .xml)"

if [ $ASM_AVAILABLE -eq 1 ]; then
    run_memory_benchmark "asm" "$ASM_BIN" "" "$LARGE_RULE"
fi

if [ $GO_AVAILABLE -eq 1 ]; then
    run_memory_benchmark "go" "$GO_BIN" "-rules" "$LARGE_RULE"
fi

if [ $JAVA_AVAILABLE -eq 1 ]; then
    run_memory_benchmark "java" "java" "-jar $JAVA_JAR" "$LARGE_RULE"
fi

run_size_benchmark

#------------------------------------------------------------------------------
# Summary
#------------------------------------------------------------------------------

echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "SUMMARY"
echo "═══════════════════════════════════════════════════════════════════"
echo ""
echo "Results saved to:"
echo "  $REPORT_FILE"
echo "  $CSV_FILE"
echo ""
echo "To visualize results:"
echo "  python3 $SCRIPT_DIR/plot_results.py $CSV_FILE"
echo ""
