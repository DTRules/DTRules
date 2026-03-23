#!/bin/bash
# merge-states.sh - Merge separate state tax files into single EDD and DT files
#
# This script merges state-specific XML files from states/ directory into the
# main TaxReturn_edd.xml and TaxReturn_dt.xml files used by the test harness.
#
# Usage:
#   ./scripts/merge-states.sh
#
# Input:
#   - xml/TaxReturn_edd_core.xml (core entities, no state-specific constants)
#   - xml/TaxReturn_dt_core.xml (core tables, no state-specific tables)
#   - xml/states/*_edd.xml (state-specific constants)
#   - xml/states/*_dt.xml (state-specific decision tables)
#
# Output:
#   - xml/TaxReturn_edd.xml (merged)
#   - xml/TaxReturn_dt.xml (merged)

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
XML_DIR="$SCRIPT_DIR/../xml"
STATES_DIR="$XML_DIR/states"

echo "=== DTRules State Tax File Merger ==="
echo "XML directory: $XML_DIR"
echo "States directory: $STATES_DIR"
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
# Merge EDD Files
# ============================================================================

echo "Merging EDD files..."

EDD_CORE="$XML_DIR/TaxReturn_edd_core.xml"
EDD_OUTPUT="$XML_DIR/TaxReturn_edd.xml"

if [ ! -f "$EDD_CORE" ]; then
    echo "Error: Core EDD file not found: $EDD_CORE"
    echo "This file should contain core entities without state-specific constants."
    exit 1
fi

# Start with core EDD (but remove closing tag)
head -n -1 "$EDD_CORE" > "$EDD_OUTPUT"

# Process each state EDD file
STATE_COUNT=0
if [ -d "$STATES_DIR" ]; then
    for state_edd in "$STATES_DIR"/*_edd.xml; do
        if [ -f "$state_edd" ]; then
            STATE_CODE=$(basename "$state_edd" _edd.xml)
            echo "  - Merging $STATE_CODE constants..."

            # Extract entity content (skip XML declaration and root tags)
            # This assumes state files have <entity_data_dictionary><entity>...</entity></entity_data_dictionary>
            sed -n '/<entity /,/<\/entity>/p' "$state_edd" >> "$EDD_OUTPUT"

            STATE_COUNT=$((STATE_COUNT + 1))
        fi
    done
fi

# Close the EDD file
echo "</entity_data_dictionary>" >> "$EDD_OUTPUT"

echo "  ✓ Merged $STATE_COUNT state EDD files"

# Validate merged EDD
if [ $VALIDATE -eq 1 ]; then
    if xmllint --noout "$EDD_OUTPUT" 2>/dev/null; then
        echo "  ✓ Merged EDD is valid XML"
    else
        echo "  ✗ Warning: Merged EDD has XML errors"
        xmllint --noout "$EDD_OUTPUT" || true
    fi
fi

# ============================================================================
# Merge Decision Table Files
# ============================================================================

echo ""
echo "Merging Decision Table files..."

DT_CORE="$XML_DIR/TaxReturn_dt_core.xml"
DT_OUTPUT="$XML_DIR/TaxReturn_dt.xml"

if [ ! -f "$DT_CORE" ]; then
    echo "Error: Core DT file not found: $DT_CORE"
    echo "This file should contain core decision tables without state-specific tables."
    exit 1
fi

# Start with core DT (but remove closing tag)
head -n -1 "$DT_CORE" > "$DT_OUTPUT"

# Process each state DT file
STATE_COUNT=0
if [ -d "$STATES_DIR" ]; then
    for state_dt in "$STATES_DIR"/*_dt.xml; do
        if [ -f "$state_dt" ]; then
            STATE_CODE=$(basename "$state_dt" _dt.xml)
            echo "  - Merging $STATE_CODE decision tables..."

            # Extract decision_table content (skip XML declaration and root tags)
            sed -n '/<decision_table>/,/<\/decision_table>/p' "$state_dt" >> "$DT_OUTPUT"

            STATE_COUNT=$((STATE_COUNT + 1))
        fi
    done
fi

# Close the DT file
echo "</decision_tables>" >> "$DT_OUTPUT"

echo "  ✓ Merged $STATE_COUNT state DT files"

# Validate merged DT
if [ $VALIDATE -eq 1 ]; then
    if xmllint --noout "$DT_OUTPUT" 2>/dev/null; then
        echo "  ✓ Merged DT is valid XML"
    else
        echo "  ✗ Warning: Merged DT has XML errors"
        xmllint --noout "$DT_OUTPUT" || true
    fi
fi

# ============================================================================
# Summary
# ============================================================================

echo ""
echo "=== Merge Complete ==="
echo "Generated files:"
echo "  - $EDD_OUTPUT"
echo "  - $DT_OUTPUT"
echo ""
echo "These files are ready for testing with: go test ./pkg/dtrules/..."
