#!/bin/bash
# Comparison test runner
# Compares DTRules Go implementation output with Assembly implementation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
RULES_DIR="$SCRIPT_DIR/rules"
NORMALIZE_SCRIPT="$SCRIPT_DIR/normalize.py"

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

# Check for normalizer
if [ ! -f "$NORMALIZE_SCRIPT" ]; then
    echo "Warning: normalize.py not found at $NORMALIZE_SCRIPT"
    echo "Comparison will use raw output"
    NORMALIZE_CMD="cat"
else
    NORMALIZE_CMD="python3 $NORMALIZE_SCRIPT"
fi

# Check for Go binary
HAS_GO=0
if command -v "$GO_BIN" &> /dev/null; then
    HAS_GO=1
    echo "Go binary: $(which $GO_BIN)"
else
    echo "Warning: dtrules-go not found in PATH"
    echo "Running in ASM-only mode (smoke test)"
fi

echo "ASM binary: $ASM_BIN"
echo ""

# Create output directory
OUTPUT_DIR="$SCRIPT_DIR/output"
mkdir -p "$OUTPUT_DIR"

# Track results
PASS=0
FAIL=0
SKIP=0

# JSON report file
REPORT_FILE="$OUTPUT_DIR/report.json"
echo "[" > "$REPORT_FILE"
FIRST_ENTRY=1

# Run comparison tests
for ruleset in "$RULES_DIR"/*.xml; do
    if [ ! -f "$ruleset" ]; then
        echo "No rule files found in $RULES_DIR"
        exit 0
    fi

    name=$(basename "$ruleset" .xml)
    echo -n "Testing $name... "

    # Run ASM version
    asm_output_raw="$OUTPUT_DIR/${name}_asm_raw.txt"
    asm_output="$OUTPUT_DIR/${name}_asm.txt"
    asm_error="$OUTPUT_DIR/${name}_asm_error.txt"

    if "$ASM_BIN" "$ruleset" > "$asm_output_raw" 2> "$asm_error"; then
        asm_ok=1
        # Normalize output
        $NORMALIZE_CMD < "$asm_output_raw" > "$asm_output"
    else
        asm_ok=0
        asm_exit_code=$?
    fi

    if [ $HAS_GO -eq 1 ]; then
        # Run Go version
        go_output_raw="$OUTPUT_DIR/${name}_go_raw.txt"
        go_output="$OUTPUT_DIR/${name}_go.txt"
        go_error="$OUTPUT_DIR/${name}_go_error.txt"

        if "$GO_BIN" -rules "$ruleset" > "$go_output_raw" 2> "$go_error"; then
            go_ok=1
            # Normalize output
            $NORMALIZE_CMD < "$go_output_raw" > "$go_output"
        else
            go_ok=0
            go_exit_code=$?
        fi

        # Compare outputs
        result="unknown"
        details=""

        if [ $asm_ok -eq 1 ] && [ $go_ok -eq 1 ]; then
            if diff -q "$asm_output" "$go_output" > /dev/null 2>&1; then
                echo "PASS"
                result="pass"
                ((PASS++))
            else
                echo "FAIL (output differs)"
                result="fail"
                details="Output mismatch"

                # Show diff
                echo "  Diff:"
                diff -u "$go_output" "$asm_output" | head -20 | sed 's/^/    /'
                echo ""
                echo "  Full diff: diff $go_output $asm_output"
                ((FAIL++))
            fi
        elif [ $asm_ok -eq 0 ] && [ $go_ok -eq 0 ]; then
            # Both failed - check if same error
            echo "FAIL (both errored)"
            result="fail"
            details="Both implementations errored"
            ((FAIL++))
        elif [ $asm_ok -eq 0 ]; then
            echo "FAIL (ASM error)"
            result="fail"
            details="ASM execution failed"
            echo "  ASM stderr: $(cat "$asm_error" | head -5)"
            ((FAIL++))
        else
            echo "FAIL (Go error)"
            result="fail"
            details="Go execution failed"
            echo "  Go stderr: $(cat "$go_error" | head -5)"
            ((FAIL++))
        fi
    else
        # ASM-only mode
        if [ $asm_ok -eq 1 ]; then
            echo "PASS (ASM only)"
            result="pass"
            details="ASM-only smoke test"
            ((PASS++))
        else
            echo "FAIL (ASM error)"
            result="fail"
            details="ASM execution failed"
            echo "  Error: $(cat "$asm_error" | head -5)"
            ((FAIL++))
        fi
    fi

    # Add to JSON report
    if [ $FIRST_ENTRY -eq 0 ]; then
        echo "," >> "$REPORT_FILE"
    fi
    FIRST_ENTRY=0

    cat >> "$REPORT_FILE" << EOF
  {
    "name": "$name",
    "result": "$result",
    "details": "$details",
    "asm_output": "${name}_asm.txt",
    "go_output": "${name}_go.txt"
  }
EOF
done

# Close JSON array
echo "" >> "$REPORT_FILE"
echo "]" >> "$REPORT_FILE"

echo ""
echo "========================================"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo ""
echo "Output files: $OUTPUT_DIR/"
echo "JSON report:  $REPORT_FILE"

if [ $FAIL -gt 0 ]; then
    exit 1
fi
