#!/bin/bash
# DTRules Assembly Test Runner
# Orchestrates all test suites

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "DTRules Assembly Test Runner"
echo "============================"
echo ""

# Track overall results
TOTAL_PASS=0
TOTAL_FAIL=0

#------------------------------------------------------------------------------
# Check prerequisites
#------------------------------------------------------------------------------

# Check if main executable exists
if [ ! -f "$PROJECT_DIR/dtrules-asm" ]; then
    echo "Error: dtrules-asm not found. Run 'make' first."
    exit 1
fi

echo "Executable: $PROJECT_DIR/dtrules-asm"
echo ""

#------------------------------------------------------------------------------
# Basic smoke test
#------------------------------------------------------------------------------

echo "=== Basic Smoke Test ==="
echo ""

echo -n "  Usage message... "
if "$PROJECT_DIR/dtrules-asm" 2>&1 | grep -q "Usage:"; then
    echo "PASS"
    TOTAL_PASS=$((TOTAL_PASS + 1))
else
    echo "FAIL"
    TOTAL_FAIL=$((TOTAL_FAIL + 1))
fi

echo ""

#------------------------------------------------------------------------------
# Unit Tests
#------------------------------------------------------------------------------

echo "=== Unit Tests ==="
echo ""

# Check if unit test infrastructure exists
if [ -f "$SCRIPT_DIR/unit/Makefile" ]; then
    # Build unit tests first
    echo "Building unit tests..."
    if make -C "$SCRIPT_DIR/unit" all 2>/dev/null; then
        echo ""

        # Run each unit test
        for test_exe in "$SCRIPT_DIR/unit/build"/test_*; do
            if [ -x "$test_exe" ]; then
                name=$(basename "$test_exe")
                echo "Running $name..."
                if "$test_exe"; then
                    TOTAL_PASS=$((TOTAL_PASS + 1))
                else
                    TOTAL_FAIL=$((TOTAL_FAIL + 1))
                fi
                echo ""
            fi
        done
    else
        echo "Warning: Could not build unit tests"
        echo ""
    fi
else
    echo "  No unit test Makefile found"
    echo ""
fi

#------------------------------------------------------------------------------
# Integration Tests
#------------------------------------------------------------------------------

echo "=== Integration Tests ==="
echo ""

if [ -f "$SCRIPT_DIR/integration/run_tests.sh" ]; then
    if "$SCRIPT_DIR/integration/run_tests.sh"; then
        TOTAL_PASS=$((TOTAL_PASS + 1))
    else
        TOTAL_FAIL=$((TOTAL_FAIL + 1))
    fi
else
    echo "  No integration tests found (skipping)"
fi

echo ""

#------------------------------------------------------------------------------
# Summary
#------------------------------------------------------------------------------

echo "============================"
echo "Test Summary"
echo "============================"
echo "  Passed: $TOTAL_PASS"
echo "  Failed: $TOTAL_FAIL"
echo ""

if [ $TOTAL_FAIL -gt 0 ]; then
    echo "RESULT: FAILED"
    exit 1
else
    echo "RESULT: PASSED"
    exit 0
fi
