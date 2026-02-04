#!/bin/bash
# Run DTRules assembly unit tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "DTRules Assembly Test Runner"
echo "============================"
echo ""

# Check if main executable exists
if [ ! -f "$PROJECT_DIR/dtrules-asm" ]; then
    echo "Error: dtrules-asm not found. Run 'make' first."
    exit 1
fi

# Run basic functionality test
echo "Testing basic execution..."

# Test 1: Help/usage message
echo -n "  Usage message... "
if $PROJECT_DIR/dtrules-asm 2>&1 | grep -q "Usage:"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

echo ""
echo "All basic tests passed!"
echo ""

# Integration tests would go here if we had sample XML files
# For now, just verify the binary runs

echo "Test Summary"
echo "------------"
echo "Basic tests: PASS"
