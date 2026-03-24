#!/bin/bash
# merge-states.sh - Validate and export DTRules decision tables
#
# This script validates the merged TaxReturn XML files and exports them to Excel format.
# State-specific files in states/ directory are maintained separately and manually merged
# into TaxReturn_edd.xml and TaxReturn_dt.xml as needed.
#
# Usage:
#   ./scripts/merge-states.sh
#
# Input:
#   - xml/TaxReturn_edd.xml (entity data dictionary)
#   - xml/TaxReturn_dt.xml (decision tables)
#
# Output:
#   - excel/*.xlsx (Excel exports for documentation)

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
XML_DIR="$SCRIPT_DIR/../xml"

echo "=== DTRules Decision Table Exporter ==="
echo "XML directory: $XML_DIR"
echo ""

# Check for required tools
if ! command -v xmllint &> /dev/null; then
    echo "Warning: xmllint not found. XML validation will be skipped."
    echo "Install: sudo apt-get install libxml2-utils (Ubuntu/Debian)"
    VALIDATE=0
else
    VALIDATE=1
fi

# ============================================================================
# Validate EDD File
# ============================================================================

EDD_FILE="$XML_DIR/TaxReturn_edd.xml"

if [ ! -f "$EDD_FILE" ]; then
    echo "Error: EDD file not found: $EDD_FILE"
    exit 1
fi

echo "Validating EDD file..."
if [ $VALIDATE -eq 1 ]; then
    if xmllint --noout "$EDD_FILE" 2>/dev/null; then
        echo "  ✓ EDD is valid XML"
    else
        echo "  ✗ Warning: EDD has XML errors"
        xmllint --noout "$EDD_FILE" || true
    fi
else
    echo "  - Validation skipped (xmllint not available)"
fi

# ============================================================================
# Validate Decision Table File
# ============================================================================

DT_FILE="$XML_DIR/TaxReturn_dt.xml"

if [ ! -f "$DT_FILE" ]; then
    echo "Error: Decision table file not found: $DT_FILE"
    exit 1
fi

echo ""
echo "Validating Decision Table file..."
if [ $VALIDATE -eq 1 ]; then
    if xmllint --noout "$DT_FILE" 2>/dev/null; then
        echo "  ✓ DT is valid XML"
    else
        echo "  ✗ Warning: DT has XML errors"
        xmllint --noout "$DT_FILE" || true
    fi
else
    echo "  - Validation skipped (xmllint not available)"
fi

# ============================================================================
# Summary
# ============================================================================

echo ""
echo "=== Validation Complete ==="
echo "Files validated:"
echo "  - $EDD_FILE"
echo "  - $DT_FILE"
echo ""

# Extract to Excel format
echo "Extracting to Excel format..."
if [ -f "$SCRIPT_DIR/extract-to-excel.sh" ]; then
    "$SCRIPT_DIR/extract-to-excel.sh" 2>&1 | tail -5
else
    echo "Note: extract-to-excel.sh not found, skipping Excel extraction"
fi

echo ""
echo "Files are ready for testing with: go test ./pkg/dtrules/..."
