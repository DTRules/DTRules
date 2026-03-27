#!/bin/bash
# extract-to-excel.sh - Extract XML decision tables and EDD to Excel format
#
# This script converts DTRules XML files to Excel (.xlsx) format for easier
# viewing and editing in spreadsheet applications.
#
# Usage:
#   ./scripts/extract-to-excel.sh

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/.."
XML_DIR="$PROJECT_ROOT/xml"
EXCEL_DIR="$PROJECT_ROOT/excel"
GO_DIR="$SCRIPT_DIR/../../../go"

echo "=== DTRules XML to Excel Extractor ==="
echo "Project: $PROJECT_ROOT"
echo "XML directory: $XML_DIR"
echo "Excel output: $EXCEL_DIR"
echo ""

# Check for Go tools
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Build tools if needed
cd "$GO_DIR"
echo "Building extraction tools..."
if [ ! -f "dt2excel" ] || [ "cmd/dt2excel/main.go" -nt "dt2excel" ]; then
    echo "  Building dt2excel..."
    go build -o dt2excel ./cmd/dt2excel || {
        echo "Error: Failed to build dt2excel"
        exit 1
    }
fi

if [ ! -f "edd2excel" ] || [ "cmd/edd2excel/main.go" -nt "edd2excel" ]; then
    echo "  Building edd2excel..."
    go build -o edd2excel ./cmd/edd2excel || {
        echo "Error: Failed to build edd2excel"
        exit 1
    }
fi

echo "✓ Tools built successfully"
echo ""

# Create output directory
mkdir -p "$EXCEL_DIR"

# Extract EDD
echo "Extracting Entity Data Dictionary..."
if [ -f "$XML_DIR/TaxReturn_edd.xml" ]; then
    ./edd2excel \
        -input "$XML_DIR/TaxReturn_edd.xml" \
        -output "$EXCEL_DIR/TaxReturn_edd.xlsx"
    echo ""
else
    echo "Warning: TaxReturn_edd.xml not found"
fi

# Extract Decision Tables
echo "Extracting Decision Tables..."
if [ -f "$XML_DIR/TaxReturn_dt.xml" ]; then
    ./dt2excel \
        -input "$XML_DIR/TaxReturn_dt.xml" \
        -output "$EXCEL_DIR"
    echo ""
else
    echo "Warning: TaxReturn_dt.xml not found"
fi

# Count files
DT_COUNT=$(ls -1 "$EXCEL_DIR"/*.xlsx 2>/dev/null | wc -l)
TOTAL_SIZE=$(du -sh "$EXCEL_DIR" 2>/dev/null | cut -f1)

echo "=== Extraction Complete ==="
echo "Excel files created: $DT_COUNT"
echo "Total size: $TOTAL_SIZE"
echo "Location: $EXCEL_DIR"
echo ""
echo "You can now:"
echo "  - View decision tables in Excel/LibreOffice"
echo "  - Search across tables using your file manager"
echo "  - Print documentation"
echo "  - Share with non-technical stakeholders"
