#!/bin/bash
# Quick benchmark - runs a simple comparison
# Usage: ./quick_bench.sh [iterations]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(dirname "$PROJECT_DIR")"

ITERATIONS=${1:-100}

ASM_BIN="$PROJECT_DIR/dtrules-asm"
GO_BIN="$REPO_ROOT/go/dtrules-go"

# CHIP sample project paths
CHIP_EDD="$REPO_ROOT/sampleprojects/CHIP/repository/xml/CHIP_edd.xml"
CHIP_DT="$REPO_ROOT/sampleprojects/CHIP/repository/xml/CHIP_dt.xml"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║            DTRules Quick Benchmark                           ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Check ASM binary
if [ ! -f "$ASM_BIN" ]; then
    echo "Error: dtrules-asm not found. Run 'make' in asm/ first."
    exit 1
fi

# Check Go binary
if [ ! -f "$GO_BIN" ]; then
    # Try to find in PATH
    if command -v dtrules-go &> /dev/null; then
        GO_BIN="dtrules-go"
    else
        GO_BIN=""
    fi
fi

echo "Implementations:"
echo "  ASM: $ASM_BIN ($(stat -c%s "$ASM_BIN" | numfmt --to=iec-i)B)"
if [ -n "$GO_BIN" ] && [ -f "$GO_BIN" ]; then
    echo "  Go:  $GO_BIN ($(stat -c%s "$GO_BIN" | numfmt --to=iec-i)B)"
fi
echo ""
echo "Iterations: $ITERATIONS"
echo ""

#------------------------------------------------------------------------------
# Test 1: ASM with simple rule file
#------------------------------------------------------------------------------

echo "═══════════════════════════════════════════════════════════════"
echo "TEST 1: Startup + Simple Processing"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Create a simple test file for ASM
ASM_TEST_FILE=$(mktemp --suffix=.xml)
cat > "$ASM_TEST_FILE" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="bench">
    <decision_tables>
        <decision_table name="test">
            <actions>
                <action name="compute">
                    1 2 add 3 mul 4 div
                    5 6 7 8 9
                    add add add add
                    dup dup mul mul
                    10 20 lt
                    true false and
                </action>
            </actions>
            <columns>
                <column number="1">
                    <action name="compute"/>
                </column>
            </columns>
        </decision_table>
    </decision_tables>
</ruleset>
EOF

echo "ASM (standalone rules):"
echo -n "  "
start=$(date +%s%N)
for i in $(seq 1 $ITERATIONS); do
    "$ASM_BIN" "$ASM_TEST_FILE" > /dev/null 2>&1
done
end=$(date +%s%N)
asm_total=$(echo "scale=3; ($end - $start) / 1000000" | bc)
asm_per=$(echo "scale=3; $asm_total / $ITERATIONS" | bc)
asm_ops=$(echo "scale=1; 1000 / $asm_per" | bc)
printf "%.3fms per run | %.1f ops/sec\n" $asm_per $asm_ops

rm "$ASM_TEST_FILE"

#------------------------------------------------------------------------------
# Test 2: Go with CHIP sample
#------------------------------------------------------------------------------

if [ -n "$GO_BIN" ] && [ -f "$CHIP_EDD" ]; then
    echo ""
    echo "Go (CHIP sample, -list only):"
    echo -n "  "
    start=$(date +%s%N)
    for i in $(seq 1 $ITERATIONS); do
        "$GO_BIN" -edd "$CHIP_EDD" -dt "$CHIP_DT" -list > /dev/null 2>&1
    done
    end=$(date +%s%N)
    go_total=$(echo "scale=3; ($end - $start) / 1000000" | bc)
    go_per=$(echo "scale=3; $go_total / $ITERATIONS" | bc)
    go_ops=$(echo "scale=1; 1000 / $go_per" | bc)
    printf "%.3fms per run | %.1f ops/sec\n" $go_per $go_ops

    echo ""
    echo "─────────────────────────────────────────────────────────────"
    speedup=$(echo "scale=2; $go_per / $asm_per" | bc)
    echo "Startup comparison: ASM is ${speedup}x faster"
fi

#------------------------------------------------------------------------------
# Test 3: Memory comparison
#------------------------------------------------------------------------------

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "TEST 2: Peak Memory Usage"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Create test file again
ASM_TEST_FILE=$(mktemp --suffix=.xml)
cat > "$ASM_TEST_FILE" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="bench"><decision_tables><decision_table name="t"><actions><action name="a">1 2 add</action></actions><columns><column number="1"><action name="a"/></column></columns></decision_table></decision_tables></ruleset>
EOF

echo "ASM:"
asm_time_output=$(/usr/bin/time -v "$ASM_BIN" "$ASM_TEST_FILE" 2>&1)
asm_mem=$(echo "$asm_time_output" | grep "Maximum resident" | awk '{print $6}')
if [ -z "$asm_mem" ] || [ "$asm_mem" = "0" ]; then
    # Fallback: run multiple times to get measurable memory
    asm_mem=$(/usr/bin/time -v sh -c "for i in 1 2 3 4 5; do \"$ASM_BIN\" \"$ASM_TEST_FILE\"; done" 2>&1 | grep "Maximum resident" | awk '{print $6}')
fi
echo "  Peak RSS: ${asm_mem} KB"

rm "$ASM_TEST_FILE"

if [ -n "$GO_BIN" ] && [ -f "$CHIP_EDD" ]; then
    echo ""
    echo "Go:"
    go_mem=$(/usr/bin/time -v "$GO_BIN" -edd "$CHIP_EDD" -dt "$CHIP_DT" -list 2>&1 | grep "Maximum resident" | awk '{print $6}')
    echo "  Peak RSS: ${go_mem} KB"

    if [ -n "$asm_mem" ] && [ -n "$go_mem" ] && [ "$asm_mem" -gt 0 ]; then
        mem_ratio=$(echo "scale=1; $go_mem / $asm_mem" | bc)
        echo ""
        echo "─────────────────────────────────────────────────────────────"
        echo "Memory comparison: Go uses ${mem_ratio}x more memory"
    fi
fi

#------------------------------------------------------------------------------
# Test 4: Binary size
#------------------------------------------------------------------------------

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "TEST 3: Binary Size"
echo "═══════════════════════════════════════════════════════════════"
echo ""

asm_size=$(stat -c%s "$ASM_BIN")
asm_size_kb=$((asm_size / 1024))
echo "ASM:  ${asm_size_kb} KB"

if [ -n "$GO_BIN" ] && [ -f "$GO_BIN" ]; then
    go_size=$(stat -c%s "$GO_BIN")
    go_size_kb=$((go_size / 1024))
    echo "Go:   ${go_size_kb} KB"

    size_ratio=$(echo "scale=1; $go_size / $asm_size" | bc)
    echo ""
    echo "─────────────────────────────────────────────────────────────"
    echo "Size comparison: Go binary is ${size_ratio}x larger"
fi

#------------------------------------------------------------------------------
# Summary
#------------------------------------------------------------------------------

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "SUMMARY"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "┌─────────────┬────────────────┬────────────────┬─────────────┐"
echo "│ Metric      │ ASM            │ Go             │ Ratio       │"
echo "├─────────────┼────────────────┼────────────────┼─────────────┤"
printf "│ Latency     │ %10.2fms   │ %10.2fms   │ %7.1fx     │\n" $asm_per ${go_per:-0} ${speedup:-0}
printf "│ Memory      │ %10d KB  │ %10d KB  │ %7.1fx     │\n" ${asm_mem:-0} ${go_mem:-0} ${mem_ratio:-0}
printf "│ Binary      │ %10d KB  │ %10d KB  │ %7.1fx     │\n" $asm_size_kb ${go_size_kb:-0} ${size_ratio:-0}
echo "└─────────────┴────────────────┴────────────────┴─────────────┘"
echo ""
echo "Note: ASM tests standalone rules, Go tests CHIP sample (different workloads)"
