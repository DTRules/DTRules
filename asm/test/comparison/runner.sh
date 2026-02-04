#!/bin/bash
# Comparison test runner
# Compares DTRules Go implementation output with Assembly implementation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
RULES_DIR="$SCRIPT_DIR/rules"

echo "DTRules Go vs Assembly Comparison Tests"
echo "========================================"
echo ""

# Check for executables
ASM_BIN="$PROJECT_DIR/dtrules-asm"
GO_BIN="dtrules-go"  # Assumes it's in PATH

if [ ! -f "$ASM_BIN" ]; then
    echo "Error: dtrules-asm not found at $ASM_BIN"
    echo "Run 'make' in the project directory first."
    exit 1
fi

if ! command -v "$GO_BIN" &> /dev/null; then
    echo "Warning: dtrules-go not found in PATH"
    echo "Skipping comparison tests (ASM-only mode)"
    GO_BIN=""
fi

# Create output directory
OUTPUT_DIR="$SCRIPT_DIR/output"
mkdir -p "$OUTPUT_DIR"

# Track results
PASS=0
FAIL=0
SKIP=0

# Run comparison tests
for ruleset in "$RULES_DIR"/*.xml; do
    if [ ! -f "$ruleset" ]; then
        echo "No rule files found in $RULES_DIR"
        exit 0
    fi

    name=$(basename "$ruleset" .xml)
    echo -n "Testing $name... "

    # Run ASM version
    if "$ASM_BIN" "$ruleset" > "$OUTPUT_DIR/${name}_asm.txt" 2>&1; then
        asm_ok=1
    else
        asm_ok=0
    fi

    if [ -n "$GO_BIN" ]; then
        # Run Go version
        if "$GO_BIN" -rules "$ruleset" > "$OUTPUT_DIR/${name}_go.txt" 2>&1; then
            go_ok=1
        else
            go_ok=0
        fi

        # Compare outputs
        if [ $asm_ok -eq 1 ] && [ $go_ok -eq 1 ]; then
            if diff -q "$OUTPUT_DIR/${name}_asm.txt" "$OUTPUT_DIR/${name}_go.txt" > /dev/null 2>&1; then
                echo "PASS"
                ((PASS++))
            else
                echo "FAIL (output differs)"
                echo "  ASM: $OUTPUT_DIR/${name}_asm.txt"
                echo "  Go:  $OUTPUT_DIR/${name}_go.txt"
                ((FAIL++))
            fi
        else
            echo "FAIL (execution error)"
            ((FAIL++))
        fi
    else
        # ASM-only mode
        if [ $asm_ok -eq 1 ]; then
            echo "PASS (ASM only)"
            ((PASS++))
        else
            echo "FAIL"
            ((FAIL++))
        fi
    fi
done

echo ""
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"

if [ $FAIL -gt 0 ]; then
    exit 1
fi
