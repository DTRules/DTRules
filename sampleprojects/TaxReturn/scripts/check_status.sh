#!/bin/bash
# Quick status check for multi-file XML extraction
# Shows summary statistics and any immediate issues

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
XML_DIR="$PROJECT_DIR/xml"

echo "=== DTRules Multi-File XML Extraction Status ==="
echo ""

# Count files
echo "📊 File Statistics:"
echo "  Total XML files:        $(find "$XML_DIR" -name "*.xml" -type f | wc -l)"
echo "  State DT files:         $(find "$XML_DIR/states" -name "*_dt.xml" 2>/dev/null | wc -l)"
echo "  State EDD files:        $(find "$XML_DIR/states" -name "*_edd.xml" 2>/dev/null | wc -l)"
echo "  Core files:             $(find "$XML_DIR" -maxdepth 1 -name "*_core.xml" -type f 2>/dev/null | wc -l)"
echo ""

# Check for monolithic files
echo "📄 Monolithic Files:"
if [ -f "$XML_DIR/TaxReturn_dt.xml" ]; then
  SIZE=$(wc -l < "$XML_DIR/TaxReturn_dt.xml")
  echo "  TaxReturn_dt.xml:       ✓ exists ($SIZE lines)"
else
  echo "  TaxReturn_dt.xml:       ✗ missing"
fi

if [ -f "$XML_DIR/TaxReturn_edd.xml" ]; then
  SIZE=$(wc -l < "$XML_DIR/TaxReturn_edd.xml")
  echo "  TaxReturn_edd.xml:      ✓ exists ($SIZE lines)"
else
  echo "  TaxReturn_edd.xml:      ✗ missing"
fi
echo ""

# Quick XML validation
echo "🔍 Quick Validation:"
PARSE_ERRORS=0
for file in "$XML_DIR"/*.xml "$XML_DIR/states"/*.xml; do
  if [ -f "$file" ]; then
    if ! xmllint --noout "$file" 2>/dev/null; then
      PARSE_ERRORS=$((PARSE_ERRORS + 1))
    fi
  fi
done

if [ $PARSE_ERRORS -eq 0 ]; then
  echo "  XML parse errors:       ✓ none"
else
  echo "  XML parse errors:       ✗ $PARSE_ERRORS files have errors"
fi
echo ""

# Check for scripts
echo "🛠️  Available Tools:"
if [ -f "$SCRIPT_DIR/validate_extraction.py" ]; then
  echo "  validate_extraction.py: ✓ ready"
else
  echo "  validate_extraction.py: ✗ missing"
fi

if [ -f "$SCRIPT_DIR/merge_files.py" ]; then
  echo "  merge_files.py:         ✓ ready"
else
  echo "  merge_files.py:         ✗ missing"
fi
echo ""

# Suggest next steps
echo "📋 Next Steps:"
if [ $PARSE_ERRORS -gt 0 ]; then
  echo "  1. Fix XML parse errors (run with xmllint to find them)"
  echo "  2. Then run: python3 scripts/validate_extraction.py"
else
  echo "  1. Run full validation: python3 scripts/validate_extraction.py > /tmp/validate.log 2>&1"
  echo "  2. Review results:      tail -50 /tmp/validate.log"
  echo "  3. Merge files:         python3 scripts/merge_files.py > /tmp/merge.log 2>&1"
  echo "  4. Run tests:           cd ../../go && go test ./pkg/dtrules/..."
fi
echo ""

echo "📖 Documentation:"
echo "  Full Guide:  EXTRACTION_GUIDE.md"
echo "  Quick Ref:   QUICK_REFERENCE.md"
echo "  Scripts:     scripts/README.md"
echo ""

echo "=== End Status Check ==="
